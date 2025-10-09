package models

import (
	"cuentas/internal/codigos"
	"fmt"
	"time"
)

// InvoicePayment represents a payment made against an invoice
type InvoicePayment struct {
	ID              string    `json:"id"`
	InvoiceID       string    `json:"invoice_id"`
	PaymentDate     time.Time `json:"payment_date"`
	PaymentMethod   string    `json:"payment_method"`
	Amount          float64   `json:"amount"`
	ReferenceNumber *string   `json:"reference_number,omitempty"`
	Notes           *string   `json:"notes,omitempty"`
	CreatedBy       *string   `json:"created_by,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
}

// CreatePaymentRequest represents a request to record a payment
type CreatePaymentRequest struct {
	Amount          float64    `json:"amount" binding:"required"`
	PaymentMethod   string     `json:"payment_method" binding:"required"`
	ReferenceNumber *string    `json:"reference_number"`
	Notes           *string    `json:"notes"`
	PaymentDate     *time.Time `json:"payment_date"`
}

// Validate validates the create payment request
func (r *CreatePaymentRequest) Validate() error {
	if r.Amount <= 0 {
		return fmt.Errorf("amount must be greater than 0")
	}

	// Validate payment method against cat_012
	if !codigos.IsValidPaymentMethod(r.PaymentMethod) {
		return fmt.Errorf("invalid payment_method: must be a valid cat_012 code")
	}

	// Default payment date to now if not provided
	if r.PaymentDate == nil {
		now := time.Now()
		r.PaymentDate = &now
	}

	return nil
}
