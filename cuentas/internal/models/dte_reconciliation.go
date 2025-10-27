package models

import "time"

// DTEReconciliationRecord represents a single DTE reconciliation result
type DTEReconciliationRecord struct {
	// Internal record (from our database)
	CodigoGeneracion        string     `json:"codigo_generacion"`
	InvoiceID               string     `json:"invoice_id"`
	InvoiceNumber           string     `json:"invoice_number"`
	ClientID                string     `json:"client_id"`
	NumeroControl           string     `json:"numero_control"`
	TipoDTE                 string     `json:"tipo_dte"`
	FechaEmision            string     `json:"fecha_emision"`
	TotalAmount             float64    `json:"total_amount"`
	InternalEstado          *string    `json:"internal_estado"`
	InternalSello           *string    `json:"internal_sello"`
	InternalFhProcesamiento *time.Time `json:"internal_fh_procesamiento"`

	// Hacienda record (from API query)
	HaciendaEstado          string   `json:"hacienda_estado,omitempty"`
	HaciendaSello           string   `json:"hacienda_sello,omitempty"`
	HaciendaFhProcesamiento string   `json:"hacienda_fh_procesamiento,omitempty"`
	HaciendaCodigoMsg       string   `json:"hacienda_codigo_msg,omitempty"`
	HaciendaDescripcionMsg  string   `json:"hacienda_descripcion_msg,omitempty"`
	HaciendaObservaciones   []string `json:"hacienda_observaciones,omitempty"`

	// Reconciliation result
	Matches             bool     `json:"matches"`
	FechaEmisionMatches bool     `json:"fecha_emision_matches"`
	Discrepancies       []string `json:"discrepancies,omitempty"`
	HaciendaQueryStatus string   `json:"hacienda_query_status"` // "success", "not_found", "error"
	ErrorMessage        string   `json:"error_message,omitempty"`
	QueriedAt           string   `json:"queried_at"`
}

// DTEReconciliationSummary provides aggregate statistics
type DTEReconciliationSummary struct {
	TotalRecords       int `json:"total_records"`
	MatchedRecords     int `json:"matched_records"`
	MismatchedRecords  int `json:"mismatched_records"`
	DateMismatches     int `json:"date_mismatches"`
	NotFoundInHacienda int `json:"not_found_in_hacienda"`
	QueryErrors        int `json:"query_errors"`
}
