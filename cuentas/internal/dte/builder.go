package dte

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"cuentas/internal/models"
	"cuentas/internal/models/dte"
)

// DTEBuilder builds DTE documents from invoices
type DTEBuilder struct {
	db *sql.DB
}

// NewDTEBuilder creates a new DTE builder
func NewDTEBuilder(db *sql.DB) *DTEBuilder {
	return &DTEBuilder{
		db: db,
	}
}

// BuildFromInvoice converts an invoice into a Factura Electrónica
func (b *DTEBuilder) BuildFromInvoice(ctx context.Context, invoice *models.Invoice) (*dte.FacturaElectronica, error) {
	// Load all required data
	company, err := b.loadCompany(ctx, invoice.CompanyID)
	if err != nil {
		return nil, fmt.Errorf("failed to load company: %w", err)
	}

	establishment, err := b.loadEstablishment(ctx, invoice.EstablishmentID)
	if err != nil {
		return nil, fmt.Errorf("failed to load establishment: %w", err)
	}

	pos, err := b.loadPointOfSale(ctx, invoice.PointOfSaleID)
	if err != nil {
		return nil, fmt.Errorf("failed to load point of sale: %w", err)
	}

	client, err := b.loadClient(ctx, invoice.ClientID)
	if err != nil {
		return nil, fmt.Errorf("failed to load client: %w", err)
	}

	// Build the DTE
	factura := &dte.FacturaElectronica{
		Identificacion:  b.buildIdentificacion(invoice, company),
		Emisor:          b.buildEmisor(company, establishment),
		Receptor:        b.buildReceptor(client),
		CuerpoDocumento: b.buildCuerpoDocumento(invoice),
		Resumen:         b.buildResumen(invoice),
		Extension:       b.buildExtension(invoice),
	}

	return factura, nil
}

// buildIdentificacion builds the identificacion section
func (b *DTEBuilder) buildIdentificacion(invoice *models.Invoice, company *CompanyData) dte.Identificacion {
	now := time.Now()

	return dte.Identificacion{
		Version:          1,
		Ambiente:         company.DTEAmbiente,
		TipoDte:          "01", // Factura
		NumeroControl:    *invoice.DteNumeroControl,
		CodigoGeneracion: *invoice.DteCodigoGeneracion,
		TipoModelo:       1,
		TipoOperacion:    1,
		TipoContingencia: nil,
		MotivoContin:     nil,
		FecEmi:           now.Format("2006-01-02"),
		HorEmi:           now.Format("15:04:05"),
		TipoMoneda:       "USD",
	}
}

// buildEmisor builds the emisor section
func (b *DTEBuilder) buildEmisor(company *CompanyData, establishment *EstablishmentData) dte.Emisor {
	return dte.Emisor{
		NIT:                 fmt.Sprintf("%014d", company.NIT),
		NRC:                 fmt.Sprintf("%d", company.NCR),
		Nombre:              company.Name,
		CodActividad:        company.CodActividad,
		DescActividad:       company.DescActividad,
		NombreComercial:     company.NombreComercial,
		TipoEstablecimiento: establishment.TipoEstablecimiento,
		Direccion: dte.Direccion{
			Departamento: establishment.Departamento,
			Municipio:    establishment.Municipio,
			Complemento:  establishment.ComplementoDireccion,
		},
		Telefono:        establishment.Telefono,
		Correo:          company.Email,
		CodEstableMH:    &establishment.CodEstablecimiento,
		CodEstable:      &establishment.CodEstablecimiento,
		CodPuntoVentaMH: &establishment.CodPuntoVenta,
		CodPuntoVenta:   &establishment.CodPuntoVenta,
	}
}

// buildReceptor builds the receptor section
func (b *DTEBuilder) buildReceptor(client *ClientData) *dte.ReceptorFactura {
	// Determine document type and number
	var tipoDocumento *string
	var numDocumento *string
	var nrc *string

	if client.NIT != nil {
		td := "36" // NIT
		tipoDocumento = &td
		nitStr := fmt.Sprintf("%014d", *client.NIT)
		numDocumento = &nitStr
		if client.NCR != nil {
			ncrStr := fmt.Sprintf("%d", *client.NCR)
			nrc = &ncrStr
		}
	} else if client.DUI != nil {
		td := "13" // DUI
		tipoDocumento = &td
		duiStr := fmt.Sprintf("%08d-%d", *client.DUI/10, *client.DUI%10)
		numDocumento = &duiStr
	}

	// Build direccion
	var direccion *dte.Direccion
	if client.DepartmentCode != "" && client.MunicipalityCode != "" {
		direccion = &dte.Direccion{
			Departamento: client.DepartmentCode,
			Municipio:    client.MunicipalityCode,
			Complemento:  client.FullAddress,
		}
	}

	return &dte.ReceptorFactura{
		TipoDocumento: tipoDocumento,
		NumDocumento:  numDocumento,
		NRC:           nrc,
		Nombre:        &client.BusinessName,
		CodActividad:  nil, // Not available in client table
		DescActividad: nil, // Not available in client table
		Direccion:     direccion,
		Telefono:      nil, // Not available in client table
		Correo:        nil, // Not available in client table
	}
}

// buildCuerpoDocumento builds the line items
func (b *DTEBuilder) buildCuerpoDocumento(invoice *models.Invoice) []dte.CuerpoDocumentoFactura {
	items := make([]dte.CuerpoDocumentoFactura, len(invoice.LineItems))

	for i, lineItem := range invoice.LineItems {
		items[i] = dte.CuerpoDocumentoFactura{
			CuerpoDocumentoBase: dte.CuerpoDocumentoBase{
				NumItem:         lineItem.LineNumber,
				TipoItem:        b.parseTipoItem(lineItem.ItemTipoItem),
				NumeroDocumento: nil,
				Codigo:          &lineItem.ItemSku,
				CodTributo:      nil, // For tipoItem != 4
				Descripcion:     lineItem.ItemName,
				Cantidad:        lineItem.Quantity,
				UniMedida:       b.parseUniMedida(lineItem.UnitOfMeasure),
				PrecioUni:       lineItem.UnitPrice,
				MontoDescu:      lineItem.DiscountAmount,
				VentaNoSuj:      0, // TODO: Calculate based on taxes
				VentaExenta:     0, // TODO: Calculate based on taxes
				VentaGravada:    lineItem.TaxableAmount,
				Tributos:        b.buildTributos(lineItem.Taxes),
				Psv:             0,
				NoGravado:       0,
			},
			IvaItem: b.calculateIVA(lineItem.Taxes),
		}
	}

	return items
}

// buildTributos extracts tributo codes from line item taxes
func (b *DTEBuilder) buildTributos(taxes []models.InvoiceLineItemTax) []string {
	tributos := make([]string, 0)
	for _, tax := range taxes {
		// Extract the code part after the dot (e.g., "S1.20" -> "20")
		// For now, just use the full code - we'll refine this
		tributos = append(tributos, tax.TributoCode)
	}
	return tributos
}

// calculateIVA calculates IVA amount from taxes
func (b *DTEBuilder) calculateIVA(taxes []models.InvoiceLineItemTax) float64 {
	for _, tax := range taxes {
		// IVA 13% is typically code "20" or similar
		if tax.TaxRate == 0.13 {
			return tax.TaxAmount
		}
	}
	return 0
}

// buildResumen builds the summary section
func (b *DTEBuilder) buildResumen(invoice *models.Invoice) dte.ResumenFactura {
	return dte.ResumenFactura{
		ResumenBase: dte.ResumenBase{
			TotalNoSuj:          0, // TODO: Calculate
			TotalExenta:         0, // TODO: Calculate
			TotalGravada:        invoice.Subtotal - invoice.TotalDiscount,
			SubTotalVentas:      invoice.Subtotal,
			DescuNoSuj:          0,
			DescuExenta:         0,
			DescuGravada:        invoice.TotalDiscount,
			PorcentajeDescuento: 0, // TODO: Calculate if needed
			TotalDescu:          invoice.TotalDiscount,
			Tributos:            b.buildResumenTributos(invoice),
			SubTotal:            invoice.Subtotal + invoice.TotalTaxes - invoice.TotalDiscount,
			ReteRenta:           0,
			MontoTotalOperacion: invoice.Total,
			TotalNoGravado:      0,
			TotalPagar:          invoice.Total,
			TotalLetras:         b.numberToWords(invoice.Total),
			SaldoFavor:          0,
			CondicionOperacion:  b.parseCondicionOperacion(invoice.PaymentTerms),
			Pagos:               b.buildPagos(invoice),
			NumPagoElectronico:  nil,
		},
		IvaRete1: 0,
		TotalIva: invoice.TotalTaxes,
	}
}

// buildResumenTributos builds summary of all taxes
func (b *DTEBuilder) buildResumenTributos(invoice *models.Invoice) []dte.Tributo {
	// Aggregate taxes across all line items
	taxMap := make(map[string]*dte.Tributo)

	for _, lineItem := range invoice.LineItems {
		for _, tax := range lineItem.Taxes {
			if existing, ok := taxMap[tax.TributoCode]; ok {
				existing.Valor += tax.TaxAmount
			} else {
				taxMap[tax.TributoCode] = &dte.Tributo{
					Codigo:      tax.TributoCode,
					Descripcion: tax.TributoName,
					Valor:       tax.TaxAmount,
				}
			}
		}
	}

	tributos := make([]dte.Tributo, 0, len(taxMap))
	for _, tributo := range taxMap {
		tributos = append(tributos, *tributo)
	}

	return tributos
}

// buildPagos builds payment information
func (b *DTEBuilder) buildPagos(invoice *models.Invoice) []dte.Pago {
	pagos := make([]dte.Pago, 0)

	for _, payment := range invoice.Payments {
		pago := dte.Pago{
			Codigo:     payment.PaymentMethod,
			MontoPago:  payment.Amount,
			Referencia: payment.ReferenceNumber,
			Plazo:      nil,
			Periodo:    nil,
		}
		pagos = append(pagos, pago)
	}

	return pagos
}

// buildExtension builds the extension section
func (b *DTEBuilder) buildExtension(invoice *models.Invoice) *dte.Extension {
	return &dte.Extension{
		NombEntrega:   nil,
		DocuEntrega:   nil,
		NombRecibe:    nil,
		DocuRecibe:    nil,
		Observaciones: invoice.Notes,
		PlacaVehiculo: nil,
	}
}

// Helper functions

func (b *DTEBuilder) parseTipoItem(tipoItem string) int {
	// Convert string tipo_item to int
	// "1" -> 1, "2" -> 2, etc.
	switch tipoItem {
	case "1":
		return 1
	case "2":
		return 2
	case "3":
		return 3
	case "4":
		return 4
	default:
		return 1
	}
}

func (b *DTEBuilder) parseUniMedida(unitOfMeasure string) int {
	// Map unit of measure string to code
	// This is a simplified version - you may need a lookup table
	unitsMap := map[string]int{
		"unidad":    59, // Unidad
		"docena":    11, // Docena
		"caja":      58, // Caja
		"litro":     20, // Litro
		"kilogramo": 14, // Kilogramo
	}

	if code, ok := unitsMap[unitOfMeasure]; ok {
		return code
	}
	return 99 // Otros
}

func (b *DTEBuilder) parseCondicionOperacion(paymentTerms string) int {
	switch paymentTerms {
	case "cash":
		return 1 // Contado
	case "cuenta", "net_30", "net_60":
		return 2 // Crédito
	default:
		return 3 // Otro
	}
}

func (b *DTEBuilder) numberToWords(amount float64) string {
	// TODO: Implement proper number to words conversion in Spanish
	// For now, return a simple format
	return fmt.Sprintf("%.2f DÓLARES", amount)
}

// Data loading functions

type CompanyData struct {
	ID                   string
	NIT                  int64
	NCR                  int64
	Name                 string
	Email                string
	CodActividad         string
	DescActividad        string
	NombreComercial      *string
	DTEAmbiente          string
	Departamento         string
	Municipio            string
	ComplementoDireccion string
	Telefono             string
}

type EstablishmentData struct {
	ID                   string
	TipoEstablecimiento  string
	CodEstablecimiento   string
	CodPuntoVenta        string
	Departamento         string
	Municipio            string
	ComplementoDireccion string
	Telefono             string
}

type ClientData struct {
	ID               string
	NIT              *int64
	NCR              *int64
	DUI              *int64
	BusinessName     string
	DepartmentCode   string
	MunicipalityCode string
	FullAddress      string
}

func (b *DTEBuilder) loadCompany(ctx context.Context, companyID string) (*CompanyData, error) {
	query := `
		SELECT id, nit, ncr, name, email, cod_actividad, desc_actividad, nombre_comercial, 
		       dte_ambiente, departamento, municipio, complemento_direccion, telefono
		FROM companies
		WHERE id = $1
	`

	var company CompanyData
	err := b.db.QueryRowContext(ctx, query, companyID).Scan(
		&company.ID,
		&company.NIT,
		&company.NCR,
		&company.Name,
		&company.Email,
		&company.CodActividad,
		&company.DescActividad,
		&company.NombreComercial,
		&company.DTEAmbiente,
		&company.Departamento,
		&company.Municipio,
		&company.ComplementoDireccion,
		&company.Telefono,
	)
	if err != nil {
		return nil, err
	}

	return &company, nil
}

func (b *DTEBuilder) loadEstablishment(ctx context.Context, establishmentID string) (*EstablishmentData, error) {
	query := `
		SELECT e.id, e.tipo_establecimiento, e.cod_establecimiento, p.cod_punto_venta,
		       e.departamento, e.municipio, e.complemento_direccion, e.telefono
		FROM establishments e
		LEFT JOIN point_of_sale p ON p.establishment_id = e.id
		WHERE e.id = $1
	`

	var est EstablishmentData
	err := b.db.QueryRowContext(ctx, query, establishmentID).Scan(
		&est.ID,
		&est.TipoEstablecimiento,
		&est.CodEstablecimiento,
		&est.CodPuntoVenta,
		&est.Departamento,
		&est.Municipio,
		&est.ComplementoDireccion,
		&est.Telefono,
	)
	if err != nil {
		return nil, err
	}

	return &est, nil
}

func (b *DTEBuilder) loadPointOfSale(ctx context.Context, posID string) (*EstablishmentData, error) {
	query := `
		SELECT e.id, e.tipo_establecimiento, e.cod_establecimiento, p.cod_punto_venta,
		       e.departamento, e.municipio, e.complemento_direccion, e.telefono
		FROM point_of_sale p
		JOIN establishments e ON e.id = p.establishment_id
		WHERE p.id = $1
	`

	var est EstablishmentData
	err := b.db.QueryRowContext(ctx, query, posID).Scan(
		&est.ID,
		&est.TipoEstablecimiento,
		&est.CodEstablecimiento,
		&est.CodPuntoVenta,
		&est.Departamento,
		&est.Municipio,
		&est.ComplementoDireccion,
		&est.Telefono,
	)
	if err != nil {
		return nil, err
	}

	return &est, nil
}

func (b *DTEBuilder) loadClient(ctx context.Context, clientID string) (*ClientData, error) {
	query := `
		SELECT id, nit, ncr, dui, business_name, department_code, municipality_code, full_address
		FROM clients
		WHERE id = $1
	`

	var client ClientData
	err := b.db.QueryRowContext(ctx, query, clientID).Scan(
		&client.ID,
		&client.NIT,
		&client.NCR,
		&client.DUI,
		&client.BusinessName,
		&client.DepartmentCode,
		&client.MunicipalityCode,
		&client.FullAddress,
	)
	if err != nil {
		return nil, err
	}

	return &client, nil
}
