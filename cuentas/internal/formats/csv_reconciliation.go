package formats

import (
	"bytes"
	"cuentas/internal/models"
	"encoding/csv"
	"fmt"
	"time"
)

// WriteDTEReconciliationCSV generates a CSV report for DTE reconciliation results
func WriteDTEReconciliationCSV(
	results []models.DTEReconciliationRecord,
	summary *models.DTEReconciliationSummary,
) ([]byte, error) {
	buf := new(bytes.Buffer)
	writer := csv.NewWriter(buf)

	// Load El Salvador timezone once
	elSalvadorTZ, err := time.LoadLocation("America/El_Salvador")
	if err != nil {
		// Fallback to fixed offset if timezone database not available
		elSalvadorTZ = time.FixedZone("CST", -6*60*60)
	}

	// Write summary section (if provided)
	if summary != nil {
		summaryHeader := [][]string{
			{"RESUMEN DE RECONCILIACIÓN DE DTEs"},
			{"Zona Horaria: America/El_Salvador (CST, UTC-6)"},
			{},
			{"Total de Registros", fmt.Sprintf("%d", summary.TotalRecords)},
			{"Registros Coincidentes", fmt.Sprintf("%d", summary.MatchedRecords)},
			{"Registros con Discrepancias", fmt.Sprintf("%d", summary.MismatchedRecords)},
			{"Discrepancias de Fecha", fmt.Sprintf("%d", summary.DateMismatches)},
			{"No Encontrados en Hacienda", fmt.Sprintf("%d", summary.NotFoundInHacienda)},
			{"Errores de Consulta", fmt.Sprintf("%d", summary.QueryErrors)},
			{},
			{"Nota: Todas las fechas/horas están en Hora de El Salvador (CST)"},
			{},
		}

		for _, row := range summaryHeader {
			if err := writer.Write(row); err != nil {
				return nil, err
			}
		}
	}

	// Write column headers with clear timezone labels
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
		"Fecha Proc. Interno (CST)",  // Both in CST for easy comparison
		"Fecha Proc. Hacienda (CST)", // Both in CST for easy comparison
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

		// Format internal fecha procesamiento (convert from UTC to CST)
		fechaProcInterno := ""
		if record.InternalFhProcesamiento != nil {
			// Database stores in UTC, convert to CST for display
			fechaProcInterno = record.InternalFhProcesamiento.In(elSalvadorTZ).Format("02/01/2006 15:04:05")
		}

		// Format Hacienda fecha procesamiento (convert from UTC to CST)
		fechaProcHacienda := ""
		if record.HaciendaFhProcesamiento != "" {
			// Hacienda returns UTC timestamps, parse and convert to CST
			haciendaTime, err := time.Parse("02/01/2006 15:04:05", record.HaciendaFhProcesamiento)
			if err == nil {
				// Create as UTC timestamp
				haciendaTimeUTC := time.Date(
					haciendaTime.Year(), haciendaTime.Month(), haciendaTime.Day(),
					haciendaTime.Hour(), haciendaTime.Minute(), haciendaTime.Second(),
					0, time.UTC,
				)
				// Convert to CST for display
				fechaProcHacienda = haciendaTimeUTC.In(elSalvadorTZ).Format("02/01/2006 15:04:05")
			} else {
				// If parse fails, show original
				fechaProcHacienda = record.HaciendaFhProcesamiento
			}
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
			fechaProcInterno,  // CST
			fechaProcHacienda, // CST
			coincide,
			fechaEmisionCoincide,
			record.HaciendaQueryStatus,
			discrepancias,
			record.ErrorMessage,
			record.QueriedAt, // This can stay UTC (system timestamp)
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
