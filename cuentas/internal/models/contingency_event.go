package models

import (
	"time"
)

type ContingencyEvent struct {
	ID                  string     `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	ContingencyPeriodID string     `json:"contingency_period_id" gorm:"type:uuid;not null"`
	CodigoGeneracion    string     `json:"codigo_generacion" gorm:"type:varchar(36);not null"`
	CompanyID           string     `json:"company_id" gorm:"type:uuid;not null"`
	EstablishmentID     string     `json:"establishment_id" gorm:"type:uuid;not null"`
	PointOfSaleID       string     `json:"point_of_sale_id" gorm:"type:uuid;not null"`
	Ambiente            string     `json:"ambiente" gorm:"type:varchar(2);not null"`
	EventJSON           []byte     `json:"event_json" gorm:"type:jsonb;not null"`
	EventSigned         string     `json:"event_signed" gorm:"type:text;not null"`
	Estado              *string    `json:"estado,omitempty" gorm:"type:varchar(20)"`
	SelloRecibido       *string    `json:"sello_recibido,omitempty" gorm:"type:text"`
	HaciendaResponse    []byte     `json:"hacienda_response,omitempty" gorm:"type:jsonb"`
	SubmittedAt         *time.Time `json:"submitted_at,omitempty"`
	AcceptedAt          *time.Time `json:"accepted_at,omitempty"`
	CreatedAt           time.Time  `json:"created_at" gorm:"not null;default:now()"`
}

func (ContingencyEvent) TableName() string {
	return "contingency_events"
}

func (ce *ContingencyEvent) IsAccepted() bool {
	return ce.Estado != nil && *ce.Estado == "RECIBIDO"
}
