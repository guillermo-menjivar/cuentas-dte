package services

import "errors"

// Invoice service errors
var (
	ErrClientNotFound        = errors.New("client not found")
	ErrInventoryItemNotFound = errors.New("inventory item not found")
	ErrInvoiceNotFound       = errors.New("invoice not found")
	ErrInvoiceNotDraft       = errors.New("invoice is not in draft status")
	ErrInvoiceAlreadyVoid    = errors.New("invoice is already void")
	ErrInsufficientPayment   = errors.New("payment amount exceeds balance due")
	ErrCreditLimitExceeded   = errors.New("client credit limit exceeded")
	ErrInvalidInvoiceStatus  = errors.New("invalid invoice status for this operation")
)
