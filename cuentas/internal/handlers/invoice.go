package handlers

import (
	"net/http"

	"cuentas/internal/models"
	"cuentas/internal/services"

	"github.com/gin-gonic/gin"
)

type InvoiceHandler struct {
	invoiceService *services.InvoiceService
}

func NewInvoiceHandler() *InvoiceHandler {
	return &InvoiceHandler{
		invoiceService: services.NewInvoiceService(),
	}
}

// CreateInvoice handles POST /api/v1/invoices
func (h *InvoiceHandler) CreateInvoice(c *gin.Context) {
	companyID := c.GetString("company_id")
	if companyID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "company_id not found in context"})
		return
	}

	var req models.CreateInvoiceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	invoice, err := h.invoiceService.CreateInvoice(c.Request.Context(), companyID, &req)
	if err != nil {
		if err == services.ErrClientNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "client not found"})
			return
		}
		if err == services.ErrInventoryItemNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "inventory item not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, invoice)
}

// GetInvoice handles GET /api/v1/invoices/:id
func (h *InvoiceHandler) GetInvoice(c *gin.Context) {
	companyID := c.GetString("company_id")
	if companyID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "company_id not found in context"})
		return
	}

	invoiceID := c.Param("id")
	if invoiceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invoice_id is required"})
		return
	}

	invoice, err := h.invoiceService.GetInvoice(c.Request.Context(), companyID, invoiceID)
	if err != nil {
		if err == services.ErrInvoiceNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "invoice not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, invoice)
}

// ListInvoices handles GET /api/v1/invoices
func (h *InvoiceHandler) ListInvoices(c *gin.Context) {
	companyID := c.GetString("company_id")
	if companyID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "company_id not found in context"})
		return
	}

	// Build filters from query params
	filters := make(map[string]interface{})
	if status := c.Query("status"); status != "" {
		filters["status"] = status
	}
	if clientID := c.Query("client_id"); clientID != "" {
		filters["client_id"] = clientID
	}
	if paymentStatus := c.Query("payment_status"); paymentStatus != "" {
		filters["payment_status"] = paymentStatus
	}
	if establishmentID := c.Query("establishment_id"); establishmentID != "" {
		filters["establishment_id"] = establishmentID
	}
	if posID := c.Query("point_of_sale_id"); posID != "" {
		filters["point_of_sale_id"] = posID
	}

	invoices, err := h.invoiceService.ListInvoices(c.Request.Context(), companyID, filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"invoices": invoices,
		"count":    len(invoices),
	})
}

// DeleteInvoice handles DELETE /api/v1/invoices/:id
func (h *InvoiceHandler) DeleteInvoice(c *gin.Context) {
	companyID := c.GetString("company_id")
	if companyID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "company_id not found in context"})
		return
	}

	invoiceID := c.Param("id")
	if invoiceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invoice_id is required"})
		return
	}

	err := h.invoiceService.DeleteDraftInvoice(c.Request.Context(), companyID, invoiceID)
	if err != nil {
		if err == services.ErrInvoiceNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "invoice not found"})
			return
		}
		if err == services.ErrInvoiceNotDraft {
			c.JSON(http.StatusBadRequest, gin.H{"error": "only draft invoices can be deleted"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "invoice deleted successfully"})
}

// FinalizeInvoice handles POST /v1/invoices/:id/finalize
func (h *InvoiceHandler) FinalizeInvoice(c *gin.Context) {
	companyID := c.GetString("company_id")
	if companyID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "company_id not found in context"})
		return
	}

	invoiceID := c.Param("id")
	if invoiceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invoice_id is required"})
		return
	}

	invoice, err := h.invoiceService.FinalizeInvoice(c.Request.Context(), companyID, invoiceID)
	if err != nil {
		if err == services.ErrInvoiceNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "invoice not found"})
			return
		}
		if err == services.ErrInvoiceNotDraft {
			c.JSON(http.StatusBadRequest, gin.H{"error": "only draft invoices can be finalized"})
			return
		}
		if err == services.ErrCreditLimitExceeded {
			c.JSON(http.StatusBadRequest, gin.H{"error": "credit limit exceeded"})
			return
		}
		if err == services.ErrCreditSuspended {
			c.JSON(http.StatusBadRequest, gin.H{"error": "client credit is suspended"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, invoice)
}
