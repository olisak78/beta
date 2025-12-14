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
	apperrors "developer-portal-backend/internal/errors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Minimal mock implementing GitHubAuthService for tests
type mockAuthService struct {
	accessToken string
	baseURL     string
	tokenErr    error
	clientErr   error
}

func (m *mockAuthService) GetGitHubAccessToken(userUUID, provider string) (string, error) {
	if m.tokenErr != nil {
		return "", m.tokenErr
	}
	if m.accessToken == "" {
		return "", fmt.Errorf("no access token")
	}
	return m.accessToken, nil
}

func (m *mockAuthService) GetGitHubClient(provider string) (*auth.GitHubClient, error) {
	if m.clientErr != nil {
		return nil, m.clientErr
	}
	cfg := &auth.ProviderConfig{
		ClientID:          "test-client-id",
		ClientSecret:      "test-client-secret",
		EnterpriseBaseURL: m.baseURL,
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
			name:          "TrailingSlash",
			url:           "https://github.com/owner/repo/",
			expectedOwner: "owner",
			expectedRepo:  "repo",
			expectedFull:  "owner/repo",
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
			owner, repo, full := parseRepositoryFromURL(tt.url)
			assert.Equal(t, tt.expectedOwner, owner)
			assert.Equal(t, tt.expectedRepo, repo)
			assert.Equal(t, tt.expectedFull, full)
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
	svc := NewGitHubServiceWithAdapter(mock)

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
	svc := NewGitHubServiceWithAdapter(mock)

	ctx := context.Background()
	res, err := svc.GetUserTotalContributions(ctx, "test-uuid", "githubtools", "30d")
	require.Error(t, err)
	assert.Nil(t, res)
	assert.True(t, errors.Is(err, apperrors.ErrGitHubAPIRateLimitExceeded))
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
	svc := NewGitHubServiceWithAdapter(mock)

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

	ctx := context.Background()
	res, err := svc.GetContributionsHeatmap(ctx, "test-uuid", "githubtools", "bad")
	require.Error(t, err)
	assert.Nil(t, res)
	assert.True(t, errors.Is(err, apperrors.ErrInvalidPeriodFormat))
}

func TestAuthServiceAdapter_Nil(t *testing.T) {
	adapter := NewAuthServiceAdapter(nil)

	// GetGitHubAccessToken should error when auth service is nil
	_, err := adapter.GetGitHubAccessToken("test-uuid", "githubtools")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "auth service is not initialized")

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

	mock := &mockAuthService{
		accessToken: "test-token",
		baseURL:     server.URL,
	}
	svc := NewGitHubServiceWithAdapter(mock)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	res, err := svc.GetUserTotalContributions(ctx, "test-uuid", "githubtools", "30d")
	require.Error(t, err)
	assert.Nil(t, res)
}
