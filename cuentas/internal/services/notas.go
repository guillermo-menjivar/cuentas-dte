package services

import (
	"context"
	"cuentas/internal/codigos"
	"cuentas/internal/models"
	"fmt"
)

type Notaservice struct{}

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

	if len(ccfIDs) > 50 {
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
		if invoice.InvoiceType != codigos.DocTypeComprobanteCredito {
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
