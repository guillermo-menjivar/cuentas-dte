package services

import (
	"context"
	"cuentas/internal/hacienda"
	"cuentas/internal/models"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// DTEReconciliationService handles reconciliation of DTEs with Hacienda
type DTEReconciliationService struct {
	db              *sql.DB
	haciendaClient  *hacienda.Client
	haciendaService *HaciendaService
}

// NewDTEReconciliationService creates a new reconciliation service
func NewDTEReconciliationService(
	db *sql.DB,
	haciendaClient *hacienda.Client,
	haciendaService *HaciendaService,
) *DTEReconciliationService {
	return &DTEReconciliationService{
		db:              db,
		haciendaClient:  haciendaClient,
		haciendaService: haciendaService,
	}
}

// ReconcileDTEs performs reconciliation for DTEs matching the given filters
func (s *DTEReconciliationService) ReconcileDTEs(
	ctx context.Context,
	companyID string,
	startDate, endDate *string,
	codigoGeneracion *string,
	includeMatches bool,
) ([]models.DTEReconciliationRecord, *models.DTEReconciliationSummary, error) {

	// Get company NIT for Hacienda queries
	companyNIT, err := s.getCompanyNIT(ctx, companyID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get company NIT: %w", err)
	}

	// Build query to get DTEs from database
	query := `
		SELECT 
			codigo_generacion,
			invoice_id,
			invoice_number,
			client_id,
			numero_control,
			tipo_dte,
			fecha_emision,
			total_amount,
			hacienda_estado,
			hacienda_sello_recibido,
			hacienda_fh_procesamiento
		FROM dte_commit_log
		WHERE company_id = $1
	`

	args := []interface{}{companyID}
	argCount := 1

	// Add filters
	if codigoGeneracion != nil && *codigoGeneracion != "" {
		argCount++
		query += fmt.Sprintf(" AND codigo_generacion = $%d", argCount)
		args = append(args, *codigoGeneracion)
	}

	if startDate != nil && *startDate != "" {
		argCount++
		query += fmt.Sprintf(" AND fecha_emision >= $%d", argCount)
		args = append(args, *startDate)
	}

	if endDate != nil && *endDate != "" {
		argCount++
		query += fmt.Sprintf(" AND fecha_emision <= $%d", argCount)
		args = append(args, *endDate)
	}

	query += " ORDER BY fecha_emision DESC, submitted_at DESC"

	// Execute query
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to query DTEs: %w", err)
	}
	defer rows.Close()

	results := []models.DTEReconciliationRecord{}
	summary := &models.DTEReconciliationSummary{}

	// Get authentication token once for all queries
	authToken, err := s.haciendaService.GetAuthToken(ctx, companyID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get Hacienda auth token: %w", err)
	}

	// Process each DTE
	for rows.Next() {
		var record models.DTEReconciliationRecord
		var fechaEmision time.Time

		err := rows.Scan(
			&record.CodigoGeneracion,
			&record.InvoiceID,
			&record.InvoiceNumber,
			&record.ClientID,
			&record.NumeroControl,
			&record.TipoDTE,
			&fechaEmision,
			&record.TotalAmount,
			&record.InternalEstado,
			&record.InternalSello,
			&record.InternalFhProcesamiento,
		)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to scan row: %w", err)
		}

		record.FechaEmision = fechaEmision.Format("2006-01-02")
		record.QueriedAt = time.Now().Format(time.RFC3339)

		// Query Hacienda for this DTE
		s.queryAndCompareHacienda(ctx, authToken, companyNIT, &record)

		// Update summary
		summary.TotalRecords++
		switch record.HaciendaQueryStatus {
		case "success":
			if record.Matches {
				summary.MatchedRecords++
			} else {
				summary.MismatchedRecords++
			}
			// Count date mismatches specifically
			if !record.FechaEmisionMatches {
				summary.DateMismatches++
			}
		case "not_found":
			summary.NotFoundInHacienda++
		case "error":
			summary.QueryErrors++
		}

		// Add to results based on filter
		if includeMatches || !record.Matches {
			results = append(results, record)
		}
	}

	if err = rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return results, summary, nil
}

// queryAndCompareHacienda queries Hacienda and compares the results
func (s *DTEReconciliationService) queryAndCompareHacienda(
	ctx context.Context,
	authToken string,
	companyNIT string,
	record *models.DTEReconciliationRecord,
) {
	// Remove dashes from NIT
	nitSinGuiones := strings.ReplaceAll(companyNIT, "-", "")

	// Query Hacienda
	haciendaResp, err := s.haciendaClient.ConsultarDTE(
		ctx,
		authToken,
		nitSinGuiones,
		record.TipoDTE,
		record.CodigoGeneracion,
	)

	if err != nil {
		// Check if it's a "not found" error
		if hacErr, ok := err.(*hacienda.HaciendaError); ok {
			if hacErr.Type == "not_found" {
				record.HaciendaQueryStatus = "not_found"
				record.ErrorMessage = "DTE not found in Hacienda"
				record.Matches = false
				record.FechaEmisionMatches = false
				record.Discrepancies = []string{"DTE does not exist in Hacienda's system"}
				return
			}
		}

		// Other errors
		record.HaciendaQueryStatus = "error"
		record.ErrorMessage = err.Error()
		record.Matches = false
		record.FechaEmisionMatches = false
		return
	}

	// Successfully retrieved from Hacienda
	record.HaciendaQueryStatus = "success"
	record.HaciendaEstado = haciendaResp.Estado
	record.HaciendaSello = haciendaResp.SelloRecibido
	record.HaciendaFhProcesamiento = haciendaResp.FhProcesamiento
	record.HaciendaCodigoMsg = haciendaResp.CodigoMsg
	record.HaciendaDescripcionMsg = haciendaResp.DescripcionMsg
	record.HaciendaObservaciones = haciendaResp.Observaciones

	// Compare records (including fecha_emision)
	s.compareRecords(record, haciendaResp.FechaEmision)
}

// compareRecords compares internal and Hacienda records
// compareRecords compares internal and Hacienda records
func (s *DTEReconciliationService) compareRecords(record *models.DTEReconciliationRecord, haciendaFechaEmision string) {
	record.Discrepancies = []string{}
	record.FechaEmisionMatches = true // Default to true

	// Compare estado
	internalEstado := ""
	if record.InternalEstado != nil {
		internalEstado = *record.InternalEstado
	}

	if internalEstado != record.HaciendaEstado {
		record.Discrepancies = append(record.Discrepancies,
			fmt.Sprintf("Estado mismatch: internal='%s' hacienda='%s'",
				internalEstado, record.HaciendaEstado))
	}

	// Compare sello (if both exist)
	internalSello := ""
	if record.InternalSello != nil {
		internalSello = *record.InternalSello
	}

	if internalSello != "" && record.HaciendaSello != "" {
		if internalSello != record.HaciendaSello {
			record.Discrepancies = append(record.Discrepancies,
				fmt.Sprintf("Sello mismatch: internal='%s' hacienda='%s'",
					internalSello, record.HaciendaSello))
		}
	}

	// Compare fecha de emisión
	if haciendaFechaEmision != "" {
		haciendaDate, err := time.Parse("02/01/2006", haciendaFechaEmision)
		if err == nil {
			internalDate, err := time.Parse("2006-01-02", record.FechaEmision)
			if err == nil {
				if !haciendaDate.Equal(internalDate) {
					record.FechaEmisionMatches = false
					record.Discrepancies = append(record.Discrepancies,
						fmt.Sprintf("Fecha emisión mismatch: internal='%s' hacienda='%s'",
							record.FechaEmision, haciendaFechaEmision))
				}
			} else {
				record.FechaEmisionMatches = false
				record.Discrepancies = append(record.Discrepancies,
					fmt.Sprintf("Failed to parse internal fecha emisión: %v", err))
			}
		} else {
			record.FechaEmisionMatches = false
			record.Discrepancies = append(record.Discrepancies,
				fmt.Sprintf("Failed to parse Hacienda fecha emisión: %v", err))
		}
	}

	// Compare fecha procesamiento
	// CRITICAL FIX: Both timestamps are in UTC, compare directly
	if record.InternalFhProcesamiento != nil && record.HaciendaFhProcesamiento != "" {
		// Parse Hacienda's timestamp - format is "dd/MM/yyyy HH:mm:ss" in UTC
		haciendaTime, err := time.Parse("02/01/2006 15:04:05", record.HaciendaFhProcesamiento)
		if err == nil {
			// CRITICAL: Hacienda returns UTC timestamps, so treat as UTC
			haciendaTimeUTC := time.Date(
				haciendaTime.Year(),
				haciendaTime.Month(),
				haciendaTime.Day(),
				haciendaTime.Hour(),
				haciendaTime.Minute(),
				haciendaTime.Second(),
				0,
				time.UTC,
			)

			// Both are now in UTC, compare directly
			// Allow up to 1 minute difference due to clock skew
			diff := record.InternalFhProcesamiento.Sub(haciendaTimeUTC).Abs()
			if diff > time.Minute {
				// They don't match - add discrepancy
				record.Discrepancies = append(record.Discrepancies,
					fmt.Sprintf("Fecha procesamiento mismatch: internal='%s' hacienda='%s' (diff: %v)",
						record.InternalFhProcesamiento.UTC().Format("02/01/2006 15:04:05"),
						haciendaTimeUTC.UTC().Format("02/01/2006 15:04:05"),
						diff))
			}
		}
	}

	// Set matches flag
	record.Matches = len(record.Discrepancies) == 0 && record.FechaEmisionMatches
}

// getCompanyNIT retrieves the company's NIT from the database
func (s *DTEReconciliationService) getCompanyNIT(ctx context.Context, companyID string) (string, error) {
	var nit string
	query := `SELECT nit FROM companies WHERE id = $1`
	err := s.db.QueryRowContext(ctx, query, companyID).Scan(&nit)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("company not found")
		}
		return "", err
	}
	return nit, nil
}

// ReconcileSingleDTE reconciles a single DTE by codigo_generacion
func (s *DTEReconciliationService) ReconcileSingleDTE(
	ctx context.Context,
	companyID string,
	codigoGeneracion string,
) (*models.DTEReconciliationRecord, error) {
	cg := codigoGeneracion
	results, _, err := s.ReconcileDTEs(ctx, companyID, nil, nil, &cg, true)
	if err != nil {
		return nil, err
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("DTE not found in database")
	}

	return &results[0], nil
}
