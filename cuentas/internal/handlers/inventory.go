package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"

	"cuentas/internal/models"
	"cuentas/internal/services"

	"github.com/gin-gonic/gin"
)

// CreateInventoryItemHandler handles POST /v1/inventory/items
func CreateInventoryItemHandler(c *gin.Context) {
	// Get company_id from context (set by middleware)
	companyID, exists := c.Get("company_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error: "company_id not found in context",
			Code:  "unauthorized",
		})
		return
	}

	// Read and parse request body
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: "failed to read request body",
			Code:  "invalid_request",
		})
		return
	}

	var req models.CreateInventoryItemRequest
	if err := json.Unmarshal(bodyBytes, &req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: "invalid JSON format",
			Code:  "invalid_json",
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

	// Get database connection
	db := c.MustGet("db").(*sql.DB)

	// Create item
	inventoryService := services.NewInventoryService(db)
	item, err := inventoryService.CreateItem(c.Request.Context(), companyID.(string), &req)
	if err != nil {
		// Handle specific database errors
		if strings.Contains(err.Error(), "unique_company_sku") {
			c.JSON(http.StatusConflict, models.ErrorResponse{
				Error: "an item with this SKU already exists",
				Code:  "duplicate_sku",
			})
			return
		}
		if strings.Contains(err.Error(), "unique_company_barcode") {
			c.JSON(http.StatusConflict, models.ErrorResponse{
				Error: "an item with this barcode already exists",
				Code:  "duplicate_barcode",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "failed to create item",
			Code:  "internal_error",
		})
		return
	}

	c.JSON(http.StatusCreated, item)
}

// GetInventoryItemHandler handles GET /v1/inventory/items/:id
func GetInventoryItemHandler(c *gin.Context) {
	itemID := c.Param("id")
	companyID := c.MustGet("company_id").(string)
	db := c.MustGet("db").(*sql.DB)

	inventoryService := services.NewInventoryService(db)
	item, err := inventoryService.GetItemByID(c.Request.Context(), companyID, itemID)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error: "item not found",
				Code:  "not_found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "failed to get item",
			Code:  "internal_error",
		})
		return
	}

	c.JSON(http.StatusOK, item)
}

// ListInventoryItemsHandler handles GET /v1/inventory/items
func ListInventoryItemsHandler(c *gin.Context) {
	companyID := c.MustGet("company_id").(string)
	db := c.MustGet("db").(*sql.DB)

	// Parse query parameters
	activeOnly := c.DefaultQuery("active", "true") == "true"
	tipoItem := c.Query("tipo_item") // "" means no filter

	inventoryService := services.NewInventoryService(db)
	items, err := inventoryService.ListItems(c.Request.Context(), companyID, activeOnly, tipoItem)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "failed to list items",
			Code:  "internal_error",
		})
		return
	}

	// Always return array, even if empty
	if items == nil {
		items = []models.InventoryItem{}
	}

	c.JSON(http.StatusOK, gin.H{
		"items": items,
		"count": len(items),
	})
}

// UpdateInventoryItemHandler handles PUT /v1/inventory/items/:id
func UpdateInventoryItemHandler(c *gin.Context) {
	itemID := c.Param("id")
	companyID := c.MustGet("company_id").(string)
	db := c.MustGet("db").(*sql.DB)

	// Read and parse request body
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: "failed to read request body",
			Code:  "invalid_request",
		})
		return
	}

	var req models.UpdateInventoryItemRequest
	if err := json.Unmarshal(bodyBytes, &req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: "invalid JSON format",
			Code:  "invalid_json",
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

	// Update item
	inventoryService := services.NewInventoryService(db)
	item, err := inventoryService.UpdateItem(c.Request.Context(), companyID, itemID, &req)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error: "item not found",
				Code:  "not_found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "failed to update item",
			Code:  "internal_error",
		})
		return
	}

	c.JSON(http.StatusOK, item)
}

// DeleteInventoryItemHandler handles DELETE /v1/inventory/items/:id
func DeleteInventoryItemHandler(c *gin.Context) {
	itemID := c.Param("id")
	companyID := c.MustGet("company_id").(string)
	db := c.MustGet("db").(*sql.DB)

	inventoryService := services.NewInventoryService(db)
	err := inventoryService.DeleteItem(c.Request.Context(), companyID, itemID)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error: "item not found",
				Code:  "not_found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "failed to delete item",
			Code:  "internal_error",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "item deleted successfully",
	})
}

// GetItemTaxesHandler handles GET /v1/inventory/items/:id/taxes
func GetItemTaxesHandler(c *gin.Context) {
	itemID := c.Param("id")
	companyID := c.MustGet("company_id").(string)
	db := c.MustGet("db").(*sql.DB)

	inventoryService := services.NewInventoryService(db)
	taxes, err := inventoryService.GetItemTaxes(c.Request.Context(), companyID, itemID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "failed to get taxes",
			Code:  "internal_error",
		})
		return
	}

	// Always return array, even if empty
	if taxes == nil {
		taxes = []models.InventoryItemTax{}
	}

	c.JSON(http.StatusOK, gin.H{
		"taxes": taxes,
		"count": len(taxes),
	})
}

// AddItemTaxHandler handles POST /v1/inventory/items/:id/taxes
func AddItemTaxHandler(c *gin.Context) {
	itemID := c.Param("id")
	companyID := c.MustGet("company_id").(string)
	db := c.MustGet("db").(*sql.DB)

	var req models.AddItemTaxRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: "invalid JSON format",
			Code:  "invalid_json",
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

	inventoryService := services.NewInventoryService(db)
	tax, err := inventoryService.AddItemTax(c.Request.Context(), companyID, itemID, &req)
	if err != nil {
		if strings.Contains(err.Error(), "unique_item_tributo") {
			c.JSON(http.StatusConflict, models.ErrorResponse{
				Error: "this tax is already assigned to this item",
				Code:  "duplicate_tax",
			})
			return
		}
		if strings.Contains(err.Error(), "item not found") {
			c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error: "item not found",
				Code:  "not_found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "failed to add tax",
			Code:  "internal_error",
		})
		return
	}

	c.JSON(http.StatusCreated, tax)
}

// RemoveItemTaxHandler handles DELETE /v1/inventory/items/:id/taxes/:code
func RemoveItemTaxHandler(c *gin.Context) {
	itemID := c.Param("id")
	tributoCode := c.Param("code")
	companyID := c.MustGet("company_id").(string)
	db := c.MustGet("db").(*sql.DB)

	inventoryService := services.NewInventoryService(db)
	err := inventoryService.RemoveItemTax(c.Request.Context(), companyID, itemID, tributoCode)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error: "tax not found for this item",
				Code:  "not_found",
			})
			return
		}
		if strings.Contains(err.Error(), "item not found") {
			c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error: "item not found",
				Code:  "not_found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "failed to remove tax",
			Code:  "internal_error",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "tax removed successfully",
	})
}

// RecordPurchaseHandler handles POST /v1/inventory/items/:id/purchase
func RecordPurchaseHandler(c *gin.Context) {
	itemID := c.Param("id")
	companyID := c.MustGet("company_id").(string)
	db := c.MustGet("db").(*sql.DB)

	log.Printf("[DEBUG] RecordPurchase called - ItemID: %s, CompanyID: %s", itemID, companyID)

	// Parse request
	var req models.RecordPurchaseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("[ERROR] Failed to bind JSON: %v", err)
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: "invalid JSON format",
			Code:  "invalid_json",
		})
		return
	}

	log.Printf("[DEBUG] Request parsed - Quantity: %f, UnitCost: %f", req.Quantity, req.UnitCost)

	// Validate request
	if err := req.Validate(); err != nil {
		log.Printf("[ERROR] Validation failed: %v", err)
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: err.Error(),
			Code:  "validation_failed",
		})
		return
	}

	// Record purchase
	inventoryService := services.NewInventoryService(db)
	event, err := inventoryService.RecordPurchase(c.Request.Context(), companyID, itemID, &req)
	if err != nil {
		log.Printf("[ERROR] RecordPurchase failed: %v", err)
		if strings.Contains(err.Error(), "item not found") {
			c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error: "item not found",
				Code:  "not_found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: fmt.Sprintf("failed to record purchase: %v", err),
			Code:  "internal_error",
		})
		return
	}

	log.Printf("[DEBUG] Purchase recorded successfully - EventID: %d", event.EventID)
	c.JSON(http.StatusCreated, event)
}

// RecordAdjustmentHandler handles POST /v1/inventory/items/:id/adjustment
func RecordAdjustmentHandler(c *gin.Context) {
	itemID := c.Param("id")
	companyID := c.MustGet("company_id").(string)
	db := c.MustGet("db").(*sql.DB)

	// Parse request
	var req models.RecordAdjustmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: "invalid JSON format",
			Code:  "invalid_json",
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

	// Record adjustment
	inventoryService := services.NewInventoryService(db)
	event, err := inventoryService.RecordAdjustment(c.Request.Context(), companyID, itemID, &req)
	if err != nil {
		if strings.Contains(err.Error(), "item not found") {
			c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error: "item not found",
				Code:  "not_found",
			})
			return
		}
		if strings.Contains(err.Error(), "negative quantity") {
			c.JSON(http.StatusBadRequest, models.ErrorResponse{
				Error: err.Error(),
				Code:  "invalid_quantity",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "failed to record adjustment",
			Code:  "internal_error",
		})
		return
	}

	c.JSON(http.StatusCreated, event)
}

// GetInventoryStateHandler handles GET /v1/inventory/items/:id/state
func GetInventoryStateHandler(c *gin.Context) {
	itemID := c.Param("id")
	companyID := c.MustGet("company_id").(string)
	db := c.MustGet("db").(*sql.DB)

	fmt.Println("we are calling the new inventory service")
	inventoryService := services.NewInventoryService(db)
	fmt.Println("we are creating a state")
	state, err := inventoryService.GetInventoryState(c.Request.Context(), companyID, itemID)
	if err != nil {
		if strings.Contains(err.Error(), "item not found") {
			c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error: "item not found",
				Code:  "not_found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "failed to get inventory state",
			Code:  "internal_error",
		})
		return
	}

	c.JSON(http.StatusOK, state)
}

// ListInventoryStatesHandler handles GET /v1/inventory/states
func ListInventoryStatesHandler(c *gin.Context) {
	companyID := c.MustGet("company_id").(string)
	db := c.MustGet("db").(*sql.DB)

	// Parse query parameters
	inStockOnly := c.DefaultQuery("in_stock_only", "false") == "true"

	inventoryService := services.NewInventoryService(db)
	states, err := inventoryService.ListInventoryStates(c.Request.Context(), companyID, inStockOnly)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "failed to list inventory states",
			Code:  "internal_error",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"states": states,
		"count":  len(states),
	})
}

// GetCostHistoryHandler handles GET /v1/inventory/items/:id/cost-history
func GetCostHistoryHandler(c *gin.Context) {
	itemID := c.Param("id")
	companyID := c.MustGet("company_id").(string)
	db := c.MustGet("db").(*sql.DB)

	// Parse query parameters
	limit := 50 // default
	if limitStr := c.Query("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil {
			limit = parsedLimit
		}
	}

	inventoryService := services.NewInventoryService(db)
	events, err := inventoryService.GetCostHistory(c.Request.Context(), companyID, itemID, limit)
	if err != nil {
		if strings.Contains(err.Error(), "item not found") {
			c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error: "item not found",
				Code:  "not_found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "failed to get cost history",
			Code:  "internal_error",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"events": events,
		"count":  len(events),
	})
}
