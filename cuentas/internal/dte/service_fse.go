package dte

import (
	"context"
	"cuentas/internal/hacienda"
	"cuentas/internal/models"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/google/uuid"
)

// ============================================
// PROCESS FSE (TYPE 14)
// ============================================

// ProcessFSE builds, signs, and submits an FSE purchase to Hacienda
func (s *DTEService) ProcessFSE(ctx context.Context, purchase *models.Purchase) (*hacienda.ReceptionResponse, error) {
	log.Printf("[ProcessFSE] Starting process for purchase ID: %s", purchase.ID)

	// Step 1: Build FSE DTE from purchase
	log.Println("[ProcessFSE] Step 1: Building FSE DTE from purchase...")
	fseJSON, err := s.builder.BuildFSE(ctx, purchase)
	if err != nil {
		return nil, fmt.Errorf("failed to build FSE: %w", err)
	}

	// Parse to struct for access to fields
	var fse FSE
	if err := json.Unmarshal(fseJSON, &fse); err != nil {
		return nil, fmt.Errorf("failed to unmarshal FSE: %w", err)
	}

	// Pretty print the FSE for debugging
	fsePretty, _ := json.MarshalIndent(fse, "", "  ")
	log.Println("[ProcessFSE] FSE DTE Generated:")
	log.Println(string(fsePretty))

	// Step 2: Load company credentials and sign
	log.Println("[ProcessFSE] Step 2: Loading credentials and signing DTE...")
	companyID, err := uuid.Parse(purchase.CompanyID)
	if err != nil {
		return nil, fmt.Errorf("invalid company ID: %w", err)
	}

	creds, err := s.LoadCredentials(ctx, companyID)
	if err != nil {
		return nil, fmt.Errorf("failed to load credentials: %w", err)
	}

	// Ensure codigo generacion is uppercase
	fse.Identificacion.CodigoGeneracion = strings.ToUpper(fse.Identificacion.CodigoGeneracion)

	signedDTE, err := s.firmador.Sign(ctx, creds.NIT, creds.Password, fse)
	if err != nil {
		return nil, fmt.Errorf("failed to sign FSE: %w", err)
	}

	// Step 3: Print signed DTE info
	log.Println("[ProcessFSE] Step 3: FSE Signed Successfully!")
	log.Printf("[ProcessFSE] Signed DTE length: %d characters\n", len(signedDTE))
	if len(signedDTE) > 500 {
		log.Printf("[ProcessFSE] Signed DTE (first 500 chars): %s...\n", signedDTE[:500])
	}

	// Step 4: Authenticate with Hacienda
	log.Println("[ProcessFSE] Step 4: Authenticating with Hacienda...")
	authResponse, err := s.haciendaService.AuthenticateCompany(ctx, companyID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to authenticate with Hacienda: %w", err)
	}

	log.Printf("[ProcessFSE] ✅ Authenticated! Token: %s...\n", authResponse.Body.Token[:50])

	// Step 5: Submit to Hacienda
	log.Println("[ProcessFSE] Step 5: Submitting FSE to Ministerio de Hacienda...")

	response, err := s.hacienda.SubmitDTE(
		ctx,
		authResponse.Body.Token,
		fse.Identificacion.Ambiente,
		fse.Identificacion.TipoDte, // "14"
		strings.ToUpper(fse.Identificacion.CodigoGeneracion),
		signedDTE,
	)

	if err != nil {
		// Check if it's a rejection error (we still got a response)
		if hacErr, ok := err.(*hacienda.HaciendaError); ok && hacErr.Type == "rejection" {
			log.Printf("[ProcessFSE] ❌ FSE REJECTED by Hacienda!\n")
			if response != nil {
				log.Printf("[ProcessFSE] Code: %s\n", response.CodigoMsg)
				log.Printf("[ProcessFSE] Message: %s\n", response.DescripcionMsg)
				if len(response.Observaciones) > 0 {
					log.Println("[ProcessFSE] Observations:")
					for _, obs := range response.Observaciones {
						log.Printf("[ProcessFSE]   - %s\n", obs)
					}
				}
			}
			return response, err
		}
		return nil, fmt.Errorf("failed to submit to Hacienda: %w", err)
	}

	// Check if response is nil
	if response == nil {
		return nil, fmt.Errorf("no response received from Hacienda")
	}

	// Step 6: Success! 🎉
	log.Println("[ProcessFSE] ✅ SUCCESS! FSE ACCEPTED BY HACIENDA!")
	log.Printf("[ProcessFSE] Estado: %s\n", response.Estado)
	log.Printf("[ProcessFSE] Código de Generación: %s\n", response.CodigoGeneracion)
	log.Printf("[ProcessFSE] Sello Recibido: %s\n", response.SelloRecibido)
	log.Printf("[ProcessFSE] Fecha Procesamiento: %s\n", response.FhProcesamiento)
	if response.DescripcionMsg != "" {
		log.Printf("[ProcessFSE] Message: %s\n", response.DescripcionMsg)
	}

	// Step 7: Save Hacienda response to purchase
	if response.Estado == "PROCESADO" {
		err = s.saveFSEHaciendaResponse(ctx, purchase.ID, response)
		if err != nil {
			// Log error but don't fail - DTE was accepted
			log.Printf("[ProcessFSE] ⚠️  Warning: failed to save Hacienda response: %v\n", err)
		} else {
			log.Println("[ProcessFSE] ✅ Hacienda response saved to purchase")
		}
	}

	// Step 8: Log to commit log
	err = s.logFSEToCommitLog(ctx, purchase, &fse, signedDTE, response)
	if err != nil {
		// Log error but don't fail - DTE was already accepted
		log.Printf("[ProcessFSE] ⚠️  Warning: failed to log to commit log: %v\n", err)
	} else {
		log.Println("[ProcessFSE] ✅ FSE submission logged to commit log")
	}

	return response, nil
}

// ============================================
// SAVE HACIENDA RESPONSE
// ============================================

// saveFSEHaciendaResponse updates the purchase with Hacienda's response
func (s *DTEService) saveFSEHaciendaResponse(ctx context.Context, purchaseID string, response *hacienda.ReceptionResponse) error {
	// Marshal response to JSON
	responseJSON, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("failed to marshal response: %w", err)
	}

	query := `
        UPDATE purchases
        SET dte_status = $1,
            dte_hacienda_response = $2,
            dte_sello_recibido = $3,
            dte_submitted_at = NOW()
        WHERE id = $4
    `

	_, err = s.db.ExecContext(ctx, query,
		response.Estado,
		string(responseJSON),
		response.SelloRecibido,
		purchaseID,
	)

	if err != nil {
		return fmt.Errorf("failed to update purchase: %w", err)
	}

	return nil
}

// ============================================
// COMMIT LOG
// ============================================

// logFSEToCommitLog logs the FSE submission to the commit log
func (s *DTEService) logFSEToCommitLog(
	ctx context.Context,
	purchase *models.Purchase,
	fse *FSE,
	signedDTE string,
	response *hacienda.ReceptionResponse,
) error {
	log.Printf("[logFSEToCommitLog] Logging FSE to commit log for purchase: %s", purchase.ID)

	// Marshal FSE to JSON
	fseJSON, err := json.Marshal(fse)
	if err != nil {
		return fmt.Errorf("failed to marshal FSE: %w", err)
	}

	// Marshal response
	responseJSON, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("failed to marshal response: %w", err)
	}

	query := `
        INSERT INTO dte_commit_log (
            codigo_generacion,
            purchase_id,
            company_id,
            tipo_dte,
            numero_control,
            ambiente,
            version,
            tipo_modelo,
            tipo_operacion,
            fecha_emision,
            hora_emision,
            dte_json,
            dte_firmado,
            estado_hacienda,
            sello_recibido,
            fecha_procesamiento,
            hacienda_response,
            observaciones,
            created_at
        ) VALUES (
            $1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
            $11, $12, $13, $14, $15, $16, $17, $18, NOW()
        )
    `

	// Extract observaciones if present
	var observaciones *string
	if len(response.Observaciones) > 0 {
		obs := strings.Join(response.Observaciones, "; ")
		observaciones = &obs
	}

	_, err = s.db.ExecContext(ctx, query,
		fse.Identificacion.CodigoGeneracion,
		purchase.ID,
		purchase.CompanyID,
		fse.Identificacion.TipoDte,
		fse.Identificacion.NumeroControl,
		fse.Identificacion.Ambiente,
		fse.Identificacion.Version,
		fse.Identificacion.TipoModelo,
		fse.Identificacion.TipoOperacion,
		fse.Identificacion.FecEmi,
		fse.Identificacion.HorEmi,
		string(fseJSON),
		signedDTE,
		response.Estado,
		response.SelloRecibido,
		response.FhProcesamiento,
		string(responseJSON),
		observaciones,
	)

	if err != nil {
		return fmt.Errorf("failed to insert commit log: %w", err)
	}

	log.Printf("[logFSEToCommitLog] ✅ Successfully logged FSE to commit log")
	return nil
}
