package services

import (
	"context"
	"cuentas/internal/codigos"
	"cuentas/internal/database"
	"cuentas/internal/models"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Add these methods to internal/services/invoice_service.go

// ============================================
// EXPORT INVOICE METHODS
// ============================================

// insertExportDocuments inserts export documents for a Type 11 invoice
func (s *InvoiceService) insertExportDocuments(
	ctx context.Context,
	tx *sql.Tx,
	invoiceID string,
	docs []models.CreateInvoiceExportDocumentRequest,
) error {
	if len(docs) == 0 {
		return nil
	}

	query := `
		INSERT INTO invoice_export_documents (
			invoice_id, cod_doc_asociado, desc_documento, detalle_documento,
			placa_trans, modo_transp, num_conductor, nombre_conductor,
			created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	for _, doc := range docs {
		_, err := tx.ExecContext(ctx, query,
			invoiceID,
			doc.CodDocAsociado,
			doc.DescDocumento,
			doc.DetalleDocumento,
			doc.PlacaTrans,
			doc.ModoTransp,
			doc.NumConductor,
			doc.NombreConductor,
			time.Now(),
		)
		if err != nil {
			return fmt.Errorf("failed to insert export document: %w", err)
		}
	}

	return nil
}

// getExportDocuments retrieves export documents for an invoice
func (s *InvoiceService) getExportDocuments(
	ctx context.Context,
	invoiceID string,
) ([]models.InvoiceExportDocument, error) {
	query := `
		SELECT
			id, invoice_id, cod_doc_asociado, desc_documento, detalle_documento,
			placa_trans, modo_transp, num_conductor, nombre_conductor,
			created_at
		FROM invoice_export_documents
		WHERE invoice_id = $1
		ORDER BY cod_doc_asociado
	`

	rows, err := database.DB.QueryContext(ctx, query, invoiceID)
	if err != nil {
		return nil, fmt.Errorf("failed to query export documents: %w", err)
	}
	defer rows.Close()

	var docs []models.InvoiceExportDocument
	for rows.Next() {
		var doc models.InvoiceExportDocument
		err := rows.Scan(
			&doc.ID,
			&doc.InvoiceID,
			&doc.CodDocAsociado,
			&doc.DescDocumento,
			&doc.DetalleDocumento,
			&doc.PlacaTrans,
			&doc.ModoTransp,
			&doc.NumConductor,
			&doc.NombreConductor,
			&doc.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan export document: %w", err)
		}
		docs = append(docs, doc)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating export documents: %w", err)
	}

	return docs, nil
}

// GetInvoiceExport retrieves a complete export invoice with all fields and documents
// Use this for Type 11 export invoices to get export-specific fields
func (s *InvoiceService) GetInvoiceExport(ctx context.Context, companyID, invoiceID string) (*models.Invoice, error) {
	// Get invoice header with export fields
	invoice, err := s.getInvoiceHeaderExport(ctx, companyID, invoiceID)
	if err != nil {
		return nil, err
	}

	// Get line items
	lineItems, err := s.getLineItems(ctx, invoiceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get line items: %w", err)
	}

	// Get taxes for each line item
	for i := range lineItems {
		taxes, err := s.getLineItemTaxes(ctx, lineItems[i].ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get taxes for line item: %w", err)
		}
		lineItems[i].Taxes = taxes
	}

	invoice.LineItems = lineItems

	// Load export documents
	if invoice.IsExportInvoice() {
		exportDocs, err := s.getExportDocuments(ctx, invoiceID)
		if err != nil {
			return nil, fmt.Errorf("failed to get export documents: %w", err)
		}
		invoice.ExportDocuments = exportDocs
	}

	return invoice, nil
}

// processLineItemsExport processes line items for export invoices (Type 11 - 0% IVA)
func (s *InvoiceService) processLineItemsExport(
	ctx context.Context,
	tx *sql.Tx,
	companyID string,
	reqItems []models.CreateInvoiceLineItemRequest,
) ([]models.InvoiceLineItem, float64, float64, float64, error) {
	var lineItems []models.InvoiceLineItem
	var subtotal, totalDiscount, totalTaxes float64

	for _, reqItem := range reqItems {
		// 1. Snapshot inventory item
		item, err := s.snapshotInventoryItem(ctx, tx, companyID, reqItem.ItemID)
		if err != nil {
			return nil, 0, 0, 0, err
		}

		// 2. Calculate line amounts with 0% IVA (export rate)
		lineSubtotal := round(item.UnitPrice * reqItem.Quantity)
		discountAmount := round(lineSubtotal * (reqItem.DiscountPercentage / 100))
		taxableAmount := round(lineSubtotal - discountAmount)

		// 3. Get taxes for this item - should be tributo C3 (0% export)
		taxes, lineTaxTotal, err := s.snapshotItemTaxes(ctx, tx, reqItem.ItemID, taxableAmount)
		if err != nil {
			return nil, 0, 0, 0, err
		}

		// Validate: Type 11 items should have tributo C3 (0% export IVA)
		hasExportTributo := false
		for _, tax := range taxes {
			if tax.TributoCode == codigos.TributoIVAExportaciones {
				hasExportTributo = true
				break
			}
		}

		if !hasExportTributo {
			return nil, 0, 0, 0, fmt.Errorf(
				"export invoice item '%s' must have tributo C3 (IVA Exportaciones 0%%)",
				item.Name,
			)
		}

		lineTotal := round(taxableAmount + lineTaxTotal)

		// 4. Create line item
		lineItem := models.InvoiceLineItem{
			ItemID:             &reqItem.ItemID,
			ItemSku:            item.SKU,
			ItemName:           item.Name,
			ItemDescription:    item.Description,
			ItemTipoItem:       item.TipoItem,
			UnitOfMeasure:      item.UnitOfMeasure,
			UnitPrice:          item.UnitPrice,
			Quantity:           reqItem.Quantity,
			LineSubtotal:       lineSubtotal,
			DiscountPercentage: reqItem.DiscountPercentage,
			DiscountAmount:     discountAmount,
			TaxableAmount:      taxableAmount,
			TotalTaxes:         lineTaxTotal, // Will be 0 for export
			LineTotal:          lineTotal,
			Taxes:              taxes,
			CreatedAt:          time.Now(),
		}

		lineItems = append(lineItems, lineItem)

		// 5. Accumulate totals
		subtotal += lineSubtotal
		totalDiscount += discountAmount
		totalTaxes += lineTaxTotal // Should be 0 for Type 11
	}

	// Round final totals
	return lineItems, round(subtotal), round(totalDiscount), round(totalTaxes), nil
}

// ============================================
// SEPARATE EXPORT INVOICE METHODS (No conflicts)
// ============================================

// insertInvoiceExport inserts an export invoice with export fields
func (s *InvoiceService) insertInvoiceExport(ctx context.Context, tx *sql.Tx, invoice *models.Invoice) (string, error) {
	id := strings.ToUpper(uuid.New().String())
	query := `
		INSERT INTO invoices (
            id,
			company_id, establishment_id, point_of_sale_id, client_id, invoice_number, invoice_type,
			client_name, client_legal_name, client_nit, client_ncr, client_dui,
			client_address, client_tipo_contribuyente, client_tipo_persona,
			subtotal, total_discount, total_taxes, total,
			currency, payment_terms, payment_method, payment_status, amount_paid, balance_due, due_date, dte_codigo_generacion,
			status, notes, contact_email, contact_whatsapp, created_at,
			export_tipo_item_expor, export_recinto_fiscal, export_regimen,
			export_incoterms_code, export_incoterms_desc, export_seguro, export_flete, export_observaciones,
			export_receptor_cod_pais, export_receptor_nombre_pais, export_receptor_tipo_documento,
			export_receptor_num_documento, export_receptor_complemento
		) VALUES (
			$1, $2, $3, $4, $5, $6,
			$7, $8, $9, $10, $11,
			$12, $13, $14,
			$15, $16, $17, $18,
			$19, $20, $21, $22, $23, $24,
			$25, $26, $27, $28, $29, $30, $31, $32,
			$33, $34, $35,
			$36, $37, $38, $39, $40,
			$41, $42, $43,
			$44, $45
		) RETURNING id
	`

	_, err := tx.ExecContext(ctx, query,
		id,
		invoice.CompanyID, invoice.EstablishmentID, invoice.PointOfSaleID, invoice.ClientID, invoice.InvoiceNumber, invoice.InvoiceType,
		invoice.ClientName, invoice.ClientLegalName, invoice.ClientNit, invoice.ClientNcr, invoice.ClientDui,
		invoice.ClientAddress, invoice.ClientTipoContribuyente, invoice.ClientTipoPersona,
		invoice.Subtotal, invoice.TotalDiscount, invoice.TotalTaxes, invoice.Total,
		invoice.Currency, invoice.PaymentTerms, invoice.PaymentMethod, invoice.PaymentStatus, invoice.AmountPaid, invoice.BalanceDue, invoice.DueDate, id,
		invoice.Status, invoice.Notes, invoice.ContactEmail, invoice.ContactWhatsapp, invoice.CreatedAt,
		invoice.ExportTipoItemExpor, invoice.ExportRecintoFiscal, invoice.ExportRegimen,
		invoice.ExportIncotermsCode, invoice.ExportIncotermsDesc, invoice.ExportSeguro, invoice.ExportFlete, invoice.ExportObservaciones,
		invoice.ExportReceptorCodPais, invoice.ExportReceptorNombrePais, invoice.ExportReceptorTipoDocumento,
		invoice.ExportReceptorNumDocumento, invoice.ExportReceptorComplemento,
	)

	if err != nil {
		return "", err
	}

	return id, nil
}

// getInvoiceHeaderExport retrieves an export invoice header with export fields
func (s *InvoiceService) getInvoiceHeaderExport(ctx context.Context, companyID, invoiceID string) (*models.Invoice, error) {
	query := `
		SELECT
			id, company_id, establishment_id, point_of_sale_id, client_id,
			invoice_number, invoice_type,
			references_invoice_id, void_reason,
			client_name, client_legal_name, client_nit, client_ncr, client_dui,
			client_address, client_tipo_contribuyente, client_tipo_persona,
			subtotal, total_discount, total_taxes, total,
			currency,
			payment_terms, payment_method, payment_status, amount_paid, balance_due, due_date,
			status,
			dte_numero_control, dte_status, dte_hacienda_response, dte_submitted_at, dte_type,
			created_at, finalized_at, voided_at,
			created_by, voided_by, notes,
			contact_email, contact_whatsapp,
			export_tipo_item_expor, export_recinto_fiscal, export_regimen,
			export_incoterms_code, export_incoterms_desc, export_seguro, export_flete, export_observaciones,
			export_receptor_cod_pais, export_receptor_nombre_pais, export_receptor_tipo_documento,
			export_receptor_num_documento, export_receptor_complemento
		FROM invoices
		WHERE id = $1 AND company_id = $2
	`

	invoice := &models.Invoice{}
	err := database.DB.QueryRowContext(ctx, query, invoiceID, companyID).Scan(
		&invoice.ID, &invoice.CompanyID, &invoice.EstablishmentID, &invoice.PointOfSaleID, &invoice.ClientID,
		&invoice.InvoiceNumber, &invoice.InvoiceType,
		&invoice.ReferencesInvoiceID, &invoice.VoidReason,
		&invoice.ClientName, &invoice.ClientLegalName, &invoice.ClientNit, &invoice.ClientNcr, &invoice.ClientDui,
		&invoice.ClientAddress, &invoice.ClientTipoContribuyente, &invoice.ClientTipoPersona,
		&invoice.Subtotal, &invoice.TotalDiscount, &invoice.TotalTaxes, &invoice.Total,
		&invoice.Currency,
		&invoice.PaymentTerms, &invoice.PaymentMethod, &invoice.PaymentStatus, &invoice.AmountPaid, &invoice.BalanceDue, &invoice.DueDate,
		&invoice.Status,
		&invoice.DteNumeroControl, &invoice.DteStatus, &invoice.DteHaciendaResponse, &invoice.DteSubmittedAt, &invoice.DteType,
		&invoice.CreatedAt, &invoice.FinalizedAt, &invoice.VoidedAt,
		&invoice.CreatedBy, &invoice.VoidedBy, &invoice.Notes,
		&invoice.ContactEmail, &invoice.ContactWhatsapp,
		&invoice.ExportTipoItemExpor, &invoice.ExportRecintoFiscal, &invoice.ExportRegimen,
		&invoice.ExportIncotermsCode, &invoice.ExportIncotermsDesc, &invoice.ExportSeguro, &invoice.ExportFlete, &invoice.ExportObservaciones,
		&invoice.ExportReceptorCodPais, &invoice.ExportReceptorNombrePais, &invoice.ExportReceptorTipoDocumento,
		&invoice.ExportReceptorNumDocumento, &invoice.ExportReceptorComplemento,
	)
	if err == sql.ErrNoRows {
		return nil, ErrInvoiceNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query invoice: %w", err)
	}

	return invoice, nil
}

func (s *InvoiceService) ValidateExportInvoice(ctx context.Context, invoice *models.Invoice) error {
	var errors []string

	// 1. Basic invoice validation
	if invoice.ID == "" {
		errors = append(errors, "invoice ID is required")
	}
	if invoice.CompanyID == "" {
		errors = append(errors, "company ID is required")
	}
	if invoice.ClientID == "" {
		errors = append(errors, "client ID is required")
	}

	// 2. Export-specific required fields
	if invoice.ExportTipoItemExpor == nil {
		errors = append(errors, "export_tipo_item_expor is required for export invoices")
	} else if *invoice.ExportTipoItemExpor < 1 || *invoice.ExportTipoItemExpor > 3 {
		errors = append(errors, "export_tipo_item_expor must be 1, 2, or 3")
	}

	// Recinto fiscal required for tipo 1 and 3
	if invoice.ExportTipoItemExpor != nil && *invoice.ExportTipoItemExpor != 2 {
		if invoice.ExportRecintoFiscal == nil {
			errors = append(errors, "export_recinto_fiscal is required for tipo_item_expor 1 and 3")
		} else {
			// ✅ Validate recinto fiscal code
			if !codigos.IsValidTaxEnclosure(*invoice.ExportRecintoFiscal) {
				errors = append(errors, fmt.Sprintf("invalid export_recinto_fiscal code: %s", *invoice.ExportRecintoFiscal))
			}
		}

		if invoice.ExportRegimen == nil {
			errors = append(errors, "export_regimen is required for tipo_item_expor 1 and 3")
		} else {
			// ✅ Validate regimen code using the catalog
			if !codigos.IsValidRegimenType(*invoice.ExportRegimen) {
				errors = append(errors, fmt.Sprintf("invalid export_regimen code: %s", *invoice.ExportRegimen))
			}
		}
	}

	// 3. INCOTERMS validation
	if invoice.ExportIncotermsCode == nil || *invoice.ExportIncotermsCode == "" {
		errors = append(errors, "export_incoterms_code is required")
	} else {
		// ✅ Validate INCOTERMS code
		if !codigos.IsValidIncoterm(*invoice.ExportIncotermsCode) {
			errors = append(errors, fmt.Sprintf("invalid export_incoterms_code: %s", *invoice.ExportIncotermsCode))
		}
	}

	if invoice.ExportIncotermsDesc == nil || *invoice.ExportIncotermsDesc == "" {
		errors = append(errors, "export_incoterms_desc is required")
	}

	// 4. Receptor (international client) validation
	if invoice.ExportReceptorCodPais == nil || *invoice.ExportReceptorCodPais == "" {
		errors = append(errors, "export_receptor_cod_pais is required")
	}
	if invoice.ExportReceptorNombrePais == nil || *invoice.ExportReceptorNombrePais == "" {
		errors = append(errors, "export_receptor_nombre_pais is required")
	}
	if invoice.ExportReceptorTipoDocumento == nil || *invoice.ExportReceptorTipoDocumento == "" {
		errors = append(errors, "export_receptor_tipo_documento is required")
	}
	if invoice.ExportReceptorNumDocumento == nil || *invoice.ExportReceptorNumDocumento == "" {
		errors = append(errors, "export_receptor_num_documento is required")
	}
	if invoice.ExportReceptorComplemento == nil || *invoice.ExportReceptorComplemento == "" {
		errors = append(errors, "export_receptor_complemento (address) is required")
	}

	// validate country
	if invoice.ExportReceptorNombrePais != nil {
		cc, err := codigos.CountryCodeFromName(*invoice.ExportReceptorNombrePais)
		if err != nil {
			errors = append(errors, "country code is invalid")
		} else {
			invoice.ExportReceptorCodPais = &cc
		}
	}

	// 5. Validate numDocumento format based on tipoDocumento
	if invoice.ExportReceptorTipoDocumento != nil && invoice.ExportReceptorNumDocumento != nil {
		tipoDoc := *invoice.ExportReceptorTipoDocumento
		numDoc := *invoice.ExportReceptorNumDocumento

		switch tipoDoc {
		case "36": // NIT - must be 9 or 14 digits
			if !isDigitsOnly(numDoc) {
				errors = append(errors, "export_receptor_num_documento for tipo 36 (NIT) must contain only digits")
			} else if len(numDoc) != 9 && len(numDoc) != 14 {
				errors = append(errors, "export_receptor_num_documento for tipo 36 (NIT) must be 9 or 14 digits")
			}
		case "13": // DUI - must be 8 digits + dash + 1 digit
			if !strings.Contains(numDoc, "-") {
				errors = append(errors, "export_receptor_num_documento for tipo 13 (DUI) must have format: 12345678-9")
			}
		}
	}

	// 6. Email required for invoices > $10,000
	if invoice.Total >= 10000 {
		if invoice.ContactEmail == nil || *invoice.ContactEmail == "" {
			errors = append(errors, "contact_email is required for export invoices with total >= $10,000")
		}
	}

	// 7. Validate line items
	if len(invoice.LineItems) == 0 {
		errors = append(errors, "at least one line item is required")
	}

	// 8. Validate export documents exist
	if len(invoice.ExportDocuments) == 0 {
		errors = append(errors, "at least one export document is required")
	} else {
		// ✅ Validate each export document
		for i, doc := range invoice.ExportDocuments {
			// Validate cod_doc_asociado (convert int to string for validation)
			if doc.CodDocAsociado == 0 {
				errors = append(errors, fmt.Sprintf("export_document[%d]: cod_doc_asociado is required", i))
			} else {
				codDocStr := fmt.Sprintf("%d", doc.CodDocAsociado)
				if !codigos.IsValidAssociatedDocument(codDocStr) {
					errors = append(errors, fmt.Sprintf("export_document[%d]: invalid cod_doc_asociado: %d", i, doc.CodDocAsociado))
				}
			}

			// For transport documents (code 4), additional fields are required
			if doc.CodDocAsociado == 4 {
				if doc.PlacaTrans == nil || *doc.PlacaTrans == "" {
					errors = append(errors, fmt.Sprintf("export_document[%d]: placa_trans is required for transport documents", i))
				}
				if doc.ModoTransp == nil || *doc.ModoTransp == 0 {
					errors = append(errors, fmt.Sprintf("export_document[%d]: modo_transp is required for transport documents", i))
				}
				if doc.NumConductor == nil || *doc.NumConductor == "" {
					errors = append(errors, fmt.Sprintf("export_document[%d]: num_conductor is required for transport documents", i))
				}
				if doc.NombreConductor == nil || *doc.NombreConductor == "" {
					errors = append(errors, fmt.Sprintf("export_document[%d]: nombre_conductor is required for transport documents", i))
				}
			}
		}
	}

	// 9. Validate totals are positive
	if invoice.Total <= 0 {
		errors = append(errors, "invoice total must be greater than 0")
	}

	// If any validation errors, return them
	if len(errors) > 0 {
		return fmt.Errorf("export invoice validation failed:\n  - %s", strings.Join(errors, "\n  - "))
	}

	return nil
}

func isDigitsOnly(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return len(s) > 0
}
