package handlers

import (
	"cuentas/internal/dte"
	"cuentas/internal/models"
	"cuentas/internal/services"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

// internal/handlers/invoice_handler.go

// FinalizeNotaDebito handles POST /v1/invoices/:id/finalize-nota-debito
func (h *InvoiceHandler) FinalizeNotaDebito(c *gin.Context) {
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

	// Parse request body with payment info (same as regular invoice)
	var req models.FinalizeInvoiceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get the invoice first
	existingInvoice, err := h.invoiceService.GetInvoice(c.Request.Context(), companyID, invoiceID)
	if err != nil {
		if err == services.ErrInvoiceNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "invoice not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Validate invoice type is Nota de Débito (type 6)
	if existingInvoice.InvoiceType != 6 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("invoice is not a Nota de Débito (type: %d)", existingInvoice.InvoiceType),
		})
		return
	}

	// Validate related documents exist
	if len(existingInvoice.RelatedDocuments) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Nota de Débito requires at least one related document",
		})
		return
	}

	// Validate payment against invoice total
	if err := req.Validate(existingInvoice.Total); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// TODO: Get user ID from auth context when auth is implemented
	userID := "00000000-0000-0000-0000-000000000000" // Placeholder

	// Finalize invoice with payment info
	invoice, err := h.invoiceService.FinalizeInvoice(c.Request.Context(), companyID, invoiceID, userID, &req.Payment)
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

	// ===== DTE PROCESSING FOR NOTA DE DÉBITO =====
	dteServiceInterface, exists := c.Get("dteService")
	if exists {
		dteService := dteServiceInterface.(*dte.DTEService)

		fmt.Println("\n=== Starting Nota de Débito DTE Processing ===")
		haciendaResponse, err := dteService.ProcessNotaDebito(c.Request.Context(), invoice)
		if err != nil {
			// Log the error but don't fail the invoice finalization
			fmt.Printf("❌ Nota de Débito DTE processing failed: %v\n", err)
			// Update invoice status to indicate DTE issue
			dteStatus := "failed_signing"
			invoice.DteStatus = &dteStatus
		} else {
			// Successfully processed
			fmt.Printf("✅ Nota de Débito DTE processed successfully for invoice %s\n", invoice.ID)

			fmt.Printf("Estado: %s\n", haciendaResponse.Estado)
			fmt.Printf("Código de Generación: %s\n", haciendaResponse.CodigoGeneracion)
			if haciendaResponse.SelloRecibido != "" {
				fmt.Printf("Sello Recibido: %s\n", haciendaResponse.SelloRecibido)
			}

			dteStatus := "signed"
			invoice.DteStatus = &dteStatus

			// Response is already saved in ProcessNotaDebito
			_ = haciendaResponse
		}
	} else {
		fmt.Println("⚠️  Warning: DTE service not available in context")
	}
	// ===== END DTE PROCESSING =====

	c.JSON(http.StatusOK, invoice)
}
