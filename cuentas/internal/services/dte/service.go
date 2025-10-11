package dte

import (
	"context"
	"database/sql"
	"fmt"

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
	}
}

// SignDTE signs a DTE document for a company
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

	// TODO: Log transaction

	return signedDTE, nil
}

// InvalidateCredentials invalidates cached credentials for a company
func (s *DTEService) InvalidateCredentials(ctx context.Context, companyID uuid.UUID) error {
	return s.credCache.Invalidate(ctx, companyID)
}
