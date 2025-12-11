package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"developer-portal-backend/internal/errors"
	"developer-portal-backend/internal/mocks"
	"developer-portal-backend/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

type AICoreHandlerTestSuite struct {
	suite.Suite
	ctrl          *gomock.Controller
	handler       *AICoreHandler
	aicoreService *mocks.MockAICoreServiceInterface
	validator     *validator.Validate
	router        *gin.Engine
}

func (suite *AICoreHandlerTestSuite) SetupTest() {
	gin.SetMode(gin.TestMode)

	suite.ctrl = gomock.NewController(suite.T())
	suite.aicoreService = mocks.NewMockAICoreServiceInterface(suite.ctrl)
	suite.validator = validator.New()
	suite.handler = NewAICoreHandler(suite.aicoreService, suite.validator)

	suite.router = gin.New()
	suite.router.GET("/ai-core/deployments", suite.handler.GetDeployments)
	suite.router.GET("/ai-core/deployments/:deploymentId", suite.handler.GetDeploymentDetails)
	suite.router.GET("/ai-core/models", suite.handler.GetModels)
	suite.router.GET("/ai-core/configurations", suite.handler.GetConfigurations)
	suite.router.POST("/ai-core/configurations", suite.handler.CreateConfiguration)
	suite.router.POST("/ai-core/deployments", suite.handler.CreateDeployment)
	suite.router.PATCH("/ai-core/deployments/:deploymentId", suite.handler.UpdateDeployment)
	suite.router.DELETE("/ai-core/deployments/:deploymentId", suite.handler.DeleteDeployment)
}

func (suite *AICoreHandlerTestSuite) TearDownTest() {
	suite.ctrl.Finish()
}

func (suite *AICoreHandlerTestSuite) TestGetDeployments_Success() {
	// Setup
	expectedResponse := &service.AICoreDeploymentsResponse{
		Count: 2,
		Deployments: []service.AICoreTeamDeployments{
			{
				Team: "team-alpha",
				Deployments: []service.AICoreDeployment{
					{
						ID:              "deployment-1",
						ConfigurationID: "config-1",
						Status:          "RUNNING",
						StatusMessage:   "Deployment is running",
						DeploymentURL:   "https://api.example.com/v1/deployments/deployment-1",
						CreatedAt:       "2023-01-01T00:00:00Z",
						ModifiedAt:      "2023-01-01T01:00:00Z",
					},
				},
			},
			{
				Team: "team-beta",
				Deployments: []service.AICoreDeployment{
					{
						ID:              "deployment-2",
						ConfigurationID: "config-2",
						Status:          "STOPPED",
						StatusMessage:   "Deployment is stopped",
						DeploymentURL:   "https://api.example.com/v1/deployments/deployment-2",
						CreatedAt:       "2023-01-01T00:00:00Z",
						ModifiedAt:      "2023-01-01T02:00:00Z",
					},
				},
			},
		},
	}

	suite.aicoreService.EXPECT().GetDeployments(gomock.Any()).Return(expectedResponse, nil)

	// Execute
	req := httptest.NewRequest("GET", "/ai-core/deployments", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Assert
	suite.Equal(http.StatusOK, w.Code)

	var response service.AICoreDeploymentsResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err)
	suite.Equal(2, response.Count)
	suite.Len(response.Deployments, 2)
	suite.Equal("team-alpha", response.Deployments[0].Team)
	suite.Equal("team-beta", response.Deployments[1].Team)
	suite.Equal("deployment-1", response.Deployments[0].Deployments[0].ID)
	suite.Equal("RUNNING", response.Deployments[0].Deployments[0].Status)
}

func (suite *AICoreHandlerTestSuite) TestGetDeployments_PartialCredentials_Success() {
	// Setup - Only one team has credentials, the other is skipped
	expectedResponse := &service.AICoreDeploymentsResponse{
		Count: 1,
		Deployments: []service.AICoreTeamDeployments{
			{
				Team: "team-alpha",
				Deployments: []service.AICoreDeployment{
					{
						ID:              "deployment-1",
						ConfigurationID: "config-1",
						Status:          "RUNNING",
						StatusMessage:   "Deployment is running",
						DeploymentURL:   "https://api.example.com/v1/deployments/deployment-1",
						CreatedAt:       "2023-01-01T00:00:00Z",
						ModifiedAt:      "2023-01-01T01:00:00Z",
					},
				},
			},
		},
	}

	suite.aicoreService.EXPECT().GetDeployments(gomock.Any()).Return(expectedResponse, nil)

	// Execute
	req := httptest.NewRequest("GET", "/ai-core/deployments", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Assert
	suite.Equal(http.StatusOK, w.Code)

	var response service.AICoreDeploymentsResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err)
	suite.Equal(1, response.Count)
	suite.Len(response.Deployments, 1) // Only team with credentials returned
	suite.Equal("team-alpha", response.Deployments[0].Team)
}

func (suite *AICoreHandlerTestSuite) TestGetDeployments_EmptyResult_Success() {
	// Setup
	expectedResponse := &service.AICoreDeploymentsResponse{
		Count:       0,
		Deployments: []service.AICoreTeamDeployments{},
	}

	suite.aicoreService.EXPECT().GetDeployments(gomock.Any()).Return(expectedResponse, nil)

	// Execute
	req := httptest.NewRequest("GET", "/ai-core/deployments", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Assert
	suite.Equal(http.StatusOK, w.Code)

	var response service.AICoreDeploymentsResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err)
	suite.Equal(0, response.Count)
	suite.Len(response.Deployments, 0)
}

func (suite *AICoreHandlerTestSuite) TestGetDeployments_AuthenticationError() {
	// Setup
	suite.aicoreService.EXPECT().GetDeployments(gomock.Any()).Return(nil, errors.ErrUserEmailNotFound)

	// Execute
	req := httptest.NewRequest("GET", "/ai-core/deployments", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Assert
	suite.Equal(http.StatusUnauthorized, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err)
	suite.Equal(errors.ErrAuthenticationRequired.Message, response["error"])
}

func (suite *AICoreHandlerTestSuite) TestGetDeployments_UserNotFoundError() {
	// Setup
	suite.aicoreService.EXPECT().GetDeployments(gomock.Any()).Return(nil, errors.ErrUserNotFoundInDB)

	// Execute
	req := httptest.NewRequest("GET", "/ai-core/deployments", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Assert
	suite.Equal(http.StatusForbidden, w.Code) // AuthorizationError returns 403

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err)
	suite.Equal(errors.ErrUserNotFoundInDB.Message, response["error"]) // Actual error message
}

func (suite *AICoreHandlerTestSuite) TestGetDeployments_UserNotAssignedToTeamError() {
	// Setup
	suite.aicoreService.EXPECT().GetDeployments(gomock.Any()).Return(nil, errors.ErrUserNotAssignedToTeam)

	// Execute
	req := httptest.NewRequest("GET", "/ai-core/deployments", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Assert
	suite.Equal(http.StatusForbidden, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err)
	suite.Equal(errors.ErrUserNotAssignedToTeam.Message, response["error"])
}

func (suite *AICoreHandlerTestSuite) TestGetDeployments_NoCredentialsError() {
	// Setup
	credentialsError := errors.NewAICoreCredentialsNotFoundError("team-alpha")
	suite.aicoreService.EXPECT().GetDeployments(gomock.Any()).Return(nil, credentialsError)

	// Execute
	req := httptest.NewRequest("GET", "/ai-core/deployments", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Assert
	suite.Equal(http.StatusForbidden, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err)
	suite.Equal(errors.ErrAICoreCredentialsNotConfigured.Message, response["error"])
}

func (suite *AICoreHandlerTestSuite) TestGetDeployments_InternalServerError() {
	// Setup
	suite.aicoreService.EXPECT().GetDeployments(gomock.Any()).Return(nil, errors.ErrInternalError)

	// Execute
	req := httptest.NewRequest("GET", "/ai-core/deployments", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Assert
	suite.Equal(http.StatusInternalServerError, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err)
	suite.Equal(errors.ErrInternalError.Error(), response["error"])
}

func (suite *AICoreHandlerTestSuite) TestGetModels_Success() {
	// Setup
	scenarioID := "foundation-models"
	expectedResponse := &service.AICoreModelsResponse{
		Count: 1,
		Resources: []service.AICoreModel{
			{
				Model:        "gpt-4",
				ExecutableID: "exec-1",
				Description:  "GPT-4 model",
				DisplayName:  "GPT-4",
				AccessType:   "public",
				Provider:     "openai",
			},
		},
	}

	suite.aicoreService.EXPECT().GetModels(gomock.Any(), scenarioID).Return(expectedResponse, nil)

	// Execute
	req := httptest.NewRequest("GET", fmt.Sprintf("/ai-core/models?scenarioId=%s", scenarioID), nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Assert
	suite.Equal(http.StatusOK, w.Code)

	var response service.AICoreModelsResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err)
	suite.Equal(1, response.Count)
	suite.Len(response.Resources, 1)
	suite.Equal("gpt-4", response.Resources[0].Model)
}

func (suite *AICoreHandlerTestSuite) TestGetModels_MissingScenarioID() {
	// Execute
	req := httptest.NewRequest("GET", "/ai-core/models", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Assert
	suite.Equal(http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err)
	suite.Equal(errors.ErrMissingScenarioID.Error(), response["error"])
}

func (suite *AICoreHandlerTestSuite) TestCreateConfiguration_Success() {
	// Setup
	requestBody := service.AICoreConfigurationRequest{
		Name:         "test-config",
		ExecutableID: "exec-1",
		ScenarioID:   "foundation-models",
	}

	expectedResponse := &service.AICoreConfigurationResponse{
		ID:      "config-1",
		Message: "Configuration created successfully",
	}

	suite.aicoreService.EXPECT().CreateConfiguration(gomock.Any(), &requestBody).Return(expectedResponse, nil)

	// Execute
	body, _ := json.Marshal(requestBody)
	req := httptest.NewRequest("POST", "/ai-core/configurations", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Assert
	suite.Equal(http.StatusCreated, w.Code)

	var response service.AICoreConfigurationResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err)
	suite.Equal("config-1", response.ID)
	suite.Equal("Configuration created successfully", response.Message)
}

func (suite *AICoreHandlerTestSuite) TestCreateConfiguration_InvalidJSON() {
	// Execute
	req := httptest.NewRequest("POST", "/ai-core/configurations", bytes.NewBufferString("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Assert
	suite.Equal(http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err)
	suite.Contains(response["error"].(string), "invalid character")
}

func (suite *AICoreHandlerTestSuite) TestCreateConfiguration_ValidationError() {
	// Setup - missing required fields
	requestBody := service.AICoreConfigurationRequest{
		Name: "test-config",
		// Missing ExecutableID and ScenarioID
	}

	// Execute
	body, _ := json.Marshal(requestBody)
	req := httptest.NewRequest("POST", "/ai-core/configurations", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Assert
	suite.Equal(http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err)
	suite.Contains(response["error"].(string), "required")
}

func (suite *AICoreHandlerTestSuite) TestCreateDeployment_WithConfigurationID_Success() {
	// Setup - Test scenario 1: Direct deployment with configurationId
	configID := "config-1"
	requestBody := service.AICoreDeploymentRequest{
		ConfigurationID: &configID,
		TTL:             "1h",
	}

	expectedResponse := &service.AICoreDeploymentResponse{
		ID:            "deployment-1",
		Message:       "Deployment created successfully",
		DeploymentURL: "https://api.example.com/v1/deployments/deployment-1",
		Status:        "PENDING",
		TTL:           "1h",
	}

	suite.aicoreService.EXPECT().CreateDeployment(gomock.Any(), gomock.Any()).DoAndReturn(
		func(c *gin.Context, req *service.AICoreDeploymentRequest) (*service.AICoreDeploymentResponse, error) {
			if req.ConfigurationID != nil && *req.ConfigurationID == "config-1" && req.ConfigurationRequest == nil {
				return expectedResponse, nil
			}
			return nil, fmt.Errorf("unexpected request")
		})

	// Execute
	body, _ := json.Marshal(requestBody)
	req := httptest.NewRequest("POST", "/ai-core/deployments", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Assert
	suite.Equal(http.StatusAccepted, w.Code)

	var response service.AICoreDeploymentResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err)
	suite.Equal("deployment-1", response.ID)
	suite.Equal("Deployment created successfully", response.Message)
}

func (suite *AICoreHandlerTestSuite) TestCreateDeployment_WithConfigurationRequest_Success() {
	// Setup - Test scenario 2: Deployment with configuration creation
	requestBody := service.AICoreDeploymentRequest{
		ConfigurationRequest: &service.AICoreConfigurationRequest{
			Name:         "my-llm-config",
			ExecutableID: "aicore-llm",
			ScenarioID:   "foundation-models",
			ParameterBindings: []map[string]string{
				{"key": "modelName", "value": "gpt-4"},
				{"key": "modelVersion", "value": "latest"},
			},
		},
		TTL: "2h",
	}

	expectedResponse := &service.AICoreDeploymentResponse{
		ID:            "deployment-2",
		Message:       "Deployment created successfully",
		DeploymentURL: "https://api.example.com/v1/deployments/deployment-2",
		Status:        "PENDING",
		TTL:           "2h",
	}

	suite.aicoreService.EXPECT().CreateDeployment(gomock.Any(), gomock.Any()).DoAndReturn(
		func(c *gin.Context, req *service.AICoreDeploymentRequest) (*service.AICoreDeploymentResponse, error) {
			if req.ConfigurationID == nil && req.ConfigurationRequest != nil &&
				req.ConfigurationRequest.Name == "my-llm-config" &&
				req.ConfigurationRequest.ExecutableID == "aicore-llm" &&
				req.ConfigurationRequest.ScenarioID == "foundation-models" {
				return expectedResponse, nil
			}
			return nil, fmt.Errorf("unexpected request")
		})

	// Execute
	body, _ := json.Marshal(requestBody)
	req := httptest.NewRequest("POST", "/ai-core/deployments", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Assert
	suite.Equal(http.StatusAccepted, w.Code)

	var response service.AICoreDeploymentResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err)
	suite.Equal("deployment-2", response.ID)
	suite.Equal("Deployment created successfully", response.Message)
}

func (suite *AICoreHandlerTestSuite) TestCreateDeployment_BothFieldsProvided_Error() {
	// Setup - Test invalid scenario: both configurationId and configurationRequest provided
	configID := "config-1"
	requestBody := service.AICoreDeploymentRequest{
		ConfigurationID: &configID,
		ConfigurationRequest: &service.AICoreConfigurationRequest{
			Name:         "my-llm-config",
			ExecutableID: "aicore-llm",
			ScenarioID:   "foundation-models",
		},
		TTL: "1h",
	}

	// Execute
	body, _ := json.Marshal(requestBody)
	req := httptest.NewRequest("POST", "/ai-core/deployments", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Assert
	suite.Equal(http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err)
	suite.Equal(errors.ErrBothConfigurationInputs.Error(), response["error"])
}

func (suite *AICoreHandlerTestSuite) TestCreateDeployment_NeitherFieldProvided_Error() {
	// Setup - Test invalid scenario: neither configurationId nor configurationRequest provided
	requestBody := service.AICoreDeploymentRequest{
		TTL: "1h",
	}

	// Execute
	body, _ := json.Marshal(requestBody)
	req := httptest.NewRequest("POST", "/ai-core/deployments", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Assert
	suite.Equal(http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err)
	suite.Equal(errors.ErrMissingConfigurationInput.Error(), response["error"])
}

func (suite *AICoreHandlerTestSuite) TestCreateDeployment_InvalidConfigurationRequest_Error() {
	// Setup - Test invalid scenario: configurationRequest with missing required fields
	requestBody := service.AICoreDeploymentRequest{
		ConfigurationRequest: &service.AICoreConfigurationRequest{
			Name: "my-llm-config",
			// Missing ExecutableID and ScenarioID
		},
		TTL: "1h",
	}

	// Execute
	body, _ := json.Marshal(requestBody)
	req := httptest.NewRequest("POST", "/ai-core/deployments", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Assert
	suite.Equal(http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err)
	suite.Contains(response["error"].(string), "required")
}

func (suite *AICoreHandlerTestSuite) TestUpdateDeployment_Success() {
	// Setup
	deploymentID := "deployment-1"
	requestBody := service.AICoreDeploymentModificationRequest{
		TargetStatus: "STOPPED",
	}

	expectedResponse := &service.AICoreDeploymentModificationResponse{
		ID:           "deployment-1",
		Message:      "Deployment updated successfully",
		Status:       "RUNNING",
		TargetStatus: "STOPPED",
	}

	suite.aicoreService.EXPECT().UpdateDeployment(gomock.Any(), deploymentID, &requestBody).Return(expectedResponse, nil)

	// Execute
	body, _ := json.Marshal(requestBody)
	req := httptest.NewRequest("PATCH", fmt.Sprintf("/ai-core/deployments/%s", deploymentID), bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Assert
	suite.Equal(http.StatusAccepted, w.Code)

	var response service.AICoreDeploymentModificationResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err)
	suite.Equal("deployment-1", response.ID)
	suite.Equal("STOPPED", response.TargetStatus)
}

func (suite *AICoreHandlerTestSuite) TestUpdateDeployment_MissingDeploymentID() {
	// Execute
	requestBody := service.AICoreDeploymentModificationRequest{
		TargetStatus: "STOPPED",
	}
	body, _ := json.Marshal(requestBody)
	req := httptest.NewRequest("PATCH", "/ai-core/deployments/", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Assert
	suite.Equal(http.StatusNotFound, w.Code) // Gin returns 404 for missing path parameters
}

func (suite *AICoreHandlerTestSuite) TestUpdateDeployment_EmptyRequest() {
	// Setup
	deploymentID := "deployment-1"
	requestBody := service.AICoreDeploymentModificationRequest{
		// Both fields empty
	}

	// Execute
	body, _ := json.Marshal(requestBody)
	req := httptest.NewRequest("PATCH", fmt.Sprintf("/ai-core/deployments/%s", deploymentID), bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Assert
	suite.Equal(http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err)
	suite.Equal(errors.ErrMissingTargetStatusOrConfigID.Error(), response["error"])
}

func (suite *AICoreHandlerTestSuite) TestDeleteDeployment_Success() {
	// Setup
	deploymentID := "deployment-1"
	expectedResponse := &service.AICoreDeploymentDeletionResponse{
		ID:      "deployment-1",
		Message: "Deployment deleted successfully",
	}

	suite.aicoreService.EXPECT().DeleteDeployment(gomock.Any(), deploymentID).Return(expectedResponse, nil)

	// Execute
	req := httptest.NewRequest("DELETE", fmt.Sprintf("/ai-core/deployments/%s", deploymentID), nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Assert
	suite.Equal(http.StatusAccepted, w.Code)

	var response service.AICoreDeploymentDeletionResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err)
	suite.Equal("deployment-1", response.ID)
	suite.Equal("Deployment deleted successfully", response.Message)
}

func (suite *AICoreHandlerTestSuite) TestGetDeploymentDetails_Success() {
	// Setup
	deploymentID := "deployment-1"
	expectedResponse := &service.AICoreDeploymentDetailsResponse{
		ID:                "deployment-1",
		DeploymentURL:     "https://api.example.com/v1/deployments/deployment-1",
		ConfigurationID:   "config-1",
		ConfigurationName: "test-config",
		ExecutableID:      "exec-1",
		ScenarioID:        "foundation-models",
		Status:            "RUNNING",
		StatusMessage:     "Deployment is running",
		TargetStatus:      "RUNNING",
		LastOperation:     "CREATE",
		TTL:               "1h",
		CreatedAt:         "2023-01-01T00:00:00Z",
		ModifiedAt:        "2023-01-01T01:00:00Z",
	}

	suite.aicoreService.EXPECT().GetDeploymentDetails(gomock.Any(), deploymentID).Return(expectedResponse, nil)

	// Execute
	req := httptest.NewRequest("GET", fmt.Sprintf("/ai-core/deployments/%s", deploymentID), nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Assert
	suite.Equal(http.StatusOK, w.Code)

	var response service.AICoreDeploymentDetailsResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err)
	suite.Equal("deployment-1", response.ID)
	suite.Equal("RUNNING", response.Status)
	suite.Equal("config-1", response.ConfigurationID)
}

func (suite *AICoreHandlerTestSuite) TestGetDeploymentDetails_NotFound() {
	// Setup
	deploymentID := "nonexistent-deployment"
	suite.aicoreService.EXPECT().GetDeploymentDetails(gomock.Any(), deploymentID).Return(nil, errors.ErrAICoreDeploymentNotFound)

	// Execute
	req := httptest.NewRequest("GET", fmt.Sprintf("/ai-core/deployments/%s", deploymentID), nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Assert
	suite.Equal(http.StatusNotFound, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err)
	suite.Contains(response["error"].(string), "not found")
}

func (suite *AICoreHandlerTestSuite) TestGetConfigurations_Success() {
	// Setup
	expectedResponse := &service.AICoreConfigurationsResponse{
		Count: 2,
		Resources: []service.AICoreConfiguration{
			{
				ID:           "config-1",
				Name:         "test-config-1",
				ExecutableID: "exec-1",
				ScenarioID:   "foundation-models",
				CreatedAt:    "2023-01-01T00:00:00Z",
			},
			{
				ID:           "config-2",
				Name:         "test-config-2",
				ExecutableID: "exec-2",
				ScenarioID:   "foundation-models",
				CreatedAt:    "2023-01-02T00:00:00Z",
			},
		},
	}

	suite.aicoreService.EXPECT().GetConfigurations(gomock.Any()).Return(expectedResponse, nil)

	// Execute
	req := httptest.NewRequest("GET", "/ai-core/configurations", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Assert
	suite.Equal(http.StatusOK, w.Code)

	var response service.AICoreConfigurationsResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err)
	suite.Equal(2, response.Count)
	suite.Len(response.Resources, 2)
	suite.Equal("config-1", response.Resources[0].ID)
	suite.Equal("test-config-1", response.Resources[0].Name)
	suite.Equal("foundation-models", response.Resources[0].ScenarioID)
}

func (suite *AICoreHandlerTestSuite) TestGetConfigurations_EmptyResult() {
	// Setup
	expectedResponse := &service.AICoreConfigurationsResponse{
		Count:     0,
		Resources: []service.AICoreConfiguration{},
	}

	suite.aicoreService.EXPECT().GetConfigurations(gomock.Any()).Return(expectedResponse, nil)

	// Execute
	req := httptest.NewRequest("GET", "/ai-core/configurations", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Assert
	suite.Equal(http.StatusOK, w.Code)

	var response service.AICoreConfigurationsResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err)
	suite.Equal(0, response.Count)
	suite.Len(response.Resources, 0)
}

func (suite *AICoreHandlerTestSuite) TestGetConfigurations_AuthenticationError() {
	// Setup
	suite.aicoreService.EXPECT().GetConfigurations(gomock.Any()).Return(nil, errors.ErrUserEmailNotFound)

	// Execute
	req := httptest.NewRequest("GET", "/ai-core/configurations", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Assert
	suite.Equal(http.StatusUnauthorized, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err)
	suite.Equal(errors.ErrAuthenticationRequired.Message, response["error"])
}

func (suite *AICoreHandlerTestSuite) TestGetConfigurations_UserNotAssignedToTeamError() {
	// Setup
	suite.aicoreService.EXPECT().GetConfigurations(gomock.Any()).Return(nil, errors.ErrUserNotAssignedToTeam)

	// Execute
	req := httptest.NewRequest("GET", "/ai-core/configurations", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Assert
	suite.Equal(http.StatusForbidden, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err)
	suite.Equal(errors.ErrUserNotAssignedToTeam.Message, response["error"])
}

func (suite *AICoreHandlerTestSuite) TestGetConfigurations_NoCredentialsError() {
	// Setup
	credentialsError := errors.NewAICoreCredentialsNotFoundError("team-alpha")
	suite.aicoreService.EXPECT().GetConfigurations(gomock.Any()).Return(nil, credentialsError)

	// Execute
	req := httptest.NewRequest("GET", "/ai-core/configurations", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Assert
	suite.Equal(http.StatusForbidden, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err)
	suite.Equal(errors.ErrAICoreCredentialsNotConfigured.Message, response["error"])
}

func (suite *AICoreHandlerTestSuite) TestGetConfigurations_InternalServerError() {
	// Setup
	suite.aicoreService.EXPECT().GetConfigurations(gomock.Any()).Return(nil, errors.ErrInternalError)

	// Execute
	req := httptest.NewRequest("GET", "/ai-core/configurations", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Assert
	suite.Equal(http.StatusInternalServerError, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err)
	suite.Equal(errors.ErrInternalError.Error(), response["error"])
}

func (suite *AICoreHandlerTestSuite) TestGetMe_Success() {
	// Setup
	expectedResponse := &service.AICoreMeResponse{
		User:        "john.doe",
		AIInstances: []string{"team-alpha", "team-beta"},
	}

	suite.aicoreService.EXPECT().GetMe(gomock.Any()).Return(expectedResponse, nil)

	// Execute
	req := httptest.NewRequest("GET", "/ai-core/me", nil)
	w := httptest.NewRecorder()

	// Add route for GetMe
	suite.router.GET("/ai-core/me", suite.handler.GetMe)
	suite.router.ServeHTTP(w, req)

	// Assert
	suite.Equal(http.StatusOK, w.Code)

	var response service.AICoreMeResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err)
	suite.Equal("john.doe", response.User)
	suite.Len(response.AIInstances, 2)
	suite.Equal("team-alpha", response.AIInstances[0])
	suite.Equal("team-beta", response.AIInstances[1])
}

func (suite *AICoreHandlerTestSuite) TestGetMe_AuthenticationError() {
	// Setup
	suite.aicoreService.EXPECT().GetMe(gomock.Any()).Return(nil, errors.ErrUserEmailNotFound)

	// Execute
	req := httptest.NewRequest("GET", "/ai-core/me", nil)
	w := httptest.NewRecorder()

	// Add route for GetMe
	suite.router.GET("/ai-core/me", suite.handler.GetMe)
	suite.router.ServeHTTP(w, req)

	// Assert
	suite.Equal(http.StatusUnauthorized, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err)
	suite.Equal(errors.ErrAuthenticationRequired.Message, response["error"])
}

func (suite *AICoreHandlerTestSuite) TestGetMe_UserNotFoundError() {
	// Setup
	suite.aicoreService.EXPECT().GetMe(gomock.Any()).Return(nil, errors.ErrUserNotFoundInDB)

	// Execute
	req := httptest.NewRequest("GET", "/ai-core/me", nil)
	w := httptest.NewRecorder()

	// Add route for GetMe
	suite.router.GET("/ai-core/me", suite.handler.GetMe)
	suite.router.ServeHTTP(w, req)

	// Assert
	suite.Equal(http.StatusForbidden, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err)
	suite.Equal(errors.ErrUserNotFoundInDB.Message, response["error"])
}

func (suite *AICoreHandlerTestSuite) TestGetMe_InternalServerError() {
	// Setup
	suite.aicoreService.EXPECT().GetMe(gomock.Any()).Return(nil, errors.ErrInternalError)

	// Execute
	req := httptest.NewRequest("GET", "/ai-core/me", nil)
	w := httptest.NewRecorder()

	// Add route for GetMe
	suite.router.GET("/ai-core/me", suite.handler.GetMe)
	suite.router.ServeHTTP(w, req)

	// Assert
	suite.Equal(http.StatusInternalServerError, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err)
	suite.Equal(errors.ErrInternalError.Error(), response["error"])
}

func (suite *AICoreHandlerTestSuite) TestChatInference_Success() {
	// Setup
	requestBody := service.AICoreInferenceRequest{
		DeploymentID: "deployment-1",
		Messages: []service.AICoreInferenceMessage{
			{
				Role:    "user",
				Content: "Hello, how are you?",
			},
		},
		MaxTokens:   100,
		Temperature: 0.7,
	}

	expectedResponse := &service.AICoreInferenceResponse{
		ID:      "chatcmpl-123",
		Object:  "chat.completion",
		Created: 1234567890,
		Model:   "gpt-4",
		Choices: []service.AICoreInferenceChoice{
			{
				Index: 0,
				Message: service.AICoreInferenceMessage{
					Role:    "assistant",
					Content: "I'm doing well, thank you!",
				},
				FinishReason: "stop",
			},
		},
		Usage: service.AICoreInferenceUsage{
			PromptTokens:     10,
			CompletionTokens: 8,
			TotalTokens:      18,
		},
	}

	suite.aicoreService.EXPECT().ChatInference(gomock.Any(), &requestBody).Return(expectedResponse, nil)

	// Execute
	body, _ := json.Marshal(requestBody)
	req := httptest.NewRequest("POST", "/ai-core/chat/inference", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Add route for ChatInference
	suite.router.POST("/ai-core/chat/inference", suite.handler.ChatInference)
	suite.router.ServeHTTP(w, req)

	// Assert
	suite.Equal(http.StatusOK, w.Code)

	var response service.AICoreInferenceResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err)
	suite.Equal("chatcmpl-123", response.ID)
	suite.Equal("gpt-4", response.Model)
	suite.Len(response.Choices, 1)
	suite.Equal("I'm doing well, thank you!", response.Choices[0].Message.Content)
}

func (suite *AICoreHandlerTestSuite) TestChatInference_InvalidJSON() {
	// Execute
	req := httptest.NewRequest("POST", "/ai-core/chat/inference", bytes.NewBufferString("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Add route for ChatInference
	suite.router.POST("/ai-core/chat/inference", suite.handler.ChatInference)
	suite.router.ServeHTTP(w, req)

	// Assert
	suite.Equal(http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err)
	suite.Contains(response["error"].(string), "invalid character")
}

func (suite *AICoreHandlerTestSuite) TestChatInference_ValidationError_MissingDeploymentID() {
	// Setup - missing required deploymentId
	requestBody := service.AICoreInferenceRequest{
		Messages: []service.AICoreInferenceMessage{
			{
				Role:    "user",
				Content: "Hello",
			},
		},
	}

	// Execute
	body, _ := json.Marshal(requestBody)
	req := httptest.NewRequest("POST", "/ai-core/chat/inference", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Add route for ChatInference
	suite.router.POST("/ai-core/chat/inference", suite.handler.ChatInference)
	suite.router.ServeHTTP(w, req)

	// Assert
	suite.Equal(http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err)
	suite.Contains(response["error"].(string), "required")
}

func (suite *AICoreHandlerTestSuite) TestChatInference_ValidationError_EmptyMessages() {
	// Setup - empty messages array
	requestBody := service.AICoreInferenceRequest{
		DeploymentID: "deployment-1",
		Messages:     []service.AICoreInferenceMessage{},
	}

	// Execute
	body, _ := json.Marshal(requestBody)
	req := httptest.NewRequest("POST", "/ai-core/chat/inference", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Add route for ChatInference
	suite.router.POST("/ai-core/chat/inference", suite.handler.ChatInference)
	suite.router.ServeHTTP(w, req)

	// Assert
	suite.Equal(http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err)
	suite.Contains(response["error"].(string), "min")
}

func (suite *AICoreHandlerTestSuite) TestChatInference_ServiceError() {
	// Setup
	requestBody := service.AICoreInferenceRequest{
		DeploymentID: "deployment-1",
		Messages: []service.AICoreInferenceMessage{
			{
				Role:    "user",
				Content: "Hello",
			},
		},
	}

	suite.aicoreService.EXPECT().ChatInference(gomock.Any(), &requestBody).Return(nil, errors.ErrInternalError)

	// Execute
	body, _ := json.Marshal(requestBody)
	req := httptest.NewRequest("POST", "/ai-core/chat/inference", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Add route for ChatInference
	suite.router.POST("/ai-core/chat/inference", suite.handler.ChatInference)
	suite.router.ServeHTTP(w, req)

	// Assert
	suite.Equal(http.StatusInternalServerError, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err)
	suite.Equal(errors.ErrInternalError.Error(), response["error"])
}

func (suite *AICoreHandlerTestSuite) TestChatInference_Streaming_Success() {
	// Setup
	requestBody := service.AICoreInferenceRequest{
		DeploymentID: "deployment-1",
		Messages: []service.AICoreInferenceMessage{
			{
				Role:    "user",
				Content: "Hello",
			},
		},
		Stream: true, // Enable streaming
	}

	// Mock the streaming service call
	suite.aicoreService.EXPECT().ChatInferenceStream(gomock.Any(), &requestBody, gomock.Any()).Return(nil)

	// Execute
	body, _ := json.Marshal(requestBody)
	req := httptest.NewRequest("POST", "/ai-core/chat/inference", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Add route for ChatInference
	suite.router.POST("/ai-core/chat/inference", suite.handler.ChatInference)
	suite.router.ServeHTTP(w, req)

	// Assert - For SSE, we check headers
	suite.Equal(http.StatusOK, w.Code)
	suite.Equal("text/event-stream", w.Header().Get("Content-Type"))
	suite.Equal("no-cache", w.Header().Get("Cache-Control"))
	suite.Equal("keep-alive", w.Header().Get("Connection"))
	suite.Equal("no", w.Header().Get("X-Accel-Buffering"))
}

func (suite *AICoreHandlerTestSuite) TestChatInference_Streaming_ServiceError() {
	// Setup
	requestBody := service.AICoreInferenceRequest{
		DeploymentID: "deployment-1",
		Messages: []service.AICoreInferenceMessage{
			{
				Role:    "user",
				Content: "Hello",
			},
		},
		Stream: true, // Enable streaming
	}

	// Mock the streaming service call to return an error
	suite.aicoreService.EXPECT().ChatInferenceStream(gomock.Any(), &requestBody, gomock.Any()).Return(errors.ErrInternalError)

	// Execute
	body, _ := json.Marshal(requestBody)
	req := httptest.NewRequest("POST", "/ai-core/chat/inference", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Add route for ChatInference
	suite.router.POST("/ai-core/chat/inference", suite.handler.ChatInference)
	suite.router.ServeHTTP(w, req)

	// Assert - SSE headers should still be set, and error event should be sent
	suite.Equal(http.StatusOK, w.Code)                                  // SSE always returns 200
	suite.Contains(w.Header().Get("Content-Type"), "text/event-stream") // May include charset

	// The response body should contain an SSE error event (no space after colon in SSE format)
	responseBody := w.Body.String()
	suite.Contains(responseBody, "event:error")
	suite.Contains(responseBody, errors.ErrInternalError.Error())
}

func (suite *AICoreHandlerTestSuite) TestUploadAttachment_Success_SingleFile() {
	// Setup
	expectedResponse := map[string]interface{}{
		"id":       "file-123",
		"filename": "test.txt",
		"size":     100,
	}

	suite.aicoreService.EXPECT().UploadAttachment(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(c *gin.Context, file multipart.File, header *multipart.FileHeader) (map[string]interface{}, error) {
			if header.Filename == "test.txt" {
				return expectedResponse, nil
			}
			return nil, fmt.Errorf("unexpected file")
		})

	// Create multipart form
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "test.txt")
	part.Write([]byte("test file content"))
	writer.Close()

	// Execute
	req := httptest.NewRequest("POST", "/ai-core/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()

	// Add route for UploadAttachment
	suite.router.POST("/ai-core/upload", suite.handler.UploadAttachment)
	suite.router.ServeHTTP(w, req)

	// Assert
	suite.Equal(http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err)
	suite.Equal(float64(1), response["count"])
	suite.NotNil(response["files"])
}

func (suite *AICoreHandlerTestSuite) TestUploadAttachment_Success_MultipleFiles() {
	// Setup - Mock for multiple files
	expectedResponse1 := map[string]interface{}{
		"id":       "file-123",
		"filename": "test1.txt",
		"size":     100,
	}
	expectedResponse2 := map[string]interface{}{
		"id":       "file-456",
		"filename": "test2.txt",
		"size":     200,
	}

	suite.aicoreService.EXPECT().UploadAttachment(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(c *gin.Context, file multipart.File, header *multipart.FileHeader) (map[string]interface{}, error) {
			if header.Filename == "test1.txt" {
				return expectedResponse1, nil
			}
			if header.Filename == "test2.txt" {
				return expectedResponse2, nil
			}
			return nil, fmt.Errorf("unexpected file")
		}).Times(2)

	// Create multipart form with multiple files
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part1, _ := writer.CreateFormFile("files", "test1.txt")
	part1.Write([]byte("test file 1 content"))
	part2, _ := writer.CreateFormFile("files", "test2.txt")
	part2.Write([]byte("test file 2 content"))
	writer.Close()

	// Execute
	req := httptest.NewRequest("POST", "/ai-core/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()

	// Add route for UploadAttachment
	suite.router.POST("/ai-core/upload", suite.handler.UploadAttachment)
	suite.router.ServeHTTP(w, req)

	// Assert
	suite.Equal(http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err)
	suite.Equal(float64(2), response["count"])
	suite.NotNil(response["files"])
}

func (suite *AICoreHandlerTestSuite) TestUploadAttachment_NoFilesProvided() {
	// Create multipart form without files
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	writer.Close()

	// Execute
	req := httptest.NewRequest("POST", "/ai-core/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()

	// Add route for UploadAttachment
	suite.router.POST("/ai-core/upload", suite.handler.UploadAttachment)
	suite.router.ServeHTTP(w, req)

	// Assert
	suite.Equal(http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err)
	suite.Equal(errors.ErrNoFilesProvided.Error(), response["error"])
}

func (suite *AICoreHandlerTestSuite) TestUploadAttachment_FileTooLarge() {
	// Create a file larger than 5MB
	largeContent := make([]byte, 6*1024*1024) // 6MB

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "large.txt")
	part.Write(largeContent)
	writer.Close()

	// Execute
	req := httptest.NewRequest("POST", "/ai-core/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()

	// Add route for UploadAttachment
	suite.router.POST("/ai-core/upload", suite.handler.UploadAttachment)
	suite.router.ServeHTTP(w, req)

	// Assert
	suite.Equal(http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err)
	suite.Equal(errors.ErrCombinedFileSizeExceeds.Error(), response["error"])
}

func (suite *AICoreHandlerTestSuite) TestUploadAttachment_ServiceError() {
	// Setup
	suite.aicoreService.EXPECT().UploadAttachment(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, errors.ErrInternalError)

	// Create multipart form
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "test.txt")
	part.Write([]byte("test content"))
	writer.Close()

	// Execute
	req := httptest.NewRequest("POST", "/ai-core/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()

	// Add route for UploadAttachment
	suite.router.POST("/ai-core/upload", suite.handler.UploadAttachment)
	suite.router.ServeHTTP(w, req)

	// Assert
	suite.Equal(http.StatusInternalServerError, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err)
	suite.Equal(errors.ErrInternalError.Error(), response["error"])
}

func TestAICoreHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(AICoreHandlerTestSuite))
}
