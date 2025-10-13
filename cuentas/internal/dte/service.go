package dte

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"cuentas/internal/models"
	"cuentas/internal/services"
	"cuentas/internal/services/firmador"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// DTEService handles DTE signing and submission
type DTEService struct {
	db        *sql.DB
	redis     *redis.Client
	firmador  *firmador.Client
	vault     *services.VaultService
	credCache *CredentialCache
	builder   *DTEBuilder
}

// NewDTEService creates a new DTE service (singleton)
func NewDTEService(
	db *sql.DB,
	redis *redis.Client,
	firmador *firmador.Client,
	vault *services.VaultService,
) *DTEService {
	return &DTEService{
		db:        db,
		redis:     redis,
		firmador:  firmador,
		vault:     vault,
		credCache: NewCredentialCache(redis),
		builder:   NewDTEBuilder(db),
	}
}

// ProcessInvoice builds DTE from invoice, signs it, and prepares for transmission
func (s *DTEService) ProcessInvoice(ctx context.Context, invoice *models.Invoice) (string, error) {
	// Step 1: Build DTE from invoice
	fmt.Println("Step 1: Building DTE from invoice...")
	factura, err := s.builder.BuildFromInvoice(ctx, invoice)
	if err != nil {
		return "", fmt.Errorf("failed to build DTE: %w", err)
	}

	// Pretty print the DTE for debugging
	dteJSON, err := json.MarshalIndent(factura, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal DTE: %w", err)
	}
	fmt.Println("DTE Generated:")
	fmt.Println(string(dteJSON))

	// Step 2: Load company credentials and sign
	fmt.Println("\nStep 2: Loading credentials and signing DTE...")
	companyID, err := uuid.Parse(invoice.CompanyID)
	if err != nil {
		return "", fmt.Errorf("invalid company ID: %w", err)
	}

	creds, err := s.LoadCredentials(ctx, companyID)
	if err != nil {
		return "", fmt.Errorf("failed to load credentials: %w", err)
	}

	signedDTE, err := s.firmador.Sign(ctx, creds.NIT, creds.Password, factura)
	if err != nil {
		return "", fmt.Errorf("failed to sign DTE: %w", err)
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

	// TODO: Step 4: Transmit to Hacienda (not implemented yet)
	// TODO: Step 5: Log transaction

	return signedDTE, nil
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
