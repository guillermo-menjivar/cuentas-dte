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

func (s *DTEService) ProcessExportInvoice(ctx context.Context, invoice *models.Invoice) (*hacienda.ReceptionResponse, error) {
	// Step 1: Build DTE from export invoice
	fmt.Printf("\n🔄 Processing Export Invoice (Type 11): %s\n", invoice.InvoiceNumber)
	fmt.Println("Step 1: Building export DTE...")

	dteJSON, err := s.builder.BuildFacturaExportacion(ctx, invoice)
	if err != nil {
		return nil, fmt.Errorf("failed to build export DTE: %w", err)
	}

	// Pretty print for debugging
	var dtePretty interface{}
	json.Unmarshal(dteJSON, &dtePretty)
	prettyJSON, _ := json.MarshalIndent(dtePretty, "", "  ")
	fmt.Println("Export DTE Generated:")
	fmt.Println(string(prettyJSON))

	// Parse identificacion from JSON
	var exportDTE struct {
		Identificacion struct {
			TipoDte          string `json:"tipoDte"`
			Ambiente         string `json:"ambiente"`
			NumeroControl    string `json:"numeroControl"`
			CodigoGeneracion string `json:"codigoGeneracion"`
		} `json:"identificacion"`
	}
	if err := json.Unmarshal(dteJSON, &exportDTE); err != nil {
		return nil, fmt.Errorf("failed to parse export DTE: %w", err)
	}

	// Step 2: Load credentials and sign
	fmt.Println("\nStep 2: Loading credentials and signing export DTE...")
	companyID, err := uuid.Parse(invoice.CompanyID)
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
		return nil, fmt.Errorf("failed to sign export DTE: %w", err)
	}

	fmt.Println("\nStep 3: Export DTE Signed Successfully!")
	fmt.Printf("Signed DTE length: %d characters\n", len(signedDTE))

	// Step 4: Authenticate with Hacienda
	fmt.Println("\nStep 4: Authenticating with Hacienda...")
	authResponse, err := s.haciendaService.AuthenticateCompany(ctx, companyID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to authenticate with Hacienda: %w", err)
	}

	fmt.Printf("✅ Authenticated! Token: %s...\n", authResponse.Body.Token[:50])

	// Step 5: Submit to Hacienda
	fmt.Println("\nStep 5: Submitting Export Invoice (Type 11) to Ministerio de Hacienda...")
	fmt.Printf("📋 Código: %s | Ambiente: %s\n",
		strings.ToUpper(exportDTE.Identificacion.CodigoGeneracion),
		exportDTE.Identificacion.Ambiente)

	response, err := s.hacienda.SubmitDTE(
		ctx,
		authResponse.Body.Token,
		exportDTE.Identificacion.Ambiente,
		"11", // Type 11 - Factura de Exportación
		strings.ToUpper(exportDTE.Identificacion.CodigoGeneracion),
		signedDTE,
	)

	if err != nil {
		// Check if it's a rejection
		if hacErr, ok := err.(*hacienda.HaciendaError); ok && hacErr.Type == "rejection" {
			fmt.Printf("\n❌ Export Invoice REJECTED by Hacienda!\n")
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
	fmt.Println("\n✅ SUCCESS! EXPORT INVOICE ACCEPTED BY HACIENDA!")
	fmt.Printf("Estado: %s\n", response.Estado)
	fmt.Printf("Código de Generación: %s\n", response.CodigoGeneracion)
	fmt.Printf("Sello Recibido: %s\n", response.SelloRecibido)
	fmt.Printf("Fecha Procesamiento: %s\n", response.FhProcesamiento)

	// Step 7: Save response
	if response.Estado == "PROCESADO" {
		err = s.saveHaciendaResponse(ctx, invoice.ID, response)
		if err != nil {
			fmt.Printf("⚠️  Warning: failed to save Hacienda response: %v\n", err)
		} else {
			fmt.Println("✅ Hacienda response saved to invoice")
		}
	}

	// Step 8: Log to commit log
	err = s.logExportToCommitLog(
		ctx,
		invoice,
		exportDTE.Identificacion.TipoDte,
		exportDTE.Identificacion.Ambiente,
		exportDTE.Identificacion.NumeroControl,
		strings.ToUpper(exportDTE.Identificacion.CodigoGeneracion),
		dteJSON,
		signedDTE,
		response,
	)
	if err != nil {
		fmt.Printf("⚠️  Warning: failed to log to commit log: %v\n", err)
	} else {
		fmt.Println("✅ Export invoice submission logged to commit log")
	}

	return response, nil
}

// logExportToCommitLog creates an immutable audit record for export invoice (Type 11)
func (s *DTEService) logExportToCommitLog(
	ctx context.Context,
	invoice *models.Invoice,
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
	if invoice.FinalizedAt != nil {
		fechaEmision = *invoice.FinalizedAt
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

	// For Type 11: IVA is 0%, so iva_amount = 0
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

	_, err = s.db.ExecContext(ctx, query,
		codigoGeneracion,
		invoice.ID,
		invoice.InvoiceNumber,
		invoice.CompanyID,
		invoice.ClientID,
		invoice.EstablishmentID,
		invoice.PointOfSaleID,
		invoice.Subtotal,
		invoice.TotalDiscount,
		invoice.TotalTaxes,
		ivaAmount, // Always 0 for Type 11
		invoice.Total,
		invoice.Currency,
		invoice.PaymentMethod,
		invoice.PaymentTerms,
		invoice.ReferencesInvoiceID,
		numeroControl,
		tipoDte, // "11"
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
		invoice.CreatedBy,
	)

	return err
}
