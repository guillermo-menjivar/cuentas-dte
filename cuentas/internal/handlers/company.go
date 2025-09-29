package handlers

import (
	"database/sql"
	"fmt"
	"net/http"

	"cuentas/internal/models"
	"cuentas/internal/services"

	"github.com/gin-gonic/gin"
)

func CreateCompanyHandler(c *gin.Context) {
	// Parse request body
	var req models.CreateCompanyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: fmt.Sprintf("invalid request body: %v", err),
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
