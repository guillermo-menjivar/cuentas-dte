package dte

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"cuentas/internal/models"

	"github.com/google/uuid"
)

// ContingencyEventJSON represents the complete structure of Evento de Contingencia
type ContingencyEventJSON struct {
	Identificacion IdentificacionContingencia `json:"identificacion"`
	Emisor         EmisorContingencia         `json:"emisor"`
	DetalleDTE     []DetalleDTEContingencia   `json:"detalleDTE"`
	Motivo         MotivoContingencia         `json:"motivo"`
}

type IdentificacionContingencia struct {
	Version          int    `json:"version"`
	Ambiente         string `json:"ambiente"`
	CodigoGeneracion string `json:"codigoGeneracion"`
	FTransmision     string `json:"fTransmision"`
	HTransmision     string `json:"hTransmision"`
}

type EmisorContingencia struct {
	NIT                  string  `json:"nit"`
	Nombre               string  `json:"nombre"`
	NombreResponsable    string  `json:"nombreResponsable"`
	TipoDocResponsable   string  `json:"tipoDocResponsable"`
	NumeroDocResponsable string  `json:"numeroDocResponsable"`
	TipoEstablecimiento  string  `json:"tipoEstablecimiento"`
	CodEstableMH         *string `json:"codEstableMH,omitempty"`
	CodPuntoVenta        *string `json:"codPuntoVenta,omitempty"`
	Telefono             string  `json:"telefono"`
	Correo               string  `json:"correo"`
}

type DetalleDTEContingencia struct {
	NoItem           int    `json:"noItem"`
	CodigoGeneracion string `json:"codigoGeneracion"`
	TipoDoc          string `json:"tipoDoc"`
}

type MotivoContingencia struct {
	FInicio            string  `json:"fInicio"`
	FFin               string  `json:"fFin"`
	HInicio            string  `json:"hInicio"`
	HFin               string  `json:"hFin"`
	TipoContingencia   int     `json:"tipoContingencia"`
	MotivoContingencia *string `json:"motivoContingencia,omitempty"`
}

// BuildContingencyEvent creates the JSON for Evento de Contingencia (schema v3)
func (s *ContingencyService) BuildContingencyEvent(
	ctx context.Context,
	companyID string,
	ambiente string,
	dtes []*models.ContingencyQueueItem,
	tipoContingencia int,
	motivoContingencia string,
) ([]byte, string, error) {

	if len(dtes) == 0 {
		return nil, "", fmt.Errorf("no DTEs provided for contingency event")
	}

	if len(dtes) > 1000 {
		return nil, "", fmt.Errorf("too many DTEs: max 1000, got %d", len(dtes))
	}

	// Get company info
	company, err := s.getCompany(ctx, companyID)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get company: %w", err)
	}

	// Generate event codigo
	codigoGeneracion := strings.ToUpper(uuid.New().String())

	// Get El Salvador timezone
	loc, _ := time.LoadLocation("America/El_Salvador")
	if loc == nil {
		loc = time.FixedZone("CST", -6*60*60)
	}

	now := time.Now().In(loc)

	// Find date/time range from DTEs
	var fechaInicio, fechaFin time.Time

	for i, dte := range dtes {
		dteTime := dte.FailureTimestamp.In(loc)

		if i == 0 {
			fechaInicio = dteTime
			fechaFin = dteTime
		} else {
			if dteTime.Before(fechaInicio) {
				fechaInicio = dteTime
			}
			if dteTime.After(fechaFin) {
				fechaFin = dteTime
			}
		}
	}

	// Build detalleDTE array
	detalleDTE := make([]DetalleDTEContingencia, len(dtes))
	for i, dte := range dtes {
		detalleDTE[i] = DetalleDTEContingencia{
			NoItem:           i + 1,
			CodigoGeneracion: dte.CodigoGeneracion,
			TipoDoc:          dte.TipoDte,
		}
	}

	// Build emisor
	emisor := EmisorContingencia{
		NIT:                  company.NIT,
		Nombre:               company.LegalName,
		NombreResponsable:    company.LegalRepresentativeName,
		TipoDocResponsable:   company.LegalRepresentativeDocType,
		NumeroDocResponsable: company.LegalRepresentativeDocNumber,
		TipoEstablecimiento:  company.EstablishmentType,
		Telefono:             company.Phone,
		Correo:               company.Email,
	}

	// Add optional fields
	if company.EstablishmentCodeMH != "" {
		emisor.CodEstableMH = &company.EstablishmentCodeMH
	}

	if company.PointOfSaleCode != "" {
		emisor.CodPuntoVenta = &company.PointOfSaleCode
	}

	// Build motivo
	var motivoPtr *string
	if motivoContingencia != "" {
		motivoPtr = &motivoContingencia
	}

	motivo := MotivoContingencia{
		FInicio:            fechaInicio.Format("2006-01-02"),
		FFin:               fechaFin.Format("2006-01-02"),
		HInicio:            fechaInicio.Format("15:04:05"),
		HFin:               fechaFin.Format("15:04:05"),
		TipoContingencia:   tipoContingencia,
		MotivoContingencia: motivoPtr,
	}

	// Build complete event
	event := ContingencyEventJSON{
		Identificacion: IdentificacionContingencia{
			Version:          3,
			Ambiente:         ambiente,
			CodigoGeneracion: codigoGeneracion,
			FTransmision:     now.Format("2006-01-02"),
			HTransmision:     now.Format("15:04:05"),
		},
		Emisor:     emisor,
		DetalleDTE: detalleDTE,
		Motivo:     motivo,
	}

	eventJSON, err := json.Marshal(event)
	if err != nil {
		return nil, "", fmt.Errorf("failed to marshal event: %w", err)
	}

	return eventJSON, codigoGeneracion, nil
}

type CompanyInfo struct {
	NIT                          string
	LegalName                    string
	LegalRepresentativeName      string
	LegalRepresentativeDocType   string
	LegalRepresentativeDocNumber string
	EstablishmentType            string
	EstablishmentCodeMH          string
	PointOfSaleCode              string
	Phone                        string
	Email                        string
}

func (s *ContingencyService) getCompany(ctx context.Context, companyID string) (*CompanyInfo, error) {
	query := `
        SELECT 
            nit,
            legal_name,
            legal_representative_name,
            legal_representative_doc_type,
            legal_representative_doc_number,
            establishment_type,
            COALESCE(establishment_code_mh, ''),
            COALESCE(point_of_sale_code, ''),
            phone,
            email
        FROM companies
        WHERE id = $1
    `

	var company CompanyInfo
	err := s.db.QueryRowContext(ctx, query, companyID).Scan(
		&company.NIT,
		&company.LegalName,
		&company.LegalRepresentativeName,
		&company.LegalRepresentativeDocType,
		&company.LegalRepresentativeDocNumber,
		&company.EstablishmentType,
		&company.EstablishmentCodeMH,
		&company.PointOfSaleCode,
		&company.Phone,
		&company.Email,
	)

	if err != nil {
		return nil, err
	}

	return &company, nil
}
