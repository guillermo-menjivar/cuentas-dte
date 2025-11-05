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

// BuildNotaDebito converts a Nota de Débito into a DTE document
func (b *Builder) BuildNotaDebito(ctx context.Context, nota *models.NotaDebito) ([]byte, error) {
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
	identificacion := b.buildNotaDebitoIdentificacion(nota, company)
	emisor := b.buildEmisor(company, establishment)
	receptor := b.buildNotaDebitoReceptor(client)
	documentosRelacionados := b.buildDocumentosRelacionados(nota)
	cuerpoDocumento, itemAmounts := b.buildNotaDebitoCuerpoDocumento(nota)
	resumen := b.buildNotaDebitoResumen(nota, itemAmounts)

	// Assemble the DTE
	dte := &DTE{
		Identificacion:       identificacion,
		DocumentoRelacionado: documentosRelacionados,
		Emisor:               emisor,
		Receptor:             receptor,
		OtrosDocumentos:      nil,
		VentaTercero:         nil,
		CuerpoDocumento:      cuerpoDocumento,
		Resumen:              resumen,
		Extension:            b.buildNotaDebitoExtension(nota),
		Apendice:             nil,
	}

	// Marshal to JSON
	return json.Marshal(dte)
}

// buildNotaDebitoIdentificacion builds the Identificacion section for Nota de Débito
func (b *Builder) buildNotaDebitoIdentificacion(nota *models.NotaDebito, company *CompanyData) Identificacion {
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

	// Nota de Débito uses version 2 for tipo DTE "06"
	return Identificacion{
		Version:          2,
		Ambiente:         company.DTEAmbiente,
		TipoDte:          "06", // Nota de Débito
		NumeroControl:    strings.ToUpper(*nota.DteNumeroControl),
		CodigoGeneracion: nota.ID,
		TipoModelo:       1,
		TipoOperacion:    1,
		TipoContingencia: nil,
		MotivoContin:     nil,
		FecEmi:           emissionTime.Format("2006-01-02"),
		HorEmi:           emissionTime.Format("15:04:05"),
		TipoMoneda:       "USD",
	}
}

// buildNotaDebitoReceptor builds receptor based on client tipo persona
func (b *Builder) buildNotaDebitoReceptor(client *ClientData) *Receptor {
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

// buildDocumentosRelacionados builds array of referenced CCF documents
func (b *Builder) buildDocumentosRelacionados(nota *models.NotaDebito) *[]DocumentoRelacionado {
	if len(nota.CCFReferences) == 0 {
		return nil
	}

	docs := make([]DocumentoRelacionado, len(nota.CCFReferences))

	for i, ref := range nota.CCFReferences {
		docs[i] = DocumentoRelacionado{
			TipoDocumento:   "03", // CCF
			TipoGeneracion:  1,    // Proceso normal
			NumeroDocumento: ref.CCFNumber,
			FechaEmision:    ref.CCFDate.Format("2006-01-02"),
		}
	}

	return &docs
}

// buildNotaDebitoCuerpoDocumento builds line items for nota
func (b *Builder) buildNotaDebitoCuerpoDocumento(nota *models.NotaDebito) ([]CuerpoDocumentoItem, []ItemAmounts) {
	items := make([]CuerpoDocumentoItem, len(nota.LineItems))
	amounts := make([]ItemAmounts, len(nota.LineItems))

	for i, lineItem := range nota.LineItems {
		// Calculate amounts for this adjustment
		itemAmount := b.calculator.CalculateCreditoFiscal(
			lineItem.AdjustmentAmount,
			lineItem.OriginalQuantity,
			0, // No discount on notas typically
		)

		amounts[i] = itemAmount

		var tributos []string
		if itemAmount.VentaGravada > 0 {
			tributos = []string{"20"} // IVA 13%
		}

		items[i] = CuerpoDocumentoItem{
			NumItem:      lineItem.LineNumber,
			TipoItem:     b.parseTipoItem(lineItem.OriginalItemTipoItem),
			Cantidad:     lineItem.OriginalQuantity,
			Codigo:       &lineItem.OriginalItemSku,
			UniMedida:    b.parseUniMedida(lineItem.OriginalUnitOfMeasure),
			Descripcion:  lineItem.OriginalItemName,
			PrecioUni:    itemAmount.PrecioUni,
			MontoDescu:   0,
			VentaNoSuj:   0,
			VentaExenta:  0,
			VentaGravada: itemAmount.VentaGravada,
			Tributos:     tributos,
		}
	}

	return items, amounts
}

// buildNotaDebitoResumen builds resumen for nota
func (b *Builder) buildNotaDebitoResumen(nota *models.NotaDebito, itemAmounts []ItemAmounts) Resumen {
	// Use CCF calculator for resumen
	resumenAmounts := b.calculator.CalculateResumenCCF(itemAmounts)

	resumen := Resumen{
		TotalGravada:        resumenAmounts.TotalGravada,
		SubTotalVentas:      resumenAmounts.SubTotalVentas,
		DescuGravada:        resumenAmounts.DescuGravada,
		TotalDescu:          resumenAmounts.TotalDescu,
		SubTotal:            resumenAmounts.SubTotal,
		MontoTotalOperacion: resumenAmounts.MontoTotalOperacion,
		TotalPagar:          resumenAmounts.TotalPagar,
		TotalLetras:         b.numberToWords(resumenAmounts.TotalPagar),
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

// buildNotaDebitoExtension builds extension section
func (b *Builder) buildNotaDebitoExtension(nota *models.NotaDebito) *Extension {
	return &Extension{
		Observaciones: nota.Notes,
	}
}
