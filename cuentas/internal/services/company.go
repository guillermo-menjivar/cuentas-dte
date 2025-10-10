package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"

	"cuentas/internal/codigos"
	"cuentas/internal/models"
	"cuentas/internal/tools"

	"github.com/google/uuid"
)

type CompanyService struct {
	db           *sql.DB
	vaultService *VaultService
}

func NewCompanyService(db *sql.DB, vaultService *VaultService) (*CompanyService, error) {
	if db == nil {
		return nil, fmt.Errorf("database connection is required")
	}
	if vaultService == nil {
		return nil, fmt.Errorf("vault service is required")
	}

	return &CompanyService{
		db:           db,
		vaultService: vaultService,
	}, nil
}

// CreateCompany handles the complete company creation flow
func (s *CompanyService) CreateCompany(ctx context.Context, req *models.CreateCompanyRequest) (*models.Company, error) {
	// Generate UUID
	companyID := uuid.New().String()

	// Strip dashes from NIT and NCR, convert to int64
	nitStripped := tools.StripNIT(req.NIT)
	ncrStripped := tools.StripNRC(req.NCR)

	nitInt, err := strconv.ParseInt(nitStripped, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse NIT: %v", err)
	}

	ncrInt, err := strconv.ParseInt(ncrStripped, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse NCR: %v", err)
	}

	descActividad, exists := codigos.GetEconomicActivityName(req.CodActividad)
	if !exists {
		fmt.Println("you submitted", req.CodActividad)
		return nil, errors.New("invalid economic activity code")
	}

	// Store password in Vault FIRST
	vaultRef, err := s.vaultService.StoreCompanyPassword(companyID, req.HCPassword)
	if err != nil {
		return nil, fmt.Errorf("failed to store password in vault: %v", err)
	}

	// Insert into database
	company, err := s.insertCompany(ctx, companyID, req, nitInt, ncrInt, vaultRef)
	if err != nil {
		// Cleanup: delete from Vault if DB insert fails
		if delErr := s.vaultService.DeleteCompanyPassword(vaultRef); delErr != nil {
			fmt.Printf("Warning: failed to cleanup vault entry: %v\n", delErr)
		}
		return nil, fmt.Errorf("failed to insert company: %v", err)
	}

	// Store both in database
	company.CodActividad = req.CodActividad
	company.DescActividad = descActividad

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

	err := s.db.QueryRowContext(ctx, query, id).Scan(
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

	// Format NIT and NCR for JSON response
	company.NITFormatted = tools.FormatNIT(fmt.Sprintf("%d", company.NIT))
	company.NCRFormatted = tools.FormatNRC(fmt.Sprintf("%d", company.NCR))

	return &company, nil
}

// insertCompany inserts a company into the database
func (s *CompanyService) insertCompany(ctx context.Context, companyID string, req *models.CreateCompanyRequest, nitInt, ncrInt int64, vaultRef string) (*models.Company, error) {
	tx, err := s.db.BeginTx(ctx, nil)
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
		nitInt,
		ncrInt,
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

	// Format NIT and NCR for JSON response
	company.NITFormatted = tools.FormatNIT(fmt.Sprintf("%d", company.NIT))
	company.NCRFormatted = tools.FormatNRC(fmt.Sprintf("%d", company.NCR))

	return &company, nil
}
