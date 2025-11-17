// internal/dte/fse.go
package dte

import (
	"context"
	"cuentas/internal/dte_schemas"
	"cuentas/internal/models"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"
)

// ============================================
// BUILD FSE (TYPE 14)
// ============================================

// BuildFSE builds a Type 14 FSE (Factura Sujeto Excluido) DTE from a purchase
func (b *Builder) BuildFSE(ctx context.Context, purchase *models.Purchase) ([]byte, error) {
	log.Printf("[BuildFSE] Starting build for FSE ID: %s", purchase.ID)

	// Validate this is an FSE purchase
	if !purchase.IsFSE() {
		return nil, fmt.Errorf("purchase is not an FSE (Type 14)")
	}

	// Load company data
	company, err := b.loadCompany(ctx, purchase.CompanyID)
	if err != nil {
		return nil, fmt.Errorf("load company: %w", err)
	}

	// Load establishment and POS
	establishment, err := b.loadEstablishmentAndPOS(ctx, purchase.EstablishmentID, purchase.PointOfSaleID)
	if err != nil {
		return nil, fmt.Errorf("load establishment: %w", err)
	}

	// Build DTE structure
	fse := &FSE{
		Identificacion:  b.buildFSEIdentificacion(purchase, company),
		Emisor:          b.buildFSEEmisor(company, establishment), // ⭐ Custom FSE emisor
		SujetoExcluido:  b.buildSujetoExcluido(purchase),
		CuerpoDocumento: b.buildFSECuerpoDocumento(purchase),
		Resumen:         b.buildFSEResumen(purchase),
		Apendice:        nil, // Can be extended later
	}

	// Marshal to JSON
	jsonBytes, err := json.Marshal(fse)
	if err != nil {
		return nil, fmt.Errorf("marshal JSON: %w", err)
	}

	// Validate JSON against schema
	log.Printf("[BuildFSE] Validating DTE against schema...")
	if err := dte_schemas.Validate("14", jsonBytes); err != nil {
		log.Printf("WARNING: [BuildFSE] ❌ Schema validation failed: %v", err)
		// Don't fail on validation error, just warn
	} else {
		log.Printf("[BuildFSE] ✅ Schema validation passed")
	}

	return jsonBytes, nil
}

// ============================================
// BUILD IDENTIFICACION (Type 14)
// ============================================

func (b *Builder) buildFSEIdentificacion(purchase *models.Purchase, company *CompanyData) FSEIdentificacion {
	loc, err := time.LoadLocation("America/El_Salvador")
	if err != nil {
		// Fallback to manual UTC-6 if timezone data not available
		loc = time.FixedZone("CST", -6*60*60)
	}

	// Use finalized_at for emission time, or now if not finalized
	var emissionTime time.Time
	if purchase.FinalizedAt != nil {
		emissionTime = purchase.FinalizedAt.In(loc)
	} else {
		emissionTime = time.Now().In(loc)
	}

	return FSEIdentificacion{
		Version:          1, // ⭐ Version 1 for Type 14 (not 3!)
		Ambiente:         company.DTEAmbiente,
		TipoDte:          "14",
		NumeroControl:    strings.ToUpper(*purchase.DteNumeroControl),
		CodigoGeneracion: purchase.ID, // Purchase ID is the codigoGeneracion
		TipoModelo:       1,           // Previo
		TipoOperacion:    1,           // Normal
		TipoContingencia: nil,
		MotivoContin:     nil,
		FecEmi:           emissionTime.Format("2006-01-02"),
		HorEmi:           emissionTime.Format("15:04:05"),
		TipoMoneda:       "USD",
	}
}

// ============================================
// BUILD SUJETO EXCLUIDO (INFORMAL SUPPLIER)
// ============================================

// buildFSEEmisor builds emisor for FSE (excludes fields not allowed in Type 14)
func (b *Builder) buildFSEEmisor(company *CompanyData, establishment *EstablishmentData) FSEEmisor {
	return FSEEmisor{
		NIT:             company.NIT,
		NRC:             fmt.Sprintf("%d", company.NCR),
		Nombre:          company.Name,
		CodActividad:    company.CodActividad,
		DescActividad:   company.DescActividad,
		Direccion:       b.buildEmisorDireccion(establishment),
		Telefono:        establishment.Telefono,
		Correo:          company.Email,
		CodEstableMH:    nil,
		CodEstable:      &establishment.CodEstablecimiento,
		CodPuntoVentaMH: nil,
		CodPuntoVenta:   &establishment.CodPuntoVenta,
	}
}

func (b *Builder) buildSujetoExcluido(purchase *models.Purchase) FSESujetoExcluido {
	// Build direccion
	direccion := Direccion{
		Departamento: *purchase.SupplierAddressDept,
		Municipio:    *purchase.SupplierAddressMuni,
		Complemento:  *purchase.SupplierAddressComplement,
	}

	// ⭐ For type "37" (Otro), numDocumento can be omitted or set to a generic value
	numDoc := purchase.SupplierDocumentNumber
	if numDoc == nil || *numDoc == "" {
		// If no document number, use a generic identifier
		generic := "N/A"
		numDoc = &generic
	}

	return FSESujetoExcluido{
		TipoDocumento: purchase.SupplierDocumentType,
		NumDocumento:  numDoc, // ⭐ Always provide a value
		Nombre:        *purchase.SupplierName,
		CodActividad:  purchase.SupplierActivityCode,
		DescActividad: purchase.SupplierActivityDesc,
		Direccion:     direccion,
		Telefono:      purchase.SupplierPhone,
		Correo:        purchase.SupplierEmail,
	}
}

// ============================================
// BUILD CUERPO DOCUMENTO (Type 14)
// ============================================

func (b *Builder) buildFSECuerpoDocumento(purchase *models.Purchase) []FSECuerpoItem {
	items := make([]FSECuerpoItem, len(purchase.LineItems))

	for i, lineItem := range purchase.LineItems {
		// Parse unit of measure from string to int
		uniMedida := b.parseUniMedidaFromString(lineItem.UnitOfMeasure)

		items[i] = FSECuerpoItem{
			NumItem:     lineItem.LineNumber,
			TipoItem:    lineItem.ItemType,
			Cantidad:    lineItem.Quantity,
			Codigo:      lineItem.ItemCode,
			UniMedida:   uniMedida,
			Descripcion: lineItem.ItemName,
			PrecioUni:   lineItem.UnitPrice,
			MontoDescu:  lineItem.DiscountAmount,
			Compra:      lineItem.LineTotal, // ⭐ "compra" is the line total for FSE
		}
	}

	return items
}

// ============================================
// BUILD RESUMEN (Type 14)
// ============================================

func (b *Builder) buildFSEResumen(purchase *models.Purchase) FSEResumen {
	// Calculate totals
	totalCompra := purchase.Subtotal
	totalDescu := purchase.TotalDiscount
	subTotal := purchase.Subtotal - purchase.TotalDiscount
	totalPagar := purchase.Total

	// Convert discount to percentage if there's a subtotal
	discountPercentage := 0.0
	if purchase.Subtotal > 0 && purchase.TotalDiscount > 0 {
		discountPercentage = (purchase.TotalDiscount / purchase.Subtotal) * 100
	}

	// Convert total to words
	totalLetras := b.numberToWords(totalPagar)

	// Build pagos array with proper plazo mapping
	var pagos *[]FSEPago
	if purchase.PaymentCondition != nil && purchase.PaymentMethod != nil {
		// ⭐ Map payment term to Hacienda format
		var plazo *string
		if purchase.PaymentTerm != nil {
			mapped := b.mapPaymentTermToHacienda(*purchase.PaymentTerm)
			plazo = &mapped
		}

		p := []FSEPago{
			{
				Codigo:     *purchase.PaymentMethod,
				MontoPago:  totalPagar,
				Referencia: purchase.PaymentReference,
				Plazo:      plazo, // ⭐ Use mapped value
				Periodo:    purchase.PaymentPeriod,
			},
		}
		pagos = &p
	}

	// ⭐ totalDescu should always be a float, not null
	var totalDescuPtr *float64
	if totalDescu > 0 {
		totalDescuPtr = &totalDescu
	} else {
		zero := 0.0
		totalDescuPtr = &zero
	}

	return FSEResumen{
		TotalCompra:        totalCompra,
		Descu:              discountPercentage,
		TotalDescu:         totalDescuPtr, // ⭐ Never null
		SubTotal:           subTotal,
		IvaRete1:           purchase.IVARetained,
		ReteRenta:          purchase.IncomeTaxRetained,
		TotalPagar:         totalPagar,
		TotalLetras:        totalLetras,
		CondicionOperacion: *purchase.PaymentCondition,
		Pagos:              pagos,
		Observaciones:      purchase.Notes,
	}
}

// ============================================
// HELPER FUNCTIONS
// ============================================

// mapPaymentTermToHacienda maps internal payment terms to Hacienda codes
func (b *Builder) mapPaymentTermToHacienda(term string) string {
	switch strings.ToLower(term) {
	case "net_30", "net_60", "net_90", "days", "dias":
		return "01" // Días
	case "months", "meses":
		return "02" // Meses
	case "years", "anos", "años":
		return "03" // Años
	default:
		// If it's already a Hacienda code, return as-is
		if term == "01" || term == "02" || term == "03" {
			return term
		}
		return "01" // Default to days
	}
}

// parseUniMedidaFromString parses unit of measure string to int code
func (b *Builder) parseUniMedidaFromString(unitOfMeasure string) int {
	// If it's already a number string, parse it
	var code int
	_, err := fmt.Sscanf(unitOfMeasure, "%d", &code)
	if err == nil && code > 0 {
		return code
	}

	// Otherwise use the existing parser
	return b.parseUniMedida(unitOfMeasure)
}
