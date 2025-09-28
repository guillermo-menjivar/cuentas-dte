package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// HealthHandler handles the health check endpoint
func HealthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"message": "Cuentas API is healthy",
	})
}
