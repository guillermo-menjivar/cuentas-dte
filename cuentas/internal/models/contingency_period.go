package models

import (
	"time"
)

type ContingencyPeriod struct {
	ID                 string    `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	CompanyID          string    `json:"company_id" gorm:"type:uuid;not null"`
	EstablishmentID    string    `json:"establishment_id" gorm:"type:uuid;not null"`
	PointOfSaleID      string    `json:"point_of_sale_id" gorm:"type:uuid;not null"`
	Ambiente           string    `json:"ambiente" gorm:"type:varchar(2);not null"`
	FInicio            string    `json:"f_inicio" gorm:"type:date;not null"`
	HInicio            string    `json:"h_inicio" gorm:"type:time;not null"`
	FFin               *string   `json:"f_fin,omitempty" gorm:"type:date"`
	HFin               *string   `json:"h_fin,omitempty" gorm:"type:time"`
	TipoContingencia   int       `json:"tipo_contingencia" gorm:"not null"`
	MotivoContingencia *string   `json:"motivo_contingencia,omitempty" gorm:"type:text"`
	Status             string    `json:"status" gorm:"type:varchar(20);not null;default:'active'"`
	Processing         bool      `json:"processing" gorm:"default:false"`
	CreatedAt          time.Time `json:"created_at" gorm:"not null;default:now()"`
	UpdatedAt          time.Time `json:"updated_at" gorm:"not null;default:now()"`
}

func (ContingencyPeriod) TableName() string {
	return "contingency_periods"
}

func (cp *ContingencyPeriod) IsActive() bool {
	return cp.Status == "active"
}

func (cp *ContingencyPeriod) IsReporting() bool {
	return cp.Status == "reporting"
}

func (cp *ContingencyPeriod) IsCompleted() bool {
	return cp.Status == "completed"
}

func (cp *ContingencyPeriod) IsClosed() bool {
	return cp.FFin != nil && cp.HFin != nil
}
