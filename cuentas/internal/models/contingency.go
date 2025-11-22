package models

import (
	"database/sql"
	"time"
)

type ContingencyQueueItem struct {
	ID                   string         `db:"id"`
	InvoiceID            sql.NullString `db:"invoice_id"`  // VARCHAR(36)
	PurchaseID           *string        `db:"purchase_id"` // UUID
	TipoDte              string         `db:"tipo_dte"`
	CodigoGeneracion     string         `db:"codigo_generacion"`
	Ambiente             string         `db:"ambiente"`
	FailureStage         string         `db:"failure_stage"`
	FailureReason        string         `db:"failure_reason"`
	FailureTimestamp     time.Time      `db:"failure_timestamp"`
	DTEUnsigned          []byte         `db:"dte_unsigned"`
	DTESigned            sql.NullString `db:"dte_signed"`
	ContingencyEventID   *string        `db:"contingency_event_id"`
	BatchID              *string        `db:"batch_id"`
	Status               string         `db:"status"`
	RetryCount           int            `db:"retry_count"`
	MaxRetries           int            `db:"max_retries"`
	SelloRecibido        sql.NullString `db:"sello_recibido"`
	HaciendaResponse     []byte         `db:"hacienda_response"`
	HaciendaObservations []string       `db:"hacienda_observations"`
	CreatedAt            time.Time      `db:"created_at"`
	UpdatedAt            time.Time      `db:"updated_at"`
	CompletedAt          *time.Time     `db:"completed_at"`
	CompanyID            string         `db:"company_id"`
	CreatedBy            *string        `db:"created_by"`
}

type ContingencyEvent struct {
	ID                 string         `db:"id"`
	CodigoGeneracion   string         `db:"codigo_generacion"`
	CompanyID          string         `db:"company_id"`
	Ambiente           string         `db:"ambiente"`
	FechaInicio        time.Time      `db:"fecha_inicio"`
	FechaFin           time.Time      `db:"fecha_fin"`
	HoraInicio         time.Time      `db:"hora_inicio"`
	HoraFin            time.Time      `db:"hora_fin"`
	TipoContingencia   int            `db:"tipo_contingencia"`
	MotivoContingencia sql.NullString `db:"motivo_contingencia"`
	EventUnsigned      []byte         `db:"event_unsigned"`
	EventSigned        sql.NullString `db:"event_signed"`
	Status             string         `db:"status"`
	SelloRecibido      sql.NullString `db:"sello_recibido"`
	HaciendaResponse   []byte         `db:"hacienda_response"`
	CreatedAt          time.Time      `db:"created_at"`
	SubmittedAt        *time.Time     `db:"submitted_at"`
	AcceptedAt         *time.Time     `db:"accepted_at"`
	CreatedBy          *string        `db:"created_by"`
	DTECount           int            `db:"dte_count"`
}

type ContingencyBatch struct {
	ID                 string         `db:"id"`
	ContingencyEventID string         `db:"contingency_event_id"`
	CodigoLote         sql.NullString `db:"codigo_lote"`
	CompanyID          string         `db:"company_id"`
	Ambiente           string         `db:"ambiente"`
	Status             string         `db:"status"`
	TotalDTEs          int            `db:"total_dtes"`
	ProcessedCount     int            `db:"processed_count"`
	RejectedCount      int            `db:"rejected_count"`
	HaciendaResponse   []byte         `db:"hacienda_response"`
	CreatedAt          time.Time      `db:"created_at"`
	SubmittedAt        *time.Time     `db:"submitted_at"`
	CompletedAt        *time.Time     `db:"completed_at"`
	RetryCount         int            `db:"retry_count"`
	MaxRetries         int            `db:"max_retries"`
	NextRetryAt        *time.Time     `db:"next_retry_at"`
}
