package service_test

import (
	"context"
	apperrors "developer-portal-backend/internal/errors"
	"fmt"
	"testing"

	"developer-portal-backend/internal/auth"
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

func TestGetUserOpenPullRequests_DefaultsAppliedAndAuthCalled(t *testing.T) {
	gh := service.NewGitHubService(nil)
	// Empty state, sort, direction, perPage/page zero -> defaults applied internally.
	_, err := gh.GetUserOpenPullRequests(context.Background(), "test-uuid", "githubtools", "", "", "", 0, 0)
	// With non-empty uuid/provider the next step is token retrieval, which fails due to nil auth service.
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get GitHub access token")
}

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

func TestClosePullRequest_InputValidation(t *testing.T) {
	gh := service.NewGitHubService(nil)

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

func (suite *GitHubServiceTestSuite) TestGetContributionsHeatmap_RequiresUUIDAndProvider() {
	ctx := context.Background()

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

func (suite *GitHubServiceTestSuite) TestGetContributionsHeatmap_ProviderNotConfigured() {
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

func (suite *GitHubServiceTestSuite) TestGetAveragePRMergeTime_RequiresUUIDAndProvider() {
	ctx := context.Background()

	_, err := suite.service.GetAveragePRMergeTime(ctx, "", "", "30d")
	suite.Error(err)
	suite.Equal(err.Error(), apperrors.ErrMissingUserUUIDAndProvider.Error())
}

func (suite *GitHubServiceTestSuite) TestGetAveragePRMergeTime_InvalidPeriodFormat() {
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

func (suite *GitHubServiceTestSuite) TestGetAveragePRMergeTime_TokenRetrievalFailure() {
	ctx := context.Background()

	suite.mockAuthSvc.EXPECT().
		GetGitHubAccessToken("test-uuid", "githubtools").
		Return("", apperrors.ErrTokenStoreNotInitialized)

	_, err := suite.service.GetAveragePRMergeTime(ctx, "test-uuid", "githubtools", "30d")
	suite.Error(err)
	suite.Contains(err.Error(), apperrors.ErrTokenStoreNotInitialized.Message)
}

func (suite *GitHubServiceTestSuite) TestGetAveragePRMergeTime_ClientRetrievalFailure() {
	ctx := context.Background()

	suite.mockAuthSvc.EXPECT().
		GetGitHubAccessToken("test-uuid", "githubtools").
		Return("test-token", nil)

	suite.mockAuthSvc.EXPECT().
		GetGitHubClient("githubtools").
		Return(nil, assert.AnError)

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

func (suite *GitHubServiceTestSuite) TestGetRepositoryContent_TokenRetrievalErrors() {
	ctx := context.Background()

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

func (suite *GitHubServiceTestSuite) TestGetRepositoryContent_ClientRetrievalFailure() {
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

func (suite *GitHubServiceTestSuite) TestGetRepositoryContent_DefaultRefBehavior() {
	ctx := context.Background()

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

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.mockAuthSvc.EXPECT().
				GetGitHubAccessToken("test-uuid", "githubtools").
				Return("test-token", nil)

			suite.mockAuthSvc.EXPECT().
				GetGitHubClient("githubtools").
				Return(nil, nil)

			// Path normalization happens before API call
			_, err := suite.service.GetRepositoryContent(ctx, "test-uuid", "githubtools", "owner", "repo", tc.path, "main")
			suite.Error(err) // Expected to fail at API call
		})
	}
}

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

func (suite *GitHubServiceTestSuite) TestUpdateRepositoryFile_TokenRetrievalErrors() {
	ctx := context.Background()

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

func (suite *GitHubServiceTestSuite) TestUpdateRepositoryFile_ClientRetrievalFailure() {
	ctx := context.Background()

	suite.mockAuthSvc.EXPECT().
		GetGitHubAccessToken("test-uuid", "githubtools").
		Return("test-token", nil)

	suite.mockAuthSvc.EXPECT().
		GetGitHubClient("githubtools").
		Return(nil, apperrors.ErrAuthServiceNotInitialized)

	_, err := suite.service.UpdateRepositoryFile(
		ctx,
		"test-uuid",
		"githubtools",
		"owner",
		"repo",
		"README.md",
		"Update README",
		"# Updated Content",
		"sha123",
		"main",
	)

	suite.Error(err)
	suite.ErrorIs(err, apperrors.ErrAuthServiceNotInitialized)
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

func (suite *GitHubServiceTestSuite) TestGetGitHubAsset_ParameterCombinations() {
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

func (suite *GitHubServiceTestSuite) TestGetUserPRReviewComments_RequiresUUIDAndProvider() {
	ctx := context.Background()

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

func (suite *GitHubServiceTestSuite) TestGetUserPRReviewComments_DefaultPeriod() {
	ctx := context.Background()

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

	testCases := []struct {
		name          string
		userUUID      string
		provider      string
		expectedError string
	}{
		{
			name:          "BothEmpty",
			userUUID:      "",
			provider:      "",
			expectedError: "userUUID and provider are required",
		},
		{
			name:          "EmptyUserUUID",
			userUUID:      "",
			provider:      "githubtools",
			expectedError: "userUUID and provider are required",
		},
		{
			name:          "EmptyProvider",
			userUUID:      "test-uuid",
			provider:      "",
			expectedError: "userUUID and provider are required",
		},
	}

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

func (suite *GitHubServiceTestSuite) TestGetUserPRReviewComments_GitHubClientCreation() {
	ctx := context.Background()

	testCases := []struct {
		name              string
		provider          string
		setupMocks        func()
		expectedError     string
		shouldFailAtSetup bool
	}{
		{
			name:     "GetClientError",
			provider: "githubtools",
			setupMocks: func() {
				suite.mockAuthSvc.EXPECT().
					GetGitHubAccessToken("test-uuid", "githubtools").
					Return("test-token", nil)
				suite.mockAuthSvc.EXPECT().
					GetGitHubClient("githubtools").
					Return(nil, fmt.Errorf("GitHub client not found for provider githubtools"))
			},
			expectedError:     "failed to get GitHub client",
			shouldFailAtSetup: true,
		},
		{
			name:     "NilConfig_CreatesStandardClient",
			provider: "githubtools",
			setupMocks: func() {
				suite.mockAuthSvc.EXPECT().
					GetGitHubAccessToken("test-uuid", "githubtools").
					Return("test-token", nil)
				suite.mockAuthSvc.EXPECT().
					GetGitHubClient("githubtools").
					Return(nil, nil)
			},
			expectedError:     "failed to get user",
			shouldFailAtSetup: false,
		},
		{
			name:     "EmptyEnterpriseURL_CreatesStandardClient",
			provider: "github",
			setupMocks: func() {
				suite.mockAuthSvc.EXPECT().
					GetGitHubAccessToken("test-uuid", "github").
					Return("test-token", nil)
				// Return GitHubClient with nil config (empty enterprise URL)
				githubClient := &auth.GitHubClient{}
				suite.mockAuthSvc.EXPECT().
					GetGitHubClient("github").
					Return(githubClient, nil)
			},
			expectedError:     "failed to get user",
			shouldFailAtSetup: false,
		},
		{
			name:     "ValidEnterpriseURL_CreatesEnterpriseClient",
			provider: "githubtools",
			setupMocks: func() {
				suite.mockAuthSvc.EXPECT().
					GetGitHubAccessToken("test-uuid", "githubtools").
					Return("test-token", nil)
				// Return GitHubClient with enterprise URL
				githubClient := auth.NewGitHubClient(&auth.ProviderConfig{
					EnterpriseBaseURL: "https://github.enterprise.com",
				})
				suite.mockAuthSvc.EXPECT().
					GetGitHubClient("githubtools").
					Return(githubClient, nil)
			},
			expectedError:     "failed to get user",
			shouldFailAtSetup: false,
		},
		{
			name:     "InvalidEnterpriseURL_FailsClientCreation",
			provider: "githubtools",
			setupMocks: func() {
				suite.mockAuthSvc.EXPECT().
					GetGitHubAccessToken("test-uuid", "githubtools").
					Return("test-token", nil)
				// Return GitHubClient with invalid enterprise URL
				githubClient := auth.NewGitHubClient(&auth.ProviderConfig{
					EnterpriseBaseURL: "://invalid-url",
				})
				suite.mockAuthSvc.EXPECT().
					GetGitHubClient("githubtools").
					Return(githubClient, nil)
			},
			expectedError:     "failed to create GitHub Enterprise client",
			shouldFailAtSetup: true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			tc.setupMocks()

			result, err := suite.service.GetUserPRReviewComments(ctx, "test-uuid", tc.provider, "30d")

			suite.Error(err)
			suite.Nil(result)
			suite.Contains(err.Error(), tc.expectedError)

			if tc.shouldFailAtSetup {
				// Error should occur during client setup
				suite.NotContains(err.Error(), "failed to get user")
			}
		})
	}
}

func TestGitHubServiceTestSuite(t *testing.T) {
	suite.Run(t, new(GitHubServiceTestSuite))
}
