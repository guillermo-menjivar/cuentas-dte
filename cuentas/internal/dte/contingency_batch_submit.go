package dte

import (
	"context"
	"encoding/json"
	"fmt"

	"cuentas/internal/models"

	"github.com/google/uuid"
)

// NEW: CreateAndSubmitBatch - STEP 2 of contingency process
func (s *ContingencyService) CreateAndSubmitBatch(
	ctx context.Context,
	eventID string,
) (*models.ContingencyBatch, error) {

	event, err := s.getEvent(ctx, eventID)
	if err != nil {
		return nil, fmt.Errorf("failed to get event: %w", err)
	}

	if event.Status != "accepted" {
		return nil, fmt.Errorf("event not accepted, cannot submit batch (status: %s)", event.Status)
	}

	dtes, err := s.getDTEsByEvent(ctx, eventID)
	if err != nil {
		return nil, fmt.Errorf("failed to get DTEs: %w", err)
	}

	if len(dtes) == 0 {
		return nil, fmt.Errorf("no DTEs found for event")
	}

	// Check all DTEs are signed
	for _, dte := range dtes {
		if !dte.DTESigned.Valid || dte.DTESigned.String == "" {
			return nil, fmt.Errorf("DTE not signed: %s", dte.ID)
		}
	}

	// Prepare batch
	documentos := make([]string, len(dtes))
	for i, dte := range dtes {
		documentos[i] = dte.DTESigned.String
	}

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

	// Link DTEs to batch
	for _, dte := range dtes {
		s.LinkDTEToBatch(ctx, dte.ID, batchID)
	}

	// Authenticate
	authResponse, err := s.haciendaService.AuthenticateCompany(ctx, event.CompanyID)
	if err != nil {
		s.updateBatchStatus(ctx, batchID, "failed")
		return nil, fmt.Errorf("failed to authenticate: %w", err)
	}

	// Get credentials
	companyUUID, _ := uuid.Parse(event.CompanyID)
	creds, err := s.loadCredentials(ctx, companyUUID)
	if err != nil {
		s.updateBatchStatus(ctx, batchID, "failed")
		return nil, fmt.Errorf("failed to load credentials: %w", err)
	}

	// Submit batch
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
		return nil, fmt.Errorf("batch rejected: %s", response.DescripcionMsg)
	}

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

	// Update DTEs status
	for _, dte := range dtes {
		s.UpdateDTEStatus(ctx, dte.ID, "batch_submitted")
	}

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

// NEW: getEvent
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

	return &event, err
}

// NEW: getDTEsByEvent
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

// NEW: updateBatchStatus
func (s *ContingencyService) updateBatchStatus(ctx context.Context, batchID, status string) error {
	query := `
        UPDATE dte_contingency_batches
        SET status = $1
        WHERE id = $2
    `
	_, err := s.db.ExecContext(ctx, query, status, batchID)
	return err
}
