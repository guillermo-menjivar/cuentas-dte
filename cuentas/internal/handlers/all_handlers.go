// File: handlers/actividad_economica_handler.go
package handlers

import (
	"net/http"
	"strconv"

	"cuentas/internal/services"

	"github.com/gin-gonic/gin"
)

type ActividadEconomicaHandler struct {
	actividadService *services.ActividadEconomicaService
}

func NewActividadEconomicaHandler() *ActividadEconomicaHandler {
	return &ActividadEconomicaHandler{
		actividadService: services.NewActividadEconomicaService(),
	}
}

// GetCategories handles GET /api/v1/actividades-economicas/categories
func (h *ActividadEconomicaHandler) GetCategories(c *gin.Context) {
	categories := h.actividadService.GetAllCategories()

	c.JSON(http.StatusOK, gin.H{
		"categories": categories,
		"count":      len(categories),
	})
}

// GetCategoryByCode handles GET /api/v1/actividades-economicas/categories/:code
func (h *ActividadEconomicaHandler) GetCategoryByCode(c *gin.Context) {
	code := c.Param("code")
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "code is required"})
		return
	}

	category, exists := h.actividadService.GetCategoryByCode(code)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "category not found"})
		return
	}

	c.JSON(http.StatusOK, category)
}

// GetActivitiesByCategory handles GET /api/v1/actividades-economicas/categories/:code/activities
func (h *ActividadEconomicaHandler) GetActivitiesByCategory(c *gin.Context) {
	code := c.Param("code")
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "code is required"})
		return
	}

	activities := h.actividadService.GetActivitiesByCategory(code)

	c.JSON(http.StatusOK, gin.H{
		"category_code": code,
		"activities":    activities,
		"count":         len(activities),
	})
}

// SearchActivities handles GET /api/v1/actividades-economicas/search
func (h *ActividadEconomicaHandler) SearchActivities(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "query parameter 'q' is required"})
		return
	}

	// Get limit from query params, default to 10
	limitStr := c.DefaultQuery("limit", "10")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 10
	}
	if limit > 50 {
		limit = 50 // Cap at 50 results
	}

	results := h.actividadService.SearchActivities(query, limit)

	c.JSON(http.StatusOK, gin.H{
		"query":   query,
		"results": results,
		"count":   len(results),
	})
}

// GetActivityDetails handles GET /api/v1/actividades-economicas/:code
func (h *ActividadEconomicaHandler) GetActivityDetails(c *gin.Context) {
	code := c.Param("code")
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "code is required"})
		return
	}

	details, exists := h.actividadService.GetActivityDetails(code)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "activity not found"})
		return
	}

	c.JSON(http.StatusOK, details)
}
package handlers

import (
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"cuentas/internal/models"
	"cuentas/internal/services"

	"github.com/gin-gonic/gin"
)

// parseClientJSONError converts technical JSON errors to user-friendly messages for client creation
func parseClientJSONError(err error) string {
	errMsg := err.Error()

	// Handle type mismatch errors for client fields
	if strings.Contains(errMsg, "cannot unmarshal number") && strings.Contains(errMsg, "nit") {
		return "nit must be a text value with dashes (e.g., \"0614-123456-001-2\")"
	}
	if strings.Contains(errMsg, "cannot unmarshal number") && strings.Contains(errMsg, "ncr") {
		return "ncr must be a text value with dashes (e.g., \"12345-6\")"
	}
	if strings.Contains(errMsg, "cannot unmarshal number") && strings.Contains(errMsg, "dui") {
		return "dui must be a text value with dashes (e.g., \"12345678-9\")"
	}

	// Handle missing required fields
	if strings.Contains(errMsg, "required") {
		return "missing required fields in request"
	}

	// Generic fallback
	return "invalid request format - please check your input"
}

// CreateClientHandler handles POST /v1/clients
func CreateClientHandler(c *gin.Context) {
	// Get company_id from context (set by auth middleware)
	companyID, exists := c.Get("company_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error: "company_id not found in context",
			Code:  "unauthorized",
		})
		return
	}

	// Read request body
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: "failed to read request body",
			Code:  "invalid_request",
		})
		return
	}

	// Parse request
	var req models.CreateClientRequest
	if err := json.Unmarshal(bodyBytes, &req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: parseClientJSONError(err),
			Code:  "invalid_request",
		})
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: err.Error(),
			Code:  "validation_failed",
		})
		return
	}

	// Get database from context
	dbInterface, exists := c.Get("db")
	if !exists {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "database connection not available",
			Code:  "internal_error",
		})
		return
	}
	db := dbInterface.(*sql.DB)

	// Create client service and execute
	clientService := services.NewClientService(db)
	client, err := clientService.CreateClient(c.Request.Context(), companyID.(string), &req)
	if err != nil {
		if strings.Contains(err.Error(), "unique_client_nit_per_company") {
			c.JSON(http.StatusConflict, models.ErrorResponse{
				Error: "a client with this NIT already exists for this company",
				Code:  "duplicate_nit",
			})
			return
		}
		if strings.Contains(err.Error(), "unique_client_dui_per_company") {
			c.JSON(http.StatusConflict, models.ErrorResponse{
				Error: "a client with this DUI already exists for this company",
				Code:  "duplicate_dui",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: err.Error(),
			Code:  "internal_error",
		})
		return
	}

	c.JSON(http.StatusCreated, client)
}

// GetClientHandler handles GET /v1/clients/:id
func GetClientHandler(c *gin.Context) {
	clientID := c.Param("id")

	// Get company_id from context
	companyID, exists := c.Get("company_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error: "company_id not found in context",
			Code:  "unauthorized",
		})
		return
	}

	// Get database from context
	dbInterface, exists := c.Get("db")
	if !exists {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "database connection not available",
			Code:  "internal_error",
		})
		return
	}
	db := dbInterface.(*sql.DB)

	// Get client
	clientService := services.NewClientService(db)
	client, err := clientService.GetClientByID(c.Request.Context(), companyID.(string), clientID)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error: "client not found",
				Code:  "not_found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: err.Error(),
			Code:  "internal_error",
		})
		return
	}

	c.JSON(http.StatusOK, client)
}

// ListClientsHandler handles GET /v1/clients
func ListClientsHandler(c *gin.Context) {
	// Get company_id from context
	companyID, exists := c.Get("company_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error: "company_id not found in context",
			Code:  "unauthorized",
		})
		return
	}

	// Get database from context
	dbInterface, exists := c.Get("db")
	if !exists {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "database connection not available",
			Code:  "internal_error",
		})
		return
	}
	db := dbInterface.(*sql.DB)

	// Parse query parameters
	activeOnly := c.DefaultQuery("active", "true") == "true"

	// List clients
	clientService := services.NewClientService(db)
	clients, err := clientService.ListClients(c.Request.Context(), companyID.(string), activeOnly)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: err.Error(),
			Code:  "internal_error",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"clients": clients,
		"count":   len(clients),
	})
}

// UpdateClientHandler handles PUT /v1/clients/:id
func UpdateClientHandler(c *gin.Context) {
	clientID := c.Param("id")

	// Get company_id from context
	companyID, exists := c.Get("company_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error: "company_id not found in context",
			Code:  "unauthorized",
		})
		return
	}

	// Read request body
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: "failed to read request body",
			Code:  "invalid_request",
		})
		return
	}

	// Parse request
	var req models.CreateClientRequest
	if err := json.Unmarshal(bodyBytes, &req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: parseClientJSONError(err),
			Code:  "invalid_request",
		})
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: err.Error(),
			Code:  "validation_failed",
		})
		return
	}

	// Get database from context
	dbInterface, exists := c.Get("db")
	if !exists {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "database connection not available",
			Code:  "internal_error",
		})
		return
	}
	db := dbInterface.(*sql.DB)

	// Update client
	clientService := services.NewClientService(db)
	client, err := clientService.UpdateClient(c.Request.Context(), companyID.(string), clientID, &req)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error: "client not found",
				Code:  "not_found",
			})
			return
		}

		if strings.Contains(err.Error(), "unique_client_nit_per_company") {
			c.JSON(http.StatusConflict, models.ErrorResponse{
				Error: "a client with this NIT already exists for this company",
				Code:  "duplicate_nit",
			})
			return
		}
		if strings.Contains(err.Error(), "unique_client_dui_per_company") {
			c.JSON(http.StatusConflict, models.ErrorResponse{
				Error: "a client with this DUI already exists for this company",
				Code:  "duplicate_dui",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: err.Error(),
			Code:  "internal_error",
		})
		return
	}

	c.JSON(http.StatusOK, client)
}

// DeleteClientHandler handles DELETE /v1/clients/:id (soft delete)
func DeleteClientHandler(c *gin.Context) {
	clientID := c.Param("id")

	// Get company_id from context
	companyID, exists := c.Get("company_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error: "company_id not found in context",
			Code:  "unauthorized",
		})
		return
	}

	// Get database from context
	dbInterface, exists := c.Get("db")
	if !exists {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "database connection not available",
			Code:  "internal_error",
		})
		return
	}
	db := dbInterface.(*sql.DB)

	// Delete (deactivate) client
	clientService := services.NewClientService(db)
	err := clientService.DeleteClient(c.Request.Context(), companyID.(string), clientID)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error: "client not found",
				Code:  "not_found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: err.Error(),
			Code:  "internal_error",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "client deactivated successfully",
	})
}
package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"time"

	"cuentas/internal/database"
	"cuentas/internal/models"
	"cuentas/internal/services"

	"github.com/gin-gonic/gin"
)

// parseJSONError converts technical JSON errors to user-friendly messages
func parseJSONError(err error) string {
	errMsg := err.Error()

	print(errMsg)

	// Handle type mismatch errors
	if strings.Contains(errMsg, "cannot unmarshal number") && strings.Contains(errMsg, "nit") {
		return "nit must be a text value with dashes (e.g., \"0614-123456-001-2\")"
	}
	if strings.Contains(errMsg, "cannot unmarshal number") && strings.Contains(errMsg, "ncr") {
		return "ncr must be a text value with dashes (e.g., \"12345-6\")"
	}

	// Handle missing required fields
	if strings.Contains(errMsg, "required") {
		return "missing required fields in request"
	}

	// Generic fallback
	return "invalid request format - please check your input"
}

func CreateCompanyHandler(c *gin.Context) {

	// Parse request body with custom error handling
	var req models.CreateCompanyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fmt.Println(err)
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: parseJSONError(err),
			Code:  "invalid_request",
		})
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: err.Error(),
			Code:  "validation_failed",
		})
		return
	}

	// Get database from context
	dbInterface, exists := c.Get("db")
	if !exists {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "database connection not available",
			Code:  "internal_error",
		})
		return
	}
	db := dbInterface.(*sql.DB)

	// Get Vault service from context
	vaultServiceInterface, exists := c.Get("vaultService")
	if !exists {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "vault service not available",
			Code:  "internal_error",
		})
		return
	}
	vaultService := vaultServiceInterface.(*services.VaultService)

	// Create company service with BOTH db and vaultService
	companyService, err := services.NewCompanyService(db, vaultService)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: fmt.Sprintf("failed to initialize service: %v", err),
			Code:  "internal_error",
		})
		return
	}

	// Create company
	company, err := companyService.CreateCompany(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: err.Error(),
			Code:  "internal_error",
		})
		return
	}

	c.JSON(http.StatusCreated, company)
}

// GetCompanyHandler handles GET /v1/companies/:id
func GetCompanyHandler(c *gin.Context) {
	companyID := c.Param("id")

	dbInterface, exists := c.Get("db")
	if !exists {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "database connection not available",
			Code:  "internal_error",
		})
		return
	}
	db := dbInterface.(*sql.DB)

	// Get Vault service from context
	vaultServiceInterface, exists := c.Get("vaultService")
	if !exists {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "vault service not available",
			Code:  "internal_error",
		})
		return
	}
	vaultService := vaultServiceInterface.(*services.VaultService)

	// Create company service
	companyService, err := services.NewCompanyService(db, vaultService)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "vault service not available",
			Code:  "internal_error",
		})
		return
	}

	// Get company
	company, err := companyService.GetCompanyByID(c.Request.Context(), companyID)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Error: err.Error(),
			Code:  "not_found",
		})
		return
	}

	c.JSON(http.StatusOK, company)
}

func ListCompaniesHandler(c *gin.Context) {
	query := `
		SELECT id, name, nit, ncr, email, active, created_at, updated_at
		FROM companies
		ORDER BY created_at DESC
	`

	rows, err := database.DB.QueryContext(c.Request.Context(), query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query companies"})
		return
	}
	defer rows.Close()

	var companies []map[string]interface{}
	for rows.Next() {
		var id, name, nit, ncr, email string
		var active bool
		var createdAt, updatedAt time.Time

		if err := rows.Scan(&id, &name, &nit, &ncr, &email, &active, &createdAt, &updatedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to scan company"})
			return
		}

		companies = append(companies, map[string]interface{}{
			"id":         id,
			"name":       name,
			"nit":        nit,
			"ncr":        ncr,
			"email":      email,
			"active":     active,
			"created_at": createdAt,
			"updated_at": updatedAt,
		})
	}

	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error iterating companies"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"companies": companies,
		"count":     len(companies),
	})
}
package handlers

import (
	"net/http"

	"cuentas/internal/models"
	"cuentas/internal/services"

	"github.com/gin-gonic/gin"
)

type EstablishmentHandler struct {
	service *services.EstablishmentService
}

func NewEstablishmentHandler() *EstablishmentHandler {
	return &EstablishmentHandler{
		service: services.NewEstablishmentService(),
	}
}

// CreateEstablishment creates a new establishment
// POST /v1/establishments
func (h *EstablishmentHandler) CreateEstablishment(c *gin.Context) {
	companyID := c.GetString("company_id")
	if companyID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "company_id not found in context"})
		return
	}

	var req models.CreateEstablishmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	establishment, err := h.service.CreateEstablishment(c.Request.Context(), companyID, &req)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, establishment)
}

// GetEstablishment retrieves an establishment by ID
// GET /v1/establishments/:id
func (h *EstablishmentHandler) GetEstablishment(c *gin.Context) {
	companyID := c.GetString("company_id")
	if companyID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "company_id not found in context"})
		return
	}

	establishmentID := c.Param("id")
	if establishmentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "establishment id is required"})
		return
	}

	establishment, err := h.service.GetEstablishment(c.Request.Context(), companyID, establishmentID)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, establishment)
}

// ListEstablishments lists all establishments for a company
// GET /v1/establishments
func (h *EstablishmentHandler) ListEstablishments(c *gin.Context) {
	companyID := c.GetString("company_id")
	if companyID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "company_id not found in context"})
		return
	}

	// Check for active_only query parameter
	activeOnly := c.DefaultQuery("active_only", "true") == "true"

	establishments, err := h.service.ListEstablishments(c.Request.Context(), companyID, activeOnly)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"establishments": establishments,
		"count":          len(establishments),
	})
}

// UpdateEstablishment updates an establishment
// PATCH /v1/establishments/:id
func (h *EstablishmentHandler) UpdateEstablishment(c *gin.Context) {
	companyID := c.GetString("company_id")
	if companyID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "company_id not found in context"})
		return
	}

	establishmentID := c.Param("id")
	if establishmentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "establishment id is required"})
		return
	}

	var req models.UpdateEstablishmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	establishment, err := h.service.UpdateEstablishment(c.Request.Context(), companyID, establishmentID, &req)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, establishment)
}

// DeactivateEstablishment deactivates an establishment
// DELETE /v1/establishments/:id
func (h *EstablishmentHandler) DeactivateEstablishment(c *gin.Context) {
	companyID := c.GetString("company_id")
	if companyID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "company_id not found in context"})
		return
	}

	establishmentID := c.Param("id")
	if establishmentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "establishment id is required"})
		return
	}

	err := h.service.DeactivateEstablishment(c.Request.Context(), companyID, establishmentID)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "establishment deactivated successfully"})
}

// CreatePointOfSale creates a new point of sale for an establishment
// POST /v1/establishments/:id/pos
func (h *EstablishmentHandler) CreatePointOfSale(c *gin.Context) {
	companyID := c.GetString("company_id")
	if companyID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "company_id not found in context"})
		return
	}

	establishmentID := c.Param("id")
	if establishmentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "establishment id is required"})
		return
	}

	var req models.CreatePOSRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	pos, err := h.service.CreatePointOfSale(c.Request.Context(), companyID, establishmentID, &req)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, pos)
}

// GetPointOfSale retrieves a point of sale by ID
// GET /v1/pos/:id
func (h *EstablishmentHandler) GetPointOfSale(c *gin.Context) {
	companyID := c.GetString("company_id")
	if companyID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "company_id not found in context"})
		return
	}

	posID := c.Param("id")
	if posID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "pos id is required"})
		return
	}

	pos, err := h.service.GetPointOfSale(c.Request.Context(), companyID, posID)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, pos)
}

// ListPointsOfSale lists all points of sale for an establishment
// GET /v1/establishments/:id/pos
func (h *EstablishmentHandler) ListPointsOfSale(c *gin.Context) {
	companyID := c.GetString("company_id")
	if companyID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "company_id not found in context"})
		return
	}

	establishmentID := c.Param("id")
	if establishmentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "establishment id is required"})
		return
	}

	// Check for active_only query parameter
	activeOnly := c.DefaultQuery("active_only", "true") == "true"

	pointsOfSale, err := h.service.ListPointsOfSale(c.Request.Context(), companyID, establishmentID, activeOnly)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"points_of_sale": pointsOfSale,
		"count":          len(pointsOfSale),
	})
}

// UpdatePointOfSale updates a point of sale
// PATCH /v1/pos/:id
func (h *EstablishmentHandler) UpdatePointOfSale(c *gin.Context) {
	companyID := c.GetString("company_id")
	if companyID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "company_id not found in context"})
		return
	}

	posID := c.Param("id")
	if posID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "pos id is required"})
		return
	}

	var req models.UpdatePOSRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	pos, err := h.service.UpdatePointOfSale(c.Request.Context(), companyID, posID, &req)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, pos)
}

// UpdatePOSLocation updates the GPS location of a point of sale
// PATCH /v1/pos/:id/location
func (h *EstablishmentHandler) UpdatePOSLocation(c *gin.Context) {
	companyID := c.GetString("company_id")
	if companyID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "company_id not found in context"})
		return
	}

	posID := c.Param("id")
	if posID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "pos id is required"})
		return
	}

	var req models.UpdatePOSLocationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.service.UpdatePOSLocation(c.Request.Context(), companyID, posID, &req)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "location updated successfully"})
}

// DeactivatePointOfSale deactivates a point of sale
// DELETE /v1/pos/:id
func (h *EstablishmentHandler) DeactivatePointOfSale(c *gin.Context) {
	companyID := c.GetString("company_id")
	if companyID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "company_id not found in context"})
		return
	}

	posID := c.Param("id")
	if posID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "pos id is required"})
		return
	}

	err := h.service.DeactivatePointOfSale(c.Request.Context(), companyID, posID)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "point of sale deactivated successfully"})
}

// Helper function to handle errors consistently
func handleError(c *gin.Context, err error) {
	switch err {
	case models.ErrEstablishmentNotFound:
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case models.ErrPointOfSaleNotFound:
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case models.ErrInvalidTipoEstablecimiento:
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case models.ErrInvalidDepartamento:
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case models.ErrInvalidMunicipio:
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case models.ErrInvalidCodEstablecimientoMH:
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case models.ErrInvalidCodEstablecimiento:
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case models.ErrInvalidCodPuntoVentaMH:
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case models.ErrInvalidCodPuntoVenta:
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case models.ErrInvalidLatitude:
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case models.ErrInvalidLongitude:
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
	}
}
package handlers

import (
	"database/sql"
	"net/http"

	"cuentas/internal/models"
	"cuentas/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// AuthenticateCompanyHandler handles POST /v1/companies/:id/authenticate
func AuthenticateCompanyHandler(c *gin.Context) {
	companyID := c.Param("id")

	// Get database from context
	dbInterface, exists := c.Get("db")
	if !exists {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "database connection not available",
			Code:  "internal_error",
		})
		return
	}
	db := dbInterface.(*sql.DB)

	// Get Vault service from context
	vaultServiceInterface, exists := c.Get("vaultService")
	if !exists {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "vault service not available",
			Code:  "internal_error",
		})
		return
	}
	vaultService := vaultServiceInterface.(*services.VaultService)

	// Get Redis client from context
	redisInterface, exists := c.Get("redis")
	if !exists {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "redis connection not available",
			Code:  "internal_error",
		})
		return
	}
	redisClient := redisInterface.(*redis.Client)

	// Create Hacienda service
	haciendaService, err := services.NewHaciendaService(db, vaultService, redisClient)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "failed to initialize hacienda service",
			Code:  "internal_error",
		})
		return
	}

	// Authenticate with Hacienda (service handles caching internally)
	authResponse, err := haciendaService.AuthenticateCompany(c.Request.Context(), companyID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error: err.Error(),
			Code:  "authentication_failed",
		})
		return
	}

	// Update last activity
	if err := haciendaService.UpdateLastActivity(c.Request.Context(), companyID); err != nil {
		c.Writer.Header().Set("X-Activity-Update-Failed", "true")
	}

	c.JSON(http.StatusOK, gin.H{
		"company_id":     companyID,
		"authentication": authResponse,
	})
}

// InvalidateTokenHandler handles DELETE /v1/companies/:id/token
func InvalidateTokenHandler(c *gin.Context) {
	companyID := c.Param("id")

	// Get database from context
	dbInterface, exists := c.Get("db")
	if !exists {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "database connection not available",
			Code:  "internal_error",
		})
		return
	}
	db := dbInterface.(*sql.DB)

	// Get Vault service from context
	vaultServiceInterface, exists := c.Get("vaultService")
	if !exists {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "vault service not available",
			Code:  "internal_error",
		})
		return
	}
	vaultService := vaultServiceInterface.(*services.VaultService)

	// Get Redis client from context
	redisInterface, exists := c.Get("redis")
	if !exists {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "redis connection not available",
			Code:  "internal_error",
		})
		return
	}
	redisClient := redisInterface.(*redis.Client)

	// Create Hacienda service
	haciendaService, err := services.NewHaciendaService(db, vaultService, redisClient)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "failed to initialize hacienda service",
			Code:  "internal_error",
		})
		return
	}

	// Service handles cache invalidation internally
	if err := haciendaService.InvalidateToken(c.Request.Context(), companyID); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: err.Error(),
			Code:  "internal_error",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    "token invalidated successfully",
		"company_id": companyID,
	})
}
package handlers

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
)

// HealthResponse represents the health check response
type HealthResponse struct {
	Status   string            `json:"status"`
	Message  string            `json:"message"`
	Services map[string]string `json:"services"`
}

// HealthHandler handles the health check endpoint
func HealthHandler(c *gin.Context) {
	response := HealthResponse{
		Status:   "ok",
		Message:  "Cuentas API is healthy",
		Services: make(map[string]string),
	}

	// Check Vault status
	vaultService, exists := c.Get("vaultService")
	if exists && vaultService != nil {
		response.Services["vault"] = "healthy"
	} else {
		response.Services["vault"] = "unavailable"
		response.Status = "degraded"
	}

	// Check Postgres status
	databaseURL := viper.GetString("database_url")
	if databaseURL != "" {
		db, err := sql.Open("postgres", databaseURL)
		if err != nil {
			response.Services["postgres"] = "connection_failed"
			response.Status = "degraded"
		} else {
			defer db.Close()
			if err := db.Ping(); err != nil {
				response.Services["postgres"] = "ping_failed"
				response.Status = "degraded"
			} else {
				response.Services["postgres"] = "healthy"
			}
		}
	} else {
		response.Services["postgres"] = "not_configured"
		response.Status = "degraded"
	}

	// Check Redis status
	redisInterface, exists := c.Get("redis")
	if !exists || redisInterface == nil {
		response.Services["redis"] = "unavailable"
		response.Status = "degraded"
	} else {
		redisClient := redisInterface.(*redis.Client)
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		if err := redisClient.Ping(ctx).Err(); err != nil {
			response.Services["redis"] = "ping_failed"
			response.Status = "degraded"
		} else {
			response.Services["redis"] = "healthy"
		}
	}

	// Set HTTP status code based on overall health
	httpStatus := http.StatusOK
	if response.Status == "degraded" {
		httpStatus = http.StatusServiceUnavailable
	}

	c.JSON(httpStatus, response)
}
package handlers

import (
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"cuentas/internal/models"
	"cuentas/internal/services"

	"github.com/gin-gonic/gin"
)

// CreateInventoryItemHandler handles POST /v1/inventory/items
func CreateInventoryItemHandler(c *gin.Context) {
	// Get company_id from context (set by middleware)
	companyID, exists := c.Get("company_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error: "company_id not found in context",
			Code:  "unauthorized",
		})
		return
	}

	// Read and parse request body
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: "failed to read request body",
			Code:  "invalid_request",
		})
		return
	}

	var req models.CreateInventoryItemRequest
	if err := json.Unmarshal(bodyBytes, &req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: "invalid JSON format",
			Code:  "invalid_json",
		})
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: err.Error(),
			Code:  "validation_failed",
		})
		return
	}

	// Get database connection
	db := c.MustGet("db").(*sql.DB)

	// Create item
	inventoryService := services.NewInventoryService(db)
	item, err := inventoryService.CreateItem(c.Request.Context(), companyID.(string), &req)
	if err != nil {
		// Handle specific database errors
		if strings.Contains(err.Error(), "unique_company_sku") {
			c.JSON(http.StatusConflict, models.ErrorResponse{
				Error: "an item with this SKU already exists",
				Code:  "duplicate_sku",
			})
			return
		}
		if strings.Contains(err.Error(), "unique_company_barcode") {
			c.JSON(http.StatusConflict, models.ErrorResponse{
				Error: "an item with this barcode already exists",
				Code:  "duplicate_barcode",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "failed to create item",
			Code:  "internal_error",
		})
		return
	}

	c.JSON(http.StatusCreated, item)
}

// GetInventoryItemHandler handles GET /v1/inventory/items/:id
func GetInventoryItemHandler(c *gin.Context) {
	itemID := c.Param("id")
	companyID := c.MustGet("company_id").(string)
	db := c.MustGet("db").(*sql.DB)

	inventoryService := services.NewInventoryService(db)
	item, err := inventoryService.GetItemByID(c.Request.Context(), companyID, itemID)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error: "item not found",
				Code:  "not_found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "failed to get item",
			Code:  "internal_error",
		})
		return
	}

	c.JSON(http.StatusOK, item)
}

// ListInventoryItemsHandler handles GET /v1/inventory/items
func ListInventoryItemsHandler(c *gin.Context) {
	companyID := c.MustGet("company_id").(string)
	db := c.MustGet("db").(*sql.DB)

	// Parse query parameters
	activeOnly := c.DefaultQuery("active", "true") == "true"
	tipoItem := c.Query("tipo_item") // "" means no filter

	inventoryService := services.NewInventoryService(db)
	items, err := inventoryService.ListItems(c.Request.Context(), companyID, activeOnly, tipoItem)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "failed to list items",
			Code:  "internal_error",
		})
		return
	}

	// Always return array, even if empty
	if items == nil {
		items = []models.InventoryItem{}
	}

	c.JSON(http.StatusOK, gin.H{
		"items": items,
		"count": len(items),
	})
}

// UpdateInventoryItemHandler handles PUT /v1/inventory/items/:id
func UpdateInventoryItemHandler(c *gin.Context) {
	itemID := c.Param("id")
	companyID := c.MustGet("company_id").(string)
	db := c.MustGet("db").(*sql.DB)

	// Read and parse request body
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: "failed to read request body",
			Code:  "invalid_request",
		})
		return
	}

	var req models.UpdateInventoryItemRequest
	if err := json.Unmarshal(bodyBytes, &req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: "invalid JSON format",
			Code:  "invalid_json",
		})
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: err.Error(),
			Code:  "validation_failed",
		})
		return
	}

	// Update item
	inventoryService := services.NewInventoryService(db)
	item, err := inventoryService.UpdateItem(c.Request.Context(), companyID, itemID, &req)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error: "item not found",
				Code:  "not_found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "failed to update item",
			Code:  "internal_error",
		})
		return
	}

	c.JSON(http.StatusOK, item)
}

// DeleteInventoryItemHandler handles DELETE /v1/inventory/items/:id
func DeleteInventoryItemHandler(c *gin.Context) {
	itemID := c.Param("id")
	companyID := c.MustGet("company_id").(string)
	db := c.MustGet("db").(*sql.DB)

	inventoryService := services.NewInventoryService(db)
	err := inventoryService.DeleteItem(c.Request.Context(), companyID, itemID)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error: "item not found",
				Code:  "not_found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "failed to delete item",
			Code:  "internal_error",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "item deleted successfully",
	})
}

// GetItemTaxesHandler handles GET /v1/inventory/items/:id/taxes
func GetItemTaxesHandler(c *gin.Context) {
	itemID := c.Param("id")
	companyID := c.MustGet("company_id").(string)
	db := c.MustGet("db").(*sql.DB)

	inventoryService := services.NewInventoryService(db)
	taxes, err := inventoryService.GetItemTaxes(c.Request.Context(), companyID, itemID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "failed to get taxes",
			Code:  "internal_error",
		})
		return
	}

	// Always return array, even if empty
	if taxes == nil {
		taxes = []models.InventoryItemTax{}
	}

	c.JSON(http.StatusOK, gin.H{
		"taxes": taxes,
		"count": len(taxes),
	})
}

// AddItemTaxHandler handles POST /v1/inventory/items/:id/taxes
func AddItemTaxHandler(c *gin.Context) {
	itemID := c.Param("id")
	companyID := c.MustGet("company_id").(string)
	db := c.MustGet("db").(*sql.DB)

	var req models.AddItemTaxRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: "invalid JSON format",
			Code:  "invalid_json",
		})
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: err.Error(),
			Code:  "validation_failed",
		})
		return
	}

	inventoryService := services.NewInventoryService(db)
	tax, err := inventoryService.AddItemTax(c.Request.Context(), companyID, itemID, &req)
	if err != nil {
		if strings.Contains(err.Error(), "unique_item_tributo") {
			c.JSON(http.StatusConflict, models.ErrorResponse{
				Error: "this tax is already assigned to this item",
				Code:  "duplicate_tax",
			})
			return
		}
		if strings.Contains(err.Error(), "item not found") {
			c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error: "item not found",
				Code:  "not_found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "failed to add tax",
			Code:  "internal_error",
		})
		return
	}

	c.JSON(http.StatusCreated, tax)
}

// RemoveItemTaxHandler handles DELETE /v1/inventory/items/:id/taxes/:code
func RemoveItemTaxHandler(c *gin.Context) {
	itemID := c.Param("id")
	tributoCode := c.Param("code")
	companyID := c.MustGet("company_id").(string)
	db := c.MustGet("db").(*sql.DB)

	inventoryService := services.NewInventoryService(db)
	err := inventoryService.RemoveItemTax(c.Request.Context(), companyID, itemID, tributoCode)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error: "tax not found for this item",
				Code:  "not_found",
			})
			return
		}
		if strings.Contains(err.Error(), "item not found") {
			c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error: "item not found",
				Code:  "not_found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "failed to remove tax",
			Code:  "internal_error",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "tax removed successfully",
	})
}
package handlers

import (
	"net/http"

	"cuentas/internal/models"
	"cuentas/internal/services"

	"github.com/gin-gonic/gin"
)

type InvoiceHandler struct {
	invoiceService *services.InvoiceService
}

func NewInvoiceHandler() *InvoiceHandler {
	return &InvoiceHandler{
		invoiceService: services.NewInvoiceService(),
	}
}

// CreateInvoice handles POST /api/v1/invoices
func (h *InvoiceHandler) CreateInvoice(c *gin.Context) {
	companyID := c.GetString("company_id")
	if companyID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "company_id not found in context"})
		return
	}

	var req models.CreateInvoiceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	invoice, err := h.invoiceService.CreateInvoice(c.Request.Context(), companyID, &req)
	if err != nil {
		if err == services.ErrClientNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "client not found"})
			return
		}
		if err == services.ErrInventoryItemNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "inventory item not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, invoice)
}

// GetInvoice handles GET /api/v1/invoices/:id
func (h *InvoiceHandler) GetInvoice(c *gin.Context) {
	companyID := c.GetString("company_id")
	if companyID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "company_id not found in context"})
		return
	}

	invoiceID := c.Param("id")
	if invoiceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invoice_id is required"})
		return
	}

	invoice, err := h.invoiceService.GetInvoice(c.Request.Context(), companyID, invoiceID)
	if err != nil {
		if err == services.ErrInvoiceNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "invoice not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, invoice)
}

// ListInvoices handles GET /api/v1/invoices
func (h *InvoiceHandler) ListInvoices(c *gin.Context) {
	companyID := c.GetString("company_id")
	if companyID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "company_id not found in context"})
		return
	}

	// Build filters from query params
	filters := make(map[string]interface{})
	if status := c.Query("status"); status != "" {
		filters["status"] = status
	}
	if clientID := c.Query("client_id"); clientID != "" {
		filters["client_id"] = clientID
	}
	if paymentStatus := c.Query("payment_status"); paymentStatus != "" {
		filters["payment_status"] = paymentStatus
	}
	if establishmentID := c.Query("establishment_id"); establishmentID != "" {
		filters["establishment_id"] = establishmentID
	}
	if posID := c.Query("point_of_sale_id"); posID != "" {
		filters["point_of_sale_id"] = posID
	}

	invoices, err := h.invoiceService.ListInvoices(c.Request.Context(), companyID, filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"invoices": invoices,
		"count":    len(invoices),
	})
}

// DeleteInvoice handles DELETE /api/v1/invoices/:id
func (h *InvoiceHandler) DeleteInvoice(c *gin.Context) {
	companyID := c.GetString("company_id")
	if companyID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "company_id not found in context"})
		return
	}

	invoiceID := c.Param("id")
	if invoiceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invoice_id is required"})
		return
	}

	err := h.invoiceService.DeleteDraftInvoice(c.Request.Context(), companyID, invoiceID)
	if err != nil {
		if err == services.ErrInvoiceNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "invoice not found"})
			return
		}
		if err == services.ErrInvoiceNotDraft {
			c.JSON(http.StatusBadRequest, gin.H{"error": "only draft invoices can be deleted"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "invoice deleted successfully"})
}

// FinalizeInvoice handles POST /v1/invoices/:id/finalize
func (h *InvoiceHandler) FinalizeInvoice(c *gin.Context) {
	companyID := c.GetString("company_id")
	if companyID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "company_id not found in context"})
		return
	}

	invoiceID := c.Param("id")
	if invoiceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invoice_id is required"})
		return
	}

	// Parse request body with payment info
	var req models.FinalizeInvoiceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get the invoice first to validate payment amount
	existingInvoice, err := h.invoiceService.GetInvoice(c.Request.Context(), companyID, invoiceID)
	if err != nil {
		if err == services.ErrInvoiceNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "invoice not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Validate payment against invoice total
	if err := req.Validate(existingInvoice.Total); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// TODO: Get user ID from auth context when auth is implemented
	userID := "00000000-0000-0000-0000-000000000000" // Placeholder

	// Finalize invoice with payment info
	invoice, err := h.invoiceService.FinalizeInvoice(c.Request.Context(), companyID, invoiceID, userID, &req.Payment)
	if err != nil {
		if err == services.ErrInvoiceNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "invoice not found"})
			return
		}
		if err == services.ErrInvoiceNotDraft {
			c.JSON(http.StatusBadRequest, gin.H{"error": "only draft invoices can be finalized"})
			return
		}
		if err == services.ErrCreditLimitExceeded {
			c.JSON(http.StatusBadRequest, gin.H{"error": "credit limit exceeded"})
			return
		}
		if err == services.ErrCreditSuspended {
			c.JSON(http.StatusBadRequest, gin.H{"error": "client credit is suspended"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, invoice)
}
