package models

import "time"

// Establishment represents a physical location for a company
type Establishment struct {
	ID                   string    `json:"id"`
	CompanyID            string    `json:"company_id"`
	TipoEstablecimiento  string    `json:"tipo_establecimiento"`
	Nombre               string    `json:"nombre"`
	CodEstablecimiento   *string   `json:"cod_establecimiento,omitempty"`
	Departamento         string    `json:"departamento"`
	Municipio            string    `json:"municipio"`
	ComplementoDireccion string    `json:"complemento_direccion"`
	Telefono             string    `json:"telefono"`
	Active               bool      `json:"active"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

// CreateEstablishmentRequest is the request to create a new establishment
type CreateEstablishmentRequest struct {
	TipoEstablecimiento  string  `json:"tipo_establecimiento" binding:"required"`
	Nombre               string  `json:"nombre" binding:"required,min=3,max=100"`
	CodEstablecimiento   *string `json:"cod_establecimiento,omitempty"`
	Departamento         string  `json:"departamento" binding:"required,len=2"`
	Municipio            string  `json:"municipio" binding:"required,len=2"`
	ComplementoDireccion string  `json:"complemento_direccion" binding:"required,min=1,max=200"`
	Telefono             string  `json:"telefono" binding:"required,min=8,max=30"`
}

// UpdateEstablishmentRequest is the request to update an establishment
type UpdateEstablishmentRequest struct {
	TipoEstablecimiento  *string `json:"tipo_establecimiento,omitempty"`
	Nombre               *string `json:"nombre,omitempty"`
	CodEstablecimiento   *string `json:"cod_establecimiento,omitempty"`
	Departamento         *string `json:"departamento,omitempty"`
	Municipio            *string `json:"municipio,omitempty"`
	ComplementoDireccion *string `json:"complemento_direccion,omitempty"`
	Telefono             *string `json:"telefono,omitempty"`
}

// Validate validates the CreateEstablishmentRequest
func (r *CreateEstablishmentRequest) Validate() error {
	// Validate tipo_establecimiento using codigos
	if !IsValidTipoEstablecimiento(r.TipoEstablecimiento) {
		return ErrInvalidTipoEstablecimiento
	}

	// Validate departamento using codigos
	if !isValidDepartamento(r.Departamento) {
		return ErrInvalidDepartamento
	}

	// Validate municipio for the given departamento using codigos
	if !isValidMunicipio(r.Departamento, r.Municipio) {
		return ErrInvalidMunicipio
	}

	// Validate MH codes if provided
	if r.CodEstablecimiento != nil && len(*r.CodEstablecimiento) != 4 {
		return ErrInvalidCodEstablecimientoMH
	}

	return nil
}

// PointOfSale represents a point of sale terminal
type PointOfSale struct {
	ID              string    `json:"id"`
	EstablishmentID string    `json:"establishment_id"`
	Nombre          string    `json:"nombre"`
	CodPuntoVenta   *string   `json:"cod_punto_venta,omitempty"`
	Latitude        *float64  `json:"latitude,omitempty"`
	Longitude       *float64  `json:"longitude,omitempty"`
	IsPortable      bool      `json:"is_portable"`
	Active          bool      `json:"active"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// CreatePOSRequest is the request to create a new point of sale
type CreatePOSRequest struct {
	Nombre        string   `json:"nombre" binding:"required,min=3,max=100"`
	CodPuntoVenta *string  `json:"cod_punto_venta,omitempty"`
	Latitude      *float64 `json:"latitude,omitempty"`
	Longitude     *float64 `json:"longitude,omitempty"`
	IsPortable    bool     `json:"is_portable"`
}

// UpdatePOSRequest is the request to update a point of sale
type UpdatePOSRequest struct {
	Nombre        *string  `json:"nombre,omitempty"`
	CodPuntoVenta *string  `json:"cod_punto_venta,omitempty"`
	Latitude      *float64 `json:"latitude,omitempty"`
	Longitude     *float64 `json:"longitude,omitempty"`
	IsPortable    *bool    `json:"is_portable,omitempty"`
}

// UpdatePOSLocationRequest is the request to update POS location
type UpdatePOSLocationRequest struct {
	Latitude  float64 `json:"latitude" binding:"required,min=-90,max=90"`
	Longitude float64 `json:"longitude" binding:"required,min=-180,max=180"`
}

// Validate validates the CreatePOSRequest
func (r *CreatePOSRequest) Validate() error {
	// Validate MH codes if provided
	if r.CodPuntoVenta != nil && len(*r.CodPuntoVenta) != 4 {
		return ErrInvalidCodPuntoVentaMH
	}

	// Validate coordinates if provided
	if r.Latitude != nil && (*r.Latitude < -90 || *r.Latitude > 90) {
		return ErrInvalidLatitude
	}

	if r.Longitude != nil && (*r.Longitude < -180 || *r.Longitude > 180) {
		return ErrInvalidLongitude
	}

	return nil
}

// Helper validation functions (will use codigos package)
func IsValidTipoEstablecimiento(tipo string) bool {
	validTypes := []string{"01", "02", "04", "07", "20"}
	for _, v := range validTypes {
		if tipo == v {
			return true
		}
	}
	return false
}

func isValidDepartamento(dep string) bool {
	// Use codigos package to validate
	// For now, basic check
	return len(dep) == 2
}

func isValidMunicipio(dep, mun string) bool {
	// Use codigos package to validate municipio for given departamento
	// For now, basic check
	return len(mun) == 2
}
