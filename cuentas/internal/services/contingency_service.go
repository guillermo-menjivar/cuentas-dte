package services

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	"cuentas/internal/models"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// ContingencyService handles contingency period management and invoice queueing
type ContingencyService struct {
	db *sql.DB
}

// NewContingencyService creates a new contingency service
func NewContingencyService(db *sql.DB) *ContingencyService {
	return &ContingencyService{db: db}
}

// QueueInvoiceForContingency queues a failed invoice for contingency processing
// This is the main function you'll call from your DTE processors when signing/submission fails
func (s *ContingencyService) QueueInvoiceForContingency(
	ctx context.Context,
	invoice *models.Invoice,
	failureType string, // "firmador_failed", "hacienda_auth_failed", "hacienda_timeout"
	dteUnsigned []byte,
	dteSigned *string,
	ambiente string,
) error {
	log.Printf("[Contingency] Queueing invoice %s for contingency (failure: %s)", invoice.ID, failureType)

	// Determine tipo_contingencia based on failure
	tipoContingencia, motivoContingencia := s.determineContingencyType(failureType)

	// Find or create contingency period for this POS
	period, err := s.findOrCreatePeriod(
		ctx,
		invoice.CompanyID,
		invoice.EstablishmentID,
		invoice.PointOfSaleID,
		ambiente,
		tipoContingencia,
		motivoContingencia,
	)
	if err != nil {
		return fmt.Errorf("failed to find/create contingency period: %w", err)
	}

	// Determine invoice status based on whether we have a signature
	var status string
	if dteSigned != nil && *dteSigned != "" {
		status = models.DTEStatusFailedRetry // Signed but Hacienda failed
	} else {
		status = models.DTEStatusPendingSignature // Unsigned, firmador failed
	}

	// Update invoice with contingency info
	query := `
		UPDATE invoices
		SET contingency_period_id = $1,
			dte_transmission_status = $2,
			dte_unsigned = $3,
			dte_signed = $4,
			signature_retry_count = COALESCE(signature_retry_count, 0)
		WHERE id = $5
	`

	_, err = s.db.ExecContext(ctx, query,
		period.ID,
		status,
		dteUnsigned,
		dteSigned,
		invoice.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update invoice for contingency: %w", err)
	}

	log.Printf("[Contingency] ✅ Invoice %s queued in period %s (status: %s)", invoice.ID, period.ID, status)
	return nil
}

// determineContingencyType maps failure type to Hacienda's tipo_contingencia codes
func (s *ContingencyService) determineContingencyType(failureType string) (int, string) {
	switch failureType {
	case "firmador_failed":
		return models.TipoContingenciaOther, "Falla en servicio de firmador - no se pudo firmar el DTE"
	case "hacienda_auth_failed":
		return models.TipoContingenciaMHDown, "No fue posible autenticarse con el sistema del MH"
	case "hacienda_timeout":
		return models.TipoContingenciaMHDown, "No disponibilidad de sistema del MH - timeout después de reintentos"
	case "hacienda_rejected":
		return models.TipoContingenciaMHDown, "Error en recepción del MH"
	case "internet_outage":
		return models.TipoContingenciaInternet, "Falla en el suministro de servicio de Internet"
	case "power_outage":
		return models.TipoContingenciaPower, "Falla en el suministro de energía eléctrica"
	default:
		return models.TipoContingenciaOther, fmt.Sprintf("Error no clasificado: %s", failureType)
	}
}

// findOrCreatePeriod finds an existing active period or creates a new one
func (s *ContingencyService) findOrCreatePeriod(
	ctx context.Context,
	companyID, establishmentID, pointOfSaleID, ambiente string,
	tipoContingencia int,
	motivoContingencia string,
) (*models.ContingencyPeriod, error) {
	// Try to find existing active period for this POS + ambiente
	var period models.ContingencyPeriod
	query := `
		SELECT id, company_id, establishment_id, point_of_sale_id, ambiente,
			   f_inicio, h_inicio, f_fin, h_fin,
			   tipo_contingencia, motivo_contingencia, status, processing,
			   created_at, updated_at
		FROM contingency_periods
		WHERE company_id = $1
		  AND establishment_id = $2
		  AND point_of_sale_id = $3
		  AND ambiente = $4
		  AND status = 'active'
		LIMIT 1
	`

	err := s.db.QueryRowContext(ctx, query,
		companyID, establishmentID, pointOfSaleID, ambiente,
	).Scan(
		&period.ID, &period.CompanyID, &period.EstablishmentID, &period.PointOfSaleID,
		&period.Ambiente, &period.FInicio, &period.HInicio, &period.FFin, &period.HFin,
		&period.TipoContingencia, &period.MotivoContingencia, &period.Status, &period.Processing,
		&period.CreatedAt, &period.UpdatedAt,
	)

	if err == nil {
		// Found existing period
		log.Printf("[Contingency] Using existing period %s", period.ID)
		return &period, nil
	}

	if err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to query existing period: %w", err)
	}

	// Create new period
	return s.createPeriod(ctx, companyID, establishmentID, pointOfSaleID, ambiente, tipoContingencia, motivoContingencia)
}

// createPeriod creates a new contingency period
func (s *ContingencyService) createPeriod(
	ctx context.Context,
	companyID, establishmentID, pointOfSaleID, ambiente string,
	tipoContingencia int,
	motivoContingencia string,
) (*models.ContingencyPeriod, error) {
	// Use El Salvador timezone
	loc, err := time.LoadLocation("America/El_Salvador")
	if err != nil {
		loc = time.FixedZone("CST", -6*60*60)
	}
	now := time.Now().In(loc)

	periodID := strings.ToUpper(uuid.New().String())

	query := `
		INSERT INTO contingency_periods (
			id, company_id, establishment_id, point_of_sale_id, ambiente,
			f_inicio, h_inicio, tipo_contingencia, motivo_contingencia, status
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, 'active')
		RETURNING id, company_id, establishment_id, point_of_sale_id, ambiente,
				  f_inicio, h_inicio, f_fin, h_fin,
				  tipo_contingencia, motivo_contingencia, status, processing,
				  created_at, updated_at
	`

	var period models.ContingencyPeriod
	err = s.db.QueryRowContext(ctx, query,
		periodID,
		companyID,
		establishmentID,
		pointOfSaleID,
		ambiente,
		now.Format("2006-01-02"), // f_inicio (DATE)
		now.Format("15:04:05"),   // h_inicio (TIME)
		tipoContingencia,
		motivoContingencia,
	).Scan(
		&period.ID, &period.CompanyID, &period.EstablishmentID, &period.PointOfSaleID,
		&period.Ambiente, &period.FInicio, &period.HInicio, &period.FFin, &period.HFin,
		&period.TipoContingencia, &period.MotivoContingencia, &period.Status, &period.Processing,
		&period.CreatedAt, &period.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create period: %w", err)
	}

	log.Printf("[Contingency] ✅ Created new period %s (tipo: %d)", period.ID, tipoContingencia)
	return &period, nil
}

// GetPeriodByID retrieves a contingency period by ID
func (s *ContingencyService) GetPeriodByID(ctx context.Context, periodID string) (*models.ContingencyPeriod, error) {
	query := `
		SELECT id, company_id, establishment_id, point_of_sale_id, ambiente,
			   f_inicio, h_inicio, f_fin, h_fin,
			   tipo_contingencia, motivo_contingencia, status, processing,
			   created_at, updated_at
		FROM contingency_periods
		WHERE id = $1
	`

	var period models.ContingencyPeriod
	err := s.db.QueryRowContext(ctx, query, periodID).Scan(
		&period.ID, &period.CompanyID, &period.EstablishmentID, &period.PointOfSaleID,
		&period.Ambiente, &period.FInicio, &period.HInicio, &period.FFin, &period.HFin,
		&period.TipoContingencia, &period.MotivoContingencia, &period.Status, &period.Processing,
		&period.CreatedAt, &period.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("period not found: %s", periodID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get period: %w", err)
	}

	return &period, nil
}

// GetInvoicesForPeriod retrieves all invoices in a contingency period
func (s *ContingencyService) GetInvoicesForPeriod(ctx context.Context, periodID string) ([]models.Invoice, error) {
	query := `
		SELECT id, company_id, establishment_id, point_of_sale_id,
			   invoice_number, dte_type, dte_codigo_generacion,
			   contingency_period_id, contingency_event_id, lote_id,
			   dte_transmission_status, dte_unsigned, dte_signed,
			   dte_sello_recibido, hacienda_observaciones, signature_retry_count,
			   finalized_at
		FROM invoices
		WHERE contingency_period_id = $1
		ORDER BY finalized_at ASC
	`

	rows, err := s.db.QueryContext(ctx, query, periodID)
	if err != nil {
		return nil, fmt.Errorf("failed to query invoices: %w", err)
	}
	defer rows.Close()

	var invoices []models.Invoice
	for rows.Next() {
		var inv models.Invoice
		var haciendaObs pq.StringArray

		err := rows.Scan(
			&inv.ID, &inv.CompanyID, &inv.EstablishmentID, &inv.PointOfSaleID,
			&inv.InvoiceNumber, &inv.DteType, &inv.DteCodigoGeneracion,
			&inv.ContingencyPeriodID, &inv.ContingencyEventID, &inv.LoteID,
			&inv.DteTransmissionStatus, &inv.DteUnsigned, &inv.DteSigned,
			&inv.DteSello, &haciendaObs, &inv.SignatureRetryCount,
			&inv.FinalizedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan invoice: %w", err)
		}

		inv.HaciendaObservaciones = haciendaObs
		invoices = append(invoices, inv)
	}

	return invoices, nil
}

// GetUnreportedInvoicesForPeriod gets invoices not yet in an event
func (s *ContingencyService) GetUnreportedInvoicesForPeriod(ctx context.Context, periodID string, limit int) ([]models.Invoice, error) {
	query := `
		SELECT id, company_id, establishment_id, point_of_sale_id,
			   invoice_number, dte_type, dte_codigo_generacion,
			   contingency_period_id, contingency_event_id, lote_id,
			   dte_transmission_status, dte_unsigned, dte_signed,
			   dte_sello_recibido, hacienda_observaciones, signature_retry_count,
			   finalized_at
		FROM invoices
		WHERE contingency_period_id = $1
		  AND contingency_event_id IS NULL
		  AND dte_transmission_status IN ($2, $3, $4)
		ORDER BY finalized_at ASC
		LIMIT $5
	`

	rows, err := s.db.QueryContext(ctx, query,
		periodID,
		models.DTEStatusPendingSignature,
		models.DTEStatusFailedRetry,
		models.DTEStatusContingencyQueue,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query unreported invoices: %w", err)
	}
	defer rows.Close()

	var invoices []models.Invoice
	for rows.Next() {
		var inv models.Invoice
		var haciendaObs pq.StringArray

		err := rows.Scan(
			&inv.ID, &inv.CompanyID, &inv.EstablishmentID, &inv.PointOfSaleID,
			&inv.InvoiceNumber, &inv.DteType, &inv.DteCodigoGeneracion,
			&inv.ContingencyPeriodID, &inv.ContingencyEventID, &inv.LoteID,
			&inv.DteTransmissionStatus, &inv.DteUnsigned, &inv.DteSigned,
			&inv.DteSello, &haciendaObs, &inv.SignatureRetryCount,
			&inv.FinalizedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan invoice: %w", err)
		}

		inv.HaciendaObservaciones = haciendaObs
		invoices = append(invoices, inv)
	}

	return invoices, nil
}

// UpdateInvoiceSignature updates an invoice with a new signature
func (s *ContingencyService) UpdateInvoiceSignature(ctx context.Context, invoiceID string, signedDTE string) error {
	query := `
		UPDATE invoices
		SET dte_signed = $1,
			dte_transmission_status = $2,
			signature_retry_count = 0
		WHERE id = $3
	`

	_, err := s.db.ExecContext(ctx, query, signedDTE, models.DTEStatusContingencyQueue, invoiceID)
	return err
}

// IncrementSignatureRetryCount increments the retry count for an invoice
func (s *ContingencyService) IncrementSignatureRetryCount(ctx context.Context, invoiceID string) (int, error) {
	query := `
		UPDATE invoices
		SET signature_retry_count = signature_retry_count + 1
		WHERE id = $1
		RETURNING signature_retry_count
	`

	var count int
	err := s.db.QueryRowContext(ctx, query, invoiceID).Scan(&count)
	return count, err
}

// ClosePeriod closes a period (sets f_fin, h_fin, status='reporting')
func (s *ContingencyService) ClosePeriod(ctx context.Context, periodID string) error {
	loc, _ := time.LoadLocation("America/El_Salvador")
	if loc == nil {
		loc = time.FixedZone("CST", -6*60*60)
	}
	now := time.Now().In(loc)

	query := `
		UPDATE contingency_periods
		SET f_fin = $1,
			h_fin = $2,
			status = 'reporting'
		WHERE id = $3
		  AND status = 'active'
	`

	_, err := s.db.ExecContext(ctx, query,
		now.Format("2006-01-02"),
		now.Format("15:04:05"),
		periodID,
	)
	return err
}

// CompletePeriod marks a period as completed
func (s *ContingencyService) CompletePeriod(ctx context.Context, periodID string) error {
	query := `
		UPDATE contingency_periods
		SET status = 'completed'
		WHERE id = $1
	`
	_, err := s.db.ExecContext(ctx, query, periodID)
	return err
}

// CreateContingencyEvent creates a new contingency event
func (s *ContingencyService) CreateContingencyEvent(
	ctx context.Context,
	period *models.ContingencyPeriod,
	eventJSON []byte,
	eventSigned string,
	codigoGeneracion string,
) (*models.ContingencyEvent, error) {
	eventID := strings.ToUpper(uuid.New().String())

	query := `
		INSERT INTO contingency_events (
			id, contingency_period_id, codigo_generacion,
			company_id, establishment_id, point_of_sale_id, ambiente,
			event_json, event_signed
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, contingency_period_id, codigo_generacion,
				  company_id, establishment_id, point_of_sale_id, ambiente,
				  event_json, event_signed, estado, sello_recibido, hacienda_response,
				  submitted_at, accepted_at, created_at
	`

	var event models.ContingencyEvent
	err := s.db.QueryRowContext(ctx, query,
		eventID,
		period.ID,
		codigoGeneracion,
		period.CompanyID,
		period.EstablishmentID,
		period.PointOfSaleID,
		period.Ambiente,
		eventJSON,
		eventSigned,
	).Scan(
		&event.ID, &event.ContingencyPeriodID, &event.CodigoGeneracion,
		&event.CompanyID, &event.EstablishmentID, &event.PointOfSaleID, &event.Ambiente,
		&event.EventJSON, &event.EventSigned, &event.Estado, &event.SelloRecibido,
		&event.HaciendaResponse, &event.SubmittedAt, &event.AcceptedAt, &event.CreatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create event: %w", err)
	}

	return &event, nil
}

// UpdateEventWithHaciendaResponse updates event after Hacienda response
func (s *ContingencyService) UpdateEventWithHaciendaResponse(
	ctx context.Context,
	eventID string,
	estado string,
	selloRecibido string,
	haciendaResponse []byte,
) error {
	now := time.Now()

	query := `
		UPDATE contingency_events
		SET estado = $1,
			sello_recibido = $2,
			hacienda_response = $3,
			submitted_at = $4,
			accepted_at = CASE WHEN $1 = 'RECIBIDO' THEN $4 ELSE NULL END
		WHERE id = $5
	`

	_, err := s.db.ExecContext(ctx, query,
		estado,
		selloRecibido,
		haciendaResponse,
		now,
		eventID,
	)
	return err
}

// LinkInvoicesToEvent links invoices to a contingency event
func (s *ContingencyService) LinkInvoicesToEvent(ctx context.Context, invoiceIDs []string, eventID string) error {
	if len(invoiceIDs) == 0 {
		return nil
	}

	query := `
		UPDATE invoices
		SET contingency_event_id = $1
		WHERE id = ANY($2)
	`

	_, err := s.db.ExecContext(ctx, query, eventID, pq.Array(invoiceIDs))
	return err
}

// CreateLote creates a new lote for batch submission
func (s *ContingencyService) CreateLote(
	ctx context.Context,
	event *models.ContingencyEvent,
	dteCount int,
) (*models.Lote, error) {
	loteID := strings.ToUpper(uuid.New().String())

	query := `
		INSERT INTO lotes (
			id, contingency_event_id, company_id, establishment_id, point_of_sale_id, dte_count, status
		) VALUES ($1, $2, $3, $4, $5, $6, 'pending')
		RETURNING id, contingency_event_id, codigo_lote, company_id, establishment_id, point_of_sale_id,
				  dte_count, status, processing, hacienda_response,
				  submitted_at, last_polled_at, completed_at, created_at, updated_at
	`

	var lote models.Lote
	err := s.db.QueryRowContext(ctx, query,
		loteID,
		event.ID,
		event.CompanyID,
		event.EstablishmentID,
		event.PointOfSaleID,
		dteCount,
	).Scan(
		&lote.ID, &lote.ContingencyEventID, &lote.CodigoLote,
		&lote.CompanyID, &lote.EstablishmentID, &lote.PointOfSaleID,
		&lote.DTECount, &lote.Status, &lote.Processing, &lote.HaciendaResponse,
		&lote.SubmittedAt, &lote.LastPolledAt, &lote.CompletedAt, &lote.CreatedAt, &lote.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create lote: %w", err)
	}

	return &lote, nil
}

// LinkInvoicesToLote links invoices to a lote
func (s *ContingencyService) LinkInvoicesToLote(ctx context.Context, invoiceIDs []string, loteID string) error {
	if len(invoiceIDs) == 0 {
		return nil
	}

	query := `
		UPDATE invoices
		SET lote_id = $1
		WHERE id = ANY($2)
	`

	_, err := s.db.ExecContext(ctx, query, loteID, pq.Array(invoiceIDs))
	return err
}

// UpdateLoteSubmitted updates lote after submission to Hacienda
func (s *ContingencyService) UpdateLoteSubmitted(ctx context.Context, loteID string, codigoLote string) error {
	query := `
		UPDATE lotes
		SET codigo_lote = $1,
			status = 'submitted',
			submitted_at = NOW()
		WHERE id = $2
	`

	_, err := s.db.ExecContext(ctx, query, codigoLote, loteID)
	return err
}

// UpdateLoteCompleted marks a lote as completed
func (s *ContingencyService) UpdateLoteCompleted(ctx context.Context, loteID string) error {
	query := `
		UPDATE lotes
		SET status = 'completed',
			completed_at = NOW()
		WHERE id = $1
	`

	_, err := s.db.ExecContext(ctx, query, loteID)
	return err
}

// UpdateInvoiceFromHaciendaResult updates invoice based on Hacienda lote result
func (s *ContingencyService) UpdateInvoiceFromHaciendaResult(
	ctx context.Context,
	codigoGeneracion string,
	status string,
	selloRecibido string,
	observaciones []string,
) error {
	query := `
		UPDATE invoices
		SET dte_transmission_status = $1,
			dte_sello_recibido = $2,
			hacienda_observaciones = $3
		WHERE dte_codigo_generacion = $4
	`

	_, err := s.db.ExecContext(ctx, query,
		status,
		selloRecibido,
		pq.Array(observaciones),
		codigoGeneracion,
	)
	return err
}

// GetCompanyAmbiente gets the dte_ambiente for a company
func (s *ContingencyService) GetCompanyAmbiente(ctx context.Context, companyID string) (string, error) {
	var ambiente string
	query := `SELECT COALESCE(dte_ambiente, '00') FROM companies WHERE id = $1`
	err := s.db.QueryRowContext(ctx, query, companyID).Scan(&ambiente)
	if err != nil {
		return "00", err
	}
	return ambiente, nil
}

// ClaimPeriodsForProcessing claims periods for worker processing (prevents double processing)
func (s *ContingencyService) ClaimPeriodsForProcessing(ctx context.Context, limit int) ([]models.ContingencyPeriod, error) {
	query := `
		UPDATE contingency_periods
		SET processing = true
		WHERE id IN (
			SELECT id FROM contingency_periods
			WHERE status IN ('active', 'reporting')
			  AND processing = false
			ORDER BY created_at ASC
			LIMIT $1
			FOR UPDATE SKIP LOCKED
		)
		RETURNING id, company_id, establishment_id, point_of_sale_id, ambiente,
				  f_inicio, h_inicio, f_fin, h_fin,
				  tipo_contingencia, motivo_contingencia, status, processing,
				  created_at, updated_at
	`

	rows, err := s.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to claim periods: %w", err)
	}
	defer rows.Close()

	var periods []models.ContingencyPeriod
	for rows.Next() {
		var p models.ContingencyPeriod
		err := rows.Scan(
			&p.ID, &p.CompanyID, &p.EstablishmentID, &p.PointOfSaleID, &p.Ambiente,
			&p.FInicio, &p.HInicio, &p.FFin, &p.HFin,
			&p.TipoContingencia, &p.MotivoContingencia, &p.Status, &p.Processing,
			&p.CreatedAt, &p.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan period: %w", err)
		}
		periods = append(periods, p)
	}

	return periods, nil
}

// ReleasePeriodProcessing releases the processing lock on a period
func (s *ContingencyService) ReleasePeriodProcessing(ctx context.Context, periodID string) error {
	query := `UPDATE contingency_periods SET processing = false WHERE id = $1`
	_, err := s.db.ExecContext(ctx, query, periodID)
	return err
}

// ClaimLotesForProcessing claims lotes for worker processing
func (s *ContingencyService) ClaimLotesForProcessing(ctx context.Context, status string, limit int) ([]models.Lote, error) {
	query := `
		UPDATE lotes
		SET processing = true
		WHERE id IN (
			SELECT id FROM lotes
			WHERE status = $1
			  AND processing = false
			ORDER BY created_at ASC
			LIMIT $2
			FOR UPDATE SKIP LOCKED
		)
		RETURNING id, contingency_event_id, codigo_lote, company_id, establishment_id, point_of_sale_id,
				  dte_count, status, processing, hacienda_response,
				  submitted_at, last_polled_at, completed_at, created_at, updated_at
	`

	rows, err := s.db.QueryContext(ctx, query, status, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to claim lotes: %w", err)
	}
	defer rows.Close()

	var lotes []models.Lote
	for rows.Next() {
		var l models.Lote
		err := rows.Scan(
			&l.ID, &l.ContingencyEventID, &l.CodigoLote,
			&l.CompanyID, &l.EstablishmentID, &l.PointOfSaleID,
			&l.DTECount, &l.Status, &l.Processing, &l.HaciendaResponse,
			&l.SubmittedAt, &l.LastPolledAt, &l.CompletedAt, &l.CreatedAt, &l.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan lote: %w", err)
		}
		lotes = append(lotes, l)
	}

	return lotes, nil
}

// ReleaseLoteProcessing releases the processing lock on a lote
func (s *ContingencyService) ReleaseLoteProcessing(ctx context.Context, loteID string) error {
	query := `UPDATE lotes SET processing = false WHERE id = $1`
	_, err := s.db.ExecContext(ctx, query, loteID)
	return err
}

// UpdateLoteLastPolled updates the last_polled_at timestamp
func (s *ContingencyService) UpdateLoteLastPolled(ctx context.Context, loteID string) error {
	query := `UPDATE lotes SET last_polled_at = NOW() WHERE id = $1`
	_, err := s.db.ExecContext(ctx, query, loteID)
	return err
}

// GetInvoicesForLote gets all invoices in a lote
func (s *ContingencyService) GetInvoicesForLote(ctx context.Context, loteID string) ([]models.Invoice, error) {
	query := `
		SELECT id, company_id, establishment_id, point_of_sale_id,
			   invoice_number, dte_type, dte_codigo_generacion,
			   contingency_period_id, contingency_event_id, lote_id,
			   dte_transmission_status, dte_unsigned, dte_signed,
			   dte_sello_recibido, hacienda_observaciones, signature_retry_count
		FROM invoices
		WHERE lote_id = $1
		ORDER BY finalized_at ASC
	`

	rows, err := s.db.QueryContext(ctx, query, loteID)
	if err != nil {
		return nil, fmt.Errorf("failed to query invoices: %w", err)
	}
	defer rows.Close()

	var invoices []models.Invoice
	for rows.Next() {
		var inv models.Invoice
		var haciendaObs pq.StringArray

		err := rows.Scan(
			&inv.ID, &inv.CompanyID, &inv.EstablishmentID, &inv.PointOfSaleID,
			&inv.InvoiceNumber, &inv.DteType, &inv.DteCodigoGeneracion,
			&inv.ContingencyPeriodID, &inv.ContingencyEventID, &inv.LoteID,
			&inv.DteTransmissionStatus, &inv.DteUnsigned, &inv.DteSigned,
			&inv.DteSello, &haciendaObs, &inv.SignatureRetryCount,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan invoice: %w", err)
		}

		inv.HaciendaObservaciones = haciendaObs
		invoices = append(invoices, inv)
	}

	return invoices, nil
}

// CheckLoteCompletion checks if all invoices in a lote are finalized
func (s *ContingencyService) CheckLoteCompletion(ctx context.Context, loteID string) (bool, error) {
	query := `
		SELECT COUNT(*) FROM invoices
		WHERE lote_id = $1
		  AND dte_transmission_status NOT IN ($2, $3)
	`

	var remaining int
	err := s.db.QueryRowContext(ctx, query, loteID, models.DTEStatusProcesado, models.DTEStatusRechazado).Scan(&remaining)
	if err != nil {
		return false, err
	}

	return remaining == 0, nil
}

// CheckPeriodCompletion checks if all invoices in a period are finalized
func (s *ContingencyService) CheckPeriodCompletion(ctx context.Context, periodID string) (bool, error) {
	query := `
		SELECT COUNT(*) FROM invoices
		WHERE contingency_period_id = $1
		  AND dte_transmission_status NOT IN ($2, $3)
	`

	var remaining int
	err := s.db.QueryRowContext(ctx, query, periodID, models.DTEStatusProcesado, models.DTEStatusRechazado).Scan(&remaining)
	if err != nil {
		return false, err
	}

	return remaining == 0, nil
}
