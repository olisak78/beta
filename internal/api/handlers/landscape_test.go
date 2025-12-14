package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"developer-portal-backend/internal/api/handlers"
	apperrors "developer-portal-backend/internal/errors"
	"developer-portal-backend/internal/mocks"
	"developer-portal-backend/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

// LandscapeHandlerTestSuite defines the test suite for LandscapeHandler
type LandscapeHandlerTestSuite struct {
	suite.Suite
	ctrl        *gomock.Controller
	mockService *mocks.MockLandscapeServiceInterface
	handler     *handlers.LandscapeHandler
	router      *gin.Engine
}

// SetupTest sets up the test suite
func (suite *LandscapeHandlerTestSuite) SetupTest() {
	gin.SetMode(gin.TestMode)
	suite.ctrl = gomock.NewController(suite.T())
	suite.mockService = mocks.NewMockLandscapeServiceInterface(suite.ctrl)
	suite.handler = handlers.NewLandscapeHandler(suite.mockService)
	suite.router = gin.New()
	suite.setupRoutes()
}

// TearDownTest cleans up after each test
func (suite *LandscapeHandlerTestSuite) TearDownTest() {
	suite.ctrl.Finish()
}

// setupRoutes sets up the routes for testing
func (suite *LandscapeHandlerTestSuite) setupRoutes() {
	suite.router.GET("/landscapes", suite.handler.ListLandscapesByQuery)
	suite.router.DELETE("/landscapes/:id", suite.handler.DeleteLandscape)
}

// TestListLandscapesByQuery_MissingProjectName tests when project-name parameter is missing
func (suite *LandscapeHandlerTestSuite) TestListLandscapesByQuery_MissingProjectName() {
	req := httptest.NewRequest(http.MethodGet, "/landscapes", nil)
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Contains(suite.T(), response["error"], "project-name")
}

// TestListLandscapesByQuery_Success tests successful retrieval of landscapes by project name
func (suite *LandscapeHandlerTestSuite) TestListLandscapesByQuery_Success() {
	projectName := "test-project"
	expectedLandscapes := []service.LandscapeMinimalResponse{
		{
			ID:   uuid.New(),
			Name: "dev-landscape",
		},
		{
			ID:   uuid.New(),
			Name: "prod-landscape",
		},
	}

	suite.mockService.EXPECT().GetByProjectNameAll(projectName).Return(expectedLandscapes, nil)

	req := httptest.NewRequest(http.MethodGet, "/landscapes?project-name="+projectName, nil)
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response []service.LandscapeMinimalResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), response, 2)
	assert.Equal(suite.T(), "dev-landscape", response[0].Name)
	assert.Equal(suite.T(), "prod-landscape", response[1].Name)
}

// TestListLandscapesByQuery_EmptyResult tests when no landscapes are found for a project
func (suite *LandscapeHandlerTestSuite) TestListLandscapesByQuery_EmptyResult() {
	projectName := "empty-project"
	emptyLandscapes := []service.LandscapeMinimalResponse{}

	suite.mockService.EXPECT().GetByProjectNameAll(projectName).Return(emptyLandscapes, nil)

	req := httptest.NewRequest(http.MethodGet, "/landscapes?project-name="+projectName, nil)
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response []service.LandscapeMinimalResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), response, 0)
}

// TestListLandscapesByQuery_ProjectNotFound tests when the project does not exist
func (suite *LandscapeHandlerTestSuite) TestListLandscapesByQuery_ProjectNotFound() {
	projectName := "non-existent-project"

	suite.mockService.EXPECT().GetByProjectNameAll(projectName).Return(nil, apperrors.ErrProjectNotFound)

	req := httptest.NewRequest(http.MethodGet, "/landscapes?project-name="+projectName, nil)
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusNotFound, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Contains(suite.T(), response["error"], apperrors.ErrProjectNotFound.Error())
}

// TestListLandscapesByQuery_InternalServerError tests when service returns an internal error
func (suite *LandscapeHandlerTestSuite) TestListLandscapesByQuery_InternalServerError() {
	projectName := "test-project"
	internalError := apperrors.ErrDatabaseConnection

	suite.mockService.EXPECT().GetByProjectNameAll(projectName).Return(nil, internalError)

	req := httptest.NewRequest(http.MethodGet, "/landscapes?project-name="+projectName, nil)
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusInternalServerError, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Contains(suite.T(), response["error"], apperrors.ErrDatabaseConnection.Message)
}

// TestListLandscapesByQuery_SpecialCharactersInProjectName tests handling of special characters
func (suite *LandscapeHandlerTestSuite) TestListLandscapesByQuery_SpecialCharactersInProjectName() {
	projectName := "test-project-with-special-chars-123"
	expectedLandscapes := []service.LandscapeMinimalResponse{
		{
			ID:   uuid.New(),
			Name: "special-landscape",
		},
	}

	suite.mockService.EXPECT().GetByProjectNameAll(projectName).Return(expectedLandscapes, nil)

	req := httptest.NewRequest(http.MethodGet, "/landscapes?project-name="+projectName, nil)
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response []service.LandscapeMinimalResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), response, 1)
	assert.Equal(suite.T(), "special-landscape", response[0].Name)
}

// TestListLandscapesByQuery_MultipleLandscapes tests retrieval of multiple landscapes
func (suite *LandscapeHandlerTestSuite) TestListLandscapesByQuery_MultipleLandscapes() {
	projectName := "multi-landscape-project"
	expectedLandscapes := []service.LandscapeMinimalResponse{
		{
			ID:   uuid.New(),
			Name: "dev",
		},
		{
			ID:   uuid.New(),
			Name: "staging",
		},
		{
			ID:   uuid.New(),
			Name: "prod",
		},
		{
			ID:   uuid.New(),
			Name: "qa",
		},
	}

	suite.mockService.EXPECT().GetByProjectNameAll(projectName).Return(expectedLandscapes, nil)

	req := httptest.NewRequest(http.MethodGet, "/landscapes?project-name="+projectName, nil)
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response []service.LandscapeMinimalResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), response, 4)
	assert.Equal(suite.T(), "dev", response[0].Name)
	assert.Equal(suite.T(), "staging", response[1].Name)
	assert.Equal(suite.T(), "prod", response[2].Name)
	assert.Equal(suite.T(), "qa", response[3].Name)
}

// TestDeleteLandscape_Success tests successful deletion of a landscape
func (suite *LandscapeHandlerTestSuite) TestDeleteLandscape_Success() {
	landscapeID := uuid.New()

	suite.mockService.EXPECT().DeleteLandscape(landscapeID).Return(nil)

	req := httptest.NewRequest(http.MethodDelete, "/landscapes/"+landscapeID.String(), nil)
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusNoContent, w.Code)
	assert.Empty(suite.T(), w.Body.String())
}

// TestDeleteLandscape_MissingID tests when landscape ID is missing
func (suite *LandscapeHandlerTestSuite) TestDeleteLandscape_MissingID() {
	req := httptest.NewRequest(http.MethodDelete, "/landscapes/", nil)
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusNotFound, w.Code)
}

// TestDeleteLandscape_InvalidID tests when landscape ID format is invalid
func (suite *LandscapeHandlerTestSuite) TestDeleteLandscape_InvalidID() {
	invalidID := "invalid-uuid"

	req := httptest.NewRequest(http.MethodDelete, "/landscapes/"+invalidID, nil)
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Contains(suite.T(), response["error"], "invalid landscape ID format")
}

// TestDeleteLandscape_LandscapeNotFound tests when landscape does not exist
func (suite *LandscapeHandlerTestSuite) TestDeleteLandscape_LandscapeNotFound() {
	landscapeID := uuid.New()

	suite.mockService.EXPECT().DeleteLandscape(landscapeID).Return(apperrors.ErrLandscapeNotFound)

	req := httptest.NewRequest(http.MethodDelete, "/landscapes/"+landscapeID.String(), nil)
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusNotFound, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Contains(suite.T(), response["error"], apperrors.ErrLandscapeNotFound.Error())
}

// TestDeleteLandscape_InternalServerError tests when service returns an internal error
func (suite *LandscapeHandlerTestSuite) TestDeleteLandscape_InternalServerError() {
	landscapeID := uuid.New()
	internalError := apperrors.ErrDatabaseConnection

	suite.mockService.EXPECT().DeleteLandscape(landscapeID).Return(internalError)

	req := httptest.NewRequest(http.MethodDelete, "/landscapes/"+landscapeID.String(), nil)
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusInternalServerError, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Contains(suite.T(), response["error"], apperrors.ErrDatabaseConnection.Message)
}

// TestDeleteLandscape_EmptyID tests when landscape ID is empty string
func (suite *LandscapeHandlerTestSuite) TestDeleteLandscape_EmptyID() {
	req := httptest.NewRequest(http.MethodDelete, "/landscapes/", nil)
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusNotFound, w.Code)
}

// Run the test suite
func TestLandscapeHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(LandscapeHandlerTestSuite))
}
