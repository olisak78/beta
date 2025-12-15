package service

import (
	"developer-portal-backend/internal/cache"
	"developer-portal-backend/internal/database/models"
	apperrors "developer-portal-backend/internal/errors"
	"developer-portal-backend/internal/repository"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// LandscapeService handles business logic for landscapes
type LandscapeService struct {
	repo             repository.LandscapeRepositoryInterface
	organizationRepo repository.OrganizationRepositoryInterface
	projectRepo      repository.ProjectRepositoryInterface
	validator        *validator.Validate
	cache            cache.CacheService
	ttlConfig        cache.TTLConfig
}

// NewLandscapeService creates a new landscape service
func NewLandscapeService(repo repository.LandscapeRepositoryInterface, orgRepo repository.OrganizationRepositoryInterface, projectRepo repository.ProjectRepositoryInterface, validator *validator.Validate) *LandscapeService {
	return &LandscapeService{
		repo:             repo,
		organizationRepo: orgRepo,
		projectRepo:      projectRepo,
		validator:        validator,
		cache:            cache.NewNoOpCache(), // Default to no-op cache
		ttlConfig:        cache.DefaultTTLConfig(),
	}
}

// NewLandscapeServiceWithCache creates a new landscape service with caching support
func NewLandscapeServiceWithCache(
	repo repository.LandscapeRepositoryInterface,
	orgRepo repository.OrganizationRepositoryInterface,
	projectRepo repository.ProjectRepositoryInterface,
	validator *validator.Validate,
	cacheService cache.CacheService,
	ttlConfig cache.TTLConfig,
) *LandscapeService {
	return &LandscapeService{
		repo:             repo,
		organizationRepo: orgRepo,
		projectRepo:      projectRepo,
		validator:        validator,
		cache:            cacheService,
		ttlConfig:        ttlConfig,
	}
}

// SetCache sets the cache service (useful for testing or late initialization)
func (s *LandscapeService) SetCache(cacheService cache.CacheService) {
	s.cache = cacheService
}

// SetTTLConfig sets the TTL configuration
func (s *LandscapeService) SetTTLConfig(config cache.TTLConfig) {
	s.ttlConfig = config
}

// CreateLandscapeRequest represents the request to create a landscape (new model)
type CreateLandscapeRequest struct {
	Name        string          `json:"name" validate:"required,min=1,max=40"`
	Title       string          `json:"title" validate:"required,min=1,max=100"`
	Description string          `json:"description,omitempty" validate:"max=200"`
	ProjectID   uuid.UUID       `json:"project_id" validate:"required"`
	Domain      string          `json:"domain" validate:"required,max=200"`
	Environment string          `json:"environment" validate:"required,max=20"`
	Metadata    json.RawMessage `json:"metadata,omitempty" swaggertype:"object"`
}

// UpdateLandscapeRequest represents the request to update a landscape (new model)
type UpdateLandscapeRequest struct {
	Title       string          `json:"title" validate:"required,min=1,max=100"`
	Description string          `json:"description,omitempty" validate:"max=200"`
	ProjectID   *uuid.UUID      `json:"project_id,omitempty"`
	Domain      string          `json:"domain,omitempty" validate:"max=200"`
	Environment string          `json:"environment,omitempty" validate:"max=20"`
	Metadata    json.RawMessage `json:"metadata,omitempty" swaggertype:"object"`
}

// LandscapeResponse represents the response for landscape operations (new model)
type LandscapeResponse struct {
	ID          uuid.UUID       `json:"id"`
	Name        string          `json:"name"`
	Title       string          `json:"title"`
	Description string          `json:"description"`
	ProjectID   uuid.UUID       `json:"project_id"`
	Domain      string          `json:"domain"`
	Environment string          `json:"environment"`
	Metadata    json.RawMessage `json:"metadata,omitempty" swaggertype:"object"`
	CreatedAt   string          `json:"created_at"`
	UpdatedAt   string          `json:"updated_at"`
}

// LandscapeMinimalResponse represents a trimmed landscape projection for list endpoints
type LandscapeMinimalResponse struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Title       string    `json:"title"`
	Description string    `json:"description"`

	Auditlog         string `json:"auditlog,omitempty"`
	Cam              string `json:"cam,omitempty"`
	Cockpit          string `json:"cockpit,omitempty"`
	Concourse        string `json:"concourse,omitempty"`
	ControlCenter    string `json:"control-center,omitempty"`
	Domain           string `json:"domain"`
	Dynatrace        string `json:"dynatrace,omitempty"`
	Environment      string `json:"environment"`
	Extension        bool   `json:"extension,omitempty"`
	Gardener         string `json:"gardener,omitempty"`
	Git              string `json:"git,omitempty"`
	Grafana          string `json:"grafana,omitempty"`
	Health           string `json:"health,omitempty"`
	IaasConsole      string `json:"iaas-console,omitempty"`
	IsCentralRegion  bool   `json:"is-central-region,omitempty"`
	Kibana           string `json:"kibana,omitempty"`
	Monitoring       string `json:"monitoring,omitempty"`
	OperationConsole string `json:"operation-console,omitempty"`
	Plutono          string `json:"plutono,omitempty"`
	Prometheus       string `json:"prometheus,omitempty"`
	Type             string `json:"type,omitempty"`
}

// LandscapeListResponse represents a paginated list of landscapes
type LandscapeListResponse struct {
	Landscapes []LandscapeResponse `json:"landscapes"`
	Total      int64               `json:"total"`
	Page       int                 `json:"page"`
	PageSize   int                 `json:"page_size"`
}

// CreateLandscape creates a new landscape
func (s *LandscapeService) CreateLandscape(req *CreateLandscapeRequest) (*LandscapeResponse, error) {
	// Validate request
	if err := s.validator.Struct(req); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Check if landscape with same name exists (global scope in new model)
	existingByName, err := s.repo.GetByName(req.Name)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("failed to check existing landscape by name: %w", err)
	}
	if existingByName != nil {
		return nil, apperrors.ErrLandscapeExists
	}

	// Create landscape (new model)
	landscape := &models.Landscape{
		BaseModel: models.BaseModel{
			Name:        req.Name,
			Title:       req.Title,
			Description: req.Description,
			Metadata:    req.Metadata,
		},
		ProjectID:   req.ProjectID,
		Domain:      req.Domain,
		Environment: req.Environment,
	}

	if err := s.repo.Create(landscape); err != nil {
		return nil, fmt.Errorf("failed to create landscape: %w", err)
	}

	// Invalidate relevant caches after creation
	s.invalidateLandscapeCaches(landscape)

	return s.toResponse(landscape), nil
}

// GetLandscapeByID retrieves a landscape by ID with caching
func (s *LandscapeService) GetLandscapeByID(id uuid.UUID) (*LandscapeResponse, error) {
	cacheKey := cache.BuildKey(cache.KeyPrefixLandscapeByID, id.String())

	// Create a cache wrapper for this operation
	wrapper := cache.NewCacheWrapper[*LandscapeResponse](s.cache)

	return wrapper.GetOrFetch(cacheKey, s.ttlConfig.LandscapeByID, func() (*LandscapeResponse, error) {
		landscape, err := s.repo.GetByID(id)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, apperrors.ErrLandscapeNotFound
			}
			return nil, fmt.Errorf("failed to get landscape: %w", err)
		}
		return s.toResponse(landscape), nil
	})
}

// GetByName retrieves a landscape by name with caching (organization scope not applicable in new model)
func (s *LandscapeService) GetByName(_ uuid.UUID, name string) (*LandscapeResponse, error) {
	cacheKey := cache.BuildKey(cache.KeyPrefixLandscapeByName, name)

	wrapper := cache.NewCacheWrapper[*LandscapeResponse](s.cache)

	return wrapper.GetOrFetch(cacheKey, s.ttlConfig.LandscapeByName, func() (*LandscapeResponse, error) {
		landscape, err := s.repo.GetByName(name)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, apperrors.ErrLandscapeNotFound
			}
			return nil, fmt.Errorf("failed to get landscape: %w", err)
		}
		return s.toResponse(landscape), nil
	})
}

// GetLandscapesByOrganization retrieves landscapes for an organization with pagination and caching
// Note: organization scope is not present in the new model; returns all landscapes paginated.
func (s *LandscapeService) GetLandscapesByOrganization(_ uuid.UUID, limit, offset int) ([]LandscapeResponse, int64, error) {
	if limit < 1 || limit > 100 {
		limit = 20
	}

	cacheKey := cache.BuildKey(cache.KeyPrefixLandscapeList, fmt.Sprintf("limit:%d:offset:%d", limit, offset))

	type cachedResult struct {
		Responses []LandscapeResponse `json:"responses"`
		Total     int64               `json:"total"`
	}

	wrapper := cache.NewCacheWrapper[cachedResult](s.cache)

	result, err := wrapper.GetOrFetch(cacheKey, s.ttlConfig.LandscapeList, func() (cachedResult, error) {
		landscapes, total, repoErr := s.repo.GetActiveLandscapes(limit, offset)
		if repoErr != nil {
			return cachedResult{}, fmt.Errorf("failed to get landscapes: %w", repoErr)
		}

		responses := make([]LandscapeResponse, len(landscapes))
		for i, landscape := range landscapes {
			responses[i] = *s.toResponse(&landscape)
		}

		return cachedResult{Responses: responses, Total: total}, nil
	})

	if err != nil {
		return nil, 0, err
	}

	return result.Responses, result.Total, nil
}

// GetByProjectName retrieves landscapes by project name with pagination and caching
func (s *LandscapeService) GetByProjectName(projectName string) (*LandscapeListResponse, error) {
	cacheKey := cache.BuildKey(cache.KeyPrefixLandscapeByProject, projectName)

	wrapper := cache.NewCacheWrapper[*LandscapeListResponse](s.cache)

	return wrapper.GetOrFetch(cacheKey, s.ttlConfig.LandscapeByProject, func() (*LandscapeListResponse, error) {
		// Get project by name
		project, err := s.projectRepo.GetByName(projectName)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, apperrors.ErrProjectNotFound
			}
			return nil, fmt.Errorf("failed to find project: %w", err)
		}

		// Check if project is nil
		if project == nil {
			return nil, apperrors.ErrProjectNotFound
		}

		// Get landscapes by project ID (using active landscapes query with project filter)
		landscapes, total, err := s.repo.GetLandscapesByProjectID(project.ID, 100, 0)
		if err != nil {
			return nil, fmt.Errorf("failed to get landscapes by project: %w", err)
		}

		responses := make([]LandscapeResponse, len(landscapes))
		for i, landscape := range landscapes {
			responses[i] = *s.toResponse(&landscape)
		}

		return &LandscapeListResponse{
			Landscapes: responses,
			Total:      total,
			Page:       1,
			PageSize:   100,
		}, nil
	})
}

// GetByProjectNameAll retrieves all landscapes by project name (minimal response) with caching
func (s *LandscapeService) GetByProjectNameAll(projectName string) ([]LandscapeMinimalResponse, error) {
	cacheKey := cache.BuildKey(cache.KeyPrefixLandscapeByProject, projectName, "all")

	wrapper := cache.NewCacheWrapper[[]LandscapeMinimalResponse](s.cache)

	return wrapper.GetOrFetch(cacheKey, s.ttlConfig.LandscapeByProject, func() ([]LandscapeMinimalResponse, error) {
		// Get project by name
		project, err := s.projectRepo.GetByName(projectName)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, apperrors.ErrProjectNotFound
			}
			return nil, fmt.Errorf("failed to find project: %w", err)
		}

		// Get all landscapes by project ID
		landscapes, _, err := s.repo.GetLandscapesByProjectID(project.ID, 1000, 0)
		if err != nil {
			return nil, fmt.Errorf("failed to get landscapes by project: %w", err)
		}

		responses := make([]LandscapeMinimalResponse, len(landscapes))
		for i, landscape := range landscapes {
			responses[i] = s.toMinimalResponse(&landscape)
		}

		return responses, nil
	})
}

// ListByQuery searches landscapes with filters and caching
func (s *LandscapeService) ListByQuery(q string, domains []string, environments []string, limit int, offset int) (*LandscapeListResponse, error) {
	// Validate limit to prevent divide by zero
	if limit < 1 {
		limit = 20
	}

	// Convert limit/offset to page/pageSize
	page := (offset / limit) + 1
	pageSize := limit

	// For now, use basic search (filters can be enhanced later)
	return s.Search(uuid.Nil, q, page, pageSize)
}

// Search searches landscapes by name, title, or description with caching (org ignored)
func (s *LandscapeService) Search(_ uuid.UUID, query string, page, pageSize int) (*LandscapeListResponse, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize
	cacheKey := cache.BuildKey(cache.KeyPrefixLandscapeSearch, fmt.Sprintf("q:%s:page:%d:size:%d", query, page, pageSize))

	wrapper := cache.NewCacheWrapper[*LandscapeListResponse](s.cache)

	return wrapper.GetOrFetch(cacheKey, s.ttlConfig.LandscapeSearch, func() (*LandscapeListResponse, error) {
		landscapes, total, err := s.repo.Search(uuid.Nil, query, pageSize, offset)
		if err != nil {
			return nil, fmt.Errorf("failed to search landscapes: %w", err)
		}

		responses := make([]LandscapeResponse, len(landscapes))
		for i, landscape := range landscapes {
			responses[i] = *s.toResponse(&landscape)
		}

		return &LandscapeListResponse{
			Landscapes: responses,
			Total:      total,
			Page:       page,
			PageSize:   pageSize,
		}, nil
	})
}

// UpdateLandscape updates a landscape
func (s *LandscapeService) UpdateLandscape(id uuid.UUID, req *UpdateLandscapeRequest) (*LandscapeResponse, error) {
	// Validate request
	if err := s.validator.Struct(req); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Get existing landscape
	landscape, err := s.repo.GetByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.ErrLandscapeNotFound
		}
		return nil, fmt.Errorf("failed to get landscape: %w", err)
	}

	// Update fields aligned with new model
	landscape.Title = req.Title
	landscape.Description = req.Description
	if req.ProjectID != nil {
		landscape.ProjectID = *req.ProjectID
	}
	if req.Domain != "" {
		landscape.Domain = req.Domain
	}
	if req.Environment != "" {
		landscape.Environment = req.Environment
	}
	if req.Metadata != nil {
		landscape.Metadata = req.Metadata
	}

	if err := s.repo.Update(landscape); err != nil {
		return nil, fmt.Errorf("failed to update landscape: %w", err)
	}

	// Invalidate caches after update
	s.invalidateLandscapeCaches(landscape)

	return s.toResponse(landscape), nil
}

// DeleteLandscape deletes a landscape
func (s *LandscapeService) DeleteLandscape(id uuid.UUID) error {
	// Check if landscape exists and get it for cache invalidation
	landscape, err := s.repo.GetByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperrors.ErrLandscapeNotFound
		}
		return fmt.Errorf("failed to get landscape: %w", err)
	}

	if err := s.repo.Delete(id); err != nil {
		return fmt.Errorf("failed to delete landscape: %w", err)
	}

	// Invalidate caches after deletion
	s.invalidateLandscapeCaches(landscape)

	return nil
}

// SetStatus sets the status of a landscape (no-op in new model; kept for API compatibility)
func (s *LandscapeService) SetStatus(id uuid.UUID, status string) error {
	// Check if landscape exists
	landscape, err := s.repo.GetByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperrors.ErrLandscapeNotFound
		}
		return fmt.Errorf("failed to get landscape: %w", err)
	}

	if err := s.repo.SetStatus(id, status); err != nil {
		return fmt.Errorf("failed to set landscape status: %w", err)
	}

	// Invalidate caches after status change
	s.invalidateLandscapeCaches(landscape)

	return nil
}

// GetWithOrganization retrieves a landscape with organization details (no org relation in new model)
func (s *LandscapeService) GetWithOrganization(id uuid.UUID) (*models.Landscape, error) {
	landscape, err := s.repo.GetByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.ErrLandscapeNotFound
		}
		return nil, fmt.Errorf("failed to get landscape: %w", err)
	}

	return landscape, nil
}

// invalidateLandscapeCaches invalidates all cache entries related to a landscape
func (s *LandscapeService) invalidateLandscapeCaches(landscape *models.Landscape) {
	// Invalidate by ID
	_ = s.cache.Delete(cache.BuildKey(cache.KeyPrefixLandscapeByID, landscape.ID.String()))

	// Invalidate by name
	_ = s.cache.Delete(cache.BuildKey(cache.KeyPrefixLandscapeByName, landscape.Name))

	// Clear list caches (they will be rebuilt on next request)
	// Note: For a more sophisticated implementation, you might want to track all
	// cache keys and invalidate them selectively
	s.cache.Clear()
}

// InvalidateAllCaches clears all landscape-related caches
func (s *LandscapeService) InvalidateAllCaches() {
	s.cache.Clear()
}

// toResponse converts a landscape model to response
func (s *LandscapeService) toResponse(landscape *models.Landscape) *LandscapeResponse {
	return &LandscapeResponse{
		ID:          landscape.ID,
		Name:        landscape.Name,
		Title:       landscape.Title,
		Description: landscape.Description,
		ProjectID:   landscape.ProjectID,
		Domain:      landscape.Domain,
		Environment: landscape.Environment,
		Metadata:    landscape.Metadata,
		CreatedAt:   landscape.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:   landscape.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
}

// toMinimalResponse converts a landscape model to minimal response with metadata enrichment
func (s *LandscapeService) toMinimalResponse(landscape *models.Landscape) LandscapeMinimalResponse {
	resp := LandscapeMinimalResponse{
		ID:          landscape.ID,
		Name:        landscape.Name,
		Title:       landscape.Title,
		Description: landscape.Description,
		Domain:      landscape.Domain,
		Environment: landscape.Environment,
	}

	// Enrich from metadata if present
	if len(landscape.Metadata) > 0 {
		var m map[string]interface{}
		if err := json.Unmarshal(landscape.Metadata, &m); err == nil {
			resp = s.enrichMinimalResponse(resp, m)
		}
	}

	return resp
}

// enrichMinimalResponse enriches minimal response with metadata fields
func (s *LandscapeService) enrichMinimalResponse(enr LandscapeMinimalResponse, m map[string]interface{}) LandscapeMinimalResponse {
	if auditlog, ok := m["auditlog"].(string); ok && auditlog != "" {
		enr.Auditlog = auditlog
	}
	if cam, ok := m["cam"].(string); ok && cam != "" {
		enr.Cam = cam
	}
	if cockpit, ok := m["cockpit"].(string); ok && cockpit != "" {
		enr.Cockpit = cockpit
	}
	if concourse, ok := m["concourse"].(string); ok && concourse != "" {
		enr.Concourse = concourse
	}
	if controlCenter, ok := m["control-center"].(string); ok && controlCenter != "" {
		enr.ControlCenter = controlCenter
	}
	if dynatrace, ok := m["dynatrace"].(string); ok && dynatrace != "" {
		enr.Dynatrace = dynatrace
	}
	if extension, ok := m["extension"].(bool); ok {
		enr.Extension = extension
	}
	if gardener, ok := m["gardener"].(string); ok && gardener != "" {
		enr.Gardener = gardener
	}
	if git, ok := m["git"].(string); ok && git != "" {
		enr.Git = git
	}
	if grafana, ok := m["grafana"].(string); ok && grafana != "" {
		enr.Grafana = grafana
	}
	if health, ok := m["health"].(string); ok && health != "" {
		enr.Health = health
	}
	if iaasConsole, ok := m["iaas-console"].(string); ok && iaasConsole != "" {
		enr.IaasConsole = iaasConsole
	}
	if isCentralRegion, ok := m["is-central-region"].(bool); ok {
		enr.IsCentralRegion = isCentralRegion
	}
	if kibana, ok := m["kibana"].(string); ok && kibana != "" {
		enr.Kibana = kibana
	}
	if monitoring, ok := m["monitoring"].(string); ok && monitoring != "" {
		enr.Monitoring = monitoring
	}
	if operationConsole, ok := m["operation-console"].(string); ok && operationConsole != "" {
		enr.OperationConsole = operationConsole
	}
	if plutono, ok := m["plutono"].(string); ok && plutono != "" {
		enr.Plutono = plutono
	}
	if prometheus, ok := m["prometheus"].(string); ok && prometheus != "" {
		enr.Prometheus = prometheus
	}
	if landscapeType, ok := m["type"].(string); ok && landscapeType != "" {
		enr.Type = landscapeType
	}

	return enr
}
