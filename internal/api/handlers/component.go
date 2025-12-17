package handlers

import (
	apperrors "developer-portal-backend/internal/errors"
	"developer-portal-backend/internal/logger"
	"developer-portal-backend/internal/service"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ComponentHandler handles HTTP requests for component operations
type ComponentHandler struct {
	componentService service.ComponentServiceInterface
	landscapeService service.LandscapeServiceInterface
	teamService      service.TeamServiceInterface
	projectService   service.ProjectServiceInterface
}

// NewComponentHandler creates a component handler with all services
func NewComponentHandler(componentService service.ComponentServiceInterface, landscapeService service.LandscapeServiceInterface, teamService service.TeamServiceInterface, projectService service.ProjectServiceInterface) *ComponentHandler {
	return &ComponentHandler{
		componentService: componentService,
		landscapeService: landscapeService,
		teamService:      teamService,
		projectService:   projectService,
	}
}

// ComponentHealth handles GET /components/health
// @Param component-id query string false "Component ID (UUID)"
// @Param landscape-id query string false "Landscape ID (UUID)"
// @Security BearerAuth
// @Router /components/health [get]
func (h *ComponentHandler) ComponentHealth(c *gin.Context) {
	componentIDStr := c.Query("component-id")
	landscapeIDStr := c.Query("landscape-id")
	if componentIDStr != "" && landscapeIDStr != "" {
		compID, err := uuid.Parse(componentIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"healthy": false, "error": apperrors.ErrInvalidComponentID.Error(), "details": fmt.Sprintf("failed to parse component-id: %s", componentIDStr)})
			return
		}
		landID, err := uuid.Parse(landscapeIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"healthy": false, "error": apperrors.ErrInvalidLandscapeID.Error(), "details": fmt.Sprintf("failed to parse landscape-id: %s", landscapeIDStr)})
			return
		}
		// Compose health URL via helper (fully encapsulated: component/landscape/template fetch + composition)
		healthURL, healthSuccessRegEx, err := BuildComponentHealthURL(h.componentService, h.landscapeService, h.projectService, compID, landID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"healthy": false, "error": err.Error(), "details": fmt.Sprintf("failed to build component health URL. healthURL=%s successRegEx=%s", healthURL, healthSuccessRegEx)})
			return
		}

		logger.FromGinContext(c).Debugf("components health proxy URL=%s", healthURL)

		// Fetch URL with timeout
		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Get(healthURL)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"healthy": false, "error": "failed to fetch component health", "details": err.Error(), "healthURL": healthURL})
			return
		}
		defer func(Body io.ReadCloser) {
			err := Body.Close()
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"healthy": false, "error": "failed to close component health response body", "details": err.Error(), "healthURL": healthURL})
				return
			}
		}(resp.Body)
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"healthy": false, "error": "failed to read component health response", "details": err.Error(), "healthURL": healthURL, "statusCode": resp.StatusCode})
			return
		}
		responseBody := string(bodyBytes)
		// check if responseBody matches healthSuccessRegEx
		healthy, err := regexp.MatchString(healthSuccessRegEx, responseBody)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"healthy": false, "error": err.Error(), "details": "failed to apply health success regex", "healthURL": healthURL, "statusCode": resp.StatusCode})
			return
		}
		c.JSON(http.StatusOK, gin.H{"healthy": healthy, "details": responseBody, "healthURL": healthURL, "statusCode": resp.StatusCode})
		return
	}
	// Missing parameters: both component-id and landscape-id are required
	c.JSON(http.StatusBadRequest, gin.H{"healthy": false, "error": apperrors.ErrMissingHealthParams.Error()})

}

// ListComponents handles GET /components
// @Summary List components
// @Description List components filtered by either team-id or project-name. Returns an array of minimal component views. One of team-id or project-name is required.
// @Tags components
// @Accept json
// @Produce json
// @Param team-id query string false "Team ID (UUID) to filter by owner_id"
// @Param project-name query string false "Project name"
// @Success 200 {array} object "Successfully retrieved components"
// @Failure 400 {object} map[string]interface{} "Invalid parameters"
// @Failure 404 {object} map[string]interface{} "Not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Security BearerAuth
// @Router /components [get]
func (h *ComponentHandler) ListComponents(c *gin.Context) {
	projectName := c.Query("project-name")
	// If team-id is provided, return components owned by that team (uses pagination)
	teamIDStr := c.Query("team-id")
	if teamIDStr != "" {
		teamID, err := uuid.Parse(teamIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": apperrors.ErrInvalidTeamID.Error()})
			return
		}
		components, _, err := h.teamService.GetTeamComponentsByID(teamID, 1, 1000000)
		if err != nil {
			if errors.Is(err, apperrors.ErrTeamNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		// Build minimal view items with project info (same fields as project-name view plus project_id and project_title)
		items := make([]gin.H, len(components))
		for i, comp := range components {
			// Extract qos, sonar, github from metadata if present
			var qos, sonar, github string
			var centralService, isLibrary, health *bool

			if len(comp.Metadata) > 0 {
				var meta map[string]interface{}
				if err := json.Unmarshal(comp.Metadata, &meta); err == nil {
					// qos from metadata.ci.qos
					if ciRaw, ok := meta["ci"]; ok {
						if ciMap, ok := ciRaw.(map[string]interface{}); ok {
							if qosRaw, ok := ciMap["qos"]; ok {
								if qosStr, ok := qosRaw.(string); ok {
									qos = qosStr
								}
							}
						}
					}
					// sonar from metadata.sonar.project_id
					if sonarRaw, ok := meta["sonar"]; ok {
						if sonarMap, ok := sonarRaw.(map[string]interface{}); ok {
							if pidRaw, ok := sonarMap["project_id"]; ok {
								if pidStr, ok := pidRaw.(string); ok {
									sonar = "https://sonar.tools.sap/dashboard?id=" + pidStr
								}
							}
						}
					}
					// GitHub from metadata.github.url
					if ghRaw, ok := meta["github"]; ok {
						if ghMap, ok := ghRaw.(map[string]interface{}); ok {
							if urlRaw, ok := ghMap["url"]; ok {
								if urlStr, ok := urlRaw.(string); ok {
									github = urlStr
								}
							}
						}
					}
					// central-service from metadata["central-service"]
					if csRaw, ok := meta["central-service"]; ok {
						if csBool, ok := csRaw.(bool); ok {
							b := csBool
							centralService = &b
						}
					}
					// is-library from metadata["isLibrary"] (mapped to is-library)
					if ilRaw, ok := meta["isLibrary"]; ok {
						if ilBool, ok := ilRaw.(bool); ok {
							b := ilBool
							isLibrary = &b
						}
					}
					// health from metadata["health"] (boolean)
					if hRaw, ok := meta["health"]; ok {
						if hBool, ok := hRaw.(bool); ok {
							b := hBool
							health = &b
						}
					}
				}
			}

			// Fetch project title (non-fatal if not found)
			projectTitle := ""
			if title, err := h.componentService.GetProjectTitleByID(comp.ProjectID); err == nil {
				projectTitle = title
			}

			m := gin.H{
				"id":            comp.ID,
				"owner_id":      comp.OwnerID,
				"name":          comp.Name,
				"title":         comp.Title,
				"description":   comp.Description,
				"qos":           qos,
				"sonar":         sonar,
				"github":        github,
				"project_id":    comp.ProjectID,
				"project_title": projectTitle,
			}
			if centralService != nil {
				m["central-service"] = *centralService
			}
			if isLibrary != nil {
				m["is-library"] = *isLibrary
			}
			if health != nil {
				m["health"] = *health
			}
			items[i] = m
		}
		c.JSON(http.StatusOK, items)
		return
	}

	// If project-name is provided, return ALL components for the project (unpaginated minimal view) and ignore organization_id requirement
	if projectName != "" {
		views, err := h.componentService.GetByProjectNameAllView(projectName)
		if err != nil {
			if errors.Is(err, apperrors.ErrProjectNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, views)
		return
	}

	c.JSON(http.StatusBadRequest, gin.H{"error": apperrors.ErrMissingTeamOrProjectName.Error()})
	return
}
