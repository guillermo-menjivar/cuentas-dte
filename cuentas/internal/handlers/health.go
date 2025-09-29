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
