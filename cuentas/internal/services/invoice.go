package services

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"cuentas/internal/database"
	"cuentas/internal/models"
)

type InvoiceService struct{}

func NewInvoiceService() *InvoiceService {
	return &InvoiceService{}
}

// CreateInvoice creates a new draft invoice with complete snapshots
func (s *InvoiceService) CreateInvoice(ctx context.Context, companyID string, req *models.CreateInvoiceRequest) (*models.Invoice, error) {
	// Validate request
	if err := req.Validate(); err != nil {
		return nil, err
	}

	// Begin transaction
	tx, err := database.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// 1. Snapshot client data
	client, err := s.snapshotClient(ctx, tx, companyID, req.ClientID)
	if err != nil {
		return nil, err
	}

	// 2. Generate invoice number
	invoiceNumber, err := s.generateInvoiceNumber(ctx, tx, companyID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate invoice number: %w", err)
	}

	// 3. Process line items and calculate totals
	lineItems, subtotal, totalDiscount, totalTaxes, err := s.processLineItems(ctx, tx, companyID, req.LineItems)
	if err != nil {
		return nil, err
	}

	total := subtotal - totalDiscount + totalTaxes

	// 4. Calculate due date if needed
	var dueDate *time.Time
	if req.DueDate != nil {
		dueDate = req.DueDate
	} else if req.PaymentTerms == "net_30" {
		date := time.Now().AddDate(0, 0, 30)
		dueDate = &date
	} else if req.PaymentTerms == "net_60" {
		date := time.Now().AddDate(0, 0, 60)
		dueDate = &date
	}

	// 5. Create invoice record
	invoice := &models.Invoice{
		CompanyID:               companyID,
		ClientID:                req.ClientID,
		InvoiceNumber:           invoiceNumber,
		InvoiceType:             "sale",
		ClientName:              client.ClientName,
		ClientLegalName:         client.ClientLegalName,
		ClientNit:               client.ClientNit,
		ClientNcr:               client.ClientNcr,
		ClientDui:               client.ClientDui,
		ClientAddress:           client.ClientAddress,
		ClientTipoContribuyente: client.ClientTipoContribuyente,
		ClientTipoPersona:       client.ClientTipoPersona,
		Subtotal:                subtotal,
		TotalDiscount:           totalDiscount,
		TotalTaxes:              totalTaxes,
		Total:                   total,
		Currency:                "USD",
		PaymentTerms:            req.PaymentTerms,
		PaymentStatus:           "unpaid",
		AmountPaid:              0,
		BalanceDue:              total,
		DueDate:                 dueDate,
		Status:                  "draft",
		Notes:                   req.Notes,
		ContactEmail:            req.ContactEmail,
		ContactWhatsapp:         req.ContactWhatsapp,
		CreatedAt:               time.Now(),
	}

	// 6. Insert invoice
	invoiceID, err := s.insertInvoice(ctx, tx, invoice)
	if err != nil {
		return nil, fmt.Errorf("failed to insert invoice: %w", err)
	}
	invoice.ID = invoiceID

	// 7. Insert line items and taxes
	for i, lineItem := range lineItems {
		lineItem.InvoiceID = invoiceID
		lineItem.LineNumber = i + 1

		lineItemID, err := s.insertLineItem(ctx, tx, &lineItem)
		if err != nil {
			return nil, fmt.Errorf("failed to insert line item %d: %w", i+1, err)
		}
		lineItem.ID = lineItemID

		// Insert taxes for this line item
		for _, tax := range lineItem.Taxes {
			tax.LineItemID = lineItemID
			if err := s.insertLineItemTax(ctx, tx, &tax); err != nil {
				return nil, fmt.Errorf("failed to insert tax for line item %d: %w", i+1, err)
			}
		}
	}

	// 8. Commit transaction
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// 9. Attach line items to invoice
	invoice.LineItems = lineItems

	return invoice, nil
}

// ClientSnapshot represents the snapshot of client data at transaction time
type ClientSnapshot struct {
	ClientName              string
	ClientLegalName         string
	ClientNit               *string
	ClientNcr               *string
	ClientDui               *string
	ClientAddress           string
	ClientTipoContribuyente *string
	ClientTipoPersona       *string
}

// snapshotClient retrieves and snapshots client data
func (s *InvoiceService) snapshotClient(ctx context.Context, tx *sql.Tx, companyID, clientID string) (*ClientSnapshot, error) {
	query := `
		SELECT 
			business_name,
			legal_business_name,
			nit,
			ncr,
			dui,
			full_address,
			tipo_contribuyente,
			tipo_persona
		FROM clients
		WHERE id = $1 AND company_id = $2 AND active = true
	`

	var snapshot ClientSnapshot
	err := tx.QueryRowContext(ctx, query, clientID, companyID).Scan(
		&snapshot.ClientName,
		&snapshot.ClientLegalName,
		&snapshot.ClientNit,
		&snapshot.ClientNcr,
		&snapshot.ClientDui,
		&snapshot.ClientAddress,
		&snapshot.ClientTipoContribuyente,
		&snapshot.ClientTipoPersona,
	)

	if err == sql.ErrNoRows {
		return nil, ErrClientNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query client: %w", err)
	}

	return &snapshot, nil
}

// generateInvoiceNumber generates a sequential invoice number
func (s *InvoiceService) generateInvoiceNumber(ctx context.Context, tx *sql.Tx, companyID string) (string, error) {
	// Get the last invoice number for this company
	var lastNumber sql.NullString
	query := `
		SELECT invoice_number 
		FROM invoices 
		WHERE company_id = $1 
		ORDER BY created_at DESC 
		LIMIT 1
	`
	err := tx.QueryRowContext(ctx, query, companyID).Scan(&lastNumber)
	if err != nil && err != sql.ErrNoRows {
		return "", fmt.Errorf("failed to query last invoice number: %w", err)
	}

	// Parse sequence or start at 1
	var sequence int64 = 1
	if lastNumber.Valid {
		// Format: INV-2025-00001
		// Extract the sequence number (last part)
		var year int
		fmt.Sscanf(lastNumber.String, "INV-%d-%d", &year, &sequence)
		sequence++
	}

	// Generate new number
	currentYear := time.Now().Year()
	invoiceNumber := fmt.Sprintf("INV-%d-%05d", currentYear, sequence)

	return invoiceNumber, nil
}
