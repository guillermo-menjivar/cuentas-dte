package handlers

import (
	"fmt"
	"net/http"

	"cuentas/internal/dte"
	"cuentas/internal/hacienda"
	"cuentas/internal/models"
	"cuentas/internal/services"

	"github.com/gin-gonic/gin"
)

type PurchaseHandler struct {
	purchaseService *services.PurchaseService
}

func NewPurchaseHandler(svc *services.PurchaseService) *PurchaseHandler {
	return &PurchaseHandler{
		purchaseService: svc,
	}
}

// CreateFSE handles POST /api/v1/purchases/fse
func (h *PurchaseHandler) CreateFSE(c *gin.Context) {
	companyID := c.GetString("company_id")
	if companyID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "company_id not found in context"})
		return
	}

	var req models.CreateFSERequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	purchase, err := h.purchaseService.CreateFSE(c.Request.Context(), companyID, &req)
	if err != nil {
		if err == services.ErrPointOfSaleNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "point of sale not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, purchase)
}

// GetPurchase handles GET /api/v1/purchases/:id
func (h *PurchaseHandler) GetPurchase(c *gin.Context) {
	companyID := c.GetString("company_id")
	if companyID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "company_id not found in context"})
		return
	}

	purchaseID := c.Param("id")
	if purchaseID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "purchase_id is required"})
		return
	}

	purchase, err := h.purchaseService.GetPurchaseByID(c.Request.Context(), companyID, purchaseID)
	if err != nil {
		if err == services.ErrPurchaseNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "purchase not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, purchase)
}

// ListPurchases handles GET /api/v1/purchases
func (h *PurchaseHandler) ListPurchases(c *gin.Context) {
	companyID := c.GetString("company_id")
	if companyID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "company_id not found in context"})
		return
	}

	// TODO: Add filters from query params if needed
	// For now, use default pagination
	limit := 50
	offset := 0

	purchases, err := h.purchaseService.ListPurchases(c.Request.Context(), companyID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"purchases": purchases,
		"count":     len(purchases),
	})
}

// FinalizePurchase handles POST /api/v1/purchases/:id/finalize
func (h *PurchaseHandler) FinalizePurchase(c *gin.Context) {
	companyID := c.GetString("company_id")
	if companyID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "company_id not found in context"})
		return
	}

	purchaseID := c.Param("id")
	if purchaseID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "purchase_id is required"})
		return
	}

	// TODO: Get user ID from auth context when auth is implemented
	userID := "00000000-0000-0000-0000-000000000000" // Placeholder

	// Finalize purchase (generates numero control, updates status)
	purchase, err := h.purchaseService.FinalizePurchase(c.Request.Context(), companyID, purchaseID, userID)
	if err != nil {
		if err == services.ErrPurchaseNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "purchase not found"})
			return
		}
		if err == services.ErrPurchaseNotDraft {
			c.JSON(http.StatusBadRequest, gin.H{"error": "only draft purchases can be finalized"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// ===== DTE PROCESSING =====
	// Process FSE DTE (build, sign, submit to Hacienda)
	dteServiceInterface, exists := c.Get("dteService")
	if exists {
		dteService := dteServiceInterface.(*dte.DTEService)

		fmt.Println("\n=== Starting FSE DTE Processing ===")

		var response *hacienda.ReceptionResponse
		var err error

		if purchase.IsFSE() {
			fmt.Println("📦 FSE Purchase (Type 14) detected - processing...")
			response, err = dteService.ProcessFSE(c.Request.Context(), purchase)
		} else {
			fmt.Printf("⚠️  Purchase type '%s' not yet supported for DTE processing\n", purchase.PurchaseType)
		}

		if err != nil {
			// Log the error but don't fail the finalization
			fmt.Printf("❌ FSE DTE processing failed: %v\n", err)
			dteStatus := "failed_signing"
			purchase.DteStatus = &dteStatus
		} else if response != nil {
			// Successfully processed
			fmt.Printf("✅ FSE DTE signed and submitted successfully for purchase %s\n", purchase.ID)
			fmt.Printf("Estado: %s\n", response.Estado)
			fmt.Printf("Código de Generación: %s\n", response.CodigoGeneracion)
			if response.SelloRecibido != "" {
				fmt.Printf("Sello Recibido: %s\n", response.SelloRecibido)
			}

			dteStatus := response.Estado
			purchase.DteStatus = &dteStatus
			purchase.DteSelloRecibido = &response.SelloRecibido
		}
	} else {
		fmt.Println("⚠️  Warning: DTE service not available in context")
	}
	// ===== END DTE PROCESSING =====

	c.JSON(http.StatusOK, purchase)
}
