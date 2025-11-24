package dte

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"cuentas/internal/models"

	"github.com/google/uuid"
)

// CreateAndSubmitContingencyEvent - STEP 1 of contingency process
func (s *ContingencyService) CreateAndSubmitContingencyEvent(
	ctx context.Context,
	companyID string,
	dtes []*models.ContingencyQueueItem,
) (*models.ContingencyEvent, error) {

	if len(dtes) == 0 {
		return nil, fmt.Errorf("no DTEs to create event")
	}

	ambiente := dtes[0].Ambiente

	log.Printf("[Contingency] STEP 1: Creating event for %d DTEs (company: %s)", len(dtes), companyID)

	// Build contingency event JSON
	eventJSON, codigoGeneracion, err := s.BuildContingencyEvent(
		ctx,
		companyID,
		ambiente,
		dtes,
		1, // tipoContingencia
		"",
	)

	if err != nil {
		return nil, fmt.Errorf("failed to build event: %w", err)
	}

	companyUUID, _ := uuid.Parse(companyID)
	creds, err := s.loadCredentials(ctx, companyUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to load credentials: %w", err)
	}

	// Sign event
	var eventForSigning interface{}
	json.Unmarshal(eventJSON, &eventForSigning)

	signedEvent, err := s.firmador.Sign(ctx, creds.NIT, creds.Password, eventForSigning)
	if err != nil {
		return nil, fmt.Errorf("failed to sign event: %w", err)
	}

	// Authenticate with Hacienda
	authResponse, err := s.haciendaService.AuthenticateCompany(ctx, companyID)
	if err != nil {
		return nil, fmt.Errorf("failed to authenticate: %w", err)
	}

	// Submit to Hacienda
	response, err := s.hacienda.SubmitContingencyEvent(
		ctx,
		authResponse.Body.Token,
		creds.NIT,
		signedEvent,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to submit event: %w", err)
	}

	if response.Estado != "RECIBIDO" {
		return nil, fmt.Errorf("event rejected: %s - %s", response.Estado, response.Mensaje) // FIXED: was DescripcionMsg
	}

	// Store event in database
	eventID := uuid.New().String()
	responseJSON, _ := json.Marshal(response)

	insertQuery := `
		INSERT INTO dte_contingency_events (
			id, codigo_generacion, company_id, ambiente,
			status, dte_count, event_unsigned, event_signed,
			hacienda_response, sello_recibido, accepted_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW())
	`

	_, err = s.db.ExecContext(ctx, insertQuery,
		eventID,
		codigoGeneracion,
		companyID,
		ambiente,
		"accepted",
		len(dtes),
		eventJSON,
		signedEvent,
		responseJSON,
		response.SelloRecibido,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to store event: %w", err)
	}

	// Link all DTEs to this event
	for _, dte := range dtes {
		s.LinkDTEToEvent(ctx, dte.ID, eventID)
	}

	return &models.ContingencyEvent{
		ID:               eventID,
		CodigoGeneracion: codigoGeneracion,
		CompanyID:        companyID,
		Ambiente:         ambiente,
		Status:           "accepted",
		DTECount:         len(dtes),
	}, nil
}

// getEventsReadyForBatch returns events that are accepted but don't have batches yet
func (s *ContingencyService) getEventsReadyForBatch(ctx context.Context) ([]*EventInfo, error) {
	query := `
        SELECT e.id, e.codigo_generacion, e.company_id, e.ambiente
        FROM dte_contingency_events e
        WHERE e.status = 'accepted'
        AND NOT EXISTS (
            SELECT 1 FROM dte_contingency_batches b
            WHERE b.contingency_event_id = e.id
        )
        ORDER BY e.accepted_at ASC
    `

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*EventInfo
	for rows.Next() {
		var event EventInfo
		err := rows.Scan(&event.ID, &event.CodigoGeneracion, &event.CompanyID, &event.Ambiente)
		if err != nil {
			return nil, err
		}
		events = append(events, &event)
	}

	return events, nil
}

type EventInfo struct {
	ID               string
	CodigoGeneracion string
	CompanyID        string
	Ambiente         string
}

// incrementDTERetryCount increments the retry count for a DTE
func (s *ContingencyService) incrementDTERetryCount(ctx context.Context, dteID string) error {
	query := `
        UPDATE dte_contingency_queue
        SET retry_count = retry_count + 1,
            updated_at = NOW()
        WHERE id = $1
    `
	_, err := s.db.ExecContext(ctx, query, dteID)
	return err
}
