// File: handlers/actividad_economica_handler.go
package handlers

import (
	"net/http"
	"strconv"

	"cuentas/internal/services"

	"github.com/gin-gonic/gin"
)

type ActividadEconomicaHandler struct {
	actividadService *services.ActividadEconomicaService
}

func NewActividadEconomicaHandler() *ActividadEconomicaHandler {
	return &ActividadEconomicaHandler{
		actividadService: services.NewActividadEconomicaService(),
	}
}

// GetCategories handles GET /api/v1/actividades-economicas/categories
func (h *ActividadEconomicaHandler) GetCategories(c *gin.Context) {
	categories := h.actividadService.GetAllCategories()

	c.JSON(http.StatusOK, gin.H{
		"categories": categories,
		"count":      len(categories),
	})
}

// GetCategoryByCode handles GET /api/v1/actividades-economicas/categories/:code
func (h *ActividadEconomicaHandler) GetCategoryByCode(c *gin.Context) {
	code := c.Param("code")
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "code is required"})
		return
	}

	category, exists := h.actividadService.GetCategoryByCode(code)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "category not found"})
		return
	}

	c.JSON(http.StatusOK, category)
}

// GetActivitiesByCategory handles GET /api/v1/actividades-economicas/categories/:code/activities
func (h *ActividadEconomicaHandler) GetActivitiesByCategory(c *gin.Context) {
	code := c.Param("code")
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "code is required"})
		return
	}

	activities := h.actividadService.GetActivitiesByCategory(code)

	c.JSON(http.StatusOK, gin.H{
		"category_code": code,
		"activities":    activities,
		"count":         len(activities),
	})
}

// SearchActivities handles GET /api/v1/actividades-economicas/search
func (h *ActividadEconomicaHandler) SearchActivities(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "query parameter 'q' is required"})
		return
	}

	// Get limit from query params, default to 10
	limitStr := c.DefaultQuery("limit", "10")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 10
	}
	if limit > 50 {
		limit = 50 // Cap at 50 results
	}

	results := h.actividadService.SearchActivities(query, limit)

	c.JSON(http.StatusOK, gin.H{
		"query":   query,
		"results": results,
		"count":   len(results),
	})
}

// GetActivityDetails handles GET /api/v1/actividades-economicas/:code
func (h *ActividadEconomicaHandler) GetActivityDetails(c *gin.Context) {
	code := c.Param("code")
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "code is required"})
		return
	}

	details, exists := h.actividadService.GetActivityDetails(code)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "activity not found"})
		return
	}

	c.JSON(http.StatusOK, details)
}
