package repository

import (
	"developer-portal-backend/internal/database/models"
	"encoding/json"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ProjectRepository handles database operations for projects
type ProjectRepository struct {
	db *gorm.DB
}

// NewProjectRepository creates a new project repository
func NewProjectRepository(db *gorm.DB) *ProjectRepository {
	return &ProjectRepository{db: db}
}

// Create creates a new project
func (r *ProjectRepository) Create(project *models.Project) error {
	return r.db.Create(project).Error
}

// GetByID retrieves a project by ID
func (r *ProjectRepository) GetByID(id uuid.UUID) (*models.Project, error) {
	var project models.Project
	err := r.db.First(&project, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &project, nil
}

// GetByName retrieves a project by name within an organization
func (r *ProjectRepository) GetByName(name string) (*models.Project, error) {
	var project models.Project
	err := r.db.First(&project, "name = ?", name).Error
	if err != nil {
		return nil, err
	}
	return &project, nil
}

// GetByOrganizationID retrieves all projects for an organization with pagination
func (r *ProjectRepository) GetByOrganizationID(orgID uuid.UUID, limit, offset int) ([]models.Project, int64, error) {
	var projects []models.Project
	var total int64

	// Get total count
	if err := r.db.Model(&models.Project{}).Where("organization_id = ?", orgID).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Get paginated results
	err := r.db.Where("organization_id = ?", orgID).Limit(limit).Offset(offset).Find(&projects).Error
	if err != nil {
		return nil, 0, err
	}

	return projects, total, nil
}

// GetByStatus retrieves all projects with a specific status in an organization

// GetActiveProjects retrieves all active projects for an organization

// Update updates a project
func (r *ProjectRepository) Update(project *models.Project) error {
	return r.db.Save(project).Error
}

// Delete deletes a project
func (r *ProjectRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&models.Project{}, "id = ?", id).Error
}

// GetWithOrganization retrieves a project with organization details
func (r *ProjectRepository) GetWithOrganization(id uuid.UUID) (*models.Project, error) {
	var project models.Project
	err := r.db.Preload("Organization").First(&project, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &project, nil
}

// GetWithComponents retrieves a project with all its components
func (r *ProjectRepository) GetWithComponents(id uuid.UUID) (*models.Project, error) {
	var project models.Project
	err := r.db.Preload("Components").First(&project, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &project, nil
}

// GetWithLandscapes retrieves a project with all its landscapes
func (r *ProjectRepository) GetWithLandscapes(id uuid.UUID) (*models.Project, error) {
	var project models.Project
	err := r.db.Preload("Landscapes").First(&project, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &project, nil
}

// GetWithComponentsAndLandscapes retrieves a project with components and landscapes
func (r *ProjectRepository) GetWithComponentsAndLandscapes(id uuid.UUID) (*models.Project, error) {
	var project models.Project
	err := r.db.Preload("Components").Preload("Landscapes").First(&project, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &project, nil
}

// GetWithDeploymentTimelines retrieves a project with deployment timelines
func (r *ProjectRepository) GetWithDeploymentTimelines(id uuid.UUID) (*models.Project, error) {
	var project models.Project
	err := r.db.Preload("DeploymentTimelines").First(&project, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &project, nil
}

// GetWithFullDetails retrieves a project with all relationships
func (r *ProjectRepository) GetWithFullDetails(id uuid.UUID) (*models.Project, error) {
	var project models.Project
	err := r.db.
		Preload("Organization").
		Preload("Components").
		Preload("Landscapes").
		Preload("DeploymentTimelines").
		First(&project, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &project, nil
}

// SetStatus sets the status of a project

// Search searches for projects by name or description
func (r *ProjectRepository) Search(orgID uuid.UUID, query string, limit, offset int) ([]models.Project, int64, error) {
	var projects []models.Project
	var total int64

	searchQuery := r.db.Model(&models.Project{}).Where("organization_id = ? AND (name ILIKE ? OR description ILIKE ?)", orgID, "%"+query+"%", "%"+query+"%")

	// Get total count
	if err := searchQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Get paginated results
	err := searchQuery.Limit(limit).Offset(offset).Find(&projects).Error
	if err != nil {
		return nil, 0, err
	}

	return projects, total, nil
}

// GetProjectsWithCounts retrieves projects with their component and landscape counts
func (r *ProjectRepository) GetProjectsWithCounts(orgID uuid.UUID, limit, offset int) ([]map[string]interface{}, int64, error) {
	var projects []models.Project
	var total int64
	var results []map[string]interface{}

	// Get total count
	if err := r.db.Model(&models.Project{}).Where("organization_id = ?", orgID).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Get projects with counts
	err := r.db.Raw(`
		SELECT p.*, 
			COUNT(DISTINCT pc.component_id) as component_count,
			COUNT(DISTINCT pl.landscape_id) as landscape_count
		FROM projects p
		LEFT JOIN project_components pc ON p.id = pc.project_id
		LEFT JOIN project_landscapes pl ON p.id = pl.project_id
		WHERE p.organization_id = ?
		GROUP BY p.id
		ORDER BY p.created_at DESC
		LIMIT ? OFFSET ?
	`, orgID, limit, offset).Scan(&projects).Error

	if err != nil {
		return nil, 0, err
	}

	// Convert to map format for easier JSON handling
	for _, project := range projects {
		projectMap := map[string]interface{}{
			"id":           project.ID,
			"name":         project.Name,
			"display_name": project.Title,
			"description":  project.Description,
			"created_at":   project.CreatedAt,
			"updated_at":   project.UpdatedAt,
		}
		results = append(results, projectMap)
	}

	return results, total, nil
}

// CheckProjectExists checks if a project exists by ID
func (r *ProjectRepository) CheckProjectExists(id uuid.UUID) (bool, error) {
	var count int64
	err := r.db.Model(&models.Project{}).Where("id = ?", id).Count(&count).Error
	return count > 0, err
}

// GetHealthMetadata returns the project's health URL template and success regex from metadata.
// If the project is not found, returns an error.
// If metadata or the fields are missing or not strings, returns empty strings with nil error.
func (r *ProjectRepository) GetHealthMetadata(projectID uuid.UUID) (string, string, error) {
	var project models.Project
	if err := r.db.Select("id", "metadata").First(&project, "id = ?", projectID).Error; err != nil {
		return "", "", err
	}
	// Handle null or empty metadata
	if len(project.Metadata) == 0 || string(project.Metadata) == "null" {
		return "", "", nil
	}
	var meta map[string]interface{}
	if err := json.Unmarshal(project.Metadata, &meta); err != nil {
		return "", "", err
	}
	// URL template from metadata["health"]
	urlTemplate := ""
	if raw, ok := meta["health"]; ok {
		if s, ok := raw.(string); ok {
			urlTemplate = s
		}
	}
	// Success regex from metadata["health_success_regex"]
	successRegex := ""
	if raw, ok := meta["health_success_regex"]; ok {
		if s, ok := raw.(string); ok {
			successRegex = s
		}
	}
	return urlTemplate, successRegex, nil
}

// GetAllProjects retrieves all projects
func (r *ProjectRepository) GetAllProjects() ([]models.Project, error) {
	var projects []models.Project
	err := r.db.Find(&projects).Error
	if err != nil {
		return nil, err
	}
	return projects, nil
}

// CheckProjectNameExists checks if a project name exists within an organization
func (r *ProjectRepository) CheckProjectNameExists(orgID uuid.UUID, name string, excludeID *uuid.UUID) (bool, error) {
	query := r.db.Model(&models.Project{}).Where("organization_id = ? AND name = ?", orgID, name)
	if excludeID != nil {
		query = query.Where("id != ?", *excludeID)
	}

	var count int64
	err := query.Count(&count).Error
	return count > 0, err
}
