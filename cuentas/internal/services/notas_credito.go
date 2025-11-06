package dte

import (
	"context"
	"cuentas/internal/codigos"
	"cuentas/internal/hacienda"
	"cuentas/internal/models"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/lib/pq"
)

// ProcessNotaCredito handles complete DTE flow for nota de crédito
func (s *DTEService) ProcessNotaCredito(
	ctx context.Context,
	nota *models.NotaCredito,
) (*hacienda.ReceptionResponse, error) {

	// Step 1: Build DTE
	dteJSON, err := s.builder.BuildNotaCredito(ctx, nota)
	if err != nil {
		return nil, fmt.Errorf("failed to build DTE: %w", err)
	}

	// Step 2: Sign DTE with Firmador
	company, _ := s.getCompany(ctx, nota.CompanyID)
	signedDTE, err := s.firmador.SignDTE(dteJSON, company.FirmadorNit, company.FirmadorPassword)
	if err != nil {
		return nil, fmt.Errorf("failed to sign DTE: %w", err)
	}

	// Step 3: Authenticate with Hacienda
	authToken, err := s.haciendaClient.Authenticate(ctx, company.HaciendaNit, company.HaciendaPassword)
	if err != nil {
		return nil, fmt.Errorf("failed to authenticate: %w", err)
	}

	// Step 4: Submit to Hacienda
	response, err := s.haciendaClient.SubmitDTE(ctx, hacienda.SubmitDTERequest{
		Ambiente:         codigos.MODE_PRUEBA, // "00" or "01"
		IdEnvio:          1,
		Version:          3,    // ⭐ Version 3 for tipo 05
		TipoDte:          "05", // ⭐ Nota de Crédito
		Documento:        signedDTE,
		CodigoGeneracion: strings.ToUpper(nota.ID),
	}, authToken)

	if err != nil {
		return nil, fmt.Errorf("failed to submit DTE: %w", err)
	}

	// Step 5: Save Hacienda response
	if err := s.saveNotaCreditoHaciendaResponse(ctx, nota, response); err != nil {
		return nil, fmt.Errorf("failed to save response: %w", err)
	}

	// Step 6: Log to commit log (ONE ENTRY PER CCF)
	if err := s.logNotaCreditoToCommitLog(ctx, nota, codigos.DocTypeNotaCredito,
		codigos.MODE_PRUEBA, dteJSON, signedDTE, response); err != nil {
		log.Printf("⚠️  Warning: failed to log to commit log: %v", err)
	}

	return response, nil
}

func (s *DTEService) saveNotaCreditoHaciendaResponse(
	ctx context.Context,
	nota *models.NotaCredito,
	response *hacienda.ReceptionResponse,
) error {
	responseJSON, _ := json.Marshal(response)

	var fechaProcesamiento *time.Time
	if response.FhProcesamiento != "" {
		t, err := time.Parse("02/01/2006 15:04:05", response.FhProcesamiento)
		if err == nil {
			fechaProcesamiento = &t
		}
	}

	_, err := s.db.ExecContext(ctx, `
        UPDATE notas_credito
        SET dte_codigo_generacion = $1,
            dte_sello_recibido = $2,
            dte_status = $3,
            dte_hacienda_response = $4,
            dte_submitted_at = $5
        WHERE id = $6
    `,
		strings.ToUpper(nota.ID),
		response.SelloRecibido,
		response.Estado,
		responseJSON,
		fechaProcesamiento,
		nota.ID,
	)

	return err
}

func (s *DTEService) logNotaCreditoToCommitLog(
	ctx context.Context,
	nota *models.NotaCredito,
	tipoDte string,
	ambiente string,
	dteUnsigned []byte,
	dteSigned string,
	response *hacienda.ReceptionResponse,
) error {
	// ⭐ IDENTICAL to logNotaToCommitLog from nota débito!
	// Parse dates, build DTE URL, marshal response... (same code)

	// Create ONE commit log entry PER CCF
	for _, ccfRef := range nota.CCFReferences {
		_, err = s.db.ExecContext(ctx, query,
			strings.ToUpper(nota.ID), // codigo_generacion (SAME for all CCFs)
			ccfRef.CCFId,             // invoice_id (DIFFERENT for each CCF)
			nota.NotaNumber,          // invoice_number
			nota.CompanyID,           // company_id
			nota.ClientID,            // client_id
			nota.EstablishmentID,     // establishment_id
			nota.PointOfSaleID,       // point_of_sale_id
			nota.Subtotal,            // subtotal
			nota.TotalDiscount,       // total_discount
			nota.TotalTaxes,          // total_taxes
			nota.TotalTaxes,          // iva_amount
			nota.Total,               // total_amount
			nota.Currency,            // currency
			nota.PaymentMethod,       // payment_method
			nota.PaymentTerms,        // payment_terms
			ccfRef.CCFId,             // references_invoice_id
			nota.DteNumeroControl,    // numero_control
			tipoDte,                  // tipo_dte ("05")
			ambiente,                 // ambiente
			fechaEmision,             // fecha_emision
			fiscalYear,               // fiscal_year
			fiscalMonth,              // fiscal_month
			dteURL,                   // dte_url
			dteUnsigned,              // dte_unsigned
			dteSigned,                // dte_signed
			estado,                   // hacienda_estado
			selloRecibido,            // hacienda_sello_recibido
			fechaProcesamiento,       // hacienda_fh_procesamiento
			codigoMsg,                // hacienda_codigo_msg
			descripcionMsg,           // hacienda_descripcion_msg
			pq.Array(observaciones),  // hacienda_observaciones
			haciendaResponseFull,     // hacienda_response_full
			nota.CreatedBy,           // created_by
			now,                      // submitted_at
			now,                      // created_at
		)

		if err != nil {
			return fmt.Errorf("failed to log CCF %s: %w", ccfRef.CCFId, err)
		}
	}

	return nil
}
