package models

import (
	"fmt"
	"strings"
	"time"
)

// Retention represents a DTE 07 (Comprobante de Retención) - IVA retention certificate
type Retention struct {
	ID              string `json:"id"`
	CompanyID       string `json:"company_id"`
	PurchaseID      string `json:"purchase_id"`
	EstablishmentID string `json:"establishment_id"`
	PointOfSaleID   string `json:"point_of_sale_id"`

	// Supplier info (snapshot from purchase at time of retention)
	SupplierID   *string `json:"supplier_id,omitempty"`
	SupplierName string  `json:"supplier_name"`
	SupplierNIT  *string `json:"supplier_nit,omitempty"` // Required for formal suppliers
	SupplierNRC  *string `json:"supplier_nrc,omitempty"` // Required for formal suppliers

	// DTE identifiers
	CodigoGeneracion string `json:"codigo_generacion"`
	NumeroControl    string `json:"numero_control"`
	TipoDte          string `json:"tipo_dte"` // Always "07"
	Ambiente         string `json:"ambiente"`

	// Purchase/Invoice reference
	PurchaseNumeroControl    string    `json:"purchase_numero_control"`
	PurchaseCodigoGeneracion string    `json:"purchase_codigo_generacion"`
	PurchaseTipoDte          string    `json:"purchase_tipo_dte"`
	PurchaseFechaEmision     time.Time `json:"purchase_fecha_emision"`

	// Retention amounts
	MontoSujetoGrav float64 `json:"monto_sujeto_grav"` // Taxable amount
	IVARetenido     float64 `json:"iva_retenido"`      // Retained IVA
	RetentionRate   float64 `json:"retention_rate"`    // 1.00, 2.00, or 13.00
	RetentionCode   string  `json:"retention_code"`    // "22", "C4", "C9"

	// Dates
	FechaEmision       time.Time  `json:"fecha_emision"`
	FechaProcesamiento *time.Time `json:"fecha_procesamiento,omitempty"`

	// DTE data
	DteJSON   string `json:"dte_json"`
	DteSigned string `json:"dte_signed"`

	// Hacienda response
	HaciendaEstado          *string    `json:"hacienda_estado,omitempty"`
	HaciendaSelloRecibido   *string    `json:"hacienda_sello_recibido,omitempty"`
	HaciendaFhProcesamiento *time.Time `json:"hacienda_fh_procesamiento,omitempty"`
	HaciendaCodigoMsg       *string    `json:"hacienda_codigo_msg,omitempty"`
	HaciendaDescripcionMsg  *string    `json:"hacienda_descripcion_msg,omitempty"`
	HaciendaObservaciones   []string   `json:"hacienda_observaciones,omitempty"`
	HaciendaResponse        *string    `json:"hacienda_response,omitempty"`

	// Audit
	CreatedBy   *string    `json:"created_by,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	SubmittedAt *time.Time `json:"submitted_at,omitempty"`
}

// CreateRetentionRequest represents the request to create a retention (DTE 07)
type CreateRetentionRequest struct {
	PurchaseID string `json:"purchase_id" binding:"required"`
}

// Validate validates the create retention request
func (r *CreateRetentionRequest) Validate() error {
	if strings.TrimSpace(r.PurchaseID) == "" {
		return fmt.Errorf("purchase_id is required")
	}
	return nil
}

// RetentionValidationResult represents the result of retention validation
type RetentionValidationResult struct {
	CanCreateRetention bool     `json:"can_create_retention"`
	Reason             string   `json:"reason,omitempty"`
	Errors             []string `json:"errors,omitempty"`
}

// Helper methods

// IsProcessed checks if the retention was successfully processed by Hacienda
func (r *Retention) IsProcessed() bool {
	return r.HaciendaEstado != nil && *r.HaciendaEstado == "PROCESADO"
}

// IsRejected checks if the retention was rejected by Hacienda
func (r *Retention) IsRejected() bool {
	return r.HaciendaEstado != nil && *r.HaciendaEstado == "RECHAZADO"
}

// HasSupplierID checks if retention references a registered supplier
func (r *Retention) HasSupplierID() bool {
	return r.SupplierID != nil && *r.SupplierID != ""
}

// GetRetentionPercentage returns the retention rate as a percentage string
func (r *Retention) GetRetentionPercentage() string {
	return fmt.Sprintf("%.2f%%", r.RetentionRate)
}

// GetRetentionCodeDescription returns the description for the retention code
func (r *Retention) GetRetentionCodeDescription() string {
	switch r.RetentionCode {
	case "22":
		return "1% IVA Retention (Art 162)"
	case "C4":
		return "2% IVA Retention (Special)"
	case "C9":
		return "13% IVA Retention (Government)"
	default:
		return "Unknown"
	}
}

// CalculateAmountToPaySupplier calculates how much should be paid to supplier after retention
// Formula: Purchase Total - IVA Retenido
func (r *Retention) CalculateAmountToPaySupplier(purchaseTotal float64) float64 {
	return purchaseTotal - r.IVARetenido
}

// Constants for retention codes
const (
	RetentionCode1Percent  = "22" // 1% retention (Art 162)
	RetentionCode2Percent  = "C4" // 2% retention (special regimes)
	RetentionCode13Percent = "C9" // 13% retention (government)
)

// Constants for retention rates
const (
	RetentionRate1Percent  = 1.00
	RetentionRate2Percent  = 2.00
	RetentionRate13Percent = 13.00
)

// GetRetentionCodeForRate returns the MH code for a given retention rate
func GetRetentionCodeForRate(rate float64) string {
	switch rate {
	case RetentionRate1Percent:
		return RetentionCode1Percent
	case RetentionRate2Percent:
		return RetentionCode2Percent
	case RetentionRate13Percent:
		return RetentionCode13Percent
	default:
		return RetentionCode1Percent // Default to 1%
	}
}

// GetRetentionRateForCode returns the rate for a given MH code
func GetRetentionRateForCode(code string) float64 {
	switch code {
	case RetentionCode1Percent:
		return RetentionRate1Percent
	case RetentionCode2Percent:
		return RetentionRate2Percent
	case RetentionCode13Percent:
		return RetentionRate13Percent
	default:
		return RetentionRate1Percent // Default to 1%
	}
}

// ValidateRetentionRate validates if a retention rate is valid
func ValidateRetentionRate(rate float64) error {
	if rate != RetentionRate1Percent && rate != RetentionRate2Percent && rate != RetentionRate13Percent {
		return fmt.Errorf("invalid retention rate: must be 1.00, 2.00, or 13.00")
	}
	return nil
}

// ValidateRetentionCode validates if a retention code is valid
func ValidateRetentionCode(code string) error {
	if code != RetentionCode1Percent && code != RetentionCode2Percent && code != RetentionCode13Percent {
		return fmt.Errorf("invalid retention code: must be 22, C4, or C9")
	}
	return nil
}

// CalculateIVARetention calculates the IVA retention amount
// Formula: Monto Sujeto a Retención * (Retention Rate / 100)
func CalculateIVARetention(montoSujetoGrav float64, retentionRate float64) float64 {
	return montoSujetoGrav * (retentionRate / 100)
}
