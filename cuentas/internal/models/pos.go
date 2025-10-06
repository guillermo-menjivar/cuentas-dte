package models

import "time"

type PointOfSale struct {
	ID              string
	EstablishmentID string
	Nombre          string
	CodPuntoVentaMH *string
	CodPuntoVenta   *string
	Latitude        *float64
	Longitude       *float64
	IsPortable      bool
	Active          bool
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type CreatePOSRequest struct {
	Nombre          string   `json:"nombre" binding:"required"`
	CodPuntoVentaMH *string  `json:"cod_punto_venta_mh,omitempty"`
	CodPuntoVenta   *string  `json:"cod_punto_venta,omitempty"`
	Latitude        *float64 `json:"latitude,omitempty"`
	Longitude       *float64 `json:"longitude,omitempty"`
	IsPortable      bool     `json:"is_portable"`
}

type UpdatePOSLocationRequest struct {
	Latitude  float64 `json:"latitude" binding:"required"`
	Longitude float64 `json:"longitude" binding:"required"`
}
