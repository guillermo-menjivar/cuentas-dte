package services

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"

	"cuentas/internal/codigos"
	"cuentas/internal/database"
	"cuentas/internal/models"
	"cuentas/internal/models/dte"
)

type InvoiceService struct{}

func NewInvoiceService() *InvoiceService {
	return &InvoiceService{}
}

func (s *InvoiceService) validatePointOfSale(ctx context.Context, tx *sql.Tx, companyID, establishmentID, posID string) error {
	query := `
		SELECT pos.id
		FROM point_of_sale pos
		JOIN establishments e ON pos.establishment_id = e.id
		WHERE pos.id = $1 
			AND e.id = $2 
			AND e.company_id = $3 
			AND pos.active = true 
			AND e.active = true
	`

	var id string
	err := tx.QueryRowContext(ctx, query, posID, establishmentID, companyID).Scan(&id)
	if err == sql.ErrNoRows {
		return ErrPointOfSaleNotFound
	}
	if err != nil {
		return fmt.Errorf("failed to validate point of sale: %w", err)
	}

	return nil
}

func (s *InvoiceService) CreateInvoice(ctx context.Context, companyID string, req *models.CreateInvoiceRequest) (*models.Invoice, error) {
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

	// 1. Validate establishment and POS belong together and to company
	if err := s.validatePointOfSale(ctx, tx, companyID, req.EstablishmentID, req.PointOfSaleID); err != nil {
		return nil, err
	}

	// 2. Snapshot client data
	client, err := s.snapshotClient(ctx, tx, companyID, req.ClientID)
	if err != nil {
		return nil, err
	}

	// 3. Generate invoice number
	invoiceNumber, err := s.generateInvoiceNumber(ctx, tx, companyID)
	fmt.Println("this is the invoice number I got", invoiceNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to generate invoice number: %w", err)
	}

	// 4. Process line items and calculate totals
	lineItems, subtotal, totalDiscount, totalTaxes, err := s.processLineItems(ctx, tx, companyID, req.LineItems)
	if err != nil {
		return nil, err
	}

	total := round(subtotal - totalDiscount + totalTaxes)

	// 5. Calculate due date if needed
	var dueDate *time.Time
	if req.DueDate != nil {
		dueDate = req.DueDate
	} else if req.PaymentTerms == "net_30" {
		date := time.Now().AddDate(0, 0, 30)
		dueDate = &date
	} else if req.PaymentTerms == "net_60" {
		date := time.Now().AddDate(0, 0, 60)
		dueDate = &date
	}

	// 6. Create invoice record
	invoice := &models.Invoice{
		CompanyID:               companyID,
		EstablishmentID:         req.EstablishmentID,
		PointOfSaleID:           req.PointOfSaleID,
		ClientID:                req.ClientID,
		InvoiceNumber:           invoiceNumber,
		InvoiceType:             "sale",
		ClientName:              client.ClientName,
		ClientLegalName:         client.ClientLegalName,
		ClientNit:               client.ClientNit,
		ClientNcr:               client.ClientNcr,
		ClientDui:               client.ClientDui,
		ClientAddress:           client.ClientAddress,
		ClientTipoContribuyente: client.ClientTipoContribuyente,
		ClientTipoPersona:       client.ClientTipoPersona,
		Subtotal:                subtotal,
		TotalDiscount:           totalDiscount,
		TotalTaxes:              totalTaxes,
		Total:                   total,
		Currency:                "USD",
		PaymentMethod:           req.PaymentMethod,
		PaymentTerms:            req.PaymentTerms,
		PaymentStatus:           "unpaid",
		AmountPaid:              0,
		BalanceDue:              total,
		DueDate:                 dueDate,
		Status:                  "draft",
		Notes:                   req.Notes,
		ContactEmail:            req.ContactEmail,
		ContactWhatsapp:         req.ContactWhatsapp,
		CreatedAt:               time.Now(),
	}

	// 7. Insert invoice
	invoiceID, err := s.insertInvoice(ctx, tx, invoice)
	if err != nil {
		return nil, fmt.Errorf("failed to insert invoice: %w", err)
	}
	invoice.ID = invoiceID

	// 8. Insert line items and taxes

	for i := range lineItems {
		lineItems[i].InvoiceID = invoiceID
		lineItems[i].LineNumber = i + 1

		lineItemID, err := s.insertLineItem(ctx, tx, &lineItems[i])
		if err != nil {
			return nil, fmt.Errorf("failed to insert line item %d: %w", i+1, err)
		}
		lineItems[i].ID = lineItemID

		// Insert taxes for this line item
		for j := range lineItems[i].Taxes {
			lineItems[i].Taxes[j].LineItemID = lineItemID
			taxID, err := s.insertLineItemTax(ctx, tx, &lineItems[i].Taxes[j])
			if err != nil {
				return nil, fmt.Errorf("failed to insert tax for line item %d: %w", i+1, err)
			}
			lineItems[i].Taxes[j].ID = taxID // SET THE ID
		}
	}

	// 9. Commit transaction
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// 10. Attach line items to invoice
	invoice.LineItems = lineItems

	return invoice, nil
}

// ClientSnapshot represents the snapshot of client data at transaction time
type ClientSnapshot struct {
	ClientName              string
	ClientLegalName         string
	ClientNit               *string
	ClientNcr               *string
	ClientDui               *string
	ClientAddress           string
	ClientTipoContribuyente *string
	ClientTipoPersona       *string
}

// snapshotClient retrieves and snapshots client data
func (s *InvoiceService) snapshotClient(ctx context.Context, tx *sql.Tx, companyID, clientID string) (*ClientSnapshot, error) {
	query := `
		SELECT
			business_name,
			legal_business_name,
			nit,
			ncr,
			dui,
			full_address,
			tipo_contribuyente,
			tipo_persona
		FROM clients
		WHERE id = $1 AND company_id = $2 AND active = true
	`

	var snapshot ClientSnapshot
	err := tx.QueryRowContext(ctx, query, clientID, companyID).Scan(
		&snapshot.ClientName,
		&snapshot.ClientLegalName,
		&snapshot.ClientNit,
		&snapshot.ClientNcr,
		&snapshot.ClientDui,
		&snapshot.ClientAddress,
		&snapshot.ClientTipoContribuyente,
		&snapshot.ClientTipoPersona,
	)

	if err == sql.ErrNoRows {
		return nil, ErrClientNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query client: %w", err)
	}

	return &snapshot, nil
}

// generateInvoiceNumber generates a sequential invoice number
func (s *InvoiceService) generateInvoiceNumber(ctx context.Context, tx *sql.Tx, companyID string) (string, error) {
	var lastNumber sql.NullString
	query := `
		SELECT invoice_number
		FROM invoices
		WHERE company_id = $1
		ORDER BY created_at DESC
		LIMIT 1
	`
	err := tx.QueryRowContext(ctx, query, companyID).Scan(&lastNumber)
	if err != nil && err != sql.ErrNoRows {
		return "", fmt.Errorf("failed to query last invoice number: %w", err)
	}

	fmt.Printf("🔍 DEBUG generateInvoiceNumber:\n")
	fmt.Printf("  companyID: %s\n", companyID)
	fmt.Printf("  lastNumber.Valid: %v\n", lastNumber.Valid)
	if lastNumber.Valid {
		fmt.Printf("  lastNumber.String: %s\n", lastNumber.String)
	}

	var sequence int64 = 1
	if lastNumber.Valid {
		var year int
		n, err := fmt.Sscanf(lastNumber.String, "INV-%d-%d", &year, &sequence)
		fmt.Printf("  Sscanf result: n=%d, err=%v, year=%d, sequence=%d\n", n, err, year, sequence)

		if err != nil || n != 2 {
			parts := strings.Split(lastNumber.String, "-")
			if len(parts) == 3 {
				fmt.Sscanf(parts[2], "%d", &sequence)
			}
		}
		sequence++
	}

	currentYear := time.Now().Year()
	invoiceNumber := fmt.Sprintf("INV-%d-%05d", currentYear, sequence)
	fmt.Printf("  Generated: %s\n\n", invoiceNumber)

	return invoiceNumber, nil
}

// processLineItems processes all line items, snapshots data, and calculates totals
// processLineItems processes all line items, snapshots data, and calculates totals
func (s *InvoiceService) processLineItems(ctx context.Context, tx *sql.Tx, companyID string, reqItems []models.CreateInvoiceLineItemRequest) ([]models.InvoiceLineItem, float64, float64, float64, error) {
	var lineItems []models.InvoiceLineItem
	var subtotal, totalDiscount, totalTaxes float64

	for _, reqItem := range reqItems {
		// 1. Snapshot inventory item
		item, err := s.snapshotInventoryItem(ctx, tx, companyID, reqItem.ItemID)
		if err != nil {
			return nil, 0, 0, 0, err
		}

		// 2. Calculate line amounts with rounding
		lineSubtotal := round(item.UnitPrice * reqItem.Quantity)
		discountAmount := round(lineSubtotal * (reqItem.DiscountPercentage / 100))
		taxableAmount := round(lineSubtotal - discountAmount)

		// 3. Get taxes for this item
		taxes, lineTaxTotal, err := s.snapshotItemTaxes(ctx, tx, reqItem.ItemID, taxableAmount)
		if err != nil {
			return nil, 0, 0, 0, err
		}

		lineTotal := round(taxableAmount + lineTaxTotal)

		// 4. Create line item
		lineItem := models.InvoiceLineItem{
			ItemID:             &reqItem.ItemID,
			ItemSku:            item.SKU,
			ItemName:           item.Name,
			ItemDescription:    item.Description,
			ItemTipoItem:       item.TipoItem,
			UnitOfMeasure:      item.UnitOfMeasure,
			UnitPrice:          item.UnitPrice,
			Quantity:           reqItem.Quantity,
			LineSubtotal:       lineSubtotal,
			DiscountPercentage: reqItem.DiscountPercentage,
			DiscountAmount:     discountAmount,
			TaxableAmount:      taxableAmount,
			TotalTaxes:         lineTaxTotal,
			LineTotal:          lineTotal,
			Taxes:              taxes,
			CreatedAt:          time.Now(),
		}

		lineItems = append(lineItems, lineItem)

		// 5. Accumulate totals
		subtotal += lineSubtotal
		totalDiscount += discountAmount
		totalTaxes += lineTaxTotal
	}

	// Round final totals
	return lineItems, round(subtotal), round(totalDiscount), round(totalTaxes), nil
}

// ItemSnapshot represents the snapshot of inventory item at transaction time
type ItemSnapshot struct {
	SKU           string
	Name          string
	Description   *string
	TipoItem      string
	UnitOfMeasure string
	UnitPrice     float64
}

// snapshotInventoryItem retrieves and snapshots inventory item data
func (s *InvoiceService) snapshotInventoryItem(ctx context.Context, tx *sql.Tx, companyID, itemID string) (*ItemSnapshot, error) {
	query := `
		SELECT
			sku,
			name,
			description,
			tipo_item,
			unit_of_measure,
			unit_price
		FROM inventory_items
		WHERE id = $1 AND company_id = $2 AND active = true
	`

	var snapshot ItemSnapshot
	err := tx.QueryRowContext(ctx, query, itemID, companyID).Scan(
		&snapshot.SKU,
		&snapshot.Name,
		&snapshot.Description,
		&snapshot.TipoItem,
		&snapshot.UnitOfMeasure,
		&snapshot.UnitPrice,
	)

	if err == sql.ErrNoRows {
		return nil, ErrInventoryItemNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query inventory item: %w", err)
	}

	return &snapshot, nil
}

// satePercent / 100retrieves taxes for an item and calculates tax amounts
func (s *InvoiceService) snapshotItemTaxes(ctx context.Context, tx *sql.Tx, itemID string, taxableBase float64) ([]models.InvoiceLineItemTax, float64, error) {
	query := `
		SELECT tributo_code
		FROM inventory_item_taxes
		WHERE item_id = $1
	`

	rows, err := tx.QueryContext(ctx, query, itemID)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query item taxes: %w", err)
	}
	defer rows.Close()

	var taxes []models.InvoiceLineItemTax
	var totalTax float64

	for rows.Next() {
		var tributoCode string

		if err := rows.Scan(&tributoCode); err != nil {
			return nil, 0, fmt.Errorf("failed to scan tax: %w", err)
		}

		// Get tax name from Go codigos package
		tributoName, exists := codigos.GetTributoName(tributoCode)
		if !exists {
			return nil, 0, fmt.Errorf("invalid tributo code: %s", tributoCode)
		}

		// For now, extract percentage from IVA tax code or default to 0
		// You can extend this with a proper tax rate lookup
		var taxRatePercent float64
		if tributoCode == codigos.TributoIVA13 {
			taxRatePercent = 13.00
		} else if tributoCode == codigos.TributoIVAExportaciones {
			taxRatePercent = 0.00
		} else {
			// Add other tax rates as needed
			taxRatePercent = 0.00
		}

		// Convert percentage to decimal (13% -> 0.13)
		taxRate := taxRatePercent / 100
		taxAmount := round(taxableBase * taxRate)

		tax := models.InvoiceLineItemTax{
			TributoCode: tributoCode,
			TributoName: tributoName,
			TaxRate:     taxRate,
			TaxableBase: taxableBase,
			TaxAmount:   taxAmount,
			CreatedAt:   time.Now(),
		}

		taxes = append(taxes, tax)
		totalTax += taxAmount
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating taxes: %w", err)
	}

	return taxes, totalTax, nil
}

// insertLineItem inserts a line item and returns the ID
func (s *InvoiceService) insertLineItemTax(ctx context.Context, tx *sql.Tx, tax *models.InvoiceLineItemTax) (string, error) {
	query := `
		INSERT INTO invoice_line_item_taxes (
			line_item_id, tributo_code, tributo_name,
			tax_rate, taxable_base, tax_amount,
			created_at
		) VALUES (
			$1, $2, $3,
			$4, $5, $6,
			$7
		) RETURNING id
	`

	var id string
	err := tx.QueryRowContext(ctx, query,
		tax.LineItemID, tax.TributoCode, tax.TributoName,
		tax.TaxRate, tax.TaxableBase, tax.TaxAmount,
		tax.CreatedAt,
	).Scan(&id)

	return id, err
}

func (s *InvoiceService) insertLineItem(ctx context.Context, tx *sql.Tx, lineItem *models.InvoiceLineItem) (string, error) {
	query := `
		INSERT INTO invoice_line_items (
			invoice_id, line_number, item_id,
			item_sku, item_name, item_description, item_tipo_item, unit_of_measure,
			unit_price, quantity, line_subtotal,
			discount_percentage, discount_amount,
			taxable_amount, total_taxes, line_total,
			created_at
		) VALUES (
			$1, $2, $3,
			$4, $5, $6, $7, $8,
			$9, $10, $11,
			$12, $13,
			$14, $15, $16,
			$17
		) RETURNING id
	`

	var id string
	err := tx.QueryRowContext(ctx, query,
		lineItem.InvoiceID, lineItem.LineNumber, lineItem.ItemID,
		lineItem.ItemSku, lineItem.ItemName, lineItem.ItemDescription, lineItem.ItemTipoItem, lineItem.UnitOfMeasure,
		lineItem.UnitPrice, lineItem.Quantity, lineItem.LineSubtotal,
		lineItem.DiscountPercentage, lineItem.DiscountAmount,
		lineItem.TaxableAmount, lineItem.TotalTaxes, lineItem.LineTotal,
		lineItem.CreatedAt,
	).Scan(&id)

	if err != nil {
		return "", err
	}

	return id, nil
}

// GetInvoice retrieves a complete invoice with line items and taxes
func (s *InvoiceService) GetInvoice(ctx context.Context, companyID, invoiceID string) (*models.Invoice, error) {
	// Get invoice header
	invoice, err := s.getInvoiceHeader(ctx, companyID, invoiceID)
	if err != nil {
		return nil, err
	}

	// Get line items
	lineItems, err := s.getLineItems(ctx, invoiceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get line items: %w", err)
	}

	// Get taxes for each line item
	for i := range lineItems {
		taxes, err := s.getLineItemTaxes(ctx, lineItems[i].ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get taxes for line item: %w", err)
		}
		lineItems[i].Taxes = taxes
	}

	invoice.LineItems = lineItems

	return invoice, nil
}

// getLineItems retrieves all line items for an invoice
func (s *InvoiceService) getLineItems(ctx context.Context, invoiceID string) ([]models.InvoiceLineItem, error) {
	query := `
		SELECT
			id, invoice_id, line_number, item_id,
			item_sku, item_name, item_description, item_tipo_item, unit_of_measure,
			unit_price, quantity, line_subtotal,
			discount_percentage, discount_amount,
			taxable_amount, total_taxes, line_total,
			created_at
		FROM invoice_line_items
		WHERE invoice_id = $1
		ORDER BY line_number
	`

	rows, err := database.DB.QueryContext(ctx, query, invoiceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lineItems []models.InvoiceLineItem
	for rows.Next() {
		var item models.InvoiceLineItem
		err := rows.Scan(
			&item.ID, &item.InvoiceID, &item.LineNumber, &item.ItemID,
			&item.ItemSku, &item.ItemName, &item.ItemDescription, &item.ItemTipoItem, &item.UnitOfMeasure,
			&item.UnitPrice, &item.Quantity, &item.LineSubtotal,
			&item.DiscountPercentage, &item.DiscountAmount,
			&item.TaxableAmount, &item.TotalTaxes, &item.LineTotal,
			&item.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		lineItems = append(lineItems, item)
	}

	return lineItems, rows.Err()
}

// getLineItemTaxes retrieves all taxes for a line item
func (s *InvoiceService) getLineItemTaxes(ctx context.Context, lineItemID string) ([]models.InvoiceLineItemTax, error) {
	query := `
		SELECT
			id, line_item_id, tributo_code, tributo_name,
			tax_rate, taxable_base, tax_amount,
			created_at
		FROM invoice_line_item_taxes
		WHERE line_item_id = $1
	`

	rows, err := database.DB.QueryContext(ctx, query, lineItemID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var taxes []models.InvoiceLineItemTax
	for rows.Next() {
		var tax models.InvoiceLineItemTax
		err := rows.Scan(
			&tax.ID, &tax.LineItemID, &tax.TributoCode, &tax.TributoName,
			&tax.TaxRate, &tax.TaxableBase, &tax.TaxAmount,
			&tax.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		taxes = append(taxes, tax)
	}

	return taxes, rows.Err()
}

// DeleteDraftInvoice deletes a draft invoice (only drafts can be deleted)
func (s *InvoiceService) DeleteDraftInvoice(ctx context.Context, companyID, invoiceID string) error {
	// Verify it's a draft
	invoice, err := s.getInvoiceHeader(ctx, companyID, invoiceID)
	if err != nil {
		return err
	}

	if invoice.Status != "draft" {
		return ErrInvoiceNotDraft
	}

	// Delete (cascade will handle line items and taxes)
	query := `DELETE FROM invoices WHERE id = $1 AND company_id = $2`
	result, err := database.DB.ExecContext(ctx, query, invoiceID, companyID)
	if err != nil {
		return fmt.Errorf("failed to delete invoice: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrInvoiceNotFound
	}

	return nil
}

func round(val float64) float64 {
	return math.Round(val*100) / 100
}

// / new
// Update insertInvoice to include establishment_id
func (s *InvoiceService) insertInvoice(ctx context.Context, tx *sql.Tx, invoice *models.Invoice) (string, error) {
	id := strings.ToUpper(uuid.New().String())
	query := `
		INSERT INTO invoices (
            id,
			company_id, establishment_id, point_of_sale_id, client_id, invoice_number, invoice_type,
			client_name, client_legal_name, client_nit, client_ncr, client_dui,
			client_address, client_tipo_contribuyente, client_tipo_persona,
			subtotal, total_discount, total_taxes, total,
			currency, payment_terms, payment_method, payment_status, amount_paid, balance_due, due_date, dte_codigo_generacion,
			status, notes, contact_email, contact_whatsapp, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6,
			$7, $8, $9, $10, $11,
			$12, $13, $14,
			$15, $16, $17, $18,
			$19, $20, $21, $22, $23, $24,
			$25, $26, $27, $28, $29, $30, $31, $32
		) RETURNING id
	`

	_, err := tx.ExecContext(ctx, query,
		id,
		invoice.CompanyID, invoice.EstablishmentID, invoice.PointOfSaleID, invoice.ClientID, invoice.InvoiceNumber, invoice.InvoiceType,
		invoice.ClientName, invoice.ClientLegalName, invoice.ClientNit, invoice.ClientNcr, invoice.ClientDui,
		invoice.ClientAddress, invoice.ClientTipoContribuyente, invoice.ClientTipoPersona,
		invoice.Subtotal, invoice.TotalDiscount, invoice.TotalTaxes, invoice.Total,
		invoice.Currency, invoice.PaymentTerms, invoice.PaymentMethod, invoice.PaymentStatus, invoice.AmountPaid, invoice.BalanceDue, invoice.DueDate, id,
		invoice.Status, invoice.Notes, invoice.ContactEmail, invoice.ContactWhatsapp, invoice.CreatedAt,
	)

	if err != nil {
		return "", err
	}

	return id, nil
}

// Update getInvoiceHeader to include establishment_id
func (s *InvoiceService) getInvoiceHeader(ctx context.Context, companyID, invoiceID string) (*models.Invoice, error) {
	query := `
		SELECT
			id, company_id, establishment_id, point_of_sale_id, client_id,
			invoice_number, invoice_type,
			references_invoice_id, void_reason,
			client_name, client_legal_name, client_nit, client_ncr, client_dui,
			client_address, client_tipo_contribuyente, client_tipo_persona,
			subtotal, total_discount, total_taxes, total,
			currency,
			payment_terms, payment_method, payment_status, amount_paid, balance_due, due_date,
			status,
			dte_numero_control, dte_status, dte_hacienda_response, dte_submitted_at, dte_type,
			created_at, finalized_at, voided_at,
			created_by, voided_by, notes,
			contact_email, contact_whatsapp
		FROM invoices
		WHERE id = $1 AND company_id = $2
	`

	invoice := &models.Invoice{}
	err := database.DB.QueryRowContext(ctx, query, invoiceID, companyID).Scan(
		&invoice.ID, &invoice.CompanyID, &invoice.EstablishmentID, &invoice.PointOfSaleID, &invoice.ClientID,
		&invoice.InvoiceNumber, &invoice.InvoiceType,
		&invoice.ReferencesInvoiceID, &invoice.VoidReason,
		&invoice.ClientName, &invoice.ClientLegalName, &invoice.ClientNit, &invoice.ClientNcr, &invoice.ClientDui,
		&invoice.ClientAddress, &invoice.ClientTipoContribuyente, &invoice.ClientTipoPersona,
		&invoice.Subtotal, &invoice.TotalDiscount, &invoice.TotalTaxes, &invoice.Total,
		&invoice.Currency,
		&invoice.PaymentTerms, &invoice.PaymentMethod, &invoice.PaymentStatus, &invoice.AmountPaid, &invoice.BalanceDue, &invoice.DueDate,
		&invoice.Status,
		&invoice.DteNumeroControl, &invoice.DteStatus, &invoice.DteHaciendaResponse, &invoice.DteSubmittedAt, &invoice.DteType, // ⭐ Removed DteCodigoGeneracion
		&invoice.CreatedAt, &invoice.FinalizedAt, &invoice.VoidedAt,
		&invoice.CreatedBy, &invoice.VoidedBy, &invoice.Notes,
		&invoice.ContactEmail, &invoice.ContactWhatsapp,
	)
	if err == sql.ErrNoRows {
		return nil, ErrInvoiceNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query invoice: %w", err)
	}

	return invoice, nil
}

// Update ListInvoices to include establishment_id
func (s *InvoiceService) ListInvoices(ctx context.Context, companyID string, filters map[string]interface{}) ([]models.Invoice, error) {
	query := `
		SELECT
			id, company_id, establishment_id, point_of_sale_id, client_id,
			invoice_number, invoice_type,
			client_name, client_legal_name,
			subtotal, total_discount, total_taxes, total,
			payment_terms, payment_status, amount_paid, balance_due, due_date,
			status,
			dte_status, dte_numero_control, dte_type, dte_codigo_generacion,
			created_at, finalized_at,
			notes
		FROM invoices
		WHERE company_id = $1
	`

	args := []interface{}{companyID}
	argCount := 1

	// Add filters
	if status, ok := filters["status"].(string); ok && status != "" {
		argCount++
		query += fmt.Sprintf(" AND status = $%d", argCount)
		args = append(args, status)
	}

	if clientID, ok := filters["client_id"].(string); ok && clientID != "" {
		argCount++
		query += fmt.Sprintf(" AND client_id = $%d", argCount)
		args = append(args, clientID)
	}

	if paymentStatus, ok := filters["payment_status"].(string); ok && paymentStatus != "" {
		argCount++
		query += fmt.Sprintf(" AND payment_status = $%d", argCount)
		args = append(args, paymentStatus)
	}

	if establishmentID, ok := filters["establishment_id"].(string); ok && establishmentID != "" {
		argCount++
		query += fmt.Sprintf(" AND establishment_id = $%d", argCount)
		args = append(args, establishmentID)
	}

	if posID, ok := filters["point_of_sale_id"].(string); ok && posID != "" {
		argCount++
		query += fmt.Sprintf(" AND point_of_sale_id = $%d", argCount)
		args = append(args, posID)
	}

	// ADD: Filter by DTE status
	if dteStatus, ok := filters["dte_status"].(string); ok && dteStatus != "" {
		argCount++
		query += fmt.Sprintf(" AND dte_status = $%d", argCount)
		args = append(args, dteStatus)
	}

	query += " ORDER BY created_at DESC"

	rows, err := database.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
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
			&inv.PaymentTerms, &inv.PaymentStatus, &inv.AmountPaid, &inv.BalanceDue, &inv.DueDate,
			&inv.Status,
			&inv.DteStatus, &inv.DteNumeroControl, &inv.DteType, &inv.DteSello,
			&inv.CreatedAt, &inv.FinalizedAt,
			&inv.Notes,
		)
		if err != nil {
			return nil, err
		}
		invoices = append(invoices, inv)
	}

	return invoices, rows.Err()
}

func (s *InvoiceService) determineDTEType(tipoPersona string) string {
	if tipoPersona == codigos.PersonTypeJuridica {
		return codigos.DocTypeComprobanteCredito // CCF for businesses
	}
	return codigos.DocTypeFactura // Factura for individuals (default)
}

// generateNumeroControl generates the DTE numero control with strict validation
func (s *InvoiceService) generateNumeroControl(ctx context.Context, tx *sql.Tx, establishmentID, pointOfSaleID, posID, tipoDte string) (string, error) {
	// Get establishment and POS codes (no COALESCE - must be set)
	query := `
		SELECT 
			e.cod_establecimiento,
			pos.cod_punto_venta,
			e.nombre as establishment_name,
			pos.nombre as pos_name
		FROM point_of_sale pos
		JOIN establishments e ON pos.establishment_id = e.id
		WHERE pos.id = $1
	`

	var MHEstablishmentCode, MHPOSCode *string
	var establishmentName, posName string
	fmt.Println("these are the detaisl you sent the generate numero control")
	fmt.Println(establishmentID, pointOfSaleID)

	err := tx.QueryRowContext(ctx, query, posID).Scan(&MHEstablishmentCode, &MHPOSCode, &establishmentName, &posName)
	if err != nil {
		return "", fmt.Errorf("failed to get establishment codes: %w", err)
	}

	// Strict validation: MH codes must be set
	if MHEstablishmentCode == nil || *MHEstablishmentCode == "" {
		return "", fmt.Errorf("establishment '%s' must have cod_establecimiento assigned by Hacienda before finalizing invoices", establishmentName)
	}
	if MHPOSCode == nil || *MHPOSCode == "" {
		return "", fmt.Errorf("point of sale '%s' must have cod_punto_venta assigned by Hacienda before finalizing invoices", posName)
	}

	// Validate 4-character format
	if len(*MHEstablishmentCode) != 4 {
		return "", fmt.Errorf("establishment '%s' cod_establecimiento must be exactly 4 characters, got %d: '%s'",
			establishmentName, len(*MHEstablishmentCode), *MHEstablishmentCode)
	}
	if len(*MHPOSCode) != 4 {
		return "", fmt.Errorf("point of sale '%s' cod_punto_venta must be exactly 4 characters, got %d: '%s'",
			posName, len(*MHPOSCode), *MHPOSCode)
	}

	// Validate codes are alphanumeric (Hacienda uses numeric, but spec allows alphanumeric)
	if !s.isValidMHCode(*MHEstablishmentCode) {
		return "", fmt.Errorf("establishment '%s' cod_establecimiento contains invalid characters: '%s'",
			establishmentName, *MHEstablishmentCode)
	}
	if !s.isValidMHCode(*MHPOSCode) {
		return "", fmt.Errorf("point of sale '%s' cod_punto_venta contains invalid characters: '%s'",
			posName, *MHPOSCode)
	}

	// Get next sequence for this POS and tipoDte
	sequence, err := s.getAndIncrementDTESequence(ctx, tx, posID, tipoDte)
	if err != nil {
		return "", err
	}

	// Build numero control using the validator (ensures correctness)
	numeroControl, err := dte.BuildNumeroControl(tipoDte, *MHEstablishmentCode, *MHPOSCode, sequence)
	if err != nil {
		return "", fmt.Errorf("failed to build numero control: %w", err)
	}

	return numeroControl, nil
}

// isValidMHCode checks if an MH code contains only alphanumeric characters
func (s *InvoiceService) isValidMHCode(code string) bool {
	// Hacienda codes are typically numeric, but spec allows alphanumeric
	for _, char := range code {
		if !((char >= '0' && char <= '9') || (char >= 'A' && char <= 'Z')) {
			return false
		}
	}
	return true
}

func (s *InvoiceService) getAndIncrementDTESequence(ctx context.Context, tx *sql.Tx, posID, tipoDte string) (int64, error) {
	// Try to get existing sequence with row lock
	var currentSeq int64
	query := `
		SELECT last_sequence
		FROM dte_sequences
		WHERE point_of_sale_id = $1 AND tipo_dte = $2
		FOR UPDATE
	`

	err := tx.QueryRowContext(ctx, query, posID, tipoDte).Scan(&currentSeq)

	if err == sql.ErrNoRows {
		// First time - insert new sequence starting at 1
		insertQuery := `
			INSERT INTO dte_sequences (point_of_sale_id, tipo_dte, last_sequence, updated_at)
			VALUES ($1, $2, 1, $3)
		`
		_, err = tx.ExecContext(ctx, insertQuery, posID, tipoDte, time.Now())
		if err != nil {
			return 0, fmt.Errorf("failed to initialize sequence: %w", err)
		}
		return 1, nil
	}

	if err != nil {
		return 0, fmt.Errorf("failed to get sequence: %w", err)
	}

	// Increment sequence
	newSeq := currentSeq + 1
	updateQuery := `
		UPDATE dte_sequences
		SET last_sequence = $1, updated_at = $2
		WHERE point_of_sale_id = $3 AND tipo_dte = $4
	`

	_, err = tx.ExecContext(ctx, updateQuery, newSeq, time.Now(), posID, tipoDte)
	if err != nil {
		return 0, fmt.Errorf("failed to increment sequence: %w", err)
	}

	return newSeq, nil
}

func (s *InvoiceService) checkCreditLimit(ctx context.Context, tx *sql.Tx, clientID string, invoiceTotal float64) error {
	query := `
		SELECT credit_limit, current_balance, credit_status
		FROM clients
		WHERE id = $1
	`

	var creditLimit, currentBalance float64
	var creditStatus string

	err := tx.QueryRowContext(ctx, query, clientID).Scan(&creditLimit, &currentBalance, &creditStatus)
	if err != nil {
		return fmt.Errorf("failed to check credit limit: %w", err)
	}

	if creditStatus == "suspended" {
		return models.ErrCreditSuspended
	}

	newBalance := currentBalance + invoiceTotal
	if newBalance > creditLimit {
		return ErrCreditLimitExceeded
	}

	return nil
}

// updateClientBalance updates the client's current balance after finalization
func (s *InvoiceService) updateClientBalance(ctx context.Context, tx *sql.Tx, clientID string, amount float64) error {
	query := `
		UPDATE clients
		SET current_balance = current_balance + $1,
		    credit_status = CASE
		        WHEN current_balance + $1 > credit_limit THEN 'over_limit'
		        ELSE 'good_standing'
		    END
		WHERE id = $2
	`

	_, err := tx.ExecContext(ctx, query, amount, clientID)
	return err
}

// FinalizeInvoice finalizes a draft invoice and generates DTE identifiers
func (s *InvoiceService) FinalizeInvoice(ctx context.Context, companyID, invoiceID, userID string, payment *models.CreatePaymentRequest) (*models.Invoice, error) {
	// Begin transaction
	tx, err := database.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// 1. Get invoice and verify it's a draft (with row lock)
	invoice, err := s.getInvoiceForUpdate(ctx, tx, companyID, invoiceID)
	if err != nil {
		return nil, err
	}
	fmt.Println(invoice)

	if invoice.Status != "draft" {
		return nil, ErrInvoiceNotDraft
	}

	// 2. Check credit limit if credit transaction
	if invoice.PaymentTerms == "cuenta" || invoice.PaymentTerms == "net_30" || invoice.PaymentTerms == "net_60" {
		if err := s.checkCreditLimit(ctx, tx, invoice.ClientID, invoice.Total); err != nil {
			return nil, err
		}
	}

	// 3. Determine DTE type based on client tipo_persona
	var tipoDte string
	if invoice.ClientTipoPersona != nil {
		tipoDte = s.determineDTEType(*invoice.ClientTipoPersona)
	} else {
		tipoDte = "01" // Default to factura
	}

	// 4. Generate DTE identifiers
	numeroControl, err := s.generateNumeroControl(ctx, tx, invoice.EstablishmentID, invoice.PointOfSaleID, invoice.PointOfSaleID, tipoDte)
	fmt.Println("this is the numerocontrol I build", numeroControl)
	if err != nil {
		return nil, fmt.Errorf("failed to generate numero control: %w", err)
	}

	for _, lineItem := range invoice.LineItems {
		if lineItem.ItemID != nil {
			item, err := s.inventoryService.GetInventoryItemByID(ctx, *lineItem.ItemID, companyID)
			if err != nil {
				return nil, fmt.Errorf("no se pudo obtener el artículo del inventario: %w", err)
			}

			// Only deduct if item tracks inventory
			if item.TrackInventory {
				// Calculate tax info from line item
				taxExempt := lineItem.TotalTaxes == 0
				taxRate := 0.0
				if !taxExempt && lineItem.TaxableAmount > 0 {
					taxRate = lineItem.TotalTaxes / lineItem.TaxableAmount
				}

				// Calculate discount per unit
				discountPerUnit := models.Money(0)
				if lineItem.DiscountAmount > 0 && lineItem.Quantity > 0 {
					discountPerUnit = models.Money(lineItem.DiscountAmount / lineItem.Quantity)
				}

				// Calculate net unit price (price after discount, before tax)
				netUnitPrice := models.Money(lineItem.TaxableAmount / lineItem.Quantity)

				// Prepare sale request
				saleReq := &models.RecordSaleRequest{
					Quantity:      lineItem.Quantity,
					UnitSalePrice: models.Money(lineItem.UnitPrice),
					DiscountAmount: func() *models.Money {
						if discountPerUnit > 0 {
							return &discountPerUnit
						}
						return nil
					}(),
					NetUnitPrice:      netUnitPrice,
					TaxExempt:         taxExempt,
					TaxRate:           taxRate,
					TaxAmount:         models.Money(lineItem.TotalTaxes),
					DocumentType:      tipoDte,       // Use the determined DTE type
					DocumentNumber:    numeroControl, // Use the generated numero control
					InvoiceID:         invoice.ID,
					InvoiceLineID:     lineItem.ID,
					CustomerName:      invoice.ClientName,
					CustomerNIT:       invoice.ClientNit,
					CustomerTaxExempt: invoice.ClientTipoContribuyente != nil && *invoice.ClientTipoContribuyente == "02", // 02 = Consumidor Final
				}

				// Record sale (deducts inventory)
				// Note: RecordSale has its own transaction, but we're passing the parent context
				// The inventory deduction will be part of this transaction scope
				_, err = s.inventoryService.RecordSale(ctx, *lineItem.ItemID, companyID, saleReq)
				if err != nil {
					log.Printf("[ERROR] Failed to record sale for item %s: %v", *lineItem.ItemID, err)
					return nil, fmt.Errorf("no se pudo registrar la venta del artículo %s: %w", item.Name, err)
				}

				log.Printf("[DEBUG] Inventory deducted for item %s: %.2f units", item.Name, lineItem.Quantity)
			}
		}
	}

	// 5. Calculate payment status based on amount paid
	paymentStatus := s.calculatePaymentStatus(payment.Amount, invoice.Total)
	balanceDue := invoice.Total - payment.Amount

	// 6. Update invoice to finalized
	now := time.Now()

	fmt.Printf("DEBUG UPDATE VALUES:\n")
	fmt.Printf("  payment.PaymentMethod: %v\n", payment.PaymentMethod)
	fmt.Printf("  paymentStatus: %v\n", paymentStatus)
	fmt.Printf("  payment.Amount: %v\n", payment.Amount)
	fmt.Printf("  balanceDue: %v\n", balanceDue)
	fmt.Printf("  numeroControl: %v\n", numeroControl)
	fmt.Printf("  now: %v\n", now)
	fmt.Printf("  userID: %v\n", userID)
	fmt.Printf("  invoiceID: %v\n", invoiceID)
	fmt.Printf("  companyID: %v\n", companyID)
	updateQuery := `
		UPDATE invoices
		SET status = 'finalized',
		    payment_method = $1,
		    payment_status = $2,
		    amount_paid = $3,
		    balance_due = $4,
		    dte_numero_control = $5,
            dte_type = $6,
		    dte_status = 'not_submitted',
		    finalized_at = $7,
		    created_by = $8
		WHERE id = $9 AND company_id = $10
	`

	_, err = tx.ExecContext(ctx, updateQuery,
		payment.PaymentMethod,
		paymentStatus,
		payment.Amount,
		balanceDue,
		numeroControl,
		tipoDte,
		now,
		userID,
		invoiceID,
		companyID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update invoice: %w", err)
	}

	// 7. Record the payment in payments table
	paymentID := uuid.New().String()
	paymentDate := now
	if payment.PaymentDate != nil {
		paymentDate = *payment.PaymentDate
	}

	insertPaymentQuery := `
		INSERT INTO payments (
			id, company_id, invoice_id, amount, payment_method, 
			payment_reference, payment_date, created_by, notes
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err = tx.ExecContext(ctx, insertPaymentQuery,
		paymentID,
		companyID,
		invoiceID,
		payment.Amount,
		payment.PaymentMethod,
		payment.ReferenceNumber,
		paymentDate,
		userID,
		payment.Notes,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to record payment: %w", err)
	}

	// 8. Update client balance if credit
	if invoice.PaymentTerms == "cuenta" || invoice.PaymentTerms == "net_30" || invoice.PaymentTerms == "net_60" {
		// Add the balance due (not total) to client's balance
		if err := s.updateClientBalance(ctx, tx, invoice.ClientID, balanceDue); err != nil {
			return nil, fmt.Errorf("failed to update client balance: %w", err)
		}
	}

	// 9. Commit transaction
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// 10. Get and return the finalized invoice
	finalizedInvoice, err := s.GetInvoice(ctx, companyID, invoiceID)
	if err != nil {
		return nil, err
	}

	return finalizedInvoice, nil
}

// Helper: Calculate payment status based on amount paid
func (s *InvoiceService) calculatePaymentStatus(amountPaid, total float64) string {
	if amountPaid == 0 {
		return "unpaid"
	} else if amountPaid >= total {
		return "paid"
	}
	return "partial"
}

// getInvoiceForUpdate gets an invoice with a row lock for safe concurrent updates
func (s *InvoiceService) getInvoiceForUpdate(ctx context.Context, tx *sql.Tx, companyID, invoiceID string) (*models.Invoice, error) {
	query := `
		SELECT
			id, company_id, establishment_id, point_of_sale_id, client_id,
			invoice_number, invoice_type, status,
			client_name, client_legal_name, client_nit, client_ncr, client_dui,
			client_address, client_tipo_contribuyente, client_tipo_persona,
			subtotal, total_discount, total_taxes, total,
			currency, payment_terms, payment_method, payment_status, 
			amount_paid, balance_due, due_date,
			created_at
		FROM invoices
		WHERE id = $1 AND company_id = $2
		FOR UPDATE
	`

	invoice := &models.Invoice{}
	err := tx.QueryRowContext(ctx, query, invoiceID, companyID).Scan(
		&invoice.ID, &invoice.CompanyID, &invoice.EstablishmentID, &invoice.PointOfSaleID, &invoice.ClientID,
		&invoice.InvoiceNumber, &invoice.InvoiceType, &invoice.Status,
		&invoice.ClientName, &invoice.ClientLegalName, &invoice.ClientNit, &invoice.ClientNcr, &invoice.ClientDui,
		&invoice.ClientAddress, &invoice.ClientTipoContribuyente, &invoice.ClientTipoPersona,
		&invoice.Subtotal, &invoice.TotalDiscount, &invoice.TotalTaxes, &invoice.Total,
		&invoice.Currency, &invoice.PaymentTerms, &invoice.PaymentMethod, &invoice.PaymentStatus,
		&invoice.AmountPaid, &invoice.BalanceDue, &invoice.DueDate,
		&invoice.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrInvoiceNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query invoice: %w", err)
	}

	return invoice, nil
}
