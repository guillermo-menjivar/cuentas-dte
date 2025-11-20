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
	"github.com/lib/pq"
)

// ============================================
// ERRORS
// ============================================

var (
	ErrRetentionNotFound        = errors.New("retention not found")
	ErrRetentionAlreadyExists   = errors.New("retention already exists for this purchase")
	ErrCompanyNotRetentionAgent = errors.New("company is not a retention agent")
	ErrSupplierNotFormal        = errors.New("supplier must be formal (have NIT and NRC)")
	ErrPurchaseNotEligible      = errors.New("purchase is not eligible for retention")
	ErrPurchaseNotFinalized     = errors.New("purchase must be finalized before creating retention")
	ErrInvalidRetentionRate     = errors.New("invalid retention rate: must be 1.00, 2.00, or 13.00")
)

// ============================================
// SERVICE DEFINITION
// ============================================

type RetentionService struct {
	// Add dependencies as needed
}

func NewRetentionService() *RetentionService {
	return &RetentionService{}
}

// ============================================
// VALIDATE RETENTION ELIGIBILITY
// ============================================

// ValidateRetentionEligibility checks if a purchase is eligible for DTE 07 retention
// Returns validation result with reasons for ineligibility
func (s *RetentionService) ValidateRetentionEligibility(
	ctx context.Context,
	companyID string,
	purchaseID string,
) (*models.RetentionValidationResult, error) {
	result := &models.RetentionValidationResult{
		CanCreateRetention: false,
		Errors:             []string{},
	}

	// 1. Load company
	company, err := s.getCompany(ctx, companyID)
	if err != nil {
		return nil, fmt.Errorf("failed to load company: %w", err)
	}

	// 2. Load purchase
	purchaseService := NewPurchaseService()
	purchase, err := purchaseService.GetPurchaseByID(ctx, companyID, purchaseID)
	if err != nil {
		return nil, fmt.Errorf("failed to load purchase: %w", err)
	}

	// 3. Check if retention already exists
	existing, err := s.getRetentionByPurchaseID(ctx, companyID, purchaseID)
	if err != nil && err != ErrRetentionNotFound {
		return nil, fmt.Errorf("failed to check existing retention: %w", err)
	}
	if existing != nil {
		result.Errors = append(result.Errors, "retention already exists for this purchase")
		result.Reason = "Retention already created"
		return result, nil
	}

	// 4. Validate: Company must be retention agent
	if !company.IsRetentionAgent {
		result.Errors = append(result.Errors, "company is not designated as retention agent")
	}

	// 5. Validate: Company must have valid retention rate
	if err := models.ValidateRetentionRate(company.RetentionRate); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("invalid retention rate: %v", err))
	}

	// 6. Validate: Purchase must be finalized
	if !purchase.IsFinalized() {
		result.Errors = append(result.Errors, "purchase must be finalized first")
	}

	// 7. Validate: Purchase type must NOT be FSE
	if purchase.IsFSE() {
		result.Errors = append(result.Errors, "FSE purchases (DTE 14) cannot have retention - informal suppliers")
	}

	// 8. Validate: Supplier must be formal (have NIT and NRC)
	if purchase.SupplierNRC == nil || *purchase.SupplierNRC == "" {
		result.Errors = append(result.Errors, "supplier must have NRC (formal IVA contributor)")
	}

	// For non-FSE, we'd need supplier NIT from clients table
	// For now, we check if supplier_nrc is present as proxy for formality

	// 9. Validate: Purchase must have taxable amount (venta gravada)
	// For FSE: no IVA, so no retention
	// For regular purchases: would check ventaGravada field
	if purchase.TotalTaxes == 0 && !purchase.IsFSE() {
		// This might be exenta or no sujeta
		result.Errors = append(result.Errors, "purchase must have taxable (gravada) amount")
	}

	// 10. Validate: Purchase must have DTE type 01 or 03
	// FSE is 14, which we already excluded
	if purchase.DteType != "01" && purchase.DteType != "03" {
		result.Errors = append(result.Errors, "purchase DTE must be type 01 (Factura) or 03 (CCF)")
	}

	// Determine eligibility
	if len(result.Errors) == 0 {
		result.CanCreateRetention = true
		result.Reason = "Purchase is eligible for retention"
	} else {
		result.Reason = "Purchase is not eligible for retention"
	}

	return result, nil
}

// ============================================
// CREATE RETENTION
// ============================================

// CreateRetention creates a DTE 07 retention for a purchase
func (s *RetentionService) CreateRetention(
	ctx context.Context,
	companyID string,
	req *models.CreateRetentionRequest,
	userID string,
) (*models.Retention, error) {
	// 1. Validate request
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// 2. Validate eligibility
	validation, err := s.ValidateRetentionEligibility(ctx, companyID, req.PurchaseID)
	if err != nil {
		return nil, fmt.Errorf("failed to validate eligibility: %w", err)
	}
	if !validation.CanCreateRetention {
		return nil, fmt.Errorf("%s: %s", ErrPurchaseNotEligible.Error(), strings.Join(validation.Errors, "; "))
	}

	// 3. Begin transaction
	tx, err := database.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// 4. Load company
	company, err := s.getCompany(ctx, companyID)
	if err != nil {
		return nil, fmt.Errorf("failed to load company: %w", err)
	}

	// 5. Load purchase (with lock)
	purchaseService := NewPurchaseService()
	purchase, err := purchaseService.GetPurchaseByID(ctx, companyID, req.PurchaseID)
	if err != nil {
		return nil, fmt.Errorf("failed to load purchase: %w", err)
	}

	// 6. Calculate retention amounts
	// For FSE: would use subtotal (but FSE can't have retention)
	// For regular: would use ventaGravada field
	// For now, use subtotal as proxy
	montoSujetoGrav := purchase.Subtotal
	ivaRetenido := models.CalculateIVARetention(montoSujetoGrav, company.RetentionRate)
	retentionCode := models.GetRetentionCodeForRate(company.RetentionRate)

	// 7. Generate retention identifiers
	codigoGeneracion := strings.ToUpper(uuid.New().String())
	numeroControl, err := s.generateRetentionNumeroControl(ctx, tx, purchase.EstablishmentID, purchase.PointOfSaleID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate numero control: %w", err)
	}

	// 8. Get ambiente from company
	ambiente := company.DTEAmbiente // "00" or "01"

	// 9. Create retention record
	now := time.Now()
	retention := &models.Retention{
		ID:              strings.ToUpper(uuid.New().String()),
		CompanyID:       companyID,
		PurchaseID:      purchase.ID,
		EstablishmentID: purchase.EstablishmentID,
		PointOfSaleID:   purchase.PointOfSaleID,

		// Supplier snapshot
		SupplierID:   purchase.SupplierID,
		SupplierName: purchase.GetSupplierName(),
		SupplierNIT:  nil, // Would get from clients table if supplier_id present
		SupplierNRC:  purchase.SupplierNRC,

		// DTE identifiers
		CodigoGeneracion: codigoGeneracion,
		NumeroControl:    numeroControl,
		TipoDte:          "07",
		Ambiente:         ambiente,

		// Purchase reference
		PurchaseNumeroControl:    *purchase.DteNumeroControl,
		PurchaseCodigoGeneracion: purchase.ID, // Purchase ID is codigoGeneracion
		PurchaseTipoDte:          purchase.DteType,
		PurchaseFechaEmision:     purchase.PurchaseDate,

		// Retention amounts
		MontoSujetoGrav: round(montoSujetoGrav),
		IVARetenido:     round(ivaRetenido),
		RetentionRate:   company.RetentionRate,
		RetentionCode:   retentionCode,

		// Dates
		FechaEmision: now,

		// DTE data (will be populated after signing)
		DteJSON:   "{}",
		DteSigned: "",

		// Audit
		CreatedBy: &userID,
		CreatedAt: now,
	}

	// 10. Insert retention
	if err := s.insertRetention(ctx, tx, retention); err != nil {
		return nil, fmt.Errorf("failed to insert retention: %w", err)
	}

	// 11. Update purchase to mark retention created
	if err := s.updatePurchaseRetention(ctx, tx, purchase.ID, retention.ID); err != nil {
		return nil, fmt.Errorf("failed to update purchase: %w", err)
	}

	// 12. Commit transaction
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// 13. Return the created retention
	return retention, nil
}

// ============================================
// DATABASE OPERATIONS
// ============================================

// insertRetention inserts a retention record
func (s *RetentionService) insertRetention(ctx context.Context, tx *sql.Tx, retention *models.Retention) error {
	query := `
        INSERT INTO retentions (
            id, company_id, purchase_id, establishment_id, point_of_sale_id,
            supplier_id, supplier_name, supplier_nit, supplier_nrc,
            codigo_generacion, numero_control, tipo_dte, ambiente,
            purchase_numero_control, purchase_codigo_generacion, purchase_tipo_dte, purchase_fecha_emision,
            monto_sujeto_grav, iva_retenido, retention_rate, retention_code,
            fecha_emision, fecha_procesamiento,
            dte_json, dte_signed,
            hacienda_estado, hacienda_sello_recibido, hacienda_fh_procesamiento,
            hacienda_codigo_msg, hacienda_descripcion_msg, hacienda_observaciones, hacienda_response,
            created_by, created_at, submitted_at
        ) VALUES (
            $1, $2, $3, $4, $5,
            $6, $7, $8, $9,
            $10, $11, $12, $13,
            $14, $15, $16, $17,
            $18, $19, $20, $21,
            $22, $23,
            $24, $25,
            $26, $27, $28,
            $29, $30, $31, $32,
            $33, $34, $35
        )
    `

	_, err := tx.ExecContext(ctx, query,
		retention.ID, retention.CompanyID, retention.PurchaseID, retention.EstablishmentID, retention.PointOfSaleID,
		retention.SupplierID, retention.SupplierName, retention.SupplierNIT, retention.SupplierNRC,
		retention.CodigoGeneracion, retention.NumeroControl, retention.TipoDte, retention.Ambiente,
		retention.PurchaseNumeroControl, retention.PurchaseCodigoGeneracion, retention.PurchaseTipoDte, retention.PurchaseFechaEmision,
		retention.MontoSujetoGrav, retention.IVARetenido, retention.RetentionRate, retention.RetentionCode,
		retention.FechaEmision, retention.FechaProcesamiento,
		retention.DteJSON, retention.DteSigned,
		retention.HaciendaEstado, retention.HaciendaSelloRecibido, retention.HaciendaFhProcesamiento,
		retention.HaciendaCodigoMsg, retention.HaciendaDescripcionMsg, pq.Array(retention.HaciendaObservaciones), retention.HaciendaResponse,
		retention.CreatedBy, retention.CreatedAt, retention.SubmittedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to insert retention: %w", err)
	}

	return nil
}

// updatePurchaseRetention marks a purchase as having retention
func (s *RetentionService) updatePurchaseRetention(ctx context.Context, tx *sql.Tx, purchaseID, retentionID string) error {
	query := `
        UPDATE purchases
        SET has_retention = true,
            retention_id = $1
        WHERE id = $2
    `

	_, err := tx.ExecContext(ctx, query, retentionID, purchaseID)
	if err != nil {
		return fmt.Errorf("failed to update purchase retention: %w", err)
	}

	return nil
}

// ============================================
// QUERY OPERATIONS
// ============================================

// GetRetentionByID retrieves a retention by ID
func (s *RetentionService) GetRetentionByID(ctx context.Context, companyID, retentionID string) (*models.Retention, error) {
	query := `
        SELECT
            id, company_id, purchase_id, establishment_id, point_of_sale_id,
            supplier_id, supplier_name, supplier_nit, supplier_nrc,
            codigo_generacion, numero_control, tipo_dte, ambiente,
            purchase_numero_control, purchase_codigo_generacion, purchase_tipo_dte, purchase_fecha_emision,
            monto_sujeto_grav, iva_retenido, retention_rate, retention_code,
            fecha_emision, fecha_procesamiento,
            dte_json, dte_signed,
            hacienda_estado, hacienda_sello_recibido, hacienda_fh_procesamiento,
            hacienda_codigo_msg, hacienda_descripcion_msg, hacienda_observaciones, hacienda_response,
            created_by, created_at, submitted_at
        FROM retentions
        WHERE id = $1 AND company_id = $2
    `

	retention := &models.Retention{}
	var observaciones []string

	err := database.DB.QueryRowContext(ctx, query, retentionID, companyID).Scan(
		&retention.ID, &retention.CompanyID, &retention.PurchaseID, &retention.EstablishmentID, &retention.PointOfSaleID,
		&retention.SupplierID, &retention.SupplierName, &retention.SupplierNIT, &retention.SupplierNRC,
		&retention.CodigoGeneracion, &retention.NumeroControl, &retention.TipoDte, &retention.Ambiente,
		&retention.PurchaseNumeroControl, &retention.PurchaseCodigoGeneracion, &retention.PurchaseTipoDte, &retention.PurchaseFechaEmision,
		&retention.MontoSujetoGrav, &retention.IVARetenido, &retention.RetentionRate, &retention.RetentionCode,
		&retention.FechaEmision, &retention.FechaProcesamiento,
		&retention.DteJSON, &retention.DteSigned,
		&retention.HaciendaEstado, &retention.HaciendaSelloRecibido, &retention.HaciendaFhProcesamiento,
		&retention.HaciendaCodigoMsg, &retention.HaciendaDescripcionMsg, pq.Array(&observaciones), &retention.HaciendaResponse,
		&retention.CreatedBy, &retention.CreatedAt, &retention.SubmittedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrRetentionNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query retention: %w", err)
	}

	retention.HaciendaObservaciones = observaciones

	return retention, nil
}

// getRetentionByPurchaseID retrieves a retention by purchase ID
func (s *RetentionService) getRetentionByPurchaseID(ctx context.Context, companyID, purchaseID string) (*models.Retention, error) {
	query := `
        SELECT
            id, company_id, purchase_id, establishment_id, point_of_sale_id,
            supplier_id, supplier_name, supplier_nit, supplier_nrc,
            codigo_generacion, numero_control, tipo_dte, ambiente,
            purchase_numero_control, purchase_codigo_generacion, purchase_tipo_dte, purchase_fecha_emision,
            monto_sujeto_grav, iva_retenido, retention_rate, retention_code,
            fecha_emision, fecha_procesamiento,
            dte_json, dte_signed,
            hacienda_estado, hacienda_sello_recibido, hacienda_fh_procesamiento,
            hacienda_codigo_msg, hacienda_descripcion_msg, hacienda_observaciones, hacienda_response,
            created_by, created_at, submitted_at
        FROM retentions
        WHERE purchase_id = $1 AND company_id = $2
    `

	retention := &models.Retention{}
	var observaciones []string

	err := database.DB.QueryRowContext(ctx, query, purchaseID, companyID).Scan(
		&retention.ID, &retention.CompanyID, &retention.PurchaseID, &retention.EstablishmentID, &retention.PointOfSaleID,
		&retention.SupplierID, &retention.SupplierName, &retention.SupplierNIT, &retention.SupplierNRC,
		&retention.CodigoGeneracion, &retention.NumeroControl, &retention.TipoDte, &retention.Ambiente,
		&retention.PurchaseNumeroControl, &retention.PurchaseCodigoGeneracion, &retention.PurchaseTipoDte, &retention.PurchaseFechaEmision,
		&retention.MontoSujetoGrav, &retention.IVARetenido, &retention.RetentionRate, &retention.RetentionCode,
		&retention.FechaEmision, &retention.FechaProcesamiento,
		&retention.DteJSON, &retention.DteSigned,
		&retention.HaciendaEstado, &retention.HaciendaSelloRecibido, &retention.HaciendaFhProcesamiento,
		&retention.HaciendaCodigoMsg, &retention.HaciendaDescripcionMsg, pq.Array(&observaciones), &retention.HaciendaResponse,
		&retention.CreatedBy, &retention.CreatedAt, &retention.SubmittedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrRetentionNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query retention: %w", err)
	}

	retention.HaciendaObservaciones = observaciones

	return retention, nil
}

// ============================================
// LIST OPERATIONS
// ============================================

// ListRetentions retrieves all retentions for a company with pagination
func (s *RetentionService) ListRetentions(ctx context.Context, companyID string, limit, offset int) ([]models.Retention, error) {
	query := `
        SELECT
            id, company_id, purchase_id, establishment_id, point_of_sale_id,
            supplier_id, supplier_name, supplier_nit, supplier_nrc,
            codigo_generacion, numero_control, tipo_dte, ambiente,
            purchase_numero_control, purchase_codigo_generacion, purchase_tipo_dte, purchase_fecha_emision,
            monto_sujeto_grav, iva_retenido, retention_rate, retention_code,
            fecha_emision, fecha_procesamiento,
            dte_json, dte_signed,
            hacienda_estado, hacienda_sello_recibido, hacienda_fh_procesamiento,
            hacienda_codigo_msg, hacienda_descripcion_msg, hacienda_observaciones, hacienda_response,
            created_by, created_at, submitted_at
        FROM retentions
        WHERE company_id = $1
        ORDER BY created_at DESC
        LIMIT $2 OFFSET $3
    `

	rows, err := database.DB.QueryContext(ctx, query, companyID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query retentions: %w", err)
	}
	defer rows.Close()

	var retentions []models.Retention
	for rows.Next() {
		var r models.Retention
		var observaciones []string

		err := rows.Scan(
			&r.ID, &r.CompanyID, &r.PurchaseID, &r.EstablishmentID, &r.PointOfSaleID,
			&r.SupplierID, &r.SupplierName, &r.SupplierNIT, &r.SupplierNRC,
			&r.CodigoGeneracion, &r.NumeroControl, &r.TipoDte, &r.Ambiente,
			&r.PurchaseNumeroControl, &r.PurchaseCodigoGeneracion, &r.PurchaseTipoDte, &r.PurchaseFechaEmision,
			&r.MontoSujetoGrav, &r.IVARetenido, &r.RetentionRate, &r.RetentionCode,
			&r.FechaEmision, &r.FechaProcesamiento,
			&r.DteJSON, &r.DteSigned,
			&r.HaciendaEstado, &r.HaciendaSelloRecibido, &r.HaciendaFhProcesamiento,
			&r.HaciendaCodigoMsg, &r.HaciendaDescripcionMsg, pq.Array(&observaciones), &r.HaciendaResponse,
			&r.CreatedBy, &r.CreatedAt, &r.SubmittedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan retention: %w", err)
		}

		r.HaciendaObservaciones = observaciones
		retentions = append(retentions, r)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating retentions: %w", err)
	}

	return retentions, nil
}

// ============================================
// HELPER FUNCTIONS
// ============================================

// generateRetentionNumeroControl generates a numero control for retention DTE 07
func (s *RetentionService) generateRetentionNumeroControl(ctx context.Context, tx *sql.Tx, establishmentID, posID string) (string, error) {
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
        SELECT MAX(CAST(SUBSTRING(numero_control FROM 21 FOR 15) AS BIGINT))
        FROM retentions
        WHERE establishment_id = $1 
          AND point_of_sale_id = $2
          AND numero_control IS NOT NULL
    `

	err = tx.QueryRowContext(ctx, seqQuery, establishmentID, posID).Scan(&lastSeq)
	if err != nil && err != sql.ErrNoRows {
		return "", fmt.Errorf("failed to query last sequence: %w", err)
	}

	// Increment sequence
	nextSeq := int64(1)
	if lastSeq.Valid {
		nextSeq = lastSeq.Int64 + 1
	}

	// Format: DTE-07-{codEstable}{codPOS}-{sequence}
	// Example: DTE-07-M001P001-000000000000001
	numeroControl := fmt.Sprintf("DTE-07-%s%s-%015d",
		codEstablecimiento,
		codPuntoVenta,
		nextSeq,
	)

	return numeroControl, nil
}

// getCompany loads company with retention configuration
func (s *RetentionService) getCompany(ctx context.Context, companyID string) (*CompanyRetentionConfig, error) {
	query := `
        SELECT id, is_retention_agent, retention_rate, dte_ambiente
        FROM companies
        WHERE id = $1
    `

	company := &CompanyRetentionConfig{}
	err := database.DB.QueryRowContext(ctx, query, companyID).Scan(
		&company.ID,
		&company.IsRetentionAgent,
		&company.RetentionRate,
		&company.DTEAmbiente,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("company not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query company: %w", err)
	}

	return company, nil
}

// CompanyRetentionConfig holds company retention configuration
type CompanyRetentionConfig struct {
	ID               string
	IsRetentionAgent bool
	RetentionRate    float64
	DTEAmbiente      string
}
