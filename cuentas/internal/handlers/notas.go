// internal/handlers/nota_handler.go

package handlers

import (
	"fmt"
	"net/http"

	"cuentas/internal/codigos"
	"cuentas/internal/dte"
	"cuentas/internal/models"
	"cuentas/internal/services"

	"github.com/gin-gonic/gin"
)

// NotaHandler handles nota-related HTTP requests
type NotaHandler struct {
	notaService *services.NotaService
}

// NewNotaHandler creates a new nota handler
func NewNotaHandler(notaService *services.NotaService) *NotaHandler {
	return &NotaHandler{
		notaService: notaService,
	}
}

// CreateNotaDebito handles POST /v1/notas/debito
func (h *NotaHandler) CreateNotaDebito(c *gin.Context) {
	companyID := c.GetString("company_id")
	if companyID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "company_id not found in context"})
		return
	}

	var req models.CreateNotaRequest
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

	nota, err := h.notaService.CreateNota(c.Request.Context(), companyID, codigos.DocTypeNotaDebito, &req)
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

	var req models.CreateNotaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if len(req.RelatedDocuments) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "at least one related document is required"})
		return
	}
	if len(req.RelatedDocuments) > 50 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "maximum 50 related documents allowed"})
		return
	}

	nota, err := h.notaService.CreateNota(c.Request.Context(), companyID, codigos.DocTypeNotaCredito, &req)
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

	var req models.FinalizeNotaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get the nota
	nota, err := h.notaService.GetNota(c.Request.Context(), companyID, notaID)
	if err != nil {
		if err == services.ErrNotaNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "nota not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Validate using Hacienda code
	if nota.Type != codigos.DocTypeNotaDebito {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("nota is not a débito (type: %s)", nota.Type),
		})
		return
	}

	if err := req.Validate(nota.Total); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := "00000000-0000-0000-0000-000000000000"

	nota, err = h.notaService.FinalizeNota(c.Request.Context(), companyID, notaID, userID, &req.Payment)
	if err != nil {
		if err == services.ErrNotaNotDraft {
			c.JSON(http.StatusBadRequest, gin.H{"error": "only draft notas can be finalized"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// DTE Processing
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

	c.JSON(http.StatusOK, nota)
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

	filters := make(map[string]interface{})
	if notaType := c.Query("type"); notaType != "" {
		filters["type"] = notaType
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

	var req models.FinalizeNotaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get the nota
	nota, err := h.notaService.GetNota(c.Request.Context(), companyID, notaID)
	if err != nil {
		if err == services.ErrNotaNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "nota not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Validate using Hacienda code
	if nota.Type != codigos.DocTypeNotaDebito {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("nota is not a débito (type: %s)", nota.Type),
		})
		return
	}

	// ⭐ FIX 1: Remove argument - Validate() takes no params
	if err := req.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := "00000000-0000-0000-0000-000000000000"

	// ⭐ FIX 2: Pass nil - notas don't require payment on finalize
	nota, err = h.notaService.FinalizeNota(c.Request.Context(), companyID, notaID, userID, nil)
	if err != nil {
		if err == services.ErrNotaNotDraft {
			c.JSON(http.StatusBadRequest, gin.H{"error": "only draft notas can be finalized"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// DTE Processing
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

	c.JSON(http.StatusOK, nota)
}

// FinalizeNotaCredito handles POST /v1/notas/credito/:id/finalize
func (h *NotaHandler) FinalizeNotaCredito(c *gin.Context) {
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

	var req models.FinalizeNotaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	nota, err := h.notaService.GetNota(c.Request.Context(), companyID, notaID)
	if err != nil {
		if err == services.ErrNotaNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "nota not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Validate using Hacienda code
	if nota.Type != codigos.DocTypeNotaCredito {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("nota is not a crédito (type: %s)", nota.Type),
		})
		return
	}

	// ⭐ FIX 1: Remove argument
	if err := req.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := "00000000-0000-0000-0000-000000000000"

	// ⭐ FIX 2: Pass nil
	nota, err = h.notaService.FinalizeNota(c.Request.Context(), companyID, notaID, userID, nil)
	if err != nil {
		if err == services.ErrNotaNotDraft {
			c.JSON(http.StatusBadRequest, gin.H{"error": "only draft notas can be finalized"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// DTE Processing
	dteServiceInterface, exists := c.Get("dteService")
	if exists {
		dteService := dteServiceInterface.(*dte.DTEService)

		fmt.Println("\n=== Starting Nota de Crédito DTE Processing ===")
		haciendaResponse, err := dteService.ProcessNotaCredito(c.Request.Context(), nota)
		if err != nil {
			fmt.Printf("❌ DTE processing failed: %v\n", err)
			dteStatus := "failed_signing"
			nota.DteStatus = &dteStatus
		} else {
			fmt.Printf("✅ Nota de Crédito accepted by Hacienda!\n")
			fmt.Printf("Sello: %s\n", haciendaResponse.SelloRecibido)

			dteStatus := "signed"
			nota.DteStatus = &dteStatus
		}
	}

	c.JSON(http.StatusOK, nota)
}
