package dte

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"cuentas/internal/codigos"
	"cuentas/internal/models"
)

// Builder builds DTE documents from invoices
type Builder struct {
	db         *sql.DB
	calculator *Calculator
	generator  *Generator
}

// NewBuilder creates a new DTE builder
func NewBuilder(db *sql.DB) *Builder {
	return &Builder{
		db:         db,
		calculator: NewCalculator(),
		generator:  NewGenerator(),
	}
}

// BuildFromInvoice converts an invoice into a Factura Electrónica
func (b *Builder) BuildFromInvoice(ctx context.Context, invoice *models.Invoice) (*DTE, error) {
	// Load all required data
	company, err := b.loadCompany(ctx, invoice.CompanyID)
	if err != nil {
		return nil, fmt.Errorf("load company: %w", err)
	}

	establishment, err := b.loadEstablishmentAndPOS(ctx, invoice.EstablishmentID, invoice.PointOfSaleID)
	if err != nil {
		return nil, fmt.Errorf("load establishment: %w", err)
	}

	client, err := b.loadClient(ctx, invoice.ClientID)
	if err != nil {
		return nil, fmt.Errorf("load client: %w", err)
	}

	// Determine invoice type based on client
	invoiceType := b.determineInvoiceType(client)

	// Build the DTE with proper calculations
	cuerpoDocumento, itemAmounts := b.buildCuerpoDocumento(invoice, invoiceType)
	resumen := b.buildResumen(invoice, itemAmounts, invoiceType)

	factura := &DTE{
		Identificacion:       b.buildIdentificacion(invoiceType, invoice, company),
		DocumentoRelacionado: nil,
		Emisor:               b.buildEmisor(company, establishment),
		Receptor:             b.buildReceptor(invoiceType, client),
		OtrosDocumentos:      nil,
		VentaTercero:         nil,
		CuerpoDocumento:      cuerpoDocumento,
		Resumen:              resumen,
		Extension:            b.buildExtension(invoice),
		Apendice:             nil,
	}

	return factura, nil
}

// ============================================
// INVOICE TYPE DETERMINATION
// ============================================

// determineInvoiceType determines if this is B2C or B2B based on client
func (b *Builder) determineInvoiceType(client *ClientData) string {
	// If client has NIT, it's a business (Crédito Fiscal / B2B)

	if client.TipoPersona == "2" {
		return codigos.PersonTypeJuridica // Business (CCF)
	}
	// Otherwise it's a consumer (Consumidor Final / B2C)
	return codigos.PersonTypeNatural
}

// ============================================
// BUILD IDENTIFICACION
// ============================================

func (b *Builder) buildIdentificacion(invoiceType string, invoice *models.Invoice, company *CompanyData) Identificacion {

	switch invoiceType {
	case codigos.PersonTypeJuridica:
		fmt.Println("creating identification CCF")
		return b.buildCCFIdentificacion(invoice, company)
	default:

		// Load El Salvador timezone
		loc, err := time.LoadLocation("America/El_Salvador")
		if err != nil {
			// Fallback to CST offset if timezone data not available
			loc = time.FixedZone("CST", -6*3600)
		}

		// Use finalized_at as the emission date/time
		// Convert to El Salvador timezone
		var emissionTime time.Time
		if invoice.FinalizedAt != nil {
			emissionTime = invoice.FinalizedAt.In(loc)
		} else {
			emissionTime = time.Now().In(loc)
		}

		return Identificacion{
			Version:          1,
			Ambiente:         company.DTEAmbiente,
			TipoDte:          TipoDteFactura,
			NumeroControl:    strings.ToUpper(*invoice.DteNumeroControl),
			CodigoGeneracion: invoice.ID,
			TipoModelo:       1,
			TipoOperacion:    1,
			TipoContingencia: nil,
			MotivoContin:     nil,
			FecEmi:           emissionTime.Format("2006-01-02"),
			HorEmi:           emissionTime.Format("15:04:05"),
			TipoMoneda:       "USD",
		}
	}
}

// ============================================
// BUILD EMISOR
// ============================================

func (b *Builder) buildEmisor(company *CompanyData, establishment *EstablishmentData) Emisor {
	return Emisor{
		NIT:                 company.NIT,
		NRC:                 fmt.Sprintf("%d", company.NCR),
		Nombre:              company.Name,
		CodActividad:        company.CodActividad,
		DescActividad:       company.DescActividad,
		NombreComercial:     &company.NombreComercial,
		TipoEstablecimiento: establishment.TipoEstablecimiento,
		Direccion:           b.buildEmisorDireccion(establishment),
		Telefono:            establishment.Telefono,
		Correo:              company.Email,
		CodEstableMH:        nil,
		CodEstable:          &establishment.CodEstablecimiento,
		CodPuntoVentaMH:     nil,
		CodPuntoVenta:       &establishment.CodPuntoVenta,
	}
}

// ============================================
// BUILD RECEPTOR
// ============================================

func (b *Builder) buildReceptor(invoiceType string, client *ClientData) *Receptor {
	switch invoiceType {
	case codigos.PersonTypeJuridica:
		fmt.Println("building receptor CCF")
		return b.buildCCFReceptor(client)
	default:
		// Determine document type and number
		var tipoDocumento *string
		var numDocumento *string
		var nrc *string

		if client.NIT != nil {
			td := DocTypeNIT
			tipoDocumento = &td
			nitStr := fmt.Sprintf("%014d", *client.NIT)
			numDocumento = &nitStr
			if client.NCR != nil {
				ncrStr := fmt.Sprintf("%d", *client.NCR)
				nrc = &ncrStr
			}
		} else if client.DUI != nil {
			td := DocTypeDUI
			tipoDocumento = &td
			duiStr := fmt.Sprintf("%08d-%d", *client.DUI/10, *client.DUI%10)
			numDocumento = &duiStr
		}

		// Build direccion
		var direccion *Direccion
		if client.DepartmentCode != "" && client.MunicipalityCode != "" {
			dir := b.buildReceptorDireccion(client)
			direccion = &dir
		}

		return &Receptor{
			TipoDocumento: tipoDocumento,
			NumDocumento:  numDocumento,
			NRC:           nrc,
			Nombre:        client.BusinessName,
			CodActividad:  nil, // Not available in client table
			DescActividad: nil, // Not available in client table
			Direccion:     direccion,
			Telefono:      nil, // Not available in client table
			Correo:        nil, // Not available in client table
		}
	}
}

// ============================================
// BUILD CUERPO DOCUMENTO (WITH CALCULATOR!)
// ============================================

func (b *Builder) buildCuerpoDocumento(invoice *models.Invoice, invoiceType string) ([]CuerpoDocumentoItem, []ItemAmounts) {
	fmt.Println("This is the invoiceType you passed", invoiceType)
	fmt.Println("this is the invoice type from codigos", codigos.DocTypeComprobanteCredito)

	items := make([]CuerpoDocumentoItem, len(invoice.LineItems))
	amounts := make([]ItemAmounts, len(invoice.LineItems))

	switch invoiceType {
	case codigos.PersonTypeJuridica:
		fmt.Println("processing CCF cuerpo Documento")
		return b.buildCCFCuerpoDocumento(invoice)
	default:
		fmt.Println("WE ARE TREATING THIS AS A FACTURA")

		for i, lineItem := range invoice.LineItems {
			// ⭐ USE CALCULATOR - This is the critical fix!
			var itemAmount ItemAmounts

			itemAmount = b.calculator.CalculateConsumidorFinal(
				lineItem.UnitPrice,
				lineItem.Quantity,
				lineItem.DiscountAmount,
			)
			//if invoiceType == codigos.PersonTypeNatural {
			// For B2C: lineItem.UnitPrice INCLUDES IVA
			//	itemAmount = b.calculator.CalculateConsumidorFinal(
			//		lineItem.UnitPrice,
			//		lineItem.Quantity,
			//		lineItem.DiscountAmount,
			//	)
			//} else {
			//	// For B2B: lineItem.UnitPrice EXCLUDES IVA
			//	itemAmount = b.calculator.CalculateCreditoFiscal(
			//		lineItem.UnitPrice,
			//		lineItem.Quantity,
			//		lineItem.DiscountAmount,
			//	)
			//}

			amounts[i] = itemAmount

			items[i] = CuerpoDocumentoItem{
				NumItem:         lineItem.LineNumber,
				TipoItem:        b.parseTipoItem(lineItem.ItemTipoItem),
				NumeroDocumento: nil,
				Cantidad:        lineItem.Quantity,
				Codigo:          &lineItem.ItemSku,
				CodTributo:      nil, // Only for tipoItem = 4
				UniMedida:       b.parseUniMedida(lineItem.UnitOfMeasure),
				Descripcion:     lineItem.ItemName,
				PrecioUni:       itemAmount.PrecioUni, // ✅ From calculator
				MontoDescu:      lineItem.DiscountAmount,
				VentaNoSuj:      0,
				VentaExenta:     0,
				VentaGravada:    itemAmount.VentaGravada, // ✅ From calculator
				Tributos:        nil,                     // No special tributos for regular IVA
				Psv:             0,
				NoGravado:       0,
				//IvaItem:         RoundToItemPrecision(itemAmount.IvaItem), // ✅ From calculator
				IvaItem: RoundToResumenPrecision(itemAmount.IvaItem),
			}
		}

		return items, amounts
	}
}

// ============================================
// BUILD RESUMEN (WITH CALCULATOR!)
// ============================================

func (b *Builder) buildResumen(invoice *models.Invoice, itemAmounts []ItemAmounts, invoiceType string) Resumen {
	// ⭐ USE CALCULATOR for resumen totals
	resumenAmounts := b.calculator.CalculateResumen(itemAmounts, invoiceType)

	return Resumen{
		TotalNoSuj:          resumenAmounts.TotalNoSuj,
		TotalExenta:         resumenAmounts.TotalExenta,
		TotalGravada:        resumenAmounts.TotalGravada,
		SubTotalVentas:      resumenAmounts.SubTotalVentas,
		DescuNoSuj:          resumenAmounts.DescuNoSuj,
		DescuExenta:         resumenAmounts.DescuExenta,
		DescuGravada:        resumenAmounts.DescuGravada,
		PorcentajeDescuento: 0,
		TotalDescu:          resumenAmounts.TotalDescu,
		Tributos:            nil, // No special tributos for regular IVA invoices
		SubTotal:            resumenAmounts.SubTotal,
		IvaRete1:            resumenAmounts.IvaRete1,
		ReteRenta:           resumenAmounts.ReteRenta,
		MontoTotalOperacion: resumenAmounts.MontoTotalOperacion,
		TotalNoGravado:      resumenAmounts.TotalNoGravado,
		TotalPagar:          resumenAmounts.TotalPagar,
		TotalLetras:         b.numberToWords(resumenAmounts.TotalPagar),
		TotalIva:            resumenAmounts.TotalIva,
		SaldoFavor:          resumenAmounts.SaldoFavor,
		CondicionOperacion:  b.parseCondicionOperacion(invoice.PaymentTerms),
		Pagos:               b.buildPagos(invoice),
		NumPagoElectronico:  nil,
	}
}

// ============================================
// BUILD PAGOS
// ============================================

func (b *Builder) buildPagos(invoice *models.Invoice) *[]Pago {
	if len(invoice.Payments) == 0 {
		return nil
	}

	pagos := make([]Pago, 0, len(invoice.Payments))

	for _, payment := range invoice.Payments {
		pago := Pago{
			Codigo:     payment.PaymentMethod,
			MontoPago:  payment.Amount,
			Referencia: payment.ReferenceNumber,
			Plazo:      nil,
			Periodo:    nil,
		}
		pagos = append(pagos, pago)
	}

	return &pagos
}

// ============================================
// BUILD EXTENSION
// ============================================

func (b *Builder) buildExtension(invoice *models.Invoice) *Extension {
	return &Extension{
		NombEntrega:   nil,
		DocuEntrega:   nil,
		NombRecibe:    nil,
		DocuRecibe:    nil,
		Observaciones: invoice.Notes,
		PlacaVehiculo: nil,
	}
}

// ============================================
// HELPER FUNCTIONS (PARSERS)
// ============================================

func (b *Builder) parseTipoItem(tipoItem string) int {
	// Convert string tipo_item to int
	switch tipoItem {
	case "1":
		return 1 // Bien
	case "2":
		return 2 // Servicio
	case "3":
		return 3 // Ambos
	case "4":
		return 4 // Tributo
	default:
		return 2 // Default to service
	}
}

func (b *Builder) parseUniMedida(unitOfMeasure string) int {
	// Map unit of measure string to Hacienda codes
	unitsMap := map[string]int{
		"unidad":    59, // Unidad
		"docena":    11, // Docena
		"caja":      58, // Caja
		"litro":     20, // Litro
		"kilogramo": 14, // Kilogramo
		"metro":     40, // Metro
		"servicio":  99, // Servicio/Otros
	}

	if code, ok := unitsMap[unitOfMeasure]; ok {
		return code
	}
	return 99 // Otros
}

func (b *Builder) parseCondicionOperacion(paymentTerms string) int {
	switch paymentTerms {
	case "cash", "contado":
		return 1 // Contado
	case "credit", "cuenta", "net_30", "net_60", "credito":
		return 2 // Crédito
	default:
		return 3 // Otro
	}
}

func (b *Builder) numberToWords(amount float64) string {
	// TODO: Implement proper number to words conversion in Spanish
	// For now, return a simple format
	// You can use a library like github.com/divan/num2words for this
	return fmt.Sprintf("%.2f DÓLARES", amount)
}

// ============================================
// DATABASE QUERIES
// ============================================

func (b *Builder) loadCompany(ctx context.Context, companyID string) (*CompanyData, error) {
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
		return nil, fmt.Errorf("query company: %w", err)
	}

	return &company, nil
}

func (b *Builder) loadClient(ctx context.Context, clientID string) (*ClientData, error) {
	query := `
		SELECT 
			id, nit, ncr, dui, business_name, 
			department_code, municipality_code, full_address, 
			tipo_persona,
			cod_actividad, desc_actividad, telefono, correo
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
		&client.TipoPersona,
		&client.CodActividad,
		&client.DescActividad,
		&client.Telefono,
		&client.Correo,
		// ⭐ Removed nombre_comercial and LegalBusinessName scan
	)
	if err != nil {
		return nil, fmt.Errorf("query client: %w", err)
	}

	return &client, nil
}

func (b *Builder) loadEstablishmentAndPOS(ctx context.Context, establishmentID, posID string) (*EstablishmentData, error) {
	query := `
		SELECT e.id, e.tipo_establecimiento, e.cod_establecimiento, p.cod_punto_venta,
		       e.departamento, e.municipio, e.complemento_direccion, e.telefono
		FROM establishments e
		JOIN point_of_sale p ON p.establishment_id = e.id
		WHERE e.id = $1 AND p.id = $2
	`

	var est EstablishmentData
	err := b.db.QueryRowContext(ctx, query, establishmentID, posID).Scan(
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
		return nil, fmt.Errorf("query establishment: %w", err)
	}

	return &est, nil
}

func (b *Builder) _buildEmisorDireccion(company *CompanyData) Direccion {
	// Validate and extract municipality code
	munCode, valid := codigos.ValidateMunicipalityWithDepartment(
		company.Departamento,
		company.Municipio,
	)
	if !valid {
		// Log warning but don't fail - use as-is
		fmt.Printf("Warning: Invalid municipality code for emisor: dept=%s, mun=%s\n",
			company.Departamento, company.Municipio)
		munCode = codigos.ExtractMunicipalityCode(company.Municipio)
	}

	return Direccion{
		Departamento: company.Departamento,
		Municipio:    munCode,
		Complemento:  company.ComplementoDireccion,
	}
}

func (b *Builder) buildEmisorDireccion(establishment *EstablishmentData) Direccion {
	munCode, valid := codigos.ValidateMunicipalityWithDepartment(
		establishment.Departamento,
		establishment.Municipio,
	)
	if !valid {
		fmt.Printf("Warning: Invalid municipality code for emisor: dept=%s, mun=%s\n",
			establishment.Departamento, establishment.Municipio)
		munCode = codigos.ExtractMunicipalityCode(establishment.Municipio)
	}

	return Direccion{
		Departamento: establishment.Departamento,
		Municipio:    munCode,
		Complemento:  establishment.ComplementoDireccion,
	}
}

// For Receptor
func (b *Builder) buildReceptorDireccion(client *ClientData) Direccion {
	munCode, valid := client.GetMunicipalityCode()

	if !valid {
		fmt.Printf("Warning: Invalid municipality code for receptor: dept=%s, mun=%s\n",
			client.DepartmentCode, munCode)
	}

	return Direccion{
		Departamento: client.DepartmentCode,
		Municipio:    munCode,
		Complemento:  client.FullAddress,
	}
}

func (b *Builder) buildCCFCuerpoDocumento(invoice *models.Invoice) ([]CuerpoDocumentoItem, []ItemAmounts) {
	items := make([]CuerpoDocumentoItem, len(invoice.LineItems))
	amounts := make([]ItemAmounts, len(invoice.LineItems))

	for i, lineItem := range invoice.LineItems {
		// CCF uses CalculateCreditoFiscal
		itemAmount := b.calculator.CalculateCreditoFiscal(
			lineItem.UnitPrice,
			lineItem.Quantity,
			lineItem.DiscountAmount,
		)

		amounts[i] = itemAmount

		// Tributos required for CCF
		var tributos []string
		if itemAmount.VentaGravada > 0 {
			tribs := []string{"20"} // IVA 13%
			tributos = tribs
		}

		items[i] = CuerpoDocumentoItem{
			NumItem:         lineItem.LineNumber,
			TipoItem:        b.parseTipoItem(lineItem.ItemTipoItem),
			NumeroDocumento: nil,
			Cantidad:        lineItem.Quantity,
			Codigo:          &lineItem.ItemSku,
			CodTributo:      nil,
			UniMedida:       b.parseUniMedida(lineItem.UnitOfMeasure),
			Descripcion:     lineItem.ItemName,
			PrecioUni:       itemAmount.PrecioUni,
			MontoDescu:      lineItem.DiscountAmount,
			VentaNoSuj:      0,
			VentaExenta:     0,
			VentaGravada:    itemAmount.VentaGravada,
			Tributos:        tributos, // Required for CCF
			Psv:             0,
			NoGravado:       0,
			IvaItem:         RoundToResumenPrecision(itemAmount.IvaItem),
		}
	}

	return items, amounts
}

func (b *Builder) buildCCFIdentificacion(invoice *models.Invoice, company *CompanyData) Identificacion {
	loc, err := time.LoadLocation("America/El_Salvador")
	if err != nil {
		loc = time.FixedZone("CST", -6*3600)
	}

	emissionTime := invoice.FinalizedAt.In(loc)

	return Identificacion{
		Version:          3,
		Ambiente:         company.DTEAmbiente,
		TipoDte:          codigos.DocTypeComprobanteCredito, // "03"
		NumeroControl:    strings.ToUpper(*invoice.DteNumeroControl),
		CodigoGeneracion: invoice.ID,
		TipoModelo:       1,
		TipoOperacion:    1,
		TipoContingencia: nil,
		MotivoContin:     nil,
		FecEmi:           emissionTime.Format("2006-01-02"),
		HorEmi:           emissionTime.Format("15:04:05"),
		TipoMoneda:       "USD",
	}
}

func (b *Builder) buildCCFReceptor(client *ClientData) *Receptor {
	direccion := b.buildReceptorDireccion(client)

	var nitStr, nrcStr string
	if client.NIT != nil {
		nitStr = fmt.Sprintf("%014d", *client.NIT)
	}
	if client.NCR != nil {
		nrcStr = fmt.Sprintf("%d", *client.NCR)
	}

	return &Receptor{
		NIT:             &nitStr,
		NRC:             &nrcStr,
		Nombre:          client.BusinessName,
		CodActividad:    client.CodActividad,
		DescActividad:   client.DescActividad,
		Direccion:       &direccion,
		Telefono:        client.Telefono,
		Correo:          client.Correo,
		NombreComercial: client.BusinessName,
	}
}
