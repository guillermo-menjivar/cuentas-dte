package handlers

import (
	"fmt"
	"net/http"

	"cuentas/internal/models"
	"cuentas/internal/services"

	"github.com/gin-gonic/gin"
)

// CreateCompanyHandler handles POST /v1/companies
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
	companyService := services.NewCompanyService(vaultService)

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
	companyService := services.NewCompanyService(vaultService)

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
