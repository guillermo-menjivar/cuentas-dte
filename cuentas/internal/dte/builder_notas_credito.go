package dte

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"cuentas/internal/codigos"
	"cuentas/internal/models"
)

type NotaCreditoIdentificacion struct {
	Version          int     `json:"version"`
	Ambiente         string  `json:"ambiente"`
	TipoDte          string  `json:"tipoDte"`
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

type NotaCreditoDTE struct {
	Identificacion       Identificacion          `json:"identificacion"`
	DocumentoRelacionado []DocumentoRelacionado  `json:"documentoRelacionado"`
	Emisor               Emisor                  `json:"emisor"`
	Receptor             Receptor                `json:"receptor"`
	VentaTercero         *VentaTercero           `json:"ventaTercero"`
	CuerpoDocumento      []NotaCreditoCuerpoItem `json:"cuerpoDocumento"`
	Resumen              NotaCreditoResumen      `json:"resumen"`
	Extension            *NotaCreditoExtension   `json:"extension"`
	Apendice             *[]Apendice             `json:"apendice"`
}

type NotaCreditoResumen struct {
	TotalNoSuj          float64    `json:"totalNoSuj"`
	TotalExenta         float64    `json:"totalExenta"`
	TotalGravada        float64    `json:"totalGravada"`
	SubTotalVentas      float64    `json:"subTotalVentas"`
	DescuNoSuj          float64    `json:"descuNoSuj"`
	DescuExenta         float64    `json:"descuExenta"`
	DescuGravada        float64    `json:"descuGravada"`
	TotalDescu          float64    `json:"totalDescu"`
	SubTotal            float64    `json:"subTotal"`
	IvaPerci1           float64    `json:"ivaPerci1"`
	IvaRete1            float64    `json:"ivaRete1"`
	ReteRenta           float64    `json:"reteRenta"`
	MontoTotalOperacion float64    `json:"montoTotalOperacion"`
	TotalLetras         string     `json:"totalLetras"`
	CondicionOperacion  int        `json:"condicionOperacion"`
	Tributos            *[]Tributo `json:"tributos,omitempty"`
}

type NotaCreditoExtension struct {
	NombEntrega   *string `json:"nombEntrega"`
	DocuEntrega   *string `json:"docuEntrega"`
	NombRecibe    *string `json:"nombRecibe"`
	DocuRecibe    *string `json:"docuRecibe"`
	Observaciones *string `json:"observaciones"`
}

type NotaCreditoCuerpoItem struct {
	NumItem         int      `json:"numItem"`
	TipoItem        int      `json:"tipoItem"`
	NumeroDocumento string   `json:"numeroDocumento"` // Required: CCF number being credited
	Cantidad        float64  `json:"cantidad"`
	Codigo          *string  `json:"codigo"`
	CodTributo      *string  `json:"codTributo"`
	UniMedida       int      `json:"uniMedida"`
	Descripcion     *string  `json:"descripcion"`
	PrecioUni       float64  `json:"precioUni"`
	MontoDescu      float64  `json:"montoDescu"`
	VentaNoSuj      float64  `json:"ventaNoSuj"`
	VentaExenta     float64  `json:"ventaExenta"`
	VentaGravada    float64  `json:"ventaGravada"`
	Tributos        []string `json:"tributos"`
}

func (b *Builder) BuildNotaCredito(ctx context.Context, nota *models.NotaCredito) ([]byte, error) {
	// Load required data
	company, err := b.loadCompany(ctx, nota.CompanyID)
	if err != nil {
		return nil, fmt.Errorf("load company: %w", err)
	}

	establishment, err := b.loadEstablishmentAndPOS(ctx, nota.EstablishmentID, nota.PointOfSaleID)
	if err != nil {
		return nil, fmt.Errorf("load establishment: %w", err)
	}

	client, err := b.loadClient(ctx, nota.ClientID)
	if err != nil {
		return nil, fmt.Errorf("load client: %w", err)
	}

	// Build DTE sections
	identificacion := b.buildNotaCreditoIdentificacion(nota, company)
	emisor := b.buildNotaCreditoEmisor(company, establishment)
	receptor := b.buildNotaCreditoReceptor(client)
	documentosRelacionados := b.buildNotaCreditoDocumentosRelacionados(nota)
	cuerpoDocumento, itemAmounts := b.buildNotaCreditoCuerpoDocumento(nota)
	resumen := b.buildNotaCreditoResumen(nota, itemAmounts)
	extension := b.buildNotaCreditoExtension(nota)

	// Tipo "05" uses the EXACT SAME structure as tipo "06"

	dte := &NotaCreditoDTE{
		Identificacion:       identificacion,
		DocumentoRelacionado: *documentosRelacionados,
		Emisor:               emisor,
		Receptor:             *receptor,
		VentaTercero:         nil,
		CuerpoDocumento:      cuerpoDocumento,
		Resumen:              resumen,
		Extension:            extension,
		Apendice:             nil,
	}

	// Marshal to JSON
	return json.Marshal(dte)
}

// buildNotaCreditoIdentificacion builds the Identificacion section for Nota de Crédito
func (b *Builder) buildNotaCreditoIdentificacion(nota *models.NotaCredito, company *CompanyData) Identificacion {
	loc, err := time.LoadLocation("America/El_Salvador")
	if err != nil {
		loc = time.FixedZone("CST", -6*3600)
	}

	var emissionTime time.Time
	if nota.FinalizedAt != nil {
		emissionTime = nota.FinalizedAt.In(loc)
	} else {
		emissionTime = time.Now().In(loc)
	}

	return Identificacion{
		Version:          3,
		Ambiente:         company.DTEAmbiente,
		TipoDte:          codigos.DocTypeNotaCredito, // "05"
		NumeroControl:    strings.ToUpper(*nota.DteNumeroControl),
		CodigoGeneracion: strings.ToUpper(nota.ID),
		TipoModelo:       1,
		TipoOperacion:    1,
		TipoContingencia: nil,
		MotivoContin:     nil,
		FecEmi:           emissionTime.Format("2006-01-02"),
		HorEmi:           emissionTime.Format("15:04:05"),
		TipoMoneda:       "USD",
	}
}

// buildNotaCreditoEmisor - without establishment codes (same as débito)
func (b *Builder) buildNotaCreditoEmisor(company *CompanyData, establishment *EstablishmentData) Emisor {
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
	}
}

// buildNotaCreditoReceptor builds receptor based on client tipo persona (same as débito)
func (b *Builder) buildNotaCreditoReceptor(client *ClientData) *Receptor {
	if client.TipoPersona == codigos.PersonTypeJuridica {
		return b.buildCCFReceptor(client)
	}

	// For natural persons
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
		Direccion:     direccion,
	}
}

// buildNotaCreditoDocumentosRelacionados builds array of referenced CCF documents
func (b *Builder) buildNotaCreditoDocumentosRelacionados(nota *models.NotaCredito) *[]DocumentoRelacionado {
	if len(nota.CCFReferences) == 0 {
		return nil
	}

	docs := make([]DocumentoRelacionado, len(nota.CCFReferences))

	for i, ref := range nota.CCFReferences {
		docs[i] = DocumentoRelacionado{
			TipoDocumento:   codigos.DocTypeComprobanteCredito, // CCF
			TipoGeneracion:  1,                                 // Proceso normal
			NumeroDocumento: ref.CCFNumber,
			FechaEmision:    ref.CCFDate.Format("2006-01-02"),
		}
	}

	return &docs
}

// buildNotaCreditoCuerpoDocumento - builds line items for credit note
func (b *Builder) buildNotaCreditoCuerpoDocumento(nota *models.NotaCredito) ([]NotaCreditoCuerpoItem, []ItemAmounts) {
	items := make([]NotaCreditoCuerpoItem, len(nota.LineItems))
	amounts := make([]ItemAmounts, len(nota.LineItems))

	for i, lineItem := range nota.LineItems {
		// Calculate amounts for this credit
		itemAmount := b.calculator.CalculateCreditoFiscal(
			lineItem.CreditAmount,
			lineItem.QuantityCredited,
			0, // No discount on notas typically
		)

		amounts[i] = itemAmount

		var tributos []string
		if itemAmount.VentaGravada > 0 {
			tributos = []string{"20"} // IVA 13%
		}

		items[i] = NotaCreditoCuerpoItem{
			NumItem:         lineItem.LineNumber,
			TipoItem:        b.parseTipoItem(lineItem.OriginalItemTipoItem),
			NumeroDocumento: lineItem.RelatedCCFNumber, // Required: CCF being credited
			Cantidad:        lineItem.QuantityCredited,
			Codigo:          &lineItem.OriginalItemSku,
			CodTributo:      nil, // Usually nil for standard items
			UniMedida:       b.parseUniMedida(lineItem.OriginalUnitOfMeasure),
			Descripcion:     &lineItem.OriginalItemName,
			PrecioUni:       itemAmount.PrecioUni,
			MontoDescu:      0,
			VentaNoSuj:      0,
			VentaExenta:     0,
			VentaGravada:    itemAmount.VentaGravada,
			Tributos:        tributos,
		}
	}

	return items, amounts
}

// buildNotaCreditoResumen - without forbidden fields (same as débito)
func (b *Builder) buildNotaCreditoResumen(nota *models.NotaCredito, itemAmounts []ItemAmounts) NotaCreditoResumen {
	// Use CCF calculator for resumen
	resumenAmounts := b.calculator.CalculateResumenCCF(itemAmounts)

	resumen := NotaCreditoResumen{
		TotalNoSuj:          0,
		TotalExenta:         0,
		TotalGravada:        resumenAmounts.TotalGravada,
		SubTotalVentas:      resumenAmounts.SubTotalVentas,
		DescuNoSuj:          0,
		DescuExenta:         0,
		DescuGravada:        resumenAmounts.DescuGravada,
		TotalDescu:          resumenAmounts.TotalDescu,
		SubTotal:            resumenAmounts.SubTotal,
		IvaPerci1:           0,
		IvaRete1:            0,
		ReteRenta:           0,
		MontoTotalOperacion: resumenAmounts.MontoTotalOperacion,
		TotalLetras:         b.numberToWords(resumenAmounts.MontoTotalOperacion),
		CondicionOperacion:  b.parseCondicionOperacion(nota.PaymentTerms),
	}

	// Add tributos for IVA
	if resumenAmounts.TotalIva > 0 {
		tributos := []Tributo{
			{
				Codigo:      "20",
				Descripcion: "Impuesto al Valor Agregado 13",
				Valor:       resumenAmounts.TotalIva,
			},
		}
		resumen.Tributos = &tributos
	}

	return resumen
}

// buildNotaCreditoExtension - without placaVehiculo (same as débito)
func (b *Builder) buildNotaCreditoExtension(nota *models.NotaCredito) *NotaCreditoExtension {
	return &NotaCreditoExtension{
		NombEntrega:   nil,
		DocuEntrega:   nil,
		NombRecibe:    nil,
		DocuRecibe:    nil,
		Observaciones: nota.Notes,
	}
}
