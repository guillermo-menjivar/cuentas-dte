package dte

import (
	"context"
	"cuentas/internal/codigos"
	"cuentas/internal/models"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// BuildNotaCredito builds DTE for nota de crédito (tipo 05)
// ⭐ REUSES NotaDebitoDTE structs - only changes tipoDte!
func (b *Builder) BuildNotaCredito(ctx context.Context, nota *models.NotaCredito) ([]byte, error) {
	// Load company, establishment, client (SAME as débito)
	company, err := b.getCompany(ctx, nota.CompanyID)
	if err != nil {
		return nil, err
	}

	establishment, err := b.getEstablishment(ctx, nota.EstablishmentID)
	if err != nil {
		return nil, err
	}

	client, err := b.getClient(ctx, nota.ClientID)
	if err != nil {
		return nil, err
	}

	// Build DTE sections
	identificacion := b.buildNotaCreditoIdentificacion(nota)
	emisor := b.buildNotaCreditoEmisor(company, establishment)
	receptor := b.buildNotaCreditoReceptor(client)
	cuerpoDocumento := b.buildNotaCreditoCuerpoDocumento(nota)
	resumen := b.buildNotaCreditoResumen(nota)
	extension := b.buildNotaCreditoExtension(nota)
	documentosRelacionados := b.buildNotaCreditoDocumentosRelacionados(nota)

	// Assemble DTE using NotaDebitoDTE struct
	dte := NotaDebitoDTE{
		Identificacion:       identificacion,
		Emisor:               emisor,
		Receptor:             receptor,
		CuerpoDocumento:      cuerpoDocumento,
		Resumen:              resumen,
		Extension:            extension,
		DocumentoRelacionado: documentosRelacionados,
		Apendice:             nil,
	}

	// Marshal to JSON
	dteJSON, err := json.Marshal(dte)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal DTE: %w", err)
	}

	return dteJSON, nil
}

func (b *Builder) buildNotaCreditoIdentificacion(nota *models.NotaCredito) NotaDebitoIdentificacion {
	var fechaEmision string
	if nota.FinalizedAt != nil {
		fechaEmision = nota.FinalizedAt.Format("2006-01-02")
	} else {
		fechaEmision = time.Now().Format("2006-01-02")
	}

	return NotaDebitoIdentificacion{
		Version:          3,                   // Version 3 for tipo 05
		Ambiente:         codigos.MODE_PRUEBA, // "00" or "01"
		TipoDte:          "05",                // ⭐ NOTA DE CRÉDITO
		NumeroControl:    nota.DteNumeroControl,
		CodigoGeneracion: strings.ToUpper(nota.ID), // UPPERCASE UUID
		TipoModelo:       1,
		TipoOperacion:    1,
		TipoContingencia: nil,
		MotivoContin:     nil,
		FechaEmision:     fechaEmision,
		HoraEmision:      time.Now().Format("15:04:05"),
		TipoMoneda:       "USD",
	}
}

func (b *Builder) buildNotaCreditoEmisor(company, establishment) NotaDebitoEmisor {
	// SAME as nota débito - NO establishment codes
	return NotaDebitoEmisor{
		Nit:    company.Nit,
		Nrc:    company.Nrc,
		Nombre: company.LegalName,
		// ... rest of fields
	}
}

func (b *Builder) buildNotaCreditoReceptor(client) NotaDebitoReceptor {
	// SAME as nota débito
	return NotaDebitoReceptor{
		TipoDocumento: client.TipoDocumento,
		NumDocumento:  client.NumDocumento,
		Nombre:        client.Name,
		// ... rest of fields
	}
}

func (b *Builder) buildNotaCreditoCuerpoDocumento(nota *models.NotaCredito) []NotaDebitoCuerpoDocumentoItem {
	items := make([]NotaDebitoCuerpoDocumentoItem, 0, len(nota.LineItems))

	for _, lineItem := range nota.LineItems {
		// Use CalculateCreditoFiscal calculator (SAME as débito!)
		calc := CalculateCreditoFiscal(
			lineItem.QuantityCredited,
			lineItem.CreditAmount,
			0, // no discount
		)

		item := NotaDebitoCuerpoDocumentoItem{
			NumItem:         lineItem.LineNumber,
			TipoItem:        1,                          // Producto
			NumeroDocumento: &lineItem.RelatedCCFNumber, // ⭐ REQUIRED
			Cantidad:        lineItem.QuantityCredited,
			Codigo:          &lineItem.OriginalItemSku,
			Unimedida:       99, // Unidad
			Descripcion:     lineItem.OriginalItemName,
			PrecioUni:       lineItem.CreditAmount,
			MontoDescu:      0,
			VentaGravada:    calc.VentaGravada,
			Tributos:        []string{"20"}, // IVA
			IvaItem:         calc.IvaItem,
		}

		items = append(items, item)
	}

	return items
}

func (b *Builder) buildNotaCreditoResumen(nota *models.NotaCredito) NotaDebitoResumen {
	// Use CalculateResumenCCF calculator (SAME as débito!)
	calc := CalculateResumenCCF(nota.Subtotal, nota.TotalDiscount)

	tributos := []NotaDebitoTributo{
		{
			Codigo:      "20",
			Descripcion: "Impuesto al Valor Agregado 13%",
			Valor:       nota.TotalTaxes,
		},
	}

	return NotaDebitoResumen{
		TotalGravada:        calc.TotalGravada,
		TotalDescu:          nota.TotalDiscount,
		SubTotal:            calc.SubTotal,
		IvaRete1:            0,
		ReteRenta:           0,
		MontoTotalOperacion: nota.Total,
		TotalNoSuj:          0,
		TotalExenta:         0,
		TotalLetras:         numeroALetras(nota.Total),
		CondicionOperacion:  getCondicionOperacion(nota.PaymentTerms),
		NumPagoElectronico:  nil,
		Tributos:            tributos,
	}
}

func (b *Builder) buildNotaCreditoExtension(nota *models.NotaCredito) *NotaDebitoExtension {
	// SAME as nota débito
	return &NotaDebitoExtension{
		NombEntrega:   nil,
		DocuEntrega:   nil,
		NombRecibe:    nil,
		DocuRecibe:    nil,
		Observaciones: nota.Notes,
	}
}

func (b *Builder) buildNotaCreditoDocumentosRelacionados(nota *models.NotaCredito) []NotaDebitoDocumentoRelacionado {
	docs := make([]NotaDebitoDocumentoRelacionado, 0, len(nota.CCFReferences))

	for _, ref := range nota.CCFReferences {
		doc := NotaDebitoDocumentoRelacionado{
			TipoDocumento:   "03", // CCF
			TipoGeneracion:  1,    // Normal process
			NumeroDocumento: ref.CCFNumber,
			FechaEmision:    ref.CCFDate.Format("2006-01-02"),
		}
		docs = append(docs, doc)
	}

	return docs
}
