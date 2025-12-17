package repository

import (
	"developer-portal-backend/internal/database/models"

	"github.com/google/uuid"
)

//go:generate mockgen -source=interfaces.go -destination=../mocks/repository_mocks.go -package=mocks

// OrganizationRepositoryInterface defines the interface for organization repository operations
type OrganizationRepositoryInterface interface {
	Create(org *models.Organization) error
	GetByID(id uuid.UUID) (*models.Organization, error)
	GetByName(name string) (*models.Organization, error)
	GetByDomain(domain string) (*models.Organization, error)
	GetAll(limit, offset int) ([]models.Organization, int64, error)
	Update(org *models.Organization) error
	Delete(id uuid.UUID) error
	GetWithMembers(id uuid.UUID) (*models.Organization, error)
	GetWithGroups(id uuid.UUID) (*models.Organization, error)
	GetWithProjects(id uuid.UUID) (*models.Organization, error)
	GetWithComponents(id uuid.UUID) (*models.Organization, error)
	GetWithLandscapes(id uuid.UUID) (*models.Organization, error)
	GetWithAllRelations(id uuid.UUID) (*models.Organization, error)
}

// UserRepositoryInterface defines the interface for member repository operations
type UserRepositoryInterface interface {
	Create(member *models.User) error
	GetByID(id uuid.UUID) (*models.User, error)
	GetByEmail(email string) (*models.User, error)
	GetByName(name string) (*models.User, error)
	GetByUserID(userID string) (*models.User, error)
	GetAll(limit, offset int) ([]models.User, int64, error)
	GetByOrganizationID(orgID uuid.UUID, limit, offset int) ([]models.User, int64, error)
	GetByTeamID(teamID uuid.UUID, limit, offset int) ([]models.User, int64, error)
	GetWithOrganization(id uuid.UUID) (*models.User, error)
	SearchByOrganization(orgID uuid.UUID, query string, limit, offset int) ([]models.User, int64, error)
	SearchByNameOrTitleGlobal(query string, limit, offset int) ([]models.User, int64, error)
	GetActiveByOrganization(orgID uuid.UUID, limit, offset int) ([]models.User, int64, error)
	GetUserIDsByPrefix(prefix string) ([]string, error)
	GetExistingUserIDs(ids []string) ([]string, error)
	Update(member *models.User) error
	Delete(id uuid.UUID) error
}

// GroupRepositoryInterface defines the interface for group repository operations
type GroupRepositoryInterface interface {
	Create(group *models.Group) error
	GetByID(id uuid.UUID) (*models.Group, error)
	GetByName(orgID uuid.UUID, name string) (*models.Group, error)
	GetByOrganizationID(orgID uuid.UUID, limit, offset int) ([]models.Group, int64, error)
	Search(organizationID uuid.UUID, query string, limit, offset int) ([]models.Group, int64, error)
	Update(id uuid.UUID, updates map[string]interface{}) error
	Delete(id uuid.UUID) error
	GetWithTeams(id uuid.UUID) (*models.Group, error)
	GetWithOrganization(id uuid.UUID) (*models.Group, error)
}

// TeamRepositoryInterface defines the interface for team repository operations
type TeamRepositoryInterface interface {
	Create(team *models.Team) error
	GetByID(id uuid.UUID) (*models.Team, error)
	GetByName(groupID uuid.UUID, name string) (*models.Team, error)
	GetByOrganizationID(orgID uuid.UUID, limit, offset int) ([]models.Team, int64, error)
	GetByGroupID(groupID uuid.UUID, limit, offset int) ([]models.Team, int64, error)
	GetAll() ([]models.Team, error)
	GetByNameGlobal(name string) (*models.Team, error)
	GetWithMembers(id uuid.UUID) (*models.Team, error)
	Update(team *models.Team) error
	Delete(id uuid.UUID) error
}

// ProjectRepositoryInterface defines the interface for project repository operations
type ProjectRepositoryInterface interface {
	Create(project *models.Project) error
	GetByID(id uuid.UUID) (*models.Project, error)
	GetHealthMetadata(projectID uuid.UUID) (string, string, error)
	GetByName(name string) (*models.Project, error)
	GetByOrganizationID(orgID uuid.UUID, limit, offset int) ([]models.Project, int64, error)
	Update(project *models.Project) error
	Delete(id uuid.UUID) error
}

// ComponentRepositoryInterface defines the interface for component repository operations
type ComponentRepositoryInterface interface {
	Create(component *models.Component) error
	GetByID(id uuid.UUID) (*models.Component, error)
	GetByName(projectID uuid.UUID, name string) (*models.Component, error)
	GetByProjectID(projectID uuid.UUID, limit, offset int) ([]models.Component, int64, error)
	GetByOwnerID(ownerID uuid.UUID, limit, offset int) ([]models.Component, int64, error)
	GetComponentsByTeamID(teamID uuid.UUID, limit, offset int) ([]models.Component, int64, error)
	Update(component *models.Component) error
	Delete(id uuid.UUID) error
}

// LandscapeRepositoryInterface defines the interface for landscape repository operations
type LandscapeRepositoryInterface interface {
	Create(landscape *models.Landscape) error
	GetByID(id uuid.UUID) (*models.Landscape, error)
	GetByName(name string) (*models.Landscape, error)
	GetByOrganizationID(orgID uuid.UUID, limit, offset int) ([]models.Landscape, int64, error)
	GetByType(orgID uuid.UUID, landscapeType string, limit, offset int) ([]models.Landscape, int64, error)
	GetByStatus(status string, limit, offset int) ([]models.Landscape, int64, error)
	GetActiveLandscapes(limit, offset int) ([]models.Landscape, int64, error)
	GetLandscapesByTypeAndStatus(orgID uuid.UUID, landscapeType string, status string, limit, offset int) ([]models.Landscape, int64, error)
	GetLandscapesByProjectID(projectID uuid.UUID, limit, offset int) ([]models.Landscape, int64, error)
	Search(orgID uuid.UUID, query string, limit, offset int) ([]models.Landscape, int64, error)
	Update(landscape *models.Landscape) error
	Delete(id uuid.UUID) error
	SetStatus(landscapeID uuid.UUID, status string) error
	GetWithOrganization(id uuid.UUID) (*models.Landscape, error)
	GetWithProjects(id uuid.UUID) (*models.Landscape, error)
	GetWithFullDetails(id uuid.UUID) (*models.Landscape, error)
}

// CategoryRepositoryInterface defines the interface for category repository operations
type CategoryRepositoryInterface interface {
	GetAll(limit, offset int) ([]models.Category, int64, error)
	GetByID(id uuid.UUID) (*models.Category, error)
}

// LinkRepositoryInterface defines the interface for link repository operations
type LinkRepositoryInterface interface {
	GetByOwner(owner uuid.UUID) ([]models.Link, error)
	GetByIDs(ids []uuid.UUID) ([]models.Link, error)
	Create(link *models.Link) error
	Delete(id uuid.UUID) error
	GetByID(id uuid.UUID) (*models.Link, error)
	Update(link *models.Link) error
}

// DocumentationRepositoryInterface defines the interface for documentation repository operations
type DocumentationRepositoryInterface interface {
	Create(doc *models.Documentation) error
	GetByID(id uuid.UUID) (*models.Documentation, error)
	GetByTeamID(teamID uuid.UUID) ([]models.Documentation, error)
	Update(doc *models.Documentation) error
	Delete(id uuid.UUID) error
	GetAll(limit, offset int) ([]models.Documentation, int64, error)
}

// PluginRepositoryInterface defines the interface for plugin repository operations
type PluginRepositoryInterface interface {
	Create(plugin *models.Plugin) error
	GetByID(id uuid.UUID) (*models.Plugin, error)
	GetByName(name string) (*models.Plugin, error)
	GetAll(limit, offset int) ([]models.Plugin, int64, error)
	Update(plugin *models.Plugin) error
	Delete(id uuid.UUID) error
}
