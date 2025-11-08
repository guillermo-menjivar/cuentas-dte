package dte

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"cuentas/internal/codigos"
	"cuentas/internal/models"
)

func init() {
	// Initialize validator on package load
	if err := InitGlobalValidator(); err != nil {
		log.Printf("WARNING: Failed to initialize DTE validator: %v", err)
		log.Printf("DTE validation will be skipped!")
	}
}

// ============================================
// TYPE 11 - FACTURA DE EXPORTACIÓN TYPES
// ============================================

// FacturaExportacionDTE represents a Type 11 export invoice DTE
type FacturaExportacionDTE struct {
	Identificacion       FacturaExportacionIdentificacion `json:"identificacion"`
	DocumentoRelacionado *[]DocumentoRelacionado          `json:"documentoRelacionado,omitempty"`
	Emisor               FacturaExportacionEmisor         `json:"emisor"`
	Receptor             *FacturaExportacionReceptor      `json:"receptor"`
	OtrosDocumentos      *[]FacturaExportacionOtroDoc     `json:"otrosDocumentos"`
	VentaTercero         *VentaTercero                    `json:"ventaTercero"`
	CuerpoDocumento      []FacturaExportacionCuerpoItem   `json:"cuerpoDocumento"`
	Resumen              FacturaExportacionResumen        `json:"resumen"`
	Apendice             *[]Apendice                      `json:"apendice"`
}

// FacturaExportacionIdentificacion - Type 11 uses version 1
type FacturaExportacionIdentificacion struct {
	Version           int     `json:"version"` // Always 1 for Type 11
	Ambiente          string  `json:"ambiente"`
	TipoDte           string  `json:"tipoDte"` // Always "11"
	NumeroControl     string  `json:"numeroControl"`
	CodigoGeneracion  string  `json:"codigoGeneracion"`
	TipoModelo        int     `json:"tipoModelo"`
	TipoOperacion     int     `json:"tipoOperacion"`
	TipoContingencia  *int    `json:"tipoContingencia"`
	MotivoContigencia *string `json:"motivoContigencia"` // Note: typo in schema
	FecEmi            string  `json:"fecEmi"`
	HorEmi            string  `json:"horEmi"`
	TipoMoneda        string  `json:"tipoMoneda"`
}

// FacturaExportacionEmisor - Type 11 has export-specific fields
type FacturaExportacionEmisor struct {
	NIT                 string    `json:"nit"`
	NRC                 string    `json:"nrc"`
	Nombre              string    `json:"nombre"`
	CodActividad        string    `json:"codActividad"`
	DescActividad       string    `json:"descActividad"`
	NombreComercial     *string   `json:"nombreComercial"`
	TipoEstablecimiento string    `json:"tipoEstablecimiento"`
	Direccion           Direccion `json:"direccion"`
	Telefono            string    `json:"telefono"`
	Correo              string    `json:"correo"`
	CodEstableMH        *string   `json:"codEstableMH"`
	CodEstable          *string   `json:"codEstable"`
	CodPuntoVentaMH     *string   `json:"codPuntoVentaMH"`
	CodPuntoVenta       *string   `json:"codPuntoVenta"`
	TipoItemExpor       int       `json:"tipoItemExpor"` // 1, 2, or 3
	RecintoFiscal       *string   `json:"recintoFiscal"` // Nullable
	Regimen             *string   `json:"regimen"`       // Nullable
}

// FacturaExportacionReceptor - International client (no El Salvador address)
type FacturaExportacionReceptor struct {
	Nombre          string  `json:"nombre"`
	TipoDocumento   string  `json:"tipoDocumento"`   // 36,13,02,03,37
	NumDocumento    string  `json:"numDocumento"`    // International format
	NombreComercial *string `json:"nombreComercial"` // Nullable
	CodPais         string  `json:"codPais"`         // Country code (9xxx)
	NombrePais      string  `json:"nombrePais"`      // Country name
	Complemento     string  `json:"complemento"`     // Free-form address
	TipoPersona     int     `json:"tipoPersona"`     // 1 or 2
	DescActividad   string  `json:"descActividad"`   // Required
	Telefono        *string `json:"telefono"`        // Nullable
	Correo          *string `json:"correo"`          // Nullable (required if total > $10k)
}

// FacturaExportacionOtroDoc - Export documents (customs, transport, etc.)
type FacturaExportacionOtroDoc struct {
	CodDocAsociado   int     `json:"codDocAsociado"` // 1-4
	DescDocumento    *string `json:"descDocumento"`
	DetalleDocumento *string `json:"detalleDocumento"`
	PlacaTrans       *string `json:"placaTrans"`
	ModoTransp       *int    `json:"modoTransp"` // 1-7
	NumConductor     *string `json:"numConductor"`
	NombreConductor  *string `json:"nombreConductor"`
}

// FacturaExportacionCuerpoItem - Type 11 line item
type FacturaExportacionCuerpoItem struct {
	NumItem      int      `json:"numItem"`
	Cantidad     float64  `json:"cantidad"`
	Codigo       *string  `json:"codigo"`
	UniMedida    int      `json:"uniMedida"`
	Descripcion  string   `json:"descripcion"`
	PrecioUni    float64  `json:"precioUni"`
	MontoDescu   float64  `json:"montoDescu"`
	VentaGravada float64  `json:"ventaGravada"` // 0% IVA rate
	Tributos     []string `json:"tributos"`     // ["C3"] for 0% export
	NoGravado    float64  `json:"noGravado"`
}

// FacturaExportacionResumen - Type 11 summary (0% IVA)
type FacturaExportacionResumen struct {
	TotalGravada        float64  `json:"totalGravada"`
	Descuento           float64  `json:"descuento"`
	PorcentajeDescuento float64  `json:"porcentajeDescuento"`
	TotalDescu          float64  `json:"totalDescu"`
	Seguro              *float64 `json:"seguro"` // Insurance
	Flete               *float64 `json:"flete"`  // Freight
	MontoTotalOperacion float64  `json:"montoTotalOperacion"`
	TotalNoGravado      float64  `json:"totalNoGravado"` // Seguro + Flete
	TotalPagar          float64  `json:"totalPagar"`
	TotalLetras         string   `json:"totalLetras"`
	CondicionOperacion  int      `json:"condicionOperacion"`
	Pagos               *[]Pago  `json:"pagos"`
	CodIncoterms        *string  `json:"codIncoterms"`  // FOB, CIF, etc.
	DescIncoterms       *string  `json:"descIncoterms"` // Description
	NumPagoElectronico  *string  `json:"numPagoElectronico"`
	Observaciones       *string  `json:"observaciones"`
}

// ============================================
// BUILDER METHOD FOR TYPE 11
// ============================================

// BuildFacturaExportacion converts an export invoice (Type 11) into a DTE
func (b *Builder) BuildFacturaExportacion(ctx context.Context, invoice *models.Invoice) ([]byte, error) {
	// Validate this is an export invoice
	if !invoice.IsExportInvoice() {
		return nil, fmt.Errorf("invoice is not an export invoice (Type 11)")
	}

	// Load all required data
	company, err := b.loadCompany(ctx, invoice.CompanyID)
	if err != nil {
		return nil, fmt.Errorf("load company: %w", err)
	}

	establishment, err := b.loadEstablishmentAndPOS(ctx, invoice.EstablishmentID, invoice.PointOfSaleID)
	if err != nil {
		return nil, fmt.Errorf("load establishment: %w", err)
	}

	// Load export documents
	exportDocs, err := b.loadExportDocuments(ctx, invoice.ID)
	if err != nil {
		return nil, fmt.Errorf("load export documents: %w", err)
	}

	// Build the DTE
	cuerpoDocumento, itemAmounts := b.buildExportacionCuerpoDocumento(invoice)
	resumen := b.buildExportacionResumen(invoice, itemAmounts)

	factura := &FacturaExportacionDTE{
		Identificacion:       b.buildExportacionIdentificacion(invoice, company),
		DocumentoRelacionado: nil,
		Emisor:               b.buildExportacionEmisor(company, establishment, invoice),
		Receptor:             b.buildExportacionReceptor(invoice),
		OtrosDocumentos:      b.buildExportacionOtrosDocumentos(exportDocs),
		VentaTercero:         nil,
		CuerpoDocumento:      cuerpoDocumento,
		Resumen:              resumen,
		Apendice:             nil,
	}

	log.Printf("[BuildFacturaExportacion] Validating DTE against schema...")
	if err := ValidateBeforeSubmission("11", factura); err != nil {
		log.Printf("[BuildFacturaExportacion] ❌ Schema validation failed: %v", err)
		return nil, fmt.Errorf("schema validation failed: %w", err)
	}
	log.Printf("[BuildFacturaExportacion] ✅ Schema validation passed")
	// Marshal to JSON
	return MarshalFacturaExportacion(factura)
}

// ============================================
// BUILD IDENTIFICACION (Type 11)
// ============================================

func (b *Builder) buildExportacionIdentificacion(invoice *models.Invoice, company *CompanyData) FacturaExportacionIdentificacion {
	loc, err := time.LoadLocation("America/El_Salvador")
	if err != nil {
		loc = time.FixedZone("CST", -6*3600)
	}

	var emissionTime time.Time
	if invoice.FinalizedAt != nil {
		emissionTime = invoice.FinalizedAt.In(loc)
	} else {
		emissionTime = time.Now().In(loc)
	}

	return FacturaExportacionIdentificacion{
		Version:           1, // Type 11 uses version 1
		Ambiente:          company.DTEAmbiente,
		TipoDte:           codigos.DocTypeFacturasExportacion, // "11"
		NumeroControl:     strings.ToUpper(*invoice.DteNumeroControl),
		CodigoGeneracion:  invoice.ID,
		TipoModelo:        1,
		TipoOperacion:     1,
		TipoContingencia:  nil,
		MotivoContigencia: nil,
		FecEmi:            emissionTime.Format("2006-01-02"),
		HorEmi:            emissionTime.Format("15:04:05"),
		TipoMoneda:        "USD",
	}
}

// ============================================
// BUILD EMISOR (Type 11)
// ============================================

func (b *Builder) buildExportacionEmisor(company *CompanyData, establishment *EstablishmentData, invoice *models.Invoice) FacturaExportacionEmisor {
	return FacturaExportacionEmisor{
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
		TipoItemExpor:       *invoice.ExportTipoItemExpor,
		RecintoFiscal:       invoice.ExportRecintoFiscal,
		Regimen:             invoice.ExportRegimen,
	}
}

// ============================================
// BUILD RECEPTOR (Type 11 - International)
// ============================================

func (b *Builder) buildExportacionReceptor(invoice *models.Invoice) *FacturaExportacionReceptor {
	// Parse tipo_persona from string to int
	tipoPersona := 1 // Default to natural
	if invoice.ClientTipoPersona != nil && *invoice.ClientTipoPersona == "2" {
		tipoPersona = 2 // Juridica
	}

	return &FacturaExportacionReceptor{
		Nombre:          invoice.ClientName,
		TipoDocumento:   *invoice.ExportReceptorTipoDocumento,
		NumDocumento:    *invoice.ExportReceptorNumDocumento,
		NombreComercial: &invoice.ClientLegalName,
		CodPais:         *invoice.ExportReceptorCodPais,
		NombrePais:      *invoice.ExportReceptorNombrePais,
		Complemento:     *invoice.ExportReceptorComplemento,
		TipoPersona:     tipoPersona,
		DescActividad:   getDefaultActivity(),
		Telefono:        invoice.ContactWhatsapp,
		Correo:          invoice.ContactEmail,
	}
}

// ============================================
// BUILD OTROS DOCUMENTOS (Type 11)
// ============================================

func (b *Builder) buildExportacionOtrosDocumentos(docs []models.InvoiceExportDocument) *[]FacturaExportacionOtroDoc {
	if len(docs) == 0 {
		return nil
	}

	otrosDocs := make([]FacturaExportacionOtroDoc, len(docs))
	for i, doc := range docs {
		otrosDocs[i] = FacturaExportacionOtroDoc{
			CodDocAsociado:   doc.CodDocAsociado,
			DescDocumento:    doc.DescDocumento,
			DetalleDocumento: doc.DetalleDocumento,
			PlacaTrans:       doc.PlacaTrans,
			ModoTransp:       doc.ModoTransp,
			NumConductor:     doc.NumConductor,
			NombreConductor:  doc.NombreConductor,
		}
	}

	return &otrosDocs
}

// ============================================
// BUILD CUERPO DOCUMENTO (Type 11 - 0% IVA)
// ============================================

func (b *Builder) buildExportacionCuerpoDocumento(invoice *models.Invoice) ([]FacturaExportacionCuerpoItem, []ItemAmounts) {
	items := make([]FacturaExportacionCuerpoItem, len(invoice.LineItems))
	amounts := make([]ItemAmounts, len(invoice.LineItems))

	for i, lineItem := range invoice.LineItems {
		// Use export calculator (0% IVA)
		itemAmount := b.calculator.CalculateExportacion(
			lineItem.UnitPrice,
			lineItem.Quantity,
			lineItem.DiscountAmount,
		)

		amounts[i] = itemAmount

		// Tributos must be C3 (0% export)

		tributoFull := codigos.TributoIVAExportaciones                     // "S1.C3"
		tributoCode := tributoFull[strings.LastIndex(tributoFull, ".")+1:] // Extract "C3"
		tributos := []string{tributoCode}

		items[i] = FacturaExportacionCuerpoItem{
			NumItem:      lineItem.LineNumber,
			Cantidad:     lineItem.Quantity,
			Codigo:       &lineItem.ItemSku,
			UniMedida:    b.parseUniMedida(lineItem.UnitOfMeasure),
			Descripcion:  lineItem.ItemName,
			PrecioUni:    itemAmount.PrecioUni,
			MontoDescu:   lineItem.DiscountAmount,
			VentaGravada: itemAmount.VentaGravada,
			Tributos:     tributos,
			NoGravado:    0,
		}
	}

	return items, amounts
}

// ============================================
// BUILD RESUMEN (Type 11 - 0% IVA)
// ============================================

func (b *Builder) buildExportacionResumen(invoice *models.Invoice, itemAmounts []ItemAmounts) FacturaExportacionResumen {
	// Get seguro and flete values
	var seguro, flete float64
	if invoice.ExportSeguro != nil {
		seguro = *invoice.ExportSeguro
	}
	if invoice.ExportFlete != nil {
		flete = *invoice.ExportFlete
	}

	// Use export calculator
	resumenAmounts := b.calculator.CalculateResumenExportacion(itemAmounts, seguro, flete)

	return FacturaExportacionResumen{
		TotalGravada:        resumenAmounts.TotalGravada,
		Descuento:           0, // Global discount not typically used
		PorcentajeDescuento: 0,
		TotalDescu:          resumenAmounts.TotalDescu,
		Seguro:              invoice.ExportSeguro,
		Flete:               invoice.ExportFlete,
		MontoTotalOperacion: resumenAmounts.MontoTotalOperacion,
		TotalNoGravado:      resumenAmounts.TotalNoGravado, // Seguro + Flete
		TotalPagar:          resumenAmounts.TotalPagar,
		TotalLetras:         b.numberToWords(resumenAmounts.TotalPagar),
		CondicionOperacion:  b.parseCondicionOperacion(invoice.PaymentTerms),
		Pagos:               b.buildPagos(invoice),
		CodIncoterms:        invoice.ExportIncotermsCode,
		DescIncoterms:       invoice.ExportIncotermsDesc,
		NumPagoElectronico:  nil,
		Observaciones:       invoice.ExportObservaciones,
	}
}

// ============================================
// HELPER FUNCTIONS
// ============================================

func (b *Builder) loadExportDocuments(ctx context.Context, invoiceID string) ([]models.InvoiceExportDocument, error) {
	query := `
		SELECT id, invoice_id, cod_doc_asociado, desc_documento, detalle_documento,
		       placa_trans, modo_transp, num_conductor, nombre_conductor, created_at
		FROM invoice_export_documents
		WHERE invoice_id = $1
		ORDER BY cod_doc_asociado
	`

	rows, err := b.db.QueryContext(ctx, query, invoiceID)
	if err != nil {
		return nil, fmt.Errorf("query export documents: %w", err)
	}
	defer rows.Close()

	var docs []models.InvoiceExportDocument
	for rows.Next() {
		var doc models.InvoiceExportDocument
		err := rows.Scan(
			&doc.ID,
			&doc.InvoiceID,
			&doc.CodDocAsociado,
			&doc.DescDocumento,
			&doc.DetalleDocumento,
			&doc.PlacaTrans,
			&doc.ModoTransp,
			&doc.NumConductor,
			&doc.NombreConductor,
			&doc.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan export document: %w", err)
		}
		docs = append(docs, doc)
	}

	return docs, rows.Err()
}

func getDefaultActivity() string {
	return "Exportación de bienes y servicios"
}

func MarshalFacturaExportacion(dte *FacturaExportacionDTE) ([]byte, error) {
	jsonBytes, err := json.MarshalIndent(dte, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal export DTE to JSON: %w", err)
	}
	return jsonBytes, nil
}
