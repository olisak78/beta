package service_test

import (
	"context"
	apperrors "developer-portal-backend/internal/errors"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"
	"net/http"
	"net/http/httptest"
	"testing"

	"developer-portal-backend/internal/auth"
	"developer-portal-backend/internal/cache"
	"developer-portal-backend/internal/mocks"
	"developer-portal-backend/internal/service"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// TestGetUserOpenPullRequests_FullFlow_WithMocks tests the complete flow with mocked auth service
func TestGetUserOpenPullRequests_FullFlow_WithMocks(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create mock GitHub API server
	var mockGitHubServer *httptest.Server
	mockGitHubServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request structure
		assert.Equal(t, "GET", r.Method)
		assert.Contains(t, r.URL.Path, "/search/issues")
		assert.Contains(t, r.URL.RawQuery, "is%3Apr")
		assert.Contains(t, r.URL.RawQuery, "author%3A%40me")

		// Return mock PR data
		response := map[string]interface{}{
			"total_count": 2,
			"items": []map[string]interface{}{
				{
					"id":         int64(123456789),
					"number":     42,
					"title":      "Add new feature",
					"state":      "open",
					"created_at": "2025-01-01T12:00:00Z",
					"updated_at": "2025-01-02T12:00:00Z",
					"html_url":   mockGitHubServer.URL + "/owner/repo/pull/42",
					"draft":      false,
					"user": map[string]interface{}{
						"login":      "testuser",
						"id":         int64(12345),
						"avatar_url": "https://avatars.githubusercontent.com/u/12345",
					},
					"pull_request": map[string]interface{}{
						"url": mockGitHubServer.URL + "/repos/owner/repo/pulls/42",
					},
					"repository": map[string]interface{}{
						"name":      "test-repo",
						"full_name": "owner/test-repo",
						"private":   false,
						"owner": map[string]interface{}{
							"login": "owner",
						},
					},
				},
				{
					"id":         int64(987654321),
					"number":     43,
					"title":      "Fix critical bug",
					"state":      "open",
					"created_at": "2025-01-03T12:00:00Z",
					"updated_at": "2025-01-04T12:00:00Z",
					"html_url":   mockGitHubServer.URL + "/owner/repo/pull/43",
					"draft":      true,
					"user": map[string]interface{}{
						"login":      "testuser",
						"id":         int(12345),
						"avatar_url": "https://avatars.githubusercontent.com/u/12345",
					},
					"pull_request": map[string]interface{}{
						"url": mockGitHubServer.URL + "/repos/owner/repo/pulls/43",
					},
					"repository": map[string]interface{}{
						"name":      "another-repo",
						"full_name": "owner/another-repo",
						"private":   true,
						"owner": map[string]interface{}{
							"login": "owner",
						},
					},
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer mockGitHubServer.Close()

	// Create mock auth service
	mockAuthService := mocks.NewMockGitHubAuthService(ctrl)

	// Set expectations
	mockAuthService.EXPECT().
		GetGitHubAccessToken(gomock.Any(), gomock.Any()).
		Return("test_github_token_123", nil).
		Times(1)

	// Create a GitHub client that points to our mock server
	envConfig := &auth.ProviderConfig{
		ClientID:          "test_client_id",
		ClientSecret:      "test_client_secret",
		EnterpriseBaseURL: mockGitHubServer.URL,
	}
	githubClient := auth.NewGitHubClient(envConfig)

	mockAuthService.EXPECT().
		GetGitHubClient("githubtools").
		Return(githubClient, nil).
		Times(1)

	// Create GitHub service with mock auth
	mockCache := cache.NewInMemoryCache(5*time.Minute, 10*time.Minute)
	githubService := service.NewGitHubServiceWithAdapter(mockAuthService, mockCache)

	provider := "githubtools"
	claims := &auth.AuthClaims{
		Username: "testuser",
		Email:    "test@example.com",
		UUID:     "test-uuid",
	}

	ctx := context.Background()

	// Execute the test
	result, err := githubService.GetUserOpenPullRequests(ctx, claims.UUID, provider, "open", "created", "desc", 30, 1)

	// Assertions
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 2, result.Total)
	assert.Len(t, result.PullRequests, 2)

	// Verify first PR
	pr1 := result.PullRequests[0]
	assert.Equal(t, int64(123456789), pr1.ID)
	assert.Equal(t, 42, pr1.Number)
	assert.Equal(t, "Add new feature", pr1.Title)
	assert.Equal(t, "open", pr1.State)
	assert.False(t, pr1.Draft)
	assert.Equal(t, "testuser", pr1.User.Login)
	assert.Equal(t, "test-repo", pr1.Repo.Name)
	assert.Equal(t, "owner", pr1.Repo.Owner)
	assert.False(t, pr1.Repo.Private)

	// Verify second PR
	pr2 := result.PullRequests[1]
	assert.Equal(t, int64(987654321), pr2.ID)
	assert.Equal(t, 43, pr2.Number)
	assert.Equal(t, "Fix critical bug", pr2.Title)
	assert.True(t, pr2.Draft)
	assert.Equal(t, "another-repo", pr2.Repo.Name)
	assert.True(t, pr2.Repo.Private)
}

// TestGetUserOpenPullRequests_ClosedState tests fetching closed PRs
func TestGetUserOpenPullRequests_ClosedState(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	var mockGitHubServer *httptest.Server
	mockGitHubServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.RawQuery, "state%3Aclosed")

		response := map[string]interface{}{
			"total_count": 1,
			"items": []map[string]interface{}{
				{
					"id":         int64(111),
					"number":     10,
					"title":      "Closed PR",
					"state":      "closed",
					"created_at": "2025-01-01T12:00:00Z",
					"updated_at": "2025-01-05T12:00:00Z",
					"html_url":   mockGitHubServer.URL + "/owner/repo/pull/10",
					"draft":      false,
					"user": map[string]interface{}{
						"login":      "testuser",
						"id":         int64(12345),
						"avatar_url": "https://avatars.githubusercontent.com/u/12345",
					},
					"pull_request": map[string]interface{}{
						"url": mockGitHubServer.URL + "/repos/owner/repo/pulls/10",
					},
					"repository": map[string]interface{}{
						"name":      "test-repo",
						"full_name": "owner/test-repo",
						"private":   false,
						"owner": map[string]interface{}{
							"login": "owner",
						},
					},
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer mockGitHubServer.Close()

	mockAuthService := mocks.NewMockGitHubAuthService(ctrl)
	mockAuthService.EXPECT().GetGitHubAccessToken(gomock.Any(), gomock.Any()).Return("token", nil)

	envConfig := &auth.ProviderConfig{EnterpriseBaseURL: mockGitHubServer.URL}
	mockAuthService.EXPECT().GetGitHubClient(gomock.Any()).Return(auth.NewGitHubClient(envConfig), nil)

	mockCache := cache.NewInMemoryCache(5*time.Minute, 10*time.Minute)
	githubService := service.NewGitHubServiceWithAdapter(mockAuthService, mockCache)
	claims := &auth.AuthClaims{
		UUID: "test-uuid",
	}

	result, err := githubService.GetUserOpenPullRequests(context.Background(), claims.UUID, provider, "closed", "created", "desc", 30, 1)

	require.NoError(t, err)
	assert.Equal(t, 1, result.Total)
	assert.Equal(t, "closed", result.PullRequests[0].State)
}

// TestGetUserOpenPullRequests_EmptyResults tests when no PRs are found
func TestGetUserOpenPullRequests_EmptyResults(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	var mockGitHubServer *httptest.Server
	mockGitHubServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"total_count": 0,
			"items":       []interface{}{},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer mockGitHubServer.Close()

	mockAuthService := mocks.NewMockGitHubAuthService(ctrl)
	mockAuthService.EXPECT().GetGitHubAccessToken(gomock.Any(), gomock.Any()).Return("token", nil)

	envConfig := &auth.ProviderConfig{EnterpriseBaseURL: mockGitHubServer.URL}
	mockAuthService.EXPECT().GetGitHubClient(gomock.Any()).Return(auth.NewGitHubClient(envConfig), nil)

	mockCache := cache.NewInMemoryCache(5*time.Minute, 10*time.Minute)
	githubService := service.NewGitHubServiceWithAdapter(mockAuthService, mockCache)
	claims := &auth.AuthClaims{UserID: 12345, Provider: "githubtools"}

	result, err := githubService.GetUserOpenPullRequests(context.Background(), claims.UUID, provider, "open", "created", "desc", 30, 1)

	require.NoError(t, err)
	assert.Equal(t, 0, result.Total)
	assert.Empty(t, result.PullRequests)
}

// TestGetUserOpenPullRequests_TokenRetrievalFailure tests auth service failure
func TestGetUserOpenPullRequests_TokenRetrievalFailure(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAuthService := mocks.NewMockGitHubAuthService(ctrl)
	mockAuthService.EXPECT().
		GetGitHubAccessToken(gomock.Any(), gomock.Any()).
		Return("", fmt.Errorf("no valid session found")).
		Times(1)

	mockCache := cache.NewInMemoryCache(5*time.Minute, 10*time.Minute)
	githubService := service.NewGitHubServiceWithAdapter(mockAuthService, mockCache)
	claims := &auth.AuthClaims{UserID: 12345}

	result, err := githubService.GetUserOpenPullRequests(context.Background(), claims.UUID, provider, "open", "created", "desc", 30, 1)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to get GitHub access token")
}

// TestGetUserOpenPullRequests_GitHubClientRetrievalFailure tests client retrieval failure
func TestGetUserOpenPullRequests_GitHubClientRetrievalFailure(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAuthService := mocks.NewMockGitHubAuthService(ctrl)
	mockAuthService.EXPECT().GetGitHubAccessToken(gomock.Any(), gomock.Any()).Return("token", nil)
	mockAuthService.EXPECT().
		GetGitHubClient(gomock.Any()).
		Return(nil, fmt.Errorf("client not found")).
		Times(1)

	mockCache := cache.NewInMemoryCache(5*time.Minute, 10*time.Minute)
	githubService := service.NewGitHubServiceWithAdapter(mockAuthService, mockCache)
	claims := &auth.AuthClaims{UserID: 12345, Provider: "invalid"}

	result, err := githubService.GetUserOpenPullRequests(context.Background(), claims.UUID, provider, "open", "created", "desc", 30, 1)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to get GitHub client")
}

// TestGetUserOpenPullRequests_GitHubAPIRateLimit tests rate limit handling
func TestGetUserOpenPullRequests_GitHubAPIRateLimit(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	var mockGitHubServer *httptest.Server
	mockGitHubServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		response := map[string]interface{}{
			"message": "API rate limit exceeded",
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer mockGitHubServer.Close()

	mockAuthService := mocks.NewMockGitHubAuthService(ctrl)
	mockAuthService.EXPECT().GetGitHubAccessToken(gomock.Any(), gomock.Any()).Return("token", nil)

	envConfig := &auth.ProviderConfig{EnterpriseBaseURL: mockGitHubServer.URL}
	mockAuthService.EXPECT().GetGitHubClient(gomock.Any()).Return(auth.NewGitHubClient(envConfig), nil)

	mockCache := cache.NewInMemoryCache(5*time.Minute, 10*time.Minute)
	githubService := service.NewGitHubServiceWithAdapter(mockAuthService, mockCache)
	claims := &auth.AuthClaims{UserID: 12345, Provider: "githubtools"}

	result, err := githubService.GetUserOpenPullRequests(context.Background(), claims.UUID, provider, "open", "created", "desc", 30, 1)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "rate limit exceeded")
}

// TestGetUserOpenPullRequests_DefaultParameterNormalization tests parameter defaults
func TestGetUserOpenPullRequests_DefaultParameterNormalization(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	var capturedQuery string
	var mockGitHubServer *httptest.Server
	mockGitHubServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedQuery = r.URL.RawQuery
		response := map[string]interface{}{
			"total_count": 0,
			"items":       []interface{}{},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer mockGitHubServer.Close()

	mockAuthService := mocks.NewMockGitHubAuthService(ctrl)
	mockAuthService.EXPECT().GetGitHubAccessToken(gomock.Any(), gomock.Any()).Return("token", nil)

	envConfig := &auth.ProviderConfig{EnterpriseBaseURL: mockGitHubServer.URL}
	mockAuthService.EXPECT().GetGitHubClient(gomock.Any()).Return(auth.NewGitHubClient(envConfig), nil)

	mockCache := cache.NewInMemoryCache(5*time.Minute, 10*time.Minute)
	githubService := service.NewGitHubServiceWithAdapter(mockAuthService, mockCache)
	claims := &auth.AuthClaims{UserID: 12345, Provider: "githubtools"}

	// Call with empty parameters to test defaults
	_, err := githubService.GetUserOpenPullRequests(context.Background(), claims.UUID, provider, "", "", "", 0, 0)

	require.NoError(t, err)
	// Verify defaults were applied: state=open, sort=created, order=desc
	assert.Contains(t, capturedQuery, "state%3Aopen")
	assert.Contains(t, capturedQuery, "sort=created")
	assert.Contains(t, capturedQuery, "order=desc")
}

// TestGetUserOpenPullRequests_PaginationParameters tests pagination
func TestGetUserOpenPullRequests_PaginationParameters(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	var capturedQuery string
	var mockGitHubServer *httptest.Server
	mockGitHubServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedQuery = r.URL.RawQuery
		response := map[string]interface{}{
			"total_count": 100,
			"items":       []interface{}{},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer mockGitHubServer.Close()

	mockAuthService := mocks.NewMockGitHubAuthService(ctrl)
	mockAuthService.EXPECT().GetGitHubAccessToken(gomock.Any(), gomock.Any()).Return("token", nil)

	envConfig := &auth.ProviderConfig{EnterpriseBaseURL: mockGitHubServer.URL}
	mockAuthService.EXPECT().GetGitHubClient(gomock.Any()).Return(auth.NewGitHubClient(envConfig), nil)

	mockCache := cache.NewInMemoryCache(5*time.Minute, 10*time.Minute)
	githubService := service.NewGitHubServiceWithAdapter(mockAuthService, mockCache)
	claims := &auth.AuthClaims{UserID: 12345, Provider: "githubtools"}

	// Test with specific pagination
	_, err := githubService.GetUserOpenPullRequests(context.Background(), claims.UUID, provider, "open", "created", "desc", 50, 3)

	require.NoError(t, err)
	assert.Contains(t, capturedQuery, "per_page=50")
	assert.Contains(t, capturedQuery, "page=3")
}

// TestGetUserOpenPullRequests_PRDataParsing tests that all PR fields are correctly parsed
func TestGetUserOpenPullRequests_PRDataParsing(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	var mockGitHubServer *httptest.Server
	mockGitHubServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"total_count": 1,
			"items": []map[string]interface{}{
				{
					"id":         int64(999),
					"number":     99,
					"title":      "Comprehensive Test PR",
					"state":      "open",
					"created_at": "2025-01-10T10:00:00Z",
					"updated_at": "2025-01-11T10:00:00Z",
					"html_url":   "https://github.com/test/repo/pull/99",
					"draft":      true,
					"user": map[string]interface{}{
						"login":      "contributor",
						"id":         int64(54321),
						"avatar_url": "https://avatars.githubusercontent.com/u/54321",
					},
					"pull_request": map[string]interface{}{
						"url": "https://api.github.com/repos/test/repo/pulls/99",
					},
					"repository": map[string]interface{}{
						"name":      "comprehensive-repo",
						"full_name": "test/comprehensive-repo",
						"private":   true,
						"owner": map[string]interface{}{
							"login": "test",
						},
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer mockGitHubServer.Close()

	mockAuthService := mocks.NewMockGitHubAuthService(ctrl)
	mockAuthService.EXPECT().GetGitHubAccessToken(gomock.Any(), gomock.Any()).Return("token", nil)

	envConfig := &auth.ProviderConfig{EnterpriseBaseURL: mockGitHubServer.URL}
	mockAuthService.EXPECT().GetGitHubClient(gomock.Any()).Return(auth.NewGitHubClient(envConfig), nil)

	mockCache := cache.NewInMemoryCache(5*time.Minute, 10*time.Minute)
	githubService := service.NewGitHubServiceWithAdapter(mockAuthService, mockCache)
	claims := &auth.AuthClaims{UserID: 12345, Provider: "githubtools"}

	result, err := githubService.GetUserOpenPullRequests(context.Background(), claims.UUID, provider, "open", "created", "desc", 30, 1)

	require.NoError(t, err)
	require.Len(t, result.PullRequests, 1)

	pr := result.PullRequests[0]
	assert.Equal(t, int64(999), pr.ID)
	assert.Equal(t, 99, pr.Number)
	assert.Equal(t, "Comprehensive Test PR", pr.Title)
	assert.Equal(t, "open", pr.State)
	assert.True(t, pr.Draft)
	assert.Equal(t, "contributor", pr.User.Login)
	assert.Equal(t, int64(54321), pr.User.ID)
	assert.Equal(t, "comprehensive-repo", pr.Repo.Name)
	assert.Equal(t, "test/comprehensive-repo", pr.Repo.FullName)
	assert.Equal(t, "test", pr.Repo.Owner)
	assert.True(t, pr.Repo.Private)
}

// TestGetContributionsHeatmap_GraphQLResponseParsing tests GraphQL response parsing scenarios
func TestGetContributionsHeatmap_GraphQLResponseParsing(t *testing.T) {
	testCases := []struct {
		name           string
		responseBody   string
		responseStatus int
		expectedError  string
	}{
		{
			name:           "InvalidJSON_DecodeError",
			responseBody:   `{invalid json`,
			responseStatus: http.StatusOK,
			expectedError:  "failed to decode GraphQL response",
		},
		{
			name: "GraphQLError_InResponse",
			responseBody: `{
				"data": null,
				"errors": [
					{
						"message": "Field 'contributionsCollection' doesn't exist on type 'User'",
						"path": ["viewer", "contributionsCollection"]
					}
				]
			}`,
			responseStatus: http.StatusOK,
			expectedError:  "GraphQL error: Field 'contributionsCollection' doesn't exist on type 'User'",
		},
		{
			name: "InvalidDataStructure_UnmarshalError",
			responseBody: `{
				"data": {
					"viewer": {
						"contributionsCollection": "invalid_structure_should_be_object"
					}
				}
			}`,
			responseStatus: http.StatusOK,
			expectedError:  "failed to unmarshal result",
		},
		{
			name: "Success_ValidResponse",
			responseBody: `{
				"data": {
					"viewer": {
						"contributionsCollection": {
							"startedAt": "2024-12-07T00:00:00Z",
							"endedAt": "2025-12-07T23:59:59Z",
							"contributionCalendar": {
								"totalContributions": 150,
								"weeks": [
									{
										"firstDay": "2024-12-01",
										"contributionDays": [
											{
												"date": "2024-12-01",
												"contributionCount": 5,
												"contributionLevel": "SECOND_QUARTILE",
												"color": "#40c463"
											},
											{
												"date": "2024-12-02",
												"contributionCount": 3,
												"contributionLevel": "FIRST_QUARTILE",
												"color": "#9be9a8"
											}
										]
									}
								]
							}
						}
					}
				}
			}`,
			responseStatus: http.StatusOK,
			expectedError:  "", // No error expected
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			// Create mock GraphQL server
			mockGraphQLServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "POST", r.Method)
				assert.Equal(t, "/api/graphql", r.URL.Path)
				assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tc.responseStatus)
				w.Write([]byte(tc.responseBody))
			}))
			defer mockGraphQLServer.Close()

			// Create mock auth service
			mockAuthService := mocks.NewMockGitHubAuthService(ctrl)

			// First call for provider validation
			envConfig := &auth.ProviderConfig{
				ClientID:          "test_client_id",
				ClientSecret:      "test_client_secret",
				EnterpriseBaseURL: mockGraphQLServer.URL,
			}
			githubClient := auth.NewGitHubClient(envConfig)

			mockAuthService.EXPECT().
				GetGitHubClient("githubtools").
				Return(githubClient, nil).
				Times(2) // Called twice: validation + actual use

			mockAuthService.EXPECT().
				GetGitHubAccessToken("test-uuid", "githubtools").
				Return("test-token", nil).
				Times(1)

			// Create GitHub service
			githubService := service.NewGitHubServiceWithAdapter(mockAuthService)

			// Execute test
			result, err := githubService.GetContributionsHeatmap(
				context.Background(),
				"test-uuid",
				"githubtools",
				"30d",
			)

			// Assertions
			if tc.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedError)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
				assert.Equal(t, 150, result.TotalContributions)
				assert.Equal(t, "2024-12-07T00:00:00Z", result.From)
				assert.Equal(t, "2025-12-07T23:59:59Z", result.To)
				assert.Len(t, result.Weeks, 1)
				assert.Equal(t, "2024-12-01", result.Weeks[0].FirstDay)
				assert.Len(t, result.Weeks[0].ContributionDays, 2)
				assert.Equal(t, "2024-12-01", result.Weeks[0].ContributionDays[0].Date)
				assert.Equal(t, 5, result.Weeks[0].ContributionDays[0].ContributionCount)
				assert.Equal(t, "SECOND_QUARTILE", result.Weeks[0].ContributionDays[0].ContributionLevel)
				assert.Equal(t, "#40c463", result.Weeks[0].ContributionDays[0].Color)
			}
		})
	}
}

// TestGetContributionsHeatmap_HTTPErrors tests various HTTP error scenarios
func TestGetContributionsHeatmap_HTTPErrors(t *testing.T) {
	testCases := []struct {
		name           string
		responseStatus int
		responseBody   string
		expectedError  string
	}{
		{
			name:           "RateLimit_403",
			responseStatus: http.StatusForbidden,
			responseBody:   `{"message": "API rate limit exceeded"}`,
			expectedError:  "rate limit exceeded",
		},
		{
			name:           "Unauthorized_401",
			responseStatus: http.StatusUnauthorized,
			responseBody:   `{"message": "Bad credentials"}`,
			expectedError:  "GraphQL query failed with status 401",
		},
		{
			name:           "NotFound_404",
			responseStatus: http.StatusNotFound,
			responseBody:   `{"message": "Not Found"}`,
			expectedError:  "GraphQL query failed with status 404",
		},
		{
			name:           "InternalServerError_500",
			responseStatus: http.StatusInternalServerError,
			responseBody:   `{"message": "Internal Server Error"}`,
			expectedError:  "GraphQL query failed with status 500",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			// Create mock GraphQL server
			mockGraphQLServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tc.responseStatus)
				w.Write([]byte(tc.responseBody))
			}))
			defer mockGraphQLServer.Close()

			// Create mock auth service
			mockAuthService := mocks.NewMockGitHubAuthService(ctrl)

			envConfig := &auth.ProviderConfig{
				EnterpriseBaseURL: mockGraphQLServer.URL,
			}
			githubClient := auth.NewGitHubClient(envConfig)

			mockAuthService.EXPECT().
				GetGitHubClient("githubtools").
				Return(githubClient, nil).
				Times(2)

			mockAuthService.EXPECT().
				GetGitHubAccessToken("test-uuid", "githubtools").
				Return("test-token", nil).
				Times(1)

			// Create GitHub service
			githubService := service.NewGitHubServiceWithAdapter(mockAuthService)

			// Execute test
			result, err := githubService.GetContributionsHeatmap(
				context.Background(),
				"test-uuid",
				"githubtools",
				"30d",
			)

			// Assertions
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.expectedError)
			assert.Nil(t, result)
		})
	}
}

// TestGetContributionsHeatmap_DefaultPeriod tests using default period (no period specified)
func TestGetContributionsHeatmap_DefaultPeriod(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create mock GraphQL server
	mockGraphQLServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the query doesn't contain "from" and "to" parameters for default period
		var requestBody map[string]interface{}
		json.NewDecoder(r.Body).Decode(&requestBody)
		query := requestBody["query"].(string)

		// Default period should not have from/to in the query
		assert.NotContains(t, query, "from:")
		assert.NotContains(t, query, "to:")

		response := `{
			"data": {
				"viewer": {
					"contributionsCollection": {
						"startedAt": "2024-01-01T00:00:00Z",
						"endedAt": "2024-12-31T23:59:59Z",
						"contributionCalendar": {
							"totalContributions": 365,
							"weeks": []
						}
					}
				}
			}
		}`
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(response))
	}))
	defer mockGraphQLServer.Close()

	// Create mock auth service
	mockAuthService := mocks.NewMockGitHubAuthService(ctrl)

	envConfig := &auth.ProviderConfig{
		EnterpriseBaseURL: mockGraphQLServer.URL,
	}
	githubClient := auth.NewGitHubClient(envConfig)

	mockAuthService.EXPECT().
		GetGitHubClient("githubtools").
		Return(githubClient, nil).
		Times(2)

	mockAuthService.EXPECT().
		GetGitHubAccessToken("test-uuid", "githubtools").
		Return("test-token", nil).
		Times(1)

	// Create GitHub service
	githubService := service.NewGitHubServiceWithAdapter(mockAuthService)

	// Execute test with empty period
	result, err := githubService.GetContributionsHeatmap(
		context.Background(),
		"test-uuid",
		"githubtools",
		"", // Empty period - should use GitHub's default
	)

	// Assertions
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 365, result.TotalContributions)
	assert.Equal(t, "2024-01-01T00:00:00Z", result.From)
	assert.Equal(t, "2024-12-31T23:59:59Z", result.To)
}

// TestGetAveragePRMergeTime_Success tests successful scenarios with different PR data
func TestGetAveragePRMergeTime_Success(t *testing.T) {
	testCases := []struct {
		name                    string
		period                  string
		mockPRs                 []map[string]interface{}
		expectedAvgHours        float64
		expectedPRCount         int
		expectedTimeSeriesCount int
	}{
		{
			name:                    "NoPRs_ReturnsZero",
			period:                  "30d",
			mockPRs:                 []map[string]interface{}{},
			expectedAvgHours:        0,
			expectedPRCount:         0,
			expectedTimeSeriesCount: 4, // Always 4 weeks
		},
		{
			name:   "SinglePR_CalculatesCorrectly",
			period: "30d",
			mockPRs: []map[string]interface{}{
				{
					"number":    1,
					"createdAt": "2024-12-01T10:00:00Z",
					"mergedAt":  "2024-12-02T10:00:00Z", // 24 hours later
					"repository": map[string]interface{}{
						"name": "test-repo",
						"owner": map[string]interface{}{
							"login": "test-owner",
						},
					},
				},
			},
			expectedAvgHours:        24.0,
			expectedPRCount:         1,
			expectedTimeSeriesCount: 4,
		},
		{
			name:   "MultiplePRs_CalculatesAverage",
			period: "30d",
			mockPRs: []map[string]interface{}{
				{
					"number":    1,
					"createdAt": "2024-12-01T10:00:00Z",
					"mergedAt":  "2024-12-02T10:00:00Z", // 24 hours
					"repository": map[string]interface{}{
						"name": "test-repo",
						"owner": map[string]interface{}{
							"login": "test-owner",
						},
					},
				},
				{
					"number":    2,
					"createdAt": "2024-12-03T10:00:00Z",
					"mergedAt":  "2024-12-05T10:00:00Z", // 48 hours
					"repository": map[string]interface{}{
						"name": "test-repo",
						"owner": map[string]interface{}{
							"login": "test-owner",
						},
					},
				},
				{
					"number":    3,
					"createdAt": "2024-12-06T10:00:00Z",
					"mergedAt":  "2024-12-07T22:00:00Z", // 36 hours
					"repository": map[string]interface{}{
						"name": "test-repo",
						"owner": map[string]interface{}{
							"login": "test-owner",
						},
					},
				},
			},
			expectedAvgHours:        36.0, // (24 + 48 + 36) / 3 = 36
			expectedPRCount:         3,
			expectedTimeSeriesCount: 4,
		},
		{
			name:   "PRWithMissingMergedAt_Skipped",
			period: "30d",
			mockPRs: []map[string]interface{}{
				{
					"number":    1,
					"createdAt": "2024-12-01T10:00:00Z",
					"mergedAt":  "2024-12-02T10:00:00Z", // 24 hours
					"repository": map[string]interface{}{
						"name": "test-repo",
						"owner": map[string]interface{}{
							"login": "test-owner",
						},
					},
				},
				{
					"number":    2,
					"createdAt": "2024-12-03T10:00:00Z",
					"mergedAt":  "", // Missing - should be skipped
					"repository": map[string]interface{}{
						"name": "test-repo",
						"owner": map[string]interface{}{
							"login": "test-owner",
						},
					},
				},
			},
			expectedAvgHours:        24.0, // Only first PR counted
			expectedPRCount:         1,
			expectedTimeSeriesCount: 4,
		},
		{
			name:   "FastMerge_LessThanOneHour",
			period: "7d",
			mockPRs: []map[string]interface{}{
				{
					"number":    1,
					"createdAt": "2024-12-07T10:00:00Z",
					"mergedAt":  "2024-12-07T10:30:00Z", // 0.5 hours
					"repository": map[string]interface{}{
						"name": "test-repo",
						"owner": map[string]interface{}{
							"login": "test-owner",
						},
					},
				},
			},
			expectedAvgHours:        0.5,
			expectedPRCount:         1,
			expectedTimeSeriesCount: 4,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			// Create mock GraphQL server
			mockGraphQLServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "POST", r.Method)
				assert.Equal(t, "/api/graphql", r.URL.Path)
				assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))

				// Build response with mock PRs
				nodes := make([]interface{}, len(tc.mockPRs))
				for i, pr := range tc.mockPRs {
					nodes[i] = pr
				}

				response := map[string]interface{}{
					"data": map[string]interface{}{
						"search": map[string]interface{}{
							"pageInfo": map[string]interface{}{
								"hasNextPage": false,
								"endCursor":   "",
							},
							"nodes": nodes,
						},
					},
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(response)
			}))
			defer mockGraphQLServer.Close()

			// Create mock auth service
			mockAuthService := mocks.NewMockGitHubAuthService(ctrl)

			envConfig := &auth.ProviderConfig{
				EnterpriseBaseURL: mockGraphQLServer.URL,
			}
			githubClient := auth.NewGitHubClient(envConfig)

			mockAuthService.EXPECT().
				GetGitHubAccessToken("test-uuid", "githubtools").
				Return("test-token", nil)

			mockAuthService.EXPECT().
				GetGitHubClient("githubtools").
				Return(githubClient, nil)

			// Create GitHub service
			githubService := service.NewGitHubServiceWithAdapter(mockAuthService)

			// Execute test
			result, err := githubService.GetAveragePRMergeTime(
				context.Background(),
				"test-uuid",
				"githubtools",
				tc.period,
			)

			// Assertions
			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Equal(t, tc.expectedAvgHours, result.AveragePRMergeTimeHours)
			assert.Equal(t, tc.expectedPRCount, result.PRCount)
			assert.Equal(t, tc.period, result.Period)
			assert.NotEmpty(t, result.From)
			assert.NotEmpty(t, result.To)
			assert.Len(t, result.TimeSeries, tc.expectedTimeSeriesCount)

			// Verify time series structure
			for _, dataPoint := range result.TimeSeries {
				assert.NotEmpty(t, dataPoint.WeekStart)
				assert.NotEmpty(t, dataPoint.WeekEnd)
				assert.GreaterOrEqual(t, dataPoint.AverageHours, 0.0)
				assert.GreaterOrEqual(t, dataPoint.PRCount, 0)
			}
		})
	}
}

// TestGetAveragePRMergeTime_HTTPErrors tests HTTP error scenarios
func TestGetAveragePRMergeTime_HTTPErrors(t *testing.T) {
	testCases := []struct {
		name           string
		responseStatus int
		responseBody   string
		expectedError  string
	}{
		{
			name:           "RateLimit_403",
			responseStatus: http.StatusForbidden,
			responseBody:   `{"message": "API rate limit exceeded"}`,
			expectedError:  "rate limit exceeded",
		},
		{
			name:           "Unauthorized_401",
			responseStatus: http.StatusUnauthorized,
			responseBody:   `{"message": "Bad credentials"}`,
			expectedError:  "GraphQL query failed with status 401",
		},
		{
			name:           "InternalServerError_500",
			responseStatus: http.StatusInternalServerError,
			responseBody:   `{"message": "Internal Server Error"}`,
			expectedError:  "GraphQL query failed with status 500",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			// Create mock GraphQL server
			mockGraphQLServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tc.responseStatus)
				w.Write([]byte(tc.responseBody))
			}))
			defer mockGraphQLServer.Close()

			// Create mock auth service
			mockAuthService := mocks.NewMockGitHubAuthService(ctrl)

			envConfig := &auth.ProviderConfig{
				EnterpriseBaseURL: mockGraphQLServer.URL,
			}
			githubClient := auth.NewGitHubClient(envConfig)

			mockAuthService.EXPECT().
				GetGitHubAccessToken("test-uuid", "githubtools").
				Return("test-token", nil)

			mockAuthService.EXPECT().
				GetGitHubClient("githubtools").
				Return(githubClient, nil)

			// Create GitHub service
			githubService := service.NewGitHubServiceWithAdapter(mockAuthService)

			// Execute test
			result, err := githubService.GetAveragePRMergeTime(
				context.Background(),
				"test-uuid",
				"githubtools",
				"30d",
			)

			// Assertions
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.expectedError)
			assert.Nil(t, result)
		})
	}
}

// TestGetRepositoryContent_Success tests successful file and directory retrieval
func TestGetRepositoryContent_Success(t *testing.T) {
	testCases := []struct {
		name             string
		path             string
		ref              string
		mockResponseFile bool // true for file, false for directory
		mockContent      string
		expectedType     string
	}{
		{
			name:             "FileContent_README",
			path:             "README.md",
			ref:              "main",
			mockResponseFile: true,
			mockContent:      "# Test Repository\n\nThis is a test README file.",
			expectedType:     "file",
		},
		{
			name:             "FileContent_NestedFile",
			path:             "src/main.go",
			ref:              "develop",
			mockResponseFile: true,
			mockContent:      "package main\n\nfunc main() {\n\tprintln(\"Hello\")\n}",
			expectedType:     "file",
		},
		{
			name:             "DirectoryContent_Root",
			path:             "",
			ref:              "main",
			mockResponseFile: false,
			mockContent:      "",
			expectedType:     "dir",
		},
		{
			name:             "DirectoryContent_Nested",
			path:             "src",
			ref:              "main",
			mockResponseFile: false,
			mockContent:      "",
			expectedType:     "dir",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			// Create mock GitHub API server
			var mockGitHubServer *httptest.Server
			mockGitHubServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "GET", r.Method)
				assert.Contains(t, r.URL.Path, "/repos/owner/repo/contents")

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)

				if tc.mockResponseFile {
					// Mock file response
					response := map[string]interface{}{
						"name":         "test-file.txt",
						"path":         tc.path,
						"sha":          "abc123",
						"size":         len(tc.mockContent),
						"url":          mockGitHubServer.URL + "/repos/owner/repo/contents/" + tc.path,
						"html_url":     mockGitHubServer.URL + "/owner/repo/blob/main/" + tc.path,
						"git_url":      mockGitHubServer.URL + "/repos/owner/repo/git/blobs/abc123",
						"download_url": mockGitHubServer.URL + "/owner/repo/raw/main/" + tc.path,
						"type":         "file",
						"content":      base64.StdEncoding.EncodeToString([]byte(tc.mockContent)),
						"encoding":     "base64",
						"_links": map[string]string{
							"self": mockGitHubServer.URL + "/repos/owner/repo/contents/" + tc.path,
							"git":  mockGitHubServer.URL + "/repos/owner/repo/git/blobs/abc123",
							"html": mockGitHubServer.URL + "/owner/repo/blob/main/" + tc.path,
						},
					}
					json.NewEncoder(w).Encode(response)
				} else {
					// Mock directory response
					response := []map[string]interface{}{
						{
							"name":         "file1.txt",
							"path":         tc.path + "/file1.txt",
							"sha":          "def456",
							"size":         100,
							"url":          mockGitHubServer.URL + "/repos/owner/repo/contents/" + tc.path + "/file1.txt",
							"html_url":     mockGitHubServer.URL + "/owner/repo/blob/main/" + tc.path + "/file1.txt",
							"git_url":      mockGitHubServer.URL + "/repos/owner/repo/git/blobs/def456",
							"download_url": mockGitHubServer.URL + "/owner/repo/raw/main/" + tc.path + "/file1.txt",
							"type":         "file",
							"_links": map[string]string{
								"self": mockGitHubServer.URL + "/repos/owner/repo/contents/" + tc.path + "/file1.txt",
								"git":  mockGitHubServer.URL + "/repos/owner/repo/git/blobs/def456",
								"html": mockGitHubServer.URL + "/owner/repo/blob/main/" + tc.path + "/file1.txt",
							},
						},
						{
							"name":         "subdir",
							"path":         tc.path + "/subdir",
							"sha":          "ghi789",
							"size":         0,
							"url":          mockGitHubServer.URL + "/repos/owner/repo/contents/" + tc.path + "/subdir",
							"html_url":     mockGitHubServer.URL + "/owner/repo/tree/main/" + tc.path + "/subdir",
							"git_url":      mockGitHubServer.URL + "/repos/owner/repo/git/trees/ghi789",
							"download_url": nil,
							"type":         "dir",
							"_links": map[string]string{
								"self": mockGitHubServer.URL + "/repos/owner/repo/contents/" + tc.path + "/subdir",
								"git":  mockGitHubServer.URL + "/repos/owner/repo/git/trees/ghi789",
								"html": mockGitHubServer.URL + "/owner/repo/tree/main/" + tc.path + "/subdir",
							},
						},
					}
					json.NewEncoder(w).Encode(response)
				}
			}))
			defer mockGitHubServer.Close()

			// Create mock auth service
			mockAuthService := mocks.NewMockGitHubAuthService(ctrl)

			envConfig := &auth.ProviderConfig{
				EnterpriseBaseURL: mockGitHubServer.URL,
			}
			githubClient := auth.NewGitHubClient(envConfig)

			mockAuthService.EXPECT().
				GetGitHubAccessToken("test-uuid", "githubtools").
				Return("test-token", nil)

			mockAuthService.EXPECT().
				GetGitHubClient("githubtools").
				Return(githubClient, nil)

			// Create GitHub service
			githubService := service.NewGitHubServiceWithAdapter(mockAuthService)

			// Execute test
			result, err := githubService.GetRepositoryContent(
				context.Background(),
				"test-uuid",
				"githubtools",
				"owner",
				"repo",
				tc.path,
				tc.ref,
			)

			// Assertions
			require.NoError(t, err)
			require.NotNil(t, result)

			if tc.mockResponseFile {
				// Verify file response
				fileResult, ok := result.(map[string]interface{})
				require.True(t, ok, "Expected file result to be a map")
				assert.Equal(t, "file", fileResult["type"])
				assert.Equal(t, tc.path, fileResult["path"])
				assert.NotEmpty(t, fileResult["content"])
				assert.Equal(t, tc.mockContent, fileResult["content"])
				assert.Equal(t, "base64", fileResult["encoding"])
				assert.NotEmpty(t, fileResult["sha"])
			} else {
				// Verify directory response
				dirResult, ok := result.([]map[string]interface{})
				require.True(t, ok, "Expected directory result to be an array")
				assert.Len(t, dirResult, 2) // file1.txt and subdir

				// Check first item (file)
				assert.Equal(t, "file1.txt", dirResult[0]["name"])
				assert.Equal(t, "file", dirResult[0]["type"])

				// Check second item (directory)
				assert.Equal(t, "subdir", dirResult[1]["name"])
				assert.Equal(t, "dir", dirResult[1]["type"])
			}
		})
	}
}

// TestUpdateRepositoryFile_Success tests successful file update scenarios
func TestUpdateRepositoryFile_Success(t *testing.T) {
	testCases := []struct {
		name           string
		path           string
		message        string
		content        string
		sha            string
		branch         string
		expectedCommit string
	}{
		{
			name:           "UpdateRootFile",
			path:           "README.md",
			message:        "Update README",
			content:        "# Updated README\n\nNew content here.",
			sha:            "abc123",
			branch:         "main",
			expectedCommit: "commit-sha-123",
		},
		{
			name:           "UpdateNestedFile",
			path:           "src/main.go",
			message:        "Fix bug in main.go",
			content:        "package main\n\nfunc main() {\n\t// Fixed\n}",
			sha:            "def456",
			branch:         "develop",
			expectedCommit: "commit-sha-456",
		},
		{
			name:           "UpdateOnFeatureBranch",
			path:           "config.yml",
			message:        "Update configuration",
			content:        "key: value\nother: setting",
			sha:            "ghi789",
			branch:         "feature/new-config",
			expectedCommit: "commit-sha-789",
		},
		{
			name:           "UpdateWithoutBranch_DefaultsToMain",
			path:           "default.txt",
			message:        "Update without specifying branch",
			content:        "Content without branch specified",
			sha:            "jkl012",
			branch:         "", // Empty branch - should default to main
			expectedCommit: "commit-sha-default",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			// Create mock GitHub API server
			var mockGitHubServer *httptest.Server
			mockGitHubServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "PUT", r.Method)
				assert.Contains(t, r.URL.Path, "/repos/owner/repo/contents/"+tc.path)

				// Verify request body
				var requestBody map[string]interface{}
				err := json.NewDecoder(r.Body).Decode(&requestBody)
				assert.NoError(t, err)

				// Verify commit message
				assert.Equal(t, tc.message, requestBody["message"])

				// Verify content is base64 encoded
				assert.NotEmpty(t, requestBody["content"])
				decodedContent, err := base64.StdEncoding.DecodeString(requestBody["content"].(string))
				assert.NoError(t, err)
				assert.Equal(t, tc.content, string(decodedContent))

				// Verify SHA
				assert.Equal(t, tc.sha, requestBody["sha"])

				// Verify branch if provided
				if tc.branch != "" {
					assert.Equal(t, tc.branch, requestBody["branch"])
				}

				// Return successful update response
				response := map[string]interface{}{
					"content": map[string]interface{}{
						"name":     tc.path,
						"path":     tc.path,
						"sha":      "new-sha-after-update",
						"size":     len(tc.content),
						"url":      mockGitHubServer.URL + "/repos/owner/repo/contents/" + tc.path,
						"html_url": mockGitHubServer.URL + "/owner/repo/blob/" + tc.branch + "/" + tc.path,
						"git_url":  mockGitHubServer.URL + "/repos/owner/repo/git/blobs/new-sha",
						"type":     "file",
					},
					"commit": map[string]interface{}{
						"sha":     tc.expectedCommit,
						"message": tc.message,
						"author": map[string]interface{}{
							"name":  "Test User",
							"email": "test@example.com",
							"date":  "2025-12-07T12:00:00Z",
						},
						"committer": map[string]interface{}{
							"name":  "Test User",
							"email": "test@example.com",
							"date":  "2025-12-07T12:00:00Z",
						},
					},
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(response)
			}))
			defer mockGitHubServer.Close()

			// Create mock auth service
			mockAuthService := mocks.NewMockGitHubAuthService(ctrl)

			envConfig := &auth.ProviderConfig{
				EnterpriseBaseURL: mockGitHubServer.URL,
			}
			githubClient := auth.NewGitHubClient(envConfig)

			mockAuthService.EXPECT().
				GetGitHubAccessToken("test-uuid", "githubtools").
				Return("test-token", nil)

			mockAuthService.EXPECT().
				GetGitHubClient("githubtools").
				Return(githubClient, nil)

			// Create GitHub service
			githubService := service.NewGitHubServiceWithAdapter(mockAuthService)

			// Execute test
			result, err := githubService.UpdateRepositoryFile(
				context.Background(),
				"test-uuid",
				"githubtools",
				"owner",
				"repo",
				tc.path,
				tc.message,
				tc.content,
				tc.sha,
				tc.branch,
			)

			// Assertions
			require.NoError(t, err)
			require.NotNil(t, result)

			// Type assert result to map
			resultMap, ok := result.(map[string]interface{})
			require.True(t, ok, "Expected result to be a map")

			// Verify commit information
			commit, ok := resultMap["commit"].(map[string]interface{})
			require.True(t, ok, "Expected commit to be a map")
			assert.Equal(t, tc.expectedCommit, commit["sha"])
			assert.Equal(t, tc.message, commit["message"])

			// Verify content information
			content, ok := resultMap["content"].(map[string]interface{})
			require.True(t, ok, "Expected content to be a map")
			assert.Equal(t, tc.path, content["path"])
			assert.Equal(t, "file", content["type"])
			assert.Equal(t, "new-sha-after-update", content["sha"])
		})
	}
}

// TestGetAveragePRMergeTime_GraphQLErrors tests GraphQL-specific errors
func TestGetAveragePRMergeTime_GraphQLErrors(t *testing.T) {
	testCases := []struct {
		name          string
		responseBody  string
		expectedError string
	}{
		{
			name: "GraphQLError_InResponse",
			responseBody: `{
				"data": null,
				"errors": [
					{
						"message": "Field 'search' doesn't exist on type 'Query'",
						"path": ["search"]
					}
				]
			}`,
			expectedError: "GraphQL error: Field 'search' doesn't exist on type 'Query'",
		},
		{
			name:          "InvalidJSON_DecodeError",
			responseBody:  `{invalid json`,
			expectedError: "failed to decode GraphQL response",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			// Create mock GraphQL server
			mockGraphQLServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(tc.responseBody))
			}))
			defer mockGraphQLServer.Close()

			// Create mock auth service
			mockAuthService := mocks.NewMockGitHubAuthService(ctrl)

			envConfig := &auth.ProviderConfig{
				EnterpriseBaseURL: mockGraphQLServer.URL,
			}
			githubClient := auth.NewGitHubClient(envConfig)

			mockAuthService.EXPECT().
				GetGitHubAccessToken("test-uuid", "githubtools").
				Return("test-token", nil)

			mockAuthService.EXPECT().
				GetGitHubClient("githubtools").
				Return(githubClient, nil)

			// Create GitHub service
			githubService := service.NewGitHubServiceWithAdapter(mockAuthService)

			// Execute test
			result, err := githubService.GetAveragePRMergeTime(
				context.Background(),
				"test-uuid",
				"githubtools",
				"30d",
			)

			// Assertions
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.expectedError)
			assert.Nil(t, result)
		})
	}
}

// TestGetGitHubAsset_Success tests successful asset retrieval
func TestGetGitHubAsset_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAssetData := []byte("fake-image-data-png")

	mockAssetServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "token test-token", r.Header.Get("Authorization"))

		w.Header().Set("Content-Type", "image/png")
		w.WriteHeader(http.StatusOK)
		w.Write(mockAssetData)
	}))
	defer mockAssetServer.Close()

	mockAuthService := mocks.NewMockGitHubAuthService(ctrl)
	mockAuthService.EXPECT().GetGitHubAccessToken("test-uuid", "githubtools").Return("test-token", nil)

	githubService := service.NewGitHubServiceWithAdapter(mockAuthService)

	assetData, contentType, err := githubService.GetGitHubAsset(
		context.Background(),
		"test-uuid",
		"githubtools",
		mockAssetServer.URL+"/asset.png",
	)

	require.NoError(t, err)
	assert.Equal(t, mockAssetData, assetData)
	assert.Equal(t, "image/png", contentType)
}

// TestGetGitHubAsset_401Unauthorized_RetryWithBearer tests 401 response with retry logic
func TestGetGitHubAsset_401Unauthorized_RetryWithBearer(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAssetData := []byte("fake-image-data")
	callCount := 0

	mockAssetServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		authHeader := r.Header.Get("Authorization")

		if callCount == 1 {
			assert.Equal(t, "token test-token", authHeader)
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"message": "Bad credentials"}`))
		} else {
			assert.Equal(t, "Bearer test-token", authHeader)
			w.Header().Set("Content-Type", "image/jpeg")
			w.WriteHeader(http.StatusOK)
			w.Write(mockAssetData)
		}
	}))
	defer mockAssetServer.Close()

	mockAuthService := mocks.NewMockGitHubAuthService(ctrl)
	mockAuthService.EXPECT().GetGitHubAccessToken("test-uuid", "githubtools").Return("test-token", nil)

	githubService := service.NewGitHubServiceWithAdapter(mockAuthService)

	assetData, contentType, err := githubService.GetGitHubAsset(
		context.Background(),
		"test-uuid",
		"githubtools",
		mockAssetServer.URL+"/asset.jpg",
	)

	require.NoError(t, err)
	assert.Equal(t, mockAssetData, assetData)
	assert.Equal(t, "image/jpeg", contentType)
	assert.Equal(t, 2, callCount)
}

// TestGetGitHubAsset_403RateLimit tests 403 rate limit response
func TestGetGitHubAsset_403RateLimit(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAssetServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"message": "API rate limit exceeded"}`))
	}))
	defer mockAssetServer.Close()

	mockAuthService := mocks.NewMockGitHubAuthService(ctrl)
	mockAuthService.EXPECT().GetGitHubAccessToken("test-uuid", "githubtools").Return("test-token", nil)

	githubService := service.NewGitHubServiceWithAdapter(mockAuthService)

	assetData, contentType, err := githubService.GetGitHubAsset(
		context.Background(),
		"test-uuid",
		"githubtools",
		mockAssetServer.URL+"/asset.png",
	)

	require.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrGitHubAPIRateLimitExceeded)
	assert.Nil(t, assetData)
	assert.Empty(t, contentType)
}

// TestGetGitHubAsset_404NotFound tests 404 not found response
func TestGetGitHubAsset_404NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAssetServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"message": "Not Found"}`))
	}))
	defer mockAssetServer.Close()

	mockAuthService := mocks.NewMockGitHubAuthService(ctrl)
	mockAuthService.EXPECT().GetGitHubAccessToken("test-uuid", "githubtools").Return("test-token", nil)

	githubService := service.NewGitHubServiceWithAdapter(mockAuthService)

	assetData, contentType, err := githubService.GetGitHubAsset(
		context.Background(),
		"test-uuid",
		"githubtools",
		mockAssetServer.URL+"/nonexistent.png",
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "GitHub asset not found")
	assert.Nil(t, assetData)
	assert.Empty(t, contentType)
}

// TestClosePullRequest_Success_OpenPR tests successfully closing an open PR
func TestClosePullRequest_Success_OpenPR(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	prNumber := 42
	owner := "test-owner"
	repo := "test-repo"

	var getPRCalled, editPRCalled bool

	mockGitHubServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.Method == "GET" && r.URL.Path == fmt.Sprintf("/api/v3/repos/%s/%s/pulls/%d", owner, repo, prNumber) {
			getPRCalled = true
			response := map[string]interface{}{
				"id":         int64(123456789),
				"number":     prNumber,
				"title":      "Test PR to close",
				"state":      "open",
				"created_at": "2025-01-01T10:00:00Z",
				"updated_at": "2025-01-02T10:00:00Z",
				"html_url":   "https://github.com/" + owner + "/" + repo + "/pull/" + fmt.Sprintf("%d", prNumber),
				"draft":      false,
				"user": map[string]interface{}{
					"login":      "testuser",
					"id":         int64(12345),
					"avatar_url": "https://avatars.githubusercontent.com/u/12345",
				},
				"head": map[string]interface{}{
					"ref": "feature-branch",
					"repo": map[string]interface{}{
						"name": repo,
						"owner": map[string]interface{}{
							"login": owner,
						},
					},
				},
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(response)
			return
		}

		if r.Method == "PATCH" && r.URL.Path == fmt.Sprintf("/api/v3/repos/%s/%s/pulls/%d", owner, repo, prNumber) {
			editPRCalled = true
			var requestBody map[string]interface{}
			json.NewDecoder(r.Body).Decode(&requestBody)
			assert.Equal(t, "closed", requestBody["state"])

			response := map[string]interface{}{
				"id":         int64(123456789),
				"number":     prNumber,
				"title":      "Test PR to close",
				"state":      "closed",
				"created_at": "2025-01-01T10:00:00Z",
				"updated_at": "2025-01-02T12:00:00Z",
				"html_url":   "https://github.com/" + owner + "/" + repo + "/pull/" + fmt.Sprintf("%d", prNumber),
				"draft":      false,
				"user": map[string]interface{}{
					"login":      "testuser",
					"id":         int64(12345),
					"avatar_url": "https://avatars.githubusercontent.com/u/12345",
				},
				"head": map[string]interface{}{
					"ref": "feature-branch",
					"repo": map[string]interface{}{
						"name": repo,
						"owner": map[string]interface{}{
							"login": owner,
						},
					},
				},
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(response)
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer mockGitHubServer.Close()

	mockAuthService := mocks.NewMockGitHubAuthService(ctrl)
	mockAuthService.EXPECT().
		GetGitHubAccessToken("test-uuid", "githubtools").
		Return("test-token", nil)

	envConfig := &auth.ProviderConfig{
		EnterpriseBaseURL: mockGitHubServer.URL,
	}
	githubClient := auth.NewGitHubClient(envConfig)
	mockAuthService.EXPECT().
		GetGitHubClient("githubtools").
		Return(githubClient, nil)

	githubService := service.NewGitHubServiceWithAdapter(mockAuthService)

	result, err := githubService.ClosePullRequest(
		context.Background(),
		"test-uuid",
		"githubtools",
		owner,
		repo,
		prNumber,
		false,
	)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, getPRCalled)
	assert.True(t, editPRCalled)
	assert.Equal(t, int64(123456789), result.ID)
	assert.Equal(t, prNumber, result.Number)
	assert.Equal(t, "Test PR to close", result.Title)
	assert.Equal(t, "closed", result.State)
	assert.False(t, result.Draft)
	assert.Equal(t, "testuser", result.User.Login)
}

// TestClosePullRequest_Success_DeleteBranch_SameRepo tests closing PR and deleting branch
func TestClosePullRequest_Success_DeleteBranch_SameRepo(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	prNumber := 99
	owner := "test-owner"
	repo := "test-repo"
	branchName := "feature-to-delete"

	var getPRCalled, editPRCalled, deleteBranchCalled bool

	mockGitHubServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.Method == "GET" && r.URL.Path == fmt.Sprintf("/api/v3/repos/%s/%s/pulls/%d", owner, repo, prNumber) {
			getPRCalled = true
			response := map[string]interface{}{
				"id":         int64(999),
				"number":     prNumber,
				"title":      "PR with branch to delete",
				"state":      "open",
				"created_at": "2025-01-01T10:00:00Z",
				"updated_at": "2025-01-02T10:00:00Z",
				"html_url":   "https://github.com/" + owner + "/" + repo + "/pull/" + fmt.Sprintf("%d", prNumber),
				"draft":      false,
				"user": map[string]interface{}{
					"login":      "contributor",
					"id":         int64(54321),
					"avatar_url": "https://avatars.githubusercontent.com/u/54321",
				},
				"head": map[string]interface{}{
					"ref": branchName,
					"repo": map[string]interface{}{
						"name": repo,
						"owner": map[string]interface{}{
							"login": owner,
						},
					},
				},
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(response)
			return
		}

		if r.Method == "PATCH" && r.URL.Path == fmt.Sprintf("/api/v3/repos/%s/%s/pulls/%d", owner, repo, prNumber) {
			editPRCalled = true
			response := map[string]interface{}{
				"id":         int64(999),
				"number":     prNumber,
				"title":      "PR with branch to delete",
				"state":      "closed",
				"created_at": "2025-01-01T10:00:00Z",
				"updated_at": "2025-01-02T12:00:00Z",
				"html_url":   "https://github.com/" + owner + "/" + repo + "/pull/" + fmt.Sprintf("%d", prNumber),
				"draft":      false,
				"user": map[string]interface{}{
					"login":      "contributor",
					"id":         int64(54321),
					"avatar_url": "https://avatars.githubusercontent.com/u/54321",
				},
				"head": map[string]interface{}{
					"ref": branchName,
					"repo": map[string]interface{}{
						"name": repo,
						"owner": map[string]interface{}{
							"login": owner,
						},
					},
				},
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(response)
			return
		}

		if r.Method == "DELETE" && r.URL.Path == fmt.Sprintf("/api/v3/repos/%s/%s/git/refs/heads/%s", owner, repo, branchName) {
			deleteBranchCalled = true
			w.WriteHeader(http.StatusNoContent)
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer mockGitHubServer.Close()

	mockAuthService := mocks.NewMockGitHubAuthService(ctrl)
	mockAuthService.EXPECT().
		GetGitHubAccessToken("test-uuid", "githubtools").
		Return("test-token", nil)

	envConfig := &auth.ProviderConfig{
		EnterpriseBaseURL: mockGitHubServer.URL,
	}
	githubClient := auth.NewGitHubClient(envConfig)
	mockAuthService.EXPECT().
		GetGitHubClient("githubtools").
		Return(githubClient, nil)

	githubService := service.NewGitHubServiceWithAdapter(mockAuthService)

	result, err := githubService.ClosePullRequest(
		context.Background(),
		"test-uuid",
		"githubtools",
		owner,
		repo,
		prNumber,
		true,
	)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, getPRCalled)
	assert.True(t, editPRCalled)
	assert.True(t, deleteBranchCalled)
	assert.Equal(t, int64(999), result.ID)
	assert.Equal(t, "closed", result.State)
}

// TestClosePullRequest_Success_DeleteBranch_Fork tests closing PR from fork
func TestClosePullRequest_Success_DeleteBranch_Fork(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	prNumber := 55
	baseOwner := "upstream-owner"
	baseRepo := "upstream-repo"
	forkOwner := "fork-owner"
	forkRepo := "fork-repo"
	branchName := "fork-feature"

	var deleteBranchCalled bool

	mockGitHubServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.Method == "GET" && r.URL.Path == fmt.Sprintf("/api/v3/repos/%s/%s/pulls/%d", baseOwner, baseRepo, prNumber) {
			response := map[string]interface{}{
				"id":         int64(555),
				"number":     prNumber,
				"title":      "PR from fork",
				"state":      "open",
				"created_at": "2025-01-01T10:00:00Z",
				"updated_at": "2025-01-02T10:00:00Z",
				"html_url":   "https://github.com/" + baseOwner + "/" + baseRepo + "/pull/" + fmt.Sprintf("%d", prNumber),
				"draft":      false,
				"user": map[string]interface{}{
					"login":      "fork-contributor",
					"id":         int64(99999),
					"avatar_url": "https://avatars.githubusercontent.com/u/99999",
				},
				"head": map[string]interface{}{
					"ref": branchName,
					"repo": map[string]interface{}{
						"name": forkRepo,
						"owner": map[string]interface{}{
							"login": forkOwner,
						},
					},
				},
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(response)
			return
		}

		if r.Method == "PATCH" && r.URL.Path == fmt.Sprintf("/api/v3/repos/%s/%s/pulls/%d", baseOwner, baseRepo, prNumber) {
			response := map[string]interface{}{
				"id":         int64(555),
				"number":     prNumber,
				"title":      "PR from fork",
				"state":      "closed",
				"created_at": "2025-01-01T10:00:00Z",
				"updated_at": "2025-01-02T12:00:00Z",
				"html_url":   "https://github.com/" + baseOwner + "/" + baseRepo + "/pull/" + fmt.Sprintf("%d", prNumber),
				"draft":      false,
				"user": map[string]interface{}{
					"login":      "fork-contributor",
					"id":         int64(99999),
					"avatar_url": "https://avatars.githubusercontent.com/u/99999",
				},
				"head": map[string]interface{}{
					"ref": branchName,
					"repo": map[string]interface{}{
						"name": forkRepo,
						"owner": map[string]interface{}{
							"login": forkOwner,
						},
					},
				},
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(response)
			return
		}

		if r.Method == "DELETE" && r.URL.Path == fmt.Sprintf("/api/v3/repos/%s/%s/git/refs/heads/%s", forkOwner, forkRepo, branchName) {
			deleteBranchCalled = true
			w.WriteHeader(http.StatusNoContent)
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer mockGitHubServer.Close()

	mockAuthService := mocks.NewMockGitHubAuthService(ctrl)
	mockAuthService.EXPECT().
		GetGitHubAccessToken("test-uuid", "githubtools").
		Return("test-token", nil)

	envConfig := &auth.ProviderConfig{
		EnterpriseBaseURL: mockGitHubServer.URL,
	}
	githubClient := auth.NewGitHubClient(envConfig)
	mockAuthService.EXPECT().
		GetGitHubClient("githubtools").
		Return(githubClient, nil)

	githubService := service.NewGitHubServiceWithAdapter(mockAuthService)

	result, err := githubService.ClosePullRequest(
		context.Background(),
		"test-uuid",
		"githubtools",
		baseOwner,
		baseRepo,
		prNumber,
		true,
	)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, deleteBranchCalled)
	assert.Equal(t, int64(555), result.ID)
	assert.Equal(t, "closed", result.State)
}

// TestClosePullRequest_AlreadyClosed tests attempting to close already closed PR
func TestClosePullRequest_AlreadyClosed(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	prNumber := 10
	owner := "test-owner"
	repo := "test-repo"

	mockGitHubServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.Method == "GET" && r.URL.Path == fmt.Sprintf("/api/v3/repos/%s/%s/pulls/%d", owner, repo, prNumber) {
			response := map[string]interface{}{
				"id":         int64(100),
				"number":     prNumber,
				"title":      "Already closed PR",
				"state":      "closed",
				"created_at": "2025-01-01T10:00:00Z",
				"updated_at": "2025-01-02T10:00:00Z",
				"html_url":   "https://github.com/" + owner + "/" + repo + "/pull/" + fmt.Sprintf("%d", prNumber),
				"draft":      false,
				"user": map[string]interface{}{
					"login":      "testuser",
					"id":         int64(12345),
					"avatar_url": "https://avatars.githubusercontent.com/u/12345",
				},
				"head": map[string]interface{}{
					"ref": "old-branch",
					"repo": map[string]interface{}{
						"name": repo,
						"owner": map[string]interface{}{
							"login": owner,
						},
					},
				},
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(response)
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer mockGitHubServer.Close()

	mockAuthService := mocks.NewMockGitHubAuthService(ctrl)
	mockAuthService.EXPECT().
		GetGitHubAccessToken("test-uuid", "githubtools").
		Return("test-token", nil)

	envConfig := &auth.ProviderConfig{
		EnterpriseBaseURL: mockGitHubServer.URL,
	}
	githubClient := auth.NewGitHubClient(envConfig)
	mockAuthService.EXPECT().
		GetGitHubClient("githubtools").
		Return(githubClient, nil)

	githubService := service.NewGitHubServiceWithAdapter(mockAuthService)

	result, err := githubService.ClosePullRequest(
		context.Background(),
		"test-uuid",
		"githubtools",
		owner,
		repo,
		prNumber,
		false,
	)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorIs(t, err, apperrors.ErrInvalidStatus)
	assert.Contains(t, err.Error(), "already closed")
}

// TestClosePullRequest_PRNotFound tests when PR doesn't exist
func TestClosePullRequest_PRNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	prNumber := 999
	owner := "test-owner"
	repo := "test-repo"

	mockGitHubServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.Method == "GET" && r.URL.Path == fmt.Sprintf("/api/v3/repos/%s/%s/pulls/%d", owner, repo, prNumber) {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{
				"message": "Not Found",
			})
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer mockGitHubServer.Close()

	mockAuthService := mocks.NewMockGitHubAuthService(ctrl)
	mockAuthService.EXPECT().
		GetGitHubAccessToken("test-uuid", "githubtools").
		Return("test-token", nil)

	envConfig := &auth.ProviderConfig{
		EnterpriseBaseURL: mockGitHubServer.URL,
	}
	githubClient := auth.NewGitHubClient(envConfig)
	mockAuthService.EXPECT().
		GetGitHubClient("githubtools").
		Return(githubClient, nil)

	githubService := service.NewGitHubServiceWithAdapter(mockAuthService)

	result, err := githubService.ClosePullRequest(
		context.Background(),
		"test-uuid",
		"githubtools",
		owner,
		repo,
		prNumber,
		false,
	)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "not found")
}

// TestClosePullRequest_RateLimitExceeded tests rate limit handling
func TestClosePullRequest_RateLimitExceeded(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	prNumber := 42
	owner := "test-owner"
	repo := "test-repo"

	mockGitHubServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.Method == "GET" && r.URL.Path == fmt.Sprintf("/api/v3/repos/%s/%s/pulls/%d", owner, repo, prNumber) {
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(map[string]string{
				"message": "API rate limit exceeded",
			})
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer mockGitHubServer.Close()

	mockAuthService := mocks.NewMockGitHubAuthService(ctrl)
	mockAuthService.EXPECT().
		GetGitHubAccessToken("test-uuid", "githubtools").
		Return("test-token", nil)

	envConfig := &auth.ProviderConfig{
		EnterpriseBaseURL: mockGitHubServer.URL,
	}
	githubClient := auth.NewGitHubClient(envConfig)
	mockAuthService.EXPECT().
		GetGitHubClient("githubtools").
		Return(githubClient, nil)

	githubService := service.NewGitHubServiceWithAdapter(mockAuthService)

	result, err := githubService.ClosePullRequest(
		context.Background(),
		"test-uuid",
		"githubtools",
		owner,
		repo,
		prNumber,
		false,
	)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorIs(t, err, apperrors.ErrGitHubAPIRateLimitExceeded)
}

// TestClosePullRequest_DeleteBranch_BranchAlreadyDeleted tests when branch is already deleted
func TestClosePullRequest_DeleteBranch_BranchAlreadyDeleted(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	prNumber := 77
	owner := "test-owner"
	repo := "test-repo"
	branchName := "already-deleted-branch"

	var deleteBranchCalled bool

	mockGitHubServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.Method == "GET" && r.URL.Path == fmt.Sprintf("/api/v3/repos/%s/%s/pulls/%d", owner, repo, prNumber) {
			response := map[string]interface{}{
				"id":         int64(777),
				"number":     prNumber,
				"title":      "PR with already deleted branch",
				"state":      "open",
				"created_at": "2025-01-01T10:00:00Z",
				"updated_at": "2025-01-02T10:00:00Z",
				"html_url":   "https://github.com/" + owner + "/" + repo + "/pull/" + fmt.Sprintf("%d", prNumber),
				"draft":      false,
				"user": map[string]interface{}{
					"login":      "testuser",
					"id":         int64(12345),
					"avatar_url": "https://avatars.githubusercontent.com/u/12345",
				},
				"head": map[string]interface{}{
					"ref": branchName,
					"repo": map[string]interface{}{
						"name": repo,
						"owner": map[string]interface{}{
							"login": owner,
						},
					},
				},
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(response)
			return
		}

		if r.Method == "PATCH" && r.URL.Path == fmt.Sprintf("/api/v3/repos/%s/%s/pulls/%d", owner, repo, prNumber) {
			response := map[string]interface{}{
				"id":         int64(777),
				"number":     prNumber,
				"title":      "PR with already deleted branch",
				"state":      "closed",
				"created_at": "2025-01-01T10:00:00Z",
				"updated_at": "2025-01-02T12:00:00Z",
				"html_url":   "https://github.com/" + owner + "/" + repo + "/pull/" + fmt.Sprintf("%d", prNumber),
				"draft":      false,
				"user": map[string]interface{}{
					"login":      "testuser",
					"id":         int64(12345),
					"avatar_url": "https://avatars.githubusercontent.com/u/12345",
				},
				"head": map[string]interface{}{
					"ref": branchName,
					"repo": map[string]interface{}{
						"name": repo,
						"owner": map[string]interface{}{
							"login": owner,
						},
					},
				},
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(response)
			return
		}

		if r.Method == "DELETE" && r.URL.Path == fmt.Sprintf("/api/v3/repos/%s/%s/git/refs/heads/%s", owner, repo, branchName) {
			deleteBranchCalled = true
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{
				"message": "Reference does not exist",
			})
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer mockGitHubServer.Close()

	mockAuthService := mocks.NewMockGitHubAuthService(ctrl)
	mockAuthService.EXPECT().
		GetGitHubAccessToken("test-uuid", "githubtools").
		Return("test-token", nil)

	envConfig := &auth.ProviderConfig{
		EnterpriseBaseURL: mockGitHubServer.URL,
	}
	githubClient := auth.NewGitHubClient(envConfig)
	mockAuthService.EXPECT().
		GetGitHubClient("githubtools").
		Return(githubClient, nil)

	githubService := service.NewGitHubServiceWithAdapter(mockAuthService)

	result, err := githubService.ClosePullRequest(
		context.Background(),
		"test-uuid",
		"githubtools",
		owner,
		repo,
		prNumber,
		true,
	)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, deleteBranchCalled)
	assert.Equal(t, int64(777), result.ID)
	assert.Equal(t, "closed", result.State)
}

// TestGetUserPRReviewComments_Success tests successful scenarios with different comment counts
func TestGetUserPRReviewComments_Success(t *testing.T) {
	testCases := []struct {
		name                  string
		period                string
		mockTotalComments     int
		expectedTotalComments int
	}{
		{
			name:                  "NoComments_ReturnsZero",
			period:                "30d",
			mockTotalComments:     0,
			expectedTotalComments: 0,
		},
		{
			name:                  "SingleComment",
			period:                "7d",
			mockTotalComments:     1,
			expectedTotalComments: 1,
		},
		{
			name:                  "MultipleComments",
			period:                "30d",
			mockTotalComments:     25,
			expectedTotalComments: 25,
		},
		{
			name:                  "LargeNumberOfComments",
			period:                "90d",
			mockTotalComments:     150,
			expectedTotalComments: 150,
		},
		{
			name:                  "DefaultPeriod_30Days",
			period:                "",
			mockTotalComments:     42,
			expectedTotalComments: 42,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			// Create mock GitHub API server
			var mockGitHubServer *httptest.Server
			mockGitHubServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")

				// Handle GET /user request
				if r.Method == "GET" && r.URL.Path == "/api/v3/user" {
					response := map[string]interface{}{
						"login":      "testuser",
						"id":         int64(12345),
						"avatar_url": "https://avatars.githubusercontent.com/u/12345",
						"name":       "Test User",
						"email":      "test@example.com",
					}
					w.WriteHeader(http.StatusOK)
					json.NewEncoder(w).Encode(response)
					return
				}

				// Handle search issues request
				if r.Method == "GET" && r.URL.Path == "/api/v3/search/issues" {
					assert.Contains(t, r.URL.RawQuery, "type%3Apr")
					assert.Contains(t, r.URL.RawQuery, "reviewed-by%3Atestuser")

					response := map[string]interface{}{
						"total_count": tc.mockTotalComments,
						"items":       []interface{}{},
					}
					w.WriteHeader(http.StatusOK)
					json.NewEncoder(w).Encode(response)
					return
				}

				w.WriteHeader(http.StatusNotFound)
			}))
			defer mockGitHubServer.Close()

			// Create mock auth service
			mockAuthService := mocks.NewMockGitHubAuthService(ctrl)

			envConfig := &auth.ProviderConfig{
				EnterpriseBaseURL: mockGitHubServer.URL,
			}
			githubClient := auth.NewGitHubClient(envConfig)

			mockAuthService.EXPECT().
				GetGitHubAccessToken("test-uuid", "githubtools").
				Return("test-token", nil)

			mockAuthService.EXPECT().
				GetGitHubClient("githubtools").
				Return(githubClient, nil)

			// Create GitHub service
			githubService := service.NewGitHubServiceWithAdapter(mockAuthService)

			// Execute test
			result, err := githubService.GetUserPRReviewComments(
				context.Background(),
				"test-uuid",
				"githubtools",
				tc.period,
			)

			// Assertions
			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Equal(t, tc.expectedTotalComments, result.TotalComments)
			assert.NotEmpty(t, result.Period)
			assert.NotEmpty(t, result.From)
			assert.NotEmpty(t, result.To)

			// Verify period format
			if tc.period == "" {
				assert.Equal(t, "30d", result.Period) // Default period
			} else {
				assert.Equal(t, tc.period, result.Period)
			}
		})
	}
}

// TestGetUserPRReviewComments_Success_WithPagination tests pagination handling
func TestGetUserPRReviewComments_Success_WithPagination(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	pageCount := 0
	totalComments := 250 // More than 100 per page

	// Create mock GitHub API server
	var mockGitHubServer *httptest.Server
	mockGitHubServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Handle GET /user request
		if r.Method == "GET" && r.URL.Path == "/api/v3/user" {
			response := map[string]interface{}{
				"login":      "testuser",
				"id":         int64(12345),
				"avatar_url": "https://avatars.githubusercontent.com/u/12345",
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(response)
			return
		}

		// Handle search issues request with pagination
		if r.Method == "GET" && r.URL.Path == "/api/v3/search/issues" {
			pageCount++

			// First page
			if pageCount == 1 {
				response := map[string]interface{}{
					"total_count": 100,
					"items":       []interface{}{},
				}
				w.Header().Set("Link", fmt.Sprintf(`<%s/api/v3/search/issues?page=2>; rel="next"`, mockGitHubServer.URL))
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(response)
				return
			}

			// Second page
			if pageCount == 2 {
				response := map[string]interface{}{
					"total_count": 100,
					"items":       []interface{}{},
				}
				w.Header().Set("Link", fmt.Sprintf(`<%s/api/v3/search/issues?page=3>; rel="next"`, mockGitHubServer.URL))
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(response)
				return
			}

			// Third page (last)
			if pageCount == 3 {
				response := map[string]interface{}{
					"total_count": 50,
					"items":       []interface{}{},
				}
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(response)
				return
			}
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer mockGitHubServer.Close()

	// Create mock auth service
	mockAuthService := mocks.NewMockGitHubAuthService(ctrl)

	envConfig := &auth.ProviderConfig{
		EnterpriseBaseURL: mockGitHubServer.URL,
	}
	githubClient := auth.NewGitHubClient(envConfig)

	mockAuthService.EXPECT().
		GetGitHubAccessToken("test-uuid", "githubtools").
		Return("test-token", nil)

	mockAuthService.EXPECT().
		GetGitHubClient("githubtools").
		Return(githubClient, nil)

	// Create GitHub service
	githubService := service.NewGitHubServiceWithAdapter(mockAuthService)

	// Execute test
	result, err := githubService.GetUserPRReviewComments(
		context.Background(),
		"test-uuid",
		"githubtools",
		"30d",
	)

	// Assertions
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, totalComments, result.TotalComments) // Sum of all pages
	assert.Equal(t, 3, pageCount)                        // Verify pagination occurred
	assert.Equal(t, "30d", result.Period)
}

// TestGetUserPRReviewComments_Success_DifferentPeriods tests various period formats
func TestGetUserPRReviewComments_Success_DifferentPeriods(t *testing.T) {
	testCases := []struct {
		name   string
		period string
	}{
		{"SevenDays", "7d"},
		{"ThirtyDays", "30d"},
		{"NinetyDays", "90d"},
		{"OneYear", "365d"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			// Create mock GitHub API server
			var mockGitHubServer *httptest.Server
			mockGitHubServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")

				if r.Method == "GET" && r.URL.Path == "/api/v3/user" {
					response := map[string]interface{}{
						"login": "testuser",
						"id":    int64(12345),
					}
					w.WriteHeader(http.StatusOK)
					json.NewEncoder(w).Encode(response)
					return
				}

				if r.Method == "GET" && r.URL.Path == "/api/v3/search/issues" {
					// Verify the date range in the query
					assert.Contains(t, r.URL.RawQuery, "created%3A")

					response := map[string]interface{}{
						"total_count": 10,
						"items":       []interface{}{},
					}
					w.WriteHeader(http.StatusOK)
					json.NewEncoder(w).Encode(response)
					return
				}

				w.WriteHeader(http.StatusNotFound)
			}))
			defer mockGitHubServer.Close()

			mockAuthService := mocks.NewMockGitHubAuthService(ctrl)

			envConfig := &auth.ProviderConfig{
				EnterpriseBaseURL: mockGitHubServer.URL,
			}
			githubClient := auth.NewGitHubClient(envConfig)

			mockAuthService.EXPECT().
				GetGitHubAccessToken("test-uuid", "githubtools").
				Return("test-token", nil)

			mockAuthService.EXPECT().
				GetGitHubClient("githubtools").
				Return(githubClient, nil)

			githubService := service.NewGitHubServiceWithAdapter(mockAuthService)

			result, err := githubService.GetUserPRReviewComments(
				context.Background(),
				"test-uuid",
				"githubtools",
				tc.period,
			)

			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Equal(t, tc.period, result.Period)
			assert.Equal(t, 10, result.TotalComments)
		})
	}
}

// TestGetUserPRReviewComments_Success_EnterpriseGitHub tests with enterprise GitHub
func TestGetUserPRReviewComments_Success_EnterpriseGitHub(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create mock GitHub Enterprise API server
	var mockGitHubServer *httptest.Server
	mockGitHubServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Verify enterprise API path
		assert.Contains(t, r.URL.Path, "/api/v3/")

		if r.Method == "GET" && r.URL.Path == "/api/v3/user" {
			response := map[string]interface{}{
				"login": "enterprise-user",
				"id":    int64(99999),
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(response)
			return
		}

		if r.Method == "GET" && r.URL.Path == "/api/v3/search/issues" {
			assert.Contains(t, r.URL.RawQuery, "reviewed-by%3Aenterprise-user")

			response := map[string]interface{}{
				"total_count": 75,
				"items":       []interface{}{},
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(response)
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer mockGitHubServer.Close()

	mockAuthService := mocks.NewMockGitHubAuthService(ctrl)

	envConfig := &auth.ProviderConfig{
		EnterpriseBaseURL: mockGitHubServer.URL,
	}
	githubClient := auth.NewGitHubClient(envConfig)

	mockAuthService.EXPECT().
		GetGitHubAccessToken("test-uuid", "github-enterprise").
		Return("enterprise-token", nil)

	mockAuthService.EXPECT().
		GetGitHubClient("github-enterprise").
		Return(githubClient, nil)

	githubService := service.NewGitHubServiceWithAdapter(mockAuthService)

	result, err := githubService.GetUserPRReviewComments(
		context.Background(),
		"test-uuid",
		"github-enterprise",
		"30d",
	)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 75, result.TotalComments)
	assert.Equal(t, "30d", result.Period)
}

// TestGetUserPRReviewComments_UserGetError tests error when getting user info
func TestGetUserPRReviewComments_UserGetError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create mock GitHub API server that returns error for user endpoint
	var mockGitHubServer *httptest.Server
	mockGitHubServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.Method == "GET" && r.URL.Path == "/api/v3/user" {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{
				"message": "Bad credentials",
			})
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer mockGitHubServer.Close()

	mockAuthService := mocks.NewMockGitHubAuthService(ctrl)

	envConfig := &auth.ProviderConfig{
		EnterpriseBaseURL: mockGitHubServer.URL,
	}
	githubClient := auth.NewGitHubClient(envConfig)

	mockAuthService.EXPECT().
		GetGitHubAccessToken("test-uuid", "githubtools").
		Return("invalid-token", nil)

	mockAuthService.EXPECT().
		GetGitHubClient("githubtools").
		Return(githubClient, nil)

	githubService := service.NewGitHubServiceWithAdapter(mockAuthService)

	result, err := githubService.GetUserPRReviewComments(
		context.Background(),
		"test-uuid",
		"githubtools",
		"30d",
	)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to get user")
}

// TestGetUserPRReviewComments_RateLimitExceeded tests rate limit handling
func TestGetUserPRReviewComments_RateLimitExceeded(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create mock GitHub API server that returns rate limit error
	var mockGitHubServer *httptest.Server
	mockGitHubServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.Method == "GET" && r.URL.Path == "/api/v3/user" {
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(map[string]string{
				"message": "API rate limit exceeded",
			})
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer mockGitHubServer.Close()

	mockAuthService := mocks.NewMockGitHubAuthService(ctrl)

	envConfig := &auth.ProviderConfig{
		EnterpriseBaseURL: mockGitHubServer.URL,
	}
	githubClient := auth.NewGitHubClient(envConfig)

	mockAuthService.EXPECT().
		GetGitHubAccessToken("test-uuid", "githubtools").
		Return("test-token", nil)

	mockAuthService.EXPECT().
		GetGitHubClient("githubtools").
		Return(githubClient, nil)

	githubService := service.NewGitHubServiceWithAdapter(mockAuthService)

	result, err := githubService.GetUserPRReviewComments(
		context.Background(),
		"test-uuid",
		"githubtools",
		"30d",
	)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorIs(t, err, apperrors.ErrGitHubAPIRateLimitExceeded)
}

// TestGetUserPRReviewComments_SearchRateLimitExceeded tests rate limit on search endpoint
func TestGetUserPRReviewComments_SearchRateLimitExceeded(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create mock GitHub API server
	var mockGitHubServer *httptest.Server
	mockGitHubServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.Method == "GET" && r.URL.Path == "/api/v3/user" {
			response := map[string]interface{}{
				"login": "testuser",
				"id":    int64(12345),
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(response)
			return
		}

		if r.Method == "GET" && r.URL.Path == "/api/v3/search/issues" {
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(map[string]string{
				"message": "API rate limit exceeded",
			})
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer mockGitHubServer.Close()

	mockAuthService := mocks.NewMockGitHubAuthService(ctrl)

	envConfig := &auth.ProviderConfig{
		EnterpriseBaseURL: mockGitHubServer.URL,
	}
	githubClient := auth.NewGitHubClient(envConfig)

	mockAuthService.EXPECT().
		GetGitHubAccessToken("test-uuid", "githubtools").
		Return("test-token", nil)

	mockAuthService.EXPECT().
		GetGitHubClient("githubtools").
		Return(githubClient, nil)

	githubService := service.NewGitHubServiceWithAdapter(mockAuthService)

	result, err := githubService.GetUserPRReviewComments(
		context.Background(),
		"test-uuid",
		"githubtools",
		"30d",
	)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorIs(t, err, apperrors.ErrGitHubAPIRateLimitExceeded)
}