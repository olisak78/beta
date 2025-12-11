package handlers

import (
	"net/http"

	"developer-portal-backend/internal/auth"
	"developer-portal-backend/internal/errors"
	"developer-portal-backend/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// DocumentationHandler handles HTTP requests for documentations
type DocumentationHandler struct {
	docService service.DocumentationServiceInterface
}

// NewDocumentationHandler creates a new documentation handler
func NewDocumentationHandler(docService service.DocumentationServiceInterface) *DocumentationHandler {
	return &DocumentationHandler{
		docService: docService,
	}
}

// CreateDocumentation handles POST /documentations
// @Summary Create a new documentation
// @Description Creates a new documentation endpoint for a team. The URL should be a valid GitHub URL.
// @Description created_by is derived from the bearer token 'username' claim.
// @Tags documentations
// @Accept json
// @Produce json
// @Param documentation body service.CreateDocumentationRequest true "Documentation data"
// @Success 201 {object} service.DocumentationResponse "Successfully created documentation"
// @Failure 400 {object} map[string]interface{} "Invalid request or validation failed"
// @Failure 401 {object} map[string]interface{} "Authentication required"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Security BearerAuth
// @Router /documentations [post]
func (h *DocumentationHandler) CreateDocumentation(c *gin.Context) {
	var req service.CreateDocumentationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Populate created_by from bearer token username
	if username, ok := auth.GetUsername(c); ok && username != "" {
		req.CreatedBy = username
	} else {
		c.JSON(http.StatusUnauthorized, gin.H{"error": errors.ErrMissingUsernameInToken.Error()})
		return
	}

	doc, err := h.docService.CreateDocumentation(&req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, doc)
}

// GetDocumentationByID handles GET /documentations/:id
// @Summary Get a documentation by ID
// @Description Retrieves a documentation by its UUID
// @Tags documentations
// @Accept json
// @Produce json
// @Param id path string true "Documentation ID (UUID)"
// @Success 200 {object} service.DocumentationResponse "Successfully retrieved documentation"
// @Failure 400 {object} map[string]interface{} "Invalid documentation ID"
// @Failure 404 {object} map[string]interface{} "Documentation not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Security BearerAuth
// @Router /documentations/{id} [get]
func (h *DocumentationHandler) GetDocumentationByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": errors.ErrInvalidDocumentationID.Error()})
		return
	}

	doc, err := h.docService.GetDocumentationByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": errors.ErrDocumentationNotFound.Error()})
		return
	}

	c.JSON(http.StatusOK, doc)
}

// GetDocumentationsByTeamID handles GET /teams/:id/documentations
// @Summary List documentations for a team
// @Description Returns all documentations for a specific team
// @Tags documentations
// @Accept json
// @Produce json
// @Param id path string true "Team ID (UUID)"
// @Success 200 {array} service.DocumentationResponse "Successfully retrieved documentations"
// @Failure 400 {object} map[string]interface{} "Invalid team ID"
// @Failure 404 {object} map[string]interface{} "Team not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Security BearerAuth
// @Router /teams/{id}/documentations [get]
func (h *DocumentationHandler) GetDocumentationsByTeamID(c *gin.Context) {
	teamIDStr := c.Param("id")
	teamID, err := uuid.Parse(teamIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": errors.ErrInvalidTeamID.Error()})
		return
	}

	docs, err := h.docService.GetDocumentationsByTeamID(teamID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, docs)
}

// UpdateDocumentation handles PATCH /documentations/:id
// @Summary Update a documentation
// @Description Updates an existing documentation. All fields are optional.
// @Description updated_by is derived from the bearer token 'username' claim.
// @Tags documentations
// @Accept json
// @Produce json
// @Param id path string true "Documentation ID (UUID)"
// @Param documentation body service.UpdateDocumentationRequest true "Documentation update data"
// @Success 200 {object} service.DocumentationResponse "Successfully updated documentation"
// @Failure 400 {object} map[string]interface{} "Invalid request or validation failed"
// @Failure 401 {object} map[string]interface{} "Authentication required"
// @Failure 404 {object} map[string]interface{} "Documentation not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Security BearerAuth
// @Router /documentations/{id} [patch]
func (h *DocumentationHandler) UpdateDocumentation(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": errors.ErrInvalidDocumentationID.Error()})
		return
	}

	var req service.UpdateDocumentationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Populate updated_by from bearer token username
	if username, ok := auth.GetUsername(c); ok && username != "" {
		req.UpdatedBy = username
	} else {
		c.JSON(http.StatusUnauthorized, gin.H{"error": errors.ErrMissingUsernameInToken.Error()})
		return
	}

	doc, err := h.docService.UpdateDocumentation(id, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, doc)
}

// DeleteDocumentation handles DELETE /documentations/:id
// @Summary Delete a documentation by ID
// @Description Deletes a documentation from the documentations table by the given UUID
// @Tags documentations
// @Accept json
// @Produce json
// @Param id path string true "Documentation ID (UUID)"
// @Success 204 "Successfully deleted documentation"
// @Failure 400 {object} map[string]interface{} "Invalid documentation ID"
// @Failure 404 {object} map[string]interface{} "Documentation not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Security BearerAuth
// @Router /documentations/{id} [delete]
func (h *DocumentationHandler) DeleteDocumentation(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": errors.ErrInvalidDocumentationID.Error()})
		return
	}

	if err := h.docService.DeleteDocumentation(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": errors.ErrFailedToDeleteDocumentation.Error(), "details": err.Error()})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}
