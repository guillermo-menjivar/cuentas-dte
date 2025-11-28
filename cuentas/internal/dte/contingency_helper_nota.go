package dte

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"cuentas/internal/models"
	"cuentas/internal/services"
)

// ContingencyHelperNota provides contingency fallback for Nota DTE processing
type ContingencyHelperNota struct {
	contingencyService *services.ContingencyService
}

// NewContingencyHelperNota creates a new contingency helper for notas
func NewContingencyHelperNota(contingencyService *services.ContingencyService) *ContingencyHelperNota {
	return &ContingencyHelperNota{
		contingencyService: contingencyService,
	}
}

// ============================================
// NOTA DEBITO HANDLERS
// ============================================

// HandleNotaDebitoSigningFailure queues a nota debito when firmador fails
func (h *ContingencyHelperNota) HandleNotaDebitoSigningFailure(
	ctx context.Context,
	nota *models.NotaDebito,
	dteUnsigned interface{},
	ambiente string,
) error {
	log.Printf("[ContingencyHelperNota] Handling signing failure for nota debito %s", nota.ID)

	dteJSON, err := json.Marshal(dteUnsigned)
	if err != nil {
		return fmt.Errorf("failed to marshal unsigned DTE: %w", err)
	}

	err = h.contingencyService.QueueNotaDebitoForContingency(
		ctx,
		nota,
		"firmador_failed",
		dteJSON,
		nil,
		ambiente,
	)

	if err != nil {
		return fmt.Errorf("failed to queue for contingency: %w", err)
	}

	log.Printf("[ContingencyHelperNota] ✅ Nota Debito %s queued for contingency (firmador failed)", nota.ID)
	return nil
}

// HandleNotaDebitoAuthFailure queues a nota debito when Hacienda auth fails
func (h *ContingencyHelperNota) HandleNotaDebitoAuthFailure(
	ctx context.Context,
	nota *models.NotaDebito,
	dteUnsigned interface{},
	signedDTE string,
	ambiente string,
) error {
	log.Printf("[ContingencyHelperNota] Handling auth failure for nota debito %s", nota.ID)

	dteJSON, err := json.Marshal(dteUnsigned)
	if err != nil {
		return fmt.Errorf("failed to marshal unsigned DTE: %w", err)
	}

	err = h.contingencyService.QueueNotaDebitoForContingency(
		ctx,
		nota,
		"hacienda_auth_failed",
		dteJSON,
		&signedDTE,
		ambiente,
	)

	if err != nil {
		return fmt.Errorf("failed to queue for contingency: %w", err)
	}

	log.Printf("[ContingencyHelperNota] ✅ Nota Debito %s queued for contingency (auth failed)", nota.ID)
	return nil
}

// HandleNotaDebitoSubmissionFailure queues a nota debito when Hacienda submission fails
func (h *ContingencyHelperNota) HandleNotaDebitoSubmissionFailure(
	ctx context.Context,
	nota *models.NotaDebito,
	dteUnsigned interface{},
	signedDTE string,
	ambiente string,
) error {
	log.Printf("[ContingencyHelperNota] Handling submission failure for nota debito %s", nota.ID)

	dteJSON, err := json.Marshal(dteUnsigned)
	if err != nil {
		return fmt.Errorf("failed to marshal unsigned DTE: %w", err)
	}

	err = h.contingencyService.QueueNotaDebitoForContingency(
		ctx,
		nota,
		"hacienda_timeout",
		dteJSON,
		&signedDTE,
		ambiente,
	)

	if err != nil {
		return fmt.Errorf("failed to queue for contingency: %w", err)
	}

	log.Printf("[ContingencyHelperNota] ✅ Nota Debito %s queued for contingency (submission failed)", nota.ID)
	return nil
}

// ============================================
// NOTA CREDITO HANDLERS
// ============================================

// HandleNotaCreditoSigningFailure queues a nota credito when firmador fails
func (h *ContingencyHelperNota) HandleNotaCreditoSigningFailure(
	ctx context.Context,
	nota *models.NotaCredito,
	dteUnsigned interface{},
	ambiente string,
) error {
	log.Printf("[ContingencyHelperNota] Handling signing failure for nota credito %s", nota.ID)

	dteJSON, err := json.Marshal(dteUnsigned)
	if err != nil {
		return fmt.Errorf("failed to marshal unsigned DTE: %w", err)
	}

	err = h.contingencyService.QueueNotaCreditoForContingency(
		ctx,
		nota,
		"firmador_failed",
		dteJSON,
		nil,
		ambiente,
	)

	if err != nil {
		return fmt.Errorf("failed to queue for contingency: %w", err)
	}

	log.Printf("[ContingencyHelperNota] ✅ Nota Credito %s queued for contingency (firmador failed)", nota.ID)
	return nil
}

// HandleNotaCreditoAuthFailure queues a nota credito when Hacienda auth fails
func (h *ContingencyHelperNota) HandleNotaCreditoAuthFailure(
	ctx context.Context,
	nota *models.NotaCredito,
	dteUnsigned interface{},
	signedDTE string,
	ambiente string,
) error {
	log.Printf("[ContingencyHelperNota] Handling auth failure for nota credito %s", nota.ID)

	dteJSON, err := json.Marshal(dteUnsigned)
	if err != nil {
		return fmt.Errorf("failed to marshal unsigned DTE: %w", err)
	}

	err = h.contingencyService.QueueNotaCreditoForContingency(
		ctx,
		nota,
		"hacienda_auth_failed",
		dteJSON,
		&signedDTE,
		ambiente,
	)

	if err != nil {
		return fmt.Errorf("failed to queue for contingency: %w", err)
	}

	log.Printf("[ContingencyHelperNota] ✅ Nota Credito %s queued for contingency (auth failed)", nota.ID)
	return nil
}

// HandleNotaCreditoSubmissionFailure queues a nota credito when Hacienda submission fails
func (h *ContingencyHelperNota) HandleNotaCreditoSubmissionFailure(
	ctx context.Context,
	nota *models.NotaCredito,
	dteUnsigned interface{},
	signedDTE string,
	ambiente string,
) error {
	log.Printf("[ContingencyHelperNota] Handling submission failure for nota credito %s", nota.ID)

	dteJSON, err := json.Marshal(dteUnsigned)
	if err != nil {
		return fmt.Errorf("failed to marshal unsigned DTE: %w", err)
	}

	err = h.contingencyService.QueueNotaCreditoForContingency(
		ctx,
		nota,
		"hacienda_timeout",
		dteJSON,
		&signedDTE,
		ambiente,
	)

	if err != nil {
		return fmt.Errorf("failed to queue for contingency: %w", err)
	}

	log.Printf("[ContingencyHelperNota] ✅ Nota Credito %s queued for contingency (submission failed)", nota.ID)
	return nil
}
