package services

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"time"

	"cuentas/internal/codigos"
	"cuentas/internal/database"
	"cuentas/internal/models"
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
	invoiceNumber, err := s.generateInvoiceNumber(ctx, tx, req.PointOfSaleID)
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
	// Get the last invoice number for this company
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

	// Parse sequence or start at 1
	var sequence int64 = 1
	if lastNumber.Valid {
		// Format: INV-2025-00001
		// Extract the sequence number (last part)
		var year int
		fmt.Sscanf(lastNumber.String, "INV-%d-%d", &year, &sequence)
		sequence++
	}

	// Generate new number
	currentYear := time.Now().Year()
	invoiceNumber := fmt.Sprintf("INV-%d-%05d", currentYear, sequence)

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
	query := `
		INSERT INTO invoices (
			company_id, establishment_id, point_of_sale_id, client_id, invoice_number, invoice_type,
			client_name, client_legal_name, client_nit, client_ncr, client_dui,
			client_address, client_tipo_contribuyente, client_tipo_persona,
			subtotal, total_discount, total_taxes, total,
			currency, payment_terms, payment_method, payment_status, amount_paid, balance_due, due_date,
			status, notes, contact_email, contact_whatsapp, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6,
			$7, $8, $9, $10, $11,
			$12, $13, $14,
			$15, $16, $17, $18,
			$19, $20, $21, $22, $23, $24,
			$25, $26, $27, $28, $29, $30
		) RETURNING id
	`

	var id string
	err := tx.QueryRowContext(ctx, query,
		invoice.CompanyID, invoice.EstablishmentID, invoice.PointOfSaleID, invoice.ClientID, invoice.InvoiceNumber, invoice.InvoiceType,
		invoice.ClientName, invoice.ClientLegalName, invoice.ClientNit, invoice.ClientNcr, invoice.ClientDui,
		invoice.ClientAddress, invoice.ClientTipoContribuyente, invoice.ClientTipoPersona,
		invoice.Subtotal, invoice.TotalDiscount, invoice.TotalTaxes, invoice.Total,
		invoice.Currency, invoice.PaymentTerms, invoice.PaymentMethod, invoice.PaymentStatus, invoice.AmountPaid, invoice.BalanceDue, invoice.DueDate,
		invoice.Status, invoice.Notes, invoice.ContactEmail, invoice.ContactWhatsapp, invoice.CreatedAt,
	).Scan(&id)

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
			dte_codigo_generacion, dte_numero_control, dte_status, dte_hacienda_response, dte_submitted_at,
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
		&invoice.DteCodigoGeneracion, &invoice.DteNumeroControl, &invoice.DteStatus, &invoice.DteHaciendaResponse, &invoice.DteSubmittedAt,
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
			payment_terms, payment_method, payment_status, amount_paid, balance_due, due_date,
			status,
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

	// ADD: Filter by establishment
	if establishmentID, ok := filters["establishment_id"].(string); ok && establishmentID != "" {
		argCount++
		query += fmt.Sprintf(" AND establishment_id = $%d", argCount)
		args = append(args, establishmentID)
	}

	// ADD: Filter by POS
	if posID, ok := filters["point_of_sale_id"].(string); ok && posID != "" {
		argCount++
		query += fmt.Sprintf(" AND point_of_sale_id = $%d", argCount)
		args = append(args, posID)
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
			&inv.PaymentTerms, &inv.PaymentMethod, &inv.PaymentStatus, &inv.AmountPaid, &inv.BalanceDue, &inv.DueDate,
			&inv.Status,
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
