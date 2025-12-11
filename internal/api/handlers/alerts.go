package handlers

import (
	"developer-portal-backend/internal/auth"
	apperrors "developer-portal-backend/internal/errors"
	"developer-portal-backend/internal/service"
	"net/http"

	"github.com/gin-gonic/gin"
)

type AlertsHandler struct {
	alertsService service.AlertsServiceInterface
}

func NewAlertsHandler(alertsService service.AlertsServiceInterface) *AlertsHandler {
	return &AlertsHandler{
		alertsService: alertsService,
	}
}

// GetAlerts godoc
// @Summary Get Prometheus alerts from GitHub repository
// @Description Fetches all Prometheus alert configurations from the configured GitHub repository
// @Tags alerts
// @Accept json
// @Produce json
// @Param projectId path string true "Project ID"
// @Success 200 {object} map[string]interface{} "Alerts data"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 404 {object} map[string]interface{} "Project not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/projects/{projectId}/alerts [get]
func (h *AlertsHandler) GetAlerts(c *gin.Context) {
	// Get authenticated user claims
	claimsInterface, exists := c.Get("auth_claims")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": apperrors.ErrAuthenticationRequired.Message})
		return
	}

	claims, ok := claimsInterface.(*auth.AuthClaims)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": apperrors.ErrAuthenticationInvalidClaims.Message})
		return
	}

	projectID := c.Param("projectId")
	if projectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": apperrors.NewMissingQueryParam("projectId").Error()})
		return
	}

	// get GitHub provider from param 'provider'. TODO set 'githubtools' if not found. prepare to support multiple providers in future - which client currently doesn't support. should be mandatory.
	provider := c.DefaultQuery("provider", "githubtools")

	alerts, err := h.alertsService.GetProjectAlerts(c.Request.Context(), projectID, claims.UUID, provider)
	if err != nil {
		if err.Error() == apperrors.ErrProjectNotFound.Error() {
			c.JSON(http.StatusNotFound, gin.H{"error": apperrors.ErrProjectNotFound.Error()})
			return
		}
		if err.Error() == apperrors.ErrAlertsRepositoryNotConfigured.Error() {
			c.JSON(http.StatusNotFound, gin.H{"error": apperrors.ErrAlertsRepositoryNotConfigured.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, alerts)
}

// CreateAlertPR godoc
// @Summary Create a pull request with alert changes
// @Description Creates a pull request to update Prometheus alert configurations in GitHub
// @Tags alerts
// @Accept json
// @Produce json
// @Param projectId path string true "Project ID"
// @Param body body map[string]interface{} true "Alert changes"
// @Success 200 {object} map[string]interface{} "PR created successfully"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 404 {object} map[string]interface{} "Project not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/projects/{projectId}/alerts/pr [post]
func (h *AlertsHandler) CreateAlertPR(c *gin.Context) {
	// Get authenticated user claims
	claimsInterface, exists := c.Get("auth_claims")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": apperrors.ErrAuthenticationRequired})
		return
	}

	claims, ok := claimsInterface.(*auth.AuthClaims)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": apperrors.ErrAuthenticationInvalidClaims})
		return
	}

	projectID := c.Param("projectId")
	if projectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": apperrors.NewMissingQueryParam("projectId").Error()})
		return
	}

	// get GitHub provider from param 'provider'. TODO set 'githubtools' if not found. prepare to support multiple providers in future - which client currently doesn't support. should be mandatory.
	provider := c.DefaultQuery("provider", "githubtools")

	var payload struct {
		FileName    string `json:"fileName"`
		Content     string `json:"content"`
		Message     string `json:"message"`
		Description string `json:"description"`
	}

	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	prURL, err := h.alertsService.CreateAlertPR(c.Request.Context(), projectID, claims.UUID, provider, payload.FileName, payload.Content, payload.Message, payload.Description)
	if err != nil {
		if err.Error() == apperrors.ErrProjectNotFound.Error() {
			c.JSON(http.StatusNotFound, gin.H{"error": apperrors.ErrProjectNotFound.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Pull request created successfully",
		"prUrl":   prURL,
	})
}
