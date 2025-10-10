package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
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
	// Read the body first to provide better error messages
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: "failed to read request body",
			Code:  "invalid_request",
		})
		return
	}

	// Parse request body with custom error handling
	var req models.CreateCompanyRequest
	if err := json.Unmarshal(bodyBytes, &req); err != nil {
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
	fmt.Println(req)
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
