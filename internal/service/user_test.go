package service_test

import (
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

func uuidPtr() *uuid.UUID {
	u := uuid.New()
	return &u
}

// UserServiceTestSuite defines the test suite for UserService
type UserServiceTestSuite struct {
	suite.Suite
	ctrl         *gomock.Controller
	mockUserRepo *mocks.MockUserRepositoryInterface
	userService  *service.UserService
	validator    *validator.Validate
}

// SetupTest sets up the test suite
func (suite *UserServiceTestSuite) SetupTest() {
	suite.ctrl = gomock.NewController(suite.T())
	suite.mockUserRepo = mocks.NewMockUserRepositoryInterface(suite.ctrl)
	suite.validator = validator.New()
	mockLinkRepo := mocks.NewMockLinkRepositoryInterface(suite.ctrl)
	mockPluginRepo := mocks.NewMockPluginRepositoryInterface(suite.ctrl)

	// Create service with mock repository
	suite.userService = service.NewUserService(suite.mockUserRepo, mockLinkRepo, mockPluginRepo, suite.validator)
}

// TearDownTest cleans up after each test
func (suite *UserServiceTestSuite) TearDownTest() {
	suite.ctrl.Finish()
}

// TestCreateUser tests creating a member
func (suite *UserServiceTestSuite) TestCreateUser() {
	role := "developer"
	teamRole := "member"
	teamID := uuid.New()
	req := &service.CreateUserRequest{
		TeamID:    &teamID,
		FirstName: "John",
		LastName:  "Doe",
		Email:     "john@example.com",
		Mobile:    "+1-555-0123",
		IUser:     "I123456",
		Role:      &role,
		TeamRole:  &teamRole,
		CreatedBy: "I123456",
	}

	// Mock GetByEmail to return not found (no existing member with same email)
	suite.mockUserRepo.EXPECT().
		GetByEmail(req.Email).
		Return(nil, gorm.ErrRecordNotFound).
		Times(1)

	// Mock Create to succeed
	suite.mockUserRepo.EXPECT().
		Create(gomock.Any()).
		Return(nil).
		Times(1)

	response, err := suite.userService.CreateUser(req)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), response)
	assert.Equal(suite.T(), req.IUser, response.ID)
	assert.Equal(suite.T(), req.FirstName, response.FirstName)
	assert.Equal(suite.T(), req.LastName, response.LastName)
	assert.Equal(suite.T(), req.Email, response.Email)
	assert.Equal(suite.T(), role, response.TeamDomain)
	assert.Equal(suite.T(), teamRole, response.TeamRole)
}

// TestCreateUserWithDefaultRoleAndTeamRole tests creating a member with default role and team role
func (suite *UserServiceTestSuite) TestCreateUserWithDefaultRoleAndTeamRole() {
	teamID := uuid.New()
	req := &service.CreateUserRequest{
		TeamID:    &teamID,
		FirstName: "John",
		LastName:  "Doe",
		Email:     "john@example.com",
		Mobile:    "+1-555-0123",
		IUser:     "I123456",
		CreatedBy: "I123456",
		// Role and TeamRole are not provided - should use defaults
	}

	// Mock GetByEmail to return not found (no existing member with same email)
	suite.mockUserRepo.EXPECT().
		GetByEmail(req.Email).
		Return(nil, gorm.ErrRecordNotFound).
		Times(1)

	// Mock Create to succeed
	suite.mockUserRepo.EXPECT().
		Create(gomock.Any()).
		Return(nil).
		Times(1)

	response, err := suite.userService.CreateUser(req)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), response)
	assert.Equal(suite.T(), req.IUser, response.ID)
	assert.Equal(suite.T(), req.FirstName, response.FirstName)
	assert.Equal(suite.T(), req.LastName, response.LastName)
	assert.Equal(suite.T(), req.Email, response.Email)
	assert.Equal(suite.T(), "developer", response.TeamDomain) // Default role
	assert.Equal(suite.T(), "member", response.TeamRole)      // Default team role
}

// TestCreateUserValidationError tests creating a member with validation error
func (suite *UserServiceTestSuite) TestCreateUserValidationError() {
	role := "developer"
	req := &service.CreateUserRequest{
		// Missing required fields to trigger validation error
		FirstName: "", // required
		LastName:  "Doe",
		Email:     "john@example.com",
		IUser:     "I123456",
		Role:      &role,
	}

	response, err := suite.userService.CreateUser(req)

	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), response)
	assert.Contains(suite.T(), err.Error(), "validation failed")
}

// TestCreateUserDuplicateEmail tests creating a member with duplicate email
func (suite *UserServiceTestSuite) TestCreateUserDuplicateEmail() {
	role := "developer"
	req := &service.CreateUserRequest{
		FirstName: "John",
		LastName:  "Doe",
		Email:     "john@example.com",
		IUser:     "I123456",
		Role:      &role,
		CreatedBy: "I123456",
	}

	existingUser := &models.User{
		UserID:     "I789012",
		FirstName:  "Jane",
		LastName:   "Doe",
		Email:      req.Email,
		TeamDomain: models.TeamDomainDeveloper,
		TeamRole:   models.TeamRoleMember,
	}

	// Mock GetByEmail to return existing member
	suite.mockUserRepo.EXPECT().
		GetByEmail(req.Email).
		Return(existingUser, nil).
		Times(1)

	response, err := suite.userService.CreateUser(req)

	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), response)
	assert.Contains(suite.T(), err.Error(), "user already exists")
}

// TestGetUserByID tests getting a user by ID
func (suite *UserServiceTestSuite) TestGetUserByID() {
	userID := uuid.New()
	existingUser := &models.User{
		TeamID:     &userID,
		UserID:     "I123456",
		FirstName:  "John",
		LastName:   "Doe",
		Email:      "john@example.com",
		Mobile:     "+1-555-0123",
		TeamDomain: models.TeamDomainDeveloper,
		TeamRole:   models.TeamRoleMember,
	}

	suite.mockUserRepo.EXPECT().
		GetByID(userID).
		Return(existingUser, nil).
		Times(1)

	response, err := suite.userService.GetUserByID(userID)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), response)
	assert.Equal(suite.T(), existingUser.UserID, response.ID)
	assert.Equal(suite.T(), existingUser.FirstName, response.FirstName)
	assert.Equal(suite.T(), existingUser.LastName, response.LastName)
	assert.Equal(suite.T(), existingUser.Email, response.Email)
}

// TestGetUserByIDNotFound tests getting a member by ID when not found
func (suite *UserServiceTestSuite) TestGetUserByIDNotFound() {
	userID := uuid.New()

	suite.mockUserRepo.EXPECT().
		GetByID(userID).
		Return(nil, gorm.ErrRecordNotFound).
		Times(1)

	response, err := suite.userService.GetUserByID(userID)

	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), response)
	assert.Contains(suite.T(), err.Error(), "user not found")
}

// TestGetMembersByOrganization tests getting members by organization
func (suite *UserServiceTestSuite) TestGetMembersByOrganization() {
	orgID := uuid.New()
	limit, offset := 20, 0
	existingUsers := []models.User{
		{
			TeamID:     uuidPtr(),
			UserID:     "I123456",
			FirstName:  "John",
			LastName:   "Doe",
			Email:      "john@example.com",
			TeamDomain: models.TeamDomainDeveloper,
			TeamRole:   models.TeamRoleMember,
		},
		{
			TeamID:     uuidPtr(),
			UserID:     "I789012",
			FirstName:  "Jane",
			LastName:   "Smith",
			Email:      "jane@example.com",
			TeamDomain: models.TeamDomainPO,
			TeamRole:   models.TeamRoleManager,
		},
	}
	expectedTotal := int64(2)

	suite.mockUserRepo.EXPECT().
		GetByOrganizationID(orgID, limit, offset).
		Return(existingUsers, expectedTotal, nil).
		Times(1)

	responses, total, err := suite.userService.GetUsersByOrganization(orgID, limit, offset)

	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), expectedTotal, total)
	assert.Len(suite.T(), responses, 2)
	assert.Equal(suite.T(), existingUsers[0].FirstName, responses[0].FirstName)
	assert.Equal(suite.T(), existingUsers[0].LastName, responses[0].LastName)
	assert.Equal(suite.T(), existingUsers[1].FirstName, responses[1].FirstName)
	assert.Equal(suite.T(), existingUsers[1].LastName, responses[1].LastName)
}

// TestUpdateMember tests updating a member
func (suite *UserServiceTestSuite) TestUpdateMember() {
	userID := uuid.New()
	existingUser := &models.User{
		TeamID:     &userID,
		UserID:     "I123456",
		FirstName:  "John",
		LastName:   "Doe",
		Email:      "john@example.com",
		Mobile:     "+1-555-0123",
		TeamDomain: models.TeamDomainDeveloper,
		TeamRole:   models.TeamRoleMember,
	}

	newFirstName := "John"
	newLastName := "Updated"
	newEmail := "john.updated@example.com"
	req := &service.UpdateUserRequest{
		FirstName: &newFirstName,
		LastName:  &newLastName,
		Email:     &newEmail,
	}

	suite.mockUserRepo.EXPECT().
		GetByID(userID).
		Return(existingUser, nil).
		Times(1)

	suite.mockUserRepo.EXPECT().
		GetByEmail(newEmail).
		Return(nil, gorm.ErrRecordNotFound).
		Times(1)

	suite.mockUserRepo.EXPECT().
		Update(gomock.Any()).
		Return(nil).
		Times(1)

	response, err := suite.userService.UpdateUser(userID, req)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), response)
	assert.Equal(suite.T(), newFirstName, response.FirstName)
	assert.Equal(suite.T(), newLastName, response.LastName)
	assert.Equal(suite.T(), newEmail, response.Email)
}

// TestDeleteMember tests deleting a member
func (suite *UserServiceTestSuite) TestDeleteMember() {
	userID := uuid.New()
	existingUser := &models.User{
		TeamID:     &userID,
		UserID:     "I123456",
		FirstName:  "John",
		LastName:   "Doe",
		Email:      "john@example.com",
		TeamDomain: models.TeamDomainDeveloper,
		TeamRole:   models.TeamRoleMember,
	}

	suite.mockUserRepo.EXPECT().
		GetByID(userID).
		Return(existingUser, nil).
		Times(1)

	suite.mockUserRepo.EXPECT().
		Delete(userID).
		Return(nil).
		Times(1)

	err := suite.userService.DeleteUser(userID)

	assert.NoError(suite.T(), err)
}

// TestSearchMembers tests searching for members
func (suite *UserServiceTestSuite) TestSearchMembers() {
	orgID := uuid.New()
	query := "john"
	limit, offset := 20, 0
	existingUsers := []models.User{
		{
			TeamID:     uuidPtr(),
			UserID:     "I123456",
			FirstName:  "John",
			LastName:   "Doe",
			Email:      "john.doe@example.com",
			TeamDomain: models.TeamDomainDeveloper,
			TeamRole:   models.TeamRoleMember,
		},
		{
			TeamID:     uuidPtr(),
			UserID:     "I789012",
			FirstName:  "Mary",
			LastName:   "Johnson",
			Email:      "mary.johnson@example.com",
			TeamDomain: models.TeamDomainPO,
			TeamRole:   models.TeamRoleManager,
		},
	}
	expectedTotal := int64(2)

	suite.mockUserRepo.EXPECT().
		SearchByOrganization(orgID, query, limit, offset).
		Return(existingUsers, expectedTotal, nil).
		Times(1)

	responses, total, err := suite.userService.SearchUsers(orgID, query, limit, offset)

	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), expectedTotal, total)
	assert.Len(suite.T(), responses, 2)
	assert.Equal(suite.T(), existingUsers[0].Email, responses[0].Email)
	assert.Equal(suite.T(), existingUsers[1].Email, responses[1].Email)
}

// TestSearchMembersError tests searching for members with error
func (suite *UserServiceTestSuite) TestSearchMembersError() {
	orgID := uuid.New()
	query := "test"
	limit, offset := 20, 0

	suite.mockUserRepo.EXPECT().
		SearchByOrganization(orgID, query, limit, offset).
		Return(nil, int64(0), gorm.ErrInvalidDB).
		Times(1)

	responses, total, err := suite.userService.SearchUsers(orgID, query, limit, offset)

	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), responses)
	assert.Equal(suite.T(), int64(0), total)
	assert.Contains(suite.T(), err.Error(), "failed to search users")
}

// TestGetActiveMembers tests getting active members
func (suite *UserServiceTestSuite) TestGetActiveMembers() {
	orgID := uuid.New()
	limit, offset := 20, 0
	existingUsers := []models.User{
		{
			TeamID:     uuidPtr(),
			UserID:     "I123456",
			FirstName:  "Active",
			LastName:   "Smith",
			Email:      "active.smith@example.com",
			TeamDomain: models.TeamDomainDeveloper,
			TeamRole:   models.TeamRoleMember,
		},
		{
			TeamID:     uuidPtr(),
			UserID:     "I789012",
			FirstName:  "Active",
			LastName:   "Jones",
			Email:      "active.jones@example.com",
			TeamDomain: models.TeamDomainPO,
			TeamRole:   models.TeamRoleManager,
		},
	}
	expectedTotal := int64(2)

	suite.mockUserRepo.EXPECT().
		GetActiveByOrganization(orgID, limit, offset).
		Return(existingUsers, expectedTotal, nil).
		Times(1)

	responses, total, err := suite.userService.GetActiveUsers(orgID, limit, offset)

	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), expectedTotal, total)
	assert.Len(suite.T(), responses, 2)
	assert.Equal(suite.T(), existingUsers[0].Email, responses[0].Email)
	assert.Equal(suite.T(), existingUsers[1].Email, responses[1].Email)
}

// TestGetActiveMembersError tests getting active members with error
func (suite *UserServiceTestSuite) TestGetActiveMembersError() {
	orgID := uuid.New()
	limit, offset := 20, 0

	suite.mockUserRepo.EXPECT().
		GetActiveByOrganization(orgID, limit, offset).
		Return(nil, int64(0), gorm.ErrInvalidDB).
		Times(1)

	responses, total, err := suite.userService.GetActiveUsers(orgID, limit, offset)

	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), responses)
	assert.Equal(suite.T(), int64(0), total)
	assert.Contains(suite.T(), err.Error(), "failed to get active users")
}

// TestUpdateMemberNotFound tests updating a member that doesn't exist
func (suite *UserServiceTestSuite) TestUpdateMemberNotFound() {
	userID := uuid.New()
	newFirstName := "John"
	req := &service.UpdateUserRequest{
		FirstName: &newFirstName,
	}

	suite.mockUserRepo.EXPECT().
		GetByID(userID).
		Return(nil, gorm.ErrRecordNotFound).
		Times(1)

	response, err := suite.userService.UpdateUser(userID, req)

	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), response)
	assert.Contains(suite.T(), err.Error(), "user not found")
}

// TestUpdateMemberEmailConflict tests updating a member with a conflicting email
func (suite *UserServiceTestSuite) TestUpdateMemberEmailConflict() {
	userID := uuid.New()
	existingUser := &models.User{
		TeamID:     &userID,
		UserID:     "I123456",
		FirstName:  "John",
		LastName:   "Doe",
		Email:      "john@example.com",
		TeamDomain: models.TeamDomainDeveloper,
		TeamRole:   models.TeamRoleMember,
	}

	conflictingEmail := "taken@example.com"
	conflictingUser := &models.User{
		TeamID: uuidPtr(),
		UserID: "I999999",
		Email:  conflictingEmail,
	}

	req := &service.UpdateUserRequest{
		Email: &conflictingEmail,
	}

	suite.mockUserRepo.EXPECT().
		GetByID(userID).
		Return(existingUser, nil).
		Times(1)

	suite.mockUserRepo.EXPECT().
		GetByEmail(conflictingEmail).
		Return(conflictingUser, nil).
		Times(1)

	response, err := suite.userService.UpdateUser(userID, req)

	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), response)
	assert.Contains(suite.T(), err.Error(), "user already exists")
}

// TestDeleteMemberNotFound tests deleting a member that doesn't exist
func (suite *UserServiceTestSuite) TestDeleteMemberNotFound() {
	userID := uuid.New()

	suite.mockUserRepo.EXPECT().
		GetByID(userID).
		Return(nil, gorm.ErrRecordNotFound).
		Times(1)

	err := suite.userService.DeleteUser(userID)

	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "user not found")
}

// ===== Quick Links validation tests =====

// TestAddQuickLinkValidation tests the validation logic for adding a quick link
func TestAddQuickLinkValidation(t *testing.T) {
	validator := validator.New()

	testCases := []struct {
		name        string
		request     *service.AddQuickLinkRequest
		expectError bool
	}{
		{
			name: "Valid quick link",
			request: &service.AddQuickLinkRequest{
				URL:      "https://github.com/user/repo",
				Title:    "My Repository",
				Icon:     "github",
				Category: "repository",
			},
			expectError: false,
		},
		{
			name: "Valid quick link without optional fields",
			request: &service.AddQuickLinkRequest{
				URL:   "https://example.com",
				Title: "Example",
			},
			expectError: false,
		},
		{
			name: "Missing URL",
			request: &service.AddQuickLinkRequest{
				Title:    "My Repository",
				Icon:     "github",
				Category: "repository",
			},
			expectError: true,
		},
		{
			name: "Invalid URL",
			request: &service.AddQuickLinkRequest{
				URL:   "not-a-url",
				Title: "My Repository",
			},
			expectError: true,
		},
		{
			name: "Missing title",
			request: &service.AddQuickLinkRequest{
				URL:      "https://github.com/userpo",
				Icon:     "github",
				Category: "repository",
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

func TestUserServiceTestSuite(t *testing.T) {
	suite.Run(t, new(UserServiceTestSuite))
}
