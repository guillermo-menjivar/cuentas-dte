// internal/dte/constants.go
package dte

import (
	"cuentas/internal/codigos"
	"errors"
	"fmt"
)

// ============================================
// IVA (TAX) CONSTANTS
// ============================================

const (
	EstablishmentPrefixMatriz   = "M" // Casa Matriz
	EstablishmentPrefixSucursal = "S" // Sucursal
	EstablishmentPrefixBodega   = "B" // Bodega
	EstablishmentPrefixPatio    = "P" // Patio
	PointOfSalePrefixPOS        = "P" // Punto de Venta
)

// FormatEstablishmentCode formats establishment code with proper prefix
func FormatEstablishmentCode(tipoEstablecimiento string, number int) string {
	var prefix string
	switch tipoEstablecimiento {
	case codigos.EstablishmentCasaMatriz: // "02"
		prefix = EstablishmentPrefixMatriz // "M"
	case codigos.EstablishmentSucursal: // "01"
		prefix = EstablishmentPrefixSucursal // "S"
	case codigos.EstablishmentBodega: // "04"
		prefix = EstablishmentPrefixBodega // "B"
	case codigos.EstablishmentPatio: // "07"
		prefix = EstablishmentPrefixPatio // "P"
	default:
		prefix = EstablishmentPrefixMatriz // Default to Matriz
	}
	return fmt.Sprintf("%s%03d", prefix, number)
}

// FormatPOSCode formats point of sale code
// Example: FormatPOSCode(1) -> "P001"
func FormatPOSCode(number int) string {
	return fmt.Sprintf("%s%03d", PointOfSalePrefixPOS, number)
}

const (
	// IVARate is the tax rate in El Salvador (13%)
	IVARate = 0.13

	// IVADivisor is used to extract IVA from total price
	// Formula: base = total / 1.13
	IVADivisor = 1.13
)

// ============================================
// DOCUMENT TYPES (tipoDte)
// ============================================

const (
	TipoDteFactura                  = "01" // Factura
	TipoDteComprobanteCreditoFiscal = "03" // Comprobante de Crédito Fiscal
	TipoDteNotaRemision             = "04" // Nota de Remisión
	TipoDteNotaCredito              = "05" // Nota de Crédito
	TipoDteNotaDebito               = "06" // Nota de Débito
	TipoDteCompRetencion            = "07" // Comprobante de Retención
	TipoDteCompLiquidacion          = "08" // Comprobante de Liquidación
	TipoDteDocContingencia          = "09" // Documento Contable de Liquidación
	TipoDteFacturaExportacion       = "11" // Factura de Exportación
	TipoDteFacturaSujetoExcluido    = "14" // Factura Sujeto Excluido
	TipoDteCompDonacion             = "15" // Comprobante de Donación
)

// ============================================
// RECEPTOR DOCUMENT TYPES
// ============================================

const (
	DocTypeNIT      = "36" // NIT (Número de Identificación Tributaria)
	DocTypeDUI      = "13" // DUI (Documento Único de Identidad)
	DocTypeNIE      = "02" // NIE (Número de Identidad de Extranjero)
	DocTypePassport = "03" // Pasaporte
	DocTypeOther    = "37" // Otro
)

// ============================================
// AMBIENTES (ENVIRONMENTS)
// ============================================

const (
	AmbienteTest       = "00" // Pruebas
	AmbienteProduction = "01" // Producción
)

// ============================================
// TIPO ESTABLECIMIENTO
// ============================================

const (
	TipoEstablecimientoCasaMatriz   = "01" // Casa Matriz
	TipoEstablecimientoSucursal     = "02" // Sucursal
	TipoEstablecimientoOficinaAdmin = "04" // Oficina Administrativa
	TipoEstablecimientoAgencia      = "07" // Agencia
	TipoEstablecimientoOtros        = "20" // Otros
)

// ============================================
// TIPO ITEM
// ============================================

const (
	TipoItemBien     = 1 // Bien
	TipoItemServicio = 2 // Servicio
	TipoItemAmbos    = 3 // Bien y Servicio
	TipoItemTributo  = 4 // Tributo
)

// ============================================
// CONDICION OPERACION
// ============================================

const (
	CondicionOperacionContado = 1 // Contado
	CondicionOperacionCredito = 2 // Crédito
	CondicionOperacionOtro    = 3 // Otro
)

// ============================================
// PAYMENT METHOD CODES
// ============================================

const (
	PaymentMethodBilletes       = "01" // Billetes y Monedas
	PaymentMethodTarjetaDebito  = "02" // Tarjeta Débito
	PaymentMethodTarjetaCredito = "03" // Tarjeta Crédito
	PaymentMethodCheque         = "04" // Cheque
	PaymentMethodTransferencia  = "05" // Transferencia
	PaymentMethodGiroTelegrado  = "06" // Giro Telegráfico
	PaymentMethodDeposito       = "07" // Depósito en Cuenta
	PaymentMethodMonedero       = "08" // Monedero Electrónico
	PaymentMethodBilletera      = "09" // Billetera Electrónica
	PaymentMethodCriptoDivisa   = "10" // Cripto Divisas
	PaymentMethodCompensacion   = "11" // Compensación
	PaymentMethodPermuta        = "12" // Permuta
	PaymentMethodDacion         = "13" // Dación en Pago
	PaymentMethodCompraVenta    = "14" // Compra Venta
	PaymentMethodOtros          = "99" // Otros
)

// ============================================
// UNIT OF MEASURE CODES (Common ones)
// ============================================

const (
	UniMedidaUnidad    = 59 // Unidad
	UniMedidaDocena    = 11 // Docena
	UniMedidaCaja      = 58 // Caja
	UniMedidaLitro     = 20 // Litro
	UniMedidaKilogramo = 14 // Kilogramo
	UniMedidaMetro     = 40 // Metro
	UniMedidaServicio  = 99 // Servicio/Otros
)

// ============================================
// ERROR DEFINITIONS
// ============================================

var (
	ErrNegativePrecio       = errors.New("precio unitario cannot be negative")
	ErrNegativeVentaGravada = errors.New("venta gravada cannot be negative")
	ErrNegativeIVA          = errors.New("IVA cannot be negative")
	ErrInvalidInvoiceType   = errors.New("invalid invoice type")
)
