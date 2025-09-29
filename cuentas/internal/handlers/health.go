package handlers

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
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
		// Vault service exists, assume it's healthy since we got this far
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

			// Test the connection with a simple ping
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

	// Set HTTP status code based on overall health
	httpStatus := http.StatusOK
	if response.Status == "degraded" {
		httpStatus = http.StatusServiceUnavailable
	}

	c.JSON(httpStatus, response)
}
