package handlers

import (
	"developer-portal-backend/internal/auth"
	"developer-portal-backend/internal/database/models"
	"developer-portal-backend/internal/service"
	"errors"
	"net/http"
	"strconv"

	apperrors "developer-portal-backend/internal/errors"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// UserHandler handles HTTP requests for users (members endpoints removed)
type UserHandler struct {
	memberService service.UserServiceInterface
	teamService   service.TeamServiceInterface
}

// NewUserHandler creates a new user handler
func NewUserHandler(memberService service.UserServiceInterface, teamService service.TeamServiceInterface) *UserHandler {
	return &UserHandler{
		memberService: memberService,
		teamService:   teamService,
	}
}

// CreateUserBody represents the expected request body for POST /users
type CreateUserBody struct {
	ID         string    `json:"id" binding:"required,min=5,max=20"`
	FirstName  string    `json:"first_name" binding:"required,max=100"`
	LastName   string    `json:"last_name" binding:"required,max=100"`
	Email      string    `json:"email" binding:"required,email,max=255"`
	Mobile     string    `json:"mobile"`      // optional
	TeamDomain *string   `json:"team_domain"` // optional, defaults to 'developer' if omitted
	TeamRole   *string   `json:"team_role"`   // optional, defaults to 'member' if omitted
	TeamID     uuid.UUID `json:"team_id" binding:"required"`
}

// CreateUser handles POST /users
// @Summary Create a new user
// @Description Create a new user row in users table
// @Description Optional fields: team_domain (default 'developer'), team_role (default 'member')
// @Description created_by is derived from the bearer token 'username' claim and is NOT required in the payload
// @Tags users
// @Accept json
// @Produce json
// @Param user body CreateUserBody true "User data"
// @Success 201 {object} service.UserResponse "Successfully created user"
// @Failure 400 {object} map[string]interface{} "Invalid request body"
// @Security BearerAuth
// @Router /users [post]
func (h *UserHandler) CreateUser(c *gin.Context) {
	var body CreateUserBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate team_id exists
	if _, err := h.teamService.GetByID(body.TeamID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid team_id"})
		return
	}

	// Validate team_domain against allowed values (optional; default handled in service)
	if body.TeamDomain != nil {
		switch models.TeamDomain(*body.TeamDomain) {
		case models.TeamDomainDeveloper, models.TeamDomainDevOps, models.TeamDomainPO, models.TeamDomainArchitect:
			// ok
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid team_domain"})
			return
		}
	}

	// Validate team_role against allowed values (optional; default handled in service)
	if body.TeamRole != nil {
		switch models.TeamRole(*body.TeamRole) {
		case models.TeamRoleMember, models.TeamRoleScM, models.TeamRoleManager, models.TeamRoleMMM:
			// ok
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid team_role"})
			return
		}
	}

	req := service.CreateUserRequest{
		TeamID:    &body.TeamID,
		FirstName: body.FirstName,
		LastName:  body.LastName,
		Email:     body.Email,
		Mobile:    body.Mobile,
		IUser:     body.ID,
	}
	if body.TeamDomain != nil {
		role := *body.TeamDomain
		req.Role = &role
	}
	if body.TeamRole != nil {
		tr := *body.TeamRole
		req.TeamRole = &tr
	}

	// Populate created_by from bearer token username
	if username, ok := auth.GetUsername(c); ok && username != "" {
		req.CreatedBy = username
	} else {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing username in token"})
		return
	}

	user, err := h.memberService.CreateUser(&req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, user)
}

// GetMemberByUserID retrieves a user by UserID string (I/C/D ID)
// @Summary Get user by UserID
// @Description Get a specific user by their UserID (e.g., I123456)
// @Tags users
// @Accept json
// @Produce json
// @Param user_id path string true "UserID (I/C/D)"
// @Success 200 {object} service.UserWithLinksAndPluginsResponse "Successfully retrieved user"
// @Failure 400 {object} map[string]interface{} "Invalid user_id"
// @Failure 404 {object} map[string]interface{} "User not found"
// @Security BearerAuth
// @Router /users/{user_id} [get]
func (h *UserHandler) GetMemberByUserID(c *gin.Context) {
	userID := c.Param("user_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id is required"})
		return
	}

	member, err := h.memberService.GetUserByUserIDWithLinks(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, member)
}

// ListUsers retrieves all users with pagination
// @Summary List users
// @Description Get all users with pagination
// @Tags users
// @Accept json
// @Produce json
// @Param limit query int false "Number of items to return" default(20)
// @Param offset query int false "Number of items to skip" default(0)
// @Param q query string false "Search query by name or title (case-insensitive)"
// @Success 200 {object} service.UsersListResponse "Successfully retrieved users list"
// @Failure 400 {object} map[string]interface{} "Invalid parameters"
// @Security BearerAuth
// @Router /users [get]
func (h *UserHandler) ListUsers(c *gin.Context) {
	// If user-name query param is provided, return the user by name (with links)
	if userName := c.Query("user-name"); userName != "" {
		user, err := h.memberService.GetUserByNameWithLinks(userName)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}
		c.JSON(http.StatusOK, user)
		return
	}

	// Otherwise, list users with pagination
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	// If 'q' is provided, perform global search by name or title
	if q := c.Query("q"); q != "" {
		users, total, err := h.memberService.SearchUsersGlobal(q, limit, offset)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"users":  users,
			"total":  total,
			"limit":  limit,
			"offset": offset,
		})
		return
	}

	users, total, err := h.memberService.GetAllUsers(limit, offset)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"users":  users,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// GetCurrentUser handles GET /users/me
// @Summary Get current user
// @Description Returns the user matching the bearer token 'username' claim, mapped to users.name
// @Tags users
// @Produce json
// @Success 200 {object} service.UserWithLinksAndPluginsResponse "Successfully retrieved current user"
// @Failure 401 {object} map[string]interface{} "Missing username in token"
// @Failure 404 {object} map[string]interface{} "User not found"
// @Security BearerAuth
// @Router /users/me [get]
func (h *UserHandler) GetCurrentUser(c *gin.Context) {
	username, ok := auth.GetUsername(c)
	if !ok || username == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing username in token"})
		return
	}
	user, err := h.memberService.GetUserByNameWithLinksAndPlugins(username)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}
	c.JSON(http.StatusOK, user)
}

// UpdateUserTeamBody represents the expected request body for PUT /users
type UpdateUserTeamBody struct {
	UserUUID    string `json:"user_uuid" binding:"required"`
	NewTeamUUID string `json:"new_team_uuid" binding:"required"`
}

// UpdateUserTeam handles PUT /users
// @Summary Update user's team
// @Description Update the user's team by UUID. Sets updated_by from token; updated_at is automatic.
// @Tags users
// @Accept json
// @Produce json
// @Param body body UpdateUserTeamBody true "Update user's team payload"
// @Success 200 {object} service.UserResponse "Successfully updated user's team"
// @Failure 400 {object} map[string]interface{} "Invalid request body or UUIDs"
// @Failure 401 {object} map[string]interface{} "Missing username in token"
// @Failure 404 {object} map[string]interface{} "User not found"
// @Security BearerAuth
// @Router /users [put]
func (h *UserHandler) UpdateUserTeam(c *gin.Context) {
	var body UpdateUserTeamBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, err := uuid.Parse(body.UserUUID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_uuid"})
		return
	}
	teamID, err := uuid.Parse(body.NewTeamUUID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid new_team_uuid"})
		return
	}

	// Validate team exists
	if _, err := h.teamService.GetByID(teamID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid new_team_uuid"})
		return
	}

	// Get username from token for audit
	username, ok := auth.GetUsername(c)
	if !ok || username == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing username in token"})
		return
	}

	user, err := h.memberService.UpdateUserTeam(userID, teamID, username)
	if err != nil {
		if errors.Is(err, apperrors.ErrUserNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, user)
}

// AddFavoriteLink handles POST /users/:user_id/favorites/:link_id
// @Summary Add a favorite link to a user
// @Description Adds the given link_id to the user's metadata.favorites array. Initializes metadata and favorites if missing, and avoids duplicates.
// @Tags users
// @Accept json
// @Produce json
// @Param user_id path string true "User ID (I/C/D user id, e.g. cis.devops)"
// @Param link_id path string true "Link ID (UUID)"
// @Success 200 {object} service.UserResponse "Successfully added favorite link"
// @Failure 400 {object} map[string]interface{} "Invalid user_id or link_id"
// @Failure 404 {object} map[string]interface{} "User not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Security BearerAuth
// @Router /users/{user_id}/favorites/{link_id} [post]
func (h *UserHandler) AddFavoriteLink(c *gin.Context) {
	userID := c.Param("user_id")
	linkIDStr := c.Param("link_id")

	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id is required"})
		return
	}
	linkID, err := uuid.Parse(linkIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid link_id"})
		return
	}

	user, err := h.memberService.AddFavoriteLinkByUserID(userID, linkID)
	if err != nil {
		if errors.Is(err, apperrors.ErrUserNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to add favorite", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, user)
}

// RemoveFavoriteLink handles DELETE /users/:user_id/favorites/:link_id
// @Summary Remove a favorite link from a user
// @Description Removes the given link_id from the user's metadata.favorites array. Initializes metadata if missing. Idempotent if link not present.
// @Tags users
// @Accept json
// @Produce json
// @Param user_id path string true "User ID (I/C/D user id, e.g. cis.devops)"
// @Param link_id path string true "Link ID (UUID)"
// @Success 200 {object} service.UserResponse "Successfully removed favorite link"
// @Failure 400 {object} map[string]interface{} "Invalid user_id or link_id"
// @Failure 404 {object} map[string]interface{} "User not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Security BearerAuth
// @Router /users/{user_id}/favorites/{link_id} [delete]
func (h *UserHandler) RemoveFavoriteLink(c *gin.Context) {
	userID := c.Param("user_id")
	linkIDStr := c.Param("link_id")

	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id is required"})
		return
	}
	linkID, err := uuid.Parse(linkIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid link_id"})
		return
	}

	user, err := h.memberService.RemoveFavoriteLinkByUserID(userID, linkID)
	if err != nil {
		if errors.Is(err, apperrors.ErrUserNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to remove favorite", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, user)
}

// AddSubscribedPlugin handles POST /users/:user_id/subscribed/:plugin_id and POST /users/:user_id/plugins/:plugin_id
// @Summary Add a subscribed plugin to a user
// @Description Adds the given plugin_id to the user's metadata.subscribed array. Initializes metadata and subscribed if missing, and avoids duplicates.
// @Tags users
// @Accept json
// @Produce json
// @Param user_id path string true "User ID (I/C/D user id, e.g. cis.devops)"
// @Param plugin_id path string true "Plugin ID (UUID)"
// @Success 200 {object} service.UserResponse "Successfully added subscribed plugin"
// @Failure 400 {object} map[string]interface{} "Invalid user_id or plugin_id"
// @Failure 404 {object} map[string]interface{} "User not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Security BearerAuth
// @Router /users/{user_id}/subscribed/{plugin_id} [post]
// @Router /users/{user_id}/plugins/{plugin_id} [post]
func (h *UserHandler) AddSubscribedPlugin(c *gin.Context) {
	userID := c.Param("user_id")
	pluginIDStr := c.Param("plugin_id")

	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id is required"})
		return
	}
	pluginID, err := uuid.Parse(pluginIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid plugin_id"})
		return
	}

	user, err := h.memberService.AddSubscribedPluginByUserID(userID, pluginID)
	if err != nil {
		if errors.Is(err, apperrors.ErrUserNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to add subscribed plugin", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, user)
}

// RemoveSubscribedPlugin handles DELETE /users/:user_id/subscribed/:plugin_id and DELETE /users/:user_id/plugins/:plugin_id
// @Summary Remove a subscribed plugin from a user
// @Description Removes the given plugin_id from the user's metadata.subscribed array. Initializes metadata if missing. Idempotent if plugin not present.
// @Tags users
// @Accept json
// @Produce json
// @Param user_id path string true "User ID (I/C/D user id, e.g. cis.devops)"
// @Param plugin_id path string true "Plugin ID (UUID)"
// @Success 200 {object} service.UserResponse "Successfully removed subscribed plugin"
// @Failure 400 {object} map[string]interface{} "Invalid user_id or plugin_id"
// @Failure 404 {object} map[string]interface{} "User not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Security BearerAuth
// @Router /users/{user_id}/subscribed/{plugin_id} [delete]
// @Router /users/{user_id}/plugins/{plugin_id} [delete]
func (h *UserHandler) RemoveSubscribedPlugin(c *gin.Context) {
	userID := c.Param("user_id")
	pluginIDStr := c.Param("plugin_id")

	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id is required"})
		return
	}
	pluginID, err := uuid.Parse(pluginIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid plugin_id"})
		return
	}

	user, err := h.memberService.RemoveSubscribedPluginByUserID(userID, pluginID)
	if err != nil {
		if errors.Is(err, apperrors.ErrUserNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to remove subscribed plugin", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, user)
}
