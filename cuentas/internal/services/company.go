package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"cuentas/internal/codigos" // ⭐ Add this import
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
	// ⭐ Validate DTEAmbiente
	if err := s.validateDTEAmbiente(req.DTEAmbiente); err != nil {
		return nil, err
	}

	// Generate UUID
	companyID := uuid.New().String()

	// Strip dashes from NIT and NCR - keep as strings
	nitStripped := tools.StripNIT(req.NIT)
	ncrStripped := tools.StripNRC(req.NCR)

	descActividad, exists := codigos.GetEconomicActivityName(req.CodActividad)
	if !exists {
		return nil, errors.New("invalid economic activity code")
	}

	// Store password in Vault FIRST
	vaultRef, err := s.vaultService.StoreCompanyPassword(companyID, req.HCPassword)
	if err != nil {
		return nil, fmt.Errorf("failed to store password in vault: %v", err)
	}

	// Store firmador password in Vault
	firmadorVaultRef, err := s.vaultService.StoreCompanyPassword(companyID+"_firmador", req.FirmadorPassword)
	if err != nil {
		// Cleanup HC password if firmador fails
		if delErr := s.vaultService.DeleteCompanyPassword(vaultRef); delErr != nil {
			fmt.Printf("Warning: failed to cleanup HC vault entry: %v\n", delErr)
		}
		return nil, fmt.Errorf("failed to store firmador password in vault: %v", err)
	}

	// Insert into database
	company, err := s.insertCompany(ctx, companyID, req, nitStripped, ncrStripped, vaultRef, firmadorVaultRef, descActividad)
	if err != nil {
		// Cleanup: delete from Vault if DB insert fails
		if delErr := s.vaultService.DeleteCompanyPassword(vaultRef); delErr != nil {
			fmt.Printf("Warning: failed to cleanup HC vault entry: %v\n", delErr)
		}
		if delErr := s.vaultService.DeleteCompanyPassword(firmadorVaultRef); delErr != nil {
			fmt.Printf("Warning: failed to cleanup firmador vault entry: %v\n", delErr)
		}
		return nil, fmt.Errorf("failed to insert company: %v", err)
	}

	return company, nil
}

// ⭐ Add validation function
func (s *CompanyService) validateDTEAmbiente(ambiente string) error {
	switch ambiente {
	case codigos.MODE_PRUEBA, codigos.MODE_PRODUCTION:
		return nil
	default:
		return fmt.Errorf("invalid dte_ambiente: '%s'. Must be '00' (test) or '01' (production)", ambiente)
	}
}

// GetCompanyByID retrieves a company by ID
func (s *CompanyService) GetCompanyByID(ctx context.Context, id string) (*models.Company, error) {
	var company models.Company
	query := `
		SELECT id, name, nit, ncr, hc_username, hc_password_ref, last_activity_at, email, active, created_at, updated_at,
		       cod_actividad, desc_actividad, nombre_comercial, dte_ambiente,
		       departamento, municipio, complemento_direccion, telefono,
		       firmador_username, firmador_password_ref
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
		&company.CodActividad,
		&company.DescActividad,
		&company.NombreComercial,
		&company.DTEAmbiente,
		&company.Departamento,
		&company.Municipio,
		&company.ComplementoDireccion,
		&company.Telefono,
		&company.FirmadorUsername,
		&company.FirmadorPasswordRef,
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
func (s *CompanyService) insertCompany(
	ctx context.Context,
	companyID string,
	req *models.CreateCompanyRequest,
	nit, ncr string,
	vaultRef, firmadorVaultRef, descActividad string,
) (*models.Company, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	var company models.Company
	query := `
		INSERT INTO companies (
			id, name, nit, ncr, hc_username, hc_password_ref, email,
			cod_actividad, desc_actividad, nombre_comercial, dte_ambiente,
			departamento, municipio, complemento_direccion, telefono,
			firmador_username, firmador_password_ref
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
		RETURNING id, name, nit, ncr, hc_username, hc_password_ref, last_activity_at, email, active, created_at, updated_at,
		          cod_actividad, desc_actividad, nombre_comercial, dte_ambiente,
		          departamento, municipio, complemento_direccion, telefono,
		          firmador_username, firmador_password_ref
	`

	err = tx.QueryRowContext(ctx, query,
		companyID,
		req.Name,
		nit,
		ncr,
		req.HCUsername,
		vaultRef,
		req.Email,
		req.CodActividad,
		descActividad,
		req.NombreComercial,
		req.DTEAmbiente,
		req.Departamento,
		req.Municipio,
		req.ComplementoDireccion,
		req.Telefono,
		req.FirmadorUsername,
		firmadorVaultRef,
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
		&company.CodActividad,
		&company.DescActividad,
		&company.NombreComercial,
		&company.DTEAmbiente,
		&company.Departamento,
		&company.Municipio,
		&company.ComplementoDireccion,
		&company.Telefono,
		&company.FirmadorUsername,
		&company.FirmadorPasswordRef,
	)

	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %v", err)
	}

	return &company, nil
}
