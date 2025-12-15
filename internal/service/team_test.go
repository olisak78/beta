package service_test

import (
	apperrors "developer-portal-backend/internal/errors"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"developer-portal-backend/internal/database/models"
	"developer-portal-backend/internal/mocks"
	"developer-portal-backend/internal/service"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
	"gorm.io/gorm"
)

// TeamServiceTestSuite defines the test suite for TeamService
type TeamServiceTestSuite struct {
	suite.Suite
	ctrl              *gomock.Controller
	mockTeamRepo      *mocks.MockTeamRepositoryInterface
	mockGroupRepo     *mocks.MockGroupRepositoryInterface
	mockOrgRepo       *mocks.MockOrganizationRepositoryInterface
	mockUserRepo      *mocks.MockUserRepositoryInterface
	mockLinkRepo      *mocks.MockLinkRepositoryInterface
	mockComponentRepo *mocks.MockComponentRepositoryInterface
	teamService       *service.TeamService
	validator         *validator.Validate
}

// SetupTest sets up the test suite
func (suite *TeamServiceTestSuite) SetupTest() {
	suite.ctrl = gomock.NewController(suite.T())
	suite.mockTeamRepo = mocks.NewMockTeamRepositoryInterface(suite.ctrl)
	suite.mockGroupRepo = mocks.NewMockGroupRepositoryInterface(suite.ctrl)
	suite.mockOrgRepo = mocks.NewMockOrganizationRepositoryInterface(suite.ctrl)
	suite.mockUserRepo = mocks.NewMockUserRepositoryInterface(suite.ctrl)
	suite.mockLinkRepo = mocks.NewMockLinkRepositoryInterface(suite.ctrl)
	suite.mockComponentRepo = mocks.NewMockComponentRepositoryInterface(suite.ctrl)
	suite.validator = validator.New()

	// Initialize TeamService with mocked dependencies
	suite.teamService = service.NewTeamService(
		suite.mockTeamRepo,
		suite.mockGroupRepo,
		suite.mockOrgRepo,
		suite.mockUserRepo,
		suite.mockLinkRepo,
		suite.mockComponentRepo,
		suite.validator,
	)
}

// TearDownTest cleans up after each test
func (suite *TeamServiceTestSuite) TearDownTest() {
	suite.ctrl.Finish()
}

// GetByID Tests

func (suite *TeamServiceTestSuite) TestGetByID_Success() {
	teamID := uuid.New()
	groupID := uuid.New()
	orgID := uuid.New()

	team := &models.Team{
		BaseModel: models.BaseModel{
			ID:          teamID,
			Name:        "backend-team",
			Title:       "Backend Team",
			Description: "Team responsible for backend services",
		},
		GroupID:    groupID,
		Owner:      "I12345",
		Email:      "backend@example.com",
		PictureURL: "https://example.com/team.png",
	}

	group := &models.Group{
		BaseModel: models.BaseModel{
			ID:   groupID,
			Name: "engineering",
		},
		OrgID: orgID,
	}

	// Mock expectations
	suite.mockTeamRepo.EXPECT().GetByID(teamID).Return(team, nil)
	suite.mockGroupRepo.EXPECT().GetByID(groupID).Return(group, nil)

	// Execute
	result, err := suite.teamService.GetByID(teamID)

	// Assert
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.Equal(suite.T(), teamID, result.ID)
	assert.Equal(suite.T(), "backend-team", result.Name)
	assert.Equal(suite.T(), "Backend Team", result.Title)
	assert.Equal(suite.T(), "Team responsible for backend services", result.Description)
	assert.Equal(suite.T(), groupID, result.GroupID)
	assert.Equal(suite.T(), orgID, result.OrganizationID)
	assert.Equal(suite.T(), "I12345", result.Owner)
	assert.Equal(suite.T(), "backend@example.com", result.Email)
	assert.Equal(suite.T(), "https://example.com/team.png", result.PictureURL)
}

func (suite *TeamServiceTestSuite) TestGetByID_NotFound() {
	teamID := uuid.New()

	// Mock expectations - return gorm.ErrRecordNotFound
	suite.mockTeamRepo.EXPECT().GetByID(teamID).Return(nil, gorm.ErrRecordNotFound)

	// Execute
	result, err := suite.teamService.GetByID(teamID)

	// Assert
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), result)
	assert.ErrorIs(suite.T(), err, apperrors.ErrTeamNotFound)
}

func (suite *TeamServiceTestSuite) TestGetByID_RepositoryError() {
	teamID := uuid.New()
	expectedError := errors.New("database connection error")

	// Mock expectations - return a generic error
	suite.mockTeamRepo.EXPECT().GetByID(teamID).Return(nil, expectedError)

	// Execute
	result, err := suite.teamService.GetByID(teamID)

	// Assert
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), result)
	assert.Contains(suite.T(), err.Error(), "failed to get team")
	assert.NotErrorIs(suite.T(), err, apperrors.ErrTeamNotFound)
}

func (suite *TeamServiceTestSuite) TestGetByID_GroupRepoError() {
	teamID := uuid.New()
	groupID := uuid.New()

	team := &models.Team{
		BaseModel: models.BaseModel{
			ID:          teamID,
			Name:        "backend-team",
			Title:       "Backend Team",
			Description: "Team responsible for backend services",
		},
		GroupID:    groupID,
		Owner:      "I12345",
		Email:      "backend@example.com",
		PictureURL: "https://example.com/team.png",
	}

	// Mock expectations
	suite.mockTeamRepo.EXPECT().GetByID(teamID).Return(team, nil)
	suite.mockGroupRepo.EXPECT().GetByID(groupID).Return(nil, errors.New("group not found"))

	// Execute
	result, err := suite.teamService.GetByID(teamID)

	// Assert - should now fail when group lookup fails (proper error handling)
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), result)
	assert.Contains(suite.T(), err.Error(), "failed to get group for team")
}

// TestCreateTeamValidation tests the validation logic for creating a team
func (suite *TeamServiceTestSuite) TestCreateTeamValidation() {
	validator := validator.New()

	testCases := []struct {
		name        string
		request     *service.CreateTeamRequest
		expectError bool
		errorMsg    string
	}{
		{
			name: "Valid request",
			request: &service.CreateTeamRequest{
				GroupID:     uuid.New(),
				Name:        "backend-team",
				Title:       "Backend Team",
				Description: "Team responsible for backend services",
				Owner:       "I12345",
				Email:       "backend-team@test.com",
				PictureURL:  "https://example.com/team.png",
			},
			expectError: false,
		},
		{
			name: "Missing group ID",
			request: &service.CreateTeamRequest{
				Name:  "backend-team",
				Title: "Backend Team",
			},
			expectError: true,
			errorMsg:    "GroupID",
		},
		{
			name: "Empty name",
			request: &service.CreateTeamRequest{
				GroupID: uuid.New(),
				Name:    "",
				Title:   "Backend Team",
			},
			expectError: true,
			errorMsg:    "Name",
		},
		{
			name: "Empty title",
			request: &service.CreateTeamRequest{
				GroupID: uuid.New(),
				Name:    "backend-team",
				Title:   "",
			},
			expectError: true,
			errorMsg:    "Title",
		},
		{
			name: "Name too long",
			request: &service.CreateTeamRequest{
				GroupID: uuid.New(),
				Name:    "this-is-a-very-long-team-name-that-definitely-exceeds-one-hundred-characters-which-is-the-maximum-allowed-length-for-team-names-in-this-system-validation",
				Title:   "Backend Team",
			},
			expectError: true,
			errorMsg:    "Name",
		},
		{
			name: "Display name too long",
			request: &service.CreateTeamRequest{
				GroupID: uuid.New(),
				Name:    "backend-team",
				Title:   "This is a very long display name that definitely exceeds the maximum allowed length of two hundred characters for the display name field and should trigger a validation error when we try to create a team with this overly long display name that goes beyond the limit",
			},
			expectError: true,
			errorMsg:    "Title",
		},
	}

	for _, tc := range testCases {
		suite.T().Run(tc.name, func(t *testing.T) {
			err := validator.Struct(tc.request)
			if tc.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestUpdateTeamValidation tests the validation logic for updating a team
func (suite *TeamServiceTestSuite) TestUpdateTeamValidation() {
	validator := validator.New()

	testCases := []struct {
		name        string
		request     *service.UpdateTeamRequest
		expectError bool
		errorMsg    string
	}{
		{
			name: "Valid request",
			request: &service.UpdateTeamRequest{
				Title:       "Updated Backend Team",
				Description: "Updated description",
				Owner:       "I67890",
				Email:       "backend-updated@test.com",
				PictureURL:  "https://example.com/team-updated.png",
			},
			expectError: false,
		},
		{
			name: "Display name too long",
			request: &service.UpdateTeamRequest{
				Title: "This is an extremely long display name that definitely exceeds the maximum allowed length of exactly two hundred characters for the display name field and should absolutely trigger a validation error when we try to update a team with this incredibly long display name that goes way beyond the specified character limit of two hundred characters making it invalid for our validation system",
			},
			expectError: true,
			errorMsg:    "Title",
		},
	}

	for _, tc := range testCases {
		suite.T().Run(tc.name, func(t *testing.T) {
			err := validator.Struct(tc.request)
			if tc.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestTeamResponseSerialization tests the team response serialization
func (suite *TeamServiceTestSuite) TestTeamResponseSerialization() {
	teamID := uuid.New()
	orgID := uuid.New()
	metadata := json.RawMessage(`{"tags": ["backend", "api"]}`)

	response := &service.TeamResponse{
		ID:             teamID,
		OrganizationID: orgID,
		GroupID:        uuid.New(),
		Name:           "backend-team",
		Title:          "Backend Team",
		Description:    "Team responsible for backend services",
		Metadata:       metadata,
		CreatedAt:      "2023-01-01T00:00:00Z",
		UpdatedAt:      "2023-01-01T00:00:00Z",
	}

	// Test JSON marshaling
	jsonData, err := json.Marshal(response)
	assert.NoError(suite.T(), err)
	assert.Contains(suite.T(), string(jsonData), teamID.String())
	assert.Contains(suite.T(), string(jsonData), "backend-team")
	assert.Contains(suite.T(), string(jsonData), "Backend Team")

	// Test JSON unmarshaling
	var unmarshaled service.TeamResponse
	err = json.Unmarshal(jsonData, &unmarshaled)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), response.ID, unmarshaled.ID)
	assert.Equal(suite.T(), response.Name, unmarshaled.Name)
	assert.Equal(suite.T(), response.Title, unmarshaled.Title)
}

// TestTeamListResponseSerialization tests the team list response serialization
func (suite *TeamServiceTestSuite) TestTeamListResponseSerialization() {
	teams := []service.TeamResponse{
		{
			ID:             uuid.New(),
			OrganizationID: uuid.New(),
			Name:           "team-1",
			Title:          "Team 1",
			CreatedAt:      "2023-01-01T00:00:00Z",
			UpdatedAt:      "2023-01-01T00:00:00Z",
		},
		{
			ID:             uuid.New(),
			OrganizationID: uuid.New(),
			Name:           "team-2",
			Title:          "Team 2",
			CreatedAt:      "2023-01-01T00:00:00Z",
			UpdatedAt:      "2023-01-01T00:00:00Z",
		},
	}

	response := &service.TeamListResponse{
		Teams:    teams,
		Total:    int64(len(teams)),
		Page:     1,
		PageSize: 20,
	}

	// Test JSON marshaling
	jsonData, err := json.Marshal(response)
	assert.NoError(suite.T(), err)
	assert.Contains(suite.T(), string(jsonData), "team-1")
	assert.Contains(suite.T(), string(jsonData), "team-2")
	assert.Contains(suite.T(), string(jsonData), `"total":2`)
	assert.Contains(suite.T(), string(jsonData), `"page":1`)

	// Test JSON unmarshaling
	var unmarshaled service.TeamListResponse
	err = json.Unmarshal(jsonData, &unmarshaled)
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), unmarshaled.Teams, 2)
	assert.Equal(suite.T(), response.Total, unmarshaled.Total)
	assert.Equal(suite.T(), response.Page, unmarshaled.Page)
	assert.Equal(suite.T(), response.PageSize, unmarshaled.PageSize)
}

// TestDefaultStatusBehavior tests the default status behavior

// TestPaginationLogic tests the pagination logic
func (suite *TeamServiceTestSuite) TestPaginationLogic() {
	testCases := []struct {
		name           string
		inputPage      int
		inputSize      int
		expectedPage   int
		expectedSize   int
		expectedOffset int
	}{
		{
			name:           "Valid pagination",
			inputPage:      2,
			inputSize:      10,
			expectedPage:   2,
			expectedSize:   10,
			expectedOffset: 10,
		},
		{
			name:           "Page less than 1",
			inputPage:      0,
			inputSize:      10,
			expectedPage:   1,
			expectedSize:   10,
			expectedOffset: 0,
		},
		{
			name:           "Page size less than 1",
			inputPage:      1,
			inputSize:      0,
			expectedPage:   1,
			expectedSize:   20,
			expectedOffset: 0,
		},
		{
			name:           "Page size greater than 100",
			inputPage:      1,
			inputSize:      150,
			expectedPage:   1,
			expectedSize:   20,
			expectedOffset: 0,
		},
		{
			name:           "Both invalid",
			inputPage:      -1,
			inputSize:      -5,
			expectedPage:   1,
			expectedSize:   20,
			expectedOffset: 0,
		},
	}

	for _, tc := range testCases {
		suite.T().Run(tc.name, func(t *testing.T) {
			// Simulate the pagination logic from the service
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

// TestTeamStatusValidation tests team status validation
func (suite *TeamServiceTestSuite) TestTeamStatusValidation() {
	// Status enums removed from the model; placeholder to keep suite stable
	assert.True(suite.T(), true)
}

// TestJSONFieldsHandling tests handling of JSON fields (Links and Metadata)
func (suite *TeamServiceTestSuite) TestJSONFieldsHandling() {
	// Test valid JSON
	validLinks := json.RawMessage(`{"slack": "https://slack.com/team", "github": "https://github.com/team"}`)
	validMetadata := json.RawMessage(`{"tags": ["backend", "api"], "priority": "high"}`)

	// Test that valid JSON can be marshaled and unmarshaled
	linksData, err := json.Marshal(validLinks)
	assert.NoError(suite.T(), err)
	assert.NotEmpty(suite.T(), linksData)

	metadataData, err := json.Marshal(validMetadata)
	assert.NoError(suite.T(), err)
	assert.NotEmpty(suite.T(), metadataData)

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

// TestComponentPaginationLogic tests component pagination logic
func (suite *TeamServiceTestSuite) TestComponentPaginationLogic() {
	// Simulate the manual pagination logic for components
	components := []models.Component{
		{BaseModel: models.BaseModel{ID: uuid.New(), Name: "comp1"}},
		{BaseModel: models.BaseModel{ID: uuid.New(), Name: "comp2"}},
		{BaseModel: models.BaseModel{ID: uuid.New(), Name: "comp3"}},
		{BaseModel: models.BaseModel{ID: uuid.New(), Name: "comp4"}},
		{BaseModel: models.BaseModel{ID: uuid.New(), Name: "comp5"}},
	}

	testCases := []struct {
		name          string
		page          int
		pageSize      int
		expectedStart int
		expectedEnd   int
		expectedCount int
	}{
		{
			name:          "First page",
			page:          1,
			pageSize:      2,
			expectedStart: 0,
			expectedEnd:   2,
			expectedCount: 2,
		},
		{
			name:          "Second page",
			page:          2,
			pageSize:      2,
			expectedStart: 2,
			expectedEnd:   4,
			expectedCount: 2,
		},
		{
			name:          "Last page partial",
			page:          3,
			pageSize:      2,
			expectedStart: 4,
			expectedEnd:   5,
			expectedCount: 1,
		},
		{
			name:          "Page beyond data",
			page:          4,
			pageSize:      2,
			expectedStart: 6,
			expectedEnd:   6,
			expectedCount: 0,
		},
	}

	for _, tc := range testCases {
		suite.T().Run(tc.name, func(t *testing.T) {
			// Simulate the pagination logic from GetTeamComponentsByName
			page := tc.page
			pageSize := tc.pageSize

			if page < 1 {
				page = 1
			}
			if pageSize < 1 || pageSize > 100 {
				pageSize = 20
			}

			start := (page - 1) * pageSize
			end := start + pageSize

			assert.Equal(t, tc.expectedStart, start)

			var result []models.Component
			if start >= len(components) {
				result = []models.Component{}
			} else {
				actualEnd := end
				if actualEnd > len(components) {
					actualEnd = len(components)
				}
				result = components[start:actualEnd]
			}

			assert.Equal(t, tc.expectedCount, len(result))
		})
	}
}

// TestAddLinkValidation tests the validation logic for adding a link
func TestAddLinkValidation(t *testing.T) {
	validator := validator.New()

	testCases := []struct {
		name        string
		request     *service.AddLinkRequest
		expectError bool
	}{
		{
			name: "Valid link",
			request: &service.AddLinkRequest{
				URL:   "https://github.com/myteam/repo",
				Title: "Team Repository",
			},
			expectError: false,
		},
		{
			name: "Valid link without optional fields",
			request: &service.AddLinkRequest{
				URL:   "https://example.com",
				Title: "Example",
			},
			expectError: false,
		},
		{
			name: "Missing URL",
			request: &service.AddLinkRequest{
				Title: "Team Repository",
			},
			expectError: true,
		},
		{
			name: "Invalid URL",
			request: &service.AddLinkRequest{
				URL:   "not-a-url",
				Title: "Team Repository",
			},
			expectError: true,
		},
		{
			name: "Missing title",
			request: &service.AddLinkRequest{
				URL: "https://github.com/myteam/repo",
			},
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validator.Struct(tc.request)
			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestLinkJSONMarshaling tests that links can be properly marshaled and unmarshaled

// TestTechnicalTeamFiltering tests that the technical team is filtered out from results
func (suite *TeamServiceTestSuite) TestTechnicalTeamFiltering() {
	// Test that the technical team name is correctly filtered
	technicalTeamName := "team-developer-portal-technical"
	regularTeamName := "team-backend"

	// Test filtering logic
	teams := []models.Team{
		{BaseModel: models.BaseModel{Name: regularTeamName, Title: "Backend Team"}},
		{BaseModel: models.BaseModel{Name: technicalTeamName, Title: "Technical Team"}},
		{BaseModel: models.BaseModel{Name: "team-frontend", Title: "Frontend Team"}},
	}

	// Simulate the filtering logic from GetAllTeams
	filteredTeams := make([]models.Team, 0, len(teams))
	for _, team := range teams {
		if team.Name != technicalTeamName {
			filteredTeams = append(filteredTeams, team)
		}
	}

	// Assert that technical team is filtered out
	assert.Len(suite.T(), filteredTeams, 2, "Should have 2 teams after filtering")
	assert.Equal(suite.T(), regularTeamName, filteredTeams[0].Name, "First team should be backend team")
	assert.Equal(suite.T(), "team-frontend", filteredTeams[1].Name, "Second team should be frontend team")

	// Verify technical team is not in results
	for _, team := range filteredTeams {
		assert.NotEqual(suite.T(), technicalTeamName, team.Name, "Technical team should not be in filtered results")
	}
}

// TestTechnicalTeamFilteringTotalAdjustment tests that the total count is adjusted correctly
func (suite *TeamServiceTestSuite) TestTechnicalTeamFilteringTotalAdjustment() {
	testCases := []struct {
		name                  string
		totalFromDB           int64
		teamsFromDB           int
		filteredTeamsCount    int
		expectedAdjustedTotal int64
	}{
		{
			name:                  "Technical team present - adjust total",
			totalFromDB:           10,
			teamsFromDB:           5,
			filteredTeamsCount:    4,
			expectedAdjustedTotal: 9, // 10 - 1
		},
		{
			name:                  "Technical team not present - no adjustment",
			totalFromDB:           10,
			teamsFromDB:           5,
			filteredTeamsCount:    5,
			expectedAdjustedTotal: 10,
		},
		{
			name:                  "Single technical team - adjust to zero",
			totalFromDB:           1,
			teamsFromDB:           1,
			filteredTeamsCount:    0,
			expectedAdjustedTotal: 0,
		},
	}

	for _, tc := range testCases {
		suite.T().Run(tc.name, func(t *testing.T) {
			// Simulate the adjustment logic from GetByOrganization and Search
			adjustedTotal := tc.totalFromDB
			if tc.teamsFromDB > tc.filteredTeamsCount {
				adjustedTotal = tc.totalFromDB - int64(tc.teamsFromDB-tc.filteredTeamsCount)
			}

			assert.Equal(t, tc.expectedAdjustedTotal, adjustedTotal, "Total should be adjusted correctly")
		})
	}
}

// TestTechnicalTeamFilteringEdgeCases tests edge cases for technical team filtering
func (suite *TeamServiceTestSuite) TestTechnicalTeamFilteringEdgeCases() {
	technicalTeamName := "team-developer-portal-technical"

	testCases := []struct {
		name          string
		teams         []models.Team
		expectedCount int
		expectedNames []string
	}{
		{
			name:          "Empty team list",
			teams:         []models.Team{},
			expectedCount: 0,
			expectedNames: []string{},
		},
		{
			name: "Only technical team",
			teams: []models.Team{
				{BaseModel: models.BaseModel{Name: technicalTeamName}},
			},
			expectedCount: 0,
			expectedNames: []string{},
		},
		{
			name: "Multiple technical teams (shouldn't happen, but handle it)",
			teams: []models.Team{
				{BaseModel: models.BaseModel{Name: technicalTeamName}},
				{BaseModel: models.BaseModel{Name: technicalTeamName}},
				{BaseModel: models.BaseModel{Name: "team-regular"}},
			},
			expectedCount: 1,
			expectedNames: []string{"team-regular"},
		},
		{
			name: "Technical team in middle",
			teams: []models.Team{
				{BaseModel: models.BaseModel{Name: "team-first"}},
				{BaseModel: models.BaseModel{Name: technicalTeamName}},
				{BaseModel: models.BaseModel{Name: "team-last"}},
			},
			expectedCount: 2,
			expectedNames: []string{"team-first", "team-last"},
		},
		{
			name: "Similar but not exact name",
			teams: []models.Team{
				{BaseModel: models.BaseModel{Name: "team-developer-portal"}},
				{BaseModel: models.BaseModel{Name: "team-technical"}},
				{BaseModel: models.BaseModel{Name: technicalTeamName}},
			},
			expectedCount: 2,
			expectedNames: []string{"team-developer-portal", "team-technical"},
		},
	}

	for _, tc := range testCases {
		suite.T().Run(tc.name, func(t *testing.T) {
			// Simulate the filtering logic
			filteredTeams := make([]models.Team, 0, len(tc.teams))
			for _, team := range tc.teams {
				if team.Name != technicalTeamName {
					filteredTeams = append(filteredTeams, team)
				}
			}

			assert.Len(t, filteredTeams, tc.expectedCount, "Filtered team count should match")

			// Check that the expected names are present
			if tc.expectedCount > 0 {
				for i, expectedName := range tc.expectedNames {
					if i < len(filteredTeams) {
						assert.Equal(t, expectedName, filteredTeams[i].Name, "Team name should match")
					}
				}
			}

			// Verify technical team is never in the results
			for _, team := range filteredTeams {
				assert.NotEqual(t, technicalTeamName, team.Name, "Technical team should never be in results")
			}
		})
	}
}

// GetAllTeams Tests

func (suite *TeamServiceTestSuite) TestGetAllTeams_WithOrgID_Success() {
	orgID := uuid.New()
	groupID := uuid.New()

	org := &models.Organization{
		BaseModel: models.BaseModel{
			ID:   orgID,
			Name: "test-org",
		},
	}

	teams := []models.Team{
		{
			BaseModel: models.BaseModel{
				ID:    uuid.New(),
				Name:  "team-1",
				Title: "Team 1",
			},
			GroupID: groupID,
		},
		{
			BaseModel: models.BaseModel{
				ID:    uuid.New(),
				Name:  "team-2",
				Title: "Team 2",
			},
			GroupID: groupID,
		},
	}

	group := &models.Group{
		BaseModel: models.BaseModel{
			ID:   groupID,
			Name: "engineering",
		},
		OrgID: orgID,
	}

	// Mock expectations
	suite.mockOrgRepo.EXPECT().GetByID(orgID).Return(org, nil)
	suite.mockTeamRepo.EXPECT().GetByOrganizationID(orgID, 20, 0).Return(teams, int64(2), nil)
	suite.mockGroupRepo.EXPECT().GetByID(groupID).Return(group, nil).Times(2)

	// Execute
	result, err := suite.teamService.GetAllTeams(&orgID, 1, 20)

	// Assert
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.Len(suite.T(), result.Teams, 2)
	assert.Equal(suite.T(), int64(2), result.Total)
	assert.Equal(suite.T(), 1, result.Page)
	assert.Equal(suite.T(), 20, result.PageSize)
}

func (suite *TeamServiceTestSuite) TestGetAllTeams_WithOrgID_OrganizationNotFound() {
	orgID := uuid.New()

	// Mock expectations - organization not found
	suite.mockOrgRepo.EXPECT().GetByID(orgID).Return(nil, gorm.ErrRecordNotFound)

	// Execute
	result, err := suite.teamService.GetAllTeams(&orgID, 1, 20)

	// Assert
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), result)
	assert.ErrorIs(suite.T(), err, apperrors.ErrOrganizationNotFound)
}

func (suite *TeamServiceTestSuite) TestGetAllTeams_WithOrgID_OrganizationRepoError() {
	orgID := uuid.New()
	expectedError := errors.New("database connection error")

	// Mock expectations - database error
	suite.mockOrgRepo.EXPECT().GetByID(orgID).Return(nil, expectedError)

	// Execute
	result, err := suite.teamService.GetAllTeams(&orgID, 1, 20)

	// Assert
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), result)
	assert.Contains(suite.T(), err.Error(), "failed to verify organization")
}

func (suite *TeamServiceTestSuite) TestGetAllTeams_WithOrgID_TeamRepoError() {
	orgID := uuid.New()
	expectedError := errors.New("database query failed")

	org := &models.Organization{
		BaseModel: models.BaseModel{
			ID:   orgID,
			Name: "test-org",
		},
	}

	// Mock expectations
	suite.mockOrgRepo.EXPECT().GetByID(orgID).Return(org, nil)
	suite.mockTeamRepo.EXPECT().GetByOrganizationID(orgID, 20, 0).Return(nil, int64(0), expectedError)

	// Execute
	result, err := suite.teamService.GetAllTeams(&orgID, 1, 20)

	// Assert
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), result)
	assert.Contains(suite.T(), err.Error(), "failed to get teams")
}

func (suite *TeamServiceTestSuite) TestGetAllTeams_WithOrgID_FiltersTechnicalTeam() {
	orgID := uuid.New()
	groupID := uuid.New()

	org := &models.Organization{
		BaseModel: models.BaseModel{
			ID:   orgID,
			Name: "test-org",
		},
	}

	teams := []models.Team{
		{
			BaseModel: models.BaseModel{
				ID:    uuid.New(),
				Name:  "team-1",
				Title: "Team 1",
			},
			GroupID: groupID,
		},
		{
			BaseModel: models.BaseModel{
				ID:    uuid.New(),
				Name:  "team-developer-portal-technical",
				Title: "Technical Team",
			},
			GroupID: groupID,
		},
		{
			BaseModel: models.BaseModel{
				ID:    uuid.New(),
				Name:  "team-2",
				Title: "Team 2",
			},
			GroupID: groupID,
		},
	}

	group := &models.Group{
		BaseModel: models.BaseModel{
			ID:   groupID,
			Name: "engineering",
		},
		OrgID: orgID,
	}

	// Mock expectations
	suite.mockOrgRepo.EXPECT().GetByID(orgID).Return(org, nil)
	suite.mockTeamRepo.EXPECT().GetByOrganizationID(orgID, 20, 0).Return(teams, int64(3), nil)
	suite.mockGroupRepo.EXPECT().GetByID(groupID).Return(group, nil).Times(2) // Only for non-technical teams

	// Execute
	result, err := suite.teamService.GetAllTeams(&orgID, 1, 20)

	// Assert
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.Len(suite.T(), result.Teams, 2)          // Technical team filtered out
	assert.Equal(suite.T(), int64(2), result.Total) // Total adjusted
	assert.Equal(suite.T(), "team-1", result.Teams[0].Name)
	assert.Equal(suite.T(), "team-2", result.Teams[1].Name)
}

func (suite *TeamServiceTestSuite) TestGetAllTeams_WithOrgID_PaginationDefaults() {
	orgID := uuid.New()

	org := &models.Organization{
		BaseModel: models.BaseModel{
			ID:   orgID,
			Name: "test-org",
		},
	}

	// Mock expectations - with invalid pagination
	suite.mockOrgRepo.EXPECT().GetByID(orgID).Return(org, nil)
	suite.mockTeamRepo.EXPECT().GetByOrganizationID(orgID, 20, 0).Return([]models.Team{}, int64(0), nil)

	// Execute with invalid page and pageSize
	result, err := suite.teamService.GetAllTeams(&orgID, 0, 0)

	// Assert - defaults applied
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.Equal(suite.T(), 1, result.Page)      // Default page
	assert.Equal(suite.T(), 20, result.PageSize) // Default pageSize
}

func (suite *TeamServiceTestSuite) TestGetAllTeams_WithOrgID_PaginationMaxLimit() {
	orgID := uuid.New()

	org := &models.Organization{
		BaseModel: models.BaseModel{
			ID:   orgID,
			Name: "test-org",
		},
	}

	// Mock expectations - with pageSize > 100
	suite.mockOrgRepo.EXPECT().GetByID(orgID).Return(org, nil)
	suite.mockTeamRepo.EXPECT().GetByOrganizationID(orgID, 20, 0).Return([]models.Team{}, int64(0), nil)

	// Execute with pageSize > 100
	result, err := suite.teamService.GetAllTeams(&orgID, 1, 150)

	// Assert - capped at 20 (default)
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.Equal(suite.T(), 20, result.PageSize)
}

func (suite *TeamServiceTestSuite) TestGetAllTeams_WithOrgID_ToResponseError() {
	orgID := uuid.New()
	groupID := uuid.New()

	org := &models.Organization{
		BaseModel: models.BaseModel{
			ID:   orgID,
			Name: "test-org",
		},
	}

	teams := []models.Team{
		{
			BaseModel: models.BaseModel{
				ID:    uuid.New(),
				Name:  "team-1",
				Title: "Team 1",
			},
			GroupID: groupID,
		},
	}

	// Mock expectations - group lookup fails
	suite.mockOrgRepo.EXPECT().GetByID(orgID).Return(org, nil)
	suite.mockTeamRepo.EXPECT().GetByOrganizationID(orgID, 20, 0).Return(teams, int64(1), nil)
	suite.mockGroupRepo.EXPECT().GetByID(groupID).Return(nil, errors.New("group not found"))

	// Execute
	result, err := suite.teamService.GetAllTeams(&orgID, 1, 20)

	// Assert
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), result)
	assert.Contains(suite.T(), err.Error(), "failed to convert team to response")
}

func (suite *TeamServiceTestSuite) TestGetAllTeams_WithoutOrgID_Success() {
	groupID := uuid.New()
	orgID := uuid.New()

	teams := []models.Team{
		{
			BaseModel: models.BaseModel{
				ID:    uuid.New(),
				Name:  "team-1",
				Title: "Team 1",
			},
			GroupID: groupID,
		},
		{
			BaseModel: models.BaseModel{
				ID:    uuid.New(),
				Name:  "team-2",
				Title: "Team 2",
			},
			GroupID: groupID,
		},
	}

	group := &models.Group{
		BaseModel: models.BaseModel{
			ID:   groupID,
			Name: "engineering",
		},
		OrgID: orgID,
	}

	// Mock expectations
	suite.mockTeamRepo.EXPECT().GetAll().Return(teams, nil)
	suite.mockGroupRepo.EXPECT().GetByID(groupID).Return(group, nil).Times(2)

	// Execute
	result, err := suite.teamService.GetAllTeams(nil, 1, 20)

	// Assert
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.Len(suite.T(), result.Teams, 2)
	assert.Equal(suite.T(), int64(2), result.Total)
	assert.Equal(suite.T(), 1, result.Page)
	assert.Equal(suite.T(), 2, result.PageSize) // PageSize = number of teams
}

func (suite *TeamServiceTestSuite) TestGetAllTeams_WithoutOrgID_RepoError() {
	// Mock expectations
	suite.mockTeamRepo.EXPECT().GetAll().Return(nil, errors.New("database connection failed"))

	// Execute
	result, err := suite.teamService.GetAllTeams(nil, 1, 20)

	// Assert
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), result)
	assert.Contains(suite.T(), err.Error(), "failed to get all teams")
}

func (suite *TeamServiceTestSuite) TestGetAllTeams_WithoutOrgID_FiltersTechnicalTeam() {
	groupID := uuid.New()
	orgID := uuid.New()

	teams := []models.Team{
		{
			BaseModel: models.BaseModel{
				ID:    uuid.New(),
				Name:  "team-1",
				Title: "Team 1",
			},
			GroupID: groupID,
		},
		{
			BaseModel: models.BaseModel{
				ID:    uuid.New(),
				Name:  "team-developer-portal-technical",
				Title: "Technical Team",
			},
			GroupID: groupID,
		},
	}

	group := &models.Group{
		BaseModel: models.BaseModel{
			ID:   groupID,
			Name: "engineering",
		},
		OrgID: orgID,
	}

	// Mock expectations
	suite.mockTeamRepo.EXPECT().GetAll().Return(teams, nil)
	suite.mockGroupRepo.EXPECT().GetByID(groupID).Return(group, nil).Times(1) // Only for non-technical team

	// Execute
	result, err := suite.teamService.GetAllTeams(nil, 1, 20)

	// Assert
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.Len(suite.T(), result.Teams, 1) // Technical team filtered out
	assert.Equal(suite.T(), int64(1), result.Total)
	assert.Equal(suite.T(), "team-1", result.Teams[0].Name)
	assert.NotEqual(suite.T(), "team-developer-portal-technical", result.Teams[0].Name)
}

func (suite *TeamServiceTestSuite) TestGetAllTeams_WithoutOrgID_ToResponseError() {
	groupID := uuid.New()

	teams := []models.Team{
		{
			BaseModel: models.BaseModel{
				ID:    uuid.New(),
				Name:  "team-1",
				Title: "Team 1",
			},
			GroupID: groupID,
		},
	}

	// Mock expectations - group lookup fails
	suite.mockTeamRepo.EXPECT().GetAll().Return(teams, nil)
	suite.mockGroupRepo.EXPECT().GetByID(groupID).Return(nil, errors.New("group not found"))

	// Execute
	result, err := suite.teamService.GetAllTeams(nil, 1, 20)

	// Assert
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), result)
	assert.Contains(suite.T(), err.Error(), "failed to convert team to response")
}

// GetTeamComponentsByID Tests

func (suite *TeamServiceTestSuite) TestGetTeamComponentsByID_Success() {
	teamID := uuid.New()

	team := &models.Team{
		BaseModel: models.BaseModel{
			ID:    teamID,
			Name:  "backend-team",
			Title: "Backend Team",
		},
	}

	components := []models.Component{
		{
			BaseModel: models.BaseModel{
				ID:    uuid.New(),
				Name:  "api-service",
				Title: "API Service",
			},
		},
		{
			BaseModel: models.BaseModel{
				ID:    uuid.New(),
				Name:  "auth-service",
				Title: "Auth Service",
			},
		},
	}

	// Mock expectations
	suite.mockTeamRepo.EXPECT().GetByID(teamID).Return(team, nil)
	suite.mockComponentRepo.EXPECT().GetComponentsByTeamID(teamID, 100, 0).Return(components, int64(2), nil)

	// Execute
	result, total, err := suite.teamService.GetTeamComponentsByID(teamID, 1, 100)

	// Assert
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.Len(suite.T(), result, 2)
	assert.Equal(suite.T(), int64(2), total)
	assert.Equal(suite.T(), "api-service", result[0].Name)
	assert.Equal(suite.T(), "auth-service", result[1].Name)
}

func (suite *TeamServiceTestSuite) TestGetTeamComponentsByID_TeamNotFound() {
	teamID := uuid.New()

	// Mock expectations - team not found
	suite.mockTeamRepo.EXPECT().GetByID(teamID).Return(nil, gorm.ErrRecordNotFound)

	// Execute
	result, total, err := suite.teamService.GetTeamComponentsByID(teamID, 1, 100)

	// Assert
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), result)
	assert.Equal(suite.T(), int64(0), total)
	assert.ErrorIs(suite.T(), err, apperrors.ErrTeamNotFound)
}

func (suite *TeamServiceTestSuite) TestGetTeamComponentsByID_TeamRepoError() {
	teamID := uuid.New()
	expectedError := errors.New("database connection error")

	// Mock expectations - database error
	suite.mockTeamRepo.EXPECT().GetByID(teamID).Return(nil, expectedError)

	// Execute
	result, total, err := suite.teamService.GetTeamComponentsByID(teamID, 1, 100)

	// Assert
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), result)
	assert.Equal(suite.T(), int64(0), total)
	assert.Contains(suite.T(), err.Error(), "failed to get team")
}

func (suite *TeamServiceTestSuite) TestGetTeamComponentsByID_ComponentRepoError() {
	teamID := uuid.New()
	expectedError := errors.New("database query failed")

	team := &models.Team{
		BaseModel: models.BaseModel{
			ID:   teamID,
			Name: "backend-team",
		},
	}

	// Mock expectations
	suite.mockTeamRepo.EXPECT().GetByID(teamID).Return(team, nil)
	suite.mockComponentRepo.EXPECT().GetComponentsByTeamID(teamID, 100, 0).Return(nil, int64(0), expectedError)

	// Execute
	result, total, err := suite.teamService.GetTeamComponentsByID(teamID, 1, 100)

	// Assert
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), result)
	assert.Equal(suite.T(), int64(0), total)
	assert.Contains(suite.T(), err.Error(), "failed to get components by team")
}

func (suite *TeamServiceTestSuite) TestGetTeamComponentsByID_EmptyComponents() {
	teamID := uuid.New()

	team := &models.Team{
		BaseModel: models.BaseModel{
			ID:   teamID,
			Name: "backend-team",
		},
	}

	// Mock expectations - no components
	suite.mockTeamRepo.EXPECT().GetByID(teamID).Return(team, nil)
	suite.mockComponentRepo.EXPECT().GetComponentsByTeamID(teamID, 100, 0).Return([]models.Component{}, int64(0), nil)

	// Execute
	result, total, err := suite.teamService.GetTeamComponentsByID(teamID, 1, 100)

	// Assert
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.Len(suite.T(), result, 0)
	assert.Equal(suite.T(), int64(0), total)
}

func (suite *TeamServiceTestSuite) TestGetTeamComponentsByID_PaginationDefaults() {
	teamID := uuid.New()

	team := &models.Team{
		BaseModel: models.BaseModel{
			ID:   teamID,
			Name: "backend-team",
		},
	}

	components := []models.Component{
		{
			BaseModel: models.BaseModel{
				ID:   uuid.New(),
				Name: "component-1",
			},
		},
	}

	// Mock expectations - with invalid pagination (page < 1)
	suite.mockTeamRepo.EXPECT().GetByID(teamID).Return(team, nil)
	suite.mockComponentRepo.EXPECT().GetComponentsByTeamID(teamID, 100, 0).Return(components, int64(1), nil)

	// Execute with invalid page
	result, total, err := suite.teamService.GetTeamComponentsByID(teamID, 0, 0)

	// Assert - defaults applied (page=1, pageSize=100)
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.Len(suite.T(), result, 1)
	assert.Equal(suite.T(), int64(1), total)
}

func (suite *TeamServiceTestSuite) TestGetTeamComponentsByID_PaginationMaxLimit() {
	teamID := uuid.New()

	team := &models.Team{
		BaseModel: models.BaseModel{
			ID:   teamID,
			Name: "backend-team",
		},
	}

	components := []models.Component{
		{
			BaseModel: models.BaseModel{
				ID:   uuid.New(),
				Name: "component-1",
			},
		},
	}

	// Mock expectations - with pageSize > 100
	suite.mockTeamRepo.EXPECT().GetByID(teamID).Return(team, nil)
	suite.mockComponentRepo.EXPECT().GetComponentsByTeamID(teamID, 100, 0).Return(components, int64(1), nil)

	// Execute with pageSize > 100
	result, total, err := suite.teamService.GetTeamComponentsByID(teamID, 1, 150)

	// Assert - capped at 100
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.Len(suite.T(), result, 1)
	assert.Equal(suite.T(), int64(1), total)
}

// GetBySimpleName Tests

func (suite *TeamServiceTestSuite) TestGetBySimpleName_Success() {
	teamName := "backend-team"
	teamID := uuid.New()
	groupID := uuid.New()
	orgID := uuid.New()

	team := &models.Team{
		BaseModel: models.BaseModel{
			ID:    teamID,
			Name:  teamName,
			Title: "Backend Team",
		},
		GroupID: groupID,
		Owner:   "I12345",
		Email:   "backend@example.com",
	}

	group := &models.Group{
		BaseModel: models.BaseModel{
			ID:   groupID,
			Name: "engineering",
		},
		OrgID: orgID,
	}

	members := []models.User{
		{
			BaseModel: models.BaseModel{
				ID: uuid.New(),
			},
			UserID:     "I12345",
			FirstName:  "John",
			LastName:   "Doe",
			Email:      "john@example.com",
			TeamID:     &teamID,
			TeamDomain: models.TeamDomainDeveloper,
			TeamRole:   models.TeamRoleMember,
		},
	}

	links := []models.Link{
		{
			BaseModel: models.BaseModel{
				ID:   uuid.New(),
				Name: "github-repo",
			},
			URL:        "https://github.com/team/repo",
			Owner:      teamID,
			CategoryID: uuid.New(),
		},
	}

	// Mock expectations
	suite.mockTeamRepo.EXPECT().GetByNameGlobal(teamName).Return(team, nil)
	suite.mockUserRepo.EXPECT().GetByTeamID(teamID, 1000, 0).Return(members, int64(1), nil)
	suite.mockGroupRepo.EXPECT().GetByID(groupID).Return(group, nil)
	suite.mockLinkRepo.EXPECT().GetByOwner(teamID).Return(links, nil)

	// Execute
	result, err := suite.teamService.GetBySimpleName(teamName)

	// Assert
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.Equal(suite.T(), teamID, result.ID)
	assert.Equal(suite.T(), teamName, result.Name)
	assert.Len(suite.T(), result.Members, 1)
	assert.Equal(suite.T(), "John", result.Members[0].FirstName)
	assert.Len(suite.T(), result.Links, 1)
}

func (suite *TeamServiceTestSuite) TestGetBySimpleName_EmptyName() {
	// Execute with empty name
	result, err := suite.teamService.GetBySimpleName("")

	// Assert
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), result)
	assert.Equal(suite.T(), err.Error(), apperrors.NewMissingQueryParam("team_name").Error())
}

func (suite *TeamServiceTestSuite) TestGetBySimpleName_TeamNotFound() {
	teamName := "non-existent-team"

	// Mock expectations - team not found
	suite.mockTeamRepo.EXPECT().GetByNameGlobal(teamName).Return(nil, gorm.ErrRecordNotFound)

	// Execute
	result, err := suite.teamService.GetBySimpleName(teamName)

	// Assert
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), result)
	assert.ErrorIs(suite.T(), err, apperrors.ErrTeamNotFound)
}

func (suite *TeamServiceTestSuite) TestGetBySimpleName_TeamRepoError() {
	teamName := "backend-team"

	// Mock expectations - database error
	suite.mockTeamRepo.EXPECT().GetByNameGlobal(teamName).Return(nil, errors.New("database connection error"))

	// Execute
	result, err := suite.teamService.GetBySimpleName(teamName)

	// Assert
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), result)
	assert.Contains(suite.T(), err.Error(), "failed to get team by name")
}

func (suite *TeamServiceTestSuite) TestGetBySimpleName_UserRepoError() {
	teamName := "backend-team"
	teamID := uuid.New()
	expectedError := errors.New("database query failed")

	team := &models.Team{
		BaseModel: models.BaseModel{
			ID:   teamID,
			Name: teamName,
		},
		GroupID: uuid.New(),
	}

	// Mock expectations
	suite.mockTeamRepo.EXPECT().GetByNameGlobal(teamName).Return(team, nil)
	suite.mockUserRepo.EXPECT().GetByTeamID(teamID, 1000, 0).Return(nil, int64(0), expectedError)

	// Execute
	result, err := suite.teamService.GetBySimpleName(teamName)

	// Assert
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), result)
	assert.Contains(suite.T(), err.Error(), "failed to get team members")
}

func (suite *TeamServiceTestSuite) TestGetBySimpleName_GroupRepoError() {
	teamName := "backend-team"
	teamID := uuid.New()
	groupID := uuid.New()

	team := &models.Team{
		BaseModel: models.BaseModel{
			ID:   teamID,
			Name: teamName,
		},
		GroupID: groupID,
	}

	members := []models.User{}

	// Mock expectations - group lookup fails
	suite.mockTeamRepo.EXPECT().GetByNameGlobal(teamName).Return(team, nil)
	suite.mockUserRepo.EXPECT().GetByTeamID(teamID, 1000, 0).Return(members, int64(0), nil)
	suite.mockGroupRepo.EXPECT().GetByID(groupID).Return(nil, errors.New("group not found"))

	// Execute
	result, err := suite.teamService.GetBySimpleName(teamName)

	// Assert
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), result)
	assert.Contains(suite.T(), err.Error(), "failed to convert team to response")
}

func (suite *TeamServiceTestSuite) TestGetBySimpleName_NoMembers() {
	teamName := "backend-team"
	teamID := uuid.New()
	groupID := uuid.New()
	orgID := uuid.New()

	team := &models.Team{
		BaseModel: models.BaseModel{
			ID:   teamID,
			Name: teamName,
		},
		GroupID: groupID,
	}

	group := &models.Group{
		BaseModel: models.BaseModel{
			ID:   groupID,
			Name: "engineering",
		},
		OrgID: orgID,
	}

	// Mock expectations - no members
	suite.mockTeamRepo.EXPECT().GetByNameGlobal(teamName).Return(team, nil)
	suite.mockUserRepo.EXPECT().GetByTeamID(teamID, 1000, 0).Return([]models.User{}, int64(0), nil)
	suite.mockGroupRepo.EXPECT().GetByID(groupID).Return(group, nil)
	suite.mockLinkRepo.EXPECT().GetByOwner(teamID).Return([]models.Link{}, nil)

	// Execute
	result, err := suite.teamService.GetBySimpleName(teamName)

	// Assert
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.Len(suite.T(), result.Members, 0)
	assert.Len(suite.T(), result.Links, 0)
}

func (suite *TeamServiceTestSuite) TestGetBySimpleName_LinkRepoError() {
	teamName := "backend-team"
	teamID := uuid.New()
	groupID := uuid.New()
	orgID := uuid.New()

	team := &models.Team{
		BaseModel: models.BaseModel{
			ID:   teamID,
			Name: teamName,
		},
		GroupID: groupID,
	}

	group := &models.Group{
		BaseModel: models.BaseModel{
			ID:   groupID,
			Name: "engineering",
		},
		OrgID: orgID,
	}

	members := []models.User{
		{
			BaseModel: models.BaseModel{
				ID: uuid.New(),
			},
			UserID:    "I12345",
			FirstName: "John",
			LastName:  "Doe",
			TeamID:    &teamID,
		},
	}

	// Mock expectations - link repo error (should be handled gracefully)
	suite.mockTeamRepo.EXPECT().GetByNameGlobal(teamName).Return(team, nil)
	suite.mockUserRepo.EXPECT().GetByTeamID(teamID, 1000, 0).Return(members, int64(1), nil)
	suite.mockGroupRepo.EXPECT().GetByID(groupID).Return(group, nil)
	suite.mockLinkRepo.EXPECT().GetByOwner(teamID).Return(nil, errors.New("link fetch error"))

	// Execute
	result, err := suite.teamService.GetBySimpleName(teamName)

	// Assert - should succeed with empty links (error is logged internally)
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.Len(suite.T(), result.Members, 1)
	assert.Len(suite.T(), result.Links, 0)
}

func (suite *TeamServiceTestSuite) TestGetBySimpleName_MultipleMembers() {
	teamName := "backend-team"
	teamID := uuid.New()
	groupID := uuid.New()
	orgID := uuid.New()

	team := &models.Team{
		BaseModel: models.BaseModel{
			ID:   teamID,
			Name: teamName,
		},
		GroupID: groupID,
	}

	group := &models.Group{
		BaseModel: models.BaseModel{
			ID:   groupID,
			Name: "engineering",
		},
		OrgID: orgID,
	}

	members := []models.User{
		{
			BaseModel: models.BaseModel{
				ID: uuid.New(),
			},
			UserID:     "I12345",
			FirstName:  "John",
			LastName:   "Doe",
			Email:      "john@example.com",
			TeamID:     &teamID,
			TeamDomain: models.TeamDomainDeveloper,
			TeamRole:   models.TeamRoleMember,
		},
		{
			BaseModel: models.BaseModel{
				ID: uuid.New(),
			},
			UserID:     "I67890",
			FirstName:  "Jane",
			LastName:   "Smith",
			Email:      "jane@example.com",
			TeamID:     &teamID,
			TeamDomain: models.TeamDomainDeveloper,
			TeamRole:   models.TeamRoleManager,
		},
	}

	// Mock expectations
	suite.mockTeamRepo.EXPECT().GetByNameGlobal(teamName).Return(team, nil)
	suite.mockUserRepo.EXPECT().GetByTeamID(teamID, 1000, 0).Return(members, int64(2), nil)
	suite.mockGroupRepo.EXPECT().GetByID(groupID).Return(group, nil)
	suite.mockLinkRepo.EXPECT().GetByOwner(teamID).Return([]models.Link{}, nil)

	// Execute
	result, err := suite.teamService.GetBySimpleName(teamName)

	// Assert
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.Len(suite.T(), result.Members, 2)
	assert.Equal(suite.T(), "John", result.Members[0].FirstName)
	assert.Equal(suite.T(), "Jane", result.Members[1].FirstName)
	assert.Equal(suite.T(), "member", result.Members[0].TeamRole)
	assert.Equal(suite.T(), "manager", result.Members[1].TeamRole)
}

func (suite *TeamServiceTestSuite) TestGetBySimpleName_MultipleLinks() {
	teamName := "backend-team"
	teamID := uuid.New()
	groupID := uuid.New()
	orgID := uuid.New()

	team := &models.Team{
		BaseModel: models.BaseModel{
			ID:   teamID,
			Name: teamName,
		},
		GroupID: groupID,
	}

	group := &models.Group{
		BaseModel: models.BaseModel{
			ID:   groupID,
			Name: "engineering",
		},
		OrgID: orgID,
	}

	links := []models.Link{
		{
			BaseModel: models.BaseModel{
				ID:   uuid.New(),
				Name: "github-repo",
			},
			URL:        "https://github.com/team/repo",
			Owner:      teamID,
			CategoryID: uuid.New(),
		},
		{
			BaseModel: models.BaseModel{
				ID:   uuid.New(),
				Name: "jira-board",
			},
			URL:        "https://jira.example.com/board/123",
			Owner:      teamID,
			CategoryID: uuid.New(),
		},
	}

	// Mock expectations
	suite.mockTeamRepo.EXPECT().GetByNameGlobal(teamName).Return(team, nil)
	suite.mockUserRepo.EXPECT().GetByTeamID(teamID, 1000, 0).Return([]models.User{}, int64(0), nil)
	suite.mockGroupRepo.EXPECT().GetByID(groupID).Return(group, nil)
	suite.mockLinkRepo.EXPECT().GetByOwner(teamID).Return(links, nil)

	// Execute
	result, err := suite.teamService.GetBySimpleName(teamName)

	// Assert
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.Len(suite.T(), result.Links, 2)
	assert.Equal(suite.T(), "github-repo", result.Links[0].Name)
	assert.Equal(suite.T(), "jira-board", result.Links[1].Name)
}

// GetBySimpleNameWithViewer Tests

func (suite *TeamServiceTestSuite) TestGetBySimpleNameWithViewer_Success_WithFavorites() {
	teamName := "backend-team"
	viewerName := "john.doe"
	teamID := uuid.New()
	groupID := uuid.New()
	orgID := uuid.New()
	linkID1 := uuid.New()
	linkID2 := uuid.New()

	team := &models.Team{
		BaseModel: models.BaseModel{
			ID:   teamID,
			Name: teamName,
		},
		GroupID: groupID,
	}

	group := &models.Group{
		BaseModel: models.BaseModel{
			ID:   groupID,
			Name: "engineering",
		},
		OrgID: orgID,
	}

	links := []models.Link{
		{
			BaseModel: models.BaseModel{
				ID:   linkID1,
				Name: "github-repo",
			},
			URL:        "https://github.com/team/repo",
			Owner:      teamID,
			CategoryID: uuid.New(),
		},
		{
			BaseModel: models.BaseModel{
				ID:   linkID2,
				Name: "jira-board",
			},
			URL:        "https://jira.example.com/board/123",
			Owner:      teamID,
			CategoryID: uuid.New(),
		},
	}

	// Viewer with favorites metadata
	viewerMetadata := json.RawMessage(fmt.Sprintf(`{"favorites": ["%s"]}`, linkID1.String()))
	viewer := &models.User{
		BaseModel: models.BaseModel{
			ID: uuid.New(),
		},
		UserID:   viewerName,
		Metadata: viewerMetadata,
	}

	// Mock expectations
	suite.mockTeamRepo.EXPECT().GetByNameGlobal(teamName).Return(team, nil)
	suite.mockUserRepo.EXPECT().GetByTeamID(teamID, 1000, 0).Return([]models.User{}, int64(0), nil)
	suite.mockGroupRepo.EXPECT().GetByID(groupID).Return(group, nil)
	suite.mockLinkRepo.EXPECT().GetByOwner(teamID).Return(links, nil)
	suite.mockUserRepo.EXPECT().GetByName(viewerName).Return(viewer, nil)

	// Execute
	result, err := suite.teamService.GetBySimpleNameWithViewer(teamName, viewerName)

	// Assert
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.Len(suite.T(), result.Links, 2)
	assert.True(suite.T(), result.Links[0].Favorite, "First link should be marked as favorite")
	assert.False(suite.T(), result.Links[1].Favorite, "Second link should not be marked as favorite")
}

func (suite *TeamServiceTestSuite) TestGetBySimpleNameWithViewer_EmptyViewerName() {
	teamName := "backend-team"
	teamID := uuid.New()
	groupID := uuid.New()
	orgID := uuid.New()

	team := &models.Team{
		BaseModel: models.BaseModel{
			ID:   teamID,
			Name: teamName,
		},
		GroupID: groupID,
	}

	group := &models.Group{
		BaseModel: models.BaseModel{
			ID:   groupID,
			Name: "engineering",
		},
		OrgID: orgID,
	}

	links := []models.Link{
		{
			BaseModel: models.BaseModel{
				ID:   uuid.New(),
				Name: "github-repo",
			},
			URL:        "https://github.com/team/repo",
			Owner:      teamID,
			CategoryID: uuid.New(),
		},
	}

	// Mock expectations - no viewer lookup should happen
	suite.mockTeamRepo.EXPECT().GetByNameGlobal(teamName).Return(team, nil)
	suite.mockUserRepo.EXPECT().GetByTeamID(teamID, 1000, 0).Return([]models.User{}, int64(0), nil)
	suite.mockGroupRepo.EXPECT().GetByID(groupID).Return(group, nil)
	suite.mockLinkRepo.EXPECT().GetByOwner(teamID).Return(links, nil)
	// No GetByName call expected

	// Execute with empty viewer name
	result, err := suite.teamService.GetBySimpleNameWithViewer(teamName, "")

	// Assert
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.Len(suite.T(), result.Links, 1)
	assert.False(suite.T(), result.Links[0].Favorite, "Link should not be marked as favorite")
}

func (suite *TeamServiceTestSuite) TestGetBySimpleNameWithViewer_ViewerNotFound() {
	teamName := "backend-team"
	viewerName := "non-existent-user"
	teamID := uuid.New()
	groupID := uuid.New()
	orgID := uuid.New()

	team := &models.Team{
		BaseModel: models.BaseModel{
			ID:   teamID,
			Name: teamName,
		},
		GroupID: groupID,
	}

	group := &models.Group{
		BaseModel: models.BaseModel{
			ID:   groupID,
			Name: "engineering",
		},
		OrgID: orgID,
	}

	links := []models.Link{
		{
			BaseModel: models.BaseModel{
				ID:   uuid.New(),
				Name: "github-repo",
			},
			URL:        "https://github.com/team/repo",
			Owner:      teamID,
			CategoryID: uuid.New(),
		},
	}

	// Mock expectations
	suite.mockTeamRepo.EXPECT().GetByNameGlobal(teamName).Return(team, nil)
	suite.mockUserRepo.EXPECT().GetByTeamID(teamID, 1000, 0).Return([]models.User{}, int64(0), nil)
	suite.mockGroupRepo.EXPECT().GetByID(groupID).Return(group, nil)
	suite.mockLinkRepo.EXPECT().GetByOwner(teamID).Return(links, nil)
	suite.mockUserRepo.EXPECT().GetByName(viewerName).Return(nil, gorm.ErrRecordNotFound)

	// Execute
	result, err := suite.teamService.GetBySimpleNameWithViewer(teamName, viewerName)

	// Assert - should succeed, just without favorites marked
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.Len(suite.T(), result.Links, 1)
	assert.False(suite.T(), result.Links[0].Favorite)
}

func (suite *TeamServiceTestSuite) TestGetBySimpleNameWithViewer_ViewerWithNoMetadata() {
	teamName := "backend-team"
	viewerName := "john.doe"
	teamID := uuid.New()
	groupID := uuid.New()
	orgID := uuid.New()

	team := &models.Team{
		BaseModel: models.BaseModel{
			ID:   teamID,
			Name: teamName,
		},
		GroupID: groupID,
	}

	group := &models.Group{
		BaseModel: models.BaseModel{
			ID:   groupID,
			Name: "engineering",
		},
		OrgID: orgID,
	}

	links := []models.Link{
		{
			BaseModel: models.BaseModel{
				ID:   uuid.New(),
				Name: "github-repo",
			},
			URL:        "https://github.com/team/repo",
			Owner:      teamID,
			CategoryID: uuid.New(),
		},
	}

	// Viewer with no metadata
	viewer := &models.User{
		BaseModel: models.BaseModel{
			ID: uuid.New(),
		},
		UserID:   viewerName,
		Metadata: nil,
	}

	// Mock expectations
	suite.mockTeamRepo.EXPECT().GetByNameGlobal(teamName).Return(team, nil)
	suite.mockUserRepo.EXPECT().GetByTeamID(teamID, 1000, 0).Return([]models.User{}, int64(0), nil)
	suite.mockGroupRepo.EXPECT().GetByID(groupID).Return(group, nil)
	suite.mockLinkRepo.EXPECT().GetByOwner(teamID).Return(links, nil)
	suite.mockUserRepo.EXPECT().GetByName(viewerName).Return(viewer, nil)

	// Execute
	result, err := suite.teamService.GetBySimpleNameWithViewer(teamName, viewerName)

	// Assert
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.Len(suite.T(), result.Links, 1)
	assert.False(suite.T(), result.Links[0].Favorite)
}

func (suite *TeamServiceTestSuite) TestGetBySimpleNameWithViewer_ViewerWithEmptyFavorites() {
	teamName := "backend-team"
	viewerName := "john.doe"
	teamID := uuid.New()
	groupID := uuid.New()
	orgID := uuid.New()

	team := &models.Team{
		BaseModel: models.BaseModel{
			ID:   teamID,
			Name: teamName,
		},
		GroupID: groupID,
	}

	group := &models.Group{
		BaseModel: models.BaseModel{
			ID:   groupID,
			Name: "engineering",
		},
		OrgID: orgID,
	}

	links := []models.Link{
		{
			BaseModel: models.BaseModel{
				ID:   uuid.New(),
				Name: "github-repo",
			},
			URL:        "https://github.com/team/repo",
			Owner:      teamID,
			CategoryID: uuid.New(),
		},
	}

	// Viewer with empty favorites array
	viewerMetadata := json.RawMessage(`{"favorites": []}`)
	viewer := &models.User{
		BaseModel: models.BaseModel{
			ID: uuid.New(),
		},
		UserID:   viewerName,
		Metadata: viewerMetadata,
	}

	// Mock expectations
	suite.mockTeamRepo.EXPECT().GetByNameGlobal(teamName).Return(team, nil)
	suite.mockUserRepo.EXPECT().GetByTeamID(teamID, 1000, 0).Return([]models.User{}, int64(0), nil)
	suite.mockGroupRepo.EXPECT().GetByID(groupID).Return(group, nil)
	suite.mockLinkRepo.EXPECT().GetByOwner(teamID).Return(links, nil)
	suite.mockUserRepo.EXPECT().GetByName(viewerName).Return(viewer, nil)

	// Execute
	result, err := suite.teamService.GetBySimpleNameWithViewer(teamName, viewerName)

	// Assert
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.Len(suite.T(), result.Links, 1)
	assert.False(suite.T(), result.Links[0].Favorite)
}

func (suite *TeamServiceTestSuite) TestGetBySimpleNameWithViewer_MultipleFavorites() {
	teamName := "backend-team"
	viewerName := "john.doe"
	teamID := uuid.New()
	groupID := uuid.New()
	orgID := uuid.New()
	linkID1 := uuid.New()
	linkID2 := uuid.New()
	linkID3 := uuid.New()

	team := &models.Team{
		BaseModel: models.BaseModel{
			ID:   teamID,
			Name: teamName,
		},
		GroupID: groupID,
	}

	group := &models.Group{
		BaseModel: models.BaseModel{
			ID:   groupID,
			Name: "engineering",
		},
		OrgID: orgID,
	}

	links := []models.Link{
		{
			BaseModel: models.BaseModel{
				ID:   linkID1,
				Name: "github-repo",
			},
			URL:        "https://github.com/team/repo",
			Owner:      teamID,
			CategoryID: uuid.New(),
		},
		{
			BaseModel: models.BaseModel{
				ID:   linkID2,
				Name: "jira-board",
			},
			URL:        "https://jira.example.com/board/123",
			Owner:      teamID,
			CategoryID: uuid.New(),
		},
		{
			BaseModel: models.BaseModel{
				ID:   linkID3,
				Name: "confluence",
			},
			URL:        "https://confluence.example.com",
			Owner:      teamID,
			CategoryID: uuid.New(),
		},
	}

	// Viewer with multiple favorites
	viewerMetadata := json.RawMessage(fmt.Sprintf(`{"favorites": ["%s", "%s"]}`, linkID1.String(), linkID3.String()))
	viewer := &models.User{
		BaseModel: models.BaseModel{
			ID: uuid.New(),
		},
		UserID:   viewerName,
		Metadata: viewerMetadata,
	}

	// Mock expectations
	suite.mockTeamRepo.EXPECT().GetByNameGlobal(teamName).Return(team, nil)
	suite.mockUserRepo.EXPECT().GetByTeamID(teamID, 1000, 0).Return([]models.User{}, int64(0), nil)
	suite.mockGroupRepo.EXPECT().GetByID(groupID).Return(group, nil)
	suite.mockLinkRepo.EXPECT().GetByOwner(teamID).Return(links, nil)
	suite.mockUserRepo.EXPECT().GetByName(viewerName).Return(viewer, nil)

	// Execute
	result, err := suite.teamService.GetBySimpleNameWithViewer(teamName, viewerName)

	// Assert
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.Len(suite.T(), result.Links, 3)
	assert.True(suite.T(), result.Links[0].Favorite, "First link should be favorite")
	assert.False(suite.T(), result.Links[1].Favorite, "Second link should not be favorite")
	assert.True(suite.T(), result.Links[2].Favorite, "Third link should be favorite")
}

func (suite *TeamServiceTestSuite) TestGetBySimpleNameWithViewer_InvalidFavoritesFormat() {
	teamName := "backend-team"
	viewerName := "john.doe"
	teamID := uuid.New()
	groupID := uuid.New()
	orgID := uuid.New()

	team := &models.Team{
		BaseModel: models.BaseModel{
			ID:   teamID,
			Name: teamName,
		},
		GroupID: groupID,
	}

	group := &models.Group{
		BaseModel: models.BaseModel{
			ID:   groupID,
			Name: "engineering",
		},
		OrgID: orgID,
	}

	links := []models.Link{
		{
			BaseModel: models.BaseModel{
				ID:   uuid.New(),
				Name: "github-repo",
			},
			URL:        "https://github.com/team/repo",
			Owner:      teamID,
			CategoryID: uuid.New(),
		},
	}

	// Viewer with invalid favorites format (not an array)
	viewerMetadata := json.RawMessage(`{"favorites": "not-an-array"}`)
	viewer := &models.User{
		BaseModel: models.BaseModel{
			ID: uuid.New(),
		},
		UserID:   viewerName,
		Metadata: viewerMetadata,
	}

	// Mock expectations
	suite.mockTeamRepo.EXPECT().GetByNameGlobal(teamName).Return(team, nil)
	suite.mockUserRepo.EXPECT().GetByTeamID(teamID, 1000, 0).Return([]models.User{}, int64(0), nil)
	suite.mockGroupRepo.EXPECT().GetByID(groupID).Return(group, nil)
	suite.mockLinkRepo.EXPECT().GetByOwner(teamID).Return(links, nil)
	suite.mockUserRepo.EXPECT().GetByName(viewerName).Return(viewer, nil)

	// Execute
	result, err := suite.teamService.GetBySimpleNameWithViewer(teamName, viewerName)

	// Assert - should handle gracefully
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.Len(suite.T(), result.Links, 1)
	assert.False(suite.T(), result.Links[0].Favorite)
}

func (suite *TeamServiceTestSuite) TestGetBySimpleNameWithViewer_InvalidUUIDInFavorites() {
	teamName := "backend-team"
	viewerName := "john.doe"
	teamID := uuid.New()
	groupID := uuid.New()
	orgID := uuid.New()
	linkID := uuid.New()

	team := &models.Team{
		BaseModel: models.BaseModel{
			ID:   teamID,
			Name: teamName,
		},
		GroupID: groupID,
	}

	group := &models.Group{
		BaseModel: models.BaseModel{
			ID:   groupID,
			Name: "engineering",
		},
		OrgID: orgID,
	}

	links := []models.Link{
		{
			BaseModel: models.BaseModel{
				ID:   linkID,
				Name: "github-repo",
			},
			URL:        "https://github.com/team/repo",
			Owner:      teamID,
			CategoryID: uuid.New(),
		},
	}

	// Viewer with invalid UUID in favorites (should be skipped)
	viewerMetadata := json.RawMessage(fmt.Sprintf(`{"favorites": ["not-a-uuid", "%s", "also-invalid"]}`, linkID.String()))
	viewer := &models.User{
		BaseModel: models.BaseModel{
			ID: uuid.New(),
		},
		UserID:   viewerName,
		Metadata: viewerMetadata,
	}

	// Mock expectations
	suite.mockTeamRepo.EXPECT().GetByNameGlobal(teamName).Return(team, nil)
	suite.mockUserRepo.EXPECT().GetByTeamID(teamID, 1000, 0).Return([]models.User{}, int64(0), nil)
	suite.mockGroupRepo.EXPECT().GetByID(groupID).Return(group, nil)
	suite.mockLinkRepo.EXPECT().GetByOwner(teamID).Return(links, nil)
	suite.mockUserRepo.EXPECT().GetByName(viewerName).Return(viewer, nil)

	// Execute
	result, err := suite.teamService.GetBySimpleNameWithViewer(teamName, viewerName)

	// Assert - should only mark valid UUID as favorite
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.Len(suite.T(), result.Links, 1)
	assert.True(suite.T(), result.Links[0].Favorite, "Link with valid UUID should be marked as favorite")
}

func (suite *TeamServiceTestSuite) TestGetBySimpleNameWithViewer_TeamNotFound() {
	teamName := "non-existent-team"
	viewerName := "john.doe"

	// Mock expectations - team not found
	suite.mockTeamRepo.EXPECT().GetByNameGlobal(teamName).Return(nil, gorm.ErrRecordNotFound)

	// Execute
	result, err := suite.teamService.GetBySimpleNameWithViewer(teamName, viewerName)

	// Assert
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), result)
	assert.ErrorIs(suite.T(), err, apperrors.ErrTeamNotFound)
}

func (suite *TeamServiceTestSuite) TestGetBySimpleNameWithViewer_GetBySimpleNameError() {
	teamName := "backend-team"
	viewerName := "john.doe"

	// Mock expectations - database error
	suite.mockTeamRepo.EXPECT().GetByNameGlobal(teamName).Return(nil, errors.New("database connection error"))

	// Execute
	result, err := suite.teamService.GetBySimpleNameWithViewer(teamName, viewerName)

	// Assert
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), result)
	assert.Contains(suite.T(), err.Error(), "failed to get team by name")
}

func (suite *TeamServiceTestSuite) TestGetBySimpleNameWithViewer_NoLinks() {
	teamName := "backend-team"
	viewerName := "john.doe"
	teamID := uuid.New()
	groupID := uuid.New()
	orgID := uuid.New()

	team := &models.Team{
		BaseModel: models.BaseModel{
			ID:   teamID,
			Name: teamName,
		},
		GroupID: groupID,
	}

	group := &models.Group{
		BaseModel: models.BaseModel{
			ID:   groupID,
			Name: "engineering",
		},
		OrgID: orgID,
	}

	// Viewer with favorites
	viewerMetadata := json.RawMessage(fmt.Sprintf(`{"favorites": ["%s"]}`, uuid.New().String()))
	viewer := &models.User{
		BaseModel: models.BaseModel{
			ID: uuid.New(),
		},
		UserID:   viewerName,
		Metadata: viewerMetadata,
	}

	// Mock expectations - no links
	suite.mockTeamRepo.EXPECT().GetByNameGlobal(teamName).Return(team, nil)
	suite.mockUserRepo.EXPECT().GetByTeamID(teamID, 1000, 0).Return([]models.User{}, int64(0), nil)
	suite.mockGroupRepo.EXPECT().GetByID(groupID).Return(group, nil)
	suite.mockLinkRepo.EXPECT().GetByOwner(teamID).Return([]models.Link{}, nil)
	suite.mockUserRepo.EXPECT().GetByName(viewerName).Return(viewer, nil)

	// Execute
	result, err := suite.teamService.GetBySimpleNameWithViewer(teamName, viewerName)

	// Assert
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.Len(suite.T(), result.Links, 0)
}

func (suite *TeamServiceTestSuite) TestGetBySimpleNameWithViewer_InvalidJSON() {
	teamName := "backend-team"
	viewerName := "john.doe"
	teamID := uuid.New()
	groupID := uuid.New()
	orgID := uuid.New()

	team := &models.Team{
		BaseModel: models.BaseModel{
			ID:   teamID,
			Name: teamName,
		},
		GroupID: groupID,
	}

	group := &models.Group{
		BaseModel: models.BaseModel{
			ID:   groupID,
			Name: "engineering",
		},
		OrgID: orgID,
	}

	links := []models.Link{
		{
			BaseModel: models.BaseModel{
				ID:   uuid.New(),
				Name: "github-repo",
			},
			URL:        "https://github.com/team/repo",
			Owner:      teamID,
			CategoryID: uuid.New(),
		},
	}

	// Viewer with invalid JSON metadata
	viewerMetadata := json.RawMessage(`{invalid json}`)
	viewer := &models.User{
		BaseModel: models.BaseModel{
			ID: uuid.New(),
		},
		UserID:   viewerName,
		Metadata: viewerMetadata,
	}

	// Mock expectations
	suite.mockTeamRepo.EXPECT().GetByNameGlobal(teamName).Return(team, nil)
	suite.mockUserRepo.EXPECT().GetByTeamID(teamID, 1000, 0).Return([]models.User{}, int64(0), nil)
	suite.mockGroupRepo.EXPECT().GetByID(groupID).Return(group, nil)
	suite.mockLinkRepo.EXPECT().GetByOwner(teamID).Return(links, nil)
	suite.mockUserRepo.EXPECT().GetByName(viewerName).Return(viewer, nil)

	// Execute
	result, err := suite.teamService.GetBySimpleNameWithViewer(teamName, viewerName)

	// Assert - should handle gracefully
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.Len(suite.T(), result.Links, 1)
	assert.False(suite.T(), result.Links[0].Favorite)
}

// UpdateTeamMetadata Tests

func (suite *TeamServiceTestSuite) TestUpdateTeamMetadata_Success_MergeNewFields() {
	teamID := uuid.New()
	groupID := uuid.New()
	orgID := uuid.New()

	// Existing team with metadata
	existingMetadata := json.RawMessage(`{"tags": ["backend"], "priority": "high"}`)
	team := &models.Team{
		BaseModel: models.BaseModel{
			ID:       teamID,
			Name:     "backend-team",
			Title:    "Backend Team",
			Metadata: existingMetadata,
		},
		GroupID: groupID,
	}

	group := &models.Group{
		BaseModel: models.BaseModel{
			ID:   groupID,
			Name: "engineering",
		},
		OrgID: orgID,
	}

	// New metadata to merge
	newMetadata := json.RawMessage(`{"status": "active", "owner": "john.doe"}`)

	// Mock expectations
	suite.mockTeamRepo.EXPECT().GetByID(teamID).Return(team, nil)
	suite.mockTeamRepo.EXPECT().Update(gomock.Any()).DoAndReturn(func(t *models.Team) error {
		// Verify merged metadata
		var merged map[string]interface{}
		err := json.Unmarshal(t.Metadata, &merged)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), "backend", merged["tags"].([]interface{})[0])
		assert.Equal(suite.T(), "high", merged["priority"])
		assert.Equal(suite.T(), "active", merged["status"])
		assert.Equal(suite.T(), "john.doe", merged["owner"])
		return nil
	})
	suite.mockGroupRepo.EXPECT().GetByID(groupID).Return(group, nil)

	// Execute
	result, err := suite.teamService.UpdateTeamMetadata(teamID, newMetadata)

	// Assert
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.Equal(suite.T(), teamID, result.ID)
}

func (suite *TeamServiceTestSuite) TestUpdateTeamMetadata_Success_UpdateExistingFields() {
	teamID := uuid.New()
	groupID := uuid.New()
	orgID := uuid.New()

	// Existing team with metadata
	existingMetadata := json.RawMessage(`{"tags": ["backend"], "priority": "high", "status": "inactive"}`)
	team := &models.Team{
		BaseModel: models.BaseModel{
			ID:       teamID,
			Name:     "backend-team",
			Title:    "Backend Team",
			Metadata: existingMetadata,
		},
		GroupID: groupID,
	}

	group := &models.Group{
		BaseModel: models.BaseModel{
			ID:   groupID,
			Name: "engineering",
		},
		OrgID: orgID,
	}

	// New metadata to update existing fields
	newMetadata := json.RawMessage(`{"priority": "critical", "status": "active"}`)

	// Mock expectations
	suite.mockTeamRepo.EXPECT().GetByID(teamID).Return(team, nil)
	suite.mockTeamRepo.EXPECT().Update(gomock.Any()).DoAndReturn(func(t *models.Team) error {
		// Verify updated metadata
		var merged map[string]interface{}
		err := json.Unmarshal(t.Metadata, &merged)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), "backend", merged["tags"].([]interface{})[0])
		assert.Equal(suite.T(), "critical", merged["priority"]) // Updated
		assert.Equal(suite.T(), "active", merged["status"])     // Updated
		return nil
	})
	suite.mockGroupRepo.EXPECT().GetByID(groupID).Return(group, nil)

	// Execute
	result, err := suite.teamService.UpdateTeamMetadata(teamID, newMetadata)

	// Assert
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.Equal(suite.T(), teamID, result.ID)
}

func (suite *TeamServiceTestSuite) TestUpdateTeamMetadata_Success_EmptyExistingMetadata() {
	teamID := uuid.New()
	groupID := uuid.New()
	orgID := uuid.New()

	// Team with no existing metadata
	team := &models.Team{
		BaseModel: models.BaseModel{
			ID:       teamID,
			Name:     "backend-team",
			Title:    "Backend Team",
			Metadata: nil,
		},
		GroupID: groupID,
	}

	group := &models.Group{
		BaseModel: models.BaseModel{
			ID:   groupID,
			Name: "engineering",
		},
		OrgID: orgID,
	}

	// New metadata
	newMetadata := json.RawMessage(`{"tags": ["backend"], "priority": "high"}`)

	// Mock expectations
	suite.mockTeamRepo.EXPECT().GetByID(teamID).Return(team, nil)
	suite.mockTeamRepo.EXPECT().Update(gomock.Any()).DoAndReturn(func(t *models.Team) error {
		// Verify new metadata
		var merged map[string]interface{}
		err := json.Unmarshal(t.Metadata, &merged)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), "backend", merged["tags"].([]interface{})[0])
		assert.Equal(suite.T(), "high", merged["priority"])
		return nil
	})
	suite.mockGroupRepo.EXPECT().GetByID(groupID).Return(group, nil)

	// Execute
	result, err := suite.teamService.UpdateTeamMetadata(teamID, newMetadata)

	// Assert
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.Equal(suite.T(), teamID, result.ID)
}

func (suite *TeamServiceTestSuite) TestUpdateTeamMetadata_Success_EmptyJSONExistingMetadata() {
	teamID := uuid.New()
	groupID := uuid.New()
	orgID := uuid.New()

	// Team with empty JSON metadata
	existingMetadata := json.RawMessage(`{}`)
	team := &models.Team{
		BaseModel: models.BaseModel{
			ID:       teamID,
			Name:     "backend-team",
			Title:    "Backend Team",
			Metadata: existingMetadata,
		},
		GroupID: groupID,
	}

	group := &models.Group{
		BaseModel: models.BaseModel{
			ID:   groupID,
			Name: "engineering",
		},
		OrgID: orgID,
	}

	// New metadata
	newMetadata := json.RawMessage(`{"status": "active"}`)

	// Mock expectations
	suite.mockTeamRepo.EXPECT().GetByID(teamID).Return(team, nil)
	suite.mockTeamRepo.EXPECT().Update(gomock.Any()).DoAndReturn(func(t *models.Team) error {
		// Verify new metadata
		var merged map[string]interface{}
		err := json.Unmarshal(t.Metadata, &merged)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), "active", merged["status"])
		return nil
	})
	suite.mockGroupRepo.EXPECT().GetByID(groupID).Return(group, nil)

	// Execute
	result, err := suite.teamService.UpdateTeamMetadata(teamID, newMetadata)

	// Assert
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.Equal(suite.T(), teamID, result.ID)
}

func (suite *TeamServiceTestSuite) TestUpdateTeamMetadata_Success_ComplexNestedMetadata() {
	teamID := uuid.New()
	groupID := uuid.New()
	orgID := uuid.New()

	// Existing team with complex nested metadata
	existingMetadata := json.RawMessage(`{"tags": ["backend"], "config": {"env": "prod", "region": "us-east"}}`)
	team := &models.Team{
		BaseModel: models.BaseModel{
			ID:       teamID,
			Name:     "backend-team",
			Title:    "Backend Team",
			Metadata: existingMetadata,
		},
		GroupID: groupID,
	}

	group := &models.Group{
		BaseModel: models.BaseModel{
			ID:   groupID,
			Name: "engineering",
		},
		OrgID: orgID,
	}

	// New metadata with nested structure
	newMetadata := json.RawMessage(`{"config": {"env": "staging", "replicas": 3}, "owner": "john.doe"}`)

	// Mock expectations
	suite.mockTeamRepo.EXPECT().GetByID(teamID).Return(team, nil)
	suite.mockTeamRepo.EXPECT().Update(gomock.Any()).DoAndReturn(func(t *models.Team) error {
		// Verify merged metadata
		var merged map[string]interface{}
		err := json.Unmarshal(t.Metadata, &merged)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), "backend", merged["tags"].([]interface{})[0])
		assert.Equal(suite.T(), "john.doe", merged["owner"])
		// Config should be completely replaced (not deep merged)
		config := merged["config"].(map[string]interface{})
		assert.Equal(suite.T(), "staging", config["env"])
		assert.Equal(suite.T(), float64(3), config["replicas"])
		return nil
	})
	suite.mockGroupRepo.EXPECT().GetByID(groupID).Return(group, nil)

	// Execute
	result, err := suite.teamService.UpdateTeamMetadata(teamID, newMetadata)

	// Assert
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.Equal(suite.T(), teamID, result.ID)
}

func (suite *TeamServiceTestSuite) TestUpdateTeamMetadata_TeamNotFound() {
	teamID := uuid.New()

	// New metadata
	newMetadata := json.RawMessage(`{"status": "active"}`)

	// Mock expectations - team not found
	suite.mockTeamRepo.EXPECT().GetByID(teamID).Return(nil, gorm.ErrRecordNotFound)

	// Execute
	result, err := suite.teamService.UpdateTeamMetadata(teamID, newMetadata)

	// Assert
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), result)
	assert.ErrorIs(suite.T(), err, apperrors.ErrTeamNotFound)
}

func (suite *TeamServiceTestSuite) TestUpdateTeamMetadata_GetByIDError() {
	teamID := uuid.New()
	expectedError := errors.New("database connection error")

	// New metadata
	newMetadata := json.RawMessage(`{"status": "active"}`)

	// Mock expectations - database error
	suite.mockTeamRepo.EXPECT().GetByID(teamID).Return(nil, expectedError)

	// Execute
	result, err := suite.teamService.UpdateTeamMetadata(teamID, newMetadata)

	// Assert
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), result)
	assert.Contains(suite.T(), err.Error(), "failed to get team")
	assert.NotErrorIs(suite.T(), err, apperrors.ErrTeamNotFound)
}

func (suite *TeamServiceTestSuite) TestUpdateTeamMetadata_InvalidExistingMetadata() {
	teamID := uuid.New()

	// Team with invalid existing metadata
	existingMetadata := json.RawMessage(`{invalid json}`)
	team := &models.Team{
		BaseModel: models.BaseModel{
			ID:       teamID,
			Name:     "backend-team",
			Title:    "Backend Team",
			Metadata: existingMetadata,
		},
		GroupID: uuid.New(),
	}

	// New metadata
	newMetadata := json.RawMessage(`{"status": "active"}`)

	// Mock expectations
	suite.mockTeamRepo.EXPECT().GetByID(teamID).Return(team, nil)

	// Execute
	result, err := suite.teamService.UpdateTeamMetadata(teamID, newMetadata)

	// Assert
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), result)
	assert.Contains(suite.T(), err.Error(), "failed to parse existing metadata")
}

func (suite *TeamServiceTestSuite) TestUpdateTeamMetadata_InvalidNewMetadata() {
	teamID := uuid.New()

	// Existing team with valid metadata
	existingMetadata := json.RawMessage(`{"tags": ["backend"]}`)
	team := &models.Team{
		BaseModel: models.BaseModel{
			ID:       teamID,
			Name:     "backend-team",
			Title:    "Backend Team",
			Metadata: existingMetadata,
		},
		GroupID: uuid.New(),
	}

	// Invalid new metadata
	newMetadata := json.RawMessage(`{invalid json}`)

	// Mock expectations
	suite.mockTeamRepo.EXPECT().GetByID(teamID).Return(team, nil)

	// Execute
	result, err := suite.teamService.UpdateTeamMetadata(teamID, newMetadata)

	// Assert
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), result)
	assert.Contains(suite.T(), err.Error(), "failed to parse new metadata")
}

func (suite *TeamServiceTestSuite) TestUpdateTeamMetadata_UpdateRepositoryError() {
	teamID := uuid.New()
	expectedError := errors.New("database update failed")

	// Existing team with metadata
	existingMetadata := json.RawMessage(`{"tags": ["backend"]}`)
	team := &models.Team{
		BaseModel: models.BaseModel{
			ID:       teamID,
			Name:     "backend-team",
			Title:    "Backend Team",
			Metadata: existingMetadata,
		},
		GroupID: uuid.New(),
	}

	// New metadata
	newMetadata := json.RawMessage(`{"status": "active"}`)

	// Mock expectations
	suite.mockTeamRepo.EXPECT().GetByID(teamID).Return(team, nil)
	suite.mockTeamRepo.EXPECT().Update(gomock.Any()).Return(expectedError)

	// Execute
	result, err := suite.teamService.UpdateTeamMetadata(teamID, newMetadata)

	// Assert
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), result)
	assert.Contains(suite.T(), err.Error(), "failed to update team metadata")
}

func (suite *TeamServiceTestSuite) TestUpdateTeamMetadata_ToResponseError() {
	teamID := uuid.New()
	groupID := uuid.New()

	// Existing team with metadata
	existingMetadata := json.RawMessage(`{"tags": ["backend"]}`)
	team := &models.Team{
		BaseModel: models.BaseModel{
			ID:       teamID,
			Name:     "backend-team",
			Title:    "Backend Team",
			Metadata: existingMetadata,
		},
		GroupID: groupID,
	}

	// New metadata
	newMetadata := json.RawMessage(`{"status": "active"}`)

	// Mock expectations
	suite.mockTeamRepo.EXPECT().GetByID(teamID).Return(team, nil)
	suite.mockTeamRepo.EXPECT().Update(gomock.Any()).Return(nil)
	suite.mockGroupRepo.EXPECT().GetByID(groupID).Return(nil, errors.New("group not found"))

	// Execute
	result, err := suite.teamService.UpdateTeamMetadata(teamID, newMetadata)

	// Assert
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), result)
	assert.Contains(suite.T(), err.Error(), "failed to get group for team")
}

func (suite *TeamServiceTestSuite) TestUpdateTeamMetadata_PreservesUnmentionedFields() {
	teamID := uuid.New()
	groupID := uuid.New()
	orgID := uuid.New()

	// Existing team with multiple metadata fields
	existingMetadata := json.RawMessage(`{"tags": ["backend"], "priority": "high", "status": "active", "owner": "jane.doe"}`)
	team := &models.Team{
		BaseModel: models.BaseModel{
			ID:       teamID,
			Name:     "backend-team",
			Title:    "Backend Team",
			Metadata: existingMetadata,
		},
		GroupID: groupID,
	}

	group := &models.Group{
		BaseModel: models.BaseModel{
			ID:   groupID,
			Name: "engineering",
		},
		OrgID: orgID,
	}

	// New metadata updating only one field
	newMetadata := json.RawMessage(`{"priority": "critical"}`)

	// Mock expectations
	suite.mockTeamRepo.EXPECT().GetByID(teamID).Return(team, nil)
	suite.mockTeamRepo.EXPECT().Update(gomock.Any()).DoAndReturn(func(t *models.Team) error {
		// Verify all fields are preserved except updated one
		var merged map[string]interface{}
		err := json.Unmarshal(t.Metadata, &merged)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), "backend", merged["tags"].([]interface{})[0])
		assert.Equal(suite.T(), "critical", merged["priority"]) // Updated
		assert.Equal(suite.T(), "active", merged["status"])     // Preserved
		assert.Equal(suite.T(), "jane.doe", merged["owner"])    // Preserved
		return nil
	})
	suite.mockGroupRepo.EXPECT().GetByID(groupID).Return(group, nil)

	// Execute
	result, err := suite.teamService.UpdateTeamMetadata(teamID, newMetadata)

	// Assert
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.Equal(suite.T(), teamID, result.ID)
}

// TestTeamServiceTestSuite runs the test suite
func TestTeamServiceTestSuite(t *testing.T) {
	suite.Run(t, new(TeamServiceTestSuite))
}
