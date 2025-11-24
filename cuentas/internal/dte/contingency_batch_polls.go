package dte

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"cuentas/internal/models"
)

// NEW: PollBatches - STEP 3 of contingency process
func (s *ContingencyService) PollBatches(ctx context.Context) error {
	batches, err := s.getProcessingBatches(ctx)
	if err != nil {
		return fmt.Errorf("failed to get processing batches: %w", err)
	}

	if len(batches) == 0 {
		log.Printf("[BatchPolling] No batches to poll")
		return nil
	}

	for _, batch := range batches {
		err := s.pollBatch(ctx, batch)
		if err != nil {
			log.Printf("[BatchPolling] ⚠️  Error polling batch %s: %v", batch.ID, err)
			continue
		}
	}

	return nil
}

func (s *ContingencyService) pollBatch(ctx context.Context, batch *models.ContingencyBatch) error {
	if !batch.CodigoLote.Valid || batch.CodigoLote.String == "" {
		return fmt.Errorf("batch has no codigo_lote")
	}

	authResponse, err := s.haciendaService.AuthenticateCompany(ctx, batch.CompanyID)
	if err != nil {
		return fmt.Errorf("failed to authenticate: %w", err)
	}

	response, err := s.hacienda.QueryBatchStatus(
		ctx,
		authResponse.Body.Token,
		batch.CodigoLote.String,
	)

	if err != nil {
		return fmt.Errorf("failed to query batch: %w", err)
	}

	totalProcessed := len(response.Procesados) + len(response.Rechazados)

	if totalProcessed == 0 {
		log.Printf("[BatchPolling] Batch %s still processing...", batch.ID)
		return nil
	}

	// Process successful DTEs
	for _, resultado := range response.Procesados {
		err := s.UpdateDTEWithSello(ctx, resultado.CodigoGeneracion, resultado.SelloRecibido, resultado)
		if err != nil {
			log.Printf("[BatchPolling] ⚠️  Failed to update DTE %s: %v", resultado.CodigoGeneracion, err)
		}
	}

	// Process rejected DTEs
	for _, resultado := range response.Rechazados {
		err := s.MarkDTEFailed(ctx, resultado.CodigoGeneracion, resultado.DescripcionMsg)
		if err != nil {
			log.Printf("[BatchPolling] ⚠️  Failed to mark DTE %s as failed: %v", resultado.CodigoGeneracion, err)
		}
	}

	// Update batch as complete
	responseJSON, _ := json.Marshal(response)

	updateQuery := `
        UPDATE dte_contingency_batches
        SET status = 'completed',
            processed_count = $1,
            rejected_count = $2,
            hacienda_response = $3,
            completed_at = NOW()
        WHERE id = $4
    `

	_, err = s.db.ExecContext(ctx, updateQuery,
		len(response.Procesados),
		len(response.Rechazados),
		responseJSON,
		batch.ID,
	)

	return err
}

// NEW: getProcessingBatches
func (s *ContingencyService) getProcessingBatches(ctx context.Context) ([]*models.ContingencyBatch, error) {
	query := `
        SELECT id, contingency_event_id, codigo_lote, company_id, ambiente,
               status, total_dtes, processed_count, rejected_count
        FROM dte_contingency_batches
        WHERE status IN ('submitted', 'processing')
        ORDER BY submitted_at ASC
    `

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var batches []*models.ContingencyBatch
	for rows.Next() {
		var batch models.ContingencyBatch
		err := rows.Scan(
			&batch.ID,
			&batch.ContingencyEventID,
			&batch.CodigoLote,
			&batch.CompanyID,
			&batch.Ambiente,
			&batch.Status,
			&batch.TotalDTEs,
			&batch.ProcessedCount,
			&batch.RejectedCount,
		)
		if err != nil {
			return nil, err
		}
		batches = append(batches, &batch)
	}

	return batches, nil
}
