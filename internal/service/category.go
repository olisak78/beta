package service

import (
	"fmt"

	"developer-portal-backend/internal/database/models"
	"developer-portal-backend/internal/repository"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

// CategoryService provides category-related business logic
type CategoryService struct {
	repo      repository.CategoryRepositoryInterface
	validator *validator.Validate
}

// Ensure CategoryService implements CategoryServiceInterface
var _ CategoryServiceInterface = (*CategoryService)(nil)

// NewCategoryService creates a new CategoryService
func NewCategoryService(repo repository.CategoryRepositoryInterface, validator *validator.Validate) *CategoryService {
	return &CategoryService{
		repo:      repo,
		validator: validator,
	}
}

// CategoryResponse represents a single category in API responses
type CategoryResponse struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Icon        string    `json:"icon"`
	Color       string    `json:"color"`
}

// CategoryListResponse represents a paginated list of categories
type CategoryListResponse struct {
	Categories []CategoryResponse `json:"categories"`
	Total      int64              `json:"total"`
	Page       int                `json:"page"`
	PageSize   int                `json:"page_size"`
}

// GetAll retrieves categories with pagination
func (s *CategoryService) GetAll(page, pageSize int) (*CategoryListResponse, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 1000 {
		// allow larger default to better match "return all categories" use-case
		pageSize = 1000
	}

	offset := (page - 1) * pageSize
	cats, total, err := s.repo.GetAll(pageSize, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get categories: %w", err)
	}

	responses := make([]CategoryResponse, len(cats))
	for i, c := range cats {
		responses[i] = s.toResponse(&c)
	}

	return &CategoryListResponse{
		Categories: responses,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
	}, nil
}

// toResponse converts a Category model to API response
func (s *CategoryService) toResponse(cat *models.Category) CategoryResponse {
	return CategoryResponse{
		ID:          cat.ID,
		Name:        cat.Name,
		Title:       cat.Title,
		Description: cat.Description,
		Icon:        cat.Icon,
		Color:       cat.Color,
	}
}
