package service

import (
	"encoding/json"
	"errors"
	"fmt"

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
	repo             repository.TeamRepositoryInterface
	groupRepo        repository.GroupRepositoryInterface
	organizationRepo repository.OrganizationRepositoryInterface
	userRepo         repository.UserRepositoryInterface
	linkRepo         repository.LinkRepositoryInterface
	componentRepo    repository.ComponentRepositoryInterface
	validator        *validator.Validate
}

// NewTeamService creates a new team service
func NewTeamService(repo repository.TeamRepositoryInterface, groupRepo repository.GroupRepositoryInterface, orgRepo repository.OrganizationRepositoryInterface, userRepo repository.UserRepositoryInterface, linkRepo repository.LinkRepositoryInterface, compRepo repository.ComponentRepositoryInterface, validator *validator.Validate) *TeamService {
	return &TeamService{
		repo:             repo,
		groupRepo:        groupRepo,
		organizationRepo: orgRepo,
		userRepo:         userRepo,
		linkRepo:         linkRepo,
		componentRepo:    compRepo,
		validator:        validator,
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
	team, err := s.repo.GetByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.ErrTeamNotFound
		}
		return nil, fmt.Errorf("failed to get team: %w", err)
	}

	return s.toResponse(team)
}

// GetAllTeams retrieves teams for a specific organization or all teams if organizationID is nil
func (s *TeamService) GetAllTeams(organizationID *uuid.UUID, page, pageSize int) (*TeamListResponse, error) {
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
		for i := range filteredTeams {
			resp, err := s.toResponse(&filteredTeams[i])
			if err != nil {
				return nil, fmt.Errorf("failed to convert team to response: %w", err)
			}
			responses[i] = *resp
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

	// Get all teams across all organizations (no pagination since user mentioned <100 teams)
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
	for i := range filteredTeams {
		resp, err := s.toResponse(&filteredTeams[i])
		if err != nil {
			return nil, fmt.Errorf("failed to convert team to response: %w", err)
		}
		responses[i] = *resp
	}

	return &TeamListResponse{
		Teams:    responses,
		Total:    int64(len(filteredTeams)),
		Page:     1,
		PageSize: len(filteredTeams),
	}, nil
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
		return nil, apperrors.NewMissingQueryParam("team_name")
	}

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
	teamResp, err := s.toResponse(team)
	if err != nil {
		return nil, fmt.Errorf("failed to convert team to response: %w", err)
	}

	// Convert members to UserResponse including metadata fallback
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
		teamLinks, err := s.linkRepo.GetByOwner(team.ID)
		if err != nil {
			// Log the error but don't fail the request - links are non-critical
			logger.New().WithFields(map[string]interface{}{
				"team_id":   team.ID,
				"team_name": team.Name,
				"error":     err.Error(),
			}).Warn("Failed to fetch links for team")
		} else {
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
}

// GetBySimpleNameWithViewer retrieves a team by name across all organizations (with members and links)
// and marks each link's Favorite=true if the logged-in viewer has the link UUID in their metadata.favorites.
func (s *TeamService) GetBySimpleNameWithViewer(teamName string, viewerName string) (*TeamWithMembersResponse, error) {
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
		if unmarshalErr := json.Unmarshal(viewer.Metadata, &meta); unmarshalErr != nil {
			// Log the error but don't fail the request - favorites are non-critical
			logger.New().WithFields(map[string]interface{}{
				"viewer_name": viewerName,
				"user_id":     viewer.ID,
				"error":       unmarshalErr.Error(),
			}).Warn("Failed to parse viewer metadata for favorites")
		} else if meta != nil {
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
}

// toResponse converts a team model to response
func (s *TeamService) toResponse(team *models.Team) (*TeamResponse, error) {
	// Get organization ID through group (for backwards compatibility)
	group, err := s.groupRepo.GetByID(team.GroupID)
	if err != nil {
		return nil, fmt.Errorf("failed to get group for team: %w", err)
	}

	return &TeamResponse{
		ID:             team.ID,
		GroupID:        team.GroupID,
		OrganizationID: group.OrgID,
		Name:           team.Name,
		Title:          team.Title,
		Description:    team.Description,
		Owner:          team.Owner,
		Email:          team.Email,
		PictureURL:     team.PictureURL,
		Metadata:       team.Metadata,
	}, nil
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

	return s.toResponse(team)
}
