package handlers

import (
	"database/sql"
	"net/http"

	"cuentas/internal/services"

	"github.com/gin-gonic/gin"
)

// ContingencyHandler handles contingency-related API endpoints
type ContingencyHandler struct {
	contingencyService *services.ContingencyService
}

// NewContingencyHandler creates a new contingency handler
func NewContingencyHandler(contingencyService *services.ContingencyService) *ContingencyHandler {
	return &ContingencyHandler{
		contingencyService: contingencyService,
	}
}

// ListPeriods returns all contingency periods for the company
// GET /v1/contingency/periods
func (h *ContingencyHandler) ListPeriods(c *gin.Context) {
	companyID := c.GetString("company_id")
	if companyID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "company_id header required"})
		return
	}

	db := c.MustGet("db").(*sql.DB)

	query := `
		SELECT id, company_id, establishment_id, point_of_sale_id, ambiente,
			   f_inicio, h_inicio, f_fin, h_fin,
			   tipo_contingencia, motivo_contingencia, status, processing,
			   created_at, updated_at
		FROM contingency_periods
		WHERE company_id = $1
		ORDER BY created_at DESC
		LIMIT 100
	`

	rows, err := db.QueryContext(c.Request.Context(), query, companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query periods"})
		return
	}
	defer rows.Close()

	var periods []gin.H
	for rows.Next() {
		var id, companyID, establishmentID, pointOfSaleID, ambiente string
		var fInicio, hInicio string
		var fFin, hFin, motivoContingencia *string
		var tipoContingencia int
		var status string
		var processing bool
		var createdAt, updatedAt interface{}

		err := rows.Scan(
			&id, &companyID, &establishmentID, &pointOfSaleID, &ambiente,
			&fInicio, &hInicio, &fFin, &hFin,
			&tipoContingencia, &motivoContingencia, &status, &processing,
			&createdAt, &updatedAt,
		)
		if err != nil {
			continue
		}

		periods = append(periods, gin.H{
			"id":                  id,
			"company_id":          companyID,
			"establishment_id":    establishmentID,
			"point_of_sale_id":    pointOfSaleID,
			"ambiente":            ambiente,
			"f_inicio":            fInicio,
			"h_inicio":            hInicio,
			"f_fin":               fFin,
			"h_fin":               hFin,
			"tipo_contingencia":   tipoContingencia,
			"motivo_contingencia": motivoContingencia,
			"status":              status,
			"processing":          processing,
			"created_at":          createdAt,
			"updated_at":          updatedAt,
		})
	}

	if periods == nil {
		periods = []gin.H{}
	}

	c.JSON(http.StatusOK, gin.H{
		"periods": periods,
		"count":   len(periods),
	})
}

// GetPeriod returns a single contingency period with details
// GET /v1/contingency/periods/:id
func (h *ContingencyHandler) GetPeriod(c *gin.Context) {
	companyID := c.GetString("company_id")
	if companyID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "company_id header required"})
		return
	}

	periodID := c.Param("id")
	if periodID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "period id required"})
		return
	}

	db := c.MustGet("db").(*sql.DB)

	// Get period
	query := `
		SELECT id, company_id, establishment_id, point_of_sale_id, ambiente,
			   f_inicio, h_inicio, f_fin, h_fin,
			   tipo_contingencia, motivo_contingencia, status, processing,
			   created_at, updated_at
		FROM contingency_periods
		WHERE id = $1 AND company_id = $2
	`

	var id, cID, establishmentID, pointOfSaleID, ambiente string
	var fInicio, hInicio string
	var fFin, hFin, motivoContingencia *string
	var tipoContingencia int
	var status string
	var processing bool
	var createdAt, updatedAt interface{}

	err := db.QueryRowContext(c.Request.Context(), query, periodID, companyID).Scan(
		&id, &cID, &establishmentID, &pointOfSaleID, &ambiente,
		&fInicio, &hInicio, &fFin, &hFin,
		&tipoContingencia, &motivoContingencia, &status, &processing,
		&createdAt, &updatedAt,
	)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "period not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get period"})
		return
	}

	// Get invoice count
	var invoiceCount int
	countQuery := `SELECT COUNT(*) FROM invoices WHERE contingency_period_id = $1`
	db.QueryRowContext(c.Request.Context(), countQuery, periodID).Scan(&invoiceCount)

	// Get invoice status breakdown
	statusQuery := `
		SELECT dte_transmission_status, COUNT(*) 
		FROM invoices 
		WHERE contingency_period_id = $1 
		GROUP BY dte_transmission_status
	`
	statusRows, _ := db.QueryContext(c.Request.Context(), statusQuery, periodID)
	defer statusRows.Close()

	statusBreakdown := make(map[string]int)
	for statusRows.Next() {
		var s string
		var count int
		if err := statusRows.Scan(&s, &count); err == nil {
			statusBreakdown[s] = count
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"id":                  id,
		"company_id":          cID,
		"establishment_id":    establishmentID,
		"point_of_sale_id":    pointOfSaleID,
		"ambiente":            ambiente,
		"f_inicio":            fInicio,
		"h_inicio":            hInicio,
		"f_fin":               fFin,
		"h_fin":               hFin,
		"tipo_contingencia":   tipoContingencia,
		"motivo_contingencia": motivoContingencia,
		"status":              status,
		"processing":          processing,
		"created_at":          createdAt,
		"updated_at":          updatedAt,
		"invoice_count":       invoiceCount,
		"status_breakdown":    statusBreakdown,
	})
}

// GetPeriodInvoices returns all invoices in a contingency period
// GET /v1/contingency/periods/:id/invoices
func (h *ContingencyHandler) GetPeriodInvoices(c *gin.Context) {
	companyID := c.GetString("company_id")
	if companyID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "company_id header required"})
		return
	}

	periodID := c.Param("id")
	if periodID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "period id required"})
		return
	}

	db := c.MustGet("db").(*sql.DB)

	// Verify period belongs to company
	var exists bool
	checkQuery := `SELECT EXISTS(SELECT 1 FROM contingency_periods WHERE id = $1 AND company_id = $2)`
	db.QueryRowContext(c.Request.Context(), checkQuery, periodID, companyID).Scan(&exists)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "period not found"})
		return
	}

	// Get invoices
	query := `
		SELECT id, invoice_number, dte_type, dte_codigo_generacion,
			   dte_transmission_status, dte_sello_recibido,
			   contingency_event_id, lote_id,
			   signature_retry_count, finalized_at
		FROM invoices
		WHERE contingency_period_id = $1
		ORDER BY finalized_at ASC
	`

	rows, err := db.QueryContext(c.Request.Context(), query, periodID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query invoices"})
		return
	}
	defer rows.Close()

	var invoices []gin.H
	for rows.Next() {
		var id, invoiceNumber string
		var dteType, dteCodigoGeneracion, dteSelloRecibido *string
		var dteTransmissionStatus string
		var contingencyEventID, loteID *string
		var signatureRetryCount int
		var finalizedAt interface{}

		err := rows.Scan(
			&id, &invoiceNumber, &dteType, &dteCodigoGeneracion,
			&dteTransmissionStatus, &dteSelloRecibido,
			&contingencyEventID, &loteID,
			&signatureRetryCount, &finalizedAt,
		)
		if err != nil {
			continue
		}

		invoices = append(invoices, gin.H{
			"id":                      id,
			"invoice_number":          invoiceNumber,
			"dte_type":                dteType,
			"dte_codigo_generacion":   dteCodigoGeneracion,
			"dte_transmission_status": dteTransmissionStatus,
			"dte_sello_recibido":      dteSelloRecibido,
			"contingency_event_id":    contingencyEventID,
			"lote_id":                 loteID,
			"signature_retry_count":   signatureRetryCount,
			"finalized_at":            finalizedAt,
		})
	}

	if invoices == nil {
		invoices = []gin.H{}
	}

	c.JSON(http.StatusOK, gin.H{
		"invoices": invoices,
		"count":    len(invoices),
	})
}

// ClosePeriod manually closes an active contingency period
// POST /v1/contingency/periods/:id/close
func (h *ContingencyHandler) ClosePeriod(c *gin.Context) {
	companyID := c.GetString("company_id")
	if companyID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "company_id header required"})
		return
	}

	periodID := c.Param("id")
	if periodID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "period id required"})
		return
	}

	// Verify period belongs to company and is active
	period, err := h.contingencyService.GetPeriodByID(c.Request.Context(), periodID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "period not found"})
		return
	}

	if period.CompanyID != companyID {
		c.JSON(http.StatusForbidden, gin.H{"error": "period does not belong to this company"})
		return
	}

	if period.Status != "active" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "period is not active", "current_status": period.Status})
		return
	}

	// Close the period
	err = h.contingencyService.ClosePeriod(c.Request.Context(), periodID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to close period"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   "period closed successfully",
		"period_id": periodID,
		"status":    "reporting",
	})
}

// ListLotes returns all lotes for the company
// GET /v1/contingency/lotes
func (h *ContingencyHandler) ListLotes(c *gin.Context) {
	companyID := c.GetString("company_id")
	if companyID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "company_id header required"})
		return
	}

	db := c.MustGet("db").(*sql.DB)

	query := `
		SELECT l.id, l.contingency_event_id, l.codigo_lote,
			   l.company_id, l.establishment_id, l.point_of_sale_id,
			   l.dte_count, l.status, l.processing,
			   l.submitted_at, l.last_polled_at, l.completed_at,
			   l.created_at, l.updated_at
		FROM lotes l
		WHERE l.company_id = $1
		ORDER BY l.created_at DESC
		LIMIT 100
	`

	rows, err := db.QueryContext(c.Request.Context(), query, companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query lotes"})
		return
	}
	defer rows.Close()

	var lotes []gin.H
	for rows.Next() {
		var id, contingencyEventID, companyID, establishmentID, pointOfSaleID string
		var codigoLote *string
		var dteCount int
		var status string
		var processing bool
		var submittedAt, lastPolledAt, completedAt, createdAt, updatedAt interface{}

		err := rows.Scan(
			&id, &contingencyEventID, &codigoLote,
			&companyID, &establishmentID, &pointOfSaleID,
			&dteCount, &status, &processing,
			&submittedAt, &lastPolledAt, &completedAt,
			&createdAt, &updatedAt,
		)
		if err != nil {
			continue
		}

		lotes = append(lotes, gin.H{
			"id":                   id,
			"contingency_event_id": contingencyEventID,
			"codigo_lote":          codigoLote,
			"company_id":           companyID,
			"establishment_id":     establishmentID,
			"point_of_sale_id":     pointOfSaleID,
			"dte_count":            dteCount,
			"status":               status,
			"processing":           processing,
			"submitted_at":         submittedAt,
			"last_polled_at":       lastPolledAt,
			"completed_at":         completedAt,
			"created_at":           createdAt,
			"updated_at":           updatedAt,
		})
	}

	if lotes == nil {
		lotes = []gin.H{}
	}

	c.JSON(http.StatusOK, gin.H{
		"lotes": lotes,
		"count": len(lotes),
	})
}

// GetLote returns a single lote with details
// GET /v1/contingency/lotes/:id
func (h *ContingencyHandler) GetLote(c *gin.Context) {
	companyID := c.GetString("company_id")
	if companyID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "company_id header required"})
		return
	}

	loteID := c.Param("id")
	if loteID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "lote id required"})
		return
	}

	db := c.MustGet("db").(*sql.DB)

	query := `
		SELECT l.id, l.contingency_event_id, l.codigo_lote,
			   l.company_id, l.establishment_id, l.point_of_sale_id,
			   l.dte_count, l.status, l.processing, l.hacienda_response,
			   l.submitted_at, l.last_polled_at, l.completed_at,
			   l.created_at, l.updated_at
		FROM lotes l
		WHERE l.id = $1 AND l.company_id = $2
	`

	var id, contingencyEventID, cID, establishmentID, pointOfSaleID string
	var codigoLote *string
	var dteCount int
	var status string
	var processing bool
	var haciendaResponse []byte
	var submittedAt, lastPolledAt, completedAt, createdAt, updatedAt interface{}

	err := db.QueryRowContext(c.Request.Context(), query, loteID, companyID).Scan(
		&id, &contingencyEventID, &codigoLote,
		&cID, &establishmentID, &pointOfSaleID,
		&dteCount, &status, &processing, &haciendaResponse,
		&submittedAt, &lastPolledAt, &completedAt,
		&createdAt, &updatedAt,
	)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "lote not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get lote"})
		return
	}

	// Get invoice status breakdown for this lote
	statusQuery := `
		SELECT dte_transmission_status, COUNT(*) 
		FROM invoices 
		WHERE lote_id = $1 
		GROUP BY dte_transmission_status
	`
	statusRows, _ := db.QueryContext(c.Request.Context(), statusQuery, loteID)
	defer statusRows.Close()

	statusBreakdown := make(map[string]int)
	for statusRows.Next() {
		var s string
		var count int
		if err := statusRows.Scan(&s, &count); err == nil {
			statusBreakdown[s] = count
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"id":                   id,
		"contingency_event_id": contingencyEventID,
		"codigo_lote":          codigoLote,
		"company_id":           cID,
		"establishment_id":     establishmentID,
		"point_of_sale_id":     pointOfSaleID,
		"dte_count":            dteCount,
		"status":               status,
		"processing":           processing,
		"hacienda_response":    string(haciendaResponse),
		"submitted_at":         submittedAt,
		"last_polled_at":       lastPolledAt,
		"completed_at":         completedAt,
		"created_at":           createdAt,
		"updated_at":           updatedAt,
		"status_breakdown":     statusBreakdown,
	})
}

// ListEvents returns all contingency events for the company
// GET /v1/contingency/events
func (h *ContingencyHandler) ListEvents(c *gin.Context) {
	companyID := c.GetString("company_id")
	if companyID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "company_id header required"})
		return
	}

	db := c.MustGet("db").(*sql.DB)

	query := `
		SELECT id, contingency_period_id, codigo_generacion,
			   company_id, establishment_id, point_of_sale_id, ambiente,
			   estado, sello_recibido,
			   submitted_at, accepted_at, created_at
		FROM contingency_events
		WHERE company_id = $1
		ORDER BY created_at DESC
		LIMIT 100
	`

	rows, err := db.QueryContext(c.Request.Context(), query, companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query events"})
		return
	}
	defer rows.Close()

	var events []gin.H
	for rows.Next() {
		var id, contingencyPeriodID, codigoGeneracion string
		var companyID, establishmentID, pointOfSaleID, ambiente string
		var estado, selloRecibido *string
		var submittedAt, acceptedAt, createdAt interface{}

		err := rows.Scan(
			&id, &contingencyPeriodID, &codigoGeneracion,
			&companyID, &establishmentID, &pointOfSaleID, &ambiente,
			&estado, &selloRecibido,
			&submittedAt, &acceptedAt, &createdAt,
		)
		if err != nil {
			continue
		}

		events = append(events, gin.H{
			"id":                    id,
			"contingency_period_id": contingencyPeriodID,
			"codigo_generacion":     codigoGeneracion,
			"company_id":            companyID,
			"establishment_id":      establishmentID,
			"point_of_sale_id":      pointOfSaleID,
			"ambiente":              ambiente,
			"estado":                estado,
			"sello_recibido":        selloRecibido,
			"submitted_at":          submittedAt,
			"accepted_at":           acceptedAt,
			"created_at":            createdAt,
		})
	}

	if events == nil {
		events = []gin.H{}
	}

	c.JSON(http.StatusOK, gin.H{
		"events": events,
		"count":  len(events),
	})
}
