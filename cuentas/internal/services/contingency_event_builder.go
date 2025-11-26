package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"cuentas/internal/models"

	"github.com/google/uuid"
)

// ContingencyEventBuilder builds Evento de Contingencia JSON per Hacienda schema v3
type ContingencyEventBuilder struct {
	db *sql.DB
}

// NewContingencyEventBuilder creates a new event builder
func NewContingencyEventBuilder(db *sql.DB) *ContingencyEventBuilder {
	return &ContingencyEventBuilder{db: db}
}

// EventoContingencia represents the full event structure for Hacienda
type EventoContingencia struct {
	Identificacion IdentificacionEvento `json:"identificacion"`
	Emisor         EmisorEvento         `json:"emisor"`
	DetalleDTE     []DetalleDTEItem     `json:"detalleDTE"`
	Motivo         MotivoContingencia   `json:"motivo"`
}

// IdentificacionEvento - event identification
type IdentificacionEvento struct {
	Version          int    `json:"version"`
	Ambiente         string `json:"ambiente"`
	CodigoGeneracion string `json:"codigoGeneracion"`
	FTransmision     string `json:"fTransmision"`
	HTransmision     string `json:"hTransmision"`
}

// EmisorEvento - company info for event
type EmisorEvento struct {
	NIT                 string  `json:"nit"`
	Nombre              string  `json:"nombre"`
	NombreComercial     *string `json:"nombreComercial,omitempty"`
	TipoEstablecimiento string  `json:"tipoEstablecimiento"`
	CodEstablecimiento  *string `json:"codEstablecimiento,omitempty"`
	CodPuntoVenta       *string `json:"codPuntoVenta,omitempty"`
	Telefono            string  `json:"telefono"`
	Correo              string  `json:"correo"`
}

// DetalleDTEItem - individual DTE in the event
type DetalleDTEItem struct {
	NoItem           int    `json:"noItem"`
	CodigoGeneracion string `json:"codigoGeneracion"`
	TipoDoc          string `json:"tipoDoc"`
}

// MotivoContingencia - reason for contingency
type MotivoContingencia struct {
	FInicio            string  `json:"fInicio"`
	FPeriodo           string  `json:"fFin"`
	HInicio            string  `json:"hInicio"`
	HFin               string  `json:"hFin"`
	TipoContingencia   int     `json:"tipoContingencia"`
	MotivoContingencia *string `json:"motivoContingencia,omitempty"`
}

// CompanyInfo holds company data needed for event building
type CompanyInfo struct {
	NIT             string
	Nombre          string
	NombreComercial *string
	Telefono        string
	Correo          string
}

// EstablishmentInfo holds establishment data
type EstablishmentInfo struct {
	Codigo string
	Tipo   string
}

// PointOfSaleInfo holds POS data
type PointOfSaleInfo struct {
	Codigo string
}

// BuildEvent builds a complete Evento de Contingencia
func (b *ContingencyEventBuilder) BuildEvent(
	ctx context.Context,
	period *models.ContingencyPeriod,
	invoices []models.Invoice,
) (*EventoContingencia, string, error) {
	if len(invoices) == 0 {
		return nil, "", fmt.Errorf("no invoices provided")
	}

	if len(invoices) > 1000 {
		return nil, "", fmt.Errorf("max 1000 DTEs per event, got %d", len(invoices))
	}

	// Load company info
	company, err := b.loadCompanyInfo(ctx, period.CompanyID)
	if err != nil {
		return nil, "", fmt.Errorf("failed to load company: %w", err)
	}

	// Load establishment info
	establishment, err := b.loadEstablishmentInfo(ctx, period.EstablishmentID)
	if err != nil {
		return nil, "", fmt.Errorf("failed to load establishment: %w", err)
	}

	// Load POS info
	pos, err := b.loadPointOfSaleInfo(ctx, period.PointOfSaleID)
	if err != nil {
		return nil, "", fmt.Errorf("failed to load point of sale: %w", err)
	}

	// Generate event codigo_generacion
	codigoGeneracion := strings.ToUpper(uuid.New().String())

	// Get current time in El Salvador
	loc, _ := time.LoadLocation("America/El_Salvador")
	if loc == nil {
		loc = time.FixedZone("CST", -6*60*60)
	}
	now := time.Now().In(loc)

	// Build detalleDTE
	detalleDTE := make([]DetalleDTEItem, len(invoices))
	for i, inv := range invoices {
		// Get tipo_dte from invoice
		tipoDte := "01" // Default to factura
		if inv.DteType != nil && *inv.DteType != "" {
			tipoDte = *inv.DteType
		}

		// Get codigo_generacion - try dte_codigo_generacion first
		var codigoGen string
		if inv.DteCodigoGeneracion != nil && *inv.DteCodigoGeneracion != "" {
			codigoGen = strings.ToUpper(*inv.DteCodigoGeneracion)
		} else {
			// Parse from dte_unsigned if available
			codigoGen, _ = b.extractCodigoFromUnsigned(inv.DteUnsigned)
		}

		detalleDTE[i] = DetalleDTEItem{
			NoItem:           i + 1,
			CodigoGeneracion: codigoGen,
			TipoDoc:          tipoDte,
		}
	}

	// Build motivo
	motivo := MotivoContingencia{
		FInicio:          period.FInicio,
		FPeriodo:         *period.FFin,
		HInicio:          period.HInicio,
		HFin:             *period.HFin,
		TipoContingencia: period.TipoContingencia,
	}

	// Add motivo text if tipo = 5 (Otro)
	if period.TipoContingencia == models.TipoContingenciaOther && period.MotivoContingencia != nil {
		motivo.MotivoContingencia = period.MotivoContingencia
	}

	// Build complete event
	event := &EventoContingencia{
		Identificacion: IdentificacionEvento{
			Version:          3,
			Ambiente:         period.Ambiente,
			CodigoGeneracion: codigoGeneracion,
			FTransmision:     now.Format("2006-01-02"),
			HTransmision:     now.Format("15:04:05"),
		},
		Emisor: EmisorEvento{
			NIT:                 company.NIT,
			Nombre:              company.Nombre,
			NombreComercial:     company.NombreComercial,
			TipoEstablecimiento: establishment.Tipo,
			CodEstablecimiento:  &establishment.Codigo,
			CodPuntoVenta:       &pos.Codigo,
			Telefono:            company.Telefono,
			Correo:              company.Correo,
		},
		DetalleDTE: detalleDTE,
		Motivo:     motivo,
	}

	return event, codigoGeneracion, nil
}

// BuildEventJSON builds and returns JSON bytes
func (b *ContingencyEventBuilder) BuildEventJSON(
	ctx context.Context,
	period *models.ContingencyPeriod,
	invoices []models.Invoice,
) ([]byte, string, error) {
	event, codigoGeneracion, err := b.BuildEvent(ctx, period, invoices)
	if err != nil {
		return nil, "", err
	}

	eventJSON, err := json.Marshal(event)
	if err != nil {
		return nil, "", fmt.Errorf("failed to marshal event: %w", err)
	}

	return eventJSON, codigoGeneracion, nil
}

// loadCompanyInfo loads company data from database
func (b *ContingencyEventBuilder) loadCompanyInfo(ctx context.Context, companyID string) (*CompanyInfo, error) {
	query := `
		SELECT nit, name, nombre_comercial, COALESCE(telefono, ''), email
		FROM companies
		WHERE id = $1
	`

	var company CompanyInfo
	err := b.db.QueryRowContext(ctx, query, companyID).Scan(
		&company.NIT,
		&company.Nombre,
		&company.NombreComercial,
		&company.Telefono,
		&company.Correo,
	)

	if err != nil {
		return nil, err
	}

	return &company, nil
}

// loadEstablishmentInfo loads establishment data
func (b *ContingencyEventBuilder) loadEstablishmentInfo(ctx context.Context, establishmentID string) (*EstablishmentInfo, error) {
	query := `
		SELECT COALESCE(codigo_establecimiento, '0000'), COALESCE(tipo_establecimiento, '01')
		FROM establishments
		WHERE id = $1
	`

	var est EstablishmentInfo
	err := b.db.QueryRowContext(ctx, query, establishmentID).Scan(
		&est.Codigo,
		&est.Tipo,
	)

	if err != nil {
		return nil, err
	}

	return &est, nil
}

// loadPointOfSaleInfo loads POS data
func (b *ContingencyEventBuilder) loadPointOfSaleInfo(ctx context.Context, posID string) (*PointOfSaleInfo, error) {
	query := `
		SELECT COALESCE(codigo_punto_venta, '0000')
		FROM point_of_sale
		WHERE id = $1
	`

	var pos PointOfSaleInfo
	err := b.db.QueryRowContext(ctx, query, posID).Scan(&pos.Codigo)

	if err != nil {
		return nil, err
	}

	return &pos, nil
}

// extractCodigoFromUnsigned extracts codigoGeneracion from unsigned DTE JSON
func (b *ContingencyEventBuilder) extractCodigoFromUnsigned(dteUnsigned []byte) (string, error) {
	if len(dteUnsigned) == 0 {
		return "", fmt.Errorf("empty dte_unsigned")
	}

	var dte struct {
		Identificacion struct {
			CodigoGeneracion string `json:"codigoGeneracion"`
		} `json:"identificacion"`
	}

	if err := json.Unmarshal(dteUnsigned, &dte); err != nil {
		return "", err
	}

	return strings.ToUpper(dte.Identificacion.CodigoGeneracion), nil
}

// BuildLotePayload builds the lote submission payload for Hacienda
func (b *ContingencyEventBuilder) BuildLotePayload(
	ambiente string,
	nitEmisor string,
	signedDTEs []string,
) (map[string]interface{}, string) {
	idEnvio := strings.ToUpper(uuid.New().String())

	payload := map[string]interface{}{
		"ambiente":   ambiente,
		"idEnvio":    idEnvio,
		"version":    1,
		"nitEmisor":  nitEmisor,
		"documentos": signedDTEs,
	}

	return payload, idEnvio
}
