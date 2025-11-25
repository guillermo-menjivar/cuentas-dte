package dte

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"cuentas/internal/hacienda"
	"cuentas/internal/models"
	"cuentas/internal/services"
)

// ContingencyHelper provides contingency fallback for DTE processing
type ContingencyHelper struct {
	contingencyService *services.ContingencyService
}

// NewContingencyHelper creates a new contingency helper
func NewContingencyHelper(contingencyService *services.ContingencyService) *ContingencyHelper {
	return &ContingencyHelper{
		contingencyService: contingencyService,
	}
}

// HandleSigningFailure queues an invoice when firmador fails
// Call this when s.firmador.Sign() fails after retries
func (h *ContingencyHelper) HandleSigningFailure(
	ctx context.Context,
	invoice *models.Invoice,
	dteUnsigned interface{},
	ambiente string,
) error {
	log.Printf("[ContingencyHelper] Handling signing failure for invoice %s", invoice.ID)

	// Marshal unsigned DTE to JSON
	dteJSON, err := json.Marshal(dteUnsigned)
	if err != nil {
		return fmt.Errorf("failed to marshal unsigned DTE: %w", err)
	}

	// Queue for contingency (no signature)
	err = h.contingencyService.QueueInvoiceForContingency(
		ctx,
		invoice,
		"firmador_failed",
		dteJSON,
		nil, // No signature
		ambiente,
	)

	if err != nil {
		return fmt.Errorf("failed to queue for contingency: %w", err)
	}

	log.Printf("[ContingencyHelper] ✅ Invoice %s queued for contingency (firmador failed)", invoice.ID)
	return nil
}

// HandleAuthFailure queues an invoice when Hacienda auth fails
// Call this when s.haciendaService.AuthenticateCompany() fails
func (h *ContingencyHelper) HandleAuthFailure(
	ctx context.Context,
	invoice *models.Invoice,
	dteUnsigned interface{},
	signedDTE string,
	ambiente string,
) error {
	log.Printf("[ContingencyHelper] Handling auth failure for invoice %s", invoice.ID)

	dteJSON, err := json.Marshal(dteUnsigned)
	if err != nil {
		return fmt.Errorf("failed to marshal unsigned DTE: %w", err)
	}

	err = h.contingencyService.QueueInvoiceForContingency(
		ctx,
		invoice,
		"hacienda_auth_failed",
		dteJSON,
		&signedDTE,
		ambiente,
	)

	if err != nil {
		return fmt.Errorf("failed to queue for contingency: %w", err)
	}

	log.Printf("[ContingencyHelper] ✅ Invoice %s queued for contingency (auth failed)", invoice.ID)
	return nil
}

// HandleSubmissionFailure queues an invoice when Hacienda submission fails
// Call this when s.hacienda.SubmitDTE() fails after retries (timeout, network error)
func (h *ContingencyHelper) HandleSubmissionFailure(
	ctx context.Context,
	invoice *models.Invoice,
	dteUnsigned interface{},
	signedDTE string,
	ambiente string,
) error {
	log.Printf("[ContingencyHelper] Handling submission failure for invoice %s", invoice.ID)

	dteJSON, err := json.Marshal(dteUnsigned)
	if err != nil {
		return fmt.Errorf("failed to marshal unsigned DTE: %w", err)
	}

	err = h.contingencyService.QueueInvoiceForContingency(
		ctx,
		invoice,
		"hacienda_timeout",
		dteJSON,
		&signedDTE,
		ambiente,
	)

	if err != nil {
		return fmt.Errorf("failed to queue for contingency: %w", err)
	}

	log.Printf("[ContingencyHelper] ✅ Invoice %s queued for contingency (submission failed)", invoice.ID)
	return nil
}

// IsRetryableHaciendaError checks if a Hacienda error should trigger contingency
func IsRetryableHaciendaError(err error) bool {
	if err == nil {
		return false
	}

	hacErr, ok := err.(*hacienda.HaciendaError)
	if !ok {
		// Unknown error type - assume retryable (network issue)
		return true
	}

	// Rejections are NOT retryable - they need correction
	if hacErr.Type == "rejection" {
		return false
	}

	// Network, server, timeout errors ARE retryable via contingency
	return hacErr.Type == "network" || hacErr.Type == "server"
}

// ShouldQueueForContingency determines if an error should trigger contingency queueing
func ShouldQueueForContingency(err error) bool {
	if err == nil {
		return false
	}

	hacErr, ok := err.(*hacienda.HaciendaError)
	if !ok {
		// Unknown error - queue for safety
		return true
	}

	switch hacErr.Type {
	case "network":
		return true // Network failures -> contingency
	case "server":
		return true // Server errors -> contingency
	case "rejection":
		return false // Rejections need manual fix, not contingency
	case "validation":
		return false // Validation errors need fix, not contingency
	default:
		return true // Unknown -> queue for safety
	}
}
