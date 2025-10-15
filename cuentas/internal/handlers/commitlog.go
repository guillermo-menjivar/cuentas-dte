package handlers

import (
	"database/sql"
	"net/http"
	"time"

	"cuentas/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
)

// DTECommitLogEntry represents a commit log entry for API responses
type DTECommitLogEntry struct {
	CodigoGeneracion        string    `json:"codigo_generacion"`
	InvoiceID               string    `json:"invoice_id"`
	InvoiceNumber           string    `json:"invoice_number"`
	CompanyID               string    `json:"company_id"`
	ClientID                string    `json:"client_id"`
	EstablishmentID         string    `json:"establishment_id"`
	PointOfSaleID           string    `json:"point_of_sale_id"`
	Subtotal                float64   `json:"subtotal"`
	TotalDiscount           float64   `json:"total_discount"`
	TotalTaxes              float64   `json:"total_taxes"`
	IVAAmount               float64   `json:"iva_amount"`
	TotalAmount             float64   `json:"total_amount"`
	Currency                string    `json:"currency"`
	PaymentMethod           string    `json:"payment_method"`
	PaymentTerms            string    `json:"payment_terms"`
	ReferencesInvoiceID     *string   `json:"references_invoice_id,omitempty"`
	NumeroControl           string    `json:"numero_control"`
	TipoDte                 string    `json:"tipo_dte"`
	Ambiente                string    `json:"ambiente"`
	FechaEmision            string    `json:"fecha_emision"`
	FiscalYear              int       `json:"fiscal_year"`
	FiscalMonth             int       `json:"fiscal_month"`
	DteURL                  string    `json:"dte_url"`
	HaciendaEstado          *string   `json:"hacienda_estado,omitempty"`
	HaciendaSelloRecibido   *string   `json:"hacienda_sello_recibido,omitempty"`
	HaciendaFhProcesamiento *string   `json:"hacienda_fh_procesamiento,omitempty"`
	HaciendaCodigoMsg       *string   `json:"hacienda_codigo_msg,omitempty"`
	HaciendaDescripcionMsg  *string   `json:"hacienda_descripcion_msg,omitempty"`
	HaciendaObservaciones   []string  `json:"hacienda_observaciones,omitempty"`
	CreatedBy               *string   `json:"created_by,omitempty"`
	SubmittedAt             time.Time `json:"submitted_at"`
	CreatedAt               time.Time `json:"created_at"`
}

// ListDTECommitLogHandler handles GET /v1/dte/commit-log
func ListDTECommitLogHandler(c *gin.Context) {
	companyID := c.MustGet("company_id").(string)
	db := c.MustGet("db").(*sql.DB)

	// Build query with filters
	query := `
		SELECT 
			codigo_generacion, invoice_id, invoice_number, company_id, client_id,
			establishment_id, point_of_sale_id,
			subtotal, total_discount, total_taxes, iva_amount, total_amount, currency,
			payment_method, payment_terms, references_invoice_id,
			numero_control, tipo_dte, ambiente, fecha_emision,
			fiscal_year, fiscal_month, dte_url,
			hacienda_estado, hacienda_sello_recibido, hacienda_fh_procesamiento,
			hacienda_codigo_msg, hacienda_descripcion_msg, hacienda_observaciones,
			created_by, submitted_at, created_at
		FROM dte_commit_log
		WHERE company_id = $1
	`

	args := []interface{}{companyID}
	argCount := 1

	// Filter by client_id
	if clientID := c.Query("client_id"); clientID != "" {
		argCount++
		query += ` AND client_id = $` + string(rune(argCount+'0'))
		args = append(args, clientID)
	}

	// Filter by codigo_generacion
	if codigoGen := c.Query("codigo_generacion"); codigoGen != "" {
		argCount++
		query += ` AND codigo_generacion = $` + string(rune(argCount+'0'))
		args = append(args, codigoGen)
	}

	// Filter by hacienda_estado
	if estado := c.Query("hacienda_estado"); estado != "" {
		argCount++
		query += ` AND hacienda_estado = $` + string(rune(argCount+'0'))
		args = append(args, estado)
	}

	// Filter by establishment_id
	if establishmentID := c.Query("establishment_id"); establishmentID != "" {
		argCount++
		query += ` AND establishment_id = $` + string(rune(argCount+'0'))
		args = append(args, establishmentID)
	}

	// Filter by point_of_sale_id
	if posID := c.Query("point_of_sale_id"); posID != "" {
		argCount++
		query += ` AND point_of_sale_id = $` + string(rune(argCount+'0'))
		args = append(args, posID)
	}

	// Filter by fiscal period
	if fiscalYear := c.Query("fiscal_year"); fiscalYear != "" {
		argCount++
		query += ` AND fiscal_year = $` + string(rune(argCount+'0'))
		args = append(args, fiscalYear)
	}

	if fiscalMonth := c.Query("fiscal_month"); fiscalMonth != "" {
		argCount++
		query += ` AND fiscal_month = $` + string(rune(argCount+'0'))
		args = append(args, fiscalMonth)
	}

	// Filter by date range
	if startDate := c.Query("start_date"); startDate != "" {
		argCount++
		query += ` AND fecha_emision >= $` + string(rune(argCount+'0'))
		args = append(args, startDate)
	}

	if endDate := c.Query("end_date"); endDate != "" {
		argCount++
		query += ` AND fecha_emision <= $` + string(rune(argCount+'0'))
		args = append(args, endDate)
	}

	// Order by most recent first
	query += ` ORDER BY submitted_at DESC`

	// Optional limit
	limit := c.DefaultQuery("limit", "100")
	argCount++
	query += ` LIMIT $` + string(rune(argCount+'0'))
	args = append(args, limit)

	// Execute query
	rows, err := db.QueryContext(c.Request.Context(), query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "failed to query commit log",
			Code:  "internal_error",
		})
		return
	}
	defer rows.Close()

	// Parse results
	entries := []DTECommitLogEntry{}
	for rows.Next() {
		var entry DTECommitLogEntry
		var haciendaFhProc *time.Time
		var observaciones []string

		err := rows.Scan(
			&entry.CodigoGeneracion, &entry.InvoiceID, &entry.InvoiceNumber,
			&entry.CompanyID, &entry.ClientID, &entry.EstablishmentID, &entry.PointOfSaleID,
			&entry.Subtotal, &entry.TotalDiscount, &entry.TotalTaxes, &entry.IVAAmount,
			&entry.TotalAmount, &entry.Currency,
			&entry.PaymentMethod, &entry.PaymentTerms, &entry.ReferencesInvoiceID,
			&entry.NumeroControl, &entry.TipoDte, &entry.Ambiente, &entry.FechaEmision,
			&entry.FiscalYear, &entry.FiscalMonth, &entry.DteURL,
			&entry.HaciendaEstado, &entry.HaciendaSelloRecibido, &haciendaFhProc,
			&entry.HaciendaCodigoMsg, &entry.HaciendaDescripcionMsg,
			pq.Array(&observaciones),
			&entry.CreatedBy, &entry.SubmittedAt, &entry.CreatedAt,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, models.ErrorResponse{
				Error: "failed to scan commit log entry",
				Code:  "internal_error",
			})
			return
		}

		// Convert timestamp to string if present
		if haciendaFhProc != nil {
			fh := haciendaFhProc.Format("02/01/2006 15:04:05")
			entry.HaciendaFhProcesamiento = &fh
		}

		entry.HaciendaObservaciones = observaciones

		entries = append(entries, entry)
	}

	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "error iterating commit log",
			Code:  "internal_error",
		})
		return
	}

	// Always return array, even if empty
	if entries == nil {
		entries = []DTECommitLogEntry{}
	}

	c.JSON(http.StatusOK, gin.H{
		"entries": entries,
		"count":   len(entries),
	})
}

// GetDTECommitLogEntryHandler handles GET /v1/dte/commit-log/:codigo_generacion
func GetDTECommitLogEntryHandler(c *gin.Context) {
	codigoGeneracion := c.Param("codigo_generacion")
	companyID := c.MustGet("company_id").(string)
	db := c.MustGet("db").(*sql.DB)

	query := `
		SELECT 
			codigo_generacion, invoice_id, invoice_number, company_id, client_id,
			establishment_id, point_of_sale_id,
			subtotal, total_discount, total_taxes, iva_amount, total_amount, currency,
			payment_method, payment_terms, references_invoice_id,
			numero_control, tipo_dte, ambiente, fecha_emision,
			fiscal_year, fiscal_month, dte_url,
			dte_unsigned, dte_signed,
			hacienda_estado, hacienda_sello_recibido, hacienda_fh_procesamiento,
			hacienda_codigo_msg, hacienda_descripcion_msg, hacienda_observaciones,
			hacienda_response_full,
			created_by, submitted_at, created_at
		FROM dte_commit_log
		WHERE codigo_generacion = $1 AND company_id = $2
	`

	var entry struct {
		DTECommitLogEntry
		DteUnsigned          string `json:"dte_unsigned"`
		DteSigned            string `json:"dte_signed"`
		HaciendaResponseFull string `json:"hacienda_response_full"`
	}

	var haciendaFhProc *time.Time
	var observaciones []string

	err := db.QueryRowContext(c.Request.Context(), query, codigoGeneracion, companyID).Scan(
		&entry.CodigoGeneracion, &entry.InvoiceID, &entry.InvoiceNumber,
		&entry.CompanyID, &entry.ClientID, &entry.EstablishmentID, &entry.PointOfSaleID,
		&entry.Subtotal, &entry.TotalDiscount, &entry.TotalTaxes, &entry.IVAAmount,
		&entry.TotalAmount, &entry.Currency,
		&entry.PaymentMethod, &entry.PaymentTerms, &entry.ReferencesInvoiceID,
		&entry.NumeroControl, &entry.TipoDte, &entry.Ambiente, &entry.FechaEmision,
		&entry.FiscalYear, &entry.FiscalMonth, &entry.DteURL,
		&entry.DteUnsigned, &entry.DteSigned,
		&entry.HaciendaEstado, &entry.HaciendaSelloRecibido, &haciendaFhProc,
		&entry.HaciendaCodigoMsg, &entry.HaciendaDescripcionMsg,
		pq.Array(&observaciones),
		&entry.HaciendaResponseFull,
		&entry.CreatedBy, &entry.SubmittedAt, &entry.CreatedAt,
	)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Error: "commit log entry not found",
			Code:  "not_found",
		})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "failed to get commit log entry",
			Code:  "internal_error",
		})
		return
	}

	// Convert timestamp to string if present
	if haciendaFhProc != nil {
		fh := haciendaFhProc.Format("02/01/2006 15:04:05")
		entry.HaciendaFhProcesamiento = &fh
	}

	entry.HaciendaObservaciones = observaciones

	c.JSON(http.StatusOK, entry)
}
