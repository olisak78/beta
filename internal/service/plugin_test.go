package service

import (
	"context"
	"errors"
	"testing"

	"developer-portal-backend/internal/database/models"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockPluginRepository is a mock implementation of PluginRepositoryInterface
type MockPluginRepository struct {
	mock.Mock
}

func (m *MockPluginRepository) Create(plugin *models.Plugin) error {
	args := m.Called(plugin)
	return args.Error(0)
}

func (m *MockPluginRepository) GetByID(id uuid.UUID) (*models.Plugin, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Plugin), args.Error(1)
}

func (m *MockPluginRepository) GetByName(name string) (*models.Plugin, error) {
	args := m.Called(name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Plugin), args.Error(1)
}

func (m *MockPluginRepository) GetAll(limit, offset int) ([]models.Plugin, int64, error) {
	args := m.Called(limit, offset)
	return args.Get(0).([]models.Plugin), args.Get(1).(int64), args.Error(2)
}

func (m *MockPluginRepository) Update(plugin *models.Plugin) error {
	args := m.Called(plugin)
	return args.Error(0)
}

func (m *MockPluginRepository) Delete(id uuid.UUID) error {
	args := m.Called(id)
	return args.Error(0)
}

// MockUserRepository is a mock implementation of UserRepositoryInterface
type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) GetByName(name string) (*models.User, error) {
	args := m.Called(name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepository) Create(user *models.User) error {
	args := m.Called(user)
	return args.Error(0)
}

func (m *MockUserRepository) GetByID(id uuid.UUID) (*models.User, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepository) GetAll(limit, offset int) ([]models.User, int64, error) {
	args := m.Called(limit, offset)
	return args.Get(0).([]models.User), args.Get(1).(int64), args.Error(2)
}

func (m *MockUserRepository) Update(user *models.User) error {
	args := m.Called(user)
	return args.Error(0)
}

func (m *MockUserRepository) Delete(id uuid.UUID) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockUserRepository) GetByEmail(email string) (*models.User, error) {
	args := m.Called(email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepository) GetByUserID(userID string) (*models.User, error) {
	args := m.Called(userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepository) GetByOrganizationID(orgID uuid.UUID, limit, offset int) ([]models.User, int64, error) {
	args := m.Called(orgID, limit, offset)
	return args.Get(0).([]models.User), args.Get(1).(int64), args.Error(2)
}

func (m *MockUserRepository) GetByTeamID(teamID uuid.UUID, limit, offset int) ([]models.User, int64, error) {
	args := m.Called(teamID, limit, offset)
	return args.Get(0).([]models.User), args.Get(1).(int64), args.Error(2)
}

func (m *MockUserRepository) GetWithOrganization(id uuid.UUID) (*models.User, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepository) SearchByOrganization(orgID uuid.UUID, query string, limit, offset int) ([]models.User, int64, error) {
	args := m.Called(orgID, query, limit, offset)
	return args.Get(0).([]models.User), args.Get(1).(int64), args.Error(2)
}

func (m *MockUserRepository) SearchByNameOrTitleGlobal(query string, limit, offset int) ([]models.User, int64, error) {
	args := m.Called(query, limit, offset)
	return args.Get(0).([]models.User), args.Get(1).(int64), args.Error(2)
}

func (m *MockUserRepository) GetActiveByOrganization(orgID uuid.UUID, limit, offset int) ([]models.User, int64, error) {
	args := m.Called(orgID, limit, offset)
	return args.Get(0).([]models.User), args.Get(1).(int64), args.Error(2)
}

func (m *MockUserRepository) GetUserIDsByPrefix(prefix string) ([]string, error) {
	args := m.Called(prefix)
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockUserRepository) GetExistingUserIDs(ids []string) ([]string, error) {
	args := m.Called(ids)
	return args.Get(0).([]string), args.Error(1)
}

// MockGitHubService is a mock implementation of GitHubServiceInterface
type MockGitHubService struct {
	mock.Mock
}

func (m *MockGitHubService) GetUserOpenPullRequests(ctx context.Context, uuid, provider, state, sort, direction string, perPage, page int) (*PullRequestsResponse, error) {
	args := m.Called(ctx, uuid, provider, state, sort, direction, perPage, page)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*PullRequestsResponse), args.Error(1)
}

func (m *MockGitHubService) GetUserTotalContributions(ctx context.Context, uuid, provider, period string) (*TotalContributionsResponse, error) {
	args := m.Called(ctx, uuid, provider, period)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*TotalContributionsResponse), args.Error(1)
}

func (m *MockGitHubService) GetContributionsHeatmap(ctx context.Context, uuid, provider, period string) (*ContributionsHeatmapResponse, error) {
	args := m.Called(ctx, uuid, provider, period)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ContributionsHeatmapResponse), args.Error(1)
}

func (m *MockGitHubService) GetAveragePRMergeTime(ctx context.Context, uuid, provider, period string) (*AveragePRMergeTimeResponse, error) {
	args := m.Called(ctx, uuid, provider, period)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*AveragePRMergeTimeResponse), args.Error(1)
}

func (m *MockGitHubService) GetUserPRReviewComments(ctx context.Context, uuid, provider, period string) (*PRReviewCommentsResponse, error) {
	args := m.Called(ctx, uuid, provider, period)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*PRReviewCommentsResponse), args.Error(1)
}

func (m *MockGitHubService) GetRepositoryContent(ctx context.Context, userUUID, provider, owner, repo, path, ref string) (interface{}, error) {
	args := m.Called(ctx, userUUID, provider, owner, repo, path, ref)
	return args.Get(0), args.Error(1)
}

func (m *MockGitHubService) UpdateRepositoryFile(ctx context.Context, uuid, provider, owner, repo, path, message, content, sha, branch string) (interface{}, error) {
	args := m.Called(ctx, uuid, provider, owner, repo, path, message, content, sha, branch)
	return args.Get(0), args.Error(1)
}

func (m *MockGitHubService) ClosePullRequest(ctx context.Context, uuid, provider, owner, repo string, prNumber int, deleteBranch bool) (*PullRequest, error) {
	args := m.Called(ctx, uuid, provider, owner, repo, prNumber, deleteBranch)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*PullRequest), args.Error(1)
}

func (m *MockGitHubService) GetGitHubAsset(ctx context.Context, uuid, provider, assetURL string) ([]byte, string, error) {
	args := m.Called(ctx, uuid, provider, assetURL)
	return args.Get(0).([]byte), args.Get(1).(string), args.Error(2)
}

func TestNewPluginService(t *testing.T) {
	mockPluginRepo := new(MockPluginRepository)
	mockUserRepo := new(MockUserRepository)
	validator := validator.New()

	service := NewPluginService(mockPluginRepo, mockUserRepo, validator)

	assert.NotNil(t, service)
	assert.Equal(t, mockPluginRepo, service.pluginRepo)
	assert.Equal(t, mockUserRepo, service.userRepo)
	assert.Equal(t, validator, service.validator)
}

func TestPluginService_GetAllPlugins(t *testing.T) {
	tests := []struct {
		name           string
		limit          int
		offset         int
		mockPlugins    []models.Plugin
		mockTotal      int64
		mockError      error
		expectedLimit  int
		expectedOffset int
		expectError    bool
	}{
		{
			name:   "successful request with default pagination",
			limit:  0,
			offset: 0,
			mockPlugins: []models.Plugin{
				{
					BaseModel: models.BaseModel{
						ID:          uuid.New(),
						Name:        "test-plugin",
						Title:       "Test Plugin",
						Description: "A test plugin",
					},
					Icon:               "TestIcon",
					ReactComponentPath: "/plugins/test/Test.jsx",
					BackendServerURL:   "http://localhost:3001",
					Owner:              "Test Team",
				},
			},
			mockTotal:      1,
			mockError:      nil,
			expectedLimit:  20,
			expectedOffset: 0,
			expectError:    false,
		},
		{
			name:           "successful request with custom pagination",
			limit:          10,
			offset:         5,
			mockPlugins:    []models.Plugin{},
			mockTotal:      0,
			mockError:      nil,
			expectedLimit:  10,
			expectedOffset: 5,
			expectError:    false,
		},
		{
			name:           "negative limit defaults to 20",
			limit:          -5,
			offset:         0,
			mockPlugins:    []models.Plugin{},
			mockTotal:      0,
			mockError:      nil,
			expectedLimit:  20,
			expectedOffset: 0,
			expectError:    false,
		},
		{
			name:           "negative offset defaults to 0",
			limit:          20,
			offset:         -10,
			mockPlugins:    []models.Plugin{},
			mockTotal:      0,
			mockError:      nil,
			expectedLimit:  20,
			expectedOffset: 0,
			expectError:    false,
		},
		{
			name:           "repository error",
			limit:          20,
			offset:         0,
			mockPlugins:    nil,
			mockTotal:      0,
			mockError:      errors.New("database error"),
			expectedLimit:  20,
			expectedOffset: 0,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockPluginRepo := new(MockPluginRepository)
			mockUserRepo := new(MockUserRepository)
			validator := validator.New()
			service := NewPluginService(mockPluginRepo, mockUserRepo, validator)

			mockPluginRepo.On("GetAll", tt.expectedLimit, tt.expectedOffset).Return(tt.mockPlugins, tt.mockTotal, tt.mockError)

			result, err := service.GetAllPlugins(tt.limit, tt.offset)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.mockTotal, result.Total)
				assert.Equal(t, tt.expectedLimit, result.Limit)
				assert.Equal(t, tt.expectedOffset, result.Offset)
				assert.Equal(t, len(tt.mockPlugins), len(result.Plugins))
			}

			mockPluginRepo.AssertExpectations(t)
		})
	}
}

func TestPluginService_GetAllPluginsWithViewer(t *testing.T) {
	pluginID1 := uuid.New()
	pluginID2 := uuid.New()
	userID := uuid.New()

	tests := []struct {
		name               string
		limit              int
		offset             int
		viewerName         string
		mockUser           *models.User
		mockUserError      error
		mockPlugins        []models.Plugin
		mockTotal          int64
		mockPluginsError   error
		expectedSubscribed []bool
		expectError        bool
		expectFallback     bool
	}{
		{
			name:       "successful request with subscribed plugins",
			limit:      20,
			offset:     0,
			viewerName: "testuser",
			mockUser: &models.User{
				BaseModel: models.BaseModel{
					ID:   userID,
					Name: "testuser",
				},
				Metadata: []byte(`{"subscribed":["` + pluginID1.String() + `"]}`),
			},
			mockUserError: nil,
			mockPlugins: []models.Plugin{
				{
					BaseModel: models.BaseModel{
						ID:   pluginID1,
						Name: "subscribed-plugin",
					},
				},
				{
					BaseModel: models.BaseModel{
						ID:   pluginID2,
						Name: "unsubscribed-plugin",
					},
				},
			},
			mockTotal:          2,
			mockPluginsError:   nil,
			expectedSubscribed: []bool{true, false},
			expectError:        false,
			expectFallback:     false,
		},
		{
			name:             "empty viewer name falls back to GetAllPlugins",
			limit:            20,
			offset:           0,
			viewerName:       "",
			mockUser:         nil,
			mockUserError:    nil,
			mockPlugins:      []models.Plugin{},
			mockTotal:        0,
			mockPluginsError: nil,
			expectError:      false,
			expectFallback:   true,
		},
		{
			name:             "whitespace viewer name falls back to GetAllPlugins",
			limit:            20,
			offset:           0,
			viewerName:       "   ",
			mockUser:         nil,
			mockUserError:    nil,
			mockPlugins:      []models.Plugin{},
			mockTotal:        0,
			mockPluginsError: nil,
			expectError:      false,
			expectFallback:   true,
		},
		{
			name:             "user not found falls back to GetAllPlugins",
			limit:            20,
			offset:           0,
			viewerName:       "nonexistent",
			mockUser:         nil,
			mockUserError:    errors.New("user not found"),
			mockPlugins:      []models.Plugin{},
			mockTotal:        0,
			mockPluginsError: nil,
			expectError:      false,
			expectFallback:   true,
		},
		{
			name:       "user with invalid metadata JSON",
			limit:      20,
			offset:     0,
			viewerName: "testuser",
			mockUser: &models.User{
				BaseModel: models.BaseModel{
					ID:   userID,
					Name: "testuser",
				},
				Metadata: []byte(`invalid json`),
			},
			mockUserError: nil,
			mockPlugins: []models.Plugin{
				{
					BaseModel: models.BaseModel{
						ID:   pluginID1,
						Name: "plugin1",
					},
				},
			},
			mockTotal:          1,
			mockPluginsError:   nil,
			expectedSubscribed: []bool{false},
			expectError:        false,
			expectFallback:     false,
		},
		{
			name:             "plugins repository error",
			limit:            20,
			offset:           0,
			viewerName:       "testuser",
			mockUser:         &models.User{BaseModel: models.BaseModel{ID: userID, Name: "testuser"}},
			mockUserError:    nil,
			mockPlugins:      nil,
			mockTotal:        0,
			mockPluginsError: errors.New("database error"),
			expectError:      true,
			expectFallback:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockPluginRepo := new(MockPluginRepository)
			mockUserRepo := new(MockUserRepository)
			validator := validator.New()
			service := NewPluginService(mockPluginRepo, mockUserRepo, validator)

			if tt.expectFallback {
				mockPluginRepo.On("GetAll", 20, 0).Return(tt.mockPlugins, tt.mockTotal, tt.mockPluginsError)
			} else {
				if tt.viewerName != "" && tt.viewerName != "   " {
					mockUserRepo.On("GetByName", tt.viewerName).Return(tt.mockUser, tt.mockUserError)
				}
				if tt.mockUser != nil || tt.mockUserError == nil {
					mockPluginRepo.On("GetAll", tt.limit, tt.offset).Return(tt.mockPlugins, tt.mockTotal, tt.mockPluginsError)
				}
			}

			// Special case: if user not found, we still need to set up the GetByName expectation
			// but it should fall back to GetAllPlugins
			if tt.viewerName == "nonexistent" && tt.mockUserError != nil {
				mockUserRepo.On("GetByName", tt.viewerName).Return(tt.mockUser, tt.mockUserError)
				mockPluginRepo.On("GetAll", 20, 0).Return(tt.mockPlugins, tt.mockTotal, tt.mockPluginsError)
			}

			result, err := service.GetAllPluginsWithViewer(tt.limit, tt.offset, tt.viewerName)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.mockTotal, result.Total)
				assert.Equal(t, len(tt.mockPlugins), len(result.Plugins))

				if tt.expectedSubscribed != nil {
					for i, expected := range tt.expectedSubscribed {
						if i < len(result.Plugins) {
							assert.Equal(t, expected, result.Plugins[i].Subscribed, "Plugin %d subscription mismatch", i)
						}
					}
				}
			}

			mockPluginRepo.AssertExpectations(t)
			mockUserRepo.AssertExpectations(t)
		})
	}
}

func TestPluginService_GetPluginByID(t *testing.T) {
	pluginID := uuid.New()

	tests := []struct {
		name        string
		pluginID    uuid.UUID
		mockPlugin  *models.Plugin
		mockError   error
		expectError bool
	}{
		{
			name:     "successful request",
			pluginID: pluginID,
			mockPlugin: &models.Plugin{
				BaseModel: models.BaseModel{
					ID:          pluginID,
					Name:        "test-plugin",
					Title:       "Test Plugin",
					Description: "A test plugin",
				},
				Icon:               "TestIcon",
				ReactComponentPath: "/plugins/test/Test.jsx",
				BackendServerURL:   "http://localhost:3001",
				Owner:              "Test Team",
			},
			mockError:   nil,
			expectError: false,
		},
		{
			name:        "plugin not found",
			pluginID:    pluginID,
			mockPlugin:  nil,
			mockError:   errors.New("record not found"),
			expectError: true,
		},
		{
			name:        "repository error",
			pluginID:    pluginID,
			mockPlugin:  nil,
			mockError:   errors.New("database error"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockPluginRepo := new(MockPluginRepository)
			mockUserRepo := new(MockUserRepository)
			validator := validator.New()
			service := NewPluginService(mockPluginRepo, mockUserRepo, validator)

			mockPluginRepo.On("GetByID", tt.pluginID).Return(tt.mockPlugin, tt.mockError)

			result, err := service.GetPluginByID(tt.pluginID)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.mockPlugin.ID, result.ID)
				assert.Equal(t, tt.mockPlugin.Name, result.Name)
				assert.Equal(t, tt.mockPlugin.Title, result.Title)
			}

			mockPluginRepo.AssertExpectations(t)
		})
	}
}

func TestPluginService_CreatePlugin(t *testing.T) {
	tests := []struct {
		name               string
		request            *CreatePluginRequest
		mockExistingPlugin *models.Plugin
		mockExistingError  error
		mockCreateError    error
		expectError        bool
		expectedErrorType  string
	}{
		{
			name: "successful creation",
			request: &CreatePluginRequest{
				Name:               "new-plugin",
				Title:              "New Plugin",
				Description:        "A new plugin",
				Icon:               "NewIcon",
				ReactComponentPath: "/plugins/new/New.jsx",
				BackendServerURL:   "http://localhost:3002",
				Owner:              "New Team",
			},
			mockExistingPlugin: nil,
			mockExistingError:  errors.New("record not found"),
			mockCreateError:    nil,
			expectError:        false,
		},
		{
			name: "validation error - missing required fields",
			request: &CreatePluginRequest{
				Description: "Missing required fields",
			},
			expectError:       true,
			expectedErrorType: "validation",
		},
		{
			name: "duplicate plugin name",
			request: &CreatePluginRequest{
				Name:               "existing-plugin",
				Title:              "Existing Plugin",
				Description:        "This plugin already exists",
				Icon:               "ExistingIcon",
				ReactComponentPath: "/plugins/existing/Existing.jsx",
				BackendServerURL:   "http://localhost:3003",
				Owner:              "Existing Team",
			},
			mockExistingPlugin: &models.Plugin{
				BaseModel: models.BaseModel{
					Name: "existing-plugin",
				},
			},
			mockExistingError: nil,
			expectError:       true,
			expectedErrorType: "validation",
		},
		{
			name: "repository create error",
			request: &CreatePluginRequest{
				Name:               "error-plugin",
				Title:              "Error Plugin",
				Description:        "This will cause an error",
				Icon:               "ErrorIcon",
				ReactComponentPath: "/plugins/error/Error.jsx",
				BackendServerURL:   "http://localhost:3004",
				Owner:              "Error Team",
			},
			mockExistingPlugin: nil,
			mockExistingError:  errors.New("record not found"),
			mockCreateError:    errors.New("database error"),
			expectError:        true,
			expectedErrorType:  "repository",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockPluginRepo := new(MockPluginRepository)
			mockUserRepo := new(MockUserRepository)
			validator := validator.New()
			service := NewPluginService(mockPluginRepo, mockUserRepo, validator)

			if tt.expectedErrorType != "validation" || tt.name == "duplicate plugin name" {
				mockPluginRepo.On("GetByName", tt.request.Name).Return(tt.mockExistingPlugin, tt.mockExistingError)
				if tt.mockExistingPlugin == nil {
					mockPluginRepo.On("Create", mock.AnythingOfType("*models.Plugin")).Return(tt.mockCreateError)
				}
			}

			result, err := service.CreatePlugin(tt.request)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
				if tt.expectedErrorType == "validation" {
					_, isValidationError := err.(*ValidationError)
					assert.True(t, isValidationError || err.Error() != "")
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.request.Name, result.Name)
				assert.Equal(t, tt.request.Title, result.Title)
			}

			mockPluginRepo.AssertExpectations(t)
		})
	}
}

func TestPluginService_UpdatePlugin(t *testing.T) {
	pluginID := uuid.New()
	existingPlugin := &models.Plugin{
		BaseModel: models.BaseModel{
			ID:          pluginID,
			Name:        "existing-plugin",
			Title:       "Existing Plugin",
			Description: "An existing plugin",
		},
		Icon:               "ExistingIcon",
		ReactComponentPath: "/plugins/existing/Existing.jsx",
		BackendServerURL:   "http://localhost:3001",
		Owner:              "Existing Team",
	}

	tests := []struct {
		name                string
		pluginID            uuid.UUID
		request             *UpdatePluginRequest
		mockExistingPlugin  *models.Plugin
		mockExistingError   error
		mockNameCheckPlugin *models.Plugin
		mockNameCheckError  error
		mockUpdateError     error
		expectError         bool
		expectedErrorType   string
		expectNameCheck     bool
	}{
		{
			name:     "successful update",
			pluginID: pluginID,
			request: &UpdatePluginRequest{
				Title:       stringPtr("Updated Plugin"),
				Description: stringPtr("Updated description"),
			},
			mockExistingPlugin: existingPlugin,
			mockExistingError:  nil,
			mockUpdateError:    nil,
			expectError:        false,
			expectNameCheck:    false,
		},
		{
			name:     "successful update with name change",
			pluginID: pluginID,
			request: &UpdatePluginRequest{
				Name: stringPtr("new-name"),
			},
			mockExistingPlugin:  existingPlugin,
			mockExistingError:   nil,
			mockNameCheckPlugin: nil,
			mockNameCheckError:  errors.New("record not found"),
			mockUpdateError:     nil,
			expectError:         false,
			expectNameCheck:     true,
		},
		{
			name:     "plugin not found",
			pluginID: pluginID,
			request: &UpdatePluginRequest{
				Title: stringPtr("Updated Title"),
			},
			mockExistingPlugin: nil,
			mockExistingError:  errors.New("record not found"),
			expectError:        true,
			expectedErrorType:  "not_found",
		},
		{
			name:     "duplicate name error",
			pluginID: pluginID,
			request: &UpdatePluginRequest{
				Name: stringPtr("duplicate-name"),
			},
			mockExistingPlugin: existingPlugin,
			mockExistingError:  nil,
			mockNameCheckPlugin: &models.Plugin{
				BaseModel: models.BaseModel{
					Name: "duplicate-name",
				},
			},
			mockNameCheckError: nil,
			expectError:        true,
			expectedErrorType:  "validation",
			expectNameCheck:    true,
		},
		{
			name:     "repository update error",
			pluginID: pluginID,
			request: &UpdatePluginRequest{
				Title: stringPtr("Error Update"),
			},
			mockExistingPlugin: existingPlugin,
			mockExistingError:  nil,
			mockUpdateError:    errors.New("database error"),
			expectError:        true,
			expectedErrorType:  "repository",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockPluginRepo := new(MockPluginRepository)
			mockUserRepo := new(MockUserRepository)
			validator := validator.New()
			service := NewPluginService(mockPluginRepo, mockUserRepo, validator)

			mockPluginRepo.On("GetByID", tt.pluginID).Return(tt.mockExistingPlugin, tt.mockExistingError)

			if tt.expectNameCheck && tt.mockExistingPlugin != nil {
				mockPluginRepo.On("GetByName", *tt.request.Name).Return(tt.mockNameCheckPlugin, tt.mockNameCheckError)
			}

			if tt.mockExistingPlugin != nil && tt.expectedErrorType != "validation" {
				mockPluginRepo.On("Update", mock.AnythingOfType("*models.Plugin")).Return(tt.mockUpdateError)
			}

			result, err := service.UpdatePlugin(tt.pluginID, tt.request)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
				if tt.expectedErrorType == "validation" {
					_, isValidationError := err.(*ValidationError)
					assert.True(t, isValidationError)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.pluginID, result.ID)
			}

			mockPluginRepo.AssertExpectations(t)
		})
	}
}

func TestPluginService_DeletePlugin(t *testing.T) {
	pluginID := uuid.New()

	tests := []struct {
		name              string
		pluginID          uuid.UUID
		mockPlugin        *models.Plugin
		mockGetError      error
		mockDeleteError   error
		expectError       bool
		expectedErrorType string
	}{
		{
			name:     "successful deletion",
			pluginID: pluginID,
			mockPlugin: &models.Plugin{
				BaseModel: models.BaseModel{
					ID: pluginID,
				},
			},
			mockGetError:    nil,
			mockDeleteError: nil,
			expectError:     false,
		},
		{
			name:              "plugin not found",
			pluginID:          pluginID,
			mockPlugin:        nil,
			mockGetError:      errors.New("record not found"),
			expectError:       true,
			expectedErrorType: "not_found",
		},
		{
			name:     "repository delete error",
			pluginID: pluginID,
			mockPlugin: &models.Plugin{
				BaseModel: models.BaseModel{
					ID: pluginID,
				},
			},
			mockGetError:      nil,
			mockDeleteError:   errors.New("database error"),
			expectError:       true,
			expectedErrorType: "repository",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockPluginRepo := new(MockPluginRepository)
			mockUserRepo := new(MockUserRepository)
			validator := validator.New()
			service := NewPluginService(mockPluginRepo, mockUserRepo, validator)

			mockPluginRepo.On("GetByID", tt.pluginID).Return(tt.mockPlugin, tt.mockGetError)
			if tt.mockPlugin != nil {
				mockPluginRepo.On("Delete", tt.pluginID).Return(tt.mockDeleteError)
			}

			err := service.DeletePlugin(tt.pluginID)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			mockPluginRepo.AssertExpectations(t)
		})
	}
}

func TestPluginService_GetPluginUIContent(t *testing.T) {
	pluginID := uuid.New()
	userUUID := "test-user-uuid"
	provider := "github"

	tests := []struct {
		name               string
		pluginID           uuid.UUID
		userUUID           string
		provider           string
		mockPlugin         *models.Plugin
		mockPluginError    error
		mockGitHubResponse interface{}
		mockGitHubError    error
		expectError        bool
		expectedContent    string
	}{
		{
			name:     "successful UI content retrieval",
			pluginID: pluginID,
			userUUID: userUUID,
			provider: provider,
			mockPlugin: &models.Plugin{
				BaseModel: models.BaseModel{
					ID: pluginID,
				},
				ReactComponentPath: "https://github.com/owner/repo/blob/main/src/Component.tsx",
			},
			mockPluginError: nil,
			mockGitHubResponse: map[string]interface{}{
				"content": "import React from 'react';",
			},
			mockGitHubError: nil,
			expectError:     false,
			expectedContent: "import React from 'react';",
		},
		{
			name:            "plugin not found",
			pluginID:        pluginID,
			userUUID:        userUUID,
			provider:        provider,
			mockPlugin:      nil,
			mockPluginError: errors.New("record not found"),
			expectError:     true,
		},
		{
			name:     "empty react component path",
			pluginID: pluginID,
			userUUID: userUUID,
			provider: provider,
			mockPlugin: &models.Plugin{
				BaseModel: models.BaseModel{
					ID: pluginID,
				},
				ReactComponentPath: "",
			},
			mockPluginError: nil,
			expectError:     true,
		},
		{
			name:     "invalid GitHub URL",
			pluginID: pluginID,
			userUUID: userUUID,
			provider: provider,
			mockPlugin: &models.Plugin{
				BaseModel: models.BaseModel{
					ID: pluginID,
				},
				ReactComponentPath: "invalid-url",
			},
			mockPluginError: nil,
			expectError:     true,
		},
		{
			name:     "GitHub service error",
			pluginID: pluginID,
			userUUID: userUUID,
			provider: provider,
			mockPlugin: &models.Plugin{
				BaseModel: models.BaseModel{
					ID: pluginID,
				},
				ReactComponentPath: "https://github.com/owner/repo/blob/main/src/Component.tsx",
			},
			mockPluginError:    nil,
			mockGitHubResponse: nil,
			mockGitHubError:    errors.New("GitHub API error"),
			expectError:        true,
		},
		{
			name:     "empty content in GitHub response",
			pluginID: pluginID,
			userUUID: userUUID,
			provider: provider,
			mockPlugin: &models.Plugin{
				BaseModel: models.BaseModel{
					ID: pluginID,
				},
				ReactComponentPath: "https://github.com/owner/repo/blob/main/src/Component.tsx",
			},
			mockPluginError: nil,
			mockGitHubResponse: map[string]interface{}{
				"content": "",
			},
			mockGitHubError: nil,
			expectError:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockPluginRepo := new(MockPluginRepository)
			mockUserRepo := new(MockUserRepository)
			mockGitHubService := new(MockGitHubService)
			validator := validator.New()
			service := NewPluginService(mockPluginRepo, mockUserRepo, validator)

			mockPluginRepo.On("GetByID", tt.pluginID).Return(tt.mockPlugin, tt.mockPluginError)
			if tt.mockPlugin != nil && tt.mockPlugin.ReactComponentPath != "" && tt.mockPlugin.ReactComponentPath != "invalid-url" {
				mockGitHubService.On("GetRepositoryContent", mock.Anything, tt.userUUID, tt.provider, "owner", "repo", "src/Component.tsx", "main").Return(tt.mockGitHubResponse, tt.mockGitHubError)
			}

			result, err := service.GetPluginUIContent(context.Background(), tt.pluginID, mockGitHubService, tt.userUUID, tt.provider)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.expectedContent, result.Content)
				assert.Equal(t, "text/typescript", result.ContentType)
			}

			mockPluginRepo.AssertExpectations(t)
			mockGitHubService.AssertExpectations(t)
		})
	}
}

func TestParsePluginGitHubURL(t *testing.T) {
	tests := []struct {
		name          string
		githubURL     string
		expectedOwner string
		expectedRepo  string
		expectedPath  string
		expectedRef   string
		expectError   bool
	}{
		{
			name:          "valid GitHub.com URL",
			githubURL:     "https://github.com/owner/repo/blob/main/src/Component.tsx",
			expectedOwner: "owner",
			expectedRepo:  "repo",
			expectedPath:  "src/Component.tsx",
			expectedRef:   "main",
			expectError:   false,
		},
		{
			name:          "valid GitHub Enterprise URL",
			githubURL:     "https://github.tools.sap/owner/repo/blob/develop/plugins/MyPlugin.jsx",
			expectedOwner: "owner",
			expectedRepo:  "repo",
			expectedPath:  "plugins/MyPlugin.jsx",
			expectedRef:   "develop",
			expectError:   false,
		},
		{
			name:        "invalid URL format",
			githubURL:   "not-a-url",
			expectError: true,
		},
		{
			name:        "non-GitHub URL",
			githubURL:   "https://gitlab.com/owner/repo/blob/main/file.tsx",
			expectError: true,
		},
		{
			name:        "invalid GitHub blob URL format",
			githubURL:   "https://github.com/owner/repo/tree/main/src",
			expectError: true,
		},
		{
			name:        "missing path components",
			githubURL:   "https://github.com/owner",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			owner, repo, filePath, ref, err := parsePluginGitHubURL(tt.githubURL)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedOwner, owner)
				assert.Equal(t, tt.expectedRepo, repo)
				assert.Equal(t, tt.expectedPath, filePath)
				assert.Equal(t, tt.expectedRef, ref)
			}
		})
	}
}

// Helper function for creating string pointers
func stringPtr(s string) *string {
	return &s
}
