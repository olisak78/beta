package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"developer-portal-backend/internal/cache"
	"developer-portal-backend/internal/database/models"
	apperrors "developer-portal-backend/internal/errors"
	"developer-portal-backend/internal/logger"
	"developer-portal-backend/internal/repository"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// TeamService handles business logic for teams
type TeamService struct {
	repo             *repository.TeamRepository
	groupRepo        repository.GroupRepositoryInterface
	organizationRepo *repository.OrganizationRepository
	userRepo         *repository.UserRepository
	linkRepo         repository.LinkRepositoryInterface
	componentRepo    *repository.ComponentRepository
	validator        *validator.Validate
	cache            cache.CacheService
	cacheWrapper     *cache.CacheWrapper
	cacheTTL         time.Duration
}

// NewTeamService creates a new team service
func NewTeamService(repo *repository.TeamRepository, groupRepo repository.GroupRepositoryInterface, orgRepo *repository.OrganizationRepository, userRepo *repository.UserRepository, linkRepo repository.LinkRepositoryInterface, compRepo *repository.ComponentRepository, validator *validator.Validate, cacheService cache.CacheService) *TeamService {
	return &TeamService{
		repo:             repo,
		groupRepo:        groupRepo,
		organizationRepo: orgRepo,
		userRepo:         userRepo,
		linkRepo:         linkRepo,
		componentRepo:    compRepo,
		validator:        validator,
		cache:            cacheService,
		cacheWrapper:     cache.NewCacheWrapper(cacheService, 10*time.Minute),
		cacheTTL:         10 * time.Minute,
	}
}

// CreateTeamRequest represents the request to create a team
type CreateTeamRequest struct {
	GroupID     uuid.UUID       `json:"group_id" validate:"required"`
	Name        string          `json:"name" validate:"required,min=1,max=40"`
	Title       string          `json:"title" validate:"required,min=1,max=100"`
	Description string          `json:"description" validate:"max=200"`
	Owner       string          `json:"owner" validate:"required,min=5,max=20"`
	Email       string          `json:"email" validate:"required,min=5,max=50"`
	PictureURL  string          `json:"picture_url" validate:"required,min=5,max=200"`
	Metadata    json.RawMessage `json:"metadata" swaggertype:"object"`
}

// UpdateTeamRequest represents the request to update a team
type UpdateTeamRequest struct {
	Title       string          `json:"title" validate:"omitempty,min=1,max=100"`
	Description string          `json:"description" validate:"omitempty,max=200"`
	Owner       string          `json:"owner" validate:"omitempty,min=5,max=20"`
	Email       string          `json:"email" validate:"omitempty,min=5,max=50"`
	PictureURL  string          `json:"picture_url" validate:"omitempty,min=5,max=200"`
	Metadata    json.RawMessage `json:"metadata" swaggertype:"object"`
}

// TeamResponse represents the response for team operations
type TeamResponse struct {
	ID             uuid.UUID       `json:"id"`
	GroupID        uuid.UUID       `json:"group_id"`
	OrganizationID uuid.UUID       `json:"organization_id"` // Include org ID for backwards compatibility
	Name           string          `json:"name"`
	Title          string          `json:"title"`
	Description    string          `json:"description"`
	Owner          string          `json:"owner"`
	Email          string          `json:"email"`
	PictureURL     string          `json:"picture_url"`
	Metadata       json.RawMessage `json:"metadata" swaggertype:"object"`
	CreatedAt      string          `json:"created_at"`
	UpdatedAt      string          `json:"updated_at"`
}

// TeamListResponse represents a paginated list of teams
type TeamListResponse struct {
	Teams    []TeamResponse `json:"teams"`
	Total    int64          `json:"total"`
	Page     int            `json:"page"`
	PageSize int            `json:"page_size"`
}

// TeamWithMembersResponse represents a team with its members
type TeamWithMembersResponse struct {
	TeamResponse
	Members []UserResponse `json:"members"`
	Links   []LinkResponse `json:"links"`
}

// GetByID retrieves a team by ID
func (s *TeamService) GetByID(id uuid.UUID) (*TeamResponse, error) {
	cacheKey := fmt.Sprintf("team:id:%s", id.String())

	var response TeamResponse
	err := s.cacheWrapper.GetOrSetTyped(cacheKey, s.cacheTTL, &response, func() (interface{}, error) {
		team, err := s.repo.GetByID(id)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, apperrors.ErrTeamNotFound
			}
			return nil, fmt.Errorf("failed to get team: %w", err)
		}
		return s.toResponse(team), nil
	})

	if err != nil {
		return nil, err
	}

	return &response, nil
}

// GetAllTeams retrieves teams for a specific organization or all teams if organizationID is nil
func (s *TeamService) GetAllTeams(organizationID *uuid.UUID, page, pageSize int) (*TeamListResponse, error) {
	var cacheKey string
	if organizationID != nil {
		cacheKey = fmt.Sprintf("teams:org:%s:page=%d:size=%d", organizationID.String(), page, pageSize)
	} else {
		cacheKey = "teams:all"
	}

	var response TeamListResponse
	err := s.cacheWrapper.GetOrSetTyped(cacheKey, s.cacheTTL, &response, func() (interface{}, error) {
		if organizationID != nil {
			// Validate organization exists
			_, err := s.organizationRepo.GetByID(*organizationID)
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return nil, apperrors.ErrOrganizationNotFound
				}
				return nil, fmt.Errorf("failed to verify organization: %w", err)
			}

			if page < 1 {
				page = 1
			}
			if pageSize < 1 || pageSize > 100 {
				pageSize = 20
			}

			offset := (page - 1) * pageSize
			teams, total, err := s.repo.GetByOrganizationID(*organizationID, pageSize, offset)
			if err != nil {
				return nil, fmt.Errorf("failed to get teams: %w", err)
			}

			// Filter out the technical team
			filteredTeams := make([]models.Team, 0, len(teams))
			for _, team := range teams {
				if team.Name != "team-developer-portal-technical" {
					filteredTeams = append(filteredTeams, team)
				}
			}

			responses := make([]TeamResponse, len(filteredTeams))
			for i, team := range filteredTeams {
				responses[i] = *s.toResponse(&team)
			}

			// Adjust total count to exclude filtered teams
			adjustedTotal := total
			if len(teams) > len(filteredTeams) {
				adjustedTotal = total - int64(len(teams)-len(filteredTeams))
			}

			return &TeamListResponse{
				Teams:    responses,
				Total:    adjustedTotal,
				Page:     page,
				PageSize: pageSize,
			}, nil
		}

		// Get all teams across all organizations
		teams, err := s.repo.GetAll()
		if err != nil {
			return nil, fmt.Errorf("failed to get all teams: %w", err)
		}

		// Filter out the technical team
		filteredTeams := make([]models.Team, 0, len(teams))
		for _, team := range teams {
			if team.Name != "team-developer-portal-technical" {
				filteredTeams = append(filteredTeams, team)
			}
		}

		responses := make([]TeamResponse, len(filteredTeams))
		for i, team := range filteredTeams {
			responses[i] = *s.toResponse(&team)
		}

		return &TeamListResponse{
			Teams:    responses,
			Total:    int64(len(filteredTeams)),
			Page:     1,
			PageSize: len(filteredTeams),
		}, nil
	})

	if err != nil {
		return nil, err
	}

	return &response, nil
}

// GetTeamComponentsByID retrieves components owned by a team by team ID
func (s *TeamService) GetTeamComponentsByID(id uuid.UUID, page, pageSize int) ([]models.Component, int64, error) {
	// Verify team exists
	if _, err := s.repo.GetByID(id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, 0, apperrors.ErrTeamNotFound
		}
		return nil, 0, fmt.Errorf("failed to get team: %w", err)
	}

	// Set pagination defaults
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 100
	}
	offset := (page - 1) * pageSize

	components, total, err := s.componentRepo.GetComponentsByTeamID(id, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get components by team: %w", err)
	}
	return components, total, nil
}

// GetBySimpleName retrieves a team by name across all organizations and includes its members
func (s *TeamService) GetBySimpleName(teamName string) (*TeamWithMembersResponse, error) {
	if teamName == "" {
		return nil, fmt.Errorf("team name is required")
	}

	cacheKey := fmt.Sprintf("team:name:%s:with-members", teamName)

	var response TeamWithMembersResponse
	err := s.cacheWrapper.GetOrSetTyped(cacheKey, s.cacheTTL, &response, func() (interface{}, error) {
		team, err := s.repo.GetByNameGlobal(teamName)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, apperrors.ErrTeamNotFound
			}
			return nil, fmt.Errorf("failed to get team by name: %w", err)
		}

		// Get all members of the team (no pagination)
		members, _, err := s.userRepo.GetByTeamID(team.ID, 1000, 0)
		if err != nil {
			return nil, fmt.Errorf("failed to get team members: %w", err)
		}

		// Convert team to response
		teamResp := s.toResponse(team)

		// Convert members to UserResponse
		memberResponses := make([]UserResponse, len(members))
		for i, m := range members {
			memberResponses[i] = UserResponse{
				ID:         m.UserID,
				UUID:       m.BaseModel.ID.String(),
				TeamID:     m.TeamID,
				FirstName:  m.FirstName,
				LastName:   m.LastName,
				Email:      m.Email,
				Mobile:     m.Mobile,
				TeamDomain: string(m.TeamDomain),
				TeamRole:   string(m.TeamRole),
			}
		}

		// Fetch links owned by team
		var linkResponses []LinkResponse
		if s.linkRepo != nil {
			if teamLinks, err := s.linkRepo.GetByOwner(team.ID); err == nil {
				linkResponses = make([]LinkResponse, 0, len(teamLinks))
				for i := range teamLinks {
					linkResponses = append(linkResponses, toLinkResponse(&teamLinks[i]))
				}
			}
		}

		return &TeamWithMembersResponse{
			TeamResponse: *teamResp,
			Members:      memberResponses,
			Links:        linkResponses,
		}, nil
	})

	if err != nil {
		return nil, err
	}

	return &response, nil
}

// GetBySimpleNameWithViewer retrieves a team by name across all organizations (with members and links)
// and marks each link's Favorite=true if the logged-in viewer has the link UUID in their metadata.favorites.
func (s *TeamService) GetBySimpleNameWithViewer(teamName string, viewerName string) (*TeamWithMembersResponse, error) {
	cacheKey := fmt.Sprintf("team:name:%s:viewer:%s:with-members", teamName, viewerName)

	var response TeamWithMembersResponse
	err := s.cacheWrapper.GetOrSetTyped(cacheKey, s.cacheTTL, &response, func() (interface{}, error) {
		// Reuse the base implementation
		resp, err := s.GetBySimpleName(teamName)
		if err != nil {
			return nil, err
		}
		if viewerName == "" {
			// No viewer information available; return as-is
			return resp, nil
		}

		// Load viewer by name and parse favorites
		viewer, err := s.userRepo.GetByName(viewerName)
		if err != nil || viewer == nil {
			// Viewer not found; return unmodified
			return resp, nil
		}

		// Build a set of favorite link UUIDs as strings
		favSet := make(map[string]struct{})
		if len(viewer.Metadata) > 0 {
			var meta map[string]interface{}
			if err := json.Unmarshal(viewer.Metadata, &meta); err == nil && meta != nil {
				if v, ok := meta["favorites"]; ok && v != nil {
					switch arr := v.(type) {
					case []interface{}:
						for _, it := range arr {
							if s, ok := it.(string); ok && s != "" {
								if _, parseErr := uuid.Parse(s); parseErr == nil {
									favSet[s] = struct{}{}
								}
							}
						}
					case []string:
						for _, s2 := range arr {
							if _, parseErr := uuid.Parse(s2); parseErr == nil {
								favSet[s2] = struct{}{}
							}
						}
					}
				}
			}
		}

		// Mark favorites in-place
		if len(favSet) > 0 {
			for i := range resp.Links {
				if _, ok := favSet[resp.Links[i].ID]; ok {
					resp.Links[i].Favorite = true
				}
			}
		}

		return resp, nil
	})

	if err != nil {
		return nil, err
	}

	return &response, nil
}

// toResponse converts a team model to response
func (s *TeamService) toResponse(team *models.Team) *TeamResponse {
	// Get organization ID through group (for backwards compatibility)
	var organizationID uuid.UUID
	if group, err := s.groupRepo.GetByID(team.GroupID); err == nil {
		organizationID = group.OrgID
	}

	return &TeamResponse{
		ID:             team.ID,
		GroupID:        team.GroupID,
		OrganizationID: organizationID,
		Name:           team.Name,
		Title:          team.Title,
		Description:    team.Description,
		Owner:          team.Owner,
		Email:          team.Email,
		PictureURL:     team.PictureURL,
		Metadata:       team.Metadata,
	}
}

// UpdateTeamMetadata updates only specific fields in the team's metadata (merge, not replace)
func (s *TeamService) UpdateTeamMetadata(id uuid.UUID, newMetadata json.RawMessage) (*TeamResponse, error) {
	// Get the team first to ensure it exists
	team, err := s.repo.GetByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.ErrTeamNotFound
		}
		return nil, fmt.Errorf("failed to get team: %w", err)
	}

	// Parse existing metadata
	var existingMeta map[string]interface{}
	if len(team.Metadata) > 0 {
		if err := json.Unmarshal(team.Metadata, &existingMeta); err != nil {
			return nil, fmt.Errorf("failed to parse existing metadata: %w", err)
		}
	} else {
		existingMeta = make(map[string]interface{})
	}

	// Parse new metadata to merge
	var newMeta map[string]interface{}
	if err := json.Unmarshal(newMetadata, &newMeta); err != nil {
		return nil, fmt.Errorf("failed to parse new metadata: %w", err)
	}

	// Merge: update existing fields, add new fields, preserve unmentioned fields
	for key, value := range newMeta {
		existingMeta[key] = value
	}

	// Marshal back to JSON
	mergedMetadata, err := json.Marshal(existingMeta)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal merged metadata: %w", err)
	}

	// Update the metadata field
	team.Metadata = mergedMetadata

	// Save the updated team
	if err := s.repo.Update(team); err != nil {
		return nil, fmt.Errorf("failed to update team metadata: %w", err)
	}

	// INVALIDATE related caches
	s.invalidateTeamCaches(team.ID, team.Name)

	// Get organization ID through group (for backwards compatibility)
	var organizationID uuid.UUID
	if group, err := s.groupRepo.GetByID(team.GroupID); err == nil {
		organizationID = group.OrgID
	}

	return &TeamResponse{
		ID:             team.ID,
		GroupID:        team.GroupID,
		OrganizationID: organizationID,
		Name:           team.Name,
		Title:          team.Title,
		Description:    team.Description,
		Owner:          team.Owner,
		Email:          team.Email,
		PictureURL:     team.PictureURL,
		Metadata:       team.Metadata,
	}, nil
}

// invalidateTeamCaches invalidates cache entries related to a team
func (s *TeamService) invalidateTeamCaches(teamID uuid.UUID, teamName string) {
	// Invalidate by ID
	if err := s.cache.Delete(fmt.Sprintf("team:id:%s", teamID.String())); err != nil {
		logger.New().WithFields(map[string]interface{}{
			"team_id":   teamID.String(),
			"cache_key": fmt.Sprintf("team:id:%s", teamID.String()),
		}).WithError(err).Warn("Failed to invalidate team cache by ID")
	}

	// Invalidate by name (with and without members)
	if err := s.cache.Delete(fmt.Sprintf("team:name:%s:with-members", teamName)); err != nil {
		logger.New().WithFields(map[string]interface{}{
			"team_name": teamName,
			"cache_key": fmt.Sprintf("team:name:%s:with-members", teamName),
		}).WithError(err).Warn("Failed to invalidate team cache by name")
	}

	// Invalidate all teams list cache
	if err := s.cache.Delete("teams:all"); err != nil {
		logger.New().WithField("cache_key", "teams:all").WithError(err).Warn("Failed to invalidate teams cache")
	}
}