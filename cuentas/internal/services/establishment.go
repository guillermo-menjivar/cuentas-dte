package services

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"cuentas/internal/database"
	"cuentas/internal/dte"
	"cuentas/internal/models"
)

type EstablishmentService struct{}

func NewEstablishmentService() *EstablishmentService {
	return &EstablishmentService{}
}

// CreateEstablishment creates a new establishment for a company
func (s *EstablishmentService) CreateEstablishment(ctx context.Context, companyID string, req *models.CreateEstablishmentRequest) (*models.Establishment, error) {
	// Validate request
	if err := req.Validate(); err != nil {
		return nil, err
	}
	var codEstablecimiento string

	if req.CodEstablecimiento != nil && *req.CodEstablecimiento != "" {
		// Parse the number from whatever format user sent ("0001", "1", etc.)
		number, err := strconv.Atoi(strings.TrimLeft(*req.CodEstablecimiento, "0"))
		if err != nil {
			return nil, fmt.Errorf("invalid cod_establecimiento: must be numeric")
		}
		// Format with proper prefix based on tipo
		codEstablecimiento = dte.FormatEstablishmentCode(req.TipoEstablecimiento, number)
	}

	establishment := &models.Establishment{
		CompanyID:            companyID,
		TipoEstablecimiento:  req.TipoEstablecimiento,
		Nombre:               req.Nombre,
		CodEstablecimiento:   &codEstablecimiento,
		Departamento:         req.Departamento,
		Municipio:            req.Municipio,
		ComplementoDireccion: req.ComplementoDireccion,
		Telefono:             req.Telefono,
		Active:               true,
	}

	now := time.Now()
	err := database.DB.QueryRowContext(ctx, query,
		establishment.CompanyID,
		establishment.TipoEstablecimiento,
		establishment.Nombre,
		establishment.CodEstablecimiento,
		establishment.Departamento,
		establishment.Municipio,
		establishment.ComplementoDireccion,
		establishment.Telefono,
		establishment.Active,
		now,
		now,
	).Scan(&establishment.ID, &establishment.CreatedAt, &establishment.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create establishment: %w", err)
	}

	return establishment, nil
}

// GetEstablishment retrieves an establishment by ID
func (s *EstablishmentService) GetEstablishment(ctx context.Context, companyID, establishmentID string) (*models.Establishment, error) {
	query := `
		SELECT
			id, company_id, tipo_establecimiento, nombre,
			cod_establecimiento,
			departamento, municipio, complemento_direccion,
			telefono, active, created_at, updated_at
		FROM establishments
		WHERE id = $1 AND company_id = $2
	`

	establishment := &models.Establishment{}
	err := database.DB.QueryRowContext(ctx, query, establishmentID, companyID).Scan(
		&establishment.ID,
		&establishment.CompanyID,
		&establishment.TipoEstablecimiento,
		&establishment.Nombre,
		&establishment.CodEstablecimiento,
		&establishment.Departamento,
		&establishment.Municipio,
		&establishment.ComplementoDireccion,
		&establishment.Telefono,
		&establishment.Active,
		&establishment.CreatedAt,
		&establishment.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, models.ErrEstablishmentNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get establishment: %w", err)
	}

	return establishment, nil
}

// ListEstablishments retrieves all establishments for a company
func (s *EstablishmentService) ListEstablishments(ctx context.Context, companyID string, activeOnly bool) ([]models.Establishment, error) {
	query := `
		SELECT
			id, company_id, tipo_establecimiento, nombre,
			cod_establecimiento,
			departamento, municipio, complemento_direccion,
			telefono, active, created_at, updated_at
		FROM establishments
		WHERE company_id = $1
	`

	if activeOnly {
		query += " AND active = true"
	}

	query += " ORDER BY created_at DESC"

	rows, err := database.DB.QueryContext(ctx, query, companyID)
	if err != nil {
		return nil, fmt.Errorf("failed to list establishments: %w", err)
	}
	defer rows.Close()

	var establishments []models.Establishment
	for rows.Next() {
		var est models.Establishment
		err := rows.Scan(
			&est.ID,
			&est.CompanyID,
			&est.TipoEstablecimiento,
			&est.Nombre,
			&est.CodEstablecimiento,
			&est.Departamento,
			&est.Municipio,
			&est.ComplementoDireccion,
			&est.Telefono,
			&est.Active,
			&est.CreatedAt,
			&est.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan establishment: %w", err)
		}
		establishments = append(establishments, est)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating establishments: %w", err)
	}

	return establishments, nil
}

// UpdateEstablishment updates an establishment
func (s *EstablishmentService) UpdateEstablishment(ctx context.Context, companyID, establishmentID string, req *models.UpdateEstablishmentRequest) (*models.Establishment, error) {
	// First verify establishment exists and belongs to company
	existing, err := s.GetEstablishment(ctx, companyID, establishmentID)
	if err != nil {
		return nil, err
	}

	// Build dynamic update query
	updates := []string{}
	args := []interface{}{establishmentID, companyID}
	argCount := 2

	if req.TipoEstablecimiento != nil {
		if !models.IsValidTipoEstablecimiento(*req.TipoEstablecimiento) {
			return nil, models.ErrInvalidTipoEstablecimiento
		}
		argCount++
		updates = append(updates, fmt.Sprintf("tipo_establecimiento = $%d", argCount))
		args = append(args, *req.TipoEstablecimiento)
	}

	if req.Nombre != nil {
		argCount++
		updates = append(updates, fmt.Sprintf("nombre = $%d", argCount))
		args = append(args, *req.Nombre)
	}

	if req.CodEstablecimiento != nil {
		if len(*req.CodEstablecimiento) != 4 {
			return nil, models.ErrInvalidCodEstablecimiento
		}
		argCount++
		updates = append(updates, fmt.Sprintf("cod_establecimiento = $%d", argCount))
		args = append(args, *req.CodEstablecimiento)
	}

	if req.Departamento != nil {
		argCount++
		updates = append(updates, fmt.Sprintf("departamento = $%d", argCount))
		args = append(args, *req.Departamento)
	}

	if req.Municipio != nil {
		argCount++
		updates = append(updates, fmt.Sprintf("municipio = $%d", argCount))
		args = append(args, *req.Municipio)
	}

	if req.ComplementoDireccion != nil {
		argCount++
		updates = append(updates, fmt.Sprintf("complemento_direccion = $%d", argCount))
		args = append(args, *req.ComplementoDireccion)
	}

	if req.Telefono != nil {
		argCount++
		updates = append(updates, fmt.Sprintf("telefono = $%d", argCount))
		args = append(args, *req.Telefono)
	}

	if len(updates) == 0 {
		return existing, nil // Nothing to update
	}

	// Add updated_at
	argCount++
	updates = append(updates, fmt.Sprintf("updated_at = $%d", argCount))
	args = append(args, time.Now())

	query := fmt.Sprintf(`
		UPDATE establishments
		SET %s
		WHERE id = $1 AND company_id = $2
		RETURNING id, company_id, tipo_establecimiento, nombre,
			cod_establecimiento,
			departamento, municipio, complemento_direccion,
			telefono, active, created_at, updated_at
	`, joinStrings(updates, ", "))

	establishment := &models.Establishment{}
	err = database.DB.QueryRowContext(ctx, query, args...).Scan(
		&establishment.ID,
		&establishment.CompanyID,
		&establishment.TipoEstablecimiento,
		&establishment.Nombre,
		&establishment.CodEstablecimiento,
		&establishment.Departamento,
		&establishment.Municipio,
		&establishment.ComplementoDireccion,
		&establishment.Telefono,
		&establishment.Active,
		&establishment.CreatedAt,
		&establishment.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to update establishment: %w", err)
	}

	return establishment, nil
}

// DeactivateEstablishment deactivates an establishment
func (s *EstablishmentService) DeactivateEstablishment(ctx context.Context, companyID, establishmentID string) error {
	query := `
		UPDATE establishments
		SET active = false, updated_at = $1
		WHERE id = $2 AND company_id = $3
	`

	result, err := database.DB.ExecContext(ctx, query, time.Now(), establishmentID, companyID)
	if err != nil {
		return fmt.Errorf("failed to deactivate establishment: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return models.ErrEstablishmentNotFound
	}

	return nil
}

// CreatePointOfSale creates a new point of sale for an establishment
func (s *EstablishmentService) CreatePointOfSale(ctx context.Context, companyID, establishmentID string, req *models.CreatePOSRequest) (*models.PointOfSale, error) {
	// Validate request
	if err := req.Validate(); err != nil {
		return nil, err
	}

	// Verify establishment exists and belongs to company
	_, err := s.GetEstablishment(ctx, companyID, establishmentID)
	if err != nil {
		return nil, err
	}

	query := `
		INSERT INTO point_of_sale (
			establishment_id, nombre,
			cod_punto_venta,
			latitude, longitude, is_portable,
			active, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9
		) RETURNING id, created_at, updated_at
	`

	pos := &models.PointOfSale{
		EstablishmentID: establishmentID,
		Nombre:          req.Nombre,
		CodPuntoVenta:   req.CodPuntoVenta,
		Latitude:        req.Latitude,
		Longitude:       req.Longitude,
		IsPortable:      req.IsPortable,
		Active:          true,
	}

	now := time.Now()
	err = database.DB.QueryRowContext(ctx, query,
		pos.EstablishmentID,
		pos.Nombre,
		pos.CodPuntoVenta,
		pos.Latitude,
		pos.Longitude,
		pos.IsPortable,
		pos.Active,
		now,
		now,
	).Scan(&pos.ID, &pos.CreatedAt, &pos.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create point of sale: %w", err)
	}

	return pos, nil
}

// GetPointOfSale retrieves a point of sale by ID
func (s *EstablishmentService) GetPointOfSale(ctx context.Context, companyID, posID string) (*models.PointOfSale, error) {
	query := `
		SELECT
			pos.id, pos.establishment_id, pos.nombre,
			pos.cod_punto_venta,
			pos.latitude, pos.longitude, pos.is_portable,
			pos.active, pos.created_at, pos.updated_at
		FROM point_of_sale pos
		JOIN establishments e ON pos.establishment_id = e.id
		WHERE pos.id = $1 AND e.company_id = $2
	`

	pos := &models.PointOfSale{}
	err := database.DB.QueryRowContext(ctx, query, posID, companyID).Scan(
		&pos.ID,
		&pos.EstablishmentID,
		&pos.Nombre,
		&pos.CodPuntoVenta,
		&pos.Latitude,
		&pos.Longitude,
		&pos.IsPortable,
		&pos.Active,
		&pos.CreatedAt,
		&pos.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, models.ErrPointOfSaleNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get point of sale: %w", err)
	}

	return pos, nil
}

// ListPointsOfSale retrieves all points of sale for an establishment
func (s *EstablishmentService) ListPointsOfSale(ctx context.Context, companyID, establishmentID string, activeOnly bool) ([]models.PointOfSale, error) {
	// Verify establishment belongs to company
	_, err := s.GetEstablishment(ctx, companyID, establishmentID)
	if err != nil {
		return nil, err
	}

	query := `
		SELECT
			id, establishment_id, nombre,
			cod_punto_venta,
			latitude, longitude, is_portable,
			active, created_at, updated_at
		FROM point_of_sale
		WHERE establishment_id = $1
	`

	if activeOnly {
		query += " AND active = true"
	}

	query += " ORDER BY created_at DESC"

	rows, err := database.DB.QueryContext(ctx, query, establishmentID)
	if err != nil {
		return nil, fmt.Errorf("failed to list points of sale: %w", err)
	}
	defer rows.Close()

	var pointsOfSale []models.PointOfSale
	for rows.Next() {
		var pos models.PointOfSale
		err := rows.Scan(
			&pos.ID,
			&pos.EstablishmentID,
			&pos.Nombre,
			&pos.CodPuntoVenta,
			&pos.Latitude,
			&pos.Longitude,
			&pos.IsPortable,
			&pos.Active,
			&pos.CreatedAt,
			&pos.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan point of sale: %w", err)
		}
		pointsOfSale = append(pointsOfSale, pos)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating points of sale: %w", err)
	}

	return pointsOfSale, nil
}

// UpdatePointOfSale updates a point of sale
func (s *EstablishmentService) UpdatePointOfSale(ctx context.Context, companyID, posID string, req *models.UpdatePOSRequest) (*models.PointOfSale, error) {
	// Verify POS exists and belongs to company
	existing, err := s.GetPointOfSale(ctx, companyID, posID)
	if err != nil {
		return nil, err
	}

	// Build dynamic update query
	updates := []string{}
	args := []interface{}{posID}
	argCount := 1

	if req.Nombre != nil {
		argCount++
		updates = append(updates, fmt.Sprintf("nombre = $%d", argCount))
		args = append(args, *req.Nombre)
	}

	if req.CodPuntoVenta != nil {
		if len(*req.CodPuntoVenta) != 4 {
			return nil, models.ErrInvalidCodPuntoVenta
		}
		argCount++
		updates = append(updates, fmt.Sprintf("cod_punto_venta = $%d", argCount))
		args = append(args, *req.CodPuntoVenta)
	}

	if req.Latitude != nil {
		argCount++
		updates = append(updates, fmt.Sprintf("latitude = $%d", argCount))
		args = append(args, *req.Latitude)
	}

	if req.Longitude != nil {
		argCount++
		updates = append(updates, fmt.Sprintf("longitude = $%d", argCount))
		args = append(args, *req.Longitude)
	}

	if req.IsPortable != nil {
		argCount++
		updates = append(updates, fmt.Sprintf("is_portable = $%d", argCount))
		args = append(args, *req.IsPortable)
	}

	if len(updates) == 0 {
		return existing, nil // Nothing to update
	}

	// Add updated_at
	argCount++
	updates = append(updates, fmt.Sprintf("updated_at = $%d", argCount))
	args = append(args, time.Now())

	query := fmt.Sprintf(`
		UPDATE point_of_sale
		SET %s
		WHERE id = $1
		RETURNING id, establishment_id, nombre,
			cod_punto_venta,
			latitude, longitude, is_portable,
			active, created_at, updated_at
	`, joinStrings(updates, ", "))

	pos := &models.PointOfSale{}
	err = database.DB.QueryRowContext(ctx, query, args...).Scan(
		&pos.ID,
		&pos.EstablishmentID,
		&pos.Nombre,
		&pos.CodPuntoVenta,
		&pos.Latitude,
		&pos.Longitude,
		&pos.IsPortable,
		&pos.Active,
		&pos.CreatedAt,
		&pos.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to update point of sale: %w", err)
	}

	return pos, nil
}

// UpdatePOSLocation updates only the GPS coordinates of a point of sale
func (s *EstablishmentService) UpdatePOSLocation(ctx context.Context, companyID, posID string, req *models.UpdatePOSLocationRequest) error {
	// Verify POS exists and belongs to company
	_, err := s.GetPointOfSale(ctx, companyID, posID)
	if err != nil {
		return err
	}

	query := `
		UPDATE point_of_sale
		SET latitude = $1, longitude = $2, updated_at = $3
		WHERE id = $4
	`

	result, err := database.DB.ExecContext(ctx, query, req.Latitude, req.Longitude, time.Now(), posID)
	if err != nil {
		return fmt.Errorf("failed to update POS location: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return models.ErrPointOfSaleNotFound
	}

	return nil
}

// DeactivatePointOfSale deactivates a point of sale
func (s *EstablishmentService) DeactivatePointOfSale(ctx context.Context, companyID, posID string) error {
	// Verify POS exists and belongs to company
	_, err := s.GetPointOfSale(ctx, companyID, posID)
	if err != nil {
		return err
	}

	query := `
		UPDATE point_of_sale
		SET active = false, updated_at = $1
		WHERE id = $2
	`

	result, err := database.DB.ExecContext(ctx, query, time.Now(), posID)
	if err != nil {
		return fmt.Errorf("failed to deactivate point of sale: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return models.ErrPointOfSaleNotFound
	}

	return nil
}

// Helper function to join strings
func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}
