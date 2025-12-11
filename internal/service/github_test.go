package service_test

import (
	"context"
	apperrors "developer-portal-backend/internal/errors"
	"fmt"
	"testing"

	"developer-portal-backend/internal/auth"
	"developer-portal-backend/internal/cache"
	"developer-portal-backend/internal/mocks"
	"developer-portal-backend/internal/service"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

// These tests validate signatures and basic input validation without relying on external services.
// They intentionally construct the service with a nil auth service so calls fail early with clear errors.

// GitHubServiceTestSuite - Test suite for comprehensive unit tests with mocks
type GitHubServiceTestSuite struct {
	suite.Suite
	service     *service.GitHubService
	mockAuthSvc *mocks.MockGitHubAuthService
	ctrl        *gomock.Controller
}

func (suite *GitHubServiceTestSuite) SetupTest() {
	suite.ctrl = gomock.NewController(suite.T())
	suite.mockAuthSvc = mocks.NewMockGitHubAuthService(suite.ctrl)
	suite.service = service.NewGitHubServiceWithAdapter(suite.mockAuthSvc)
}

func (suite *GitHubServiceTestSuite) TearDownTest() {
	if suite.ctrl != nil {
		suite.ctrl.Finish()
	}
}

// tests below use a GitHubService constructed with nil auth service to validate input checks
func TestGetUserOpenPullRequests_RequiresUUIDAndProvider(t *testing.T) {
	gh := service.NewGitHubService(nil)
	_, err := gh.GetUserOpenPullRequests(context.Background(), "", "", "open", "created", "desc", 30, 1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "userUUID and provider are required")
}

	mockCache := cache.NewInMemoryCache(5*time.Minute, 10*time.Minute)
	githubService := service.NewGitHubService(authService, mockCache)

func TestGetUserTotalContributions_RequiresUUIDAndProvider(t *testing.T) {
	gh := service.NewGitHubService(nil)
	_, err := gh.GetUserTotalContributions(context.Background(), "", "", "30d")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "userUUID and provider are required")
}

func TestGetUserTotalContributions_AuthCalled(t *testing.T) {
	gh := service.NewGitHubService(nil)
	_, err := gh.GetUserTotalContributions(context.Background(), "test-uuid", "githubtools", "30d")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get GitHub access token")
}

	// Create service with nil auth service for this test
	mockCache := cache.NewInMemoryCache(5*time.Minute, 10*time.Minute)
	githubService := service.NewGitHubService(nil, mockCache)

	// Missing uuid/provider
	_, err := gh.ClosePullRequest(context.Background(), "", "", "owner", "repo", 1, false)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "userUUID and provider are required")

	// Missing owner
	uuid := "test-uuid"
	provider := "githubtools"
	_, err = gh.ClosePullRequest(context.Background(), uuid, provider, "", "repo", 1, false)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "owner and repository are required")

	// Missing repo
	_, err = gh.ClosePullRequest(context.Background(), uuid, provider, "owner", "", 1, false)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "owner and repository are required")
}

// The following tests use the test suite with mocked auth service

	mockCache := cache.NewInMemoryCache(5*time.Minute, 10*time.Minute)
	githubService := service.NewGitHubService(nil, mockCache)

	_, err := suite.service.GetContributionsHeatmap(ctx, "", "", "30d")
	suite.Error(err)
	suite.Contains(err.Error(), "userUUID and provider are required")
}

func (suite *GitHubServiceTestSuite) TestGetContributionsHeatmap_InvalidPeriodFormat() {
	ctx := context.Background()

	// Mock GetGitHubClient to avoid unexpected call error
	suite.mockAuthSvc.EXPECT().
		GetGitHubClient("githubtools").
		Return(nil, nil).
		AnyTimes()

	_, err := suite.service.GetContributionsHeatmap(ctx, "test-uuid", "githubtools", "invalid")
	suite.Error(err)
	suite.Contains(err.Error(), "invalid period format")
}

// TestGetUserOpenPullRequests_DefaultParameters tests with default parameters
func (suite *GitHubServiceTestSuite) TestGetUserOpenPullRequests_DefaultParameters() {

	mockCache := cache.NewInMemoryCache(5*time.Minute, 10*time.Minute)
	githubService := service.NewGitHubService(nil, mockCache)

	claims := &auth.AuthClaims{
		UserID:   12345,
		Username: "testuser",
		Email:    "test@example.com",
		Provider: "githubtools",
	}

	ctx := context.Background()

	suite.mockAuthSvc.EXPECT().
		GetGitHubClient("invalid-provider").
		Return(nil, assert.AnError)

	_, err := suite.service.GetContributionsHeatmap(ctx, "test-uuid", "invalid-provider", "30d")
	suite.Error(err)
	suite.Contains(err.Error(), "provider 'invalid-provider'")
}

func (suite *GitHubServiceTestSuite) TestGetContributionsHeatmap_TokenRetrievalFailure() {
	ctx := context.Background()

	suite.mockAuthSvc.EXPECT().
		GetGitHubClient("githubtools").
		Return(nil, nil)

	suite.mockAuthSvc.EXPECT().
		GetGitHubAccessToken("test-uuid", "githubtools").
		Return("", apperrors.ErrTokenStoreNotInitialized)

	_, err := suite.service.GetContributionsHeatmap(ctx, "test-uuid", "githubtools", "30d")
	suite.Error(err)
	suite.Contains(err.Error(), "failed to get GitHub access token")
}

func (suite *GitHubServiceTestSuite) TestGetContributionsHeatmap_ClientRetrievalFailure() {
	ctx := context.Background()

	// First call succeeds for validation
	suite.mockAuthSvc.EXPECT().
		GetGitHubClient("githubtools").
		Return(nil, nil).
		Times(1)

	suite.mockAuthSvc.EXPECT().
		GetGitHubAccessToken("test-uuid", "githubtools").
		Return("test-token", nil)

	// Second call fails
	suite.mockAuthSvc.EXPECT().
		GetGitHubClient("githubtools").
		Return(nil, assert.AnError).
		Times(1)

	_, err := suite.service.GetContributionsHeatmap(ctx, "test-uuid", "githubtools", "30d")
	suite.Error(err)
	suite.Contains(err.Error(), "failed to get GitHub client")
}

// TestGetUserOpenPullRequests_NoValidSession tests when user has no valid GitHub session
func (suite *GitHubServiceTestSuite) TestGetUserOpenPullRequests_NoValidSession() {
	// This test demonstrates the error case when no valid session exists

	claims := &auth.AuthClaims{
		UserID:   12345,
		Username: "testuser",
		Email:    "test@example.com",
		Provider: "githubtools",
	}

	// Create service with nil auth service to simulate no session
	mockCache := cache.NewInMemoryCache(5*time.Minute, 10*time.Minute)
	githubService := service.NewGitHubService(nil, mockCache)

	ctx := context.Background()

	_, err := suite.service.GetAveragePRMergeTime(ctx, "", "", "30d")
	suite.Error(err)
	suite.Equal(err.Error(), apperrors.ErrMissingUserUUIDAndProvider.Error())
}

// TestParameterValidation tests parameter validation and defaults
func (suite *GitHubServiceTestSuite) TestParameterValidation() {

	mockCache := cache.NewInMemoryCache(5*time.Minute, 10*time.Minute)
	githubService := service.NewGitHubService(nil, mockCache)

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
		{"MissingD", "30"},
		{"NonNumeric", "abcd"},
		{"NegativeDays", "-30d"},
		{"ZeroDays", "0d"},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			_, err := suite.service.GetAveragePRMergeTime(ctx, "test-uuid", "githubtools", tc.period)
			suite.Error(err)
			suite.Contains(err.Error(), "invalid period format")
		})
	}
}

// TestGitHubUserStructure tests GitHubUser structure
func (suite *GitHubServiceTestSuite) TestGitHubUserStructure() {
	user := service.GitHubUser{
		Login:     "johndoe",
		ID:        12345,
		AvatarURL: "https://avatars.githubusercontent.com/u/12345",
	}

	assert.Equal(suite.T(), "johndoe", user.Login)
	assert.Equal(suite.T(), int64(12345), user.ID)
	assert.Contains(suite.T(), user.AvatarURL, "avatars.githubusercontent.com")
}

// TestRepositoryStructure tests Repository structure
func (suite *GitHubServiceTestSuite) TestRepositoryStructure() {
	repo := service.Repository{
		Name:     "my-awesome-repo",
		FullName: "octocat/my-awesome-repo",
		Owner:    "octocat",
		Private:  true,
	}

	assert.Equal(suite.T(), "my-awesome-repo", repo.Name)
	assert.Equal(suite.T(), "octocat/my-awesome-repo", repo.FullName)
	assert.Equal(suite.T(), "octocat", repo.Owner)
	assert.True(suite.T(), repo.Private)
}

// TestEmptyPullRequestsResponse tests response with no PRs
func (suite *GitHubServiceTestSuite) TestEmptyPullRequestsResponse() {
	response := service.PullRequestsResponse{
		PullRequests: []service.PullRequest{},
		Total:        0,
	}

	assert.Equal(suite.T(), 0, response.Total)
	assert.Empty(suite.T(), response.PullRequests)
	assert.NotNil(suite.T(), response.PullRequests) // Should be empty slice, not nil
}

// TestContextCancellation tests context cancellation handling
func (suite *GitHubServiceTestSuite) TestContextCancellation() {
	mockCache := cache.NewInMemoryCache(5*time.Minute, 10*time.Minute)
	githubService := service.NewGitHubService(nil, mockCache)

	claims := &auth.AuthClaims{
		UserID:   12345,
		Username: "testuser",
		Email:    "test@example.com",
		Provider: "githubtools",
	}

	// Create a cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	result, err := githubService.GetUserOpenPullRequests(ctx, claims, "open", "created", "desc", 30, 1)

	// Should fail (either at auth or context check)
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), result)
}

// TestContextTimeout tests context timeout handling
func (suite *GitHubServiceTestSuite) TestContextTimeout() {
	mockCache := cache.NewInMemoryCache(5*time.Minute, 10*time.Minute)
	githubService := service.NewGitHubService(nil, mockCache)

	claims := &auth.AuthClaims{
		UserID:   12345,
		Username: "testuser",
		Email:    "test@example.com",
		Provider: "githubtools",
	}

	// Create a context with very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	// Sleep to ensure timeout
	time.Sleep(1 * time.Millisecond)

	result, err := githubService.GetUserOpenPullRequests(ctx, claims, "open", "created", "desc", 30, 1)

	// Should fail due to timeout or auth
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), result)
}

// TestServiceCreationWithNilAuthService tests service creation
func (suite *GitHubServiceTestSuite) TestServiceCreationWithNilAuthService() {
	mockCache := cache.NewInMemoryCache(5*time.Minute, 10*time.Minute)
	githubService := service.NewGitHubService(nil, mockCache)

	assert.NotNil(suite.T(), githubService)
	// Service should be created even with nil auth service
	// It will fail on actual API calls, but creation should succeed
}

// TestGetUserTotalContributions_NilClaims tests with nil claims
func (suite *GitHubServiceTestSuite) TestGetUserTotalContributions_NilClaims() {
	mockCache := cache.NewInMemoryCache(5*time.Minute, 10*time.Minute)
	githubService := service.NewGitHubService(nil, mockCache)
	ctx := context.Background()

	suite.mockAuthSvc.EXPECT().
		GetGitHubAccessToken("test-uuid", "githubtools").
		Return("", apperrors.ErrTokenStoreNotInitialized)

	_, err := suite.service.GetAveragePRMergeTime(ctx, "test-uuid", "githubtools", "30d")
	suite.Error(err)
	suite.Contains(err.Error(), apperrors.ErrTokenStoreNotInitialized.Message)
}

// TestGetUserTotalContributions_DefaultPeriod tests with default period
func (suite *GitHubServiceTestSuite) TestGetUserTotalContributions_DefaultPeriod() {
	mockCache := cache.NewInMemoryCache(5*time.Minute, 10*time.Minute)
	githubService := service.NewGitHubService(nil, mockCache)

	claims := &auth.AuthClaims{
		UserID:   12345,
		Username: "testuser",
		Email:    "test@example.com",
		Provider: "githubtools",
	}

	ctx := context.Background()

	suite.mockAuthSvc.EXPECT().
		GetGitHubAccessToken("test-uuid", "githubtools").
		Return("test-token", nil)

// TestGetUserTotalContributions_ValidPeriods tests various valid period formats
func (suite *GitHubServiceTestSuite) TestGetUserTotalContributions_ValidPeriods() {
	mockCache := cache.NewInMemoryCache(5*time.Minute, 10*time.Minute)
	githubService := service.NewGitHubService(nil, mockCache)

	_, err := suite.service.GetAveragePRMergeTime(ctx, "test-uuid", "githubtools", "30d")
	suite.Error(err)
	suite.Contains(err.Error(), "failed to get GitHub client")
}

func (suite *GitHubServiceTestSuite) TestGetAveragePRMergeTime_ValidPeriods() {
	ctx := context.Background()

	validPeriods := []string{"1d", "7d", "30d", "90d", "180d", "365d"}

	for _, period := range validPeriods {
		suite.Run(period, func() {
			suite.mockAuthSvc.EXPECT().
				GetGitHubAccessToken("test-uuid", "githubtools").
				Return("test-token", nil)

			suite.mockAuthSvc.EXPECT().
				GetGitHubClient("githubtools").
				Return(nil, nil)

			// Will fail at HTTP but period validation passes
			_, err := suite.service.GetAveragePRMergeTime(ctx, "test-uuid", "githubtools", period)
			suite.Error(err)
			// Should NOT contain period format error
			suite.NotContains(err.Error(), "period must be in format")
			suite.NotContains(err.Error(), "invalid period format")
		})
	}
}

// TestGetUserTotalContributions_InvalidPeriods tests various invalid period formats
func (suite *GitHubServiceTestSuite) TestGetUserTotalContributions_InvalidPeriods() {
	mockCache := cache.NewInMemoryCache(5*time.Minute, 10*time.Minute)
githubService := service.NewGitHubService(nil, mockCache)

	testCases := []struct {
		name          string
		userUUID      string
		provider      string
		mockError     error
		expectedError string
	}{
		{
			name:          "AuthServiceNotInitialized",
			userUUID:      "test-uuid",
			provider:      "githubtools",
			mockError:     apperrors.ErrAuthServiceNotInitialized,
			expectedError: "auth service is not initialized",
		},
		{
			name:          "UserUUIDMissing",
			userUUID:      "test-uuid",
			provider:      "githubtools",
			mockError:     apperrors.ErrUserUUIDMissing,
			expectedError: "userUUID cannot be empty",
		},
		{
			name:          "ProviderMissing",
			userUUID:      "test-uuid",
			provider:      "githubtools",
			mockError:     apperrors.ErrProviderMissing,
			expectedError: "provider cannot be empty",
		},
		{
			name:          "TokenStoreNotInitialized",
			userUUID:      "test-uuid",
			provider:      "githubtools",
			mockError:     apperrors.ErrTokenStoreNotInitialized,
			expectedError: "token store not initialized",
		},
		{
			name:          "InvalidUserUUID",
			userUUID:      "test-uuid",
			provider:      "githubtools",
			mockError:     fmt.Errorf("invalid userUUID: invalid UUID format"),
			expectedError: "invalid userUUID",
		},
		{
			name:          "NoValidToken",
			userUUID:      "test-uuid",
			provider:      "githubtools",
			mockError:     fmt.Errorf("no valid GitHub token found for user test-uuid with provider githubtools"),
			expectedError: "no valid GitHub token found",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.mockAuthSvc.EXPECT().
				GetGitHubAccessToken(tc.userUUID, tc.provider).
				Return("", tc.mockError)

			_, err := suite.service.GetRepositoryContent(ctx, tc.userUUID, tc.provider, "owner", "repo", "README.md", "main")

			suite.Error(err)
			suite.Contains(err.Error(), "failed to get access token")
			suite.Contains(err.Error(), tc.expectedError)
		})
	}
}

// TestGetUserTotalContributions_LargePeriod tests period larger than 365 days
func (suite *GitHubServiceTestSuite) TestGetUserTotalContributions_LargePeriod() {
	mockCache := cache.NewInMemoryCache(5*time.Minute, 10*time.Minute)
githubService := service.NewGitHubService(nil, mockCache)

	claims := &auth.AuthClaims{
		UserID:   12345,
		Username: "testuser",
		Email:    "test@example.com",
		Provider: "githubtools",
	}

	ctx := context.Background()

	suite.mockAuthSvc.EXPECT().
		GetGitHubAccessToken("test-uuid", "githubtools").
		Return("test-token", nil)

	suite.mockAuthSvc.EXPECT().
		GetGitHubClient("githubtools").
		Return(nil, apperrors.ErrAuthServiceNotInitialized)

	_, err := suite.service.GetRepositoryContent(ctx, "test-uuid", "githubtools", "owner", "repo", "README.md", "main")
	suite.Error(err)
	suite.ErrorIs(err, apperrors.ErrAuthServiceNotInitialized)
}

// TestGetUserTotalContributions_ContextCancellation tests context cancellation handling
func (suite *GitHubServiceTestSuite) TestGetUserTotalContributions_ContextCancellation() {
	mockCache := cache.NewInMemoryCache(5*time.Minute, 10*time.Minute)
githubService := service.NewGitHubService(nil, mockCache)

	claims := &auth.AuthClaims{
		UserID:   12345,
		Username: "testuser",
		Email:    "test@example.com",
		Provider: "githubtools",
	}

	// Create a cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	suite.mockAuthSvc.EXPECT().
		GetGitHubAccessToken("test-uuid", "githubtools").
		Return("test-token", nil)

	suite.mockAuthSvc.EXPECT().
		GetGitHubClient("githubtools").
		Return(nil, nil)

	// Empty ref should default to "main" - will fail at API call but validates ref defaulting
	_, err := suite.service.GetRepositoryContent(ctx, "test-uuid", "githubtools", "owner", "repo", "README.md", "")
	suite.Error(err) // Expected to fail at API call
}

func (suite *GitHubServiceTestSuite) TestGetRepositoryContent_PathNormalization() {
	ctx := context.Background()

	testCases := []struct {
		name        string
		path        string
		description string
	}{
		{
			name:        "PathWithLeadingSlash",
			path:        "/path/to/file.txt",
			description: "Leading slash should be removed",
		},
		{
			name:        "PathWithoutLeadingSlash",
			path:        "path/to/file.txt",
			description: "Path without leading slash remains unchanged",
		},
		{
			name:        "RootPath",
			path:        "",
			description: "Empty path (root directory)",
		},
		{
			name:        "SingleFile",
			path:        "README.md",
			description: "Single file in root",
		},
	}

	// 2. Create service (with real auth service in production)
	mockCache := cache.NewInMemoryCache(5*time.Minute, 10*time.Minute)
githubService := service.NewGitHubService(nil, mockCache)


func (suite *GitHubServiceTestSuite) TestGetRepositoryContent_VariousParameters() {
	ctx := context.Background()

	testCases := []struct {
		name        string
		owner       string
		repo        string
		path        string
		ref         string
		description string
	}{
		{
			name:        "RootDirectory",
			owner:       "owner",
			repo:        "repo",
			path:        "",
			ref:         "main",
			description: "Fetch root directory contents",
		},
		{
			name:        "NestedPath",
			owner:       "owner",
			repo:        "repo",
			path:        "src/main/java/App.java",
			ref:         "develop",
			description: "Fetch deeply nested file",
		},
		{
			name:        "CustomBranch",
			owner:       "owner",
			repo:        "repo",
			path:        "config.yml",
			ref:         "feature/new-feature",
			description: "Fetch from custom branch",
		},
		{
			name:        "TagRef",
			owner:       "owner",
			repo:        "repo",
			path:        "CHANGELOG.md",
			ref:         "v1.0.0",
			description: "Fetch from tag",
		},
		{
			name:        "CommitSHA",
			owner:       "owner",
			repo:        "repo",
			path:        "README.md",
			ref:         "abc123def456",
			description: "Fetch from specific commit",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.mockAuthSvc.EXPECT().
				GetGitHubAccessToken("test-uuid", "githubtools").
				Return("test-token", nil)

			suite.mockAuthSvc.EXPECT().
				GetGitHubClient("githubtools").
				Return(nil, nil)

			// All will fail at API call but validates parameter handling
			_, err := suite.service.GetRepositoryContent(ctx, "test-uuid", "githubtools", tc.owner, tc.repo, tc.path, tc.ref)
			suite.Error(err) // Expected to fail at API call
		})
	}
}

// TestErrorMessages tests that error messages are descriptive
func (suite *GitHubServiceTestSuite) TestErrorMessages() {
	mockCache := cache.NewInMemoryCache(5*time.Minute, 10*time.Minute)
githubService := service.NewGitHubService(nil, mockCache)

	testCases := []struct {
		name          string
		userUUID      string
		provider      string
		mockError     error
		expectedError string
	}{
		{
			name:          "AuthServiceNotInitialized",
			userUUID:      "test-uuid",
			provider:      "githubtools",
			mockError:     apperrors.ErrAuthServiceNotInitialized,
			expectedError: "auth service is not initialized",
		},
		{
			name:          "UserUUIDMissing",
			userUUID:      "test-uuid",
			provider:      "githubtools",
			mockError:     apperrors.ErrUserUUIDMissing,
			expectedError: "userUUID cannot be empty",
		},
		{
			name:          "ProviderMissing",
			userUUID:      "test-uuid",
			provider:      "githubtools",
			mockError:     apperrors.ErrProviderMissing,
			expectedError: "provider cannot be empty",
		},
		{
			name:          "TokenStoreNotInitialized",
			userUUID:      "test-uuid",
			provider:      "githubtools",
			mockError:     apperrors.ErrTokenStoreNotInitialized,
			expectedError: "token store not initialized",
		},
		{
			name:          "InvalidUserUUID",
			userUUID:      "test-uuid",
			provider:      "githubtools",
			mockError:     fmt.Errorf("invalid userUUID: invalid UUID format"),
			expectedError: "invalid userUUID",
		},
		{
			name:          "NoValidToken",
			userUUID:      "test-uuid",
			provider:      "githubtools",
			mockError:     fmt.Errorf("no valid GitHub token found for user test-uuid with provider githubtools"),
			expectedError: "no valid GitHub token found",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.mockAuthSvc.EXPECT().
				GetGitHubAccessToken(tc.userUUID, tc.provider).
				Return("", tc.mockError)

			_, err := suite.service.UpdateRepositoryFile(
				ctx,
				tc.userUUID,
				tc.provider,
				"owner",
				"repo",
				"file.txt",
				"Update file",
				"new content",
				"abc123",
				"main",
			)

			suite.Error(err)
			suite.Contains(err.Error(), "failed to get access token")
			suite.Contains(err.Error(), tc.expectedError)
		})
	}
}

// TestGetUserOpenPullRequests_StateAllParameter tests the specific fix for state=all
// This test documents the behavior where state=all should work correctly
// Bug fix: GitHub Search API doesn't support state:all qualifier, so we omit it
func (suite *GitHubServiceTestSuite) TestGetUserOpenPullRequests_StateAllParameter() {
	mockCache := cache.NewInMemoryCache(5*time.Minute, 10*time.Minute)
githubService := service.NewGitHubService(nil, mockCache)

	claims := &auth.AuthClaims{
		UserID:   12345,
		Username: "testuser",
		Email:    "test@example.com",
		Provider: "githubtools",
	}

func (suite *GitHubServiceTestSuite) TestUpdateRepositoryFile_ParameterValidation() {
	ctx := context.Background()

	testCases := []struct {
		name        string
		owner       string
		repo        string
		path        string
		message     string
		content     string
		sha         string
		branch      string
		description string
	}{
		{
			name:        "EmptyPath",
			owner:       "owner",
			repo:        "repo",
			path:        "",
			message:     "Update",
			content:     "content",
			sha:         "sha123",
			branch:      "main",
			description: "Empty file path",
		},
		{
			name:        "EmptyMessage",
			owner:       "owner",
			repo:        "repo",
			path:        "file.txt",
			message:     "",
			content:     "content",
			sha:         "sha123",
			branch:      "main",
			description: "Empty commit message",
		},
		{
			name:        "EmptySHA",
			owner:       "owner",
			repo:        "repo",
			path:        "file.txt",
			message:     "Update",
			content:     "content",
			sha:         "",
			branch:      "main",
			description: "Empty SHA (required for updates)",
		},
		{
			name:        "EmptyBranch",
			owner:       "owner",
			repo:        "repo",
			path:        "file.txt",
			message:     "Update",
			content:     "content",
			sha:         "sha123",
			branch:      "",
			description: "Empty branch (should default to main)",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.mockAuthSvc.EXPECT().
				GetGitHubAccessToken("test-uuid", "githubtools").
				Return("test-token", nil)

			suite.mockAuthSvc.EXPECT().
				GetGitHubClient("githubtools").
				Return(nil, nil)

			// Will fail at API call but validates parameter handling
			_, err := suite.service.UpdateRepositoryFile(
				ctx,
				"test-uuid",
				"githubtools",
				tc.owner,
				tc.repo,
				tc.path,
				tc.message,
				tc.content,
				tc.sha,
				tc.branch,
			)

			suite.Error(err) // Expected to fail at API call
		})
	}
}

func (suite *GitHubServiceTestSuite) TestUpdateRepositoryFile_VariousScenarios() {
	ctx := context.Background()

	testCases := []struct {
		name        string
		path        string
		message     string
		content     string
		sha         string
		branch      string
		description string
	}{
		{
			name:        "UpdateRootFile",
			path:        "README.md",
			message:     "Update README",
			content:     "# Updated README\n\nNew content here.",
			sha:         "abc123def456",
			branch:      "main",
			description: "Update file in root directory",
		},
		{
			name:        "UpdateNestedFile",
			path:        "src/main/java/App.java",
			message:     "Fix bug in App.java",
			content:     "public class App {\n  // Fixed code\n}",
			sha:         "def456ghi789",
			branch:      "develop",
			description: "Update deeply nested file",
		},
		{
			name:        "UpdateOnFeatureBranch",
			path:        "config.yml",
			message:     "Update configuration",
			content:     "key: value\nother: setting",
			sha:         "ghi789jkl012",
			branch:      "feature/new-config",
			description: "Update file on feature branch",
		},
		{
			name:        "LargeFileUpdate",
			path:        "data/large-file.json",
			message:     "Update large data file",
			content:     string(make([]byte, 10000)), // 10KB content
			sha:         "jkl012mno345",
			branch:      "main",
			description: "Update large file",
		},
		{
			name:        "SpecialCharactersInMessage",
			path:        "file.txt",
			message:     "Fix: Update file with special chars (äöü) & symbols!",
			content:     "content",
			sha:         "mno345pqr678",
			branch:      "main",
			description: "Commit message with special characters",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.mockAuthSvc.EXPECT().
				GetGitHubAccessToken("test-uuid", "githubtools").
				Return("test-token", nil)

			suite.mockAuthSvc.EXPECT().
				GetGitHubClient("githubtools").
				Return(nil, nil)

			// All will fail at API call but validates parameter handling
			_, err := suite.service.UpdateRepositoryFile(
				ctx,
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

			suite.Error(err) // Expected to fail at API call
		})
	}
}

func (suite *GitHubServiceTestSuite) TestGetGitHubAsset_URLValidation() {
	ctx := context.Background()

	testCases := []struct {
		name          string
		assetURL      string
		expectedError string
		description   string
	}{
		{
			name:          "EmptyURL",
			assetURL:      "",
			expectedError: "asset URL is required",
			description:   "Empty asset URL should fail",
		},
		{
			name:          "InvalidURL",
			assetURL:      "not-a-valid-url",
			expectedError: "invalid asset URL",
			description:   "Malformed URL should fail",
		},
		{
			name:          "NonGitHubURL",
			assetURL:      "https://example.com/image.png",
			expectedError: "asset URL must be from GitHub",
			description:   "Non-GitHub URL should fail",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.mockAuthSvc.EXPECT().
				GetGitHubAccessToken("test-uuid", "githubtools").
				Return("test-token", nil).
				MaxTimes(1)

			_, _, err := suite.service.GetGitHubAsset(ctx, "test-uuid", "githubtools", tc.assetURL)

			suite.Error(err)

		})
	}
}

// Benchmark test for response structure creation
func BenchmarkPullRequestCreation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = service.PullRequest{
			ID:        int64(i),
			Number:    i,
			Title:     fmt.Sprintf("PR %d", i),
			State:     "open",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			HTMLURL:   fmt.Sprintf("https://github.com/owner/repo/pull/%d", i),
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
		}
	}
}

// TestGetAveragePRMergeTime_NilClaims tests with nil claims
func (suite *GitHubServiceTestSuite) TestGetAveragePRMergeTime_NilClaims() {
	mockCache := cache.NewInMemoryCache(5*time.Minute, 10*time.Minute)
githubService := service.NewGitHubService(nil, mockCache)
	ctx := context.Background()

	result, err := githubService.GetAveragePRMergeTime(ctx, nil, "30d")

	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), result)
	assert.Contains(suite.T(), err.Error(), "authentication required")
}

// TestGetAveragePRMergeTime_InvalidPeriod tests with invalid period format
func (suite *GitHubServiceTestSuite) TestGetAveragePRMergeTime_InvalidPeriod() {
	mockCache := cache.NewInMemoryCache(5*time.Minute, 10*time.Minute)
githubService := service.NewGitHubService(nil, mockCache)

	claims := &auth.AuthClaims{
		UserID:   12345,
		Username: "testuser",
		Email:    "test@example.com",
		Provider: "githubtools",
	}

	ctx := context.Background()

	testCases := []struct {
		name        string
		userUUID    string
		provider    string
		assetURL    string
		description string
	}{
		{
			name:        "StandardGitHubTools",
			userUUID:    "user-123",
			provider:    "githubtools",
			assetURL:    "https://github.com/owner/repo/raw/main/asset.png",
			description: "Standard githubtools provider",
		},
		{
			name:        "EnterpriseGitHub",
			userUUID:    "user-456",
			provider:    "github-enterprise",
			assetURL:    "https://github.enterprise.com/owner/repo/raw/main/asset.png",
			description: "Enterprise GitHub provider",
		},
		{
			name:        "DifferentUserUUID",
			userUUID:    "different-uuid-789",
			provider:    "githubtools",
			assetURL:    "https://github.com/owner/repo/raw/main/asset.png",
			description: "Different user UUID",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.mockAuthSvc.EXPECT().
				GetGitHubAccessToken(tc.userUUID, tc.provider).
				Return("test-token", nil)

			// Will fail at HTTP call but validates parameter handling
			_, _, err := suite.service.GetGitHubAsset(ctx, tc.userUUID, tc.provider, tc.assetURL)

			suite.Error(err) // Expected to fail at HTTP call
		})
	}
}

// TestGetAveragePRMergeTime_DefaultPeriod tests default period handling
func (suite *GitHubServiceTestSuite) TestGetAveragePRMergeTime_DefaultPeriod() {
	mockCache := cache.NewInMemoryCache(5*time.Minute, 10*time.Minute)
githubService := service.NewGitHubService(nil, mockCache)

	_, err := suite.service.GetUserPRReviewComments(ctx, "", "", "30d")
	suite.Error(err)
	suite.Contains(err.Error(), "userUUID and provider are required")
}

func (suite *GitHubServiceTestSuite) TestGetUserPRReviewComments_InvalidPeriodFormat() {
	ctx := context.Background()

	_, err := suite.service.GetUserPRReviewComments(ctx, "test-uuid", "githubtools", "invalid")
	suite.Error(err)
	// The actual error message is slightly different - it says "period must contain a positive number of days"
	suite.Contains(err.Error(), "invalid period format")
}

// TestGetAveragePRMergeTime_NoAuthService tests when auth service fails
func (suite *GitHubServiceTestSuite) TestGetAveragePRMergeTime_NoAuthService() {
	mockCache := cache.NewInMemoryCache(5*time.Minute, 10*time.Minute)
githubService := service.NewGitHubService(nil, mockCache)

	suite.mockAuthSvc.EXPECT().
		GetGitHubAccessToken("test-uuid", "githubtools").
		Return("", assert.AnError)

	_, err := suite.service.GetUserPRReviewComments(ctx, "test-uuid", "githubtools", "")
	suite.Error(err)
	suite.Contains(err.Error(), "failed to get GitHub access token")
}

func (suite *GitHubServiceTestSuite) TestGetContributionsHeatmap_Success_WithDefaultPeriod() {
	// This test would require mocking HTTP server similar to github_mock_test.go
	// For now, we test that the flow reaches the HTTP call stage
	ctx := context.Background()

	suite.mockAuthSvc.EXPECT().
		GetGitHubClient("githubtools").
		Return(nil, nil).
		Times(2) // Called twice: validation + actual use

	suite.mockAuthSvc.EXPECT().
		GetGitHubAccessToken("test-uuid", "githubtools").
		Return("test-token", nil)

	// This will fail at HTTP execution, but validates the flow up to that point
	_, err := suite.service.GetContributionsHeatmap(ctx, "test-uuid", "githubtools", "")
	suite.Error(err) // Expected to fail at HTTP call since we don't have a mock server
	suite.Contains(err.Error(), "GraphQL query failed with status")
}

func (suite *GitHubServiceTestSuite) TestGetContributionsHeatmap_PeriodValidation() {
	ctx := context.Background()

	testCases := []struct {
		name          string
		period        string
		shouldBeValid bool
		setupMocks    func()
	}{
		{
			name:          "NegativeDays",
			period:        "-30d",
			shouldBeValid: false,
			setupMocks: func() {
				suite.mockAuthSvc.EXPECT().
					GetGitHubClient("githubtools").
					Return(nil, nil)
			},
		},
		{
			name:          "MissingD",
			period:        "30",
			shouldBeValid: false,
			setupMocks: func() {
				suite.mockAuthSvc.EXPECT().
					GetGitHubClient("githubtools").
					Return(nil, nil)
			},
		},
		{
			name:          "NonNumeric",
			period:        "abcd",
			shouldBeValid: false,
			setupMocks: func() {
				suite.mockAuthSvc.EXPECT().
					GetGitHubClient("githubtools").
					Return(nil, nil)
			},
		},
		{
			name:          "ZeroDays",
			period:        "0d",
			shouldBeValid: false,
			setupMocks: func() {
				suite.mockAuthSvc.EXPECT().
					GetGitHubClient("githubtools").
					Return(nil, nil)
			},
		},
		{
			name:          "Valid_1Day",
			period:        "1d",
			shouldBeValid: true,
			setupMocks: func() {
				suite.mockAuthSvc.EXPECT().
					GetGitHubClient("githubtools").
					Return(nil, nil).
					Times(2)
				suite.mockAuthSvc.EXPECT().
					GetGitHubAccessToken("test-uuid", "githubtools").
					Return("test-token", nil)
			},
		},
		{
			name:          "Valid_7Days",
			period:        "7d",
			shouldBeValid: true,
			setupMocks: func() {
				suite.mockAuthSvc.EXPECT().
					GetGitHubClient("githubtools").
					Return(nil, nil).
					Times(2)
				suite.mockAuthSvc.EXPECT().
					GetGitHubAccessToken("test-uuid", "githubtools").
					Return("test-token", nil)
			},
		},
		{
			name:          "Valid_30Days",
			period:        "30d",
			shouldBeValid: true,
			setupMocks: func() {
				suite.mockAuthSvc.EXPECT().
					GetGitHubClient("githubtools").
					Return(nil, nil).
					Times(2)
				suite.mockAuthSvc.EXPECT().
					GetGitHubAccessToken("test-uuid", "githubtools").
					Return("test-token", nil)
			},
		},
		{
			name:          "Valid_90Days",
			period:        "90d",
			shouldBeValid: true,
			setupMocks: func() {
				suite.mockAuthSvc.EXPECT().
					GetGitHubClient("githubtools").
					Return(nil, nil).
					Times(2)
				suite.mockAuthSvc.EXPECT().
					GetGitHubAccessToken("test-uuid", "githubtools").
					Return("test-token", nil)
			},
		},
		{
			name:          "Valid_365Days",
			period:        "365d",
			shouldBeValid: true,
			setupMocks: func() {
				suite.mockAuthSvc.EXPECT().
					GetGitHubClient("githubtools").
					Return(nil, nil).
					Times(2)
				suite.mockAuthSvc.EXPECT().
					GetGitHubAccessToken("test-uuid", "githubtools").
					Return("test-token", nil)
			},
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			tc.setupMocks()

			_, err := suite.service.GetContributionsHeatmap(ctx, "test-uuid", "githubtools", tc.period)

			if tc.shouldBeValid {
				// Valid periods will fail at HTTP call (no mock server), but NOT at validation
				suite.Error(err)
				suite.NotContains(err.Error(), "period must be in format")
				suite.NotContains(err.Error(), "invalid period format")
			} else {
				// Invalid periods should fail at validation
				suite.Error(err)
				suite.Contains(err.Error(), "invalid period format")
			}
		})
	}
}

func (suite *GitHubServiceTestSuite) TestClosePullRequest_MissingUserUUIDOrProvider() {
	ctx := context.Background()

// TestGetAveragePRMergeTime_VariousPeriods tests various valid period formats
func (suite *GitHubServiceTestSuite) TestGetAveragePRMergeTime_VariousPeriods() {
	mockCache := cache.NewInMemoryCache(5*time.Minute, 10*time.Minute)
githubService := service.NewGitHubService(nil, mockCache)

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			_, err := suite.service.ClosePullRequest(ctx, tc.userUUID, tc.provider, "owner", "repo", 1, false)
			suite.Error(err)
			suite.ErrorIs(err, apperrors.ErrMissingUserUUIDAndProvider)
		})
	}
}

func (suite *GitHubServiceTestSuite) TestClosePullRequest_MissingOwnerOrRepo() {
	ctx := context.Background()

	testCases := []struct {
		name          string
		owner         string
		repo          string
		expectedError string
	}{
		{
			name:          "BothEmpty",
			owner:         "",
			repo:          "",
			expectedError: "owner and repository are required",
		},
		{
			name:          "EmptyOwner",
			owner:         "",
			repo:          "repo",
			expectedError: "owner and repository are required",
		},
		{
			name:          "EmptyRepo",
			owner:         "owner",
			repo:          "",
			expectedError: "owner and repository are required",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			_, err := suite.service.ClosePullRequest(ctx, "test-uuid", "githubtools", tc.owner, tc.repo, 1, false)
			suite.Error(err)
			suite.ErrorIs(err, apperrors.ErrOwnerAndRepositoryMissing)
		})
	}
}

func (suite *GitHubServiceTestSuite) TestClosePullRequest_GetAccessTokenErrors() {
	ctx := context.Background()

	testCases := []struct {
		name          string
		mockError     error
		expectedError string
	}{
		{
			name:      "TokenStoreNotInitialized",
			mockError: apperrors.ErrTokenStoreNotInitialized,
		},
		{
			name:      "AuthServiceNotInitialized",
			mockError: apperrors.ErrAuthServiceNotInitialized,
		},
		{
			name:      "UserUUIDMissing",
			mockError: apperrors.ErrUserUUIDMissing,
		},
		{
			name:      "ProviderMissing",
			mockError: apperrors.ErrProviderMissing,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.mockAuthSvc.EXPECT().
				GetGitHubAccessToken("test-uuid", "githubtools").
				Return("", tc.mockError)

			_, err := suite.service.ClosePullRequest(ctx, "test-uuid", "githubtools", "owner", "repo", 1, false)
			suite.Error(err)
			suite.ErrorIs(err, tc.mockError)
		})
	}
}

func (suite *GitHubServiceTestSuite) TestClosePullRequest_GetClientErrors() {
	ctx := context.Background()

	testCases := []struct {
		name      string
		mockError error
	}{
		{
			name:      "AuthServiceNotInitialized",
			mockError: apperrors.ErrAuthServiceNotInitialized,
		},
		{
			name:      "ClientNotFoundForProvider",
			mockError: fmt.Errorf("GitHub client not found for provider githubtools"),
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.mockAuthSvc.EXPECT().
				GetGitHubAccessToken("test-uuid", "githubtools").
				Return("test-token", nil)

			suite.mockAuthSvc.EXPECT().
				GetGitHubClient("githubtools").
				Return(nil, tc.mockError)

			_, err := suite.service.ClosePullRequest(ctx, "test-uuid", "githubtools", "owner", "repo", 1, false)
			suite.Error(err)
			suite.Contains(err.Error(), "failed to get GitHub client")
		})
	}
}

// ClosePullRequest tests

// TestClosePullRequest_Success_WithBranchDeletion verifies closing an open PR and deleting its branch
func TestClosePullRequest_Success_WithBranchDeletion(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	capturedDelete := false

	// Mock GitHub API server
	var mockGitHubServer *httptest.Server
	mockGitHubServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Handle GET PR
		if r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/repos/owner/repo/pulls/42") {
			resp := map[string]interface{}{
				"id":         int64(123456789),
				"number":     42,
				"title":      "Test PR",
				"state":      "open",
				"created_at": "2025-01-01T12:00:00Z",
				"updated_at": "2025-01-01T12:00:00Z",
				"html_url":   mockGitHubServer.URL + "/owner/repo/pull/42",
				"draft":      false,
				"user": map[string]interface{}{
					"login":      "testuser",
					"id":         int64(12345),
					"avatar_url": "https://avatars.githubusercontent.com/u/12345",
				},
				"head": map[string]interface{}{
					"ref": "feature-branch",
					"repo": map[string]interface{}{
						"name": "repo",
						"owner": map[string]interface{}{
							"login": "owner",
						},
					},
				},
			}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}

		// Handle PATCH (close PR)
		if r.Method == http.MethodPatch && strings.Contains(r.URL.Path, "/repos/owner/repo/pulls/42") {
			resp := map[string]interface{}{
				"id":         int64(123456789),
				"number":     42,
				"title":      "Test PR",
				"state":      "closed",
				"created_at": "2025-01-01T12:00:00Z",
				"updated_at": "2025-01-02T12:00:00Z",
				"html_url":   mockGitHubServer.URL + "/owner/repo/pull/42",
				"draft":      false,
				"user": map[string]interface{}{
					"login":      "testuser",
					"id":         int64(12345),
					"avatar_url": "https://avatars.githubusercontent.com/u/12345",
				},
				"head": map[string]interface{}{
					"ref": "feature-branch",
					"repo": map[string]interface{}{
						"name": "repo",
						"owner": map[string]interface{}{
							"login": "owner",
						},
					},
				},
			}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}

		// Handle DELETE ref (branch deletion)
		if r.Method == http.MethodDelete && strings.Contains(r.URL.Path, "/repos/owner/repo/git/refs/heads/feature-branch") {
			capturedDelete = true
			w.WriteHeader(http.StatusNoContent)
			return
		}

		// Fallback
		w.WriteHeader(http.StatusNotFound)
	}))
	defer mockGitHubServer.Close()

	// Mock auth service
	mockAuthService := mocks.NewMockGitHubAuthService(ctrl)
	mockAuthService.EXPECT().GetGitHubAccessTokenFromClaims(gomock.Any()).Return("token", nil)
	envConfig := &auth.ProviderConfig{EnterpriseBaseURL: mockGitHubServer.URL}
	mockAuthService.EXPECT().GetGitHubClient(gomock.Any()).Return(auth.NewGitHubClient(envConfig), nil)

	// Service under test
	mockCache := cache.NewInMemoryCache(5*time.Minute, 10*time.Minute)
	githubService := service.NewGitHubServiceWithAdapter(mockAuthService, mockCache)
	claims := &auth.AuthClaims{UserID: 123, Provider: "githubtools"}

	// Execute
	result, err := githubService.ClosePullRequest(context.Background(), claims, "owner", "repo", 42, true)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "closed", result.State)
	assert.Equal(t, 42, result.Number)
	assert.Equal(t, "owner", result.Repo.Owner)
	assert.Equal(t, "repo", result.Repo.Name)
	assert.True(t, capturedDelete, "branch deletion should be attempted")
}

// TestClosePullRequest_AlreadyClosed verifies error when PR is already closed
func TestClosePullRequest_AlreadyClosed(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	var mockGitHubServer *httptest.Server
	mockGitHubServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/repos/owner/repo/pulls/99") {
			w.Header().Set("Content-Type", "application/json")
			resp := map[string]interface{}{
				"id":         int64(111),
				"number":     99,
				"title":      "Closed PR",
				"state":      "closed",
				"created_at": "2025-01-01T12:00:00Z",
				"updated_at": "2025-01-05T12:00:00Z",
				"html_url":   mockGitHubServer.URL + "/owner/repo/pull/99",
				"user": map[string]interface{}{
					"login":      "testuser",
					"id":         int64(12345),
					"avatar_url": "https://avatars.githubusercontent.com/u/12345",
				},
			}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer mockGitHubServer.Close()

	mockAuthService := mocks.NewMockGitHubAuthService(ctrl)
	mockAuthService.EXPECT().GetGitHubAccessTokenFromClaims(gomock.Any()).Return("token", nil)
	envConfig := &auth.ProviderConfig{EnterpriseBaseURL: mockGitHubServer.URL}
	mockAuthService.EXPECT().GetGitHubClient(gomock.Any()).Return(auth.NewGitHubClient(envConfig), nil)

	mockCache := cache.NewInMemoryCache(5*time.Minute, 10*time.Minute)
	githubService := service.NewGitHubServiceWithAdapter(mockAuthService, mockCache)
	claims := &auth.AuthClaims{UserID: 123, Provider: "githubtools"}

	result, err := githubService.ClosePullRequest(context.Background(), claims, "owner", "repo", 99, true)

	require.Error(t, err)
	require.Nil(t, result)
	assert.Contains(t, err.Error(), "already closed")
}

// TestClosePullRequest_NotFound verifies not found error when PR does not exist
func TestClosePullRequest_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	var mockGitHubServer *httptest.Server
	mockGitHubServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/repos/owner/repo/pulls/7") {
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{"message": "Not Found"})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer mockGitHubServer.Close()

	mockAuthService := mocks.NewMockGitHubAuthService(ctrl)
	mockAuthService.EXPECT().GetGitHubAccessTokenFromClaims(gomock.Any()).Return("token", nil)
	envConfig := &auth.ProviderConfig{EnterpriseBaseURL: mockGitHubServer.URL}
	mockAuthService.EXPECT().GetGitHubClient(gomock.Any()).Return(auth.NewGitHubClient(envConfig), nil)

	mockCache := cache.NewInMemoryCache(5*time.Minute, 10*time.Minute)
	githubService := service.NewGitHubServiceWithAdapter(mockAuthService, mockCache)
	claims := &auth.AuthClaims{UserID: 123, Provider: "githubtools"}

	result, err := githubService.ClosePullRequest(context.Background(), claims, "owner", "repo", 7, true)

	require.Error(t, err)
	require.Nil(t, result)
	assert.Contains(t, err.Error(), "not found")
}

// TestClosePullRequest_RateLimitOnGet verifies rate limit on initial PR fetch
func TestClosePullRequest_RateLimitOnGet(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	var mockGitHubServer *httptest.Server
	mockGitHubServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/repos/owner/repo/pulls/1") {
			w.WriteHeader(http.StatusForbidden)
			_ = json.NewEncoder(w).Encode(map[string]string{"message": "API rate limit exceeded"})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer mockGitHubServer.Close()

	mockAuthService := mocks.NewMockGitHubAuthService(ctrl)
	mockAuthService.EXPECT().GetGitHubAccessTokenFromClaims(gomock.Any()).Return("token", nil)
	envConfig := &auth.ProviderConfig{EnterpriseBaseURL: mockGitHubServer.URL}
	mockAuthService.EXPECT().GetGitHubClient(gomock.Any()).Return(auth.NewGitHubClient(envConfig), nil)

	mockCache := cache.NewInMemoryCache(5*time.Minute, 10*time.Minute)
	githubService := service.NewGitHubServiceWithAdapter(mockAuthService, mockCache)
	claims := &auth.AuthClaims{UserID: 123, Provider: "githubtools"}

	result, err := githubService.ClosePullRequest(context.Background(), claims, "owner", "repo", 1, true)

	require.Error(t, err)
	require.Nil(t, result)
	assert.Contains(t, err.Error(), "rate limit")
}

// TestClosePullRequest_DeleteBranch404Ignored verifies 404 during branch deletion is ignored
func TestClosePullRequest_DeleteBranch404Ignored(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	var mockGitHubServer *httptest.Server
	mockGitHubServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// GET open PR
		if r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/repos/owner/repo/pulls/50") {
			w.Header().Set("Content-Type", "application/json")
			resp := map[string]interface{}{
				"id":         int64(500),
				"number":     50,
				"title":      "PR to close",
				"state":      "open",
				"created_at": "2025-01-01T12:00:00Z",
				"updated_at": "2025-01-01T12:00:00Z",
				"html_url":   mockGitHubServer.URL + "/owner/repo/pull/50",
				"user": map[string]interface{}{
					"login":      "testuser",
					"id":         int64(12345),
					"avatar_url": "https://avatars.githubusercontent.com/u/12345",
				},
				"head": map[string]interface{}{
					"ref": "feature-branch",
					"repo": map[string]interface{}{
						"name": "repo",
						"owner": map[string]interface{}{
							"login": "owner",
						},
					},
				},
			}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}
		// PATCH close PR
		if r.Method == http.MethodPatch && strings.Contains(r.URL.Path, "/repos/owner/repo/pulls/50") {
			w.Header().Set("Content-Type", "application/json")
			resp := map[string]interface{}{
				"id":         int64(500),
				"number":     50,
				"title":      "PR to close",
				"state":      "closed",
				"created_at": "2025-01-01T12:00:00Z",
				"updated_at": "2025-01-02T12:00:00Z",
				"html_url":   mockGitHubServer.URL + "/owner/repo/pull/50",
				"user": map[string]interface{}{
					"login":      "testuser",
					"id":         int64(12345),
					"avatar_url": "https://avatars.githubusercontent.com/u/12345",
				},
				"head": map[string]interface{}{
					"ref": "feature-branch",
					"repo": map[string]interface{}{
						"name": "repo",
						"owner": map[string]interface{}{
							"login": "owner",
						},
					},
				},
			}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}
		// DELETE branch -> 404 (ignored by service)
		if r.Method == http.MethodDelete && strings.Contains(r.URL.Path, "/repos/owner/repo/git/refs/heads/feature-branch") {
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{"message": "branch not found"})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer mockGitHubServer.Close()

	mockAuthService := mocks.NewMockGitHubAuthService(ctrl)
	mockAuthService.EXPECT().GetGitHubAccessTokenFromClaims(gomock.Any()).Return("token", nil)
	envConfig := &auth.ProviderConfig{EnterpriseBaseURL: mockGitHubServer.URL}
	mockAuthService.EXPECT().GetGitHubClient(gomock.Any()).Return(auth.NewGitHubClient(envConfig), nil)

	mockCache := cache.NewInMemoryCache(5*time.Minute, 10*time.Minute)
	githubService := service.NewGitHubServiceWithAdapter(mockAuthService, mockCache)
	claims := &auth.AuthClaims{UserID: 123, Provider: "githubtools"}

	result, err := githubService.ClosePullRequest(context.Background(), claims, "owner", "repo", 50, true)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "closed", result.State)
}

			if tc.shouldFailAtSetup {
				// Error should occur during client setup
				suite.NotContains(err.Error(), "failed to get user")
			}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}
		// DELETE branch -> 500 error (should bubble up)
		if r.Method == http.MethodDelete && strings.Contains(r.URL.Path, "/repos/owner/repo/git/refs/heads/feature-branch") {
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]string{"message": "internal error"})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer mockGitHubServer.Close()

	mockAuthService := mocks.NewMockGitHubAuthService(ctrl)
	mockAuthService.EXPECT().GetGitHubAccessTokenFromClaims(gomock.Any()).Return("token", nil)
	envConfig := &auth.ProviderConfig{EnterpriseBaseURL: mockGitHubServer.URL}
	mockAuthService.EXPECT().GetGitHubClient(gomock.Any()).Return(auth.NewGitHubClient(envConfig), nil)

	mockCache := cache.NewInMemoryCache(5*time.Minute, 10*time.Minute)
	githubService := service.NewGitHubServiceWithAdapter(mockAuthService, mockCache)
	claims := &auth.AuthClaims{UserID: 123, Provider: "githubtools"}

	result, err := githubService.ClosePullRequest(context.Background(), claims, "owner", "repo", 77, true)

	require.Error(t, err)
	require.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to delete branch")
}

// TestClosePullRequest_InputValidation verifies input validation errors
func TestClosePullRequest_InputValidation(t *testing.T) {
	mockCache := cache.NewInMemoryCache(5*time.Minute, 10*time.Minute)
githubService := service.NewGitHubService(nil, mockCache)

	// Nil claims
	res, err := githubService.ClosePullRequest(context.Background(), nil, "owner", "repo", 1, false)
	require.Error(t, err)
	require.Nil(t, res)
	assert.Contains(t, err.Error(), "authentication required")

	// Missing owner/repo
	claims := &auth.AuthClaims{UserID: 1, Provider: "githubtools"}
	res, err = githubService.ClosePullRequest(context.Background(), claims, "", "repo", 1, false)
	require.Error(t, err)
	require.Nil(t, res)
	assert.Contains(t, err.Error(), "owner and repo are required")
}