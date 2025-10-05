package services

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"

	"cuentas/internal/models"
	"cuentas/internal/tools"
)

type ClientService struct {
	db *sql.DB
}

func NewClientService(db *sql.DB) *ClientService {
	return &ClientService{db: db}
}

func (s *ClientService) CreateClient(ctx context.Context, companyID string, req *models.CreateClientRequest) (*models.Client, error) {
	var ncrInt, nitInt, duiInt *int64

	// Process NCR if provided
	if req.NCR != "" {
		ncrStripped := tools.StripNRC(req.NCR)
		ncr, err := strconv.ParseInt(ncrStripped, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid NCR format: %w", err)
		}
		ncrInt = &ncr
	}

	// Process NIT if provided
	if req.NIT != "" {
		nitStripped := tools.StripNIT(req.NIT)
		nit, err := strconv.ParseInt(nitStripped, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid NIT format: %w", err)
		}
		nitInt = &nit
	}

	// Process DUI if provided
	if req.DUI != "" {
		duiStripped := tools.StripDUI(req.DUI)
		dui, err := strconv.ParseInt(duiStripped, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid DUI format: %w", err)
		}
		duiInt = &dui
	}

	// Construct full municipality code with dot notation: "06.23"
	fullMunicipalityCode := fmt.Sprintf("%s.%s", req.DepartmentCode, req.MunicipalityCode)

	// Insert into database
	query := `
	INSERT INTO clients (
		company_id, ncr, nit, dui,
		business_name, legal_business_name, giro, tipo_contribuyente, tipo_persona,
		full_address, country_code, department_code, municipality_code
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	RETURNING id, company_id, ncr, nit, dui, 
			  business_name, legal_business_name, giro, tipo_contribuyente, tipo_persona,
			  full_address, country_code, department_code, municipality_code,
			  active, created_at, updated_at
`

	var client models.Client
	// Update the QueryRowContext parameters:
	err := s.db.QueryRowContext(ctx, query,
		companyID, ncrInt, nitInt, duiInt,
		req.BusinessName, req.LegalBusinessName, req.Giro, req.TipoContribuyente, req.TipoPersona,
		req.FullAddress, req.CountryCode, req.DepartmentCode, fullMunicipalityCode,
	).Scan(
		&client.ID, &client.CompanyID, &client.NCR, &client.NIT, &client.DUI,
		&client.BusinessName, &client.LegalBusinessName, &client.Giro, &client.TipoContribuyente, &client.TipoPersona,
		&client.FullAddress, &client.CountryCode, &client.DepartmentCode, &client.MunicipalityCode,
		&client.Active, &client.CreatedAt, &client.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	// Format the numbers for JSON output (only if they exist)
	if client.NCR != nil {
		client.NCRFormatted = tools.FormatNRC(fmt.Sprintf("%d", *client.NCR))
	}
	if client.NIT != nil {
		client.NITFormatted = tools.FormatNIT(fmt.Sprintf("%d", *client.NIT))
	}
	if client.DUI != nil {
		client.DUIFormatted = tools.FormatDUI(fmt.Sprintf("%d", *client.DUI))
	}

	return &client, nil
}

func (s *ClientService) GetClientByID(ctx context.Context, companyID, clientID string) (*models.Client, error) {
	query := `
		SELECT id, company_id, ncr, nit, dui,
			   business_name, legal_business_name, giro, tipo_contribuyente, tipo_persona,
			   full_address, country_code, department_code, municipality_code,
			   active, created_at, updated_at
		FROM clients
		WHERE id = $1 AND company_id = $2
	`

	var client models.Client
	err := s.db.QueryRowContext(ctx, query, clientID, companyID).Scan(
		&client.ID, &client.CompanyID, &client.NCR, &client.NIT, &client.DUI,
		&client.BusinessName, &client.LegalBusinessName, &client.Giro, &client.TipoContribuyente, &client.TipoPersona,
		&client.FullAddress, &client.CountryCode, &client.DepartmentCode, &client.MunicipalityCode,
		&client.Active, &client.CreatedAt, &client.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	// Format the numbers for JSON output (only if they exist)
	if client.NCR != nil {
		client.NCRFormatted = tools.FormatNRC(fmt.Sprintf("%d", *client.NCR))
	}
	if client.NIT != nil {
		client.NITFormatted = tools.FormatNIT(fmt.Sprintf("%d", *client.NIT))
	}
	if client.DUI != nil {
		client.DUIFormatted = tools.FormatDUI(fmt.Sprintf("%d", *client.DUI))
	}

	return &client, nil
}

func (s *ClientService) ListClients(ctx context.Context, companyID string, activeOnly bool) ([]models.Client, error) {
	query := `
		SELECT id, company_id, ncr, nit, dui,
			   business_name, legal_business_name, giro, tipo_contribuyente,
			   full_address, country_code, department_code, municipality_code,
			   active, created_at, updated_at
		FROM clients
		WHERE company_id = $1
	`

	args := []interface{}{companyID}
	if activeOnly {
		query += " AND active = $2"
		args = append(args, true)
	}

	query += " ORDER BY business_name ASC"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list clients: %w", err)
	}
	defer rows.Close()

	var clients []models.Client
	for rows.Next() {
		var client models.Client
		err := rows.Scan(
			&client.ID, &client.CompanyID, &client.NCR, &client.NIT, &client.DUI,
			&client.BusinessName, &client.LegalBusinessName, &client.Giro, &client.TipoContribuyente,
			&client.FullAddress, &client.CountryCode, &client.DepartmentCode, &client.MunicipalityCode,
			&client.Active, &client.CreatedAt, &client.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan client: %w", err)
		}

		// Format the numbers for JSON output (only if they exist)
		if client.NCR != nil {
			client.NCRFormatted = tools.FormatNRC(fmt.Sprintf("%d", *client.NCR))
		}
		if client.NIT != nil {
			client.NITFormatted = tools.FormatNIT(fmt.Sprintf("%d", *client.NIT))
		}
		if client.DUI != nil {
			client.DUIFormatted = tools.FormatDUI(fmt.Sprintf("%d", *client.DUI))
		}

		clients = append(clients, client)
	}

	return clients, nil
}

func (s *ClientService) UpdateClient(ctx context.Context, companyID, clientID string, req *models.CreateClientRequest) (*models.Client, error) {
	var ncrInt, nitInt, duiInt *int64

	// Process NCR if provided
	if req.NCR != "" {
		ncrStripped := tools.StripNRC(req.NCR)
		ncr, err := strconv.ParseInt(ncrStripped, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid NCR format: %w", err)
		}
		ncrInt = &ncr
	}

	// Process NIT if provided
	if req.NIT != "" {
		nitStripped := tools.StripNIT(req.NIT)
		nit, err := strconv.ParseInt(nitStripped, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid NIT format: %w", err)
		}
		nitInt = &nit
	}

	// Process DUI if provided
	if req.DUI != "" {
		duiStripped := tools.StripDUI(req.DUI)
		dui, err := strconv.ParseInt(duiStripped, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid DUI format: %w", err)
		}
		duiInt = &dui
	}

	// Construct full municipality code with dot notation: "06.23"
	fullMunicipalityCode := fmt.Sprintf("%s.%s", req.DepartmentCode, req.MunicipalityCode)

	query := `
		UPDATE clients SET
			ncr = $3, nit = $4, dui = $5,
			business_name = $6, legal_business_name = $7, giro = $8, tipo_contribuyente = $9,
			full_address = $10, country_code = $11, department_code = $12, municipality_code = $13
		WHERE id = $1 AND company_id = $2
		RETURNING id, company_id, ncr, nit, dui,
				  business_name, legal_business_name, giro, tipo_contribuyente,
				  full_address, country_code, department_code, municipality_code,
				  active, created_at, updated_at
	`

	var client models.Client
	err := s.db.QueryRowContext(ctx, query,
		clientID, companyID, ncrInt, nitInt, duiInt,
		req.BusinessName, req.LegalBusinessName, req.Giro, req.TipoContribuyente,
		req.FullAddress, req.CountryCode, req.DepartmentCode, fullMunicipalityCode,
	).Scan(
		&client.ID, &client.CompanyID, &client.NCR, &client.NIT, &client.DUI,
		&client.BusinessName, &client.LegalBusinessName, &client.Giro, &client.TipoContribuyente,
		&client.FullAddress, &client.CountryCode, &client.DepartmentCode, &client.MunicipalityCode,
		&client.Active, &client.CreatedAt, &client.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	// Format the numbers for JSON output (only if they exist)
	if client.NCR != nil {
		client.NCRFormatted = tools.FormatNRC(fmt.Sprintf("%d", *client.NCR))
	}
	if client.NIT != nil {
		client.NITFormatted = tools.FormatNIT(fmt.Sprintf("%d", *client.NIT))
	}
	if client.DUI != nil {
		client.DUIFormatted = tools.FormatDUI(fmt.Sprintf("%d", *client.DUI))
	}

	return &client, nil
}

func (s *ClientService) DeleteClient(ctx context.Context, companyID, clientID string) error {
	query := `
		UPDATE clients SET active = false
		WHERE id = $1 AND company_id = $2
	`

	result, err := s.db.ExecContext(ctx, query, clientID, companyID)
	if err != nil {
		return fmt.Errorf("failed to delete client: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check affected rows: %w", err)
	}

	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}
