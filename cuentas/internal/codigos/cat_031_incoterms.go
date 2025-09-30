package codigos

import "strings"

// Incoterm represents an international commercial term
type Incoterm struct {
	Code  string
	Value string
}

// Incoterm codes
const (
	IncotermEXW = "01"
	IncotermFCA = "02"
	IncotermCPT = "03"
	IncotermCIP = "04"
	IncotermDAP = "05"
	IncotermDPU = "06"
	IncotermDDP = "07"
	IncotermFAS = "08"
	IncotermFOB = "09"
	IncotermCFR = "10"
	IncotermCIF = "11"
)

// Incoterms is a map of all incoterms
var Incoterms = map[string]string{
	IncotermEXW: "EXW-En fabrica",
	IncotermFCA: "FCA-Libre transportista",
	IncotermCPT: "CPT-Transporte pagado hasta",
	IncotermCIP: "CIP-Transporte y seguro pagado hasta",
	IncotermDAP: "DAP-Entrega en el lugar",
	IncotermDPU: "DPU-Entregado en el lugar descargado",
	IncotermDDP: "DDP-Entrega con impuestos pagados",
	IncotermFAS: "FAS-Libre al costado del buque",
	IncotermFOB: "FOB-Libre a bordo",
	IncotermCFR: "CFR-Costo y flete",
	IncotermCIF: "CIF- Costo seguro y flete",
}

// GetIncotermName returns the name of an incoterm by code
func GetIncotermName(code string) (string, bool) {
	name, exists := Incoterms[code]
	return name, exists
}

// GetIncotermCode returns the code for an incoterm by name (case-insensitive)
func GetIncotermCode(name string) (string, bool) {
	nameLower := strings.ToLower(strings.TrimSpace(name))

	for code, value := range Incoterms {
		if strings.ToLower(value) == nameLower {
			return code, true
		}
	}
	return "", false
}

// GetAllIncoterms returns a slice of all incoterms
func GetAllIncoterms() []Incoterm {
	terms := make([]Incoterm, 0, len(Incoterms))
	for code, value := range Incoterms {
		terms = append(terms, Incoterm{
			Code:  code,
			Value: value,
		})
	}
	return terms
}

// IsValidIncoterm checks if an incoterm code is valid
func IsValidIncoterm(code string) bool {
	_, exists := Incoterms[code]
	return exists
}
