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
