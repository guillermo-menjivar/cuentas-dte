package dte

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

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
	db              *sql.DB
	redis           *redis.Client
	firmador        *firmador.Client
	vault           *services.VaultService
	credCache       *CredentialCache
	hacienda        *hacienda.Client
	haciendaService *services.HaciendaService
	builder         *Builder
}

// NewDTEService creates a new DTE service (singleton)
func NewDTEService(
	db *sql.DB,
	redis *redis.Client,
	firmador *firmador.Client,
	vault *services.VaultService,
	haciendaClient *hacienda.Client,
	haciendaService *services.HaciendaService,
) *DTEService {
	return &DTEService{
		db:              db,
		hacienda:        haciendaClient,
		redis:           redis,
		firmador:        firmador,
		vault:           vault,
		credCache:       NewCredentialCache(redis),
		builder:         NewBuilder(db),
		haciendaService: haciendaService,
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
	}
	// TODO: Step 6: Log transaction to database

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
