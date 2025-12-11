package handlers_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"developer-portal-backend/internal/api/handlers"
	apperrors "developer-portal-backend/internal/errors"
	"developer-portal-backend/internal/mocks"
	"developer-portal-backend/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

// SonarHandlerTestSuite defines the test suite for SonarHandler
type SonarHandlerTestSuite struct {
	suite.Suite
	ctrl        *gomock.Controller
	mockService *mocks.MockSonarServiceInterface
	handler     *handlers.SonarHandler
	router      *gin.Engine
}

func (suite *SonarHandlerTestSuite) SetupTest() {
	gin.SetMode(gin.TestMode)
	suite.ctrl = gomock.NewController(suite.T())
	suite.mockService = mocks.NewMockSonarServiceInterface(suite.ctrl)
	suite.handler = handlers.NewSonarHandler(suite.mockService)

	suite.router = gin.New()
	suite.router.GET("/sonar/measures", suite.handler.GetMeasures)
}

func (suite *SonarHandlerTestSuite) TearDownTest() {
	suite.ctrl.Finish()
}

// TestGetMeasures_MissingComponentParameter tests validation error when component parameter is missing
func (suite *SonarHandlerTestSuite) TestGetMeasures_MissingComponentParameter() {
	req := httptest.NewRequest(http.MethodGet, "/sonar/measures", nil)
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)

	assert.Contains(suite.T(), w.Body.String(), apperrors.NewMissingQueryParam("component").Error())
}

// TestGetMeasures_Success tests successful retrieval of Sonar measures for a valid component
func (suite *SonarHandlerTestSuite) TestGetMeasures_Success() {
	expectedResponse := &service.SonarCombinedResponse{
		Measures: []service.SonarMeasure{
			{
				Metric:    "coverage",
				Value:     "85.5",
				BestValue: false,
			},
			{
				Metric:    "vulnerabilities",
				Value:     "2",
				BestValue: true,
			},
			{
				Metric:    "code_smells",
				Value:     "15",
				BestValue: false,
			},
		},
	}

	suite.mockService.EXPECT().GetComponentMeasures("my-project-key").Return(expectedResponse, nil)

	req := httptest.NewRequest(http.MethodGet, "/sonar/measures?component=my-project-key", nil)
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var got service.SonarCombinedResponse
	err := json.Unmarshal(w.Body.Bytes(), &got)
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), got.Measures, 3)
	assert.Equal(suite.T(), "coverage", got.Measures[0].Metric)
	assert.Equal(suite.T(), "85.5", got.Measures[0].Value)
	assert.Equal(suite.T(), "vulnerabilities", got.Measures[1].Metric)
	assert.Equal(suite.T(), "2", got.Measures[1].Value)
	assert.Equal(suite.T(), "code_smells", got.Measures[2].Metric)
	assert.Equal(suite.T(), "15", got.Measures[2].Value)
}

// TestGetMeasures_ServiceError_ConnectionTimeout tests handling of connection timeout errors
func (suite *SonarHandlerTestSuite) TestGetMeasures_ServiceError_ConnectionTimeout() {
	suite.mockService.EXPECT().GetComponentMeasures("invalid-project").Return(nil, errors.New("connection timeout"))

	req := httptest.NewRequest(http.MethodGet, "/sonar/measures?component=invalid-project", nil)
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusBadGateway, w.Code)
	body := w.Body.String()
	assert.Contains(suite.T(), body, "sonar request failed")
	assert.Contains(suite.T(), body, "connection timeout")
}

// TestGetMeasures_ServiceError_ProjectNotFound tests handling of project not found errors
func (suite *SonarHandlerTestSuite) TestGetMeasures_ServiceError_ProjectNotFound() {
	suite.mockService.EXPECT().GetComponentMeasures("non-existent-project").Return(nil, errors.New("project not found in Sonar"))

	req := httptest.NewRequest(http.MethodGet, "/sonar/measures?component=non-existent-project", nil)
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusBadGateway, w.Code)
	body := w.Body.String()
	assert.Contains(suite.T(), body, "sonar request failed")
	assert.Contains(suite.T(), body, "project not found in Sonar")
}

// TestGetMeasures_EmptyMeasuresResponse tests handling of empty measures response
func (suite *SonarHandlerTestSuite) TestGetMeasures_EmptyMeasuresResponse() {
	expectedResponse := &service.SonarCombinedResponse{
		Measures: []service.SonarMeasure{},
	}

	suite.mockService.EXPECT().GetComponentMeasures("empty-project").Return(expectedResponse, nil)

	req := httptest.NewRequest(http.MethodGet, "/sonar/measures?component=empty-project", nil)
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var got service.SonarCombinedResponse
	err := json.Unmarshal(w.Body.Bytes(), &got)
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), got.Measures, 0)
}

func TestSonarHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(SonarHandlerTestSuite))
}
