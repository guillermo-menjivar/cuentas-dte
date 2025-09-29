package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"cuentas/internal/models"
	"cuentas/internal/services"

	"github.com/gin-gonic/gin"
)

// parseJSONError converts technical JSON errors to user-friendly messages
func parseJSONError(err error) string {
	errMsg := err.Error()

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
	print(errMsg)

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
