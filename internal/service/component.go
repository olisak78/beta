package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"developer-portal-backend/internal/cache"
	"developer-portal-backend/internal/database/models"
	apperrors "developer-portal-backend/internal/errors"
	"developer-portal-backend/internal/repository"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ComponentService handles business logic for components
type ComponentService struct {
	repo             *repository.ComponentRepository
	organizationRepo *repository.OrganizationRepository
	projectRepo      *repository.ProjectRepository
	validator        *validator.Validate
	cache            cache.CacheService
	cacheWrapper     *cache.CacheWrapper
	cacheTTL         time.Duration
}

// NewComponentService creates a new component service
func NewComponentService(repo *repository.ComponentRepository, orgRepo *repository.OrganizationRepository, projRepo *repository.ProjectRepository, validator *validator.Validate, cacheService cache.CacheService) *ComponentService {
	return &ComponentService{
		repo:             repo,
		organizationRepo: orgRepo,
		projectRepo:      projRepo,
		validator:        validator,
		cache:            cacheService,
		cacheWrapper:     cache.NewCacheWrapper(cacheService, 10*time.Minute),
		cacheTTL:         10 * time.Minute,
	}
}

// ComponentProjectView is a minimal view for /components?project-name=<name>
type ComponentProjectView struct {
	ID             uuid.UUID `json:"id"`
	OwnerID        uuid.UUID `json:"owner_id"`
	Name           string    `json:"name"`
	Title          string    `json:"title"`
	Description    string    `json:"description"`
	QOS            string    `json:"qos,omitempty"`
	Sonar          string    `json:"sonar,omitempty"`
	GitHub         string    `json:"github,omitempty"`
	CentralService *bool     `json:"central-service,omitempty"`
	IsLibrary      *bool     `json:"is-library,omitempty"`
	Health         *bool     `json:"health,omitempty"`
}

// GetByProjectNameAllView returns ALL components for a project (unpaginated) with a minimal view:
// - Omits project_id, created_at, updated_at, metadata
// - Adds fields: qos (metadata.ci.qos), sonar (metadata.sonar.project_id), github (metadata.github.url), central-service (metadata["central-service"]), is-library (metadata["isLibrary"])
func (s *ComponentService) GetByProjectNameAllView(projectName string) ([]ComponentProjectView, error) {
	if projectName == "" {
		return []ComponentProjectView{}, nil
	}

	cacheKey := fmt.Sprintf("components:project=%s:all", projectName)

	var views []ComponentProjectView
	err := s.cacheWrapper.GetOrSetTyped(cacheKey, s.cacheTTL, &views, func() (interface{}, error) {
		project, err := s.projectRepo.GetByName(projectName)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, apperrors.ErrProjectNotFound
			}
			return nil, fmt.Errorf("failed to resolve project by name: %w", err)
		}

		components, _, err := s.repo.GetComponentsByProjectID(project.ID, 1000000, 0)
		if err != nil {
			return nil, fmt.Errorf("failed to get components by project: %w", err)
		}

		componentViews := make([]ComponentProjectView, len(components))
		for i, c := range components {
			componentViews[i] = ComponentProjectView{
				ID:          c.ID,
				OwnerID:     c.OwnerID,
				Name:        c.Name,
				Title:       c.Title,
				Description: c.Description,
			}

			// Extract qos, sonar, github, central-service, is-library, health from metadata if present
			if len(c.Metadata) > 0 {
				var meta map[string]interface{}
				if err := json.Unmarshal(c.Metadata, &meta); err == nil {
					// qos from metadata.ci.qos
					if ciRaw, ok := meta["ci"]; ok {
						if ciMap, ok := ciRaw.(map[string]interface{}); ok {
							if qosRaw, ok := ciMap["qos"]; ok {
								if qosStr, ok := qosRaw.(string); ok {
									componentViews[i].QOS = qosStr
								}
							}
						}
					}
					// sonar from metadata.sonar.project_id
					if sonarRaw, ok := meta["sonar"]; ok {
						if sonarMap, ok := sonarRaw.(map[string]interface{}); ok {
							if pidRaw, ok := sonarMap["project_id"]; ok {
								if pidStr, ok := pidRaw.(string); ok {
									componentViews[i].Sonar = "https://sonar.tools.sap/dashboard?id=" + pidStr
								}
							}
						}
					}
					// github from metadata.github.url
					if ghRaw, ok := meta["github"]; ok {
						if ghMap, ok := ghRaw.(map[string]interface{}); ok {
							if urlRaw, ok := ghMap["url"]; ok {
								if urlStr, ok := urlRaw.(string); ok {
									componentViews[i].GitHub = urlStr
								}
							}
						}
					}
					// central-service from metadata["central-service"]
					if csRaw, ok := meta["central-service"]; ok {
						if csBool, ok := csRaw.(bool); ok {
							b := csBool
							componentViews[i].CentralService = &b
						}
					}
					// is-library from metadata["isLibrary"] (mapped to is-library)
					if ilRaw, ok := meta["isLibrary"]; ok {
						if ilBool, ok := ilRaw.(bool); ok {
							b := ilBool
							componentViews[i].IsLibrary = &b
						}
					}
					// health from metadata["health"] (boolean)
					if hRaw, ok := meta["health"]; ok {
						if hBool, ok := hRaw.(bool); ok {
							b := hBool
							componentViews[i].Health = &b
						}
					}
				}
			}
		}

		return componentViews, nil
	})

	if err != nil {
		return nil, err
	}

	return views, nil
}

// GetProjectTitleByID returns the project's title by ID
func (s *ComponentService) GetProjectTitleByID(id uuid.UUID) (string, error) {
	cacheKey := fmt.Sprintf("project:title:%s", id.String())

	var title string
	err := s.cacheWrapper.GetOrSetTyped(cacheKey, s.cacheTTL, &title, func() (interface{}, error) {
		project, err := s.projectRepo.GetByID(id)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, apperrors.ErrProjectNotFound
			}
			return nil, fmt.Errorf("failed to get project: %w", err)
		}
		return project.Title, nil
	})

	if err != nil {
		return "", err
	}

	return title, nil
}

// GetByID returns a component by its ID
func (s *ComponentService) GetByID(id uuid.UUID) (*models.Component, error) {
	cacheKey := fmt.Sprintf("component:id:%s", id.String())

	var component models.Component
	err := s.cacheWrapper.GetOrSetTyped(cacheKey, s.cacheTTL, &component, func() (interface{}, error) {
		comp, err := s.repo.GetByID(id)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, apperrors.ErrComponentNotFound
			}
			return nil, fmt.Errorf("failed to get component: %w", err)
		}
		return comp, nil
	})

	if err != nil {
		return nil, err
	}

	return &component, nil
}