package handlers_test

import (
	apperrors "developer-portal-backend/internal/errors"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"developer-portal-backend/internal/api/handlers"
	"developer-portal-backend/internal/auth"
	"developer-portal-backend/internal/mocks"
	"developer-portal-backend/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

// JiraHandlerTestSuite defines the test suite for JiraHandler
type JiraHandlerTestSuite struct {
	suite.Suite
	ctrl        *gomock.Controller
	mockService *mocks.MockJiraServiceInterface
	handler     *handlers.JiraHandler
	router      *gin.Engine
}

// SetupTest sets up the test suite
func (suite *JiraHandlerTestSuite) SetupTest() {
	gin.SetMode(gin.TestMode)
	suite.ctrl = gomock.NewController(suite.T())
	suite.mockService = mocks.NewMockJiraServiceInterface(suite.ctrl)

	// Create handler with mock service
	suite.handler = handlers.NewJiraHandler(suite.mockService)
	suite.router = gin.New()
	suite.setupRoutes()
}

// TearDownTest cleans up after each test
func (suite *JiraHandlerTestSuite) TearDownTest() {
	// no-op
}

// setupRoutes sets up the routes for testing
func (suite *JiraHandlerTestSuite) setupRoutes() {
	suite.router.GET("/jira/issues", suite.handler.GetIssues)
	suite.router.GET("/jira/issues/me", suite.handler.GetMyIssues)
	suite.router.GET("/jira/issues/me/count", suite.handler.GetMyIssuesCount)
}

// TestGetIssues tests the consolidated GetIssues handler
func (suite *JiraHandlerTestSuite) TestGetIssues() {
	suite.T().Run("Successful request with all parameters", func(t *testing.T) {
		expectedResponse := &service.JiraIssuesResponse{
			Total: 1,
			Issues: []service.JiraIssue{
				{
					ID:  "1",
					Key: "SAPBTPCFS-100",
					Fields: service.JiraIssueFields{
						Summary:   "Complete test issue",
						Status:    service.JiraStatus{ID: "1", Name: "Open"},
						IssueType: service.JiraIssueType{ID: "1", Name: "Bug"},
						Created:   "2023-01-01T00:00:00.000Z",
						Updated:   "2023-01-02T00:00:00.000Z",
					},
				},
			},
		}

		suite.mockService.EXPECT().
			GetIssues(gomock.Any()).
			DoAndReturn(func(filters service.JiraIssueFilters) (*service.JiraIssuesResponse, error) {
				assert.Equal(t, "SAPBTPCFS", filters.Project)
				assert.Equal(t, "Open", filters.Status)
				assert.Equal(t, "TestTeam", filters.Team)
				assert.Equal(t, "testuser", filters.Assignee)
				assert.Equal(t, "Bug", filters.Type)
				assert.Equal(t, "test", filters.Summary)
				assert.Equal(t, "SAPBTPCFS-100", filters.Key)
				assert.Equal(t, 2, filters.Page)
				assert.Equal(t, 25, filters.Limit)
				return expectedResponse, nil
			})

		req := httptest.NewRequest(http.MethodGet, "/jira/issues?project=SAPBTPCFS&status=Open&team=TestTeam&assignee=testuser&type=Bug&summary=test&key=SAPBTPCFS-100&page=2&limit=25", nil)
		w := httptest.NewRecorder()

		suite.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "SAPBTPCFS-100")
	})

	suite.T().Run("Successful request with project and status", func(t *testing.T) {
		expectedResponse := &service.JiraIssuesResponse{
			Total: 2,
			Issues: []service.JiraIssue{
				{
					ID:  "1",
					Key: "SAPBTPCFS-123",
					Fields: service.JiraIssueFields{
						Summary:   "Test issue 1",
						Status:    service.JiraStatus{ID: "1", Name: "Open"},
						IssueType: service.JiraIssueType{ID: "1", Name: "Story"},
						Created:   "2023-01-01T00:00:00.000Z",
						Updated:   "2023-01-02T00:00:00.000Z",
					},
				},
				{
					ID:  "2",
					Key: "SAPBTPCFS-124",
					Fields: service.JiraIssueFields{
						Summary:   "Test issue 2",
						Status:    service.JiraStatus{ID: "2", Name: "In Progress"},
						IssueType: service.JiraIssueType{ID: "1", Name: "Story"},
						Created:   "2023-01-01T00:00:00000Z",
						Updated:   "2023-01-02T00:00:00.000Z",
					},
				},
			},
		}

		suite.mockService.EXPECT().
			GetIssues(gomock.Any()).
			DoAndReturn(func(filters service.JiraIssueFilters) (*service.JiraIssuesResponse, error) {
				assert.Equal(t, "SAPBTPCFS", filters.Project)
				assert.Equal(t, "Open,In Progress", filters.Status)
				assert.Equal(t, "TestTeam", filters.Team)
				return expectedResponse, nil
			})

		req := httptest.NewRequest(http.MethodGet, "/jira/issues?project=SAPBTPCFS&status=Open%2CIn+Progress&team=TestTeam", nil)
		w := httptest.NewRecorder()

		suite.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "SAPBTPCFS-123")
		assert.Contains(t, w.Body.String(), "Test issue 1")
		assert.Contains(t, w.Body.String(), `"total":2`)
	})

	suite.T().Run("Successful request with minimal parameters", func(t *testing.T) {
		expectedResponse := &service.JiraIssuesResponse{
			Total: 1,
			Issues: []service.JiraIssue{
				{
					ID:  "1",
					Key: "SAPBTPCFS-999",
					Fields: service.JiraIssueFields{
						Summary:   "Minimal test issue",
						Status:    service.JiraStatus{ID: "1", Name: "Open"},
						IssueType: service.JiraIssueType{ID: "1", Name: "Bug"},
						Created:   "2023-01-01T00:00:00.000Z",
						Updated:   "2023-01-02T00:00:00.000Z",
					},
				},
			},
		}

		suite.mockService.EXPECT().
			GetIssues(gomock.Any()).
			DoAndReturn(func(filters service.JiraIssueFilters) (*service.JiraIssuesResponse, error) {
				assert.Equal(t, "", filters.Project)
				assert.Equal(t, "", filters.Status)
				assert.Equal(t, "", filters.Team)
				return expectedResponse, nil
			})

		req := httptest.NewRequest(http.MethodGet, "/jira/issues", nil)
		w := httptest.NewRecorder()

		suite.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "SAPBTPCFS-999")
		assert.Contains(t, w.Body.String(), `"total":1`)
	})

	suite.T().Run("Service error", func(t *testing.T) {
		suite.mockService.EXPECT().
			GetIssues(gomock.Any()).
			Return(nil, errors.New("jira connection failed"))

		req := httptest.NewRequest(http.MethodGet, "/jira/issues?project=SAPBTPCFS", nil)
		w := httptest.NewRecorder()

		suite.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadGateway, w.Code)
		assert.Contains(t, w.Body.String(), "jira search failed")
		assert.Contains(t, w.Body.String(), "jira connection failed")
	})

	suite.T().Run("Invalid page parameter - not a number", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/jira/issues?page=invalid", nil)
		w := httptest.NewRecorder()

		suite.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "invalid page parameter")
	})

	suite.T().Run("Invalid page parameter - zero", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/jira/issues?page=0", nil)
		w := httptest.NewRecorder()

		suite.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "must be greater than 0")
	})

	suite.T().Run("Invalid limit parameter - not a number", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/jira/issues?limit=invalid", nil)
		w := httptest.NewRecorder()

		suite.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "invalid limit parameter")
	})

	suite.T().Run("Invalid limit parameter - zero", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/jira/issues?limit=0", nil)
		w := httptest.NewRecorder()

		suite.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "must be greater than 0")
	})

	suite.T().Run("Invalid limit parameter - exceeds maximum", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/jira/issues?limit=101", nil)
		w := httptest.NewRecorder()

		suite.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "maximum allowed is 100")
	})

	suite.T().Run("Valid pagination parameters", func(t *testing.T) {
		expectedResponse := &service.JiraIssuesResponse{
			Total:  0,
			Issues: []service.JiraIssue{},
		}

		suite.mockService.EXPECT().
			GetIssues(gomock.Any()).
			DoAndReturn(func(filters service.JiraIssueFilters) (*service.JiraIssuesResponse, error) {
				assert.Equal(t, 3, filters.Page)
				assert.Equal(t, 100, filters.Limit)
				return expectedResponse, nil
			})

		req := httptest.NewRequest(http.MethodGet, "/jira/issues?page=3&limit=100", nil)
		w := httptest.NewRecorder()

		suite.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

// TestGetMyIssues tests the consolidated GetMyIssues handler
func (suite *JiraHandlerTestSuite) TestGetMyIssues() {
	suite.T().Run("Missing authentication", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/jira/issues/me", nil)
		w := httptest.NewRecorder()

		suite.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "Authentication required")
	})

	suite.T().Run("Invalid authentication claims", func(t *testing.T) {
		router := gin.New()
		router.GET("/jira/issues/me", func(c *gin.Context) {
			c.Set("auth_claims", "invalid_claims")
			suite.handler.GetMyIssues(c)
		})

		req := httptest.NewRequest(http.MethodGet, "/jira/issues/me", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "Invalid authentication claims")
	})

	suite.T().Run("Missing username in claims", func(t *testing.T) {
		router := gin.New()
		router.GET("/jira/issues/me", func(c *gin.Context) {
			claims := &auth.AuthClaims{} // Username empty
			c.Set("auth_claims", claims)
			suite.handler.GetMyIssues(c)
		})

		req := httptest.NewRequest(http.MethodGet, "/jira/issues/me", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "Username not available in authentication claims")
	})

	suite.T().Run("Successful request with full issues", func(t *testing.T) {
		expectedResponse := &service.JiraIssuesResponse{
			Total: 2,
			Issues: []service.JiraIssue{
				{
					ID:  "1",
					Key: "SAPBTPCFS-999",
					Fields: service.JiraIssueFields{
						Summary:   "My issue 1",
						Status:    service.JiraStatus{ID: "1", Name: "Open"},
						IssueType: service.JiraIssueType{ID: "1", Name: "Story"},
						Assignee: &service.JiraUser{
							AccountID:   "12345",
							DisplayName: "Test User",
						},
						Created: "2023-01-01T00:00:00.000Z",
						Updated: "2023-01-02T00:00:00.000Z",
					},
				},
			},
		}

		suite.mockService.EXPECT().
			GetIssues(gomock.Any()).
			DoAndReturn(func(filters service.JiraIssueFilters) (*service.JiraIssuesResponse, error) {
				assert.Equal(t, "testuser", filters.User)
				assert.Equal(t, "Open", filters.Status)
				assert.Equal(t, "SAPBTPCFS", filters.Project)
				return expectedResponse, nil
			})

		router := gin.New()
		router.GET("/jira/issues/me", func(c *gin.Context) {
			claims := &auth.AuthClaims{Username: "testuser"}
			c.Set("auth_claims", claims)
			suite.handler.GetMyIssues(c)
		})

		req := httptest.NewRequest(http.MethodGet, "/jira/issues/me?status=Open&project=SAPBTPCFS", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "SAPBTPCFS-999")
		assert.Contains(t, w.Body.String(), "My issue 1")
		assert.Contains(t, w.Body.String(), `"total":2`)
	})

	suite.T().Run("Service error", func(t *testing.T) {
		suite.mockService.EXPECT().
			GetIssues(gomock.Any()).
			Return(nil, errors.New("jira service unavailable"))

		router := gin.New()
		router.GET("/jira/issues/me", func(c *gin.Context) {
			claims := &auth.AuthClaims{Username: "testuser"}
			c.Set("auth_claims", claims)
			suite.handler.GetMyIssues(c)
		})

		req := httptest.NewRequest(http.MethodGet, "/jira/issues/me", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadGateway, w.Code)
		assert.Contains(t, w.Body.String(), "jira search failed")
		assert.Contains(t, w.Body.String(), "jira service unavailable")
	})

	suite.T().Run("Invalid pagination parameters", func(t *testing.T) {
		router := gin.New()
		router.GET("/jira/issues/me", func(c *gin.Context) {
			claims := &auth.AuthClaims{
				Username: "testuser",
			}
			c.Set("auth_claims", claims)
			suite.handler.GetMyIssues(c)
		})

		req := httptest.NewRequest(http.MethodGet, "/jira/issues/me?page=invalid", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "invalid page parameter")
	})
}

// TestGetMyIssuesCount tests the consolidated GetMyIssuesCount handler
func (suite *JiraHandlerTestSuite) TestGetMyIssuesCount() {
	suite.T().Run("Missing authentication", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/jira/issues/me/count?status=Resolved", nil)
		w := httptest.NewRecorder()

		suite.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "Authentication required")
	})

	suite.T().Run("Invalid authentication claims", func(t *testing.T) {
		router := gin.New()
		router.GET("/jira/issues/me/count", func(c *gin.Context) {
			c.Set("auth_claims", "invalid_claims")
			suite.handler.GetMyIssuesCount(c)
		})

		req := httptest.NewRequest(http.MethodGet, "/jira/issues/me/count?status=Resolved", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "Invalid authentication claims")
	})

	suite.T().Run("Missing username in claims", func(t *testing.T) {
		router := gin.New()
		router.GET("/jira/issues/me/count", func(c *gin.Context) {
			claims := &auth.AuthClaims{
				UUID: "test-uuid",
				// Username is empty
			}
			c.Set("auth_claims", claims)
			suite.handler.GetMyIssuesCount(c)
		})

		req := httptest.NewRequest(http.MethodGet, "/jira/issues/me/count?status=Resolved", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "Username not available in authentication claims")
	})

	suite.T().Run("Missing status parameter", func(t *testing.T) {
		router := gin.New()
		router.GET("/jira/issues/me/count", func(c *gin.Context) {
			claims := &auth.AuthClaims{Username: "testuser"}
			c.Set("auth_claims", claims)
			suite.handler.GetMyIssuesCount(c)
		})

		req := httptest.NewRequest(http.MethodGet, "/jira/issues/me/count", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), apperrors.NewMissingQueryParam("status").Error())
	})

	suite.T().Run("Invalid date format", func(t *testing.T) {
		router := gin.New()
		router.GET("/jira/issues/me/count", func(c *gin.Context) {
			claims := &auth.AuthClaims{Username: "testuser"}
			c.Set("auth_claims", claims)
			suite.handler.GetMyIssuesCount(c)
		})

		req := httptest.NewRequest(http.MethodGet, "/jira/issues/me/count?status=Resolved&date=invalid-date", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "invalid date format: must be yyyy-MM-dd")
	})

	suite.T().Run("Successful request with resolved status and default date", func(t *testing.T) {
		suite.mockService.EXPECT().
			GetIssuesCount(gomock.Any()).
			DoAndReturn(func(filters service.JiraIssueFilters) (int, error) {
				assert.Equal(t, "testuser", filters.User)
				assert.Equal(t, "Resolved", filters.Status)
				assert.NotEmpty(t, filters.Date) // Should have default date
				return 7, nil
			})

		router := gin.New()
		router.GET("/jira/issues/me/count", func(c *gin.Context) {
			claims := &auth.AuthClaims{Username: "testuser"}
			c.Set("auth_claims", claims)
			suite.handler.GetMyIssuesCount(c)
		})

		req := httptest.NewRequest(http.MethodGet, "/jira/issues/me/count?status=Resolved", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), `"count":7`)
	})

	suite.T().Run("Successful request with custom date", func(t *testing.T) {
		suite.mockService.EXPECT().
			GetIssuesCount(gomock.Any()).
			DoAndReturn(func(filters service.JiraIssueFilters) (int, error) {
				assert.Equal(t, "testuser", filters.User)
				assert.Equal(t, "Resolved", filters.Status)
				assert.Equal(t, "2023-06-01", filters.Date)
				return 4, nil
			})

		router := gin.New()
		router.GET("/jira/issues/me/count", func(c *gin.Context) {
			claims := &auth.AuthClaims{Username: "testuser"}
			c.Set("auth_claims", claims)
			suite.handler.GetMyIssuesCount(c)
		})

		req := httptest.NewRequest(http.MethodGet, "/jira/issues/me/count?status=Resolved&date=2023-06-01", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), `"count":4`)
	})

	suite.T().Run("Service error", func(t *testing.T) {
		suite.mockService.EXPECT().
			GetIssuesCount(gomock.Any()).
			Return(0, errors.New("jira configuration missing"))

		router := gin.New()
		router.GET("/jira/issues/me/count", func(c *gin.Context) {
			claims := &auth.AuthClaims{Username: "testuser"}
			c.Set("auth_claims", claims)
			suite.handler.GetMyIssuesCount(c)
		})

		req := httptest.NewRequest(http.MethodGet, "/jira/issues/me/count?status=Resolved", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadGateway, w.Code)
		assert.Contains(t, w.Body.String(), "jira search failed")
		assert.Contains(t, w.Body.String(), "jira configuration missing")
	})

	suite.T().Run("Successful request with non-resolved status and no date", func(t *testing.T) {
		suite.mockService.EXPECT().
			GetIssuesCount(gomock.Any()).
			DoAndReturn(func(filters service.JiraIssueFilters) (int, error) {
				assert.Equal(t, "testuser", filters.User)
				assert.Equal(t, "Open", filters.Status)
				assert.Equal(t, "", filters.Date) // No date for non-resolved status
				return 5, nil
			})

		router := gin.New()
		router.GET("/jira/issues/me/count", func(c *gin.Context) {
			claims := &auth.AuthClaims{
				UUID:     "test-uuid",
				Username: "testuser",
			}
			c.Set("auth_claims", claims)
			suite.handler.GetMyIssuesCount(c)
		})

		req := httptest.NewRequest(http.MethodGet, "/jira/issues/me/count?status=Open", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), `"count":5`)
	})

	suite.T().Run("Successful request with project filter", func(t *testing.T) {
		suite.mockService.EXPECT().
			GetIssuesCount(gomock.Any()).
			DoAndReturn(func(filters service.JiraIssueFilters) (int, error) {
				assert.Equal(t, "testuser", filters.User)
				assert.Equal(t, "Resolved", filters.Status)
				assert.Equal(t, "SAPBTPCFS", filters.Project)
				assert.NotEmpty(t, filters.Date)
				return 3, nil
			})

		router := gin.New()
		router.GET("/jira/issues/me/count", func(c *gin.Context) {
			claims := &auth.AuthClaims{
				UUID:     "test-uuid",
				Username: "testuser",
			}
			c.Set("auth_claims", claims)
			suite.handler.GetMyIssuesCount(c)
		})

		req := httptest.NewRequest(http.MethodGet, "/jira/issues/me/count?status=Resolved&project=SAPBTPCFS", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), `"count":3`)
	})
}

// TestJiraHandlerTestSuite runs the test suite
func TestJiraHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(JiraHandlerTestSuite))
}
