package dte

import (
	"errors"
	"fmt"
	"regexp"
)

var (
	// Regex patterns
	nitPattern                  = regexp.MustCompile(`^([0-9]{14}|[0-9]{9})$`)
	codigoGeneracionPattern     = regexp.MustCompile(`^[A-F0-9]{8}-[A-F0-9]{4}-[A-F0-9]{4}-[A-F0-9]{4}-[A-F0-9]{12}$`)
	numeroControlFacturaPattern = regexp.MustCompile(`^DTE-01-[A-Z0-9]{8}-[0-9]{15}$`)
	numeroControlCCFPattern     = regexp.MustCompile(`^DTE-03-[A-Z0-9]{8}-[0-9]{15}$`)
)

// ValidateFactura validates a Factura Electrónica
func ValidateFactura(f *FacturaElectronica) error {
	// Validate Identificacion
	if f.Identificacion.Version != VersionFactura {
		return fmt.Errorf("invalid version: expected %d, got %d", VersionFactura, f.Identificacion.Version)
	}

	if f.Identificacion.TipoDte != TipoDteFactura {
		return fmt.Errorf("invalid tipoDte: expected %s, got %s", TipoDteFactura, f.Identificacion.TipoDte)
	}

	if !numeroControlFacturaPattern.MatchString(f.Identificacion.NumeroControl) {
		return errors.New("invalid numeroControl format for Factura")
	}

	if !codigoGeneracionPattern.MatchString(f.Identificacion.CodigoGeneracion) {
		return errors.New("invalid codigoGeneracion format")
	}

	// Validate Emisor
	if !nitPattern.MatchString(f.Emisor.NIT) {
		return errors.New("invalid emisor NIT format")
	}

	// Add more validation rules as needed...

	return nil
}

// ValidateCreditoFiscal validates a Crédito Fiscal
func ValidateCreditoFiscal(c *CreditoFiscal) error {
	// Validate Identificacion
	if c.Identificacion.Version != VersionCreditoFiscal {
		return fmt.Errorf("invalid version: expected %d, got %d", VersionCreditoFiscal, c.Identificacion.Version)
	}

	if c.Identificacion.TipoDte != TipoDteCreditoFiscal {
		return fmt.Errorf("invalid tipoDte: expected %s, got %s", TipoDteCreditoFiscal, c.Identificacion.TipoDte)
	}

	if !numeroControlCCFPattern.MatchString(c.Identificacion.NumeroControl) {
		return errors.New("invalid numeroControl format for Crédito Fiscal")
	}

	if !codigoGeneracionPattern.MatchString(c.Identificacion.CodigoGeneracion) {
		return errors.New("invalid codigoGeneracion format")
	}

	// Validate Emisor
	if !nitPattern.MatchString(c.Emisor.NIT) {
		return errors.New("invalid emisor NIT format")
	}

	// Validate Receptor (required for CCF)
	if !nitPattern.MatchString(c.Receptor.NIT) {
		return errors.New("invalid receptor NIT format")
	}

	// Add more validation rules as needed...

	return nil
}
