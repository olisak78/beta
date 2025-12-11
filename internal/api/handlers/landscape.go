package handlers

import (
	"errors"
	"net/http"

	apperrors "developer-portal-backend/internal/errors"
	"developer-portal-backend/internal/service"

	"github.com/gin-gonic/gin"
)

// LandscapeHandler handles HTTP requests for landscape operations
type LandscapeHandler struct {
	landscapeService service.LandscapeServiceInterface
}

// NewLandscapeHandler creates a new landscape handler
func NewLandscapeHandler(landscapeService service.LandscapeServiceInterface) *LandscapeHandler {
	return &LandscapeHandler{
		landscapeService: landscapeService,
	}
}

// ListLandscapesByQuery handles GET /landscapes?project-name=<project_name>
// @Summary List landscapes by project name
// @Description Return all landscapes that belong to the specified project (unpaginated, minimal fields)
// @Tags landscapes
// @Accept json
// @Produce json
// @Param project-name query string true "Project name"
// @Success 200 {array} service.LandscapeMinimalResponse "Successfully retrieved landscapes by project name"
// @Failure 400 {object} map[string]interface{} "project-name is required"
// @Failure 404 {object} map[string]interface{} "Project not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Security BearerAuth
// @Router /landscapes [get]
func (h *LandscapeHandler) ListLandscapesByQuery(c *gin.Context) {
	projectName := c.Query("project-name")
	if projectName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": apperrors.NewMissingQueryParam("project-name").Error()})
		return
	}

	mins, err := h.landscapeService.GetByProjectNameAll(projectName)
	if err != nil {
		if errors.Is(err, apperrors.ErrProjectNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, mins)
}
