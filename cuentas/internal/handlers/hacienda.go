package handlers

import (
	"database/sql"
	"net/http"

	"cuentas/internal/models"
	"cuentas/internal/services"

	"github.com/gin-gonic/gin"
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

	// Create Hacienda service
	haciendaService, err := services.NewHaciendaService(db, vaultService)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "failed to initialize hacienda service",
			Code:  "internal_error",
		})
		return
	}

	// Authenticate with Hacienda
	token, err := haciendaService.AuthenticateCompany(c.Request.Context(), companyID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error: err.Error(),
			Code:  "authentication_failed",
		})
		return
	}

	// Update last activity
	if err := haciendaService.UpdateLastActivity(c.Request.Context(), companyID); err != nil {
		// Log but don't fail the request
		c.Writer.Header().Set("X-Activity-Update-Failed", "true")
	}

	c.JSON(http.StatusOK, gin.H{
		"token":      token,
		"company_id": companyID,
	})
}
