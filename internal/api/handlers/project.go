package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"developer-portal-backend/internal/service"

	"github.com/gin-gonic/gin"
)

// ProjectHandler handles HTTP requests for project operations
type ProjectHandler struct {
	projectService service.ProjectServiceInterface
}

// NewProjectHandler creates a new project handler
func NewProjectHandler(projectService service.ProjectServiceInterface) *ProjectHandler {
	return &ProjectHandler{
		projectService: projectService,
	}
}

// Project represents the structure of a project (local struct)
type Project struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// ProjectResponse represents a trimmed project projection for list endpoints
type ProjectResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Title       string `json:"title"`
	Description string `json:"description"`
	// Enriched fields extracted from metadata
	Alerts            string `json:"alerts,omitempty"`
	Views             string `json:"views,omitempty"`
	ComponentsMetrics bool   `json:"components-metrics"`
}

// GetAllProjects handles GET /projects
func (h *ProjectHandler) GetAllProjects(c *gin.Context) {
	serviceProjects, err := h.projectService.GetAllProjects()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var responses []ProjectResponse
	for _, sp := range serviceProjects {
		// Unmarshal Metadata from json.RawMessage to map[string]interface{}
		var metadata map[string]interface{}
		if len(sp.Metadata) > 0 {
			if err := json.Unmarshal(sp.Metadata, &metadata); err != nil {
				metadata = map[string]interface{}{} // fallback if unmarshal fails
			}
		} else {
			metadata = map[string]interface{}{}
		}

		alerts, _ := metadata["alerts"].(string)
		componentsMetrics, _ := metadata["components-metrics"].(bool)

		// Extract views from metadata.views (array) and join with commas
		rawViews, _ := metadata["views"].([]interface{})
		views := make([]string, 0, len(rawViews))
		for _, v := range rawViews {
			if s, ok := v.(string); ok {
				views = append(views, s)
			}
		}
		viewsStr := strings.Join(views, ",")

		response := ProjectResponse{
			ID:                sp.ID.String(),
			Name:              sp.Name,
			Title:             sp.Title,
			Description:       sp.Description,
			Alerts:            alerts,
			Views:             viewsStr,
			ComponentsMetrics: componentsMetrics,
		}

		responses = append(responses, response)
	}

	c.JSON(http.StatusOK, responses)
}
