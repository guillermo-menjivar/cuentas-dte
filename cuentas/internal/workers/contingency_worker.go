package workers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"cuentas/internal/hacienda"
	"cuentas/internal/models"
	"cuentas/internal/services"
	"cuentas/internal/services/firmador"

	"github.com/redis/go-redis/v9"
)

// ContingencyWorker handles background processing of contingency periods
type ContingencyWorker struct {
	db                 *sql.DB
	redis              *redis.Client
	contingencyService *services.ContingencyService
	eventBuilder       *services.ContingencyEventBuilder
	firmador           *firmador.Client
	haciendaClient     *hacienda.Client
	haciendaService    *services.HaciendaService
	vaultService       *services.VaultService

	// Configuration
	signatureRetryInterval time.Duration
	periodCheckInterval    time.Duration
	loteSubmitInterval     time.Duration
	lotePollInterval       time.Duration
	maxSignatureRetries    int
	maxDTEsPerLote         int
}

// WorkerConfig holds configuration for the contingency worker
type WorkerConfig struct {
	SignatureRetryInterval time.Duration
	PeriodCheckInterval    time.Duration
	LoteSubmitInterval     time.Duration
	LotePollInterval       time.Duration
	MaxSignatureRetries    int
	MaxDTEsPerLote         int
}

// DefaultWorkerConfig returns sensible defaults
func DefaultWorkerConfig() *WorkerConfig {
	return &WorkerConfig{
		SignatureRetryInterval: 5 * time.Minute,
		PeriodCheckInterval:    2 * time.Minute,
		LoteSubmitInterval:     1 * time.Minute,
		LotePollInterval:       30 * time.Second,
		MaxSignatureRetries:    5,
		MaxDTEsPerLote:         50, // Hacienda limit is higher, but 50 is safe
	}
}

// NewContingencyWorker creates a new contingency worker
func NewContingencyWorker(
	db *sql.DB,
	redisClient *redis.Client,
	contingencyService *services.ContingencyService,
	firmadorClient *firmador.Client,
	haciendaClient *hacienda.Client,
	haciendaService *services.HaciendaService,
	vaultService *services.VaultService,
	config *WorkerConfig,
) *ContingencyWorker {
	if config == nil {
		config = DefaultWorkerConfig()
	}

	return &ContingencyWorker{
		db:                     db,
		redis:                  redisClient,
		contingencyService:     contingencyService,
		eventBuilder:           services.NewContingencyEventBuilder(db),
		firmador:               firmadorClient,
		haciendaClient:         haciendaClient,
		haciendaService:        haciendaService,
		vaultService:           vaultService,
		signatureRetryInterval: config.SignatureRetryInterval,
		periodCheckInterval:    config.PeriodCheckInterval,
		loteSubmitInterval:     config.LoteSubmitInterval,
		lotePollInterval:       config.LotePollInterval,
		maxSignatureRetries:    config.MaxSignatureRetries,
		maxDTEsPerLote:         config.MaxDTEsPerLote,
	}
}

// Start begins all worker goroutines
func (w *ContingencyWorker) Start(ctx context.Context) {
	log.Println("[ContingencyWorker] Starting workers...")

	// Worker 1: Retry signatures for unsigned invoices
	go w.runSignatureRetryWorker(ctx)

	// Worker 2: Check if services recovered and close periods
	go w.runPeriodCheckWorker(ctx)

	// Worker 3: Submit pending lotes to Hacienda
	go w.runLoteSubmitWorker(ctx)

	// Worker 4: Poll submitted lotes for results
	go w.runLotePollWorker(ctx)

	log.Println("[ContingencyWorker] All workers started")
}

// =============================================================================
// Worker 1: Signature Retry Worker
// =============================================================================

func (w *ContingencyWorker) runSignatureRetryWorker(ctx context.Context) {
	log.Println("[SignatureRetryWorker] Started")
	ticker := time.NewTicker(w.signatureRetryInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("[SignatureRetryWorker] Shutting down")
			return
		case <-ticker.C:
			w.processUnsignedInvoices(ctx)
		}
	}
}

func (w *ContingencyWorker) processUnsignedInvoices(ctx context.Context) {
	log.Println("[SignatureRetryWorker] Checking for unsigned invoices...")

	// Get invoices with pending_signature status
	query := `
		SELECT i.id, i.company_id, i.dte_unsigned, i.signature_retry_count
		FROM invoices i
		WHERE i.dte_transmission_status = $1
		  AND i.signature_retry_count < $2
		ORDER BY i.finalized_at ASC
		LIMIT 50
	`

	rows, err := w.db.QueryContext(ctx, query, models.DTEStatusPendingSignature, w.maxSignatureRetries)
	if err != nil {
		log.Printf("[SignatureRetryWorker] Failed to query invoices: %v", err)
		return
	}
	defer rows.Close()

	type unsignedInvoice struct {
		ID                  string
		CompanyID           string
		DteUnsigned         []byte
		SignatureRetryCount int
	}

	var invoices []unsignedInvoice
	for rows.Next() {
		var inv unsignedInvoice
		if err := rows.Scan(&inv.ID, &inv.CompanyID, &inv.DteUnsigned, &inv.SignatureRetryCount); err != nil {
			log.Printf("[SignatureRetryWorker] Failed to scan invoice: %v", err)
			continue
		}
		invoices = append(invoices, inv)
	}

	if len(invoices) == 0 {
		log.Println("[SignatureRetryWorker] No unsigned invoices to process")
		return
	}

	log.Printf("[SignatureRetryWorker] Found %d unsigned invoices to retry", len(invoices))

	for _, inv := range invoices {
		w.retrySignature(ctx, inv.ID, inv.CompanyID, inv.DteUnsigned)
	}
}

func (w *ContingencyWorker) retrySignature(ctx context.Context, invoiceID, companyID string, dteUnsigned []byte) {
	log.Printf("[SignatureRetryWorker] Retrying signature for invoice %s", invoiceID)

	// Load credentials
	creds, err := w.loadCredentials(ctx, companyID)
	if err != nil {
		log.Printf("[SignatureRetryWorker] Failed to load credentials for %s: %v", companyID, err)
		w.contingencyService.IncrementSignatureRetryCount(ctx, invoiceID)
		return
	}

	// Parse the unsigned DTE
	var dteDoc map[string]interface{}
	if err := json.Unmarshal(dteUnsigned, &dteDoc); err != nil {
		log.Printf("[SignatureRetryWorker] Failed to parse DTE for %s: %v", invoiceID, err)
		w.contingencyService.IncrementSignatureRetryCount(ctx, invoiceID)
		return
	}

	// Attempt to sign
	signedDTE, err := w.firmador.Sign(ctx, creds.NIT, creds.Password, dteDoc)
	if err != nil {
		log.Printf("[SignatureRetryWorker] Firmador still unavailable for %s: %v", invoiceID, err)
		w.contingencyService.IncrementSignatureRetryCount(ctx, invoiceID)
		return
	}

	// Success! Update the invoice
	err = w.contingencyService.UpdateInvoiceSignature(ctx, invoiceID, signedDTE)
	if err != nil {
		log.Printf("[SignatureRetryWorker] Failed to save signature for %s: %v", invoiceID, err)
		return
	}

	log.Printf("[SignatureRetryWorker] ✅ Successfully signed invoice %s", invoiceID)
}

// =============================================================================
// Worker 2: Period Check Worker
// =============================================================================

func (w *ContingencyWorker) runPeriodCheckWorker(ctx context.Context) {
	log.Println("[PeriodCheckWorker] Started")
	ticker := time.NewTicker(w.periodCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("[PeriodCheckWorker] Shutting down")
			return
		case <-ticker.C:
			w.checkAndProcessPeriods(ctx)
		}
	}
}

func (w *ContingencyWorker) checkAndProcessPeriods(ctx context.Context) {
	log.Println("[PeriodCheckWorker] Checking active periods...")

	// Claim periods for processing (prevents double processing)
	periods, err := w.contingencyService.ClaimPeriodsForProcessing(ctx, 10)
	if err != nil {
		log.Printf("[PeriodCheckWorker] Failed to claim periods: %v", err)
		return
	}

	if len(periods) == 0 {
		log.Println("[PeriodCheckWorker] No periods to process")
		return
	}

	log.Printf("[PeriodCheckWorker] Processing %d periods", len(periods))

	for _, period := range periods {
		w.processPeriod(ctx, &period)
		// Always release the processing lock
		w.contingencyService.ReleasePeriodProcessing(ctx, period.ID)
	}
}

func (w *ContingencyWorker) processPeriod(ctx context.Context, period *models.ContingencyPeriod) {
	log.Printf("[PeriodCheckWorker] Processing period %s (status: %s)", period.ID, period.Status)

	switch period.Status {
	case models.PeriodStatusActive:
		w.processActivePeriod(ctx, period)
	case models.PeriodStatusReporting:
		w.processReportingPeriod(ctx, period)
	}
}

func (w *ContingencyWorker) processActivePeriod(ctx context.Context, period *models.ContingencyPeriod) {
	// Check if services have recovered by attempting a test authentication
	_, err := w.haciendaService.AuthenticateCompany(ctx, period.CompanyID)
	if err != nil {
		log.Printf("[PeriodCheckWorker] Services still down for period %s: %v", period.ID, err)
		return
	}

	log.Printf("[PeriodCheckWorker] ✅ Services recovered! Closing period %s", period.ID)

	// Close the period
	if err := w.contingencyService.ClosePeriod(ctx, period.ID); err != nil {
		log.Printf("[PeriodCheckWorker] Failed to close period %s: %v", period.ID, err)
		return
	}

	// Refresh period data
	period, err = w.contingencyService.GetPeriodByID(ctx, period.ID)
	if err != nil {
		log.Printf("[PeriodCheckWorker] Failed to refresh period %s: %v", period.ID, err)
		return
	}

	// Now process as reporting
	w.processReportingPeriod(ctx, period)
}

func (w *ContingencyWorker) processReportingPeriod(ctx context.Context, period *models.ContingencyPeriod) {
	log.Printf("[PeriodCheckWorker] Processing reporting period %s", period.ID)

	// Get unreported invoices (not yet in an event)
	invoices, err := w.contingencyService.GetUnreportedInvoicesForPeriod(ctx, period.ID, 1000)
	if err != nil {
		log.Printf("[PeriodCheckWorker] Failed to get invoices for period %s: %v", period.ID, err)
		return
	}

	if len(invoices) == 0 {
		// Check if period is complete
		complete, err := w.contingencyService.CheckPeriodCompletion(ctx, period.ID)
		if err != nil {
			log.Printf("[PeriodCheckWorker] Failed to check period completion: %v", err)
			return
		}

		if complete {
			log.Printf("[PeriodCheckWorker] ✅ Period %s completed!", period.ID)
			w.contingencyService.CompletePeriod(ctx, period.ID)
		}
		return
	}

	// Filter to only signed invoices
	var signedInvoices []models.Invoice
	for _, inv := range invoices {
		if inv.DteSigned != nil && *inv.DteSigned != "" {
			signedInvoices = append(signedInvoices, inv)
		}
	}

	if len(signedInvoices) == 0 {
		log.Printf("[PeriodCheckWorker] No signed invoices ready for period %s", period.ID)
		return
	}

	// Build and submit contingency event
	w.submitContingencyEvent(ctx, period, signedInvoices)
}

func (w *ContingencyWorker) submitContingencyEvent(ctx context.Context, period *models.ContingencyPeriod, invoices []models.Invoice) {
	log.Printf("[PeriodCheckWorker] Building contingency event for %d invoices", len(invoices))

	// Build event JSON
	eventJSON, codigoGeneracion, err := w.eventBuilder.BuildEventJSON(ctx, period, invoices)
	if err != nil {
		log.Printf("[PeriodCheckWorker] Failed to build event: %v", err)
		return
	}

	// Load credentials for signing
	creds, err := w.loadCredentials(ctx, period.CompanyID)
	if err != nil {
		log.Printf("[PeriodCheckWorker] Failed to load credentials: %v", err)
		return
	}

	// Parse event for signing
	var eventDoc map[string]interface{}
	if err := json.Unmarshal(eventJSON, &eventDoc); err != nil {
		log.Printf("[PeriodCheckWorker] Failed to parse event JSON: %v", err)
		return
	}

	// Sign the event
	signedEvent, err := w.firmador.Sign(ctx, creds.NIT, creds.Password, eventDoc)
	if err != nil {
		log.Printf("[PeriodCheckWorker] Failed to sign event: %v", err)
		return
	}

	// Create event record
	event, err := w.contingencyService.CreateContingencyEvent(ctx, period, eventJSON, signedEvent, codigoGeneracion)
	if err != nil {
		log.Printf("[PeriodCheckWorker] Failed to create event record: %v", err)
		return
	}

	// Link invoices to event
	invoiceIDs := make([]string, len(invoices))
	for i, inv := range invoices {
		invoiceIDs[i] = inv.ID
	}
	if err := w.contingencyService.LinkInvoicesToEvent(ctx, invoiceIDs, event.ID); err != nil {
		log.Printf("[PeriodCheckWorker] Failed to link invoices to event: %v", err)
		return
	}

	// Submit event to Hacienda
	w.submitEventToHacienda(ctx, period, event)

	// Create lote for batch DTE submission
	w.createLoteForEvent(ctx, event, invoices)
}

func (w *ContingencyWorker) submitEventToHacienda(ctx context.Context, period *models.ContingencyPeriod, event *models.ContingencyEvent) {
	log.Printf("[PeriodCheckWorker] Submitting contingency event %s to Hacienda", event.ID)

	// Authenticate
	authResponse, err := w.haciendaService.AuthenticateCompany(ctx, period.CompanyID)
	if err != nil {
		log.Printf("[PeriodCheckWorker] Failed to authenticate for event submission: %v", err)
		return
	}

	// Submit contingency event (type 15)
	response, err := w.haciendaClient.SubmitDTE(
		ctx,
		authResponse.Body.Token,
		period.Ambiente,
		"15", // Evento de Contingencia
		event.CodigoGeneracion,
		event.EventSigned,
	)

	if err != nil {
		log.Printf("[PeriodCheckWorker] Failed to submit event to Hacienda: %v", err)
		// Save error response
		responseJSON, _ := json.Marshal(map[string]string{"error": err.Error()})
		w.contingencyService.UpdateEventWithHaciendaResponse(ctx, event.ID, "ERROR", "", responseJSON)
		return
	}

	// Save success response
	responseJSON, _ := json.Marshal(response)
	w.contingencyService.UpdateEventWithHaciendaResponse(
		ctx,
		event.ID,
		response.Estado,
		response.SelloRecibido,
		responseJSON,
	)

	log.Printf("[PeriodCheckWorker] ✅ Contingency event %s accepted (sello: %s)", event.ID, response.SelloRecibido)
}

func (w *ContingencyWorker) createLoteForEvent(ctx context.Context, event *models.ContingencyEvent, invoices []models.Invoice) {
	log.Printf("[PeriodCheckWorker] Creating lote for %d DTEs", len(invoices))

	lote, err := w.contingencyService.CreateLote(ctx, event, len(invoices))
	if err != nil {
		log.Printf("[PeriodCheckWorker] Failed to create lote: %v", err)
		return
	}

	// Link invoices to lote
	invoiceIDs := make([]string, len(invoices))
	for i, inv := range invoices {
		invoiceIDs[i] = inv.ID
	}
	if err := w.contingencyService.LinkInvoicesToLote(ctx, invoiceIDs, lote.ID); err != nil {
		log.Printf("[PeriodCheckWorker] Failed to link invoices to lote: %v", err)
		return
	}

	log.Printf("[PeriodCheckWorker] ✅ Created lote %s", lote.ID)
}

// =============================================================================
// Worker 3: Lote Submit Worker
// =============================================================================

func (w *ContingencyWorker) runLoteSubmitWorker(ctx context.Context) {
	log.Println("[LoteSubmitWorker] Started")
	ticker := time.NewTicker(w.loteSubmitInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("[LoteSubmitWorker] Shutting down")
			return
		case <-ticker.C:
			w.submitPendingLotes(ctx)
		}
	}
}

func (w *ContingencyWorker) submitPendingLotes(ctx context.Context) {
	log.Println("[LoteSubmitWorker] Checking for pending lotes...")

	lotes, err := w.contingencyService.ClaimLotesForProcessing(ctx, models.LoteStatusPending, 5)
	if err != nil {
		log.Printf("[LoteSubmitWorker] Failed to claim lotes: %v", err)
		return
	}

	if len(lotes) == 0 {
		log.Println("[LoteSubmitWorker] No pending lotes")
		return
	}

	log.Printf("[LoteSubmitWorker] Submitting %d lotes", len(lotes))

	for _, lote := range lotes {
		w.submitLote(ctx, &lote)
		w.contingencyService.ReleaseLoteProcessing(ctx, lote.ID)
	}
}

func (w *ContingencyWorker) submitLote(ctx context.Context, lote *models.Lote) {
	log.Printf("[LoteSubmitWorker] Submitting lote %s", lote.ID)

	// Get invoices in lote
	invoices, err := w.contingencyService.GetInvoicesForLote(ctx, lote.ID)
	if err != nil {
		log.Printf("[LoteSubmitWorker] Failed to get invoices for lote %s: %v", lote.ID, err)
		return
	}

	// Collect signed DTEs
	var signedDTEs []string
	for _, inv := range invoices {
		if inv.DteSigned != nil && *inv.DteSigned != "" {
			signedDTEs = append(signedDTEs, *inv.DteSigned)
		}
	}

	if len(signedDTEs) == 0 {
		log.Printf("[LoteSubmitWorker] No signed DTEs in lote %s", lote.ID)
		return
	}

	// Get company NIT
	var nit string
	err = w.db.QueryRowContext(ctx, "SELECT nit FROM companies WHERE id = $1", lote.CompanyID).Scan(&nit)
	if err != nil {
		log.Printf("[LoteSubmitWorker] Failed to get company NIT: %v", err)
		return
	}

	// Get ambiente from company
	ambiente, err := w.contingencyService.GetCompanyAmbiente(ctx, lote.CompanyID)
	if err != nil {
		log.Printf("[LoteSubmitWorker] Failed to get ambiente: %v", err)
		return
	}

	// Authenticate
	authResponse, err := w.haciendaService.AuthenticateCompany(ctx, lote.CompanyID)
	if err != nil {
		log.Printf("[LoteSubmitWorker] Failed to authenticate: %v", err)
		return
	}

	// Build lote payload

	// Submit batch
	batchResponse, err := w.haciendaClient.SubmitBatch(
		ctx,
		authResponse.Body.Token,
		ambiente,
		nit,
		signedDTEs,
	)

	if err != nil {
		log.Printf("[LoteSubmitWorker] Failed to submit lote: %v", err)
		return
	}

	// Update lote with codigo_lote from Hacienda
	err = w.contingencyService.UpdateLoteSubmitted(ctx, lote.ID, batchResponse.CodigoLote)
	if err != nil {
		log.Printf("[LoteSubmitWorker] Failed to update lote: %v", err)
		return
	}

	log.Printf("[LoteSubmitWorker] ✅ Lote %s submitted (codigo: %s)", lote.ID, batchResponse.CodigoLote)
}

// =============================================================================
// Worker 4: Lote Poll Worker
// =============================================================================

func (w *ContingencyWorker) runLotePollWorker(ctx context.Context) {
	log.Println("[LotePollWorker] Started")
	ticker := time.NewTicker(w.lotePollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("[LotePollWorker] Shutting down")
			return
		case <-ticker.C:
			w.pollSubmittedLotes(ctx)
		}
	}
}

func (w *ContingencyWorker) pollSubmittedLotes(ctx context.Context) {
	log.Println("[LotePollWorker] Checking submitted lotes...")

	lotes, err := w.contingencyService.ClaimLotesForProcessing(ctx, models.LoteStatusSubmitted, 10)
	if err != nil {
		log.Printf("[LotePollWorker] Failed to claim lotes: %v", err)
		return
	}

	if len(lotes) == 0 {
		log.Println("[LotePollWorker] No submitted lotes to poll")
		return
	}

	log.Printf("[LotePollWorker] Polling %d lotes", len(lotes))

	for _, lote := range lotes {
		w.pollLote(ctx, &lote)
		w.contingencyService.ReleaseLoteProcessing(ctx, lote.ID)
	}
}

func (w *ContingencyWorker) pollLote(ctx context.Context, lote *models.Lote) {
	if lote.CodigoLote == nil || *lote.CodigoLote == "" {
		log.Printf("[LotePollWorker] Lote %s has no codigo_lote, skipping", lote.ID)
		return
	}

	log.Printf("[LotePollWorker] Polling lote %s (codigo: %s)", lote.ID, *lote.CodigoLote)

	// Get ambiente
	ambiente, err := w.contingencyService.GetCompanyAmbiente(ctx, lote.CompanyID)
	if err != nil {
		log.Printf("[LotePollWorker] Failed to get ambiente: %v", err)
		return
	}
	fmt.Println("this is the ambiente = but we are not using it...", ambiente)

	// Authenticate
	authResponse, err := w.haciendaService.AuthenticateCompany(ctx, lote.CompanyID)
	if err != nil {
		log.Printf("[LotePollWorker] Failed to authenticate: %v", err)
		return
	}

	// Query batch status
	queryResponse, err := w.haciendaClient.QueryBatchStatus(
		ctx,
		authResponse.Body.Token,
		*lote.CodigoLote,
	)

	if err != nil {
		log.Printf("[LotePollWorker] Failed to query lote status: %v", err)
		return
	}

	// Update last polled timestamp
	w.contingencyService.UpdateLoteLastPolled(ctx, lote.ID)

	// Process results
	w.processLoteResults(ctx, lote, queryResponse)
}

func (w *ContingencyWorker) processLoteResults(ctx context.Context, lote *models.Lote, response *hacienda.BatchQueryResponse) {
	log.Printf("[LotePollWorker] Processing results for lote %s: %d procesados, %d rechazados",
		lote.ID, len(response.Procesados), len(response.Rechazados))

	// Update processed invoices
	for _, result := range response.Procesados {
		err := w.contingencyService.UpdateInvoiceFromHaciendaResult(
			ctx,
			result.CodigoGeneracion,
			models.DTEStatusProcesado,
			result.SelloRecibido,
			result.Observaciones,
		)
		if err != nil {
			log.Printf("[LotePollWorker] Failed to update processed invoice %s: %v", result.CodigoGeneracion, err)
		} else {
			log.Printf("[LotePollWorker] ✅ Invoice %s procesado", result.CodigoGeneracion)
		}
	}

	// Update rejected invoices
	for _, result := range response.Rechazados {
		err := w.contingencyService.UpdateInvoiceFromHaciendaResult(
			ctx,
			result.CodigoGeneracion,
			models.DTEStatusRechazado,
			"",
			result.Observaciones,
		)
		if err != nil {
			log.Printf("[LotePollWorker] Failed to update rejected invoice %s: %v", result.CodigoGeneracion, err)
		} else {
			log.Printf("[LotePollWorker] ❌ Invoice %s rechazado: %v", result.CodigoGeneracion, result.Observaciones)
		}
	}

	// Check if lote is complete
	complete, err := w.contingencyService.CheckLoteCompletion(ctx, lote.ID)
	if err != nil {
		log.Printf("[LotePollWorker] Failed to check lote completion: %v", err)
		return
	}

	if complete {
		w.contingencyService.UpdateLoteCompleted(ctx, lote.ID)
		log.Printf("[LotePollWorker] ✅ Lote %s completed", lote.ID)
	}
}

// =============================================================================
// Helper Functions
// =============================================================================

type firmadorCredentials struct {
	NIT      string
	Password string
}

func (w *ContingencyWorker) loadCredentials(ctx context.Context, companyID string) (*firmadorCredentials, error) {
	// Get company info
	var nit, firmadorUsername, firmadorPasswordRef string
	query := `SELECT nit, firmador_username, firmador_password_ref FROM companies WHERE id = $1`
	err := w.db.QueryRowContext(ctx, query, companyID).Scan(&nit, &firmadorUsername, &firmadorPasswordRef)
	if err != nil {
		return nil, fmt.Errorf("failed to get company: %w", err)
	}

	// Load password from Vault
	password, err := w.vaultService.GetCompanyPassword(firmadorPasswordRef)
	if err != nil {
		return nil, fmt.Errorf("failed to load firmador password: %w", err)
	}

	return &firmadorCredentials{
		NIT:      nit,
		Password: password,
	}, nil
}
