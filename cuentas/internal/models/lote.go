package models

import (
	"time"
)

type Lote struct {
	ID                 string     `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	ContingencyEventID string     `json:"contingency_event_id" gorm:"type:uuid;not null"`
	CodigoLote         *string    `json:"codigo_lote,omitempty" gorm:"type:varchar(100)"`
	CompanyID          string     `json:"company_id" gorm:"type:uuid;not null"`
	EstablishmentID    string     `json:"establishment_id" gorm:"type:uuid;not null"`
	PointOfSaleID      string     `json:"point_of_sale_id" gorm:"type:uuid;not null"`
	DTECount           int        `json:"dte_count" gorm:"not null"`
	Status             string     `json:"status" gorm:"type:varchar(20);not null;default:'pending'"`
	Processing         bool       `json:"processing" gorm:"default:false"`
	HaciendaResponse   []byte     `json:"hacienda_response,omitempty" gorm:"type:jsonb"`
	SubmittedAt        *time.Time `json:"submitted_at,omitempty"`
	LastPolledAt       *time.Time `json:"last_polled_at,omitempty"`
	CompletedAt        *time.Time `json:"completed_at,omitempty"`
	CreatedAt          time.Time  `json:"created_at" gorm:"not null;default:now()"`
	UpdatedAt          time.Time  `json:"updated_at" gorm:"not null;default:now()"`
}

func (Lote) TableName() string {
	return "lotes"
}

func (l *Lote) IsPending() bool {
	return l.Status == "pending"
}

func (l *Lote) IsSubmitted() bool {
	return l.Status == "submitted"
}

func (l *Lote) IsCompleted() bool {
	return l.Status == "completed"
}
