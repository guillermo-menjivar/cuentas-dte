package services

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"cuentas/internal/codigos"
	"cuentas/internal/database"
	"cuentas/internal/hacienda"
	"cuentas/internal/models"

	"github.com/google/uuid"
)

const (
	MaxCCFRequests      = 50
	MaxLineItems        = 2000
	MaxAdjustmentFactor = 10.0 // Warn if adjustment is >10x original price
)

type NotaService struct{}

func NewNotaService() *NotaService {
	return &NotaService{}
}

// CreateNotaDebito creates a new Nota de Débito
func (s *NotaService) CreateNotaDebito(
	ctx context.Context,
	companyID string,
	req *models.CreateNotaDebitoRequest,
	invoiceService *InvoiceService,
) (*models.NotaDebito, error) {

	fmt.Println("🚀 Starting Nota de Débito creation...")

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

	// Step 3: Validate line items (adjustments to existing CCF items only)
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

	// Step 5: Create nota record in database
	nota, err := s.createNotaRecord(ctx, companyID, req, ccfs, lineItems, totals)
	if err != nil {
		return nil, fmt.Errorf("failed to create nota record: %w", err)
	}

	fmt.Printf("\n✅ Nota de Débito created successfully: %s\n", nota.NotaNumber)

	return nota, nil
}

// validateAndFetchCCFs validates and fetches all referenced CCFs
func (s *NotaService) validateAndFetchCCFs(
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
				"document %s is not a CCF (type: %s). Notas de Débito can only reference CCF invoices (type 03)",
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

		// Validate CCF is not voided
		if invoice.VoidedAt != nil {
			return nil, fmt.Errorf(
				"CCF %s has been voided and cannot be referenced",
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

// validateLineItems validates that all line items are valid adjustments to existing CCF items
func (s *NotaService) validateLineItems(
	ctx context.Context,
	lineItems []models.CreateNotaDebitoLineItemRequest,
	ccfs []*models.Invoice,
) error {

	fmt.Printf("\n🔍 Validating %d line item adjustment(s)...\n", len(lineItems))

	if len(lineItems) > MaxLineItems {
		return fmt.Errorf("maximum %d line items allowed per nota", MaxLineItems)
	}

	// Create a map for quick CCF lookup
	ccfMap := make(map[string]*models.Invoice)
	for _, ccf := range ccfs {
		ccfMap[ccf.ID] = ccf
	}

	// Track seen line item references to prevent duplicate adjustments
	seenRefs := make(map[string]bool)

	for i, item := range lineItems {
		fmt.Printf("  [%d/%d] Validating adjustment to line item in CCF %s\n",
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

		// Validate the adjustment
		if err := s.validateExistingItemAdjustment(ctx, &item, ccf, i+1); err != nil {
			return err
		}

		// Check for duplicate adjustments to same line item
		refKey := fmt.Sprintf("%s-%s", item.RelatedCCFId, item.CCFLineItemId)
		if seenRefs[refKey] {
			return fmt.Errorf(
				"duplicate adjustment to CCF line item %s in CCF %s",
				item.CCFLineItemId,
				item.RelatedCCFId,
			)
		}
		seenRefs[refKey] = true

		fmt.Printf("    ✅ Valid adjustment\n")
	}

	fmt.Printf("✅ All line items validated\n")
	return nil
}

// validateExistingItemAdjustment validates an adjustment to an existing CCF line item
func (s *NotaService) validateExistingItemAdjustment(
	ctx context.Context,
	item *models.CreateNotaDebitoLineItemRequest,
	ccf *models.Invoice,
	lineNumber int,
) error {

	fmt.Printf("      → Validating price adjustment\n")

	// Validate adjustment_amount is positive
	if item.AdjustmentAmount <= 0 {
		return fmt.Errorf(
			"line item %d: adjustment_amount must be positive for Nota de Débito (got %.2f)",
			lineNumber,
			item.AdjustmentAmount,
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
	fmt.Printf("      → Adjustment per unit: $%.2f\n", item.AdjustmentAmount)
	fmt.Printf("      → Quantity: %.2f\n", originalLineItem.Quantity)
	fmt.Printf("      → Total adjustment: $%.2f\n", item.AdjustmentAmount*originalLineItem.Quantity)

	// Validate the adjustment is reasonable
	if item.AdjustmentAmount > originalLineItem.UnitPrice*MaxAdjustmentFactor {
		return fmt.Errorf(
			"line item %d: adjustment amount ($%.2f) is suspiciously large compared to original price ($%.2f). "+
				"Please verify the amount is correct",
			lineNumber,
			item.AdjustmentAmount,
			originalLineItem.UnitPrice,
		)
	}

	// Warn if adjustment is very small
	if item.AdjustmentAmount < 0.01 {
		fmt.Printf("      ⚠️  Warning: Very small adjustment amount ($%.2f)\n", item.AdjustmentAmount)
	}

	return nil
}

// NotaTotals holds calculated totals
type NotaTotals struct {
	Subtotal      float64
	TotalDiscount float64
	TotalTaxes    float64
	Total         float64
}

// calculateTotals calculates all totals for the nota
func (s *NotaService) calculateTotals(
	ctx context.Context,
	lineItems []models.CreateNotaDebitoLineItemRequest,
	ccfs []*models.Invoice,
) ([]models.NotaDebitoLineItem, *NotaTotals, error) {

	fmt.Println("\n🧮 Calculating totals...")

	// Create CCF map for lookups
	ccfMap := make(map[string]*models.Invoice)
	for _, ccf := range ccfs {
		ccfMap[ccf.ID] = ccf
	}

	calculatedLineItems := make([]models.NotaDebitoLineItem, 0, len(lineItems))
	totals := &NotaTotals{}

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

		// Calculate line totals
		// Subtotal = adjustment_amount × quantity
		lineSubtotal := item.AdjustmentAmount * originalLineItem.Quantity
		discountAmount := 0.0 // Notas typically don't have discounts
		taxableAmount := lineSubtotal - discountAmount

		// Calculate taxes (13% IVA for El Salvador)
		// TODO: This should ideally fetch the actual tax rates from the original line item
		lineTaxes := taxableAmount * 0.13

		lineTotal := taxableAmount + lineTaxes

		// Create the calculated line item
		calculatedLine := models.NotaDebitoLineItem{
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
			AdjustmentAmount:      item.AdjustmentAmount,
			LineSubtotal:          lineSubtotal,
			DiscountAmount:        discountAmount,
			TaxableAmount:         taxableAmount,
			TotalTaxes:            lineTaxes,
			LineTotal:             lineTotal,
		}

		if item.AdjustmentReason != "" {
			reason := item.AdjustmentReason
			calculatedLine.AdjustmentReason = &reason
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

// createNotaRecord creates the nota record in the database
func (s *NotaService) createNotaRecord(
	ctx context.Context,
	companyID string,
	req *models.CreateNotaDebitoRequest,
	ccfs []*models.Invoice,
	lineItems []models.NotaDebitoLineItem,
	totals *NotaTotals,
) (*models.NotaDebito, error) {

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

	nota := &models.NotaDebito{
		ID:                      notaID,
		CompanyID:               companyID,
		EstablishmentID:         firstCCF.EstablishmentID,
		PointOfSaleID:           firstCCF.PointOfSaleID,
		NotaNumber:              notaNumber,
		NotaType:                "06", // Nota de Débito
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

	if req.Notes != "" {
		nota.Notes = &req.Notes
	}

	// Insert nota record
	query := `
		INSERT INTO notas_debito (
			id, company_id, establishment_id, point_of_sale_id,
			nota_number, nota_type,
			client_id, client_name, client_legal_name, client_nit, client_ncr, client_dui,
			contact_email, contact_whatsapp, client_address,
			client_tipo_contribuyente, client_tipo_persona,
			subtotal, total_discount, total_taxes, total,
			currency, payment_terms, payment_method,
			status, created_at, notes
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17,
			$18, $19, $20, $21, $22, $23, $24, $25, $26, $27
		)
	`

	_, err = tx.ExecContext(ctx, query,
		nota.ID, nota.CompanyID, nota.EstablishmentID, nota.PointOfSaleID,
		nota.NotaNumber, nota.NotaType,
		nota.ClientID, nota.ClientName, nota.ClientLegalName, nota.ClientNit, nota.ClientNcr, nota.ClientDui,
		nota.ContactEmail, nota.ContactWhatsapp, nota.ClientAddress,
		nota.ClientTipoContribuyente, nota.ClientTipoPersona,
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
		line.NotaDebitoID = notaID
		line.CreatedAt = now

		lineQuery := `
			INSERT INTO nota_debito_line_items (
				id, nota_debito_id, line_number,
				related_ccf_id, related_ccf_number, ccf_line_item_id,
				original_item_sku, original_item_name, original_unit_price, original_quantity,
				original_item_tipo_item, original_unit_of_measure,
				adjustment_amount, adjustment_reason,
				line_subtotal, discount_amount, taxable_amount, total_taxes, line_total,
				created_at
			) VALUES (
				$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20
			)
		`

		_, err = tx.ExecContext(ctx, lineQuery,
			line.ID, line.NotaDebitoID, line.LineNumber,
			line.RelatedCCFId, line.RelatedCCFNumber, line.CCFLineItemId,
			line.OriginalItemSku, line.OriginalItemName, line.OriginalUnitPrice, line.OriginalQuantity,
			line.OriginalItemTipoItem, line.OriginalUnitOfMeasure,
			line.AdjustmentAmount, line.AdjustmentReason,
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
			INSERT INTO nota_debito_ccf_references (
				id, nota_debito_id, ccf_id, ccf_number, ccf_date, created_at
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
	nota.CCFReferences = make([]models.NotaDebitoCCFReference, 0, len(ccfs))
	for _, ccf := range ccfs {
		ccfDate := ccf.CreatedAt
		if ccf.FinalizedAt != nil {
			ccfDate = *ccf.FinalizedAt
		}
		nota.CCFReferences = append(nota.CCFReferences, models.NotaDebitoCCFReference{
			NotaDebitoID: notaID,
			CCFId:        ccf.ID,
			CCFNumber:    ccf.InvoiceNumber,
			CCFDate:      ccfDate,
		})
	}

	return nota, nil
}

// generateNotaNumber generates a unique nota number for the company
func (s *NotaService) generateNotaNumber(ctx context.Context, tx *sql.Tx, companyID string) (string, error) {
	// Get the next sequence number for this company
	var seqNum int
	query := `
		SELECT COALESCE(MAX(CAST(SUBSTRING(nota_number FROM 'ND-(\d+)') AS INTEGER)), 0) + 1
		FROM notas_debito
		WHERE company_id = $1
	`
	err := tx.QueryRowContext(ctx, query, companyID).Scan(&seqNum)
	if err != nil && err != sql.ErrNoRows {
		return "", err
	}

	if seqNum == 0 {
		seqNum = 1
	}

	return fmt.Sprintf("ND-%08d", seqNum), nil
}

// FinalizeNotaDebito finalizes a nota and submits it to Hacienda
func (s *NotaService) FinalizeNotaDebito(
	ctx context.Context,
	notaID string,
	companyID string,
) (*models.NotaDebito, error) {

	fmt.Printf("🔄 Finalizing Nota de Débito: %s\n", notaID)

	// Step 1: Fetch the nota (must be in draft status)
	nota, err := s.getNotaDebito(ctx, notaID, companyID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch nota: %w", err)
	}

	// Step 2: Validate nota is in draft status
	if nota.Status != "draft" {
		return nil, fmt.Errorf("nota is not in draft status (current status: %s)", nota.Status)
	}

	fmt.Printf("   Nota Number: %s\n", nota.NotaNumber)
	fmt.Printf("   Client: %s\n", nota.ClientName)
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

	// Step 4: Process with DTE service (build, sign, submit)
	fmt.Println("\n📤 Submitting to Hacienda...")
	response, err := dteService.ProcessNotaDebito(ctx, nota)
	if err != nil {
		// Update status to 'rejected' if Hacienda rejected it
		if response != nil && response.Estado != "PROCESADO" {
			_ = s.updateNotaStatus(ctx, notaID, "rejected")
		}
		return nil, fmt.Errorf("failed to process nota with Hacienda: %w", err)
	}

	// Step 5: Update nota status to finalized
	now := time.Now()
	err = s.finalizeNotaInDB(ctx, notaID, now, response)
	if err != nil {
		return nil, fmt.Errorf("failed to finalize nota in database: %w", err)
	}

	// Step 6: Reload nota with updated data
	nota, err = s.getNotaDebito(ctx, notaID, companyID)
	if err != nil {
		return nil, fmt.Errorf("failed to reload nota: %w", err)
	}

	fmt.Printf("\n✅ Nota de Débito finalized successfully!\n")
	fmt.Printf("   Status: %s\n", nota.Status)
	fmt.Printf("   Sello Recibido: %s\n", *nota.DteSelloRecibido)

	return nota, nil
}

// getNotaDebito fetches a nota with all relationships
func (s *NotaService) getNotaDebito(ctx context.Context, notaID, companyID string) (*models.NotaDebito, error) {
	query := `
		SELECT 
			id, company_id, establishment_id, point_of_sale_id,
			nota_number, nota_type,
			client_id, client_name, client_legal_name, client_nit, client_ncr, client_dui,
			contact_email, contact_whatsapp, client_address,
			client_tipo_contribuyente, client_tipo_persona,
			subtotal, total_discount, total_taxes, total,
			currency, payment_terms, payment_method, due_date,
			status,
			dte_numero_control, dte_codigo_generacion, dte_sello_recibido, 
			dte_status, dte_hacienda_response, dte_submitted_at,
			created_at, finalized_at, voided_at,
			created_by, notes
		FROM notas_debito
		WHERE id = $1 AND company_id = $2
	`

	var nota models.NotaDebito
	err := database.DB.QueryRowContext(ctx, query, notaID, companyID).Scan(
		&nota.ID, &nota.CompanyID, &nota.EstablishmentID, &nota.PointOfSaleID,
		&nota.NotaNumber, &nota.NotaType,
		&nota.ClientID, &nota.ClientName, &nota.ClientLegalName, &nota.ClientNit, &nota.ClientNcr, &nota.ClientDui,
		&nota.ContactEmail, &nota.ContactWhatsapp, &nota.ClientAddress,
		&nota.ClientTipoContribuyente, &nota.ClientTipoPersona,
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
func (s *NotaService) getNotaLineItems(ctx context.Context, notaID string) ([]models.NotaDebitoLineItem, error) {
	query := `
		SELECT 
			id, nota_debito_id, line_number,
			related_ccf_id, related_ccf_number, ccf_line_item_id,
			original_item_sku, original_item_name, original_unit_price, original_quantity,
			original_item_tipo_item, original_unit_of_measure,
			adjustment_amount, adjustment_reason,
			line_subtotal, discount_amount, taxable_amount, total_taxes, line_total,
			created_at
		FROM nota_debito_line_items
		WHERE nota_debito_id = $1
		ORDER BY line_number
	`

	rows, err := database.DB.QueryContext(ctx, query, notaID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lineItems []models.NotaDebitoLineItem
	for rows.Next() {
		var item models.NotaDebitoLineItem
		err := rows.Scan(
			&item.ID, &item.NotaDebitoID, &item.LineNumber,
			&item.RelatedCCFId, &item.RelatedCCFNumber, &item.CCFLineItemId,
			&item.OriginalItemSku, &item.OriginalItemName, &item.OriginalUnitPrice, &item.OriginalQuantity,
			&item.OriginalItemTipoItem, &item.OriginalUnitOfMeasure,
			&item.AdjustmentAmount, &item.AdjustmentReason,
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
func (s *NotaService) getNotaCCFReferences(ctx context.Context, notaID string) ([]models.NotaDebitoCCFReference, error) {
	query := `
		SELECT 
			id, nota_debito_id, ccf_id, ccf_number, ccf_date, created_at
		FROM nota_debito_ccf_references
		WHERE nota_debito_id = $1
		ORDER BY created_at
	`

	rows, err := database.DB.QueryContext(ctx, query, notaID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var refs []models.NotaDebitoCCFReference
	for rows.Next() {
		var ref models.NotaDebitoCCFReference
		err := rows.Scan(
			&ref.ID, &ref.NotaDebitoID, &ref.CCFId, &ref.CCFNumber, &ref.CCFDate, &ref.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		refs = append(refs, ref)
	}

	return refs, rows.Err()
}

// generateNumeroControl generates a numero de control for the nota
func (s *NotaService) generateNumeroControl(ctx context.Context, companyID string, nota *models.NotaDebito) (string, error) {
	// Get establishment and POS codes
	var codEstablecimiento, codPuntoVenta string
	query := `
		SELECT e.cod_establecimiento, p.cod_punto_venta
		FROM establishments e
		JOIN points_of_sale p ON p.establishment_id = e.id
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
		SELECT COALESCE(MAX(CAST(SUBSTRING(dte_numero_control FROM 'DTE-06-.*-(\d{15})') AS BIGINT)), 0) + 1
		FROM notas_debito
		WHERE company_id = $1
	`

	err = database.DB.QueryRowContext(ctx, seqQuery, companyID).Scan(&sequence)
	if err != nil {
		return "", fmt.Errorf("failed to get sequence: %w", err)
	}

	// Generate numero control
	// Format: DTE-06-{codEstable}{codPuntoVenta}-{15-digit sequence}
	numeroControl := fmt.Sprintf("DTE-06-%s%s-%015d", codEstablecimiento, codPuntoVenta, sequence)

	return numeroControl, nil
}

// saveNumeroControl saves the numero control to the nota
func (s *NotaService) saveNumeroControl(ctx context.Context, notaID, numeroControl string) error {
	query := `
		UPDATE notas_debito
		SET dte_numero_control = $1
		WHERE id = $2
	`

	_, err := database.DB.ExecContext(ctx, query, numeroControl, notaID)
	return err
}

// updateNotaStatus updates the nota status
func (s *NotaService) updateNotaStatus(ctx context.Context, notaID, status string) error {
	query := `
		UPDATE notas_debito
		SET status = $1
		WHERE id = $2
	`

	_, err := database.DB.ExecContext(ctx, query, status, notaID)
	return err
}

// finalizeNotaInDB updates the nota to finalized status with DTE info
func (s *NotaService) finalizeNotaInDB(ctx context.Context, notaID string, finalizedAt time.Time, response *hacienda.ReceptionResponse) error {
	query := `
		UPDATE notas_debito
		SET 
			status = 'finalized',
			finalized_at = $1,
			dte_submitted_at = $2
		WHERE id = $3
	`

	_, err := database.DB.ExecContext(ctx, query, finalizedAt, finalizedAt, notaID)
	return err
}
