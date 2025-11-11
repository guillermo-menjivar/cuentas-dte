package models

import (
	"fmt"
	"strings"
	"time"
)

// ============================================
// REMISION REQUEST MODELS
// Add these to internal/models/invoice.go
// ============================================

// RelatedDocumentInput represents a related document when creating/updating
type RelatedDocumentInput struct {
	TipoDocumento   string    `json:"tipo_documento" binding:"required"`   // "01" or "03"
	TipoGeneracion  int       `json:"tipo_generacion" binding:"required"`  // 1=physical, 2=electronic
	NumeroDocumento string    `json:"numero_documento" binding:"required"` // UUID or correlativo
	FechaEmision    time.Time `json:"fecha_emision" binding:"required"`
}

// Validate validates a related document input
func (r *RelatedDocumentInput) Validate() error {
	// Validate tipo_documento
	if r.TipoDocumento != "01" && r.TipoDocumento != "03" {
		return fmt.Errorf("tipo_documento must be '01' or '03', got: %s", r.TipoDocumento)
	}

	// Validate tipo_generacion
	if r.TipoGeneracion != 1 && r.TipoGeneracion != 2 {
		return fmt.Errorf("tipo_generacion must be 1 or 2, got: %d", r.TipoGeneracion)
	}

	// Validate numero_documento format if electronic
	if r.TipoGeneracion == 2 {
		// Must be UUID format
		if len(r.NumeroDocumento) != 36 {
			return fmt.Errorf("electronic document number must be UUID format (36 chars)")
		}
	}

	// Validate fecha_emision is not in future
	if r.FechaEmision.After(time.Now()) {
		return fmt.Errorf("fecha_emision cannot be in the future")
	}

	return nil
}

// CreateRemisionRequest represents the request to create a remision (Type 04)
type CreateRemisionRequest struct {
	// Required fields
	EstablishmentID string                         `json:"establishment_id" binding:"required"`
	PointOfSaleID   string                         `json:"point_of_sale_id" binding:"required"`
	LineItems       []CreateInvoiceLineItemRequest `json:"line_items" binding:"required,min=1"`
	RemisionType    string                         `json:"remision_type" binding:"required"`

	// Optional fields
	ClientID         *string                `json:"client_id,omitempty"`         // Null for internal transfers
	DeliveryPerson   *string                `json:"delivery_person,omitempty"`   // Driver name
	VehiclePlate     *string                `json:"vehicle_plate,omitempty"`     // Vehicle identification
	DeliveryNotes    *string                `json:"delivery_notes,omitempty"`    // Transport notes
	Notes            *string                `json:"notes,omitempty"`             // General notes
	RelatedDocuments []RelatedDocumentInput `json:"related_documents,omitempty"` // References to invoices
}

// Validate validates the create remision request
func (r *CreateRemisionRequest) Validate() error {
	// Validate establishment_id
	if strings.TrimSpace(r.EstablishmentID) == "" {
		return fmt.Errorf("establishment_id is required")
	}

	// Validate point_of_sale_id
	if strings.TrimSpace(r.PointOfSaleID) == "" {
		return fmt.Errorf("point_of_sale_id is required")
	}

	// Validate remision_type
	validTypes := []string{"pre_invoice_delivery", "inter_branch_transfer", "route_sales", "other"}
	if !contains(validTypes, r.RemisionType) {
		return fmt.Errorf("invalid remision_type: must be one of %v", validTypes)
	}

	// Validate line items
	if len(r.LineItems) == 0 {
		return fmt.Errorf("at least one line item is required")
	}

	for i, item := range r.LineItems {
		if err := item.Validate(); err != nil {
			return fmt.Errorf("line item %d: %w", i+1, err)
		}
	}

	// Validate related documents
	if len(r.RelatedDocuments) > 50 {
		return fmt.Errorf("maximum 50 related documents allowed, got %d", len(r.RelatedDocuments))
	}

	for i, doc := range r.RelatedDocuments {
		if err := doc.Validate(); err != nil {
			return fmt.Errorf("related document %d: %w", i+1, err)
		}
	}

	// Business rule: inter_branch_transfer should not have external client
	if r.RemisionType == "inter_branch_transfer" && r.ClientID != nil && *r.ClientID != "" {
		return fmt.Errorf("inter_branch_transfer should not have a client_id (internal transfer only)")
	}

	// Business rule: pre_invoice_delivery and route_sales should have receptor
	if (r.RemisionType == "pre_invoice_delivery" || r.RemisionType == "route_sales") && (r.ClientID == nil || *r.ClientID == "") {
		return fmt.Errorf("%s requires a client_id", r.RemisionType)
	}

	return nil
}

// FinalizeRemisionRequest represents the request to finalize a remision
// Remisiones don't require payment info since they're not sales
type FinalizeRemisionRequest struct {
	// No payment required for remisiones
	// This struct is here for API consistency but has no fields
}

// Validate validates the finalize remision request
func (r *FinalizeRemisionRequest) Validate() error {
	// Nothing to validate - remisiones have no payment
	return nil
}

// ============================================
// HELPER: Add RemisionInvoiceLink model
// ============================================

// RemisionInvoiceLink tracks which invoices reference which remisiones (for route sales)
type RemisionInvoiceLink struct {
	ID         string    `json:"id"`
	RemisionID string    `json:"remision_id"`
	InvoiceID  string    `json:"invoice_id"`
	CreatedAt  time.Time `json:"created_at"`
}
