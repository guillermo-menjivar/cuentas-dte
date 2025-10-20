package handlers

import (
	"cuentas/internal/services"
	"net/http"

	"github.com/gin-gonic/gin"
)

type NotasHandler struct {
	invoiceService *service.InvoiceService
}

func NewNotasHandler() *NoasHandler {
	return &NoasHandler{
		invoiceService: services.NewInvoiceService(),
	}
}

func (n *NotasHandler) CreateNota(c *gin.Context) {
	companyID := c.GetString("company_id")

	if companyID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "company_id not found in context"})
	}

	invoice, err := h.invoiceService.GetInvoice(c.Request.Context(), companyID, invoiceID)
}

/*
 CCF exists
 CCF is type "03"
 CCF belongs to same client
 CCF is finalized (not draft/voided)

Line Item Validations:

 If item SKU exists in original CCF:

 Quantity matches (or is sensible)
 Price adjustment is positive (débito only increases)
 Unit of measure matches


 If item SKU is NEW (not in CCF):

 Item exists in company's inventory
 Item is active
 Item details are valid


 related_document_ref matches a document in related_documents[]
 Item type is valid (1-4)
 Unit of measure is valid

Business Rules:

 Total nota amount isn't absurdly large (>100% of original?)
 Not too many line items (max 2000)
 All prices are positive
 All quantities are positive
    /*
	// lets make sure the request sent to us is a CCF
	// lets make sure the CCF exist and is finalized
	// inspect if there are items in nota de debito exist in the CCF
	// inspect same client and other attributes match with CCF
	// if item is found and or new items amek sure the total aligns
	// create the nota de debito
	// submit to hacienda

*/
