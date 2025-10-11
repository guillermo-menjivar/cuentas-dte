// internal/services/dte/credentials.go
package dte

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
)

// LoadCredentials loads company credentials from DB + Vault with caching
func (s *DTEService) LoadCredentials(ctx context.Context, companyID uuid.UUID) (*CachedCredentials, error) {
	// Check cache first
	if creds, found := s.credCache.Get(ctx, companyID); found {
		return creds, nil
	}

	// Cache miss - load from DB + Vault
	creds, err := s.loadCredentialsFromSource(ctx, companyID)
	if err != nil {
		return nil, err
	}

	// Store in cache for next time
	if err := s.credCache.Set(ctx, creds); err != nil {
		// Log error but don't fail - we have the credentials
		// TODO: Add proper logging
		_ = err
	}

	return creds, nil
}

// loadCredentialsFromSource loads credentials from database and Vault
func (s *DTEService) loadCredentialsFromSource(ctx context.Context, companyID uuid.UUID) (*CachedCredentials, error) {
	query := `
		SELECT 
			id, 
			nit, 
			nombre, 
			nombre_comercial,
			firmador_password_ref
		FROM companies
		WHERE id = $1 AND active = true
	`

	var (
		id                  uuid.UUID
		nit                 int64
		nombre              string
		nombreComercial     *string
		firmadorPasswordRef *string
	)

	err := s.db.QueryRowContext(ctx, query, companyID).Scan(
		&id,
		&nit,
		&nombre,
		&nombreComercial,
		&firmadorPasswordRef,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("company not found or inactive: %s", companyID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to load company: %w", err)
	}

	// Check if firmador credentials are configured
	if firmadorPasswordRef == nil || *firmadorPasswordRef == "" {
		return nil, fmt.Errorf("firmador credentials not configured for company: %s", companyID)
	}

	// Load password from Vault
	password, err := s.vault.GetSecret(*firmadorPasswordRef)
	if err != nil {
		return nil, fmt.Errorf("failed to load firmador password from Vault: %w", err)
	}

	// Convert NIT to string (firmador expects string without dashes)
	nitStr := fmt.Sprintf("%d", nit)

	return &CachedCredentials{
		CompanyID:       id,
		NIT:             nitStr,
		Password:        password,
		Nombre:          nombre,
		NombreComercial: nombreComercial,
	}, nil
}
