// internal/dte/service_fse.go
package dte

import (
	"context"
	"cuentas/internal/hacienda"
	"cuentas/internal/models"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
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

	// === CONTINGENCY: Handle signing failure ===
	signedDTE, err := s.firmador.Sign(ctx, creds.NIT, creds.Password, fse)
	if err != nil {
		log.Printf("[ProcessFSE] ⚠️  Firmador failed: %v", err)

		if s.contingencyHelperPurchase != nil {
			if queueErr := s.contingencyHelperPurchase.HandleSigningFailure(
				ctx,
				purchase,
				fse,
				fse.Identificacion.Ambiente,
			); queueErr != nil {
				return nil, fmt.Errorf("firmador failed and contingency queue failed: %w", queueErr)
			}
			log.Println("[ProcessFSE] 📋 Purchase queued for contingency (firmador unavailable)")
			return nil, fmt.Errorf("purchase queued for contingency: firmador unavailable")
		}

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
		log.Printf("[ProcessFSE] ⚠️  Hacienda auth failed: %v", err)

		// === CONTINGENCY: Handle auth failure ===
		if s.contingencyHelperPurchase != nil {
			if queueErr := s.contingencyHelperPurchase.HandleAuthFailure(
				ctx,
				purchase,
				fse,
				signedDTE,
				fse.Identificacion.Ambiente,
			); queueErr != nil {
				return nil, fmt.Errorf("auth failed and contingency queue failed: %w", queueErr)
			}
			log.Println("[ProcessFSE] 📋 Purchase queued for contingency (Hacienda auth unavailable)")
			return nil, fmt.Errorf("purchase queued for contingency: Hacienda auth unavailable")
		}

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
			// Rejections are permanent - don't queue for contingency
			return response, err
		}

		// === CONTINGENCY: Handle submission failure (timeout, network) ===
		log.Printf("[ProcessFSE] ⚠️  Hacienda submission failed: %v", err)
		if s.contingencyHelperPurchase != nil {
			if queueErr := s.contingencyHelperPurchase.HandleSubmissionFailure(
				ctx,
				purchase,
				fse,
				signedDTE,
				fse.Identificacion.Ambiente,
			); queueErr != nil {
				return nil, fmt.Errorf("submission failed and contingency queue failed: %w", queueErr)
			}
			log.Println("[ProcessFSE] 📋 Purchase queued for contingency (Hacienda unavailable)")
			return nil, fmt.Errorf("purchase queued for contingency: Hacienda unavailable")
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
		UploadDTEToS3Async(fseJSON, "unsigned", "14", purchase.CompanyID, strings.ToUpper(fse.Identificacion.CodigoGeneracion))
		UploadDTEToS3Async([]byte(signedDTE), "signed", "14", purchase.CompanyID, strings.ToUpper(fse.Identificacion.CodigoGeneracion))
		haciendaResponseJSON, _ := json.MarshalIndent(response, "", "  ")
		UploadDTEToS3Async(haciendaResponseJSON, "hacienda_response", "14", purchase.CompanyID, strings.ToUpper(fse.Identificacion.CodigoGeneracion))

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

	// Extract observaciones array if present
	var observaciones interface{}
	if len(response.Observaciones) > 0 {
		observaciones = pq.Array(response.Observaciones)
	} else {
		observaciones = pq.Array([]string{}) // Empty array
	}

	// Parse fecha_emision from string "YYYY-MM-DD"
	fechaEmision, err := time.Parse("2006-01-02", fse.Identificacion.FecEmi)
	if err != nil {
		return fmt.Errorf("failed to parse fecha_emision: %w", err)
	}

	// Parse Hacienda's FhProcesamiento timestamp
	// Format: "17/11/2025 19:21:25" → needs to be converted to time.Time
	var fhProcesamiento *time.Time
	if response.FhProcesamiento != "" {
		// Parse Hacienda's format: "DD/MM/YYYY HH:MM:SS"
		t, err := time.Parse("02/01/2006 15:04:05", response.FhProcesamiento)
		if err != nil {
			log.Printf("[logFSEToCommitLog] Warning: failed to parse FhProcesamiento '%s': %v", response.FhProcesamiento, err)
		} else {
			fhProcesamiento = &t
		}
	}

	query := `
        INSERT INTO dte_commit_log (
            codigo_generacion,
            purchase_id,
            invoice_id,
            invoice_number,
            company_id,
            client_id,
            establishment_id,
            point_of_sale_id,
            subtotal,
            total_discount,
            total_taxes,
            iva_amount,
            total_amount,
            currency,
            payment_method,
            payment_terms,
            numero_control,
            tipo_dte,
            ambiente,
            fecha_emision,
            fiscal_year,
            fiscal_month,
            dte_url,
            dte_unsigned,
            dte_signed,
            hacienda_estado,
            hacienda_sello_recibido,
            hacienda_fh_procesamiento,
            hacienda_codigo_msg,
            hacienda_descripcion_msg,
            hacienda_observaciones,
            hacienda_response_full,
            submitted_at
        ) VALUES (
            $1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
            $11, $12, $13, $14, $15, $16, $17, $18, $19, $20,
            $21, $22, $23, $24, $25, $26, $27, $28, $29, $30,
            $31, $32, NOW()
        )
    `

	_, err = s.db.ExecContext(ctx, query,
		fse.Identificacion.CodigoGeneracion, // $1 codigo_generacion
		purchase.ID,                         // $2 purchase_id (NOT NULL)
		nil,                                 // $3 invoice_id (NULL for purchases)
		nil,                                 // $4 invoice_number (NULL for purchases)
		purchase.CompanyID,                  // $5
		nil,                                 // $6 client_id (NULL for FSE - informal supplier)
		purchase.EstablishmentID,            // $7
		purchase.PointOfSaleID,              // $8
		purchase.Subtotal,                   // $9
		purchase.TotalDiscount,              // $10
		0.0,                                 // $11 total_taxes (FSE has no IVA)
		0.0,                                 // $12 iva_amount (FSE has no IVA)
		purchase.Total,                      // $13
		purchase.Currency,                   // $14
		*purchase.PaymentMethod,             // $15
		func() string { // $16 payment_terms
			if purchase.PaymentCondition != nil && *purchase.PaymentCondition == 1 {
				return "cash"
			}
			return "credit"
		}(),
		fse.Identificacion.NumeroControl, // $17
		fse.Identificacion.TipoDte,       // $18
		fse.Identificacion.Ambiente,      // $19
		fechaEmision,                     // $20
		fechaEmision.Year(),              // $21 fiscal_year
		int(fechaEmision.Month()),        // $22 fiscal_month
		"",                               // $23 dte_url (can be added later)
		string(fseJSON),                  // $24 dte_unsigned
		signedDTE,                        // $25 dte_signed
		response.Estado,                  // $26 hacienda_estado
		response.SelloRecibido,           // $27 hacienda_sello_recibido
		fhProcesamiento,                  // $28 hacienda_fh_procesamiento (parsed timestamp)
		response.CodigoMsg,               // $29 hacienda_codigo_msg
		response.DescripcionMsg,          // $30 hacienda_descripcion_msg
		observaciones,                    // $31 hacienda_observaciones (array)
		string(responseJSON),             // $32 hacienda_response_full
	)

	if err != nil {
		return fmt.Errorf("failed to insert commit log: %w", err)
	}

	log.Printf("[logFSEToCommitLog] ✅ Successfully logged FSE to commit log")
	return nil
}
