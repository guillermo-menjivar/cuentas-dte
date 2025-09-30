package codigos

import "strings"

// GoodsTitle represents the title under which goods are remitted
type GoodsTitle struct {
	Code  string
	Value string
}

// Goods title codes
const (
	GoodsTitleDeposito     = "01"
	GoodsTitlePropiedad    = "02"
	GoodsTitleConsignacion = "03"
	GoodsTitleTraslado     = "04"
	GoodsTitleOtros        = "05"
)

// GoodsTitles is a map of all goods titles
var GoodsTitles = map[string]string{
	GoodsTitleDeposito:     "Depósito",
	GoodsTitlePropiedad:    "Propiedad",
	GoodsTitleConsignacion: "Consignación",
	GoodsTitleTraslado:     "Traslado",
	GoodsTitleOtros:        "Otros",
}

// GetGoodsTitleName returns the name of a goods title by code
func GetGoodsTitleName(code string) (string, bool) {
	name, exists := GoodsTitles[code]
	return name, exists
}

// GetGoodsTitleCode returns the code for a goods title by name (case-insensitive)
func GetGoodsTitleCode(name string) (string, bool) {
	nameLower := strings.ToLower(strings.TrimSpace(name))

	for code, value := range GoodsTitles {
		if strings.ToLower(value) == nameLower {
			return code, true
		}
	}
	return "", false
}

// GetAllGoodsTitles returns a slice of all goods titles
func GetAllGoodsTitles() []GoodsTitle {
	titles := make([]GoodsTitle, 0, len(GoodsTitles))
	for code, value := range GoodsTitles {
		titles = append(titles, GoodsTitle{
			Code:  code,
			Value: value,
		})
	}
	return titles
}

// IsValidGoodsTitle checks if a goods title code is valid
func IsValidGoodsTitle(code string) bool {
	_, exists := GoodsTitles[code]
	return exists
}
