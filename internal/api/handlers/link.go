package handlers

import (
	"net/http"
	"errors"

	"developer-portal-backend/internal/auth"
	"developer-portal-backend/internal/service"
	apperrors "developer-portal-backend/internal/errors"
	"developer-portal-backend/internal/logger"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// LinkHandler handles HTTP requests for links
type LinkHandler struct {
	linkService service.LinkServiceInterface
}

// NewLinkHandler creates a new link handler
func NewLinkHandler(linkService service.LinkServiceInterface) *LinkHandler {
	return &LinkHandler{
		linkService: linkService,
	}
}

 // ListLinks handles GET /links?owner=<owner_name>
 // @Summary List links by owner name
 // @Description Returns all links owned by the user with the given owner name (user_id). Example: owner=cis.devops will return all links created in the initial data.
 // @Tags links
 // @Accept json
 // @Produce json
 // @Param owner query string false "Owner name (user_id). Defaults to 'cis.devops' if not provided" example(cis.devops)
 // @Success 200 {array} service.LinkResponse "Successfully retrieved links"
 // @Failure 400 {object} map[string]interface{} "Missing or invalid owner name"
 // @Failure 404 {object} map[string]interface{} "Owner user not found"
 // @Failure 500 {object} map[string]interface{} "Internal server error"
 // @Security BearerAuth
 // @Router /links [get]
func (h *LinkHandler) ListLinks(c *gin.Context) {
	ownerUserID := c.Query("owner")
	if ownerUserID == "" {
		ownerUserID = "cis.devops"
	}

	// Get logged-in username from token (set by auth middleware)
	viewerName, _ := auth.GetUsername(c)

	var (
		links []service.LinkResponse
		err   error
	)

	if viewerName != "" {
		// Prefer enriched response that marks favorites based on viewer's metadata
		links, err = h.linkService.GetByOwnerUserIDWithViewer(ownerUserID, viewerName)
	} else {
		// Fallback (shouldn't happen due to RequireAuth) to non-enriched response
		links, err = h.linkService.GetByOwnerUserID(ownerUserID)
	}

	if err != nil {
		// Distinguish between not found and other errors by simple string check
		if links == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get links", "details": err.Error()})
		return
	}

	// Return array of links (without audit or owner fields)
	c.JSON(http.StatusOK, links)
}

// CreateLink handles POST /links
// @Summary Create a new link
// @Description Creates a new link. Title will mirror name. Validates owner (must be existing user or team) and category_id exists. Tags are optional.
// @Description created_by is derived from the bearer token 'username' claim and is NOT required in the payload.
// @Tags links
// @Accept json
// @Produce json
// @Param link body service.CreateLinkRequest true "Link data"
// @Success 201 {object} service.LinkResponse "Successfully created link"
// @Failure 400 {object} map[string]interface{} "Invalid request or validation failed"
// @Failure 401 {object} map[string]interface{} "Authentication required"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Security BearerAuth
// @Router /links [post]
func (h *LinkHandler) CreateLink(c *gin.Context) {
	var req service.CreateLinkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Populate created_by from bearer token username
	if username, ok := auth.GetUsername(c); ok && username != "" {
		req.CreatedBy = username
	} else {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing username in token"})
		return
	}

	link, err := h.linkService.CreateLink(&req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, link)
}

// UpdateLink handles PUT /links/:id
// @Summary Update a link by ID
// @Description Updates an existing link. Title will mirror name. Validates category_id exists. Tags are optional. Owner field cannot be changed.
// @Description updated_by is derived from the bearer token 'username' claim and is NOT required in the payload.
// @Tags links
// @Accept json
// @Produce json
// @Param id path string true "Link ID (UUID)"
// @Param link body service.UpdateLinkRequest true "Link data"
// @Success 200 {object} service.LinkResponse "Successfully updated link"
// @Failure 400 {object} map[string]interface{} "Invalid request or validation failed"
// @Failure 401 {object} map[string]interface{} "Authentication required"
// @Failure 404 {object} map[string]interface{} "Link not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Security BearerAuth
// @Router /links/{id} [put]
func (h *LinkHandler) UpdateLink(c *gin.Context) {
	log := logger.FromGinContext(c)

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		log.WithField("link_id", idStr).Warn("Link update failed: invalid link ID")
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid link ID"})
		return
	}

	var req service.UpdateLinkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.WithFields(map[string]interface{}{
			"link_id": id.String(),
			"error":   err.Error(),
		}).Warn("Link update failed: invalid request body")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Populate updated_by from bearer token username
	username, ok := auth.GetUsername(c)
	if !ok || username == "" {
		log.WithField("link_id", id.String()).Warn("Link update failed: missing username in token")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing username in token"})
		return
	}
	req.UpdatedBy = username

	log.WithFields(map[string]interface{}{
		"link_id":     id.String(),
		"link_name":   req.Name,
		"url":         req.URL,
		"category_id": req.CategoryID,
		"updated_by":  req.UpdatedBy,
	}).Info("Updating link")

	link, err := h.linkService.UpdateLink(id, &req)
	if err != nil {
		// Handle specific error types using error constants
		if errors.Is(err, apperrors.ErrLinkNotFound) {
			log.WithFields(map[string]interface{}{
				"link_id": id.String(),
				"error":   err.Error(),
			}).Warn("Link update failed: link not found")
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		if errors.Is(err, apperrors.ErrUserNotFound) {
			log.WithFields(map[string]interface{}{
				"link_id": id.String(),
				"error":   err.Error(),
			}).Warn("Link update failed: user not found")
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if errors.Is(err, apperrors.ErrCategoryNotFound) {
			log.WithFields(map[string]interface{}{
				"link_id": id.String(),
				"error":   err.Error(),
			}).Warn("Link update failed: category not found")
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if apperrors.IsValidation(err) {
			log.WithFields(map[string]interface{}{
				"link_id": id.String(),
				"error":   err.Error(),
			}).Warn("Link update failed: validation error")
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		// Generic error
		log.WithFields(map[string]interface{}{
			"link_id": id.String(),
			"error":   err.Error(),
		}).Error("Link update failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update link"})
		return
	}

	log.WithFields(map[string]interface{}{
		"link_id":   link.ID,
		"link_name": link.Name,
	}).Info("Link updated successfully")

	c.JSON(http.StatusOK, link)
}

// DeleteLink handles DELETE /links/:id
// @Summary Delete a link by ID
// @Description Deletes a link from the links table by the given UUID
// @Tags links
// @Accept json
// @Produce json
// @Param id path string true "Link ID (UUID)"
// @Success 204 "Successfully deleted link"
// @Failure 400 {object} map[string]interface{} "Invalid link ID"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Security BearerAuth
// @Router /links/{id} [delete]
func (h *LinkHandler) DeleteLink(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid link ID"})
		return
	}

	if err := h.linkService.DeleteLink(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete link", "details": err.Error()})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}
