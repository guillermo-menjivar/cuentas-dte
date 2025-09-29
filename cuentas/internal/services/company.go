package services

import (
	"context"
	"database/sql"
	"fmt"

	"cuentas/internal/database"
	"cuentas/internal/models"

	"github.com/google/uuid"
)

type CompanyService struct {
	vaultService *VaultService
}

func NewCompanyService(vaultService *VaultService) *CompanyService {
	return &CompanyService{
		vaultService: vaultService,
	}
}

// CreateCompany handles the complete company creation flow
func (s *CompanyService) CreateCompany(ctx context.Context, req *models.CreateCompanyRequest) (*models.Company, error) {
	// Generate UUID
	companyID := uuid.New().String()

	// Store password in Vault FIRST
	vaultRef, err := s.vaultService.StoreCompanyPassword(companyID, req.HCPassword)
	if err != nil {
		return nil, fmt.Errorf("failed to store password in vault: %v", err)
	}

	// Insert into database
	company, err := s.insertCompany(ctx, companyID, req, vaultRef)
	if err != nil {
		return nil, fmt.Errorf("failed to insert company: %v", err)
	}

	return company, nil
}

// GetCompanyByID retrieves a company by ID
func (s *CompanyService) GetCompanyByID(ctx context.Context, id string) (*models.Company, error) {
	var company models.Company
	query := `
		SELECT id, name, nit, ncr, hc_username, hc_password_ref, last_activity_at, email, active, created_at, updated_at
		FROM companies
		WHERE id = $1
	`

	err := database.DB.QueryRowContext(ctx, query, id).Scan(
		&company.ID,
		&company.Name,
		&company.NIT,
		&company.NCR,
		&company.HCUsername,
		&company.HCPasswordRef,
		&company.LastActivityAt,
		&company.Email,
		&company.Active,
		&company.CreatedAt,
		&company.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("company not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query company: %v", err)
	}

	return &company, nil
}

// insertCompany inserts a company into the database
func (s *CompanyService) insertCompany(ctx context.Context, companyID string, req *models.CreateCompanyRequest, vaultRef string) (*models.Company, error) {
	tx, err := database.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	var company models.Company
	query := `
		INSERT INTO companies (id, name, nit, ncr, hc_username, hc_password_ref, email)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, name, nit, ncr, hc_username, hc_password_ref, last_activity_at, email, active, created_at, updated_at
	`

	err = tx.QueryRowContext(ctx, query,
		companyID,
		req.Name,
		req.NIT,
		req.NCR,
		req.HCUsername,
		vaultRef,
		req.Email,
	).Scan(
		&company.ID,
		&company.Name,
		&company.NIT,
		&company.NCR,
		&company.HCUsername,
		&company.HCPasswordRef,
		&company.LastActivityAt,
		&company.Email,
		&company.Active,
		&company.CreatedAt,
		&company.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %v", err)
	}

	return &company, nil
}
