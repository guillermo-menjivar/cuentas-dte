import (
	"fmt"
	"net/http"

	"cuentas/internal/dte"
	"cuentas/internal/models"

	"github.com/gin-gonic/gin"
)

// CreateNotaCredito creates a new Nota de Crédito
func (h *NotaHandler) CreateNotaCredito(c *gin.Context) {
	companyID := c.GetString("company_id")
	if companyID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "company_id not found in context"})
		return
	}

	var request models.CreateNotaCreditoRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	nota, err := h.notaCreditoService.CreateNotaCredito(
		c.Request.Context(),
		companyID,
		&request,
		h.invoiceService,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"nota": nota,
	})
}

// GetNotaCredito retrieves a single nota by ID
func (h *NotaHandler) GetNotaCredito(c *gin.Context) {
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

	// Get the nota
	nota, err := h.notaCreditoService.GetNotaCredito(
		c.Request.Context(),
		notaID,
		companyID,
	)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if err.Error() == "nota not found" {
			statusCode = http.StatusNotFound
		}

		c.JSON(statusCode, gin.H{"error": err.Error()})
		return
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

	// Finalize the nota (generates numero control, updates status)
	nota, err := h.notaCreditoService.FinalizeNotaCredito(
		c.Request.Context(),
		notaID,
		companyID,
	)
	if err != nil {
		if err.Error() == "nota not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "nota not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Process DTE (build, sign, and submit to Hacienda) - same pattern as invoice
	dteServiceInterface, exists := c.Get("dteService")
	if exists {
		dteService := dteServiceInterface.(*dte.DTEService)

		fmt.Println("\n=== Starting DTE Processing for Nota de Crédito ===")
		response, err := dteService.ProcessNotaCredito(c.Request.Context(), nota)
		if err != nil {
			// Log the error but don't fail the finalization
			fmt.Printf("❌ DTE processing failed: %v\n", err)
			dteStatus := "failed_signing"
			nota.DteStatus = &dteStatus
		} else {
			// Successfully signed and submitted
			fmt.Printf("✅ DTE processed successfully for nota %s\n", nota.ID)
			fmt.Printf("Estado: %s\n", response.Estado)
			fmt.Printf("Código de Generación: %s\n", response.CodigoGeneracion)
			if response.SelloRecibido != "" {
				fmt.Printf("Sello Recibido: %s\n", response.SelloRecibido)
			}

			dteStatus := "submitted"
			nota.DteStatus = &dteStatus
			nota.DteSelloRecibido = &response.SelloRecibido
		}
	} else {
		fmt.Println("⚠️  Warning: DTE service not available in context")
	}

	c.JSON(http.StatusOK, gin.H{
		"nota": nota,
	})
}
