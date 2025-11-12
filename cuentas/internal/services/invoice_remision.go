package services

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"cuentas/internal/database"
	"cuentas/internal/models"
)

// CreateRemision creates a new remision (Type 04 - goods movement document)
func (s *InvoiceService) CreateRemision(ctx context.Context, companyID string, req *models.CreateRemisionRequest) (*models.Invoice, error) {
	// Validate request
	if err := req.Validate(); err != nil {
		return nil, err
	}

	// Begin transaction
	tx, err := database.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// 1. Validate establishment and POS
	if err := s.validatePointOfSale(ctx, tx, companyID, req.EstablishmentID, req.PointOfSaleID); err != nil {
		return nil, err
	}

	// 2. Snapshot client data (if receptor provided - can be null for internal transfers)
	var client *ClientSnapshot
	if req.ClientID != nil && *req.ClientID != "" {
		var err error
		client, err = s.snapshotClient(ctx, tx, companyID, *req.ClientID)
		if err != nil {
			return nil, err
		}
	}

	// 3. Generate invoice number
	invoiceNumber, err := s.generateInvoiceNumber(ctx, tx, companyID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate invoice number: %w", err)
	}

	// 4. Process line items - for remisiones, typically $0 amounts
	lineItems, subtotal, totalDiscount, totalTaxes, err := s.processLineItemsRemision(ctx, tx, companyID, req.LineItems)
	if err != nil {
		return nil, err
	}

	// 5. Total for remision is typically 0
	total := round(subtotal - totalDiscount + totalTaxes)

	// 6. Create remision record
	remision := &models.Invoice{
		CompanyID:       companyID,
		EstablishmentID: req.EstablishmentID,
		PointOfSaleID:   req.PointOfSaleID,
		InvoiceNumber:   invoiceNumber,
		InvoiceType:     "sale",
		RemisionType:    &req.RemisionType,
		DeliveryPerson:  req.DeliveryPerson,
		VehiclePlate:    req.VehiclePlate,
		DeliveryNotes:   req.DeliveryNotes,
		Subtotal:        subtotal,
		TotalDiscount:   totalDiscount,
		TotalTaxes:      totalTaxes,
		Total:           total,
		Currency:        "USD",
		// Payment fields are NULL for remisiones (not a sale)
		AmountPaid: 0,
		BalanceDue: 0,
		Status:     "draft",
		Notes:      req.Notes,
		CreatedAt:  time.Now(),
	}

	// Set client fields if receptor provided
	if client != nil {
		remision.ClientID = *req.ClientID
		remision.ClientName = client.ClientName
		remision.ClientLegalName = client.ClientLegalName
		remision.ClientNit = client.ClientNit
		remision.ClientNcr = client.ClientNcr
		remision.ClientDui = client.ClientDui
		remision.ClientAddress = client.ClientAddress
		remision.ClientTipoContribuyente = client.ClientTipoContribuyente
		remision.ClientTipoPersona = client.ClientTipoPersona
	} else {
		// Internal transfer - no receptor
		remision.ClientID = ""
		remision.ClientName = "TRASLADO INTERNO"
		remision.ClientLegalName = "TRASLADO INTERNO"
		remision.ClientAddress = "N/A"
	}

	// 7. Insert remision
	remisionID, err := s.insertRemision(ctx, tx, remision)
	if err != nil {
		return nil, fmt.Errorf("failed to insert remision: %w", err)
	}
	remision.ID = remisionID

	// 8. Insert line items
	for i := range lineItems {
		lineItems[i].InvoiceID = remisionID
		lineItems[i].LineNumber = i + 1

		lineItemID, err := s.insertLineItem(ctx, tx, &lineItems[i])
		if err != nil {
			return nil, fmt.Errorf("failed to insert line item %d: %w", i+1, err)
		}
		lineItems[i].ID = lineItemID

		// Insert taxes (typically none for remision)
		for j := range lineItems[i].Taxes {
			lineItems[i].Taxes[j].LineItemID = lineItemID
			taxID, err := s.insertLineItemTax(ctx, tx, &lineItems[i].Taxes[j])
			if err != nil {
				return nil, fmt.Errorf("failed to insert tax for line item %d: %w", i+1, err)
			}
			lineItems[i].Taxes[j].ID = taxID
		}
	}

	// 9. Insert related documents if provided
	if len(req.RelatedDocuments) > 0 {
		if err := s.insertRelatedDocuments(ctx, tx, remisionID, req.RelatedDocuments); err != nil {
			return nil, fmt.Errorf("failed to insert related documents: %w", err)
		}
	}

	// 10. Commit transaction
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// 11. Attach line items
	remision.LineItems = lineItems

	return remision, nil
}

// processLineItemsRemision processes line items for remision (typically $0 amounts)
func (s *InvoiceService) processLineItemsRemision(ctx context.Context, tx *sql.Tx, companyID string, reqItems []models.CreateInvoiceLineItemRequest) ([]models.InvoiceLineItem, float64, float64, float64, error) {
	var lineItems []models.InvoiceLineItem

	for _, reqItem := range reqItems {
		// 1. Snapshot inventory item
		item, err := s.snapshotInventoryItem(ctx, tx, companyID, reqItem.ItemID)
		if err != nil {
			return nil, 0, 0, 0, err
		}

		// 2. For remision: amounts are typically 0 (just tracking movement)
		// We track quantity for inventory purposes, but no monetary value
		lineSubtotal := 0.0
		discountAmount := 0.0
		taxableAmount := 0.0
		lineTaxTotal := 0.0
		lineTotal := 0.0

		// 3. Create line item (tracking quantity but with $0 amounts)
		lineItem := models.InvoiceLineItem{
			ItemID:             &reqItem.ItemID,
			ItemSku:            item.SKU,
			ItemName:           item.Name,
			ItemDescription:    item.Description,
			ItemTipoItem:       item.TipoItem,
			UnitOfMeasure:      item.UnitOfMeasure,
			UnitPrice:          0, // Remision: no sale price
			Quantity:           reqItem.Quantity,
			LineSubtotal:       lineSubtotal,
			DiscountPercentage: 0,
			DiscountAmount:     discountAmount,
			TaxableAmount:      taxableAmount,
			TotalTaxes:         lineTaxTotal,
			LineTotal:          lineTotal,
			Taxes:              []models.InvoiceLineItemTax{}, // No taxes for remision
			CreatedAt:          time.Now(),
		}

		lineItems = append(lineItems, lineItem)
	}

	// All totals are 0 for remision
	return lineItems, 0, 0, 0, nil
}

// insertRemision inserts a remision with nullable client and payment fields
func (s *InvoiceService) insertRemision(ctx context.Context, tx *sql.Tx, invoice *models.Invoice) (string, error) {
	id := strings.ToUpper(uuid.New().String())

	query := `
        INSERT INTO invoices (
            id, company_id, establishment_id, point_of_sale_id, client_id,
            invoice_number, invoice_type,
            remision_type, delivery_person, vehicle_plate, delivery_notes,
            client_name, client_legal_name, client_nit, client_ncr, client_dui,
            client_address, client_tipo_contribuyente, client_tipo_persona,
            subtotal, total_discount, total_taxes, total,
            currency,
            amount_paid, balance_due,
            dte_codigo_generacion,
            status, notes, created_at
        ) VALUES (
            $1, $2, $3, $4, $5,
            $6, $7,
            $8, $9, $10, $11,
            $12, $13, $14, $15, $16,
            $17, $18, $19,
            $20, $21, $22, $23,
            $24,
            $25, $26,
            $27,
            $28, $29, $30
        ) RETURNING id
    `

	// Handle NULL client_id for internal transfers
	var clientID interface{}
	if invoice.ClientID == "" {
		clientID = nil
	} else {
		clientID = invoice.ClientID
	}

	_, err := tx.ExecContext(ctx, query,
		id, invoice.CompanyID, invoice.EstablishmentID, invoice.PointOfSaleID, clientID, // Use interface{} for NULL
		invoice.InvoiceNumber, invoice.InvoiceType,
		invoice.RemisionType, invoice.DeliveryPerson, invoice.VehiclePlate, invoice.DeliveryNotes,
		invoice.ClientName, invoice.ClientLegalName, invoice.ClientNit, invoice.ClientNcr, invoice.ClientDui,
		invoice.ClientAddress, invoice.ClientTipoContribuyente, invoice.ClientTipoPersona,
		invoice.Subtotal, invoice.TotalDiscount, invoice.TotalTaxes, invoice.Total,
		invoice.Currency,
		invoice.AmountPaid, invoice.BalanceDue,
		id, // dte_codigo_generacion = invoice ID
		invoice.Status, invoice.Notes, invoice.CreatedAt,
	)

	if err != nil {
		return "", err
	}

	return id, nil
}

// insertRelatedDocuments inserts related documents (documentoRelacionado)
func (s *InvoiceService) insertRelatedDocuments(ctx context.Context, tx *sql.Tx, invoiceID string, docs []models.RelatedDocumentInput) error {
	query := `
        INSERT INTO invoice_related_documents (
            invoice_id, tipo_documento, tipo_generacion,
            numero_documento, fecha_emision
        ) VALUES ($1, $2, $3, $4, $5)
    `

	for _, doc := range docs {
		_, err := tx.ExecContext(ctx, query,
			invoiceID, doc.TipoDocumento, doc.TipoGeneracion,
			doc.NumeroDocumento, doc.FechaEmision,
		)
		if err != nil {
			return fmt.Errorf("failed to insert related document: %w", err)
		}
	}

	return nil
}

// FinalizeRemision finalizes a draft remision and generates DTE identifiers
func (s *InvoiceService) FinalizeRemision(ctx context.Context, companyID, remisionID, userID string) (*models.Invoice, error) {
	// Begin transaction
	tx, err := database.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// 1. Get remision and verify it's a draft
	remision, err := s.getInvoiceForUpdate(ctx, tx, companyID, remisionID)
	if err != nil {
		return nil, err
	}

	if remision.Status != "draft" {
		return nil, ErrInvoiceNotDraft
	}

	// 2. Verify it's actually a remision
	if !remision.IsRemision() {
		return nil, fmt.Errorf("document is not a remision (Type 04)")
	}

	// 3. Generate DTE identifiers for Type 04
	numeroControl, err := s.generateNumeroControl(ctx, tx, remision.EstablishmentID, remision.PointOfSaleID, remision.PointOfSaleID, "04")
	if err != nil {
		return nil, fmt.Errorf("failed to generate numero control: %w", err)
	}

	// 4. Update remision to finalized
	now := time.Now()
	tipoDte := "04"

	updateQuery := `
        UPDATE invoices
        SET status = 'finalized',
            dte_numero_control = $1,
            dte_type = $2,
            dte_status = 'not_submitted',
            finalized_at = $3,
            created_by = $4
        WHERE id = $5 AND company_id = $6
    `

	_, err = tx.ExecContext(ctx, updateQuery,
		numeroControl,
		tipoDte,
		now,
		userID,
		remisionID,
		companyID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update remision: %w", err)
	}

	// 5. Note: Remisiones do NOT deduct inventory
	// Inventory is deducted when the actual sale invoice is created later

	// 6. Commit transaction
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// 7. Get and return the finalized remision
	finalizedRemision, err := s.GetInvoice(ctx, companyID, remisionID)
	if err != nil {
		return nil, err
	}

	return finalizedRemision, nil
}

// LinkRemisionToInvoice creates a link between a remision and an invoice
// Used for route sales scenario where one remision leads to multiple invoices
func (s *InvoiceService) LinkRemisionToInvoice(ctx context.Context, companyID, remisionID, invoiceID string) error {
	// Verify remision exists and is Type 04
	remision, err := s.GetInvoice(ctx, companyID, remisionID)
	if err != nil {
		return fmt.Errorf("remision not found: %w", err)
	}
	if !remision.IsRemision() {
		return fmt.Errorf("document %s is not a remision", remisionID)
	}

	// Verify invoice exists
	invoice, err := s.GetInvoice(ctx, companyID, invoiceID)
	if err != nil {
		return fmt.Errorf("invoice not found: %w", err)
	}
	if invoice.IsRemision() {
		return fmt.Errorf("cannot link remision to another remision")
	}

	// Create link
	query := `
        INSERT INTO remision_invoice_links (remision_id, invoice_id)
        VALUES ($1, $2)
        ON CONFLICT (remision_id, invoice_id) DO NOTHING
    `

	_, err = database.DB.ExecContext(ctx, query, remisionID, invoiceID)
	if err != nil {
		return fmt.Errorf("failed to create remision-invoice link: %w", err)
	}

	return nil
}

// GetRemisionLinkedInvoices retrieves all invoices linked to a remision
// Useful for route sales tracking
func (s *InvoiceService) GetRemisionLinkedInvoices(ctx context.Context, companyID, remisionID string) ([]models.Invoice, error) {
	query := `
        SELECT
            i.id, i.company_id, i.establishment_id, i.point_of_sale_id, i.client_id,
            i.invoice_number, i.invoice_type,
            i.client_name, i.client_legal_name,
            i.subtotal, i.total_discount, i.total_taxes, i.total,
            i.payment_terms, i.payment_status, i.amount_paid, i.balance_due,
            i.status, i.dte_status, i.dte_numero_control, i.dte_type,
            i.created_at, i.finalized_at
        FROM invoices i
        INNER JOIN remision_invoice_links ril ON i.id = ril.invoice_id
        WHERE ril.remision_id = $1 AND i.company_id = $2
        ORDER BY i.created_at DESC
    `

	rows, err := database.DB.QueryContext(ctx, query, remisionID, companyID)
	if err != nil {
		return nil, fmt.Errorf("failed to query linked invoices: %w", err)
	}
	defer rows.Close()

	var invoices []models.Invoice
	for rows.Next() {
		var inv models.Invoice
		err := rows.Scan(
			&inv.ID, &inv.CompanyID, &inv.EstablishmentID, &inv.PointOfSaleID, &inv.ClientID,
			&inv.InvoiceNumber, &inv.InvoiceType,
			&inv.ClientName, &inv.ClientLegalName,
			&inv.Subtotal, &inv.TotalDiscount, &inv.TotalTaxes, &inv.Total,
			&inv.PaymentTerms, &inv.PaymentStatus, &inv.AmountPaid, &inv.BalanceDue,
			&inv.Status, &inv.DteStatus, &inv.DteNumeroControl, &inv.DteType,
			&inv.CreatedAt, &inv.FinalizedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan invoice: %w", err)
		}
		invoices = append(invoices, inv)
	}

	return invoices, rows.Err()
}
