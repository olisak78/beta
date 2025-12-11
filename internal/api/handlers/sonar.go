package handlers

import (
	"developer-portal-backend/internal/errors"
	"net/http"

	"developer-portal-backend/internal/service"

	"github.com/gin-gonic/gin"
)

// SonarHandler handles Sonar-related HTTP requests
type SonarHandler struct {
	service service.SonarServiceInterface
}

// NewSonarHandler creates a new Sonar handler
func NewSonarHandler(s service.SonarServiceInterface) *SonarHandler {
	return &SonarHandler{service: s}
}

// GetMeasures merges Sonar measures and quality gate status for a given component (project key).
// @Summary Get Sonar measures and quality gate status
// @Description Calls Sonar APIs to retrieve measures (coverage, vulnerabilities, code_smells) and quality gate status for the given component key, merges them and returns as JSON.
// @Tags sonar
// @Produce json
// @Param component query string true "Sonar project key"
// @Success 200 {object} service.SonarCombinedResponse
// @Failure 400 {object} map[string]string "Missing or invalid query parameter"
// @Failure 502 {object} map[string]string "Sonar request failed"
// @Security BearerAuth
// @Router /sonar/measures [get]
func (h *SonarHandler) GetMeasures(c *gin.Context) {
	component := c.Query("component")
	if component == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": errors.NewMissingQueryParam("component").Error()})
		return
	}

	resp, err := h.service.GetComponentMeasures(component)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "sonar request failed: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}
