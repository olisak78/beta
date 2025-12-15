package service

import (
	"developer-portal-backend/internal/database/models"
	"developer-portal-backend/internal/repository"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

// ProjectService handles business logic for project operations
type ProjectService struct {
	projectRepo *repository.ProjectRepository
	validator   *validator.Validate
}

// NewProjectService creates a new project service
func NewProjectService(projectRepo *repository.ProjectRepository, validator *validator.Validate) *ProjectService {
	return &ProjectService{
		projectRepo: projectRepo,
		validator:   validator,
	}
}

// GetAllProjects retrieves all projects from the database
func (s *ProjectService) GetAllProjects() ([]models.Project, error) {
	return s.projectRepo.GetAllProjects()
}

// GetHealthURL retrieves the health URL for a given project ID
func (s *ProjectService) GetHealthURL(projectID uuid.UUID) (string, error) {
	return s.projectRepo.GetHealthURL(projectID)
}
