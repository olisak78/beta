package service

import (
	"encoding/json"
	"errors"
	"fmt"

	"developer-portal-backend/internal/database/models"
	apperrors "developer-portal-backend/internal/errors"
	"developer-portal-backend/internal/repository"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ComponentService handles business logic for components
type ComponentService struct {
	repo             repository.ComponentRepositoryInterface
	organizationRepo repository.OrganizationRepositoryInterface
	projectRepo      repository.ProjectRepositoryInterface
	validator        *validator.Validate
}

// NewComponentService creates a new component service
func NewComponentService(repo repository.ComponentRepositoryInterface, orgRepo repository.OrganizationRepositoryInterface, projRepo repository.ProjectRepositoryInterface, validator *validator.Validate) *ComponentService {
	return &ComponentService{
		repo:             repo,
		organizationRepo: orgRepo,
		projectRepo:      projRepo,
		validator:        validator,
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

	views := make([]ComponentProjectView, len(components))
	for i, c := range components {
		views[i] = ComponentProjectView{
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
								views[i].QOS = qosStr
							}
						}
					}
				}
				// sonar from metadata.sonar.project_id
				if sonarRaw, ok := meta["sonar"]; ok {
					if sonarMap, ok := sonarRaw.(map[string]interface{}); ok {
						if pidRaw, ok := sonarMap["project_id"]; ok {
							if pidStr, ok := pidRaw.(string); ok {
								views[i].Sonar = "https://sonar.tools.sap/dashboard?id=" + pidStr
							}
						}
					}
				}
				// github from metadata.github.url
				if ghRaw, ok := meta["github"]; ok {
					if ghMap, ok := ghRaw.(map[string]interface{}); ok {
						if urlRaw, ok := ghMap["url"]; ok {
							if urlStr, ok := urlRaw.(string); ok {
								views[i].GitHub = urlStr
							}
						}
					}
				}
				// central-service from metadata["central-service"]
				if csRaw, ok := meta["central-service"]; ok {
					if csBool, ok := csRaw.(bool); ok {
						b := csBool
						views[i].CentralService = &b
					}
				}
				// is-library from metadata["isLibrary"] (mapped to is-library)
				if ilRaw, ok := meta["isLibrary"]; ok {
					if ilBool, ok := ilRaw.(bool); ok {
						b := ilBool
						views[i].IsLibrary = &b
					}
				}
				// health from metadata["health"] (boolean)
				if hRaw, ok := meta["health"]; ok {
					if hBool, ok := hRaw.(bool); ok {
						b := hBool
						views[i].Health = &b
					}
				}
			}
		}
	}

	return views, nil
}

// GetProjectTitleByID returns the project's title by ID
func (s *ComponentService) GetProjectTitleByID(id uuid.UUID) (string, error) {
	project, err := s.projectRepo.GetByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", apperrors.ErrProjectNotFound
		}
		return "", fmt.Errorf("failed to get project: %w", err)
	}
	return project.Title, nil
}

// GetByID returns a component by its ID
func (s *ComponentService) GetByID(id uuid.UUID) (*models.Component, error) {
	component, err := s.repo.GetByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.ErrComponentNotFound
		}
		return nil, fmt.Errorf("failed to get component: %w", err)
	}
	return component, nil
}
