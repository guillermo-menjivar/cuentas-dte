package models

import "errors"

var (
	// Establishment errors
	ErrInvalidTipoEstablecimiento  = errors.New("invalid tipo establecimiento")
	ErrInvalidDepartamento         = errors.New("invalid departamento")
	ErrInvalidMunicipio            = errors.New("invalid municipio")
	ErrInvalidCodEstablecimientoMH = errors.New("cod establecimiento MH must be 4 characters")
	ErrInvalidCodEstablecimiento   = errors.New("cod establecimiento must be 1-10 characters")
	ErrEstablishmentNotFound       = errors.New("establishment not found")

	// Point of Sale errors
	ErrInvalidCodPuntoVentaMH = errors.New("cod punto venta MH must be 4 characters")
	ErrInvalidCodPuntoVenta   = errors.New("cod punto venta must be 1-15 characters")
	ErrInvalidLatitude        = errors.New("latitude must be between -90 and 90")
	ErrInvalidLongitude       = errors.New("longitude must be between -180 and 180")
	ErrPointOfSaleNotFound    = errors.New("point of sale not found")
	ErrInvalidPaymentMethod   = errors.New("invalid payment method code")
)
