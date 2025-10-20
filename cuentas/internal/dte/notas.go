// internal/dte/notas.go

package dte

import (
	"context"
	"fmt"
	"strings"
	"time"

	"cuentas/internal/codigos"
	"cuentas/internal/models"
)

func (b *Builder) BuildNotaDebito(ctx context.Context, nota *models.Nota) (*NotaDebitoElectronica, error) {
	// Validate
	if nota.Type != codigos.DocTypeNotaDebito {
		return nil, fmt.Errorf("invalid nota type")
	}
	if len(nota.RelatedDocuments) == 0 {
		return nil, fmt.Errorf("nota requires related documents")
	}

	// Load data using YOUR EXACT methods
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

	// Build line items
	items := make([]CuerpoDocumentoNota, len(nota.LineItems))
	amounts := make([]ItemAmounts, len(nota.LineItems))

	for i, line := range nota.LineItems {
		// Use CCF calculator (nota uses same logic as CCF)
		amount := b.calculator.CalculateCreditoFiscal(
			line.UnitPrice,
			line.Quantity,
			line.DiscountAmount,
		)
		amounts[i] = amount

		ref := ""
		if line.RelatedDocumentRef != nil {
			ref = *line.RelatedDocumentRef
		}

		var tributos []string
		if amount.VentaGravada > 0 {
			tributos = []string{"20"}
		}

		items[i] = CuerpoDocumentoNota{
			NumItem:         line.LineNumber,
			TipoItem:        line.ItemType,
			NumeroDocumento: ref,
			Cantidad:        line.Quantity,
			Codigo:          &line.ItemSku,
			CodTributo:      nil,
			UniMedida:       line.UnitOfMeasure,
			Descripcion:     line.ItemName,
			PrecioUni:       amount.PrecioUni,
			MontoDescu:      line.DiscountAmount,
			VentaNoSuj:      0,
			VentaExenta:     0,
			VentaGravada:    amount.VentaGravada,
			Tributos:        tributos,
		}
	}

	// Build related documents
	relDocs := make([]DocumentoRelacionado, len(nota.RelatedDocuments))
	for i, doc := range nota.RelatedDocuments {
		relDocs[i] = DocumentoRelacionado{
			TipoDocumento:   doc.DocumentType,
			TipoGeneracion:  doc.GenerationType,
			NumeroDocumento: doc.DocumentNumber,
			FechaEmision:    doc.DocumentDate.Format("2006-01-02"),
		}
	}

	// Build resumen using YOUR calculator
	resumenAmounts := b.calculator.CalculateResumenCCF(amounts)

	// Build identificacion
	loc, _ := time.LoadLocation("America/El_Salvador")
	var emissionTime time.Time
	if nota.FinalizedAt != nil {
		emissionTime = nota.FinalizedAt.In(loc)
	} else {
		emissionTime = time.Now().In(loc)
	}

	// Build emisor using YOUR method
	emisor := b.buildEmisor(company, establishment)

	// Build receptor using YOUR method - nota uses CCF type
	receptor := b.buildReceptor(codigos.PersonTypeJuridica, client)

	nd := &NotaDebitoElectronica{
		Identificacion: Identificacion{
			Version:          3,
			Ambiente:         company.DTEAmbiente,
			TipoDte:          "06",
			NumeroControl:    strings.ToUpper(*nota.DteNumeroControl),
			CodigoGeneracion: nota.ID,
			TipoModelo:       1,
			TipoOperacion:    1,
			TipoContingencia: nil,
			MotivoContin:     nil,
			FecEmi:           emissionTime.Format("2006-01-02"),
			HorEmi:           emissionTime.Format("15:04:05"),
			TipoMoneda:       "USD",
		},
		DocumentoRelacionado: relDocs,
		Emisor:               emisor,
		Receptor:             *receptor, // Dereference pointer
		VentaTercero:         nil,
		CuerpoDocumento:      items,
		Resumen: Resumen{
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
			Tributos: &[]Tributo{
				{
					Codigo:      "20",
					Descripcion: "Impuesto al Valor Agregado 13%",
					Valor:       resumenAmounts.TotalIva,
				},
			},
		},
		Extension: &Extension{
			NombEntrega:   nil,
			DocuEntrega:   nil,
			NombRecibe:    nil,
			DocuRecibe:    nil,
			Observaciones: nota.Notes,
			PlacaVehiculo: nil,
		},
		Apendice: nil,
	}

	return nd, nil
}
