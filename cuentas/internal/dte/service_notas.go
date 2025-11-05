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
		nota.ID,
		fechaEmision.Format("2006-01-02"),
	)

	// Marshal full response
	haciendaResponseFull, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("failed to marshal Hacienda response: %w", err)
	}

	query := `
		INSERT INTO dte_commit_log (
			codigo_generacion, invoice_id, invoice_number, company_id, 
			establishment_id, point_of_sale_id,
			subtotal, total_discount, total_taxes, iva_amount, total_amount, currency,
			payment_method, payment_terms,
			numero_control, tipo_dte, ambiente, fecha_emision,
			fiscal_year, fiscal_month, dte_url,
			dte_unsigned, dte_signed,
			hacienda_estado, hacienda_sello_recibido, hacienda_fh_procesamiento,
			hacienda_codigo_msg, hacienda_descripcion_msg, hacienda_observaciones,
			hacienda_response_full, created_by, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19,
			$20, $21, $22, $23, $24, $25, $26, $27, $28, $29, $30, $31
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

	_, err = s.db.ExecContext(ctx, query,
		nota.ID, // $1 - codigo_generacion
		nota.ID, // $2 - invoice_id
		nota.NotaNumber,
		nota.CompanyID,          // $3 - company_id
		nota.EstablishmentID,    // $4 - establishment_id
		nota.PointOfSaleID,      // $5 - point_of_sale_id
		nota.Subtotal,           // $6 - subtotal
		nota.TotalDiscount,      // $7 - total_discount
		nota.TotalTaxes,         // $8 - total_taxes
		nota.TotalTaxes,         // $9 - iva_amount
		nota.Total,              // $10 - total_amount
		nota.Currency,           // $11 - currency
		nota.PaymentMethod,      // $12 - payment_method
		nota.PaymentTerms,       // $13 - payment_terms
		nota.DteNumeroControl,   // $14 - numero_control
		tipoDte,                 // $15 - tipo_dte
		ambiente,                // $16 - ambiente
		fechaEmision,            // $17 - fecha_emision
		fiscalYear,              // $18 - fiscal_year
		fiscalMonth,             // $19 - fiscal_month
		dteURL,                  // $20 - dte_url
		dteUnsigned,             // $21 - dte_unsigned
		dteSigned,               // $22 - dte_signed
		estado,                  // $23 - hacienda_estado
		selloRecibido,           // $24 - hacienda_sello_recibido
		fechaProcesamiento,      // $25 - hacienda_fh_procesamiento
		codigoMsg,               // $26 - hacienda_codigo_msg
		descripcionMsg,          // $27 - hacienda_descripcion_msg
		pq.Array(observaciones), // $28 - hacienda_observaciones
		haciendaResponseFull,    // $29 - hacienda_response_full
		nota.CreatedBy,          // $30 - created_by
		time.Now(),              // $31 - created_at
	)

	return err
}
