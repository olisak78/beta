package service_test

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"developer-portal-backend/internal/cache"
	"developer-portal-backend/internal/database/models"
	apperrors "developer-portal-backend/internal/errors"
	"developer-portal-backend/internal/mocks"
	"developer-portal-backend/internal/service"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
	"gorm.io/gorm"
)

// LandscapeServiceTestSuite defines the test suite for LandscapeService
type LandscapeServiceTestSuite struct {
	suite.Suite
	ctrl              *gomock.Controller
	mockLandscapeRepo *mocks.MockLandscapeRepositoryInterface
	mockOrgRepo       *mocks.MockOrganizationRepositoryInterface
	mockProjectRepo   *mocks.MockProjectRepositoryInterface
	mockCache         *mocks.MockCacheService
	landscapeService  *service.LandscapeService
	validator         *validator.Validate
	cacheService      cache.CacheService
	ttlConfig         cache.TTLConfig
}

// SetupTest sets up the test suite
func (suite *LandscapeServiceTestSuite) SetupTest() {
	suite.ctrl = gomock.NewController(suite.T())
	suite.mockLandscapeRepo = mocks.NewMockLandscapeRepositoryInterface(suite.ctrl)
	suite.mockOrgRepo = mocks.NewMockOrganizationRepositoryInterface(suite.ctrl)
	suite.mockProjectRepo = mocks.NewMockProjectRepositoryInterface(suite.ctrl)
	suite.mockCache = mocks.NewMockCacheService(suite.ctrl)
	suite.validator = validator.New()
	suite.ttlConfig = cache.DefaultTTLConfig()

	// Create a real in-memory cache for integration tests
	cacheConfig := cache.CacheConfig{
		DefaultTTL:      100 * time.Millisecond,
		CleanupInterval: 50 * time.Millisecond,
		Enabled:         true,
	}
	suite.cacheService = cache.NewInMemoryCache(cacheConfig)

	// Create landscape service with mocked dependencies
	suite.landscapeService = service.NewLandscapeServiceWithCache(
		suite.mockLandscapeRepo,
		suite.mockOrgRepo,
		suite.mockProjectRepo,
		suite.validator,
		suite.mockCache,
		suite.ttlConfig,
	)
}

// TearDownTest cleans up after each test
func (suite *LandscapeServiceTestSuite) TearDownTest() {
	suite.ctrl.Finish()
}

// TestCreateLandscapeRequest_Validation tests validation of create request
func (suite *LandscapeServiceTestSuite) TestCreateLandscapeRequest_Validation() {
	testCases := []struct {
		name        string
		request     service.CreateLandscapeRequest
		shouldError bool
	}{
		{
			name: "valid request",
			request: service.CreateLandscapeRequest{
				Name:        "test-landscape",
				Title:       "Test Landscape",
				Description: "A test landscape",
				ProjectID:   uuid.New(),
				Domain:      "test-domain",
				Environment: "dev",
			},
			shouldError: false,
		},
		{
			name: "missing name",
			request: service.CreateLandscapeRequest{
				Title:       "Test Landscape",
				ProjectID:   uuid.New(),
				Domain:      "test-domain",
				Environment: "dev",
			},
			shouldError: true,
		},
		{
			name: "missing title",
			request: service.CreateLandscapeRequest{
				Name:        "test-landscape",
				ProjectID:   uuid.New(),
				Domain:      "test-domain",
				Environment: "dev",
			},
			shouldError: true,
		},
		{
			name: "name too long",
			request: service.CreateLandscapeRequest{
				Name:        "this-is-a-very-long-landscape-name-that-exceeds-the-maximum-length-of-40-characters",
				Title:       "Test Landscape",
				ProjectID:   uuid.New(),
				Domain:      "test-domain",
				Environment: "dev",
			},
			shouldError: true,
		},
	}

	for _, tc := range testCases {
		suite.T().Run(tc.name, func(t *testing.T) {
			err := suite.validator.Struct(tc.request)
			if tc.shouldError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestUpdateLandscapeRequest_Validation tests validation of update request
func (suite *LandscapeServiceTestSuite) TestUpdateLandscapeRequest_Validation() {
	testCases := []struct {
		name        string
		request     service.UpdateLandscapeRequest
		shouldError bool
	}{
		{
			name: "valid request",
			request: service.UpdateLandscapeRequest{
				Title:       "Updated Landscape",
				Description: "Updated description",
				Domain:      "updated-domain",
				Environment: "prod",
			},
			shouldError: false,
		},
		{
			name: "missing title",
			request: service.UpdateLandscapeRequest{
				Description: "Updated description",
			},
			shouldError: true,
		},
		{
			name: "title too long",
			request: service.UpdateLandscapeRequest{
				Title: "this-is-a-very-long-title-that-exceeds-the-maximum-length-of-100-characters-and-should-fail-validation-rules",
			},
			shouldError: true,
		},
	}

	for _, tc := range testCases {
		suite.T().Run(tc.name, func(t *testing.T) {
			err := suite.validator.Struct(tc.request)
			if tc.shouldError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestPaginationLogic tests pagination calculations
func (suite *LandscapeServiceTestSuite) TestPaginationLogic() {
	testCases := []struct {
		name           string
		inputPage      int
		inputSize      int
		expectedPage   int
		expectedSize   int
		expectedOffset int
	}{
		{
			name:           "valid pagination",
			inputPage:      2,
			inputSize:      20,
			expectedPage:   2,
			expectedSize:   20,
			expectedOffset: 20,
		},
		{
			name:           "page less than 1",
			inputPage:      0,
			inputSize:      20,
			expectedPage:   1,
			expectedSize:   20,
			expectedOffset: 0,
		},
		{
			name:           "negative page",
			inputPage:      -5,
			inputSize:      20,
			expectedPage:   1,
			expectedSize:   20,
			expectedOffset: 0,
		},
		{
			name:           "size less than 1",
			inputPage:      1,
			inputSize:      0,
			expectedPage:   1,
			expectedSize:   20,
			expectedOffset: 0,
		},
		{
			name:           "size greater than 100",
			inputPage:      1,
			inputSize:      200,
			expectedPage:   1,
			expectedSize:   20,
			expectedOffset: 0,
		},
		{
			name:           "page 3 size 10",
			inputPage:      3,
			inputSize:      10,
			expectedPage:   3,
			expectedSize:   10,
			expectedOffset: 20,
		},
	}

	for _, tc := range testCases {
		suite.T().Run(tc.name, func(t *testing.T) {
			// Replicate pagination logic from the service
			page := tc.inputPage
			pageSize := tc.inputSize

			if page < 1 {
				page = 1
			}
			if pageSize < 1 || pageSize > 100 {
				pageSize = 20
			}

			offset := (page - 1) * pageSize

			assert.Equal(t, tc.expectedPage, page)
			assert.Equal(t, tc.expectedSize, pageSize)
			assert.Equal(t, tc.expectedOffset, offset)
		})
	}
}

// TestJSONFieldsHandling tests handling of JSON fields (Metadata)
func (suite *LandscapeServiceTestSuite) TestJSONFieldsHandling() {
	// Test valid JSON for metadata
	validMetadata := json.RawMessage(`{"region": "us-east-1", "tier": "production", "tags": ["prod", "critical"]}`)

	// Test that valid JSON can be marshaled and unmarshaled
	metadataData, err := json.Marshal(validMetadata)
	assert.NoError(suite.T(), err)
	assert.NotEmpty(suite.T(), metadataData)
	assert.Contains(suite.T(), string(metadataData), "region")
	assert.Contains(suite.T(), string(metadataData), "us-east-1")

	// Test empty JSON
	emptyJSON := json.RawMessage(`{}`)
	emptyData, err := json.Marshal(emptyJSON)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), `{}`, string(emptyData))

	// Test nil JSON (should be handled gracefully)
	var nilJSON json.RawMessage
	nilData, err := json.Marshal(nilJSON)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "null", string(nilData))
}

// TestLandscapeResponse_Structure tests that landscape response has correct structure
func (suite *LandscapeServiceTestSuite) TestLandscapeResponse_Structure() {
	response := service.LandscapeResponse{
		ID:          uuid.New(),
		Name:        "test-landscape",
		Title:       "Test Landscape",
		Description: "A test landscape",
		ProjectID:   uuid.New(),
		Domain:      "test-domain",
		Environment: "dev",
		Metadata:    json.RawMessage(`{"key": "value"}`),
		CreatedAt:   "2024-01-01T00:00:00Z",
		UpdatedAt:   "2024-01-02T00:00:00Z",
	}

	// Verify all fields are set correctly
	assert.NotEqual(suite.T(), uuid.Nil, response.ID)
	assert.Equal(suite.T(), "test-landscape", response.Name)
	assert.Equal(suite.T(), "Test Landscape", response.Title)
	assert.Equal(suite.T(), "A test landscape", response.Description)
	assert.NotEqual(suite.T(), uuid.Nil, response.ProjectID)
	assert.Equal(suite.T(), "test-domain", response.Domain)
	assert.Equal(suite.T(), "dev", response.Environment)
	assert.NotNil(suite.T(), response.Metadata)
}

// TestLandscapeMinimalResponse_Structure tests minimal response structure
func (suite *LandscapeServiceTestSuite) TestLandscapeMinimalResponse_Structure() {
	response := service.LandscapeMinimalResponse{
		ID:              uuid.New(),
		Name:            "test-landscape",
		Title:           "Test Landscape",
		Description:     "A test landscape",
		Domain:          "test-domain",
		Environment:     "dev",
		Grafana:         "https://grafana.example.com",
		Prometheus:      "https://prometheus.example.com",
		Health:          "https://health.example.com",
		Extension:       true,
		IsCentralRegion: false,
	}

	// Verify fields
	assert.NotEqual(suite.T(), uuid.Nil, response.ID)
	assert.Equal(suite.T(), "test-landscape", response.Name)
	assert.Equal(suite.T(), "https://grafana.example.com", response.Grafana)
	assert.True(suite.T(), response.Extension)
	assert.False(suite.T(), response.IsCentralRegion)
}

// TestLandscapeListResponse_Structure tests list response structure
func (suite *LandscapeServiceTestSuite) TestLandscapeListResponse_Structure() {
	landscapes := []service.LandscapeResponse{
		{
			ID:   uuid.New(),
			Name: "landscape-1",
		},
		{
			ID:   uuid.New(),
			Name: "landscape-2",
		},
	}

	response := service.LandscapeListResponse{
		Landscapes: landscapes,
		Total:      100,
		Page:       2,
		PageSize:   20,
	}

	assert.Len(suite.T(), response.Landscapes, 2)
	assert.Equal(suite.T(), int64(100), response.Total)
	assert.Equal(suite.T(), 2, response.Page)
	assert.Equal(suite.T(), 20, response.PageSize)
}

// TestCacheKeyGeneration tests cache key generation
func (suite *LandscapeServiceTestSuite) TestCacheKeyGeneration() {
	// Test various cache key patterns
	id := uuid.New()

	// By ID
	keyByID := cache.BuildKey(cache.KeyPrefixLandscapeByID, id.String())
	assert.Contains(suite.T(), keyByID, "landscape:id:")
	assert.Contains(suite.T(), keyByID, id.String())

	// By Name
	keyByName := cache.BuildKey(cache.KeyPrefixLandscapeByName, "test-landscape")
	assert.Equal(suite.T(), "landscape:name:test-landscape", keyByName)

	// By Project
	keyByProject := cache.BuildKey(cache.KeyPrefixLandscapeByProject, "my-project", "all")
	assert.Equal(suite.T(), "landscape:project:my-project:all", keyByProject)

	// Search
	keySearch := cache.BuildKey(cache.KeyPrefixLandscapeSearch, "q:test:page:1:size:20")
	assert.Equal(suite.T(), "landscape:search:q:test:page:1:size:20", keySearch)
}

// TestTTLConfiguration tests TTL configuration values
func (suite *LandscapeServiceTestSuite) TestTTLConfiguration() {
	config := cache.DefaultTTLConfig()

	// Landscape TTLs
	assert.Equal(suite.T(), 5*time.Minute, config.LandscapeList)
	assert.Equal(suite.T(), 5*time.Minute, config.LandscapeByID)
	assert.Equal(suite.T(), 5*time.Minute, config.LandscapeByName)
	assert.Equal(suite.T(), 5*time.Minute, config.LandscapeByProject)
	assert.Equal(suite.T(), 2*time.Minute, config.LandscapeSearch)

	// These should be reasonable values
	assert.True(suite.T(), config.LandscapeList > 0)
	assert.True(suite.T(), config.LandscapeSearch <= config.LandscapeList)
}

// TestErrorHandling tests error types
func (suite *LandscapeServiceTestSuite) TestErrorHandling() {
	// Test that app errors are properly typed
	assert.NotNil(suite.T(), apperrors.ErrLandscapeNotFound)
	assert.NotNil(suite.T(), apperrors.ErrLandscapeExists)
	assert.NotNil(suite.T(), apperrors.ErrProjectNotFound)

	// Test error messages
	assert.Contains(suite.T(), apperrors.ErrLandscapeNotFound.Error(), "not found")
}

// TestGormErrorHandling tests GORM error handling
func (suite *LandscapeServiceTestSuite) TestGormErrorHandling() {
	// Verify GORM error types work correctly
	assert.NotNil(suite.T(), gorm.ErrRecordNotFound)
}

// TestMetadataEnrichment tests metadata enrichment in minimal response
func (suite *LandscapeServiceTestSuite) TestMetadataEnrichment() {
	// Create a landscape model with metadata
	landscape := &models.Landscape{
		BaseModel: models.BaseModel{
			ID:          uuid.New(),
			Name:        "test-landscape",
			Title:       "Test Landscape",
			Description: "Test description",
			Metadata: json.RawMessage(`{
				"grafana": "https://grafana.example.com",
				"prometheus": "https://prometheus.example.com",
				"health": "https://health.example.com",
				"extension": true,
				"is-central-region": false,
				"type": "production"
			}`),
		},
		Domain:      "test-domain",
		Environment: "prod",
		ProjectID:   uuid.New(),
	}

	// Parse metadata
	var metadata map[string]interface{}
	err := json.Unmarshal(landscape.Metadata, &metadata)
	assert.NoError(suite.T(), err)

	// Verify metadata fields
	assert.Equal(suite.T(), "https://grafana.example.com", metadata["grafana"])
	assert.Equal(suite.T(), "https://prometheus.example.com", metadata["prometheus"])
	assert.Equal(suite.T(), true, metadata["extension"])
	assert.Equal(suite.T(), false, metadata["is-central-region"])
	assert.Equal(suite.T(), "production", metadata["type"])
}

// TestCreateLandscape_Success tests successful landscape creation
func (suite *LandscapeServiceTestSuite) TestCreateLandscape_Success() {
	projectID := uuid.New()
	metadata := json.RawMessage(`{"region": "us-east-1"}`)

	req := &service.CreateLandscapeRequest{
		Name:        "prod-landscape",
		Title:       "Production Landscape",
		Description: "Main production environment",
		ProjectID:   projectID,
		Domain:      "production.example.com",
		Environment: "production",
		Metadata:    metadata,
	}

	// Mock: Check if landscape with same name exists (should not exist)
	suite.mockLandscapeRepo.EXPECT().
		GetByName("prod-landscape").
		Return(nil, gorm.ErrRecordNotFound)

	// Mock: Create landscape
	suite.mockLandscapeRepo.EXPECT().
		Create(gomock.Any()).
		DoAndReturn(func(landscape *models.Landscape) error {
			landscape.ID = uuid.New()
			return nil
		})

	// Mock: Cache invalidation (called by invalidateLandscapeCaches)
	// The service calls Delete for ID and Name, then Clear
	suite.mockCache.EXPECT().
		Delete(gomock.Any()).
		Return(nil).
		Times(2) // Once for ID, once for Name

	suite.mockCache.EXPECT().
		Clear().
		Times(1)

	// Execute
	response, err := suite.landscapeService.CreateLandscape(req)

	// Assert
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), response)
	assert.Equal(suite.T(), "prod-landscape", response.Name)
	assert.Equal(suite.T(), "Production Landscape", response.Title)
	assert.Equal(suite.T(), projectID, response.ProjectID)
	assert.Equal(suite.T(), "production.example.com", response.Domain)
	assert.Equal(suite.T(), "production", response.Environment)
}

// TestCreateLandscape_DuplicateName tests duplicate name error
func (suite *LandscapeServiceTestSuite) TestCreateLandscape_DuplicateName() {
	projectID := uuid.New()
	existingLandscape := &models.Landscape{
		BaseModel: models.BaseModel{
			Name:  "prod-landscape",
			Title: "Existing",
		},
	}

	req := &service.CreateLandscapeRequest{
		Name:        "prod-landscape",
		Title:       "Production Landscape",
		ProjectID:   projectID,
		Domain:      "production.example.com",
		Environment: "production",
	}

	// Mock: Landscape with same name already exists
	suite.mockLandscapeRepo.EXPECT().
		GetByName("prod-landscape").
		Return(existingLandscape, nil)

	response, err := suite.landscapeService.CreateLandscape(req)

	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), response)
	assert.Equal(suite.T(), apperrors.ErrLandscapeExists, err)
}

// TestCreateLandscape_RepositoryError tests repository error handling
func (suite *LandscapeServiceTestSuite) TestCreateLandscape_RepositoryError() {
	projectID := uuid.New()

	req := &service.CreateLandscapeRequest{
		Name:        "prod-landscape",
		Title:       "Production Landscape",
		ProjectID:   projectID,
		Domain:      "production.example.com",
		Environment: "production",
	}

	suite.mockLandscapeRepo.EXPECT().
		GetByName("prod-landscape").
		Return(nil, gorm.ErrRecordNotFound)

	suite.mockLandscapeRepo.EXPECT().
		Create(gomock.Any()).
		Return(errors.New("database error"))

	response, err := suite.landscapeService.CreateLandscape(req)

	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), response)
	assert.Contains(suite.T(), err.Error(), "failed to create landscape")
}

// TestGetLandscapeByID_Success tests successful retrieval by ID
func (suite *LandscapeServiceTestSuite) TestGetLandscapeByID_Success() {
	landscapeID := uuid.New()
	projectID := uuid.New()

	landscape := &models.Landscape{
		BaseModel: models.BaseModel{
			ID:          landscapeID,
			Name:        "prod-landscape",
			Title:       "Production",
			Description: "Production environment",
		},
		ProjectID:   projectID,
		Domain:      "prod.example.com",
		Environment: "production",
	}

	// Mock: Cache miss (not in cache)
	suite.mockCache.EXPECT().
		Get(gomock.Any()).
		Return(nil, cache.ErrCacheMiss)

	// Mock: Repository call
	suite.mockLandscapeRepo.EXPECT().
		GetByID(landscapeID).
		Return(landscape, nil)

	// Mock: Cache set (store result in cache)
	suite.mockCache.EXPECT().
		Set(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil)

	response, err := suite.landscapeService.GetLandscapeByID(landscapeID)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), response)
	assert.Equal(suite.T(), landscapeID, response.ID)
	assert.Equal(suite.T(), "prod-landscape", response.Name)
}

// TestGetLandscapeByID_NotFound tests not found error
func (suite *LandscapeServiceTestSuite) TestGetLandscapeByID_NotFound() {
	landscapeID := uuid.New()

	// Mock: Cache miss
	suite.mockCache.EXPECT().
		Get(gomock.Any()).
		Return(nil, cache.ErrCacheMiss)

	// Mock: Repository returns not found
	suite.mockLandscapeRepo.EXPECT().
		GetByID(landscapeID).
		Return(nil, gorm.ErrRecordNotFound)

	response, err := suite.landscapeService.GetLandscapeByID(landscapeID)

	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), response)
	assert.Equal(suite.T(), apperrors.ErrLandscapeNotFound, err)
}

// TestGetLandscapeByID_RepositoryError tests repository error handling
func (suite *LandscapeServiceTestSuite) TestGetLandscapeByID_RepositoryError() {
	landscapeID := uuid.New()

	// Mock: Cache miss
	suite.mockCache.EXPECT().
		Get(gomock.Any()).
		Return(nil, cache.ErrCacheMiss)

	// Mock: Repository returns error
	suite.mockLandscapeRepo.EXPECT().
		GetByID(landscapeID).
		Return(nil, errors.New("database connection error"))

	response, err := suite.landscapeService.GetLandscapeByID(landscapeID)

	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), response)
	assert.Contains(suite.T(), err.Error(), "failed to get landscape")
}

// TestGetByName_Success tests successful retrieval by name
func (suite *LandscapeServiceTestSuite) TestGetByName_Success() {
	landscapeID := uuid.New()
	projectID := uuid.New()
	orgID := uuid.New()

	landscape := &models.Landscape{
		BaseModel: models.BaseModel{
			ID:    landscapeID,
			Name:  "prod-landscape",
			Title: "Production",
		},
		ProjectID:   projectID,
		Domain:      "prod.example.com",
		Environment: "production",
	}

	// Mock: Cache miss
	suite.mockCache.EXPECT().
		Get(gomock.Any()).
		Return(nil, cache.ErrCacheMiss)

	// Mock: Repository call
	suite.mockLandscapeRepo.EXPECT().
		GetByName("prod-landscape").
		Return(landscape, nil)

	// Mock: Cache set
	suite.mockCache.EXPECT().
		Set(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil)

	response, err := suite.landscapeService.GetByName(orgID, "prod-landscape")

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), response)
	assert.Equal(suite.T(), "prod-landscape", response.Name)
}

// TestGetByName_NotFound tests not found error
func (suite *LandscapeServiceTestSuite) TestGetByName_NotFound() {
	orgID := uuid.New()

	// Mock: Cache miss
	suite.mockCache.EXPECT().
		Get(gomock.Any()).
		Return(nil, cache.ErrCacheMiss)

	// Mock: Repository returns not found
	suite.mockLandscapeRepo.EXPECT().
		GetByName("nonexistent-landscape").
		Return(nil, gorm.ErrRecordNotFound)

	response, err := suite.landscapeService.GetByName(orgID, "nonexistent-landscape")

	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), response)
	assert.Equal(suite.T(), apperrors.ErrLandscapeNotFound, err)
}

// TestGetByName_RepositoryError tests repository error handling
func (suite *LandscapeServiceTestSuite) TestGetByName_RepositoryError() {
	orgID := uuid.New()

	// Mock: Cache miss
	suite.mockCache.EXPECT().
		Get(gomock.Any()).
		Return(nil, cache.ErrCacheMiss)

	// Mock: Repository returns error
	suite.mockLandscapeRepo.EXPECT().
		GetByName("prod-landscape").
		Return(nil, errors.New("database connection error"))

	response, err := suite.landscapeService.GetByName(orgID, "prod-landscape")

	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), response)
	assert.Contains(suite.T(), err.Error(), "failed to get landscape")
}

// TestGetLandscapesByOrganization_Success tests paginated retrieval
func (suite *LandscapeServiceTestSuite) TestGetLandscapesByOrganization_Success() {
	orgID := uuid.New()
	landscapes := []models.Landscape{
		{
			BaseModel: models.BaseModel{
				ID:    uuid.New(),
				Name:  "landscape1",
				Title: "Landscape 1",
			},
			ProjectID:   uuid.New(),
			Domain:      "domain1.com",
			Environment: "production",
		},
		{
			BaseModel: models.BaseModel{
				ID:    uuid.New(),
				Name:  "landscape2",
				Title: "Landscape 2",
			},
			ProjectID:   uuid.New(),
			Domain:      "domain2.com",
			Environment: "development",
		},
	}

	// Mock: Cache miss
	suite.mockCache.EXPECT().
		Get(gomock.Any()).
		Return(nil, cache.ErrCacheMiss)

	// Mock: Repository call
	suite.mockLandscapeRepo.EXPECT().
		GetActiveLandscapes(20, 0).
		Return(landscapes, int64(2), nil)

	// Mock: Cache set
	suite.mockCache.EXPECT().
		Set(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil)

	responses, total, err := suite.landscapeService.GetLandscapesByOrganization(orgID, 20, 0)

	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), responses, 2)
	assert.Equal(suite.T(), int64(2), total)
	assert.Equal(suite.T(), "landscape1", responses[0].Name)
	assert.Equal(suite.T(), "landscape2", responses[1].Name)
}

// TestGetLandscapesByOrganization_RepositoryError tests repository error handling
func (suite *LandscapeServiceTestSuite) TestGetLandscapesByOrganization_RepositoryError() {
	orgID := uuid.New()

	// Mock: Cache miss
	suite.mockCache.EXPECT().
		Get(gomock.Any()).
		Return(nil, cache.ErrCacheMiss)

	// Mock: Repository returns error
	suite.mockLandscapeRepo.EXPECT().
		GetActiveLandscapes(20, 0).
		Return(nil, int64(0), errors.New("database connection error"))

	responses, total, err := suite.landscapeService.GetLandscapesByOrganization(orgID, 20, 0)

	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), responses)
	assert.Equal(suite.T(), int64(0), total)
	assert.Contains(suite.T(), err.Error(), "failed to get landscapes")
}

// TestGetLandscapesByOrganization_EmptyResult tests empty result handling
func (suite *LandscapeServiceTestSuite) TestGetLandscapesByOrganization_EmptyResult() {
	orgID := uuid.New()

	// Mock: Cache miss
	suite.mockCache.EXPECT().
		Get(gomock.Any()).
		Return(nil, cache.ErrCacheMiss)

	// Mock: Repository returns empty result
	suite.mockLandscapeRepo.EXPECT().
		GetActiveLandscapes(20, 0).
		Return([]models.Landscape{}, int64(0), nil)

	// Mock: Cache set
	suite.mockCache.EXPECT().
		Set(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil)

	responses, total, err := suite.landscapeService.GetLandscapesByOrganization(orgID, 20, 0)

	assert.NoError(suite.T(), err)
	assert.Empty(suite.T(), responses)
	assert.Equal(suite.T(), int64(0), total)
}

// TestGetLandscapesByOrganization_LimitValidation tests the limit validation logic (limit < 1 || limit > 100)
func (suite *LandscapeServiceTestSuite) TestGetLandscapesByOrganization_LimitValidation() {
	orgID := uuid.New()

	testCases := []struct {
		name               string
		inputLimit         int
		expectedLimit      int
		expectedInRepoCall int
	}{
		{
			name:               "Valid limit within range (50)",
			inputLimit:         50,
			expectedLimit:      50,
			expectedInRepoCall: 50,
		},
		{
			name:               "Limit at lower boundary (1)",
			inputLimit:         1,
			expectedLimit:      1,
			expectedInRepoCall: 1,
		},
		{
			name:               "Limit at upper boundary (100)",
			inputLimit:         100,
			expectedLimit:      100,
			expectedInRepoCall: 100,
		},
		{
			name:               "Limit less than 1 (0) - should default to 20",
			inputLimit:         0,
			expectedLimit:      20,
			expectedInRepoCall: 20,
		},
		{
			name:               "Limit less than 1 (-10) - should default to 20",
			inputLimit:         -10,
			expectedLimit:      20,
			expectedInRepoCall: 20,
		},
		{
			name:               "Limit greater than 100 (101) - should default to 20",
			inputLimit:         101,
			expectedLimit:      20,
			expectedInRepoCall: 20,
		},
		{
			name:               "Limit greater than 100 (1000) - should default to 20",
			inputLimit:         1000,
			expectedLimit:      20,
			expectedInRepoCall: 20,
		},
	}

	for _, tc := range testCases {
		suite.T().Run(tc.name, func(t *testing.T) {
			// Mock: Cache miss
			suite.mockCache.EXPECT().
				Get(gomock.Any()).
				Return(nil, cache.ErrCacheMiss)

			// Mock expects the corrected limit value to be passed to repository
			suite.mockLandscapeRepo.EXPECT().
				GetActiveLandscapes(tc.expectedInRepoCall, 0).
				Return([]models.Landscape{}, int64(0), nil)

			// Mock: Cache set
			suite.mockCache.EXPECT().
				Set(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(nil)

			responses, total, err := suite.landscapeService.GetLandscapesByOrganization(orgID, tc.inputLimit, 0)

			assert.NoError(t, err)
			assert.Empty(t, responses)
			assert.Equal(t, int64(0), total)
		})
	}
}

// TestGetByProjectName_Success tests retrieval by project name
func (suite *LandscapeServiceTestSuite) TestGetByProjectName_Success() {
	projectID := uuid.New()
	project := &models.Project{
		BaseModel: models.BaseModel{
			ID:   projectID,
			Name: "my-project",
		},
	}

	landscapes := []models.Landscape{
		{
			BaseModel: models.BaseModel{
				ID:    uuid.New(),
				Name:  "landscape1",
				Title: "Landscape 1",
			},
			ProjectID:   projectID,
			Domain:      "domain1.com",
			Environment: "production",
		},
	}

	// Mock: Cache miss
	suite.mockCache.EXPECT().
		Get(gomock.Any()).
		Return(nil, cache.ErrCacheMiss)

	// Mock: Get project by name
	suite.mockProjectRepo.EXPECT().
		GetByName("my-project").
		Return(project, nil)

	// Mock: GetByProjectName calls GetLandscapesByProjectID with limit=100, offset=0
	suite.mockLandscapeRepo.EXPECT().
		GetLandscapesByProjectID(projectID, 100, 0).
		Return(landscapes, int64(1), nil)

	// Mock: Cache set
	suite.mockCache.EXPECT().
		Set(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil)

	response, err := suite.landscapeService.GetByProjectName("my-project")

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), response)
	assert.Len(suite.T(), response.Landscapes, 1)
}

// TestGetByProjectName_EmptyName tests empty project name
func (suite *LandscapeServiceTestSuite) TestGetByProjectName_EmptyName() {
	// Mock: Cache miss
	suite.mockCache.EXPECT().
		Get(gomock.Any()).
		Return(nil, cache.ErrCacheMiss)

	// Mock: Get project by empty name
	suite.mockProjectRepo.EXPECT().
		GetByName("").
		Return(nil, gorm.ErrRecordNotFound)

	response, err := suite.landscapeService.GetByProjectName("")

	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), response)
	assert.Equal(suite.T(), apperrors.ErrProjectNotFound, err)
}

// TestGetByProjectName_ProjectNotFound tests project not found
func (suite *LandscapeServiceTestSuite) TestGetByProjectName_ProjectNotFound() {
	// Mock: Cache miss
	suite.mockCache.EXPECT().
		Get(gomock.Any()).
		Return(nil, cache.ErrCacheMiss)

	// Mock: Project not found
	suite.mockProjectRepo.EXPECT().
		GetByName("nonexistent").
		Return(nil, gorm.ErrRecordNotFound)

	response, err := suite.landscapeService.GetByProjectName("nonexistent")

	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), response)
	assert.Equal(suite.T(), apperrors.ErrProjectNotFound, err)
}

// TestGetByProjectName_ProjectRepositoryError tests project repository error
func (suite *LandscapeServiceTestSuite) TestGetByProjectName_ProjectRepositoryError() {
	// Mock: Cache miss
	suite.mockCache.EXPECT().
		Get(gomock.Any()).
		Return(nil, cache.ErrCacheMiss)

	// Mock: Project repository error
	suite.mockProjectRepo.EXPECT().
		GetByName("my-project").
		Return(nil, errors.New("database connection error"))

	response, err := suite.landscapeService.GetByProjectName("my-project")

	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), response)
	assert.Contains(suite.T(), err.Error(), "failed to find project")
}

// TestGetByProjectName_ProjectNil tests when project is nil (service now checks for nil)
func (suite *LandscapeServiceTestSuite) TestGetByProjectName_ProjectNil() {
	// Mock: Cache miss
	suite.mockCache.EXPECT().
		Get(gomock.Any()).
		Return(nil, cache.ErrCacheMiss)

	// Mock: Project returns nil without error
	suite.mockProjectRepo.EXPECT().
		GetByName("my-project").
		Return(nil, nil)

	response, err := suite.landscapeService.GetByProjectName("my-project")

	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), response)
	assert.Equal(suite.T(), apperrors.ErrProjectNotFound, err)
}

// TestGetByProjectName_LandscapeRepositoryError tests landscape repository error after finding project
func (suite *LandscapeServiceTestSuite) TestGetByProjectName_LandscapeRepositoryError() {
	projectID := uuid.New()
	project := &models.Project{
		BaseModel: models.BaseModel{
			ID:   projectID,
			Name: "my-project",
		},
	}

	// Mock: Cache miss
	suite.mockCache.EXPECT().
		Get(gomock.Any()).
		Return(nil, cache.ErrCacheMiss)

	// Mock: Get project by name
	suite.mockProjectRepo.EXPECT().
		GetByName("my-project").
		Return(project, nil)

	// Mock: Landscape repository error
	suite.mockLandscapeRepo.EXPECT().
		GetLandscapesByProjectID(projectID, 100, 0).
		Return(nil, int64(0), errors.New("database connection error"))

	response, err := suite.landscapeService.GetByProjectName("my-project")

	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), response)
	assert.Contains(suite.T(), err.Error(), "failed to get landscapes by project")
}

// TestUpdateLandscape_Success tests successful landscape update
func (suite *LandscapeServiceTestSuite) TestUpdateLandscape_Success() {
	landscapeID := uuid.New()
	projectID := uuid.New()
	newProjectID := uuid.New()

	existingLandscape := &models.Landscape{
		BaseModel: models.BaseModel{
			ID:          landscapeID,
			Name:        "prod-landscape",
			Title:       "Old Title",
			Description: "Old description",
		},
		ProjectID:   projectID,
		Domain:      "old.com",
		Environment: "production",
	}

	req := &service.UpdateLandscapeRequest{
		Title:       "New Title",
		Description: "New description",
		ProjectID:   &newProjectID,
		Domain:      "new.com",
		Environment: "staging",
	}

	suite.mockLandscapeRepo.EXPECT().
		GetByID(landscapeID).
		Return(existingLandscape, nil)

	suite.mockLandscapeRepo.EXPECT().
		Update(gomock.Any()).
		DoAndReturn(func(landscape *models.Landscape) error {
			assert.Equal(suite.T(), "New Title", landscape.Title)
			assert.Equal(suite.T(), "New description", landscape.Description)
			assert.Equal(suite.T(), newProjectID, landscape.ProjectID)
			assert.Equal(suite.T(), "new.com", landscape.Domain)
			assert.Equal(suite.T(), "staging", landscape.Environment)
			return nil
		})

	// Mock: Cache invalidation (called by invalidateLandscapeCaches)
	suite.mockCache.EXPECT().
		Delete(gomock.Any()).
		Return(nil).
		Times(2) // Once for ID, once for Name

	suite.mockCache.EXPECT().
		Clear().
		Times(1)

	response, err := suite.landscapeService.UpdateLandscape(landscapeID, req)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), response)
	assert.Equal(suite.T(), "New Title", response.Title)
}

// TestUpdateLandscape_NotFound tests update of non-existent landscape
func (suite *LandscapeServiceTestSuite) TestUpdateLandscape_NotFound() {
	landscapeID := uuid.New()

	req := &service.UpdateLandscapeRequest{
		Title: "New Title",
	}

	suite.mockLandscapeRepo.EXPECT().
		GetByID(landscapeID).
		Return(nil, gorm.ErrRecordNotFound)

	response, err := suite.landscapeService.UpdateLandscape(landscapeID, req)

	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), response)
	assert.Equal(suite.T(), apperrors.ErrLandscapeNotFound, err)
}

// TestUpdateLandscape_ValidationError tests validation error on update
func (suite *LandscapeServiceTestSuite) TestUpdateLandscape_ValidationError() {
	landscapeID := uuid.New()

	req := &service.UpdateLandscapeRequest{
		Title: "", // Empty title should fail validation
	}

	response, err := suite.landscapeService.UpdateLandscape(landscapeID, req)

	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), response)
	assert.Contains(suite.T(), err.Error(), "validation failed")
}

// TestUpdateLandscape_RepositoryError tests repository error during update
func (suite *LandscapeServiceTestSuite) TestUpdateLandscape_RepositoryError() {
	landscapeID := uuid.New()
	projectID := uuid.New()

	existingLandscape := &models.Landscape{
		BaseModel: models.BaseModel{
			ID:          landscapeID,
			Name:        "prod-landscape",
			Title:       "Old Title",
			Description: "Old description",
		},
		ProjectID:   projectID,
		Domain:      "old.com",
		Environment: "production",
	}

	req := &service.UpdateLandscapeRequest{
		Title:       "New Title",
		Description: "New description",
	}

	suite.mockLandscapeRepo.EXPECT().
		GetByID(landscapeID).
		Return(existingLandscape, nil)

	suite.mockLandscapeRepo.EXPECT().
		Update(gomock.Any()).
		Return(errors.New("database connection error"))

	response, err := suite.landscapeService.UpdateLandscape(landscapeID, req)

	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), response)
	assert.Contains(suite.T(), err.Error(), "failed to update landscape")
}

// TestUpdateLandscape_OnlyRequiredFields tests updating only required fields (Title, Description)
func (suite *LandscapeServiceTestSuite) TestUpdateLandscape_OnlyRequiredFields() {
	landscapeID := uuid.New()
	projectID := uuid.New()

	existingLandscape := &models.Landscape{
		BaseModel: models.BaseModel{
			ID:          landscapeID,
			Name:        "prod-landscape",
			Title:       "Old Title",
			Description: "Old description",
		},
		ProjectID:   projectID,
		Domain:      "old.com",
		Environment: "production",
	}

	req := &service.UpdateLandscapeRequest{
		Title:       "New Title",
		Description: "New description",
		// No optional fields provided
	}

	suite.mockLandscapeRepo.EXPECT().
		GetByID(landscapeID).
		Return(existingLandscape, nil)

	suite.mockLandscapeRepo.EXPECT().
		Update(gomock.Any()).
		DoAndReturn(func(landscape *models.Landscape) error {
			// Verify only Title and Description changed
			assert.Equal(suite.T(), "New Title", landscape.Title)
			assert.Equal(suite.T(), "New description", landscape.Description)
			// Verify other fields remain unchanged
			assert.Equal(suite.T(), projectID, landscape.ProjectID)
			assert.Equal(suite.T(), "old.com", landscape.Domain)
			assert.Equal(suite.T(), "production", landscape.Environment)
			return nil
		})

	// Mock: Cache invalidation
	suite.mockCache.EXPECT().
		Delete(gomock.Any()).
		Return(nil).
		Times(2)

	suite.mockCache.EXPECT().
		Clear().
		Times(1)

	response, err := suite.landscapeService.UpdateLandscape(landscapeID, req)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), response)
	assert.Equal(suite.T(), "New Title", response.Title)
	assert.Equal(suite.T(), "New description", response.Description)
}

// TestUpdateLandscape_AllFieldsUpdated tests updating all fields at once (entering all conditions)
func (suite *LandscapeServiceTestSuite) TestUpdateLandscape_AllFieldsUpdated() {
	landscapeID := uuid.New()
	oldProjectID := uuid.New()
	newProjectID := uuid.New()
	oldMetadata := json.RawMessage(`{"old": "data"}`)
	newMetadata := json.RawMessage(`{"new": "data", "region": "eu-central-1", "tier": "premium"}`)

	existingLandscape := &models.Landscape{
		BaseModel: models.BaseModel{
			ID:          landscapeID,
			Name:        "prod-landscape",
			Title:       "Old Title",
			Description: "Old description",
			Metadata:    oldMetadata,
		},
		ProjectID:   oldProjectID,
		Domain:      "old-domain.com",
		Environment: "production",
	}

	req := &service.UpdateLandscapeRequest{
		Title:       "Completely New Title",
		Description: "Completely new description",
		ProjectID:   &newProjectID,
		Domain:      "completely-new-domain.com",
		Environment: "development",
		Metadata:    newMetadata,
	}

	suite.mockLandscapeRepo.EXPECT().
		GetByID(landscapeID).
		Return(existingLandscape, nil)

	suite.mockLandscapeRepo.EXPECT().
		Update(gomock.Any()).
		DoAndReturn(func(landscape *models.Landscape) error {
			// Verify ALL fields were updated (all conditions entered)
			assert.Equal(suite.T(), "Completely New Title", landscape.Title)
			assert.Equal(suite.T(), "Completely new description", landscape.Description)
			assert.Equal(suite.T(), newProjectID, landscape.ProjectID)
			assert.Equal(suite.T(), "completely-new-domain.com", landscape.Domain)
			assert.Equal(suite.T(), "development", landscape.Environment)
			assert.Equal(suite.T(), newMetadata, landscape.Metadata)
			return nil
		})

	// Mock: Cache invalidation
	suite.mockCache.EXPECT().
		Delete(gomock.Any()).
		Return(nil).
		Times(2)

	suite.mockCache.EXPECT().
		Clear().
		Times(1)

	response, err := suite.landscapeService.UpdateLandscape(landscapeID, req)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), response)
	assert.Equal(suite.T(), "Completely New Title", response.Title)
	assert.Equal(suite.T(), "Completely new description", response.Description)
	assert.Equal(suite.T(), newProjectID, response.ProjectID)
	assert.Equal(suite.T(), "completely-new-domain.com", response.Domain)
	assert.Equal(suite.T(), "development", response.Environment)
	assert.Equal(suite.T(), newMetadata, response.Metadata)
}

// TestUpdateLandscape_EmptyOptionalFields tests that empty optional fields don't update
func (suite *LandscapeServiceTestSuite) TestUpdateLandscape_EmptyOptionalFields() {
	landscapeID := uuid.New()
	projectID := uuid.New()
	metadata := json.RawMessage(`{"existing": "data"}`)

	existingLandscape := &models.Landscape{
		BaseModel: models.BaseModel{
			ID:          landscapeID,
			Name:        "prod-landscape",
			Title:       "Old Title",
			Description: "Old description",
			Metadata:    metadata,
		},
		ProjectID:   projectID,
		Domain:      "existing-domain.com",
		Environment: "production",
	}

	req := &service.UpdateLandscapeRequest{
		Title:       "New Title",
		Description: "New description",
		ProjectID:   nil, // nil should not update
		Domain:      "",  // empty should not update
		Environment: "",  // empty should not update
		Metadata:    nil, // nil should not update
	}

	suite.mockLandscapeRepo.EXPECT().
		GetByID(landscapeID).
		Return(existingLandscape, nil)

	suite.mockLandscapeRepo.EXPECT().
		Update(gomock.Any()).
		DoAndReturn(func(landscape *models.Landscape) error {
			// Verify only Title and Description changed, others remain unchanged
			assert.Equal(suite.T(), "New Title", landscape.Title)
			assert.Equal(suite.T(), "New description", landscape.Description)
			assert.Equal(suite.T(), projectID, landscape.ProjectID)
			assert.Equal(suite.T(), "existing-domain.com", landscape.Domain)
			assert.Equal(suite.T(), "production", landscape.Environment)
			assert.Equal(suite.T(), metadata, landscape.Metadata)
			return nil
		})

	// Mock: Cache invalidation
	suite.mockCache.EXPECT().
		Delete(gomock.Any()).
		Return(nil).
		Times(2)

	suite.mockCache.EXPECT().
		Clear().
		Times(1)

	response, err := suite.landscapeService.UpdateLandscape(landscapeID, req)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), response)
	// Verify response has unchanged values
	assert.Equal(suite.T(), projectID, response.ProjectID)
	assert.Equal(suite.T(), "existing-domain.com", response.Domain)
	assert.Equal(suite.T(), "production", response.Environment)
}

// TestDeleteLandscape_Success tests successful landscape deletion
func (suite *LandscapeServiceTestSuite) TestDeleteLandscape_Success() {
	landscapeID := uuid.New()

	landscape := &models.Landscape{
		BaseModel: models.BaseModel{
			ID:    landscapeID,
			Name:  "prod-landscape",
			Title: "Production",
		},
	}

	suite.mockLandscapeRepo.EXPECT().
		GetByID(landscapeID).
		Return(landscape, nil)

	suite.mockLandscapeRepo.EXPECT().
		Delete(landscapeID).
		Return(nil)

	// Mock: Cache invalidation (called by invalidateLandscapeCaches)
	suite.mockCache.EXPECT().
		Delete(gomock.Any()).
		Return(nil).
		Times(2) // Once for ID, once for Name

	suite.mockCache.EXPECT().
		Clear().
		Times(1)

	err := suite.landscapeService.DeleteLandscape(landscapeID)

	assert.NoError(suite.T(), err)
}

// TestDeleteLandscape_NotFound tests deletion of non-existent landscape
func (suite *LandscapeServiceTestSuite) TestDeleteLandscape_NotFound() {
	landscapeID := uuid.New()

	suite.mockLandscapeRepo.EXPECT().
		GetByID(landscapeID).
		Return(nil, gorm.ErrRecordNotFound)

	err := suite.landscapeService.DeleteLandscape(landscapeID)

	assert.Error(suite.T(), err)
	assert.Equal(suite.T(), apperrors.ErrLandscapeNotFound, err)
}

// TestDeleteLandscape_GetByIDError tests error during GetByID check
func (suite *LandscapeServiceTestSuite) TestDeleteLandscape_GetByIDError() {
	landscapeID := uuid.New()

	suite.mockLandscapeRepo.EXPECT().
		GetByID(landscapeID).
		Return(nil, errors.New("database connection error"))

	err := suite.landscapeService.DeleteLandscape(landscapeID)

	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "failed to get landscape")
}

// TestDeleteLandscape_DeleteError tests error during actual deletion
func (suite *LandscapeServiceTestSuite) TestDeleteLandscape_DeleteError() {
	landscapeID := uuid.New()

	landscape := &models.Landscape{
		BaseModel: models.BaseModel{
			ID:    landscapeID,
			Name:  "prod-landscape",
			Title: "Production",
		},
	}

	suite.mockLandscapeRepo.EXPECT().
		GetByID(landscapeID).
		Return(landscape, nil)

	suite.mockLandscapeRepo.EXPECT().
		Delete(landscapeID).
		Return(errors.New("database constraint violation"))

	err := suite.landscapeService.DeleteLandscape(landscapeID)

	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "failed to delete landscape")
}

// TestSetStatus_Success tests setting landscape status
func (suite *LandscapeServiceTestSuite) TestSetStatus_Success() {
	landscapeID := uuid.New()

	landscape := &models.Landscape{
		BaseModel: models.BaseModel{
			ID:    landscapeID,
			Name:  "prod-landscape",
			Title: "Production",
		},
	}

	suite.mockLandscapeRepo.EXPECT().
		GetByID(landscapeID).
		Return(landscape, nil)

	suite.mockLandscapeRepo.EXPECT().
		SetStatus(landscapeID, "active").
		Return(nil)

	// Mock: Cache invalidation (called by invalidateLandscapeCaches)
	suite.mockCache.EXPECT().
		Delete(gomock.Any()).
		Return(nil).
		Times(2) // Once for ID, once for Name

	suite.mockCache.EXPECT().
		Clear().
		Times(1)

	err := suite.landscapeService.SetStatus(landscapeID, "active")

	assert.NoError(suite.T(), err)
}

// TestSetStatus_NotFound tests setting status on non-existent landscape
func (suite *LandscapeServiceTestSuite) TestSetStatus_NotFound() {
	landscapeID := uuid.New()

	suite.mockLandscapeRepo.EXPECT().
		GetByID(landscapeID).
		Return(nil, gorm.ErrRecordNotFound)

	err := suite.landscapeService.SetStatus(landscapeID, "active")

	assert.Error(suite.T(), err)
	assert.Equal(suite.T(), apperrors.ErrLandscapeNotFound, err)
}

// TestSetStatus_GetByIDError tests error during GetByID check
func (suite *LandscapeServiceTestSuite) TestSetStatus_GetByIDError() {
	landscapeID := uuid.New()

	suite.mockLandscapeRepo.EXPECT().
		GetByID(landscapeID).
		Return(nil, errors.New("database connection error"))

	err := suite.landscapeService.SetStatus(landscapeID, "active")

	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "failed to get landscape")
}

// TestSetStatus_SetStatusError tests error during actual status update
func (suite *LandscapeServiceTestSuite) TestSetStatus_SetStatusError() {
	landscapeID := uuid.New()

	landscape := &models.Landscape{
		BaseModel: models.BaseModel{
			ID:    landscapeID,
			Name:  "prod-landscape",
			Title: "Production",
		},
	}

	suite.mockLandscapeRepo.EXPECT().
		GetByID(landscapeID).
		Return(landscape, nil)

	suite.mockLandscapeRepo.EXPECT().
		SetStatus(landscapeID, "active").
		Return(errors.New("database update error"))

	err := suite.landscapeService.SetStatus(landscapeID, "active")

	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "failed to set landscape status")
}

// TestGetWithOrganization_Success tests retrieval with organization details
func (suite *LandscapeServiceTestSuite) TestGetWithOrganization_Success() {
	landscapeID := uuid.New()

	landscape := &models.Landscape{
		BaseModel: models.BaseModel{
			ID:    landscapeID,
			Name:  "prod-landscape",
			Title: "Production",
		},
	}

	suite.mockLandscapeRepo.EXPECT().
		GetByID(landscapeID).
		Return(landscape, nil)

	result, err := suite.landscapeService.GetWithOrganization(landscapeID)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.Equal(suite.T(), landscapeID, result.ID)
}

// TestGetWithOrganization_NotFound tests not found error
func (suite *LandscapeServiceTestSuite) TestGetWithOrganization_NotFound() {
	landscapeID := uuid.New()

	suite.mockLandscapeRepo.EXPECT().
		GetByID(landscapeID).
		Return(nil, gorm.ErrRecordNotFound)

	result, err := suite.landscapeService.GetWithOrganization(landscapeID)

	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), result)
	assert.Equal(suite.T(), apperrors.ErrLandscapeNotFound, err)
}

// TestGetWithOrganization_RepositoryError tests repository error
func (suite *LandscapeServiceTestSuite) TestGetWithOrganization_RepositoryError() {
	landscapeID := uuid.New()

	suite.mockLandscapeRepo.EXPECT().
		GetByID(landscapeID).
		Return(nil, errors.New("database connection error"))

	result, err := suite.landscapeService.GetWithOrganization(landscapeID)

	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), result)
	assert.Contains(suite.T(), err.Error(), "failed to get landscape")
}

// TestGetByProjectNameAll_Success tests successful retrieval of all landscapes by project name
func (suite *LandscapeServiceTestSuite) TestGetByProjectNameAll_Success() {
	projectID := uuid.New()
	project := &models.Project{
		BaseModel: models.BaseModel{
			ID:   projectID,
			Name: "my-project",
		},
	}

	landscapes := []models.Landscape{
		{
			BaseModel: models.BaseModel{
				ID:    uuid.New(),
				Name:  "landscape1",
				Title: "Landscape 1",
				Metadata: json.RawMessage(`{
					"grafana": "https://grafana1.example.com",
					"prometheus": "https://prometheus1.example.com"
				}`),
			},
			ProjectID:   projectID,
			Domain:      "domain1.com",
			Environment: "production",
		},
		{
			BaseModel: models.BaseModel{
				ID:    uuid.New(),
				Name:  "landscape2",
				Title: "Landscape 2",
				Metadata: json.RawMessage(`{
					"grafana": "https://grafana2.example.com",
					"health": "https://health2.example.com"
				}`),
			},
			ProjectID:   projectID,
			Domain:      "domain2.com",
			Environment: "development",
		},
	}

	// Mock: Cache miss
	suite.mockCache.EXPECT().
		Get(gomock.Any()).
		Return(nil, cache.ErrCacheMiss)

	// Mock: Get project by name
	suite.mockProjectRepo.EXPECT().
		GetByName("my-project").
		Return(project, nil)

	// Mock: GetByProjectNameAll calls GetLandscapesByProjectID with limit=1000, offset=0
	suite.mockLandscapeRepo.EXPECT().
		GetLandscapesByProjectID(projectID, 1000, 0).
		Return(landscapes, int64(2), nil)

	// Mock: Cache set
	suite.mockCache.EXPECT().
		Set(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil)

	responses, err := suite.landscapeService.GetByProjectNameAll("my-project")

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), responses)
	assert.Len(suite.T(), responses, 2)
	assert.Equal(suite.T(), "landscape1", responses[0].Name)
	assert.Equal(suite.T(), "Landscape 1", responses[0].Title)
	assert.Equal(suite.T(), "landscape2", responses[1].Name)
	assert.Equal(suite.T(), "Landscape 2", responses[1].Title)
}

// TestGetByProjectNameAll_EmptyResult tests when project has no landscapes
func (suite *LandscapeServiceTestSuite) TestGetByProjectNameAll_EmptyResult() {
	projectID := uuid.New()
	project := &models.Project{
		BaseModel: models.BaseModel{
			ID:   projectID,
			Name: "empty-project",
		},
	}

	// Mock: Cache miss
	suite.mockCache.EXPECT().
		Get(gomock.Any()).
		Return(nil, cache.ErrCacheMiss)

	// Mock: Get project by name
	suite.mockProjectRepo.EXPECT().
		GetByName("empty-project").
		Return(project, nil)

	// Mock: No landscapes found for this project
	suite.mockLandscapeRepo.EXPECT().
		GetLandscapesByProjectID(projectID, 1000, 0).
		Return([]models.Landscape{}, int64(0), nil)

	// Mock: Cache set
	suite.mockCache.EXPECT().
		Set(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil)

	responses, err := suite.landscapeService.GetByProjectNameAll("empty-project")

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), responses)
	assert.Empty(suite.T(), responses)
}

// TestGetByProjectNameAll_ProjectNotFound tests when project doesn't exist
func (suite *LandscapeServiceTestSuite) TestGetByProjectNameAll_ProjectNotFound() {
	// Mock: Cache miss
	suite.mockCache.EXPECT().
		Get(gomock.Any()).
		Return(nil, cache.ErrCacheMiss)

	// Mock: Project not found
	suite.mockProjectRepo.EXPECT().
		GetByName("nonexistent-project").
		Return(nil, gorm.ErrRecordNotFound)

	responses, err := suite.landscapeService.GetByProjectNameAll("nonexistent-project")

	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), responses)
	assert.Equal(suite.T(), apperrors.ErrProjectNotFound, err)
}

// TestGetByProjectNameAll_ProjectRepositoryError tests project repository error
func (suite *LandscapeServiceTestSuite) TestGetByProjectNameAll_ProjectRepositoryError() {
	// Mock: Cache miss
	suite.mockCache.EXPECT().
		Get(gomock.Any()).
		Return(nil, cache.ErrCacheMiss)

	// Mock: Project repository error
	suite.mockProjectRepo.EXPECT().
		GetByName("my-project").
		Return(nil, errors.New("database connection error"))

	responses, err := suite.landscapeService.GetByProjectNameAll("my-project")

	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), responses)
	assert.Contains(suite.T(), err.Error(), "failed to find project")
}

// TestGetByProjectNameAll_LandscapeRepositoryError tests landscape repository error
func (suite *LandscapeServiceTestSuite) TestGetByProjectNameAll_LandscapeRepositoryError() {
	projectID := uuid.New()
	project := &models.Project{
		BaseModel: models.BaseModel{
			ID:   projectID,
			Name: "my-project",
		},
	}

	// Mock: Cache miss
	suite.mockCache.EXPECT().
		Get(gomock.Any()).
		Return(nil, cache.ErrCacheMiss)

	// Mock: Get project by name
	suite.mockProjectRepo.EXPECT().
		GetByName("my-project").
		Return(project, nil)

	// Mock: Landscape repository error
	suite.mockLandscapeRepo.EXPECT().
		GetLandscapesByProjectID(projectID, 1000, 0).
		Return(nil, int64(0), errors.New("database query failed"))

	responses, err := suite.landscapeService.GetByProjectNameAll("my-project")

	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), responses)
	assert.Contains(suite.T(), err.Error(), "failed to get landscapes by project")
}

// TestGetByProjectNameAll_CacheHit tests when data is retrieved from cache
func (suite *LandscapeServiceTestSuite) TestGetByProjectNameAll_CacheHit() {
	cachedResponses := []service.LandscapeMinimalResponse{
		{
			ID:          uuid.New(),
			Name:        "cached-landscape",
			Title:       "Cached Landscape",
			Domain:      "cached.com",
			Environment: "production",
		},
	}

	// Mock: Cache hit (data found in cache)
	suite.mockCache.EXPECT().
		Get(gomock.Any()).
		DoAndReturn(func(key string) ([]byte, error) {
			data, _ := json.Marshal(cachedResponses)
			return data, nil
		})

	responses, err := suite.landscapeService.GetByProjectNameAll("cached-project")

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), responses)
	assert.Len(suite.T(), responses, 1)
	assert.Equal(suite.T(), "cached-landscape", responses[0].Name)
}

// TestListByQuery_Success tests successful query with pagination conversion
func (suite *LandscapeServiceTestSuite) TestListByQuery_Success() {
	landscapes := []models.Landscape{
		{
			BaseModel: models.BaseModel{
				ID:    uuid.New(),
				Name:  "prod-landscape",
				Title: "Production Landscape",
			},
			ProjectID:   uuid.New(),
			Domain:      "prod.example.com",
			Environment: "production",
		},
		{
			BaseModel: models.BaseModel{
				ID:    uuid.New(),
				Name:  "dev-landscape",
				Title: "Development Landscape",
			},
			ProjectID:   uuid.New(),
			Domain:      "dev.example.com",
			Environment: "development",
		},
	}

	// Mock: Cache miss
	suite.mockCache.EXPECT().
		Get(gomock.Any()).
		Return(nil, cache.ErrCacheMiss)

	// ListByQuery converts limit=20, offset=40 to page=3, pageSize=20
	// Then calls Search which converts back to offset=(3-1)*20=40
	suite.mockLandscapeRepo.EXPECT().
		Search(uuid.Nil, "test", 20, 40).
		Return(landscapes, int64(100), nil)

	// Mock: Cache set
	suite.mockCache.EXPECT().
		Set(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil)

	response, err := suite.landscapeService.ListByQuery("test", nil, nil, 20, 40)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), response)
	assert.Len(suite.T(), response.Landscapes, 2)
	assert.Equal(suite.T(), int64(100), response.Total)
	assert.Equal(suite.T(), 3, response.Page)
	assert.Equal(suite.T(), 20, response.PageSize)
}

// TestListByQuery_FirstPage tests querying the first page
func (suite *LandscapeServiceTestSuite) TestListByQuery_FirstPage() {
	landscapes := []models.Landscape{
		{
			BaseModel: models.BaseModel{
				ID:    uuid.New(),
				Name:  "landscape1",
				Title: "Landscape 1",
			},
			ProjectID:   uuid.New(),
			Domain:      "domain1.com",
			Environment: "production",
		},
	}

	// Mock: Cache miss
	suite.mockCache.EXPECT().
		Get(gomock.Any()).
		Return(nil, cache.ErrCacheMiss)

	// ListByQuery converts limit=10, offset=0 to page=1, pageSize=10
	suite.mockLandscapeRepo.EXPECT().
		Search(uuid.Nil, "landscape", 10, 0).
		Return(landscapes, int64(1), nil)

	// Mock: Cache set
	suite.mockCache.EXPECT().
		Set(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil)

	response, err := suite.landscapeService.ListByQuery("landscape", nil, nil, 10, 0)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), response)
	assert.Len(suite.T(), response.Landscapes, 1)
	assert.Equal(suite.T(), 1, response.Page)
	assert.Equal(suite.T(), 10, response.PageSize)
}

// TestListByQuery_EmptyQuery tests search with empty query string
func (suite *LandscapeServiceTestSuite) TestListByQuery_EmptyQuery() {
	landscapes := []models.Landscape{
		{
			BaseModel: models.BaseModel{
				ID:    uuid.New(),
				Name:  "all-landscape",
				Title: "All Landscape",
			},
			ProjectID:   uuid.New(),
			Domain:      "all.example.com",
			Environment: "production",
		},
	}

	// Mock: Cache miss
	suite.mockCache.EXPECT().
		Get(gomock.Any()).
		Return(nil, cache.ErrCacheMiss)

	// Empty query should still work
	suite.mockLandscapeRepo.EXPECT().
		Search(uuid.Nil, "", 20, 0).
		Return(landscapes, int64(1), nil)

	// Mock: Cache set
	suite.mockCache.EXPECT().
		Set(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil)

	response, err := suite.landscapeService.ListByQuery("", nil, nil, 20, 0)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), response)
	assert.Len(suite.T(), response.Landscapes, 1)
}

// TestListByQuery_NoResults tests when search returns no results
func (suite *LandscapeServiceTestSuite) TestListByQuery_NoResults() {
	// Mock: Cache miss
	suite.mockCache.EXPECT().
		Get(gomock.Any()).
		Return(nil, cache.ErrCacheMiss)

	// No results found
	suite.mockLandscapeRepo.EXPECT().
		Search(uuid.Nil, "nonexistent", 20, 0).
		Return([]models.Landscape{}, int64(0), nil)

	// Mock: Cache set
	suite.mockCache.EXPECT().
		Set(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil)

	response, err := suite.landscapeService.ListByQuery("nonexistent", nil, nil, 20, 0)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), response)
	assert.Empty(suite.T(), response.Landscapes)
	assert.Equal(suite.T(), int64(0), response.Total)
}

// TestListByQuery_RepositoryError tests repository error handling
func (suite *LandscapeServiceTestSuite) TestListByQuery_RepositoryError() {
	// Mock: Cache miss
	suite.mockCache.EXPECT().
		Get(gomock.Any()).
		Return(nil, cache.ErrCacheMiss)

	// Repository returns error
	suite.mockLandscapeRepo.EXPECT().
		Search(uuid.Nil, "test", 20, 0).
		Return(nil, int64(0), errors.New("database connection error"))

	response, err := suite.landscapeService.ListByQuery("test", nil, nil, 20, 0)

	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), response)
	assert.Contains(suite.T(), err.Error(), "failed to search landscapes")
}

// TestListByQuery_PageSizeValidation tests page size validation through Search
func (suite *LandscapeServiceTestSuite) TestListByQuery_PageSizeValidation() {
	testCases := []struct {
		name             string
		inputLimit       int
		inputOffset      int
		expectedPageSize int
		expectedOffset   int
	}{
		{
			name:             "Valid limit within range",
			inputLimit:       50,
			inputOffset:      0,
			expectedPageSize: 50,
			expectedOffset:   0,
		},
		{
			name:             "Limit less than 1 - defaults to 20",
			inputLimit:       0,
			inputOffset:      0,
			expectedPageSize: 20,
			expectedOffset:   0,
		},
		{
			name:             "Limit greater than 100 - defaults to 20",
			inputLimit:       150,
			inputOffset:      0,
			expectedPageSize: 20,
			expectedOffset:   0,
		},
		{
			name:             "Negative limit - defaults to 20",
			inputLimit:       -10,
			inputOffset:      0,
			expectedPageSize: 20,
			expectedOffset:   0,
		},
	}

	for _, tc := range testCases {
		suite.T().Run(tc.name, func(t *testing.T) {
			// Mock: Cache miss
			suite.mockCache.EXPECT().
				Get(gomock.Any()).
				Return(nil, cache.ErrCacheMiss)

			// Verify the corrected pageSize is used
			suite.mockLandscapeRepo.EXPECT().
				Search(uuid.Nil, "test", tc.expectedPageSize, tc.expectedOffset).
				Return([]models.Landscape{}, int64(0), nil)

			// Mock: Cache set
			suite.mockCache.EXPECT().
				Set(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(nil)

			response, err := suite.landscapeService.ListByQuery("test", nil, nil, tc.inputLimit, tc.inputOffset)

			assert.NoError(t, err)
			assert.NotNil(t, response)
			assert.Equal(t, tc.expectedPageSize, response.PageSize)
		})
	}
}

// TestListByQuery_PaginationConversion tests limit/offset to page/pageSize conversion
func (suite *LandscapeServiceTestSuite) TestListByQuery_PaginationConversion() {
	testCases := []struct {
		name           string
		limit          int
		offset         int
		expectedPage   int
		expectedOffset int
	}{
		{
			name:           "Page 1: offset=0, limit=20",
			limit:          20,
			offset:         0,
			expectedPage:   1,
			expectedOffset: 0,
		},
		{
			name:           "Page 2: offset=20, limit=20",
			limit:          20,
			offset:         20,
			expectedPage:   2,
			expectedOffset: 20,
		},
		{
			name:           "Page 3: offset=40, limit=20",
			limit:          20,
			offset:         40,
			expectedPage:   3,
			expectedOffset: 40,
		},
		{
			name:           "Page 5: offset=200, limit=50",
			limit:          50,
			offset:         200,
			expectedPage:   5,
			expectedOffset: 200,
		},
	}

	for _, tc := range testCases {
		suite.T().Run(tc.name, func(t *testing.T) {
			// Mock: Cache miss
			suite.mockCache.EXPECT().
				Get(gomock.Any()).
				Return(nil, cache.ErrCacheMiss)

			suite.mockLandscapeRepo.EXPECT().
				Search(uuid.Nil, "test", tc.limit, tc.expectedOffset).
				Return([]models.Landscape{}, int64(0), nil)

			// Mock: Cache set
			suite.mockCache.EXPECT().
				Set(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(nil)

			response, err := suite.landscapeService.ListByQuery("test", nil, nil, tc.limit, tc.offset)

			assert.NoError(t, err)
			assert.NotNil(t, response)
			assert.Equal(t, tc.expectedPage, response.Page)
		})
	}
}

// TestListByQuery_CacheHit tests when data is retrieved from cache
func (suite *LandscapeServiceTestSuite) TestListByQuery_CacheHit() {
	cachedResponse := &service.LandscapeListResponse{
		Landscapes: []service.LandscapeResponse{
			{
				ID:    uuid.New(),
				Name:  "cached-landscape",
				Title: "Cached Landscape",
			},
		},
		Total:    1,
		Page:     1,
		PageSize: 20,
	}

	// Mock: Cache hit
	suite.mockCache.EXPECT().
		Get(gomock.Any()).
		DoAndReturn(func(key string) ([]byte, error) {
			data, _ := json.Marshal(cachedResponse)
			return data, nil
		})

	response, err := suite.landscapeService.ListByQuery("cached", nil, nil, 20, 0)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), response)
	assert.Len(suite.T(), response.Landscapes, 1)
	assert.Equal(suite.T(), "cached-landscape", response.Landscapes[0].Name)
}

// TestSearch_Success tests successful search with valid parameters
func (suite *LandscapeServiceTestSuite) TestSearch_Success() {
	orgID := uuid.New()
	landscapes := []models.Landscape{
		{
			BaseModel: models.BaseModel{
				ID:    uuid.New(),
				Name:  "prod-landscape",
				Title: "Production Landscape",
			},
			ProjectID:   uuid.New(),
			Domain:      "prod.example.com",
			Environment: "production",
		},
		{
			BaseModel: models.BaseModel{
				ID:    uuid.New(),
				Name:  "dev-landscape",
				Title: "Development Landscape",
			},
			ProjectID:   uuid.New(),
			Domain:      "dev.example.com",
			Environment: "development",
		},
	}

	// Mock: Cache miss
	suite.mockCache.EXPECT().
		Get(gomock.Any()).
		Return(nil, cache.ErrCacheMiss)

	// Search with page=2, pageSize=20 should convert to offset=20
	suite.mockLandscapeRepo.EXPECT().
		Search(uuid.Nil, "landscape", 20, 20).
		Return(landscapes, int64(50), nil)

	// Mock: Cache set
	suite.mockCache.EXPECT().
		Set(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil)

	response, err := suite.landscapeService.Search(orgID, "landscape", 2, 20)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), response)
	assert.Len(suite.T(), response.Landscapes, 2)
	assert.Equal(suite.T(), int64(50), response.Total)
	assert.Equal(suite.T(), 2, response.Page)
	assert.Equal(suite.T(), 20, response.PageSize)
	assert.Equal(suite.T(), "prod-landscape", response.Landscapes[0].Name)
	assert.Equal(suite.T(), "dev-landscape", response.Landscapes[1].Name)
}

// TestSearch_EmptyQuery tests search with empty query string
func (suite *LandscapeServiceTestSuite) TestSearch_EmptyQuery() {
	orgID := uuid.New()
	landscapes := []models.Landscape{
		{
			BaseModel: models.BaseModel{
				ID:    uuid.New(),
				Name:  "all-landscape",
				Title: "All Landscape",
			},
			ProjectID:   uuid.New(),
			Domain:      "all.example.com",
			Environment: "production",
		},
	}

	// Mock: Cache miss
	suite.mockCache.EXPECT().
		Get(gomock.Any()).
		Return(nil, cache.ErrCacheMiss)

	// Empty query should still work
	suite.mockLandscapeRepo.EXPECT().
		Search(uuid.Nil, "", 20, 0).
		Return(landscapes, int64(1), nil)

	// Mock: Cache set
	suite.mockCache.EXPECT().
		Set(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil)

	response, err := suite.landscapeService.Search(orgID, "", 1, 20)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), response)
	assert.Len(suite.T(), response.Landscapes, 1)
}

// TestSearch_NoResults tests when search returns no results
func (suite *LandscapeServiceTestSuite) TestSearch_NoResults() {
	orgID := uuid.New()

	// Mock: Cache miss
	suite.mockCache.EXPECT().
		Get(gomock.Any()).
		Return(nil, cache.ErrCacheMiss)

	// No results found
	suite.mockLandscapeRepo.EXPECT().
		Search(uuid.Nil, "nonexistent", 20, 0).
		Return([]models.Landscape{}, int64(0), nil)

	// Mock: Cache set
	suite.mockCache.EXPECT().
		Set(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil)

	response, err := suite.landscapeService.Search(orgID, "nonexistent", 1, 20)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), response)
	assert.Empty(suite.T(), response.Landscapes)
	assert.Equal(suite.T(), int64(0), response.Total)
}

// TestSearch_RepositoryError tests repository error handling
func (suite *LandscapeServiceTestSuite) TestSearch_RepositoryError() {
	orgID := uuid.New()

	// Mock: Cache miss
	suite.mockCache.EXPECT().
		Get(gomock.Any()).
		Return(nil, cache.ErrCacheMiss)

	// Repository returns error
	suite.mockLandscapeRepo.EXPECT().
		Search(uuid.Nil, "test", 20, 0).
		Return(nil, int64(0), errors.New("database connection error"))

	response, err := suite.landscapeService.Search(orgID, "test", 1, 20)

	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), response)
	assert.Contains(suite.T(), err.Error(), "failed to search landscapes")
}

// TestSearch_PageValidation tests page validation
func (suite *LandscapeServiceTestSuite) TestSearch_PageValidation() {
	orgID := uuid.New()

	testCases := []struct {
		name         string
		inputPage    int
		expectedPage int
	}{
		{
			name:         "Valid page",
			inputPage:    5,
			expectedPage: 5,
		},
		{
			name:         "Page less than 1 - defaults to 1",
			inputPage:    0,
			expectedPage: 1,
		},
		{
			name:         "Negative page - defaults to 1",
			inputPage:    -5,
			expectedPage: 1,
		},
	}

	for _, tc := range testCases {
		suite.T().Run(tc.name, func(t *testing.T) {
			// Mock: Cache miss
			suite.mockCache.EXPECT().
				Get(gomock.Any()).
				Return(nil, cache.ErrCacheMiss)

			// Calculate expected offset
			expectedOffset := (tc.expectedPage - 1) * 20

			suite.mockLandscapeRepo.EXPECT().
				Search(uuid.Nil, "test", 20, expectedOffset).
				Return([]models.Landscape{}, int64(0), nil)

			// Mock: Cache set
			suite.mockCache.EXPECT().
				Set(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(nil)

			response, err := suite.landscapeService.Search(orgID, "test", tc.inputPage, 20)

			assert.NoError(t, err)
			assert.NotNil(t, response)
			assert.Equal(t, tc.expectedPage, response.Page)
		})
	}
}

// TestSearch_PageSizeValidation tests page size validation
func (suite *LandscapeServiceTestSuite) TestSearch_PageSizeValidation() {
	orgID := uuid.New()

	testCases := []struct {
		name             string
		inputPageSize    int
		expectedPageSize int
	}{
		{
			name:             "Valid page size within range (50)",
			inputPageSize:    50,
			expectedPageSize: 50,
		},
		{
			name:             "Page size at lower boundary (1)",
			inputPageSize:    1,
			expectedPageSize: 1,
		},
		{
			name:             "Page size at upper boundary (100)",
			inputPageSize:    100,
			expectedPageSize: 100,
		},
		{
			name:             "Page size less than 1 - defaults to 20",
			inputPageSize:    0,
			expectedPageSize: 20,
		},
		{
			name:             "Negative page size - defaults to 20",
			inputPageSize:    -10,
			expectedPageSize: 20,
		},
		{
			name:             "Page size greater than 100 - defaults to 20",
			inputPageSize:    150,
			expectedPageSize: 20,
		},
	}

	for _, tc := range testCases {
		suite.T().Run(tc.name, func(t *testing.T) {
			// Mock: Cache miss
			suite.mockCache.EXPECT().
				Get(gomock.Any()).
				Return(nil, cache.ErrCacheMiss)

			suite.mockLandscapeRepo.EXPECT().
				Search(uuid.Nil, "test", tc.expectedPageSize, 0).
				Return([]models.Landscape{}, int64(0), nil)

			// Mock: Cache set
			suite.mockCache.EXPECT().
				Set(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(nil)

			response, err := suite.landscapeService.Search(orgID, "test", 1, tc.inputPageSize)

			assert.NoError(t, err)
			assert.NotNil(t, response)
			assert.Equal(t, tc.expectedPageSize, response.PageSize)
		})
	}
}

// TestSearch_CacheHit tests when data is retrieved from cache
func (suite *LandscapeServiceTestSuite) TestSearch_CacheHit() {
	orgID := uuid.New()
	cachedResponse := &service.LandscapeListResponse{
		Landscapes: []service.LandscapeResponse{
			{
				ID:    uuid.New(),
				Name:  "cached-landscape",
				Title: "Cached Landscape",
			},
		},
		Total:    1,
		Page:     1,
		PageSize: 20,
	}

	// Mock: Cache hit
	suite.mockCache.EXPECT().
		Get(gomock.Any()).
		DoAndReturn(func(key string) ([]byte, error) {
			data, _ := json.Marshal(cachedResponse)
			return data, nil
		})

	response, err := suite.landscapeService.Search(orgID, "cached", 1, 20)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), response)
	assert.Len(suite.T(), response.Landscapes, 1)
	assert.Equal(suite.T(), "cached-landscape", response.Landscapes[0].Name)
}

// Run the test suite
func TestLandscapeServiceTestSuite(t *testing.T) {
	suite.Run(t, new(LandscapeServiceTestSuite))
}

// CacheIntegrationTestSuite tests caching behavior with real cache
type CacheIntegrationTestSuite struct {
	suite.Suite
	cache     *cache.InMemoryCache
	ttlConfig cache.TTLConfig
}

func (suite *CacheIntegrationTestSuite) SetupTest() {
	cacheConfig := cache.CacheConfig{
		DefaultTTL:      100 * time.Millisecond,
		CleanupInterval: 50 * time.Millisecond,
		Enabled:         true,
	}
	suite.cache = cache.NewInMemoryCache(cacheConfig)
	suite.ttlConfig = cache.DefaultTTLConfig()
}

func (suite *CacheIntegrationTestSuite) TestCacheWrapper_GetOrFetch() {
	wrapper := cache.NewCacheWrapper[service.LandscapeResponse](suite.cache)

	expectedResponse := service.LandscapeResponse{
		ID:          uuid.New(),
		Name:        "cached-landscape",
		Title:       "Cached Landscape",
		Description: "From cache",
	}

	// First call should fetch
	fetchCount := 0
	result, err := wrapper.GetOrFetch("test-key", 5*time.Minute, func() (service.LandscapeResponse, error) {
		fetchCount++
		return expectedResponse, nil
	})

	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), expectedResponse.Name, result.Name)
	assert.Equal(suite.T(), 1, fetchCount)

	// Second call should use cache
	result, err = wrapper.GetOrFetch("test-key", 5*time.Minute, func() (service.LandscapeResponse, error) {
		fetchCount++
		return service.LandscapeResponse{Name: "should-not-fetch"}, nil
	})

	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), expectedResponse.Name, result.Name)
	assert.Equal(suite.T(), 1, fetchCount) // Should still be 1
}

func (suite *CacheIntegrationTestSuite) TestCacheInvalidation() {
	wrapper := cache.NewCacheWrapper[string](suite.cache)

	// Populate cache
	_, _ = wrapper.GetOrFetch("invalidate-test", 5*time.Minute, func() (string, error) {
		return "original", nil
	})

	// Invalidate
	err := wrapper.Invalidate("invalidate-test")
	assert.NoError(suite.T(), err)

	// Next fetch should call the function again
	fetchCount := 0
	result, err := wrapper.GetOrFetch("invalidate-test", 5*time.Minute, func() (string, error) {
		fetchCount++
		return "new-value", nil
	})

	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "new-value", result)
	assert.Equal(suite.T(), 1, fetchCount)
}

func (suite *CacheIntegrationTestSuite) TestCacheClear() {
	// Add multiple items
	suite.cache.Set("key1", []byte("value1"), 5*time.Minute)
	suite.cache.Set("key2", []byte("value2"), 5*time.Minute)
	suite.cache.Set("key3", []byte("value3"), 5*time.Minute)

	// Clear all
	suite.cache.Clear()

	// All should be gone
	_, err := suite.cache.Get("key1")
	assert.ErrorIs(suite.T(), err, cache.ErrCacheMiss)
	_, err = suite.cache.Get("key2")
	assert.ErrorIs(suite.T(), err, cache.ErrCacheMiss)
	_, err = suite.cache.Get("key3")
	assert.ErrorIs(suite.T(), err, cache.ErrCacheMiss)
}

func TestCacheIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(CacheIntegrationTestSuite))
}
