package dte

// Add this to internal/dte/service.go

import (
	"context"
	"cuentas/internal/codigos"
	"cuentas/internal/hacienda"
	"cuentas/internal/models"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// ProcessNotaDebito processes a Nota de Débito through the complete DTE flow
func (s *DTEService) ProcessNotaDebito(ctx context.Context, nota *models.NotaDebito) (*hacienda.ReceptionResponse, error) {
	// Step 1: Build DTE from nota
	fmt.Printf("\n🔄 Processing Nota de Débito: %s\n", nota.NotaNumber)
	fmt.Println("Step 1: Building DTE from nota de débito...")

	dteJSON, err := s.builder.BuildNotaDebito(ctx, nota)
	if err != nil {
		return nil, fmt.Errorf("failed to build DTE: %w", err)
	}

	// Pretty print for debugging
	var dtePretty interface{}
	json.Unmarshal(dteJSON, &dtePretty)
	prettyJSON, _ := json.MarshalIndent(dtePretty, "", "  ")
	fmt.Println("DTE Generated:")
	fmt.Println(string(prettyJSON))

	// Step 2: Load credentials and sign
	fmt.Println("\nStep 2: Loading credentials and signing DTE...")
	companyID, err := uuid.Parse(nota.CompanyID)
	if err != nil {
		return nil, fmt.Errorf("invalid company ID: %w", err)
	}

	creds, err := s.LoadCredentials(ctx, companyID)
	if err != nil {
		return nil, fmt.Errorf("failed to load credentials: %w", err)
	}

	// Unmarshal to get codigo generacion for signing
	var dteForSigning interface{}
	json.Unmarshal(dteJSON, &dteForSigning)

	signedDTE, err := s.firmador.Sign(ctx, creds.NIT, creds.Password, dteForSigning)
	if err != nil {
		return nil, fmt.Errorf("failed to sign DTE: %w", err)
	}

	fmt.Println("\nStep 3: DTE Signed Successfully!")
	fmt.Printf("Signed DTE length: %d characters\n", len(signedDTE))

	// Step 4: Authenticate with Hacienda
	fmt.Println("\nStep 4: Authenticating with Hacienda...")
	authResponse, err := s.haciendaService.AuthenticateCompany(ctx, nota.CompanyID)
	if err != nil {
		return nil, fmt.Errorf("failed to authenticate with Hacienda: %w", err)
	}

	fmt.Printf("✅ Authenticated! Token: %s...\n", authResponse.Body.Token[:50])

	// Step 5: Submit to Hacienda
	fmt.Println("\nStep 5: Submitting Nota de Débito to Ministerio de Hacienda...")

	response, err := s.hacienda.SubmitDTE(
		ctx,
		authResponse.Body.Token,
		"00", // ambiente - get from company
		"06", // tipo DTE - Nota de Débito
		strings.ToUpper(nota.ID),
		signedDTE,
	)

	if err != nil {
		// Check if it's a rejection
		if hacErr, ok := err.(*hacienda.HaciendaError); ok && hacErr.Type == "rejection" {
			fmt.Printf("\n❌ Nota de Débito REJECTED by Hacienda!\n")
			if response != nil {
				fmt.Printf("Code: %s\n", response.CodigoMsg)
				fmt.Printf("Message: %s\n", response.DescripcionMsg)
				if len(response.Observaciones) > 0 {
					fmt.Println("Observations:")
					for _, obs := range response.Observaciones {
						fmt.Printf("  - %s\n", obs)
					}
				}
			}
			return response, err
		}
		return nil, fmt.Errorf("failed to submit to Hacienda: %w", err)
	}

	if response == nil {
		return nil, fmt.Errorf("no response received from Hacienda")
	}

	// Step 6: Success!
	fmt.Println("\n✅ SUCCESS! NOTA DE DÉBITO ACCEPTED BY HACIENDA!")
	fmt.Printf("Estado: %s\n", response.Estado)
	fmt.Printf("Código de Generación: %s\n", response.CodigoGeneracion)
	fmt.Printf("Sello Recibido: %s\n", response.SelloRecibido)
	fmt.Printf("Fecha Procesamiento: %s\n", response.FhProcesamiento)

	// Step 7: Save response to database
	if response.Estado == "PROCESADO" {
		err = s.saveNotaHaciendaResponse(ctx, nota.ID, response)
		if err != nil {
			fmt.Printf("⚠️  Warning: failed to save Hacienda response: %v\n", err)
		} else {
			fmt.Println("✅ Hacienda response saved to nota")
		}
		UploadDTEToS3Async(dteJSON, "unsigned", codigos.DocTypeNotaDebito, nota.CompanyID, strings.ToUpper(nota.ID))
		UploadDTEToS3Async([]byte(signedDTE), "signed", codigos.DocTypeNotaDebito, nota.CompanyID, strings.ToUpper(nota.ID))
		haciendaResponseJSON, _ := json.MarshalIndent(response, "", "  ")
		UploadDTEToS3Async(haciendaResponseJSON, "hacienda_response", codigos.DocTypeNotaDebito, nota.CompanyID, strings.ToUpper(nota.ID))
	}

	// Step 8: Log to commit log
	err = s.logNotaToCommitLog(
		ctx,
		nota,
		codigos.DocTypeNotaDebito, // tipoDte string
		codigos.MODE_PRUEBA,       // ambiente string
		dteJSON,                   // dteUnsigned []byte
		signedDTE,                 // dteSigned string
		response,
	)
	if err != nil {
		fmt.Printf("⚠️  Warning: failed to log to commit log: %v\n", err)
	} else {
		fmt.Println("✅ Nota submission logged to commit log")
	}

	return response, nil
}

// saveNotaHaciendaResponse saves Hacienda's response to the nota
func (s *DTEService) saveNotaHaciendaResponse(ctx context.Context, notaID string, response *hacienda.ReceptionResponse) error {
	query := `
		UPDATE notas_debito
		SET 
			dte_codigo_generacion = $1,
			dte_sello_recibido = $2,
			dte_status = $3,
			dte_hacienda_response = $4,
			dte_submitted_at = $5
		WHERE id = $6
	`

	// Parse processing date
	var fechaProcesamiento *time.Time
	if response.FhProcesamiento != "" {
		t, err := time.Parse("02/01/2006 15:04:05", response.FhProcesamiento)
		if err == nil {
			fechaProcesamiento = &t
		}
	}

	responseJSON, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("failed to marshal response: %w", err)
	}

	_, err = s.db.ExecContext(ctx, query,
		response.CodigoGeneracion,
		response.SelloRecibido,
		response.Estado,
		responseJSON,
		fechaProcesamiento,
		notaID,
	)

	return err
}

////

func (s *DTEService) logNotaToCommitLog(
	ctx context.Context,
	nota *models.NotaDebito,
	tipoDte string,
	ambiente string,
	dteUnsigned []byte,
	dteSigned string,
	response *hacienda.ReceptionResponse,
) error {
	// Parse processing date
	var fechaProcesamiento *time.Time
	if response != nil && response.FhProcesamiento != "" {
		t, err := time.Parse("02/01/2006 15:04:05", response.FhProcesamiento)
		if err == nil {
			fechaProcesamiento = &t
		}
	}

	// Parse fecha emision
	var fechaEmision time.Time
	if nota.FinalizedAt != nil {
		fechaEmision = *nota.FinalizedAt
	} else {
		fechaEmision = time.Now()
	}

	fiscalYear := fechaEmision.Year()
	fiscalMonth := int(fechaEmision.Month())

	// Build DTE URL
	dteURL := fmt.Sprintf(
		"https://admin.factura.gob.sv/consultaPublica?ambiente=%s&codGen=%s&fechaEmi=%s",
		ambiente,
		strings.ToUpper(nota.ID), // ← UPPERCASE
		fechaEmision.Format("2006-01-02"),
	)

	// Marshal full response
	haciendaResponseFull, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("failed to marshal Hacienda response: %w", err)
	}

	// ← ADDED: invoice_number, client_id, references_invoice_id, submitted_at
	query := `
		INSERT INTO dte_commit_log (
			codigo_generacion, invoice_id, invoice_number, company_id, client_id,
			establishment_id, point_of_sale_id,
			subtotal, total_discount, total_taxes, iva_amount, total_amount, currency,
			payment_method, payment_terms, references_invoice_id,
			numero_control, tipo_dte, ambiente, fecha_emision,
			fiscal_year, fiscal_month, dte_url,
			dte_unsigned, dte_signed,
			hacienda_estado, hacienda_sello_recibido, hacienda_fh_procesamiento,
			hacienda_codigo_msg, hacienda_descripcion_msg, hacienda_observaciones,
			hacienda_response_full, created_by, submitted_at, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16,
			$17, $18, $19, $20, $21, $22, $23, $24, $25, $26, $27, $28, $29, $30,
			$31, $32, $33, $34, $35
		)
	`

	var estado, selloRecibido, codigoMsg, descripcionMsg *string
	var observaciones []string

	if response != nil {
		estado = &response.Estado
		if response.SelloRecibido != "" {
			selloRecibido = &response.SelloRecibido
		}
		if response.CodigoMsg != "" {
			codigoMsg = &response.CodigoMsg
		}
		if response.DescripcionMsg != "" {
			descripcionMsg = &response.DescripcionMsg
		}
		observaciones = response.Observaciones
	}

	now := time.Now()

	// ← CHANGED: Loop through each CCF and create separate entry
	for _, ccfRef := range nota.CCFReferences {
		_, err = s.db.ExecContext(ctx, query,
			strings.ToUpper(nota.ID), // $1 - codigo_generacion (UPPERCASE)
			ccfRef.CCFId,             // $2 - invoice_id (the CCF being referenced)
			nota.NotaNumber,          // $3 - invoice_number
			nota.CompanyID,           // $4 - company_id
			nota.ClientID,            // $5 - client_id
			nota.EstablishmentID,     // $6 - establishment_id
			nota.PointOfSaleID,       // $7 - point_of_sale_id
			nota.Subtotal,            // $8 - subtotal
			nota.TotalDiscount,       // $9 - total_discount
			nota.TotalTaxes,          // $10 - total_taxes
			nota.TotalTaxes,          // $11 - iva_amount
			nota.Total,               // $12 - total_amount
			nota.Currency,            // $13 - currency
			nota.PaymentMethod,       // $14 - payment_method
			nota.PaymentTerms,        // $15 - payment_terms
			ccfRef.CCFId,             // $16 - references_invoice_id (the CCF)
			nota.DteNumeroControl,    // $17 - numero_control
			tipoDte,                  // $18 - tipo_dte
			ambiente,                 // $19 - ambiente
			fechaEmision,             // $20 - fecha_emision
			fiscalYear,               // $21 - fiscal_year
			fiscalMonth,              // $22 - fiscal_month
			dteURL,                   // $23 - dte_url
			dteUnsigned,              // $24 - dte_unsigned
			dteSigned,                // $25 - dte_signed
			estado,                   // $26 - hacienda_estado
			selloRecibido,            // $27 - hacienda_sello_recibido
			fechaProcesamiento,       // $28 - hacienda_fh_procesamiento
			codigoMsg,                // $29 - hacienda_codigo_msg
			descripcionMsg,           // $30 - hacienda_descripcion_msg
			pq.Array(observaciones),  // $31 - hacienda_observaciones
			haciendaResponseFull,     // $32 - hacienda_response_full
			nota.CreatedBy,           // $33 - created_by
			now,                      // $34 - submitted_at
			now,                      // $35 - created_at
		)

		if err != nil {
			return fmt.Errorf("failed to log CCF %s: %w", ccfRef.CCFId, err)
		}
	}

	return nil
}
