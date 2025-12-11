package handlers

import (
	"developer-portal-backend/internal/auth"
	"developer-portal-backend/internal/service"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// pluginProxyCache represents a cached response with expiration
type pluginProxyCache struct {
	response  map[string]interface{}
	expiresAt time.Time
}

// PluginHandler handles HTTP requests for plugins
type PluginHandler struct {
	pluginService service.PluginServiceInterface
	githubService service.GitHubServiceInterface
	proxyCache    map[string]*pluginProxyCache
	cacheMux      sync.RWMutex
}

// NewPluginHandler creates a new plugin handler
func NewPluginHandler(pluginService service.PluginServiceInterface) *PluginHandler {
	return &PluginHandler{
		pluginService: pluginService,
		proxyCache:    make(map[string]*pluginProxyCache),
	}
}

// NewPluginHandlerWithGitHub creates a new plugin handler with GitHub service
func NewPluginHandlerWithGitHub(pluginService service.PluginServiceInterface, githubService service.GitHubServiceInterface) *PluginHandler {
	return &PluginHandler{
		pluginService: pluginService,
		githubService: githubService,
		proxyCache:    make(map[string]*pluginProxyCache),
	}
}

// GetAllPlugins handles GET /plugins
// @Summary Get all plugins or only subscribed plugins
// @Description Retrieve all plugins with pagination. When subscribed=true, returns only plugins the authenticated user is subscribed to. When subscribed=false or omitted, returns all plugins with subscription status marked for authenticated users.
// @Tags plugins
// @Accept json
// @Produce json
// @Param limit query int false "Number of items to return" default(20)
// @Param offset query int false "Number of items to skip" default(0)
// @Param subscribed query bool false "When true, return only subscribed plugins. When false, return all plugins with subscription status." default(false)
// @Success 200 {object} service.PluginListResponse "Successfully retrieved plugins list"
// @Failure 400 {object} map[string]interface{} "Invalid parameters"
// @Failure 401 {object} map[string]interface{} "Authentication required when subscribed=true"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Security BearerAuth
// @Router /plugins [get]
func (h *PluginHandler) GetAllPlugins(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	subscribed := c.DefaultQuery("subscribed", "false") == "true"

	// Validate pagination parameters
	if limit < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "limit must be non-negative"})
		return
	}
	if offset < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "offset must be non-negative"})
		return
	}

	var plugins *service.PluginListResponse
	var err error

	if subscribed {
		// Extract user claims from authentication context
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

		// Get all plugins with subscription status, then filter to only subscribed ones
		allPlugins, err := h.pluginService.GetAllPluginsWithViewer(limit, offset, claims.Username)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve plugins", "details": err.Error()})
			return
		}

		// Filter to only subscribed plugins
		subscribedPlugins := make([]service.PluginResponse, 0)
		for _, plugin := range allPlugins.Plugins {
			if plugin.Subscribed {
				subscribedPlugins = append(subscribedPlugins, plugin)
			}
		}

		// Create filtered response
		plugins = &service.PluginListResponse{
			Plugins: subscribedPlugins,
			Total:   int64(len(subscribedPlugins)),
			Limit:   allPlugins.Limit,
			Offset:  allPlugins.Offset,
		}
	} else {
		// Use the viewer method to include subscription status for all plugins
		claimsInterface, exists := c.Get("auth_claims")
		if exists {
			if claims, ok := claimsInterface.(*auth.AuthClaims); ok {
				// Include subscription status for authenticated users
				plugins, err = h.pluginService.GetAllPluginsWithViewer(limit, offset, claims.Username)
			} else {
				// Fallback to standard method if claims are invalid
				plugins, err = h.pluginService.GetAllPlugins(limit, offset)
			}
		} else {
			// Use standard method for unauthenticated requests
			plugins, err = h.pluginService.GetAllPlugins(limit, offset)
		}
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve plugins", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, plugins)
}

// GetPluginByID handles GET /plugins/{id}
// @Summary Get plugin by ID
// @Description Retrieve a specific plugin by its ID
// @Tags plugins
// @Accept json
// @Produce json
// @Param id path string true "Plugin ID (UUID)"
// @Success 200 {object} service.PluginResponse "Successfully retrieved plugin"
// @Failure 400 {object} map[string]interface{} "Invalid plugin ID"
// @Failure 404 {object} map[string]interface{} "Plugin not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Security BearerAuth
// @Router /plugins/{id} [get]
func (h *PluginHandler) GetPluginByID(c *gin.Context) {
	idStr := c.Param("id")
	if idStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "plugin ID is required"})
		return
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid plugin ID format"})
		return
	}

	// Always use the standard method (subscription logic removed)
	plugin, err := h.pluginService.GetPluginByID(id)

	if err != nil {
		// Check if it's a "not found" error (GORM returns specific error types)
		if err.Error() == "record not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Plugin not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve plugin", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, plugin)
}

// CreatePlugin handles POST /plugins
// @Summary Create a new plugin
// @Description Create a new plugin with the provided information
// @Tags plugins
// @Accept json
// @Produce json
// @Param plugin body service.CreatePluginRequest true "Plugin creation request"
// @Success 201 {object} service.PluginResponse "Successfully created plugin"
// @Failure 400 {object} map[string]interface{} "Invalid request body or validation error"
// @Failure 409 {object} map[string]interface{} "Plugin with same name already exists"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Security BearerAuth
// @Router /plugins [post]
func (h *PluginHandler) CreatePlugin(c *gin.Context) {
	var req service.CreatePluginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	plugin, err := h.pluginService.CreatePlugin(&req)
	if err != nil {
		// Check if it's a validation error
		if validationErr, ok := err.(*service.ValidationError); ok {
			c.JSON(http.StatusConflict, gin.H{"error": validationErr.Message})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create plugin", "details": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, plugin)
}

// UpdatePlugin handles PUT /plugins/{id}
// @Summary Update a plugin
// @Description Update an existing plugin with the provided information
// @Tags plugins
// @Accept json
// @Produce json
// @Param id path string true "Plugin ID (UUID)"
// @Param plugin body service.UpdatePluginRequest true "Plugin update request"
// @Success 200 {object} service.PluginResponse "Successfully updated plugin"
// @Failure 400 {object} map[string]interface{} "Invalid plugin ID or request body"
// @Failure 404 {object} map[string]interface{} "Plugin not found"
// @Failure 409 {object} map[string]interface{} "Plugin with same name already exists"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Security BearerAuth
// @Router /plugins/{id} [put]
func (h *PluginHandler) UpdatePlugin(c *gin.Context) {
	idStr := c.Param("id")
	if idStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "plugin ID is required"})
		return
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid plugin ID format"})
		return
	}

	var req service.UpdatePluginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	plugin, err := h.pluginService.UpdatePlugin(id, &req)
	if err != nil {
		// Check if it's a "not found" error
		if err.Error() == "record not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Plugin not found"})
			return
		}
		// Check if it's a validation error
		if validationErr, ok := err.(*service.ValidationError); ok {
			c.JSON(http.StatusConflict, gin.H{"error": validationErr.Message})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update plugin", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, plugin)
}

// DeletePlugin handles DELETE /plugins/{id}
// @Summary Delete a plugin
// @Description Delete a plugin by its ID
// @Tags plugins
// @Accept json
// @Produce json
// @Param id path string true "Plugin ID (UUID)"
// @Success 204 "Successfully deleted plugin"
// @Failure 400 {object} map[string]interface{} "Invalid plugin ID"
// @Failure 404 {object} map[string]interface{} "Plugin not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Security BearerAuth
// @Router /plugins/{id} [delete]
func (h *PluginHandler) DeletePlugin(c *gin.Context) {
	idStr := c.Param("id")
	if idStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "plugin ID is required"})
		return
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid plugin ID format"})
		return
	}

	err = h.pluginService.DeletePlugin(id)
	if err != nil {
		// Check if it's a "not found" error
		if err.Error() == "record not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Plugin not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete plugin", "details": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// GetPluginUI handles GET /plugins/{id}/ui
// @Summary Get plugin UI content
// @Description Retrieve the TSX React component content from GitHub for a specific plugin
// @Tags plugins
// @Accept json
// @Produce json
// @Param id path string true "Plugin ID (UUID)"
// @Success 200 {object} service.PluginUIResponse "Successfully retrieved plugin UI content"
// @Failure 400 {object} map[string]interface{} "Invalid plugin ID"
// @Failure 404 {object} map[string]interface{} "Plugin not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Security BearerAuth
// @Router /plugins/{id}/ui [get]
func (h *PluginHandler) GetPluginUI(c *gin.Context) {
	idStr := c.Param("id")
	if idStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "plugin ID is required"})
		return
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid plugin ID format"})
		return
	}

	// Check if GitHub service is available
	if h.githubService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "GitHub service not available"})
		return
	}

	// Extract user claims from authentication context
	// This follows the same pattern as other GitHub endpoints in the codebase
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

	// Default to "githubtools" provider - this could be made configurable
	provider := c.DefaultQuery("provider", "githubtools")

	// Get plugin UI content
	uiContent, err := h.pluginService.GetPluginUIContent(c.Request.Context(), id, h.githubService, claims.UUID, provider)
	if err != nil {
		// Check if it's a "not found" error
		if err.Error() == "record not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Plugin not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve plugin UI content", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, uiContent)
}

// ProxyPluginBackend handles GET /plugins/{id}/proxy?path={targetPath}
// @Summary Proxy requests to plugin backend
// @Description Proxy requests to plugin backend server with 30-second timeout, caching (5 minutes for successful GET responses), and standardized response format. Always returns 200 OK with metadata including actual backend status, response time, and success flag.
// @Tags plugins
// @Accept json
// @Produce json
// @Param id path string true "Plugin ID (UUID)" format(uuid)
// @Param path query string true "Target path to proxy to plugin backend (e.g., /api/health, /api/status)"
// @Success 200 {object} object{data=object,responseTime=int,statusCode=int,pluginSuccess=bool,error=string} "Proxy response with metadata - always returns 200 OK regardless of backend status"
// @Failure 400 {object} ErrorResponse "Invalid plugin ID format or missing path parameter"
// @Failure 404 {object} ErrorResponse "Plugin not found"
// @Failure 500 {object} ErrorResponse "Internal server error (e.g., database connection issues)"
// @Security BearerAuth
// @Router /plugins/{id}/proxy [get]
func (h *PluginHandler) ProxyPluginBackend(c *gin.Context) {
	idStr := c.Param("id")
	if idStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "plugin ID is required"})
		return
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid plugin ID format"})
		return
	}

	targetPath := c.Query("path")
	if targetPath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "path parameter is required"})
		return
	}

	// Create cache key from plugin ID and target path
	cacheKey := fmt.Sprintf("%s:%s", id.String(), targetPath)

	// Check cache first (only for GET requests)
	if c.Request.Method == "GET" {
		h.cacheMux.RLock()
		if cached, exists := h.proxyCache[cacheKey]; exists {
			if time.Now().Before(cached.expiresAt) {
				h.cacheMux.RUnlock()
				log.Printf("Cache hit for plugin proxy: %s", cacheKey)
				c.JSON(http.StatusOK, cached.response)
				return
			}
		}
		h.cacheMux.RUnlock()
	}

	// Get the plugin to retrieve backend server URL
	plugin, err := h.pluginService.GetPluginByID(id)
	if err != nil {
		if err.Error() == "record not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Plugin not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve plugin", "details": err.Error()})
		return
	}

	// Construct the target URL
	backendURL := plugin.BackendServerURL
	if !strings.HasSuffix(backendURL, "/") && !strings.HasPrefix(targetPath, "/") {
		backendURL += "/"
	}
	targetURL := backendURL + targetPath

	// Log the request
	log.Printf("Proxying request to plugin backend: %s -> %s", c.Request.URL.String(), targetURL)

	// Create HTTP client with 30-second timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Make request to plugin backend
	startTime := time.Now()
	resp, err := client.Get(targetURL)
	responseTime := time.Since(startTime).Milliseconds()

	// Prepare response structure
	response := map[string]interface{}{
		"responseTime": responseTime,
	}

	if err != nil {
		response["statusCode"] = 200
		response["pluginSuccess"] = false
		response["error"] = fmt.Sprintf("Failed to fetch from plugin backend: %s", err.Error())

		log.Printf("Plugin backend request failed: %s - %v", targetURL, err)
		c.JSON(http.StatusOK, response)
		return
	}
	defer resp.Body.Close()

	// Read response body
	var result interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		response["statusCode"] = resp.StatusCode
		response["pluginSuccess"] = false
		response["error"] = "Invalid JSON response from plugin backend"

		log.Printf("Plugin backend returned invalid JSON: %s - %v", targetURL, err)
		c.JSON(http.StatusOK, response)
		return
	}

	// Set response data
	response["data"] = result
	response["statusCode"] = resp.StatusCode
	response["pluginSuccess"] = resp.StatusCode >= 200 && resp.StatusCode < 300

	if !response["pluginSuccess"].(bool) {
		response["error"] = fmt.Sprintf("Plugin backend returned error status: %d", resp.StatusCode)
		log.Printf("Plugin backend returned error status: %s - %d", targetURL, resp.StatusCode)
	} else {
		log.Printf("Plugin backend request successful: %s - %d (%dms)", targetURL, resp.StatusCode, responseTime)

		// Cache successful GET responses for 5 minutes
		if c.Request.Method == "GET" {
			h.cacheMux.Lock()
			h.proxyCache[cacheKey] = &pluginProxyCache{
				response:  response,
				expiresAt: time.Now().Add(5 * time.Minute),
			}
			h.cacheMux.Unlock()
			log.Printf("Cached plugin proxy response: %s", cacheKey)
		}
	}

	// Always return 200 OK with metadata
	c.JSON(http.StatusOK, response)
}
