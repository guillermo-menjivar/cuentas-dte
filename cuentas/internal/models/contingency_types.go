package models

// Tipo Contingencia (Hacienda catalog)
const (
	TipoContingenciaMHDown       = 1 // No disponibilidad de sistema del MH
	TipoContingenciaEmisorSystem = 2 // No disponibilidad de sistema del emisor
	TipoContingenciaInternet     = 3 // Falla de internet
	TipoContingenciaPower        = 4 // Falla de energía
	TipoContingenciaOther        = 5 // Otro (requires motivo)
)

// DTE transmission status
const (
	DTEStatusPending          = "pending"
	DTEStatusPendingSignature = "pending_signature"
	DTEStatusContingencyQueue = "contingency_queued"
	DTEStatusFailedRetry      = "failed_retry"
	DTEStatusProcesado        = "procesado"
	DTEStatusRechazado        = "rechazado"
)

// Period status
const (
	PeriodStatusActive    = "active"
	PeriodStatusReporting = "reporting"
	PeriodStatusCompleted = "completed"
)

// Lote status
const (
	LoteStatusPending   = "pending"
	LoteStatusSubmitted = "submitted"
	LoteStatusCompleted = "completed"
)

func IsValidTipoContingencia(tipo int) bool {
	return tipo >= 1 && tipo <= 5
}
