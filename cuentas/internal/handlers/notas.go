package handlers

import (
	"cuentas/internal/models"
	"cuentas/internal/services"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

type NotaHandler struct {
	notaService    *services.NotaService
	invoiceService *services.InvoiceService
}

// NewNotaHandler creates a new nota handler
func NewNotaHandler(notaService *services.NotaService, invoiceService *services.InvoiceService) *NotaHandler {
	return &NotaHandler{
		notaService:    notaService,
		invoiceService: invoiceService,
	}
}

// CreateNotaDebito creates a new Nota de Débito
func (h *NotaHandler) CreateNotaDebito(c *gin.Context) {
	companyID := c.GetString("company_id")
	if companyID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "company_id not found in context"})
		return
	}

	var request models.CreateNotaDebitoRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	fmt.Println("📥 Received Nota de Débito request:")
	fmt.Printf("   CCFs: %v\n", request.CCFIds)
	fmt.Printf("   Line Items: %d\n", len(request.LineItems))

	// Create the nota
	nota, err := h.notaService.CreateNotaDebito(
		c.Request.Context(),
		companyID,
		&request,
		h.invoiceService,
	)
	if err != nil {
		// Determine appropriate status code based on error type
		statusCode := http.StatusInternalServerError

		// Check for validation errors
		if isValidationError(err) {
			statusCode = http.StatusBadRequest
		}

		c.JSON(statusCode, gin.H{"error": err.Error()})
		return
	}

	// Return the created nota
	c.JSON(http.StatusCreated, gin.H{
		"message": "Nota de Débito created successfully",
		"nota":    nota,
	})
}

// isValidationError checks if the error is a validation error
func isValidationError(err error) bool {
	errMsg := err.Error()

	// List of validation error prefixes
	validationPrefixes := []string{
		"validation failed",
		"at least one",
		"maximum",
		"duplicate",
		"not found",
		"is not a CCF",
		"is not finalized",
		"has been voided",
		"must belong to the same client",
		"references CCF",
		"adjustment_amount must be",
		"line item",
	}

	for _, prefix := range validationPrefixes {
		if len(errMsg) >= len(prefix) && errMsg[:len(prefix)] == prefix {
			return true
		}
	}

	return false
}
