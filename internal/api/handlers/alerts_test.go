package handlers_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"developer-portal-backend/internal/api/handlers"
	"developer-portal-backend/internal/auth"
	apperrors "developer-portal-backend/internal/errors"
	"developer-portal-backend/internal/mocks"
	"developer-portal-backend/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

type AlertsHandlerTestSuite struct {
	suite.Suite
	ctrl       *gomock.Controller
	mockAlerts *mocks.MockAlertsServiceInterface
	handler    *handlers.AlertsHandler
	router     *gin.Engine
	claims     *auth.AuthClaims
}

func (suite *AlertsHandlerTestSuite) SetupTest() {
	gin.SetMode(gin.TestMode)
	suite.ctrl = gomock.NewController(suite.T())
	suite.mockAlerts = mocks.NewMockAlertsServiceInterface(suite.ctrl)
	suite.handler = handlers.NewAlertsHandler(suite.mockAlerts)

	// Setup valid claims for authenticated requests
	suite.claims = &auth.AuthClaims{
		Username: "testuser",
		Email:    "test@example.com",
		UUID:     "550e8400-e29b-41d4-a716-446655440000",
	}

	// Setup router
	suite.router = gin.New()
	suite.router.GET("/api/v1/projects/:projectId/alerts", suite.handler.GetAlerts)
	suite.router.POST("/api/v1/projects/:projectId/alerts/pr", suite.handler.CreateAlertPR)
}

func (suite *AlertsHandlerTestSuite) TearDownTest() {
	suite.ctrl.Finish()
}

// Helper function to add auth claims to context
func (suite *AlertsHandlerTestSuite) addAuthMiddleware(claims *auth.AuthClaims) gin.HandlerFunc {
	return func(c *gin.Context) {
		if claims != nil {
			c.Set("auth_claims", claims)
		}
		c.Next()
	}
}

// GetAlerts Tests

// TestGetAlerts_Success verifies that GetAlerts returns alerts successfully with valid authentication
func (suite *AlertsHandlerTestSuite) TestGetAlerts_Success() {
	projectID := "test-project"
	expectedResponse := &service.AlertsResponse{
		Files: []service.AlertFile{
			{
				Name:     "test-alerts.yaml",
				Path:     "alerts/test-alerts.yaml",
				Content:  "alert: TestAlert",
				Category: "Test",
				Alerts: []map[string]interface{}{
					{
						"alert": "TestAlert",
						"expr":  "up == 0",
					},
				},
			},
		},
	}

	suite.mockAlerts.EXPECT().GetProjectAlerts(gomock.Any(), projectID, suite.claims.UUID, "githubtools").Return(expectedResponse, nil)

	// Setup router with auth middleware
	router := gin.New()
	router.Use(suite.addAuthMiddleware(suite.claims))
	router.GET("/api/v1/projects/:projectId/alerts", suite.handler.GetAlerts)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/"+projectID+"/alerts", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response service.AlertsResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), response.Files, 1)
	assert.Equal(suite.T(), "test-alerts.yaml", response.Files[0].Name)
}

// TestGetAlerts_NoAuthClaims verifies that GetAlerts returns 401 when authentication claims are missing
func (suite *AlertsHandlerTestSuite) TestGetAlerts_NoAuthClaims() {
	projectID := "test-project"

	// Setup router without auth middleware
	router := gin.New()
	router.GET("/api/v1/projects/:projectId/alerts", suite.handler.GetAlerts)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/"+projectID+"/alerts", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusUnauthorized, w.Code)
	assert.Contains(suite.T(), w.Body.String(), apperrors.ErrAuthenticationRequired.Message)
}

// TestGetAlerts_InvalidAuthClaims verifies that GetAlerts returns 500 when authentication claims have wrong type
func (suite *AlertsHandlerTestSuite) TestGetAlerts_InvalidAuthClaims() {
	projectID := "test-project"

	// Setup router with invalid claims (not *auth.AuthClaims)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("auth_claims", "invalid-claims")
		c.Next()
	})
	router.GET("/api/v1/projects/:projectId/alerts", suite.handler.GetAlerts)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/"+projectID+"/alerts", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusInternalServerError, w.Code)
	assert.Contains(suite.T(), w.Body.String(), apperrors.ErrAuthenticationInvalidClaims.Message)
}

// TestGetAlerts_MissingProjectID verifies that GetAlerts returns 400 when project ID parameter is missing
func (suite *AlertsHandlerTestSuite) TestGetAlerts_MissingProjectID() {
	// Setup router with auth middleware
	router := gin.New()
	router.Use(suite.addAuthMiddleware(suite.claims))
	router.GET("/api/v1/projects/:projectId/alerts", suite.handler.GetAlerts)

	// Request with empty projectId
	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects//alerts", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)
	assert.Contains(suite.T(), w.Body.String(), "projectId")
}

// TestGetAlerts_ProjectNotFound verifies that GetAlerts returns 404 when the project doesn't exist
func (suite *AlertsHandlerTestSuite) TestGetAlerts_ProjectNotFound() {
	projectID := "nonexistent-project"

	suite.mockAlerts.EXPECT().GetProjectAlerts(gomock.Any(), projectID, suite.claims.UUID, "githubtools").Return(nil, apperrors.ErrProjectNotFound)

	// Setup router with auth middleware
	router := gin.New()
	router.Use(suite.addAuthMiddleware(suite.claims))
	router.GET("/api/v1/projects/:projectId/alerts", suite.handler.GetAlerts)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/"+projectID+"/alerts", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusNotFound, w.Code)
	assert.Contains(suite.T(), w.Body.String(), apperrors.ErrProjectNotFound.Error())
}

// TestGetAlerts_AlertsRepositoryNotConfigured verifies that GetAlerts returns 404 when alerts repository is not configured
func (suite *AlertsHandlerTestSuite) TestGetAlerts_AlertsRepositoryNotConfigured() {
	projectID := "test-project"

	suite.mockAlerts.EXPECT().GetProjectAlerts(gomock.Any(), projectID, suite.claims.UUID, "githubtools").Return(nil, apperrors.ErrAlertsRepositoryNotConfigured)

	// Setup router with auth middleware
	router := gin.New()
	router.Use(suite.addAuthMiddleware(suite.claims))
	router.GET("/api/v1/projects/:projectId/alerts", suite.handler.GetAlerts)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/"+projectID+"/alerts", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusNotFound, w.Code)
	assert.Contains(suite.T(), w.Body.String(), apperrors.ErrAlertsRepositoryNotConfigured.Error())
}

// TestGetAlerts_InternalServerError verifies that GetAlerts returns 500 when service encounters an internal error
func (suite *AlertsHandlerTestSuite) TestGetAlerts_InternalServerError() {
	projectID := "test-project"
	expectedError := apperrors.ErrInternalError

	suite.mockAlerts.EXPECT().
		GetProjectAlerts(gomock.Any(), projectID, suite.claims.UUID, "githubtools").
		Return(nil, expectedError)

	// Setup router with auth middleware
	router := gin.New()
	router.Use(suite.addAuthMiddleware(suite.claims))
	router.GET("/api/v1/projects/:projectId/alerts", suite.handler.GetAlerts)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/"+projectID+"/alerts", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusInternalServerError, w.Code)
	assert.Contains(suite.T(), w.Body.String(), expectedError.Error())
}

// CreateAlertPR Tests

// TestCreateAlertPR_Success verifies that CreateAlertPR successfully creates a pull request with valid input
func (suite *AlertsHandlerTestSuite) TestCreateAlertPR_Success() {
	projectID := "test-project"
	expectedPRURL := "https://github.com/org/repo/pull/123"

	payload := map[string]string{
		"fileName":    "test-alerts.yaml",
		"content":     "alert: UpdatedAlert",
		"message":     "Update alert configuration",
		"description": "This PR updates the alert configuration",
	}

	suite.mockAlerts.EXPECT().
		CreateAlertPR(
			gomock.Any(),
			projectID,
			suite.claims.UUID,
			"githubtools",
			payload["fileName"],
			payload["content"],
			payload["message"],
			payload["description"],
		).
		Return(expectedPRURL, nil)

	// Setup router with auth middleware
	router := gin.New()
	router.Use(suite.addAuthMiddleware(suite.claims))
	router.POST("/api/v1/projects/:projectId/alerts/pr", suite.handler.CreateAlertPR)

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/"+projectID+"/alerts/pr", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "Pull request created successfully", response["message"])
	assert.Equal(suite.T(), expectedPRURL, response["prUrl"])
}

// TestCreateAlertPR_NoAuthClaims verifies that CreateAlertPR returns 401 when authentication claims are missing
func (suite *AlertsHandlerTestSuite) TestCreateAlertPR_NoAuthClaims() {
	projectID := "test-project"
	payload := map[string]string{
		"fileName": "test-alerts.yaml",
		"content":  "alert: UpdatedAlert",
		"message":  "Update alert",
	}

	// Setup router without auth middleware
	router := gin.New()
	router.POST("/api/v1/projects/:projectId/alerts/pr", suite.handler.CreateAlertPR)

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/"+projectID+"/alerts/pr", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusUnauthorized, w.Code)
}

// TestCreateAlertPR_InvalidAuthClaims verifies that CreateAlertPR returns 500 when authentication claims have wrong type
func (suite *AlertsHandlerTestSuite) TestCreateAlertPR_InvalidAuthClaims() {
	projectID := "test-project"
	payload := map[string]string{
		"fileName": "test-alerts.yaml",
		"content":  "alert: UpdatedAlert",
	}

	// Setup router with invalid claims
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("auth_claims", "invalid-claims")
		c.Next()
	})
	router.POST("/api/v1/projects/:projectId/alerts/pr", suite.handler.CreateAlertPR)

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/"+projectID+"/alerts/pr", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusInternalServerError, w.Code)
}

// TestCreateAlertPR_MissingProjectID verifies that CreateAlertPR returns 400 when project ID parameter is missing
func (suite *AlertsHandlerTestSuite) TestCreateAlertPR_MissingProjectID() {
	payload := map[string]string{
		"fileName": "test-alerts.yaml",
		"content":  "alert: UpdatedAlert",
	}

	// Setup router with auth middleware
	router := gin.New()
	router.Use(suite.addAuthMiddleware(suite.claims))
	router.POST("/api/v1/projects/:projectId/alerts/pr", suite.handler.CreateAlertPR)

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects//alerts/pr", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)
	assert.Contains(suite.T(), w.Body.String(), "projectId")
}

// TestCreateAlertPR_InvalidJSON verifies that CreateAlertPR returns 400 when request body contains invalid JSON
func (suite *AlertsHandlerTestSuite) TestCreateAlertPR_InvalidJSON() {
	projectID := "test-project"

	// Setup router with auth middleware
	router := gin.New()
	router.Use(suite.addAuthMiddleware(suite.claims))
	router.POST("/api/v1/projects/:projectId/alerts/pr", suite.handler.CreateAlertPR)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/"+projectID+"/alerts/pr", bytes.NewBufferString("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)
}

// TestCreateAlertPR_ProjectNotFound verifies that CreateAlertPR returns 404 when the project doesn't exist
func (suite *AlertsHandlerTestSuite) TestCreateAlertPR_ProjectNotFound() {
	projectID := "nonexistent-project"
	payload := map[string]string{
		"fileName":    "test-alerts.yaml",
		"content":     "alert: UpdatedAlert",
		"message":     "Update alert",
		"description": "Description",
	}

	suite.mockAlerts.EXPECT().
		CreateAlertPR(
			gomock.Any(),
			projectID,
			suite.claims.UUID,
			"githubtools",
			payload["fileName"],
			payload["content"],
			payload["message"],
			payload["description"],
		).
		Return("", apperrors.ErrProjectNotFound)

	// Setup router with auth middleware
	router := gin.New()
	router.Use(suite.addAuthMiddleware(suite.claims))
	router.POST("/api/v1/projects/:projectId/alerts/pr", suite.handler.CreateAlertPR)

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/"+projectID+"/alerts/pr", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusNotFound, w.Code)
	assert.Contains(suite.T(), w.Body.String(), apperrors.ErrProjectNotFound.Error())
}

// TestCreateAlertPR_InternalServerError verifies that CreateAlertPR returns 500 when service encounters an error
func (suite *AlertsHandlerTestSuite) TestCreateAlertPR_InternalServerError() {
	projectID := "test-project"
	expectedError := errors.New("failed to create PR")
	payload := map[string]string{
		"fileName":    "test-alerts.yaml",
		"content":     "alert: UpdatedAlert",
		"message":     "Update alert",
		"description": "Description",
	}

	suite.mockAlerts.EXPECT().
		CreateAlertPR(
			gomock.Any(),
			projectID,
			suite.claims.UUID,
			"githubtools",
			payload["fileName"],
			payload["content"],
			payload["message"],
			payload["description"],
		).
		Return("", expectedError)

	// Setup router with auth middleware
	router := gin.New()
	router.Use(suite.addAuthMiddleware(suite.claims))
	router.POST("/api/v1/projects/:projectId/alerts/pr", suite.handler.CreateAlertPR)

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/"+projectID+"/alerts/pr", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusInternalServerError, w.Code)
	assert.Contains(suite.T(), w.Body.String(), expectedError.Error())
}

// TestCreateAlertPR_EmptyPayloadFields verifies that CreateAlertPR handles empty payload fields gracefully
func (suite *AlertsHandlerTestSuite) TestCreateAlertPR_EmptyPayloadFields() {
	projectID := "test-project"
	payload := map[string]string{
		"fileName":    "",
		"content":     "",
		"message":     "",
		"description": "",
	}

	suite.mockAlerts.EXPECT().
		CreateAlertPR(
			gomock.Any(),
			projectID,
			suite.claims.UUID,
			"githubtools",
			"",
			"",
			"",
			"",
		).
		Return("https://github.com/org/repo/pull/123", nil)

	// Setup router with auth middleware
	router := gin.New()
	router.Use(suite.addAuthMiddleware(suite.claims))
	router.POST("/api/v1/projects/:projectId/alerts/pr", suite.handler.CreateAlertPR)

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/"+projectID+"/alerts/pr", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should still succeed if service accepts empty fields
	assert.Equal(suite.T(), http.StatusOK, w.Code)
}

func TestAlertsHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(AlertsHandlerTestSuite))
}
