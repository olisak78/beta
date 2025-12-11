package handlers

import (
	"developer-portal-backend/internal/errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"developer-portal-backend/internal/auth"
	"developer-portal-backend/internal/service"

	"github.com/gin-gonic/gin"
)

// JiraHandler handles Jira-related HTTP requests
type JiraHandler struct {
	service service.JiraServiceInterface
}

// NewJiraHandler creates a new Jira handler
func NewJiraHandler(s service.JiraServiceInterface) *JiraHandler {
	return &JiraHandler{service: s}
}

// GetIssues returns Jira issues for teams/projects with real Jira values that convert to JQL.
// @Summary Get Jira issues for teams/projects
// @Description Returns Jira issues filtered by project, status, team, assignee, type, summary, and key using real Jira values with pagination
// @Tags jira
// @Produce json
// @Param project query string false "Jira project key (e.g., SAPBTPCFS)"
// @Param status query string false "Jira status values (e.g., 'Open,In Progress,Re Opened')"
// @Param team query string false "Team name for filtering"
// @Param assignee query string false "Assignee username for filtering"
// @Param type query string false "Issue type (e.g., 'Bug,Task,Story')"
// @Param summary query string false "Free text search in summary"
// @Param key query string false "Specific issue key (e.g., 'BUG-1234')"
// @Param page query int false "Page number (default: 1)"
// @Param limit query int false "Number of items per page (default: 50, max: 100)"
// @Success 200 {object} service.JiraIssuesResponse "Issues"
// @Failure 400 {object} map[string]string "Invalid pagination parameters"
// @Failure 502 {object} map[string]string "Jira request failed"
// @Security BearerAuth
// @Router /jira/issues [get]
func (h *JiraHandler) GetIssues(c *gin.Context) {
	// Get query parameters
	project := c.Query("project")
	status := c.Query("status")
	team := c.Query("team")
	assignee := c.Query("assignee")
	issueType := c.Query("type")
	summary := c.Query("summary")
	key := c.Query("key")

	// Parse pagination parameters
	page, limit, err := h.parsePaginationParams(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Create filters with real Jira values
	filters := service.JiraIssueFilters{
		Project:  project,
		Status:   status,
		Team:     team,
		Assignee: assignee,
		Type:     issueType,
		Summary:  summary,
		Key:      key,
		Page:     page,
		Limit:    limit,
	}

	issues, err := h.service.GetIssues(filters)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "jira search failed: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, issues)
}

// GetMyIssues returns Jira issues for the current authenticated user.
// @Summary Get my Jira issues
// @Description Returns Jira issues for the current authenticated user with optional filtering and pagination
// @Tags jira
// @Produce json
// @Param status query string false "Jira status values (e.g., 'Open,In Progress')"
// @Param project query string false "Jira project key (e.g., SAPBTPCFS)"
// @Param page query int false "Page number (default: 1)"
// @Param limit query int false "Number of items per page (default: 50, max: 100)"
// @Success 200 {object} service.JiraIssuesResponse "Issues"
// @Failure 400 {object} map[string]string "Invalid pagination parameters"
// @Failure 401 {object} map[string]string "Authentication required"
// @Failure 502 {object} map[string]string "Jira request failed"
// @Security BearerAuth
// @Router /jira/issues/me [get]
func (h *JiraHandler) GetMyIssues(c *gin.Context) {
	// Get authenticated user claims from context (set by auth middleware)
	claimsInterface, exists := c.Get("auth_claims")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	claims, ok := claimsInterface.(*auth.AuthClaims)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authentication claims"})
		return
	}

	// Use the username from the authenticated user
	username := claims.Username
	if username == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Username not available in authentication claims"})
		return
	}

	// Get query parameters
	status := c.Query("status")
	project := c.Query("project")

	// Parse pagination parameters
	page, limit, err := h.parsePaginationParams(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Create filters with user from auth context
	filters := service.JiraIssueFilters{
		Status:  status,
		Project: project,
		User:    username,
		Page:    page,
		Limit:   limit,
	}

	issues, err := h.service.GetIssues(filters)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "jira search failed: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, issues)
}

// GetMyIssuesCount returns the count of Jira issues for the current authenticated user by status.
// @Summary Get count of my Jira issues by status
// @Description Returns the count of Jira issues for the current authenticated user filtered by status
// @Tags jira
// @Produce json
// @Param status query string true "Jira status value (e.g., 'Resolved')"
// @Param project query string false "Jira project key (e.g., SAPBTPCFS)"
// @Param date query string false "Date in yyyy-MM-dd format for date filtering (default: one year ago for resolved issues)"
// @Success 200 {object} map[string]int "Count"
// @Failure 400 {object} map[string]string "Missing or invalid query parameter"
// @Failure 401 {object} map[string]string "Authentication required"
// @Failure 502 {object} map[string]string "Jira request failed"
// @Security BearerAuth
// @Router /jira/issues/me/count [get]
func (h *JiraHandler) GetMyIssuesCount(c *gin.Context) {
	// Get authenticated user claims from context (set by auth middleware)
	claimsInterface, exists := c.Get("auth_claims")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	claims, ok := claimsInterface.(*auth.AuthClaims)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authentication claims"})
		return
	}

	// Use the username from the authenticated user
	username := claims.Username
	if username == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Username not available in authentication claims"})
		return
	}

	// Get query parameters
	status := c.Query("status")
	if status == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": errors.NewMissingQueryParam("status").Error()})
		return
	}

	project := c.Query("project")
	date := c.Query("date")

	// Set default date for resolved issues if not provided
	if date == "" && status == "Resolved" {
		date = time.Now().AddDate(-1, 0, 0).Format("2006-01-02")
	} else if date != "" {
		// Validate date format if provided
		if _, err := time.Parse("2006-01-02", date); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid date format: must be yyyy-MM-dd"})
			return
		}
	}

	// Create filters with user from auth context
	filters := service.JiraIssueFilters{
		Status:  status,
		Project: project,
		User:    username,
		Date:    date,
	}

	count, err := h.service.GetIssuesCount(filters)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "jira search failed: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"count": count})
}

// parsePaginationParams parses and validates pagination parameters from the request
func (h *JiraHandler) parsePaginationParams(c *gin.Context) (page, limit int, err error) {
	// Default values
	page = 1
	limit = 50

	// Parse page parameter
	if pageStr := c.Query("page"); pageStr != "" {
		if page, err = strconv.Atoi(pageStr); err != nil {
			return 0, 0, fmt.Errorf("invalid page parameter: must be a positive integer")
		}
		if page < 1 {
			return 0, 0, fmt.Errorf("invalid page parameter: must be greater than 0")
		}
	}

	// Parse limit parameter
	if limitStr := c.Query("limit"); limitStr != "" {
		if limit, err = strconv.Atoi(limitStr); err != nil {
			return 0, 0, fmt.Errorf("invalid limit parameter: must be a positive integer")
		}
		if limit < 1 {
			return 0, 0, fmt.Errorf("invalid limit parameter: must be greater than 0")
		}
		if limit > 100 {
			return 0, 0, fmt.Errorf("invalid limit parameter: maximum allowed is 100")
		}
	}

	return page, limit, nil
}
