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
