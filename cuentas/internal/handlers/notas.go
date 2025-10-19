// internal/handlers/nota_handler.go

package handlers

import (
	"fmt"
	"net/http"

	"cuentas/internal/dte"
	"cuentas/internal/models"
	"cuentas/internal/services"

	"github.com/gin-gonic/gin"
)

type NotaHandler struct {
	notaService *services.NotaService
}

func NewNotaHandler() *NotaHandler {
	return &NotaHandler{
		notaService: services.NewNotaService(),
	}
}

// CreateNotaDebito handles POST /v1/notas/debito
func (h *NotaHandler) CreateNotaDebito(c *gin.Context) {
	companyID := c.GetString("company_id")
	if companyID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "company_id not found in context"})
		return
	}

	var req models.CreateNotaDebitoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate related documents
	if len(req.RelatedDocuments) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "at least one related document is required"})
		return
	}
	if len(req.RelatedDocuments) > 50 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "maximum 50 related documents allowed"})
		return
	}

	nota, err := h.notaService.CreateNotaDebito(c.Request.Context(), companyID, &req)
	if err != nil {
		if err == services.ErrClientNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "client not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, nota)
}

// CreateNotaCredito handles POST /v1/notas/credito
func (h *NotaHandler) CreateNotaCredito(c *gin.Context) {
	companyID := c.GetString("company_id")
	if companyID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "company_id not found in context"})
		return
	}

	var req models.CreateNotaCreditoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate related documents
	if len(req.RelatedDocuments) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "at least one related document is required"})
		return
	}
	if len(req.RelatedDocuments) > 50 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "maximum 50 related documents allowed"})
		return
	}

	nota, err := h.notaService.CreateNotaCredito(c.Request.Context(), companyID, &req)
	if err != nil {
		if err == services.ErrClientNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "client not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, nota)
}

// FinalizeNotaDebito handles POST /v1/notas/debito/:id/finalize
func (h *NotaHandler) FinalizeNotaDebito(c *gin.Context) {
	companyID := c.GetString("company_id")
	if companyID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "company_id not found in context"})
		return
	}

	notaID := c.Param("id")
	if notaID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "nota_id is required"})
		return
	}

	// Parse payment info
	var req models.FinalizeNotaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get the nota first
	nota, err := h.notaService.GetNota(c.Request.Context(), companyID, notaID)
	if err != nil {
		if err == services.ErrNotaNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "nota not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Validate it's a Nota de Débito
	if nota.Type != "debito" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("nota is not a débito (type: %s)", nota.Type),
		})
		return
	}

	// Validate payment
	if err := req.Validate(nota.Total); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// TODO: Get user ID from auth
	userID := "00000000-0000-0000-0000-000000000000"

	// Finalize nota
	nota, err = h.notaService.FinalizeNota(c.Request.Context(), companyID, notaID, userID, &req.Payment)
	if err != nil {
		if err == services.ErrNotaNotDraft {
			c.JSON(http.StatusBadRequest, gin.H{"error": "only draft notas can be finalized"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// ===== DTE PROCESSING =====
	dteServiceInterface, exists := c.Get("dteService")
	if exists {
		dteService := dteServiceInterface.(*dte.DTEService)

		fmt.Println("\n=== Starting Nota de Débito DTE Processing ===")
		haciendaResponse, err := dteService.ProcessNotaDebito(c.Request.Context(), nota)
		if err != nil {
			fmt.Printf("❌ DTE processing failed: %v\n", err)
			dteStatus := "failed_signing"
			nota.DteStatus = &dteStatus
		} else {
			fmt.Printf("✅ Nota de Débito accepted by Hacienda!\n")
			fmt.Printf("Sello: %s\n", haciendaResponse.SelloRecibido)

			dteStatus := "signed"
			nota.DteStatus = &dteStatus
		}
	}
	// ===== END DTE PROCESSING =====

	c.JSON(http.StatusOK, nota)
}

// FinalizeNotaCredito handles POST /v1/notas/credito/:id/finalize
func (h *NotaHandler) FinalizeNotaCredito(c *gin.Context) {
	// Same structure as FinalizeNotaDebito but calls ProcessNotaCredito
	// ... (implement similarly)
}

// GetNota handles GET /v1/notas/:id
func (h *NotaHandler) GetNota(c *gin.Context) {
	companyID := c.GetString("company_id")
	if companyID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "company_id not found in context"})
		return
	}

	notaID := c.Param("id")
	nota, err := h.notaService.GetNota(c.Request.Context(), companyID, notaID)
	if err != nil {
		if err == services.ErrNotaNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "nota not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, nota)
}

// ListNotas handles GET /v1/notas
func (h *NotaHandler) ListNotas(c *gin.Context) {
	companyID := c.GetString("company_id")
	if companyID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "company_id not found in context"})
		return
	}

	// Filter by type if specified
	filters := make(map[string]interface{})
	if notaType := c.Query("type"); notaType != "" {
		filters["type"] = notaType // "debito" or "credito"
	}
	if status := c.Query("status"); status != "" {
		filters["status"] = status
	}

	notas, err := h.notaService.ListNotas(c.Request.Context(), companyID, filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"notas": notas,
		"count": len(notas),
	})
}

// DeleteNota handles DELETE /v1/notas/:id
func (h *NotaHandler) DeleteNota(c *gin.Context) {
	companyID := c.GetString("company_id")
	if companyID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "company_id not found in context"})
		return
	}

	notaID := c.Param("id")
	err := h.notaService.DeleteDraftNota(c.Request.Context(), companyID, notaID)
	if err != nil {
		if err == services.ErrNotaNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "nota not found"})
			return
		}
		if err == services.ErrNotaNotDraft {
			c.JSON(http.StatusBadRequest, gin.H{"error": "only draft notas can be deleted"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "nota deleted successfully"})
}
