package handlers

import (
	"encoding/json"
	"net/http"

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

// ProjectMinimalResponse represents a trimmed project projection for list endpoints
type ProjectMinimalResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Title       string `json:"title"`
	Description string `json:"description"`
	// Enriched nested fields extracted from metadata
	Alerts   map[string]interface{} `json:"alerts,omitempty"`
	Health   map[string]interface{} `json:"health,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// ProjectEnrichment holds enriched metadata fields and remaining metadata
type ProjectEnrichment struct {
	Alerts            map[string]interface{}
	Health            map[string]interface{}
	RemainingMetadata map[string]interface{}
}

// enrichProjectMetadata extracts alerts and health from metadata, keeps remaining metadata
func enrichProjectMetadata(metadata map[string]interface{}) ProjectEnrichment {
	enr := ProjectEnrichment{
		Alerts:            make(map[string]interface{}),
		Health:            make(map[string]interface{}),
		RemainingMetadata: make(map[string]interface{}),
	}

	// Extract alerts and health, keep everything else in remaining metadata
	for k, v := range metadata {
		switch k {
		case "alerts":
			if alertsMap, ok := v.(map[string]interface{}); ok {
				enr.Alerts = alertsMap
			}
		case "health":
			if healthMap, ok := v.(map[string]interface{}); ok {
				enr.Health = healthMap
			}
		default:
			// Keep all other metadata fields
			enr.RemainingMetadata[k] = v
		}
	}

	return enr
}

// GetAllProjects handles GET /projects
func (h *ProjectHandler) GetAllProjects(c *gin.Context) {
	serviceProjects, err := h.projectService.GetAllProjects()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var responses []ProjectMinimalResponse
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

		// Enrich metadata (following landscape pattern)
		enr := enrichProjectMetadata(metadata)

		// Create response struct directly with enriched fields (following landscape pattern)
		response := ProjectMinimalResponse{
			ID:          sp.ID.String(),
			Name:        sp.Name,
			Title:       sp.Title,
			Description: sp.Description,
		}

		// Only add alerts, health, and metadata if they contain data
		if len(enr.Alerts) > 0 {
			response.Alerts = enr.Alerts
		}
		if len(enr.Health) > 0 {
			response.Health = enr.Health
		}
		if len(enr.RemainingMetadata) > 0 {
			response.Metadata = enr.RemainingMetadata
		}

		responses = append(responses, response)
	}

	c.JSON(http.StatusOK, responses)
}
