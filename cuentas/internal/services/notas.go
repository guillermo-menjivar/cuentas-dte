package services

import (
	"context"
	"cuentas/internal/codigos"
	"cuentas/internal/database"
	"cuentas/internal/models"
	"database/sql"
	"fmt"
)

const (
	MaxCCFRequests = 50
)

type NotaService struct{}

func NewNotaService() *NotaService {
	return &NotaService{}
}

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
		return nil, fmt.Errorf("maximum 50 CCFs allowed per nota")
	}

	// Store fetched CCFs
	ccfs := make([]*models.Invoice, 0, len(ccfIDs))

	// Track seen IDs to prevent duplicates
	seenIDs := make(map[string]bool)

	fmt.Printf("🔍 Validating %d CCF(s)...\n", len(ccfIDs))

	for i, ccfID := range ccfIDs {
		// Check for duplicate IDs in request
		if seenIDs[ccfID] {
			return nil, fmt.Errorf("duplicate CCF ID found: %s", ccfID)
		}
		seenIDs[ccfID] = true

		fmt.Printf("  [%d/%d] Fetching CCF: %s\n", i+1, len(ccfIDs), ccfID)

		// Fetch the invoice using InvoiceService
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

		// Validate CCF is finalized (can't adjust draft invoices)
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

	// Additional validation: All CCFs must belong to the same client
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

func (s *NotaService) CreateNotaDebito(
	ctx context.Context,
	companyID string,
	req *models.CreateNotaDebitoRequest,
	invoiceService *InvoiceService,
) error {

	// Step 1: Validate and fetch all referenced CCFs
	ccfs, err := s.validateAndFetchCCFs(ctx, companyID, req.CCFIds, invoiceService)
	if err != nil {
		return err
	}

	fmt.Printf("\n✅ Successfully validated %d CCF(s)\n", len(ccfs))
	// Step 2 Validate line items
	if err := s.validateLineItems(ctx, companyID, req.LineItems, ccfs); err != nil {
		return nil, err
	}
	// lets inspect if the item is either new or existing.
	// if its new we just add it to the total. If its existing lets make sure the price has increased

	// TODO: Next steps will be:
	// - Validate line items
	// - Calculate totals
	// - Create nota record
	return nil

}

func (s *NotaService) validateLineItems(
	ctx context.Context,
	companyID string,
	lineItems []models.CreateNotaLineItemRequest,
	ccfs []*models.Invoice,
) error {

	fmt.Printf("\n🔍 Validating %d line item(s)...\n", len(lineItems))

	// Create a map for quick CCF lookup
	ccfMap := make(map[string]*models.Invoice)
	for _, ccf := range ccfs {
		ccfMap[ccf.ID] = ccf
	}

	// Track seen line item references to prevent duplicates
	seenRefs := make(map[string]bool)

	for i, item := range lineItems {
		fmt.Printf("  [%d/%d] Validating line item: %s\n", i+1, len(lineItems), item.ItemName)

		// Validate related_ccf_id exists in the ccf_ids array
		ccf, exists := ccfMap[item.RelatedCCFId]
		if !exists {
			return fmt.Errorf(
				"line item %d references CCF %s which is not in the ccf_ids list",
				i+1,
				item.RelatedCCFId,
			)
		}

		if item.IsNewItem {
			// ========================================
			// NEW ITEM VALIDATION
			// ========================================
			if err := s.validateNewItem(ctx, companyID, &item, i+1); err != nil {
				return err
			}

		} else {
			// ========================================
			// EXISTING ITEM ADJUSTMENT VALIDATION
			// ========================================
			if err := s.validateExistingItemAdjustment(ctx, &item, ccf, i+1); err != nil {
				return err
			}
		}

		// Check for duplicate adjustments to same line item
		if !item.IsNewItem && item.CCFLineItemId != nil {
			refKey := fmt.Sprintf("%s-%s", item.RelatedCCFId, *item.CCFLineItemId)
			if seenRefs[refKey] {
				return fmt.Errorf(
					"duplicate adjustment to CCF line item %s in CCF %s",
					*item.CCFLineItemId,
					item.RelatedCCFId,
				)
			}
			seenRefs[refKey] = true
		}

		fmt.Printf("    ✅ Valid line item\n")
	}

	fmt.Printf("✅ All line items validated\n")
	return nil
}

// validateExistingItemAdjustment validates an adjustment to an existing CCF line item
func (s *NotaService) validateExistingItemAdjustment(
	ctx context.Context,
	item *models.CreateNotaLineItemRequest,
	ccf *models.Invoice,
	lineNumber int,
) error {

	fmt.Printf("      → Adjusting existing CCF line item\n")

	// Validate required fields for existing item adjustment
	if item.CCFLineItemId == nil || *item.CCFLineItemId == "" {
		return fmt.Errorf(
			"line item %d: ccf_line_item_id is required when adjusting existing items (is_new_item=false)",
			lineNumber,
		)
	}

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
		if ccf.LineItems[i].ID == *item.CCFLineItemId {
			originalLineItem = &ccf.LineItems[i]
			break
		}
	}

	if originalLineItem == nil {
		return fmt.Errorf(
			"line item %d: CCF line item %s not found in CCF %s",
			lineNumber,
			*item.CCFLineItemId,
			ccf.ID,
		)
	}

	fmt.Printf("      → Found original line item: %s (SKU: %s)\n",
		originalLineItem.ItemName,
		originalLineItem.ItemSku,
	)
	fmt.Printf("      → Original unit price: $%.2f\n", originalLineItem.UnitPrice)
	fmt.Printf("      → Adjustment amount: $%.2f\n", item.AdjustmentAmount)

	// Validate the adjustment makes sense
	// The adjustment should be reasonable compared to original price
	if item.AdjustmentAmount > originalLineItem.UnitPrice*10 {
		return fmt.Errorf(
			"line item %d: adjustment amount ($%.2f) is suspiciously large compared to original price ($%.2f). "+
				"Please verify the amount is correct",
			lineNumber,
			item.AdjustmentAmount,
			originalLineItem.UnitPrice,
		)
	}

	// Optional: Warn if adjustment is very small
	if item.AdjustmentAmount < 0.01 {
		fmt.Printf("      ⚠️  Warning: Very small adjustment amount ($%.2f)\n", item.AdjustmentAmount)
	}

	return nil
}

// validateNewItem validates a new item being added to the nota
func (s *NotaService) validateNewItem(
	ctx context.Context,
	companyID string,
	item *models.CreateNotaLineItemRequest,
	lineNumber int,
) error {

	fmt.Printf("      → Adding new item\n")

	// Validate required fields for new item
	if item.InventoryItemId == nil || *item.InventoryItemId == "" {
		return fmt.Errorf(
			"line item %d: inventory_item_id is required when adding new items (is_new_item=true)",
			lineNumber,
		)
	}

	if item.Quantity <= 0 {
		return fmt.Errorf(
			"line item %d: quantity must be greater than 0 (got %.2f)",
			lineNumber,
			item.Quantity,
		)
	}

	if item.UnitPrice < 0 {
		return fmt.Errorf(
			"line item %d: unit_price cannot be negative (got %.2f)",
			lineNumber,
			item.UnitPrice,
		)
	}

	// Fetch inventory item to validate it exists
	query := `
		SELECT 
			id, sku, name, tipo_item, unit_of_measure, unit_price, active
		FROM inventory_items
		WHERE id = $1 AND company_id = $2
	`

	var invItem struct {
		ID            string
		SKU           string
		Name          string
		TipoItem      string
		UnitOfMeasure string
		UnitPrice     float64
		Active        bool
	}

	err := database.DB.QueryRowContext(ctx, query, *item.InventoryItemId, companyID).Scan(
		&invItem.ID,
		&invItem.SKU,
		&invItem.Name,
		&invItem.TipoItem,
		&invItem.UnitOfMeasure,
		&invItem.UnitPrice,
		&invItem.Active,
	)

	if err == sql.ErrNoRows {
		return fmt.Errorf(
			"line item %d: inventory item %s not found",
			lineNumber,
			*item.InventoryItemId,
		)
	}
	if err != nil {
		return fmt.Errorf(
			"line item %d: failed to fetch inventory item: %w",
			lineNumber,
			err,
		)
	}

	// Validate inventory item is active
	if !invItem.Active {
		return fmt.Errorf(
			"line item %d: inventory item '%s' (SKU: %s) is not active",
			lineNumber,
			invItem.Name,
			invItem.SKU,
		)
	}

	fmt.Printf("      → Inventory item: %s (SKU: %s)\n", invItem.Name, invItem.SKU)
	fmt.Printf("      → Quantity: %.2f x Unit Price: $%.2f\n", item.Quantity, item.UnitPrice)

	// Optional: Warn if price differs significantly from inventory
	priceDiff := item.UnitPrice - invItem.UnitPrice
	if priceDiff != 0 {
		priceDiffPercent := (priceDiff / invItem.UnitPrice) * 100
		if priceDiffPercent > 50 || priceDiffPercent < -50 {
			fmt.Printf("      ⚠️  Warning: Price differs %.0f%% from inventory ($%.2f vs $%.2f)\n",
				priceDiffPercent,
				item.UnitPrice,
				invItem.UnitPrice,
			)
		}
	}

	return nil
}
