package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"developer-portal-backend/internal/auth"
	apperrors "developer-portal-backend/internal/errors"
	"developer-portal-backend/internal/service"

	"github.com/gin-gonic/gin"
)

// GitHubHandler handles GitHub-related HTTP requests
type GitHubHandler struct {
	service service.GitHubServiceInterface
}

// NewGitHubHandler creates a new GitHub handler
func NewGitHubHandler(s service.GitHubServiceInterface) *GitHubHandler {
	return &GitHubHandler{service: s}
}

// getAuthClaims extracts and validates auth claims from the Gin context
// Returns the claims if successful, or sends an error response and returns nil
func getAuthClaims(c *gin.Context) *auth.AuthClaims {
	claimsInterface, exists := c.Get("auth_claims")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": apperrors.ErrAuthenticationRequired})
		return nil
	}

	claims, ok := claimsInterface.(*auth.AuthClaims)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": apperrors.ErrAuthenticationInvalidClaims})
		return nil
	}

	return claims
}

// GetMyPullRequests returns all pull requests created by the authenticated user
// @Summary Get my pull requests
// @Description Returns all pull requests created by the authenticated user across all repositories they have access to
// @Tags github
// @Produce json
// @Param state query string false "Filter by state: open, closed, all" default(open)
// @Param sort query string false "Sort by: created, updated, popularity, long-running" default(created)
// @Param direction query string false "Sort direction: asc, desc" default(desc)
// @Param per_page query int false "Results per page (1-100)" default(30)
// @Param page query int false "Page number" default(1)
// @Success 200 {object} service.PullRequestsResponse
// @Failure 401 {object} ErrorResponse "Unauthorized"
// @Failure 502 {object} ErrorResponse "GitHub API error"
// @Security BearerAuth
// @Router /github/pull-requests [get]
func (h *GitHubHandler) GetMyPullRequests(c *gin.Context) {
	claims := getAuthClaims(c)
	if claims == nil {
		return
	}

	// Get query parameters
	state := c.DefaultQuery("state", "open")
	sort := c.DefaultQuery("sort", "created")
	direction := c.DefaultQuery("direction", "desc")

	perPageStr := c.DefaultQuery("per_page", "30")
	perPage, err := strconv.Atoi(perPageStr)
	if err != nil || perPage <= 0 || perPage > 100 {
		perPage = 30
	}

	pageStr := c.DefaultQuery("page", "1")
	page, err := strconv.Atoi(pageStr)
	if err != nil || page <= 0 {
		page = 1
	}

	// Validate state parameter
	if state != "open" && state != "closed" && state != "all" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid state parameter. Must be: open, closed, or all"})
		return
	}

	// Validate sort parameter
	validSorts := map[string]bool{
		"created":      true,
		"updated":      true,
		"popularity":   true,
		"long-running": true,
	}
	if !validSorts[sort] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid sort parameter. Must be: created, updated, popularity, or long-running"})
		return
	}

	// Validate direction parameter
	if direction != "asc" && direction != "desc" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid direction parameter. Must be: asc or desc"})
		return
	}

	// get GitHub provider from param 'provider'. TODO set 'githubtools' if not found. prepare to support multiple providers in future - which client currently doesn't support. should be mandatory.
	provider := c.DefaultQuery("provider", "githubtools")

	// Call service to get pull requests
	response, err := h.service.GetUserOpenPullRequests(c.Request.Context(), claims.UUID, provider, state, sort, direction, perPage, page)
	if err != nil {
		// Check for specific error types
		if errors.Is(err, apperrors.ErrGitHubAPIRateLimitExceeded) {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusBadGateway, gin.H{"error": "Failed to fetch pull requests: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

// GetUserTotalContributions returns the total contributions count for the authenticated user
// @Summary Get user total contributions
// @Description Returns the total number of contributions made by the authenticated user. If no period specified, uses GitHub's default (last year based on user's timezone). Uses GitHub GraphQL API to fetch contribution data.
// @Tags github
// @Produce json
// @Param period query string false "Time period in days (e.g., '30d', '90d', '365d'). If omitted, uses GitHub's default period. Maximum: 365 days"
// @Success 200 {object} service.TotalContributionsResponse
// @Failure 400 {object} ErrorResponse "Invalid period parameter"
// @Failure 401 {object} ErrorResponse "Unauthorized"
// @Failure 429 {object} ErrorResponse "Rate limit exceeded"
// @Failure 502 {object} ErrorResponse "GitHub API error"
// @Security BearerAuth
// @Router /github/contributions [get]
func (h *GitHubHandler) GetUserTotalContributions(c *gin.Context) {
	claims := getAuthClaims(c)
	if claims == nil {
		return
	}

	// Get query parameter for period (empty = use GitHub's default)
	period := c.Query("period")

	// get GitHub provider from param 'provider'. TODO set 'githubtools' if not found. prepare to support multiple providers in future - which client currently doesn't support. should be mandatory.
	provider := c.DefaultQuery("provider", "githubtools")

	// Call service to get total contributions
	response, err := h.service.GetUserTotalContributions(c.Request.Context(), claims.UUID, provider, period)
	if err != nil {
		// Check for specific error types
		if errors.Is(err, apperrors.ErrGitHubAPIRateLimitExceeded) {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": err.Error()})
			return
		}
		// Check if it's a validation error (invalid period format)
		if errors.Is(err, apperrors.ErrInvalidPeriodFormat) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusBadGateway, gin.H{"error": "Failed to fetch contributions: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

// GetContributionsHeatmap returns the contribution heatmap for the authenticated user
// @Summary Get user contribution heatmap
// @Description Returns the contribution heatmap (contribution calendar) for the authenticated user. The heatmap shows daily contributions organized by weeks, similar to GitHub's contribution graph. If no period is specified, uses GitHub's default (last year). The provider parameter must match both the authenticated user's provider and be a valid provider configured in auth.yaml.
// @Tags github
// @Produce json
// @Param provider path string true "GitHub provider (must be configured in auth.yaml, e.g., 'githubtools', 'githubwdf')"
// @Param period query string false "Time period in days (e.g., '30d', '90d', '365d'). If omitted, uses GitHub's default period. Maximum: 365 days"
// @Success 200 {object} service.ContributionsHeatmapResponse
// @Failure 400 {object} ErrorResponse "Invalid period parameter or provider not configured"
// @Failure 401 {object} ErrorResponse "Unauthorized"
// @Failure 429 {object} ErrorResponse "Rate limit exceeded"
// @Failure 502 {object} ErrorResponse "GitHub API error"
// @Security BearerAuth
// @Router /github/{provider}/heatmap [get]
func (h *GitHubHandler) GetContributionsHeatmap(c *gin.Context) {
	claims := getAuthClaims(c)
	if claims == nil {
		return
	}

	// get GitHub provider from param 'provider'. TODO set 'githubtools' if not found. prepare to support multiple providers in future - which client currently doesn't support. should be mandatory.
	provider := c.DefaultQuery("provider", "githubtools")

	// Get query parameter for period (empty = use GitHub's default)
	period := c.Query("period")

	// Call service to get contribution heatmap
	response, err := h.service.GetContributionsHeatmap(c.Request.Context(), claims.UUID, provider, period)
	if err != nil {
		// Check for specific error types
		if errors.Is(err, apperrors.ErrGitHubAPIRateLimitExceeded) {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": err.Error()})
			return
		}
		// Check if it's a provider configuration error
		if errors.Is(err, apperrors.ErrProviderNotConfigured) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		// Check if it's a validation error (invalid period format)
		if errors.Is(err, apperrors.ErrInvalidPeriodFormat) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusBadGateway, gin.H{"error": "Failed to fetch contribution heatmap: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

// GetAveragePRMergeTime returns the average time to merge PRs for the authenticated user
// @Summary Get average PR merge time
// @Description Returns the average time to merge pull requests for the authenticated user over a specified period (default 30 days). The response includes both aggregate metrics (overall average and PR count) and a time series breakdown by date for visualization. The time is calculated as the duration between PR creation and merge (mergedAt - createdAt) in hours.
// @Tags github
// @Produce json
// @Param period query string false "Time period in days (e.g., '30d', '90d', '180d'). Default: '30d'"
// @Success 200 {object} service.AveragePRMergeTimeResponse
// @Failure 400 {object} ErrorResponse "Invalid period parameter"
// @Failure 401 {object} ErrorResponse "Unauthorized"
// @Failure 429 {object} ErrorResponse "Rate limit exceeded"
// @Failure 502 {object} ErrorResponse "GitHub API error"
// @Security BearerAuth
// @Router /github/average-pr-time [get]
func (h *GitHubHandler) GetAveragePRMergeTime(c *gin.Context) {
	claims := getAuthClaims(c)
	if claims == nil {
		return
	}

	// Get query parameter for period (default to 30d)
	period := c.DefaultQuery("period", "30d")

	// get GitHub provider from param 'provider'. TODO set 'githubtools' if not found. prepare to support multiple providers in future - which client currently doesn't support. should be mandatory.
	provider := c.DefaultQuery("provider", "githubtools")

	// Call service to get average PR merge time
	response, err := h.service.GetAveragePRMergeTime(c.Request.Context(), claims.UUID, provider, period)
	if err != nil {
		// Check for specific error types
		if errors.Is(err, apperrors.ErrGitHubAPIRateLimitExceeded) {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": err.Error()})
			return
		}
		// Check if it's a validation error (invalid period format)
		if errors.Is(err, apperrors.ErrInvalidPeriodFormat) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusBadGateway, gin.H{"error": "Failed to fetch average PR merge time: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

// GetRepositoryContent proxies GitHub repository content requests
// @Summary Get repository file or directory content
// @Description Proxies requests to GitHub API to fetch repository file or directory contents. Used by the documentation viewer.
// @Tags github
// @Produce json
// @Param owner path string true "Repository owner (organization or user)"
// @Param repo path string true "Repository name"
// @Param path path string false "Path to file or directory (can be empty for root)"
// @Param ref query string false "Git reference (branch, tag, or commit SHA)" default(main)
// @Success 200 {object} interface{} "GitHub API response (array for directories, object for files)"
// @Failure 401 {object} ErrorResponse "Unauthorized"
// @Failure 404 {object} ErrorResponse "Repository or path not found"
// @Failure 502 {object} ErrorResponse "GitHub API error"
// @Security BearerAuth
// @Router /github/repos/{owner}/{repo}/contents/{path} [get]
func (h *GitHubHandler) GetRepositoryContent(c *gin.Context) {
	claims := getAuthClaims(c)
	if claims == nil {
		return
	}

	// Get path parameters
	owner := c.Param("owner")
	repo := c.Param("repo")
	path := c.Param("path")
	ref := c.DefaultQuery("ref", "main")
	// get GitHub provider from param 'provider'. TODO set 'githubtools' if not found. prepare to support multiple providers in future - which client currently doesn't support. should be mandatory.
	provider := c.DefaultQuery("provider", "githubtools")

	// Call service to get repository content
	content, err := h.service.GetRepositoryContent(c.Request.Context(), claims.UUID, provider, owner, repo, path, ref)
	if err != nil {
		// Check for specific error types
		if errors.Is(err, apperrors.ErrGitHubAPIRateLimitExceeded) {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": err.Error()})
			return
		}
		if apperrors.IsNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusBadGateway, gin.H{"error": "Failed to fetch repository content: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, content)
}

// GetGitHubAsset proxies GitHub asset requests (images, etc.)
// @Summary Get GitHub asset (image, file, etc.)
// @Description Proxies requests to GitHub assets with authentication. Used by the documentation viewer for images.
// @Tags github
// @Produce octet-stream
// @Param url query string true "Full URL to the GitHub asset"
// @Success 200 {file} binary "Asset binary data"
// @Failure 401 {object} ErrorResponse "Unauthorized"
// @Failure 404 {object} ErrorResponse "Asset not found"
// @Failure 502 {object} ErrorResponse "GitHub API error"
// @Security BearerAuth
// @Router /github/asset [get]
func (h *GitHubHandler) GetGitHubAsset(c *gin.Context) {
	claims := getAuthClaims(c)
	if claims == nil {
		return
	}

	// Get asset URL from query parameter
	assetURL := c.Query("url")
	if assetURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Asset URL is required"})
		return
	}
	// get GitHub provider from param 'provider'. TODO set 'githubtools' if not found. prepare to support multiple providers in future - which client currently doesn't support. should be mandatory.
	provider := c.DefaultQuery("provider", "githubtools")

	// Call service to fetch the asset
	assetData, contentType, err := h.service.GetGitHubAsset(c.Request.Context(), claims.UUID, provider, assetURL)
	if err != nil {
		// Check for specific error types
		if errors.Is(err, apperrors.ErrGitHubAPIRateLimitExceeded) {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": err.Error()})
			return
		}
		if apperrors.IsNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusBadGateway, gin.H{"error": "Failed to fetch GitHub asset: " + err.Error()})
		return
	}

	// Set content type and return binary data
	c.Header("Content-Type", contentType)
	c.Header("Cache-Control", "public, max-age=3600") // Cache for 1 hour
	c.Data(http.StatusOK, contentType, assetData)
}

// UpdateRepositoryFileRequest represents the request body for updating a file
type UpdateRepositoryFileRequest struct {
	Message string `json:"message" binding:"required"`
	Content string `json:"content" binding:"required"`
	SHA     string `json:"sha" binding:"required"`
	Branch  string `json:"branch"`
}

// UpdateRepositoryFile updates a file in a GitHub repository
// @Summary Update repository file
// @Description Updates a file in a GitHub repository on behalf of the authenticated user
// @Tags github
// @Accept json
// @Produce json
// @Param owner path string true "Repository owner (organization or user)"
// @Param repo path string true "Repository name"
// @Param path path string true "Path to file"
// @Param body body UpdateRepositoryFileRequest true "Update file request"
// @Success 200 {object} interface{} "GitHub API response with commit details"
// @Failure 400 {object} ErrorResponse "Invalid request"
// @Failure 401 {object} ErrorResponse "Unauthorized"
// @Failure 404 {object} ErrorResponse "Repository or path not found"
// @Failure 502 {object} ErrorResponse "GitHub API error"
// @Security BearerAuth
// @Router /github/repos/{owner}/{repo}/contents/{path} [put]
func (h *GitHubHandler) UpdateRepositoryFile(c *gin.Context) {
	// Get authenticated user claims from context (set by auth middleware)
	claimsInterface, exists := c.Get("auth_claims")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	claims, ok := claimsInterface.(*auth.AuthClaims)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid authentication claims"})
		return
	}

	// Get path parameters
	owner := c.Param("owner")
	repo := c.Param("repo")
	path := c.Param("path")

	// Bind request body
	var req UpdateRepositoryFileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}
	// get GitHub provider from param 'provider'. TODO set 'githubtools' if not found. prepare to support multiple providers in future - which client currently doesn't support. should be mandatory.
	provider := c.DefaultQuery("provider", "githubtools")

	// Call service to update repository file
	response, err := h.service.UpdateRepositoryFile(c.Request.Context(), claims.UUID, provider, owner, repo, path, req.Message, req.Content, req.SHA, req.Branch)
	if err != nil {
		// Check for specific error types
		if errors.Is(err, apperrors.ErrGitHubAPIRateLimitExceeded) {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": err.Error()})
			return
		}
		if apperrors.IsNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusBadGateway, gin.H{"error": "Failed to update repository file: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

// ClosePullRequestRequest represents the request body for closing a PR
type ClosePullRequestRequest struct {
	Owner        string `json:"owner" binding:"required"`
	Repo         string `json:"repo" binding:"required"`
	DeleteBranch bool   `json:"delete_branch"`
}

// ClosePullRequest closes a pull request
// @Summary Close pull request
// @Description Closes an open PR for the authenticated user. Optionally deletes the PR branch when delete_branch is 'true'.
// @Tags github
// @Accept json
// @Produce json
// @Param pr_number path int true "Pull request number"
// @Param body body ClosePullRequestRequest true "Request body: owner, repo, delete_branch ('true' to delete the PR branch)"
// @Success 200 {object} service.PullRequest
// @Failure 400 {object} ErrorResponse "Invalid request"
// @Failure 401 {object} ErrorResponse "Unauthorized"
// @Failure 404 {object} ErrorResponse "Pull request not found"
// @Failure 429 {object} ErrorResponse "Rate limit exceeded"
// @Failure 502 {object} ErrorResponse "GitHub API error"
// @Security BearerAuth
// @Router /github/pull-requests/close/{pr_number} [patch]
func (h *GitHubHandler) ClosePullRequest(c *gin.Context) {
	claims := getAuthClaims(c)
	if claims == nil {
		return
	}

	// Parse PR number from path
	prNumberStr := c.Param("pr_number")
	prNumber, err := strconv.Atoi(prNumberStr)
	if err != nil || prNumber <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid pr_number parameter"})
		return
	}

	// Bind request body
	var req ClosePullRequestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	// Validate owner/repo
	if req.Owner == "" || req.Repo == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "owner and repo are required"})
		return
	}

	// Parse delete_branch flag
	deleteBranch := req.DeleteBranch

	// get GitHub provider from param 'provider'. TODO set 'githubtools' if not found. prepare to support multiple providers in future - which client currently doesn't support. should be mandatory.
	provider := c.DefaultQuery("provider", "githubtools")

	// Close PR (and optionally delete branch)
	updatedPR, err := h.service.ClosePullRequest(c.Request.Context(), claims.UUID, provider, req.Owner, req.Repo, prNumber, deleteBranch)
	if err != nil {
		// Check for specific error types
		if errors.Is(err, apperrors.ErrInvalidStatus) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if errors.Is(err, apperrors.ErrGitHubAPIRateLimitExceeded) {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": err.Error()})
			return
		}
		if apperrors.IsNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusBadGateway, gin.H{"error": "Failed to close pull request: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, updatedPR)
}

// GetPRReviewComments returns the total number of PR review comments by the authenticated user
// @Summary Get PR review comments count
// @Description Returns the total number of pull request reviews performed by the authenticated user over a specified period (default 30 days)
// @Tags github
// @Produce json
// @Param period query string false "Time period in days (e.g., '30d', '90d', '180d'). Default: '30d'"
// @Success 200 {object} service.PRReviewCommentsResponse
// @Failure 400 {object} ErrorResponse "Invalid period parameter"
// @Failure 401 {object} ErrorResponse "Unauthorized"
// @Failure 429 {object} ErrorResponse "Rate limit exceeded"
// @Failure 502 {object} ErrorResponse "GitHub API error"
// @Security BearerAuth
// @Router /github/pr-review-comments [get]
func (h *GitHubHandler) GetPRReviewComments(c *gin.Context) {
	claims := getAuthClaims(c)
	if claims == nil {
		return
	}

	// Get query parameter for period (default to 30d)
	period := c.DefaultQuery("period", "30d")
	// get GitHub provider from param 'provider'. TODO set 'githubtools' if not found. prepare to support multiple providers in future - which client currently doesn't support. should be mandatory.
	provider := c.DefaultQuery("provider", "githubtools")

	// Call service to get PR review comments count
	response, err := h.service.GetUserPRReviewComments(c.Request.Context(), claims.UUID, provider, period)
	if err != nil {
		// Check for specific error types
		if errors.Is(err, apperrors.ErrGitHubAPIRateLimitExceeded) {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": err.Error()})
			return
		}
		// Check if it's a validation error (invalid period format)
		if errors.Is(err, apperrors.ErrInvalidPeriodFormat) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusBadGateway, gin.H{"error": "Failed to fetch PR review comments: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}
