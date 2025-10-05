package codigos

import "strings"

// EconomicActivity represents an economic activity code
type EconomicActivity struct {
	Code  string
	Value string
}

// Economic activity codes - Agriculture, Livestock, Forestry and Fishing
const (
	// AGRICULTURA, GANADERÍA, SILVICULTURA Y PESCA
	// PRODUCCIÓN AGRÍCOLA, PECUARIA, CAZA Y ACTIVIDADES DE SERVICIOS CONEXAS
	ActEcon01111 = "01111"
	ActEcon01112 = "01112"
	ActEcon01113 = "01113"
	ActEcon01114 = "01114"
	ActEcon01119 = "01119"
	ActEcon01120 = "01120"
	ActEcon01131 = "01131"
	ActEcon01132 = "01132"
	ActEcon01133 = "01133"
	ActEcon01134 = "01134"
	ActEcon01140 = "01140"
	ActEcon01150 = "01150"
	ActEcon01161 = "01161"
	ActEcon01162 = "01162"
	ActEcon01191 = "01191"
	ActEcon01192 = "01192"
	ActEcon01199 = "01199"
	ActEcon01220 = "01220"
	ActEcon01230 = "01230"
	ActEcon01240 = "01240"
	ActEcon01251 = "01251"
	ActEcon01252 = "01252"
	ActEcon01260 = "01260"
	ActEcon01271 = "01271"
	ActEcon01272 = "01272"
	ActEcon01281 = "01281"
	ActEcon01282 = "01282"
	ActEcon01291 = "01291"
	ActEcon01292 = "01292"
	ActEcon01299 = "01299"
	ActEcon01300 = "01300"
	ActEcon01301 = "01301"
)

// EconomicActivities is a map of all economic activity codes
var EconomicActivities = map[string]string{
	ActEcon01111: "Cultivo de cereales excepto arroz y para forrajes",
	ActEcon01112: "Cultivo de legumbres",
	ActEcon01113: "Cultivo de semillas oleaginosas",
	ActEcon01114: "Cultivo de plantas para la preparación de semillas",
	ActEcon01119: "Cultivo de otros cereales excepto arroz y forrajeros n.c.p.",
	ActEcon01120: "Cultivo de arroz",
	ActEcon01131: "Cultivo de raíces y tubérculos",
	ActEcon01132: "Cultivo de brotes, bulbos, vegetales tubérculos y cultivos similares",
	ActEcon01133: "Cultivo hortícola de fruto",
	ActEcon01134: "Cultivo de hortalizas de hoja y otras hortalizas ncp",
	ActEcon01140: "Cultivo de caña de azúcar",
	ActEcon01150: "Cultivo de tabaco",
	ActEcon01161: "Cultivo de algodón",
	ActEcon01162: "Cultivo de fibras vegetales excepto algodón",
	ActEcon01191: "Cultivo de plantas no perennes para la producción de semillas y flores",
	ActEcon01192: "Cultivo de cereales y pastos para la alimentación animal",
	ActEcon01199: "Producción de cultivos no estacionales ncp",
	ActEcon01220: "Cultivo de frutas tropicales",
	ActEcon01230: "Cultivo de cítricos",
	ActEcon01240: "Cultivo de frutas de pepita y hueso",
	ActEcon01251: "Cultivo de frutas ncp",
	ActEcon01252: "Cultivo de otros frutos y nueces de árboles y arbustos",
	ActEcon01260: "Cultivo de frutos oleaginosos",
	ActEcon01271: "Cultivo de café",
	ActEcon01272: "Cultivo de plantas para la elaboración de bebidas excepto café",
	ActEcon01281: "Cultivo de especias y aromáticas",
	ActEcon01282: "Cultivo de plantas para la obtención de productos medicinales y farmacéuticos",
	ActEcon01291: "Cultivo de árboles de hule (caucho) para la obtención de látex",
	ActEcon01292: "Cultivo de plantas para la obtención de productos químicos y colorantes",
	ActEcon01299: "Producción de cultivos perennes ncp",
	ActEcon01300: "Propagación de plantas",
	ActEcon01301: "Cultivo de plantas y flores ornamentales",
}

// GetEconomicActivityName returns the name of an economic activity by code
func GetEconomicActivityName(code string) (string, bool) {
	name, exists := EconomicActivities[code]
	return name, exists
}

// GetEconomicActivityCode returns the code for an economic activity by name (case-insensitive)
func GetEconomicActivityCode(name string) (string, bool) {
	nameLower := strings.ToLower(strings.TrimSpace(name))
	for code, value := range EconomicActivities {
		if strings.ToLower(value) == nameLower {
			return code, true
		}
	}
	return "", false
}

// GetAllEconomicActivities returns a slice of all economic activities
func GetAllEconomicActivities() []EconomicActivity {
	activities := make([]EconomicActivity, 0, len(EconomicActivities))
	for code, value := range EconomicActivities {
		activities = append(activities, EconomicActivity{
			Code:  code,
			Value: value,
		})
	}
	return activities
}

// IsValidEconomicActivity checks if an economic activity code is valid
func IsValidEconomicActivity(code string) bool {
	_, exists := EconomicActivities[code]
	return exists
}
