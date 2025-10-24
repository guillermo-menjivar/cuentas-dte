package handlers

import (
	"cuentas/internal/formats"
	"cuentas/internal/models"
	"cuentas/internal/services"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// GetCostHistoryHandler handles GET /v1/inventory/items/:id/cost-history
func GetCostHistoryHandler(c *gin.Context) {
	itemID := c.Param("id")
	companyID := c.MustGet("company_id").(string)
	db := c.MustGet("db").(*sql.DB)

	// Parse query params
	limit := 50 // default
	if limitStr := c.Query("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	// Parse sort order (default: desc = newest first)
	sortOrder := c.DefaultQuery("sort", "desc")
	if sortOrder != "asc" && sortOrder != "desc" {
		sortOrder = "desc"
	}

	// Parse date range filters
	startDate := c.Query("start_date") // ISO format: 2024-01-01
	endDate := c.Query("end_date")     // ISO format: 2024-12-31

	// Get cost history
	inventoryService := services.NewInventoryService(db)
	events, err := inventoryService.GetCostHistory(c.Request.Context(), companyID, itemID, limit, sortOrder, startDate, endDate)
	if err != nil {
		if strings.Contains(err.Error(), "item not found") {
			c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error: "item not found",
				Code:  "not_found",
			})
			return
		}
		if strings.Contains(err.Error(), "invalid date") {
			c.JSON(http.StatusBadRequest, models.ErrorResponse{
				Error: err.Error(),
				Code:  "invalid_date",
			})
			return
		}
		log.Printf("[ERROR] GetCostHistory failed: %v", err)
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "failed to get cost history",
			Code:  "internal_error",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"events": events,
	})
}

// GetAllEventsHandler handles GET /v1/inventory/events
func GetAllEventsHandler(c *gin.Context) {
	companyID := c.MustGet("company_id").(string)
	db := c.MustGet("db").(*sql.DB)

	// Parse query params
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")
	eventType := c.Query("event_type")
	sortOrder := c.DefaultQuery("sort", "desc")
	format := formats.DetermineFormat(c.GetHeader("Accept"), c.Query("format"))
	language := formats.DetermineLanguage(c.Query("language")) // Default: Spanish

	if sortOrder != "asc" && sortOrder != "desc" {
		sortOrder = "desc"
	}

	// Get all events
	inventoryService := services.NewInventoryService(db)
	events, err := inventoryService.GetAllEvents(c.Request.Context(), companyID, startDate, endDate, eventType, sortOrder)
	if err != nil {
		if strings.Contains(err.Error(), "invalid date") {
			c.JSON(http.StatusBadRequest, models.ErrorResponse{
				Error: err.Error(),
				Code:  "invalid_date",
			})
			return
		}
		log.Printf("[ERROR] GetAllEvents failed: %v", err)
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "failed to get events",
			Code:  "internal_error",
		})
		return
	}

	// Return based on format
	if format == "csv" {
		csvData, err := formats.WriteEventsCSV(events, language) // Pass language
		if err != nil {
			log.Printf("[ERROR] Failed to generate CSV: %v", err)
			c.JSON(http.StatusInternalServerError, models.ErrorResponse{
				Error: "failed to generate CSV",
				Code:  "internal_error",
			})
			return
		}

		// Generate filename
		filename := fmt.Sprintf("inventory_events_%s.csv", time.Now().Format("20060102_150405"))
		if startDate != "" && endDate != "" {
			filename = fmt.Sprintf("inventory_events_%s_to_%s.csv", startDate, endDate)
		}

		c.Header("Content-Type", "text/csv; charset=utf-8")
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
		c.Data(http.StatusOK, "text/csv", csvData)
		return
	}

	// Default: JSON
	c.JSON(http.StatusOK, gin.H{
		"events": events,
	})
}

// GetInventoryValuationHandler handles GET /v1/inventory/valuation
func GetInventoryValuationHandler(c *gin.Context) {
	companyID := c.MustGet("company_id").(string)
	db := c.MustGet("db").(*sql.DB)

	// Parse as_of_date (required)
	asOfDate := c.Query("as_of_date")
	if asOfDate == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: "as_of_date parameter is required (format: YYYY-MM-DD)",
			Code:  "missing_parameter",
		})
		return
	}

	// Validate date format
	_, err := time.Parse("2006-01-02", asOfDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: "invalid date format, use YYYY-MM-DD",
			Code:  "invalid_date",
		})
		return
	}

	format := formats.DetermineFormat(c.GetHeader("Accept"), c.Query("format"))
	language := formats.DetermineLanguage(c.Query("language")) // Default: Spanish

	// Get valuation
	inventoryService := services.NewInventoryService(db)
	valuation, err := inventoryService.GetInventoryValuationAtDate(c.Request.Context(), companyID, asOfDate)
	if err != nil {
		log.Printf("[ERROR] GetInventoryValuation failed: %v", err)
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "failed to get inventory valuation",
			Code:  "internal_error",
		})
		return
	}

	// Return based on format
	if format == "csv" {
		csvData, err := formats.WriteValuationCSV(valuation, language) // Pass language
		if err != nil {
			log.Printf("[ERROR] Failed to generate CSV: %v", err)
			c.JSON(http.StatusInternalServerError, models.ErrorResponse{
				Error: "failed to generate CSV",
				Code:  "internal_error",
			})
			return
		}

		filename := fmt.Sprintf("inventory_valuation_%s.csv", asOfDate)
		c.Header("Content-Type", "text/csv; charset=utf-8")
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
		c.Data(http.StatusOK, "text/csv", csvData)
		return
	}

	// Default: JSON
	c.JSON(http.StatusOK, valuation)
}

// ListInventoryStatesHandler handles GET /v1/inventory/states
func ListInventoryStatesHandler(c *gin.Context) {
	companyID := c.MustGet("company_id").(string)
	db := c.MustGet("db").(*sql.DB)

	// Parse query params
	inStockOnly := c.Query("in_stock_only") == "true"
	format := formats.DetermineFormat(c.GetHeader("Accept"), c.Query("format"))
	language := formats.DetermineLanguage(c.Query("language")) // Default: Spanish

	// Get states
	inventoryService := services.NewInventoryService(db)
	states, err := inventoryService.ListInventoryStates(c.Request.Context(), companyID, inStockOnly)
	if err != nil {
		log.Printf("[ERROR] ListInventoryStates failed: %v", err)
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "failed to list inventory states",
			Code:  "internal_error",
		})
		return
	}

	// Return based on format
	if format == "csv" {
		csvData, err := formats.WriteInventoryStatesCSV(states, language) // Pass language
		if err != nil {
			log.Printf("[ERROR] Failed to generate CSV: %v", err)
			c.JSON(http.StatusInternalServerError, models.ErrorResponse{
				Error: "failed to generate CSV",
				Code:  "internal_error",
			})
			return
		}

		filename := "inventory_states.csv"
		if inStockOnly {
			filename = "inventory_in_stock.csv"
		}

		c.Header("Content-Type", "text/csv; charset=utf-8")
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
		c.Data(http.StatusOK, "text/csv", csvData)
		return
	}

	// Default: JSON
	c.JSON(http.StatusOK, gin.H{
		"states": states,
	})
}

// Generates Article 142-A compliant inventory register for a specific item
func GetLegalInventoryRegisterHandler(c *gin.Context) {
	itemID := c.Param("id")
	companyID := c.MustGet("company_id").(string)
	db := c.MustGet("db").(*sql.DB)

	// Parse query params
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")
	language := formats.DetermineLanguage(c.Query("language"))

	// Validate required parameters
	if startDate == "" || endDate == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: "start_date and end_date are required",
			Code:  "missing_parameters",
		})
		return
	}

	// Validate date formats
	if _, err := time.Parse("2006-01-02", startDate); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: "invalid start_date format, use YYYY-MM-DD",
			Code:  "invalid_date",
		})
		return
	}
	if _, err := time.Parse("2006-01-02", endDate); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: "invalid end_date format, use YYYY-MM-DD",
			Code:  "invalid_date",
		})
		return
	}

	inventoryService := services.NewInventoryService(db)

	// Get company legal info for report header
	var companyInfo models.CompanyLegalInfo
	err := db.QueryRowContext(c.Request.Context(),
		"SELECT COALESCE(nombre_comercial, name) as legal_name, nit, ncr FROM companies WHERE id = $1",
		companyID,
	).Scan(&companyInfo.LegalName, &companyInfo.NIT, &companyInfo.NRC)
	if err != nil {
		log.Printf("[ERROR] Failed to get company info: %v", err)
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "failed to get company information",
			Code:  "internal_error",
		})
		return
	}

	// Get item info
	item, err := inventoryService.GetItemByID(c.Request.Context(), companyID, itemID)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Error: "item not found",
			Code:  "not_found",
		})
		return
	}

	// Get events for this item in date range
	events, err := inventoryService.GetCostHistory(c.Request.Context(), companyID, itemID, 10000, "asc", startDate, endDate)
	if err != nil {
		log.Printf("[ERROR] GetCostHistory failed: %v", err)
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "failed to get cost history",
			Code:  "internal_error",
		})
		return
	}

	// Convert to InventoryEventWithItem (include SKU and name)
	eventsWithItem := make([]models.InventoryEventWithItem, len(events))
	for i, event := range events {
		eventsWithItem[i] = models.InventoryEventWithItem{
			InventoryEvent: event,
			SKU:            item.SKU,
			ItemName:       item.Name,
		}
	}

	// Generate legal CSV register
	csvData, err := formats.WriteLegalInventoryRegisterCSV(&companyInfo, item, eventsWithItem, startDate, endDate, language)
	if err != nil {
		log.Printf("[ERROR] Failed to generate legal register: %v", err)
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "failed to generate register",
			Code:  "internal_error",
		})
		return
	}

	// Return CSV file
	filename := fmt.Sprintf("registro_inventario_%s_%s_a_%s.csv", item.SKU, startDate, endDate)
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.Data(http.StatusOK, "text/csv", csvData)
}

// GetLegalInventoryRegisterHandler handles GET /v1/inventory/items/:id/legal-register
// Generates Article 142-A compliant inventory register for a specific item
func GetLegalInventoryRegisterHandler(c *gin.Context) {
	itemID := c.Param("id")
	companyID := c.MustGet("company_id").(string)
	db := c.MustGet("db").(*sql.DB)

	// Parse query params
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")
	language := formats.DetermineLanguage(c.Query("language"))

	// Validate required parameters
	if startDate == "" || endDate == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: "start_date and end_date are required",
			Code:  "missing_parameters",
		})
		return
	}

	// Validate date formats
	if _, err := time.Parse("2006-01-02", startDate); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: "invalid start_date format, use YYYY-MM-DD",
			Code:  "invalid_date",
		})
		return
	}
	if _, err := time.Parse("2006-01-02", endDate); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: "invalid end_date format, use YYYY-MM-DD",
			Code:  "invalid_date",
		})
		return
	}

	inventoryService := services.NewInventoryService(db)

	// Get company legal info for report header
	var companyInfo models.CompanyLegalInfo
	err := db.QueryRowContext(c.Request.Context(),
		"SELECT COALESCE(nombre_comercial, name) as legal_name, nit, ncr FROM companies WHERE id = $1",
		companyID,
	).Scan(&companyInfo.LegalName, &companyInfo.NIT, &companyInfo.NRC)
	if err != nil {
		log.Printf("[ERROR] Failed to get company info: %v", err)
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "failed to get company information",
			Code:  "internal_error",
		})
		return
	}

	// Get item info
	item, err := inventoryService.GetItemByID(c.Request.Context(), companyID, itemID)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Error: "item not found",
			Code:  "not_found",
		})
		return
	}

	// Get events for this item in date range
	events, err := inventoryService.GetCostHistory(c.Request.Context(), companyID, itemID, 10000, "asc", startDate, endDate)
	if err != nil {
		log.Printf("[ERROR] GetCostHistory failed: %v", err)
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "failed to get cost history",
			Code:  "internal_error",
		})
		return
	}

	// Convert to InventoryEventWithItem (include SKU and name)
	eventsWithItem := make([]models.InventoryEventWithItem, len(events))
	for i, event := range events {
		eventsWithItem[i] = models.InventoryEventWithItem{
			InventoryEvent: event,
			SKU:            item.SKU,
			ItemName:       item.Name,
		}
	}

	// Generate legal CSV register
	csvData, err := formats.WriteLegalInventoryRegisterCSV(&companyInfo, item, eventsWithItem, startDate, endDate, language)
	if err != nil {
		log.Printf("[ERROR] Failed to generate legal register: %v", err)
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "failed to generate register",
			Code:  "internal_error",
		})
		return
	}

	// Return CSV file
	filename := fmt.Sprintf("registro_inventario_%s_%s_a_%s.csv", item.SKU, startDate, endDate)
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.Data(http.StatusOK, "text/csv", csvData)
}
