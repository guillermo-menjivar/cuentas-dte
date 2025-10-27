package i18n

// Language represents supported languages
type Language string

const (
	Spanish Language = "es"
	English Language = "en"
)

// Translations holds all translatable strings
type Translations struct {
	lang Language
}

// New creates a new Translations instance with the given language
func New(lang string) *Translations {
	if lang == "en" {
		return &Translations{lang: English}
	}
	// Default to Spanish
	return &Translations{lang: Spanish}
}

// EventType translates inventory event types
func (t *Translations) EventType(eventType string) string {
	if t.lang == English {
		return eventType // Already in English
	}

	// Spanish translations
	translations := map[string]string{
		"PURCHASE":   "COMPRA",
		"SALE":       "VENTA",
		"ADJUSTMENT": "AJUSTE",
	}

	if translated, ok := translations[eventType]; ok {
		return translated
	}
	return eventType
}

// ItemType translates item types
func (t *Translations) ItemType(tipoItem string) string {
	if t.lang == English {
		if tipoItem == "1" {
			return "Product"
		}
		return "Service"
	}

	// Spanish translations
	if tipoItem == "1" {
		return "Producto"
	}
	return "Servicio"
}

// InventoryEventsHeaders returns CSV headers for inventory events
func (t *Translations) InventoryEventsHeaders() []string {
	if t.lang == English {
		return []string{
			"Event ID", "Timestamp", "SKU", "Item Name", "Event Type",
			"Quantity", "Unit Cost", "Total Cost",
			"Avg Cost Before", "Avg Cost After",
			"Balance Qty", "Balance Value", "Notes",
		}
	}

	// Spanish headers
	return []string{
		"ID de Evento", "Fecha y Hora", "SKU", "Nombre del Artículo", "Tipo de Evento",
		"Cantidad", "Costo Unitario", "Costo Total",
		"Costo Promedio Antes", "Costo Promedio Después",
		"Cantidad en Existencia", "Valor en Existencia", "Notas",
	}
}

// InventoryStatesHeaders returns CSV headers for inventory states
func (t *Translations) InventoryStatesHeaders() []string {
	if t.lang == English {
		return []string{
			"SKU", "Item Name", "Type", "Quantity", "Avg Cost", "Total Value", "Last Updated",
		}
	}

	// Spanish headers
	return []string{
		"SKU", "Nombre del Artículo", "Tipo", "Cantidad", "Costo Promedio", "Valor Total", "Última Actualización",
	}
}

// ValuationSummaryHeaders returns headers for valuation summary section
func (t *Translations) ValuationSummaryHeaders() []string {
	if t.lang == English {
		return []string{"INVENTORY VALUATION SUMMARY"}
	}
	return []string{"RESUMEN DE VALUACIÓN DE INVENTARIO"}
}

// ValuationSummaryRow returns a translated row for valuation summary
func (t *Translations) ValuationSummaryRow(key string) string {
	if t.lang == English {
		return key
	}

	// Spanish translations
	translations := map[string]string{
		"As of Date":     "Fecha de Corte",
		"Total Value":    "Valor Total",
		"Total Quantity": "Cantidad Total",
		"Item Count":     "Cantidad de Artículos",
	}

	if translated, ok := translations[key]; ok {
		return translated
	}
	return key
}

// ValuationDetailHeaders returns headers for valuation detail section
func (t *Translations) ValuationDetailHeaders() []string {
	if t.lang == English {
		return []string{"SKU", "Item Name", "Quantity", "Avg Cost", "Total Value", "Last Event Date"}
	}

	// Spanish headers
	return []string{"SKU", "Nombre del Artículo", "Cantidad", "Costo Promedio", "Valor Total", "Fecha Último Evento"}
}

// InventoryRegisterHeaders returns CSV headers for legal inventory register (Article 142-A)
func (t *Translations) InventoryRegisterHeaders() []string {
	if t.lang == "en" {
		return []string{
			"Correlative",
			"Date",
			"Event Type",
			"Doc Type",
			"Document No.",
			"Supplier/Customer",
			"NIT",
			"Nationality",
			"Source/Reference",
			"Units In",
			"Units Out",
			"Balance Units",
			"Cost In",
			"Cost Out",
			"Balance Cost",
			"Average Cost",
			"Remarks", // NEW
		}
	}
	// Spanish (default)
	return []string{
		"Correlativo",
		"Fecha",
		"Tipo Evento",
		"Tipo Doc",
		"No. Documento",
		"Proveedor/Cliente",
		"NIT",
		"Nacionalidad",
		"Fuente/Referencia",
		"Unidades Entrada",
		"Unidades Salida",
		"Saldo Unidades",
		"Costo Entrada",
		"Costo Salida",
		"Saldo Costo",
		"Costo Promedio",
		"Observaciones", // NEW
	}
}

// FormatRegisterHeader formats the header section for inventory register
func (t *Translations) FormatRegisterHeader() string {
	if t.lang == English {
		return "INVENTORY CONTROL REGISTER"
	}
	return "REGISTRO DE CONTROL DE INVENTARIOS"
}

// FormatCompanyLabel returns label for company name
func (t *Translations) FormatCompanyLabel() string {
	if t.lang == English {
		return "Company"
	}
	return "Empresa"
}

// FormatPeriodLabel returns label for period
func (t *Translations) FormatPeriodLabel() string {
	if t.lang == English {
		return "Period"
	}
	return "Período"
}

// FormatItemLabel returns label for item/article
func (t *Translations) FormatItemLabel() string {
	if t.lang == English {
		return "Item"
	}
	return "Artículo"
}
