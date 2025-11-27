package handlers_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"developer-portal-backend/internal/api/handlers"
	"developer-portal-backend/internal/database/models"
	"developer-portal-backend/internal/mocks"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

type ProjectHandlerTestSuite struct {
	suite.Suite
	ctrl        *gomock.Controller
	mockProject *mocks.MockProjectServiceInterface
	handler     *handlers.ProjectHandler
}

func (suite *ProjectHandlerTestSuite) SetupTest() {
	gin.SetMode(gin.TestMode)
	suite.ctrl = gomock.NewController(suite.T())
	suite.mockProject = mocks.NewMockProjectServiceInterface(suite.ctrl)
	suite.handler = handlers.NewProjectHandler(suite.mockProject)
}

func (suite *ProjectHandlerTestSuite) TearDownTest() {
	suite.ctrl.Finish()
}

// helper to build router with authentication middleware simulation
func (suite *ProjectHandlerTestSuite) newRouter(withAuth bool, username string) *gin.Engine {
	r := gin.New()
	if withAuth {
		r.Use(func(c *gin.Context) {
			c.Set("username", username)
			c.Next()
		})
	}
	r.GET("/api/v1/projects", suite.handler.GetAllProjects)
	return r
}

func (suite *ProjectHandlerTestSuite) TestGetAllProjects_Success_EmptyList() {
	router := suite.newRouter(true, "testuser")

	// Mock empty project list
	projects := []models.Project{}
	suite.mockProject.EXPECT().GetAllProjects().Return(projects, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	// The handler returns flattened structure
	var response []map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Empty(suite.T(), response)
}

func (suite *ProjectHandlerTestSuite) TestGetAllProjects_Success_WithProjects() {
	router := suite.newRouter(true, "testuser")

	// Create test projects
	projectID1 := uuid.New()
	projectID2 := uuid.New()
	now := time.Now()

	projects := []models.Project{
		{
			BaseModel: models.BaseModel{
				ID:          projectID1,
				CreatedAt:   now,
				CreatedBy:   "user1",
				UpdatedAt:   now,
				UpdatedBy:   "user1",
				Name:        "test-project-1",
				Title:       "Test Project 1",
				Description: "First test project",
				Metadata:    json.RawMessage(`{"env": "dev", "tech": "go"}`),
			},
		},
		{
			BaseModel: models.BaseModel{
				ID:          projectID2,
				CreatedAt:   now.Add(-time.Hour),
				CreatedBy:   "user2",
				UpdatedAt:   now.Add(-time.Minute * 30),
				UpdatedBy:   "user2",
				Name:        "test-project-2",
				Title:       "Test Project 2",
				Description: "Second test project",
				Metadata:    json.RawMessage(`{"env": "prod", "tech": "java"}`),
			},
		},
	}

	suite.mockProject.EXPECT().GetAllProjects().Return(projects, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	// The handler returns flattened structure, not the original model structure
	var response []map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), response, 2)

	// Verify first project (enriched structure)
	assert.Equal(suite.T(), projectID1.String(), response[0]["id"])
	assert.Equal(suite.T(), "test-project-1", response[0]["name"])
	assert.Equal(suite.T(), "Test Project 1", response[0]["title"])
	assert.Equal(suite.T(), "First test project", response[0]["description"])
	// Metadata fields are no longer flattened - only specific enriched fields are included
	assert.NotContains(suite.T(), response[0], "env")
	assert.NotContains(suite.T(), response[0], "tech")

	// Verify second project (enriched structure)
	assert.Equal(suite.T(), projectID2.String(), response[1]["id"])
	assert.Equal(suite.T(), "test-project-2", response[1]["name"])
	assert.Equal(suite.T(), "Test Project 2", response[1]["title"])
	assert.Equal(suite.T(), "Second test project", response[1]["description"])
	// Metadata fields are no longer flattened - only specific enriched fields are included
	assert.NotContains(suite.T(), response[1], "env")
	assert.NotContains(suite.T(), response[1], "tech")
}

func (suite *ProjectHandlerTestSuite) TestGetAllProjects_Success_WithNullMetadata() {
	router := suite.newRouter(true, "testuser")

	// Create test project with null metadata
	projectID := uuid.New()
	now := time.Now()

	projects := []models.Project{
		{
			BaseModel: models.BaseModel{
				ID:          projectID,
				CreatedAt:   now,
				CreatedBy:   "user1",
				UpdatedAt:   now,
				UpdatedBy:   "user1",
				Name:        "simple-project",
				Title:       "Simple Project",
				Description: "Project without metadata",
				Metadata:    json.RawMessage("null"), // null metadata
			},
		},
	}

	suite.mockProject.EXPECT().GetAllProjects().Return(projects, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	// The handler returns flattened structure
	var response []map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), response, 1)
	assert.Equal(suite.T(), projectID.String(), response[0]["id"])
	assert.Equal(suite.T(), "simple-project", response[0]["name"])
	assert.Equal(suite.T(), "Simple Project", response[0]["title"])
	assert.Equal(suite.T(), "Project without metadata", response[0]["description"])
	// With null metadata, no additional fields should be added
}

func (suite *ProjectHandlerTestSuite) TestGetAllProjects_ServiceError() {
	router := suite.newRouter(true, "testuser")

	// Mock service error
	suite.mockProject.EXPECT().GetAllProjects().Return(nil, errors.New("database connection failed"))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusInternalServerError, w.Code)

	var errorResponse map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "database connection failed", errorResponse["error"])
}

func (suite *ProjectHandlerTestSuite) TestGetAllProjects_WithoutAuthentication() {
	// This test simulates what would happen without auth middleware
	// In the actual application, auth middleware is required and would block this
	router := suite.newRouter(false, "")

	projects := []models.Project{}
	suite.mockProject.EXPECT().GetAllProjects().Return(projects, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Without auth middleware in test, it should still work
	// In production, auth middleware would return 401
	assert.Equal(suite.T(), http.StatusOK, w.Code)
}

func (suite *ProjectHandlerTestSuite) TestGetAllProjects_ResponseFormat() {
	router := suite.newRouter(true, "testuser")

	// Create a project with specific data to verify response format
	projectID := uuid.New()
	now := time.Now()

	projects := []models.Project{
		{
			BaseModel: models.BaseModel{
				ID:          projectID,
				CreatedAt:   now,
				CreatedBy:   "testuser",
				UpdatedAt:   now,
				UpdatedBy:   "testuser",
				Name:        "format-test",
				Title:       "Format Test Project",
				Description: "Testing response format",
				Metadata:    json.RawMessage(`{"version": "1.0.0"}`),
			},
		},
	}

	suite.mockProject.EXPECT().GetAllProjects().Return(projects, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)
	assert.Equal(suite.T(), "application/json; charset=utf-8", w.Header().Get("Content-Type"))

	// Verify JSON structure matches expected flattened format
	var response []map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), response, 1)

	project := response[0]

	// Verify core fields are present (enriched structure)
	assert.Contains(suite.T(), project, "id")
	assert.Contains(suite.T(), project, "name")
	assert.Contains(suite.T(), project, "title")
	assert.Contains(suite.T(), project, "description")
	// Metadata fields are no longer flattened - only specific enriched fields are included
	assert.NotContains(suite.T(), project, "version")

	// Verify field values
	assert.Equal(suite.T(), projectID.String(), project["id"])
	assert.Equal(suite.T(), "format-test", project["name"])
	assert.Equal(suite.T(), "Format Test Project", project["title"])
	assert.Equal(suite.T(), "Testing response format", project["description"])
}

func (suite *ProjectHandlerTestSuite) TestGetAllProjects_HTTPMethod() {
	router := suite.newRouter(true, "testuser")

	// Test that only GET method is supported
	methods := []string{"POST", "PUT", "DELETE", "PATCH"}

	for _, method := range methods {
		req := httptest.NewRequest(method, "/api/v1/projects", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(suite.T(), http.StatusNotFound, w.Code, "Method %s should not be allowed", method)
	}
}

func (suite *ProjectHandlerTestSuite) TestGetAllProjects_Success_WithEnrichedMetadata() {
	router := suite.newRouter(true, "testuser")

	// Create test project with repo and endpoint metadata
	projectID := uuid.New()
	now := time.Now()

	projects := []models.Project{
		{
			BaseModel: models.BaseModel{
				ID:          projectID,
				CreatedAt:   now,
				CreatedBy:   "testuser",
				UpdatedAt:   now,
				UpdatedBy:   "testuser",
				Name:        "enriched-test",
				Title:       "Enriched Test Project",
				Description: "Testing enriched metadata structure",
				Metadata:    json.RawMessage(`{"alerts": "https://github.com/example/repo", "components-metrics": true, "views": ["overview","cis"], "version": "2.0.0"}`),
			},
		},
	}

	suite.mockProject.EXPECT().GetAllProjects().Return(projects, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response []map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), response, 1)

	project := response[0]

	// Verify core fields
	assert.Equal(suite.T(), projectID.String(), project["id"])
	assert.Equal(suite.T(), "enriched-test", project["name"])
	assert.Equal(suite.T(), "Enriched Test Project", project["title"])
	assert.Equal(suite.T(), "Testing enriched metadata structure", project["description"])

	// Verify enriched nested structures
	assert.Contains(suite.T(), project, "alerts")

	// alerts is a flattened string (not nested under repo)
	alertsStr, ok := project["alerts"].(string)
	assert.True(suite.T(), ok)
	assert.Equal(suite.T(), "https://github.com/example/repo", alertsStr)

	// components-metrics should be present as a boolean
	assert.Contains(suite.T(), project, "components-metrics")
	cmBool, ok := project["components-metrics"].(bool)
	assert.True(suite.T(), ok)
	assert.True(suite.T(), cmBool)

	// views should be a flattened string from metadata.views array
	assert.Contains(suite.T(), project, "views")
	viewsStr, ok := project["views"].(string)
	assert.True(suite.T(), ok)
	assert.Equal(suite.T(), "overview,cis", viewsStr)

	// Metadata should not be included at top level
	assert.NotContains(suite.T(), project, "metadata")

	// Verify extracted fields are not at top level
	assert.NotContains(suite.T(), project, "repo")
	assert.NotContains(suite.T(), project, "endpoint")
}

func TestProjectHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(ProjectHandlerTestSuite))
}
