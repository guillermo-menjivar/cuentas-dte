package handlers

import (
	"database/sql"
	"encoding/csv"
	"io"
	"net/http"
	"strings"

	"cuentas/internal/models"
	"cuentas/internal/services"

	"github.com/gin-gonic/gin"
)

// BulkUploadClientsHandler handles POST /v1/clients/bulk-upload
func BulkUploadClientsHandler(c *gin.Context) {
	// Get company_id from context
	companyID, exists := c.Get("company_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error: "company_id not found in context",
			Code:  "unauthorized",
		})
		return
	}

	// Get the uploaded file
	file, _, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: "failed to read uploaded file",
			Code:  "invalid_request",
		})
		return
	}
	defer file.Close()

	// Parse CSV
	reader := csv.NewReader(file)
	reader.TrimLeadingSpace = true

	// Read header row
	headers, err := reader.Read()
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: "failed to read CSV headers",
			Code:  "invalid_csv",
		})
		return
	}

	// Validate headers
	expectedHeaders := []string{
		"ncr", "nit", "dui", "business_name", "legal_business_name",
		"giro", "tipo_contribuyente", "tipo_persona", "full_address",
		"country_code", "department_code", "municipality_code",
		"cod_actividad", "correo", "telefono",
	}

	headerMap := make(map[string]int)
	for i, header := range headers {
		headerMap[strings.ToLower(strings.TrimSpace(header))] = i
	}

	// Check if all required headers exist
	for _, expected := range expectedHeaders {
		if _, exists := headerMap[expected]; !exists {
			c.JSON(http.StatusBadRequest, models.ErrorResponse{
				Error: "missing required CSV header: " + expected,
				Code:  "invalid_csv_headers",
			})
			return
		}
	}

	// Parse all rows
	var requests []models.CreateClientRequest
	var parseErrors []BulkUploadError
	rowNumber := 1 // Start at 1 (header is row 0)

	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			c.JSON(http.StatusBadRequest, models.ErrorResponse{
				Error: "failed to parse CSV row " + string(rune(rowNumber+1)),
				Code:  "invalid_csv",
			})
			return
		}

		rowNumber++

		// Parse row into CreateClientRequest
		req := models.CreateClientRequest{
			NCR:               getCSVValue(row, headerMap, "ncr"),
			NIT:               getCSVValue(row, headerMap, "nit"),
			DUI:               getCSVValue(row, headerMap, "dui"),
			BusinessName:      getCSVValue(row, headerMap, "business_name"),
			LegalBusinessName: getCSVValue(row, headerMap, "legal_business_name"),
			Giro:              getCSVValue(row, headerMap, "giro"),
			TipoContribuyente: getCSVValue(row, headerMap, "tipo_contribuyente"),
			TipoPersona:       getCSVValue(row, headerMap, "tipo_persona"),
			FullAddress:       getCSVValue(row, headerMap, "full_address"),
			CountryCode:       getCSVValue(row, headerMap, "country_code"),
			DepartmentCode:    getCSVValue(row, headerMap, "department_code"),
			MunicipalityCode:  getCSVValue(row, headerMap, "municipality_code"),
		}

		// Handle optional fields
		if codActividad := getCSVValue(row, headerMap, "cod_actividad"); codActividad != "" {
			req.CodActividad = &codActividad
		}
		if correo := getCSVValue(row, headerMap, "correo"); correo != "" {
			req.Correo = &correo
		}
		if telefono := getCSVValue(row, headerMap, "telefono"); telefono != "" {
			req.Telefono = &telefono
		}

		// Validate each request
		if err := req.Validate(); err != nil {
			parseErrors = append(parseErrors, BulkUploadError{
				Row:   rowNumber,
				Error: err.Error(),
			})
			continue
		}

		// Additional CCF validation for juridical persons (those with NIT/NCR)
		if req.TipoPersona == "2" && req.NIT != "" {
			if err := req.ValidateForCCF(); err != nil {
				parseErrors = append(parseErrors, BulkUploadError{
					Row:   rowNumber,
					Error: err.Error(),
				})
				continue
			}
		}

		requests = append(requests, req)
	}

	// If there are validation errors, return them
	if len(parseErrors) > 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":       "validation failed for some rows",
			"code":        "validation_failed",
			"errors":      parseErrors,
			"valid_rows":  len(requests),
			"failed_rows": len(parseErrors),
		})
		return
	}

	// If no valid records, return error
	if len(requests) == 0 {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: "no valid records found in CSV",
			Code:  "no_valid_records",
		})
		return
	}

	// Get database from context
	dbInterface, exists := c.Get("db")
	if !exists {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "database connection not available",
			Code:  "internal_error",
		})
		return
	}
	db := dbInterface.(*sql.DB)

	// Create client service and execute bulk insert
	clientService := services.NewClientService(db)
	result, err := clientService.BulkCreateClients(c.Request.Context(), companyID.(string), requests)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: err.Error(),
			Code:  "bulk_insert_failed",
		})
		return
	}

	c.JSON(http.StatusCreated, result)
}

// BulkUploadError represents an error for a specific row in CSV
type BulkUploadError struct {
	Row   int    `json:"row"`
	Error string `json:"error"`
}

// getCSVValue safely retrieves a value from a CSV row
func getCSVValue(row []string, headerMap map[string]int, columnName string) string {
	if idx, exists := headerMap[columnName]; exists && idx < len(row) {
		return strings.TrimSpace(row[idx])
	}
	return ""
}
