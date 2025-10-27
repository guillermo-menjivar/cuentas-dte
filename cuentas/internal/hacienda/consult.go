package hacienda

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/hashicorp/go-retryablehttp"
)

// DTEReconciliationRecord represents a single DTE reconciliation result
type DTEReconciliationRecord struct {
	// Internal record (from our database)
	CodigoGeneracion        string     `json:"codigo_generacion"`
	InvoiceID               string     `json:"invoice_id"`
	InvoiceNumber           string     `json:"invoice_number"`
	ClientID                string     `json:"client_id"`
	NumeroControl           string     `json:"numero_control"`
	TipoDTE                 string     `json:"tipo_dte"`
	FechaEmision            string     `json:"fecha_emision"` // Internal emission date
	TotalAmount             float64    `json:"total_amount"`
	InternalEstado          *string    `json:"internal_estado"`
	InternalSello           *string    `json:"internal_sello"`
	InternalFhProcesamiento *time.Time `json:"internal_fh_procesamiento"`

	// Hacienda record (from API query)
	HaciendaEstado          string   `json:"hacienda_estado,omitempty"`
	HaciendaSello           string   `json:"hacienda_sello,omitempty"`
	HaciendaFhProcesamiento string   `json:"hacienda_fh_procesamiento,omitempty"`
	HaciendaCodigoMsg       string   `json:"hacienda_codigo_msg,omitempty"`
	HaciendaDescripcionMsg  string   `json:"hacienda_descripcion_msg,omitempty"`
	HaciendaObservaciones   []string `json:"hacienda_observaciones,omitempty"`

	// Reconciliation result
	Matches             bool     `json:"matches"`
	FechaEmisionMatches bool     `json:"fecha_emision_matches"` // NEW: Date comparison
	Discrepancies       []string `json:"discrepancies,omitempty"`
	HaciendaQueryStatus string   `json:"hacienda_query_status"` // "success", "not_found", "error"
	ErrorMessage        string   `json:"error_message,omitempty"`
	QueriedAt           string   `json:"queried_at"`
}

type ConsultaDTEResponse struct {
	Version          int      `json:"version"`
	Ambiente         string   `json:"ambiente"`
	VersionApp       int      `json:"versionApp"`
	Estado           string   `json:"estado"` // "PROCESADO", "RECHAZADO", "RECIBIDO"
	CodigoGeneracion string   `json:"codigoGeneracion"`
	SelloRecibido    string   `json:"selloRecibido,omitempty"`
	FhProcesamiento  string   `json:"fhProcesamiento"`        // Format: "dd/MM/yyyy HH:mm:ss"
	FechaEmision     string   `json:"fechaEmision,omitempty"` // NEW: Format: "dd/MM/yyyy"
	ClasificaMsg     string   `json:"clasificaMsg,omitempty"`
	CodigoMsg        string   `json:"codigoMsg,omitempty"`
	DescripcionMsg   string   `json:"descripcionMsg,omitempty"`
	Observaciones    []string `json:"observaciones,omitempty"`
}

// GetConsultaURL returns the configured URL for DTE consultation
func (c *Client) GetConsultaURL() string {
	return c.consultaURL
}

// ConsultarDTE queries Hacienda for a specific DTE's status
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - authToken: The authentication token from HaciendaService
//   - nitEmisor: Company NIT WITHOUT dashes (e.g., "06142305911306")
//   - tipoDte: "01" for Factura, "03" for CCF, etc.
//   - codigoGeneracion: UUID from the DTE
//
// Returns the consultation response or an error
func (c *Client) ConsultarDTE(
	ctx context.Context,
	authToken string,
	nitEmisor string,
	tipoDte string,
	codigoGeneracion string,
) (*ConsultaDTEResponse, error) {

	// Prepare request payload
	reqPayload := ConsultaDTERequest{
		NitEmisor:        nitEmisor,
		TipoDTE:          tipoDte,
		CodigoGeneracion: codigoGeneracion,
	}

	// Marshal request
	reqBody, err := json.Marshal(reqPayload)
	if err != nil {
		return nil, &HaciendaError{
			Type:      "validation",
			Code:      "MARSHAL_ERROR",
			Message:   fmt.Sprintf("failed to marshal consultation request: %v", err),
			Timestamp: time.Now(),
		}
	}

	// Use configured consultation URL
	fmt.Println("\n🔍 CONSULTA DTE DEBUG:")
	fmt.Printf("URL: %s\n", c.consultaURL)
	fmt.Printf("Request Body: %s\n", string(reqBody))
	fmt.Printf("NIT Emisor: %s\n", nitEmisor)
	fmt.Printf("Tipo DTE: %s\n", tipoDte)
	fmt.Printf("Código Generación: %s\n", codigoGeneracion)
	fmt.Println()

	// Create HTTP request
	req, err := retryablehttp.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.consultaURL,
		bytes.NewBuffer(reqBody),
	)
	if err != nil {
		return nil, &HaciendaError{
			Type:      "network",
			Code:      "REQUEST_CREATE_ERROR",
			Message:   fmt.Sprintf("failed to create consultation request: %v", err),
			Timestamp: time.Now(),
		}
	}

	// Set headers (same as SubmitDTE)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "CuentasApp/1.0")
	req.Header.Set("Authorization", authToken)

	// Execute request with retries
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, &HaciendaError{
			Type:      "network",
			Code:      "CONNECTION_ERROR",
			Message:   fmt.Sprintf("failed to connect to Hacienda consultation service: %v", err),
			Timestamp: time.Now(),
		}
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, &HaciendaError{
			Type:      "network",
			Code:      "RESPONSE_READ_ERROR",
			Message:   fmt.Sprintf("failed to read consultation response: %v", err),
			Timestamp: time.Now(),
		}
	}

	fmt.Printf("📥 CONSULTA RESPONSE (Status %d): %s\n\n", resp.StatusCode, string(respBody))

	// Check HTTP status code
	if resp.StatusCode == http.StatusNotFound {
		return nil, &HaciendaError{
			Type:      "not_found",
			Code:      "DTE_NOT_FOUND",
			Message:   fmt.Sprintf("DTE not found in Hacienda: %s", codigoGeneracion),
			Details:   string(respBody),
			Timestamp: time.Now(),
		}
	}

	if resp.StatusCode != http.StatusOK {
		return nil, &HaciendaError{
			Type:      "server",
			Code:      fmt.Sprintf("HTTP_%d", resp.StatusCode),
			Message:   fmt.Sprintf("Hacienda consultation returned status %d: %s", resp.StatusCode, string(respBody)),
			Details:   string(respBody),
			Timestamp: time.Now(),
		}
	}

	// Parse JSON response
	var consultaResp ConsultaDTEResponse
	if err := json.Unmarshal(respBody, &consultaResp); err != nil {
		return nil, &HaciendaError{
			Type:      "validation",
			Code:      "RESPONSE_PARSE_ERROR",
			Message:   fmt.Sprintf("failed to parse consultation response: %v", err),
			Details:   string(respBody),
			Timestamp: time.Now(),
		}
	}

	return &consultaResp, nil
}
