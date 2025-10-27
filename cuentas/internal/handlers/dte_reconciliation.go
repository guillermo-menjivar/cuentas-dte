package handlers

import (
	"cuentas/internal/formats"
	"cuentas/internal/services"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type DTEReconciliationHandler struct {
	service *services.DTEReconciliationService
}

func NewDTEReconciliationHandler(service *services.DTEReconciliationService) *DTEReconciliationHandler {
	return &DTEReconciliationHandler{
		service: service,
	}
}

// ReconcileDTEs handles GET /v1/dte/reconciliation
func (h *DTEReconciliationHandler) ReconcileDTEs(c *gin.Context) {
	companyID := c.GetString("company_id")
	if companyID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "company_id is required"})
		return
	}

	// Parse query parameters
	var startDate, endDate, codigoGeneracion *string

	// Handle date range
	if sd := c.Query("start_date"); sd != "" {
		if _, err := time.Parse("2006-01-02", sd); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "invalid start_date format, use YYYY-MM-DD",
			})
			return
		}
		startDate = &sd
	}

	if ed := c.Query("end_date"); ed != "" {
		if _, err := time.Parse("2006-01-02", ed); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "invalid end_date format, use YYYY-MM-DD",
			})
			return
		}
		endDate = &ed
	}

	// Handle specific date (converts to start/end of day)
	if date := c.Query("date"); date != "" {
		if _, err := time.Parse("2006-01-02", date); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "invalid date format, use YYYY-MM-DD",
			})
			return
		}
		startDate = &date
		endDate = &date
	}

	// Handle month (converts to start/end of month)
	if month := c.Query("month"); month != "" {
		// Parse YYYY-MM
		start := month + "-01"
		t, err := time.Parse("2006-01-02", start)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "invalid month format, use YYYY-MM",
			})
			return
		}
		// Get last day of month
		end := t.AddDate(0, 1, -1).Format("2006-01-02")
		startDate = &start
		endDate = &end
	}

	// Handle specific codigo_generacion
	if cg := c.Query("codigo_generacion"); cg != "" {
		codigoGeneracion = &cg
	}

	// Include matches flag (default: true)
	includeMatches := c.DefaultQuery("include_matches", "true") == "true"

	// Determine output format
	format := formats.DetermineFormat(c.GetHeader("Accept"), c.Query("format"))

	log.Printf("[Reconciliation] Company: %s, StartDate: %v, EndDate: %v, CodigoGen: %v, IncludeMatches: %v",
		companyID, startDate, endDate, codigoGeneracion, includeMatches)

	// Perform reconciliation
	results, summary, err := h.service.ReconcileDTEs(
		c.Request.Context(),
		companyID,
		startDate,
		endDate,
		codigoGeneracion,
		includeMatches,
	)
	if err != nil {
		log.Printf("[ERROR] Reconciliation failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "reconciliation failed",
			"details": err.Error(),
		})
		return
	}

	// Return based on format
	if format == "csv" {
		csvData, err := formats.WriteDTEReconciliationCSV(results, summary)
		if err != nil {
			log.Printf("[ERROR] Failed to generate CSV: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "failed to generate CSV",
			})
			return
		}

		// Generate filename
		filename := fmt.Sprintf("dte_reconciliation_%s.csv", time.Now().Format("20060102_150405"))
		if startDate != nil && endDate != nil {
			filename = fmt.Sprintf("dte_reconciliation_%s_to_%s.csv", *startDate, *endDate)
		} else if codigoGeneracion != nil {
			filename = fmt.Sprintf("dte_reconciliation_%s.csv", *codigoGeneracion)
		}

		c.Header("Content-Type", "text/csv; charset=utf-8")
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
		c.Data(http.StatusOK, "text/csv", csvData)
		return
	}

	// Default: JSON
	c.JSON(http.StatusOK, gin.H{
		"summary": summary,
		"results": results,
	})
}

// ReconcileSingleDTE handles GET /v1/dte/reconciliation/:codigo_generacion
func (h *DTEReconciliationHandler) ReconcileSingleDTE(c *gin.Context) {
	companyID := c.GetString("company_id")
	if companyID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "company_id is required"})
		return
	}

	codigoGeneracion := c.Param("codigo_generacion")
	if codigoGeneracion == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "codigo_generacion is required"})
		return
	}

	log.Printf("[Reconciliation] Single DTE: %s for Company: %s", codigoGeneracion, companyID)

	// Reconcile single DTE
	result, err := h.service.ReconcileSingleDTE(
		c.Request.Context(),
		companyID,
		codigoGeneracion,
	)
	if err != nil {
		if err.Error() == "DTE not found in database" {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "DTE not found in database",
			})
			return
		}

		log.Printf("[ERROR] Single DTE reconciliation failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "reconciliation failed",
			"details": err.Error(),
		})
		return
	}

	// Determine output format
	format := formats.DetermineFormat(c.GetHeader("Accept"), c.Query("format"))

	if format == "csv" {
		csvData, err := formats.WriteDTEReconciliationCSV([]services.DTEReconciliationRecord{*result}, nil)
		if err != nil {
			log.Printf("[ERROR] Failed to generate CSV: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "failed to generate CSV",
			})
			return
		}

		filename := fmt.Sprintf("dte_reconciliation_%s.csv", codigoGeneracion)
		c.Header("Content-Type", "text/csv; charset=utf-8")
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
		c.Data(http.StatusOK, "text/csv", csvData)
		return
	}

	// Default: JSON
	c.JSON(http.StatusOK, result)
}
