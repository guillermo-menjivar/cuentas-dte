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

// ProcessInvoiceWithContingency - Enhanced ProcessInvoice with contingency support
func (s *DTEService) ProcessInvoiceWithContingency(ctx context.Context, invoice *models.Invoice) (*hacienda.ReceptionResponse, error) {
	log.Printf("[ProcessInvoice] Starting process for invoice: %s", invoice.InvoiceNumber)

	// Step 1: Build DTE (local - should always work)
	log.Println("[ProcessInvoice] Step 1: Building DTE...")
	dteJSON, err := s.builder.BuildInvoice(ctx, invoice)
	if err != nil {
		// This is a programming error, not external - don't queue
		return nil, fmt.Errorf("failed to build DTE: %w", err)
	}

	// Parse for codigo generacion
	var invoiceDTE struct {
		Identificacion struct {
			CodigoGeneracion string `json:"codigoGeneracion"`
			Ambiente         string `json:"ambiente"`
		} `json:"identificacion"`
	}
	json.Unmarshal(dteJSON, &invoiceDTE)
	codigoGeneracion := strings.ToUpper(invoiceDTE.Identificacion.CodigoGeneracion)
	ambiente := invoiceDTE.Identificacion.Ambiente

	log.Printf("[ProcessInvoice] ✅ DTE built: %s", codigoGeneracion)

	// Step 2: Sign DTE (external service - can fail)
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
			TipoDte:          "01", // Adjust based on invoice type
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

	// Step 3: Authenticate with Hacienda (external service - can fail)
	log.Println("[ProcessInvoice] Step 3: Authenticating with Hacienda...")
	authResponse, err := s.haciendaService.AuthenticateCompany(ctx, companyID.String())
	if err != nil {
		// ❌ HACIENDA AUTH FAILED - Add to contingency queue
		log.Printf("[ProcessInvoice] ❌ Hacienda auth failed: %v", err)

		queueErr := s.contingencyService.AddToQueue(ctx, AddToQueueParams{
			InvoiceID:        &invoice.ID,
			TipoDte:          "01",
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

	// Step 4: Submit to Hacienda (external service - can fail)
	log.Println("[ProcessInvoice] Step 4: Submitting to Hacienda...")
	response, request, err := s.hacienda.SubmitDTE(
		ctx,
		authResponse.Body.Token,
		ambiente,
		"01",
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
				UploadDTEToS3Async(dteJSON, "unsigned", "01", invoice.CompanyID, codigoGeneracion)
				UploadDTEToS3Async(haciendaReqJSON, "hacienda_request", "01", invoice.CompanyID, codigoGeneracion)
				UploadDTEToS3Async(haciendaRespJSON, "hacienda_response", "01", invoice.CompanyID, codigoGeneracion)
			}()

			return response, err // Return rejection, don't queue
		}

		// ❌ NETWORK/TIMEOUT ERROR - Add to contingency queue
		log.Printf("[ProcessInvoice] ❌ Hacienda submission failed: %v", err)

		queueErr := s.contingencyService.AddToQueue(ctx, AddToQueueParams{
			InvoiceID:        &invoice.ID,
			TipoDte:          "01",
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

	// ✅ SUCCESS!
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

		// Upload to S3 async
		haciendaReqJSON, _ := json.Marshal(request)
		haciendaRespJSON, _ := json.Marshal(response)
		go func() {
			UploadDTEToS3Async(dteJSON, "unsigned", "01", invoice.CompanyID, codigoGeneracion)
			UploadDTEToS3Async(haciendaReqJSON, "hacienda_request", "01", invoice.CompanyID, codigoGeneracion)
			UploadDTEToS3Async(haciendaRespJSON, "hacienda_response", "01", invoice.CompanyID, codigoGeneracion)
		}()

		// Log to commit log
		s.logToCommitLog(ctx, invoice, response)
	}

	return response, nil
}
