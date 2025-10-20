package dte

import (
	"context"
	"cuentas/internal/codigos"
	"cuentas/internal/models"
	"fmt"
	"strings"
	"time"
)

// BuildNotaDebito builds a Nota de Débito DTE (separate from invoice)
func (b *Builder) BuildNotaDebito(ctx context.Context, nota *models.Nota) (*NotaDebitoElectronica, error) {
	// Validate nota type
	if nota.Type != codigos.DocTypeNotaDebito {
		return nil, fmt.Errorf("invalid nota type for Nota de Débito: expected %s, got %s",
			codigos.DocTypeNotaDebito, nota.Type)
	}

	// Validate related documents
	if len(nota.RelatedDocuments) == 0 {
		return nil, fmt.Errorf("nota de débito requires related documents")
	}
	if len(nota.RelatedDocuments) > 50 {
		return nil, fmt.Errorf("nota de débito can reference max 50 documents, got %d", len(nota.RelatedDocuments))
	}

	fmt.Printf("Building Nota de Débito with %d related documents\n", len(nota.RelatedDocuments))

	// Load company data
	company, err := b.loadCompanyData(ctx, nota.CompanyID)
	if err != nil {
		return nil, fmt.Errorf("failed to load company data: %w", err)
	}

	// Build items
	itemAmounts := b.buildNotaCuerpoDocumento(nota)

	// Build the DTE
	nd := &NotaDebitoElectronica{
		Identificacion:       b.buildNotaIdentificacion(codigos.DocTypeNotaDebito, nota, company),
		DocumentoRelacionado: b.buildDocumentoRelacionado(nota.RelatedDocuments),
		Emisor:               b.buildNotaEmisor(nota, company),
		Receptor:             b.buildNotaReceptor(ctx, nota),
		VentaTercero:         nil, // TODO: implement if needed
		CuerpoDocumento:      itemAmounts.Items,
		Resumen:              b.buildNotaResumen(nota, itemAmounts.Amounts),
		Extension:            b.buildNotaExtension(nota),
		Apendice:             nil, // TODO: implement if needed
	}

	return nd, nil
}

// BuildNotaCredito builds a Nota de Crédito DTE
func (b *Builder) BuildNotaCredito(ctx context.Context, nota *models.Nota) (*NotaCreditoElectronica, error) {
	// Similar to BuildNotaDebito but with DocTypeNotaCredito
	if nota.Type != codigos.DocTypeNotaCredito {
		return nil, fmt.Errorf("invalid nota type for Nota de Crédito: expected %s, got %s",
			codigos.DocTypeNotaCredito, nota.Type)
	}

	// ... rest similar to BuildNotaDebito
}

// buildNotaIdentificacion builds identificacion specifically for Notas
func (b *Builder) buildNotaIdentificacion(notaType string, nota *models.Nota, company *CompanyData) Identificacion {
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
		Version:          3, // Version 3 for notas
		Ambiente:         company.DTEAmbiente,
		TipoDte:          notaType, // "05" or "06"
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

// buildNotaCuerpoDocumento builds body for Nota (débito or crédito)
func (b *Builder) buildNotaCuerpoDocumento(nota *models.Nota) struct {
	Items   []CuerpoDocumentoItem
	Amounts []ItemAmounts
} {
	items := make([]CuerpoDocumentoItem, len(nota.LineItems))
	amounts := make([]ItemAmounts, len(nota.LineItems))

	for i, lineItem := range nota.LineItems {
		// Use CCF calculation (prices exclude IVA)
		itemAmount := b.calculator.CalculateCreditoFiscal(
			lineItem.UnitPrice,
			lineItem.Quantity,
			lineItem.DiscountAmount,
		)

		amounts[i] = itemAmount

		var tributos []string
		if itemAmount.VentaGravada > 0 {
			tributos = []string{"20"}
		}

		// Get related document reference
		numeroDocumento := ""
		if lineItem.RelatedDocumentRef != nil {
			numeroDocumento = *lineItem.RelatedDocumentRef
		}

		items[i] = CuerpoDocumentoItem{
			NumItem:         lineItem.LineNumber,
			TipoItem:        lineItem.ItemType,
			NumeroDocumento: numeroDocumento, // ⭐ Reference to parent doc
			Cantidad:        lineItem.Quantity,
			Codigo:          &lineItem.ItemSku,
			CodTributo:      nil,
			UniMedida:       lineItem.UnitOfMeasure,
			Descripcion:     lineItem.ItemName,
			PrecioUni:       itemAmount.PrecioUni,
			MontoDescu:      lineItem.DiscountAmount,
			VentaNoSuj:      0,
			VentaExenta:     0,
			VentaGravada:    itemAmount.VentaGravada,
			Tributos:        tributos,
		}
	}

	return struct {
		Items   []CuerpoDocumentoItem
		Amounts []ItemAmounts
	}{items, amounts}
}

// buildNotaResumen builds resumen for Nota (reuse CCF logic)
func (b *Builder) buildNotaResumen(nota *models.Nota, itemAmounts []ItemAmounts) Resumen {
	// ⭐ Reuse CCF calculation!
	resumenAmounts := b.calculator.CalculateResumenCCF(itemAmounts)

	resumen := Resumen{
		TotalNoSuj:          resumenAmounts.TotalNoSuj,
		TotalExenta:         resumenAmounts.TotalExenta,
		TotalGravada:        resumenAmounts.TotalGravada,
		SubTotalVentas:      resumenAmounts.SubTotalVentas,
		DescuNoSuj:          resumenAmounts.DescuNoSuj,
		DescuExenta:         resumenAmounts.DescuExenta,
		DescuGravada:        resumenAmounts.DescuGravada,
		TotalDescu:          resumenAmounts.TotalDescu,
		SubTotal:            resumenAmounts.SubTotal,
		IvaPerci1:           resumenAmounts.IvaPerci1,
		IvaRete1:            resumenAmounts.IvaRete1,
		ReteRenta:           resumenAmounts.ReteRenta,
		MontoTotalOperacion: resumenAmounts.MontoTotalOperacion,
		TotalPagar:          resumenAmounts.TotalPagar,
		TotalLetras:         b.numberToWords(resumenAmounts.TotalPagar),
		CondicionOperacion:  b.parseCondicionOperacion(*nota.PaymentTerms),
		NumPagoElectronico:  nil,
	}

	// Add tributos array
	if resumenAmounts.TotalIva > 0 {
		resumen.Tributos = &[]Tributo{
			{
				Codigo:      "20",
				Descripcion: "Impuesto al Valor Agregado 13",
				Valor:       resumenAmounts.TotalIva,
			},
		}
	}

	return resumen
}

// buildDocumentoRelacionado builds the related documents section
func (b *Builder) buildDocumentoRelacionado(relatedDocs []models.NotaRelatedDocument) *[]DocumentoRelacionado {
	if len(relatedDocs) == 0 {
		return nil
	}

	docs := make([]DocumentoRelacionado, len(relatedDocs))
	for i, doc := range relatedDocs {
		docs[i] = DocumentoRelacionado{
			TipoDocumento:   doc.DocumentType,
			TipoGeneracion:  doc.GenerationType,
			NumeroDocumento: doc.DocumentNumber,
			FechaEmision:    doc.DocumentDate.Format("2006-01-02"),
		}
	}

	return &docs
}

// buildNotaEmisor, buildNotaReceptor, buildNotaExtension...
// (Similar to invoice versions but accept *models.Nota)
