package dte

import (
	"context"
	"cuentas/internal/hacienda"
	"cuentas/internal/models"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

func (s *DTEService) ProcessNotaDebito(ctx context.Context, nota *models.NotaDebito) (*hacienda.ReceptionResponse, error) {
	fmt.Printf("\n🔄 Processing Nota de Débito DTE: %s\n", nota.NotaNumber)

	// Step 1: Build the DTE JSON
	fmt.Println("📝 Building DTE JSON...")
	builder := NewBuilder(s.db)
	dteJSON, err := builder.BuildNotaDebito(ctx, nota)
	if err != nil {
		return nil, fmt.Errorf("failed to build DTE: %w", err)
	}

	fmt.Printf("✅ DTE JSON built (%d bytes)\n", len(dteJSON))

	// Step 2: Sign the DTE
	fmt.Println("🔏 Signing DTE...")
	signedDTE, err := s.signDTE(ctx, nota.CompanyID, dteJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to sign DTE: %w", err)
	}

	fmt.Printf("✅ DTE signed (Código: %s)\n", signedDTE.CodigoGeneracion)

	// Step 3: Submit to Hacienda
	fmt.Println("📤 Submitting to Hacienda...")
	response, err := s.submitToHacienda(ctx, nota.CompanyID, signedDTE)
	if err != nil {
		return nil, fmt.Errorf("failed to submit to Hacienda: %w", err)
	}

	fmt.Printf("✅ Hacienda response: %s\n", response.Estado)

	// Step 4: Save to commit log
	fmt.Println("💾 Saving to commit log...")
	err = s.saveNotaHaciendaResponse(ctx, nota, signedDTE, response)
	if err != nil {
		fmt.Printf("⚠️  Warning: Failed to save to commit log: %v\n", err)
	}

	// Step 5: Log to audit
	err = s.logNotaToCommitLog(ctx, nota, signedDTE, response)
	if err != nil {
		fmt.Printf("⚠️  Warning: Failed to log to commit log: %v\n", err)
	}

	fmt.Printf("✅ Nota de Débito DTE processing complete\n\n")

	return response, nil
}

// saveNotaHaciendaResponse saves the Hacienda response for a nota
func (s *DTEService) saveNotaHaciendaResponse(ctx context.Context, nota *models.NotaDebito, signedDTE *firmador.SignedDTE, response *hacienda.ReceptionResponse) error {
	query := `
		UPDATE notas_debito
		SET 
			dte_codigo_generacion = $1,
			dte_sello_recibido = $2,
			dte_status = $3,
			dte_hacienda_response = $4,
			dte_submitted_at = $5
		WHERE id = $6
	`

	responseJSON, _ := json.Marshal(response)
	now := time.Now()

	_, err := s.db.ExecContext(ctx, query,
		signedDTE.CodigoGeneracion,
		response.SelloRecibido,
		response.Estado,
		responseJSON,
		now,
		nota.ID,
	)

	return err
}

// logNotaToCommitLog creates an audit log entry for the nota
func (s *DTEService) logNotaToCommitLog(ctx context.Context, nota *models.NotaDebito, signedDTE *firmador.SignedDTE, response *hacienda.ReceptionResponse) error {
	query := `
		INSERT INTO dte_commit_log (
			id, company_id, document_id, document_type,
			codigo_generacion, numero_control, sello_recibido,
			estado, observaciones, hacienda_response,
			signed_dte, submitted_at, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`

	logID := uuid.New().String()
	responseJSON, _ := json.Marshal(response)
	now := time.Now()

	observaciones := sql.NullString{}
	if len(response.Observaciones) > 0 {
		obs, _ := json.Marshal(response.Observaciones)
		observaciones = sql.NullString{String: string(obs), Valid: true}
	}

	_, err := s.db.ExecContext(ctx, query,
		logID,
		nota.CompanyID,
		nota.ID,
		"nota_debito",
		signedDTE.CodigoGeneracion,
		nota.DteNumeroControl,
		response.SelloRecibido,
		response.Estado,
		observaciones,
		responseJSON,
		signedDTE.Body,
		now,
		now,
	)

	return err
}
