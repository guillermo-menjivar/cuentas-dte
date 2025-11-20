package services

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"cuentas/internal/models"
	"cuentas/internal/tools"
)

// BulkCreateResult represents the result of a bulk create operation
type BulkCreateResult struct {
	TotalRows      int                     `json:"total_rows"`
	SuccessCount   int                     `json:"success_count"`
	FailedCount    int                     `json:"failed_count"`
	SuccessClients []models.Client         `json:"success_clients,omitempty"`
	FailedClients  []BulkCreateClientError `json:"failed_clients,omitempty"`
}

// BulkCreateClientError represents a failed client creation in bulk operation
type BulkCreateClientError struct {
	Row          int    `json:"row"`
	BusinessName string `json:"business_name"`
	Error        string `json:"error"`
}

// BulkCreateClients creates multiple clients in a single transaction
func (s *ClientService) BulkCreateClients(ctx context.Context, companyID string, requests []models.CreateClientRequest) (*BulkCreateResult, error) {
	result := &BulkCreateResult{
		TotalRows:      len(requests),
		SuccessClients: []models.Client{},
		FailedClients:  []BulkCreateClientError{},
	}

	// Start transaction
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	// Prepare insert statement
	insertQuery := `
        INSERT INTO clients (
            company_id, ncr, nit, dui,
            business_name, legal_business_name, giro, tipo_contribuyente, tipo_persona,
            full_address, country_code, department_code, municipality_code, 
            cod_actividad, desc_actividad, correo, telefono
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
        RETURNING id, company_id, ncr, nit, dui, 
                  business_name, legal_business_name, giro, tipo_contribuyente, tipo_persona,
                  full_address, country_code, department_code, municipality_code, 
                  cod_actividad, desc_actividad, correo, telefono,
                  active, created_at, updated_at
    `

	stmt, err := tx.PrepareContext(ctx, insertQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	// Insert each client
	for i, req := range requests {
		rowNumber := i + 2 // +2 because row 1 is header, and i starts at 0

		var ncrInt, nitInt, duiInt *int64

		// Process NCR if provided
		if req.NCR != "" {
			ncrStripped := tools.StripNRC(req.NCR)
			ncr, err := strconv.ParseInt(ncrStripped, 10, 64)
			if err != nil {
				result.FailedClients = append(result.FailedClients, BulkCreateClientError{
					Row:          rowNumber,
					BusinessName: req.BusinessName,
					Error:        fmt.Sprintf("invalid NCR format: %v", err),
				})
				result.FailedCount++
				continue
			}
			ncrInt = &ncr
		}

		// Process NIT if provided
		if req.NIT != "" {
			nitStripped := tools.StripNIT(req.NIT)
			nit, err := strconv.ParseInt(nitStripped, 10, 64)
			if err != nil {
				result.FailedClients = append(result.FailedClients, BulkCreateClientError{
					Row:          rowNumber,
					BusinessName: req.BusinessName,
					Error:        fmt.Sprintf("invalid NIT format: %v", err),
				})
				result.FailedCount++
				continue
			}
			nitInt = &nit
		}

		// Process DUI if provided
		if req.DUI != "" {
			duiStripped := tools.StripDUI(req.DUI)
			dui, err := strconv.ParseInt(duiStripped, 10, 64)
			if err != nil {
				result.FailedClients = append(result.FailedClients, BulkCreateClientError{
					Row:          rowNumber,
					BusinessName: req.BusinessName,
					Error:        fmt.Sprintf("invalid DUI format: %v", err),
				})
				result.FailedCount++
				continue
			}
			duiInt = &dui
		}

		// Construct full municipality code
		fullMunicipalityCode := fmt.Sprintf("%s.%s", req.DepartmentCode, req.MunicipalityCode)

		// Execute insert
		var client models.Client
		err := stmt.QueryRowContext(ctx,
			companyID, ncrInt, nitInt, duiInt,
			req.BusinessName, req.LegalBusinessName, req.Giro, req.TipoContribuyente, req.TipoPersona,
			req.FullAddress, req.CountryCode, req.DepartmentCode, fullMunicipalityCode,
			req.CodActividad, req.CodActividadDescription, req.Correo, req.Telefono,
		).Scan(
			&client.ID, &client.CompanyID, &client.NCR, &client.NIT, &client.DUI,
			&client.BusinessName, &client.LegalBusinessName, &client.Giro, &client.TipoContribuyente, &client.TipoPersona,
			&client.FullAddress, &client.CountryCode, &client.DepartmentCode, &client.MunicipalityCode,
			&client.CodActividad, &client.CodActividadDescription, &client.Correo, &client.Telefono,
			&client.Active, &client.CreatedAt, &client.UpdatedAt,
		)

		if err != nil {
			errorMsg := err.Error()

			// Handle duplicate constraints
			if strings.Contains(errorMsg, "unique_client_nit_per_company") {
				errorMsg = "duplicate NIT for this company"
			} else if strings.Contains(errorMsg, "unique_client_dui_per_company") {
				errorMsg = "duplicate DUI for this company"
			}

			result.FailedClients = append(result.FailedClients, BulkCreateClientError{
				Row:          rowNumber,
				BusinessName: req.BusinessName,
				Error:        errorMsg,
			})
			result.FailedCount++
			continue
		}

		// Format the numbers for JSON output
		if client.NCR != nil {
			client.NCRFormatted = tools.FormatNRC(fmt.Sprintf("%d", *client.NCR))
		}
		if client.NIT != nil {
			client.NITFormatted = tools.FormatNIT(fmt.Sprintf("%d", *client.NIT))
		}
		if client.DUI != nil {
			client.DUIFormatted = tools.FormatDUI(fmt.Sprintf("%d", *client.DUI))
		}

		result.SuccessClients = append(result.SuccessClients, client)
		result.SuccessCount++
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return result, nil
}
