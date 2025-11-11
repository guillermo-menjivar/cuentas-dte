package dte

import (
	"context"
	"cuentas/internal/hacienda"
	"cuentas/internal/models"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// ProcessRemision processes a remision (Type 04) - builds, signs, and submits to Hacienda
// Add this method to internal/dte/service.go in the DTEService struct

func (s *DTEService) ProcessRemision(ctx context.Context, remision *models.Invoice) (*hacienda.ReceptionResponse, error) {
	// Step 1: Build DTE from remision
	fmt.Printf("\n🔄 Processing Remision (Type 04): %s\n", remision.InvoiceNumber)
	fmt.Println("Step 1: Building remision DTE...")

	dteJSON, err := s.builder.BuildNotaRemision(ctx, remision)
	if err != nil {
		return nil, fmt.Errorf("failed to build remision DTE: %w", err)
	}

	// Pretty print for debugging
	var dtePretty interface{}
	json.Unmarshal(dteJSON, &dtePretty)
	prettyJSON, _ := json.MarshalIndent(dtePretty, "", "  ")
	fmt.Println("Remision DTE Generated:")
	fmt.Println(string(prettyJSON))

	// ✅ CRITICAL: Schema validation happens in BuildNotaRemision
	// This ensures the JSON is valid BEFORE we attempt to sign
	fmt.Println("\n✅ Schema validation already passed in builder")

	// Parse identificacion from JSON
	var remisionDTE struct {
		Identificacion struct {
			TipoDte          string `json:"tipoDte"`
			Ambiente         string `json:"ambiente"`
			NumeroControl    string `json:"numeroControl"`
			CodigoGeneracion string `json:"codigoGeneracion"`
		} `json:"identificacion"`
	}
	if err := json.Unmarshal(dteJSON, &remisionDTE); err != nil {
		return nil, fmt.Errorf("failed to parse remision DTE: %w", err)
	}

	// Step 2: Load credentials and sign
	fmt.Println("\nStep 2: Loading credentials and signing remision DTE...")
	companyID, err := uuid.Parse(remision.CompanyID)
	if err != nil {
		return nil, fmt.Errorf("invalid company ID: %w", err)
	}

	creds, err := s.LoadCredentials(ctx, companyID)
	if err != nil {
		return nil, fmt.Errorf("failed to load credentials: %w", err)
	}

	// Unmarshal for signing
	var dteForSigning interface{}
	json.Unmarshal(dteJSON, &dteForSigning)

	signedDTE, err := s.firmador.Sign(ctx, creds.NIT, creds.Password, dteForSigning)
	if err != nil {
		return nil, fmt.Errorf("failed to sign remision DTE: %w", err)
	}

	fmt.Println("\nStep 3: Remision DTE Signed Successfully!")
	fmt.Printf("Signed DTE length: %d characters\n", len(signedDTE))

	// Step 4: Authenticate with Hacienda
	fmt.Println("\nStep 4: Authenticating with Hacienda...")
	authResponse, err := s.haciendaService.AuthenticateCompany(ctx, companyID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to authenticate with Hacienda: %w", err)
	}

	fmt.Printf("✅ Authenticated! Token: %s...\n", authResponse.Body.Token[:50])

	// Step 5: Submit to Hacienda
	fmt.Println("\nStep 5: Submitting Remision (Type 04) to Ministerio de Hacienda...")
	fmt.Printf("📋 Código: %s | Ambiente: %s\n",
		strings.ToUpper(remisionDTE.Identificacion.CodigoGeneracion),
		remisionDTE.Identificacion.Ambiente)

	response, err := s.hacienda.SubmitDTE(
		ctx,
		authResponse.Body.Token,
		remisionDTE.Identificacion.Ambiente,
		"04", // Type 04 - Nota de Remisión
		strings.ToUpper(remisionDTE.Identificacion.CodigoGeneracion),
		signedDTE,
	)

	if err != nil {
		// Check if it's a rejection
		if hacErr, ok := err.(*hacienda.HaciendaError); ok && hacErr.Type == "rejection" {
			fmt.Printf("\n❌ Remision REJECTED by Hacienda!\n")
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
	fmt.Println("\n✅ SUCCESS! REMISION ACCEPTED BY HACIENDA!")
	fmt.Printf("Estado: %s\n", response.Estado)
	fmt.Printf("Código de Generación: %s\n", response.CodigoGeneracion)
	fmt.Printf("Sello Recibido: %s\n", response.SelloRecibido)
	fmt.Printf("Fecha Procesamiento: %s\n", response.FhProcesamiento)

	// Step 7: Save response
	if response.Estado == "PROCESADO" {
		err = s.saveHaciendaResponse(ctx, remision.ID, response)
		if err != nil {
			fmt.Printf("⚠️  Warning: failed to save Hacienda response: %v\n", err)
		} else {
			fmt.Println("✅ Hacienda response saved to remision")
		}
	}

	// Step 8: Log to commit log
	err = s.logRemisionToCommitLog(
		ctx,
		remision,
		remisionDTE.Identificacion.TipoDte,
		remisionDTE.Identificacion.Ambiente,
		remisionDTE.Identificacion.NumeroControl,
		strings.ToUpper(remisionDTE.Identificacion.CodigoGeneracion),
		dteJSON,
		signedDTE,
		response,
	)
	if err != nil {
		fmt.Printf("⚠️  Warning: failed to log to commit log: %v\n", err)
	} else {
		fmt.Println("✅ Remision submission logged to commit log")
	}

	return response, nil
}

// logRemisionToCommitLog creates an immutable audit record for remision (Type 04)
func (s *DTEService) logRemisionToCommitLog(
	ctx context.Context,
	remision *models.Invoice,
	tipoDte string,
	ambiente string,
	numeroControl string,
	codigoGeneracion string,
	dteUnsigned []byte,
	signedDTE string,
	response *hacienda.ReceptionResponse,
) error {
	// Marshal full Hacienda response
	haciendaResponseFull, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("failed to marshal Hacienda response: %w", err)
	}

	// Parse processing date
	var fechaProcesamiento *time.Time
	if response != nil && response.FhProcesamiento != "" {
		t, err := time.Parse("02/01/2006 15:04:05", response.FhProcesamiento)
		if err == nil {
			fechaProcesamiento = &t
		}
	}

	// Use finalized_at as fecha_emision
	var fechaEmision time.Time
	if remision.FinalizedAt != nil {
		fechaEmision = *remision.FinalizedAt
	} else {
		fechaEmision = time.Now()
	}

	// Extract fiscal period
	fiscalYear := fechaEmision.Year()
	fiscalMonth := int(fechaEmision.Month())

	// Generate DTE URL
	dteURL := fmt.Sprintf(
		"https://admin.factura.gob.sv/consultaPublica?ambiente=%s&codGen=%s&fechaEmi=%s",
		ambiente,
		codigoGeneracion,
		fechaEmision.Format("2006-01-02"),
	)

	// For Type 04 (remision): amounts are typically 0
	ivaAmount := 0.0

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
            hacienda_response_full, created_by
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26, $27, $28, $29, $30, $31, $32, $33)
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

	// Handle NULL client_id for internal transfers
	var clientID *string
	if remision.ClientID != "" {
		clientID = &remision.ClientID
	}

	_, err = s.db.ExecContext(ctx, query,
		codigoGeneracion,
		remision.ID,
		remision.InvoiceNumber,
		remision.CompanyID,
		clientID, // Can be NULL for internal transfers
		remision.EstablishmentID,
		remision.PointOfSaleID,
		remision.Subtotal,      // Typically 0
		remision.TotalDiscount, // Typically 0
		remision.TotalTaxes,    // Typically 0
		ivaAmount,              // Always 0 for Type 04
		remision.Total,         // Typically 0
		remision.Currency,
		remision.PaymentMethod, // NULL for remision
		remision.PaymentTerms,  // NULL for remision
		remision.ReferencesInvoiceID,
		numeroControl,
		tipoDte, // "04"
		ambiente,
		fechaEmision,
		fiscalYear,
		fiscalMonth,
		dteURL,
		dteUnsigned,
		signedDTE,
		estado,
		selloRecibido,
		fechaProcesamiento,
		codigoMsg,
		descripcionMsg,
		pq.Array(observaciones),
		haciendaResponseFull,
		remision.CreatedBy,
	)

	return err
}
