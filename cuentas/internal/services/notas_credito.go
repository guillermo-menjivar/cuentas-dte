package services

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"cuentas/internal/codigos"
	"cuentas/internal/database"
	"cuentas/internal/models"

	"github.com/google/uuid"
)

const (
	MaxCreditQuantityFactor = 1.0 // Cannot credit more than original quantity
)

type NotaCreditoService struct {
}

func NewNotaCreditoService() *NotaCreditoService {
	return &NotaCreditoService{}
}

// CreateNotaCredito creates a new Nota de Crédito
func (s *NotaCreditoService) CreateNotaCredito(
	ctx context.Context,
	companyID string,
	req *models.CreateNotaCreditoRequest,
	invoiceService *InvoiceService,
) (*models.NotaCredito, error) {

	fmt.Println("🚀 Starting Nota de Crédito creation...")

	// Step 1: Validate request
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Step 2: Validate and fetch all referenced CCFs
	ccfs, err := s.validateAndFetchCCFs(ctx, companyID, req.CCFIds, invoiceService)
	if err != nil {
		return nil, err
	}

	fmt.Printf("\n✅ Successfully validated %d CCF(s)\n", len(ccfs))

	// Step 3: Validate line items (credits to existing CCF items only)
	if err := s.validateLineItems(ctx, req.LineItems, ccfs); err != nil {
		return nil, err
	}

	fmt.Printf("\n✅ All line items validated\n")

	// Step 4: Calculate totals
	lineItems, totals, err := s.calculateTotals(ctx, req.LineItems, ccfs)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate totals: %w", err)
	}

	fmt.Printf("\n💰 Calculated Totals:\n")
	fmt.Printf("   Subtotal: $%.2f\n", totals.Subtotal)
	fmt.Printf("   Taxes: $%.2f\n", totals.TotalTaxes)
	fmt.Printf("   Total: $%.2f\n", totals.Total)

	// Step 5: Check if this is a full annulment
	isFullAnnulment := s.isFullAnnulment(lineItems, ccfs)
	if isFullAnnulment {
		fmt.Printf("\n⚠️  This is a FULL ANNULMENT (voiding 100%% of all CCFs)\n")
	}

	// Step 6: Create nota record in database
	nota, err := s.createNotaRecord(ctx, companyID, req, ccfs, lineItems, totals, isFullAnnulment)
	if err != nil {
		return nil, fmt.Errorf("failed to create nota record: %w", err)
	}

	fmt.Printf("\n✅ Nota de Crédito created successfully: %s\n", nota.NotaNumber)

	return nota, nil
}

// GetNotaCredito retrieves a nota by ID (public method for handler)
func (s *NotaCreditoService) GetNotaCredito(ctx context.Context, notaID, companyID string) (*models.NotaCredito, error) {
	return s.getNotaCredito(ctx, notaID, companyID)
}

// FinalizeNotaCredito finalizes a nota and prepares it for DTE submission
func (s *NotaCreditoService) FinalizeNotaCredito(
	ctx context.Context,
	notaID string,
	companyID string,
) (*models.NotaCredito, error) {

	fmt.Printf("🔄 Finalizing Nota de Crédito: %s\n", notaID)

	// Step 1: Fetch the nota (must be in draft status)
	nota, err := s.getNotaCredito(ctx, notaID, companyID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch nota: %w", err)
	}

	// Step 2: Validate nota is in draft status
	if nota.Status != "draft" {
		return nil, fmt.Errorf("nota is not in draft status (current status: %s)", nota.Status)
	}

	fmt.Printf("   Nota Number: %s\n", nota.NotaNumber)
	fmt.Printf("   Client: %s\n", nota.ClientName)
	fmt.Printf("   Credit Reason: %s\n", nota.CreditReason)
	fmt.Printf("   Total: $%.2f\n", nota.Total)

	// Step 3: Generate numero de control if not present
	if nota.DteNumeroControl == nil {
		numeroControl, err := s.generateNumeroControl(ctx, companyID, nota)
		if err != nil {
			return nil, fmt.Errorf("failed to generate numero control: %w", err)
		}
		nota.DteNumeroControl = &numeroControl

		// Save numero control to database
		err = s.saveNumeroControl(ctx, notaID, numeroControl)
		if err != nil {
			return nil, fmt.Errorf("failed to save numero control: %w", err)
		}

		fmt.Printf("   Generated Numero Control: %s\n", numeroControl)
	}

	// Step 4: Update nota status to finalized
	now := time.Now()
	err = s.updateNotaStatusToFinalized(ctx, notaID, now)
	if err != nil {
		return nil, fmt.Errorf("failed to finalize nota in database: %w", err)
	}

	// Step 5: Reload nota with updated data
	nota, err = s.getNotaCredito(ctx, notaID, companyID)
	if err != nil {
		return nil, fmt.Errorf("failed to reload nota: %w", err)
	}

	fmt.Printf("\n✅ Nota de Crédito finalized (ready for DTE submission)!\n")
	fmt.Printf("   Status: %s\n", nota.Status)

	return nota, nil
}

// validateAndFetchCCFs validates and fetches all referenced CCFs
func (s *NotaCreditoService) validateAndFetchCCFs(
	ctx context.Context,
	companyID string,
	ccfIDs []string,
	invoiceService *InvoiceService,
) ([]*models.Invoice, error) {

	if len(ccfIDs) == 0 {
		return nil, fmt.Errorf("at least one CCF ID is required")
	}

	if len(ccfIDs) > MaxCCFRequests {
		return nil, fmt.Errorf("maximum %d CCFs allowed per nota", MaxCCFRequests)
	}

	ccfs := make([]*models.Invoice, 0, len(ccfIDs))
	seenIDs := make(map[string]bool)

	fmt.Printf("🔍 Validating %d CCF(s)...\n", len(ccfIDs))

	for i, ccfID := range ccfIDs {
		// Check for duplicate IDs in request
		if seenIDs[ccfID] {
			return nil, fmt.Errorf("duplicate CCF ID found: %s", ccfID)
		}
		seenIDs[ccfID] = true

		fmt.Printf("  [%d/%d] Fetching CCF: %s\n", i+1, len(ccfIDs), ccfID)

		// Fetch the invoice
		invoice, err := invoiceService.GetInvoice(ctx, companyID, ccfID)
		if err != nil {
			if err == ErrInvoiceNotFound {
				return nil, fmt.Errorf("CCF not found: %s", ccfID)
			}
			return nil, fmt.Errorf("failed to fetch CCF %s: %w", ccfID, err)
		}

		// Validate it's a CCF (type "03")
		if *invoice.DteType != codigos.DocTypeComprobanteCredito {
			return nil, fmt.Errorf(
				"document %s is not a CCF (type: %s). Notas de Crédito can only reference CCF invoices (type 03)",
				ccfID,
				invoice.InvoiceType,
			)
		}

		// Validate CCF is finalized
		if invoice.Status != "finalized" {
			return nil, fmt.Errorf(
				"CCF %s is not finalized (status: %s). Can only create notas for finalized invoices",
				ccfID,
				invoice.Status,
			)
		}

		// Validate CCF has been accepted by Hacienda
		if invoice.DteSelloRecibido == nil || *invoice.DteSelloRecibido == "" {
			return nil, fmt.Errorf(
				"CCF %s has not been accepted by Hacienda (no sello recibido)",
				ccfID,
			)
		}

		// Validate CCF is not voided
		if invoice.VoidedAt != nil {
			return nil, fmt.Errorf(
				"CCF %s has been voided and cannot be credited",
				ccfID,
			)
		}

		fmt.Printf("    ✅ Valid CCF: %s (%s) - Total: $%.2f\n",
			invoice.InvoiceNumber,
			invoice.ClientName,
			invoice.Total,
		)

		ccfs = append(ccfs, invoice)
	}

	// All CCFs must belong to the same client
	if len(ccfs) > 0 {
		firstClientID := ccfs[0].ClientID
		firstClientName := ccfs[0].ClientName

		for _, ccf := range ccfs[1:] {
			if ccf.ClientID != firstClientID {
				return nil, fmt.Errorf(
					"all CCFs must belong to the same client. Found CCFs for '%s' and '%s'",
					firstClientName,
					ccf.ClientName,
				)
			}
		}

		fmt.Printf("✅ All CCFs belong to client: %s\n", firstClientName)
	}

	return ccfs, nil
}

// validateLineItems validates that all line items are valid credits to existing CCF items
func (s *NotaCreditoService) validateLineItems(
	ctx context.Context,
	lineItems []models.CreateNotaCreditoLineItemRequest,
	ccfs []*models.Invoice,
) error {

	fmt.Printf("\n🔍 Validating %d line item credit(s)...\n", len(lineItems))

	if len(lineItems) > MaxLineItems {
		return fmt.Errorf("maximum %d line items allowed per nota", MaxLineItems)
	}

	// Create a map for quick CCF lookup
	ccfMap := make(map[string]*models.Invoice)
	for _, ccf := range ccfs {
		ccfMap[ccf.ID] = ccf
	}

	// Track seen line item references to prevent duplicate credits
	seenRefs := make(map[string]bool)

	for i, item := range lineItems {
		fmt.Printf("  [%d/%d] Validating credit to line item in CCF %s\n",
			i+1, len(lineItems), item.RelatedCCFId)

		// Validate related_ccf_id exists in the ccf_ids array
		ccf, exists := ccfMap[item.RelatedCCFId]
		if !exists {
			return fmt.Errorf(
				"line item %d references CCF %s which is not in the ccf_ids list",
				i+1,
				item.RelatedCCFId,
			)
		}

		// Validate the credit
		if err := s.validateCreditLineItem(ctx, &item, ccf, i+1); err != nil {
			return err
		}

		// Check for existing credits to prevent over-crediting
		if err := s.checkExistingCredits(ctx, &item, ccf, i+1); err != nil {
			return err
		}

		// Check for duplicate credits in this request
		refKey := fmt.Sprintf("%s-%s", item.RelatedCCFId, item.CCFLineItemId)
		if seenRefs[refKey] {
			return fmt.Errorf(
				"duplicate credit to CCF line item %s in CCF %s",
				item.CCFLineItemId,
				item.RelatedCCFId,
			)
		}
		seenRefs[refKey] = true

		fmt.Printf("    ✅ Valid credit\n")
	}

	fmt.Printf("✅ All line items validated\n")
	return nil
}

// validateCreditLineItem validates a credit to an existing CCF line item
func (s *NotaCreditoService) validateCreditLineItem(
	ctx context.Context,
	item *models.CreateNotaCreditoLineItemRequest,
	ccf *models.Invoice,
	lineNumber int,
) error {

	fmt.Printf("      → Validating credit details\n")

	// Validate quantity_credited is positive
	if item.QuantityCredited <= 0 {
		return fmt.Errorf(
			"line item %d: quantity_credited must be positive (got %.8f)",
			lineNumber,
			item.QuantityCredited,
		)
	}

	// Validate credit_amount is not negative
	if item.CreditAmount < 0 {
		return fmt.Errorf(
			"line item %d: credit_amount cannot be negative (got %.2f)",
			lineNumber,
			item.CreditAmount,
		)
	}

	// Find the original line item in the CCF
	var originalLineItem *models.InvoiceLineItem
	for i := range ccf.LineItems {
		if ccf.LineItems[i].ID == item.CCFLineItemId {
			originalLineItem = &ccf.LineItems[i]
			break
		}
	}

	if originalLineItem == nil {
		return fmt.Errorf(
			"line item %d: CCF line item %s not found in CCF %s",
			lineNumber,
			item.CCFLineItemId,
			ccf.ID,
		)
	}

	fmt.Printf("      → Found original line item: %s (SKU: %s)\n",
		originalLineItem.ItemName,
		originalLineItem.ItemSku,
	)
	fmt.Printf("      → Original unit price: $%.2f\n", originalLineItem.UnitPrice)
	fmt.Printf("      → Original quantity: %.2f\n", originalLineItem.Quantity)
	fmt.Printf("      → Credit per unit: $%.2f\n", item.CreditAmount)
	fmt.Printf("      → Quantity credited: %.2f\n", item.QuantityCredited)
	fmt.Printf("      → Total credit: $%.2f\n", item.CreditAmount*item.QuantityCredited)

	// Validate quantity does not exceed original
	if item.QuantityCredited > originalLineItem.Quantity {
		return fmt.Errorf(
			"line item %d: quantity_credited (%.8f) exceeds original quantity (%.8f)",
			lineNumber,
			item.QuantityCredited,
			originalLineItem.Quantity,
		)
	}

	// Validate credit amount is reasonable (allow flexibility for compensation)
	// Note: We allow credit_amount > original_unit_price for compensation cases
	if item.CreditAmount > originalLineItem.UnitPrice*MaxAdjustmentFactor {
		return fmt.Errorf(
			"line item %d: credit amount ($%.2f) is suspiciously large compared to original price ($%.2f). "+
				"Please verify the amount is correct",
			lineNumber,
			item.CreditAmount,
			originalLineItem.UnitPrice,
		)
	}

	return nil
}

// checkExistingCredits ensures we're not over-crediting a line item
func (s *NotaCreditoService) checkExistingCredits(
	ctx context.Context,
	item *models.CreateNotaCreditoLineItemRequest,
	ccf *models.Invoice,
	lineNumber int,
) error {

	// Find original line item
	var originalLineItem *models.InvoiceLineItem
	for i := range ccf.LineItems {
		if ccf.LineItems[i].ID == item.CCFLineItemId {
			originalLineItem = &ccf.LineItems[i]
			break
		}
	}

	if originalLineItem == nil {
		return fmt.Errorf("line item not found")
	}

	// Get total already credited for this line item
	var totalCredited float64
	query := `
		SELECT COALESCE(SUM(ncli.quantity_credited), 0)
		FROM notas_credito_line_items ncli
		JOIN notas_credito nc ON nc.id = ncli.nota_credito_id
		WHERE ncli.ccf_line_item_id = $1
		  AND nc.status IN ('draft', 'finalized')
	`

	err := database.DB.QueryRowContext(ctx, query, item.CCFLineItemId).Scan(&totalCredited)
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("failed to check existing credits: %w", err)
	}

	// Check if new credit would exceed original quantity
	totalAfterCredit := totalCredited + item.QuantityCredited
	if totalAfterCredit > originalLineItem.Quantity {
		return fmt.Errorf(
			"line item %d: total credited quantity (%.8f existing + %.8f new = %.8f) would exceed original quantity (%.8f)",
			lineNumber,
			totalCredited,
			item.QuantityCredited,
			totalAfterCredit,
			originalLineItem.Quantity,
		)
	}

	if totalCredited > 0 {
		fmt.Printf("      → Existing credits: %.2f (%.2f remaining available)\n",
			totalCredited, originalLineItem.Quantity-totalCredited)
	}

	return nil
}

// CreditTotals holds calculated totals
type CreditTotals struct {
	Subtotal      float64
	TotalDiscount float64
	TotalTaxes    float64
	Total         float64
}

// calculateTotals calculates all totals for the nota
func (s *NotaCreditoService) calculateTotals(
	ctx context.Context,
	lineItems []models.CreateNotaCreditoLineItemRequest,
	ccfs []*models.Invoice,
) ([]models.NotaCreditoLineItem, *CreditTotals, error) {

	fmt.Println("\n🧮 Calculating totals...")

	// Create CCF map for lookups
	ccfMap := make(map[string]*models.Invoice)
	for _, ccf := range ccfs {
		ccfMap[ccf.ID] = ccf
	}

	calculatedLineItems := make([]models.NotaCreditoLineItem, 0, len(lineItems))
	totals := &CreditTotals{}

	for i, item := range lineItems {
		ccf := ccfMap[item.RelatedCCFId]

		// Find the original line item
		var originalLineItem *models.InvoiceLineItem
		for j := range ccf.LineItems {
			if ccf.LineItems[j].ID == item.CCFLineItemId {
				originalLineItem = &ccf.LineItems[j]
				break
			}
		}

		if originalLineItem == nil {
			return nil, nil, fmt.Errorf("line item not found: %s", item.CCFLineItemId)
		}

		// Calculate line totals using SAME logic as CCF
		// Subtotal = credit_amount × quantity_credited
		lineSubtotal := item.CreditAmount * item.QuantityCredited
		discountAmount := 0.0 // Notas typically don't have discounts
		taxableAmount := lineSubtotal - discountAmount

		// Calculate taxes (13% IVA for El Salvador)
		lineTaxes := taxableAmount * 0.13

		lineTotal := taxableAmount + lineTaxes

		// Create the calculated line item
		calculatedLine := models.NotaCreditoLineItem{
			ID:                    uuid.New().String(),
			LineNumber:            i + 1,
			RelatedCCFId:          item.RelatedCCFId,
			RelatedCCFNumber:      ccf.InvoiceNumber,
			CCFLineItemId:         item.CCFLineItemId,
			OriginalItemSku:       originalLineItem.ItemSku,
			OriginalItemName:      originalLineItem.ItemName,
			OriginalUnitPrice:     originalLineItem.UnitPrice,
			OriginalQuantity:      originalLineItem.Quantity,
			OriginalItemTipoItem:  originalLineItem.ItemTipoItem,
			OriginalUnitOfMeasure: originalLineItem.UnitOfMeasure,
			QuantityCredited:      item.QuantityCredited,
			CreditAmount:          item.CreditAmount,
			LineSubtotal:          lineSubtotal,
			DiscountAmount:        discountAmount,
			TaxableAmount:         taxableAmount,
			TotalTaxes:            lineTaxes,
			LineTotal:             lineTotal,
		}

		if item.CreditReason != "" {
			reason := item.CreditReason
			calculatedLine.CreditReason = &reason
		}

		calculatedLineItems = append(calculatedLineItems, calculatedLine)

		// Add to totals
		totals.Subtotal += lineSubtotal
		totals.TotalDiscount += discountAmount
		totals.TotalTaxes += lineTaxes
		totals.Total += lineTotal

		fmt.Printf("  Line %d: Subtotal $%.2f + Tax $%.2f = Total $%.2f\n",
			i+1, lineSubtotal, lineTaxes, lineTotal)
	}

	return calculatedLineItems, totals, nil
}

// isFullAnnulment determines if this credit note voids 100% of all referenced CCFs
func (s *NotaCreditoService) isFullAnnulment(
	lineItems []models.NotaCreditoLineItem,
	ccfs []*models.Invoice,
) bool {

	// For each CCF, check if ALL its line items are being credited at 100%
	for _, ccf := range ccfs {
		// Check each CCF line item
		for i := range ccf.LineItems {
			ccfLine := &ccf.LineItems[i]

			// Find corresponding credit line item
			var creditLine *models.NotaCreditoLineItem
			for j := range lineItems {
				if lineItems[j].CCFLineItemId == ccfLine.ID {
					creditLine = &lineItems[j]
					break
				}
			}

			// If any line item is NOT being credited, this is not a full annulment
			if creditLine == nil {
				return false
			}

			// If not crediting full quantity, this is not a full annulment
			if creditLine.QuantityCredited != ccfLine.Quantity {
				return false
			}

			// If not crediting full price, this is not a full annulment
			if creditLine.CreditAmount != ccfLine.UnitPrice {
				return false
			}
		}
	}

	// All line items of all CCFs are being credited at 100%
	return true
}

// createNotaRecord creates the nota record in the database
func (s *NotaCreditoService) createNotaRecord(
	ctx context.Context,
	companyID string,
	req *models.CreateNotaCreditoRequest,
	ccfs []*models.Invoice,
	lineItems []models.NotaCreditoLineItem,
	totals *CreditTotals,
	isFullAnnulment bool,
) (*models.NotaCredito, error) {

	fmt.Println("\n💾 Creating nota record in database...")

	// Start transaction
	tx, err := database.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Use first CCF for client and establishment info
	firstCCF := ccfs[0]

	// Generate nota number
	notaNumber, err := s.generateNotaNumber(ctx, tx, companyID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate nota number: %w", err)
	}

	// Create main nota record
	notaID := uuid.New().String()
	now := time.Now()

	nota := &models.NotaCredito{
		ID:                      notaID,
		CompanyID:               companyID,
		EstablishmentID:         firstCCF.EstablishmentID,
		PointOfSaleID:           firstCCF.PointOfSaleID,
		NotaNumber:              notaNumber,
		NotaType:                "05", // Nota de Crédito
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
		CreditReason:            req.CreditReason,
		IsFullAnnulment:         isFullAnnulment,
		Subtotal:                totals.Subtotal,
		TotalDiscount:           totals.TotalDiscount,
		TotalTaxes:              totals.TotalTaxes,
		Total:                   totals.Total,
		Currency:                "USD",
		PaymentTerms:            req.PaymentTerms,
		PaymentMethod:           firstCCF.PaymentMethod,
		Status:                  "draft",
		CreatedAt:               now,
	}

	if req.CreditDescription != "" {
		nota.CreditDescription = &req.CreditDescription
	}

	if req.Notes != "" {
		nota.Notes = &req.Notes
	}

	// Insert nota record
	query := `
		INSERT INTO notas_credito (
			id, company_id, establishment_id, point_of_sale_id,
			nota_number, nota_type,
			client_id, client_name, client_legal_name, client_nit, client_ncr, client_dui,
			contact_email, contact_whatsapp, client_address,
			client_tipo_contribuyente, client_tipo_persona,
			credit_reason, credit_description, is_full_annulment,
			subtotal, total_discount, total_taxes, total,
			currency, payment_terms, payment_method,
			status, created_at, notes
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17,
			$18, $19, $20, $21, $22, $23, $24, $25, $26, $27, $28, $29, $30
		)
	`

	_, err = tx.ExecContext(ctx, query,
		nota.ID, nota.CompanyID, nota.EstablishmentID, nota.PointOfSaleID,
		nota.NotaNumber, nota.NotaType,
		nota.ClientID, nota.ClientName, nota.ClientLegalName, nota.ClientNit, nota.ClientNcr, nota.ClientDui,
		nota.ContactEmail, nota.ContactWhatsapp, nota.ClientAddress,
		nota.ClientTipoContribuyente, nota.ClientTipoPersona,
		nota.CreditReason, nota.CreditDescription, nota.IsFullAnnulment,
		nota.Subtotal, nota.TotalDiscount, nota.TotalTaxes, nota.Total,
		nota.Currency, nota.PaymentTerms, nota.PaymentMethod,
		nota.Status, nota.CreatedAt, nota.Notes,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to insert nota: %w", err)
	}

	fmt.Printf("   ✅ Nota record created: %s\n", notaNumber)

	// Insert line items
	for _, line := range lineItems {
		line.NotaCreditoID = notaID
		line.CreatedAt = now

		lineQuery := `
			INSERT INTO notas_credito_line_items (
				id, nota_credito_id, line_number,
				related_ccf_id, related_ccf_number, ccf_line_item_id,
				original_item_sku, original_item_name, original_unit_price, original_quantity,
				original_item_tipo_item, original_unit_of_measure,
				quantity_credited, credit_amount, credit_reason,
				line_subtotal, discount_amount, taxable_amount, total_taxes, line_total,
				created_at
			) VALUES (
				$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15,
				$16, $17, $18, $19, $20, $21
			)
		`

		_, err = tx.ExecContext(ctx, lineQuery,
			line.ID, line.NotaCreditoID, line.LineNumber,
			line.RelatedCCFId, line.RelatedCCFNumber, line.CCFLineItemId,
			line.OriginalItemSku, line.OriginalItemName, line.OriginalUnitPrice, line.OriginalQuantity,
			line.OriginalItemTipoItem, line.OriginalUnitOfMeasure,
			line.QuantityCredited, line.CreditAmount, line.CreditReason,
			line.LineSubtotal, line.DiscountAmount, line.TaxableAmount, line.TotalTaxes, line.LineTotal,
			line.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to insert line item %d: %w", line.LineNumber, err)
		}
	}

	fmt.Printf("   ✅ %d line items inserted\n", len(lineItems))

	// Insert CCF references
	for _, ccf := range ccfs {
		refID := uuid.New().String()
		refQuery := `
			INSERT INTO notas_credito_ccf_references (
				id, nota_credito_id, ccf_id, ccf_number, ccf_date, created_at
			) VALUES ($1, $2, $3, $4, $5, $6)
		`

		ccfDate := firstCCF.CreatedAt
		if firstCCF.FinalizedAt != nil {
			ccfDate = *firstCCF.FinalizedAt
		}

		_, err = tx.ExecContext(ctx, refQuery,
			refID, notaID, ccf.ID, ccf.InvoiceNumber, ccfDate, now,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to insert CCF reference: %w", err)
		}
	}

	fmt.Printf("   ✅ %d CCF references inserted\n", len(ccfs))

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	fmt.Println("   ✅ Transaction committed")

	// Load complete nota with relationships
	nota.LineItems = lineItems
	nota.CCFReferences = make([]models.NotaCreditoCCFReference, 0, len(ccfs))
	for _, ccf := range ccfs {
		ccfDate := ccf.CreatedAt
		if ccf.FinalizedAt != nil {
			ccfDate = *ccf.FinalizedAt
		}
		nota.CCFReferences = append(nota.CCFReferences, models.NotaCreditoCCFReference{
			NotaCreditoID: notaID,
			CCFId:         ccf.ID,
			CCFNumber:     ccf.InvoiceNumber,
			CCFDate:       ccfDate,
		})
	}

	return nota, nil
}

// generateNotaNumber generates a unique nota number for the company
func (s *NotaCreditoService) generateNotaNumber(ctx context.Context, tx *sql.Tx, companyID string) (string, error) {
	// Get the next sequence number for this company
	var seqNum int
	query := `
		SELECT COALESCE(MAX(CAST(SUBSTRING(nota_number FROM 'NC-(\d+)') AS INTEGER)), 0) + 1
		FROM notas_credito
		WHERE company_id = $1
	`
	err := tx.QueryRowContext(ctx, query, companyID).Scan(&seqNum)
	if err != nil && err != sql.ErrNoRows {
		return "", err
	}

	if seqNum == 0 {
		seqNum = 1
	}

	return fmt.Sprintf("NC-%08d", seqNum), nil
}

// getNotaCredito fetches a nota with all relationships
func (s *NotaCreditoService) getNotaCredito(ctx context.Context, notaID, companyID string) (*models.NotaCredito, error) {
	query := `
		SELECT 
			id, company_id, establishment_id, point_of_sale_id,
			nota_number, nota_type,
			client_id, client_name, client_legal_name, client_nit, client_ncr, client_dui,
			contact_email, contact_whatsapp, client_address,
			client_tipo_contribuyente, client_tipo_persona,
			credit_reason, credit_description, is_full_annulment,
			subtotal, total_discount, total_taxes, total,
			currency, payment_terms, payment_method, due_date,
			status,
			dte_numero_control, dte_codigo_generacion, dte_sello_recibido, 
			dte_status, dte_hacienda_response, dte_submitted_at,
			created_at, finalized_at, voided_at,
			created_by, notes
		FROM notas_credito
		WHERE id = $1 AND company_id = $2
	`

	var nota models.NotaCredito
	err := database.DB.QueryRowContext(ctx, query, notaID, companyID).Scan(
		&nota.ID, &nota.CompanyID, &nota.EstablishmentID, &nota.PointOfSaleID,
		&nota.NotaNumber, &nota.NotaType,
		&nota.ClientID, &nota.ClientName, &nota.ClientLegalName, &nota.ClientNit, &nota.ClientNcr, &nota.ClientDui,
		&nota.ContactEmail, &nota.ContactWhatsapp, &nota.ClientAddress,
		&nota.ClientTipoContribuyente, &nota.ClientTipoPersona,
		&nota.CreditReason, &nota.CreditDescription, &nota.IsFullAnnulment,
		&nota.Subtotal, &nota.TotalDiscount, &nota.TotalTaxes, &nota.Total,
		&nota.Currency, &nota.PaymentTerms, &nota.PaymentMethod, &nota.DueDate,
		&nota.Status,
		&nota.DteNumeroControl, &nota.DteCodigoGeneracion, &nota.DteSelloRecibido,
		&nota.DteStatus, &nota.DteHaciendaResponse, &nota.DteSubmittedAt,
		&nota.CreatedAt, &nota.FinalizedAt, &nota.VoidedAt,
		&nota.CreatedBy, &nota.Notes,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("nota not found")
	}
	if err != nil {
		return nil, fmt.Errorf("query nota: %w", err)
	}

	// Load line items
	lineItems, err := s.getNotaLineItems(ctx, notaID)
	if err != nil {
		return nil, fmt.Errorf("failed to load line items: %w", err)
	}
	nota.LineItems = lineItems

	// Load CCF references
	ccfRefs, err := s.getNotaCCFReferences(ctx, notaID)
	if err != nil {
		return nil, fmt.Errorf("failed to load CCF references: %w", err)
	}
	nota.CCFReferences = ccfRefs

	return &nota, nil
}

// getNotaLineItems loads line items for a nota
func (s *NotaCreditoService) getNotaLineItems(ctx context.Context, notaID string) ([]models.NotaCreditoLineItem, error) {
	query := `
		SELECT 
			id, nota_credito_id, line_number,
			related_ccf_id, related_ccf_number, ccf_line_item_id,
			original_item_sku, original_item_name, original_unit_price, original_quantity,
			original_item_tipo_item, original_unit_of_measure,
			quantity_credited, credit_amount, credit_reason,
			line_subtotal, discount_amount, taxable_amount, total_taxes, line_total,
			created_at
		FROM notas_credito_line_items
		WHERE nota_credito_id = $1
		ORDER BY line_number
	`

	rows, err := database.DB.QueryContext(ctx, query, notaID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lineItems []models.NotaCreditoLineItem
	for rows.Next() {
		var item models.NotaCreditoLineItem
		err := rows.Scan(
			&item.ID, &item.NotaCreditoID, &item.LineNumber,
			&item.RelatedCCFId, &item.RelatedCCFNumber, &item.CCFLineItemId,
			&item.OriginalItemSku, &item.OriginalItemName, &item.OriginalUnitPrice, &item.OriginalQuantity,
			&item.OriginalItemTipoItem, &item.OriginalUnitOfMeasure,
			&item.QuantityCredited, &item.CreditAmount, &item.CreditReason,
			&item.LineSubtotal, &item.DiscountAmount, &item.TaxableAmount, &item.TotalTaxes, &item.LineTotal,
			&item.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		lineItems = append(lineItems, item)
	}

	return lineItems, rows.Err()
}

// getNotaCCFReferences loads CCF references for a nota
func (s *NotaCreditoService) getNotaCCFReferences(ctx context.Context, notaID string) ([]models.NotaCreditoCCFReference, error) {
	query := `
		SELECT 
			id, nota_credito_id, ccf_id, ccf_number, ccf_date, created_at
		FROM notas_credito_ccf_references
		WHERE nota_credito_id = $1
		ORDER BY created_at
	`

	rows, err := database.DB.QueryContext(ctx, query, notaID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var refs []models.NotaCreditoCCFReference
	for rows.Next() {
		var ref models.NotaCreditoCCFReference
		err := rows.Scan(
			&ref.ID, &ref.NotaCreditoID, &ref.CCFId, &ref.CCFNumber, &ref.CCFDate, &ref.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		refs = append(refs, ref)
	}

	return refs, rows.Err()
}

// generateNumeroControl generates a numero de control for the nota
func (s *NotaCreditoService) generateNumeroControl(ctx context.Context, companyID string, nota *models.NotaCredito) (string, error) {
	// Get establishment and POS codes
	var codEstablecimiento, codPuntoVenta string
	query := `
		SELECT e.cod_establecimiento, p.cod_punto_venta
		FROM establishments e
		JOIN point_of_sale p ON p.establishment_id = e.id
		WHERE e.id = $1 AND p.id = $2
	`

	err := database.DB.QueryRowContext(ctx, query, nota.EstablishmentID, nota.PointOfSaleID).Scan(
		&codEstablecimiento, &codPuntoVenta,
	)
	if err != nil {
		return "", fmt.Errorf("failed to get establishment codes: %w", err)
	}

	// Get next sequence number
	var sequence int64
	seqQuery := `
		SELECT COALESCE(MAX(CAST(SUBSTRING(dte_numero_control FROM 'DTE-05-.*-(\d{15})') AS BIGINT)), 0) + 1
		FROM notas_credito
		WHERE company_id = $1
	`

	err = database.DB.QueryRowContext(ctx, seqQuery, companyID).Scan(&sequence)
	if err != nil {
		return "", fmt.Errorf("failed to get sequence: %w", err)
	}

	// Generate numero control
	// Format: DTE-05-{codEstable}{codPuntoVenta}-{15-digit sequence}
	numeroControl := fmt.Sprintf("DTE-05-%s%s-%015d", codEstablecimiento, codPuntoVenta, sequence)

	return numeroControl, nil
}

// saveNumeroControl saves the numero control to the nota
func (s *NotaCreditoService) saveNumeroControl(ctx context.Context, notaID, numeroControl string) error {
	query := `
		UPDATE notas_credito
		SET dte_numero_control = $1
		WHERE id = $2
	`

	_, err := database.DB.ExecContext(ctx, query, numeroControl, notaID)
	return err
}

// updateNotaStatusToFinalized updates the nota to finalized status
func (s *NotaCreditoService) updateNotaStatusToFinalized(ctx context.Context, notaID string, finalizedAt time.Time) error {
	query := `
		UPDATE notas_credito
		SET 
			status = 'finalized',
			finalized_at = $1
		WHERE id = $2
	`

	_, err := database.DB.ExecContext(ctx, query, finalizedAt, notaID)
	return err
}
