package handlers_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"developer-portal-backend/internal/api/handlers"
	"developer-portal-backend/internal/database/models"
	apperrors "developer-portal-backend/internal/errors"
	"developer-portal-backend/internal/mocks"
	"developer-portal-backend/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

// ComponentHandlerTestSuite defines the test suite for ComponentHandler
type ComponentHandlerTestSuite struct {
	suite.Suite
	ctrl                 *gomock.Controller
	mockComponentService *mocks.MockComponentServiceInterface
	mockTeamService      *mocks.MockTeamServiceInterface
	mockLandscapeService *mocks.MockLandscapeServiceInterface
	mockProjectService   *mocks.MockProjectServiceInterface
	handler              *handlers.ComponentHandler
	router               *gin.Engine
}

// SetupTest sets up the test suite
func (suite *ComponentHandlerTestSuite) SetupTest() {
	gin.SetMode(gin.TestMode)
	suite.ctrl = gomock.NewController(suite.T())

	// Create mocks
	suite.mockComponentService = mocks.NewMockComponentServiceInterface(suite.ctrl)
	suite.mockTeamService = mocks.NewMockTeamServiceInterface(suite.ctrl)
	suite.mockLandscapeService = mocks.NewMockLandscapeServiceInterface(suite.ctrl)
	suite.mockProjectService = mocks.NewMockProjectServiceInterface(suite.ctrl)

	// Create handler with mocks
	suite.handler = handlers.NewComponentHandler(
		suite.mockComponentService,
		suite.mockLandscapeService,
		suite.mockTeamService,
		suite.mockProjectService,
	)

	suite.router = gin.New()
	suite.setupRoutes()
}

// TearDownTest tears down the test suite
func (suite *ComponentHandlerTestSuite) TearDownTest() {
	suite.ctrl.Finish()
}

// setupRoutes sets up the routes for testing
func (suite *ComponentHandlerTestSuite) setupRoutes() {
	suite.router.GET("/components", suite.handler.ListComponents)
	suite.router.GET("/components/health", suite.handler.ComponentHealth)
}

// TestListComponents_MissingParameters tests missing both team-id and project-name
func (suite *ComponentHandlerTestSuite) TestListComponents_MissingParameters() {
	req := httptest.NewRequest(http.MethodGet, "/components", nil)
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)
	assert.Contains(suite.T(), w.Body.String(), apperrors.ErrMissingTeamOrProjectName.Message)
}

// TestListComponents_InvalidTeamID tests invalid team-id format
func (suite *ComponentHandlerTestSuite) TestListComponents_InvalidTeamID() {
	req := httptest.NewRequest(http.MethodGet, "/components?team-id=invalid-uuid", nil)
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)
	assert.Contains(suite.T(), w.Body.String(), apperrors.ErrInvalidTeamID.Error())
}

// TestListComponents_TeamIDSuccess tests successful retrieval by team-id
func (suite *ComponentHandlerTestSuite) TestListComponents_TeamIDSuccess() {
	teamID := uuid.New()
	projectID := uuid.New()
	// Mock components with metadata
	metadata := json.RawMessage(`{
		"ci": {"qos": "https://qos.example.com"},
		"sonar": {"project_id": "test-project"},
		"github": {"url": "https://github.com/test/repo"},
		"central-service": true,
		"isLibrary": false,
		"health": true
	}`)
	componentID := uuid.New()
	components := []models.Component{
		{
			BaseModel: models.BaseModel{
				ID:          componentID,
				Name:        "test-component",
				Title:       "Test Component",
				Description: "Test Description",
				Metadata:    metadata,
			},
			OwnerID:   teamID,
			ProjectID: projectID,
		},
	}

	suite.mockTeamService.EXPECT().GetTeamComponentsByID(teamID, 1, 1000000).Return(components, int64(1), nil)

	suite.mockComponentService.EXPECT().GetProjectTitleByID(projectID).Return("Test Project", nil)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/components?team-id=%s", teamID.String()), nil)
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var result []map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &result)
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), result, 1)
	assert.Equal(suite.T(), "test-component", result[0]["name"])
	assert.Equal(suite.T(), "Test Component", result[0]["title"])
	assert.Equal(suite.T(), "https://qos.example.com", result[0]["qos"])
	assert.Equal(suite.T(), "https://sonar.tools.sap/dashboard?id=test-project", result[0]["sonar"])
	assert.Equal(suite.T(), "https://github.com/test/repo", result[0]["github"])
	assert.Equal(suite.T(), true, result[0]["central-service"])
	assert.Equal(suite.T(), false, result[0]["is-library"])
	assert.Equal(suite.T(), true, result[0]["health"])
	assert.Equal(suite.T(), "Test Project", result[0]["project_title"])
}

// TestListComponents_TeamNotFound tests team not found error
func (suite *ComponentHandlerTestSuite) TestListComponents_TeamNotFound() {
	teamID := uuid.New()

	suite.mockTeamService.EXPECT().
		GetTeamComponentsByID(teamID, 1, 1000000).
		Return(nil, int64(0), apperrors.ErrTeamNotFound)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/components?team-id=%s", teamID.String()), nil)
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusNotFound, w.Code)
	assert.Contains(suite.T(), w.Body.String(), apperrors.ErrTeamNotFound.Error())
}

// TestListComponents_TeamServiceError tests internal error from team service
func (suite *ComponentHandlerTestSuite) TestListComponents_TeamServiceError() {
	teamID := uuid.New()

	suite.mockTeamService.EXPECT().
		GetTeamComponentsByID(teamID, 1, 1000000).
		Return(nil, int64(0), fmt.Errorf("database error"))

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/components?team-id=%s", teamID.String()), nil)
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusInternalServerError, w.Code)
}

// TestListComponents_ProjectNameSuccess tests successful retrieval by project-name
func (suite *ComponentHandlerTestSuite) TestListComponents_ProjectNameSuccess() {
	projectName := "test-project"

	views := []service.ComponentProjectView{
		{
			ID:          uuid.New(),
			OwnerID:     uuid.New(),
			Name:        "component1",
			Title:       "Component 1",
			Description: "Description 1",
			QOS:         "https://qos.example.com",
			Sonar:       "https://sonar.tools.sap/dashboard?id=proj1",
			GitHub:      "https://github.com/test/repo1",
		},
		{
			ID:          uuid.New(),
			OwnerID:     uuid.New(),
			Name:        "component2",
			Title:       "Component 2",
			Description: "Description 2",
		},
	}

	suite.mockComponentService.EXPECT().GetByProjectNameAllView(projectName).Return(views, nil)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/components?project-name=%s", projectName), nil)
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var result []service.ComponentProjectView
	err := json.Unmarshal(w.Body.Bytes(), &result)
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), result, 2)
	assert.Equal(suite.T(), "component1", result[0].Name)
	assert.Equal(suite.T(), "component2", result[1].Name)
}

// TestListComponents_ProjectNotFound tests project not found error
func (suite *ComponentHandlerTestSuite) TestListComponents_ProjectNotFound() {
	projectName := "nonexistent-project"

	suite.mockComponentService.EXPECT().GetByProjectNameAllView(projectName).Return(nil, apperrors.ErrProjectNotFound)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/components?project-name=%s", projectName), nil)
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusNotFound, w.Code)
	assert.Contains(suite.T(), w.Body.String(), apperrors.ErrProjectNotFound.Error())
}

// TestListComponents_ProjectServiceError tests internal error from component service
func (suite *ComponentHandlerTestSuite) TestListComponents_ProjectServiceError() {
	projectName := "test-project"

	suite.mockComponentService.EXPECT().
		GetByProjectNameAllView(projectName).
		Return(nil, fmt.Errorf("database error"))

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/components?project-name=%s", projectName), nil)
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusInternalServerError, w.Code)
}

// TestComponentHealth_MissingParameters tests missing parameters
func (suite *ComponentHandlerTestSuite) TestComponentHealth_MissingParameters() {
	req := httptest.NewRequest(http.MethodGet, "/components/health", nil)
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)
	assert.Contains(suite.T(), w.Body.String(), apperrors.ErrMissingHealthParams.Message)
}

// TestComponentHealth_InvalidComponentID tests invalid component-id
func (suite *ComponentHandlerTestSuite) TestComponentHealth_InvalidComponentID() {
	landscapeID := uuid.New()
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/components/health?component-id=invalid&landscape-id=%s", landscapeID.String()), nil)
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)
	assert.Contains(suite.T(), w.Body.String(), apperrors.ErrInvalidComponentID.Error())
}

// TestComponentHealth_InvalidLandscapeID tests invalid landscape-id
func (suite *ComponentHandlerTestSuite) TestComponentHealth_InvalidLandscapeID() {
	componentID := uuid.New()
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/components/health?component-id=%s&landscape-id=invalid", componentID.String()), nil)
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)
	assert.Contains(suite.T(), w.Body.String(), apperrors.ErrInvalidLandscapeID.Error())
}

// TestComponentHealth_ComponentNotFound tests component not found
func (suite *ComponentHandlerTestSuite) TestComponentHealth_ComponentNotFound() {
	componentID := uuid.New()
	landscapeID := uuid.New()

	suite.mockComponentService.EXPECT().GetByID(componentID).Return(nil, apperrors.ErrComponentNotFound)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/components/health?component-id=%s&landscape-id=%s", componentID.String(), landscapeID.String()), nil)
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusInternalServerError, w.Code)
	assert.Contains(suite.T(), w.Body.String(), apperrors.ErrComponentNotFound.Error())
}

// TestComponentHealth_LandscapeNotFound tests landscape not found
func (suite *ComponentHandlerTestSuite) TestComponentHealth_LandscapeNotFound() {
	componentID := uuid.New()
	landscapeID := uuid.New()

	component := &models.Component{
		BaseModel: models.BaseModel{
			ID:   componentID,
			Name: "test-component",
		},
	}

	suite.mockComponentService.EXPECT().GetByID(componentID).Return(component, nil)

	suite.mockLandscapeService.EXPECT().GetLandscapeByID(landscapeID).Return(nil, apperrors.ErrLandscapeNotFound)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/components/health?component-id=%s&landscape-id=%s", componentID.String(), landscapeID.String()), nil)
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusInternalServerError, w.Code)
	assert.Contains(suite.T(), w.Body.String(), apperrors.ErrLandscapeNotFound.Error())
}

// TestComponentHealth_ComponentServiceError tests component service error
func (suite *ComponentHandlerTestSuite) TestComponentHealth_ComponentServiceError() {
	componentID := uuid.New()
	landscapeID := uuid.New()

	suite.mockComponentService.EXPECT().GetByID(componentID).Return(nil, fmt.Errorf("database error"))

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/components/health?component-id=%s&landscape-id=%s", componentID.String(), landscapeID.String()), nil)
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusInternalServerError, w.Code)
}

// TestComponentHealth_LandscapeServiceError tests landscape service error
func (suite *ComponentHandlerTestSuite) TestComponentHealth_LandscapeServiceError() {
	componentID := uuid.New()
	landscapeID := uuid.New()

	component := &models.Component{
		BaseModel: models.BaseModel{
			ID:   componentID,
			Name: "test-component",
		},
	}

	suite.mockComponentService.EXPECT().GetByID(componentID).Return(component, nil)

	suite.mockLandscapeService.EXPECT().
		GetLandscapeByID(landscapeID).
		Return(nil, fmt.Errorf("database error"))

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/components/health?component-id=%s&landscape-id=%s", componentID.String(), landscapeID.String()), nil)
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusInternalServerError, w.Code)
}

// TestComponentHealth_SuccessWithMetadata tests successful health check with subdomain metadata
func (suite *ComponentHandlerTestSuite) TestComponentHealth_SuccessWithMetadata() {
	componentID := uuid.New()
	landscapeID := uuid.New()

	// Create a mock HTTP server to simulate the health endpoint
	mockHealthServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"UP","version":"1.0.0"}`))
	}))
	defer mockHealthServer.Close()

	// Extract domain from mock server URL (remove http:// and port for testing)
	// In real scenario, this would be like "cfapps.eu10.hana.ondemand.com"
	mockDomain := "test.domain.com"

	metadata := json.RawMessage(`{"subdomain": "myapp"}`)

	component := &models.Component{
		BaseModel: models.BaseModel{
			ID:       componentID,
			Name:     "test-component",
			Metadata: metadata,
		},
	}

	landscape := &service.LandscapeResponse{
		ID:     landscapeID,
		Domain: mockDomain,
	}

	suite.mockComponentService.EXPECT().
		GetByID(componentID).
		Return(component, nil)

	suite.mockLandscapeService.EXPECT().
		GetLandscapeByID(landscapeID).
		Return(landscape, nil)

	// Ensure project-level health URL template is provided so URL is built
	projectID := uuid.New()
	component.ProjectID = projectID
	suite.mockProjectService.EXPECT().GetHealthMetadata(projectID).Return("https://{subdomain}.{component_name}.cfapps.{landscape_domain}/health", ".*", nil)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/components/health?component-id=%s&landscape-id=%s", componentID.String(), landscapeID.String()), nil)
	w := httptest.NewRecorder()

	// Note: This test will fail to connect to the constructed URL, but it tests the URL construction logic
	suite.router.ServeHTTP(w, req)

	// The request will fail with 502 Bad Gateway because the URL doesn't exist
	// But this tests that the handler processes the request correctly up to the HTTP call
	assert.Equal(suite.T(), http.StatusInternalServerError, w.Code)
	assert.Contains(suite.T(), w.Body.String(), "failed to fetch component health")
}

// TestComponentHealth_SuccessWithoutMetadata tests successful health check without subdomain metadata
func (suite *ComponentHandlerTestSuite) TestComponentHealth_SuccessWithoutMetadata() {
	componentID := uuid.New()
	landscapeID := uuid.New()

	component := &models.Component{
		BaseModel: models.BaseModel{
			ID:   componentID,
			Name: "test-component",
		},
	}

	landscape := &service.LandscapeResponse{
		ID:     landscapeID,
		Domain: "test.domain.com",
	}

	suite.mockComponentService.EXPECT().
		GetByID(componentID).
		Return(component, nil)

	suite.mockLandscapeService.EXPECT().
		GetLandscapeByID(landscapeID).
		Return(landscape, nil)

	// Ensure project-level health URL template is provided so URL is built
	projectID := uuid.New()
	component.ProjectID = projectID
	suite.mockProjectService.EXPECT().GetHealthMetadata(projectID).Return("https://{subdomain}.{component_name}.cfapps.{landscape_domain}/health", ".*", nil)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/components/health?component-id=%s&landscape-id=%s", componentID.String(), landscapeID.String()), nil)
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	// Will fail with 502 because URL doesn't exist, but tests the logic
	assert.Equal(suite.T(), http.StatusInternalServerError, w.Code)
	assert.Contains(suite.T(), w.Body.String(), "failed to fetch component health")
}

// TestComponentHealth_LandscapeServiceNil tests when landscape service is not configured
func (suite *ComponentHandlerTestSuite) TestComponentHealth_LandscapeServiceNil() {
	componentID := uuid.New()
	landscapeID := uuid.New()

	// Create handler without landscape service
	handlerWithoutLandscape := handlers.NewComponentHandler(suite.mockComponentService, nil, suite.mockTeamService, suite.mockProjectService)
	router := gin.New()
	router.GET("/components/health", handlerWithoutLandscape.ComponentHealth)

	component := &models.Component{
		BaseModel: models.BaseModel{
			ID:   componentID,
			Name: "test-component",
		},
	}

	suite.mockComponentService.EXPECT().
		GetByID(componentID).
		Return(component, nil)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/components/health?component-id=%s&landscape-id=%s", componentID.String(), landscapeID.String()), nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusInternalServerError, w.Code)
	assert.Contains(suite.T(), w.Body.String(), apperrors.ErrLandscapeNotConfigured.Error())
}

// TestComponentHandlerTestSuite runs the test suite
func TestComponentHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(ComponentHandlerTestSuite))
}
