package dte

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"cuentas/internal/models"

	"github.com/google/uuid"
)

// CreateAndSubmitBatch - STEP 2 of contingency process
func (s *ContingencyService) CreateAndSubmitBatch(
	ctx context.Context,
	eventID string,
) (*models.ContingencyBatch, error) {

	// Get event details
	event, err := s.getEvent(ctx, eventID)
	if err != nil {
		return nil, fmt.Errorf("failed to get event: %w", err)
	}

	if event.Status != "accepted" {
		return nil, fmt.Errorf("event not accepted, cannot submit batch (status: %s)", event.Status)
	}

	log.Printf("[Contingency] Creating batch for event %s", eventID)

	// Get all DTEs linked to this event
	dtes, err := s.getDTEsByEvent(ctx, eventID)
	if err != nil {
		return nil, fmt.Errorf("failed to get DTEs: %w", err)
	}

	if len(dtes) == 0 {
		return nil, fmt.Errorf("no DTEs found for event")
	}

	log.Printf("[Contingency] Found %d DTEs to submit in batch", len(dtes))

	// Check if any DTEs are not signed yet
	unsignedCount := 0
	for _, dte := range dtes {
		if !dte.DTESigned.Valid || dte.DTESigned.String == "" {
			unsignedCount++
		}
	}

	if unsignedCount > 0 {
		return nil, fmt.Errorf("%d DTEs are not signed yet, cannot submit batch", unsignedCount)
	}

	// Prepare batch - collect all signed DTEs
	documentos := make([]string, len(dtes))
	for i, dte := range dtes {
		documentos[i] = dte.DTESigned.String
	}

	// Create batch record
	batchID := uuid.New().String()

	insertQuery := `
        INSERT INTO dte_contingency_batches (
            id, contingency_event_id, company_id, ambiente,
            status, total_dtes
        ) VALUES ($1, $2, $3, $4, $5, $6)
    `

	_, err = s.db.ExecContext(ctx, insertQuery,
		batchID,
		eventID,
		event.CompanyID,
		event.Ambiente,
		"pending",
		len(dtes),
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create batch: %w", err)
	}

	log.Printf("[Contingency] Batch created: %s", batchID)

	// Link all DTEs to batch
	for _, dte := range dtes {
		err = s.LinkDTEToBatch(ctx, dte.ID, batchID)
		if err != nil {
			log.Printf("[Contingency] ⚠️  Warning: failed to link DTE %s to batch: %v", dte.ID, err)
		}
	}

	// Authenticate
	log.Printf("[Contingency] Authenticating with Hacienda...")

	authResponse, err := s.haciendaService.AuthenticateCompany(ctx, event.CompanyID)
	if err != nil {
		s.updateBatchStatus(ctx, batchID, "failed")
		return nil, fmt.Errorf("failed to authenticate: %w", err)
	}

	// Get credentials for NIT
	companyUUID, _ := uuid.Parse(event.CompanyID)
	creds, err := s.loadCredentials(ctx, companyUUID)
	if err != nil {
		s.updateBatchStatus(ctx, batchID, "failed")
		return nil, fmt.Errorf("failed to load credentials: %w", err)
	}

	// Submit batch to Hacienda
	log.Printf("[Contingency] Submitting batch with %d DTEs to Hacienda...", len(documentos))

	response, err := s.hacienda.SubmitBatch(
		ctx,
		authResponse.Body.Token,
		event.Ambiente,
		creds.NIT,
		documentos,
	)

	if err != nil {
		s.updateBatchStatus(ctx, batchID, "failed")
		return nil, fmt.Errorf("failed to submit batch: %w", err)
	}

	if response.Estado != "RECIBIDO" {
		s.updateBatchStatus(ctx, batchID, "failed")
		return nil, fmt.Errorf("batch rejected: %s - %s", response.Estado, response.DescripcionMsg)
	}

	log.Printf("[Contingency] ✅ Batch accepted by Hacienda")
	log.Printf("[Contingency] Codigo Lote: %s", response.CodigoLote)

	// Update batch with response
	responseJSON, _ := json.Marshal(response)

	updateQuery := `
        UPDATE dte_contingency_batches
        SET status = 'submitted',
            codigo_lote = $1,
            hacienda_response = $2,
            submitted_at = NOW()
        WHERE id = $3
    `

	_, err = s.db.ExecContext(ctx, updateQuery,
		response.CodigoLote,
		responseJSON,
		batchID,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to update batch: %w", err)
	}

	// Update all DTEs status
	for _, dte := range dtes {
		s.UpdateDTEStatus(ctx, dte.ID, "batch_submitted")
	}

	log.Printf("[Contingency] ✅ Batch submission complete")

	batch := &models.ContingencyBatch{
		ID:                 batchID,
		ContingencyEventID: eventID,
		CompanyID:          event.CompanyID,
		Ambiente:           event.Ambiente,
		Status:             "submitted",
		TotalDTEs:          len(dtes),
	}

	if response.CodigoLote != "" {
		batch.CodigoLote.String = response.CodigoLote
		batch.CodigoLote.Valid = true
	}

	return batch, nil
}

func (s *ContingencyService) getEvent(ctx context.Context, eventID string) (*models.ContingencyEvent, error) {
	query := `
        SELECT id, codigo_generacion, company_id, ambiente, status, dte_count
        FROM dte_contingency_events
        WHERE id = $1
    `

	var event models.ContingencyEvent
	err := s.db.QueryRowContext(ctx, query, eventID).Scan(
		&event.ID,
		&event.CodigoGeneracion,
		&event.CompanyID,
		&event.Ambiente,
		&event.Status,
		&event.DTECount,
	)

	if err != nil {
		return nil, err
	}

	return &event, nil
}

func (s *ContingencyService) getDTEsByEvent(ctx context.Context, eventID string) ([]*models.ContingencyQueueItem, error) {
	query := `
        SELECT id, invoice_id, purchase_id, tipo_dte, codigo_generacion,
               ambiente, dte_unsigned, dte_signed, status, company_id
        FROM dte_contingency_queue
        WHERE contingency_event_id = $1
        ORDER BY created_at ASC
    `

	rows, err := s.db.QueryContext(ctx, query, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var dtes []*models.ContingencyQueueItem
	for rows.Next() {
		var dte models.ContingencyQueueItem
		err := rows.Scan(
			&dte.ID,
			&dte.InvoiceID,
			&dte.PurchaseID,
			&dte.TipoDte,
			&dte.CodigoGeneracion,
			&dte.Ambiente,
			&dte.DTEUnsigned,
			&dte.DTESigned,
			&dte.Status,
			&dte.CompanyID,
		)
		if err != nil {
			return nil, err
		}
		dtes = append(dtes, &dte)
	}

	return dtes, nil
}

func (s *ContingencyService) updateBatchStatus(ctx context.Context, batchID, status string) error {
	query := `
        UPDATE dte_contingency_batches
        SET status = $1
        WHERE id = $2
    `

	_, err := s.db.ExecContext(ctx, query, status, batchID)
	return err
}
