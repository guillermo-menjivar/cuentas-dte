package dte

import (
	"context"
	"cuentas/internal/codigos"
	"cuentas/internal/models"
	"fmt"
	"strings"
	"time"
)

// internal/dte/builder.go

// Add these methods to your existing Builder

// ============================================
// BUILD NOTA DE DÉBITO
// ============================================

// buildNotaIdentificacion builds identificacion for Nota de Débito
func (b *Builder) buildNotaIdentificacion(nota *models.Nota, company *CompanyData) Identificacion {
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
		TipoDte:          nota.Type, // "06" for Nota de Débito
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

// buildNotaReceptor builds receptor for Nota (always CCF-style)
func (b *Builder) buildNotaReceptor(client *ClientData) *Receptor {
	// Notas are always B2B, so use CCF receptor format
	return b.buildCCFReceptor(client)
}

func (b *Builder) buildNotaCuerpoDocumento(nota *models.Nota) struct {
	Items   []CuerpoDocumentoNota // ⭐ Not CuerpoDocumentoItem!
	Amounts []ItemAmounts
} {
	items := make([]CuerpoDocumentoNota, len(nota.LineItems)) // ⭐ CuerpoDocumentoNota
	amounts := make([]ItemAmounts, len(nota.LineItems))

	for i, lineItem := range nota.LineItems {
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

		numeroDocumento := ""
		if lineItem.RelatedDocumentRef != nil && *lineItem.RelatedDocumentRef != "" {
			numeroDocumento = *lineItem.RelatedDocumentRef
		}

		items[i] = CuerpoDocumentoNota{ // ⭐ CuerpoDocumentoNota
			NumItem:         lineItem.LineNumber,
			TipoItem:        lineItem.ItemType,
			NumeroDocumento: numeroDocumento, // ⭐ String, not pointer - CORRECT!
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
		Items   []CuerpoDocumentoNota
		Amounts []ItemAmounts
	}{items, amounts}
}

// buildNotaResumen builds resumen for Nota (use CCF calculation)
func (b *Builder) buildNotaResumen(nota *models.Nota, itemAmounts []ItemAmounts) Resumen {
	// Reuse CCF calculator
	resumenAmounts := b.calculator.CalculateResumenCCF(itemAmounts)

	resumen := Resumen{
		TotalNoSuj:          resumenAmounts.TotalNoSuj,
		TotalExenta:         resumenAmounts.TotalExenta,
		TotalGravada:        resumenAmounts.TotalGravada,
		SubTotalVentas:      resumenAmounts.SubTotalVentas,
		DescuNoSuj:          resumenAmounts.DescuNoSuj,
		DescuExenta:         resumenAmounts.DescuExenta,
		DescuGravada:        resumenAmounts.DescuGravada,
		PorcentajeDescuento: 0,
		TotalDescu:          resumenAmounts.TotalDescu,
		SubTotal:            resumenAmounts.SubTotal,
		IvaRete1:            resumenAmounts.IvaRete1,
		IvaPerci1:           resumenAmounts.IvaPerci1,
		ReteRenta:           resumenAmounts.ReteRenta,
		MontoTotalOperacion: resumenAmounts.MontoTotalOperacion,
		TotalNoGravado:      resumenAmounts.TotalNoGravado,
		TotalPagar:          resumenAmounts.TotalPagar,
		TotalLetras:         b.numberToWords(resumenAmounts.TotalPagar),
		SaldoFavor:          resumenAmounts.SaldoFavor,
		CondicionOperacion:  b.parseCondicionOperacion(*nota.PaymentTerms),
		Pagos:               nil, // Notas don't have pagos
		NumPagoElectronico:  nil,
	}

	// Add tributos array (like CCF)
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

// buildNotaExtension builds extension for Nota
func (b *Builder) buildNotaExtension(nota *models.Nota) *Extension {
	var observaciones *string
	if nota.Notes != nil {
		observaciones = nota.Notes
	}

	return &Extension{
		NombEntrega:   nil,
		DocuEntrega:   nil,
		NombRecibe:    nil,
		DocuRecibe:    nil,
		Observaciones: observaciones,
		PlacaVehiculo: nil,
	}
}

// ============================================
// BUILD NOTA DE CRÉDITO (same structure)
// ============================================

// BuildNotaCredito converts a nota into a Nota de Crédito Electrónica
func (b *Builder) BuildNotaCredito(ctx context.Context, nota *models.Nota) (*DTE, error) {
	// Validate nota type
	if nota.Type != codigos.DocTypeNotaCredito {
		return nil, fmt.Errorf("invalid nota type: expected %s, got %s", codigos.DocTypeNotaCredito, nota.Type)
	}

	// Same implementation as BuildNotaDebito - just different type validation
	// The rest is identical
	return b.BuildNotaDebito(ctx, nota)
}

func (b *Builder) BuildNotaDebito(ctx context.Context, nota *models.Nota) (*NotaDebitoElectronica, error) {
	// Validate
	if nota.Type != codigos.DocTypeNotaDebito {
		return nil, fmt.Errorf("invalid nota type: expected %s, got %s",
			codigos.DocTypeNotaDebito, nota.Type)
	}

	if len(nota.RelatedDocuments) == 0 {
		return nil, fmt.Errorf("nota requires at least 1 related document")
	}

	// Load data
	company, err := b.loadCompanyData(ctx, nota.CompanyID)
	if err != nil {
		return nil, fmt.Errorf("failed to load company: %w", err)
	}

	client, err := b.loadClientData(ctx, nota.ClientID)
	if err != nil {
		return nil, fmt.Errorf("failed to load client: %w", err)
	}

	// ⭐ FIX: Get result as single struct
	itemsResult := b.buildNotaCuerpoDocumento(nota)

	// Build resumen
	resumenAmounts := b.calculator.CalculateResumenCCF(itemsResult.Amounts)
	resumen := b.buildNotaResumen(nota, resumenAmounts)

	// Build related docs
	relDocs := b.buildNotaDocumentoRelacionado(nota.RelatedDocuments)

	nd := &NotaDebitoElectronica{
		Identificacion:       b.buildNotaIdentificacion(codigos.DocTypeNotaDebito, nota, company),
		DocumentoRelacionado: relDocs,
		Emisor:               b.buildEmisorFromCompany(company),
		Receptor:             b.buildReceptorFromClient(client),
		VentaTercero:         nil,
		CuerpoDocumento:      itemsResult.Items, // ⭐ Access .Items
		Resumen:              resumen,
		Extension:            b.buildNotaExtensionFromNota(nota),
		Apendice:             nil,
	}

	return nd, nil
}
