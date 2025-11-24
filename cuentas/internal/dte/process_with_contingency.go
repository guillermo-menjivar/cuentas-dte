package dte

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"cuentas/internal/hacienda"
	"cuentas/internal/models"

	"github.com/google/uuid"
)

// ProcessInvoiceWithContingency - Enhanced ProcessInvoice with automatic contingency fallback
// This implements the retry logic and automatic queuing for failed DTEs
func (s *DTEService) ProcessInvoiceWithContingency(
	ctx context.Context,
	invoice *models.Invoice,
) (*hacienda.ReceptionResponse, error) {

	log.Printf("[ProcessInvoice] Starting process for invoice: %s", invoice.InvoiceNumber)

	// ===========================
	// STEP 1: BUILD DTE (LOCAL)
	// ===========================
	log.Println("[ProcessInvoice] Step 1: Building DTE...")
	dteJSON, err := s.builder.BuildInvoice(ctx, invoice)
	if err != nil {
		// This is a programming error, not external - don't queue
		return nil, fmt.Errorf("failed to build DTE: %w", err)
	}

	// Parse for codigo generacion and ambiente
	var invoiceDTE struct {
		Identificacion struct {
			CodigoGeneracion string `json:"codigoGeneracion"`
			Ambiente         string `json:"ambiente"`
			TipoDte          string `json:"tipoDte"`
		} `json:"identificacion"`
	}
	json.Unmarshal(dteJSON, &invoiceDTE)
	codigoGeneracion := strings.ToUpper(invoiceDTE.Identificacion.CodigoGeneracion)
	ambiente := invoiceDTE.Identificacion.Ambiente
	tipoDte := invoiceDTE.Identificacion.TipoDte

	log.Printf("[ProcessInvoice] ✅ DTE built: %s (tipo: %s)", codigoGeneracion, tipoDte)

	// ===========================
	// STEP 2: SIGN DTE (EXTERNAL)
	// ===========================
	log.Println("[ProcessInvoice] Step 2: Signing DTE...")
	companyID, _ := uuid.Parse(invoice.CompanyID)
	creds, err := s.LoadCredentials(ctx, companyID)
	if err != nil {
		return nil, fmt.Errorf("failed to load credentials: %w", err)
	}

	var dteForSigning interface{}
	json.Unmarshal(dteJSON, &dteForSigning)

	signedDTE, err := s.firmador.Sign(ctx, creds.NIT, creds.Password, dteForSigning)
	if err != nil {
		// ❌ FIRMADOR FAILED - Add to contingency queue
		log.Printf("[ProcessInvoice] ❌ Firmador failed: %v", err)

		queueErr := s.contingencyService.AddToQueue(ctx, AddToQueueParams{
			InvoiceID:        &invoice.ID,
			PurchaseID:       nil,
			TipoDte:          tipoDte,
			CodigoGeneracion: codigoGeneracion,
			Ambiente:         ambiente,
			FailureStage:     "signing",
			FailureReason:    err.Error(),
			DTEUnsigned:      dteJSON,
			DTESigned:        nil,
			CompanyID:        invoice.CompanyID,
			CreatedBy:        invoice.CreatedBy,
		})

		if queueErr != nil {
			log.Printf("[ProcessInvoice] ⚠️  Failed to add to queue: %v", queueErr)
		} else {
			log.Println("[ProcessInvoice] ✅ Added to contingency queue")
		}

		return nil, fmt.Errorf("firmador unavailable, added to contingency queue: %w", err)
	}

	log.Println("[ProcessInvoice] ✅ DTE signed successfully")

	// ===========================
	// STEP 3: AUTHENTICATE (EXTERNAL)
	// ===========================
	log.Println("[ProcessInvoice] Step 3: Authenticating with Hacienda...")
	authResponse, err := s.haciendaService.AuthenticateCompany(ctx, companyID.String())
	if err != nil {
		// ❌ HACIENDA AUTH FAILED - Add to contingency queue
		log.Printf("[ProcessInvoice] ❌ Hacienda auth failed: %v", err)

		queueErr := s.contingencyService.AddToQueue(ctx, AddToQueueParams{
			InvoiceID:        &invoice.ID,
			PurchaseID:       nil,
			TipoDte:          tipoDte,
			CodigoGeneracion: codigoGeneracion,
			Ambiente:         ambiente,
			FailureStage:     "authentication",
			FailureReason:    err.Error(),
			DTEUnsigned:      dteJSON,
			DTESigned:        &signedDTE,
			CompanyID:        invoice.CompanyID,
			CreatedBy:        invoice.CreatedBy,
		})

		if queueErr != nil {
			log.Printf("[ProcessInvoice] ⚠️  Failed to add to queue: %v", queueErr)
		} else {
			log.Println("[ProcessInvoice] ✅ Added to contingency queue")
		}

		return nil, fmt.Errorf("hacienda auth failed, added to contingency queue: %w", err)
	}

	log.Println("[ProcessInvoice] ✅ Authenticated with Hacienda")

	// ===========================
	// STEP 4: SUBMIT TO HACIENDA (EXTERNAL)
	// ===========================
	log.Println("[ProcessInvoice] Step 4: Submitting to Hacienda...")
	response, request, err := s.hacienda.SubmitDTE(
		ctx,
		authResponse.Body.Token,
		ambiente,
		tipoDte,
		codigoGeneracion,
		signedDTE,
	)

	if err != nil {
		// Check if it's a rejection (permanent) or network error (retry)
		if hacErr, ok := err.(*hacienda.HaciendaError); ok && hacErr.Type == "rejection" {
			// ❌ REJECTED - Don't queue, it's a data problem
			log.Printf("[ProcessInvoice] ❌ DTE rejected by Hacienda: %v", err)

			if response != nil {
				log.Printf("[ProcessInvoice] Rejection code: %s", response.CodigoMsg)
				log.Printf("[ProcessInvoice] Rejection message: %s", response.DescripcionMsg)
			}

			// Still upload to S3 for audit
			haciendaReqJSON, _ := json.Marshal(request)
			haciendaRespJSON, _ := json.Marshal(response)
			go func() {
				s.UploadDTEToS3Async(dteJSON, "unsigned", tipoDte, invoice.CompanyID, codigoGeneracion)
				s.UploadDTEToS3Async(haciendaReqJSON, "hacienda_request", tipoDte, invoice.CompanyID, codigoGeneracion)
				s.UploadDTEToS3Async(haciendaRespJSON, "hacienda_response", tipoDte, invoice.CompanyID, codigoGeneracion)
			}()

			return response, err // Return rejection, don't queue
		}

		// ❌ NETWORK/TIMEOUT ERROR - Add to contingency queue
		log.Printf("[ProcessInvoice] ❌ Hacienda submission failed (network/timeout): %v", err)

		queueErr := s.contingencyService.AddToQueue(ctx, AddToQueueParams{
			InvoiceID:        &invoice.ID,
			PurchaseID:       nil,
			TipoDte:          tipoDte,
			CodigoGeneracion: codigoGeneracion,
			Ambiente:         ambiente,
			FailureStage:     "submission",
			FailureReason:    err.Error(),
			DTEUnsigned:      dteJSON,
			DTESigned:        &signedDTE,
			CompanyID:        invoice.CompanyID,
			CreatedBy:        invoice.CreatedBy,
		})

		if queueErr != nil {
			log.Printf("[ProcessInvoice] ⚠️  Failed to add to queue: %v", queueErr)
		} else {
			log.Println("[ProcessInvoice] ✅ Added to contingency queue")
		}

		return nil, fmt.Errorf("hacienda submission failed, added to contingency queue: %w", err)
	}

	// ===========================
	// ✅ SUCCESS!
	// ===========================
	log.Println("[ProcessInvoice] ✅ SUCCESS! DTE accepted by Hacienda")
	log.Printf("[ProcessInvoice] Estado: %s", response.Estado)
	log.Printf("[ProcessInvoice] Sello: %s", response.SelloRecibido)

	if response.Estado == "PROCESADO" {
		// Save sello to invoice immediately
		err = s.saveHaciendaResponse(ctx, invoice.ID, response)
		if err != nil {
			log.Printf("[ProcessInvoice] ⚠️  Warning: failed to save Hacienda response: %v", err)
		} else {
			log.Println("[ProcessInvoice] ✅ Hacienda response saved to invoice")
		}

		// Upload to S3 async for audit trail
		haciendaReqJSON, _ := json.Marshal(request)
		haciendaRespJSON, _ := json.Marshal(response)
		go func() {
			s.UploadDTEToS3Async(dteJSON, "unsigned", tipoDte, invoice.CompanyID, codigoGeneracion)
			s.UploadDTEToS3Async(haciendaReqJSON, "hacienda_request", tipoDte, invoice.CompanyID, codigoGeneracion)
			s.UploadDTEToS3Async(haciendaRespJSON, "hacienda_response", tipoDte, invoice.CompanyID, codigoGeneracion)
		}()

		// Log to commit log
		s.logToCommitLog(ctx, invoice, response)
	}

	return response, nil
}

// ProcessPurchaseWithContingency - Same logic for purchases/expense documents
func (s *DTEService) ProcessPurchaseWithContingency(
	ctx context.Context,
	purchase *models.Purchase,
	tipoDte string, // "03", "05", "06", etc.
) (*hacienda.ReceptionResponse, error) {

	log.Printf("[ProcessPurchase] Starting process for purchase: %s (tipo: %s)", purchase.ID, tipoDte)

	// Build, sign, authenticate, submit with same contingency fallback logic
	// Similar to ProcessInvoiceWithContingency but for purchases
	// Implementation details similar to above...

	return nil, fmt.Errorf("not yet implemented")
}

// Helper functions (assumed to exist in DTEService)
// - saveHaciendaResponse
// - logToCommitLog
// - UploadDTEToS3Async
// - LoadCredentials
