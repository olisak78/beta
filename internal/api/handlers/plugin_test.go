package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"developer-portal-backend/internal/auth"
	"developer-portal-backend/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockPluginService is a mock implementation of PluginServiceInterface
type MockPluginService struct {
	mock.Mock
}

func (m *MockPluginService) GetAllPlugins(limit, offset int) (*service.PluginListResponse, error) {
	args := m.Called(limit, offset)
	return args.Get(0).(*service.PluginListResponse), args.Error(1)
}

func (m *MockPluginService) GetAllPluginsWithViewer(limit, offset int, viewerName string) (*service.PluginListResponse, error) {
	args := m.Called(limit, offset, viewerName)
	return args.Get(0).(*service.PluginListResponse), args.Error(1)
}

func (m *MockPluginService) GetPluginByID(id uuid.UUID) (*service.PluginResponse, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.PluginResponse), args.Error(1)
}

func (m *MockPluginService) CreatePlugin(req *service.CreatePluginRequest) (*service.PluginResponse, error) {
	args := m.Called(req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.PluginResponse), args.Error(1)
}

func (m *MockPluginService) UpdatePlugin(id uuid.UUID, req *service.UpdatePluginRequest) (*service.PluginResponse, error) {
	args := m.Called(id, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.PluginResponse), args.Error(1)
}

func (m *MockPluginService) DeletePlugin(id uuid.UUID) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockPluginService) GetPluginUIContent(ctx context.Context, pluginID uuid.UUID, githubService service.GitHubServiceInterface, userUUID, provider string) (*service.PluginUIResponse, error) {
	args := m.Called(ctx, pluginID, githubService, userUUID, provider)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.PluginUIResponse), args.Error(1)
}

// MockGitHubService is a mock implementation of GitHubServiceInterface
type MockGitHubService struct {
	mock.Mock
}

func (m *MockGitHubService) GetUserOpenPullRequests(ctx context.Context, uuid, provider, state, sort, direction string, perPage, page int) (*service.PullRequestsResponse, error) {
	args := m.Called(ctx, uuid, provider, state, sort, direction, perPage, page)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.PullRequestsResponse), args.Error(1)
}

func (m *MockGitHubService) GetUserTotalContributions(ctx context.Context, uuid, provider, period string) (*service.TotalContributionsResponse, error) {
	args := m.Called(ctx, uuid, provider, period)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.TotalContributionsResponse), args.Error(1)
}

func (m *MockGitHubService) GetContributionsHeatmap(ctx context.Context, uuid, provider, period string) (*service.ContributionsHeatmapResponse, error) {
	args := m.Called(ctx, uuid, provider, period)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.ContributionsHeatmapResponse), args.Error(1)
}

func (m *MockGitHubService) GetAveragePRMergeTime(ctx context.Context, uuid, provider, period string) (*service.AveragePRMergeTimeResponse, error) {
	args := m.Called(ctx, uuid, provider, period)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.AveragePRMergeTimeResponse), args.Error(1)
}

func (m *MockGitHubService) GetUserPRReviewComments(ctx context.Context, uuid, provider, period string) (*service.PRReviewCommentsResponse, error) {
	args := m.Called(ctx, uuid, provider, period)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.PRReviewCommentsResponse), args.Error(1)
}

func (m *MockGitHubService) GetRepositoryContent(ctx context.Context, userUUID, provider, owner, repo, path, ref string) (interface{}, error) {
	args := m.Called(ctx, userUUID, provider, owner, repo, path, ref)
	return args.Get(0), args.Error(1)
}

func (m *MockGitHubService) UpdateRepositoryFile(ctx context.Context, uuid, provider, owner, repo, path, message, content, sha, branch string) (interface{}, error) {
	args := m.Called(ctx, uuid, provider, owner, repo, path, message, content, sha, branch)
	return args.Get(0), args.Error(1)
}

func (m *MockGitHubService) ClosePullRequest(ctx context.Context, uuid, provider, owner, repo string, prNumber int, deleteBranch bool) (*service.PullRequest, error) {
	args := m.Called(ctx, uuid, provider, owner, repo, prNumber, deleteBranch)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.PullRequest), args.Error(1)
}

func (m *MockGitHubService) GetGitHubAsset(ctx context.Context, uuid, provider, assetURL string) ([]byte, string, error) {
	args := m.Called(ctx, uuid, provider, assetURL)
	return args.Get(0).([]byte), args.Get(1).(string), args.Error(2)
}

func TestPluginHandler_GetAllPlugins(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		queryParams    string
		mockResponse   *service.PluginListResponse
		mockError      error
		expectedStatus int
		expectedBody   string
	}{
		{
			name:        "successful request with default pagination",
			queryParams: "",
			mockResponse: &service.PluginListResponse{
				Plugins: []service.PluginResponse{
					{
						ID:                 uuid.New(),
						Name:               "sample-plugin",
						Title:              "Sample Plugin",
						Description:        "A sample plugin",
						Icon:               "Puzzle",
						ReactComponentPath: "/plugins/sample/Sample.jsx",
						BackendServerURL:   "http://localhost:3001",
						Owner:              "Developer Portal Team",
					},
				},
				Total:  1,
				Limit:  20,
				Offset: 0,
			},
			mockError:      nil,
			expectedStatus: http.StatusOK,
		},
		{
			name:        "successful request with custom pagination",
			queryParams: "?limit=10&offset=5",
			mockResponse: &service.PluginListResponse{
				Plugins: []service.PluginResponse{},
				Total:   0,
				Limit:   10,
				Offset:  5,
			},
			mockError:      nil,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid limit parameter",
			queryParams:    "?limit=-1",
			mockResponse:   nil,
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"error":"limit must be non-negative"}`,
		},
		{
			name:           "invalid offset parameter",
			queryParams:    "?offset=-1",
			mockResponse:   nil,
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"error":"offset must be non-negative"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockPluginService)
			handler := NewPluginHandler(mockService)

			if tt.mockResponse != nil {
				mockService.On("GetAllPlugins", mock.AnythingOfType("int"), mock.AnythingOfType("int")).Return(tt.mockResponse, tt.mockError)
			}

			router := gin.New()
			router.GET("/plugins", handler.GetAllPlugins)

			req, _ := http.NewRequest("GET", "/plugins"+tt.queryParams, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedBody != "" {
				assert.JSONEq(t, tt.expectedBody, w.Body.String())
			} else if tt.mockResponse != nil {
				var response service.PluginListResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, tt.mockResponse.Total, response.Total)
				assert.Equal(t, tt.mockResponse.Limit, response.Limit)
				assert.Equal(t, tt.mockResponse.Offset, response.Offset)
			}

			mockService.AssertExpectations(t)
		})
	}
}

func TestPluginHandler_ProxyPluginBackend(t *testing.T) {
	gin.SetMode(gin.TestMode)

	validID := uuid.New()
	invalidID := "invalid-uuid"

	tests := []struct {
		name               string
		pluginID           string
		targetPath         string
		mockPlugin         *service.PluginResponse
		mockError          error
		expectedStatus     int
		expectedBody       string
		setupMockServer    bool
		mockServerResponse string
		mockServerStatus   int
	}{
		{
			name:       "successful proxy request",
			pluginID:   validID.String(),
			targetPath: "/api/health",
			mockPlugin: &service.PluginResponse{
				ID:               validID,
				Name:             "test-plugin",
				Title:            "Test Plugin",
				BackendServerURL: "http://localhost:8080",
			},
			mockError:          nil,
			expectedStatus:     http.StatusOK,
			setupMockServer:    true,
			mockServerResponse: `{"status": "healthy"}`,
			mockServerStatus:   http.StatusOK,
		},
		{
			name:           "invalid plugin ID format",
			pluginID:       invalidID,
			targetPath:     "/api/health",
			mockPlugin:     nil,
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"error":"invalid plugin ID format"}`,
		},
		{
			name:           "missing path parameter",
			pluginID:       validID.String(),
			targetPath:     "",
			mockPlugin:     nil,
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"error":"path parameter is required"}`,
		},
		{
			name:           "plugin not found",
			pluginID:       validID.String(),
			targetPath:     "/api/health",
			mockPlugin:     nil,
			mockError:      errors.New("record not found"),
			expectedStatus: http.StatusNotFound,
			expectedBody:   `{"error":"Plugin not found"}`,
		},
		{
			name:       "plugin backend returns error status",
			pluginID:   validID.String(),
			targetPath: "/api/error",
			mockPlugin: &service.PluginResponse{
				ID:               validID,
				Name:             "test-plugin",
				Title:            "Test Plugin",
				BackendServerURL: "http://localhost:8080",
			},
			mockError:          nil,
			expectedStatus:     http.StatusOK, // Always returns 200
			setupMockServer:    true,
			mockServerResponse: `{"error": "Internal server error"}`,
			mockServerStatus:   http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockPluginService)
			handler := NewPluginHandler(mockService)

			// Setup mock server if needed
			var mockServer *httptest.Server
			if tt.setupMockServer {
				mockServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(tt.mockServerStatus)
					w.Write([]byte(tt.mockServerResponse))
				}))
				defer mockServer.Close()

				// Update mock plugin to use mock server URL
				if tt.mockPlugin != nil {
					tt.mockPlugin.BackendServerURL = mockServer.URL
				}
			}

			// Setup mock expectations
			if tt.mockPlugin != nil || (tt.mockError != nil && tt.pluginID != invalidID && tt.targetPath != "") {
				parsedID, _ := uuid.Parse(tt.pluginID)
				mockService.On("GetPluginByID", parsedID).Return(tt.mockPlugin, tt.mockError)
			}

			router := gin.New()
			router.GET("/plugins/:id/proxy", handler.ProxyPluginBackend)

			// Build request URL
			requestURL := "/plugins/" + tt.pluginID + "/proxy"
			if tt.targetPath != "" {
				requestURL += "?path=" + tt.targetPath
			}

			req, _ := http.NewRequest("GET", requestURL, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedBody != "" {
				assert.JSONEq(t, tt.expectedBody, w.Body.String())
			} else if tt.setupMockServer {
				// Verify response structure
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)

				// Check required fields
				assert.Contains(t, response, "data")
				assert.Contains(t, response, "responseTime")
				assert.Contains(t, response, "statusCode")
				assert.Contains(t, response, "pluginSuccess")

				// Check status code and success flag
				assert.Equal(t, float64(tt.mockServerStatus), response["statusCode"])
				expectedSuccess := tt.mockServerStatus >= 200 && tt.mockServerStatus < 300
				assert.Equal(t, expectedSuccess, response["pluginSuccess"])

				// Check response time is present and reasonable
				responseTime, ok := response["responseTime"].(float64)
				assert.True(t, ok)
				assert.GreaterOrEqual(t, responseTime, float64(0))

				// Check data content for successful responses
				if expectedSuccess {
					var expectedData map[string]interface{}
					json.Unmarshal([]byte(tt.mockServerResponse), &expectedData)
					assert.Equal(t, expectedData, response["data"])
				}

				// Check error field for failed responses
				if !expectedSuccess {
					assert.Contains(t, response, "error")
				}
			}

			mockService.AssertExpectations(t)
		})
	}
}

func TestPluginHandler_ProxyPluginBackend_Caching(t *testing.T) {
	gin.SetMode(gin.TestMode)

	validID := uuid.New()
	mockPlugin := &service.PluginResponse{
		ID:               validID,
		Name:             "test-plugin",
		Title:            "Test Plugin",
		BackendServerURL: "http://localhost:8080",
	}

	// Setup mock server
	callCount := 0
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "healthy", "call": ` + string(rune(callCount+'0')) + `}`))
	}))
	defer mockServer.Close()

	mockPlugin.BackendServerURL = mockServer.URL

	mockService := new(MockPluginService)
	handler := NewPluginHandler(mockService)

	// Setup mock expectations - should be called only once (second request uses cache)
	mockService.On("GetPluginByID", validID).Return(mockPlugin, nil).Once()

	router := gin.New()
	router.GET("/plugins/:id/proxy", handler.ProxyPluginBackend)

	// First request - should hit the backend
	req1, _ := http.NewRequest("GET", "/plugins/"+validID.String()+"/proxy?path=/api/health", nil)
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)

	assert.Equal(t, http.StatusOK, w1.Code)
	assert.Equal(t, 1, callCount) // Backend should be called once

	var response1 map[string]interface{}
	err := json.Unmarshal(w1.Body.Bytes(), &response1)
	assert.NoError(t, err)
	assert.Equal(t, true, response1["pluginSuccess"])

	// Second request - should use cache
	req2, _ := http.NewRequest("GET", "/plugins/"+validID.String()+"/proxy?path=/api/health", nil)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	assert.Equal(t, http.StatusOK, w2.Code)
	assert.Equal(t, 1, callCount) // Backend should still be called only once (cached)

	var response2 map[string]interface{}
	err = json.Unmarshal(w2.Body.Bytes(), &response2)
	assert.NoError(t, err)
	assert.Equal(t, true, response2["pluginSuccess"])

	// Responses should be identical (from cache)
	assert.Equal(t, response1["data"], response2["data"])

	mockService.AssertExpectations(t)
}

func TestPluginHandler_GetAllPlugins_SubscribedLogic(t *testing.T) {
	gin.SetMode(gin.TestMode)

	pluginID1 := uuid.New()
	pluginID2 := uuid.New()

	tests := []struct {
		name                string
		queryParams         string
		setupAuth           bool
		mockAllPlugins      *service.PluginListResponse
		mockError           error
		expectedStatus      int
		expectedPluginCount int
		expectedSubscribed  []bool
		expectedBody        string
	}{
		{
			name:        "subscribed=true returns only subscribed plugins",
			queryParams: "?subscribed=true",
			setupAuth:   true,
			mockAllPlugins: &service.PluginListResponse{
				Plugins: []service.PluginResponse{
					{
						ID:          pluginID1,
						Name:        "subscribed-plugin",
						Title:       "Subscribed Plugin",
						Description: "A subscribed plugin",
						Icon:        "Check",
						Subscribed:  true,
					},
					{
						ID:          pluginID2,
						Name:        "unsubscribed-plugin",
						Title:       "Unsubscribed Plugin",
						Description: "An unsubscribed plugin",
						Icon:        "X",
						Subscribed:  false,
					},
				},
				Total:  2,
				Limit:  20,
				Offset: 0,
			},
			mockError:           nil,
			expectedStatus:      http.StatusOK,
			expectedPluginCount: 1, // Only subscribed plugins
			expectedSubscribed:  []bool{true},
		},
		{
			name:        "subscribed=false returns all plugins with subscription status",
			queryParams: "?subscribed=false",
			setupAuth:   true,
			mockAllPlugins: &service.PluginListResponse{
				Plugins: []service.PluginResponse{
					{
						ID:          pluginID1,
						Name:        "subscribed-plugin",
						Title:       "Subscribed Plugin",
						Description: "A subscribed plugin",
						Icon:        "Check",
						Subscribed:  true,
					},
					{
						ID:          pluginID2,
						Name:        "unsubscribed-plugin",
						Title:       "Unsubscribed Plugin",
						Description: "An unsubscribed plugin",
						Icon:        "X",
						Subscribed:  false,
					},
				},
				Total:  2,
				Limit:  20,
				Offset: 0,
			},
			mockError:           nil,
			expectedStatus:      http.StatusOK,
			expectedPluginCount: 2, // All plugins
			expectedSubscribed:  []bool{true, false},
		},
		{
			name:        "default behavior returns all plugins with subscription status for authenticated users",
			queryParams: "",
			setupAuth:   true,
			mockAllPlugins: &service.PluginListResponse{
				Plugins: []service.PluginResponse{
					{
						ID:          pluginID1,
						Name:        "plugin1",
						Title:       "Plugin 1",
						Description: "First plugin",
						Icon:        "One",
						Subscribed:  true,
					},
					{
						ID:          pluginID2,
						Name:        "plugin2",
						Title:       "Plugin 2",
						Description: "Second plugin",
						Icon:        "Two",
						Subscribed:  false,
					},
				},
				Total:  2,
				Limit:  20,
				Offset: 0,
			},
			mockError:           nil,
			expectedStatus:      http.StatusOK,
			expectedPluginCount: 2,
			expectedSubscribed:  []bool{true, false},
		},
		{
			name:           "subscribed=true without authentication returns 401",
			queryParams:    "?subscribed=true",
			setupAuth:      false,
			mockAllPlugins: nil,
			mockError:      nil,
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   `{"error":"Authentication required"}`,
		},
		{
			name:        "subscribed=true with no subscribed plugins returns empty list",
			queryParams: "?subscribed=true",
			setupAuth:   true,
			mockAllPlugins: &service.PluginListResponse{
				Plugins: []service.PluginResponse{
					{
						ID:          pluginID1,
						Name:        "unsubscribed-plugin1",
						Title:       "Unsubscribed Plugin 1",
						Description: "Not subscribed",
						Icon:        "X",
						Subscribed:  false,
					},
					{
						ID:          pluginID2,
						Name:        "unsubscribed-plugin2",
						Title:       "Unsubscribed Plugin 2",
						Description: "Also not subscribed",
						Icon:        "X",
						Subscribed:  false,
					},
				},
				Total:  2,
				Limit:  20,
				Offset: 0,
			},
			mockError:           nil,
			expectedStatus:      http.StatusOK,
			expectedPluginCount: 0, // No subscribed plugins
			expectedSubscribed:  []bool{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockPluginService)
			handler := NewPluginHandler(mockService)

			// Setup mock expectations
			if tt.mockAllPlugins != nil {
				if tt.setupAuth {
					mockService.On("GetAllPluginsWithViewer", mock.AnythingOfType("int"), mock.AnythingOfType("int"), mock.AnythingOfType("string")).Return(tt.mockAllPlugins, tt.mockError)
				} else {
					mockService.On("GetAllPlugins", mock.AnythingOfType("int"), mock.AnythingOfType("int")).Return(tt.mockAllPlugins, tt.mockError)
				}
			}

			router := gin.New()

			// Setup authentication middleware if needed
			if tt.setupAuth {
				router.Use(func(c *gin.Context) {
					// Create proper auth.AuthClaims structure
					mockClaims := &auth.AuthClaims{
						Username: "testuser",
						Email:    "testuser@example.com",
						UUID:     "test-uuid",
					}
					c.Set("auth_claims", mockClaims)
					c.Next()
				})
			}

			router.GET("/plugins", handler.GetAllPlugins)

			req, _ := http.NewRequest("GET", "/plugins"+tt.queryParams, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedBody != "" {
				assert.JSONEq(t, tt.expectedBody, w.Body.String())
			} else if tt.mockAllPlugins != nil {
				var response service.PluginListResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedPluginCount, len(response.Plugins))

				// Check subscription status for each plugin
				for i, plugin := range response.Plugins {
					if i < len(tt.expectedSubscribed) {
						assert.Equal(t, tt.expectedSubscribed[i], plugin.Subscribed, "Plugin %d subscription status mismatch", i)
					}
				}

				// For subscribed=true, verify total count reflects filtered results
				if strings.Contains(tt.queryParams, "subscribed=true") {
					assert.Equal(t, int64(tt.expectedPluginCount), response.Total)
				}
			}

			mockService.AssertExpectations(t)
		})
	}
}

func TestPluginHandler_GetAllPlugins_SubscribedServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockPluginService)
	handler := NewPluginHandler(mockService)

	// Mock service error
	mockService.On("GetAllPluginsWithViewer", mock.AnythingOfType("int"), mock.AnythingOfType("int"), mock.AnythingOfType("string")).Return((*service.PluginListResponse)(nil), errors.New("service error"))

	router := gin.New()
	router.Use(func(c *gin.Context) {
		// Mock authentication claims using proper auth.AuthClaims
		claims := &auth.AuthClaims{
			Username: "testuser",
			Email:    "testuser@example.com",
			UUID:     "test-uuid",
		}
		c.Set("auth_claims", claims)
		c.Next()
	})
	router.GET("/plugins", handler.GetAllPlugins)

	req, _ := http.NewRequest("GET", "/plugins?subscribed=true", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "Failed to retrieve plugins")

	mockService.AssertExpectations(t)
}

func TestPluginHandler_CreatePlugin(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		requestBody    string
		mockResponse   *service.PluginResponse
		mockError      error
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "successful creation",
			requestBody: `{
				"name": "new-plugin",
				"title": "New Plugin",
				"description": "A new plugin for testing",
				"icon": "TestIcon",
				"react_component_path": "/plugins/new/NewPlugin.jsx",
				"backend_server_url": "http://localhost:3002",
				"owner": "Test Team"
			}`,
			mockResponse: &service.PluginResponse{
				ID:                 uuid.New(),
				Name:               "new-plugin",
				Title:              "New Plugin",
				Description:        "A new plugin for testing",
				Icon:               "TestIcon",
				ReactComponentPath: "/plugins/new/NewPlugin.jsx",
				BackendServerURL:   "http://localhost:3002",
				Owner:              "Test Team",
			},
			mockError:      nil,
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "invalid JSON",
			requestBody:    `{"name": "invalid-json"`,
			mockResponse:   nil,
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"error":"Invalid request body","details":"unexpected EOF"}`,
		},
		{
			name: "validation error - missing required fields",
			requestBody: `{
				"description": "Missing required fields"
			}`,
			mockResponse:   nil,
			mockError:      errors.New("validation failed"),
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name: "duplicate plugin name",
			requestBody: `{
				"name": "existing-plugin",
				"title": "Existing Plugin",
				"description": "This plugin already exists",
				"icon": "TestIcon",
				"react_component_path": "/plugins/existing/Existing.jsx",
				"backend_server_url": "http://localhost:3003",
				"owner": "Test Team"
			}`,
			mockResponse:   nil,
			mockError:      &service.ValidationError{Message: "Plugin with this name already exists"},
			expectedStatus: http.StatusConflict,
			expectedBody:   `{"error":"Plugin with this name already exists"}`,
		},
		{
			name: "service error",
			requestBody: `{
				"name": "error-plugin",
				"title": "Error Plugin",
				"description": "This will cause a service error",
				"icon": "TestIcon",
				"react_component_path": "/plugins/error/Error.jsx",
				"backend_server_url": "http://localhost:3004",
				"owner": "Test Team"
			}`,
			mockResponse:   nil,
			mockError:      assert.AnError,
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockPluginService)
			handler := NewPluginHandler(mockService)

			if tt.mockResponse != nil || (tt.mockError != nil && tt.expectedStatus != http.StatusBadRequest) {
				mockService.On("CreatePlugin", mock.AnythingOfType("*service.CreatePluginRequest")).Return(tt.mockResponse, tt.mockError)
			}

			router := gin.New()
			router.POST("/plugins", handler.CreatePlugin)

			req, _ := http.NewRequest("POST", "/plugins", strings.NewReader(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedBody != "" {
				assert.JSONEq(t, tt.expectedBody, w.Body.String())
			} else if tt.mockResponse != nil {
				var response service.PluginResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, tt.mockResponse.Name, response.Name)
				assert.Equal(t, tt.mockResponse.Title, response.Title)
			}

			mockService.AssertExpectations(t)
		})
	}
}

func TestPluginHandler_UpdatePlugin(t *testing.T) {
	gin.SetMode(gin.TestMode)

	validID := uuid.New()
	invalidID := "invalid-uuid"

	tests := []struct {
		name           string
		pluginID       string
		requestBody    string
		mockResponse   *service.PluginResponse
		mockError      error
		expectedStatus int
		expectedBody   string
	}{
		{
			name:     "successful update",
			pluginID: validID.String(),
			requestBody: `{
				"title": "Updated Plugin Title",
				"description": "Updated description"
			}`,
			mockResponse: &service.PluginResponse{
				ID:                 validID,
				Name:               "existing-plugin",
				Title:              "Updated Plugin Title",
				Description:        "Updated description",
				Icon:               "TestIcon",
				ReactComponentPath: "/plugins/existing/Existing.jsx",
				BackendServerURL:   "http://localhost:3003",
				Owner:              "Test Team",
			},
			mockError:      nil,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid UUID format",
			pluginID:       invalidID,
			requestBody:    `{"title": "Updated Title"}`,
			mockResponse:   nil,
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"error":"invalid plugin ID format"}`,
		},
		{
			name:           "invalid JSON",
			pluginID:       validID.String(),
			requestBody:    `{"title": "invalid-json"`,
			mockResponse:   nil,
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"error":"Invalid request body","details":"unexpected EOF"}`,
		},
		{
			name:     "plugin not found",
			pluginID: validID.String(),
			requestBody: `{
				"title": "Updated Title"
			}`,
			mockResponse:   nil,
			mockError:      errors.New("record not found"),
			expectedStatus: http.StatusNotFound,
			expectedBody:   `{"error":"Plugin not found"}`,
		},
		{
			name:     "duplicate plugin name",
			pluginID: validID.String(),
			requestBody: `{
				"name": "existing-name"
			}`,
			mockResponse:   nil,
			mockError:      &service.ValidationError{Message: "Plugin with this name already exists"},
			expectedStatus: http.StatusConflict,
			expectedBody:   `{"error":"Plugin with this name already exists"}`,
		},
		{
			name:     "service error",
			pluginID: validID.String(),
			requestBody: `{
				"title": "Error Update"
			}`,
			mockResponse:   nil,
			mockError:      assert.AnError,
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockPluginService)
			handler := NewPluginHandler(mockService)

			if tt.mockResponse != nil || (tt.mockError != nil && tt.pluginID != invalidID && tt.expectedStatus != http.StatusBadRequest) {
				parsedID, _ := uuid.Parse(tt.pluginID)
				mockService.On("UpdatePlugin", parsedID, mock.AnythingOfType("*service.UpdatePluginRequest")).Return(tt.mockResponse, tt.mockError)
			}

			router := gin.New()
			router.PUT("/plugins/:id", handler.UpdatePlugin)

			req, _ := http.NewRequest("PUT", "/plugins/"+tt.pluginID, strings.NewReader(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedBody != "" {
				assert.JSONEq(t, tt.expectedBody, w.Body.String())
			} else if tt.mockResponse != nil {
				var response service.PluginResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, tt.mockResponse.ID, response.ID)
				assert.Equal(t, tt.mockResponse.Title, response.Title)
			}

			mockService.AssertExpectations(t)
		})
	}
}

func TestPluginHandler_DeletePlugin(t *testing.T) {
	gin.SetMode(gin.TestMode)

	validID := uuid.New()
	invalidID := "invalid-uuid"

	tests := []struct {
		name           string
		pluginID       string
		mockError      error
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "successful deletion",
			pluginID:       validID.String(),
			mockError:      nil,
			expectedStatus: http.StatusNoContent,
		},
		{
			name:           "invalid UUID format",
			pluginID:       invalidID,
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"error":"invalid plugin ID format"}`,
		},
		{
			name:           "plugin not found",
			pluginID:       validID.String(),
			mockError:      errors.New("record not found"),
			expectedStatus: http.StatusNotFound,
			expectedBody:   `{"error":"Plugin not found"}`,
		},
		{
			name:           "service error",
			pluginID:       validID.String(),
			mockError:      assert.AnError,
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockPluginService)
			handler := NewPluginHandler(mockService)

			if tt.pluginID != invalidID {
				parsedID, _ := uuid.Parse(tt.pluginID)
				mockService.On("DeletePlugin", parsedID).Return(tt.mockError)
			}

			router := gin.New()
			router.DELETE("/plugins/:id", handler.DeletePlugin)

			req, _ := http.NewRequest("DELETE", "/plugins/"+tt.pluginID, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedBody != "" {
				assert.JSONEq(t, tt.expectedBody, w.Body.String())
			}

			mockService.AssertExpectations(t)
		})
	}
}

func TestPluginHandler_GetPluginByID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	validID := uuid.New()
	invalidID := "invalid-uuid"

	tests := []struct {
		name           string
		pluginID       string
		mockResponse   *service.PluginResponse
		mockError      error
		expectedStatus int
		expectedBody   string
	}{
		{
			name:     "successful request",
			pluginID: validID.String(),
			mockResponse: &service.PluginResponse{
				ID:                 validID,
				Name:               "sample-plugin",
				Title:              "Sample Plugin",
				Description:        "A sample plugin",
				Icon:               "Puzzle",
				ReactComponentPath: "/plugins/sample/Sample.jsx",
				BackendServerURL:   "http://localhost:3001",
				Owner:              "Developer Portal Team",
			},
			mockError:      nil,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid UUID format",
			pluginID:       invalidID,
			mockResponse:   nil,
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"error":"invalid plugin ID format"}`,
		},
		{
			name:           "plugin not found",
			pluginID:       validID.String(),
			mockResponse:   nil,
			mockError:      assert.AnError,
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockPluginService)
			handler := NewPluginHandler(mockService)

			if tt.mockResponse != nil || (tt.mockError != nil && tt.pluginID != invalidID) {
				parsedID, _ := uuid.Parse(tt.pluginID)
				mockService.On("GetPluginByID", parsedID).Return(tt.mockResponse, tt.mockError)
			}

			router := gin.New()
			router.GET("/plugins/:id", handler.GetPluginByID)

			req, _ := http.NewRequest("GET", "/plugins/"+tt.pluginID, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedBody != "" {
				assert.JSONEq(t, tt.expectedBody, w.Body.String())
			} else if tt.mockResponse != nil {
				var response service.PluginResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, tt.mockResponse.ID, response.ID)
				assert.Equal(t, tt.mockResponse.Name, response.Name)
			}

			mockService.AssertExpectations(t)
		})
	}
}

func TestNewPluginHandlerWithGitHub(t *testing.T) {
	mockPluginService := new(MockPluginService)
	mockGitHubService := new(MockGitHubService)

	handler := NewPluginHandlerWithGitHub(mockPluginService, mockGitHubService)

	assert.NotNil(t, handler)
	assert.Equal(t, mockPluginService, handler.pluginService)
	assert.Equal(t, mockGitHubService, handler.githubService)
	assert.NotNil(t, handler.proxyCache)
}

func TestPluginHandler_GetPluginUI(t *testing.T) {
	gin.SetMode(gin.TestMode)

	validID := uuid.New()
	invalidID := "invalid-uuid"

	tests := []struct {
		name           string
		pluginID       string
		setupAuth      bool
		mockResponse   *service.PluginUIResponse
		mockError      error
		expectedStatus int
		expectedBody   string
	}{
		{
			name:      "successful UI content retrieval",
			pluginID:  validID.String(),
			setupAuth: true,
			mockResponse: &service.PluginUIResponse{
				Content:     "import React from 'react';",
				ContentType: "text/typescript",
			},
			mockError:      nil,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid plugin ID format",
			pluginID:       invalidID,
			setupAuth:      true,
			mockResponse:   nil,
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"error":"invalid plugin ID format"}`,
		},
		{
			name:           "missing authentication",
			pluginID:       validID.String(),
			setupAuth:      false,
			mockResponse:   nil,
			mockError:      nil,
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   `{"error":"Authentication required"}`,
		},
		{
			name:           "plugin not found",
			pluginID:       validID.String(),
			setupAuth:      true,
			mockResponse:   nil,
			mockError:      errors.New("record not found"),
			expectedStatus: http.StatusNotFound,
			expectedBody:   `{"error":"Plugin not found"}`,
		},
		{
			name:           "service error",
			pluginID:       validID.String(),
			setupAuth:      true,
			mockResponse:   nil,
			mockError:      errors.New("service error"),
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockPluginService := new(MockPluginService)
			mockGitHubService := new(MockGitHubService)
			handler := NewPluginHandlerWithGitHub(mockPluginService, mockGitHubService)

			if tt.mockResponse != nil || (tt.mockError != nil && tt.pluginID != invalidID && tt.setupAuth) {
				parsedID, _ := uuid.Parse(tt.pluginID)
				mockPluginService.On("GetPluginUIContent", mock.Anything, parsedID, mockGitHubService, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(tt.mockResponse, tt.mockError)
			}

			router := gin.New()

			// Setup authentication middleware if needed
			if tt.setupAuth {
				router.Use(func(c *gin.Context) {
					mockClaims := &auth.AuthClaims{
						Username: "testuser",
						Email:    "testuser@example.com",
						UUID:     "test-uuid",
					}
					c.Set("auth_claims", mockClaims)
					c.Next()
				})
			}

			router.GET("/plugins/:id/ui", handler.GetPluginUI)

			req, _ := http.NewRequest("GET", "/plugins/"+tt.pluginID+"/ui", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedBody != "" {
				assert.JSONEq(t, tt.expectedBody, w.Body.String())
			} else if tt.mockResponse != nil {
				var response service.PluginUIResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, tt.mockResponse.Content, response.Content)
				assert.Equal(t, tt.mockResponse.ContentType, response.ContentType)
			}

			mockPluginService.AssertExpectations(t)
		})
	}
}

func TestPluginHandler_GetPluginUI_NoGitHubService(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockPluginService := new(MockPluginService)
	handler := NewPluginHandler(mockPluginService) // No GitHub service

	router := gin.New()
	router.Use(func(c *gin.Context) {
		mockClaims := &auth.AuthClaims{
			Username: "testuser",
			Email:    "testuser@example.com",
			UUID:     "test-uuid",
		}
		c.Set("auth_claims", mockClaims)
		c.Next()
	})
	router.GET("/plugins/:id/ui", handler.GetPluginUI)

	validID := uuid.New()
	req, _ := http.NewRequest("GET", "/plugins/"+validID.String()+"/ui", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.JSONEq(t, `{"error":"GitHub service not available"}`, w.Body.String())
}
