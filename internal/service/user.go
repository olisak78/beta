package service

import (
	"developer-portal-backend/internal/database/models"
	apperrors "developer-portal-backend/internal/errors"
	"developer-portal-backend/internal/logger"
	"developer-portal-backend/internal/repository"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

// UserService handles business logic for members
type UserService struct {
	repo       repository.UserRepositoryInterface
	linkRepo   repository.LinkRepositoryInterface
	pluginRepo repository.PluginRepositoryInterface
	validator  *validator.Validate
}

// NewUserService creates a new member service
func NewUserService(repo repository.UserRepositoryInterface, linkRepo repository.LinkRepositoryInterface, pluginRepo repository.PluginRepositoryInterface, validator *validator.Validate) *UserService {
	return &UserService{
		repo:       repo,
		linkRepo:   linkRepo,
		pluginRepo: pluginRepo,
		validator:  validator,
	}
}

// CreateUserRequest represents the data needed to create a member
// Note: Aligned with models.Member (BaseModel + string ID for IUser)
type CreateUserRequest struct {
	TeamID    *uuid.UUID `json:"team_id"`
	FirstName string     `json:"first_name" validate:"required,max=100"`
	LastName  string     `json:"last_name" validate:"required,max=100"`
	Email     string     `json:"email" validate:"required,email,max=255"`
	Mobile    string     `json:"mobile" validate:"max=20"`
	IUser     string     `json:"iuser" validate:"required,min=5,max=20"`
	Role      *string    `json:"role" example:"developer" default:"developer"` // maps to TeamDomain
	TeamRole  *string    `json:"team_role" example:"member" default:"member"`
	CreatedBy string     `json:"-"` // derived from bearer token 'username'
}

// UpdateUserRequest represents the data needed to update a member
type UpdateUserRequest struct {
	TeamID     *uuid.UUID `json:"team_id"`
	FirstName  *string    `json:"first_name" validate:"omitempty,max=100"`
	LastName   *string    `json:"last_name" validate:"omitempty,max=100"`
	Email      *string    `json:"email" validate:"omitempty,email,max=255"`
	Mobile     *string    `json:"mobile" validate:"omitempty,max=20"`
	TeamDomain *string    `json:"team_domain"` // models.TeamDomain value
	TeamRole   *string    `json:"team_role"`   // maps to models.TeamRole
}

// UserResponse represents the response data for a member
type UserResponse struct {
	ID         string     `json:"id"`
	UUID       string     `json:"uuid"`
	TeamID     *uuid.UUID `json:"team_id,omitempty"`
	FirstName  string     `json:"first_name"`
	LastName   string     `json:"last_name"`
	Email      string     `json:"email"`
	Mobile     string     `json:"mobile"`
	TeamDomain string     `json:"team_domain"` // models.TeamDomain value
	TeamRole   string     `json:"team_role"`   // models.TeamRole value
}

type UserWithLinksAndPluginsResponse struct {
	ID          string           `json:"id"`
	UUID        string           `json:"uuid"`
	TeamID      *uuid.UUID       `json:"team_id,omitempty"`
	FirstName   string           `json:"first_name"`
	LastName    string           `json:"last_name"`
	Email       string           `json:"email"`
	Mobile      string           `json:"mobile"`
	TeamDomain  string           `json:"team_domain"`
	TeamRole    string           `json:"team_role"`
	PortalAdmin bool             `json:"portal_admin,omitempty"`
	Links       []LinkResponse   `json:"link"`
	Plugins     []PluginResponse `json:"plugins"` // subscribed plugins
}

// UsersListResponse is the swagger schema for GET /users
type UsersListResponse struct {
	Users  []UserResponse `json:"users"`
	Total  int64          `json:"total"`
	Limit  int            `json:"limit"`
	Offset int            `json:"offset"`
}

// LDAPUserSearchItem represents a single LDAP user search result item for Swagger
type LDAPUserSearchItem struct {
	ID        string `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email"`
	Mobile    string `json:"mobile"`
	New       bool   `json:"new"`
}

/* LDAPUserSearchResponse is the swagger schema for GET /users/search/new */
type LDAPUserSearchResponse struct {
	Result []LDAPUserSearchItem `json:"result"`
}

// CreateUser creates a new member
func (s *UserService) CreateUser(req *CreateUserRequest) (*UserResponse, error) {
	// Validate request
	if err := s.validator.Struct(req); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}
	// Require created_by from token
	if strings.TrimSpace(req.CreatedBy) == "" {
		return nil, fmt.Errorf("created_by is required")
	}

	// Check if email already exists (unique within system)
	if existingUser, err := s.repo.GetByEmail(req.Email); err == nil && existingUser != nil {
		logger.New().WithField("error", err).Error("Error getting user by email")
		return nil, apperrors.ErrUserExists
	}

	// Determine team domain (role) default
	teamDomain := models.TeamDomainDeveloper
	if req.Role != nil {
		teamDomain = models.TeamDomain(*req.Role)
	}

	// Determine team role default
	teamRole := models.TeamRoleMember
	if req.TeamRole != nil {
		teamRole = models.TeamRole(*req.TeamRole)
	}

	user := &models.User{
		BaseModel: models.BaseModel{
			Name:      strings.TrimSpace(req.FirstName + " " + req.LastName),
			Title:     strings.TrimSpace(req.FirstName + " " + req.LastName),
			CreatedBy: req.CreatedBy,
		},
		TeamID:     req.TeamID,
		UserID:     req.IUser, // IUser short id on the user model
		FirstName:  req.FirstName,
		LastName:   req.LastName,
		Email:      req.Email,
		Mobile:     req.Mobile,
		TeamDomain: teamDomain,
		TeamRole:   teamRole,
	}

	if err := s.repo.Create(user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return s.convertToResponse(user), nil
}

// AddFavoriteLinkByUserID adds link_id to user's metadata.favorites identified by user_id
func (s *UserService) AddFavoriteLinkByUserID(userID string, linkID uuid.UUID) (*UserResponse, error) {
	if userID == "" {
		return nil, apperrors.NewValidationError("user_id", "user_id is required")
	}
	if linkID == uuid.Nil {
		return nil, apperrors.NewValidationError("link_id", "link_id is required")
	}

	// Load user by string user_id
	user, err := s.repo.GetByUserID(userID)
	if err != nil || user == nil {
		logger.New().WithField("error", err).Error("Error getting user by userID")
		return nil, apperrors.ErrUserNotFound
	}

	// Parse or initialize metadata as a JSON object
	var meta map[string]interface{}
	if len(user.Metadata) == 0 {
		meta = map[string]interface{}{}
	} else {
		if err := json.Unmarshal(user.Metadata, &meta); err != nil || meta == nil {
			// If metadata is invalid/not an object, reset to empty object
			meta = map[string]interface{}{}
		}
	}

	// Ensure favorites array exists
	var favorites []string
	if v, ok := meta["favorites"]; ok && v != nil {
		switch arr := v.(type) {
		case []interface{}:
			for _, it := range arr {
				if str, ok := it.(string); ok && str != "" {
					favorites = append(favorites, str)
				}
			}
		case []string:
			favorites = append(favorites, arr...)
		}
	}

	// Deduplicate: add linkID if not already present
	linkStr := linkID.String()
	exists := false
	for _, id := range favorites {
		if id == linkStr {
			exists = true
			break
		}
	}
	if !exists {
		favorites = append(favorites, linkStr)
	}

	// Save back to metadata
	meta["favorites"] = favorites
	bytes, err := json.Marshal(meta)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}
	user.Metadata = json.RawMessage(bytes)

	// Persist update
	if err := s.repo.Update(user); err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	return s.convertToResponse(user), nil
}

// RemoveFavoriteLinkByUserID removes link_id from user's metadata.favorites identified by user_id
func (s *UserService) RemoveFavoriteLinkByUserID(userID string, linkID uuid.UUID) (*UserResponse, error) {
	if userID == "" {
		return nil, apperrors.NewValidationError("user_id", "user_id is required")
	}
	if linkID == uuid.Nil {
		return nil, apperrors.NewValidationError("link_id", "link_id is required")
	}

	// Load user by string user_id
	user, err := s.repo.GetByUserID(userID)
	if err != nil || user == nil {
		logger.New().WithField("error", err).Error("Error getting user by userID")
		return nil, apperrors.ErrUserNotFound
	}

	// Parse or initialize metadata as a JSON object
	var meta map[string]interface{}
	if len(user.Metadata) == 0 {
		meta = map[string]interface{}{}
	} else {
		if err := json.Unmarshal(user.Metadata, &meta); err != nil || meta == nil {
			// If metadata is invalid/not an object, reset to empty object
			meta = map[string]interface{}{}
		}
	}

	// Extract favorites array if exists
	var favorites []string
	if v, ok := meta["favorites"]; ok && v != nil {
		switch arr := v.(type) {
		case []interface{}:
			for _, it := range arr {
				if str, ok := it.(string); ok && str != "" {
					favorites = append(favorites, str)
				}
			}
		case []string:
			favorites = append(favorites, arr...)
		}
	}

	// Filter out the linkID (idempotent if not present)
	linkStr := linkID.String()
	filtered := make([]string, 0, len(favorites))
	for _, id := range favorites {
		if id != linkStr {
			filtered = append(filtered, id)
		}
	}

	// Save back to metadata
	meta["favorites"] = filtered
	bytes, err := json.Marshal(meta)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}
	user.Metadata = json.RawMessage(bytes)

	// Persist update
	if err := s.repo.Update(user); err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	return s.convertToResponse(user), nil
}

// AddSubscribedPluginByUserID adds plugin_id to user's metadata.subscribed identified by user_id
func (s *UserService) AddSubscribedPluginByUserID(userID string, pluginID uuid.UUID) (*UserResponse, error) {
	if userID == "" {
		return nil, apperrors.NewValidationError("user_id", "user_id is required")
	}
	if pluginID == uuid.Nil {
		return nil, apperrors.NewValidationError("plugin_id", "plugin_id is required")
	}

	// Load user by string user_id
	user, err := s.repo.GetByUserID(userID)
	if err != nil || user == nil {
		logger.New().WithField("error", err).Error("Error getting user by userID")
		return nil, apperrors.ErrUserNotFound
	}

	// Parse or initialize metadata as a JSON object
	var meta map[string]interface{}
	if len(user.Metadata) == 0 {
		meta = map[string]interface{}{}
	} else {
		if err := json.Unmarshal(user.Metadata, &meta); err != nil || meta == nil {
			// If metadata is invalid/not an object, reset to empty object
			meta = map[string]interface{}{}
		}
	}

	// Ensure subscribed array exists
	var subscribed []string
	if v, ok := meta["subscribed"]; ok && v != nil {
		switch arr := v.(type) {
		case []interface{}:
			for _, it := range arr {
				if str, ok := it.(string); ok && str != "" {
					subscribed = append(subscribed, str)
				}
			}
		case []string:
			subscribed = append(subscribed, arr...)
		}
	}

	// Deduplicate: add pluginID if not already present
	pluginStr := pluginID.String()
	exists := false
	for _, id := range subscribed {
		if id == pluginStr {
			exists = true
			break
		}
	}
	if !exists {
		subscribed = append(subscribed, pluginStr)
	}

	// Save back to metadata
	meta["subscribed"] = subscribed
	bytes, err := json.Marshal(meta)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}
	user.Metadata = json.RawMessage(bytes)

	// Persist update
	if err := s.repo.Update(user); err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	return s.convertToResponseWithPlugins(user), nil
}

// RemoveSubscribedPluginByUserID removes plugin_id from user's metadata.subscribed identified by user_id
func (s *UserService) RemoveSubscribedPluginByUserID(userID string, pluginID uuid.UUID) (*UserResponse, error) {
	if userID == "" {
		return nil, apperrors.NewValidationError("user_id", "user_id is required")
	}
	if pluginID == uuid.Nil {
		return nil, apperrors.NewValidationError("plugin_id", "plugin_id is required")
	}

	// Load user by string user_id
	user, err := s.repo.GetByUserID(userID)
	if err != nil || user == nil {
		logger.New().WithField("error", err).Error("Error getting user by userID")
		return nil, apperrors.ErrUserNotFound
	}

	// Parse or initialize metadata as a JSON object
	var meta map[string]interface{}
	if len(user.Metadata) == 0 {
		meta = map[string]interface{}{}
	} else {
		if err := json.Unmarshal(user.Metadata, &meta); err != nil || meta == nil {
			// If metadata is invalid/not an object, reset to empty object
			meta = map[string]interface{}{}
		}
	}

	// Extract subscribed array if exists
	var subscribed []string
	if v, ok := meta["subscribed"]; ok && v != nil {
		switch arr := v.(type) {
		case []interface{}:
			for _, it := range arr {
				if str, ok := it.(string); ok && str != "" {
					subscribed = append(subscribed, str)
				}
			}
		case []string:
			subscribed = append(subscribed, arr...)
		}
	}

	// Filter out the pluginID (idempotent if not present)
	pluginStr := pluginID.String()
	filtered := make([]string, 0, len(subscribed))
	for _, id := range subscribed {
		if id != pluginStr {
			filtered = append(filtered, id)
		}
	}

	// Save back to metadata
	meta["subscribed"] = filtered
	bytes, err := json.Marshal(meta)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}
	user.Metadata = json.RawMessage(bytes)

	// Persist update
	if err := s.repo.Update(user); err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	return s.convertToResponse(user), nil
}

// GetMemberByID retrieves a member by ID (UUID)
func (s *UserService) GetUserByID(id uuid.UUID) (*UserResponse, error) {
	user, err := s.repo.GetByID(id)
	if err != nil {
		logger.New().WithField("error", err).Error("Error getting user by ID")
		return nil, apperrors.ErrUserNotFound
	}

	return s.convertToResponse(user), nil
}

// GetUserByUserID retrieves a member by their string UserID (e.g., I123456)
func (s *UserService) GetUserByUserID(userID string) (*UserResponse, error) {
	if userID == "" {
		return nil, apperrors.NewValidationError("user_id", "user_id is required")
	}

	user, err := s.repo.GetByUserID(userID)
	if err != nil {
		logger.New().WithField("error", err).Error("Error getting user by UserID")
		return nil, apperrors.ErrUserNotFound
	}

	return s.convertToResponse(user), nil
}

// GetUserByName retrieves a user by BaseModel.Name (used to store username)
func (s *UserService) GetUserByName(name string) (*UserResponse, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, apperrors.NewValidationError("name", "name is required")
	}

	user, err := s.repo.GetByName(name)
	if err != nil || user == nil {
		logger.New().WithField("error", err).Error("Error getting user by name")
		return nil, apperrors.ErrUserNotFound
	}

	return s.convertToResponse(user), nil
}

// GetUserByNameWithLinks retrieves a user by BaseModel.Name and returns links-enriched response
func (s *UserService) GetUserByNameWithLinks(name string) (*UserWithLinksAndPluginsResponse, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, apperrors.NewValidationError("name", "name is required")
	}

	user, err := s.repo.GetByName(name)
	if err != nil || user == nil {
		return nil, apperrors.ErrUserNotFound
	}

	// Reuse existing logic by delegating to the user_id-based implementation
	return s.GetUserByUserIDWithLinks(user.UserID)
}

// GetUserByUserIDWithPlugins retrieves subscribed plugins for a user by their UserID
func (s *UserService) GetUserByUserIDWithPlugins(userID string) ([]PluginResponse, error) {
	if userID == "" {
		return nil, apperrors.NewValidationError("user_id", "user_id is required")
	}

	user, err := s.repo.GetByUserID(userID)
	if err != nil || user == nil {
		logger.New().WithField("error", err).Error("Error getting user by UserID")
		return nil, apperrors.ErrUserNotFound
	}

	return s.GetSubscribedPluginsFromUser(user), nil
}

// GetSubscribedPluginsFromUser extracts and fetches subscribed plugins from user metadata
func (s *UserService) GetSubscribedPluginsFromUser(user *models.User) []PluginResponse {
	subscribedPlugins := make([]PluginResponse, 0)
	if len(user.Metadata) > 0 {
		var meta map[string]interface{}
		if err := json.Unmarshal(user.Metadata, &meta); err == nil && meta != nil {
			if v, ok := meta["subscribed"]; ok && v != nil {
				var pluginIDs []uuid.UUID
				switch arr := v.(type) {
				case []interface{}:
					for _, it := range arr {
						if s, ok := it.(string); ok && s != "" {
							if id, err := uuid.Parse(strings.TrimSpace(s)); err == nil {
								pluginIDs = append(pluginIDs, id)
							}
						}
					}
				case []string:
					for _, s2 := range arr {
						if id, err := uuid.Parse(strings.TrimSpace(s2)); err == nil {
							pluginIDs = append(pluginIDs, id)
						}
					}
				}

				// Fetch plugin details for each subscribed plugin
				for _, pluginID := range pluginIDs {
					if plugin, err := s.pluginRepo.GetByID(pluginID); err == nil {
						subscribedPlugins = append(subscribedPlugins, PluginResponse{
							ID:                 plugin.ID,
							Name:               plugin.Name,
							Title:              plugin.Title,
							Description:        plugin.Description,
							Icon:               plugin.Icon,
							ReactComponentPath: plugin.ReactComponentPath,
							BackendServerURL:   plugin.BackendServerURL,
							Owner:              plugin.Owner,
						})
					}
				}
			}
		}
	}
	return subscribedPlugins
}

// GetUserByNameWithLinksAndPlugins retrieves a user by BaseModel.Name and returns both links and plugins
func (s *UserService) GetUserByNameWithLinksAndPlugins(name string) (*UserWithLinksAndPluginsResponse, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, apperrors.NewValidationError("name", "name is required")
	}

	user, err := s.repo.GetByName(name)
	if err != nil || user == nil {
		logger.New().WithField("error", err).Error("Error getting user by name")
		return nil, apperrors.ErrUserNotFound
	}

	// Get the user with links first
	userWithLinks, err := s.GetUserByUserIDWithLinks(user.UserID)
	if err != nil {
		return nil, err
	}

	// Get the subscribed plugins using the extracted method
	userWithLinks.Plugins = s.GetSubscribedPluginsFromUser(user)

	return userWithLinks, nil
}

// GetUserByUserIDWithLinks returns a user with links by their UserID
func (s *UserService) GetUserByUserIDWithLinks(userID string) (*UserWithLinksAndPluginsResponse, error) {
	if userID == "" {
		return nil, apperrors.NewValidationError("user_id", "user_id is required")
	}

	user, err := s.repo.GetByUserID(userID)
	if err != nil || user == nil {
		logger.New().WithField("error", err).Error("Error getting user by userID")
		return nil, apperrors.ErrUserNotFound
	}

	// Portal admin flag computed from metadata
	portalAdmin := false

	// Parse favorites from metadata
	favSet := make(map[uuid.UUID]struct{})
	if len(user.Metadata) > 0 {
		var meta map[string]interface{}
		if err := json.Unmarshal(user.Metadata, &meta); err == nil && meta != nil {
			if v, ok := meta["favorites"]; ok && v != nil {
				switch arr := v.(type) {
				case []interface{}:
					for _, it := range arr {
						if s, ok := it.(string); ok && s != "" {
							if id, err := uuid.Parse(strings.TrimSpace(s)); err == nil {
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
			// Compute portal_admin flag (supports bool, string, numeric)
			if v, ok := meta["portal_admin"]; ok && v != nil {
				switch val := v.(type) {
				case bool:
					portalAdmin = val
				case string:
					trim := strings.TrimSpace(val)
					portalAdmin = strings.EqualFold(trim, "true") || trim == "1" || strings.EqualFold(trim, "yes")
				case float64:
					portalAdmin = val != 0
				}
			}
		}
	}

	// Collect favorite IDs
	favIDs := make([]uuid.UUID, 0, len(favSet))
	for id := range favSet {
		favIDs = append(favIDs, id)
	}

	// Fetch links (favorites + owned)
	favorites, _ := s.linkRepo.GetByIDs(favIDs)
	owned, _ := s.linkRepo.GetByOwner(user.ID)

	// Merge unique by ID
	combined := make(map[uuid.UUID]models.Link)
	for _, l := range owned {
		combined[l.ID] = l
	}
	for _, l := range favorites {
		combined[l.ID] = l
	}

	// Build link responses and mark favorites
	links := make([]LinkResponse, 0, len(combined))
	for _, l := range combined {
		lr := toLinkResponse(&l)
		if _, ok := favSet[l.ID]; ok {
			lr.Favorite = true
		}
		links = append(links, lr)
	}

	resp := &UserWithLinksAndPluginsResponse{
		ID:          user.UserID,
		UUID:        user.ID.String(),
		TeamID:      user.TeamID,
		FirstName:   user.FirstName,
		LastName:    user.LastName,
		Email:       user.Email,
		Mobile:      user.Mobile,
		TeamDomain:  string(user.TeamDomain),
		TeamRole:    string(user.TeamRole),
		PortalAdmin: portalAdmin,
		Links:       links,
		Plugins:     []PluginResponse{}, // Empty array, plugins should be fetched separately if needed
	}
	return resp, nil
}

func (s *UserService) GetAllUsers(limit, offset int) ([]UserResponse, int64, error) {
	users, total, err := s.repo.GetAll(limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get users: %w", err)
	}

	responses := make([]UserResponse, len(users))
	for i, user := range users {
		responses[i] = *s.convertToResponse(&user)
	}

	return responses, total, nil
}

// SearchUsersGlobal performs case-insensitive search across BaseModel.Name and BaseModel.Title
func (s *UserService) SearchUsersGlobal(query string, limit, offset int) ([]UserResponse, int64, error) {
	users, total, err := s.repo.SearchByNameOrTitleGlobal(query, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to search users: %w", err)
	}

	responses := make([]UserResponse, len(users))
	for i, user := range users {
		responses[i] = *s.convertToResponse(&user)
	}

	return responses, total, nil
}

// GetMembersByOrganization retrieves members for an organization
func (s *UserService) GetUsersByOrganization(organizationID uuid.UUID, limit, offset int) ([]UserResponse, int64, error) {
	users, total, err := s.repo.GetByOrganizationID(organizationID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get users: %w", err)
	}

	responses := make([]UserResponse, len(users))
	for i, user := range users {
		responses[i] = *s.convertToResponse(&user)
	}

	return responses, total, nil
}

// UpdateMember updates an existing member
func (s *UserService) UpdateUser(id uuid.UUID, req *UpdateUserRequest) (*UserResponse, error) {
	// Validate request
	if err := s.validator.Struct(req); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	user, err := s.repo.GetByID(id)
	if err != nil {
		logger.New().WithField("error", err).Error("Error getting user by id")
		return nil, apperrors.ErrUserNotFound
	}

	// Check email uniqueness if email is being updated
	if req.Email != nil && *req.Email != user.Email {
		if existingUser, err := s.repo.GetByEmail(*req.Email); err == nil && existingUser != nil {
			logger.New().WithField("error", err).Error("Error getting user by email")
			return nil, apperrors.ErrUserExists
		}
	}

	// Update fields
	if req.TeamID != nil {
		user.TeamID = req.TeamID
	}
	if req.FirstName != nil {
		user.FirstName = *req.FirstName
	}
	if req.LastName != nil {
		user.LastName = *req.LastName
	}
	if req.Email != nil {
		user.Email = *req.Email
	}
	if req.Mobile != nil {
		user.Mobile = *req.Mobile
	}
	if req.TeamDomain != nil {
		user.TeamDomain = models.TeamDomain(*req.TeamDomain)
	}
	if req.TeamRole != nil {
		user.TeamRole = models.TeamRole(*req.TeamRole)
	}

	if err := s.repo.Update(user); err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	return s.convertToResponse(user), nil
}

// UpdateUserTeam sets a user's team and audit fields
func (s *UserService) UpdateUserTeam(userID uuid.UUID, teamID uuid.UUID, updatedBy string) (*UserResponse, error) {
	if strings.TrimSpace(updatedBy) == "" {
		return nil, fmt.Errorf("updated_by is required")
	}
	user, err := s.repo.GetByID(userID)
	if err != nil || user == nil {
		logger.New().WithField("error", err).Error("Error getting user by userID")
		return nil, apperrors.ErrUserNotFound
	}
	user.TeamID = &teamID
	user.UpdatedBy = updatedBy
	if err := s.repo.Update(user); err != nil {
		return nil, fmt.Errorf("failed to update user team: %w", err)
	}
	return s.convertToResponse(user), nil
}

// DeleteMember deletes a
func (s *UserService) DeleteUser(id uuid.UUID) error {
	_, err := s.repo.GetByID(id)
	if err != nil {
		logger.New().WithField("error", err).Error("Error getting user by id")
		return apperrors.ErrUserNotFound
	}

	if err := s.repo.Delete(id); err != nil {
		return fmt.Errorf("failed to delete member: %w", err)
	}

	return nil
}

// SearchMembers searches for members by first/last name or email
func (s *UserService) SearchUsers(organizationID uuid.UUID, query string, limit, offset int) ([]UserResponse, int64, error) {
	users, total, err := s.repo.SearchByOrganization(organizationID, query, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to search users: %w", err)
	}

	responses := make([]UserResponse, len(users))
	for i, user := range users {
		responses[i] = *s.convertToResponse(&user)
	}

	return responses, total, nil
}

// GetActiveMembers returns all members for an organization (is_active removed from model)
func (s *UserService) GetActiveUsers(organizationID uuid.UUID, limit, offset int) ([]UserResponse, int64, error) {
	users, total, err := s.repo.GetActiveByOrganization(organizationID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get active users: %w", err)
	}

	responses := make([]UserResponse, len(users))
	for i, user := range users {
		responses[i] = *s.convertToResponse(&user)
	}

	return responses, total, nil
}

// convertToResponse converts a member model to response
func (s *UserService) convertToResponse(user *models.User) *UserResponse {
	return &UserResponse{
		ID:         user.UserID,
		UUID:       user.ID.String(),
		TeamID:     user.TeamID,
		FirstName:  user.FirstName,
		LastName:   user.LastName,
		Email:      user.Email,
		Mobile:     user.Mobile,
		TeamDomain: string(user.TeamDomain),
		TeamRole:   string(user.TeamRole),
	}
}

// convertToResponseWithPlugins converts a member model to response with subscribed plugins fetched
func (s *UserService) convertToResponseWithPlugins(user *models.User) *UserResponse {
	return &UserResponse{
		ID:         user.UserID,
		UUID:       user.ID.String(),
		TeamID:     user.TeamID,
		FirstName:  user.FirstName,
		LastName:   user.LastName,
		Email:      user.Email,
		Mobile:     user.Mobile,
		TeamDomain: string(user.TeamDomain),
		TeamRole:   string(user.TeamRole),
	}
}

// ===== Quick Links compatibility stubs (model no longer stores metadata) =====

// AddQuickLinkRequest represents the request to add a quick link to a member
type AddQuickLinkRequest struct {
	URL      string `json:"url" validate:"required,url"`
	Title    string `json:"title" validate:"required"`
	Icon     string `json:"icon,omitempty"`
	Category string `json:"category,omitempty"`
}

// QuickLink represents a quick link in the response
type QuickLink struct {
	URL      string `json:"url"`
	Title    string `json:"title"`
	Icon     string `json:"icon,omitempty"`
	Category string `json:"category,omitempty"`
}

// QuickLinksResponse represents the response for getting quick links
type QuickLinksResponse struct {
	QuickLinks []QuickLink `json:"quick_links"`
}

// GetQuickLinks retrieves quick links from a member (returns empty since metadata removed)
func (s *UserService) GetQuickLinks(id uuid.UUID) (*QuickLinksResponse, error) {
	// Validate member exists
	if _, err := s.repo.GetByID(id); err != nil {
		logger.New().WithField("error", err).Error("Error getting user by name")
		return nil, apperrors.ErrUserNotFound
	}
	return &QuickLinksResponse{QuickLinks: []QuickLink{}}, nil
}

// AddQuickLink adds a quick link (no-op; returns member unchanged)
func (s *UserService) AddQuickLink(id uuid.UUID, req *AddQuickLinkRequest) (*UserResponse, error) {
	// Validate request
	if err := s.validator.Struct(req); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}
	user, err := s.repo.GetByID(id)
	if err != nil {
		logger.New().WithField("error", err).Error("Error getting user by id")
		return nil, apperrors.ErrUserNotFound
	}
	return s.convertToResponse(user), nil
}

// RemoveQuickLink removes a quick link (no-op; returns member unchanged)
func (s *UserService) RemoveQuickLink(id uuid.UUID, linkURL string) (*UserResponse, error) {
	if linkURL == "" {
		return nil, apperrors.NewValidationError("url", "link URL is required")
	}
	user, err := s.repo.GetByID(id)
	if err != nil {
		logger.New().WithField("error", err).Error("Error getting user by id")
		return nil, apperrors.ErrUserNotFound
	}
	return s.convertToResponse(user), nil
}
