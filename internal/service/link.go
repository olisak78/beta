package service

import (
	"encoding/json"
	"fmt"
	"strings"

	"developer-portal-backend/internal/database/models"
	apperrors "developer-portal-backend/internal/errors"
	"developer-portal-backend/internal/logger"
	"developer-portal-backend/internal/repository"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

// LinkService provides link-related business logic
type LinkService struct {
	linkRepo     repository.LinkRepositoryInterface
	userRepo     repository.UserRepositoryInterface
	teamRepo     repository.TeamRepositoryInterface
	categoryRepo repository.CategoryRepositoryInterface
	validator    *validator.Validate
}

// Ensure LinkService implements LinkServiceInterface
var _ LinkServiceInterface = (*LinkService)(nil)

// NewLinkService creates a new LinkService
func NewLinkService(linkRepo repository.LinkRepositoryInterface, userRepo repository.UserRepositoryInterface, teamRepo repository.TeamRepositoryInterface, categoryRepo repository.CategoryRepositoryInterface, validator *validator.Validate) *LinkService {
	return &LinkService{
		linkRepo:     linkRepo,
		userRepo:     userRepo,
		teamRepo:     teamRepo,
		categoryRepo: categoryRepo,
		validator:    validator,
	}
}

// LinkResponse represents a link in API responses (omits audit and owner fields)
type LinkResponse struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	URL         string   `json:"url"`
	CategoryID  string   `json:"category_id"`
	Tags        []string `json:"tags"`
	Favorite    bool     `json:"favorite,omitempty"`
}

// CreateLinkRequest represents the payload for creating a link
type CreateLinkRequest struct {
	Name        string `json:"name" validate:"required,min=1,max=40"`
	Description string `json:"description" validate:"max=200"`
	Owner       string `json:"owner" validate:"required,uuid4"`
	URL         string `json:"url" validate:"required,url,max=2000"`
	CategoryID  string `json:"category_id" validate:"required,uuid4"`
	Tags        string `json:"tags" validate:"max=200"` // optional CSV string
	CreatedBy   string `json:"-"`                       // derived from bearer token 'username'
}

// UpdateLinkRequest represents the payload for updating a link
type UpdateLinkRequest struct {
	Name        string `json:"name" validate:"required,min=1,max=40"`
	Description string `json:"description" validate:"max=200"`
	URL         string `json:"url" validate:"required,url,max=2000"`
	CategoryID  string `json:"category_id" validate:"required,uuid4"`
	Tags        string `json:"tags" validate:"max=200"` // optional CSV string
	UpdatedBy   string `json:"-"`                       // derived from bearer token 'username'
}

// CreateLink validates and creates a new link
func (s *LinkService) CreateLink(req *CreateLinkRequest) (*LinkResponse, error) {
	if err := s.validator.Struct(req); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}
	if strings.TrimSpace(req.CreatedBy) == "" {
		return nil, fmt.Errorf("created_by is required")
	}
	// Validate created_by is an existing users.user_id OR a team's name
	if _, err := s.userRepo.GetByUserID(req.CreatedBy); err != nil {
		if _, errTeam := s.teamRepo.GetByNameGlobal(req.CreatedBy); errTeam != nil {
			return nil, fmt.Errorf("created_by user or team not found")
		}
	}

	ownerUUID, err := uuid.Parse(req.Owner)
	if err != nil {
		return nil, fmt.Errorf("invalid owner UUID: %w", err)
	}
	categoryUUID, err := uuid.Parse(req.CategoryID)
	if err != nil {
		return nil, fmt.Errorf("invalid category_id UUID: %w", err)
	}

	// Validate owner exists (either a user or a team)
	ownerValid := false
	if _, err := s.userRepo.GetByID(ownerUUID); err == nil {
		ownerValid = true
	} else if _, err := s.teamRepo.GetByID(ownerUUID); err == nil {
		ownerValid = true
	}
	if !ownerValid {
		return nil, fmt.Errorf("owner not found as user or team")
	}

	// Validate category exists
	if _, err := s.categoryRepo.GetByID(categoryUUID); err != nil {
		return nil, fmt.Errorf("category not found")
	}

	link := &models.Link{
		BaseModel: models.BaseModel{
			Name:        req.Name,
			Title:       req.Name, // Title mirrors name per requirement
			Description: req.Description,
			CreatedBy:   req.CreatedBy,
		},
		Owner:      ownerUUID,
		URL:        req.URL,
		CategoryID: categoryUUID,
		Tags:       req.Tags,
	}

	if err := s.linkRepo.Create(link); err != nil {
		return nil, fmt.Errorf("failed to create link: %w", err)
	}

	res := toLinkResponse(link)
	return &res, nil
}

// UpdateLink validates and updates an existing link
func (s *LinkService) UpdateLink(id uuid.UUID, req *UpdateLinkRequest) (*LinkResponse, error) {
	log := logger.New().WithFields(map[string]interface{}{
		"operation":  "UpdateLink",
		"link_id":    id.String(),
		"updated_by": req.UpdatedBy,
		"link_name":  req.Name,
	})

	// Validate request structure
	if err := s.validator.Struct(req); err != nil {
		log.WithField("error", err.Error()).Warn("Link validation failed")
		return nil, apperrors.NewValidationError("", err.Error())
	}

	if strings.TrimSpace(req.UpdatedBy) == "" {
		log.Warn("updated_by is missing")
		return nil, apperrors.NewValidationError("updated_by", "updated_by is required")
	}

	// Validate updated_by is an existing users.user_id OR a team's name
	if _, err := s.userRepo.GetByUserID(req.UpdatedBy); err != nil {
		if _, errTeam := s.teamRepo.GetByNameGlobal(req.UpdatedBy); errTeam != nil {
			log.WithField("updated_by", req.UpdatedBy).Warn("updated_by user or team not found")
			return nil, apperrors.ErrUserOrTeamNotFound
		}
	}

	// Get existing link
	link, err := s.linkRepo.GetByID(id)
	if err != nil {
		log.WithField("error", err.Error()).Warn("Link not found in database")
		return nil, apperrors.ErrLinkNotFound
	}

	// Log the original state for audit trail
	log.WithFields(map[string]interface{}{
		"original_name":        link.Name,
		"original_url":         link.URL,
		"original_category_id": link.CategoryID.String(),
		"original_owner":       link.Owner.String(), // Log owner to verify it's not changed
	}).Debug("Retrieved link for update")

	categoryUUID, err := uuid.Parse(req.CategoryID)
	if err != nil {
		log.WithField("category_id", req.CategoryID).Warn("Invalid category_id UUID")
		return nil, apperrors.NewValidationError("category_id", "invalid UUID format")
	}

	// Validate category exists
	if _, err := s.categoryRepo.GetByID(categoryUUID); err != nil {
		log.WithField("category_id", req.CategoryID).Warn("Category not found")
		return nil, apperrors.ErrCategoryNotFound
	}

	// Update ONLY the allowed fields
	// SECURITY: Owner is intentionally NOT updated here to prevent ownership changes
	link.Name = req.Name
	link.Title = req.Name // Title mirrors name per requirement
	link.Description = req.Description
	link.URL = req.URL
	link.CategoryID = categoryUUID
	link.Tags = req.Tags
	link.UpdatedBy = req.UpdatedBy
	// link.Owner is NOT modified - ownership cannot be changed

	if err := s.linkRepo.Update(link); err != nil {
		log.WithField("error", err.Error()).Error("Failed to update link in database")
		return nil, fmt.Errorf("failed to update link: %w", err)
	}

	log.WithFields(map[string]interface{}{
		"link_id":     link.ID.String(),
		"link_name":   link.Name,
		"url":         link.URL,
		"category_id": link.CategoryID.String(),
		"owner":       link.Owner.String(), // Verify owner unchanged
	}).Info("Link updated successfully in database")

	res := toLinkResponse(link)
	return &res, nil
}

// GetByOwnerUserID returns all links owned by the user with the given user_id
func (s *LinkService) GetByOwnerUserID(ownerUserID string) ([]LinkResponse, error) {
	if strings.TrimSpace(ownerUserID) == "" {
		return nil, fmt.Errorf("owner user_id is required")
	}

	// Find user by user_id
	user, err := s.userRepo.GetByUserID(ownerUserID)
	if err != nil || user == nil {
		return nil, fmt.Errorf("owner user with user_id %q not found", ownerUserID)
	}

	// Fetch links by owner UUID
	links, err := s.linkRepo.GetByOwner(user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get links by owner: %w", err)
	}

	// Map to response type, omitting audit and owner fields
	res := make([]LinkResponse, 0, len(links))
	for i := range links {
		res = append(res, toLinkResponse(&links[i]))
	}
	return res, nil
}

// GetByOwnerUserIDWithViewer returns links owned by the given user and marks favorites based on viewer's favorites
func (s *LinkService) GetByOwnerUserIDWithViewer(ownerUserID string, viewerName string) ([]LinkResponse, error) {
	if strings.TrimSpace(ownerUserID) == "" {
		return nil, fmt.Errorf("owner user_id is required")
	}
	if strings.TrimSpace(viewerName) == "" {
		// Fallback to non-favorite response if viewer missing
		return s.GetByOwnerUserID(ownerUserID)
	}

	// Find owner by user_id
	owner, err := s.userRepo.GetByUserID(ownerUserID)
	if err != nil || owner == nil {
		return nil, fmt.Errorf("owner user with user_id %q not found", ownerUserID)
	}

	// Find viewer by name (mapped from bearer token 'username')
	viewer, err := s.userRepo.GetByName(viewerName)
	if err != nil || viewer == nil {
		// Fallback to non-favorite response if viewer not found
		return s.GetByOwnerUserID(ownerUserID)
	}

	// Parse favorites from viewer.Metadata
	favSet := make(map[uuid.UUID]struct{})
	if len(viewer.Metadata) > 0 {
		var meta map[string]interface{}
		if err := json.Unmarshal(viewer.Metadata, &meta); err == nil && meta != nil {
			if v, ok := meta["favorites"]; ok && v != nil {
				switch arr := v.(type) {
				case []interface{}:
					for _, it := range arr {
						if str, ok := it.(string); ok && str != "" {
							if id, err := uuid.Parse(strings.TrimSpace(str)); err == nil {
								favSet[id] = struct{}{}
							}
						}
					}
				case []string:
					for _, s2 := range arr {
						if id, err := uuid.Parse(strings.TrimSpace(s2)); err == nil {
							favSet[id] = struct{}{}
						}
					}
				}
			}
		}
	}

	// Fetch links by owner UUID
	links, err := s.linkRepo.GetByOwner(owner.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get links by owner: %w", err)
	}

	// Map to response type, omitting audit and owner fields; mark favorites
	res := make([]LinkResponse, 0, len(links))
	for i := range links {
		lr := toLinkResponse(&links[i])
		if _, ok := favSet[links[i].ID]; ok {
			lr.Favorite = true
		}
		res = append(res, lr)
	}
	return res, nil
}

func toLinkResponse(l *models.Link) LinkResponse {
	tags := make([]string, 0) // Initialize to empty slice instead of nil
	if strings.TrimSpace(l.Tags) != "" {
		parts := strings.Split(l.Tags, ",")
		for _, p := range parts {
			if t := strings.TrimSpace(p); t != "" {
				tags = append(tags, t)
			}
		}
	}

	return LinkResponse{
		ID:          l.ID.String(),
		Name:        l.Name,
		Title:       l.Title,
		Description: l.Description,
		URL:         l.URL,
		CategoryID:  l.CategoryID.String(),
		Tags:        tags,
	}
}

// DeleteLink deletes a link by UUID
func (s *LinkService) DeleteLink(id uuid.UUID) error {
	// Delegate to repository; repository Delete is idempotent
	if err := s.linkRepo.Delete(id); err != nil {
		return fmt.Errorf("failed to delete link: %w", err)
	}
	return nil
}
