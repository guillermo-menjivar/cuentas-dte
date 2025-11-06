package handlers

import (
	"cuentas/internal/models"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// CreateNotaCreditoRequest - API request to create credit note
type CreateNotaCreditoRequest struct {
	CCFIds            []string                    `json:"ccf_ids" binding:"required,min=1"`
	CreditReason      string                      `json:"credit_reason" binding:"required"`
	CreditDescription string                      `json:"credit_description,omitempty"`
	LineItems         []CreateNotaCreditoLineItem `json:"line_items" binding:"required,min=1"`
	PaymentTerms      string                      `json:"payment_terms,omitempty"`
	Notes             string                      `json:"notes,omitempty"`
}

type CreateNotaCreditoLineItem struct {
	RelatedCCFId     string  `json:"related_ccf_id" binding:"required,uuid"`
	CCFLineItemId    string  `json:"ccf_line_item_id" binding:"required,uuid"`
	QuantityCredited float64 `json:"quantity_credited" binding:"required,gt=0"`
	CreditAmount     float64 `json:"credit_amount" binding:"required,gte=0"`
	CreditReason     string  `json:"credit_reason,omitempty"`
}

// POST /v1/notas/credito
func (h *Handler) CreateNotaCredito(c *gin.Context) {
	var req CreateNotaCreditoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	// Validate credit reason
	validReasons := []string{"void", "return", "discount", "defect",
		"overbilling", "correction", "quality",
		"cancellation", "other"}
	if !contains(validReasons, req.CreditReason) {
		c.JSON(400, gin.H{"error": "invalid credit_reason"})
		return
	}

	// Load and validate CCFs (same as nota débito)
	ccfs, err := h.loadAndValidateCCFs(c.Request.Context(), req.CCFIds)
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	// Validate all CCFs belong to same client
	clientID := ccfs[0].ClientID
	for _, ccf := range ccfs {
		if ccf.ClientID != clientID {
			c.JSON(400, gin.H{"error": "all CCFs must belong to same client"})
			return
		}
	}

	// Validate line items reference valid CCF line items
	for _, lineItem := range req.LineItems {
		if err := h.validateCreditLineItem(c.Request.Context(), lineItem, ccfs); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
	}

	// ⭐ Calculate totals using CCF calculator (SAME as débito!)
	lineItems, totals := h.calculateNotaCreditoTotals(req.LineItems, ccfs)

	// Check if this is a full annulment
	isFullAnnulment := h.isFullAnnulment(lineItems, ccfs)

	// Begin transaction
	tx, _ := h.db.BeginTxx(c.Request.Context(), nil)
	defer tx.Rollback()

	// Generate nota number
	notaNumber, _ := h.generateNotaNumber(c.Request.Context(), tx, "NC", companyID, establishmentID, posID)

	// Create nota credito record
	notaID := uuid.New().String()
	nota := &models.NotaCredito{
		ID:                notaID,
		CompanyID:         companyID,
		EstablishmentID:   establishmentID,
		PointOfSaleID:     posID,
		NotaNumber:        notaNumber,
		NotaType:          "05",
		ClientID:          clientID,
		ClientName:        ccfs[0].ClientName,
		CreditReason:      req.CreditReason,
		CreditDescription: &req.CreditDescription,
		IsFullAnnulment:   isFullAnnulment,
		Subtotal:          totals.Subtotal,
		TotalDiscount:     totals.TotalDiscount,
		TotalTaxes:        totals.TotalTaxes,
		Total:             totals.Total,
		Status:            "draft",
		PaymentTerms:      req.PaymentTerms,
		PaymentMethod:     ccfs[0].PaymentMethod,
		CreatedBy:         getUserID(c),
	}

	// Insert nota
	if err := h.insertNotaCredito(c.Request.Context(), tx, nota); err != nil {
		c.JSON(500, gin.H{"error": "failed to create nota credito"})
		return
	}

	// Insert line items
	for _, item := range lineItems {
		if err := h.insertNotaCreditoLineItem(c.Request.Context(), tx, item); err != nil {
			c.JSON(500, gin.H{"error": "failed to insert line item"})
			return
		}
	}

	// Insert CCF references
	for _, ccf := range ccfs {
		ref := &models.NotaCreditoCCFReference{
			ID:            uuid.New().String(),
			NotaCreditoID: notaID,
			CCFId:         ccf.ID,
			CCFNumber:     ccf.InvoiceNumber,
			CCFDate:       ccf.InvoiceDate,
		}
		if err := h.insertNotaCreditoCCFReference(c.Request.Context(), tx, ref); err != nil {
			c.JSON(500, gin.H{"error": "failed to insert CCF reference"})
			return
		}
	}

	tx.Commit()

	// Load complete nota with relationships
	completeNota, _ := h.getNotaCreditoByID(c.Request.Context(), notaID)

	c.JSON(201, gin.H{"nota_credito": completeNota})
}

// POST /v1/notas/credito/:id/finalize
func (h *Handler) FinalizeNotaCredito(c *gin.Context) {
	notaID := c.Param("id")

	// Load nota
	nota, err := h.getNotaCreditoByID(c.Request.Context(), notaID)
	if err != nil {
		c.JSON(404, gin.H{"error": "nota not found"})
		return
	}

	// Verify status
	if nota.Status != "draft" {
		c.JSON(400, gin.H{"error": "nota must be in draft status"})
		return
	}

	// Generate DTE numero control
	numeroControl, _ := h.generateDTENumeroControl(c.Request.Context(),
		nota.CompanyID, nota.EstablishmentID, nota.PointOfSaleID, "05")

	// Update nota status
	tx, _ := h.db.BeginTxx(c.Request.Context(), nil)
	defer tx.Rollback()

	now := time.Now()
	_, err = tx.ExecContext(c.Request.Context(), `
        UPDATE notas_credito 
        SET status = 'finalized',
            finalized_at = $1,
            dte_numero_control = $2
        WHERE id = $3
    `, now, numeroControl, notaID)

	if err != nil {
		c.JSON(500, gin.H{"error": "failed to finalize nota"})
		return
	}

	tx.Commit()

	// Reload nota
	nota, _ = h.getNotaCreditoByID(c.Request.Context(), notaID)

	// ⭐ Process DTE (submit to Hacienda)
	response, err := h.dteService.ProcessNotaCredito(c.Request.Context(), nota)
	if err != nil {
		c.JSON(500, gin.H{"error": fmt.Sprintf("DTE processing failed: %v", err)})
		return
	}

	// Log success
	log.Printf("✅ SUCCESS! NOTA DE CRÉDITO ACCEPTED BY HACIENDA!")
	log.Printf("Estado: %s", response.Estado)
	log.Printf("Código de Generación: %s", strings.ToUpper(nota.ID))
	log.Printf("Sello Recibido: %s", response.SelloRecibido)
	log.Printf("Fecha Procesamiento: %s", response.FhProcesamiento)

	// Reload final nota
	nota, _ = h.getNotaCreditoByID(c.Request.Context(), notaID)

	c.JSON(200, gin.H{
		"nota_credito": nota,
		"message":      "Nota de crédito processed successfully",
	})
}

func (h *Handler) GetNotaCredito(c *gin.Context) {
	notaID := c.Param("id")

	nota, err := h.getNotaCreditoByID(c.Request.Context(), notaID)
	if err != nil {
		c.JSON(404, gin.H{"error": "nota not found"})
		return
	}

	c.JSON(200, gin.H{"nota_credito": nota})
}
