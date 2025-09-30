package codigos

import "strings"

// Department represents a department of El Salvador
type Department struct {
	Code  string
	Value string
}

// Department codes
const (
	DepartmentOtro         = "00"
	DepartmentAhuachapan   = "01"
	DepartmentSantaAna     = "02"
	DepartmentSonsonate    = "03"
	DepartmentChalatenango = "04"
	DepartmentLaLibertad   = "05"
	DepartmentSanSalvador  = "06"
	DepartmentCuscatlan    = "07"
	DepartmentLaPaz        = "08"
	DepartmentCabanas      = "09"
	DepartmentSanVicente   = "10"
	DepartmentUsulutan     = "11"
	DepartmentSanMiguel    = "12"
	DepartmentMorazan      = "13"
	DepartmentLaUnion      = "14"
)

// Departments is a map of all departments
var Departments = map[string]string{
	DepartmentOtro:         "Otro (Para extranjeros)",
	DepartmentAhuachapan:   "Ahuachapán",
	DepartmentSantaAna:     "Santa Ana",
	DepartmentSonsonate:    "Sonsonate",
	DepartmentChalatenango: "Chalatenango",
	DepartmentLaLibertad:   "La Libertad",
	DepartmentSanSalvador:  "San Salvador",
	DepartmentCuscatlan:    "Cuscatlán",
	DepartmentLaPaz:        "La Paz",
	DepartmentCabanas:      "Cabañas",
	DepartmentSanVicente:   "San Vicente",
	DepartmentUsulutan:     "Usulután",
	DepartmentSanMiguel:    "San Miguel",
	DepartmentMorazan:      "Morazán",
	DepartmentLaUnion:      "La Unión",
}

// GetDepartmentName returns the name of a department by code
func GetDepartmentName(code string) (string, bool) {
	name, exists := Departments[code]
	return name, exists
}

// GetDepartmentCode returns the code for a department by name (case-insensitive)
func GetDepartmentCode(name string) (string, bool) {
	nameLower := strings.ToLower(strings.TrimSpace(name))

	for code, value := range Departments {
		if strings.ToLower(value) == nameLower {
			return code, true
		}
	}
	return "", false
}

// GetAllDepartments returns a slice of all departments
func GetAllDepartments() []Department {
	departments := make([]Department, 0, len(Departments))
	for code, value := range Departments {
		departments = append(departments, Department{
			Code:  code,
			Value: value,
		})
	}
	return departments
}

// IsValidDepartment checks if a department code is valid
func IsValidDepartment(code string) bool {
	_, exists := Departments[code]
	return exists
}
