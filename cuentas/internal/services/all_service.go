package services

import (
	"cuentas/internal/codigos"
	"sort"
)

func sortCategoriesByCode(categories []codigos.ActivityCategory) []codigos.ActivityCategory {
	sort.Slice(categories, func(i, j int) bool {
		return categories[i].Code < categories[j].Code
	})
	return categories
}

func sortActivitiesByCode(activities []codigos.EconomicActivity) []codigos.EconomicActivity {
	sort.Slice(activities, func(i, j int) bool {
		return activities[i].Code < activities[j].Code
	})
	return activities
}

func sortByRelevance(results []SearchResult) []SearchResult {
	sort.Slice(results, func(i, j int) bool {
		return results[i].RelevanceScore > results[j].RelevanceScore
	})
	return results
}
package services

import (
	"cuentas/internal/codigos"
	"strings"
)

type ActividadEconomicaService struct{}

func NewActividadEconomicaService() *ActividadEconomicaService {
	return &ActividadEconomicaService{}
}

// GetAllCategories returns all top-level categories (Level 1)
func (s *ActividadEconomicaService) GetAllCategories() []codigos.ActivityCategory {
	categories := make([]codigos.ActivityCategory, 0)

	for _, cat := range codigos.ActividadEconomicaCategories {
		if cat.Level == 1 {
			categories = append(categories, cat)
		}
	}

	// Sort by code for consistent ordering
	return sortCategoriesByCode(categories)
}

// GetCategoryByCode returns a specific category
func (s *ActividadEconomicaService) GetCategoryByCode(code string) (*codigos.ActivityCategory, bool) {
	cat, exists := codigos.ActividadEconomicaCategories[code]
	return &cat, exists
}

// GetActivitiesByCategory returns all activities that belong to a category
// For example, code "46" returns all activities starting with "46"
func (s *ActividadEconomicaService) GetActivitiesByCategory(categoryCode string) []codigos.EconomicActivity {
	activities := make([]codigos.EconomicActivity, 0)

	for code, name := range codigos.EconomicActivities {
		// Check if activity code starts with category code
		if strings.HasPrefix(code, categoryCode) {
			activities = append(activities, codigos.EconomicActivity{
				Code:  code,
				Value: name,
			})
		}
	}

	return sortActivitiesByCode(activities)
}

// SearchActivities performs a search across both categories and activities
// Returns ranked results based on relevance
func (s *ActividadEconomicaService) SearchActivities(query string, limit int) []SearchResult {
	if query == "" || limit <= 0 {
		return []SearchResult{}
	}

	queryLower := strings.ToLower(strings.TrimSpace(query))
	results := make([]SearchResult, 0)

	// Search in activity codes and names
	for code, name := range codigos.EconomicActivities {
		score := calculateRelevanceScore(queryLower, code, name)
		if score > 0 {
			results = append(results, SearchResult{
				Code:           code,
				Name:           name,
				Type:           "activity",
				RelevanceScore: score,
			})
		}
	}

	// Search in categories (keywords and names)
	for code, cat := range codigos.ActividadEconomicaCategories {
		score := calculateCategoryRelevance(queryLower, code, cat)
		if score > 0 {
			results = append(results, SearchResult{
				Code:           code,
				Name:           cat.Name,
				Type:           "category",
				RelevanceScore: score,
			})
		}
	}

	// Sort by relevance score (highest first)
	results = sortByRelevance(results)

	// Limit results
	if len(results) > limit {
		results = results[:limit]
	}

	return results
}

// GetActivityDetails returns full details of a specific activity including its category
func (s *ActividadEconomicaService) GetActivityDetails(code string) (*ActivityDetails, bool) {
	name, exists := codigos.EconomicActivities[code]
	if !exists {
		return nil, false
	}

	// Find the category this activity belongs to
	categoryCode := getCategoryCodeFromActivity(code)
	category, _ := codigos.ActividadEconomicaCategories[categoryCode]

	// Find related activities (same category)
	related := s.GetActivitiesByCategory(categoryCode)
	// Remove the current activity from related
	related = filterOutActivity(related, code)
	// Limit to 5 related
	if len(related) > 5 {
		related = related[:5]
	}

	return &ActivityDetails{
		Code:              code,
		Name:              name,
		CategoryCode:      categoryCode,
		CategoryName:      category.Name,
		RelatedActivities: related,
	}, true
}

// Helper: Calculate relevance score for activities
func calculateRelevanceScore(query, code, name string) float64 {
	nameLower := strings.ToLower(name)
	score := 0.0

	// Exact code match - highest priority
	if code == query {
		score += 100.0
	} else if strings.HasPrefix(code, query) {
		score += 50.0
	} else if strings.Contains(code, query) {
		score += 25.0
	}

	// Exact phrase match in name
	if strings.Contains(nameLower, query) {
		score += 40.0
	}

	// Word matches in name
	queryWords := strings.Fields(query)
	nameWords := strings.Fields(nameLower)
	matchedWords := 0
	for _, qw := range queryWords {
		for _, nw := range nameWords {
			if strings.Contains(nw, qw) || strings.Contains(qw, nw) {
				matchedWords++
				break
			}
		}
	}
	if len(queryWords) > 0 {
		score += float64(matchedWords) / float64(len(queryWords)) * 30.0
	}

	return score
}

// Helper: Calculate relevance for categories
func calculateCategoryRelevance(query, code string, cat codigos.ActivityCategory) float64 {
	score := 0.0
	nameLower := strings.ToLower(cat.Name)

	// Code match
	if code == query {
		score += 100.0
	} else if strings.HasPrefix(code, query) {
		score += 50.0
	}

	// Name match
	if strings.Contains(nameLower, query) {
		score += 40.0
	}

	// Keyword match
	for _, keyword := range cat.Keywords {
		keywordLower := strings.ToLower(keyword)
		if keywordLower == query {
			score += 60.0
			break
		} else if strings.Contains(keywordLower, query) || strings.Contains(query, keywordLower) {
			score += 30.0
		}
	}

	return score
}

// Helper: Extract category code from activity code
// Activities like "46201" belong to category "46"
func getCategoryCodeFromActivity(activityCode string) string {
	if len(activityCode) >= 2 {
		return activityCode[:2]
	}
	return ""
}

// Helper: Filter out a specific activity from a list
func filterOutActivity(activities []codigos.EconomicActivity, codeToRemove string) []codigos.EconomicActivity {
	filtered := make([]codigos.EconomicActivity, 0)
	for _, act := range activities {
		if act.Code != codeToRemove {
			filtered = append(filtered, act)
		}
	}
	return filtered
}

// DTOs for service responses
type SearchResult struct {
	Code           string  `json:"code"`
	Name           string  `json:"name"`
	Type           string  `json:"type"` // "activity" or "category"
	RelevanceScore float64 `json:"relevance_score,omitempty"`
}

type ActivityDetails struct {
	Code              string                     `json:"code"`
	Name              string                     `json:"name"`
	CategoryCode      string                     `json:"category_code"`
	CategoryName      string                     `json:"category_name"`
	RelatedActivities []codigos.EconomicActivity `json:"related_activities"`
}
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
			   full_address, country_code, department_code, municipality_code, tipo_persona,
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
			&client.FullAddress, &client.CountryCode, &client.DepartmentCode, &client.MunicipalityCode, &client.TipoPersona,
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
	company.DTEAmbiente = req.DTEAmbiente
	company.NombreComercial = req.NombreComercial

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
package services

import "errors"

// Invoice service errors
var (
	ErrClientNotFound        = errors.New("client not found")
	ErrInventoryItemNotFound = errors.New("inventory item not found")
	ErrInvoiceNotFound       = errors.New("invoice not found")
	ErrInvoiceNotDraft       = errors.New("invoice is not in draft status")
	ErrInvoiceAlreadyVoid    = errors.New("invoice is already void")
	ErrInsufficientPayment   = errors.New("payment amount exceeds balance due")
	ErrCreditLimitExceeded   = errors.New("client credit limit exceeded")
	ErrCreditSuspended       = errors.New("client credit is suspended")
	ErrInvalidInvoiceStatus  = errors.New("invalid invoice status for this operation")
	ErrPointOfSaleNotFound   = errors.New("POS not found")
)
package services

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"cuentas/internal/database"
	"cuentas/internal/models"
)

type EstablishmentService struct{}

func NewEstablishmentService() *EstablishmentService {
	return &EstablishmentService{}
}

// CreateEstablishment creates a new establishment for a company
func (s *EstablishmentService) CreateEstablishment(ctx context.Context, companyID string, req *models.CreateEstablishmentRequest) (*models.Establishment, error) {
	// Validate request
	if err := req.Validate(); err != nil {
		return nil, err
	}

	query := `
		INSERT INTO establishments (
			company_id, tipo_establecimiento, nombre,
			cod_establecimiento_mh, cod_establecimiento,
			departamento, municipio, complemento_direccion,
			telefono, active, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
		) RETURNING id, created_at, updated_at
	`

	establishment := &models.Establishment{
		CompanyID:            companyID,
		TipoEstablecimiento:  req.TipoEstablecimiento,
		Nombre:               req.Nombre,
		CodEstablecimientoMH: req.CodEstablecimientoMH,
		CodEstablecimiento:   req.CodEstablecimiento,
		Departamento:         req.Departamento,
		Municipio:            req.Municipio,
		ComplementoDireccion: req.ComplementoDireccion,
		Telefono:             req.Telefono,
		Active:               true,
	}

	now := time.Now()
	err := database.DB.QueryRowContext(ctx, query,
		establishment.CompanyID,
		establishment.TipoEstablecimiento,
		establishment.Nombre,
		establishment.CodEstablecimientoMH,
		establishment.CodEstablecimiento,
		establishment.Departamento,
		establishment.Municipio,
		establishment.ComplementoDireccion,
		establishment.Telefono,
		establishment.Active,
		now,
		now,
	).Scan(&establishment.ID, &establishment.CreatedAt, &establishment.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create establishment: %w", err)
	}

	return establishment, nil
}

// GetEstablishment retrieves an establishment by ID
func (s *EstablishmentService) GetEstablishment(ctx context.Context, companyID, establishmentID string) (*models.Establishment, error) {
	query := `
		SELECT
			id, company_id, tipo_establecimiento, nombre,
			cod_establecimiento_mh, cod_establecimiento,
			departamento, municipio, complemento_direccion,
			telefono, active, created_at, updated_at
		FROM establishments
		WHERE id = $1 AND company_id = $2
	`

	establishment := &models.Establishment{}
	err := database.DB.QueryRowContext(ctx, query, establishmentID, companyID).Scan(
		&establishment.ID,
		&establishment.CompanyID,
		&establishment.TipoEstablecimiento,
		&establishment.Nombre,
		&establishment.CodEstablecimientoMH,
		&establishment.CodEstablecimiento,
		&establishment.Departamento,
		&establishment.Municipio,
		&establishment.ComplementoDireccion,
		&establishment.Telefono,
		&establishment.Active,
		&establishment.CreatedAt,
		&establishment.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, models.ErrEstablishmentNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get establishment: %w", err)
	}

	return establishment, nil
}

// ListEstablishments retrieves all establishments for a company
func (s *EstablishmentService) ListEstablishments(ctx context.Context, companyID string, activeOnly bool) ([]models.Establishment, error) {
	query := `
		SELECT
			id, company_id, tipo_establecimiento, nombre,
			cod_establecimiento_mh, cod_establecimiento,
			departamento, municipio, complemento_direccion,
			telefono, active, created_at, updated_at
		FROM establishments
		WHERE company_id = $1
	`

	if activeOnly {
		query += " AND active = true"
	}

	query += " ORDER BY created_at DESC"

	rows, err := database.DB.QueryContext(ctx, query, companyID)
	if err != nil {
		return nil, fmt.Errorf("failed to list establishments: %w", err)
	}
	defer rows.Close()

	var establishments []models.Establishment
	for rows.Next() {
		var est models.Establishment
		err := rows.Scan(
			&est.ID,
			&est.CompanyID,
			&est.TipoEstablecimiento,
			&est.Nombre,
			&est.CodEstablecimientoMH,
			&est.CodEstablecimiento,
			&est.Departamento,
			&est.Municipio,
			&est.ComplementoDireccion,
			&est.Telefono,
			&est.Active,
			&est.CreatedAt,
			&est.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan establishment: %w", err)
		}
		establishments = append(establishments, est)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating establishments: %w", err)
	}

	return establishments, nil
}

// UpdateEstablishment updates an establishment
func (s *EstablishmentService) UpdateEstablishment(ctx context.Context, companyID, establishmentID string, req *models.UpdateEstablishmentRequest) (*models.Establishment, error) {
	// First verify establishment exists and belongs to company
	existing, err := s.GetEstablishment(ctx, companyID, establishmentID)
	if err != nil {
		return nil, err
	}

	// Build dynamic update query
	updates := []string{}
	args := []interface{}{establishmentID, companyID}
	argCount := 2

	if req.TipoEstablecimiento != nil {
		if !models.IsValidTipoEstablecimiento(*req.TipoEstablecimiento) {
			return nil, models.ErrInvalidTipoEstablecimiento
		}
		argCount++
		updates = append(updates, fmt.Sprintf("tipo_establecimiento = $%d", argCount))
		args = append(args, *req.TipoEstablecimiento)
	}

	if req.Nombre != nil {
		argCount++
		updates = append(updates, fmt.Sprintf("nombre = $%d", argCount))
		args = append(args, *req.Nombre)
	}

	if req.CodEstablecimientoMH != nil {
		if len(*req.CodEstablecimientoMH) != 4 {
			return nil, models.ErrInvalidCodEstablecimientoMH
		}
		argCount++
		updates = append(updates, fmt.Sprintf("cod_establecimiento_mh = $%d", argCount))
		args = append(args, *req.CodEstablecimientoMH)
	}

	if req.CodEstablecimiento != nil {
		argCount++
		updates = append(updates, fmt.Sprintf("cod_establecimiento = $%d", argCount))
		args = append(args, *req.CodEstablecimiento)
	}

	if req.Departamento != nil {
		argCount++
		updates = append(updates, fmt.Sprintf("departamento = $%d", argCount))
		args = append(args, *req.Departamento)
	}

	if req.Municipio != nil {
		argCount++
		updates = append(updates, fmt.Sprintf("municipio = $%d", argCount))
		args = append(args, *req.Municipio)
	}

	if req.ComplementoDireccion != nil {
		argCount++
		updates = append(updates, fmt.Sprintf("complemento_direccion = $%d", argCount))
		args = append(args, *req.ComplementoDireccion)
	}

	if req.Telefono != nil {
		argCount++
		updates = append(updates, fmt.Sprintf("telefono = $%d", argCount))
		args = append(args, *req.Telefono)
	}

	if len(updates) == 0 {
		return existing, nil // Nothing to update
	}

	// Add updated_at
	argCount++
	updates = append(updates, fmt.Sprintf("updated_at = $%d", argCount))
	args = append(args, time.Now())

	query := fmt.Sprintf(`
		UPDATE establishments
		SET %s
		WHERE id = $1 AND company_id = $2
		RETURNING id, company_id, tipo_establecimiento, nombre,
			cod_establecimiento_mh, cod_establecimiento,
			departamento, municipio, complemento_direccion,
			telefono, active, created_at, updated_at
	`, joinStrings(updates, ", "))

	establishment := &models.Establishment{}
	err = database.DB.QueryRowContext(ctx, query, args...).Scan(
		&establishment.ID,
		&establishment.CompanyID,
		&establishment.TipoEstablecimiento,
		&establishment.Nombre,
		&establishment.CodEstablecimientoMH,
		&establishment.CodEstablecimiento,
		&establishment.Departamento,
		&establishment.Municipio,
		&establishment.ComplementoDireccion,
		&establishment.Telefono,
		&establishment.Active,
		&establishment.CreatedAt,
		&establishment.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to update establishment: %w", err)
	}

	return establishment, nil
}

// DeactivateEstablishment deactivates an establishment
func (s *EstablishmentService) DeactivateEstablishment(ctx context.Context, companyID, establishmentID string) error {
	query := `
		UPDATE establishments
		SET active = false, updated_at = $1
		WHERE id = $2 AND company_id = $3
	`

	result, err := database.DB.ExecContext(ctx, query, time.Now(), establishmentID, companyID)
	if err != nil {
		return fmt.Errorf("failed to deactivate establishment: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return models.ErrEstablishmentNotFound
	}

	return nil
}

// CreatePointOfSale creates a new point of sale for an establishment
func (s *EstablishmentService) CreatePointOfSale(ctx context.Context, companyID, establishmentID string, req *models.CreatePOSRequest) (*models.PointOfSale, error) {
	// Validate request
	if err := req.Validate(); err != nil {
		return nil, err
	}

	// Verify establishment exists and belongs to company
	_, err := s.GetEstablishment(ctx, companyID, establishmentID)
	if err != nil {
		return nil, err
	}

	query := `
		INSERT INTO point_of_sale (
			establishment_id, nombre,
			cod_punto_venta_mh, cod_punto_venta,
			latitude, longitude, is_portable,
			active, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10
		) RETURNING id, created_at, updated_at
	`

	pos := &models.PointOfSale{
		EstablishmentID: establishmentID,
		Nombre:          req.Nombre,
		CodPuntoVentaMH: req.CodPuntoVentaMH,
		CodPuntoVenta:   req.CodPuntoVenta,
		Latitude:        req.Latitude,
		Longitude:       req.Longitude,
		IsPortable:      req.IsPortable,
		Active:          true,
	}

	now := time.Now()
	err = database.DB.QueryRowContext(ctx, query,
		pos.EstablishmentID,
		pos.Nombre,
		pos.CodPuntoVentaMH,
		pos.CodPuntoVenta,
		pos.Latitude,
		pos.Longitude,
		pos.IsPortable,
		pos.Active,
		now,
		now,
	).Scan(&pos.ID, &pos.CreatedAt, &pos.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create point of sale: %w", err)
	}

	return pos, nil
}

// GetPointOfSale retrieves a point of sale by ID
func (s *EstablishmentService) GetPointOfSale(ctx context.Context, companyID, posID string) (*models.PointOfSale, error) {
	query := `
		SELECT
			pos.id, pos.establishment_id, pos.nombre,
			pos.cod_punto_venta_mh, pos.cod_punto_venta,
			pos.latitude, pos.longitude, pos.is_portable,
			pos.active, pos.created_at, pos.updated_at
		FROM point_of_sale pos
		JOIN establishments e ON pos.establishment_id = e.id
		WHERE pos.id = $1 AND e.company_id = $2
	`

	pos := &models.PointOfSale{}
	err := database.DB.QueryRowContext(ctx, query, posID, companyID).Scan(
		&pos.ID,
		&pos.EstablishmentID,
		&pos.Nombre,
		&pos.CodPuntoVentaMH,
		&pos.CodPuntoVenta,
		&pos.Latitude,
		&pos.Longitude,
		&pos.IsPortable,
		&pos.Active,
		&pos.CreatedAt,
		&pos.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, models.ErrPointOfSaleNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get point of sale: %w", err)
	}

	return pos, nil
}

// ListPointsOfSale retrieves all points of sale for an establishment
func (s *EstablishmentService) ListPointsOfSale(ctx context.Context, companyID, establishmentID string, activeOnly bool) ([]models.PointOfSale, error) {
	// Verify establishment belongs to company
	_, err := s.GetEstablishment(ctx, companyID, establishmentID)
	if err != nil {
		return nil, err
	}

	query := `
		SELECT
			id, establishment_id, nombre,
			cod_punto_venta_mh, cod_punto_venta,
			latitude, longitude, is_portable,
			active, created_at, updated_at
		FROM point_of_sale
		WHERE establishment_id = $1
	`

	if activeOnly {
		query += " AND active = true"
	}

	query += " ORDER BY created_at DESC"

	rows, err := database.DB.QueryContext(ctx, query, establishmentID)
	if err != nil {
		return nil, fmt.Errorf("failed to list points of sale: %w", err)
	}
	defer rows.Close()

	var pointsOfSale []models.PointOfSale
	for rows.Next() {
		var pos models.PointOfSale
		err := rows.Scan(
			&pos.ID,
			&pos.EstablishmentID,
			&pos.Nombre,
			&pos.CodPuntoVentaMH,
			&pos.CodPuntoVenta,
			&pos.Latitude,
			&pos.Longitude,
			&pos.IsPortable,
			&pos.Active,
			&pos.CreatedAt,
			&pos.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan point of sale: %w", err)
		}
		pointsOfSale = append(pointsOfSale, pos)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating points of sale: %w", err)
	}

	return pointsOfSale, nil
}

// UpdatePointOfSale updates a point of sale
func (s *EstablishmentService) UpdatePointOfSale(ctx context.Context, companyID, posID string, req *models.UpdatePOSRequest) (*models.PointOfSale, error) {
	// Verify POS exists and belongs to company
	existing, err := s.GetPointOfSale(ctx, companyID, posID)
	if err != nil {
		return nil, err
	}

	// Build dynamic update query
	updates := []string{}
	args := []interface{}{posID}
	argCount := 1

	if req.Nombre != nil {
		argCount++
		updates = append(updates, fmt.Sprintf("nombre = $%d", argCount))
		args = append(args, *req.Nombre)
	}

	if req.CodPuntoVentaMH != nil {
		if len(*req.CodPuntoVentaMH) != 4 {
			return nil, models.ErrInvalidCodPuntoVentaMH
		}
		argCount++
		updates = append(updates, fmt.Sprintf("cod_punto_venta_mh = $%d", argCount))
		args = append(args, *req.CodPuntoVentaMH)
	}

	if req.CodPuntoVenta != nil {
		argCount++
		updates = append(updates, fmt.Sprintf("cod_punto_venta = $%d", argCount))
		args = append(args, *req.CodPuntoVenta)
	}

	if req.Latitude != nil {
		argCount++
		updates = append(updates, fmt.Sprintf("latitude = $%d", argCount))
		args = append(args, *req.Latitude)
	}

	if req.Longitude != nil {
		argCount++
		updates = append(updates, fmt.Sprintf("longitude = $%d", argCount))
		args = append(args, *req.Longitude)
	}

	if req.IsPortable != nil {
		argCount++
		updates = append(updates, fmt.Sprintf("is_portable = $%d", argCount))
		args = append(args, *req.IsPortable)
	}

	if len(updates) == 0 {
		return existing, nil // Nothing to update
	}

	// Add updated_at
	argCount++
	updates = append(updates, fmt.Sprintf("updated_at = $%d", argCount))
	args = append(args, time.Now())

	query := fmt.Sprintf(`
		UPDATE point_of_sale
		SET %s
		WHERE id = $1
		RETURNING id, establishment_id, nombre,
			cod_punto_venta_mh, cod_punto_venta,
			latitude, longitude, is_portable,
			active, created_at, updated_at
	`, joinStrings(updates, ", "))

	pos := &models.PointOfSale{}
	err = database.DB.QueryRowContext(ctx, query, args...).Scan(
		&pos.ID,
		&pos.EstablishmentID,
		&pos.Nombre,
		&pos.CodPuntoVentaMH,
		&pos.CodPuntoVenta,
		&pos.Latitude,
		&pos.Longitude,
		&pos.IsPortable,
		&pos.Active,
		&pos.CreatedAt,
		&pos.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to update point of sale: %w", err)
	}

	return pos, nil
}

// UpdatePOSLocation updates only the GPS coordinates of a point of sale
func (s *EstablishmentService) UpdatePOSLocation(ctx context.Context, companyID, posID string, req *models.UpdatePOSLocationRequest) error {
	// Verify POS exists and belongs to company
	_, err := s.GetPointOfSale(ctx, companyID, posID)
	if err != nil {
		return err
	}

	query := `
		UPDATE point_of_sale
		SET latitude = $1, longitude = $2, updated_at = $3
		WHERE id = $4
	`

	result, err := database.DB.ExecContext(ctx, query, req.Latitude, req.Longitude, time.Now(), posID)
	if err != nil {
		return fmt.Errorf("failed to update POS location: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return models.ErrPointOfSaleNotFound
	}

	return nil
}

// DeactivatePointOfSale deactivates a point of sale
func (s *EstablishmentService) DeactivatePointOfSale(ctx context.Context, companyID, posID string) error {
	// Verify POS exists and belongs to company
	_, err := s.GetPointOfSale(ctx, companyID, posID)
	if err != nil {
		return err
	}

	query := `
		UPDATE point_of_sale
		SET active = false, updated_at = $1
		WHERE id = $2
	`

	result, err := database.DB.ExecContext(ctx, query, time.Now(), posID)
	if err != nil {
		return fmt.Errorf("failed to deactivate point of sale: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return models.ErrPointOfSaleNotFound
	}

	return nil
}

// Helper function to join strings
func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}
package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

type HaciendaService struct {
	db           *sql.DB
	vaultService *VaultService
	redisClient  *redis.Client
	httpClient   *http.Client
}

// HaciendaAuthResponse represents the complete authentication response from Hacienda
type HaciendaAuthResponse struct {
	Status string           `json:"status"`
	Body   HaciendaAuthBody `json:"body"`
}

type HaciendaAuthBody struct {
	User      string   `json:"user"`
	Token     string   `json:"token"`
	Rol       Role     `json:"rol"`
	Roles     []string `json:"roles"`
	TokenType string   `json:"tokenType"`
}

type Role struct {
	Nombre      string  `json:"nombre"`
	Codigo      string  `json:"codigo"`
	Descripcion *string `json:"descripcion"`
}

const (
	tokenTTL       = 12 * time.Hour
	redisKeyPrefix = "hacienda:token:"
)

// NewHaciendaService creates a new Hacienda service instance
func NewHaciendaService(db *sql.DB, vaultService *VaultService, redisClient *redis.Client) (*HaciendaService, error) {
	if db == nil {
		return nil, fmt.Errorf("database connection is required")
	}
	if vaultService == nil {
		return nil, fmt.Errorf("vault service is required")
	}
	if redisClient == nil {
		return nil, fmt.Errorf("redis client is required")
	}

	return &HaciendaService{
		db:           db,
		vaultService: vaultService,
		redisClient:  redisClient,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

// getRedisKey generates the Redis key for a company's token
func (h *HaciendaService) getRedisKey(companyID string) string {
	return fmt.Sprintf("%s%s", redisKeyPrefix, companyID)
}

// GetCachedToken retrieves a token from Redis cache
func (h *HaciendaService) GetCachedToken(ctx context.Context, companyID string) (*HaciendaAuthResponse, error) {
	key := h.getRedisKey(companyID)

	val, err := h.redisClient.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, nil // Cache miss
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get token from cache: %v", err)
	}

	var authResponse HaciendaAuthResponse
	if err := json.Unmarshal([]byte(val), &authResponse); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cached token: %v", err)
	}

	return &authResponse, nil
}

// CacheToken stores a token in Redis with 12-hour TTL
func (h *HaciendaService) CacheToken(ctx context.Context, companyID string, authResponse *HaciendaAuthResponse) error {
	key := h.getRedisKey(companyID)

	data, err := json.Marshal(authResponse)
	if err != nil {
		return fmt.Errorf("failed to marshal token for caching: %v", err)
	}

	if err := h.redisClient.Set(ctx, key, data, tokenTTL).Err(); err != nil {
		return fmt.Errorf("failed to cache token: %v", err)
	}

	return nil
}

// InvalidateToken removes a company's token from Redis cache
func (h *HaciendaService) InvalidateToken(ctx context.Context, companyID string) error {
	key := h.getRedisKey(companyID)

	if err := h.redisClient.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("failed to invalidate token: %v", err)
	}

	return nil
}

// AuthenticateCompany authenticates a company with the Hacienda API
// Checks Redis cache first, then authenticates with Hacienda if needed
func (h *HaciendaService) AuthenticateCompany(ctx context.Context, companyID string) (*HaciendaAuthResponse, error) {
	// Check Redis cache first
	cachedToken, err := h.GetCachedToken(ctx, companyID)
	if err != nil {
		// Log error but continue to fetch new token
		fmt.Printf("Warning: failed to get cached token: %v\n", err)
	}
	if cachedToken != nil {
		return cachedToken, nil
	}

	// Cache miss - authenticate with Hacienda
	// Retrieve company credentials from database
	var username, passwordRef string
	query := `
                SELECT hc_username, hc_password_ref
                FROM companies
                WHERE id = $1 AND active = true
        `

	err = h.db.QueryRowContext(ctx, query, companyID).Scan(&username, &passwordRef)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("company not found or inactive")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve company credentials: %v", err)
	}

	// Retrieve password from Vault
	password, err := h.vaultService.GetCompanyPassword(passwordRef)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve password from vault: %v", err)
	}
	fmt.Println("this is the password", password)

	// Make authentication request to Hacienda API
	authResponse, err := h.authenticateWithHacienda(username, password)
	if err != nil {
		return nil, fmt.Errorf("hacienda authentication failed: %v", err)
	}

	// Cache the token in Redis
	if err := h.CacheToken(ctx, companyID, authResponse); err != nil {
		// Log error but don't fail the request
		fmt.Printf("Warning: failed to cache token: %v\n", err)
	}

	return authResponse, nil
}

// authenticateWithHacienda makes the actual API call to Hacienda
func (h *HaciendaService) authenticateWithHacienda(username, password string) (*HaciendaAuthResponse, error) {
	// Prepare form data
	formData := url.Values{}
	formData.Set("user", username)
	formData.Set("pwd", password)

	// Create request
	req, err := http.NewRequest(
		"POST",
		"https://apitest.dtes.mh.gob.sv/seguridad/auth",
		strings.NewReader(formData.Encode()),
	)
	if err != nil {
		fmt.Println(err)
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "user")

	// Make request
	resp, err := h.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %v", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("authentication failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse JSON response
	var authResponse HaciendaAuthResponse
	if err := json.Unmarshal(body, &authResponse); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	// Verify the response status
	if authResponse.Status != "OK" {
		fmt.Println(authResponse.Status)
		fmt.Println(authResponse)
		return nil, fmt.Errorf("authentication failed: status is not OK")
	}

	return &authResponse, nil
}

// UpdateLastActivity updates the last_activity_at timestamp for a company
func (h *HaciendaService) UpdateLastActivity(ctx context.Context, companyID string) error {
	query := `
                UPDATE companies
                SET last_activity_at = CURRENT_TIMESTAMP
                WHERE id = $1
        `

	_, err := h.db.ExecContext(ctx, query, companyID)
	if err != nil {
		return fmt.Errorf("failed to update last activity: %v", err)
	}

	return nil
}
package services

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/binary"
	"fmt"
	"time"

	"cuentas/internal/codigos"
	"cuentas/internal/models"
)

type InventoryService struct {
	db *sql.DB
}

func NewInventoryService(db *sql.DB) *InventoryService {
	return &InventoryService{db: db}
}

// CreateItem creates a new inventory item with its taxes
func (s *InventoryService) CreateItem(ctx context.Context, companyID string, req *models.CreateInventoryItemRequest) (*models.InventoryItem, error) {
	// Generate SKU if not provided
	sku := ""
	if req.SKU != nil {
		sku = *req.SKU
	} else {
		var err error
		sku, err = s.generateSKU(ctx, companyID, req.TipoItem)
		if err != nil {
			return nil, fmt.Errorf("failed to generate SKU: %w", err)
		}
	}

	// Generate barcode if not provided (only for goods)
	var barcode *string
	if req.CodigoBarras != nil {
		barcode = req.CodigoBarras
	} else if req.TipoItem == "1" { // Only for Bienes
		generated := s.generateBarcode()
		barcode = &generated
	}

	// Start a transaction
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Insert the inventory item
	query := `
		INSERT INTO inventory_items (
			company_id, tipo_item, sku, codigo_barras,
			name, description, manufacturer, image_url,
			cost_price, unit_price, unit_of_measure, color
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id, company_id, tipo_item, sku, codigo_barras,
				  name, description, manufacturer, image_url,
				  cost_price, unit_price, unit_of_measure, color,
				  active, created_at, updated_at
	`

	var item models.InventoryItem
	err = tx.QueryRowContext(ctx, query,
		companyID, req.TipoItem, sku, barcode,
		req.Name, req.Description, req.Manufacturer, req.ImageURL,
		req.CostPrice, req.UnitPrice, req.UnitOfMeasure, req.Color,
	).Scan(
		&item.ID, &item.CompanyID, &item.TipoItem, &item.SKU, &item.CodigoBarras,
		&item.Name, &item.Description, &item.Manufacturer, &item.ImageURL,
		&item.CostPrice, &item.UnitPrice, &item.UnitOfMeasure, &item.Color,
		&item.Active, &item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create item: %w", err)
	}

	// If no taxes provided, add default based on tipo_item
	taxes := req.Taxes
	if len(taxes) == 0 {
		taxes = getDefaultTaxes(req.TipoItem)
	}

	// Insert taxes
	for _, tax := range taxes {
		taxQuery := `
			INSERT INTO inventory_item_taxes (item_id, tributo_code)
			VALUES ($1, $2)
		`
		_, err = tx.ExecContext(ctx, taxQuery, item.ID, tax.TributoCode)
		if err != nil {
			return nil, fmt.Errorf("failed to add tax %s: %w", tax.TributoCode, err)
		}
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Load taxes for response
	item.Taxes, err = s.GetItemTaxes(ctx, companyID, item.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to load taxes: %w", err)
	}

	return &item, nil
}

// GetItemByID retrieves an inventory item by ID
func (s *InventoryService) GetItemByID(ctx context.Context, companyID, itemID string) (*models.InventoryItem, error) {
	query := `
		SELECT id, company_id, tipo_item, sku, codigo_barras,
			   name, description, manufacturer, image_url,
			   cost_price, unit_price, unit_of_measure, color,
			   active, created_at, updated_at
		FROM inventory_items
		WHERE id = $1 AND company_id = $2
	`

	var item models.InventoryItem
	err := s.db.QueryRowContext(ctx, query, itemID, companyID).Scan(
		&item.ID, &item.CompanyID, &item.TipoItem, &item.SKU, &item.CodigoBarras,
		&item.Name, &item.Description, &item.Manufacturer, &item.ImageURL,
		&item.CostPrice, &item.UnitPrice, &item.UnitOfMeasure, &item.Color,
		&item.Active, &item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	// Load taxes
	item.Taxes, err = s.GetItemTaxes(ctx, companyID, item.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to load taxes: %w", err)
	}

	return &item, nil
}

// ListItems retrieves all inventory items for a company with optional filters
func (s *InventoryService) ListItems(ctx context.Context, companyID string, activeOnly bool, tipoItem string) ([]models.InventoryItem, error) {
	query := `
		SELECT id, company_id, tipo_item, sku, codigo_barras,
			   name, description, manufacturer, image_url,
			   cost_price, unit_price, unit_of_measure, color,
			   active, created_at, updated_at
		FROM inventory_items
		WHERE company_id = $1
	`

	args := []interface{}{companyID}
	argCount := 1

	if activeOnly {
		argCount++
		query += fmt.Sprintf(" AND active = $%d", argCount)
		args = append(args, true)
	}

	if tipoItem != "" {
		argCount++
		query += fmt.Sprintf(" AND tipo_item = $%d", argCount)
		args = append(args, tipoItem)
	}

	query += " ORDER BY name ASC"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list items: %w", err)
	}
	defer rows.Close()

	var items []models.InventoryItem
	for rows.Next() {
		var item models.InventoryItem
		err := rows.Scan(
			&item.ID, &item.CompanyID, &item.TipoItem, &item.SKU, &item.CodigoBarras,
			&item.Name, &item.Description, &item.Manufacturer, &item.ImageURL,
			&item.CostPrice, &item.UnitPrice, &item.UnitOfMeasure, &item.Color,
			&item.Active, &item.CreatedAt, &item.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan item: %w", err)
		}

		// Load taxes for each item
		item.Taxes, _ = s.GetItemTaxes(ctx, companyID, item.ID)

		items = append(items, item)
	}

	return items, nil
}

// UpdateItem updates an inventory item
func (s *InventoryService) UpdateItem(ctx context.Context, companyID, itemID string, req *models.UpdateInventoryItemRequest) (*models.InventoryItem, error) {
	// Build dynamic update query
	query := "UPDATE inventory_items SET updated_at = CURRENT_TIMESTAMP"
	args := []interface{}{}
	argCount := 0

	if req.Name != nil {
		argCount++
		query += fmt.Sprintf(", name = $%d", argCount)
		args = append(args, *req.Name)
	}
	if req.Description != nil {
		argCount++
		query += fmt.Sprintf(", description = $%d", argCount)
		args = append(args, *req.Description)
	}
	if req.Manufacturer != nil {
		argCount++
		query += fmt.Sprintf(", manufacturer = $%d", argCount)
		args = append(args, *req.Manufacturer)
	}
	if req.ImageURL != nil {
		argCount++
		query += fmt.Sprintf(", image_url = $%d", argCount)
		args = append(args, *req.ImageURL)
	}
	if req.CostPrice != nil {
		argCount++
		query += fmt.Sprintf(", cost_price = $%d", argCount)
		args = append(args, *req.CostPrice)
	}
	if req.UnitPrice != nil {
		argCount++
		query += fmt.Sprintf(", unit_price = $%d", argCount)
		args = append(args, *req.UnitPrice)
	}
	if req.UnitOfMeasure != nil {
		argCount++
		query += fmt.Sprintf(", unit_of_measure = $%d", argCount)
		args = append(args, *req.UnitOfMeasure)
	}
	if req.Color != nil {
		argCount++
		query += fmt.Sprintf(", color = $%d", argCount)
		args = append(args, *req.Color)
	}

	// Add WHERE clause
	argCount++
	query += fmt.Sprintf(" WHERE id = $%d", argCount)
	args = append(args, itemID)

	argCount++
	query += fmt.Sprintf(" AND company_id = $%d", argCount)
	args = append(args, companyID)

	result, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to update item: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("failed to check affected rows: %w", err)
	}

	if rowsAffected == 0 {
		return nil, sql.ErrNoRows
	}

	// Return updated item
	return s.GetItemByID(ctx, companyID, itemID)
}

// DeleteItem soft deletes an inventory item
func (s *InventoryService) DeleteItem(ctx context.Context, companyID, itemID string) error {
	query := `
		UPDATE inventory_items
		SET active = false, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1 AND company_id = $2
	`

	result, err := s.db.ExecContext(ctx, query, itemID, companyID)
	if err != nil {
		return fmt.Errorf("failed to delete item: %w", err)
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

// GetItemTaxes retrieves all taxes for an item
func (s *InventoryService) GetItemTaxes(ctx context.Context, companyID, itemID string) ([]models.InventoryItemTax, error) {
	query := `
		SELECT t.id, t.item_id, t.tributo_code, t.created_at
		FROM inventory_item_taxes t
		JOIN inventory_items i ON t.item_id = i.id
		WHERE t.item_id = $1 AND i.company_id = $2
		ORDER BY t.created_at ASC
	`

	rows, err := s.db.QueryContext(ctx, query, itemID, companyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get item taxes: %w", err)
	}
	defer rows.Close()

	var taxes []models.InventoryItemTax
	for rows.Next() {
		var tax models.InventoryItemTax
		err := rows.Scan(&tax.ID, &tax.ItemID, &tax.TributoCode, &tax.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan tax: %w", err)
		}
		taxes = append(taxes, tax)
	}

	return taxes, nil
}

// AddItemTax adds a tax to an item
func (s *InventoryService) AddItemTax(ctx context.Context, companyID, itemID string, req *models.AddItemTaxRequest) (*models.InventoryItemTax, error) {
	// Verify item exists and belongs to company
	_, err := s.GetItemByID(ctx, companyID, itemID)
	if err != nil {
		return nil, fmt.Errorf("item not found: %w", err)
	}

	query := `
		INSERT INTO inventory_item_taxes (item_id, tributo_code)
		VALUES ($1, $2)
		RETURNING id, item_id, tributo_code, created_at
	`

	var tax models.InventoryItemTax
	err = s.db.QueryRowContext(ctx, query, itemID, req.TributoCode).Scan(
		&tax.ID, &tax.ItemID, &tax.TributoCode, &tax.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to add tax: %w", err)
	}

	return &tax, nil
}

// RemoveItemTax removes a tax from an item
func (s *InventoryService) RemoveItemTax(ctx context.Context, companyID, itemID, tributoCode string) error {
	// Verify item exists and belongs to company
	_, err := s.GetItemByID(ctx, companyID, itemID)
	if err != nil {
		return fmt.Errorf("item not found: %w", err)
	}

	query := `
		DELETE FROM inventory_item_taxes
		WHERE item_id = $1 AND tributo_code = $2
	`

	result, err := s.db.ExecContext(ctx, query, itemID, tributoCode)
	if err != nil {
		return fmt.Errorf("failed to remove tax: %w", err)
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

// getDefaultTaxes returns default taxes based on tipo_item
func getDefaultTaxes(tipoItem string) []models.AddItemTaxRequest {
	switch tipoItem {
	case "1": // Bienes
		return []models.AddItemTaxRequest{
			{TributoCode: codigos.TributoIVA13}, // S1.20 - IVA 13%
		}
	case "2": // Servicios
		return []models.AddItemTaxRequest{
			{TributoCode: codigos.TributoIVA13}, // S1.20 - IVA 13%
		}
	default:
		return []models.AddItemTaxRequest{}
	}
}

// generateSKU creates a unique SKU for the company
func (s *InventoryService) generateSKU(ctx context.Context, companyID, tipoItem string) (string, error) {
	prefix := "PROD"
	if tipoItem == "2" {
		prefix = "SRV"
	}

	for attempts := 0; attempts < 10; attempts++ {
		timestamp := time.Now().Format("20060102")

		// Generate cryptographically secure random number
		var randomBytes [4]byte
		_, err := rand.Read(randomBytes[:])
		if err != nil {
			return "", fmt.Errorf("failed to generate random number: %w", err)
		}
		random := binary.BigEndian.Uint32(randomBytes[:]) % 10000

		sku := fmt.Sprintf("%s-%s-%04d", prefix, timestamp, random)

		exists, err := s.skuExists(ctx, companyID, sku)
		if err != nil {
			return "", err
		}
		if !exists {
			return sku, nil
		}
	}

	return "", fmt.Errorf("failed to generate unique SKU after 10 attempts")
}

// skuExists checks if a SKU already exists for the company
func (s *InventoryService) skuExists(ctx context.Context, companyID, sku string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM inventory_items WHERE company_id = $1 AND sku = $2)`
	err := s.db.QueryRowContext(ctx, query, companyID, sku).Scan(&exists)
	return exists, err
}

// generateBarcode creates an EAN-13 style barcode
func (s *InventoryService) generateBarcode() string {
	var randomBytes [8]byte
	rand.Read(randomBytes[:])
	random := binary.BigEndian.Uint64(randomBytes[:]) % 100000000000
	return fmt.Sprintf("20%011d", random)
}
package services

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"

	"cuentas/internal/codigos"
	"cuentas/internal/database"
	"cuentas/internal/models"
	"cuentas/internal/services/dte"
)

type InvoiceService struct{}

func NewInvoiceService() *InvoiceService {
	return &InvoiceService{}
}

func (s *InvoiceService) validatePointOfSale(ctx context.Context, tx *sql.Tx, companyID, establishmentID, posID string) error {
	query := `
		SELECT pos.id
		FROM point_of_sale pos
		JOIN establishments e ON pos.establishment_id = e.id
		WHERE pos.id = $1 
			AND e.id = $2 
			AND e.company_id = $3 
			AND pos.active = true 
			AND e.active = true
	`

	var id string
	err := tx.QueryRowContext(ctx, query, posID, establishmentID, companyID).Scan(&id)
	if err == sql.ErrNoRows {
		return ErrPointOfSaleNotFound
	}
	if err != nil {
		return fmt.Errorf("failed to validate point of sale: %w", err)
	}

	return nil
}

func (s *InvoiceService) CreateInvoice(ctx context.Context, companyID string, req *models.CreateInvoiceRequest) (*models.Invoice, error) {
	// Validate request
	if err := req.Validate(); err != nil {
		return nil, err
	}

	// Begin transaction
	tx, err := database.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// 1. Validate establishment and POS belong together and to company
	if err := s.validatePointOfSale(ctx, tx, companyID, req.EstablishmentID, req.PointOfSaleID); err != nil {
		return nil, err
	}

	// 2. Snapshot client data
	client, err := s.snapshotClient(ctx, tx, companyID, req.ClientID)
	if err != nil {
		return nil, err
	}

	// 3. Generate invoice number
	invoiceNumber, err := s.generateInvoiceNumber(ctx, tx, req.PointOfSaleID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate invoice number: %w", err)
	}

	// 4. Process line items and calculate totals
	lineItems, subtotal, totalDiscount, totalTaxes, err := s.processLineItems(ctx, tx, companyID, req.LineItems)
	if err != nil {
		return nil, err
	}

	total := round(subtotal - totalDiscount + totalTaxes)

	// 5. Calculate due date if needed
	var dueDate *time.Time
	if req.DueDate != nil {
		dueDate = req.DueDate
	} else if req.PaymentTerms == "net_30" {
		date := time.Now().AddDate(0, 0, 30)
		dueDate = &date
	} else if req.PaymentTerms == "net_60" {
		date := time.Now().AddDate(0, 0, 60)
		dueDate = &date
	}

	// 6. Create invoice record
	invoice := &models.Invoice{
		CompanyID:               companyID,
		EstablishmentID:         req.EstablishmentID,
		PointOfSaleID:           req.PointOfSaleID,
		ClientID:                req.ClientID,
		InvoiceNumber:           invoiceNumber,
		InvoiceType:             "sale",
		ClientName:              client.ClientName,
		ClientLegalName:         client.ClientLegalName,
		ClientNit:               client.ClientNit,
		ClientNcr:               client.ClientNcr,
		ClientDui:               client.ClientDui,
		ClientAddress:           client.ClientAddress,
		ClientTipoContribuyente: client.ClientTipoContribuyente,
		ClientTipoPersona:       client.ClientTipoPersona,
		Subtotal:                subtotal,
		TotalDiscount:           totalDiscount,
		TotalTaxes:              totalTaxes,
		Total:                   total,
		Currency:                "USD",
		PaymentMethod:           req.PaymentMethod,
		PaymentTerms:            req.PaymentTerms,
		PaymentStatus:           "unpaid",
		AmountPaid:              0,
		BalanceDue:              total,
		DueDate:                 dueDate,
		Status:                  "draft",
		Notes:                   req.Notes,
		ContactEmail:            req.ContactEmail,
		ContactWhatsapp:         req.ContactWhatsapp,
		CreatedAt:               time.Now(),
	}

	// 7. Insert invoice
	invoiceID, err := s.insertInvoice(ctx, tx, invoice)
	if err != nil {
		return nil, fmt.Errorf("failed to insert invoice: %w", err)
	}
	invoice.ID = invoiceID

	// 8. Insert line items and taxes

	for i := range lineItems {
		lineItems[i].InvoiceID = invoiceID
		lineItems[i].LineNumber = i + 1

		lineItemID, err := s.insertLineItem(ctx, tx, &lineItems[i])
		if err != nil {
			return nil, fmt.Errorf("failed to insert line item %d: %w", i+1, err)
		}
		lineItems[i].ID = lineItemID

		// Insert taxes for this line item
		for j := range lineItems[i].Taxes {
			lineItems[i].Taxes[j].LineItemID = lineItemID
			taxID, err := s.insertLineItemTax(ctx, tx, &lineItems[i].Taxes[j])
			if err != nil {
				return nil, fmt.Errorf("failed to insert tax for line item %d: %w", i+1, err)
			}
			lineItems[i].Taxes[j].ID = taxID // SET THE ID
		}
	}

	// 9. Commit transaction
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// 10. Attach line items to invoice
	invoice.LineItems = lineItems

	return invoice, nil
}

// ClientSnapshot represents the snapshot of client data at transaction time
type ClientSnapshot struct {
	ClientName              string
	ClientLegalName         string
	ClientNit               *string
	ClientNcr               *string
	ClientDui               *string
	ClientAddress           string
	ClientTipoContribuyente *string
	ClientTipoPersona       *string
}

// snapshotClient retrieves and snapshots client data
func (s *InvoiceService) snapshotClient(ctx context.Context, tx *sql.Tx, companyID, clientID string) (*ClientSnapshot, error) {
	query := `
		SELECT
			business_name,
			legal_business_name,
			nit,
			ncr,
			dui,
			full_address,
			tipo_contribuyente,
			tipo_persona
		FROM clients
		WHERE id = $1 AND company_id = $2 AND active = true
	`

	var snapshot ClientSnapshot
	err := tx.QueryRowContext(ctx, query, clientID, companyID).Scan(
		&snapshot.ClientName,
		&snapshot.ClientLegalName,
		&snapshot.ClientNit,
		&snapshot.ClientNcr,
		&snapshot.ClientDui,
		&snapshot.ClientAddress,
		&snapshot.ClientTipoContribuyente,
		&snapshot.ClientTipoPersona,
	)

	if err == sql.ErrNoRows {
		return nil, ErrClientNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query client: %w", err)
	}

	return &snapshot, nil
}

// generateInvoiceNumber generates a sequential invoice number
func (s *InvoiceService) generateInvoiceNumber(ctx context.Context, tx *sql.Tx, companyID string) (string, error) {
	// Get the last invoice number for this company
	var lastNumber sql.NullString
	query := `
		SELECT invoice_number
		FROM invoices
		WHERE company_id = $1
		ORDER BY created_at DESC
		LIMIT 1
	`
	err := tx.QueryRowContext(ctx, query, companyID).Scan(&lastNumber)
	if err != nil && err != sql.ErrNoRows {
		return "", fmt.Errorf("failed to query last invoice number: %w", err)
	}

	// Parse sequence or start at 1
	var sequence int64 = 1
	if lastNumber.Valid {
		// Format: INV-2025-00001
		// Extract the sequence number (last part)
		var year int
		fmt.Sscanf(lastNumber.String, "INV-%d-%d", &year, &sequence)
		sequence++
	}

	// Generate new number
	currentYear := time.Now().Year()
	invoiceNumber := fmt.Sprintf("INV-%d-%05d", currentYear, sequence)

	return invoiceNumber, nil
}

// processLineItems processes all line items, snapshots data, and calculates totals
// processLineItems processes all line items, snapshots data, and calculates totals
func (s *InvoiceService) processLineItems(ctx context.Context, tx *sql.Tx, companyID string, reqItems []models.CreateInvoiceLineItemRequest) ([]models.InvoiceLineItem, float64, float64, float64, error) {
	var lineItems []models.InvoiceLineItem
	var subtotal, totalDiscount, totalTaxes float64

	for _, reqItem := range reqItems {
		// 1. Snapshot inventory item
		item, err := s.snapshotInventoryItem(ctx, tx, companyID, reqItem.ItemID)
		if err != nil {
			return nil, 0, 0, 0, err
		}

		// 2. Calculate line amounts with rounding
		lineSubtotal := round(item.UnitPrice * reqItem.Quantity)
		discountAmount := round(lineSubtotal * (reqItem.DiscountPercentage / 100))
		taxableAmount := round(lineSubtotal - discountAmount)

		// 3. Get taxes for this item
		taxes, lineTaxTotal, err := s.snapshotItemTaxes(ctx, tx, reqItem.ItemID, taxableAmount)
		if err != nil {
			return nil, 0, 0, 0, err
		}

		lineTotal := round(taxableAmount + lineTaxTotal)

		// 4. Create line item
		lineItem := models.InvoiceLineItem{
			ItemID:             &reqItem.ItemID,
			ItemSku:            item.SKU,
			ItemName:           item.Name,
			ItemDescription:    item.Description,
			ItemTipoItem:       item.TipoItem,
			UnitOfMeasure:      item.UnitOfMeasure,
			UnitPrice:          item.UnitPrice,
			Quantity:           reqItem.Quantity,
			LineSubtotal:       lineSubtotal,
			DiscountPercentage: reqItem.DiscountPercentage,
			DiscountAmount:     discountAmount,
			TaxableAmount:      taxableAmount,
			TotalTaxes:         lineTaxTotal,
			LineTotal:          lineTotal,
			Taxes:              taxes,
			CreatedAt:          time.Now(),
		}

		lineItems = append(lineItems, lineItem)

		// 5. Accumulate totals
		subtotal += lineSubtotal
		totalDiscount += discountAmount
		totalTaxes += lineTaxTotal
	}

	// Round final totals
	return lineItems, round(subtotal), round(totalDiscount), round(totalTaxes), nil
}

// ItemSnapshot represents the snapshot of inventory item at transaction time
type ItemSnapshot struct {
	SKU           string
	Name          string
	Description   *string
	TipoItem      string
	UnitOfMeasure string
	UnitPrice     float64
}

// snapshotInventoryItem retrieves and snapshots inventory item data
func (s *InvoiceService) snapshotInventoryItem(ctx context.Context, tx *sql.Tx, companyID, itemID string) (*ItemSnapshot, error) {
	query := `
		SELECT
			sku,
			name,
			description,
			tipo_item,
			unit_of_measure,
			unit_price
		FROM inventory_items
		WHERE id = $1 AND company_id = $2 AND active = true
	`

	var snapshot ItemSnapshot
	err := tx.QueryRowContext(ctx, query, itemID, companyID).Scan(
		&snapshot.SKU,
		&snapshot.Name,
		&snapshot.Description,
		&snapshot.TipoItem,
		&snapshot.UnitOfMeasure,
		&snapshot.UnitPrice,
	)

	if err == sql.ErrNoRows {
		return nil, ErrInventoryItemNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query inventory item: %w", err)
	}

	return &snapshot, nil
}

// satePercent / 100retrieves taxes for an item and calculates tax amounts
func (s *InvoiceService) snapshotItemTaxes(ctx context.Context, tx *sql.Tx, itemID string, taxableBase float64) ([]models.InvoiceLineItemTax, float64, error) {
	query := `
		SELECT tributo_code
		FROM inventory_item_taxes
		WHERE item_id = $1
	`

	rows, err := tx.QueryContext(ctx, query, itemID)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query item taxes: %w", err)
	}
	defer rows.Close()

	var taxes []models.InvoiceLineItemTax
	var totalTax float64

	for rows.Next() {
		var tributoCode string

		if err := rows.Scan(&tributoCode); err != nil {
			return nil, 0, fmt.Errorf("failed to scan tax: %w", err)
		}

		// Get tax name from Go codigos package
		tributoName, exists := codigos.GetTributoName(tributoCode)
		if !exists {
			return nil, 0, fmt.Errorf("invalid tributo code: %s", tributoCode)
		}

		// For now, extract percentage from IVA tax code or default to 0
		// You can extend this with a proper tax rate lookup
		var taxRatePercent float64
		if tributoCode == codigos.TributoIVA13 {
			taxRatePercent = 13.00
		} else if tributoCode == codigos.TributoIVAExportaciones {
			taxRatePercent = 0.00
		} else {
			// Add other tax rates as needed
			taxRatePercent = 0.00
		}

		// Convert percentage to decimal (13% -> 0.13)
		taxRate := taxRatePercent / 100
		taxAmount := round(taxableBase * taxRate)

		tax := models.InvoiceLineItemTax{
			TributoCode: tributoCode,
			TributoName: tributoName,
			TaxRate:     taxRate,
			TaxableBase: taxableBase,
			TaxAmount:   taxAmount,
			CreatedAt:   time.Now(),
		}

		taxes = append(taxes, tax)
		totalTax += taxAmount
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating taxes: %w", err)
	}

	return taxes, totalTax, nil
}

// insertLineItem inserts a line item and returns the ID
func (s *InvoiceService) insertLineItemTax(ctx context.Context, tx *sql.Tx, tax *models.InvoiceLineItemTax) (string, error) {
	query := `
		INSERT INTO invoice_line_item_taxes (
			line_item_id, tributo_code, tributo_name,
			tax_rate, taxable_base, tax_amount,
			created_at
		) VALUES (
			$1, $2, $3,
			$4, $5, $6,
			$7
		) RETURNING id
	`

	var id string
	err := tx.QueryRowContext(ctx, query,
		tax.LineItemID, tax.TributoCode, tax.TributoName,
		tax.TaxRate, tax.TaxableBase, tax.TaxAmount,
		tax.CreatedAt,
	).Scan(&id)

	return id, err
}

func (s *InvoiceService) insertLineItem(ctx context.Context, tx *sql.Tx, lineItem *models.InvoiceLineItem) (string, error) {
	query := `
		INSERT INTO invoice_line_items (
			invoice_id, line_number, item_id,
			item_sku, item_name, item_description, item_tipo_item, unit_of_measure,
			unit_price, quantity, line_subtotal,
			discount_percentage, discount_amount,
			taxable_amount, total_taxes, line_total,
			created_at
		) VALUES (
			$1, $2, $3,
			$4, $5, $6, $7, $8,
			$9, $10, $11,
			$12, $13,
			$14, $15, $16,
			$17
		) RETURNING id
	`

	var id string
	err := tx.QueryRowContext(ctx, query,
		lineItem.InvoiceID, lineItem.LineNumber, lineItem.ItemID,
		lineItem.ItemSku, lineItem.ItemName, lineItem.ItemDescription, lineItem.ItemTipoItem, lineItem.UnitOfMeasure,
		lineItem.UnitPrice, lineItem.Quantity, lineItem.LineSubtotal,
		lineItem.DiscountPercentage, lineItem.DiscountAmount,
		lineItem.TaxableAmount, lineItem.TotalTaxes, lineItem.LineTotal,
		lineItem.CreatedAt,
	).Scan(&id)

	if err != nil {
		return "", err
	}

	return id, nil
}

// GetInvoice retrieves a complete invoice with line items and taxes
func (s *InvoiceService) GetInvoice(ctx context.Context, companyID, invoiceID string) (*models.Invoice, error) {
	// Get invoice header
	invoice, err := s.getInvoiceHeader(ctx, companyID, invoiceID)
	if err != nil {
		return nil, err
	}

	// Get line items
	lineItems, err := s.getLineItems(ctx, invoiceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get line items: %w", err)
	}

	// Get taxes for each line item
	for i := range lineItems {
		taxes, err := s.getLineItemTaxes(ctx, lineItems[i].ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get taxes for line item: %w", err)
		}
		lineItems[i].Taxes = taxes
	}

	invoice.LineItems = lineItems

	return invoice, nil
}

// getLineItems retrieves all line items for an invoice
func (s *InvoiceService) getLineItems(ctx context.Context, invoiceID string) ([]models.InvoiceLineItem, error) {
	query := `
		SELECT
			id, invoice_id, line_number, item_id,
			item_sku, item_name, item_description, item_tipo_item, unit_of_measure,
			unit_price, quantity, line_subtotal,
			discount_percentage, discount_amount,
			taxable_amount, total_taxes, line_total,
			created_at
		FROM invoice_line_items
		WHERE invoice_id = $1
		ORDER BY line_number
	`

	rows, err := database.DB.QueryContext(ctx, query, invoiceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lineItems []models.InvoiceLineItem
	for rows.Next() {
		var item models.InvoiceLineItem
		err := rows.Scan(
			&item.ID, &item.InvoiceID, &item.LineNumber, &item.ItemID,
			&item.ItemSku, &item.ItemName, &item.ItemDescription, &item.ItemTipoItem, &item.UnitOfMeasure,
			&item.UnitPrice, &item.Quantity, &item.LineSubtotal,
			&item.DiscountPercentage, &item.DiscountAmount,
			&item.TaxableAmount, &item.TotalTaxes, &item.LineTotal,
			&item.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		lineItems = append(lineItems, item)
	}

	return lineItems, rows.Err()
}

// getLineItemTaxes retrieves all taxes for a line item
func (s *InvoiceService) getLineItemTaxes(ctx context.Context, lineItemID string) ([]models.InvoiceLineItemTax, error) {
	query := `
		SELECT
			id, line_item_id, tributo_code, tributo_name,
			tax_rate, taxable_base, tax_amount,
			created_at
		FROM invoice_line_item_taxes
		WHERE line_item_id = $1
	`

	rows, err := database.DB.QueryContext(ctx, query, lineItemID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var taxes []models.InvoiceLineItemTax
	for rows.Next() {
		var tax models.InvoiceLineItemTax
		err := rows.Scan(
			&tax.ID, &tax.LineItemID, &tax.TributoCode, &tax.TributoName,
			&tax.TaxRate, &tax.TaxableBase, &tax.TaxAmount,
			&tax.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		taxes = append(taxes, tax)
	}

	return taxes, rows.Err()
}

// DeleteDraftInvoice deletes a draft invoice (only drafts can be deleted)
func (s *InvoiceService) DeleteDraftInvoice(ctx context.Context, companyID, invoiceID string) error {
	// Verify it's a draft
	invoice, err := s.getInvoiceHeader(ctx, companyID, invoiceID)
	if err != nil {
		return err
	}

	if invoice.Status != "draft" {
		return ErrInvoiceNotDraft
	}

	// Delete (cascade will handle line items and taxes)
	query := `DELETE FROM invoices WHERE id = $1 AND company_id = $2`
	result, err := database.DB.ExecContext(ctx, query, invoiceID, companyID)
	if err != nil {
		return fmt.Errorf("failed to delete invoice: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrInvoiceNotFound
	}

	return nil
}

func round(val float64) float64 {
	return math.Round(val*100) / 100
}

// / new
// Update insertInvoice to include establishment_id
func (s *InvoiceService) insertInvoice(ctx context.Context, tx *sql.Tx, invoice *models.Invoice) (string, error) {
	query := `
		INSERT INTO invoices (
			company_id, establishment_id, point_of_sale_id, client_id, invoice_number, invoice_type,
			client_name, client_legal_name, client_nit, client_ncr, client_dui,
			client_address, client_tipo_contribuyente, client_tipo_persona,
			subtotal, total_discount, total_taxes, total,
			currency, payment_terms, payment_method, payment_status, amount_paid, balance_due, due_date,
			status, notes, contact_email, contact_whatsapp, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6,
			$7, $8, $9, $10, $11,
			$12, $13, $14,
			$15, $16, $17, $18,
			$19, $20, $21, $22, $23, $24,
			$25, $26, $27, $28, $29, $30
		) RETURNING id
	`

	var id string
	err := tx.QueryRowContext(ctx, query,
		invoice.CompanyID, invoice.EstablishmentID, invoice.PointOfSaleID, invoice.ClientID, invoice.InvoiceNumber, invoice.InvoiceType,
		invoice.ClientName, invoice.ClientLegalName, invoice.ClientNit, invoice.ClientNcr, invoice.ClientDui,
		invoice.ClientAddress, invoice.ClientTipoContribuyente, invoice.ClientTipoPersona,
		invoice.Subtotal, invoice.TotalDiscount, invoice.TotalTaxes, invoice.Total,
		invoice.Currency, invoice.PaymentTerms, invoice.PaymentMethod, invoice.PaymentStatus, invoice.AmountPaid, invoice.BalanceDue, invoice.DueDate,
		invoice.Status, invoice.Notes, invoice.ContactEmail, invoice.ContactWhatsapp, invoice.CreatedAt,
	).Scan(&id)

	if err != nil {
		return "", err
	}

	return id, nil
}

// Update getInvoiceHeader to include establishment_id
func (s *InvoiceService) getInvoiceHeader(ctx context.Context, companyID, invoiceID string) (*models.Invoice, error) {
	query := `
		SELECT
			id, company_id, establishment_id, point_of_sale_id, client_id,
			invoice_number, invoice_type,
			references_invoice_id, void_reason,
			client_name, client_legal_name, client_nit, client_ncr, client_dui,
			client_address, client_tipo_contribuyente, client_tipo_persona,
			subtotal, total_discount, total_taxes, total,
			currency,
			payment_terms, payment_method, payment_status, amount_paid, balance_due, due_date,
			status,
			dte_codigo_generacion, dte_numero_control, dte_status, dte_hacienda_response, dte_submitted_at,
			created_at, finalized_at, voided_at,
			created_by, voided_by, notes,
			contact_email, contact_whatsapp
		FROM invoices
		WHERE id = $1 AND company_id = $2
	`

	invoice := &models.Invoice{}
	err := database.DB.QueryRowContext(ctx, query, invoiceID, companyID).Scan(
		&invoice.ID, &invoice.CompanyID, &invoice.EstablishmentID, &invoice.PointOfSaleID, &invoice.ClientID,
		&invoice.InvoiceNumber, &invoice.InvoiceType,
		&invoice.ReferencesInvoiceID, &invoice.VoidReason,
		&invoice.ClientName, &invoice.ClientLegalName, &invoice.ClientNit, &invoice.ClientNcr, &invoice.ClientDui,
		&invoice.ClientAddress, &invoice.ClientTipoContribuyente, &invoice.ClientTipoPersona,
		&invoice.Subtotal, &invoice.TotalDiscount, &invoice.TotalTaxes, &invoice.Total,
		&invoice.Currency,
		&invoice.PaymentTerms, &invoice.PaymentMethod, &invoice.PaymentStatus, &invoice.AmountPaid, &invoice.BalanceDue, &invoice.DueDate,
		&invoice.Status,
		&invoice.DteCodigoGeneracion, &invoice.DteNumeroControl, &invoice.DteStatus, &invoice.DteHaciendaResponse, &invoice.DteSubmittedAt,
		&invoice.CreatedAt, &invoice.FinalizedAt, &invoice.VoidedAt,
		&invoice.CreatedBy, &invoice.VoidedBy, &invoice.Notes,
		&invoice.ContactEmail, &invoice.ContactWhatsapp,
	)

	if err == sql.ErrNoRows {
		return nil, ErrInvoiceNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query invoice: %w", err)
	}

	return invoice, nil
}

// Update ListInvoices to include establishment_id
func (s *InvoiceService) ListInvoices(ctx context.Context, companyID string, filters map[string]interface{}) ([]models.Invoice, error) {
	query := `
		SELECT
			id, company_id, establishment_id, point_of_sale_id, client_id,
			invoice_number, invoice_type,
			client_name, client_legal_name,
			subtotal, total_discount, total_taxes, total,
			payment_terms, payment_status, amount_paid, balance_due, due_date,
			status,
			dte_status, dte_codigo_generacion, dte_numero_control,
			created_at, finalized_at,
			notes
		FROM invoices
		WHERE company_id = $1
	`

	args := []interface{}{companyID}
	argCount := 1

	// Add filters
	if status, ok := filters["status"].(string); ok && status != "" {
		argCount++
		query += fmt.Sprintf(" AND status = $%d", argCount)
		args = append(args, status)
	}

	if clientID, ok := filters["client_id"].(string); ok && clientID != "" {
		argCount++
		query += fmt.Sprintf(" AND client_id = $%d", argCount)
		args = append(args, clientID)
	}

	if paymentStatus, ok := filters["payment_status"].(string); ok && paymentStatus != "" {
		argCount++
		query += fmt.Sprintf(" AND payment_status = $%d", argCount)
		args = append(args, paymentStatus)
	}

	if establishmentID, ok := filters["establishment_id"].(string); ok && establishmentID != "" {
		argCount++
		query += fmt.Sprintf(" AND establishment_id = $%d", argCount)
		args = append(args, establishmentID)
	}

	if posID, ok := filters["point_of_sale_id"].(string); ok && posID != "" {
		argCount++
		query += fmt.Sprintf(" AND point_of_sale_id = $%d", argCount)
		args = append(args, posID)
	}

	// ADD: Filter by DTE status
	if dteStatus, ok := filters["dte_status"].(string); ok && dteStatus != "" {
		argCount++
		query += fmt.Sprintf(" AND dte_status = $%d", argCount)
		args = append(args, dteStatus)
	}

	query += " ORDER BY created_at DESC"

	rows, err := database.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var invoices []models.Invoice
	for rows.Next() {
		var inv models.Invoice
		err := rows.Scan(
			&inv.ID, &inv.CompanyID, &inv.EstablishmentID, &inv.PointOfSaleID, &inv.ClientID,
			&inv.InvoiceNumber, &inv.InvoiceType,
			&inv.ClientName, &inv.ClientLegalName,
			&inv.Subtotal, &inv.TotalDiscount, &inv.TotalTaxes, &inv.Total,
			&inv.PaymentTerms, &inv.PaymentStatus, &inv.AmountPaid, &inv.BalanceDue, &inv.DueDate,
			&inv.Status,
			&inv.DteStatus, &inv.DteCodigoGeneracion, &inv.DteNumeroControl,
			&inv.CreatedAt, &inv.FinalizedAt,
			&inv.Notes,
		)
		if err != nil {
			return nil, err
		}
		invoices = append(invoices, inv)
	}

	return invoices, rows.Err()
}

func (s *InvoiceService) determineDTEType(tipoPersona string) string {
	if tipoPersona == "2" {
		return "03" // CCF for businesses
	}
	return "01" // Factura for individuals (default)
}

func (s *InvoiceService) generateCodigoGeneracion() string {
	return strings.ToUpper(uuid.New().String())
}

// generateNumeroControl generates the DTE numero control using the validator
// generateNumeroControl generates the DTE numero control with strict validation
func (s *InvoiceService) generateNumeroControl(ctx context.Context, tx *sql.Tx, posID, tipoDte string) (string, error) {
	// Get establishment and POS codes (no COALESCE - must be set)
	query := `
		SELECT 
			e.cod_establecimiento_mh,
			pos.cod_punto_venta_mh,
			e.nombre as establishment_name,
			pos.nombre as pos_name
		FROM point_of_sale pos
		JOIN establishments e ON pos.establishment_id = e.id
		WHERE pos.id = $1
	`

	var codEstable, codPuntoVenta *string
	var establishmentName, posName string

	err := tx.QueryRowContext(ctx, query, posID).Scan(&codEstable, &codPuntoVenta, &establishmentName, &posName)
	if err != nil {
		return "", fmt.Errorf("failed to get establishment codes: %w", err)
	}

	// Strict validation: MH codes must be set
	if codEstable == nil || *codEstable == "" {
		return "", fmt.Errorf("establishment '%s' must have cod_establecimiento_mh assigned by Hacienda before finalizing invoices", establishmentName)
	}
	if codPuntoVenta == nil || *codPuntoVenta == "" {
		return "", fmt.Errorf("point of sale '%s' must have cod_punto_venta_mh assigned by Hacienda before finalizing invoices", posName)
	}

	// Validate 4-character format
	if len(*codEstable) != 4 {
		return "", fmt.Errorf("establishment '%s' cod_establecimiento_mh must be exactly 4 characters, got %d: '%s'",
			establishmentName, len(*codEstable), *codEstable)
	}
	if len(*codPuntoVenta) != 4 {
		return "", fmt.Errorf("point of sale '%s' cod_punto_venta_mh must be exactly 4 characters, got %d: '%s'",
			posName, len(*codPuntoVenta), *codPuntoVenta)
	}

	// Validate codes are alphanumeric (Hacienda uses numeric, but spec allows alphanumeric)
	if !s.isValidMHCode(*codEstable) {
		return "", fmt.Errorf("establishment '%s' cod_establecimiento_mh contains invalid characters: '%s'",
			establishmentName, *codEstable)
	}
	if !s.isValidMHCode(*codPuntoVenta) {
		return "", fmt.Errorf("point of sale '%s' cod_punto_venta_mh contains invalid characters: '%s'",
			posName, *codPuntoVenta)
	}

	// Get next sequence for this POS and tipoDte
	sequence, err := s.getAndIncrementDTESequence(ctx, tx, posID, tipoDte)
	if err != nil {
		return "", err
	}

	// Build numero control using the validator (ensures correctness)
	numeroControl, err := dte.BuildNumeroControl(tipoDte, *codEstable, *codPuntoVenta, sequence)
	if err != nil {
		return "", fmt.Errorf("failed to build numero control: %w", err)
	}

	return numeroControl, nil
}

// isValidMHCode checks if an MH code contains only alphanumeric characters
func (s *InvoiceService) isValidMHCode(code string) bool {
	// Hacienda codes are typically numeric, but spec allows alphanumeric
	for _, char := range code {
		if !((char >= '0' && char <= '9') || (char >= 'A' && char <= 'Z')) {
			return false
		}
	}
	return true
}

func (s *InvoiceService) getAndIncrementDTESequence(ctx context.Context, tx *sql.Tx, posID, tipoDte string) (int64, error) {
	// Try to get existing sequence with row lock
	var currentSeq int64
	query := `
		SELECT last_sequence
		FROM dte_sequences
		WHERE point_of_sale_id = $1 AND tipo_dte = $2
		FOR UPDATE
	`

	err := tx.QueryRowContext(ctx, query, posID, tipoDte).Scan(&currentSeq)

	if err == sql.ErrNoRows {
		// First time - insert new sequence starting at 1
		insertQuery := `
			INSERT INTO dte_sequences (point_of_sale_id, tipo_dte, last_sequence, updated_at)
			VALUES ($1, $2, 1, $3)
		`
		_, err = tx.ExecContext(ctx, insertQuery, posID, tipoDte, time.Now())
		if err != nil {
			return 0, fmt.Errorf("failed to initialize sequence: %w", err)
		}
		return 1, nil
	}

	if err != nil {
		return 0, fmt.Errorf("failed to get sequence: %w", err)
	}

	// Increment sequence
	newSeq := currentSeq + 1
	updateQuery := `
		UPDATE dte_sequences
		SET last_sequence = $1, updated_at = $2
		WHERE point_of_sale_id = $3 AND tipo_dte = $4
	`

	_, err = tx.ExecContext(ctx, updateQuery, newSeq, time.Now(), posID, tipoDte)
	if err != nil {
		return 0, fmt.Errorf("failed to increment sequence: %w", err)
	}

	return newSeq, nil
}

func (s *InvoiceService) checkCreditLimit(ctx context.Context, tx *sql.Tx, clientID string, invoiceTotal float64) error {
	query := `
		SELECT credit_limit, current_balance, credit_status
		FROM clients
		WHERE id = $1
	`

	var creditLimit, currentBalance float64
	var creditStatus string

	err := tx.QueryRowContext(ctx, query, clientID).Scan(&creditLimit, &currentBalance, &creditStatus)
	if err != nil {
		return fmt.Errorf("failed to check credit limit: %w", err)
	}

	if creditStatus == "suspended" {
		return models.ErrCreditSuspended
	}

	newBalance := currentBalance + invoiceTotal
	if newBalance > creditLimit {
		return ErrCreditLimitExceeded
	}

	return nil
}

// updateClientBalance updates the client's current balance after finalization
func (s *InvoiceService) updateClientBalance(ctx context.Context, tx *sql.Tx, clientID string, amount float64) error {
	query := `
		UPDATE clients
		SET current_balance = current_balance + $1,
		    credit_status = CASE
		        WHEN current_balance + $1 > credit_limit THEN 'over_limit'
		        ELSE 'good_standing'
		    END
		WHERE id = $2
	`

	_, err := tx.ExecContext(ctx, query, amount, clientID)
	return err
}

// FinalizeInvoice finalizes a draft invoice and generates DTE identifiers
func (s *InvoiceService) FinalizeInvoice(ctx context.Context, companyID, invoiceID, userID string, payment *models.CreatePaymentRequest) (*models.Invoice, error) {
	// Begin transaction
	tx, err := database.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// 1. Get invoice and verify it's a draft (with row lock)
	invoice, err := s.getInvoiceForUpdate(ctx, tx, companyID, invoiceID)
	if err != nil {
		return nil, err
	}

	if invoice.Status != "draft" {
		return nil, ErrInvoiceNotDraft
	}

	// 2. Check credit limit if credit transaction
	if invoice.PaymentTerms == "cuenta" || invoice.PaymentTerms == "net_30" || invoice.PaymentTerms == "net_60" {
		if err := s.checkCreditLimit(ctx, tx, invoice.ClientID, invoice.Total); err != nil {
			return nil, err
		}
	}

	// 3. Determine DTE type based on client tipo_persona
	var tipoDte string
	if invoice.ClientTipoPersona != nil {
		tipoDte = s.determineDTEType(*invoice.ClientTipoPersona)
	} else {
		tipoDte = "01" // Default to factura
	}

	// 4. Generate DTE identifiers
	codigoGeneracion := s.generateCodigoGeneracion()
	numeroControl, err := s.generateNumeroControl(ctx, tx, invoice.PointOfSaleID, tipoDte)
	if err != nil {
		return nil, fmt.Errorf("failed to generate numero control: %w", err)
	}

	// 5. Calculate payment status based on amount paid
	paymentStatus := s.calculatePaymentStatus(payment.Amount, invoice.Total)
	balanceDue := invoice.Total - payment.Amount

	// 6. Update invoice to finalized
	now := time.Now()

	updateQuery := `
		UPDATE invoices
		SET status = 'finalized',
		    payment_method = $1,
		    payment_status = $2,
		    amount_paid = $3,
		    balance_due = $4,
		    dte_codigo_generacion = $5,
		    dte_numero_control = $6,
		    dte_status = 'not_submitted',
		    finalized_at = $7,
		    created_by = $8
		WHERE id = $9 AND company_id = $10
	`

	_, err = tx.ExecContext(ctx, updateQuery,
		payment.PaymentMethod,
		paymentStatus,
		payment.Amount,
		balanceDue,
		codigoGeneracion,
		numeroControl,
		now,
		userID,
		invoiceID,
		companyID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update invoice: %w", err)
	}

	// 7. Record the payment in payments table
	paymentID := uuid.New().String()
	paymentDate := now
	if payment.PaymentDate != nil {
		paymentDate = *payment.PaymentDate
	}

	insertPaymentQuery := `
		INSERT INTO payments (
			id, company_id, invoice_id, amount, payment_method, 
			payment_reference, payment_date, created_by, notes
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err = tx.ExecContext(ctx, insertPaymentQuery,
		paymentID,
		companyID,
		invoiceID,
		payment.Amount,
		payment.PaymentMethod,
		payment.ReferenceNumber,
		paymentDate,
		userID,
		payment.Notes,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to record payment: %w", err)
	}

	// 8. Update client balance if credit
	if invoice.PaymentTerms == "cuenta" || invoice.PaymentTerms == "net_30" || invoice.PaymentTerms == "net_60" {
		// Add the balance due (not total) to client's balance
		if err := s.updateClientBalance(ctx, tx, invoice.ClientID, balanceDue); err != nil {
			return nil, fmt.Errorf("failed to update client balance: %w", err)
		}
	}

	// 9. Commit transaction
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// 10. Get and return the finalized invoice
	finalizedInvoice, err := s.GetInvoice(ctx, companyID, invoiceID)
	if err != nil {
		return nil, err
	}

	return finalizedInvoice, nil
}

// Helper: Calculate payment status based on amount paid
func (s *InvoiceService) calculatePaymentStatus(amountPaid, total float64) string {
	if amountPaid == 0 {
		return "unpaid"
	} else if amountPaid >= total {
		return "paid"
	}
	return "partial"
}

// getInvoiceForUpdate gets an invoice with a row lock for safe concurrent updates
func (s *InvoiceService) getInvoiceForUpdate(ctx context.Context, tx *sql.Tx, companyID, invoiceID string) (*models.Invoice, error) {
	query := `
		SELECT
			id, company_id, establishment_id, point_of_sale_id, client_id,
			invoice_number, invoice_type, status,
			client_name, client_legal_name, client_nit, client_ncr, client_dui,
			client_address, client_tipo_contribuyente, client_tipo_persona,
			subtotal, total_discount, total_taxes, total,
			currency, payment_terms, payment_method, payment_status, 
			amount_paid, balance_due, due_date,
			created_at
		FROM invoices
		WHERE id = $1 AND company_id = $2
		FOR UPDATE
	`

	invoice := &models.Invoice{}
	err := tx.QueryRowContext(ctx, query, invoiceID, companyID).Scan(
		&invoice.ID, &invoice.CompanyID, &invoice.EstablishmentID, &invoice.PointOfSaleID, &invoice.ClientID,
		&invoice.InvoiceNumber, &invoice.InvoiceType, &invoice.Status,
		&invoice.ClientName, &invoice.ClientLegalName, &invoice.ClientNit, &invoice.ClientNcr, &invoice.ClientDui,
		&invoice.ClientAddress, &invoice.ClientTipoContribuyente, &invoice.ClientTipoPersona,
		&invoice.Subtotal, &invoice.TotalDiscount, &invoice.TotalTaxes, &invoice.Total,
		&invoice.Currency, &invoice.PaymentTerms, &invoice.PaymentMethod, &invoice.PaymentStatus,
		&invoice.AmountPaid, &invoice.BalanceDue, &invoice.DueDate,
		&invoice.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrInvoiceNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query invoice: %w", err)
	}

	return invoice, nil
}
package services

import (
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/vault/api"
	"github.com/spf13/viper"
)

type VaultService struct {
	client *api.Client
}

// NewVaultService creates a new Vault service instance
func NewVaultService() (*VaultService, error) {
	// Configure Vault client
	config := api.DefaultConfig()
	config.Address = viper.GetString("vault_url")

	client, err := api.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Vault client: %v", err)
	}

	// Set token
	token := viper.GetString("vault_token")
	if token == "" {
		return nil, fmt.Errorf("vault token is required")
	}
	client.SetToken(token)

	// Test connection
	if err := testVaultConnection(client); err != nil {
		return nil, fmt.Errorf("failed to connect to Vault: %v", err)
	}

	vs := &VaultService{client: client}

	// Ensure KV v2 engine is enabled
	if err := vs.ensureKVEngine(); err != nil {
		return nil, fmt.Errorf("failed to setup KV engine: %v", err)
	}

	return vs, nil
}

// testVaultConnection verifies that we can connect to Vault
func testVaultConnection(client *api.Client) error {
	auth := client.Auth().Token()
	_, err := auth.LookupSelf()
	if err != nil {
		return fmt.Errorf("vault connection test failed: %v", err)
	}
	return nil
}

// ensureKVEngine ensures the KV v2 secrets engine is enabled at secret/
func (vs *VaultService) ensureKVEngine() error {
	sys := vs.client.Sys()

	// Check if secret/ mount exists
	mounts, err := sys.ListMounts()
	if err != nil {
		return fmt.Errorf("failed to list mounts: %v", err)
	}

	if mount, exists := mounts["secret/"]; exists {
		// Verify it's KV v2
		if mount.Type == "kv" && mount.Options["version"] == "2" {
			return nil // Already properly configured
		}
		log.Println("Warning: secret/ mount exists but is not KV v2")
	}

	return nil // In dev mode, secret/ should already be KV v2
}

// StoreCompanyPassword stores a company's password in Vault
// Returns the Vault reference path to store in the database
func (vs *VaultService) StoreCompanyPassword(companyID string, password string) (string, error) {
	// KV v2 requires /data/ in the path for writes
	path := fmt.Sprintf("secret/data/companies/%s/password", companyID)

	secretData := map[string]interface{}{
		"data": map[string]interface{}{
			"password": password,
		},
	}

	_, err := vs.client.Logical().Write(path, secretData)
	if err != nil {
		return "", fmt.Errorf("failed to store password for company %s: %v", companyID, err)
	}

	// Return the reference path (without /data/) to store as reference in database
	vaultRef := fmt.Sprintf("secret/companies/%s/password", companyID)
	return vaultRef, nil
}

// GetCompanyPassword retrieves a company's password from Vault using the reference path
func (vs *VaultService) GetCompanyPassword(vaultRef string) (string, error) {
	// KV v2 requires /data/ in the path for reads
	// Convert reference path to actual path
	readPath := fmt.Sprintf("secret/data/%s", vaultRef[7:]) // Remove "secret/" prefix and add back with /data/

	secret, err := vs.client.Logical().Read(readPath)
	if err != nil {
		return "", fmt.Errorf("failed to read secret from %s: %v", vaultRef, err)
	}

	if secret == nil {
		return "", fmt.Errorf("secret not found at %s", vaultRef)
	}

	// Extract password from KV v2 structure
	data, ok := secret.Data["data"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("invalid secret format at %s", vaultRef)
	}

	password, ok := data["password"].(string)
	if !ok {
		return "", fmt.Errorf("password not found in secret at %s", vaultRef)
	}

	return password, nil
}

// UpdateCompanyPassword updates an existing company password in Vault
func (vs *VaultService) UpdateCompanyPassword(vaultRef string, newPassword string) error {
	// KV v2 requires /data/ in the path
	writePath := fmt.Sprintf("secret/data/%s", vaultRef[7:])

	secretData := map[string]interface{}{
		"data": map[string]interface{}{
			"password": newPassword,
		},
	}

	_, err := vs.client.Logical().Write(writePath, secretData)
	if err != nil {
		return fmt.Errorf("failed to update password at %s: %v", vaultRef, err)
	}

	return nil
}

// DeleteCompanyPassword removes a company's password from Vault
func (vs *VaultService) DeleteCompanyPassword(vaultRef string) error {
	// KV v2 uses /data/ for delete as well
	deletePath := fmt.Sprintf("secret/data/%s", vaultRef[7:])

	_, err := vs.client.Logical().Delete(deletePath)
	if err != nil {
		return fmt.Errorf("failed to delete secret at %s: %v", vaultRef, err)
	}

	return nil
}

// WaitForVault waits for Vault to be available with retries
func WaitForVault(maxRetries int) error {
	for i := 0; i < maxRetries; i++ {
		_, err := NewVaultService()
		if err == nil {
			log.Println("Successfully connected to Vault")
			return nil
		}

		log.Printf("Vault connection attempt %d/%d failed: %v", i+1, maxRetries, err)
		if i < maxRetries-1 {
			time.Sleep(time.Duration(i+1) * 2 * time.Second)
		}
	}

	return fmt.Errorf("failed to connect to Vault after %d attempts", maxRetries)
}
