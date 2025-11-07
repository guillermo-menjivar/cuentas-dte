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
// UPDATE EXISTING METHODS
// ============================================

// Update insertInvoice to include export fields
func (s *InvoiceService) insertInvoice(ctx context.Context, tx *sql.Tx, invoice *models.Invoice) (string, error) {
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

// Update getInvoiceHeader to load export fields
func (s *InvoiceService) getInvoiceHeader(ctx context.Context, companyID, invoiceID string) (*models.Invoice, error) {
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
