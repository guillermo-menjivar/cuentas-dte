package formats

import (
	"bytes"
	"cuentas/internal/services"
	"encoding/csv"
	"fmt"
)

// WriteDTEReconciliationCSV generates a CSV report for DTE reconciliation results
func WriteDTEReconciliationCSV(
	results []services.DTEReconciliationRecord,
	summary *services.DTEReconciliationSummary,
) ([]byte, error) {
	buf := new(bytes.Buffer)
	writer := csv.NewWriter(buf)

	// Write summary section (if provided)
	if summary != nil {
		summaryHeader := [][]string{
			{"RESUMEN DE RECONCILIACIÓN DE DTEs"},
			{},
			{"Total de Registros", fmt.Sprintf("%d", summary.TotalRecords)},
			{"Registros Coincidentes", fmt.Sprintf("%d", summary.MatchedRecords)},
			{"Registros con Discrepancias", fmt.Sprintf("%d", summary.MismatchedRecords)},
			{"Discrepancias de Fecha", fmt.Sprintf("%d", summary.DateMismatches)},
			{"No Encontrados en Hacienda", fmt.Sprintf("%d", summary.NotFoundInHacienda)},
			{"Errores de Consulta", fmt.Sprintf("%d", summary.QueryErrors)},
			{},
		}

		for _, row := range summaryHeader {
			if err := writer.Write(row); err != nil {
				return nil, err
			}
		}
	}

	// Write column headers
	headers := []string{
		"Código Generación",
		"No. Control",
		"No. Factura",
		"Tipo DTE",
		"Fecha Emisión",
		"Monto Total",
		"Estado Interno",
		"Estado Hacienda",
		"Sello Interno",
		"Sello Hacienda",
		"Fecha Proc. Interno",
		"Fecha Proc. Hacienda",
		"Coincide",
		"Fecha Emisión Coincide",
		"Estado Consulta",
		"Discrepancias",
		"Mensaje Error",
		"Consultado En",
	}

	if err := writer.Write(headers); err != nil {
		return nil, err
	}

	// Write data rows
	for _, record := range results {
		// Format internal estado
		estadoInterno := ""
		if record.InternalEstado != nil {
			estadoInterno = *record.InternalEstado
		}

		// Format internal sello
		selloInterno := ""
		if record.InternalSello != nil {
			selloInterno = *record.InternalSello
		}

		// Format internal fecha procesamiento
		fechaProcInterno := ""
		if record.InternalFhProcesamiento != nil {
			fechaProcInterno = record.InternalFhProcesamiento.Format("02/01/2006 15:04:05")
		}

		// Format matches
		coincide := "NO"
		if record.Matches {
			coincide = "SÍ"
		}

		// Format fecha emisión matches
		fechaEmisionCoincide := "NO"
		if record.FechaEmisionMatches {
			fechaEmisionCoincide = "SÍ"
		}

		// Format discrepancies (join with semicolon)
		discrepancias := ""
		if len(record.Discrepancies) > 0 {
			for i, d := range record.Discrepancies {
				if i > 0 {
					discrepancias += "; "
				}
				discrepancias += d
			}
		}

		row := []string{
			record.CodigoGeneracion,
			record.NumeroControl,
			record.InvoiceNumber,
			record.TipoDTE,
			record.FechaEmision,
			fmt.Sprintf("%.2f", record.TotalAmount),
			estadoInterno,
			record.HaciendaEstado,
			selloInterno,
			record.HaciendaSello,
			fechaProcInterno,
			record.HaciendaFhProcesamiento,
			coincide,
			fechaEmisionCoincide,
			record.HaciendaQueryStatus,
			discrepancias,
			record.ErrorMessage,
			record.QueriedAt,
		}

		if err := writer.Write(row); err != nil {
			return nil, err
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
