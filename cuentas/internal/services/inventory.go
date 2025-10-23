package services

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/binary"
	"fmt"
	"time"

	"cuentas/internal/codigos"
	"cuentas/internal/models"
)

type InventoryService struct {
	db *sql.DB
}

func NewInventoryService(db *sql.DB) *InventoryService {
	return &InventoryService{db: db}
}

// CreateItem creates a new inventory item with its taxes
func (s *InventoryService) CreateItem(ctx context.Context, companyID string, req *models.CreateInventoryItemRequest) (*models.InventoryItem, error) {
	// Generate SKU if not provided
	sku := ""
	if req.SKU != nil {
		sku = *req.SKU
	} else {
		var err error
		sku, err = s.generateSKU(ctx, companyID, req.TipoItem)
		if err != nil {
			return nil, fmt.Errorf("failed to generate SKU: %w", err)
		}
	}

	// Generate barcode if not provided (only for goods)
	var barcode *string
	if req.CodigoBarras != nil {
		barcode = req.CodigoBarras
	} else if req.TipoItem == "1" { // Only for Bienes
		generated := s.generateBarcode()
		barcode = &generated
	}

	// Start a transaction
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Insert the inventory item
	query := `
		INSERT INTO inventory_items (
			company_id, tipo_item, sku, codigo_barras,
			name, description, manufacturer, image_url,
			unit_of_measure, color, is_tax_exempt
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id, company_id, tipo_item, sku, codigo_barras,
				  name, description, manufacturer, image_url,
				  unit_of_measure, color, is_tax_exempt,
				  active, created_at, updated_at
	`

	var item models.InventoryItem
	err = tx.QueryRowContext(ctx, query,
		companyID, req.TipoItem, sku, barcode,
		req.Name, req.Description, req.Manufacturer, req.ImageURL,
		req.UnitOfMeasure, req.Color, req.IsTaxExempt,
	).Scan(
		&item.ID, &item.CompanyID, &item.TipoItem, &item.SKU, &item.CodigoBarras,
		&item.Name, &item.Description, &item.Manufacturer, &item.ImageURL,
		&item.UnitOfMeasure, &item.Color, &item.IsTaxExempt,
		&item.Active, &item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create item: %w", err)
	}

	// If no taxes provided, add default based on tipo_item
	taxes := req.Taxes
	if len(taxes) == 0 {
		taxes = getDefaultTaxes(req.TipoItem)
	}

	// Insert taxes
	for _, tax := range taxes {
		taxQuery := `
			INSERT INTO inventory_item_taxes (item_id, tributo_code)
			VALUES ($1, $2)
		`
		_, err = tx.ExecContext(ctx, taxQuery, item.ID, tax.TributoCode)
		if err != nil {
			return nil, fmt.Errorf("failed to add tax %s: %w", tax.TributoCode, err)
		}
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Load taxes for response
	item.Taxes, err = s.GetItemTaxes(ctx, companyID, item.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to load taxes: %w", err)
	}

	return &item, nil
}

// GetItemByID retrieves an inventory item by ID
func (s *InventoryService) GetItemByID(ctx context.Context, companyID, itemID string) (*models.InventoryItem, error) {
	query := `
		SELECT id, company_id, tipo_item, sku, codigo_barras,
			   name, description, manufacturer, image_url,
			   unit_of_measure, color, is_tax_exempt,
			   active, created_at, updated_at
		FROM inventory_items
		WHERE id = $1 AND company_id = $2
	`

	var item models.InventoryItem
	err := s.db.QueryRowContext(ctx, query, itemID, companyID).Scan(
		&item.ID, &item.CompanyID, &item.TipoItem, &item.SKU, &item.CodigoBarras,
		&item.Name, &item.Description, &item.Manufacturer, &item.ImageURL,
		&item.UnitOfMeasure, &item.Color, &item.IsTaxExempt,
		&item.Active, &item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	// Load taxes
	item.Taxes, err = s.GetItemTaxes(ctx, companyID, item.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to load taxes: %w", err)
	}

	return &item, nil
}

// ListItems retrieves all inventory items for a company with optional filters
func (s *InventoryService) ListItems(ctx context.Context, companyID string, activeOnly bool, tipoItem string) ([]models.InventoryItem, error) {
	query := `
		SELECT id, company_id, tipo_item, sku, codigo_barras,
			   name, description, manufacturer, image_url,
			   unit_of_measure, color, is_tax_exempt,
			   active, created_at, updated_at
		FROM inventory_items
		WHERE company_id = $1
	`

	args := []interface{}{companyID}
	argCount := 1

	if activeOnly {
		argCount++
		query += fmt.Sprintf(" AND active = $%d", argCount)
		args = append(args, true)
	}

	if tipoItem != "" {
		argCount++
		query += fmt.Sprintf(" AND tipo_item = $%d", argCount)
		args = append(args, tipoItem)
	}

	query += " ORDER BY name ASC"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list items: %w", err)
	}
	defer rows.Close()

	var items []models.InventoryItem
	for rows.Next() {
		var item models.InventoryItem
		err := rows.Scan(
			&item.ID, &item.CompanyID, &item.TipoItem, &item.SKU, &item.CodigoBarras,
			&item.Name, &item.Description, &item.Manufacturer, &item.ImageURL,
			&item.UnitOfMeasure, &item.Color, &item.IsTaxExempt,
			&item.Active, &item.CreatedAt, &item.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan item: %w", err)
		}

		// Load taxes for each item
		item.Taxes, _ = s.GetItemTaxes(ctx, companyID, item.ID)

		items = append(items, item)
	}

	return items, nil
}

// UpdateItem updates an inventory item
func (s *InventoryService) UpdateItem(ctx context.Context, companyID, itemID string, req *models.UpdateInventoryItemRequest) (*models.InventoryItem, error) {
	// Build dynamic update query
	query := "UPDATE inventory_items SET updated_at = CURRENT_TIMESTAMP"
	args := []interface{}{}
	argCount := 0

	if req.Name != nil {
		argCount++
		query += fmt.Sprintf(", name = $%d", argCount)
		args = append(args, *req.Name)
	}
	if req.Description != nil {
		argCount++
		query += fmt.Sprintf(", description = $%d", argCount)
		args = append(args, *req.Description)
	}
	if req.Manufacturer != nil {
		argCount++
		query += fmt.Sprintf(", manufacturer = $%d", argCount)
		args = append(args, *req.Manufacturer)
	}
	if req.ImageURL != nil {
		argCount++
		query += fmt.Sprintf(", image_url = $%d", argCount)
		args = append(args, *req.ImageURL)
	}
	if req.UnitOfMeasure != nil {
		argCount++
		query += fmt.Sprintf(", unit_of_measure = $%d", argCount)
		args = append(args, *req.UnitOfMeasure)
	}
	if req.Color != nil {
		argCount++
		query += fmt.Sprintf(", color = $%d", argCount)
		args = append(args, *req.Color)
	}

	// Add WHERE clause
	argCount++
	query += fmt.Sprintf(" WHERE id = $%d", argCount)
	args = append(args, itemID)

	argCount++
	query += fmt.Sprintf(" AND company_id = $%d", argCount)
	args = append(args, companyID)

	result, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to update item: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("failed to check affected rows: %w", err)
	}

	if rowsAffected == 0 {
		return nil, sql.ErrNoRows
	}

	// Return updated item
	return s.GetItemByID(ctx, companyID, itemID)
}

// DeleteItem soft deletes an inventory item
func (s *InventoryService) DeleteItem(ctx context.Context, companyID, itemID string) error {
	query := `
		UPDATE inventory_items
		SET active = false, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1 AND company_id = $2
	`

	result, err := s.db.ExecContext(ctx, query, itemID, companyID)
	if err != nil {
		return fmt.Errorf("failed to delete item: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check affected rows: %w", err)
	}

	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

// GetItemTaxes retrieves all taxes for an item
func (s *InventoryService) GetItemTaxes(ctx context.Context, companyID, itemID string) ([]models.InventoryItemTax, error) {
	query := `
		SELECT t.id, t.item_id, t.tributo_code, t.created_at
		FROM inventory_item_taxes t
		JOIN inventory_items i ON t.item_id = i.id
		WHERE t.item_id = $1 AND i.company_id = $2
		ORDER BY t.created_at ASC
	`

	rows, err := s.db.QueryContext(ctx, query, itemID, companyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get item taxes: %w", err)
	}
	defer rows.Close()

	var taxes []models.InventoryItemTax
	for rows.Next() {
		var tax models.InventoryItemTax
		err := rows.Scan(&tax.ID, &tax.ItemID, &tax.TributoCode, &tax.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan tax: %w", err)
		}
		taxes = append(taxes, tax)
	}

	return taxes, nil
}

// AddItemTax adds a tax to an item
func (s *InventoryService) AddItemTax(ctx context.Context, companyID, itemID string, req *models.AddItemTaxRequest) (*models.InventoryItemTax, error) {
	// Verify item exists and belongs to company
	_, err := s.GetItemByID(ctx, companyID, itemID)
	if err != nil {
		return nil, fmt.Errorf("item not found: %w", err)
	}

	query := `
		INSERT INTO inventory_item_taxes (item_id, tributo_code)
		VALUES ($1, $2)
		RETURNING id, item_id, tributo_code, created_at
	`

	var tax models.InventoryItemTax
	err = s.db.QueryRowContext(ctx, query, itemID, req.TributoCode).Scan(
		&tax.ID, &tax.ItemID, &tax.TributoCode, &tax.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to add tax: %w", err)
	}

	return &tax, nil
}

// RemoveItemTax removes a tax from an item
func (s *InventoryService) RemoveItemTax(ctx context.Context, companyID, itemID, tributoCode string) error {
	// Verify item exists and belongs to company
	_, err := s.GetItemByID(ctx, companyID, itemID)
	if err != nil {
		return fmt.Errorf("item not found: %w", err)
	}

	query := `
		DELETE FROM inventory_item_taxes
		WHERE item_id = $1 AND tributo_code = $2
	`

	result, err := s.db.ExecContext(ctx, query, itemID, tributoCode)
	if err != nil {
		return fmt.Errorf("failed to remove tax: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check affected rows: %w", err)
	}

	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

// getDefaultTaxes returns default taxes based on tipo_item
func getDefaultTaxes(tipoItem string) []models.AddItemTaxRequest {
	switch tipoItem {
	case "1": // Bienes
		return []models.AddItemTaxRequest{
			{TributoCode: codigos.TributoIVA13}, // S1.20 - IVA 13%
		}
	case "2": // Servicios
		return []models.AddItemTaxRequest{
			{TributoCode: codigos.TributoIVA13}, // S1.20 - IVA 13%
		}
	default:
		return []models.AddItemTaxRequest{}
	}
}

// generateSKU creates a unique SKU for the company
func (s *InventoryService) generateSKU(ctx context.Context, companyID, tipoItem string) (string, error) {
	prefix := "PROD"
	if tipoItem == "2" {
		prefix = "SRV"
	}

	for attempts := 0; attempts < 10; attempts++ {
		timestamp := time.Now().Format("20060102")

		// Generate cryptographically secure random number
		var randomBytes [4]byte
		_, err := rand.Read(randomBytes[:])
		if err != nil {
			return "", fmt.Errorf("failed to generate random number: %w", err)
		}
		random := binary.BigEndian.Uint32(randomBytes[:]) % 10000

		sku := fmt.Sprintf("%s-%s-%04d", prefix, timestamp, random)

		exists, err := s.skuExists(ctx, companyID, sku)
		if err != nil {
			return "", err
		}
		if !exists {
			return sku, nil
		}
	}

	return "", fmt.Errorf("failed to generate unique SKU after 10 attempts")
}

// skuExists checks if a SKU already exists for the company
func (s *InventoryService) skuExists(ctx context.Context, companyID, sku string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM inventory_items WHERE company_id = $1 AND sku = $2)`
	err := s.db.QueryRowContext(ctx, query, companyID, sku).Scan(&exists)
	return exists, err
}

// generateBarcode creates an EAN-13 style barcode
func (s *InventoryService) generateBarcode() string {
	var randomBytes [8]byte
	rand.Read(randomBytes[:])
	random := binary.BigEndian.Uint64(randomBytes[:]) % 100000000000
	return fmt.Sprintf("20%011d", random)
}
