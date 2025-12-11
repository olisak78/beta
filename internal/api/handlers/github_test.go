package handlers_test

import (
	"bytes"
	"developer-portal-backend/internal/mocks"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"developer-portal-backend/internal/api/handlers"
	"developer-portal-backend/internal/auth"
	apperrors "developer-portal-backend/internal/errors"
	"developer-portal-backend/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

// GitHubHandlerTestSuite defines the test suite for GitHubHandler
type GitHubHandlerTestSuite struct {
	suite.Suite
	router       *gin.Engine
	mockGitHubSv *mocks.MockGitHubServiceInterface
	handler      *handlers.GitHubHandler
	ctrl         *gomock.Controller
}

// SetupTest sets up the test suite
func (suite *GitHubHandlerTestSuite) SetupTest() {
	gin.SetMode(gin.TestMode)
	suite.ctrl = gomock.NewController(suite.T())
	suite.mockGitHubSv = mocks.NewMockGitHubServiceInterface(suite.ctrl)
	suite.handler = handlers.NewGitHubHandler(suite.mockGitHubSv)
	suite.router = gin.New()
}

// TearDownTest cleans up after each test
func (suite *GitHubHandlerTestSuite) TearDownTest() {
	suite.ctrl.Finish()
}

// TestGetMyPullRequests_Success tests successful PR retrieval
func (suite *GitHubHandlerTestSuite) TestGetMyPullRequests_Success() {
	// Setup mock expectation
	expectedResponse := &service.PullRequestsResponse{
		PullRequests: []service.PullRequest{
			{
				ID:        123456789,
				Number:    42,
				Title:     "Add new feature",
				State:     "open",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
				HTMLURL:   "https://github.com/owner/repo/pull/42",
				Draft:     false,
				User: service.GitHubUser{
					Login:     "testuser",
					ID:        12345,
					AvatarURL: "https://avatars.githubusercontent.com/u/12345",
				},
				Repo: service.Repository{
					Name:     "test-repo",
					FullName: "owner/test-repo",
					Owner:    "owner",
					Private:  false,
				},
			},
		},
		Total: 1,
	}

	suite.mockGitHubSv.EXPECT().
		GetUserOpenPullRequests(gomock.Any(), gomock.Any(), "githubtools", "open", "created", "desc", 30, 1).
		Return(expectedResponse, nil)

	// Setup route
	suite.router.GET("/github/pull-requests", func(c *gin.Context) {
		claims := &auth.AuthClaims{
			UUID:     "test-uuid",
			Username: "testuser",
			Email:    "test@example.com",
		}
		c.Set("auth_claims", claims)
		suite.handler.GetMyPullRequests(c)
	})

	// Make request
	req, _ := http.NewRequest(http.MethodGet, "/github/pull-requests", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Assert response
	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response service.PullRequestsResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), 1, response.Total)
	assert.Len(suite.T(), response.PullRequests, 1)
	assert.Equal(suite.T(), 42, response.PullRequests[0].Number)
	assert.Equal(suite.T(), "Add new feature", response.PullRequests[0].Title)
}

// TestGetMyPullRequests_Unauthorized tests missing authentication
func (suite *GitHubHandlerTestSuite) TestGetMyPullRequests_Unauthorized() {
	// Setup route without auth claims
	suite.router.GET("/github/pull-requests", suite.handler.GetMyPullRequests)

	// Make request
	req, _ := http.NewRequest(http.MethodGet, "/github/pull-requests", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Assert response
	assert.Equal(suite.T(), http.StatusUnauthorized, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	// The error is serialized as an object with a Message field
	errorObj, ok := response["error"].(map[string]interface{})
	assert.True(suite.T(), ok)
	assert.Equal(suite.T(), apperrors.ErrAuthenticationRequired.Error(), errorObj["Message"])
}

// TestGetMyPullRequests_ServiceError tests service error handling
func (suite *GitHubHandlerTestSuite) TestGetMyPullRequests_ServiceError() {
	suite.mockGitHubSv.EXPECT().
		GetUserOpenPullRequests(gomock.Any(), gomock.Any(), "githubtools", "open", "created", "desc", 30, 1).
		Return(nil, fmt.Errorf("failed to fetch pull requests"))

	suite.router.GET("/github/pull-requests", func(c *gin.Context) {
		claims := &auth.AuthClaims{
			UUID:     "test-uuid",
			Username: "testuser",
			Email:    "test@example.com",
		}
		c.Set("auth_claims", claims)
		suite.handler.GetMyPullRequests(c)
	})

	req, _ := http.NewRequest(http.MethodGet, "/github/pull-requests", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusBadGateway, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Contains(suite.T(), response["error"], "Failed to fetch pull requests")
}

// TestGetMyPullRequests_RateLimitError tests rate limit error handling
func (suite *GitHubHandlerTestSuite) TestGetMyPullRequests_RateLimitError() {
	suite.mockGitHubSv.EXPECT().
		GetUserOpenPullRequests(gomock.Any(), gomock.Any(), "githubtools", "open", "created", "desc", 30, 1).
		Return(nil, apperrors.ErrGitHubAPIRateLimitExceeded)

	suite.router.GET("/github/pull-requests", func(c *gin.Context) {
		claims := &auth.AuthClaims{
			UUID:     "test-uuid",
			Username: "testuser",
			Email:    "test@example.com",
		}
		c.Set("auth_claims", claims)
		suite.handler.GetMyPullRequests(c)
	})

	req, _ := http.NewRequest(http.MethodGet, "/github/pull-requests", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusTooManyRequests, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Contains(suite.T(), response["error"], "rate limit exceeded")
}

// TestGetMyPullRequests_WithQueryParameters tests query parameter handling
func (suite *GitHubHandlerTestSuite) TestGetMyPullRequests_WithQueryParameters() {
	suite.mockGitHubSv.EXPECT().
		GetUserOpenPullRequests(gomock.Any(), gomock.Any(), "githubtools", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(&service.PullRequestsResponse{
			PullRequests: []service.PullRequest{},
			Total:        0,
		}, nil).
		AnyTimes()

	suite.router.GET("/github/pull-requests", func(c *gin.Context) {
		claims := &auth.AuthClaims{
			UUID:     "test-uuid",
			Username: "testuser",
			Email:    "test@example.com",
		}
		c.Set("auth_claims", claims)
		suite.handler.GetMyPullRequests(c)
	})

	testCases := []struct {
		name           string
		queryParams    string
		expectedStatus int
	}{
		{"ValidOpenState", "?state=open", http.StatusOK},
		{"ValidClosedState", "?state=closed", http.StatusOK},
		{"ValidAllState", "?state=all", http.StatusOK},
		{"InvalidState", "?state=invalid", http.StatusBadRequest},
		{"ValidSort", "?sort=created", http.StatusOK},
		{"ValidUpdatedSort", "?sort=updated", http.StatusOK},
		{"InvalidSort", "?sort=invalid", http.StatusBadRequest},
		{"ValidDirection", "?direction=asc", http.StatusOK},
		{"InvalidDirection", "?direction=invalid", http.StatusBadRequest},
		{"ValidPerPage", "?per_page=50", http.StatusOK},
		{"ValidPage", "?page=2", http.StatusOK},
		{"MultipleParams", "?state=closed&sort=updated&direction=asc&per_page=50&page=2", http.StatusOK},
	}

	for _, tc := range testCases {
		suite.T().Run(tc.name, func(t *testing.T) {
			req, _ := http.NewRequest(http.MethodGet, "/github/pull-requests"+tc.queryParams, nil)
			w := httptest.NewRecorder()
			suite.router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedStatus, w.Code)
		})
	}
}

// TestGetMyPullRequests_InvalidClaims tests invalid claims type
func (suite *GitHubHandlerTestSuite) TestGetMyPullRequests_InvalidClaims() {
	suite.router.GET("/github/pull-requests", func(c *gin.Context) {
		// Set invalid claims type
		c.Set("auth_claims", "invalid")
		suite.handler.GetMyPullRequests(c)
	})

	req, _ := http.NewRequest(http.MethodGet, "/github/pull-requests", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusInternalServerError, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	// The error is serialized as an object with a Message field
	errorObj, ok := response["error"].(map[string]interface{})
	assert.True(suite.T(), ok)
	assert.Equal(suite.T(), apperrors.ErrAuthenticationInvalidClaims.Error(), errorObj["Message"])
}

// TestGetMyPullRequests_EmptyResponse tests empty PR list
func (suite *GitHubHandlerTestSuite) TestGetMyPullRequests_EmptyResponse() {
	suite.mockGitHubSv.EXPECT().
		GetUserOpenPullRequests(gomock.Any(), gomock.Any(), "githubtools", "open", "created", "desc", 30, 1).
		Return(&service.PullRequestsResponse{
			PullRequests: []service.PullRequest{},
			Total:        0,
		}, nil)

	suite.router.GET("/github/pull-requests", func(c *gin.Context) {
		claims := &auth.AuthClaims{
			UUID:     "test-uuid",
			Username: "testuser",
			Email:    "test@example.com",
		}
		c.Set("auth_claims", claims)
		suite.handler.GetMyPullRequests(c)
	})

	req, _ := http.NewRequest(http.MethodGet, "/github/pull-requests", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response service.PullRequestsResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), 0, response.Total)
	assert.Empty(suite.T(), response.PullRequests)
}

// TestGetMyPullRequests_MultiplePRs tests response with multiple PRs
func (suite *GitHubHandlerTestSuite) TestGetMyPullRequests_MultiplePRs() {
	suite.mockGitHubSv.EXPECT().
		GetUserOpenPullRequests(gomock.Any(), gomock.Any(), "githubtools", "open", "created", "desc", 30, 1).
		Return(&service.PullRequestsResponse{
			PullRequests: []service.PullRequest{
				{
					ID:     1,
					Number: 1,
					Title:  "PR 1",
					State:  "open",
				},
				{
					ID:     2,
					Number: 2,
					Title:  "PR 2",
					State:  "open",
				},
				{
					ID:     3,
					Number: 3,
					Title:  "PR 3",
					State:  "open",
				},
			},
			Total: 3,
		}, nil)

	suite.router.GET("/github/pull-requests", func(c *gin.Context) {
		claims := &auth.AuthClaims{
			UUID:     "test-uuid",
			Username: "testuser",
			Email:    "test@example.com",
		}
		c.Set("auth_claims", claims)
		suite.handler.GetMyPullRequests(c)
	})

	req, _ := http.NewRequest(http.MethodGet, "/github/pull-requests", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response service.PullRequestsResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), 3, response.Total)
	assert.Len(suite.T(), response.PullRequests, 3)
}

// TestGetMyPullRequests_DefaultParameters tests default parameter values
func (suite *GitHubHandlerTestSuite) TestGetMyPullRequests_DefaultParameters() {
	suite.mockGitHubSv.EXPECT().
		GetUserOpenPullRequests(gomock.Any(), gomock.Any(), "githubtools", "open", "created", "desc", 30, 1).
		Return(&service.PullRequestsResponse{
			PullRequests: []service.PullRequest{},
			Total:        0,
		}, nil)

	suite.router.GET("/github/pull-requests", func(c *gin.Context) {
		claims := &auth.AuthClaims{
			UUID:     "test-uuid",
			Username: "testuser",
			Email:    "test@example.com",
		}
		c.Set("auth_claims", claims)
		suite.handler.GetMyPullRequests(c)
	})

	// Request without any query parameters
	req, _ := http.NewRequest(http.MethodGet, "/github/pull-requests", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)
}

// TestGetMyPullRequests_InvalidPerPage tests invalid per_page values
func (suite *GitHubHandlerTestSuite) TestGetMyPullRequests_InvalidPerPage() {
	suite.mockGitHubSv.EXPECT().
		GetUserOpenPullRequests(gomock.Any(), gomock.Any(), "githubtools", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(&service.PullRequestsResponse{
			PullRequests: []service.PullRequest{},
			Total:        0,
		}, nil).
		AnyTimes()

	suite.router.GET("/github/pull-requests", func(c *gin.Context) {
		claims := &auth.AuthClaims{
			UUID:     "test-uuid",
			Username: "testuser",
			Email:    "test@example.com",
		}
		c.Set("auth_claims", claims)
		suite.handler.GetMyPullRequests(c)
	})

	testCases := []string{
		"?per_page=abc", // Non-numeric
		"?per_page=-1",  // Negative
		"?per_page=0",   // Zero
		"?per_page=200", // Too large
		"?per_page=",    // Empty
	}

	for _, queryParam := range testCases {
		req, _ := http.NewRequest(http.MethodGet, "/github/pull-requests"+queryParam, nil)
		w := httptest.NewRecorder()
		suite.router.ServeHTTP(w, req)

		// Should still succeed with defaults
		assert.Equal(suite.T(), http.StatusOK, w.Code)
	}
}

// TestGetMyPullRequests_InvalidPage tests invalid page values
func (suite *GitHubHandlerTestSuite) TestGetMyPullRequests_InvalidPage() {
	suite.mockGitHubSv.EXPECT().
		GetUserOpenPullRequests(gomock.Any(), gomock.Any(), "githubtools", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(&service.PullRequestsResponse{
			PullRequests: []service.PullRequest{},
			Total:        0,
		}, nil).
		AnyTimes()

	suite.router.GET("/github/pull-requests", func(c *gin.Context) {
		claims := &auth.AuthClaims{
			UUID:     "test-uuid",
			Username: "testuser",
			Email:    "test@example.com",
		}
		c.Set("auth_claims", claims)
		suite.handler.GetMyPullRequests(c)
	})

	testCases := []string{
		"?page=abc", // Non-numeric
		"?page=-1",  // Negative
		"?page=0",   // Zero
		"?page=",    // Empty
	}

	for _, queryParam := range testCases {
		req, _ := http.NewRequest(http.MethodGet, "/github/pull-requests"+queryParam, nil)
		w := httptest.NewRecorder()
		suite.router.ServeHTTP(w, req)

		// Should still succeed with defaults
		assert.Equal(suite.T(), http.StatusOK, w.Code)
	}
}

// TestNewGitHubHandler tests handler creation
func (suite *GitHubHandlerTestSuite) TestNewGitHubHandler() {
	handler := handlers.NewGitHubHandler(suite.mockGitHubSv)

	assert.NotNil(suite.T(), handler)
}

// TestGetMyPullRequests_DifferentProviders tests different provider values
func (suite *GitHubHandlerTestSuite) TestGetMyPullRequests_DifferentProviders() {
	suite.mockGitHubSv.EXPECT().
		GetUserOpenPullRequests(gomock.Any(), gomock.Any(), "githubtools", "open", "created", "desc", 30, 1).
		Return(&service.PullRequestsResponse{
			PullRequests: []service.PullRequest{},
			Total:        0,
		}, nil).
		AnyTimes()

	providers := []string{"githubtools", "githubwdf"}

	for _, provider := range providers {
		suite.T().Run(provider, func(t *testing.T) {
			router := gin.New()
			router.GET("/github/pull-requests", func(c *gin.Context) {
				claims := &auth.AuthClaims{
					UUID:     "test-uuid",
					Username: "testuser",
					Email:    "test@example.com",
				}
				c.Set("auth_claims", claims)
				suite.handler.GetMyPullRequests(c)
			})

			req, _ := http.NewRequest(http.MethodGet, "/github/pull-requests", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
		})
	}
}

// TestGetUserTotalContributions_Success tests successful contribution retrieval
func (suite *GitHubHandlerTestSuite) TestGetUserTotalContributions_Success() {
	suite.mockGitHubSv.EXPECT().
		GetUserTotalContributions(gomock.Any(), gomock.Any(), "githubtools", "30d").
		Return(&service.TotalContributionsResponse{
			TotalContributions: 1234,
			Period:             "30d",
			From:               "2024-10-16T00:00:00Z",
			To:                 "2025-10-16T23:59:59Z",
		}, nil)

	suite.router.GET("/github/contributions", func(c *gin.Context) {
		claims := &auth.AuthClaims{
			UUID:     "test-uuid",
			Username: "testuser",
			Email:    "test@example.com",
		}
		c.Set("auth_claims", claims)
		suite.handler.GetUserTotalContributions(c)
	})

	req, _ := http.NewRequest(http.MethodGet, "/github/contributions?period=30d", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response service.TotalContributionsResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), 1234, response.TotalContributions)
	assert.Equal(suite.T(), "30d", response.Period)
	assert.NotEmpty(suite.T(), response.From)
	assert.NotEmpty(suite.T(), response.To)
}

// TestGetUserTotalContributions_Unauthorized tests missing authentication
func (suite *GitHubHandlerTestSuite) TestGetUserTotalContributions_Unauthorized() {
	// Setup route without auth claims
	suite.router.GET("/github/contributions", suite.handler.GetUserTotalContributions)

	req, _ := http.NewRequest(http.MethodGet, "/github/contributions", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusUnauthorized, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	// The error is serialized as an object with a Message field
	errorObj, ok := response["error"].(map[string]interface{})
	assert.True(suite.T(), ok)
	assert.Equal(suite.T(), apperrors.ErrAuthenticationRequired.Error(), errorObj["Message"])
}

// TestGetUserTotalContributions_InvalidClaims tests invalid claims type
func (suite *GitHubHandlerTestSuite) TestGetUserTotalContributions_InvalidClaims() {
	suite.router.GET("/github/contributions", func(c *gin.Context) {
		// Set invalid claims type
		c.Set("auth_claims", "invalid")
		suite.handler.GetUserTotalContributions(c)
	})

	req, _ := http.NewRequest(http.MethodGet, "/github/contributions", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusInternalServerError, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	// The error is serialized as an object with a Message field
	errorObj, ok := response["error"].(map[string]interface{})
	assert.True(suite.T(), ok)
	assert.Equal(suite.T(), apperrors.ErrAuthenticationInvalidClaims.Error(), errorObj["Message"])
}

// TestGetUserTotalContributions_InvalidPeriod tests invalid period format
func (suite *GitHubHandlerTestSuite) TestGetUserTotalContributions_InvalidPeriod() {
	suite.mockGitHubSv.EXPECT().
		GetUserTotalContributions(gomock.Any(), gomock.Any(), "githubtools", gomock.Any()).
		Return(nil, fmt.Errorf("%w: period must be in format '<number>d' (e.g., '30d', '90d', '365d')", apperrors.ErrInvalidPeriodFormat)).
		AnyTimes()

	suite.router.GET("/github/contributions", func(c *gin.Context) {
		claims := &auth.AuthClaims{
			UUID:     "test-uuid",
			Username: "testuser",
			Email:    "test@example.com",
		}
		c.Set("auth_claims", claims)
		suite.handler.GetUserTotalContributions(c)
	})

	testCases := []string{"30", "abc", "30days", "-30d", "0d"}

	for _, period := range testCases {
		suite.T().Run(period, func(t *testing.T) {
			req, _ := http.NewRequest(http.MethodGet, "/github/contributions?period="+period, nil)
			w := httptest.NewRecorder()
			suite.router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusBadRequest, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.Contains(t, response["error"], "invalid period format")
		})
	}
}

// TestGetUserTotalContributions_RateLimit tests rate limit error handling
func (suite *GitHubHandlerTestSuite) TestGetUserTotalContributions_RateLimit() {
	suite.mockGitHubSv.EXPECT().
		GetUserTotalContributions(gomock.Any(), gomock.Any(), "githubtools", "30d").
		Return(nil, apperrors.ErrGitHubAPIRateLimitExceeded)

	suite.router.GET("/github/contributions", func(c *gin.Context) {
		claims := &auth.AuthClaims{
			UUID:     "test-uuid",
			Username: "testuser",
			Email:    "test@example.com",
		}
		c.Set("auth_claims", claims)
		suite.handler.GetUserTotalContributions(c)
	})

	req, _ := http.NewRequest(http.MethodGet, "/github/contributions?period=30d", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusTooManyRequests, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Contains(suite.T(), response["error"], "rate limit exceeded")
}

// TestGetUserTotalContributions_ServiceError tests service error handling
func (suite *GitHubHandlerTestSuite) TestGetUserTotalContributions_ServiceError() {
	suite.mockGitHubSv.EXPECT().
		GetUserTotalContributions(gomock.Any(), gomock.Any(), "githubtools", "30d").
		Return(nil, fmt.Errorf("failed to fetch contributions"))

	suite.router.GET("/github/contributions", func(c *gin.Context) {
		claims := &auth.AuthClaims{
			UUID:     "test-uuid",
			Username: "testuser",
			Email:    "test@example.com",
		}
		c.Set("auth_claims", claims)
		suite.handler.GetUserTotalContributions(c)
	})

	req, _ := http.NewRequest(http.MethodGet, "/github/contributions?period=30d", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusBadGateway, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Contains(suite.T(), response["error"], "Failed to fetch contributions")
}

// TestGetUserTotalContributions_ValidPeriods tests various valid period values
func (suite *GitHubHandlerTestSuite) TestGetUserTotalContributions_ValidPeriods() {
	suite.mockGitHubSv.EXPECT().
		GetUserTotalContributions(gomock.Any(), gomock.Any(), "githubtools", gomock.Any()).
		Return(&service.TotalContributionsResponse{
			TotalContributions: 1234,
			Period:             "30d",
			From:               "2024-10-16T00:00:00Z",
			To:                 "2025-10-16T23:59:59Z",
		}, nil).
		AnyTimes()

	suite.router.GET("/github/contributions", func(c *gin.Context) {
		claims := &auth.AuthClaims{
			UUID:     "test-uuid",
			Username: "testuser",
			Email:    "test@example.com",
		}
		c.Set("auth_claims", claims)
		suite.handler.GetUserTotalContributions(c)
	})

	validPeriods := []string{"7d", "30d", "90d", "180d", "365d"}

	for _, period := range validPeriods {
		suite.T().Run(period, func(t *testing.T) {
			req, _ := http.NewRequest(http.MethodGet, "/github/contributions?period="+period, nil)
			w := httptest.NewRecorder()
			suite.router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)

			var response service.TotalContributionsResponse
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.Equal(t, 1234, response.TotalContributions)
		})
	}
}

// TestGetUserTotalContributions_DifferentProviders tests different provider values
func (suite *GitHubHandlerTestSuite) TestGetUserTotalContributions_DifferentProviders() {
	suite.mockGitHubSv.EXPECT().
		GetUserTotalContributions(gomock.Any(), gomock.Any(), gomock.Any(), "30d").
		Return(&service.TotalContributionsResponse{
			TotalContributions: 1234,
			Period:             "30d",
			From:               "2024-10-16T00:00:00Z",
			To:                 "2025-10-16T23:59:59Z",
		}, nil).
		AnyTimes()

	providers := []string{"githubtools", "githubwdf"}

	for _, provider := range providers {
		suite.T().Run(provider, func(t *testing.T) {
			router := gin.New()
			router.GET("/github/contributions", func(c *gin.Context) {
				claims := &auth.AuthClaims{
					UUID:     "test-uuid",
					Username: "testuser",
					Email:    "test@example.com",
				}
				c.Set("auth_claims", claims)
				suite.handler.GetUserTotalContributions(c)
			})

			req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/github/contributions?period=30d&provider=%s", provider), nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
		})
	}
}

// TestGetContributionsHeatmap_Success tests successful heatmap retrieval
func (suite *GitHubHandlerTestSuite) TestGetContributionsHeatmap_Success() {
	suite.mockGitHubSv.EXPECT().
		GetContributionsHeatmap(gomock.Any(), gomock.Any(), "githubtools", "").
		Return(&service.ContributionsHeatmapResponse{
			TotalContributions: 1234,
			Weeks: []service.ContributionWeek{
				{
					FirstDay: "2024-10-27",
					ContributionDays: []service.ContributionDay{
						{
							Date:              "2024-10-27",
							ContributionCount: 5,
							ContributionLevel: "SECOND_QUARTILE",
							Color:             "#40c463",
						},
						{
							Date:              "2024-10-28",
							ContributionCount: 10,
							ContributionLevel: "THIRD_QUARTILE",
							Color:             "#30a14e",
						},
					},
				},
			},
			From: "2024-10-30T00:00:00Z",
			To:   "2025-10-30T23:59:59Z",
		}, nil)

	suite.router.GET("/github/:provider/heatmap", func(c *gin.Context) {
		claims := &auth.AuthClaims{
			UUID:     "test-uuid",
			Username: "testuser",
			Email:    "test@example.com",
		}
		c.Set("auth_claims", claims)
		suite.handler.GetContributionsHeatmap(c)
	})

	req, _ := http.NewRequest(http.MethodGet, "/github/githubtools/heatmap", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response service.ContributionsHeatmapResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), 1234, response.TotalContributions)
	assert.Equal(suite.T(), "2024-10-30T00:00:00Z", response.From)
	assert.Equal(suite.T(), "2025-10-30T23:59:59Z", response.To)
	assert.Len(suite.T(), response.Weeks, 1)
	assert.Equal(suite.T(), "2024-10-27", response.Weeks[0].FirstDay)
	assert.Len(suite.T(), response.Weeks[0].ContributionDays, 2)
}

// TestGetContributionsHeatmap_WithPeriod tests heatmap with period parameter
func (suite *GitHubHandlerTestSuite) TestGetContributionsHeatmap_WithPeriod() {
	suite.mockGitHubSv.EXPECT().
		GetContributionsHeatmap(gomock.Any(), gomock.Any(), "githubtools", "90d").
		Return(&service.ContributionsHeatmapResponse{
			TotalContributions: 1234,
			Weeks:              []service.ContributionWeek{},
			From:               "2024-10-30T00:00:00Z",
			To:                 "2025-10-30T23:59:59Z",
		}, nil)

	suite.router.GET("/github/:provider/heatmap", func(c *gin.Context) {
		claims := &auth.AuthClaims{
			UUID:     "test-uuid",
			Username: "testuser",
			Email:    "test@example.com",
		}
		c.Set("auth_claims", claims)
		suite.handler.GetContributionsHeatmap(c)
	})

	req, _ := http.NewRequest(http.MethodGet, "/github/githubtools/heatmap?period=90d", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response service.ContributionsHeatmapResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), response)
}

// TestGetContributionsHeatmap_NoAuthClaims tests missing auth claims
func (suite *GitHubHandlerTestSuite) TestGetContributionsHeatmap_NoAuthClaims() {
	suite.router.GET("/github/:provider/heatmap", suite.handler.GetContributionsHeatmap)

	req, _ := http.NewRequest(http.MethodGet, "/github/githubtools/heatmap", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusUnauthorized, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	// The error is serialized as an object with a Message field
	errorObj, ok := response["error"].(map[string]interface{})
	assert.True(suite.T(), ok)
	assert.Equal(suite.T(), apperrors.ErrAuthenticationRequired.Error(), errorObj["Message"])
}

// TestGetContributionsHeatmap_ProviderMismatch tests provider mismatch
func (suite *GitHubHandlerTestSuite) TestGetContributionsHeatmap_ProviderMismatch() {
	// Set up mock expectation for the default provider (githubtools)
	suite.mockGitHubSv.EXPECT().
		GetContributionsHeatmap(gomock.Any(), gomock.Any(), "githubtools", "").
		Return(&service.ContributionsHeatmapResponse{
			TotalContributions: 1234,
			Weeks:              []service.ContributionWeek{},
			From:               "2024-10-30T00:00:00Z",
			To:                 "2025-10-30T23:59:59Z",
		}, nil)

	suite.router.GET("/github/:provider/heatmap", func(c *gin.Context) {
		claims := &auth.AuthClaims{
			UUID:     "test-uuid",
			Username: "testuser",
			Email:    "test@example.com",
			// User is authenticated with githubtools
		}
		c.Set("auth_claims", claims)
		suite.handler.GetContributionsHeatmap(c)
	})

	// Request with a different provider in the path; handler uses query param with default provider
	req, _ := http.NewRequest(http.MethodGet, "/github/githubwdf/heatmap", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// With current implementation, provider is taken from query (default githubtools), so this should succeed
	assert.Equal(suite.T(), http.StatusOK, w.Code)
}

// TestGetContributionsHeatmap_ServiceError tests service error handling
func (suite *GitHubHandlerTestSuite) TestGetContributionsHeatmap_ServiceError() {
	suite.mockGitHubSv.EXPECT().
		GetContributionsHeatmap(gomock.Any(), gomock.Any(), "githubtools", "").
		Return(nil, fmt.Errorf("service error"))

	suite.router.GET("/github/:provider/heatmap", func(c *gin.Context) {
		claims := &auth.AuthClaims{
			UUID:     "test-uuid",
			Username: "testuser",
			Email:    "test@example.com",
		}
		c.Set("auth_claims", claims)
		suite.handler.GetContributionsHeatmap(c)
	})

	req, _ := http.NewRequest(http.MethodGet, "/github/githubtools/heatmap", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusBadGateway, w.Code)
}

// TestGetContributionsHeatmap_RateLimitExceeded tests rate limit error
func (suite *GitHubHandlerTestSuite) TestGetContributionsHeatmap_RateLimitExceeded() {
	suite.mockGitHubSv.EXPECT().
		GetContributionsHeatmap(gomock.Any(), gomock.Any(), "githubtools", "").
		Return(nil, apperrors.ErrGitHubAPIRateLimitExceeded)

	suite.router.GET("/github/:provider/heatmap", func(c *gin.Context) {
		claims := &auth.AuthClaims{
			UUID:     "test-uuid",
			Username: "testuser",
			Email:    "test@example.com",
		}
		c.Set("auth_claims", claims)
		suite.handler.GetContributionsHeatmap(c)
	})

	req, _ := http.NewRequest(http.MethodGet, "/github/githubtools/heatmap", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusTooManyRequests, w.Code)
}

// TestGetContributionsHeatmap_InvalidPeriod tests invalid period format
func (suite *GitHubHandlerTestSuite) TestGetContributionsHeatmap_InvalidPeriod() {
	suite.mockGitHubSv.EXPECT().
		GetContributionsHeatmap(gomock.Any(), gomock.Any(), "githubtools", "invalid").
		Return(nil, fmt.Errorf("%w: period must be in format '<number>d'", apperrors.ErrInvalidPeriodFormat))

	suite.router.GET("/github/:provider/heatmap", func(c *gin.Context) {
		claims := &auth.AuthClaims{
			UUID:     "test-uuid",
			Username: "testuser",
			Email:    "test@example.com",
		}
		c.Set("auth_claims", claims)
		suite.handler.GetContributionsHeatmap(c)
	})

	req, _ := http.NewRequest(http.MethodGet, "/github/githubtools/heatmap?period=invalid", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)
}

// TestGetContributionsHeatmap_ProviderNotConfigured tests provider not configured error
func (suite *GitHubHandlerTestSuite) TestGetContributionsHeatmap_ProviderNotConfigured() {
	suite.mockGitHubSv.EXPECT().
		GetContributionsHeatmap(gomock.Any(), gomock.Any(), "githubtools", "").
		Return(nil, fmt.Errorf("%w: provider 'invalid'. Please check available providers in auth.yaml", apperrors.ErrProviderNotConfigured))

	suite.router.GET("/github/:provider/heatmap", func(c *gin.Context) {
		claims := &auth.AuthClaims{
			UUID:     "test-uuid",
			Username: "testuser",
			Email:    "test@example.com",
		}
		c.Set("auth_claims", claims)
		suite.handler.GetContributionsHeatmap(c)
	})

	req, _ := http.NewRequest(http.MethodGet, "/github/invalid/heatmap", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Contains(suite.T(), response["error"], apperrors.ErrProviderNotConfigured.Error())
}

// TestGetAveragePRMergeTime_Success tests successful average PR merge time retrieval
func (suite *GitHubHandlerTestSuite) TestGetAveragePRMergeTime_Success() {
	suite.mockGitHubSv.EXPECT().
		GetAveragePRMergeTime(gomock.Any(), gomock.Any(), "githubtools", "30d").
		Return(&service.AveragePRMergeTimeResponse{
			AveragePRMergeTimeHours: 24.5,
			PRCount:                 15,
			Period:                  "30d",
			From:                    "2024-10-03T00:00:00Z",
			To:                      "2024-11-02T23:59:59Z",
			TimeSeries: []service.PRMergeTimeDataPoint{
				{
					WeekStart:    "2024-10-26",
					WeekEnd:      "2024-11-02",
					AverageHours: 18.5,
					PRCount:      3,
				},
				{
					WeekStart:    "2024-10-19",
					WeekEnd:      "2024-10-26",
					AverageHours: 22.0,
					PRCount:      2,
				},
				{
					WeekStart:    "2024-10-12",
					WeekEnd:      "2024-10-19",
					AverageHours: 30.0,
					PRCount:      5,
				},
				{
					WeekStart:    "2024-10-05",
					WeekEnd:      "2024-10-12",
					AverageHours: 25.5,
					PRCount:      5,
				},
			},
		}, nil)

	suite.router.GET("/github/average-pr-time", func(c *gin.Context) {
		claims := &auth.AuthClaims{
			UUID:     "test-uuid",
			Username: "testuser",
			Email:    "test@example.com",
		}
		c.Set("auth_claims", claims)
		suite.handler.GetAveragePRMergeTime(c)
	})

	req, _ := http.NewRequest(http.MethodGet, "/github/average-pr-time", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response service.AveragePRMergeTimeResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), 24.5, response.AveragePRMergeTimeHours)
	assert.Equal(suite.T(), 15, response.PRCount)
	assert.Equal(suite.T(), "2024-10-03T00:00:00Z", response.From)
	assert.Equal(suite.T(), "2024-11-02T23:59:59Z", response.To)
	assert.Len(suite.T(), response.TimeSeries, 4)
}

// TestGetAveragePRMergeTime_NoAuthClaims tests missing auth claims
func (suite *GitHubHandlerTestSuite) TestGetAveragePRMergeTime_NoAuthClaims() {
	suite.router.GET("/github/average-pr-time", suite.handler.GetAveragePRMergeTime)

	req, _ := http.NewRequest(http.MethodGet, "/github/average-pr-time", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusUnauthorized, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	// The error is serialized as an object with a Message field
	errorObj, ok := response["error"].(map[string]interface{})
	assert.True(suite.T(), ok)
	assert.Equal(suite.T(), apperrors.ErrAuthenticationRequired.Error(), errorObj["Message"])
}

// TestGetAveragePRMergeTime_ServiceError tests service error handling
func (suite *GitHubHandlerTestSuite) TestGetAveragePRMergeTime_ServiceError() {
	suite.mockGitHubSv.EXPECT().
		GetAveragePRMergeTime(gomock.Any(), gomock.Any(), "githubtools", "30d").
		Return(nil, fmt.Errorf("GitHub API error"))

	suite.router.GET("/github/average-pr-time", func(c *gin.Context) {
		claims := &auth.AuthClaims{
			UUID:     "test-uuid",
			Username: "testuser",
			Email:    "test@example.com",
		}
		c.Set("auth_claims", claims)
		suite.handler.GetAveragePRMergeTime(c)
	})

	req, _ := http.NewRequest(http.MethodGet, "/github/average-pr-time", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusBadGateway, w.Code)
}

// TestGetAveragePRMergeTime_RateLimitExceeded tests rate limit error
func (suite *GitHubHandlerTestSuite) TestGetAveragePRMergeTime_RateLimitExceeded() {
	suite.mockGitHubSv.EXPECT().
		GetAveragePRMergeTime(gomock.Any(), gomock.Any(), "githubtools", "30d").
		Return(nil, apperrors.ErrGitHubAPIRateLimitExceeded)

	suite.router.GET("/github/average-pr-time", func(c *gin.Context) {
		claims := &auth.AuthClaims{
			UUID:     "test-uuid",
			Username: "testuser",
			Email:    "test@example.com",
		}
		c.Set("auth_claims", claims)
		suite.handler.GetAveragePRMergeTime(c)
	})

	req, _ := http.NewRequest(http.MethodGet, "/github/average-pr-time", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusTooManyRequests, w.Code)
}

// TestGetAveragePRMergeTime_InvalidPeriod tests invalid period format
func (suite *GitHubHandlerTestSuite) TestGetAveragePRMergeTime_InvalidPeriod() {
	suite.mockGitHubSv.EXPECT().
		GetAveragePRMergeTime(gomock.Any(), gomock.Any(), "githubtools", "invalid").
		Return(nil, apperrors.ErrInvalidPeriodFormat)

	suite.router.GET("/github/average-pr-time", func(c *gin.Context) {
		claims := &auth.AuthClaims{
			UUID:     "test-uuid",
			Username: "testuser",
			Email:    "test@example.com",
		}
		c.Set("auth_claims", claims)
		suite.handler.GetAveragePRMergeTime(c)
	})

	req, _ := http.NewRequest(http.MethodGet, "/github/average-pr-time?period=invalid", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)
}

// ClosePullRequest handler tests

func (suite *GitHubHandlerTestSuite) TestClosePR_Success() {
	suite.mockGitHubSv.EXPECT().
		ClosePullRequest(gomock.Any(), gomock.Any(), "githubtools", "owner", "repo", 42, false).
		Return(&service.PullRequest{
			ID:     1,
			Number: 42,
			Title:  "Closed PR",
			State:  "closed",
			User:   service.GitHubUser{Login: "testuser", ID: 12345, AvatarURL: "https://avatars.githubusercontent.com/u/12345"},
			Repo:   service.Repository{Name: "repo", FullName: "owner/repo", Owner: "owner", Private: false},
		}, nil)

	// Route for ClosePullRequest
	suite.router.PATCH("/github/pull-requests/:pr_number", func(c *gin.Context) {
		claims := &auth.AuthClaims{UUID: "test-uuid"}
		c.Set("auth_claims", claims)
		suite.handler.ClosePullRequest(c)
	})

	// Prepare request
	body := map[string]interface{}{
		"owner":         "owner",
		"repo":          "repo",
		"delete_branch": false,
	}
	payload, _ := json.Marshal(body)

	req, _ := http.NewRequest(http.MethodPatch, "/github/pull-requests/42", bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var pr service.PullRequest
	err := json.Unmarshal(w.Body.Bytes(), &pr)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "closed", pr.State)
	assert.Equal(suite.T(), 42, pr.Number)
	assert.Equal(suite.T(), "owner/repo", pr.Repo.FullName)
}

func (suite *GitHubHandlerTestSuite) TestClosePR_NotFound() {
	suite.mockGitHubSv.EXPECT().
		ClosePullRequest(gomock.Any(), gomock.Any(), "githubtools", "owner", "repo", 99, false).
		Return(nil, apperrors.NewNotFoundError("pull request"))

	// Route for ClosePullRequest
	suite.router.PATCH("/github/pull-requests/:pr_number", func(c *gin.Context) {
		claims := &auth.AuthClaims{UUID: "test-uuid"}
		c.Set("auth_claims", claims)
		suite.handler.ClosePullRequest(c)
	})

	// Prepare request
	body := map[string]interface{}{
		"owner":         "owner",
		"repo":          "repo",
		"delete_branch": false,
	}
	payload, _ := json.Marshal(body)

	req, _ := http.NewRequest(http.MethodPatch, "/github/pull-requests/99", bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusNotFound, w.Code)
}

// GetRepositoryContent handler tests

// TestGetRepositoryContent_Success tests successful file content retrieval
func (suite *GitHubHandlerTestSuite) TestGetRepositoryContent_Success() {
	expectedContent := map[string]interface{}{
		"name":    "README.md",
		"path":    "README.md",
		"content": "base64content",
		"type":    "file",
	}

	suite.mockGitHubSv.EXPECT().
		GetRepositoryContent(gomock.Any(), gomock.Any(), "githubtools", "owner", "repo", "/README.md", "main").
		Return(expectedContent, nil)

	suite.router.GET("/github/repos/:owner/:repo/contents/*path", func(c *gin.Context) {
		claims := &auth.AuthClaims{UUID: "test-uuid"}
		c.Set("auth_claims", claims)
		suite.handler.GetRepositoryContent(c)
	})

	req, _ := http.NewRequest(http.MethodGet, "/github/repos/owner/repo/contents/README.md", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "README.md", response["name"])
}

// TestGetRepositoryContent_SuccessDirectory tests successful directory listing
func (suite *GitHubHandlerTestSuite) TestGetRepositoryContent_SuccessDirectory() {
	expectedContent := []interface{}{
		map[string]interface{}{"name": "file1.txt", "type": "file"},
		map[string]interface{}{"name": "file2.txt", "type": "file"},
	}

	suite.mockGitHubSv.EXPECT().
		GetRepositoryContent(gomock.Any(), gomock.Any(), "githubtools", "owner", "repo", "/src", "main").
		Return(expectedContent, nil)

	suite.router.GET("/github/repos/:owner/:repo/contents/*path", func(c *gin.Context) {
		claims := &auth.AuthClaims{UUID: "test-uuid"}
		c.Set("auth_claims", claims)
		suite.handler.GetRepositoryContent(c)
	})

	req, _ := http.NewRequest(http.MethodGet, "/github/repos/owner/repo/contents/src", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)
}

// TestGetRepositoryContent_Unauthorized tests missing auth claims
func (suite *GitHubHandlerTestSuite) TestGetRepositoryContent_Unauthorized() {
	suite.router.GET("/github/repos/:owner/:repo/contents/*path", suite.handler.GetRepositoryContent)

	req, _ := http.NewRequest(http.MethodGet, "/github/repos/owner/repo/contents/file.txt", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusUnauthorized, w.Code)
}

// TestGetRepositoryContent_InvalidClaims tests invalid claims type
func (suite *GitHubHandlerTestSuite) TestGetRepositoryContent_InvalidClaims() {
	suite.router.GET("/github/repos/:owner/:repo/contents/*path", func(c *gin.Context) {
		c.Set("auth_claims", "invalid")
		suite.handler.GetRepositoryContent(c)
	})

	req, _ := http.NewRequest(http.MethodGet, "/github/repos/owner/repo/contents/file.txt", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusInternalServerError, w.Code)
}

// TestGetRepositoryContent_NotFound tests repository or path not found
func (suite *GitHubHandlerTestSuite) TestGetRepositoryContent_NotFound() {
	suite.mockGitHubSv.EXPECT().
		GetRepositoryContent(gomock.Any(), gomock.Any(), "githubtools", "owner", "repo", "/notfound.txt", "main").
		Return(nil, apperrors.NewNotFoundError("file"))

	suite.router.GET("/github/repos/:owner/:repo/contents/*path", func(c *gin.Context) {
		claims := &auth.AuthClaims{UUID: "test-uuid"}
		c.Set("auth_claims", claims)
		suite.handler.GetRepositoryContent(c)
	})

	req, _ := http.NewRequest(http.MethodGet, "/github/repos/owner/repo/contents/notfound.txt", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusNotFound, w.Code)
}

// TestGetRepositoryContent_RateLimitExceeded tests rate limit error
func (suite *GitHubHandlerTestSuite) TestGetRepositoryContent_RateLimitExceeded() {
	suite.mockGitHubSv.EXPECT().
		GetRepositoryContent(gomock.Any(), gomock.Any(), "githubtools", "owner", "repo", "/file.txt", "main").
		Return(nil, apperrors.ErrGitHubAPIRateLimitExceeded)

	suite.router.GET("/github/repos/:owner/:repo/contents/*path", func(c *gin.Context) {
		claims := &auth.AuthClaims{UUID: "test-uuid"}
		c.Set("auth_claims", claims)
		suite.handler.GetRepositoryContent(c)
	})

	req, _ := http.NewRequest(http.MethodGet, "/github/repos/owner/repo/contents/file.txt", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusTooManyRequests, w.Code)
}

// TestGetRepositoryContent_ServiceError tests generic service error
func (suite *GitHubHandlerTestSuite) TestGetRepositoryContent_ServiceError() {
	suite.mockGitHubSv.EXPECT().
		GetRepositoryContent(gomock.Any(), gomock.Any(), "githubtools", "owner", "repo", "/file.txt", "main").
		Return(nil, fmt.Errorf("service error"))

	suite.router.GET("/github/repos/:owner/:repo/contents/*path", func(c *gin.Context) {
		claims := &auth.AuthClaims{UUID: "test-uuid"}
		c.Set("auth_claims", claims)
		suite.handler.GetRepositoryContent(c)
	})

	req, _ := http.NewRequest(http.MethodGet, "/github/repos/owner/repo/contents/file.txt", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusBadGateway, w.Code)
}

// TestGetRepositoryContent_EmptyPath tests root directory content
func (suite *GitHubHandlerTestSuite) TestGetRepositoryContent_EmptyPath() {
	expectedContent := []interface{}{
		map[string]interface{}{"name": "README.md", "type": "file"},
	}

	suite.mockGitHubSv.EXPECT().
		GetRepositoryContent(gomock.Any(), gomock.Any(), "githubtools", "owner", "repo", "/", "main").
		Return(expectedContent, nil)

	suite.router.GET("/github/repos/:owner/:repo/contents/*path", func(c *gin.Context) {
		claims := &auth.AuthClaims{UUID: "test-uuid"}
		c.Set("auth_claims", claims)
		suite.handler.GetRepositoryContent(c)
	})

	req, _ := http.NewRequest(http.MethodGet, "/github/repos/owner/repo/contents/", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)
}

// GetGitHubAsset handler tests

// TestGetGitHubAsset_Success tests successful asset retrieval
func (suite *GitHubHandlerTestSuite) TestGetGitHubAsset_Success() {
	assetData := []byte("image data")
	contentType := "image/png"

	suite.mockGitHubSv.EXPECT().
		GetGitHubAsset(gomock.Any(), gomock.Any(), "githubtools", "https://github.com/owner/repo/raw/main/image.png").
		Return(assetData, contentType, nil)

	suite.router.GET("/github/asset", func(c *gin.Context) {
		claims := &auth.AuthClaims{UUID: "test-uuid"}
		c.Set("auth_claims", claims)
		suite.handler.GetGitHubAsset(c)
	})

	req, _ := http.NewRequest(http.MethodGet, "/github/asset?url=https://github.com/owner/repo/raw/main/image.png", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)
	assert.Equal(suite.T(), contentType, w.Header().Get("Content-Type"))
	assert.Equal(suite.T(), assetData, w.Body.Bytes())
}

// TestGetGitHubAsset_Unauthorized tests missing auth claims
func (suite *GitHubHandlerTestSuite) TestGetGitHubAsset_Unauthorized() {
	suite.router.GET("/github/asset", suite.handler.GetGitHubAsset)

	req, _ := http.NewRequest(http.MethodGet, "/github/asset?url=https://github.com/test.png", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusUnauthorized, w.Code)
}

// TestGetGitHubAsset_InvalidClaims tests invalid claims type
func (suite *GitHubHandlerTestSuite) TestGetGitHubAsset_InvalidClaims() {
	suite.router.GET("/github/asset", func(c *gin.Context) {
		c.Set("auth_claims", "invalid")
		suite.handler.GetGitHubAsset(c)
	})

	req, _ := http.NewRequest(http.MethodGet, "/github/asset?url=https://github.com/test.png", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusInternalServerError, w.Code)
}

// TestGetGitHubAsset_MissingURL tests missing url query parameter
func (suite *GitHubHandlerTestSuite) TestGetGitHubAsset_MissingURL() {
	suite.router.GET("/github/asset", func(c *gin.Context) {
		claims := &auth.AuthClaims{UUID: "test-uuid"}
		c.Set("auth_claims", claims)
		suite.handler.GetGitHubAsset(c)
	})

	req, _ := http.NewRequest(http.MethodGet, "/github/asset", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Contains(suite.T(), response["error"], "Asset URL is required")
}

// TestGetGitHubAsset_NotFound tests asset not found
func (suite *GitHubHandlerTestSuite) TestGetGitHubAsset_NotFound() {
	suite.mockGitHubSv.EXPECT().
		GetGitHubAsset(gomock.Any(), gomock.Any(), "githubtools", "https://github.com/notfound.png").
		Return(nil, "", apperrors.NewNotFoundError("asset"))

	suite.router.GET("/github/asset", func(c *gin.Context) {
		claims := &auth.AuthClaims{UUID: "test-uuid"}
		c.Set("auth_claims", claims)
		suite.handler.GetGitHubAsset(c)
	})

	req, _ := http.NewRequest(http.MethodGet, "/github/asset?url=https://github.com/notfound.png", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusNotFound, w.Code)
}

// TestGetGitHubAsset_RateLimitExceeded tests rate limit error
func (suite *GitHubHandlerTestSuite) TestGetGitHubAsset_RateLimitExceeded() {
	suite.mockGitHubSv.EXPECT().
		GetGitHubAsset(gomock.Any(), gomock.Any(), "githubtools", "https://github.com/test.png").
		Return(nil, "", apperrors.ErrGitHubAPIRateLimitExceeded)

	suite.router.GET("/github/asset", func(c *gin.Context) {
		claims := &auth.AuthClaims{UUID: "test-uuid"}
		c.Set("auth_claims", claims)
		suite.handler.GetGitHubAsset(c)
	})

	req, _ := http.NewRequest(http.MethodGet, "/github/asset?url=https://github.com/test.png", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusTooManyRequests, w.Code)
}

// TestGetGitHubAsset_ServiceError tests generic service error
func (suite *GitHubHandlerTestSuite) TestGetGitHubAsset_ServiceError() {
	suite.mockGitHubSv.EXPECT().
		GetGitHubAsset(gomock.Any(), gomock.Any(), "githubtools", "https://github.com/test.png").
		Return(nil, "", fmt.Errorf("service error"))

	suite.router.GET("/github/asset", func(c *gin.Context) {
		claims := &auth.AuthClaims{UUID: "test-uuid"}
		c.Set("auth_claims", claims)
		suite.handler.GetGitHubAsset(c)
	})

	req, _ := http.NewRequest(http.MethodGet, "/github/asset?url=https://github.com/test.png", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusBadGateway, w.Code)
}

// TestGetGitHubAsset_ContentType tests correct content-type header
func (suite *GitHubHandlerTestSuite) TestGetGitHubAsset_ContentType() {
	assetData := []byte("svg data")
	contentType := "image/svg+xml"

	suite.mockGitHubSv.EXPECT().
		GetGitHubAsset(gomock.Any(), gomock.Any(), "githubtools", "https://github.com/test.svg").
		Return(assetData, contentType, nil)

	suite.router.GET("/github/asset", func(c *gin.Context) {
		claims := &auth.AuthClaims{UUID: "test-uuid"}
		c.Set("auth_claims", claims)
		suite.handler.GetGitHubAsset(c)
	})

	req, _ := http.NewRequest(http.MethodGet, "/github/asset?url=https://github.com/test.svg", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)
	assert.Equal(suite.T(), contentType, w.Header().Get("Content-Type"))
}

// TestGetGitHubAsset_CacheHeaders tests cache-control headers
func (suite *GitHubHandlerTestSuite) TestGetGitHubAsset_CacheHeaders() {
	assetData := []byte("data")
	contentType := "image/png"

	suite.mockGitHubSv.EXPECT().
		GetGitHubAsset(gomock.Any(), gomock.Any(), "githubtools", "https://github.com/test.png").
		Return(assetData, contentType, nil)

	suite.router.GET("/github/asset", func(c *gin.Context) {
		claims := &auth.AuthClaims{UUID: "test-uuid"}
		c.Set("auth_claims", claims)
		suite.handler.GetGitHubAsset(c)
	})

	req, _ := http.NewRequest(http.MethodGet, "/github/asset?url=https://github.com/test.png", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)
	assert.Equal(suite.T(), "public, max-age=3600", w.Header().Get("Cache-Control"))
}

// UpdateRepositoryFile handler tests

// TestUpdateRepositoryFile_Success tests successful file update
func (suite *GitHubHandlerTestSuite) TestUpdateRepositoryFile_Success() {
	expectedResponse := map[string]interface{}{
		"commit": map[string]interface{}{
			"sha":     "abc123",
			"message": "Update file",
		},
	}

	suite.mockGitHubSv.EXPECT().
		UpdateRepositoryFile(gomock.Any(), gomock.Any(), "githubtools", "owner", "repo", "/file.txt", "Update file", "content", "sha123", "").
		Return(expectedResponse, nil)

	suite.router.PUT("/github/repos/:owner/:repo/contents/*path", func(c *gin.Context) {
		claims := &auth.AuthClaims{UUID: "test-uuid"}
		c.Set("auth_claims", claims)
		suite.handler.UpdateRepositoryFile(c)
	})

	body := map[string]interface{}{
		"message": "Update file",
		"content": "content",
		"sha":     "sha123",
	}
	payload, _ := json.Marshal(body)

	req, _ := http.NewRequest(http.MethodPut, "/github/repos/owner/repo/contents/file.txt", bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)
}

// TestUpdateRepositoryFile_WithBranch tests update on specific branch
func (suite *GitHubHandlerTestSuite) TestUpdateRepositoryFile_WithBranch() {
	expectedResponse := map[string]interface{}{"commit": map[string]interface{}{"sha": "abc123"}}

	suite.mockGitHubSv.EXPECT().
		UpdateRepositoryFile(gomock.Any(), gomock.Any(), "githubtools", "owner", "repo", "/file.txt", "Update", "content", "sha123", "develop").
		Return(expectedResponse, nil)

	suite.router.PUT("/github/repos/:owner/:repo/contents/*path", func(c *gin.Context) {
		claims := &auth.AuthClaims{UUID: "test-uuid"}
		c.Set("auth_claims", claims)
		suite.handler.UpdateRepositoryFile(c)
	})

	body := map[string]interface{}{
		"message": "Update",
		"content": "content",
		"sha":     "sha123",
		"branch":  "develop",
	}
	payload, _ := json.Marshal(body)

	req, _ := http.NewRequest(http.MethodPut, "/github/repos/owner/repo/contents/file.txt", bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)
}

// TestUpdateRepositoryFile_Unauthorized tests missing auth claims
func (suite *GitHubHandlerTestSuite) TestUpdateRepositoryFile_Unauthorized() {
	suite.router.PUT("/github/repos/:owner/:repo/contents/*path", suite.handler.UpdateRepositoryFile)

	body := map[string]interface{}{"message": "Update", "content": "content", "sha": "sha123"}
	payload, _ := json.Marshal(body)

	req, _ := http.NewRequest(http.MethodPut, "/github/repos/owner/repo/contents/file.txt", bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusUnauthorized, w.Code)
}

// TestUpdateRepositoryFile_InvalidClaims tests invalid claims type
func (suite *GitHubHandlerTestSuite) TestUpdateRepositoryFile_InvalidClaims() {
	suite.router.PUT("/github/repos/:owner/:repo/contents/*path", func(c *gin.Context) {
		c.Set("auth_claims", "invalid")
		suite.handler.UpdateRepositoryFile(c)
	})

	body := map[string]interface{}{"message": "Update", "content": "content", "sha": "sha123"}
	payload, _ := json.Marshal(body)

	req, _ := http.NewRequest(http.MethodPut, "/github/repos/owner/repo/contents/file.txt", bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusInternalServerError, w.Code)
}

// TestUpdateRepositoryFile_InvalidRequestBody tests malformed JSON
func (suite *GitHubHandlerTestSuite) TestUpdateRepositoryFile_InvalidRequestBody() {
	suite.router.PUT("/github/repos/:owner/:repo/contents/*path", func(c *gin.Context) {
		claims := &auth.AuthClaims{UUID: "test-uuid"}
		c.Set("auth_claims", claims)
		suite.handler.UpdateRepositoryFile(c)
	})

	req, _ := http.NewRequest(http.MethodPut, "/github/repos/owner/repo/contents/file.txt", bytes.NewBuffer([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)
}

// TestUpdateRepositoryFile_MissingMessage tests missing required message field
func (suite *GitHubHandlerTestSuite) TestUpdateRepositoryFile_MissingMessage() {
	suite.router.PUT("/github/repos/:owner/:repo/contents/*path", func(c *gin.Context) {
		claims := &auth.AuthClaims{UUID: "test-uuid"}
		c.Set("auth_claims", claims)
		suite.handler.UpdateRepositoryFile(c)
	})

	body := map[string]interface{}{"content": "content", "sha": "sha123"}
	payload, _ := json.Marshal(body)

	req, _ := http.NewRequest(http.MethodPut, "/github/repos/owner/repo/contents/file.txt", bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)
}

// TestUpdateRepositoryFile_MissingContent tests missing required content field
func (suite *GitHubHandlerTestSuite) TestUpdateRepositoryFile_MissingContent() {
	suite.router.PUT("/github/repos/:owner/:repo/contents/*path", func(c *gin.Context) {
		claims := &auth.AuthClaims{UUID: "test-uuid"}
		c.Set("auth_claims", claims)
		suite.handler.UpdateRepositoryFile(c)
	})

	body := map[string]interface{}{"message": "Update", "sha": "sha123"}
	payload, _ := json.Marshal(body)

	req, _ := http.NewRequest(http.MethodPut, "/github/repos/owner/repo/contents/file.txt", bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)
}

// TestUpdateRepositoryFile_MissingSHA tests missing required SHA field
func (suite *GitHubHandlerTestSuite) TestUpdateRepositoryFile_MissingSHA() {
	suite.router.PUT("/github/repos/:owner/:repo/contents/*path", func(c *gin.Context) {
		claims := &auth.AuthClaims{UUID: "test-uuid"}
		c.Set("auth_claims", claims)
		suite.handler.UpdateRepositoryFile(c)
	})

	body := map[string]interface{}{"message": "Update", "content": "content"}
	payload, _ := json.Marshal(body)

	req, _ := http.NewRequest(http.MethodPut, "/github/repos/owner/repo/contents/file.txt", bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)
}

// TestUpdateRepositoryFile_NotFound tests repository or path not found
func (suite *GitHubHandlerTestSuite) TestUpdateRepositoryFile_NotFound() {
	suite.mockGitHubSv.EXPECT().
		UpdateRepositoryFile(gomock.Any(), gomock.Any(), "githubtools", "owner", "repo", "/notfound.txt", "Update", "content", "sha123", "").
		Return(nil, apperrors.NewNotFoundError("file"))

	suite.router.PUT("/github/repos/:owner/:repo/contents/*path", func(c *gin.Context) {
		claims := &auth.AuthClaims{UUID: "test-uuid"}
		c.Set("auth_claims", claims)
		suite.handler.UpdateRepositoryFile(c)
	})

	body := map[string]interface{}{"message": "Update", "content": "content", "sha": "sha123"}
	payload, _ := json.Marshal(body)

	req, _ := http.NewRequest(http.MethodPut, "/github/repos/owner/repo/contents/notfound.txt", bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusNotFound, w.Code)
}

// TestUpdateRepositoryFile_RateLimitExceeded tests rate limit error
func (suite *GitHubHandlerTestSuite) TestUpdateRepositoryFile_RateLimitExceeded() {
	suite.mockGitHubSv.EXPECT().
		UpdateRepositoryFile(gomock.Any(), gomock.Any(), "githubtools", "owner", "repo", "/file.txt", "Update", "content", "sha123", "").
		Return(nil, apperrors.ErrGitHubAPIRateLimitExceeded)

	suite.router.PUT("/github/repos/:owner/:repo/contents/*path", func(c *gin.Context) {
		claims := &auth.AuthClaims{UUID: "test-uuid"}
		c.Set("auth_claims", claims)
		suite.handler.UpdateRepositoryFile(c)
	})

	body := map[string]interface{}{"message": "Update", "content": "content", "sha": "sha123"}
	payload, _ := json.Marshal(body)

	req, _ := http.NewRequest(http.MethodPut, "/github/repos/owner/repo/contents/file.txt", bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusTooManyRequests, w.Code)
}

// TestUpdateRepositoryFile_ServiceError tests generic service error
func (suite *GitHubHandlerTestSuite) TestUpdateRepositoryFile_ServiceError() {
	suite.mockGitHubSv.EXPECT().
		UpdateRepositoryFile(gomock.Any(), gomock.Any(), "githubtools", "owner", "repo", "/file.txt", "Update", "content", "sha123", "").
		Return(nil, fmt.Errorf("service error"))

	suite.router.PUT("/github/repos/:owner/:repo/contents/*path", func(c *gin.Context) {
		claims := &auth.AuthClaims{UUID: "test-uuid"}
		c.Set("auth_claims", claims)
		suite.handler.UpdateRepositoryFile(c)
	})

	body := map[string]interface{}{"message": "Update", "content": "content", "sha": "sha123"}
	payload, _ := json.Marshal(body)

	req, _ := http.NewRequest(http.MethodPut, "/github/repos/owner/repo/contents/file.txt", bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusBadGateway, w.Code)
}

// GetPRReviewComments handler tests

// TestGetPRReviewComments_Success tests successful review comments count retrieval
func (suite *GitHubHandlerTestSuite) TestGetPRReviewComments_Success() {
	suite.mockGitHubSv.EXPECT().
		GetUserPRReviewComments(gomock.Any(), gomock.Any(), "githubtools", "30d").
		Return(&service.PRReviewCommentsResponse{
			TotalComments: 42,
			Period:        "30d",
			From:          "2024-10-03T00:00:00Z",
			To:            "2024-11-02T23:59:59Z",
		}, nil)

	suite.router.GET("/github/pr-review-comments", func(c *gin.Context) {
		claims := &auth.AuthClaims{UUID: "test-uuid"}
		c.Set("auth_claims", claims)
		suite.handler.GetPRReviewComments(c)
	})

	req, _ := http.NewRequest(http.MethodGet, "/github/pr-review-comments", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response service.PRReviewCommentsResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), 42, response.TotalComments)
	assert.Equal(suite.T(), "30d", response.Period)
}

// TestGetPRReviewComments_WithPeriod tests custom period parameter
func (suite *GitHubHandlerTestSuite) TestGetPRReviewComments_WithPeriod() {
	suite.mockGitHubSv.EXPECT().
		GetUserPRReviewComments(gomock.Any(), gomock.Any(), "githubtools", "90d").
		Return(&service.PRReviewCommentsResponse{
			TotalComments: 100,
			Period:        "90d",
			From:          "2024-08-03T00:00:00Z",
			To:            "2024-11-02T23:59:59Z",
		}, nil)

	suite.router.GET("/github/pr-review-comments", func(c *gin.Context) {
		claims := &auth.AuthClaims{UUID: "test-uuid"}
		c.Set("auth_claims", claims)
		suite.handler.GetPRReviewComments(c)
	})

	req, _ := http.NewRequest(http.MethodGet, "/github/pr-review-comments?period=90d", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)
}

// TestGetPRReviewComments_Unauthorized tests missing auth claims
func (suite *GitHubHandlerTestSuite) TestGetPRReviewComments_Unauthorized() {
	suite.router.GET("/github/pr-review-comments", suite.handler.GetPRReviewComments)

	req, _ := http.NewRequest(http.MethodGet, "/github/pr-review-comments", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusUnauthorized, w.Code)
}

// TestGetPRReviewComments_InvalidClaims tests invalid claims type
func (suite *GitHubHandlerTestSuite) TestGetPRReviewComments_InvalidClaims() {
	suite.router.GET("/github/pr-review-comments", func(c *gin.Context) {
		c.Set("auth_claims", "invalid")
		suite.handler.GetPRReviewComments(c)
	})

	req, _ := http.NewRequest(http.MethodGet, "/github/pr-review-comments", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusInternalServerError, w.Code)
}

// TestGetPRReviewComments_InvalidPeriod tests invalid period format
func (suite *GitHubHandlerTestSuite) TestGetPRReviewComments_InvalidPeriod() {
	suite.mockGitHubSv.EXPECT().
		GetUserPRReviewComments(gomock.Any(), gomock.Any(), "githubtools", "invalid").
		Return(nil, apperrors.ErrInvalidPeriodFormat)

	suite.router.GET("/github/pr-review-comments", func(c *gin.Context) {
		claims := &auth.AuthClaims{UUID: "test-uuid"}
		c.Set("auth_claims", claims)
		suite.handler.GetPRReviewComments(c)
	})

	req, _ := http.NewRequest(http.MethodGet, "/github/pr-review-comments?period=invalid", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)
}

// TestGetPRReviewComments_RateLimitExceeded tests rate limit error
func (suite *GitHubHandlerTestSuite) TestGetPRReviewComments_RateLimitExceeded() {
	suite.mockGitHubSv.EXPECT().
		GetUserPRReviewComments(gomock.Any(), gomock.Any(), "githubtools", "30d").
		Return(nil, apperrors.ErrGitHubAPIRateLimitExceeded)

	suite.router.GET("/github/pr-review-comments", func(c *gin.Context) {
		claims := &auth.AuthClaims{UUID: "test-uuid"}
		c.Set("auth_claims", claims)
		suite.handler.GetPRReviewComments(c)
	})

	req, _ := http.NewRequest(http.MethodGet, "/github/pr-review-comments", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusTooManyRequests, w.Code)
}

// TestGetPRReviewComments_ServiceError tests generic service error
func (suite *GitHubHandlerTestSuite) TestGetPRReviewComments_ServiceError() {
	suite.mockGitHubSv.EXPECT().
		GetUserPRReviewComments(gomock.Any(), gomock.Any(), "githubtools", "30d").
		Return(nil, fmt.Errorf("service error"))

	suite.router.GET("/github/pr-review-comments", func(c *gin.Context) {
		claims := &auth.AuthClaims{UUID: "test-uuid"}
		c.Set("auth_claims", claims)
		suite.handler.GetPRReviewComments(c)
	})

	req, _ := http.NewRequest(http.MethodGet, "/github/pr-review-comments", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusBadGateway, w.Code)
}

// Run the test suite
func TestGitHubHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(GitHubHandlerTestSuite))
}
