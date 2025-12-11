package handlers

import (
	"mime/multipart"
	"net/http"

	"developer-portal-backend/internal/errors"
	"developer-portal-backend/internal/logger"
	"developer-portal-backend/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

// AICoreHandler handles HTTP requests for AI Core operations
type AICoreHandler struct {
	aicoreService service.AICoreServiceInterface
	validator     *validator.Validate
}

// NewAICoreHandler creates a new AI Core handler
func NewAICoreHandler(aicoreService service.AICoreServiceInterface, validator *validator.Validate) *AICoreHandler {
	return &AICoreHandler{
		aicoreService: aicoreService,
		validator:     validator,
	}
}

// handleAICoreError handles common AI Core service errors and returns appropriate HTTP responses
func (h *AICoreHandler) handleAICoreError(c *gin.Context, err error) {
	switch {
	case errors.IsAuthentication(err):
		c.JSON(http.StatusUnauthorized, gin.H{"error": errors.ErrAuthenticationRequired.Error()})
	case errors.IsAuthorization(err):
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
	case errors.IsConfiguration(err):
		c.JSON(http.StatusForbidden, gin.H{"error": errors.ErrAICoreCredentialsNotConfigured.Message})
	case errors.IsNotFound(err):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
}

// GetDeployments handles GET /ai-core/deployments
// @Summary Get AI Core deployments
// @Description Get all deployments from AI Core for the authenticated user's team
// @Tags ai-core
// @Accept json
// @Produce json
// @Success 200 {object} service.AICoreDeploymentsResponse "Successfully retrieved deployments"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 403 {object} map[string]interface{} "User not assigned to team or team credentials not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Security BearerAuth
// @Router /ai-core/deployments [get]
func (h *AICoreHandler) GetDeployments(c *gin.Context) {
	deployments, err := h.aicoreService.GetDeployments(c)
	if err != nil {
		logger.FromGinContext(c).WithField("handler", "GetDeployments").
			Errorf("AI Core: GetDeployments failed: %v", err)
		h.handleAICoreError(c, err)
		return
	}

	c.JSON(http.StatusOK, deployments)
}

// GetModels handles GET /ai-core/models
// @Summary Get AI Core models
// @Description Get all available models from AI Core for a specific scenario
// @Tags ai-core
// @Accept json
// @Produce json
// @Param scenarioId query string true "Scenario ID to get models for"
// @Success 200 {object} service.AICoreModelsResponse "Successfully retrieved models"
// @Failure 400 {object} map[string]interface{} "Bad request - missing scenarioId parameter"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 403 {object} map[string]interface{} "User not assigned to team or team credentials not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Security BearerAuth
// @Router /ai-core/models [get]
func (h *AICoreHandler) GetModels(c *gin.Context) {
	scenarioID := c.Query("scenarioId")
	if scenarioID == "" {
		logger.FromGinContext(c).WithField("handler", "GetModels").
			Warn("AI Core: Missing required scenarioId parameter")
		c.JSON(http.StatusBadRequest, gin.H{"error": errors.ErrMissingScenarioID.Error()})
		return
	}

	models, err := h.aicoreService.GetModels(c, scenarioID)
	if err != nil {
		logger.FromGinContext(c).WithFields(map[string]interface{}{
			"handler":     "GetModels",
			"scenario_id": scenarioID,
		}).Errorf("AI Core: GetModels failed: %v", err)
		h.handleAICoreError(c, err)
		return
	}

	c.JSON(http.StatusOK, models)
}

// GetConfigurations handles GET /ai-core/configurations
// @Summary Get AI Core configurations
// @Description Get all configurations from AI Core for the authenticated user's team
// @Tags ai-core
// @Accept json
// @Produce json
// @Success 200 {object} service.AICoreConfigurationsResponse "Successfully retrieved configurations"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 403 {object} map[string]interface{} "User not assigned to team or team credentials not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Security BearerAuth
// @Router /ai-core/configurations [get]
func (h *AICoreHandler) GetConfigurations(c *gin.Context) {
	configurations, err := h.aicoreService.GetConfigurations(c)
	if err != nil {
		h.handleAICoreError(c, err)
		return
	}

	c.JSON(http.StatusOK, configurations)
}

// CreateConfiguration handles POST /ai-core/configurations
// @Summary Create AI Core configuration
// @Description Create a new configuration in AI Core for the authenticated user's team
// @Tags ai-core
// @Accept json
// @Produce json
// @Param configuration body service.AICoreConfigurationRequest true "Configuration data"
// @Success 201 {object} service.AICoreConfigurationResponse "Successfully created configuration"
// @Failure 400 {object} map[string]interface{} "Invalid request body"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 403 {object} map[string]interface{} "User not assigned to team or team credentials not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Security BearerAuth
// @Router /ai-core/configurations [post]
func (h *AICoreHandler) CreateConfiguration(c *gin.Context) {
	var req service.AICoreConfigurationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate the request
	if err := h.validator.Struct(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	configuration, err := h.aicoreService.CreateConfiguration(c, &req)
	if err != nil {
		h.handleAICoreError(c, err)
		return
	}

	c.JSON(http.StatusCreated, configuration)
}

// CreateDeployment handles POST /ai-core/deployments
// @Summary Create AI Core deployment
// @Description Create a new deployment in AI Core using either an existing configuration ID or by creating a new configuration first
// @Tags ai-core
// @Accept json
// @Produce json
// @Param deployment body service.AICoreDeploymentRequest true "Deployment data - either configurationId or configurationRequest must be provided"
// @Success 202 {object} service.AICoreDeploymentResponse "Successfully scheduled deployment"
// @Failure 400 {object} map[string]interface{} "Invalid request body"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 403 {object} map[string]interface{} "User not assigned to team or team credentials not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Security BearerAuth
// @Router /ai-core/deployments [post]
func (h *AICoreHandler) CreateDeployment(c *gin.Context) {
	var req service.AICoreDeploymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.FromGinContext(c).WithFields(map[string]interface{}{
			"handler": "CreateDeployment",
			"error":   err.Error(),
		}).Warn("AI Core: Invalid request body")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate that either configurationId or configurationRequest is provided, but not both
	if req.ConfigurationID == nil && req.ConfigurationRequest == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": errors.ErrMissingConfigurationInput.Error()})
		return
	}
	if req.ConfigurationID != nil && req.ConfigurationRequest != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": errors.ErrBothConfigurationInputs.Error()})
		return
	}

	// If configurationRequest is provided, validate its required fields
	if req.ConfigurationRequest != nil {
		if err := h.validator.Struct(req.ConfigurationRequest); err != nil {
			logger.FromGinContext(c).WithFields(map[string]interface{}{
				"handler": "CreateDeployment",
				"error":   err.Error(),
			}).Warn("AI Core: Configuration request validation failed")
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}

	deployment, err := h.aicoreService.CreateDeployment(c, &req)
	if err != nil {
		logger.FromGinContext(c).WithField("handler", "CreateDeployment").
			Errorf("AI Core: CreateDeployment failed: %v", err)
		h.handleAICoreError(c, err)
		return
	}

	c.JSON(http.StatusAccepted, deployment)
}

// UpdateDeployment handles PATCH /ai-core/deployments/{deploymentId}
// @Summary Update AI Core deployment
// @Description Update target status or configuration of a deployment in AI Core
// @Tags ai-core
// @Accept json
// @Produce json
// @Param deploymentId path string true "Deployment ID"
// @Param modification body service.AICoreDeploymentModificationRequest true "Deployment modification data"
// @Success 202 {object} service.AICoreDeploymentModificationResponse "Successfully scheduled deployment modification"
// @Failure 400 {object} map[string]interface{} "Invalid request body"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 403 {object} map[string]interface{} "User not assigned to team or team credentials not found"
// @Failure 404 {object} map[string]interface{} "Deployment not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Security BearerAuth
// @Router /ai-core/deployments/{deploymentId} [patch]
func (h *AICoreHandler) UpdateDeployment(c *gin.Context) {
	deploymentID := c.Param("deploymentId")
	if deploymentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": errors.ErrMissingDeploymentID.Error()})
		return
	}

	var req service.AICoreDeploymentModificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// At least one field should be provided
	if req.TargetStatus == "" && req.ConfigurationID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": errors.ErrMissingTargetStatusOrConfigID.Error()})
		return
	}

	response, err := h.aicoreService.UpdateDeployment(c, deploymentID, &req)
	if err != nil {
		h.handleAICoreError(c, err)
		return
	}

	c.JSON(http.StatusAccepted, response)
}

// GetMe handles GET /ai-core/me
// @Summary Get AI Core user context
// @Description Returns current username and AI instances derived from metadata and team role
// @Tags ai-core
// @Accept json
// @Produce json
// @Success 200 {object} service.AICoreMeResponse "Successfully retrieved AI Core user context"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 403 {object} map[string]interface{} "Forbidden"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Security BearerAuth
// @Router /ai-core/me [get]
func (h *AICoreHandler) GetMe(c *gin.Context) {
	resp, err := h.aicoreService.GetMe(c)
	if err != nil {
		h.handleAICoreError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

// GetDeploymentDetails handles GET /ai-core/deployments/{deploymentId}
// @Summary Get AI Core deployment details
// @Description Get detailed information about a specific deployment from AI Core
// @Tags ai-core
// @Accept json
// @Produce json
// @Param deploymentId path string true "Deployment ID"
// @Success 200 {object} service.AICoreDeploymentDetailsResponse "Successfully retrieved deployment details"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 403 {object} map[string]interface{} "User not assigned to team or team credentials not found"
// @Failure 404 {object} map[string]interface{} "Deployment not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Security BearerAuth
// @Router /ai-core/deployments/{deploymentId} [get]
func (h *AICoreHandler) GetDeploymentDetails(c *gin.Context) {
	deploymentID := c.Param("deploymentId")
	if deploymentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": errors.ErrMissingDeploymentID})
		return
	}

	response, err := h.aicoreService.GetDeploymentDetails(c, deploymentID)
	if err != nil {
		h.handleAICoreError(c, err)
		return
	}

	c.JSON(http.StatusOK, response)
}

// DeleteDeployment handles DELETE /ai-core/deployments/{deploymentId}
// @Summary Delete AI Core deployment
// @Description Mark deployment as deleted in AI Core
// @Tags ai-core
// @Accept json
// @Produce json
// @Param deploymentId path string true "Deployment ID"
// @Success 202 {object} service.AICoreDeploymentDeletionResponse "Successfully scheduled deployment deletion"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 403 {object} map[string]interface{} "User not assigned to team or team credentials not found"
// @Failure 404 {object} map[string]interface{} "Deployment not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Security BearerAuth
// @Router /ai-core/deployments/{deploymentId} [delete]
func (h *AICoreHandler) DeleteDeployment(c *gin.Context) {
	deploymentID := c.Param("deploymentId")
	if deploymentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": errors.ErrMissingDeploymentID})
		return
	}

	response, err := h.aicoreService.DeleteDeployment(c, deploymentID)
	if err != nil {
		h.handleAICoreError(c, err)
		return
	}

	c.JSON(http.StatusAccepted, response)
}

// ChatInference handles POST /ai-core/chat/inference
// @Summary Chat inference with AI Core deployed model
// @Description Send chat messages to a deployed model and get AI responses (supports streaming)
// @Tags ai-core
// @Accept json
// @Produce json
// @Produce text/event-stream
// @Param request body service.AICoreInferenceRequest true "Chat inference request"
// @Success 200 {object} service.AICoreInferenceResponse "Successfully received inference response"
// @Failure 400 {object} map[string]interface{} "Invalid request body"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 403 {object} map[string]interface{} "User not assigned to team or team credentials not found"
// @Failure 404 {object} map[string]interface{} "Deployment not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Security BearerAuth
// @Router /ai-core/chat/inference [post]
func (h *AICoreHandler) ChatInference(c *gin.Context) {
	var req service.AICoreInferenceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate the request
	if err := h.validator.Struct(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// If streaming is requested, use Server-Sent Events (SSE)
	if req.Stream {
		h.streamChatInference(c, &req)
		return
	}

	// Non-streaming response
	response, err := h.aicoreService.ChatInference(c, &req)
	if err != nil {
		h.handleAICoreError(c, err)
		return
	}

	c.JSON(http.StatusOK, response)
}

// streamChatInference handles streaming responses using Server-Sent Events (SSE)
func (h *AICoreHandler) streamChatInference(c *gin.Context, req *service.AICoreInferenceRequest) {
	// Set headers for SSE
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no") // Disable nginx buffering

	// Use the service to stream the response
	err := h.aicoreService.ChatInferenceStream(c, req, c.Writer)
	if err != nil {
		// For streaming errors, we need to send an error event
		logger.FromGinContext(c).WithField("handler", "streamChatInference").
			Errorf("AI Core: Streaming inference failed: %v", err)
		// Send error as SSE event
		c.SSEvent("error", gin.H{"error": err.Error()})
	}
}

// UploadAttachment handles POST /ai-core/upload
// @Summary Upload file attachments for AI inference
// @Description Upload multiple files for use in multimodal AI requests (combined max 5MB)
// @Tags ai-core
// @Accept multipart/form-data
// @Produce json
// @Param files formData file true "Files to upload (multiple allowed)"
// @Success 200 {object} map[string]interface{} "Successfully uploaded files"
// @Failure 400 {object} map[string]interface{} "Invalid files or size"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Security BearerAuth
// @Router /ai-core/upload [post]
func (h *AICoreHandler) UploadAttachment(c *gin.Context) {
	// Maximum combined size for all files: 5MB
	const maxTotalSize = 5 << 20 // 5 MB

	// Parse multipart form with the size limit
	if err := c.Request.ParseMultipartForm(maxTotalSize); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": errors.ErrFileSizeTooLarge.Error()})
		return
	}

	// Get the multipart form
	form := c.Request.MultipartForm
	if form == nil || form.File == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": errors.ErrNoFilesProvided.Error()})
		return
	}

	// Get all files from the form (supports both "file" and "files" field names)
	var fileHeaders []*multipart.FileHeader
	if files, ok := form.File["files"]; ok {
		fileHeaders = append(fileHeaders, files...)
	}
	if file, ok := form.File["file"]; ok {
		fileHeaders = append(fileHeaders, file...)
	}

	if len(fileHeaders) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": errors.ErrNoFilesProvided.Error()})
		return
	}

	// Calculate combined size
	var totalSize int64
	for _, header := range fileHeaders {
		totalSize += header.Size
	}

	if totalSize > maxTotalSize {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":     errors.ErrCombinedFileSizeExceeds.Error(),
			"totalSize": totalSize,
			"maxSize":   maxTotalSize,
		})
		return
	}

	// Process all files
	uploadedFiles := make([]map[string]interface{}, 0, len(fileHeaders))

	for _, header := range fileHeaders {
		file, err := header.Open()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to open file: " + header.Filename})
			return
		}
		defer file.Close() // Ensure file is always closed

		// Process the upload using the service
		response, err := h.aicoreService.UploadAttachment(c, file, header)
		if err != nil {
			h.handleAICoreError(c, err)
			return
		}

		uploadedFiles = append(uploadedFiles, response)
	}

	c.JSON(http.StatusOK, gin.H{
		"files":     uploadedFiles,
		"count":     len(uploadedFiles),
		"totalSize": totalSize,
	})
}
