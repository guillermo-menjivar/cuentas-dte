package handlers

import (
	"net/http"

	"cuentas/internal/models"
	"cuentas/internal/services"

	"github.com/gin-gonic/gin"
)

type EstablishmentHandler struct {
	service *services.EstablishmentService
}

func NewEstablishmentHandler() *EstablishmentHandler {
	return &EstablishmentHandler{
		service: services.NewEstablishmentService(),
	}
}

// CreateEstablishment creates a new establishment
// POST /v1/establishments
func (h *EstablishmentHandler) CreateEstablishment(c *gin.Context) {
	companyID := c.GetString("company_id")
	if companyID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "company_id not found in context"})
		return
	}

	var req models.CreateEstablishmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	establishment, err := h.service.CreateEstablishment(c.Request.Context(), companyID, &req)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, establishment)
}

// GetEstablishment retrieves an establishment by ID
// GET /v1/establishments/:id
func (h *EstablishmentHandler) GetEstablishment(c *gin.Context) {
	companyID := c.GetString("company_id")
	if companyID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "company_id not found in context"})
		return
	}

	establishmentID := c.Param("id")
	if establishmentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "establishment id is required"})
		return
	}

	establishment, err := h.service.GetEstablishment(c.Request.Context(), companyID, establishmentID)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, establishment)
}

// ListEstablishments lists all establishments for a company
// GET /v1/establishments
func (h *EstablishmentHandler) ListEstablishments(c *gin.Context) {
	companyID := c.GetString("company_id")
	if companyID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "company_id not found in context"})
		return
	}

	// Check for active_only query parameter
	activeOnly := c.DefaultQuery("active_only", "true") == "true"

	establishments, err := h.service.ListEstablishments(c.Request.Context(), companyID, activeOnly)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"establishments": establishments,
		"count":          len(establishments),
	})
}

// UpdateEstablishment updates an establishment
// PATCH /v1/establishments/:id
func (h *EstablishmentHandler) UpdateEstablishment(c *gin.Context) {
	companyID := c.GetString("company_id")
	if companyID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "company_id not found in context"})
		return
	}

	establishmentID := c.Param("id")
	if establishmentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "establishment id is required"})
		return
	}

	var req models.UpdateEstablishmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	establishment, err := h.service.UpdateEstablishment(c.Request.Context(), companyID, establishmentID, &req)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, establishment)
}

// DeactivateEstablishment deactivates an establishment
// DELETE /v1/establishments/:id
func (h *EstablishmentHandler) DeactivateEstablishment(c *gin.Context) {
	companyID := c.GetString("company_id")
	if companyID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "company_id not found in context"})
		return
	}

	establishmentID := c.Param("id")
	if establishmentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "establishment id is required"})
		return
	}

	err := h.service.DeactivateEstablishment(c.Request.Context(), companyID, establishmentID)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "establishment deactivated successfully"})
}

// CreatePointOfSale creates a new point of sale for an establishment
// POST /v1/establishments/:id/pos
func (h *EstablishmentHandler) CreatePointOfSale(c *gin.Context) {
	companyID := c.GetString("company_id")
	if companyID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "company_id not found in context"})
		return
	}

	establishmentID := c.Param("id")
	if establishmentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "establishment id is required"})
		return
	}

	var req models.CreatePOSRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	pos, err := h.service.CreatePointOfSale(c.Request.Context(), companyID, establishmentID, &req)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, pos)
}

// GetPointOfSale retrieves a point of sale by ID
// GET /v1/pos/:id
func (h *EstablishmentHandler) GetPointOfSale(c *gin.Context) {
	companyID := c.GetString("company_id")
	if companyID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "company_id not found in context"})
		return
	}

	posID := c.Param("id")
	if posID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "pos id is required"})
		return
	}

	pos, err := h.service.GetPointOfSale(c.Request.Context(), companyID, posID)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, pos)
}

// ListPointsOfSale lists all points of sale for an establishment
// GET /v1/establishments/:id/pos
func (h *EstablishmentHandler) ListPointsOfSale(c *gin.Context) {
	companyID := c.GetString("company_id")
	if companyID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "company_id not found in context"})
		return
	}

	establishmentID := c.Param("id")
	if establishmentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "establishment id is required"})
		return
	}

	// Check for active_only query parameter
	activeOnly := c.DefaultQuery("active_only", "true") == "true"

	pointsOfSale, err := h.service.ListPointsOfSale(c.Request.Context(), companyID, establishmentID, activeOnly)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"points_of_sale": pointsOfSale,
		"count":          len(pointsOfSale),
	})
}

// UpdatePointOfSale updates a point of sale
// PATCH /v1/pos/:id
func (h *EstablishmentHandler) UpdatePointOfSale(c *gin.Context) {
	companyID := c.GetString("company_id")
	if companyID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "company_id not found in context"})
		return
	}

	posID := c.Param("id")
	if posID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "pos id is required"})
		return
	}

	var req models.UpdatePOSRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	pos, err := h.service.UpdatePointOfSale(c.Request.Context(), companyID, posID, &req)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, pos)
}

// UpdatePOSLocation updates the GPS location of a point of sale
// PATCH /v1/pos/:id/location
func (h *EstablishmentHandler) UpdatePOSLocation(c *gin.Context) {
	companyID := c.GetString("company_id")
	if companyID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "company_id not found in context"})
		return
	}

	posID := c.Param("id")
	if posID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "pos id is required"})
		return
	}

	var req models.UpdatePOSLocationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.service.UpdatePOSLocation(c.Request.Context(), companyID, posID, &req)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "location updated successfully"})
}

// DeactivatePointOfSale deactivates a point of sale
// DELETE /v1/pos/:id
func (h *EstablishmentHandler) DeactivatePointOfSale(c *gin.Context) {
	companyID := c.GetString("company_id")
	if companyID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "company_id not found in context"})
		return
	}

	posID := c.Param("id")
	if posID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "pos id is required"})
		return
	}

	err := h.service.DeactivatePointOfSale(c.Request.Context(), companyID, posID)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "point of sale deactivated successfully"})
}

// Helper function to handle errors consistently
func handleError(c *gin.Context, err error) {
	switch err {
	case models.ErrEstablishmentNotFound:
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case models.ErrPointOfSaleNotFound:
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case models.ErrInvalidTipoEstablecimiento:
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case models.ErrInvalidDepartamento:
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case models.ErrInvalidMunicipio:
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case models.ErrInvalidCodEstablecimientoMH:
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case models.ErrInvalidCodEstablecimiento:
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case models.ErrInvalidCodPuntoVentaMH:
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case models.ErrInvalidCodPuntoVenta:
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case models.ErrInvalidLatitude:
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case models.ErrInvalidLongitude:
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
	}
}
