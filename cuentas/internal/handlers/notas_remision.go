package handlers

import (
	"fmt"
	"net/http"

	"cuentas/internal/dte"
	"cuentas/internal/models"
	"cuentas/internal/services"

	"github.com/gin-gonic/gin"
)

type RemisionHandler struct {
	invoiceService *services.InvoiceService
}

func NewRemisionHandler(svc *services.InvoiceService) *RemisionHandler {
	return &RemisionHandler{
		invoiceService: svc,
	}
}

// CreateRemision handles POST /api/v1/remisiones
func (h *RemisionHandler) CreateRemision(c *gin.Context) {
	companyID := c.GetString("company_id")
	if companyID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "company_id not found in context"})
		return
	}

	var req models.CreateRemisionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	remision, err := h.invoiceService.CreateRemision(c.Request.Context(), companyID, &req)
	if err != nil {
		if err == services.ErrClientNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "client not found"})
			return
		}
		if err == services.ErrInventoryItemNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "inventory item not found"})
			return
		}
		if err == services.ErrPointOfSaleNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "point of sale not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, remision)
}

// GetRemision handles GET /api/v1/remisiones/:id
func (h *RemisionHandler) GetRemision(c *gin.Context) {
	companyID := c.GetString("company_id")
	if companyID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "company_id not found in context"})
		return
	}

	remisionID := c.Param("id")
	if remisionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "remision_id is required"})
		return
	}

	remision, err := h.invoiceService.GetInvoice(c.Request.Context(), companyID, remisionID)
	if err != nil {
		if err == services.ErrInvoiceNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "remision not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Verify it's actually a remision
	if !remision.IsRemision() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "document is not a remision"})
		return
	}

	c.JSON(http.StatusOK, remision)
}

// ListRemisiones handles GET /api/v1/remisiones
func (h *RemisionHandler) ListRemisiones(c *gin.Context) {
	companyID := c.GetString("company_id")
	if companyID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "company_id not found in context"})
		return
	}

	// Build filters from query params
	filters := make(map[string]interface{})
	filters["dte_type"] = "04" // Only Type 04 remisiones

	if status := c.Query("status"); status != "" {
		filters["status"] = status
	}
	if clientID := c.Query("client_id"); clientID != "" {
		filters["client_id"] = clientID
	}
	if establishmentID := c.Query("establishment_id"); establishmentID != "" {
		filters["establishment_id"] = establishmentID
	}
	if posID := c.Query("point_of_sale_id"); posID != "" {
		filters["point_of_sale_id"] = posID
	}
	if dteStatus := c.Query("dte_status"); dteStatus != "" {
		filters["dte_status"] = dteStatus
	}

	remisiones, err := h.invoiceService.ListInvoices(c.Request.Context(), companyID, filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"remisiones": remisiones,
		"count":      len(remisiones),
	})
}

// DeleteRemision handles DELETE /api/v1/remisiones/:id
func (h *RemisionHandler) DeleteRemision(c *gin.Context) {
	companyID := c.GetString("company_id")
	if companyID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "company_id not found in context"})
		return
	}

	remisionID := c.Param("id")
	if remisionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "remision_id is required"})
		return
	}

	// Verify it's a remision before deleting
	remision, err := h.invoiceService.GetInvoice(c.Request.Context(), companyID, remisionID)
	if err != nil {
		if err == services.ErrInvoiceNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "remision not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if !remision.IsRemision() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "document is not a remision"})
		return
	}

	err = h.invoiceService.DeleteDraftInvoice(c.Request.Context(), companyID, remisionID)
	if err != nil {
		if err == services.ErrInvoiceNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "remision not found"})
			return
		}
		if err == services.ErrInvoiceNotDraft {
			c.JSON(http.StatusBadRequest, gin.H{"error": "only draft remisiones can be deleted"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "remision deleted successfully"})
}

// FinalizeRemision handles POST /v1/remisiones/:id/finalize
func (h *RemisionHandler) FinalizeRemision(c *gin.Context) {
	companyID := c.GetString("company_id")
	if companyID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "company_id not found in context"})
		return
	}

	remisionID := c.Param("id")
	if remisionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "remision_id is required"})
		return
	}

	// Verify it's a remision
	existingRemision, err := h.invoiceService.GetInvoice(c.Request.Context(), companyID, remisionID)
	if err != nil {
		if err == services.ErrInvoiceNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "remision not found"})
			return
		}
		fmt.Println("we received an error at the handler level... for remision")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if !existingRemision.IsRemision() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "document is not a remision"})
		return
	}

	// TODO: Get user ID from auth context when auth is implemented
	userID := "00000000-0000-0000-0000-000000000000" // Placeholder

	// Finalize remision (generates DTE identifiers)
	remision, err := h.invoiceService.FinalizeRemision(c.Request.Context(), companyID, remisionID, userID)
	if err != nil {
		if err == services.ErrInvoiceNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "remision not found"})
			return
		}
		if err == services.ErrInvoiceNotDraft {
			c.JSON(http.StatusBadRequest, gin.H{"error": "only draft remisiones can be finalized"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	fmt.Printf("\n=== Remision Finalized ===\n")
	fmt.Printf("ID: %s\n", remision.ID)
	fmt.Printf("Número Control: %s\n", *remision.DteNumeroControl)
	fmt.Printf("Type: %s\n", *remision.DteType)

	// ===== DTE PROCESSING (BUILD, SIGN, SUBMIT) =====
	dteServiceInterface, exists := c.Get("dteService")
	if !exists {
		fmt.Println("⚠️  Warning: DTE service not available in context")
		c.JSON(http.StatusOK, remision)
		return
	}

	dteService := dteServiceInterface.(*dte.DTEService)

	fmt.Println("\n=== Starting DTE Processing for Remision (Type 04) ===")

	// Process remision DTE (build, sign, submit to Hacienda)
	signedDTE, err := dteService.ProcessRemision(c.Request.Context(), remision)
	if err != nil {
		// Log the error but don't fail the remision finalization
		fmt.Printf("❌ DTE processing failed: %v\n", err)

		// Update remision status to indicate DTE issue
		dteStatus := "failed_signing"
		remision.DteStatus = &dteStatus

		c.JSON(http.StatusOK, gin.H{
			"remision": remision,
			"warning":  "Remision finalized but DTE processing failed",
			"error":    err.Error(),
		})
		return
	}

	// Successfully processed
	fmt.Printf("✅ DTE signed and submitted successfully for remision %s\n", remision.ID)
	fmt.Printf("Estado: %s\n", signedDTE.Estado)
	fmt.Printf("Código de Generación: %s\n", signedDTE.CodigoGeneracion)

	if signedDTE.SelloRecibido != "" {
		fmt.Printf("Sello Recibido: %s\n", signedDTE.SelloRecibido)
	}

	if signedDTE.FhProcesamiento != "" {
		fmt.Printf("Fecha Procesamiento: %s\n", signedDTE.FhProcesamiento)
	}

	// Update remision with Hacienda response
	dteStatus := signedDTE.Estado
	remision.DteStatus = &dteStatus
	remision.DteSello = &signedDTE.SelloRecibido

	c.JSON(http.StatusOK, gin.H{
		"remision":          remision,
		"hacienda_status":   signedDTE.Estado,
		"sello_recibido":    signedDTE.SelloRecibido,
		"codigo_generacion": signedDTE.CodigoGeneracion,
	})
}

// LinkRemisionToInvoice handles POST /v1/remisiones/:id/link-invoice
func (h *RemisionHandler) LinkRemisionToInvoice(c *gin.Context) {
	companyID := c.GetString("company_id")
	if companyID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "company_id not found in context"})
		return
	}

	remisionID := c.Param("id")
	if remisionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "remision_id is required"})
		return
	}

	var req struct {
		InvoiceID string `json:"invoice_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.invoiceService.LinkRemisionToInvoice(c.Request.Context(), companyID, remisionID, req.InvoiceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":     "remision linked to invoice successfully",
		"remision_id": remisionID,
		"invoice_id":  req.InvoiceID,
	})
}

// GetRemisionLinkedInvoices handles GET /v1/remisiones/:id/invoices
func (h *RemisionHandler) GetRemisionLinkedInvoices(c *gin.Context) {
	companyID := c.GetString("company_id")
	if companyID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "company_id not found in context"})
		return
	}

	remisionID := c.Param("id")
	if remisionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "remision_id is required"})
		return
	}

	invoices, err := h.invoiceService.GetRemisionLinkedInvoices(c.Request.Context(), companyID, remisionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"remision_id": remisionID,
		"invoices":    invoices,
		"count":       len(invoices),
	})
}
