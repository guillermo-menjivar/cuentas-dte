package dte

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"

	"cuentas/internal/hacienda"
	"cuentas/internal/models"
)

type ContingencyService struct {
	db              *sql.DB
	firmador        Firmador
	hacienda        HaciendaClient
	haciendaService *hacienda.Service
}

func NewContingencyService(
	db *sql.DB,
	firmador Firmador,
	haciendaClient HaciendaClient,
	haciendaService *hacienda.Service,
) *ContingencyService {
	return &ContingencyService{
		db:              db,
		firmador:        firmador,
		hacienda:        haciendaClient,
		haciendaService: haciendaService,
	}
}

// AddToQueue adds a failed DTE to contingency queue
func (s *ContingencyService) AddToQueue(ctx context.Context, params AddToQueueParams) error {
	query := `
        INSERT INTO dte_contingency_queue (
            invoice_id, purchase_id, tipo_dte, codigo_generacion, ambiente,
            failure_stage, failure_reason, dte_unsigned, dte_signed,
            status, company_id, created_by
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
        RETURNING id
    `

	var id string
	err := s.db.QueryRowContext(ctx, query,
		params.InvoiceID,
		params.PurchaseID,
		params.TipoDte,
		params.CodigoGeneracion,
		params.Ambiente,
		params.FailureStage,
		params.FailureReason,
		params.DTEUnsigned,
		params.DTESigned,
		"pending",
		params.CompanyID,
		params.CreatedBy,
	).Scan(&id)

	if err != nil {
		return fmt.Errorf("failed to add to contingency queue: %w", err)
	}

	log.Printf("[Contingency] ✅ Added DTE %s to queue (stage: %s, id: %s)",
		params.CodigoGeneracion, params.FailureStage, id)

	return nil
}

type AddToQueueParams struct {
	InvoiceID        *string // VARCHAR(36) - can be nil
	PurchaseID       *string // UUID - can be nil
	TipoDte          string
	CodigoGeneracion string
	Ambiente         string
	FailureStage     string
	FailureReason    string
	DTEUnsigned      []byte
	DTESigned        *string
	CompanyID        string
	CreatedBy        *string
}

// GetPendingDTEsByCompany gets all pending DTEs for a company
func (s *ContingencyService) GetPendingDTEsByCompany(ctx context.Context, companyID string) ([]*models.ContingencyQueueItem, error) {
	query := `
        SELECT id, invoice_id, purchase_id, tipo_dte, codigo_generacion, ambiente,
               failure_stage, failure_reason, failure_timestamp,
               dte_unsigned, dte_signed, status, retry_count, max_retries,
               created_at, company_id
        FROM dte_contingency_queue
        WHERE company_id = $1
        AND status = 'pending'
        AND retry_count < max_retries
        ORDER BY created_at ASC
        LIMIT 1000
    `

	rows, err := s.db.QueryContext(ctx, query, companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*models.ContingencyQueueItem
	for rows.Next() {
		var item models.ContingencyQueueItem
		err := rows.Scan(
			&item.ID, &item.InvoiceID, &item.PurchaseID, &item.TipoDte,
			&item.CodigoGeneracion, &item.Ambiente, &item.FailureStage,
			&item.FailureReason, &item.FailureTimestamp, &item.DTEUnsigned,
			&item.DTESigned, &item.Status, &item.RetryCount, &item.MaxRetries,
			&item.CreatedAt, &item.CompanyID,
		)
		if err != nil {
			return nil, err
		}
		items = append(items, &item)
	}

	return items, nil
}

// GetCompaniesWithPendingDTEs returns list of company IDs with pending contingency DTEs
func (s *ContingencyService) GetCompaniesWithPendingDTEs(ctx context.Context) ([]string, error) {
	query := `
        SELECT DISTINCT company_id
        FROM dte_contingency_queue
        WHERE status = 'pending'
        AND retry_count < max_retries
    `

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var companies []string
	for rows.Next() {
		var companyID string
		if err := rows.Scan(&companyID); err != nil {
			return nil, err
		}
		companies = append(companies, companyID)
	}

	return companies, nil
}

// UpdateDTEStatus updates the status of a DTE in the queue
func (s *ContingencyService) UpdateDTEStatus(ctx context.Context, dteID, status string) error {
	query := `
        UPDATE dte_contingency_queue
        SET status = $1,
            updated_at = NOW()
        WHERE id = $2
    `

	_, err := s.db.ExecContext(ctx, query, status, dteID)
	return err
}

// LinkDTEToEvent links a DTE to a contingency event
func (s *ContingencyService) LinkDTEToEvent(ctx context.Context, dteID, eventID string) error {
	query := `
        UPDATE dte_contingency_queue
        SET contingency_event_id = $1,
            status = 'event_created',
            updated_at = NOW()
        WHERE id = $2
    `

	_, err := s.db.ExecContext(ctx, query, eventID, dteID)
	return err
}

// LinkDTEToBatch links a DTE to a batch
func (s *ContingencyService) LinkDTEToBatch(ctx context.Context, dteID, batchID string) error {
	query := `
        UPDATE dte_contingency_queue
        SET batch_id = $1,
            status = 'batch_created',
            updated_at = NOW()
        WHERE id = $2
    `

	_, err := s.db.ExecContext(ctx, query, batchID, dteID)
	return err
}

// UpdateDTEWithSello updates a DTE with the sello from Hacienda
func (s *ContingencyService) UpdateDTEWithSello(
	ctx context.Context,
	codigoGeneracion string,
	sello string,
	response interface{},
) error {
	responseJSON, _ := json.Marshal(response)

	query := `
        UPDATE dte_contingency_queue
        SET status = 'success',
            sello_recibido = $1,
            hacienda_response = $2,
            completed_at = NOW(),
            updated_at = NOW()
        WHERE codigo_generacion = $3
    `

	_, err := s.db.ExecContext(ctx, query, sello, responseJSON, codigoGeneracion)
	if err != nil {
		return fmt.Errorf("failed to update DTE with sello: %w", err)
	}

	// Also update the original invoice/purchase
	return s.updateOriginalDocumentWithSello(ctx, codigoGeneracion, sello, responseJSON)
}

func (s *ContingencyService) updateOriginalDocumentWithSello(
	ctx context.Context,
	codigoGeneracion string,
	sello string,
	responseJSON []byte,
) error {
	// Get the DTE to find invoice_id or purchase_id
	var invoiceID sql.NullString
	var purchaseID *string

	query := `SELECT invoice_id, purchase_id FROM dte_contingency_queue WHERE codigo_generacion = $1`
	err := s.db.QueryRowContext(ctx, query, codigoGeneracion).Scan(&invoiceID, &purchaseID)
	if err != nil {
		return err
	}

	if invoiceID.Valid {
		// Update invoice
		updateQuery := `
            UPDATE invoices
            SET dte_sello_recibido = $1,
                dte_hacienda_response = $2,
                dte_status = 'PROCESADO',
                dte_submitted_at = NOW()
            WHERE id = $3
        `
		_, err = s.db.ExecContext(ctx, updateQuery, sello, responseJSON, invoiceID.String)
		return err
	} else if purchaseID != nil {
		// Update purchase
		updateQuery := `
            UPDATE purchases
            SET dte_sello_recibido = $1,
                dte_hacienda_response = $2,
                dte_status = 'PROCESADO',
                dte_submitted_at = NOW()
            WHERE id = $3
        `
		_, err = s.db.ExecContext(ctx, updateQuery, sello, responseJSON, *purchaseID)
		return err
	}

	return nil
}

// MarkDTEFailed marks a DTE as permanently failed
func (s *ContingencyService) MarkDTEFailed(ctx context.Context, codigoGeneracion, reason string) error {
	query := `
        UPDATE dte_contingency_queue
        SET status = 'rejected',
            failure_reason = $1,
            completed_at = NOW(),
            updated_at = NOW()
        WHERE codigo_generacion = $2
    `

	_, err := s.db.ExecContext(ctx, query, reason, codigoGeneracion)
	return err
}
