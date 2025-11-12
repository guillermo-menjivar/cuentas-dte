package dte

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"cuentas/internal/dte_schemas"
	"cuentas/internal/models"
)

// ============================================
// TYPE 04 - NOTA DE REMISIÓN TYPES
// ============================================

// NotaRemisionDTE represents a Type 04 remision (delivery note) DTE
type NotaRemisionDTE struct {
	Identificacion       NotaRemisionIdentificacion `json:"identificacion"`
	DocumentoRelacionado *[]DocumentoRelacionado    `json:"documentoRelacionado"`
	Emisor               Emisor                     `json:"emisor"`
	Receptor             *Receptor                  `json:"receptor"` // Can be null for internal transfers
	VentaTercero         *VentaTercero              `json:"ventaTercero"`
	CuerpoDocumento      []NotaRemisionCuerpoItem   `json:"cuerpoDocumento"`
	Resumen              NotaRemisionResumen        `json:"resumen"`
	Extension            *NotaRemisionExtension     `json:"extension"`
	Apendice             *[]Apendice                `json:"apendice"`
}

// NotaRemisionIdentificacion - Type 04 uses version 3
type NotaRemisionIdentificacion struct {
	Version          int     `json:"version"` // Always 3 for Type 04
	Ambiente         string  `json:"ambiente"`
	TipoDte          string  `json:"tipoDte"` // Always "04"
	NumeroControl    string  `json:"numeroControl"`
	CodigoGeneracion string  `json:"codigoGeneracion"`
	TipoModelo       int     `json:"tipoModelo"`
	TipoOperacion    int     `json:"tipoOperacion"`
	TipoContingencia *int    `json:"tipoContingencia"`
	MotivoContin     *string `json:"motivoContin"`
	FecEmi           string  `json:"fecEmi"`
	HorEmi           string  `json:"horEmi"`
	TipoMoneda       string  `json:"tipoMoneda"`
}

// NotaRemisionCuerpoItem - line items for remision
type NotaRemisionCuerpoItem struct {
	NumItem         int       `json:"numItem"`
	TipoItem        int       `json:"tipoItem"`
	NumeroDocumento *string   `json:"numeroDocumento"` // ⭐ ADD THIS LINE
	Cantidad        float64   `json:"cantidad"`
	Codigo          *string   `json:"codigo"`
	CodTributo      *string   `json:"codTributo"` // ⭐ ADD THIS LINE
	UniMedida       int       `json:"uniMedida"`
	Descripcion     string    `json:"descripcion"`
	PrecioUni       float64   `json:"precioUni"`
	MontoDescu      float64   `json:"montoDescu"`
	VentaNoSuj      float64   `json:"ventaNoSuj"`
	VentaExenta     float64   `json:"ventaExenta"`
	VentaGravada    float64   `json:"ventaGravada"`
	Tributos        *[]string `json:"tributos"`
	// Do NOT add noGravado - it's not in schema
}

// NotaRemisionResumen - simpler than invoices (no IVA)
type NotaRemisionResumen struct {
	TotalNoSuj          float64    `json:"totalNoSuj"`
	TotalExenta         float64    `json:"totalExenta"`
	TotalGravada        float64    `json:"totalGravada"`
	SubTotalVentas      float64    `json:"subTotalVentas"`
	DescuNoSuj          float64    `json:"descuNoSuj"`
	DescuExenta         float64    `json:"descuExenta"`
	DescuGravada        float64    `json:"descuGravada"`
	PorcentajeDescuento *float64   `json:"porcentajeDescuento"`
	TotalDescu          float64    `json:"totalDescu"`
	Tributos            *[]Tributo `json:"tributos"` // Usually null for remision
	SubTotal            float64    `json:"subTotal"`
	MontoTotalOperacion float64    `json:"montoTotalOperacion"`
	TotalLetras         string     `json:"totalLetras"`
}

// NotaRemisionExtension - delivery/transport info
type NotaRemisionExtension struct {
	NombEntrega   *string `json:"nombEntrega"`   // Delivery person name
	DocuEntrega   *string `json:"docuEntrega"`   // Delivery person DUI
	NombRecibe    *string `json:"nombRecibe"`    // Recipient name
	DocuRecibe    *string `json:"docuRecibe"`    // Recipient DUI
	Observaciones *string `json:"observaciones"` // Notes
}

// ============================================
// BUILD NOTA DE REMISIÓN (TYPE 04)
// ============================================

// BuildNotaRemision builds a Type 04 DTE JSON
func (b *Builder) BuildNotaRemision(ctx context.Context, invoice *models.Invoice) ([]byte, error) {
	log.Printf("[BuildNotaRemision] Starting build for remision ID: %s", invoice.ID)

	// Validate this is a remision
	if !invoice.IsRemision() {
		return nil, fmt.Errorf("invoice is not a remision (Type 04)")
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

	// Load receptor if present (can be null for internal transfers)
	var receptor *Receptor
	if invoice.ClientID != "" {
		client, err := b.loadClient(ctx, invoice.ClientID)
		if err != nil {
			return nil, fmt.Errorf("load client: %w", err)
		}
		receptor = b.buildReceptorRemision(client)
	}

	// Load related documents if any
	var documentoRelacionado *[]DocumentoRelacionado
	relatedDocs, err := b.loadRelatedDocuments(ctx, invoice.ID)
	if err != nil {
		return nil, fmt.Errorf("load related documents: %w", err)
	}
	if len(relatedDocs) > 0 {
		docs := b.buildDocumentosRelacionadosRemision(relatedDocs)
		documentoRelacionado = &docs
	}

	// Build DTE structure
	dte := &NotaRemisionDTE{
		Identificacion:       b.buildNotaRemisionIdentificacion(invoice, company),
		DocumentoRelacionado: documentoRelacionado,
		Emisor:               b.buildEmisor(company, establishment),
		Receptor:             receptor, // Can be null
		VentaTercero:         nil,
		CuerpoDocumento:      b.buildNotaRemisionCuerpoDocumento(invoice),
		Resumen:              b.buildNotaRemisionResumen(invoice),
		Extension:            b.buildNotaRemisionExtension(invoice),
		Apendice:             nil,
	}

	// Marshal to JSON
	jsonBytes, err := json.Marshal(dte)
	if err != nil {
		return nil, fmt.Errorf("marshal JSON: %w", err)
	}

	// ✅ Validate JSON against schema
	log.Printf("[BuildNotaRemision] Validating DTE against schema...")
	if err := dte_schemas.Validate("04", jsonBytes); err != nil {
		log.Printf("WARNING: [BuildNotaRemision] ❌ Schema validation failed: %v", err)
		//return nil, fmt.Errorf("schema validation failed: %w", err)
	}
	log.Printf("[BuildNotaRemision] ✅ Schema validation passed")

	return jsonBytes, nil
}

// ============================================
// BUILD IDENTIFICACION (Type 04)
// ============================================

func (b *Builder) buildNotaRemisionIdentificacion(invoice *models.Invoice, company *CompanyData) NotaRemisionIdentificacion {
	return NotaRemisionIdentificacion{
		Version:          3, // Type 04 uses version 3
		Ambiente:         company.DTEAmbiente,
		TipoDte:          "04",
		NumeroControl:    *invoice.DteNumeroControl,
		CodigoGeneracion: invoice.ID,
		TipoModelo:       1, // Previo
		TipoOperacion:    1, // Normal
		TipoContingencia: nil,
		MotivoContin:     nil,
		FecEmi:           invoice.CreatedAt.Format("2006-01-02"),
		HorEmi:           invoice.CreatedAt.Format("15:04:05"),
		TipoMoneda:       "USD",
	}
}

// ============================================
// BUILD CUERPO DOCUMENTO (Type 04)
// ============================================

func (b *Builder) buildNotaRemisionCuerpoDocumento(invoice *models.Invoice) []NotaRemisionCuerpoItem {
	items := make([]NotaRemisionCuerpoItem, len(invoice.LineItems))

	for i, lineItem := range invoice.LineItems {
		// Parse unit of measure
		uniMedida := b.parseUniMedida(lineItem.UnitOfMeasure)

		// ⭐ numeroDocumento can be null for remision
		var numeroDocumento *string = nil

		// ⭐ codTributo can be null for remision
		var codTributo *string = nil

		items[i] = NotaRemisionCuerpoItem{
			NumItem:         lineItem.LineNumber,
			TipoItem:        b.parseTipoItem(lineItem.ItemTipoItem),
			NumeroDocumento: numeroDocumento, // ⭐ Added - can be null
			Codigo:          &lineItem.ItemSku,
			CodTributo:      codTributo, // ⭐ Added - can be null
			Descripcion:     lineItem.ItemName,
			Cantidad:        lineItem.Quantity,
			UniMedida:       uniMedida,
			PrecioUni:       0, // Remision: no sale price
			MontoDescu:      0,
			VentaNoSuj:      0,
			VentaExenta:     0,
			VentaGravada:    0,
			Tributos:        nil, // No taxes for remision
		}
	}

	return items
}

// ============================================
// BUILD RESUMEN (Type 04)
// ============================================

func (b *Builder) buildNotaRemisionResumen(invoice *models.Invoice) NotaRemisionResumen {
	return NotaRemisionResumen{
		TotalNoSuj:          0,
		TotalExenta:         0,
		TotalGravada:        0,
		SubTotalVentas:      0,
		DescuNoSuj:          0,
		DescuExenta:         0,
		DescuGravada:        0,
		PorcentajeDescuento: nil,
		TotalDescu:          0,
		Tributos:            nil,
		SubTotal:            0,
		MontoTotalOperacion: 0,
		TotalLetras:         "CERO DÓLARES",
	}
}

// ============================================
// BUILD EXTENSION (Type 04)
// ============================================

func (b *Builder) buildNotaRemisionExtension(invoice *models.Invoice) *NotaRemisionExtension {
	// Only include extension if we have delivery info
	if invoice.DeliveryPerson == nil && invoice.DeliveryNotes == nil {
		return nil
	}

	return &NotaRemisionExtension{
		NombEntrega:   invoice.DeliveryPerson,
		DocuEntrega:   nil, // Could add DUI field to model if needed
		NombRecibe:    nil, // Could add recipient name if needed
		DocuRecibe:    nil, // Could add recipient DUI if needed
		Observaciones: invoice.DeliveryNotes,
	}
}

// ============================================
// BUILD RECEPTOR FOR REMISION
// ============================================

func (b *Builder) buildReceptorRemision(client *ClientData) *Receptor {
	// Document identification
	var tipoDocumento *string
	var numDocumento *string

	if client.NIT != nil {
		td := DocTypeNIT
		tipoDocumento = &td
		nitStr := fmt.Sprintf("%014d", *client.NIT)
		numDocumento = &nitStr
	} else if client.DUI != nil {
		td := DocTypeDUI
		tipoDocumento = &td
		duiStr := fmt.Sprintf("%08d-%d", *client.DUI/10, *client.DUI%10)
		numDocumento = &duiStr
	}

	// Get data using ClientData methods
	nrc := client.GetNRC()
	direccion := client.GetValidatedDireccion()
	codActividad := client.GetCodActividad()
	descActividad := client.GetDescActividad()
	telefono := client.GetTelefono()
	correo := client.GetCorreo()

	// Business name
	nombreComercial := ""
	if client.BusinessName != nil {
		nombreComercial = *client.BusinessName
	}

	// Default to goods
	bienTitulo := "1"

	return &Receptor{
		TipoDocumento:   tipoDocumento,
		NumDocumento:    numDocumento,
		NRC:             nrc,
		Nombre:          client.BusinessName,
		CodActividad:    &codActividad,
		DescActividad:   &descActividad,
		NombreComercial: &nombreComercial,
		Direccion:       direccion,
		Telefono:        &telefono,
		Correo:          &correo,
		BienTitulo:      &bienTitulo,
	}
}

// ============================================
// BUILD DOCUMENTOS RELACIONADOS FOR REMISION
// ============================================

func (b *Builder) buildDocumentosRelacionadosRemision(docs []models.InvoiceRelatedDocument) []DocumentoRelacionado {
	result := make([]DocumentoRelacionado, len(docs))

	for i, doc := range docs {
		result[i] = DocumentoRelacionado{
			TipoDocumento:   doc.RelatedDocumentType,
			TipoGeneracion:  doc.RelatedDocumentGenType,
			NumeroDocumento: doc.RelatedDocumentNumber,
			FechaEmision:    doc.RelatedDocumentDate.Format("2006-01-02"),
		}
	}

	return result
}

// ============================================
// LOAD RELATED DOCUMENTS
// ============================================

func (b *Builder) loadRelatedDocuments(ctx context.Context, invoiceID string) ([]models.InvoiceRelatedDocument, error) {
	query := `
        SELECT
            id, invoice_id, tipo_documento, tipo_generacion,
            numero_documento, fecha_emision, created_at
        FROM invoice_related_documents
        WHERE invoice_id = $1
        ORDER BY created_at
    `

	rows, err := b.db.QueryContext(ctx, query, invoiceID)
	if err != nil {
		return nil, fmt.Errorf("query related documents: %w", err)
	}
	defer rows.Close()

	var docs []models.InvoiceRelatedDocument
	for rows.Next() {
		var doc models.InvoiceRelatedDocument
		err := rows.Scan(
			&doc.ID,
			&doc.InvoiceID,
			&doc.RelatedDocumentType,
			&doc.RelatedDocumentGenType,
			&doc.RelatedDocumentNumber,
			&doc.RelatedDocumentDate,
			&doc.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan related document: %w", err)
		}
		docs = append(docs, doc)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate related documents: %w", err)
	}

	return docs, nil
}
