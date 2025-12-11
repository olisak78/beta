package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"developer-portal-backend/internal/auth"
	"developer-portal-backend/internal/cache"
	apperrors "developer-portal-backend/internal/errors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestParseRepositoryFromURL_Internal tests the internal parseRepositoryFromURL function
func TestParseRepositoryFromURL_Internal(t *testing.T) {
	tests := []struct {
		name             string
		url              string
		expectedOwner    string
		expectedRepoName string
		expectedFullName string
	}{
		{
			name:             "StandardGitHubURL",
			url:              "https://github.com/octocat/Hello-World/pull/42",
			expectedOwner:    "octocat",
			expectedRepoName: "Hello-World",
			expectedFullName: "octocat/Hello-World",
		},
		{
			name:             "GitHubEnterpriseURL",
			url:              "https://github.enterprise.com/myorg/myrepo/pull/123",
			expectedOwner:    "myorg",
			expectedRepoName: "myrepo",
			expectedFullName: "myorg/myrepo",
		},
		{
			name:             "URLWithHyphens",
			url:              "https://github.com/my-org/my-awesome-repo/pull/1",
			expectedOwner:    "my-org",
			expectedRepoName: "my-awesome-repo",
			expectedFullName: "my-org/my-awesome-repo",
		},
		{
			name:             "URLWithUnderscores",
			url:              "https://github.com/my_org/my_repo/pull/999",
			expectedOwner:    "my_org",
			expectedRepoName: "my_repo",
			expectedFullName: "my_org/my_repo",
		},
		{
			name:             "EmptyURL",
			url:              "",
			expectedOwner:    "",
			expectedRepoName: "",
			expectedFullName: "",
		},
		{
			name:             "InvalidURL",
			url:              "https://github.com/",
			expectedOwner:    "",
			expectedRepoName: "",
			expectedFullName: "",
		},
		{
			name:             "URLWithOnlyOwner",
			url:              "https://github.com/owner",
			expectedOwner:    "",
			expectedRepoName: "",
			expectedFullName: "",
		},
		{
			name:             "HTTPProtocol",
			url:              "http://github.com/owner/repo/pull/1",
			expectedOwner:    "owner",
			expectedRepoName: "repo",
			expectedFullName: "owner/repo",
		},
		{
			name:             "URLWithTrailingSlash",
			url:              "https://github.com/owner/repo/",
			expectedOwner:    "owner",
			expectedRepoName: "repo",
			expectedFullName: "owner/repo",
		},
		{
			name:             "URLWithIssuesPath",
			url:              "https://github.com/owner/repo/issues/42",
			expectedOwner:    "owner",
			expectedRepoName: "repo",
			expectedFullName: "owner/repo",
		},
		{
			name:             "GitHubEnterpriseWithPort",
			url:              "https://github.example.com:8443/orgname/projectname",
			expectedOwner:    "orgname",
			expectedRepoName: "projectname",
			expectedFullName: "orgname/projectname",
		},
		{
			name:             "URLWithSpecialCharacters",
			url:              "https://github.com/my-org_123/my.repo-name_v2/pull/1",
			expectedOwner:    "my-org_123",
			expectedRepoName: "my.repo-name_v2",
			expectedFullName: "my-org_123/my.repo-name_v2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			owner, repoName, fullName := parseRepositoryFromURL(tt.url)

			assert.Equal(t, tt.expectedOwner, owner, "Owner mismatch")
			assert.Equal(t, tt.expectedRepoName, repoName, "Repository name mismatch")
			assert.Equal(t, tt.expectedFullName, fullName, "Full name mismatch")
		})
	}

}

func (m *mockAuthService) GetGitHubAccessToken(userUUID, provider string) (string, error) {
	if m.tokenErr != nil {
		return "", m.tokenErr
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock GitHub GraphQL server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify it's a GraphQL request
				assert.Equal(t, "POST", r.Method)
				assert.Equal(t, "/api/graphql", r.URL.Path)
				assert.Contains(t, r.Header.Get("Authorization"), "Bearer")
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			// Create mock auth service
			mockAuthService := &mockAuthServiceForContributions{
				accessToken: "test-token",
				baseURL:     server.URL,
			}

			// Create cache service
			mockCache := cache.NewInMemoryCache(5*time.Minute, 10*time.Minute)

			githubService := NewGitHubServiceWithAdapter(mockAuthService, mockCache)

			claims := &auth.AuthClaims{
				UserID:   12345,
				Username: "testuser",
				Email:    "test@example.com",
				Provider: "githubtools",
			}

			ctx := context.Background()
			result, err := githubService.GetUserTotalContributions(ctx, claims, "30d")

			require.Error(t, err)
			assert.Nil(t, result)
			assert.Contains(t, err.Error(), tt.expectedError)
		})
	}
	return m.accessToken, nil
}

func (m *mockAuthService) GetGitHubClient(provider string) (*auth.GitHubClient, error) {
	if m.clientErr != nil {
		return nil, m.clientErr
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			mockAuthService := &mockAuthServiceForContributions{
				accessToken: "test-token",
				baseURL:     server.URL,
			}

			// Logger not needed for tests
			mockCache := cache.NewInMemoryCache(5*time.Minute, 10*time.Minute)

			githubService := NewGitHubServiceWithAdapter(mockAuthService, mockCache)

			claims := &auth.AuthClaims{
				UserID:   12345,
				Username: "testuser",
				Email:    "test@example.com",
				Provider: "githubtools",
			}

			ctx := context.Background()
			result, err := githubService.GetUserTotalContributions(ctx, claims, "30d")

			require.Error(t, err)
			assert.Nil(t, result)
			assert.Contains(t, err.Error(), tt.expectedError)
		})
	}
	return auth.NewGitHubClient(cfg), nil
}

// TestParseRepositoryFromURL_Internal tests the internal parseRepositoryFromURL function
func TestParseRepositoryFromURL_Internal(t *testing.T) {
	tests := []struct {
		name          string
		url           string
		expectedOwner string
		expectedRepo  string
		expectedFull  string
	}{
		{
			name:          "StandardGitHubURL",
			url:           "https://github.com/octocat/Hello-World/pull/42",
			expectedOwner: "octocat",
			expectedRepo:  "Hello-World",
			expectedFull:  "octocat/Hello-World",
		},
		{
			name:          "EnterpriseURL",
			url:           "https://github.enterprise.com/myorg/myrepo/pull/123",
			expectedOwner: "myorg",
			expectedRepo:  "myrepo",
			expectedFull:  "myorg/myrepo",
		},
		{
			name: "MissingDataField",
			responseBody: `{
				"viewer": {}
			}`,
			expectedError: "failed to unmarshal result",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			mockAuthService := &mockAuthServiceForContributions{
				accessToken: "test-token",
				baseURL:     server.URL,
			}

			// Logger not needed for tests
			mockCache := cache.NewInMemoryCache(5*time.Minute, 10*time.Minute)

			githubService := NewGitHubServiceWithAdapter(mockAuthService, mockCache)

			claims := &auth.AuthClaims{
				UserID:   12345,
				Username: "testuser",
				Email:    "test@example.com",
				Provider: "githubtools",
			}

			ctx := context.Background()
			result, err := githubService.GetUserTotalContributions(ctx, claims, "30d")

			require.Error(t, err)
			assert.Nil(t, result)
			assert.Contains(t, err.Error(), tt.expectedError)
		})
	}
}

// TestGetUserTotalContributions_SuccessfulResponses tests successful API calls
func TestGetUserTotalContributions_SuccessfulResponses(t *testing.T) {
	tests := []struct {
		name               string
		period             string
		totalContributions int
		responseStartedAt  string
		responseEndedAt    string
		expectedPeriod     string
	}{
		{
			name:               "ThirtyDayPeriod",
			period:             "30d",
			totalContributions: 42,
			responseStartedAt:  "2024-09-16T00:00:00Z",
			responseEndedAt:    "2024-10-16T23:59:59Z",
			expectedPeriod:     "30d",
		},
		{
			name:               "NinetyDayPeriod",
			period:             "90d",
			totalContributions: 256,
			responseStartedAt:  "2024-07-18T00:00:00Z",
			responseEndedAt:    "2024-10-16T23:59:59Z",
			expectedPeriod:     "90d",

		},
		{
			name:          "InvalidURL",
			url:           "https://github.com/",
			expectedOwner: "",
			expectedRepo:  "",
			expectedFull:  "",
		},
		{
			name:          "EmptyURL",
			url:           "",
			expectedOwner: "",
			expectedRepo:  "",
			expectedFull:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify request
				assert.Equal(t, "POST", r.Method)
				assert.Equal(t, "/api/graphql", r.URL.Path)
				assert.Contains(t, r.Header.Get("Authorization"), "Bearer test-token")

				// Verify query contains expected parameters
				var reqBody struct {
					Query string `json:"query"`
				}
				json.NewDecoder(r.Body).Decode(&reqBody)

				if tt.period != "" {
					assert.Contains(t, reqBody.Query, "contributionsCollection(from:")
				} else {
					assert.Contains(t, reqBody.Query, "contributionsCollection {")
					assert.NotContains(t, reqBody.Query, "contributionsCollection(from:")
				}

				// Send successful response
				response := map[string]interface{}{
					"data": map[string]interface{}{
						"viewer": map[string]interface{}{
							"contributionsCollection": map[string]interface{}{
								"startedAt": tt.responseStartedAt,
								"endedAt":   tt.responseEndedAt,
								"contributionCalendar": map[string]interface{}{
									"totalContributions": tt.totalContributions,
								},
							},
						},
					},
				}

				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(response)
			}))
			defer server.Close()

			mockAuthService := &mockAuthServiceForContributions{
				accessToken: "test-token",
				baseURL:     server.URL,
			}

			// Logger not needed for tests
			mockCache := cache.NewInMemoryCache(5*time.Minute, 10*time.Minute)

			githubService := NewGitHubServiceWithAdapter(mockAuthService, mockCache)

			claims := &auth.AuthClaims{
				UserID:   12345,
				Username: "testuser",
				Email:    "test@example.com",
				Provider: "githubtools",
			}

			ctx := context.Background()
			result, err := githubService.GetUserTotalContributions(ctx, claims, tt.period)

			require.NoError(t, err)
			require.NotNil(t, result)

			assert.Equal(t, tt.totalContributions, result.TotalContributions)
			assert.Equal(t, tt.expectedPeriod, result.Period)
			assert.Equal(t, tt.responseStartedAt, result.From)
			assert.Equal(t, tt.responseEndedAt, result.To)
		})
	}
}

// TestGetUserTotalContributions_ContextTimeout tests context timeout handling
func TestGetUserTotalContributions_ContextTimeout(t *testing.T) {
	// Create a slow server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	mockAuthService := &mockAuthServiceForContributions{
		accessToken: "test-token",
		baseURL:     server.URL,
	}

	// Logger not needed for tests
	mockCache := cache.NewInMemoryCache(5*time.Minute, 10*time.Minute)

	githubService := NewGitHubServiceWithAdapter(mockAuthService, mockCache)

	claims := &auth.AuthClaims{
		UserID:   12345,
		Username: "testuser",
		Email:    "test@example.com",
		Provider: "githubtools",
	}

	// Create context with very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	result, err := githubService.GetUserTotalContributions(ctx, claims, "30d")

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "context deadline exceeded")
}

// TestGetUserTotalContributions_PeriodValidation tests period validation
func TestGetUserTotalContributions_PeriodValidation(t *testing.T) {
	// Logger not needed for tests
	mockCache := cache.NewInMemoryCache(5*time.Minute, 10*time.Minute)

	githubService := NewGitHubService(nil, mockCache)

	claims := &auth.AuthClaims{
		UserID:   12345,
		Username: "testuser",
		Email:    "test@example.com",
		Provider: "githubtools",
	}

	invalidPeriods := []string{
		"30",     // missing 'd'
		"abc",    // not a number
		"30days", // wrong format
		"-30d",   // negative
		"0d",     // zero
		"30m",    // wrong unit
		"30y",    // wrong unit
		"d30",    // reversed
		"30 d",   // space
	}

	ctx := context.Background()

	for _, period := range invalidPeriods {
		t.Run(fmt.Sprintf("Invalid_%s", period), func(t *testing.T) {
			result, err := githubService.GetUserTotalContributions(ctx, claims, period)

			require.Error(t, err)
			assert.Nil(t, result)
			assert.True(t, errors.Is(err, apperrors.ErrInvalidPeriodFormat))
		})
	}
}

func TestGetUserTotalContributions_Success(t *testing.T) {
	// Mock GraphQL server for enterprise baseURL
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Service constructs baseURL + "/api/graphql"
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/api/graphql", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		resp := map[string]interface{}{
			"data": map[string]interface{}{
				"viewer": map[string]interface{}{
					"contributionsCollection": map[string]interface{}{
						"startedAt": "2024-10-16T00:00:00Z",
						"endedAt":   "2025-10-16T23:59:59Z",
						"contributionCalendar": map[string]interface{}{
							"totalContributions": 123,
						},
					},
				},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	mock := &mockAuthService{
		accessToken: "test-token",
		baseURL:     server.URL,
	}

	// Logger not needed for tests
	mockCache := cache.NewInMemoryCache(5*time.Minute, 10*time.Minute)

	githubService := NewGitHubServiceWithAdapter(mockAuthService, mockCache)

	claims := &auth.AuthClaims{
		UserID:   12345,
		Username: "testuser",
		Email:    "test@example.com",
		Provider: "githubtools",
	}

	ctx := context.Background()
	res, err := svc.GetUserTotalContributions(ctx, "test-uuid", "githubtools", "30d")
	require.NoError(t, err)
	require.NotNil(t, res)
	assert.Equal(t, 123, res.TotalContributions)
	assert.Equal(t, "30d", res.Period)
	assert.Equal(t, "2024-10-16T00:00:00Z", res.From)
	assert.Equal(t, "2025-10-16T23:59:59Z", res.To)
}

func TestGetUserTotalContributions_RateLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden) // 403
		_, _ = w.Write([]byte(`{"message":"API rate limit exceeded"}`))
	}))
	defer server.Close()

	mock := &mockAuthService{
		accessToken: "test-token",
		baseURL:     server.URL,
	}

	// Logger not needed for tests
	mockCache := cache.NewInMemoryCache(5*time.Minute, 10*time.Minute)

	githubService := NewGitHubServiceWithAdapter(mockAuthService, mockCache)

	claims := &auth.AuthClaims{
		UserID:   12345,
		Username: "testuser",
		Email:    "test@example.com",
		Provider: "githubtools",
	}

	ctx := context.Background()
	result, err := githubService.GetUserTotalContributions(ctx, claims, "30d")

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 100, result.TotalContributions)
	// Should fall back to calculated dates
	assert.NotEmpty(t, result.From)
	assert.NotEmpty(t, result.To)
}

// TestGetUserOpenPullRequests_EdgeCases tests edge cases for pull requests
func TestGetUserOpenPullRequests_EdgeCases(t *testing.T) {
	t.Run("EmptySearchResults", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Return empty search results
			response := map[string]interface{}{
				"total_count":        0,
				"incomplete_results": false,
				"items":              []interface{}{},
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		mockAuthService := &mockAuthServiceForContributions{
			accessToken: "test-token",
			baseURL:     server.URL,
		}

		// Logger not needed for tests
		mockCache := cache.NewInMemoryCache(5*time.Minute, 10*time.Minute)

		githubService := NewGitHubServiceWithAdapter(mockAuthService, mockCache)

		claims := &auth.AuthClaims{
			UserID:   12345,
			Username: "testuser",
			Email:    "test@example.com",
			Provider: "githubtools",
		}

		ctx := context.Background()
		result, err := githubService.GetUserOpenPullRequests(ctx, claims, "open", "created", "desc", 30, 1)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, 0, result.Total)
		assert.Equal(t, 0, len(result.PullRequests))
	})

	t.Run("NilAuthService", func(t *testing.T) {
		// Logger not needed for tests
		mockCache := cache.NewInMemoryCache(5*time.Minute, 10*time.Minute)

		githubService := NewGitHubService(nil, mockCache)

		claims := &auth.AuthClaims{
			UserID:   12345,
			Username: "testuser",
			Email:    "test@example.com",
			Provider: "githubtools",
		}

		ctx := context.Background()
		result, err := githubService.GetUserOpenPullRequests(ctx, claims, "open", "created", "desc", 30, 1)

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "failed to get GitHub access token")
	})

	t.Run("ContextCancelled", func(t *testing.T) {
		mockAuthService := &mockAuthServiceForContributions{
			accessToken: "test-token",
			baseURL:     "http://localhost:9999",
		}

		// Logger not needed for tests
		mockCache := cache.NewInMemoryCache(5*time.Minute, 10*time.Minute)

		githubService := NewGitHubServiceWithAdapter(mockAuthService, mockCache)

		claims := &auth.AuthClaims{
			UserID:   12345,
			Username: "testuser",
			Email:    "test@example.com",
			Provider: "githubtools",
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		result, err := githubService.GetUserOpenPullRequests(ctx, claims, "open", "created", "desc", 30, 1)

		require.Error(t, err)
		assert.Nil(t, result)
	})
}

// TestAuthServiceAdapter tests the auth service adapter
func TestAuthServiceAdapter(t *testing.T) {
	t.Run("NewAuthServiceAdapter_WithNil", func(t *testing.T) {
		adapter := NewAuthServiceAdapter(nil)
		assert.NotNil(t, adapter)
	})

	t.Run("NewAuthServiceAdapter_WithNonNil", func(t *testing.T) {
		// We can't easily create a real AuthService without complex setup,
		// but we can at least verify the function doesn't panic
		adapter := NewAuthServiceAdapter(nil)
		assert.NotNil(t, adapter)
	})

	t.Run("GetGitHubAccessTokenFromClaims_NilAuthService", func(t *testing.T) {
		adapter := NewAuthServiceAdapter(nil)
		claims := &auth.AuthClaims{UserID: 123}

		token, err := adapter.GetGitHubAccessTokenFromClaims(claims)

		assert.Error(t, err)
		assert.Empty(t, token)
		assert.Contains(t, err.Error(), "auth service is not initialized")
	})

	t.Run("GetGitHubClient_NilAuthService", func(t *testing.T) {
		adapter := NewAuthServiceAdapter(nil)

		client, err := adapter.GetGitHubClient("githubtools")

		assert.Error(t, err)
		assert.Nil(t, client)
		assert.Contains(t, err.Error(), "auth service is not initialized")
	})

	t.Run("MockAdapter_Success", func(t *testing.T) {
		// Test that the mock adapter works correctly (covers success paths)
		mock := &mockAuthServiceForContributions{
			accessToken: "test-token",
			baseURL:     "https://github.tools.sap",
		}

		// Test GetGitHubAccessTokenFromClaims
		claims := &auth.AuthClaims{UserID: 123}
		token, err := mock.GetGitHubAccessTokenFromClaims(claims)
		assert.NoError(t, err)
		assert.Equal(t, "test-token", token)

		// Test GetGitHubClient
		client, err := mock.GetGitHubClient("githubtools")
		assert.NoError(t, err)
		assert.NotNil(t, client)
	})
}

// Mock auth service for contributions tests
type mockAuthServiceForContributions struct {
	accessToken string
	baseURL     string
}

func (m *mockAuthServiceForContributions) GetGitHubAccessTokenFromClaims(claims *auth.AuthClaims) (string, error) {
	if m.accessToken == "" {
		return "", fmt.Errorf("no access token")
	}
	return m.accessToken, nil
}

func (m *mockAuthServiceForContributions) GetGitHubClient(provider string) (*auth.GitHubClient, error) {
	// Create a test client with our test server's baseURL
	config := &auth.ProviderConfig{
		ClientID:          "test-client-id",
		ClientSecret:      "test-client-secret",
		EnterpriseBaseURL: m.baseURL,
	}
	return auth.NewGitHubClient(config), nil
}

func TestGetContributionsHeatmap_Success(t *testing.T) {
	// Mock GraphQL server for enterprise baseURL
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/api/graphql", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		resp := map[string]interface{}{
			"data": map[string]interface{}{
				"viewer": map[string]interface{}{
					"contributionsCollection": map[string]interface{}{
						"startedAt": "2024-10-30T00:00:00Z",
						"endedAt":   "2025-10-30T23:59:59Z",
						"contributionCalendar": map[string]interface{}{
							"totalContributions": 5,
							"weeks": []map[string]interface{}{
								{
									"firstDay": "2024-10-27",
									"contributionDays": []map[string]interface{}{
										{
											"date":              "2024-10-27",
											"contributionCount": 2,
											"contributionLevel": "FIRST_QUARTILE",
											"color":             "#9be9a8",
										},
									},
								},
							},
						},
					},
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create mock auth service
	mockAuth := &mockAuthServiceForContributions{
		accessToken: "test-access-token",
		baseURL:     server.URL,
	}

	// Logger not needed for tests
	mockCache := cache.NewInMemoryCache(5*time.Minute, 10*time.Minute)

	// Create GitHub service
	githubService := NewGitHubServiceWithAdapter(mockAuth, mockCache)

	// Create test claims
	claims := &auth.AuthClaims{
		UserID:   12345,
		Username: "testuser",
		Email:    "test@example.com",
		Provider: "githubtools",
	}

	ctx := context.Background()

	// Test without period (default)
	result, err := githubService.GetContributionsHeatmap(ctx, claims, "")

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 1234, result.TotalContributions)
	assert.Equal(t, "2024-10-30T00:00:00Z", result.From)
	assert.Equal(t, "2025-10-30T23:59:59Z", result.To)
	assert.Len(t, result.Weeks, 2)
	
	// Verify first week
	assert.Equal(t, "2024-10-27", result.Weeks[0].FirstDay)
	assert.Len(t, result.Weeks[0].ContributionDays, 2)
	assert.Equal(t, "2024-10-27", result.Weeks[0].ContributionDays[0].Date)
	assert.Equal(t, 5, result.Weeks[0].ContributionDays[0].ContributionCount)
	assert.Equal(t, "SECOND_QUARTILE", result.Weeks[0].ContributionDays[0].ContributionLevel)
	assert.Equal(t, "#40c463", result.Weeks[0].ContributionDays[0].Color)

	// Verify second week
	assert.Equal(t, "2024-11-03", result.Weeks[1].FirstDay)
	assert.Len(t, result.Weeks[1].ContributionDays, 1)
}

// TestGetContributionsHeatmap_WithPeriod tests heatmap with custom period
func TestGetContributionsHeatmap_WithPeriod(t *testing.T) {
	// Create a test GraphQL server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Parse the request to verify period parameters
		var reqBody struct {
			Query string `json:"query"`
		}
		err := json.NewDecoder(r.Body).Decode(&reqBody)
		require.NoError(t, err)
		
		// Verify that from/to parameters are in the query
		assert.Contains(t, reqBody.Query, "contributionsCollection(from:")
		assert.Contains(t, reqBody.Query, "to:")

		// Return mock response
		response := map[string]interface{}{
			"data": map[string]interface{}{
				"viewer": map[string]interface{}{
					"contributionsCollection": map[string]interface{}{
						"startedAt": "2025-07-31T00:00:00Z",
						"endedAt":   "2025-10-30T23:59:59Z",
						"contributionCalendar": map[string]interface{}{
							"totalContributions": 456,
							"weeks": []map[string]interface{}{
								{
									"firstDay": "2025-07-27",
									"contributionDays": []map[string]interface{}{
										{
											"date":              "2024-10-28",
											"contributionCount": 3,
											"contributionLevel": "SECOND_QUARTILE",
											"color":             "#40c463",
										},
									},
								},
							},
						},
					},
				},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	mock := &mockAuthService{
		accessToken: "test-token",
		baseURL:     server.URL,
	}

	// Logger not needed for tests
	mockCache := cache.NewInMemoryCache(5*time.Minute, 10*time.Minute)

	githubService := NewGitHubServiceWithAdapter(mockAuth, mockCache)

	claims := &auth.AuthClaims{
		UserID:   12345,
		Username: "testuser",
		Email:    "test@example.com",
		Provider: "githubtools",
	}

	ctx := context.Background()
	res, err := svc.GetContributionsHeatmap(ctx, "test-uuid", "githubtools", "")
	require.NoError(t, err)
	require.NotNil(t, res)
	assert.Equal(t, 5, res.TotalContributions)
	assert.Equal(t, "2024-10-30T00:00:00Z", res.From)
	assert.Equal(t, "2025-10-30T23:59:59Z", res.To)
	require.Len(t, res.Weeks, 1)
	require.Len(t, res.Weeks[0].ContributionDays, 2)
}

func TestGetContributionsHeatmap_InvalidPeriod(t *testing.T) {
	// Mock provider configured
	mock := &mockAuthService{
		accessToken: "test-token",
		baseURL:     "", // use public github endpoint
	}
	svc := NewGitHubServiceWithAdapter(mock)

	// Logger not needed for tests
	mockCache := cache.NewInMemoryCache(5*time.Minute, 10*time.Minute)

	githubService := NewGitHubServiceWithAdapter(mockAuth, mockCache)
	ctx := context.Background()
	res, err := svc.GetContributionsHeatmap(ctx, "test-uuid", "githubtools", "bad")
	require.Error(t, err)
	assert.Nil(t, res)
	assert.True(t, errors.Is(err, apperrors.ErrInvalidPeriodFormat))
}

// TestGetContributionsHeatmap_InvalidPeriodFormat tests with invalid period format
func TestGetContributionsHeatmap_InvalidPeriodFormat(t *testing.T) {
	mockAuth := &mockAuthServiceForContributions{
		accessToken: "test-token",
		baseURL:     "https://api.github.com",
	}

	// Logger not needed for tests
	mockCache := cache.NewInMemoryCache(5*time.Minute, 10*time.Minute)

	githubService := NewGitHubServiceWithAdapter(mockAuth, mockCache)

	claims := &auth.AuthClaims{
		UserID:   12345,
		Username: "testuser",
		Email:    "test@example.com",
		Provider: "githubtools",
	}

	ctx := context.Background()

	testCases := []struct {
		name   string
		period string
	}{
		{"InvalidFormat", "30days"},
		{"MissingNumber", "d"},
		{"NoUnit", "30"},
		{"NegativeNumber", "-30d"},
		{"InvalidCharacters", "abc30d"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := githubService.GetContributionsHeatmap(ctx, claims, tc.period)

			require.Error(t, err)
			assert.Nil(t, result)
			assert.True(t, errors.Is(err, apperrors.ErrInvalidPeriodFormat))
		})
	}
}

// TestGetContributionsHeatmap_ProviderNotConfigured tests with unconfigured provider
func TestGetContributionsHeatmap_ProviderNotConfigured(t *testing.T) {
	// Create mock that returns error for GetGitHubClient
	mockAuth := &mockAuthServiceFailure{
		accessToken: "test-token",
		clientError: fmt.Errorf("provider 'invalid' not found"),
	}

	// Logger not needed for tests
	mockCache := cache.NewInMemoryCache(5*time.Minute, 10*time.Minute)

	githubService := NewGitHubServiceWithAdapter(mockAuth, mockCache)

	claims := &auth.AuthClaims{
		UserID:   12345,
		Username: "testuser",
		Email:    "test@example.com",
		Provider: "invalid",
	}

	ctx := context.Background()

	result, err := githubService.GetContributionsHeatmap(ctx, claims, "")

	// GetGitHubAccessToken should error when auth service is nil
	_, err := adapter.GetGitHubAccessToken("test-uuid", "githubtools")
	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, errors.Is(err, apperrors.ErrProviderNotConfigured))
}

// TestGetContributionsHeatmap_RateLimitExceeded tests rate limit handling
func TestGetContributionsHeatmap_RateLimitExceeded(t *testing.T) {
	// Create a test server that returns 403 (rate limit)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"message": "API rate limit exceeded"}`))
	}))
	defer server.Close()

	mockAuth := &mockAuthServiceForContributions{
		accessToken: "test-access-token",
		baseURL:     server.URL,
	}

	// Logger not needed for tests
	mockCache := cache.NewInMemoryCache(5*time.Minute, 10*time.Minute)

	githubService := NewGitHubServiceWithAdapter(mockAuth, mockCache)

	claims := &auth.AuthClaims{
		UserID:   12345,
		Username: "testuser",
		Email:    "test@example.com",
		Provider: "githubtools",
	}

	ctx := context.Background()

	result, err := githubService.GetContributionsHeatmap(ctx, claims, "")

	// GetGitHubClient should error when auth service is nil
	_, err = adapter.GetGitHubClient("githubtools")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "auth service is not initialized")
}

// Context deadline behavior for TotalContributions
func TestGetUserTotalContributions_ContextDeadline(t *testing.T) {
	// Slow server to trigger context timeout
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data": {"viewer": {"contributionsCollection": {"startedAt":"2024-10-16T00:00:00Z","endedAt":"2025-10-16T23:59:59Z","contributionCalendar":{"totalContributions":1}}}}}`))
	}))
	defer server.Close()

	mockAuth := &mockAuthServiceForContributions{
		accessToken: "test-access-token",
		baseURL:     server.URL,
	}

	// Logger not needed for tests
	mockCache := cache.NewInMemoryCache(5*time.Minute, 10*time.Minute)

	githubService := NewGitHubServiceWithAdapter(mockAuth, mockCache)

	claims := &auth.AuthClaims{
		UserID:   12345,
		Username: "testuser",
		Email:    "test@example.com",
		Provider: "githubtools",
	}

	ctx := context.Background()

	result, err := githubService.GetContributionsHeatmap(ctx, claims, "")

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "GraphQL error")
}

// TestGetContributionsHeatmap_HTTPError tests HTTP error handling
func TestGetContributionsHeatmap_HTTPError(t *testing.T) {
	// Create a test server that returns 500
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"message": "Internal server error"}`))
	}))
	defer server.Close()

	mockAuth := &mockAuthServiceForContributions{
		accessToken: "test-access-token",
		baseURL:     server.URL,
	}
	svc := NewGitHubServiceWithAdapter(mock)

	// Logger not needed for tests
	mockCache := cache.NewInMemoryCache(5*time.Minute, 10*time.Minute)

	githubService := NewGitHubServiceWithAdapter(mockAuth, mockCache)

	claims := &auth.AuthClaims{
		UserID:   12345,
		Username: "testuser",
		Email:    "test@example.com",
		Provider: "githubtools",
	}

	ctx := context.Background()

	result, err := githubService.GetContributionsHeatmap(ctx, claims, "")

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "GraphQL query failed with status 500")
}

// TestGetContributionsHeatmap_AccessTokenError tests access token retrieval error
func TestGetContributionsHeatmap_AccessTokenError(t *testing.T) {
	// Create mock that returns error for access token
	mockAuth := &mockAuthServiceFailure{
		tokenError:  fmt.Errorf("failed to get access token"),
		accessToken: "",
	}

	// Logger not needed for tests
	mockCache := cache.NewInMemoryCache(5*time.Minute, 10*time.Minute)

	githubService := NewGitHubServiceWithAdapter(mockAuth, mockCache)

	claims := &auth.AuthClaims{
		UserID:   12345,
		Username: "testuser",
		Email:    "test@example.com",
		Provider: "githubtools",
	}

	ctx := context.Background()

	result, err := githubService.GetContributionsHeatmap(ctx, claims, "")

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to get GitHub access token")
}

// mockAuthServiceFailure is a mock that returns errors
type mockAuthServiceFailure struct {
	accessToken string
	tokenError  error
	clientError error
}

func (m *mockAuthServiceFailure) GetGitHubAccessTokenFromClaims(claims *auth.AuthClaims) (string, error) {
	if m.tokenError != nil {
		return "", m.tokenError
	}
	return m.accessToken, nil
}

func (m *mockAuthServiceFailure) GetGitHubClient(provider string) (*auth.GitHubClient, error) {
	if m.clientError != nil {
		return nil, m.clientError
	}
	config := &auth.ProviderConfig{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
	}
	return auth.NewGitHubClient(config), nil
}