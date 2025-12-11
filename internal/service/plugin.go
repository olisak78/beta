package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"developer-portal-backend/internal/database/models"
	"developer-portal-backend/internal/repository"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

// PluginService handles business logic for plugins
type PluginService struct {
	pluginRepo repository.PluginRepositoryInterface
	userRepo   repository.UserRepositoryInterface
	validator  *validator.Validate
}

// NewPluginService creates a new plugin service
func NewPluginService(pluginRepo repository.PluginRepositoryInterface, userRepo repository.UserRepositoryInterface, validator *validator.Validate) *PluginService {
	return &PluginService{
		pluginRepo: pluginRepo,
		userRepo:   userRepo,
		validator:  validator,
	}
}

// PluginResponse represents the response structure for a plugin
type PluginResponse struct {
	ID                 uuid.UUID `json:"id"`
	Name               string    `json:"name"`
	Title              string    `json:"title"`
	Description        string    `json:"description"`
	Icon               string    `json:"icon"`
	ReactComponentPath string    `json:"react_component_path"`
	BackendServerURL   string    `json:"backend_server_url"`
	Owner              string    `json:"owner"`
	Subscribed         bool      `json:"subscribed,omitempty"`
}

// PluginListResponse represents the response structure for plugin list
type PluginListResponse struct {
	Plugins []PluginResponse `json:"plugins"`
	Total   int64            `json:"total"`
	Limit   int              `json:"limit"`
	Offset  int              `json:"offset"`
}

// CreatePluginRequest represents the request structure for creating a plugin
type CreatePluginRequest struct {
	Name               string `json:"name" validate:"required,min=1,max=40"`
	Title              string `json:"title" validate:"required,min=1,max=100"`
	Description        string `json:"description" validate:"max=200"`
	Icon               string `json:"icon" validate:"required,min=3,max=50"`
	ReactComponentPath string `json:"react_component_path" validate:"required,max=500"`
	BackendServerURL   string `json:"backend_server_url" validate:"required,max=500"`
	Owner              string `json:"owner" validate:"max=100"`
}

// UpdatePluginRequest represents the request structure for updating a plugin
type UpdatePluginRequest struct {
	Name               *string `json:"name,omitempty" validate:"omitempty,min=1,max=40"`
	Title              *string `json:"title,omitempty" validate:"omitempty,min=1,max=100"`
	Description        *string `json:"description,omitempty" validate:"omitempty,max=200"`
	Icon               *string `json:"icon,omitempty" validate:"omitempty,min=3,max=50"`
	ReactComponentPath *string `json:"react_component_path,omitempty" validate:"omitempty,max=500"`
	BackendServerURL   *string `json:"backend_server_url,omitempty" validate:"omitempty,max=500"`
	Owner              *string `json:"owner,omitempty" validate:"omitempty,max=100"`
}

// GetAllPlugins retrieves all plugins with pagination
func (s *PluginService) GetAllPlugins(limit, offset int) (*PluginListResponse, error) {
	// Set default pagination values
	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	plugins, total, err := s.pluginRepo.GetAll(limit, offset)
	if err != nil {
		return nil, err
	}

	pluginResponses := make([]PluginResponse, len(plugins))
	for i, plugin := range plugins {
		pluginResponses[i] = s.toPluginResponse(&plugin)
	}

	return &PluginListResponse{
		Plugins: pluginResponses,
		Total:   total,
		Limit:   limit,
		Offset:  offset,
	}, nil
}

// GetAllPluginsWithViewer retrieves all plugins with pagination and marks subscribed plugins based on viewer's subscriptions
func (s *PluginService) GetAllPluginsWithViewer(limit, offset int, viewerName string) (*PluginListResponse, error) {
	// Set default pagination values
	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	if strings.TrimSpace(viewerName) == "" {
		// Fallback to non-subscribed response if viewer missing
		return s.GetAllPlugins(limit, offset)
	}

	// Find viewer by name (mapped from bearer token 'username')
	viewer, err := s.userRepo.GetByName(viewerName)
	if err != nil || viewer == nil {
		// Fallback to non-subscribed response if viewer not found
		return s.GetAllPlugins(limit, offset)
	}

	// Parse subscribed plugins from viewer.Metadata
	subscribedSet := make(map[uuid.UUID]struct{})
	if len(viewer.Metadata) > 0 {
		var meta map[string]interface{}
		if err := json.Unmarshal(viewer.Metadata, &meta); err == nil && meta != nil {
			if v, ok := meta["subscribed"]; ok && v != nil {
				switch arr := v.(type) {
				case []interface{}:
					for _, it := range arr {
						if str, ok := it.(string); ok && str != "" {
							if id, err := uuid.Parse(strings.TrimSpace(str)); err == nil {
								subscribedSet[id] = struct{}{}
							}
						}
					}
				case []string:
					for _, s2 := range arr {
						if id, err := uuid.Parse(strings.TrimSpace(s2)); err == nil {
							subscribedSet[id] = struct{}{}
						}
					}
				}
			}
		}
	}

	// Fetch plugins
	plugins, total, err := s.pluginRepo.GetAll(limit, offset)
	if err != nil {
		return nil, err
	}

	// Map to response type and mark subscribed plugins
	pluginResponses := make([]PluginResponse, len(plugins))
	for i, plugin := range plugins {
		pr := s.toPluginResponse(&plugin)
		if _, ok := subscribedSet[plugin.ID]; ok {
			pr.Subscribed = true
		}
		pluginResponses[i] = pr
	}

	return &PluginListResponse{
		Plugins: pluginResponses,
		Total:   total,
		Limit:   limit,
		Offset:  offset,
	}, nil
}

// GetPluginByID retrieves a plugin by ID
func (s *PluginService) GetPluginByID(id uuid.UUID) (*PluginResponse, error) {
	plugin, err := s.pluginRepo.GetByID(id)
	if err != nil {
		return nil, err
	}

	response := s.toPluginResponse(plugin)
	return &response, nil
}

// CreatePlugin creates a new plugin
func (s *PluginService) CreatePlugin(req *CreatePluginRequest) (*PluginResponse, error) {
	// Validate the request
	if err := s.validator.Struct(req); err != nil {
		return nil, err
	}

	// Check if plugin with same name already exists
	existingPlugin, err := s.pluginRepo.GetByName(req.Name)
	if err == nil && existingPlugin != nil {
		return nil, &ValidationError{Message: "Plugin with this name already exists"}
	}

	// Create the plugin model
	plugin := &models.Plugin{
		BaseModel: models.BaseModel{
			Name:        req.Name,
			Title:       req.Title,
			Description: req.Description,
		},
		Icon:               req.Icon,
		ReactComponentPath: req.ReactComponentPath,
		BackendServerURL:   req.BackendServerURL,
		Owner:              req.Owner,
	}

	// Save to database
	if err := s.pluginRepo.Create(plugin); err != nil {
		return nil, err
	}

	// Return the response
	response := s.toPluginResponse(plugin)
	return &response, nil
}

// UpdatePlugin updates an existing plugin
func (s *PluginService) UpdatePlugin(id uuid.UUID, req *UpdatePluginRequest) (*PluginResponse, error) {
	// Validate the request
	if err := s.validator.Struct(req); err != nil {
		return nil, err
	}

	// Get the existing plugin
	plugin, err := s.pluginRepo.GetByID(id)
	if err != nil {
		return nil, err
	}

	// Update fields if provided
	if req.Name != nil {
		// Check if another plugin with same name already exists
		if *req.Name != plugin.Name {
			existingPlugin, err := s.pluginRepo.GetByName(*req.Name)
			if err == nil && existingPlugin != nil {
				return nil, &ValidationError{Message: "Plugin with this name already exists"}
			}
		}
		plugin.Name = *req.Name
	}
	if req.Title != nil {
		plugin.Title = *req.Title
	}
	if req.Description != nil {
		plugin.Description = *req.Description
	}
	if req.Icon != nil {
		plugin.Icon = *req.Icon
	}
	if req.ReactComponentPath != nil {
		plugin.ReactComponentPath = *req.ReactComponentPath
	}
	if req.BackendServerURL != nil {
		plugin.BackendServerURL = *req.BackendServerURL
	}
	if req.Owner != nil {
		plugin.Owner = *req.Owner
	}

	// Save to database
	if err := s.pluginRepo.Update(plugin); err != nil {
		return nil, err
	}

	// Return the response
	response := s.toPluginResponse(plugin)
	return &response, nil
}

// DeletePlugin deletes a plugin by ID
func (s *PluginService) DeletePlugin(id uuid.UUID) error {
	// Check if plugin exists
	_, err := s.pluginRepo.GetByID(id)
	if err != nil {
		return err
	}

	// Delete the plugin
	return s.pluginRepo.Delete(id)
}

// toPluginResponse converts a plugin model to response format
func (s *PluginService) toPluginResponse(plugin *models.Plugin) PluginResponse {
	return PluginResponse{
		ID:                 plugin.ID,
		Name:               plugin.Name,
		Title:              plugin.Title,
		Description:        plugin.Description,
		Icon:               plugin.Icon,
		ReactComponentPath: plugin.ReactComponentPath,
		BackendServerURL:   plugin.BackendServerURL,
		Owner:              plugin.Owner,
	}
}

// ValidationError represents a validation error
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}

// PluginUIResponse represents the response structure for plugin UI content
type PluginUIResponse struct {
	Content     string `json:"content"`
	ContentType string `json:"content_type"`
}

// GetPluginUIContent fetches the TSX React component content from GitHub
func (s *PluginService) GetPluginUIContent(ctx context.Context, pluginID uuid.UUID, githubService GitHubServiceInterface, userUUID, provider string) (*PluginUIResponse, error) {
	// Get the plugin by ID
	plugin, err := s.pluginRepo.GetByID(pluginID)
	if err != nil {
		return nil, err
	}

	// Parse the GitHub URL from react_component_path
	githubURL := plugin.ReactComponentPath
	if githubURL == "" {
		return nil, fmt.Errorf("plugin does not have a react_component_path configured")
	}

	// Parse GitHub URL to extract owner, repo, and file path
	owner, repo, filePath, ref, err := parsePluginGitHubURL(githubURL)
	if err != nil {
		return nil, fmt.Errorf("invalid GitHub URL in react_component_path: %w", err)
	}

	// Use default provider if not provided
	if provider == "" {
		provider = "github"
	}

	// Fetch the file content from GitHub
	content, err := githubService.GetRepositoryContent(ctx, userUUID, provider, owner, repo, filePath, ref)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch content from GitHub: %w", err)
	}

	// Extract content from the GitHub API response
	var fileContent string

	if contentMap, ok := content.(map[string]interface{}); ok {
		if contentStr, exists := contentMap["content"]; exists {
			fileContent = contentStr.(string)
		}
	}

	if fileContent == "" {
		return nil, fmt.Errorf("no content found in GitHub response")
	}

	return &PluginUIResponse{
		Content:     fileContent,
		ContentType: "text/typescript",
	}, nil
}

// parsePluginGitHubURL parses a GitHub URL and extracts owner, repo, file path, and ref
// Supports URLs like:
// - https://github.com/owner/repo/blob/main/path/to/file.tsx
// - https://github.tools.sap/owner/repo/blob/main/path/to/file.tsx
func parsePluginGitHubURL(githubURL string) (owner, repo, filePath, ref string, err error) {
	// Parse the URL
	parsedURL, err := url.Parse(githubURL)
	if err != nil {
		return "", "", "", "", fmt.Errorf("invalid URL: %w", err)
	}

	// Check if it's a GitHub URL
	if !strings.Contains(parsedURL.Host, "github") {
		return "", "", "", "", fmt.Errorf("not a GitHub URL")
	}

	// Use regex to parse GitHub blob URL pattern
	// Pattern: /owner/repo/blob/ref/path/to/file
	pattern := regexp.MustCompile(`^/([^/]+)/([^/]+)/blob/([^/]+)/(.+)$`)
	matches := pattern.FindStringSubmatch(parsedURL.Path)

	if len(matches) != 5 {
		return "", "", "", "", fmt.Errorf("invalid GitHub blob URL format")
	}

	owner = matches[1]
	repo = matches[2]
	ref = matches[3]
	filePath = matches[4]

	return owner, repo, filePath, ref, nil
}
