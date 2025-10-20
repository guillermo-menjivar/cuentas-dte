package services

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"

	"cuentas/internal/codigos"
	"cuentas/internal/database"
	"cuentas/internal/models"
	"cuentas/internal/models/dte"
)

type NotaService struct{}

func NewNotaService() *NotaService {
	return &NotaService{}
}

// CreateNota creates a new nota (débito or crédito)
func (s *NotaService) CreateNota(ctx context.Context, companyID, notaType string, req *models.CreateNotaRequest) (*models.Nota, error) {
	// Validate nota type
	if notaType != codigos.DocTypeNotaDebito && notaType != codigos.DocTypeNotaCredito {
		return nil, fmt.Errorf("invalid nota type: %s", notaType)
	}

	// Begin transaction
	tx, err := database.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Validate POS
	if err := s.validatePointOfSale(ctx, tx, companyID, req.EstablishmentID, req.PointOfSaleID); err != nil {
		return nil, err
	}

	// Validate client exists
	if err := s.validateClient(ctx, tx, companyID, req.ClientID); err != nil {
		return nil, err
	}

	// Generate nota number
	notaNumber, err := s.generateNotaNumber(ctx, tx, companyID, notaType)
	if err != nil {
		return nil, fmt.Errorf("failed to generate nota number: %w", err)
	}

	// Process line items and calculate totals
	lineItems, subtotal, taxAmount, err := s.processLineItems(req.LineItems)
	if err != nil {
		return nil, err
	}

	total := round(subtotal + taxAmount)

	// Create nota record
	nota := &models.Nota{
		CompanyID:          companyID,
		Type:               notaType,
		ClientID:           req.ClientID,
		EstablishmentID:    req.EstablishmentID,
		PointOfSaleID:      req.PointOfSaleID,
		NotaNumber:         notaNumber,
		Status:             "draft",
		PaymentStatus:      "unpaid",
		ParentDocumentType: req.ParentDocumentType,
		Subtotal:           subtotal,
		TaxAmount:          taxAmount,
		Total:              total,
		BalanceDue:         total,
		Currency:           "USD",
		PaymentTerms:       &req.PaymentTerms,
		Notes:              &req.Notes,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}

	// Insert nota
	notaID, err := s.insertNota(ctx, tx, nota)
	if err != nil {
		return nil, fmt.Errorf("failed to insert nota: %w", err)
	}
	nota.ID = notaID

	// Insert related documents
	relatedDocs, err := s.insertRelatedDocuments(ctx, tx, notaID, companyID, req.RelatedDocuments)
	if err != nil {
		return nil, fmt.Errorf("failed to insert related documents: %w", err)
	}
	nota.RelatedDocuments = relatedDocs

	// Insert line items
	for i := range lineItems {
		lineItems[i].NotaID = notaID
		lineItems[i].LineNumber = i + 1

		lineItemID, err := s.insertLineItem(ctx, tx, &lineItems[i])
		if err != nil {
			return nil, fmt.Errorf("failed to insert line item %d: %w", i+1, err)
		}
		lineItems[i].ID = lineItemID
	}
	nota.LineItems = lineItems

	// Commit
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nota, nil
}

// GetNota retrieves a nota with all related data
func (s *NotaService) GetNota(ctx context.Context, companyID, notaID string) (*models.Nota, error) {
	nota, err := s.getNotaHeader(ctx, companyID, notaID)
	if err != nil {
		return nil, err
	}

	// Get related documents
	relatedDocs, err := s.getRelatedDocuments(ctx, notaID)
	if err != nil {
		return nil, fmt.Errorf("failed to get related documents: %w", err)
	}
	nota.RelatedDocuments = relatedDocs

	// Get line items
	lineItems, err := s.getLineItems(ctx, notaID)
	if err != nil {
		return nil, fmt.Errorf("failed to get line items: %w", err)
	}
	nota.LineItems = lineItems

	return nota, nil
}

// ListNotas lists notas with optional filters
func (s *NotaService) ListNotas(ctx context.Context, companyID string, filters map[string]interface{}) ([]models.Nota, error) {
	query := `
		SELECT
			id, company_id, type, client_id, establishment_id, point_of_sale_id,
			nota_number, status, payment_status,
			subtotal, tax_amount, total, balance_due,
			dte_status, dte_numero_control,
			created_at, finalized_at
		FROM notas
		WHERE company_id = $1
	`

	args := []interface{}{companyID}
	argCount := 1

	if notaType, ok := filters["type"].(string); ok && notaType != "" {
		argCount++
		query += fmt.Sprintf(" AND type = $%d", argCount)
		args = append(args, notaType)
	}

	if status, ok := filters["status"].(string); ok && status != "" {
		argCount++
		query += fmt.Sprintf(" AND status = $%d", argCount)
		args = append(args, status)
	}

	query += " ORDER BY created_at DESC"

	rows, err := database.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notas []models.Nota
	for rows.Next() {
		var n models.Nota
		err := rows.Scan(
			&n.ID, &n.CompanyID, &n.Type, &n.ClientID, &n.EstablishmentID, &n.PointOfSaleID,
			&n.NotaNumber, &n.Status, &n.PaymentStatus,
			&n.Subtotal, &n.TaxAmount, &n.Total, &n.BalanceDue,
			&n.DteStatus, &n.DteNumeroControl,
			&n.CreatedAt, &n.FinalizedAt,
		)
		if err != nil {
			return nil, err
		}
		notas = append(notas, n)
	}

	return notas, rows.Err()
}

// DeleteDraftNota deletes a draft nota
func (s *NotaService) DeleteDraftNota(ctx context.Context, companyID, notaID string) error {
	nota, err := s.getNotaHeader(ctx, companyID, notaID)
	if err != nil {
		return err
	}

	if nota.Status != "draft" {
		return ErrNotaNotDraft
	}

	query := `DELETE FROM notas WHERE id = $1 AND company_id = $2`
	result, err := database.DB.ExecContext(ctx, query, notaID, companyID)
	if err != nil {
		return fmt.Errorf("failed to delete nota: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrNotaNotFound
	}

	return nil
}

// FinalizeNota finalizes a draft nota
func (s *NotaService) FinalizeNota(ctx context.Context, companyID, notaID, userID string, payment *models.PaymentInfo) (*models.Nota, error) {
	tx, err := database.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Get nota with lock
	nota, err := s.getNotaForUpdate(ctx, tx, companyID, notaID)
	if err != nil {
		return nil, err
	}

	if nota.Status != "draft" {
		return nil, ErrNotaNotDraft
	}

	// Generate DTE numero control
	numeroControl, err := s.generateNumeroControl(ctx, tx, nota.EstablishmentID, nota.PointOfSaleID, nota.Type)
	if err != nil {
		return nil, fmt.Errorf("failed to generate numero control: %w", err)
	}

	// Calculate payment status
	paymentStatus := "unpaid"
	balanceDue := nota.Total
	if payment != nil && payment.Amount > 0 {
		if payment.Amount >= nota.Total {
			paymentStatus = "paid"
			balanceDue = 0
		} else {
			paymentStatus = "partial"
			balanceDue = nota.Total - payment.Amount
		}
	}

	// Update nota
	now := time.Now()
	updateQuery := `
		UPDATE notas
		SET status = 'finalized',
		    payment_status = $1,
		    balance_due = $2,
		    dte_numero_control = $3,
		    dte_status = 'not_submitted',
		    finalized_at = $4,
		    finalized_by = $5,
		    updated_at = $6
		WHERE id = $7 AND company_id = $8
	`

	_, err = tx.ExecContext(ctx, updateQuery,
		paymentStatus, balanceDue, numeroControl, now, userID, now, notaID, companyID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update nota: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return s.GetNota(ctx, companyID, notaID)
}

// ==================== HELPER METHODS ====================

func (s *NotaService) validatePointOfSale(ctx context.Context, tx *sql.Tx, companyID, establishmentID, posID string) error {
	query := `
		SELECT pos.id
		FROM point_of_sale pos
		JOIN establishments e ON pos.establishment_id = e.id
		WHERE pos.id = $1 AND e.id = $2 AND e.company_id = $3 
			AND pos.active = true AND e.active = true
	`
	var id string
	err := tx.QueryRowContext(ctx, query, posID, establishmentID, companyID).Scan(&id)
	if err == sql.ErrNoRows {
		return ErrPointOfSaleNotFound
	}
	return err
}

func (s *NotaService) validateClient(ctx context.Context, tx *sql.Tx, companyID, clientID string) error {
	query := `SELECT id FROM clients WHERE id = $1 AND company_id = $2 AND active = true`
	var id string
	err := tx.QueryRowContext(ctx, query, clientID, companyID).Scan(&id)
	if err == sql.ErrNoRows {
		return ErrClientNotFound
	}
	return err
}

func (s *NotaService) generateNotaNumber(ctx context.Context, tx *sql.Tx, companyID, notaType string) (string, error) {
	prefix := "ND" // Nota de Débito
	if notaType == codigos.DocTypeNotaCredito {
		prefix = "NC" // Nota de Crédito
	}

	var lastNumber sql.NullString
	query := `
		SELECT nota_number FROM notas
		WHERE company_id = $1 AND type = $2
		ORDER BY created_at DESC LIMIT 1
	`
	err := tx.QueryRowContext(ctx, query, companyID, notaType).Scan(&lastNumber)
	if err != nil && err != sql.ErrNoRows {
		return "", err
	}

	var sequence int64 = 1
	if lastNumber.Valid {
		var year int
		fmt.Sscanf(lastNumber.String, prefix+"-%d-%d", &year, &sequence)
		sequence++
	}

	currentYear := time.Now().Year()
	return fmt.Sprintf("%s-%d-%05d", prefix, currentYear, sequence), nil
}

func (s *NotaService) processLineItems(reqItems []models.CreateNotaLineItemRequest) ([]models.NotaLineItem, float64, float64, error) {
	var lineItems []models.NotaLineItem
	var subtotal, totalTax float64

	for _, req := range reqItems {
		lineSubtotal := round(req.UnitPrice * req.Quantity)
		taxableAmount := round(lineSubtotal - req.DiscountAmount)
		lineTax := round(taxableAmount * 0.13) // 13% IVA
		lineTotal := round(taxableAmount + lineTax)

		item := models.NotaLineItem{
			ItemType:           req.ItemType,
			ItemSku:            req.ItemSku,
			ItemName:           req.ItemName,
			Quantity:           req.Quantity,
			UnitOfMeasure:      req.UnitOfMeasure,
			UnitPrice:          req.UnitPrice,
			DiscountAmount:     req.DiscountAmount,
			TaxableAmount:      taxableAmount,
			TaxAmount:          lineTax,
			TotalAmount:        lineTotal,
			RelatedDocumentRef: &req.RelatedDocumentRef,
			CreatedAt:          time.Now(),
		}

		lineItems = append(lineItems, item)
		subtotal += taxableAmount
		totalTax += lineTax
	}

	return lineItems, round(subtotal), round(totalTax), nil
}

func (s *NotaService) insertNota(ctx context.Context, tx *sql.Tx, nota *models.Nota) (string, error) {
	id := strings.ToUpper(uuid.New().String())
	query := `
		INSERT INTO notas (
			id, company_id, type, client_id, establishment_id, point_of_sale_id,
			nota_number, status, payment_status, parent_document_type,
			subtotal, tax_amount, total, balance_due, currency,
			payment_terms, notes, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)
	`
	_, err := tx.ExecContext(ctx, query,
		id, nota.CompanyID, nota.Type, nota.ClientID, nota.EstablishmentID, nota.PointOfSaleID,
		nota.NotaNumber, nota.Status, nota.PaymentStatus, nota.ParentDocumentType,
		nota.Subtotal, nota.TaxAmount, nota.Total, nota.BalanceDue, nota.Currency,
		nota.PaymentTerms, nota.Notes, nota.CreatedAt, nota.UpdatedAt,
	)
	return id, err
}

func (s *NotaService) insertRelatedDocuments(ctx context.Context, tx *sql.Tx, notaID, companyID string, docs []models.CreateNotaRelatedDocumentRequest) ([]models.NotaRelatedDocument, error) {
	var result []models.NotaRelatedDocument

	for _, doc := range docs {
		docDate, _ := time.Parse("2006-01-02", doc.DocumentDate)
		id := uuid.New().String()

		query := `
			INSERT INTO nota_related_documents (
				id, nota_id, company_id, document_type, generation_type,
				document_number, document_date, created_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`
		_, err := tx.ExecContext(ctx, query,
			id, notaID, companyID, doc.DocumentType, doc.GenerationType,
			doc.DocumentNumber, docDate, time.Now(),
		)
		if err != nil {
			return nil, err
		}

		result = append(result, models.NotaRelatedDocument{
			ID:             id,
			NotaID:         notaID,
			CompanyID:      companyID,
			DocumentType:   doc.DocumentType,
			GenerationType: doc.GenerationType,
			DocumentNumber: doc.DocumentNumber,
			DocumentDate:   docDate,
			CreatedAt:      time.Now(),
		})
	}

	return result, nil
}

func (s *NotaService) insertLineItem(ctx context.Context, tx *sql.Tx, item *models.NotaLineItem) (string, error) {
	id := uuid.New().String()
	query := `
		INSERT INTO nota_line_items (
			id, nota_id, line_number, item_type, item_sku, item_name,
			quantity, unit_of_measure, unit_price, discount_amount,
			taxable_amount, tax_amount, total_amount, related_document_ref, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
	`
	_, err := tx.ExecContext(ctx, query,
		id, item.NotaID, item.LineNumber, item.ItemType, item.ItemSku, item.ItemName,
		item.Quantity, item.UnitOfMeasure, item.UnitPrice, item.DiscountAmount,
		item.TaxableAmount, item.TaxAmount, item.TotalAmount, item.RelatedDocumentRef, item.CreatedAt,
	)
	return id, err
}

func (s *NotaService) getNotaHeader(ctx context.Context, companyID, notaID string) (*models.Nota, error) {
	query := `
		SELECT id, company_id, type, client_id, establishment_id, point_of_sale_id,
			nota_number, status, payment_status, parent_document_type,
			subtotal, tax_amount, total, balance_due, currency,
			payment_method, payment_terms, dte_numero_control, dte_status, dte_sello_recibido,
			notes, created_at, updated_at, finalized_at, finalized_by
		FROM notas WHERE id = $1 AND company_id = $2
	`

	n := &models.Nota{}
	err := database.DB.QueryRowContext(ctx, query, notaID, companyID).Scan(
		&n.ID, &n.CompanyID, &n.Type, &n.ClientID, &n.EstablishmentID, &n.PointOfSaleID,
		&n.NotaNumber, &n.Status, &n.PaymentStatus, &n.ParentDocumentType,
		&n.Subtotal, &n.TaxAmount, &n.Total, &n.BalanceDue, &n.Currency,
		&n.PaymentMethod, &n.PaymentTerms, &n.DteNumeroControl, &n.DteStatus, &n.DteSelloRecibido,
		&n.Notes, &n.CreatedAt, &n.UpdatedAt, &n.FinalizedAt, &n.FinalizedBy,
	)
	if err == sql.ErrNoRows {
		return nil, ErrNotaNotFound
	}
	return n, err
}

func (s *NotaService) getRelatedDocuments(ctx context.Context, notaID string) ([]models.NotaRelatedDocument, error) {
	query := `
		SELECT id, nota_id, company_id, document_type, generation_type,
			document_number, document_date, created_at
		FROM nota_related_documents WHERE nota_id = $1 ORDER BY created_at
	`

	rows, err := database.DB.QueryContext(ctx, query, notaID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var docs []models.NotaRelatedDocument
	for rows.Next() {
		var doc models.NotaRelatedDocument
		err := rows.Scan(
			&doc.ID, &doc.NotaID, &doc.CompanyID, &doc.DocumentType, &doc.GenerationType,
			&doc.DocumentNumber, &doc.DocumentDate, &doc.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		docs = append(docs, doc)
	}
	return docs, rows.Err()
}

func (s *NotaService) getLineItems(ctx context.Context, notaID string) ([]models.NotaLineItem, error) {
	query := `
		SELECT id, nota_id, line_number, item_type, item_sku, item_name,
			quantity, unit_of_measure, unit_price, discount_amount,
			taxable_amount, tax_amount, total_amount, related_document_ref, created_at
		FROM nota_line_items WHERE nota_id = $1 ORDER BY line_number
	`

	rows, err := database.DB.QueryContext(ctx, query, notaID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []models.NotaLineItem
	for rows.Next() {
		var item models.NotaLineItem
		err := rows.Scan(
			&item.ID, &item.NotaID, &item.LineNumber, &item.ItemType, &item.ItemSku, &item.ItemName,
			&item.Quantity, &item.UnitOfMeasure, &item.UnitPrice, &item.DiscountAmount,
			&item.TaxableAmount, &item.TaxAmount, &item.TotalAmount, &item.RelatedDocumentRef, &item.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *NotaService) getNotaForUpdate(ctx context.Context, tx *sql.Tx, companyID, notaID string) (*models.Nota, error) {
	query := `
		SELECT id, company_id, type, status, total, establishment_id, point_of_sale_id
		FROM notas WHERE id = $1 AND company_id = $2 FOR UPDATE
	`

	n := &models.Nota{}
	err := tx.QueryRowContext(ctx, query, notaID, companyID).Scan(
		&n.ID, &n.CompanyID, &n.Type, &n.Status, &n.Total, &n.EstablishmentID, &n.PointOfSaleID,
	)
	if err == sql.ErrNoRows {
		return nil, ErrNotaNotFound
	}
	return n, err
}

func (s *NotaService) generateNumeroControl(ctx context.Context, tx *sql.Tx, establishmentID, posID, tipoDte string) (string, error) {
	query := `
		SELECT e.cod_establecimiento, pos.cod_punto_venta
		FROM point_of_sale pos
		JOIN establishments e ON pos.establishment_id = e.id
		WHERE pos.id = $1
	`

	var estCode, posCode *string
	err := tx.QueryRowContext(ctx, query, posID).Scan(&estCode, &posCode)
	if err != nil {
		return "", err
	}

	if estCode == nil || posCode == nil {
		return "", fmt.Errorf("establishment or POS code not configured")
	}

	sequence, err := s.getAndIncrementDTESequence(ctx, tx, posID, tipoDte)
	if err != nil {
		return "", err
	}

	return dte.BuildNumeroControl(tipoDte, *estCode, *posCode, sequence)
}

func (s *NotaService) getAndIncrementDTESequence(ctx context.Context, tx *sql.Tx, posID, tipoDte string) (int64, error) {
	var currentSeq int64
	query := `SELECT last_sequence FROM dte_sequences WHERE point_of_sale_id = $1 AND tipo_dte = $2 FOR UPDATE`

	err := tx.QueryRowContext(ctx, query, posID, tipoDte).Scan(&currentSeq)
	if err == sql.ErrNoRows {
		_, err = tx.ExecContext(ctx,
			`INSERT INTO dte_sequences (point_of_sale_id, tipo_dte, last_sequence, updated_at) VALUES ($1, $2, 1, $3)`,
			posID, tipoDte, time.Now())
		return 1, err
	}
	if err != nil {
		return 0, err
	}

	newSeq := currentSeq + 1
	_, err = tx.ExecContext(ctx,
		`UPDATE dte_sequences SET last_sequence = $1, updated_at = $2 WHERE point_of_sale_id = $3 AND tipo_dte = $4`,
		newSeq, time.Now(), posID, tipoDte)

	return newSeq, err
}

func round(val float64) float64 {
	return math.Round(val*100) / 100
}
