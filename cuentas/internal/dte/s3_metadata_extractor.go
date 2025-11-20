package dte

import (
	"encoding/json"
	"fmt"
	"time"
)

// DTEMetadataExtractor extracts metadata from different DTE types
type DTEMetadataExtractor struct{}

// NewDTEMetadataExtractor creates a new metadata extractor
func NewDTEMetadataExtractor() *DTEMetadataExtractor {
	return &DTEMetadataExtractor{}
}

// ExtractFromJSON extracts metadata from a DTE JSON for S3 upload
func (e *DTEMetadataExtractor) ExtractFromJSON(dteJSON []byte) (*DTEUploadRequest, error) {
	// Parse JSON to extract identificacion
	var parsed map[string]interface{}
	if err := json.Unmarshal(dteJSON, &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse DTE JSON: %w", err)
	}

	// Extract identificacion
	identificacion, ok := parsed["identificacion"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("identificacion field not found or invalid")
	}

	req := &DTEUploadRequest{
		UnsignedJSON: dteJSON,
	}

	// Extract required fields from identificacion
	if codigoGen, ok := identificacion["codigoGeneracion"].(string); ok {
		req.GenerationCode = codigoGen
	} else {
		return nil, fmt.Errorf("codigoGeneracion not found")
	}

	if numeroControl, ok := identificacion["numeroControl"].(string); ok {
		req.ControlNumber = numeroControl
	} else {
		return nil, fmt.Errorf("numeroControl not found")
	}

	if tipoDte, ok := identificacion["tipoDte"].(string); ok {
		req.DocumentType = tipoDte
	} else {
		return nil, fmt.Errorf("tipoDte not found")
	}

	// Parse fecha emision
	if fecEmi, ok := identificacion["fecEmi"].(string); ok {
		issueDate, err := time.Parse("2006-01-02", fecEmi)
		if err != nil {
			return nil, fmt.Errorf("failed to parse fecEmi: %w", err)
		}
		req.IssueDate = issueDate
	} else {
		return nil, fmt.Errorf("fecEmi not found")
	}

	// Extract emisor info
	if emisor, ok := parsed["emisor"].(map[string]interface{}); ok {
		if nit, ok := emisor["nit"].(string); ok {
			req.SenderNIT = &nit
		}
		if nombre, ok := emisor["nombre"].(string); ok {
			req.SenderName = &nombre
		}
	}

	// Extract receptor info (handle both Receptor and SujetoExcluido)
	if receptor, ok := parsed["receptor"].(map[string]interface{}); ok {
		if nombre, ok := receptor["nombre"].(string); ok {
			req.ReceiverName = &nombre
		}
		if nit, ok := receptor["nit"].(string); ok {
			req.ReceiverNIT = &nit
		}
	} else if sujetoExcluido, ok := parsed["sujetoExcluido"].(map[string]interface{}); ok {
		// FSE uses sujetoExcluido instead of receptor
		if nombre, ok := sujetoExcluido["nombre"].(string); ok {
			req.ReceiverName = &nombre
		}
	}

	// Extract resumen (totals)
	if resumen, ok := parsed["resumen"].(map[string]interface{}); ok {
		if totalPagar, ok := resumen["totalPagar"].(float64); ok {
			req.TotalAmount = totalPagar
		}
		if totalGravada, ok := resumen["totalGravada"].(float64); ok {
			req.TaxableAmount = &totalGravada
		}
		// Handle both totalIva (Factura) and tributos (CCF)
		if totalIva, ok := resumen["totalIva"].(float64); ok {
			req.TaxAmount = &totalIva
		} else if tributos, ok := resumen["tributos"].([]interface{}); ok && len(tributos) > 0 {
			// Extract IVA from tributos array
			if trib, ok := tributos[0].(map[string]interface{}); ok {
				if valor, ok := trib["valor"].(float64); ok {
					req.TaxAmount = &valor
				}
			}
		}
	}

	return req, nil
}

// ExtractFromDTE extracts metadata from a DTE struct (Type 01/03)
func (e *DTEMetadataExtractor) ExtractFromDTE(dte *DTE) (*DTEUploadRequest, error) {
	dteJSON, err := json.Marshal(dte)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal DTE: %w", err)
	}

	req, err := e.ExtractFromJSON(dteJSON)
	if err != nil {
		return nil, err
	}

	// Additional extraction from struct
	req.Status = "GENERATED" // Default status before Hacienda submission

	return req, nil
}

// ExtractFromFSE extracts metadata from an FSE struct (Type 14)
func (e *DTEMetadataExtractor) ExtractFromFSE(fse *FSE) (*DTEUploadRequest, error) {
	fseJSON, err := json.Marshal(fse)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal FSE: %w", err)
	}

	req, err := e.ExtractFromJSON(fseJSON)
	if err != nil {
		return nil, err
	}

	req.Status = "GENERATED"

	return req, nil
}
