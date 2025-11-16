package services

import (
	"context"
	"cuentas/internal/database"
	"cuentas/internal/models"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ============================================
// ERRORS
// ============================================

var (
	ErrPurchaseNotFound      = errors.New("purchase not found")
	ErrPurchaseNotDraft      = errors.New("purchase is not in draft status")
	ErrPurchaseAlreadyVoid   = errors.New("purchase is already void")
	ErrInvalidPurchaseStatus = errors.New("invalid purchase status for this operation")
	ErrSupplierNotFound      = errors.New("supplier not found")
)

// ============================================
// SERVICE DEFINITION
// ============================================

type PurchaseService struct {
	// Add dependencies as needed
}

func NewPurchaseService() *PurchaseService {
	return &PurchaseService{}
}

// ============================================
// CREATE FSE
// ============================================

// CreateFSE creates a new FSE (Factura Sujeto Excluido) purchase in draft status
func (s *PurchaseService) CreateFSE(ctx context.Context, companyID string, req *models.CreateFSERequest) (*models.Purchase, error) {
	// 1. Validate request
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// 2. Begin transaction
	tx, err := database.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// 3. Validate establishment and POS
	if err := s.validatePointOfSale(ctx, tx, companyID, req.EstablishmentID, req.PointOfSaleID); err != nil {
		return nil, err
	}

	// 4. Generate purchase number
	purchaseNumber, err := s.generatePurchaseNumber(ctx, tx, companyID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate purchase number: %w", err)
	}

	// 5. Process line items
	lineItems, subtotal, totalDiscount, err := s.processLineItemsFSE(ctx, tx, req.LineItems)
	if err != nil {
		return nil, fmt.Errorf("failed to process line items: %w", err)
	}

	// 6. Calculate totals
	total := round(subtotal - totalDiscount - req.IVARetained - req.IncomeTaxRetained)
	balanceDue := total

	// Adjust balance if paid immediately
	if req.Payment.Condition == 1 { // Contado
		balanceDue = 0
	}

	// 7. Create purchase record
	purchase := &models.Purchase{
		ID:              strings.ToUpper(uuid.New().String()), // This is also codigoGeneracion
		CompanyID:       companyID,
		EstablishmentID: req.EstablishmentID,
		PointOfSaleID:   req.PointOfSaleID,

		PurchaseNumber: purchaseNumber,
		PurchaseType:   "fse",
		PurchaseDate:   req.PurchaseDate.Time,

		// FSE: Embedded supplier info (no supplier_id)
		SupplierID:                nil,
		SupplierName:              &req.Supplier.Name,
		SupplierDocumentType:      &req.Supplier.DocumentType,
		SupplierDocumentNumber:    &req.Supplier.DocumentNumber,
		SupplierNRC:               req.Supplier.NRC,
		SupplierActivityCode:      &req.Supplier.ActivityCode,
		SupplierActivityDesc:      &req.Supplier.ActivityDesc,
		SupplierAddressDept:       &req.Supplier.Address.Department,
		SupplierAddressMuni:       &req.Supplier.Address.Municipality,
		SupplierAddressComplement: &req.Supplier.Address.Complement,
		SupplierPhone:             req.Supplier.Phone,
		SupplierEmail:             req.Supplier.Email,

		Subtotal:           subtotal,
		TotalDiscount:      totalDiscount,
		DiscountPercentage: req.DiscountPercentage,
		TotalTaxes:         0, // FSE has no taxes
		IVARetained:        req.IVARetained,
		IncomeTaxRetained:  req.IncomeTaxRetained,
		Total:              total,
		Currency:           "USD",

		PaymentCondition: &req.Payment.Condition,
		PaymentMethod:    &req.Payment.Method,
		PaymentReference: req.Payment.Reference,
		PaymentTerm:      req.Payment.Term,
		PaymentPeriod:    req.Payment.Period,
		PaymentStatus:    "pending",
		AmountPaid:       0,
		BalanceDue:       balanceDue,
		DueDate:          nil, // Set if credit

		DteType: "14",
		Status:  "draft",

		CreatedAt: time.Now(),
		Notes:     req.Notes,

		LineItems: lineItems,
	}

	// Set due date for credit purchases
	if req.Payment.Condition == 2 && req.Payment.Period != nil {
		dueDate := req.PurchaseDate.AddDate(0, 0, *req.Payment.Period)
		purchase.DueDate = &dueDate
	}

	// 8. Insert purchase
	purchaseID, err := s.insertPurchase(ctx, tx, purchase)
	if err != nil {
		return nil, fmt.Errorf("failed to insert purchase: %w", err)
	}

	// 9. Insert line items
	for i := range lineItems {
		lineItems[i].PurchaseID = purchaseID
		lineItems[i].LineNumber = i + 1

		if err := s.insertPurchaseLineItem(ctx, tx, &lineItems[i]); err != nil {
			return nil, fmt.Errorf("failed to insert line item %d: %w", i+1, err)
		}
	}

	// 10. Commit transaction
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// 11. Reload purchase with all relations
	purchase, err = s.GetPurchaseByID(ctx, companyID, purchaseID)
	if err != nil {
		return nil, fmt.Errorf("failed to reload purchase: %w", err)
	}

	return purchase, nil
}

// ============================================
// PROCESS LINE ITEMS
// ============================================

// processLineItemsFSE processes line items for FSE purchases
// FSE items are free-form (not from inventory) with no taxes
func (s *PurchaseService) processLineItemsFSE(
	ctx context.Context,
	tx *sql.Tx,
	reqItems []models.CreateFSELineItemRequest,
) ([]models.PurchaseLineItem, float64, float64, error) {
	var lineItems []models.PurchaseLineItem
	var subtotal, totalDiscount float64

	for _, reqItem := range reqItems {
		// 1. Calculate line amounts
		lineSubtotal := round(reqItem.UnitPrice * reqItem.Quantity)
		discountAmount := round(reqItem.DiscountAmount)
		taxableAmount := round(lineSubtotal - discountAmount)
		lineTaxTotal := 0.0 // FSE has no taxes
		lineTotal := round(taxableAmount + lineTaxTotal)

		// 2. Convert unit of measure to string
		unitOfMeasure := fmt.Sprintf("%d", reqItem.UnitOfMeasure)

		// 3. Create line item
		lineItem := models.PurchaseLineItem{
			ID:            strings.ToUpper(uuid.New().String()),
			ItemID:        nil, // FSE items are not from inventory
			ItemCode:      reqItem.Code,
			ItemName:      reqItem.Description,
			ItemType:      reqItem.ItemType,
			UnitOfMeasure: unitOfMeasure,

			Quantity:     reqItem.Quantity,
			UnitPrice:    reqItem.UnitPrice,
			LineSubtotal: lineSubtotal,

			DiscountPercentage: 0, // FSE uses fixed discount amounts
			DiscountAmount:     discountAmount,

			TaxableAmount: taxableAmount,
			TotalTaxes:    lineTaxTotal,
			LineTotal:     lineTotal,

			CreatedAt: time.Now(),
			Taxes:     []models.PurchaseLineItemTax{}, // No taxes for FSE
		}

		lineItems = append(lineItems, lineItem)

		// 4. Accumulate totals
		subtotal += lineSubtotal
		totalDiscount += discountAmount
	}

	return lineItems, round(subtotal), round(totalDiscount), nil
}

// ============================================
// DATABASE OPERATIONS
// ============================================

// insertPurchase inserts a purchase record
func (s *PurchaseService) insertPurchase(ctx context.Context, tx *sql.Tx, purchase *models.Purchase) (string, error) {
	query := `
        INSERT INTO purchases (
            id, company_id, establishment_id, point_of_sale_id,
            purchase_number, purchase_type, purchase_date,
            supplier_id,
            supplier_name, supplier_document_type, supplier_document_number,
            supplier_nrc, supplier_activity_code, supplier_activity_desc,
            supplier_address_dept, supplier_address_muni, supplier_address_complement,
            supplier_phone, supplier_email,
            subtotal, total_discount, discount_percentage,
            total_taxes, iva_retained, income_tax_retained, total,
            currency,
            payment_condition, payment_method, payment_reference,
            payment_term, payment_period, payment_status,
            amount_paid, balance_due, due_date,
            dte_type, status,
            created_at, notes
        ) VALUES (
            $1, $2, $3, $4,
            $5, $6, $7,
            $8,
            $9, $10, $11,
            $12, $13, $14,
            $15, $16, $17,
            $18, $19,
            $20, $21, $22,
            $23, $24, $25, $26,
            $27,
            $28, $29, $30,
            $31, $32, $33,
            $34, $35, $36,
            $37, $38,
            $39, $40
        ) RETURNING id
    `

	var id string
	err := tx.QueryRowContext(ctx, query,
		purchase.ID, purchase.CompanyID, purchase.EstablishmentID, purchase.PointOfSaleID,
		purchase.PurchaseNumber, purchase.PurchaseType, purchase.PurchaseDate,
		purchase.SupplierID,
		purchase.SupplierName, purchase.SupplierDocumentType, purchase.SupplierDocumentNumber,
		purchase.SupplierNRC, purchase.SupplierActivityCode, purchase.SupplierActivityDesc,
		purchase.SupplierAddressDept, purchase.SupplierAddressMuni, purchase.SupplierAddressComplement,
		purchase.SupplierPhone, purchase.SupplierEmail,
		purchase.Subtotal, purchase.TotalDiscount, purchase.DiscountPercentage,
		purchase.TotalTaxes, purchase.IVARetained, purchase.IncomeTaxRetained, purchase.Total,
		purchase.Currency,
		purchase.PaymentCondition, purchase.PaymentMethod, purchase.PaymentReference,
		purchase.PaymentTerm, purchase.PaymentPeriod, purchase.PaymentStatus,
		purchase.AmountPaid, purchase.BalanceDue, purchase.DueDate,
		purchase.DteType, purchase.Status,
		purchase.CreatedAt, purchase.Notes,
	).Scan(&id)

	if err != nil {
		return "", fmt.Errorf("failed to insert purchase: %w", err)
	}

	return id, nil
}

// insertPurchaseLineItem inserts a purchase line item
func (s *PurchaseService) insertPurchaseLineItem(ctx context.Context, tx *sql.Tx, item *models.PurchaseLineItem) error {
	query := `
        INSERT INTO purchase_line_items (
            id, purchase_id, line_number,
            item_id, item_code, item_name, item_description,
            item_type, unit_of_measure,
            quantity, unit_price,
            line_subtotal, discount_percentage, discount_amount,
            taxable_amount, total_taxes, line_total,
            created_at
        ) VALUES (
            $1, $2, $3,
            $4, $5, $6, $7,
            $8, $9,
            $10, $11,
            $12, $13, $14,
            $15, $16, $17,
            $18
        )
    `

	_, err := tx.ExecContext(ctx, query,
		item.ID, item.PurchaseID, item.LineNumber,
		item.ItemID, item.ItemCode, item.ItemName, item.ItemDescription,
		item.ItemType, item.UnitOfMeasure,
		item.Quantity, item.UnitPrice,
		item.LineSubtotal, item.DiscountPercentage, item.DiscountAmount,
		item.TaxableAmount, item.TotalTaxes, item.LineTotal,
		item.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to insert purchase line item: %w", err)
	}

	return nil
}

// ============================================
// QUERY OPERATIONS
// ============================================

// GetPurchaseByID retrieves a purchase by ID with all line items
func (s *PurchaseService) GetPurchaseByID(ctx context.Context, companyID, purchaseID string) (*models.Purchase, error) {
	// Get purchase header
	purchase, err := s.getPurchaseHeader(ctx, companyID, purchaseID)
	if err != nil {
		return nil, err
	}

	// Get line items
	lineItems, err := s.getPurchaseLineItems(ctx, purchaseID)
	if err != nil {
		return nil, fmt.Errorf("failed to get line items: %w", err)
	}

	purchase.LineItems = lineItems

	return purchase, nil
}

// getPurchaseHeader retrieves a purchase header
func (s *PurchaseService) getPurchaseHeader(ctx context.Context, companyID, purchaseID string) (*models.Purchase, error) {
	query := `
        SELECT
            id, company_id, establishment_id, point_of_sale_id,
            purchase_number, purchase_type, purchase_date,
            supplier_id,
            supplier_name, supplier_document_type, supplier_document_number,
            supplier_nrc, supplier_activity_code, supplier_activity_desc,
            supplier_address_dept, supplier_address_muni, supplier_address_complement,
            supplier_phone, supplier_email,
            subtotal, total_discount, discount_percentage,
            total_taxes, iva_retained, income_tax_retained, total,
            currency,
            payment_condition, payment_method, payment_reference,
            payment_term, payment_period, payment_status,
            amount_paid, balance_due, due_date,
            dte_numero_control, dte_status, dte_hacienda_response,
            dte_sello_recibido, dte_submitted_at, dte_type,
            status,
            created_at, finalized_at, voided_at,
            created_by, voided_by, notes
        FROM purchases
        WHERE id = $1 AND company_id = $2
    `

	purchase := &models.Purchase{}
	err := database.DB.QueryRowContext(ctx, query, purchaseID, companyID).Scan(
		&purchase.ID, &purchase.CompanyID, &purchase.EstablishmentID, &purchase.PointOfSaleID,
		&purchase.PurchaseNumber, &purchase.PurchaseType, &purchase.PurchaseDate,
		&purchase.SupplierID,
		&purchase.SupplierName, &purchase.SupplierDocumentType, &purchase.SupplierDocumentNumber,
		&purchase.SupplierNRC, &purchase.SupplierActivityCode, &purchase.SupplierActivityDesc,
		&purchase.SupplierAddressDept, &purchase.SupplierAddressMuni, &purchase.SupplierAddressComplement,
		&purchase.SupplierPhone, &purchase.SupplierEmail,
		&purchase.Subtotal, &purchase.TotalDiscount, &purchase.DiscountPercentage,
		&purchase.TotalTaxes, &purchase.IVARetained, &purchase.IncomeTaxRetained, &purchase.Total,
		&purchase.Currency,
		&purchase.PaymentCondition, &purchase.PaymentMethod, &purchase.PaymentReference,
		&purchase.PaymentTerm, &purchase.PaymentPeriod, &purchase.PaymentStatus,
		&purchase.AmountPaid, &purchase.BalanceDue, &purchase.DueDate,
		&purchase.DteNumeroControl, &purchase.DteStatus, &purchase.DteHaciendaResponse,
		&purchase.DteSelloRecibido, &purchase.DteSubmittedAt, &purchase.DteType,
		&purchase.Status,
		&purchase.CreatedAt, &purchase.FinalizedAt, &purchase.VoidedAt,
		&purchase.CreatedBy, &purchase.VoidedBy, &purchase.Notes,
	)

	if err == sql.ErrNoRows {
		return nil, ErrPurchaseNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query purchase: %w", err)
	}

	return purchase, nil
}

// getPurchaseLineItems retrieves line items for a purchase
func (s *PurchaseService) getPurchaseLineItems(ctx context.Context, purchaseID string) ([]models.PurchaseLineItem, error) {
	query := `
        SELECT
            id, purchase_id, line_number,
            item_id, item_code, item_name, item_description,
            item_type, unit_of_measure,
            quantity, unit_price,
            line_subtotal, discount_percentage, discount_amount,
            taxable_amount, total_taxes, line_total,
            created_at
        FROM purchase_line_items
        WHERE purchase_id = $1
        ORDER BY line_number
    `

	rows, err := database.DB.QueryContext(ctx, query, purchaseID)
	if err != nil {
		return nil, fmt.Errorf("failed to query line items: %w", err)
	}
	defer rows.Close()

	var lineItems []models.PurchaseLineItem
	for rows.Next() {
		var item models.PurchaseLineItem
		err := rows.Scan(
			&item.ID, &item.PurchaseID, &item.LineNumber,
			&item.ItemID, &item.ItemCode, &item.ItemName, &item.ItemDescription,
			&item.ItemType, &item.UnitOfMeasure,
			&item.Quantity, &item.UnitPrice,
			&item.LineSubtotal, &item.DiscountPercentage, &item.DiscountAmount,
			&item.TaxableAmount, &item.TotalTaxes, &item.LineTotal,
			&item.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan line item: %w", err)
		}
		lineItems = append(lineItems, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating line items: %w", err)
	}

	return lineItems, nil
}

// ============================================
// LIST OPERATIONS
// ============================================

// ListPurchases retrieves all purchases for a company with pagination
func (s *PurchaseService) ListPurchases(ctx context.Context, companyID string, limit, offset int) ([]models.Purchase, error) {
	query := `
        SELECT
            id, company_id, establishment_id, point_of_sale_id,
            purchase_number, purchase_type, purchase_date,
            supplier_id,
            supplier_name, supplier_document_type, supplier_document_number,
            supplier_nrc, supplier_activity_code, supplier_activity_desc,
            supplier_address_dept, supplier_address_muni, supplier_address_complement,
            supplier_phone, supplier_email,
            subtotal, total_discount, discount_percentage,
            total_taxes, iva_retained, income_tax_retained, total,
            currency,
            payment_condition, payment_method, payment_reference,
            payment_term, payment_period, payment_status,
            amount_paid, balance_due, due_date,
            dte_numero_control, dte_status, dte_hacienda_response,
            dte_sello_recibido, dte_submitted_at, dte_type,
            status,
            created_at, finalized_at, voided_at,
            created_by, voided_by, notes
        FROM purchases
        WHERE company_id = $1
        ORDER BY created_at DESC
        LIMIT $2 OFFSET $3
    `

	rows, err := database.DB.QueryContext(ctx, query, companyID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query purchases: %w", err)
	}
	defer rows.Close()

	var purchases []models.Purchase
	for rows.Next() {
		var p models.Purchase
		err := rows.Scan(
			&p.ID, &p.CompanyID, &p.EstablishmentID, &p.PointOfSaleID,
			&p.PurchaseNumber, &p.PurchaseType, &p.PurchaseDate,
			&p.SupplierID,
			&p.SupplierName, &p.SupplierDocumentType, &p.SupplierDocumentNumber,
			&p.SupplierNRC, &p.SupplierActivityCode, &p.SupplierActivityDesc,
			&p.SupplierAddressDept, &p.SupplierAddressMuni, &p.SupplierAddressComplement,
			&p.SupplierPhone, &p.SupplierEmail,
			&p.Subtotal, &p.TotalDiscount, &p.DiscountPercentage,
			&p.TotalTaxes, &p.IVARetained, &p.IncomeTaxRetained, &p.Total,
			&p.Currency,
			&p.PaymentCondition, &p.PaymentMethod, &p.PaymentReference,
			&p.PaymentTerm, &p.PaymentPeriod, &p.PaymentStatus,
			&p.AmountPaid, &p.BalanceDue, &p.DueDate,
			&p.DteNumeroControl, &p.DteStatus, &p.DteHaciendaResponse,
			&p.DteSelloRecibido, &p.DteSubmittedAt, &p.DteType,
			&p.Status,
			&p.CreatedAt, &p.FinalizedAt, &p.VoidedAt,
			&p.CreatedBy, &p.VoidedBy, &p.Notes,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan purchase: %w", err)
		}
		purchases = append(purchases, p)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating purchases: %w", err)
	}

	return purchases, nil
}

// ============================================
// HELPER FUNCTIONS
// ============================================

// generatePurchaseNumber generates a sequential purchase number
func (s *PurchaseService) generatePurchaseNumber(ctx context.Context, tx *sql.Tx, companyID string) (string, error) {
	var lastNumber sql.NullString
	query := `
        SELECT purchase_number
        FROM purchases
        WHERE company_id = $1
        ORDER BY created_at DESC
        LIMIT 1
    `

	err := tx.QueryRowContext(ctx, query, companyID).Scan(&lastNumber)
	if err != nil && err != sql.ErrNoRows {
		return "", fmt.Errorf("failed to query last purchase number: %w", err)
	}

	var sequence int64 = 1
	if lastNumber.Valid {
		var year int
		n, err := fmt.Sscanf(lastNumber.String, "PUR-%d-%d", &year, &sequence)
		if err != nil || n != 2 {
			parts := strings.Split(lastNumber.String, "-")
			if len(parts) == 3 {
				fmt.Sscanf(parts[2], "%d", &sequence)
			}
		}
		sequence++
	}

	currentYear := time.Now().Year()
	purchaseNumber := fmt.Sprintf("PUR-%d-%05d", currentYear, sequence)

	return purchaseNumber, nil
}

// validatePointOfSale validates that a POS exists and is active
func (s *PurchaseService) validatePointOfSale(ctx context.Context, tx *sql.Tx, companyID, establishmentID, posID string) error {
	query := `
        SELECT pos.id
        FROM point_of_sale pos
        JOIN establishments e ON pos.establishment_id = e.id
        WHERE pos.id = $1 
            AND e.id = $2 
            AND e.company_id = $3 
            AND pos.active = true 
            AND e.active = true
    `

	var id string
	err := tx.QueryRowContext(ctx, query, posID, establishmentID, companyID).Scan(&id)
	if err == sql.ErrNoRows {
		return ErrPointOfSaleNotFound
	}
	if err != nil {
		return fmt.Errorf("failed to validate point of sale: %w", err)
	}

	return nil
}

// ============================================
// FINALIZE PURCHASE
// ============================================

// FinalizePurchase finalizes a purchase and submits FSE DTE to Hacienda
func (s *PurchaseService) FinalizePurchase(ctx context.Context, companyID, purchaseID, userID string) (*models.Purchase, error) {
	// Begin transaction
	tx, err := database.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// 1. Get purchase and verify it's a draft (with row lock)
	purchase, err := s.getPurchaseForUpdate(ctx, tx, companyID, purchaseID)
	if err != nil {
		return nil, err
	}

	if purchase.Status != "draft" {
		return nil, ErrPurchaseNotDraft
	}

	// 2. Verify it's an FSE
	if !purchase.IsFSE() {
		return nil, fmt.Errorf("only FSE purchases can be finalized currently")
	}

	// 3. Generate DTE numero control
	numeroControl, err := s.generateNumeroControl(ctx, tx, purchase.EstablishmentID, purchase.PointOfSaleID, "14")
	if err != nil {
		return nil, fmt.Errorf("failed to generate numero control: %w", err)
	}

	// 4. Update purchase to finalized
	now := time.Now()
	updateQuery := `
        UPDATE purchases
        SET status = 'finalized',
            dte_numero_control = $1,
            dte_type = '14',
            dte_status = 'not_submitted',
            finalized_at = $2,
            created_by = $3
        WHERE id = $4 AND company_id = $5
    `

	_, err = tx.ExecContext(ctx, updateQuery,
		numeroControl,
		now,
		userID,
		purchaseID,
		companyID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update purchase: %w", err)
	}

	// 5. Commit transaction
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// 6. Get and return the finalized purchase
	finalizedPurchase, err := s.GetPurchaseByID(ctx, companyID, purchaseID)
	if err != nil {
		return nil, err
	}

	return finalizedPurchase, nil
}

// ============================================
// GENERATE NUMERO CONTROL
// ============================================

// generateNumeroControl generates a numero control for purchase DTE
func (s *PurchaseService) generateNumeroControl(ctx context.Context, tx *sql.Tx, establishmentID, posID, tipoDte string) (string, error) {
	// Load establishment and POS codes
	var codEstablecimiento, codPuntoVenta string
	query := `
        SELECT e.cod_establecimiento, p.cod_punto_venta
        FROM establishments e
        JOIN point_of_sale p ON p.establishment_id = e.id
        WHERE e.id = $1 AND p.id = $2
    `

	err := tx.QueryRowContext(ctx, query, establishmentID, posID).Scan(&codEstablecimiento, &codPuntoVenta)
	if err != nil {
		return "", fmt.Errorf("failed to load establishment codes: %w", err)
	}

	// Get last sequence number for this establishment/POS/type combination
	var lastSeq sql.NullInt64
	seqQuery := `
        SELECT MAX(CAST(SUBSTRING(dte_numero_control FROM 21 FOR 15) AS BIGINT))
        FROM purchases
        WHERE establishment_id = $1 
          AND point_of_sale_id = $2
          AND dte_type = $3
          AND dte_numero_control IS NOT NULL
    `

	err = tx.QueryRowContext(ctx, seqQuery, establishmentID, posID, tipoDte).Scan(&lastSeq)
	if err != nil && err != sql.ErrNoRows {
		return "", fmt.Errorf("failed to query last sequence: %w", err)
	}

	// Increment sequence
	nextSeq := int64(1)
	if lastSeq.Valid {
		nextSeq = lastSeq.Int64 + 1
	}

	// Format: DTE-{tipo}-{codEstable}{codPOS}-{sequence}
	// Example: DTE-14-M001P001-000000000000001
	numeroControl := fmt.Sprintf("DTE-%s-%s%s-%015d",
		tipoDte,
		codEstablecimiento,
		codPuntoVenta,
		nextSeq,
	)

	return numeroControl, nil
}

// ============================================
// HELPER QUERIES
// ============================================

// getPurchaseForUpdate retrieves a purchase with row lock for update
func (s *PurchaseService) getPurchaseForUpdate(ctx context.Context, tx *sql.Tx, companyID, purchaseID string) (*models.Purchase, error) {
	query := `
        SELECT
            id, company_id, establishment_id, point_of_sale_id,
            purchase_number, purchase_type, purchase_date,
            supplier_id,
            supplier_name, supplier_document_type, supplier_document_number,
            supplier_nrc, supplier_activity_code, supplier_activity_desc,
            supplier_address_dept, supplier_address_muni, supplier_address_complement,
            supplier_phone, supplier_email,
            subtotal, total_discount, discount_percentage,
            total_taxes, iva_retained, income_tax_retained, total,
            currency,
            payment_condition, payment_method, payment_reference,
            payment_term, payment_period, payment_status,
            amount_paid, balance_due, due_date,
            dte_numero_control, dte_status, dte_hacienda_response,
            dte_sello_recibido, dte_submitted_at, dte_type,
            status,
            created_at, finalized_at, voided_at,
            created_by, voided_by, notes
        FROM purchases
        WHERE id = $1 AND company_id = $2
        FOR UPDATE
    `

	purchase := &models.Purchase{}
	err := tx.QueryRowContext(ctx, query, purchaseID, companyID).Scan(
		&purchase.ID, &purchase.CompanyID, &purchase.EstablishmentID, &purchase.PointOfSaleID,
		&purchase.PurchaseNumber, &purchase.PurchaseType, &purchase.PurchaseDate,
		&purchase.SupplierID,
		&purchase.SupplierName, &purchase.SupplierDocumentType, &purchase.SupplierDocumentNumber,
		&purchase.SupplierNRC, &purchase.SupplierActivityCode, &purchase.SupplierActivityDesc,
		&purchase.SupplierAddressDept, &purchase.SupplierAddressMuni, &purchase.SupplierAddressComplement,
		&purchase.SupplierPhone, &purchase.SupplierEmail,
		&purchase.Subtotal, &purchase.TotalDiscount, &purchase.DiscountPercentage,
		&purchase.TotalTaxes, &purchase.IVARetained, &purchase.IncomeTaxRetained, &purchase.Total,
		&purchase.Currency,
		&purchase.PaymentCondition, &purchase.PaymentMethod, &purchase.PaymentReference,
		&purchase.PaymentTerm, &purchase.PaymentPeriod, &purchase.PaymentStatus,
		&purchase.AmountPaid, &purchase.BalanceDue, &purchase.DueDate,
		&purchase.DteNumeroControl, &purchase.DteStatus, &purchase.DteHaciendaResponse,
		&purchase.DteSelloRecibido, &purchase.DteSubmittedAt, &purchase.DteType,
		&purchase.Status,
		&purchase.CreatedAt, &purchase.FinalizedAt, &purchase.VoidedAt,
		&purchase.CreatedBy, &purchase.VoidedBy, &purchase.Notes,
	)

	if err == sql.ErrNoRows {
		return nil, ErrPurchaseNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query purchase: %w", err)
	}

	// Load line items
	lineItems, err := s.getPurchaseLineItems(ctx, purchaseID)
	if err != nil {
		return nil, fmt.Errorf("failed to load line items: %w", err)
	}
	purchase.LineItems = lineItems

	return purchase, nil
}
