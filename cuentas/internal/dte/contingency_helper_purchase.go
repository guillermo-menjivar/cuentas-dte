package dte

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"cuentas/internal/models"
	"cuentas/internal/services"
)

// ContingencyHelperPurchase provides contingency fallback for Purchase DTE processing
type ContingencyHelperPurchase struct {
	contingencyService *services.ContingencyService
}

// NewContingencyHelperPurchase creates a new contingency helper for purchases
func NewContingencyHelperPurchase(contingencyService *services.ContingencyService) *ContingencyHelperPurchase {
	return &ContingencyHelperPurchase{
		contingencyService: contingencyService,
	}
}

// HandleSigningFailure queues a purchase when firmador fails
func (h *ContingencyHelperPurchase) HandleSigningFailure(
	ctx context.Context,
	purchase *models.Purchase,
	dteUnsigned interface{},
	ambiente string,
) error {
	log.Printf("[ContingencyHelperPurchase] Handling signing failure for purchase %s", purchase.ID)

	// Marshal unsigned DTE to JSON
	dteJSON, err := json.Marshal(dteUnsigned)
	if err != nil {
		return fmt.Errorf("failed to marshal unsigned DTE: %w", err)
	}

	// Queue for contingency (no signature)
	err = h.contingencyService.QueuePurchaseForContingency(
		ctx,
		purchase,
		"firmador_failed",
		dteJSON,
		nil, // No signature
		ambiente,
	)

	if err != nil {
		return fmt.Errorf("failed to queue for contingency: %w", err)
	}

	log.Printf("[ContingencyHelperPurchase] ✅ Purchase %s queued for contingency (firmador failed)", purchase.ID)
	return nil
}

// HandleAuthFailure queues a purchase when Hacienda auth fails
func (h *ContingencyHelperPurchase) HandleAuthFailure(
	ctx context.Context,
	purchase *models.Purchase,
	dteUnsigned interface{},
	signedDTE string,
	ambiente string,
) error {
	log.Printf("[ContingencyHelperPurchase] Handling auth failure for purchase %s", purchase.ID)

	dteJSON, err := json.Marshal(dteUnsigned)
	if err != nil {
		return fmt.Errorf("failed to marshal unsigned DTE: %w", err)
	}

	err = h.contingencyService.QueuePurchaseForContingency(
		ctx,
		purchase,
		"hacienda_auth_failed",
		dteJSON,
		&signedDTE,
		ambiente,
	)

	if err != nil {
		return fmt.Errorf("failed to queue for contingency: %w", err)
	}

	log.Printf("[ContingencyHelperPurchase] ✅ Purchase %s queued for contingency (auth failed)", purchase.ID)
	return nil
}

// HandleSubmissionFailure queues a purchase when Hacienda submission fails
func (h *ContingencyHelperPurchase) HandleSubmissionFailure(
	ctx context.Context,
	purchase *models.Purchase,
	dteUnsigned interface{},
	signedDTE string,
	ambiente string,
) error {
	log.Printf("[ContingencyHelperPurchase] Handling submission failure for purchase %s", purchase.ID)

	dteJSON, err := json.Marshal(dteUnsigned)
	if err != nil {
		return fmt.Errorf("failed to marshal unsigned DTE: %w", err)
	}

	err = h.contingencyService.QueuePurchaseForContingency(
		ctx,
		purchase,
		"hacienda_timeout",
		dteJSON,
		&signedDTE,
		ambiente,
	)

	if err != nil {
		return fmt.Errorf("failed to queue for contingency: %w", err)
	}

	log.Printf("[ContingencyHelperPurchase] ✅ Purchase %s queued for contingency (submission failed)", purchase.ID)
	return nil
}
