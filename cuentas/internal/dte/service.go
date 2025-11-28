package dte

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"cuentas/internal/codigos"
	"cuentas/internal/hacienda"
	"cuentas/internal/models"
	"cuentas/internal/services"
	"cuentas/internal/services/firmador"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/redis/go-redis/v9"
)

// DTEService handles DTE signing and submission
type DTEService struct {
	db                        *sql.DB
	redis                     *redis.Client
	firmador                  *firmador.Client
	vault                     *services.VaultService
	credCache                 *CredentialCache
	hacienda                  *hacienda.Client
	haciendaService           *services.HaciendaService
	builder                   *Builder
	contingencyHelper         *ContingencyHelper
	contingencyHelperPurchase *ContingencyHelperPurchase
}

// NewDTEService creates a new DTE service (singleton)
func NewDTEService(
	db *sql.DB,
	redis *redis.Client,
	firmador *firmador.Client,
	vault *services.VaultService,
	haciendaClient *hacienda.Client,
	haciendaService *services.HaciendaService,
	contingencyService *services.ContingencyService,
) *DTEService {
	return &DTEService{
		db:                        db,
		hacienda:                  haciendaClient,
		redis:                     redis,
		firmador:                  firmador,
		vault:                     vault,
		credCache:                 NewCredentialCache(redis),
		builder:                   NewBuilder(db),
		haciendaService:           haciendaService,
		contingencyHelper:         NewContingencyHelper(contingencyService),
		contingencyHelperPurchase: NewContingencyHelperPurchase(contingencyService),
	}
}

func (s *DTEService) ProcessInvoice(ctx context.Context, invoice *models.Invoice) (*hacienda.ReceptionResponse, error) {
	// Step 1: Build DTE from invoice
	fmt.Println("Step 1: Building DTE from invoice...")
	factura, err := s.builder.BuildFromInvoice(ctx, invoice)
	if err != nil {
		return nil, fmt.Errorf("failed to build DTE: %w", err)
	}

	// Pretty print the DTE for debugging
	dteJSON, err := json.MarshalIndent(factura, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal DTE: %w", err)
	}
	fmt.Println("DTE Generated:")
	fmt.Println(string(dteJSON))

	// Step 2: Load company credentials and sign
	fmt.Println("\nStep 2: Loading credentials and signing DTE...")
	companyID, err := uuid.Parse(invoice.CompanyID)
	if err != nil {
		return nil, fmt.Errorf("invalid company ID: %w", err)
	}

	creds, err := s.LoadCredentials(ctx, companyID)
	if err != nil {
		return nil, fmt.Errorf("failed to load credentials: %w", err)
	}
	factura.Identificacion.CodigoGeneracion = strings.ToUpper(factura.Identificacion.CodigoGeneracion)

	// === CONTINGENCY: Handle signing failure ===
	signedDTE, err := s.firmador.Sign(ctx, creds.NIT, creds.Password, factura)
	if err != nil {
		fmt.Printf("⚠️  Firmador failed: %v\n", err)

		// Check if contingency helper is available
		if s.contingencyHelper != nil {
			if queueErr := s.contingencyHelper.HandleSigningFailure(
				ctx,
				invoice,
				factura,
				factura.Identificacion.Ambiente,
			); queueErr != nil {
				return nil, fmt.Errorf("firmador failed and contingency queue failed: %w", queueErr)
			}
			fmt.Println("📋 Invoice queued for contingency (firmador unavailable)")
			return nil, fmt.Errorf("invoice queued for contingency: firmador unavailable")
		}

		return nil, fmt.Errorf("failed to sign DTE: %w", err)
	}

	// Step 3: Print signed DTE
	fmt.Println("\nStep 3: DTE Signed Successfully!")
	fmt.Println("Signed DTE (first 500 chars):")
	if len(signedDTE) > 500 {
		fmt.Println(signedDTE[:500] + "...")
	} else {
		fmt.Println(signedDTE)
	}
	fmt.Printf("\nSigned DTE length: %d characters\n", len(signedDTE))

	// Step 4: Transmit to Hacienda 🚀
	fmt.Println("\nStep 4: Authenticating with Hacienda...")

	companyID, err = uuid.Parse(invoice.CompanyID)
	if err != nil {
		return nil, fmt.Errorf("invalid company ID: %w", err)
	}

	authResponse, err := s.haciendaService.AuthenticateCompany(ctx, companyID.String())
	if err != nil {
		fmt.Printf("⚠️  Hacienda auth failed: %v\n", err)

		// === CONTINGENCY: Handle auth failure ===
		if s.contingencyHelper != nil {
			if queueErr := s.contingencyHelper.HandleAuthFailure(
				ctx,
				invoice,
				factura,
				signedDTE,
				factura.Identificacion.Ambiente,
			); queueErr != nil {
				return nil, fmt.Errorf("auth failed and contingency queue failed: %w", queueErr)
			}
			fmt.Println("📋 Invoice queued for contingency (Hacienda auth unavailable)")
			return nil, fmt.Errorf("invoice queued for contingency: Hacienda auth unavailable")
		}

		return nil, fmt.Errorf("failed to authenticate with Hacienda: %w", err)
	}

	fmt.Printf("✅ Authenticated! Token: %s...\n", authResponse.Body.Token[:50])

	// Step 5: Submit to Hacienda (using the hacienda.Client)
	fmt.Println("\nStep 5: Submitting to Ministerio de Hacienda...")

	response, err := s.hacienda.SubmitDTE(
		ctx,
		authResponse.Body.Token,
		factura.Identificacion.Ambiente,
		factura.Identificacion.TipoDte,
		strings.ToUpper(factura.Identificacion.CodigoGeneracion),
		signedDTE,
	)

	if err != nil {
		// Check if it's a rejection error (we still got a response)
		if hacErr, ok := err.(*hacienda.HaciendaError); ok && hacErr.Type == "rejection" {
			fmt.Printf("\n❌ DTE REJECTED by Hacienda!\n")
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
			// Rejections are permanent - don't queue for contingency
			return response, err
		}

		// === CONTINGENCY: Handle submission failure (timeout, network) ===
		fmt.Printf("⚠️  Hacienda submission failed: %v\n", err)
		if s.contingencyHelper != nil {
			if queueErr := s.contingencyHelper.HandleSubmissionFailure(
				ctx,
				invoice,
				factura,
				signedDTE,
				factura.Identificacion.Ambiente,
			); queueErr != nil {
				return nil, fmt.Errorf("submission failed and contingency queue failed: %w", queueErr)
			}
			fmt.Println("📋 Invoice queued for contingency (Hacienda unavailable)")
			return nil, fmt.Errorf("invoice queued for contingency: Hacienda unavailable")
		}

		return nil, fmt.Errorf("failed to submit to Hacienda: %w", err)
	}

	// NOW check if response is nil (shouldn't be, but defensive programming)
	if response == nil {
		return nil, fmt.Errorf("no response received from Hacienda")
	}
	fmt.Println(response)

	// Step 6: Success! 🎉
	fmt.Println("\n✅ SUCCESS! DTE ACCEPTED BY HACIENDA!")
	fmt.Printf("Estado: %s\n", response.Estado)
	fmt.Printf("Código de Generación: %s\n", response.CodigoGeneracion)
	fmt.Printf("Sello Recibido: %s\n", response.SelloRecibido)
	fmt.Printf("Fecha Procesamiento: %s\n", response.FhProcesamiento)
	if response.DescripcionMsg != "" {
		fmt.Printf("Message: %s\n", response.DescripcionMsg)
	}

	if response.Estado == "PROCESADO" {
		err = s.saveHaciendaResponse(ctx, invoice.ID, response)
		if err != nil {
			// Log error but don't fail - DTE was accepted
			fmt.Printf("⚠️  Warning: failed to save Hacienda response: %v\n", err)
		} else {
			fmt.Println("✅ Hacienda response saved to invoice")
		}
		UploadDTEToS3Async(dteJSON, "unsigned", factura.Identificacion.TipoDte, invoice.CompanyID, factura.Identificacion.CodigoGeneracion)
		UploadDTEToS3Async([]byte(signedDTE), "signed", factura.Identificacion.TipoDte, invoice.CompanyID, factura.Identificacion.CodigoGeneracion)
		haciendaResponseJSON, _ := json.MarshalIndent(response, "", "  ")
		UploadDTEToS3Async(haciendaResponseJSON, "hacienda_response", factura.Identificacion.TipoDte, invoice.CompanyID, strings.ToUpper(factura.Identificacion.CodigoGeneracion))
	}
	// Step 7 submit commitlog
	err = s.logToCommitLog(ctx, invoice, factura, signedDTE, response)
	if err != nil {
		// Log error but don't fail - DTE was already accepted
		fmt.Printf("⚠️  Warning: failed to log to commit log: %v\n", err)
	} else {
		fmt.Println("✅ DTE submission logged to commit log")
	}

	return response, nil
}

func (s *DTEService) _ProcessInvoice(ctx context.Context, invoice *models.Invoice) (*hacienda.ReceptionResponse, error) {
	// Step 1: Build DTE from invoice
	fmt.Println("Step 1: Building DTE from invoice...")
	factura, err := s.builder.BuildFromInvoice(ctx, invoice)
	if err != nil {
		return nil, fmt.Errorf("failed to build DTE: %w", err)
	}

	// Pretty print the DTE for debugging
	dteJSON, err := json.MarshalIndent(factura, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal DTE: %w", err)
	}
	fmt.Println("DTE Generated:")
	fmt.Println(string(dteJSON))

	// Step 2: Load company credentials and sign
	fmt.Println("\nStep 2: Loading credentials and signing DTE...")
	companyID, err := uuid.Parse(invoice.CompanyID)
	if err != nil {
		return nil, fmt.Errorf("invalid company ID: %w", err)
	}

	creds, err := s.LoadCredentials(ctx, companyID)
	if err != nil {
		return nil, fmt.Errorf("failed to load credentials: %w", err)
	}
	factura.Identificacion.CodigoGeneracion = strings.ToUpper(factura.Identificacion.CodigoGeneracion)

	signedDTE, err := s.firmador.Sign(ctx, creds.NIT, creds.Password, factura)
	if err != nil {
		return nil, fmt.Errorf("failed to sign DTE: %w", err)
	}

	// Step 3: Print signed DTE
	fmt.Println("\nStep 3: DTE Signed Successfully!")
	fmt.Println("Signed DTE (first 500 chars):")
	if len(signedDTE) > 500 {
		fmt.Println(signedDTE[:500] + "...")
	} else {
		fmt.Println(signedDTE)
	}
	fmt.Printf("\nSigned DTE length: %d characters\n", len(signedDTE))

	// Step 4: Transmit to Hacienda 🚀
	fmt.Println("\nStep 4: Authenticating with Hacienda...")

	companyID, err = uuid.Parse(invoice.CompanyID)
	if err != nil {
		return nil, fmt.Errorf("invalid company ID: %w", err)
	}

	authResponse, err := s.haciendaService.AuthenticateCompany(ctx, companyID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to authenticate with Hacienda: %w", err)
	}

	fmt.Printf("✅ Authenticated! Token: %s...\n", authResponse.Body.Token[:50])

	// Step 5: Submit to Hacienda (using the hacienda.Client)
	fmt.Println("\nStep 5: Submitting to Ministerio de Hacienda...")

	response, err := s.hacienda.SubmitDTE(
		ctx,
		authResponse.Body.Token,
		factura.Identificacion.Ambiente,
		factura.Identificacion.TipoDte,
		strings.ToUpper(factura.Identificacion.CodigoGeneracion),
		signedDTE,
	)

	if err != nil {
		// Check if it's a rejection error (we still got a response)
		if hacErr, ok := err.(*hacienda.HaciendaError); ok && hacErr.Type == "rejection" {
			fmt.Printf("\n❌ DTE REJECTED by Hacienda!\n")
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

	// NOW check if response is nil (shouldn't be, but defensive programming)
	if response == nil {
		return nil, fmt.Errorf("no response received from Hacienda")
	}
	fmt.Println(response)

	// Step 6: Success! 🎉
	fmt.Println("\n✅ SUCCESS! DTE ACCEPTED BY HACIENDA!")
	fmt.Printf("Estado: %s\n", response.Estado)
	fmt.Printf("Código de Generación: %s\n", response.CodigoGeneracion)
	fmt.Printf("Sello Recibido: %s\n", response.SelloRecibido)
	fmt.Printf("Fecha Procesamiento: %s\n", response.FhProcesamiento)
	if response.DescripcionMsg != "" {
		fmt.Printf("Message: %s\n", response.DescripcionMsg)
	}

	if response.Estado == "PROCESADO" {
		err = s.saveHaciendaResponse(ctx, invoice.ID, response)
		if err != nil {
			// Log error but don't fail - DTE was accepted
			fmt.Printf("⚠️  Warning: failed to save Hacienda response: %v\n", err)
		} else {
			fmt.Println("✅ Hacienda response saved to invoice")
		}
		UploadDTEToS3Async(dteJSON, "unsigned", factura.Identificacion.TipoDte, invoice.CompanyID, factura.Identificacion.CodigoGeneracion)
		UploadDTEToS3Async([]byte(signedDTE), "signed", factura.Identificacion.TipoDte, invoice.CompanyID, factura.Identificacion.CodigoGeneracion)
		haciendaResponseJSON, _ := json.MarshalIndent(response, "", "  ")
		UploadDTEToS3Async(haciendaResponseJSON, "hacienda_response", factura.Identificacion.TipoDte, invoice.CompanyID, strings.ToUpper(factura.Identificacion.CodigoGeneracion))
	}
	// Step 7 submit commitlog
	err = s.logToCommitLog(ctx, invoice, factura, signedDTE, response)
	if err != nil {
		// Log error but don't fail - DTE was already accepted
		fmt.Printf("⚠️  Warning: failed to log to commit log: %v\n", err)
	} else {
		fmt.Println("✅ DTE submission logged to commit log")
	}

	return response, nil
}

// saveHaciendaResponse saves Hacienda's response to the invoice
func (s *DTEService) saveHaciendaResponse(ctx context.Context, invoiceID string, response *hacienda.ReceptionResponse) error {
	query := `
		UPDATE invoices
		SET 
			dte_sello_recibido = $1,
			dte_fecha_procesamiento = $2,
			dte_observaciones = $3,
			dte_status = $4
		WHERE id = $5
	`

	// Parse the date from Hacienda format: "14/10/2025 20:50:21"
	var fechaProcesamiento *time.Time
	if response.FhProcesamiento != "" {
		t, err := time.Parse("02/01/2006 15:04:05", response.FhProcesamiento)
		if err == nil {
			fechaProcesamiento = &t
		}
	}

	_, err := s.db.ExecContext(ctx, query,
		response.SelloRecibido,
		fechaProcesamiento,
		pq.Array(response.Observaciones),
		response.Estado,
		invoiceID,
	)

	return err
}

// SignDTE signs a DTE document for a company (existing method - keep for flexibility)
func (s *DTEService) SignDTE(ctx context.Context, companyID uuid.UUID, dteJSON interface{}) (string, error) {
	// Load credentials (with caching)
	creds, err := s.LoadCredentials(ctx, companyID)
	if err != nil {
		return "", fmt.Errorf("failed to load credentials: %w", err)
	}

	// Sign using firmador
	signedDTE, err := s.firmador.Sign(ctx, creds.NIT, creds.Password, dteJSON)
	if err != nil {
		return "", fmt.Errorf("failed to sign DTE: %w", err)
	}

	return signedDTE, nil
}

// InvalidateCredentials invalidates cached credentials for a company
func (s *DTEService) InvalidateCredentials(ctx context.Context, companyID uuid.UUID) error {
	return s.credCache.Invalidate(ctx, companyID)
}

// logToCommitLog creates an immutable audit record of the DTE submission
func (s *DTEService) logToCommitLog(ctx context.Context, invoice *models.Invoice, factura *DTE, signedDTE string, response *hacienda.ReceptionResponse) error {
	// Marshal unsigned DTE to JSON
	dteUnsigned, err := json.Marshal(factura)
	if err != nil {
		return fmt.Errorf("failed to marshal unsigned DTE: %w", err)
	}

	// Marshal full Hacienda response to JSON
	haciendaResponseFull, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("failed to marshal Hacienda response: %w", err)
	}

	// Parse the processing date
	var fechaProcesamiento *time.Time
	if response != nil && response.FhProcesamiento != "" {
		t, err := time.Parse("02/01/2006 15:04:05", response.FhProcesamiento)
		if err == nil {
			fechaProcesamiento = &t
		}
	}

	// Parse fecha_emision from factura (format: "2025-10-15")
	fechaEmision, err := time.Parse("2006-01-02", factura.Identificacion.FecEmi)
	if err != nil {
		return fmt.Errorf("failed to parse fecha emision: %w", err)
	}

	// Extract fiscal period
	fiscalYear := fechaEmision.Year()
	fiscalMonth := int(fechaEmision.Month())

	// Generate DTE URL
	dteURL := fmt.Sprintf(
		"https://admin.factura.gob.sv/consultaPublica?ambiente=%s&codGen=%s&fechaEmi=%s",
		factura.Identificacion.Ambiente,
		factura.Identificacion.CodigoGeneracion,
		factura.Identificacion.FecEmi,
	)

	// Calculate IVA amount from total taxes
	ivaAmount := invoice.TotalTaxes

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
		factura.Identificacion.CodigoGeneracion,
		invoice.ID,
		invoice.InvoiceNumber,
		invoice.CompanyID,
		invoice.ClientID,
		invoice.EstablishmentID,
		invoice.PointOfSaleID,
		invoice.Subtotal,
		invoice.TotalDiscount,
		invoice.TotalTaxes,
		ivaAmount,
		invoice.Total,
		invoice.Currency,
		invoice.PaymentMethod,
		invoice.PaymentTerms,
		invoice.ReferencesInvoiceID,
		factura.Identificacion.NumeroControl,
		factura.Identificacion.TipoDte,
		factura.Identificacion.Ambiente,
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
		codigos.DocTypeFacturasExportacion, // Type 11 - Factura de Exportación
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
		UploadDTEToS3Async(dteJSON, "unsigned", "11", invoice.CompanyID, exportDTE.Identificacion.CodigoGeneracion)
		UploadDTEToS3Async([]byte(signedDTE), "signed", "11", invoice.CompanyID, exportDTE.Identificacion.CodigoGeneracion)
		haciendaResponseJSON, _ := json.MarshalIndent(response, "", "  ")
		UploadDTEToS3Async(haciendaResponseJSON, "hacienda_response", "11", invoice.CompanyID, strings.ToUpper(exportDTE.Identificacion.CodigoGeneracion))
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
