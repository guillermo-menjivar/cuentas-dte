package services

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"cuentas/internal/models"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// NotaCreditoService handles business logic for Notas de Crédito
type NotaCreditoService struct {
	db *sqlx.DB
}

// NewNotaCreditoService creates a new NotaCreditoService
func NewNotaCreditoService(db *sqlx.DB) *NotaCreditoService {
	return &NotaCreditoService{
		db: db,
	}
}

// ============================================================================
// VALIDATION
// ============================================================================

// ValidateCreditRequest validates the entire credit request against business rules
func (s *NotaCreditoService) ValidateCreditRequest(
	ctx context.Context,
	ccfs []models.Invoice,
	request models.CreateNotaCreditoRequest,
) error {
	// Rule 1: All CCFs must be finalized
	for _, ccf := range ccfs {
		if ccf.Status != "finalized" {
			return fmt.Errorf("CCF %s is not finalized (status: %s)", ccf.InvoiceNumber, ccf.Status)
		}

		if ccf.DteSelloRecibido == nil || *ccf.DteSelloRecibido == "" {
			return fmt.Errorf("CCF %s has not been accepted by Hacienda", ccf.InvoiceNumber)
		}
	}

	// Rule 2: All CCFs must belong to same client
	if len(ccfs) > 0 {
		firstClientID := ccfs[0].ClientID
		for _, ccf := range ccfs {
			if ccf.ClientID != firstClientID {
				return fmt.Errorf("all CCFs must belong to the same client")
			}
		}
	}

	// Rule 3: Validate each line item
	for i, lineItem := range request.LineItems {
		if err := s.ValidateCreditLineItem(ctx, lineItem, ccfs); err != nil {
			return fmt.Errorf("line item %d: %w", i+1, err)
		}
	}

	// Rule 4: Check for over-crediting
	if err := s.CheckForExistingCredits(ctx, request.LineItems); err != nil {
		return err
	}

	return nil
}

// ValidateCreditLineItem validates a single line item credit
func (s *NotaCreditoService) ValidateCreditLineItem(
	ctx context.Context,
	lineItem models.CreateNotaCreditoLineItemRequest,
	ccfs []models.Invoice,
) error {
	// Find the CCF this line item belongs to
	var targetCCF *models.Invoice
	for i := range ccfs {
		if ccfs[i].ID == lineItem.RelatedCCFId {
			targetCCF = &ccfs[i]
			break
		}
	}

	if targetCCF == nil {
		return fmt.Errorf("CCF %s not found in request CCF list", lineItem.RelatedCCFId)
	}

	// Load the original line item from the CCF
	originalLineItem, err := s.getInvoiceLineItem(ctx, lineItem.CCFLineItemId)
	if err != nil {
		return fmt.Errorf("failed to load CCF line item: %w", err)
	}

	// Verify line item belongs to the specified CCF
	if originalLineItem.InvoiceID != lineItem.RelatedCCFId {
		return fmt.Errorf("line item %s does not belong to CCF %s",
			lineItem.CCFLineItemId, lineItem.RelatedCCFId)
	}

	// Validate quantity
	if lineItem.QuantityCredited > originalLineItem.Quantity {
		return fmt.Errorf("quantity_credited (%.8f) exceeds original quantity (%.8f)",
			lineItem.QuantityCredited, originalLineItem.Quantity)
	}

	// Validate credit amount (usually should not exceed original price, but allow for flexibility)
	// Note: Some businesses may credit MORE than original price as compensation
	// So we just warn if credit_amount > original_unit_price but don't fail
	if lineItem.CreditAmount > originalLineItem.UnitPrice {
		// This is allowed but unusual - business might want to log this
		// For now, we allow it
	}

	return nil
}

// CheckForExistingCredits ensures we're not over-crediting line items
func (s *NotaCreditoService) CheckForExistingCredits(
	ctx context.Context,
	lineItems []models.CreateNotaCreditoLineItemRequest,
) error {
	// For each line item, check how much has already been credited
	for _, lineItem := range lineItems {
		// Get original line item
		originalLineItem, err := s.getInvoiceLineItem(ctx, lineItem.CCFLineItemId)
		if err != nil {
			return err
		}

		// Get total already credited
		var totalCredited float64
		err = s.db.GetContext(ctx, &totalCredited, `
			SELECT COALESCE(SUM(ncli.quantity_credited), 0)
			FROM notas_credito_line_items ncli
			JOIN notas_credito nc ON nc.id = ncli.nota_credito_id
			WHERE ncli.ccf_line_item_id = $1
			  AND nc.status IN ('draft', 'finalized')
		`, lineItem.CCFLineItemId)

		if err != nil && err != sql.ErrNoRows {
			return fmt.Errorf("failed to check existing credits: %w", err)
		}

		// Check if new credit would exceed original quantity
		totalAfterCredit := totalCredited + lineItem.QuantityCredited
		if totalAfterCredit > originalLineItem.Quantity {
			return fmt.Errorf(
				"line item %s: total credited quantity (%.8f + %.8f = %.8f) would exceed original quantity (%.8f)",
				originalLineItem.ItemName,
				totalCredited,
				lineItem.QuantityCredited,
				totalAfterCredit,
				originalLineItem.Quantity,
			)
		}
	}

	return nil
}

// ============================================================================
// CALCULATION
// ============================================================================

// CalculateCreditTotals calculates financial totals for the credit note
// Returns: calculated line items and grand totals
func (s *NotaCreditoService) CalculateCreditTotals(
	ctx context.Context,
	request models.CreateNotaCreditoRequest,
	ccfs []models.Invoice,
) ([]models.NotaCreditoLineItem, CreditTotals, error) {
	lineItems := make([]models.NotaCreditoLineItem, 0, len(request.LineItems))
	totals := CreditTotals{}

	for i, reqLineItem := range request.LineItems {
		// Load original line item
		originalLineItem, err := s.getInvoiceLineItem(ctx, reqLineItem.CCFLineItemId)
		if err != nil {
			return nil, totals, fmt.Errorf("line %d: failed to load original: %w", i+1, err)
		}

		// Find the CCF this belongs to
		var ccfNumber string
		for _, ccf := range ccfs {
			if ccf.ID == reqLineItem.RelatedCCFId {
				ccfNumber = ccf.InvoiceNumber
				break
			}
		}

		// Calculate using CCF calculator (REUSE existing calculator!)
		// This ensures credit calculations match original invoice calculations
		calc := CalculateCreditoFiscal(
			reqLineItem.QuantityCredited,
			reqLineItem.CreditAmount,
			0, // no discount on credits typically
		)

		// Build line item
		lineItem := models.NotaCreditoLineItem{
			ID:               uuid.New().String(),
			LineNumber:       i + 1,
			RelatedCCFId:     reqLineItem.RelatedCCFId,
			RelatedCCFNumber: ccfNumber,
			CCFLineItemId:    reqLineItem.CCFLineItemId,

			// Original item snapshot
			OriginalItemSku:       originalLineItem.ItemSku,
			OriginalItemName:      originalLineItem.ItemName,
			OriginalUnitPrice:     originalLineItem.UnitPrice,
			OriginalQuantity:      originalLineItem.Quantity,
			OriginalItemTipoItem:  originalLineItem.ItemTipoItem,
			OriginalUnitOfMeasure: originalLineItem.UnitOfMeasure,

			// Credit details
			QuantityCredited: reqLineItem.QuantityCredited,
			CreditAmount:     reqLineItem.CreditAmount,

			// Calculated totals
			LineSubtotal:   calc.VentaGravada,
			DiscountAmount: 0,
			TaxableAmount:  calc.VentaGravada,
			TotalTaxes:     calc.IvaItem,
			LineTotal:      calc.VentaGravada + calc.IvaItem,

			CreatedAt: time.Now(),
		}

		if reqLineItem.CreditReason != "" {
			lineItem.CreditReason = &reqLineItem.CreditReason
		}

		lineItems = append(lineItems, lineItem)

		// Accumulate totals
		totals.Subtotal += lineItem.LineSubtotal
		totals.TotalDiscount += lineItem.DiscountAmount
		totals.TotalTaxes += lineItem.TotalTaxes
		totals.Total += lineItem.LineTotal
	}

	return lineItems, totals, nil
}

// CreditTotals holds the financial totals for a credit note
type CreditTotals struct {
	Subtotal      float64
	TotalDiscount float64
	TotalTaxes    float64
	Total         float64
}

// IsFullAnnulment determines if this credit note voids 100% of all referenced CCFs
func (s *NotaCreditoService) IsFullAnnulment(
	ctx context.Context,
	lineItems []models.NotaCreditoLineItem,
	ccfs []models.Invoice,
) (bool, error) {
	// For each CCF, check if ALL its line items are being credited at 100%
	for _, ccf := range ccfs {
		// Get all line items for this CCF
		ccfLineItems, err := s.getInvoiceLineItems(ctx, ccf.ID)
		if err != nil {
			return false, err
		}

		// Check each CCF line item
		for _, ccfLine := range ccfLineItems {
			// Find corresponding credit line item
			var creditLine *models.NotaCreditoLineItem
			for i := range lineItems {
				if lineItems[i].CCFLineItemId == ccfLine.ID {
					creditLine = &lineItems[i]
					break
				}
			}

			// If any line item is NOT being credited, this is not a full annulment
			if creditLine == nil {
				return false, nil
			}

			// If not crediting full quantity, this is not a full annulment
			if creditLine.QuantityCredited != ccfLine.Quantity {
				return false, nil
			}

			// If not crediting full price, this is not a full annulment
			if creditLine.CreditAmount != ccfLine.UnitPrice {
				return false, nil
			}
		}
	}

	// All line items of all CCFs are being credited at 100%
	return true, nil
}

// ============================================================================
// DATABASE OPERATIONS
// ============================================================================

// CreateNotaCredito creates a new nota de credito in DRAFT status
func (s *NotaCreditoService) CreateNotaCredito(
	ctx context.Context,
	request models.CreateNotaCreditoRequest,
	userID string,
) (*models.NotaCredito, error) {
	// Load CCFs
	ccfs, err := s.loadCCFs(ctx, request.CCFIds)
	if err != nil {
		return nil, fmt.Errorf("failed to load CCFs: %w", err)
	}

	// Validate
	if err := s.ValidateCreditRequest(ctx, ccfs, request); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Calculate totals
	lineItems, totals, err := s.CalculateCreditTotals(ctx, request, ccfs)
	if err != nil {
		return nil, fmt.Errorf("calculation failed: %w", err)
	}

	// Check if full annulment
	isFullAnnulment, err := s.IsFullAnnulment(ctx, lineItems, ccfs)
	if err != nil {
		return nil, fmt.Errorf("failed to check full annulment: %w", err)
	}

	// Begin transaction
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Use first CCF for inherited values
	firstCCF := ccfs[0]

	// Generate nota number
	notaNumber, err := s.generateNotaNumber(ctx, tx, firstCCF.CompanyID,
		firstCCF.EstablishmentID, firstCCF.PointOfSaleID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate nota number: %w", err)
	}

	// Create nota credito record
	nota := &models.NotaCredito{
		ID:              uuid.New().String(),
		CompanyID:       firstCCF.CompanyID,
		EstablishmentID: firstCCF.EstablishmentID,
		PointOfSaleID:   firstCCF.PointOfSaleID,

		NotaNumber: notaNumber,
		NotaType:   "05",

		// Client info from CCF
		ClientID:                firstCCF.ClientID,
		ClientName:              firstCCF.ClientName,
		ClientLegalName:         firstCCF.ClientLegalName,
		ClientNit:               firstCCF.ClientNit,
		ClientNcr:               firstCCF.ClientNcr,
		ClientDui:               firstCCF.ClientDui,
		ContactEmail:            firstCCF.ContactEmail,
		ContactWhatsapp:         firstCCF.ContactWhatsapp,
		ClientAddress:           firstCCF.ClientAddress,
		ClientTipoContribuyente: firstCCF.ClientTipoContribuyente,
		ClientTipoPersona:       firstCCF.ClientTipoPersona,

		// Credit details
		CreditReason:    request.CreditReason,
		IsFullAnnulment: isFullAnnulment,

		// Financial totals
		Subtotal:      totals.Subtotal,
		TotalDiscount: totals.TotalDiscount,
		TotalTaxes:    totals.TotalTaxes,
		Total:         totals.Total,
		Currency:      "USD",

		// Payment info
		PaymentTerms:  request.PaymentTerms,
		PaymentMethod: firstCCF.PaymentMethod,

		Status:    models.NotaCreditoStatusDraft,
		CreatedAt: time.Now(),
	}

	if request.CreditDescription != "" {
		nota.CreditDescription = &request.CreditDescription
	}

	if request.Notes != "" {
		nota.Notes = &request.Notes
	}

	if userID != "" {
		nota.CreatedBy = &userID
	}

	// Insert nota
	if err := s.insertNotaCredito(ctx, tx, nota); err != nil {
		return nil, fmt.Errorf("failed to insert nota: %w", err)
	}

	// Insert line items
	for i := range lineItems {
		lineItems[i].NotaCreditoID = nota.ID
		if err := s.insertNotaCreditoLineItem(ctx, tx, &lineItems[i]); err != nil {
			return nil, fmt.Errorf("failed to insert line item %d: %w", i+1, err)
		}
	}

	// Insert CCF references
	for _, ccf := range ccfs {
		ref := models.NotaCreditoCCFReference{
			ID:            uuid.New().String(),
			NotaCreditoID: nota.ID,
			CCFId:         ccf.ID,
			CCFNumber:     ccf.InvoiceNumber,
			CCFDate:       ccf.InvoiceDate,
			CreatedAt:     time.Now(),
		}
		if err := s.insertNotaCreditoCCFReference(ctx, tx, &ref); err != nil {
			return nil, fmt.Errorf("failed to insert CCF reference: %w", err)
		}
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Load complete nota with relationships
	completeNota, err := s.GetNotaCreditoByID(ctx, nota.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to load created nota: %w", err)
	}

	return completeNota, nil
}

// FinalizeNotaCredito finalizes a nota de credito (ready for DTE submission)
func (s *NotaCreditoService) FinalizeNotaCredito(
	ctx context.Context,
	notaID string,
) error {
	// Load nota
	nota, err := s.GetNotaCreditoByID(ctx, notaID)
	if err != nil {
		return fmt.Errorf("nota not found: %w", err)
	}

	// Verify status
	if nota.Status != models.NotaCreditoStatusDraft {
		return fmt.Errorf("nota must be in draft status (current: %s)", nota.Status)
	}

	// Generate DTE numero control
	numeroControl, err := s.generateDTENumeroControl(ctx,
		nota.CompanyID, nota.EstablishmentID, nota.PointOfSaleID, "05")
	if err != nil {
		return fmt.Errorf("failed to generate numero control: %w", err)
	}

	// Update status
	now := time.Now()
	_, err = s.db.ExecContext(ctx, `
		UPDATE notas_credito
		SET status = $1,
		    finalized_at = $2,
		    dte_numero_control = $3
		WHERE id = $4
	`, models.NotaCreditoStatusFinalized, now, numeroControl, notaID)

	if err != nil {
		return fmt.Errorf("failed to update nota status: %w", err)
	}

	return nil
}

// GetNotaCreditoByID retrieves a nota with all relationships
func (s *NotaCreditoService) GetNotaCreditoByID(
	ctx context.Context,
	notaID string,
) (*models.NotaCredito, error) {
	var nota models.NotaCredito

	query := `
		SELECT 
			id, company_id, establishment_id, point_of_sale_id,
			nota_number, nota_type,
			client_id, client_name, client_legal_name,
			client_nit, client_ncr, client_dui,
			contact_email, contact_whatsapp, client_address,
			client_tipo_contribuyente, client_tipo_persona,
			credit_reason, credit_description, is_full_annulment,
			subtotal, total_discount, total_taxes, total, currency,
			payment_terms, payment_method, due_date,
			status,
			dte_numero_control, dte_codigo_generacion, dte_sello_recibido,
			dte_status, dte_hacienda_response, dte_submitted_at,
			created_at, finalized_at, voided_at,
			created_by, notes
		FROM notas_credito
		WHERE id = $1
	`

	err := s.db.GetContext(ctx, &nota, query, notaID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("nota not found")
		}
		return nil, err
	}

	// Load line items
	lineItems, err := s.getNotaCreditoLineItems(ctx, notaID)
	if err != nil {
		return nil, fmt.Errorf("failed to load line items: %w", err)
	}
	nota.LineItems = lineItems

	// Load CCF references
	ccfRefs, err := s.getNotaCreditoCCFReferences(ctx, notaID)
	if err != nil {
		return nil, fmt.Errorf("failed to load CCF references: %w", err)
	}
	nota.CCFReferences = ccfRefs

	return &nota, nil
}

// ============================================================================
// HELPER METHODS
// ============================================================================

func (s *NotaCreditoService) loadCCFs(
	ctx context.Context,
	ccfIds []string,
) ([]models.Invoice, error) {
	if len(ccfIds) == 0 {
		return nil, fmt.Errorf("no CCF IDs provided")
	}

	query := `
		SELECT 
			id, company_id, establishment_id, point_of_sale_id,
			invoice_number, invoice_date, tipo_dte,
			client_id, client_name, client_legal_name,
			client_nit, client_ncr, client_dui,
			contact_email, contact_whatsapp, client_address,
			client_tipo_contribuyente, client_tipo_persona,
			subtotal, total_discount, total_taxes, total, currency,
			payment_terms, payment_method,
			status, dte_sello_recibido
		FROM invoices
		WHERE id = ANY($1)
	`

	var ccfs []models.Invoice
	err := s.db.SelectContext(ctx, &ccfs, query, ccfIds)
	if err != nil {
		return nil, err
	}

	if len(ccfs) != len(ccfIds) {
		return nil, fmt.Errorf("some CCFs not found (expected %d, got %d)",
			len(ccfIds), len(ccfs))
	}

	return ccfs, nil
}

func (s *NotaCreditoService) getInvoiceLineItem(
	ctx context.Context,
	lineItemID string,
) (*models.InvoiceLineItem, error) {
	var item models.InvoiceLineItem

	query := `
		SELECT 
			id, invoice_id, line_number,
			item_sku, item_name, quantity, unit_price,
			item_tipo_item, unit_of_measure
		FROM invoice_line_items
		WHERE id = $1
	`

	err := s.db.GetContext(ctx, &item, query, lineItemID)
	if err != nil {
		return nil, err
	}

	return &item, nil
}

func (s *NotaCreditoService) getInvoiceLineItems(
	ctx context.Context,
	invoiceID string,
) ([]models.InvoiceLineItem, error) {
	var items []models.InvoiceLineItem

	query := `
		SELECT 
			id, invoice_id, line_number,
			item_sku, item_name, quantity, unit_price,
			item_tipo_item, unit_of_measure
		FROM invoice_line_items
		WHERE invoice_id = $1
		ORDER BY line_number
	`

	err := s.db.SelectContext(ctx, &items, query, invoiceID)
	if err != nil {
		return nil, err
	}

	return items, nil
}

func (s *NotaCreditoService) getNotaCreditoLineItems(
	ctx context.Context,
	notaID string,
) ([]models.NotaCreditoLineItem, error) {
	var items []models.NotaCreditoLineItem

	query := `
		SELECT 
			id, nota_credito_id, line_number,
			related_ccf_id, related_ccf_number, ccf_line_item_id,
			original_item_sku, original_item_name,
			original_unit_price, original_quantity,
			original_item_tipo_item, original_unit_of_measure,
			quantity_credited, credit_amount, credit_reason,
			line_subtotal, discount_amount, taxable_amount,
			total_taxes, line_total,
			created_at
		FROM notas_credito_line_items
		WHERE nota_credito_id = $1
		ORDER BY line_number
	`

	err := s.db.SelectContext(ctx, &items, query, notaID)
	if err != nil {
		return nil, err
	}

	return items, nil
}

func (s *NotaCreditoService) getNotaCreditoCCFReferences(
	ctx context.Context,
	notaID string,
) ([]models.NotaCreditoCCFReference, error) {
	var refs []models.NotaCreditoCCFReference

	query := `
		SELECT id, nota_credito_id, ccf_id, ccf_number, ccf_date, created_at
		FROM notas_credito_ccf_references
		WHERE nota_credito_id = $1
		ORDER BY ccf_date
	`

	err := s.db.SelectContext(ctx, &refs, query, notaID)
	if err != nil {
		return nil, err
	}

	return refs, nil
}

func (s *NotaCreditoService) insertNotaCredito(
	ctx context.Context,
	tx *sqlx.Tx,
	nota *models.NotaCredito,
) error {
	query := `
		INSERT INTO notas_credito (
			id, company_id, establishment_id, point_of_sale_id,
			nota_number, nota_type,
			client_id, client_name, client_legal_name,
			client_nit, client_ncr, client_dui,
			contact_email, contact_whatsapp, client_address,
			client_tipo_contribuyente, client_tipo_persona,
			credit_reason, credit_description, is_full_annulment,
			subtotal, total_discount, total_taxes, total, currency,
			payment_terms, payment_method,
			status, created_at, created_by, notes
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15,
			$16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26, $27, $28, $29, $30, $31
		)
	`

	_, err := tx.ExecContext(ctx, query,
		nota.ID, nota.CompanyID, nota.EstablishmentID, nota.PointOfSaleID,
		nota.NotaNumber, nota.NotaType,
		nota.ClientID, nota.ClientName, nota.ClientLegalName,
		nota.ClientNit, nota.ClientNcr, nota.ClientDui,
		nota.ContactEmail, nota.ContactWhatsapp, nota.ClientAddress,
		nota.ClientTipoContribuyente, nota.ClientTipoPersona,
		nota.CreditReason, nota.CreditDescription, nota.IsFullAnnulment,
		nota.Subtotal, nota.TotalDiscount, nota.TotalTaxes, nota.Total, nota.Currency,
		nota.PaymentTerms, nota.PaymentMethod,
		nota.Status, nota.CreatedAt, nota.CreatedBy, nota.Notes,
	)

	return err
}

func (s *NotaCreditoService) insertNotaCreditoLineItem(
	ctx context.Context,
	tx *sqlx.Tx,
	item *models.NotaCreditoLineItem,
) error {
	query := `
		INSERT INTO notas_credito_line_items (
			id, nota_credito_id, line_number,
			related_ccf_id, related_ccf_number, ccf_line_item_id,
			original_item_sku, original_item_name,
			original_unit_price, original_quantity,
			original_item_tipo_item, original_unit_of_measure,
			quantity_credited, credit_amount, credit_reason,
			line_subtotal, discount_amount, taxable_amount,
			total_taxes, line_total,
			created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15,
			$16, $17, $18, $19, $20, $21
		)
	`

	_, err := tx.ExecContext(ctx, query,
		item.ID, item.NotaCreditoID, item.LineNumber,
		item.RelatedCCFId, item.RelatedCCFNumber, item.CCFLineItemId,
		item.OriginalItemSku, item.OriginalItemName,
		item.OriginalUnitPrice, item.OriginalQuantity,
		item.OriginalItemTipoItem, item.OriginalUnitOfMeasure,
		item.QuantityCredited, item.CreditAmount, item.CreditReason,
		item.LineSubtotal, item.DiscountAmount, item.TaxableAmount,
		item.TotalTaxes, item.LineTotal,
		item.CreatedAt,
	)

	return err
}

func (s *NotaCreditoService) insertNotaCreditoCCFReference(
	ctx context.Context,
	tx *sqlx.Tx,
	ref *models.NotaCreditoCCFReference,
) error {
	query := `
		INSERT INTO notas_credito_ccf_references (
			id, nota_credito_id, ccf_id, ccf_number, ccf_date, created_at
		) VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := tx.ExecContext(ctx, query,
		ref.ID, ref.NotaCreditoID, ref.CCFId, ref.CCFNumber, ref.CCFDate, ref.CreatedAt,
	)

	return err
}

func (s *NotaCreditoService) generateNotaNumber(
	ctx context.Context,
	tx *sqlx.Tx,
	companyID, establishmentID, posID string,
) (string, error) {
	// Format: NC-00000001
	var maxNumber int
	query := `
		SELECT COALESCE(MAX(CAST(SUBSTRING(nota_number FROM 4) AS INTEGER)), 0)
		FROM notas_credito
		WHERE company_id = $1 
		  AND establishment_id = $2 
		  AND point_of_sale_id = $3
	`

	err := tx.GetContext(ctx, &maxNumber, query, companyID, establishmentID, posID)
	if err != nil && err != sql.ErrNoRows {
		return "", err
	}

	nextNumber := maxNumber + 1
	return fmt.Sprintf("NC-%08d", nextNumber), nil
}

func (s *NotaCreditoService) generateDTENumeroControl(
	ctx context.Context,
	companyID, establishmentID, posID, tipoDte string,
) (string, error) {
	// This should call the same generator used for invoices
	// Format: DTE-{tipoDte}-M{establishmentMH}P{posMH}-{sequence}
	// For now, simplified version:
	var maxSequence int
	query := `
		SELECT COALESCE(MAX(CAST(SUBSTRING(dte_numero_control FROM '.{3}-.{2}-.{10}-(.*)') AS INTEGER)), 0)
		FROM notas_credito
		WHERE company_id = $1 
		  AND establishment_id = $2 
		  AND point_of_sale_id = $3
	`

	err := s.db.GetContext(ctx, &maxSequence, query, companyID, establishmentID, posID)
	if err != nil && err != sql.ErrNoRows {
		return "", err
	}

	// Load establishment codes
	var estCode, posCode string
	err = s.db.GetContext(ctx, &estCode,
		"SELECT codigo_establecimiento_mh FROM establishments WHERE id = $1", establishmentID)
	if err != nil {
		return "", err
	}

	err = s.db.GetContext(ctx, &posCode,
		"SELECT codigo_punto_venta_mh FROM point_of_sale WHERE id = $1", posID)
	if err != nil {
		return "", err
	}

	nextSequence := maxSequence + 1
	return fmt.Sprintf("DTE-%s-M%sP%s-%015d", tipoDte, estCode, posCode, nextSequence), nil
}

// CalculateCreditoFiscal calculates CCF line item totals
// REUSED from existing calculator - ensures consistency with invoice calculations
type CreditoFiscalCalc struct {
	VentaGravada float64
	IvaItem      float64
}

func CalculateCreditoFiscal(quantity, unitPrice, discount float64) CreditoFiscalCalc {
	subtotal := quantity * unitPrice
	afterDiscount := subtotal - discount
	iva := afterDiscount * 0.13

	return CreditoFiscalCalc{
		VentaGravada: afterDiscount,
		IvaItem:      iva,
	}
}
