package service_test

import (
	apperrors "developer-portal-backend/internal/errors"
	"developer-portal-backend/internal/testutils"
	"encoding/json"
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
	ctrl           *gomock.Controller
	mockUserRepo   *mocks.MockUserRepositoryInterface
	mockLinkRepo   *mocks.MockLinkRepositoryInterface
	mockPluginRepo *mocks.MockPluginRepositoryInterface
	userService    *service.UserService
	validator      *validator.Validate
	factories      *testutils.FactorySet
}

// SetupTest sets up the test suite
func (suite *UserServiceTestSuite) SetupTest() {
	suite.ctrl = gomock.NewController(suite.T())
	suite.mockUserRepo = mocks.NewMockUserRepositoryInterface(suite.ctrl)
	suite.mockLinkRepo = mocks.NewMockLinkRepositoryInterface(suite.ctrl)
	suite.mockPluginRepo = mocks.NewMockPluginRepositoryInterface(suite.ctrl)
	suite.validator = validator.New()
	suite.factories = testutils.NewFactorySet()

	// Create service with mock repository
	suite.userService = service.NewUserService(suite.mockUserRepo, suite.mockLinkRepo, suite.mockPluginRepo, suite.validator)
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

	existingUser := suite.factories.User.WithEmail(req.Email)

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
	existingUser := suite.factories.User.Create()
	existingUser.TeamID = &userID

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
	existingUser := suite.factories.User.Create()
	existingUser.TeamID = &userID

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
	existingUser := suite.factories.User.Create()
	existingUser.TeamID = &userID

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

func (suite *UserServiceTestSuite) TestUpdateMemberEmailConflict() {
	userID := uuid.New()
	existingUser := suite.factories.User.Create()
	existingUser.TeamID = &userID

	conflictingEmail := "taken@example.com"
	conflictingUser := suite.factories.User.WithEmail(conflictingEmail)
	conflictingUser.TeamID = uuidPtr()

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

// ===== Tests for UpdateUserTeam =====

// TestUpdateUserTeam_Success tests successfully updating a user's team
func (suite *UserServiceTestSuite) TestUpdateUserTeam_Success() {
	userID := uuid.New()
	teamID := uuid.New()
	updatedBy := "I999999"

	existingUser := suite.factories.User.Create()
	existingUser.TeamID = nil // No team initially

	suite.mockUserRepo.EXPECT().
		GetByID(userID).
		Return(existingUser, nil).
		Times(1)

	suite.mockUserRepo.EXPECT().
		Update(gomock.Any()).
		DoAndReturn(func(user *models.User) error {
			// Verify the team was updated
			assert.NotNil(suite.T(), user.TeamID)
			assert.Equal(suite.T(), teamID, *user.TeamID)
			assert.Equal(suite.T(), updatedBy, user.UpdatedBy)
			return nil
		}).
		Times(1)

	response, err := suite.userService.UpdateUserTeam(userID, teamID, updatedBy)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), response)
	assert.Equal(suite.T(), existingUser.UserID, response.ID)
}

// TestUpdateUserTeam_EmptyUpdatedBy tests error when updatedBy is empty
func (suite *UserServiceTestSuite) TestUpdateUserTeam_EmptyUpdatedBy() {
	userID := uuid.New()
	teamID := uuid.New()
	updatedBy := ""

	response, err := suite.userService.UpdateUserTeam(userID, teamID, updatedBy)

	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), response)
	assert.Contains(suite.T(), err.Error(), "updated_by is required")
}

// TestUpdateUserTeam_UserNotFound tests error when user is not found
func (suite *UserServiceTestSuite) TestUpdateUserTeam_UserNotFound() {
	userID := uuid.New()
	teamID := uuid.New()
	updatedBy := "I999999"

	suite.mockUserRepo.EXPECT().
		GetByID(userID).
		Return(nil, apperrors.ErrUserNotFound).
		Times(1)

	response, err := suite.userService.UpdateUserTeam(userID, teamID, updatedBy)

	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), response)
	assert.Contains(suite.T(), err.Error(), "user not found")
}

// TestUpdateUserTeam_UpdateFails tests error when repository update fails
func (suite *UserServiceTestSuite) TestUpdateUserTeam_UpdateFails() {
	userID := uuid.New()
	teamID := uuid.New()
	updatedBy := "I999999"

	existingUser := suite.factories.User.Create()
	existingUser.TeamID = nil

	suite.mockUserRepo.EXPECT().
		GetByID(userID).
		Return(existingUser, nil).
		Times(1)

	suite.mockUserRepo.EXPECT().
		Update(gomock.Any()).
		Return(gorm.ErrInvalidDB).
		Times(1)

	response, err := suite.userService.UpdateUserTeam(userID, teamID, updatedBy)

	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), response)
	assert.Contains(suite.T(), err.Error(), "failed to update user team")
}

// TestUpdateUserTeam_ChangeExistingTeam tests updating a user who already has a team
func (suite *UserServiceTestSuite) TestUpdateUserTeam_ChangeExistingTeam() {
	userID := uuid.New()
	oldTeamID := uuid.New()
	newTeamID := uuid.New()
	updatedBy := "I999999"

	existingUser := suite.factories.User.Create()
	existingUser.TeamID = &oldTeamID // Already has a team

	suite.mockUserRepo.EXPECT().
		GetByID(userID).
		Return(existingUser, nil).
		Times(1)

	suite.mockUserRepo.EXPECT().
		Update(gomock.Any()).
		DoAndReturn(func(user *models.User) error {
			// Verify the team was changed to the new team
			assert.NotNil(suite.T(), user.TeamID)
			assert.Equal(suite.T(), newTeamID, *user.TeamID)
			assert.NotEqual(suite.T(), oldTeamID, *user.TeamID)
			assert.Equal(suite.T(), updatedBy, user.UpdatedBy)
			return nil
		}).
		Times(1)

	response, err := suite.userService.UpdateUserTeam(userID, newTeamID, updatedBy)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), response)
	assert.Equal(suite.T(), existingUser.UserID, response.ID)
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

// TestAddFavoriteLinkByUserID_Success tests successfully adding a favorite link to a user with no existing metadata
func (suite *UserServiceTestSuite) TestAddFavoriteLinkByUserID_Success() {
	userID := "I123456"
	linkID := uuid.New()

	existingUser := suite.factories.User.Create()
	existingUser.UserID = userID
	existingUser.Metadata = nil // No existing metadata

	suite.mockUserRepo.EXPECT().
		GetByUserID(userID).
		Return(existingUser, nil).
		Times(1)

	suite.mockUserRepo.EXPECT().
		Update(gomock.Any()).
		DoAndReturn(func(user *models.User) error {
			// Verify metadata was updated correctly
			assert.NotNil(suite.T(), user.Metadata)
			var meta map[string]interface{}
			err := json.Unmarshal(user.Metadata, &meta)
			assert.NoError(suite.T(), err)

			favorites, ok := meta["favorites"]
			assert.True(suite.T(), ok)

			favArray, ok := favorites.([]interface{})
			assert.True(suite.T(), ok)
			assert.Len(suite.T(), favArray, 1)
			assert.Equal(suite.T(), linkID.String(), favArray[0])

			return nil
		}).
		Times(1)

	response, err := suite.userService.AddFavoriteLinkByUserID(userID, linkID)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), response)
	assert.Equal(suite.T(), userID, response.ID)
}

// TestAddFavoriteLinkByUserID_WithExistingMetadata tests adding a favorite link to a user with existing metadata but no favorites
func (suite *UserServiceTestSuite) TestAddFavoriteLinkByUserID_WithExistingMetadata() {
	userID := "I123456"
	linkID := uuid.New()

	existingMetadata := map[string]interface{}{
		"portal_admin": true,
		"other_field":  "value",
	}
	metadataBytes, _ := json.Marshal(existingMetadata)

	existingUser := suite.factories.User.Create()
	existingUser.UserID = userID
	existingUser.Metadata = json.RawMessage(metadataBytes)

	suite.mockUserRepo.EXPECT().
		GetByUserID(userID).
		Return(existingUser, nil).
		Times(1)

	suite.mockUserRepo.EXPECT().
		Update(gomock.Any()).
		DoAndReturn(func(user *models.User) error {
			var meta map[string]interface{}
			err := json.Unmarshal(user.Metadata, &meta)
			assert.NoError(suite.T(), err)

			// Verify existing fields are preserved
			assert.Equal(suite.T(), true, meta["portal_admin"])
			assert.Equal(suite.T(), "value", meta["other_field"])

			// Verify favorites was added
			favorites, ok := meta["favorites"]
			assert.True(suite.T(), ok)

			favArray, ok := favorites.([]interface{})
			assert.True(suite.T(), ok)
			assert.Len(suite.T(), favArray, 1)
			assert.Equal(suite.T(), linkID.String(), favArray[0])

			return nil
		}).
		Times(1)

	response, err := suite.userService.AddFavoriteLinkByUserID(userID, linkID)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), response)
}

// TestAddFavoriteLinkByUserID_WithExistingFavorites tests adding a favorite link to a user with existing favorites
func (suite *UserServiceTestSuite) TestAddFavoriteLinkByUserID_WithExistingFavorites() {
	userID := "I123456"
	existingLinkID := uuid.New()
	newLinkID := uuid.New()

	existingMetadata := map[string]interface{}{
		"favorites": []string{existingLinkID.String()},
	}
	metadataBytes, _ := json.Marshal(existingMetadata)

	existingUser := suite.factories.User.Create()
	existingUser.UserID = userID
	existingUser.Metadata = json.RawMessage(metadataBytes)

	suite.mockUserRepo.EXPECT().
		GetByUserID(userID).
		Return(existingUser, nil).
		Times(1)

	suite.mockUserRepo.EXPECT().
		Update(gomock.Any()).
		DoAndReturn(func(user *models.User) error {
			var meta map[string]interface{}
			err := json.Unmarshal(user.Metadata, &meta)
			assert.NoError(suite.T(), err)

			favorites, ok := meta["favorites"]
			assert.True(suite.T(), ok)

			favArray, ok := favorites.([]interface{})
			assert.True(suite.T(), ok)
			assert.Len(suite.T(), favArray, 2)

			// Verify both links are present
			favStrings := make([]string, len(favArray))
			for i, v := range favArray {
				favStrings[i] = v.(string)
			}
			assert.Contains(suite.T(), favStrings, existingLinkID.String())
			assert.Contains(suite.T(), favStrings, newLinkID.String())

			return nil
		}).
		Times(1)

	response, err := suite.userService.AddFavoriteLinkByUserID(userID, newLinkID)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), response)
}

// TestAddFavoriteLinkByUserID_Deduplication tests that adding a duplicate link doesn't create duplicates
func (suite *UserServiceTestSuite) TestAddFavoriteLinkByUserID_Deduplication() {
	userID := "I123456"
	linkID := uuid.New()

	existingMetadata := map[string]interface{}{
		"favorites": []string{linkID.String()},
	}
	metadataBytes, _ := json.Marshal(existingMetadata)

	existingUser := suite.factories.User.Create()
	existingUser.UserID = userID
	existingUser.Metadata = json.RawMessage(metadataBytes)

	suite.mockUserRepo.EXPECT().
		GetByUserID(userID).
		Return(existingUser, nil).
		Times(1)

	suite.mockUserRepo.EXPECT().
		Update(gomock.Any()).
		DoAndReturn(func(user *models.User) error {
			var meta map[string]interface{}
			err := json.Unmarshal(user.Metadata, &meta)
			assert.NoError(suite.T(), err)

			favorites, ok := meta["favorites"]
			assert.True(suite.T(), ok)

			favArray, ok := favorites.([]interface{})
			assert.True(suite.T(), ok)
			// Should still be 1, not duplicated
			assert.Len(suite.T(), favArray, 1)
			assert.Equal(suite.T(), linkID.String(), favArray[0])

			return nil
		}).
		Times(1)

	response, err := suite.userService.AddFavoriteLinkByUserID(userID, linkID)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), response)
}

// TestAddFavoriteLinkByUserID_EmptyUserID tests error when userID is empty
func (suite *UserServiceTestSuite) TestAddFavoriteLinkByUserID_EmptyUserID() {
	linkID := uuid.New()

	response, err := suite.userService.AddFavoriteLinkByUserID("", linkID)

	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), response)
	assert.Contains(suite.T(), err.Error(), "user_id is required")
}

// TestAddFavoriteLinkByUserID_NilLinkID tests error when linkID is nil
func (suite *UserServiceTestSuite) TestAddFavoriteLinkByUserID_NilLinkID() {
	userID := "I123456"

	response, err := suite.userService.AddFavoriteLinkByUserID(userID, uuid.Nil)

	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), response)
	assert.Contains(suite.T(), err.Error(), "link_id is required")
}

// TestAddFavoriteLinkByUserID_UserNotFound tests error when user is not found
func (suite *UserServiceTestSuite) TestAddFavoriteLinkByUserID_UserNotFound() {
	userID := "I123456"
	linkID := uuid.New()

	suite.mockUserRepo.EXPECT().
		GetByUserID(userID).
		Return(nil, gorm.ErrRecordNotFound).
		Times(1)

	response, err := suite.userService.AddFavoriteLinkByUserID(userID, linkID)

	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), response)
	assert.Contains(suite.T(), err.Error(), "user not found")
}

// TestAddFavoriteLinkByUserID_InvalidMetadata tests handling of invalid metadata JSON
func (suite *UserServiceTestSuite) TestAddFavoriteLinkByUserID_InvalidMetadata() {
	userID := "I123456"
	linkID := uuid.New()

	existingUser := suite.factories.User.Create()
	existingUser.UserID = userID
	existingUser.Metadata = json.RawMessage(`invalid json`) // Invalid JSON

	suite.mockUserRepo.EXPECT().
		GetByUserID(userID).
		Return(existingUser, nil).
		Times(1)

	suite.mockUserRepo.EXPECT().
		Update(gomock.Any()).
		DoAndReturn(func(user *models.User) error {
			// Should reset to empty object and add favorites
			var meta map[string]interface{}
			err := json.Unmarshal(user.Metadata, &meta)
			assert.NoError(suite.T(), err)

			favorites, ok := meta["favorites"]
			assert.True(suite.T(), ok)

			favArray, ok := favorites.([]interface{})
			assert.True(suite.T(), ok)
			assert.Len(suite.T(), favArray, 1)
			assert.Equal(suite.T(), linkID.String(), favArray[0])

			return nil
		}).
		Times(1)

	response, err := suite.userService.AddFavoriteLinkByUserID(userID, linkID)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), response)
}

// TestAddFavoriteLinkByUserID_UpdateFails tests error when repository update fails
func (suite *UserServiceTestSuite) TestAddFavoriteLinkByUserID_UpdateFails() {
	userID := "I123456"
	linkID := uuid.New()

	existingUser := suite.factories.User.Create()
	existingUser.UserID = userID
	existingUser.Metadata = nil

	suite.mockUserRepo.EXPECT().
		GetByUserID(userID).
		Return(existingUser, nil).
		Times(1)

	suite.mockUserRepo.EXPECT().
		Update(gomock.Any()).
		Return(gorm.ErrInvalidDB).
		Times(1)

	response, err := suite.userService.AddFavoriteLinkByUserID(userID, linkID)

	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), response)
	assert.Contains(suite.T(), err.Error(), "failed to update user")
}

// TestAddFavoriteLinkByUserID_FavoritesAsInterfaceArray tests handling favorites as []interface{} type
func (suite *UserServiceTestSuite) TestAddFavoriteLinkByUserID_FavoritesAsInterfaceArray() {
	userID := "I123456"
	existingLinkID := uuid.New()
	newLinkID := uuid.New()

	// Create metadata with favorites as []interface{}
	existingMetadata := map[string]interface{}{
		"favorites": []interface{}{existingLinkID.String()},
	}
	metadataBytes, _ := json.Marshal(existingMetadata)

	existingUser := suite.factories.User.Create()
	existingUser.UserID = userID
	existingUser.Metadata = json.RawMessage(metadataBytes)

	suite.mockUserRepo.EXPECT().
		GetByUserID(userID).
		Return(existingUser, nil).
		Times(1)

	suite.mockUserRepo.EXPECT().
		Update(gomock.Any()).
		DoAndReturn(func(user *models.User) error {
			var meta map[string]interface{}
			err := json.Unmarshal(user.Metadata, &meta)
			assert.NoError(suite.T(), err)

			favorites, ok := meta["favorites"]
			assert.True(suite.T(), ok)

			favArray, ok := favorites.([]interface{})
			assert.True(suite.T(), ok)
			assert.Len(suite.T(), favArray, 2)

			return nil
		}).
		Times(1)

	response, err := suite.userService.AddFavoriteLinkByUserID(userID, newLinkID)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), response)
}

// TestRemoveFavoriteLinkByUserID_Success tests successfully removing a favorite link from a user
func (suite *UserServiceTestSuite) TestRemoveFavoriteLinkByUserID_Success() {
	userID := "I123456"
	linkToRemove := uuid.New()
	linkToKeep := uuid.New()

	existingMetadata := map[string]interface{}{
		"favorites": []string{linkToRemove.String(), linkToKeep.String()},
	}
	metadataBytes, _ := json.Marshal(existingMetadata)

	existingUser := suite.factories.User.Create()
	existingUser.UserID = userID
	existingUser.Metadata = json.RawMessage(metadataBytes)

	suite.mockUserRepo.EXPECT().
		GetByUserID(userID).
		Return(existingUser, nil).
		Times(1)

	suite.mockUserRepo.EXPECT().
		Update(gomock.Any()).
		DoAndReturn(func(user *models.User) error {
			// Verify metadata was updated correctly
			assert.NotNil(suite.T(), user.Metadata)
			var meta map[string]interface{}
			err := json.Unmarshal(user.Metadata, &meta)
			assert.NoError(suite.T(), err)

			favorites, ok := meta["favorites"]
			assert.True(suite.T(), ok)

			favArray, ok := favorites.([]interface{})
			assert.True(suite.T(), ok)
			assert.Len(suite.T(), favArray, 1)
			assert.Equal(suite.T(), linkToKeep.String(), favArray[0])
			// Verify removed link is not present
			assert.NotContains(suite.T(), favArray, linkToRemove.String())

			return nil
		}).
		Times(1)

	response, err := suite.userService.RemoveFavoriteLinkByUserID(userID, linkToRemove)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), response)
	assert.Equal(suite.T(), userID, response.ID)
}

// TestRemoveFavoriteLinkByUserID_RemoveLastLink tests removing the last favorite link
func (suite *UserServiceTestSuite) TestRemoveFavoriteLinkByUserID_RemoveLastLink() {
	userID := "I123456"
	linkID := uuid.New()

	existingMetadata := map[string]interface{}{
		"favorites": []string{linkID.String()},
	}
	metadataBytes, _ := json.Marshal(existingMetadata)

	existingUser := suite.factories.User.Create()
	existingUser.UserID = userID
	existingUser.Metadata = json.RawMessage(metadataBytes)

	suite.mockUserRepo.EXPECT().
		GetByUserID(userID).
		Return(existingUser, nil).
		Times(1)

	suite.mockUserRepo.EXPECT().
		Update(gomock.Any()).
		DoAndReturn(func(user *models.User) error {
			var meta map[string]interface{}
			err := json.Unmarshal(user.Metadata, &meta)
			assert.NoError(suite.T(), err)

			favorites, ok := meta["favorites"]
			assert.True(suite.T(), ok)

			favArray, ok := favorites.([]interface{})
			assert.True(suite.T(), ok)
			// Should be empty array after removing last link
			assert.Len(suite.T(), favArray, 0)

			return nil
		}).
		Times(1)

	response, err := suite.userService.RemoveFavoriteLinkByUserID(userID, linkID)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), response)
}

// TestRemoveFavoriteLinkByUserID_NoExistingFavorites tests removing a link when no favorites exist
func (suite *UserServiceTestSuite) TestRemoveFavoriteLinkByUserID_NoExistingFavorites() {
	userID := "I123456"
	linkID := uuid.New()

	existingMetadata := map[string]interface{}{
		"portal_admin": true,
	}
	metadataBytes, _ := json.Marshal(existingMetadata)

	existingUser := suite.factories.User.Create()
	existingUser.UserID = userID
	existingUser.Metadata = json.RawMessage(metadataBytes)

	suite.mockUserRepo.EXPECT().
		GetByUserID(userID).
		Return(existingUser, nil).
		Times(1)

	suite.mockUserRepo.EXPECT().
		Update(gomock.Any()).
		DoAndReturn(func(user *models.User) error {
			var meta map[string]interface{}
			err := json.Unmarshal(user.Metadata, &meta)
			assert.NoError(suite.T(), err)

			// Verify other metadata is preserved
			assert.Equal(suite.T(), true, meta["portal_admin"])

			favorites, ok := meta["favorites"]
			assert.True(suite.T(), ok)

			favArray, ok := favorites.([]interface{})
			assert.True(suite.T(), ok)
			// Should be empty array
			assert.Len(suite.T(), favArray, 0)

			return nil
		}).
		Times(1)

	response, err := suite.userService.RemoveFavoriteLinkByUserID(userID, linkID)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), response)
}

// TestRemoveFavoriteLinkByUserID_NoMetadata tests removing a link when user has no metadata
func (suite *UserServiceTestSuite) TestRemoveFavoriteLinkByUserID_NoMetadata() {
	userID := "I123456"
	linkID := uuid.New()

	existingUser := suite.factories.User.Create()
	existingUser.UserID = userID
	existingUser.Metadata = nil // No metadata

	suite.mockUserRepo.EXPECT().
		GetByUserID(userID).
		Return(existingUser, nil).
		Times(1)

	suite.mockUserRepo.EXPECT().
		Update(gomock.Any()).
		DoAndReturn(func(user *models.User) error {
			var meta map[string]interface{}
			err := json.Unmarshal(user.Metadata, &meta)
			assert.NoError(suite.T(), err)

			favorites, ok := meta["favorites"]
			assert.True(suite.T(), ok)

			favArray, ok := favorites.([]interface{})
			assert.True(suite.T(), ok)
			// Should be empty array
			assert.Len(suite.T(), favArray, 0)

			return nil
		}).
		Times(1)

	response, err := suite.userService.RemoveFavoriteLinkByUserID(userID, linkID)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), response)
}

// TestRemoveFavoriteLinkByUserID_Idempotent tests that removing a non-existent link is idempotent
func (suite *UserServiceTestSuite) TestRemoveFavoriteLinkByUserID_Idempotent() {
	userID := "I123456"
	existingLinkID := uuid.New()
	nonExistentLinkID := uuid.New()

	existingMetadata := map[string]interface{}{
		"favorites": []string{existingLinkID.String()},
	}
	metadataBytes, _ := json.Marshal(existingMetadata)

	existingUser := suite.factories.User.Create()
	existingUser.UserID = userID
	existingUser.Metadata = json.RawMessage(metadataBytes)

	suite.mockUserRepo.EXPECT().
		GetByUserID(userID).
		Return(existingUser, nil).
		Times(1)

	suite.mockUserRepo.EXPECT().
		Update(gomock.Any()).
		DoAndReturn(func(user *models.User) error {
			var meta map[string]interface{}
			err := json.Unmarshal(user.Metadata, &meta)
			assert.NoError(suite.T(), err)

			favorites, ok := meta["favorites"]
			assert.True(suite.T(), ok)

			favArray, ok := favorites.([]interface{})
			assert.True(suite.T(), ok)
			// Should still have the existing link
			assert.Len(suite.T(), favArray, 1)
			assert.Equal(suite.T(), existingLinkID.String(), favArray[0])

			return nil
		}).
		Times(1)

	response, err := suite.userService.RemoveFavoriteLinkByUserID(userID, nonExistentLinkID)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), response)
}

// TestRemoveFavoriteLinkByUserID_EmptyUserID tests error when userID is empty
func (suite *UserServiceTestSuite) TestRemoveFavoriteLinkByUserID_EmptyUserID() {
	linkID := uuid.New()

	response, err := suite.userService.RemoveFavoriteLinkByUserID("", linkID)

	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), response)
	assert.Contains(suite.T(), err.Error(), "user_id is required")
}

// TestRemoveFavoriteLinkByUserID_NilLinkID tests error when linkID is nil
func (suite *UserServiceTestSuite) TestRemoveFavoriteLinkByUserID_NilLinkID() {
	userID := "I123456"

	response, err := suite.userService.RemoveFavoriteLinkByUserID(userID, uuid.Nil)

	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), response)
	assert.Contains(suite.T(), err.Error(), "link_id is required")
}

// TestRemoveFavoriteLinkByUserID_UserNotFound tests error when user is not found
func (suite *UserServiceTestSuite) TestRemoveFavoriteLinkByUserID_UserNotFound() {
	userID := "I123456"
	linkID := uuid.New()

	suite.mockUserRepo.EXPECT().
		GetByUserID(userID).
		Return(nil, apperrors.ErrUserNotFound).
		Times(1)

	response, err := suite.userService.RemoveFavoriteLinkByUserID(userID, linkID)

	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), response)
	assert.Contains(suite.T(), err.Error(), "user not found")
}

// TestRemoveFavoriteLinkByUserID_InvalidMetadata tests handling of invalid metadata JSON
func (suite *UserServiceTestSuite) TestRemoveFavoriteLinkByUserID_InvalidMetadata() {
	userID := "I123456"
	linkID := uuid.New()

	existingUser := suite.factories.User.Create()
	existingUser.UserID = userID
	existingUser.Metadata = json.RawMessage(`invalid json`) // Invalid JSON

	suite.mockUserRepo.EXPECT().
		GetByUserID(userID).
		Return(existingUser, nil).
		Times(1)

	suite.mockUserRepo.EXPECT().
		Update(gomock.Any()).
		DoAndReturn(func(user *models.User) error {
			// Should reset to empty object and create empty favorites array
			var meta map[string]interface{}
			err := json.Unmarshal(user.Metadata, &meta)
			assert.NoError(suite.T(), err)

			favorites, ok := meta["favorites"]
			assert.True(suite.T(), ok)

			favArray, ok := favorites.([]interface{})
			assert.True(suite.T(), ok)
			assert.Len(suite.T(), favArray, 0)

			return nil
		}).
		Times(1)

	response, err := suite.userService.RemoveFavoriteLinkByUserID(userID, linkID)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), response)
}

// TestRemoveFavoriteLinkByUserID_UpdateFails tests error when repository update fails
func (suite *UserServiceTestSuite) TestRemoveFavoriteLinkByUserID_UpdateFails() {
	userID := "I123456"
	linkID := uuid.New()

	existingMetadata := map[string]interface{}{
		"favorites": []string{linkID.String()},
	}
	metadataBytes, _ := json.Marshal(existingMetadata)

	existingUser := suite.factories.User.Create()
	existingUser.UserID = userID
	existingUser.Metadata = json.RawMessage(metadataBytes)

	suite.mockUserRepo.EXPECT().
		GetByUserID(userID).
		Return(existingUser, nil).
		Times(1)

	suite.mockUserRepo.EXPECT().
		Update(gomock.Any()).
		Return(gorm.ErrInvalidDB).
		Times(1)

	response, err := suite.userService.RemoveFavoriteLinkByUserID(userID, linkID)

	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), response)
	assert.Contains(suite.T(), err.Error(), "failed to update user")
}

// TestRemoveFavoriteLinkByUserID_FavoritesAsInterfaceArray tests handling favorites as []interface{} type
func (suite *UserServiceTestSuite) TestRemoveFavoriteLinkByUserID_FavoritesAsInterfaceArray() {
	userID := "I123456"
	linkToRemove := uuid.New()
	linkToKeep := uuid.New()

	// Create metadata with favorites as []interface{}
	existingMetadata := map[string]interface{}{
		"favorites": []interface{}{linkToRemove.String(), linkToKeep.String()},
	}
	metadataBytes, _ := json.Marshal(existingMetadata)

	existingUser := suite.factories.User.Create()
	existingUser.UserID = userID
	existingUser.Metadata = json.RawMessage(metadataBytes)

	suite.mockUserRepo.EXPECT().
		GetByUserID(userID).
		Return(existingUser, nil).
		Times(1)

	suite.mockUserRepo.EXPECT().
		Update(gomock.Any()).
		DoAndReturn(func(user *models.User) error {
			var meta map[string]interface{}
			err := json.Unmarshal(user.Metadata, &meta)
			assert.NoError(suite.T(), err)

			favorites, ok := meta["favorites"]
			assert.True(suite.T(), ok)

			favArray, ok := favorites.([]interface{})
			assert.True(suite.T(), ok)
			assert.Len(suite.T(), favArray, 1)
			assert.Equal(suite.T(), linkToKeep.String(), favArray[0])

			return nil
		}).
		Times(1)

	response, err := suite.userService.RemoveFavoriteLinkByUserID(userID, linkToRemove)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), response)
}

// TestRemoveFavoriteLinkByUserID_PreservesOtherMetadata tests that removing a link preserves other metadata fields
func (suite *UserServiceTestSuite) TestRemoveFavoriteLinkByUserID_PreservesOtherMetadata() {
	userID := "I123456"
	linkID := uuid.New()

	existingMetadata := map[string]interface{}{
		"favorites":    []string{linkID.String()},
		"portal_admin": true,
		"subscribed":   []string{uuid.New().String()},
		"custom_field": "custom_value",
	}
	metadataBytes, _ := json.Marshal(existingMetadata)

	existingUser := suite.factories.User.Create()
	existingUser.UserID = userID
	existingUser.Metadata = json.RawMessage(metadataBytes)

	suite.mockUserRepo.EXPECT().
		GetByUserID(userID).
		Return(existingUser, nil).
		Times(1)

	suite.mockUserRepo.EXPECT().
		Update(gomock.Any()).
		DoAndReturn(func(user *models.User) error {
			var meta map[string]interface{}
			err := json.Unmarshal(user.Metadata, &meta)
			assert.NoError(suite.T(), err)

			// Verify all other fields are preserved
			assert.Equal(suite.T(), true, meta["portal_admin"])
			assert.NotNil(suite.T(), meta["subscribed"])
			assert.Equal(suite.T(), "custom_value", meta["custom_field"])

			// Verify favorites is now empty
			favorites, ok := meta["favorites"]
			assert.True(suite.T(), ok)
			favArray, ok := favorites.([]interface{})
			assert.True(suite.T(), ok)
			assert.Len(suite.T(), favArray, 0)

			return nil
		}).
		Times(1)

	response, err := suite.userService.RemoveFavoriteLinkByUserID(userID, linkID)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), response)
}

// TestAddSubscribedPluginByUserID_Success tests successfully adding a subscribed plugin to a user with no existing metadata
func (suite *UserServiceTestSuite) TestAddSubscribedPluginByUserID_Success() {
	userID := "I123456"
	pluginID := uuid.New()

	existingUser := suite.factories.User.Create()
	existingUser.UserID = userID
	existingUser.Metadata = nil // No existing metadata

	suite.mockUserRepo.EXPECT().
		GetByUserID(userID).
		Return(existingUser, nil).
		Times(1)

	suite.mockUserRepo.EXPECT().
		Update(gomock.Any()).
		DoAndReturn(func(user *models.User) error {
			// Verify metadata was updated correctly
			assert.NotNil(suite.T(), user.Metadata)
			var meta map[string]interface{}
			err := json.Unmarshal(user.Metadata, &meta)
			assert.NoError(suite.T(), err)

			subscribed, ok := meta["subscribed"]
			assert.True(suite.T(), ok)

			subArray, ok := subscribed.([]interface{})
			assert.True(suite.T(), ok)
			assert.Len(suite.T(), subArray, 1)
			assert.Equal(suite.T(), pluginID.String(), subArray[0])

			return nil
		}).
		Times(1)

	response, err := suite.userService.AddSubscribedPluginByUserID(userID, pluginID)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), response)
	assert.Equal(suite.T(), userID, response.ID)
}

// TestAddSubscribedPluginByUserID_WithExistingMetadata tests adding a subscribed plugin to a user with existing metadata but no subscribed
func (suite *UserServiceTestSuite) TestAddSubscribedPluginByUserID_WithExistingMetadata() {
	userID := "I123456"
	pluginID := uuid.New()

	existingMetadata := map[string]interface{}{
		"portal_admin": true,
		"favorites":    []string{uuid.New().String()},
	}
	metadataBytes, _ := json.Marshal(existingMetadata)

	existingUser := suite.factories.User.Create()
	existingUser.UserID = userID
	existingUser.Metadata = json.RawMessage(metadataBytes)

	suite.mockUserRepo.EXPECT().
		GetByUserID(userID).
		Return(existingUser, nil).
		Times(1)

	suite.mockUserRepo.EXPECT().
		Update(gomock.Any()).
		DoAndReturn(func(user *models.User) error {
			var meta map[string]interface{}
			err := json.Unmarshal(user.Metadata, &meta)
			assert.NoError(suite.T(), err)

			// Verify existing fields are preserved
			assert.Equal(suite.T(), true, meta["portal_admin"])
			assert.NotNil(suite.T(), meta["favorites"])

			// Verify subscribed was added
			subscribed, ok := meta["subscribed"]
			assert.True(suite.T(), ok)

			subArray, ok := subscribed.([]interface{})
			assert.True(suite.T(), ok)
			assert.Len(suite.T(), subArray, 1)
			assert.Equal(suite.T(), pluginID.String(), subArray[0])

			return nil
		}).
		Times(1)

	response, err := suite.userService.AddSubscribedPluginByUserID(userID, pluginID)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), response)
}

// TestAddSubscribedPluginByUserID_WithExistingSubscribed tests adding a subscribed plugin to a user with existing subscribed plugins
func (suite *UserServiceTestSuite) TestAddSubscribedPluginByUserID_WithExistingSubscribed() {
	userID := "I123456"
	existingPluginID := uuid.New()
	newPluginID := uuid.New()

	existingMetadata := map[string]interface{}{
		"subscribed": []string{existingPluginID.String()},
	}
	metadataBytes, _ := json.Marshal(existingMetadata)

	existingUser := suite.factories.User.Create()
	existingUser.UserID = userID
	existingUser.Metadata = json.RawMessage(metadataBytes)

	suite.mockUserRepo.EXPECT().
		GetByUserID(userID).
		Return(existingUser, nil).
		Times(1)

	suite.mockUserRepo.EXPECT().
		Update(gomock.Any()).
		DoAndReturn(func(user *models.User) error {
			var meta map[string]interface{}
			err := json.Unmarshal(user.Metadata, &meta)
			assert.NoError(suite.T(), err)

			subscribed, ok := meta["subscribed"]
			assert.True(suite.T(), ok)

			subArray, ok := subscribed.([]interface{})
			assert.True(suite.T(), ok)
			assert.Len(suite.T(), subArray, 2)

			// Verify both plugins are present
			subStrings := make([]string, len(subArray))
			for i, v := range subArray {
				subStrings[i] = v.(string)
			}
			assert.Contains(suite.T(), subStrings, existingPluginID.String())
			assert.Contains(suite.T(), subStrings, newPluginID.String())

			return nil
		}).
		Times(1)

	response, err := suite.userService.AddSubscribedPluginByUserID(userID, newPluginID)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), response)
}

// TestAddSubscribedPluginByUserID_Deduplication tests that adding a duplicate plugin doesn't create duplicates
func (suite *UserServiceTestSuite) TestAddSubscribedPluginByUserID_Deduplication() {
	userID := "I123456"
	pluginID := uuid.New()

	existingMetadata := map[string]interface{}{
		"subscribed": []string{pluginID.String()},
	}
	metadataBytes, _ := json.Marshal(existingMetadata)

	existingUser := suite.factories.User.Create()
	existingUser.UserID = userID
	existingUser.Metadata = json.RawMessage(metadataBytes)

	suite.mockUserRepo.EXPECT().
		GetByUserID(userID).
		Return(existingUser, nil).
		Times(1)

	suite.mockUserRepo.EXPECT().
		Update(gomock.Any()).
		DoAndReturn(func(user *models.User) error {
			var meta map[string]interface{}
			err := json.Unmarshal(user.Metadata, &meta)
			assert.NoError(suite.T(), err)

			subscribed, ok := meta["subscribed"]
			assert.True(suite.T(), ok)

			subArray, ok := subscribed.([]interface{})
			assert.True(suite.T(), ok)
			// Should still be 1, not duplicated
			assert.Len(suite.T(), subArray, 1)
			assert.Equal(suite.T(), pluginID.String(), subArray[0])

			return nil
		}).
		Times(1)

	response, err := suite.userService.AddSubscribedPluginByUserID(userID, pluginID)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), response)
}

// TestAddSubscribedPluginByUserID_EmptyUserID tests error when userID is empty
func (suite *UserServiceTestSuite) TestAddSubscribedPluginByUserID_EmptyUserID() {
	pluginID := uuid.New()

	response, err := suite.userService.AddSubscribedPluginByUserID("", pluginID)

	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), response)
	assert.Contains(suite.T(), err.Error(), "user_id is required")
}

// TestAddSubscribedPluginByUserID_NilPluginID tests error when pluginID is nil
func (suite *UserServiceTestSuite) TestAddSubscribedPluginByUserID_NilPluginID() {
	userID := "I123456"

	response, err := suite.userService.AddSubscribedPluginByUserID(userID, uuid.Nil)

	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), response)
	assert.Contains(suite.T(), err.Error(), "plugin_id is required")
}

// TestAddSubscribedPluginByUserID_UserNotFound tests error when user is not found
func (suite *UserServiceTestSuite) TestAddSubscribedPluginByUserID_UserNotFound() {
	userID := "I123456"
	pluginID := uuid.New()

	suite.mockUserRepo.EXPECT().
		GetByUserID(userID).
		Return(nil, apperrors.ErrUserNotFound).
		Times(1)

	response, err := suite.userService.AddSubscribedPluginByUserID(userID, pluginID)

	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), response)
	assert.Contains(suite.T(), err.Error(), "user not found")
}

// TestAddSubscribedPluginByUserID_InvalidMetadata tests handling of invalid metadata JSON
func (suite *UserServiceTestSuite) TestAddSubscribedPluginByUserID_InvalidMetadata() {
	userID := "I123456"
	pluginID := uuid.New()

	existingUser := suite.factories.User.Create()
	existingUser.UserID = userID
	existingUser.Metadata = json.RawMessage(`invalid json`) // Invalid JSON

	suite.mockUserRepo.EXPECT().
		GetByUserID(userID).
		Return(existingUser, nil).
		Times(1)

	suite.mockUserRepo.EXPECT().
		Update(gomock.Any()).
		DoAndReturn(func(user *models.User) error {
			// Should reset to empty object and add subscribed
			var meta map[string]interface{}
			err := json.Unmarshal(user.Metadata, &meta)
			assert.NoError(suite.T(), err)

			subscribed, ok := meta["subscribed"]
			assert.True(suite.T(), ok)

			subArray, ok := subscribed.([]interface{})
			assert.True(suite.T(), ok)
			assert.Len(suite.T(), subArray, 1)
			assert.Equal(suite.T(), pluginID.String(), subArray[0])

			return nil
		}).
		Times(1)

	response, err := suite.userService.AddSubscribedPluginByUserID(userID, pluginID)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), response)
}

// TestAddSubscribedPluginByUserID_UpdateFails tests error when repository update fails
func (suite *UserServiceTestSuite) TestAddSubscribedPluginByUserID_UpdateFails() {
	userID := "I123456"
	pluginID := uuid.New()

	existingUser := suite.factories.User.Create()
	existingUser.UserID = userID
	existingUser.Metadata = nil

	suite.mockUserRepo.EXPECT().
		GetByUserID(userID).
		Return(existingUser, nil).
		Times(1)

	suite.mockUserRepo.EXPECT().
		Update(gomock.Any()).
		Return(gorm.ErrInvalidDB).
		Times(1)

	response, err := suite.userService.AddSubscribedPluginByUserID(userID, pluginID)

	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), response)
	assert.Contains(suite.T(), err.Error(), "failed to update user")
}

// TestRemoveSubscribedPluginByUserID_Success tests successfully removing a subscribed plugin from a user
func (suite *UserServiceTestSuite) TestRemoveSubscribedPluginByUserID_Success() {
	userID := "I123456"
	pluginToRemove := uuid.New()
	pluginToKeep := uuid.New()

	existingMetadata := map[string]interface{}{
		"subscribed": []string{pluginToRemove.String(), pluginToKeep.String()},
	}
	metadataBytes, _ := json.Marshal(existingMetadata)

	existingUser := suite.factories.User.Create()
	existingUser.UserID = userID
	existingUser.Metadata = json.RawMessage(metadataBytes)

	suite.mockUserRepo.EXPECT().
		GetByUserID(userID).
		Return(existingUser, nil).
		Times(1)

	suite.mockUserRepo.EXPECT().
		Update(gomock.Any()).
		DoAndReturn(func(user *models.User) error {
			// Verify metadata was updated correctly
			assert.NotNil(suite.T(), user.Metadata)
			var meta map[string]interface{}
			err := json.Unmarshal(user.Metadata, &meta)
			assert.NoError(suite.T(), err)

			subscribed, ok := meta["subscribed"]
			assert.True(suite.T(), ok)

			subArray, ok := subscribed.([]interface{})
			assert.True(suite.T(), ok)
			assert.Len(suite.T(), subArray, 1)
			assert.Equal(suite.T(), pluginToKeep.String(), subArray[0])
			// Verify removed plugin is not present
			assert.NotContains(suite.T(), subArray, pluginToRemove.String())

			return nil
		}).
		Times(1)

	response, err := suite.userService.RemoveSubscribedPluginByUserID(userID, pluginToRemove)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), response)
	assert.Equal(suite.T(), userID, response.ID)
}

// TestRemoveSubscribedPluginByUserID_RemoveLastPlugin tests removing the last subscribed plugin
func (suite *UserServiceTestSuite) TestRemoveSubscribedPluginByUserID_RemoveLastPlugin() {
	userID := "I123456"
	pluginID := uuid.New()

	existingMetadata := map[string]interface{}{
		"subscribed": []string{pluginID.String()},
	}
	metadataBytes, _ := json.Marshal(existingMetadata)

	existingUser := suite.factories.User.Create()
	existingUser.UserID = userID
	existingUser.Metadata = json.RawMessage(metadataBytes)

	suite.mockUserRepo.EXPECT().
		GetByUserID(userID).
		Return(existingUser, nil).
		Times(1)

	suite.mockUserRepo.EXPECT().
		Update(gomock.Any()).
		DoAndReturn(func(user *models.User) error {
			var meta map[string]interface{}
			err := json.Unmarshal(user.Metadata, &meta)
			assert.NoError(suite.T(), err)

			subscribed, ok := meta["subscribed"]
			assert.True(suite.T(), ok)

			subArray, ok := subscribed.([]interface{})
			assert.True(suite.T(), ok)
			// Should be empty array after removing last plugin
			assert.Len(suite.T(), subArray, 0)

			return nil
		}).
		Times(1)

	response, err := suite.userService.RemoveSubscribedPluginByUserID(userID, pluginID)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), response)
}

// TestRemoveSubscribedPluginByUserID_NoExistingSubscribed tests removing a plugin when no subscribed exist
func (suite *UserServiceTestSuite) TestRemoveSubscribedPluginByUserID_NoExistingSubscribed() {
	userID := "I123456"
	pluginID := uuid.New()

	existingMetadata := map[string]interface{}{
		"portal_admin": true,
		"favorites":    []string{uuid.New().String()},
	}
	metadataBytes, _ := json.Marshal(existingMetadata)

	existingUser := suite.factories.User.Create()
	existingUser.UserID = userID
	existingUser.Metadata = json.RawMessage(metadataBytes)

	suite.mockUserRepo.EXPECT().
		GetByUserID(userID).
		Return(existingUser, nil).
		Times(1)

	suite.mockUserRepo.EXPECT().
		Update(gomock.Any()).
		DoAndReturn(func(user *models.User) error {
			var meta map[string]interface{}
			err := json.Unmarshal(user.Metadata, &meta)
			assert.NoError(suite.T(), err)

			// Verify other metadata is preserved
			assert.Equal(suite.T(), true, meta["portal_admin"])
			assert.NotNil(suite.T(), meta["favorites"])

			subscribed, ok := meta["subscribed"]
			assert.True(suite.T(), ok)

			subArray, ok := subscribed.([]interface{})
			assert.True(suite.T(), ok)
			// Should be empty array
			assert.Len(suite.T(), subArray, 0)

			return nil
		}).
		Times(1)

	response, err := suite.userService.RemoveSubscribedPluginByUserID(userID, pluginID)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), response)
}

// TestRemoveSubscribedPluginByUserID_NoMetadata tests removing a plugin when user has no metadata
func (suite *UserServiceTestSuite) TestRemoveSubscribedPluginByUserID_NoMetadata() {
	userID := "I123456"
	pluginID := uuid.New()

	existingUser := suite.factories.User.Create()
	existingUser.UserID = userID
	existingUser.Metadata = nil // No metadata

	suite.mockUserRepo.EXPECT().
		GetByUserID(userID).
		Return(existingUser, nil).
		Times(1)

	suite.mockUserRepo.EXPECT().
		Update(gomock.Any()).
		DoAndReturn(func(user *models.User) error {
			var meta map[string]interface{}
			err := json.Unmarshal(user.Metadata, &meta)
			assert.NoError(suite.T(), err)

			subscribed, ok := meta["subscribed"]
			assert.True(suite.T(), ok)

			subArray, ok := subscribed.([]interface{})
			assert.True(suite.T(), ok)
			// Should be empty array
			assert.Len(suite.T(), subArray, 0)

			return nil
		}).
		Times(1)

	response, err := suite.userService.RemoveSubscribedPluginByUserID(userID, pluginID)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), response)
}

// TestRemoveSubscribedPluginByUserID_Idempotent tests that removing a non-existent plugin is idempotent
func (suite *UserServiceTestSuite) TestRemoveSubscribedPluginByUserID_Idempotent() {
	userID := "I123456"
	existingPluginID := uuid.New()
	nonExistentPluginID := uuid.New()

	existingMetadata := map[string]interface{}{
		"subscribed": []string{existingPluginID.String()},
	}
	metadataBytes, _ := json.Marshal(existingMetadata)

	existingUser := suite.factories.User.Create()
	existingUser.UserID = userID
	existingUser.Metadata = json.RawMessage(metadataBytes)

	suite.mockUserRepo.EXPECT().
		GetByUserID(userID).
		Return(existingUser, nil).
		Times(1)

	suite.mockUserRepo.EXPECT().
		Update(gomock.Any()).
		DoAndReturn(func(user *models.User) error {
			var meta map[string]interface{}
			err := json.Unmarshal(user.Metadata, &meta)
			assert.NoError(suite.T(), err)

			subscribed, ok := meta["subscribed"]
			assert.True(suite.T(), ok)

			subArray, ok := subscribed.([]interface{})
			assert.True(suite.T(), ok)
			// Should still have the existing plugin
			assert.Len(suite.T(), subArray, 1)
			assert.Equal(suite.T(), existingPluginID.String(), subArray[0])

			return nil
		}).
		Times(1)

	response, err := suite.userService.RemoveSubscribedPluginByUserID(userID, nonExistentPluginID)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), response)
}

// TestRemoveSubscribedPluginByUserID_EmptyUserID tests error when userID is empty
func (suite *UserServiceTestSuite) TestRemoveSubscribedPluginByUserID_EmptyUserID() {
	pluginID := uuid.New()

	response, err := suite.userService.RemoveSubscribedPluginByUserID("", pluginID)

	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), response)
	assert.Contains(suite.T(), err.Error(), "user_id is required")
}

// TestRemoveSubscribedPluginByUserID_NilPluginID tests error when pluginID is nil
func (suite *UserServiceTestSuite) TestRemoveSubscribedPluginByUserID_NilPluginID() {
	userID := "I123456"

	response, err := suite.userService.RemoveSubscribedPluginByUserID(userID, uuid.Nil)

	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), response)
	assert.Contains(suite.T(), err.Error(), "plugin_id is required")
}

// TestRemoveSubscribedPluginByUserID_UserNotFound tests error when user is not found
func (suite *UserServiceTestSuite) TestRemoveSubscribedPluginByUserID_UserNotFound() {
	userID := "I123456"
	pluginID := uuid.New()

	suite.mockUserRepo.EXPECT().
		GetByUserID(userID).
		Return(nil, apperrors.ErrUserNotFound).
		Times(1)

	response, err := suite.userService.RemoveSubscribedPluginByUserID(userID, pluginID)

	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), response)
	assert.Contains(suite.T(), err.Error(), "user not found")
}

// TestRemoveSubscribedPluginByUserID_InvalidMetadata tests handling of invalid metadata JSON
func (suite *UserServiceTestSuite) TestRemoveSubscribedPluginByUserID_InvalidMetadata() {
	userID := "I123456"
	pluginID := uuid.New()

	existingUser := suite.factories.User.Create()
	existingUser.UserID = userID
	existingUser.Metadata = json.RawMessage(`invalid json`) // Invalid JSON

	suite.mockUserRepo.EXPECT().
		GetByUserID(userID).
		Return(existingUser, nil).
		Times(1)

	suite.mockUserRepo.EXPECT().
		Update(gomock.Any()).
		DoAndReturn(func(user *models.User) error {
			// Should reset to empty object and create empty subscribed array
			var meta map[string]interface{}
			err := json.Unmarshal(user.Metadata, &meta)
			assert.NoError(suite.T(), err)

			subscribed, ok := meta["subscribed"]
			assert.True(suite.T(), ok)

			subArray, ok := subscribed.([]interface{})
			assert.True(suite.T(), ok)
			assert.Len(suite.T(), subArray, 0)

			return nil
		}).
		Times(1)

	response, err := suite.userService.RemoveSubscribedPluginByUserID(userID, pluginID)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), response)
}

// TestRemoveSubscribedPluginByUserID_UpdateFails tests error when repository update fails
func (suite *UserServiceTestSuite) TestRemoveSubscribedPluginByUserID_UpdateFails() {
	userID := "I123456"
	pluginID := uuid.New()

	existingMetadata := map[string]interface{}{
		"subscribed": []string{pluginID.String()},
	}
	metadataBytes, _ := json.Marshal(existingMetadata)

	existingUser := suite.factories.User.Create()
	existingUser.UserID = userID
	existingUser.Metadata = json.RawMessage(metadataBytes)

	suite.mockUserRepo.EXPECT().
		GetByUserID(userID).
		Return(existingUser, nil).
		Times(1)

	suite.mockUserRepo.EXPECT().
		Update(gomock.Any()).
		Return(gorm.ErrInvalidDB).
		Times(1)

	response, err := suite.userService.RemoveSubscribedPluginByUserID(userID, pluginID)

	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), response)
	assert.Contains(suite.T(), err.Error(), "failed to update user")
}

// TestRemoveSubscribedPluginByUserID_SubscribedAsInterfaceArray tests handling subscribed as []interface{} type
func (suite *UserServiceTestSuite) TestRemoveSubscribedPluginByUserID_SubscribedAsInterfaceArray() {
	userID := "I123456"
	pluginToRemove := uuid.New()
	pluginToKeep := uuid.New()

	// Create metadata with subscribed as []interface{}
	existingMetadata := map[string]interface{}{
		"subscribed": []interface{}{pluginToRemove.String(), pluginToKeep.String()},
	}
	metadataBytes, _ := json.Marshal(existingMetadata)

	existingUser := suite.factories.User.Create()
	existingUser.UserID = userID
	existingUser.Metadata = json.RawMessage(metadataBytes)

	suite.mockUserRepo.EXPECT().
		GetByUserID(userID).
		Return(existingUser, nil).
		Times(1)

	suite.mockUserRepo.EXPECT().
		Update(gomock.Any()).
		DoAndReturn(func(user *models.User) error {
			var meta map[string]interface{}
			err := json.Unmarshal(user.Metadata, &meta)
			assert.NoError(suite.T(), err)

			subscribed, ok := meta["subscribed"]
			assert.True(suite.T(), ok)

			subArray, ok := subscribed.([]interface{})
			assert.True(suite.T(), ok)
			assert.Len(suite.T(), subArray, 1)
			assert.Equal(suite.T(), pluginToKeep.String(), subArray[0])

			return nil
		}).
		Times(1)

	response, err := suite.userService.RemoveSubscribedPluginByUserID(userID, pluginToRemove)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), response)
}

// TestRemoveSubscribedPluginByUserID_PreservesOtherMetadata tests that removing a plugin preserves other metadata fields
func (suite *UserServiceTestSuite) TestRemoveSubscribedPluginByUserID_PreservesOtherMetadata() {
	userID := "I123456"
	pluginID := uuid.New()

	existingMetadata := map[string]interface{}{
		"subscribed":   []string{pluginID.String()},
		"portal_admin": true,
		"favorites":    []string{uuid.New().String()},
		"custom_field": "custom_value",
	}
	metadataBytes, _ := json.Marshal(existingMetadata)

	existingUser := suite.factories.User.Create()
	existingUser.UserID = userID
	existingUser.Metadata = json.RawMessage(metadataBytes)

	suite.mockUserRepo.EXPECT().
		GetByUserID(userID).
		Return(existingUser, nil).
		Times(1)

	suite.mockUserRepo.EXPECT().
		Update(gomock.Any()).
		DoAndReturn(func(user *models.User) error {
			var meta map[string]interface{}
			err := json.Unmarshal(user.Metadata, &meta)
			assert.NoError(suite.T(), err)

			// Verify all other fields are preserved
			assert.Equal(suite.T(), true, meta["portal_admin"])
			assert.NotNil(suite.T(), meta["favorites"])
			assert.Equal(suite.T(), "custom_value", meta["custom_field"])

			// Verify subscribed is now empty
			subscribed, ok := meta["subscribed"]
			assert.True(suite.T(), ok)
			subArray, ok := subscribed.([]interface{})
			assert.True(suite.T(), ok)
			assert.Len(suite.T(), subArray, 0)

			return nil
		}).
		Times(1)

	response, err := suite.userService.RemoveSubscribedPluginByUserID(userID, pluginID)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), response)
}

// TestGetUserByUserID_Success tests successfully getting a user by their string UserID
func (suite *UserServiceTestSuite) TestGetUserByUserID_Success() {
	userID := "I123456"
	teamID := uuid.New()

	existingUser := suite.factories.User.Create()
	existingUser.TeamID = &teamID
	existingUser.UserID = userID
	existingUser.Mobile = "+1-555-0123"

	suite.mockUserRepo.EXPECT().
		GetByUserID(userID).
		Return(existingUser, nil).
		Times(1)

	response, err := suite.userService.GetUserByUserID(userID)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), response)
	assert.Equal(suite.T(), userID, response.ID)
	assert.Equal(suite.T(), existingUser.ID.String(), response.UUID)
	assert.Equal(suite.T(), existingUser.FirstName, response.FirstName)
	assert.Equal(suite.T(), existingUser.LastName, response.LastName)
	assert.Equal(suite.T(), existingUser.Email, response.Email)
	assert.Equal(suite.T(), existingUser.Mobile, response.Mobile)
	assert.Equal(suite.T(), string(existingUser.TeamDomain), response.TeamDomain)
	assert.Equal(suite.T(), string(existingUser.TeamRole), response.TeamRole)
}

// TestGetUserByUserID_EmptyUserID tests error when userID is empty
func (suite *UserServiceTestSuite) TestGetUserByUserID_EmptyUserID() {
	response, err := suite.userService.GetUserByUserID("")

	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), response)
	assert.Contains(suite.T(), err.Error(), "user_id is required")
}

// TestGetUserByUserID_UserNotFound tests error when user is not found
func (suite *UserServiceTestSuite) TestGetUserByUserID_UserNotFound() {
	userID := "I999999"

	suite.mockUserRepo.EXPECT().
		GetByUserID(userID).
		Return(nil, apperrors.ErrUserNotFound).
		Times(1)

	response, err := suite.userService.GetUserByUserID(userID)

	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), response)
	assert.Contains(suite.T(), err.Error(), "user not found")
}

// TestGetUserByName_Success tests successfully getting a user by their name
func (suite *UserServiceTestSuite) TestGetUserByName_Success() {
	name := "John Doe"
	userID := "I123456"

	existingUser := suite.factories.User.Create()
	existingUser.Name = name
	existingUser.UserID = userID
	existingUser.Mobile = "+1-555-0123"

	suite.mockUserRepo.EXPECT().
		GetByName(name).
		Return(existingUser, nil).
		Times(1)

	response, err := suite.userService.GetUserByName(name)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), response)
	assert.Equal(suite.T(), userID, response.ID)
	assert.Equal(suite.T(), "John", response.FirstName)
	assert.Equal(suite.T(), "Doe", response.LastName)
	assert.Equal(suite.T(), existingUser.Email, response.Email)
}

// TestGetUserByName_EmptyName tests error when name is empty
func (suite *UserServiceTestSuite) TestGetUserByName_EmptyName() {
	response, err := suite.userService.GetUserByName("")

	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), response)
	assert.Contains(suite.T(), err.Error(), "name is required")
}

// TestGetUserByName_UserNotFound tests error when user is not found
func (suite *UserServiceTestSuite) TestGetUserByName_UserNotFound() {
	name := "NonExistent User"

	suite.mockUserRepo.EXPECT().
		GetByName(name).
		Return(nil, apperrors.ErrUserNotFound).
		Times(1)

	response, err := suite.userService.GetUserByName(name)

	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), response)
	assert.Contains(suite.T(), err.Error(), "user not found")
}

// TestGetUserByName_TrimsWhitespace tests that leading/trailing whitespace is trimmed
func (suite *UserServiceTestSuite) TestGetUserByName_TrimsWhitespace() {
	name := "  John Doe  "
	trimmedName := "John Doe"
	userID := "I123456"

	existingUser := suite.factories.User.Create()
	existingUser.Name = trimmedName
	existingUser.UserID = userID

	suite.mockUserRepo.EXPECT().
		GetByName(trimmedName).
		Return(existingUser, nil).
		Times(1)

	response, err := suite.userService.GetUserByName(name)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), response)
	assert.Equal(suite.T(), userID, response.ID)
}

// TestGetUserByNameWithLinks_Success tests successfully getting a user with links by name nil metadate
func (suite *UserServiceTestSuite) TestGetUserByNameWithLinks_SuccessNilMetaData() {
	name := "John Doe"
	userID := "I123456"
	userUUID := uuid.New()

	existingUser := suite.factories.User.Create()
	existingUser.ID = userUUID
	existingUser.Name = name
	existingUser.UserID = userID
	existingUser.Metadata = nil

	// First call to GetByName
	suite.mockUserRepo.EXPECT().
		GetByName(name).
		Return(existingUser, nil).
		Times(1)

	// Second call to GetByUserID (from GetUserByUserIDWithLinks)
	suite.mockUserRepo.EXPECT().
		GetByUserID(userID).
		Return(existingUser, nil).
		Times(1)

	// Mock link repository calls
	suite.mockLinkRepo.EXPECT().
		GetByIDs(gomock.Any()).
		Return([]models.Link{}, nil).
		Times(1)

	suite.mockLinkRepo.EXPECT().
		GetByOwner(userUUID).
		Return([]models.Link{}, nil).
		Times(1)

	response, err := suite.userService.GetUserByNameWithLinks(name)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), response)
	assert.Equal(suite.T(), userID, response.ID)
	assert.Equal(suite.T(), "John", response.FirstName)
	assert.Equal(suite.T(), "Doe", response.LastName)
	assert.NotNil(suite.T(), response.Links)
	assert.NotNil(suite.T(), response.Plugins)
}

// TestGetUserByNameWithLinks_EmptyName tests error when name is empty
func (suite *UserServiceTestSuite) TestGetUserByNameWithLinks_EmptyName() {
	response, err := suite.userService.GetUserByNameWithLinks("")

	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), response)
	assert.Contains(suite.T(), err.Error(), "name is required")
}

// TestGetUserByNameWithLinks_UserNotFound tests error when user is not found
func (suite *UserServiceTestSuite) TestGetUserByNameWithLinks_UserNotFound() {
	name := "NonExistent User"

	suite.mockUserRepo.EXPECT().
		GetByName(name).
		Return(nil, apperrors.ErrUserNotFound).
		Times(1)

	response, err := suite.userService.GetUserByNameWithLinks(name)

	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), response)
	assert.Contains(suite.T(), err.Error(), "user not found")
}

// TestGetUserByNameWithLinks_WithFavorites tests getting a user with favorite links in metadata
func (suite *UserServiceTestSuite) TestGetUserByNameWithLinks_WithFavorites() {
	name := "John Doe"
	userID := "I123456"
	userUUID := uuid.New()
	linkID1 := uuid.New()
	linkID2 := uuid.New()

	metadata := map[string]interface{}{
		"favorites":    []string{linkID1.String(), linkID2.String()},
		"portal_admin": "portal-test",
	}
	metadataBytes, _ := json.Marshal(metadata)

	existingUser := suite.factories.User.Create()
	existingUser.ID = userUUID
	existingUser.Name = name
	existingUser.UserID = userID
	existingUser.Metadata = json.RawMessage(metadataBytes)

	link1 := models.Link{
		BaseModel: models.BaseModel{
			ID:    linkID1,
			Name:  "link1",
			Title: "Link 1",
		},
		URL: "https://example.com/1",
	}

	link2 := models.Link{
		BaseModel: models.BaseModel{
			ID:    linkID2,
			Name:  "link2",
			Title: "Link 2",
		},
		URL: "https://example.com/2",
	}

	// First call to GetByName
	suite.mockUserRepo.EXPECT().
		GetByName(name).
		Return(existingUser, nil).
		Times(1)

	// Second call to GetByUserID (from GetUserByUserIDWithLinks)
	suite.mockUserRepo.EXPECT().
		GetByUserID(userID).
		Return(existingUser, nil).
		Times(1)

	// Mock link repository calls
	suite.mockLinkRepo.EXPECT().
		GetByIDs(gomock.Any()).
		DoAndReturn(func(ids []uuid.UUID) ([]models.Link, error) {
			// Verify the IDs passed
			assert.Len(suite.T(), ids, 2)
			assert.Contains(suite.T(), ids, linkID1)
			assert.Contains(suite.T(), ids, linkID2)
			return []models.Link{link1, link2}, nil
		}).
		Times(1)

	suite.mockLinkRepo.EXPECT().
		GetByOwner(userUUID).
		Return([]models.Link{}, nil).
		Times(1)

	response, err := suite.userService.GetUserByNameWithLinks(name)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), response)
	assert.Equal(suite.T(), userID, response.ID)
	assert.NotNil(suite.T(), response.Links)
	assert.Len(suite.T(), response.Links, 2)

	// Verify links are marked as favorites and have correct IDs
	linkIDs := make(map[string]bool)
	for _, link := range response.Links {
		assert.True(suite.T(), link.Favorite)
		linkIDs[link.ID] = true
	}
	assert.True(suite.T(), linkIDs[linkID1.String()])
	assert.True(suite.T(), linkIDs[linkID2.String()])
}

// TestGetUserByNameWithLinks_WithPortalAdmin tests getting a user with portal_admin in metadata
func (suite *UserServiceTestSuite) TestGetUserByNameWithLinks_WithPortalAdmin() {
	name := "Admin User"
	userID := "I123456"
	userUUID := uuid.New()

	metadata := map[string]interface{}{
		"portal_admin": 1,
	}
	metadataBytes, _ := json.Marshal(metadata)

	existingUser := suite.factories.User.Create()
	existingUser.ID = userUUID
	existingUser.Name = name
	existingUser.UserID = userID
	existingUser.FirstName = "Admin"
	existingUser.LastName = "User"
	existingUser.Email = "admin@example.com"
	existingUser.Metadata = json.RawMessage(metadataBytes)

	// First call to GetByName
	suite.mockUserRepo.EXPECT().
		GetByName(name).
		Return(existingUser, nil).
		Times(1)

	// Second call to GetByUserID (from GetUserByUserIDWithLinks)
	suite.mockUserRepo.EXPECT().
		GetByUserID(userID).
		Return(existingUser, nil).
		Times(1)

	// Mock link repository calls
	suite.mockLinkRepo.EXPECT().
		GetByIDs(gomock.Any()).
		Return([]models.Link{}, nil).
		Times(1)

	suite.mockLinkRepo.EXPECT().
		GetByOwner(userUUID).
		Return([]models.Link{}, nil).
		Times(1)

	response, err := suite.userService.GetUserByNameWithLinks(name)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), response)
	assert.Equal(suite.T(), userID, response.ID)
	assert.True(suite.T(), response.PortalAdmin)
}

// ===== Tests for GetSubscribedPluginsFromUser =====

// TestGetSubscribedPluginsFromUser_NoMetadata tests getting subscribed plugins when user has no metadata
func (suite *UserServiceTestSuite) TestGetSubscribedPluginsFromUser_NoMetadata() {
	user := suite.factories.User.Create()
	user.UserID = "I123456"
	user.Metadata = nil

	plugins := suite.userService.GetSubscribedPluginsFromUser(user)

	assert.NotNil(suite.T(), plugins)
	assert.Len(suite.T(), plugins, 0)
}

// TestGetSubscribedPluginsFromUser_EmptyMetadata tests getting subscribed plugins when user has empty metadata
func (suite *UserServiceTestSuite) TestGetSubscribedPluginsFromUser_EmptyMetadata() {
	user := suite.factories.User.Create()
	user.UserID = "I123456"
	user.Metadata = json.RawMessage(`{}`)

	plugins := suite.userService.GetSubscribedPluginsFromUser(user)

	assert.NotNil(suite.T(), plugins)
	assert.Len(suite.T(), plugins, 0)
}

// TestGetSubscribedPluginsFromUser_InvalidJSON tests getting subscribed plugins when metadata is invalid JSON
func (suite *UserServiceTestSuite) TestGetSubscribedPluginsFromUser_InvalidJSON() {
	user := suite.factories.User.Create()
	user.UserID = "I123456"
	user.Metadata = json.RawMessage(`invalid json`)

	plugins := suite.userService.GetSubscribedPluginsFromUser(user)

	assert.NotNil(suite.T(), plugins)
	assert.Len(suite.T(), plugins, 0)
}

// TestGetSubscribedPluginsFromUser_EmptySubscribedArray tests getting subscribed plugins when subscribed array is empty
func (suite *UserServiceTestSuite) TestGetSubscribedPluginsFromUser_EmptySubscribedArray() {
	metadata := map[string]interface{}{
		"subscribed": []string{},
	}
	metadataBytes, _ := json.Marshal(metadata)

	user := suite.factories.User.Create()
	user.UserID = "I123456"
	user.Metadata = json.RawMessage(metadataBytes)

	plugins := suite.userService.GetSubscribedPluginsFromUser(user)

	assert.NotNil(suite.T(), plugins)
	assert.Len(suite.T(), plugins, 0)
}

// TestGetSubscribedPluginsFromUser_SinglePlugin tests getting a single subscribed plugin
func (suite *UserServiceTestSuite) TestGetSubscribedPluginsFromUser_SinglePlugin() {
	pluginID := uuid.New()
	metadata := map[string]interface{}{
		"subscribed": []string{pluginID.String()},
	}
	metadataBytes, _ := json.Marshal(metadata)

	user := suite.factories.User.Create()
	user.UserID = "I123456"
	user.Metadata = json.RawMessage(metadataBytes)

	plugin := &models.Plugin{
		BaseModel: models.BaseModel{
			ID:          pluginID,
			Name:        "test-plugin",
			Title:       "Test Plugin",
			Description: "A test plugin",
		},
		Icon:               "test-icon",
		ReactComponentPath: "/path/to/component.tsx",
		BackendServerURL:   "https://backend.example.com",
		Owner:              "test-owner",
	}

	suite.mockPluginRepo.EXPECT().
		GetByID(pluginID).
		Return(plugin, nil).
		Times(1)

	plugins := suite.userService.GetSubscribedPluginsFromUser(user)

	assert.NotNil(suite.T(), plugins)
	assert.Len(suite.T(), plugins, 1)
	assert.Equal(suite.T(), pluginID, plugins[0].ID)
	assert.Equal(suite.T(), "test-plugin", plugins[0].Name)
	assert.Equal(suite.T(), "Test Plugin", plugins[0].Title)
	assert.Equal(suite.T(), "A test plugin", plugins[0].Description)
	assert.Equal(suite.T(), "test-icon", plugins[0].Icon)
	assert.Equal(suite.T(), "/path/to/component.tsx", plugins[0].ReactComponentPath)
	assert.Equal(suite.T(), "https://backend.example.com", plugins[0].BackendServerURL)
	assert.Equal(suite.T(), "test-owner", plugins[0].Owner)
}

// TestGetSubscribedPluginsFromUser_MultiplePlugins tests getting multiple subscribed plugins
func (suite *UserServiceTestSuite) TestGetSubscribedPluginsFromUser_MultiplePlugins() {
	pluginID1 := uuid.New()
	pluginID2 := uuid.New()
	pluginID3 := uuid.New()

	metadata := map[string]interface{}{
		"subscribed": []string{pluginID1.String(), pluginID2.String(), pluginID3.String()},
	}
	metadataBytes, _ := json.Marshal(metadata)

	user := suite.factories.User.Create()
	user.UserID = "I123456"
	user.Metadata = json.RawMessage(metadataBytes)

	plugin1 := &models.Plugin{
		BaseModel: models.BaseModel{
			ID:          pluginID1,
			Name:        "plugin-1",
			Title:       "Plugin 1",
			Description: "First plugin",
		},
		Icon:               "icon-1",
		ReactComponentPath: "/path/to/plugin1.tsx",
		BackendServerURL:   "https://backend1.example.com",
		Owner:              "owner-1",
	}

	plugin2 := &models.Plugin{
		BaseModel: models.BaseModel{
			ID:          pluginID2,
			Name:        "plugin-2",
			Title:       "Plugin 2",
			Description: "Second plugin",
		},
		Icon:               "icon-2",
		ReactComponentPath: "/path/to/plugin2.tsx",
		BackendServerURL:   "https://backend2.example.com",
		Owner:              "owner-2",
	}

	plugin3 := &models.Plugin{
		BaseModel: models.BaseModel{
			ID:          pluginID3,
			Name:        "plugin-3",
			Title:       "Plugin 3",
			Description: "Third plugin",
		},
		Icon:               "icon-3",
		ReactComponentPath: "/path/to/plugin3.tsx",
		BackendServerURL:   "https://backend3.example.com",
		Owner:              "owner-3",
	}

	suite.mockPluginRepo.EXPECT().
		GetByID(pluginID1).
		Return(plugin1, nil).
		Times(1)

	suite.mockPluginRepo.EXPECT().
		GetByID(pluginID2).
		Return(plugin2, nil).
		Times(1)

	suite.mockPluginRepo.EXPECT().
		GetByID(pluginID3).
		Return(plugin3, nil).
		Times(1)

	plugins := suite.userService.GetSubscribedPluginsFromUser(user)

	assert.NotNil(suite.T(), plugins)
	assert.Len(suite.T(), plugins, 3)

	// Verify all plugins are returned with correct data
	pluginMap := make(map[uuid.UUID]service.PluginResponse)
	for _, p := range plugins {
		pluginMap[p.ID] = p
	}

	// Verify plugin IDs match the ones defined at the beginning
	assert.Contains(suite.T(), pluginMap, pluginID1)
	assert.Equal(suite.T(), pluginID1, pluginMap[pluginID1].ID)
	assert.Equal(suite.T(), "plugin-1", pluginMap[pluginID1].Name)

	assert.Contains(suite.T(), pluginMap, pluginID2)
	assert.Equal(suite.T(), pluginID2, pluginMap[pluginID2].ID)
	assert.Equal(suite.T(), "plugin-2", pluginMap[pluginID2].Name)

	assert.Contains(suite.T(), pluginMap, pluginID3)
	assert.Equal(suite.T(), pluginID3, pluginMap[pluginID3].ID)
	assert.Equal(suite.T(), "plugin-3", pluginMap[pluginID3].Name)
}

// TestGetSubscribedPluginsFromUser_PluginNotFound tests when a subscribed plugin is not found in database
func (suite *UserServiceTestSuite) TestGetSubscribedPluginsFromUser_PluginNotFound() {
	pluginID1 := uuid.New()
	pluginID2 := uuid.New() // This one won't be found

	metadata := map[string]interface{}{
		"subscribed": []string{pluginID1.String(), pluginID2.String()},
	}
	metadataBytes, _ := json.Marshal(metadata)

	user := suite.factories.User.Create()
	user.UserID = "I123456"
	user.Metadata = json.RawMessage(metadataBytes)

	plugin1 := &models.Plugin{
		BaseModel: models.BaseModel{
			ID:          pluginID1,
			Name:        "plugin-1",
			Title:       "Plugin 1",
			Description: "First plugin",
		},
		Icon:               "icon-1",
		ReactComponentPath: "/path/to/plugin1.tsx",
		BackendServerURL:   "https://backend1.example.com",
		Owner:              "owner-1",
	}

	suite.mockPluginRepo.EXPECT().
		GetByID(pluginID1).
		Return(plugin1, nil).
		Times(1)

	suite.mockPluginRepo.EXPECT().
		GetByID(pluginID2).
		Return(nil, gorm.ErrRecordNotFound).
		Times(1)

	plugins := suite.userService.GetSubscribedPluginsFromUser(user)

	assert.NotNil(suite.T(), plugins)
	// Only the found plugin should be returned
	assert.Len(suite.T(), plugins, 1)
	assert.Equal(suite.T(), pluginID1, plugins[0].ID)
	assert.Equal(suite.T(), "plugin-1", plugins[0].Name)
}

// TestGetUserByUserIDWithPlugins_Success tests successfully getting plugins for a user
func (suite *UserServiceTestSuite) TestGetUserByUserIDWithPlugins_Success() {
	userID := "I123456"
	pluginID1 := uuid.New()
	pluginID2 := uuid.New()

	metadata := map[string]interface{}{
		"subscribed": []string{pluginID1.String(), pluginID2.String()},
	}
	metadataBytes, _ := json.Marshal(metadata)

	user := suite.factories.User.Create()
	user.UserID = userID
	user.Metadata = json.RawMessage(metadataBytes)

	plugin1 := &models.Plugin{
		BaseModel: models.BaseModel{
			ID:          pluginID1,
			Name:        "plugin-1",
			Title:       "Plugin 1",
			Description: "First plugin",
		},
		Icon:               "icon-1",
		ReactComponentPath: "/path/to/plugin1.tsx",
		BackendServerURL:   "https://backend1.example.com",
		Owner:              "owner-1",
	}

	plugin2 := &models.Plugin{
		BaseModel: models.BaseModel{
			ID:          pluginID2,
			Name:        "plugin-2",
			Title:       "Plugin 2",
			Description: "Second plugin",
		},
		Icon:               "icon-2",
		ReactComponentPath: "/path/to/plugin2.tsx",
		BackendServerURL:   "https://backend2.example.com",
		Owner:              "owner-2",
	}

	suite.mockUserRepo.EXPECT().
		GetByUserID(userID).
		Return(user, nil).
		Times(1)

	suite.mockPluginRepo.EXPECT().
		GetByID(pluginID1).
		Return(plugin1, nil).
		Times(1)

	suite.mockPluginRepo.EXPECT().
		GetByID(pluginID2).
		Return(plugin2, nil).
		Times(1)

	plugins, err := suite.userService.GetUserByUserIDWithPlugins(userID)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), plugins)
	assert.Len(suite.T(), plugins, 2)

	// Verify plugins are returned with correct data
	pluginMap := make(map[uuid.UUID]service.PluginResponse)
	for _, p := range plugins {
		pluginMap[p.ID] = p
	}

	assert.Contains(suite.T(), pluginMap, pluginID1)
	assert.Equal(suite.T(), "plugin-1", pluginMap[pluginID1].Name)
	assert.Equal(suite.T(), "Plugin 1", pluginMap[pluginID1].Title)

	assert.Contains(suite.T(), pluginMap, pluginID2)
	assert.Equal(suite.T(), "plugin-2", pluginMap[pluginID2].Name)
	assert.Equal(suite.T(), "Plugin 2", pluginMap[pluginID2].Title)
}

// TestGetUserByUserIDWithPlugins_EmptyUserID tests error when userID is empty
func (suite *UserServiceTestSuite) TestGetUserByUserIDWithPlugins_EmptyUserID() {
	plugins, err := suite.userService.GetUserByUserIDWithPlugins("")

	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), plugins)
	assert.Contains(suite.T(), err.Error(), "user_id is required")
}

// TestGetUserByUserIDWithPlugins_UserNotFound tests error when user is not found
func (suite *UserServiceTestSuite) TestGetUserByUserIDWithPlugins_UserNotFound() {
	userID := "I999999"

	suite.mockUserRepo.EXPECT().
		GetByUserID(userID).
		Return(nil, apperrors.ErrUserNotFound).
		Times(1)

	plugins, err := suite.userService.GetUserByUserIDWithPlugins(userID)

	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), plugins)
	assert.Contains(suite.T(), err.Error(), "user not found")
}

// TestGetUserByNameWithLinks_WithSubscribed tests getting a user with subscribed plugins in metadata and both favorite and owned links
func (suite *UserServiceTestSuite) TestGetUserByNameWithLinks_WithSubscribed() {
	name := "John Doe"
	userID := "I123456"
	userUUID := uuid.New()
	pluginID1 := uuid.New()
	pluginID2 := uuid.New()
	favoriteLinkID1 := uuid.New()
	favoriteLinkID2 := uuid.New()
	ownedLinkID1 := uuid.New()
	ownedLinkID2 := uuid.New()

	metadata := map[string]interface{}{
		"subscribed": []string{pluginID1.String(), pluginID2.String()},
		"favorites":  []string{favoriteLinkID1.String(), favoriteLinkID2.String()},
	}
	metadataBytes, _ := json.Marshal(metadata)

	existingUser := suite.factories.User.Create()
	existingUser.ID = userUUID
	existingUser.Name = name
	existingUser.UserID = userID
	existingUser.Metadata = json.RawMessage(metadataBytes)

	// Favorite links
	favoriteLink1 := models.Link{
		BaseModel: models.BaseModel{
			ID:    favoriteLinkID1,
			Name:  "favorite-link1",
			Title: "Favorite Link 1",
		},
		URL: "https://example.com/fav1",
	}

	favoriteLink2 := models.Link{
		BaseModel: models.BaseModel{
			ID:    favoriteLinkID2,
			Name:  "favorite-link2",
			Title: "Favorite Link 2",
		},
		URL: "https://example.com/fav2",
	}

	// Links owned by the user
	ownedLink1 := models.Link{
		BaseModel: models.BaseModel{
			ID:    ownedLinkID1,
			Name:  "owned-link1",
			Title: "Owned Link 1",
		},
		Owner: userUUID,
		URL:   "https://example.com/owned1",
	}

	ownedLink2 := models.Link{
		BaseModel: models.BaseModel{
			ID:    ownedLinkID2,
			Name:  "owned-link2",
			Title: "Owned Link 2",
		},
		Owner: userUUID,
		URL:   "https://example.com/owned2",
	}

	// First call to GetByName
	suite.mockUserRepo.EXPECT().
		GetByName(name).
		Return(existingUser, nil).
		Times(1)

	// Second call to GetByUserID (from GetUserByUserIDWithLinks)
	suite.mockUserRepo.EXPECT().
		GetByUserID(userID).
		Return(existingUser, nil).
		Times(1)

	// Mock link repository calls - GetByIDs for favorite links
	suite.mockLinkRepo.EXPECT().
		GetByIDs(gomock.Any()).
		DoAndReturn(func(ids []uuid.UUID) ([]models.Link, error) {
			// Verify the IDs passed are the favorite link IDs
			assert.Len(suite.T(), ids, 2)
			assert.Contains(suite.T(), ids, favoriteLinkID1)
			assert.Contains(suite.T(), ids, favoriteLinkID2)
			return []models.Link{favoriteLink1, favoriteLink2}, nil
		}).
		Times(1)

	// Mock GetByOwner for owned links
	suite.mockLinkRepo.EXPECT().
		GetByOwner(userUUID).
		Return([]models.Link{ownedLink1, ownedLink2}, nil).
		Times(1)

	response, err := suite.userService.GetUserByNameWithLinks(name)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), response)
	assert.Equal(suite.T(), userID, response.ID)

	// Verify all links are returned (favorites + owned)
	assert.NotNil(suite.T(), response.Links)
	assert.Len(suite.T(), response.Links, 4) // 2 favorites + 2 owned

	// Verify link IDs and favorite status
	linkIDs := make(map[string]bool)
	favoriteCount := 0
	for _, link := range response.Links {
		linkIDs[link.ID] = true
		if link.Favorite {
			favoriteCount++
		}
	}

	// Verify all link IDs are present
	assert.True(suite.T(), linkIDs[favoriteLinkID1.String()])
	assert.True(suite.T(), linkIDs[favoriteLinkID2.String()])
	assert.True(suite.T(), linkIDs[ownedLinkID1.String()])
	assert.True(suite.T(), linkIDs[ownedLinkID2.String()])

	// Verify only favorite links are marked as favorites
	assert.Equal(suite.T(), 2, favoriteCount)

	// Note: GetUserByUserIDWithLinks returns empty plugins array
	// Plugins are fetched separately via GetUserByUserIDWithPlugins
	assert.NotNil(suite.T(), response.Plugins)
	assert.Len(suite.T(), response.Plugins, 0)
}

// TestGetUserByNameWithLinks_WithAllMetadata tests getting a user with all metadata fields
func (suite *UserServiceTestSuite) TestGetUserByNameWithLinks_WithAllMetadata() {
	name := "John Doe"
	userID := "I123456"
	userUUID := uuid.New()
	linkID := uuid.New()
	pluginID := uuid.New()

	metadata := map[string]interface{}{
		"favorites":    []string{linkID.String()},
		"portal_admin": true,
		"subscribed":   []string{pluginID.String()},
	}
	metadataBytes, _ := json.Marshal(metadata)

	existingUser := suite.factories.User.Create()
	existingUser.ID = userUUID
	existingUser.Name = name
	existingUser.UserID = userID
	existingUser.Metadata = json.RawMessage(metadataBytes)

	link := models.Link{
		BaseModel: models.BaseModel{
			ID:    linkID,
			Name:  "link1",
			Title: "Link 1",
		},
		URL: "https://example.com/1",
	}

	// First call to GetByName
	suite.mockUserRepo.EXPECT().
		GetByName(name).
		Return(existingUser, nil).
		Times(1)

	// Second call to GetByUserID (from GetUserByUserIDWithLinks)
	suite.mockUserRepo.EXPECT().
		GetByUserID(userID).
		Return(existingUser, nil).
		Times(1)

	// Mock link repository calls
	suite.mockLinkRepo.EXPECT().
		GetByIDs(gomock.Any()).
		Return([]models.Link{link}, nil).
		Times(1)

	suite.mockLinkRepo.EXPECT().
		GetByOwner(userUUID).
		Return([]models.Link{}, nil).
		Times(1)

	response, err := suite.userService.GetUserByNameWithLinks(name)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), response)
	assert.Equal(suite.T(), userID, response.ID)
	assert.True(suite.T(), response.PortalAdmin)
	assert.Len(suite.T(), response.Links, 1)
	assert.True(suite.T(), response.Links[0].Favorite)
	assert.NotNil(suite.T(), response.Plugins)
}

// ===== Tests for GetUserByNameWithLinksAndPlugins =====

// TestGetUserByNameWithLinksAndPlugins_Success tests successfully getting a user with both links and plugins
func (suite *UserServiceTestSuite) TestGetUserByNameWithLinksAndPlugins_Success() {
	name := "John Doe"
	userID := "I123456"
	userUUID := uuid.New()
	linkID := uuid.New()
	pluginID1 := uuid.New()
	pluginID2 := uuid.New()

	metadata := map[string]interface{}{
		"favorites":  []string{linkID.String()},
		"subscribed": []string{pluginID1.String(), pluginID2.String()},
	}
	metadataBytes, _ := json.Marshal(metadata)

	existingUser := suite.factories.User.Create()
	existingUser.ID = userUUID
	existingUser.Name = name
	existingUser.UserID = userID
	existingUser.Metadata = json.RawMessage(metadataBytes)

	link := models.Link{
		BaseModel: models.BaseModel{
			ID:    linkID,
			Name:  "link1",
			Title: "Link 1",
		},
		URL: "https://example.com/1",
	}

	plugin1 := &models.Plugin{
		BaseModel: models.BaseModel{
			ID:          pluginID1,
			Name:        "plugin-1",
			Title:       "Plugin 1",
			Description: "First plugin",
		},
		Icon:               "icon-1",
		ReactComponentPath: "/path/to/plugin1.tsx",
		BackendServerURL:   "https://backend1.example.com",
		Owner:              "owner-1",
	}

	plugin2 := &models.Plugin{
		BaseModel: models.BaseModel{
			ID:          pluginID2,
			Name:        "plugin-2",
			Title:       "Plugin 2",
			Description: "Second plugin",
		},
		Icon:               "icon-2",
		ReactComponentPath: "/path/to/plugin2.tsx",
		BackendServerURL:   "https://backend2.example.com",
		Owner:              "owner-2",
	}

	// First call to GetByName
	suite.mockUserRepo.EXPECT().
		GetByName(name).
		Return(existingUser, nil).
		Times(1)

	// Second call to GetByUserID (from GetUserByUserIDWithLinks)
	suite.mockUserRepo.EXPECT().
		GetByUserID(userID).
		Return(existingUser, nil).
		Times(1)

	// Mock link repository calls
	suite.mockLinkRepo.EXPECT().
		GetByIDs(gomock.Any()).
		Return([]models.Link{link}, nil).
		Times(1)

	suite.mockLinkRepo.EXPECT().
		GetByOwner(userUUID).
		Return([]models.Link{}, nil).
		Times(1)

	// Mock plugin repository calls
	suite.mockPluginRepo.EXPECT().
		GetByID(pluginID1).
		Return(plugin1, nil).
		Times(1)

	suite.mockPluginRepo.EXPECT().
		GetByID(pluginID2).
		Return(plugin2, nil).
		Times(1)

	response, err := suite.userService.GetUserByNameWithLinksAndPlugins(name)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), response)
	assert.Equal(suite.T(), userID, response.ID)
	assert.Len(suite.T(), response.Links, 1)
	assert.Len(suite.T(), response.Plugins, 2)
	assert.True(suite.T(), response.Links[0].Favorite)
}

// TestGetUserByNameWithLinksAndPlugins_EmptyName tests error when name is empty
func (suite *UserServiceTestSuite) TestGetUserByNameWithLinksAndPlugins_EmptyName() {
	response, err := suite.userService.GetUserByNameWithLinksAndPlugins("")

	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), response)
	assert.Contains(suite.T(), err.Error(), "name is required")
}

// TestGetUserByNameWithLinksAndPlugins_UserNotFound tests error when user is not found
func (suite *UserServiceTestSuite) TestGetUserByNameWithLinksAndPlugins_UserNotFound() {
	name := "NonExistent User"

	suite.mockUserRepo.EXPECT().
		GetByName(name).
		Return(nil, apperrors.ErrUserNotFound).
		Times(1)

	response, err := suite.userService.GetUserByNameWithLinksAndPlugins(name)

	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), response)
	assert.Contains(suite.T(), err.Error(), "user not found")
}

// TestGetUserByNameWithLinksAndPlugins_NoPlugins tests getting user with links but no plugins
func (suite *UserServiceTestSuite) TestGetUserByNameWithLinksAndPlugins_NoPlugins() {
	name := "John Doe"
	userID := "I123456"
	userUUID := uuid.New()
	linkID := uuid.New()

	metadata := map[string]interface{}{
		"favorites": []string{linkID.String()},
	}
	metadataBytes, _ := json.Marshal(metadata)

	existingUser := suite.factories.User.Create()
	existingUser.ID = userUUID
	existingUser.Name = name
	existingUser.UserID = userID
	existingUser.Metadata = json.RawMessage(metadataBytes)

	link := models.Link{
		BaseModel: models.BaseModel{
			ID:    linkID,
			Name:  "link1",
			Title: "Link 1",
		},
		URL: "https://example.com/1",
	}

	suite.mockUserRepo.EXPECT().
		GetByName(name).
		Return(existingUser, nil).
		Times(1)

	suite.mockUserRepo.EXPECT().
		GetByUserID(userID).
		Return(existingUser, nil).
		Times(1)

	suite.mockLinkRepo.EXPECT().
		GetByIDs(gomock.Any()).
		Return([]models.Link{link}, nil).
		Times(1)

	suite.mockLinkRepo.EXPECT().
		GetByOwner(userUUID).
		Return([]models.Link{}, nil).
		Times(1)

	response, err := suite.userService.GetUserByNameWithLinksAndPlugins(name)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), response)
	assert.Equal(suite.T(), userID, response.ID)
	assert.Len(suite.T(), response.Links, 1)
	assert.Len(suite.T(), response.Plugins, 0)
}

// ===== Tests for GetAllUsers =====

// TestGetAllUsers_Success tests successfully getting all users
func (suite *UserServiceTestSuite) TestGetAllUsers_Success() {
	limit, offset := 20, 0
	users := []models.User{
		{
			UserID:     "I123456",
			FirstName:  "John",
			LastName:   "Doe",
			Email:      "john@example.com",
			TeamDomain: models.TeamDomainDeveloper,
			TeamRole:   models.TeamRoleMember,
		},
		{
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
		GetAll(limit, offset).
		Return(users, expectedTotal, nil).
		Times(1)

	responses, total, err := suite.userService.GetAllUsers(limit, offset)

	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), expectedTotal, total)
	assert.Len(suite.T(), responses, 2)
	assert.Equal(suite.T(), "I123456", responses[0].ID)
	assert.Equal(suite.T(), "John", responses[0].FirstName)
	assert.Equal(suite.T(), "I789012", responses[1].ID)
	assert.Equal(suite.T(), "Jane", responses[1].FirstName)
}

// TestGetAllUsers_EmptyResult tests getting all users when no users exist
func (suite *UserServiceTestSuite) TestGetAllUsers_EmptyResult() {
	limit, offset := 20, 0
	users := []models.User{}
	expectedTotal := int64(0)

	suite.mockUserRepo.EXPECT().
		GetAll(limit, offset).
		Return(users, expectedTotal, nil).
		Times(1)

	responses, total, err := suite.userService.GetAllUsers(limit, offset)

	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), expectedTotal, total)
	assert.Len(suite.T(), responses, 0)
}

// TestGetAllUsers_RepositoryError tests error when repository fails
func (suite *UserServiceTestSuite) TestGetAllUsers_RepositoryError() {
	limit, offset := 20, 0

	suite.mockUserRepo.EXPECT().
		GetAll(limit, offset).
		Return(nil, int64(0), gorm.ErrInvalidDB).
		Times(1)

	responses, total, err := suite.userService.GetAllUsers(limit, offset)

	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), responses)
	assert.Equal(suite.T(), int64(0), total)
	assert.Contains(suite.T(), err.Error(), "failed to get users")
}

// ===== Tests for SearchUsersGlobal =====

// TestSearchUsersGlobal_Success tests successfully searching users globally
func (suite *UserServiceTestSuite) TestSearchUsersGlobal_Success() {
	query := "john"
	limit, offset := 20, 0
	users := []models.User{
		{
			BaseModel: models.BaseModel{
				Name:  "John Doe",
				Title: "Senior Developer",
			},
			UserID:     "I123456",
			FirstName:  "John",
			LastName:   "Doe",
			Email:      "john@example.com",
			TeamDomain: models.TeamDomainDeveloper,
			TeamRole:   models.TeamRoleMember,
		},
		{
			BaseModel: models.BaseModel{
				Name:  "Mary Johnson",
				Title: "Product Owner",
			},
			UserID:     "I789012",
			FirstName:  "Mary",
			LastName:   "Johnson",
			Email:      "mary@example.com",
			TeamDomain: models.TeamDomainPO,
			TeamRole:   models.TeamRoleManager,
		},
	}
	expectedTotal := int64(2)

	suite.mockUserRepo.EXPECT().
		SearchByNameOrTitleGlobal(query, limit, offset).
		Return(users, expectedTotal, nil).
		Times(1)

	responses, total, err := suite.userService.SearchUsersGlobal(query, limit, offset)

	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), expectedTotal, total)
	assert.Len(suite.T(), responses, 2)
	assert.Equal(suite.T(), "I123456", responses[0].ID)
	assert.Equal(suite.T(), "John", responses[0].FirstName)
	assert.Equal(suite.T(), "I789012", responses[1].ID)
	assert.Equal(suite.T(), "Mary", responses[1].FirstName)
}

// TestSearchUsersGlobal_EmptyQuery tests searching with empty query
func (suite *UserServiceTestSuite) TestSearchUsersGlobal_EmptyQuery() {
	query := ""
	limit, offset := 20, 0
	users := []models.User{}
	expectedTotal := int64(0)

	suite.mockUserRepo.EXPECT().
		SearchByNameOrTitleGlobal(query, limit, offset).
		Return(users, expectedTotal, nil).
		Times(1)

	responses, total, err := suite.userService.SearchUsersGlobal(query, limit, offset)

	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), expectedTotal, total)
	assert.Len(suite.T(), responses, 0)
}

// TestSearchUsersGlobal_RepositoryError tests error when repository fails
func (suite *UserServiceTestSuite) TestSearchUsersGlobal_RepositoryError() {
	query := "test"
	limit, offset := 20, 0

	suite.mockUserRepo.EXPECT().
		SearchByNameOrTitleGlobal(query, limit, offset).
		Return(nil, int64(0), gorm.ErrInvalidDB).
		Times(1)

	responses, total, err := suite.userService.SearchUsersGlobal(query, limit, offset)

	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), responses)
	assert.Equal(suite.T(), int64(0), total)
	assert.Contains(suite.T(), err.Error(), "failed to search users")
}

// ===== Tests for GetQuickLinks =====

// TestGetQuickLinks_Success tests successfully getting quick links (returns empty array)
func (suite *UserServiceTestSuite) TestGetQuickLinks_Success() {
	userID := uuid.New()

	existingUser := suite.factories.User.Create()
	existingUser.UserID = "I123456"

	suite.mockUserRepo.EXPECT().
		GetByID(userID).
		Return(existingUser, nil).
		Times(1)

	response, err := suite.userService.GetQuickLinks(userID)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), response)
	assert.NotNil(suite.T(), response.QuickLinks)
	assert.Len(suite.T(), response.QuickLinks, 0) // Always returns empty array
}

// TestGetQuickLinks_UserNotFound tests error when user is not found
func (suite *UserServiceTestSuite) TestGetQuickLinks_UserNotFound() {
	userID := uuid.New()

	suite.mockUserRepo.EXPECT().
		GetByID(userID).
		Return(nil, apperrors.ErrUserNotFound).
		Times(1)

	response, err := suite.userService.GetQuickLinks(userID)

	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), response)
	assert.Contains(suite.T(), err.Error(), "user not found")
}

// ===== Tests for AddQuickLink =====

// TestAddQuickLink_Success tests successfully adding a quick link (no-op, returns user unchanged)
func (suite *UserServiceTestSuite) TestAddQuickLink_Success() {
	userID := uuid.New()

	existingUser := suite.factories.User.Create()
	existingUser.UserID = "I123456"

	req := &service.AddQuickLinkRequest{
		URL:      "https://github.com/user/repo",
		Title:    "My Repository",
		Icon:     "github",
		Category: "repository",
	}

	suite.mockUserRepo.EXPECT().
		GetByID(userID).
		Return(existingUser, nil).
		Times(1)

	response, err := suite.userService.AddQuickLink(userID, req)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), response)
	assert.Equal(suite.T(), existingUser.UserID, response.ID)
	assert.Equal(suite.T(), existingUser.FirstName, response.FirstName)
}

// TestAddQuickLink_WithoutOptionalFields tests adding a quick link without optional fields
func (suite *UserServiceTestSuite) TestAddQuickLink_WithoutOptionalFields() {
	userID := uuid.New()

	existingUser := suite.factories.User.Create()
	existingUser.UserID = "I123456"

	req := &service.AddQuickLinkRequest{
		URL:   "https://example.com",
		Title: "Example",
	}

	suite.mockUserRepo.EXPECT().
		GetByID(userID).
		Return(existingUser, nil).
		Times(1)

	response, err := suite.userService.AddQuickLink(userID, req)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), response)
	assert.Equal(suite.T(), existingUser.UserID, response.ID)
}

// TestAddQuickLink_ValidationError_MissingURL tests validation error when URL is missing
func (suite *UserServiceTestSuite) TestAddQuickLink_ValidationError_MissingURL() {
	userID := uuid.New()

	req := &service.AddQuickLinkRequest{
		Title:    "My Repository",
		Icon:     "github",
		Category: "repository",
	}

	response, err := suite.userService.AddQuickLink(userID, req)

	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), response)
	assert.Contains(suite.T(), err.Error(), "validation failed")
}

// TestAddQuickLink_ValidationError_InvalidURL tests validation error when URL is invalid
func (suite *UserServiceTestSuite) TestAddQuickLink_ValidationError_InvalidURL() {
	userID := uuid.New()

	req := &service.AddQuickLinkRequest{
		URL:   "not-a-url",
		Title: "My Repository",
	}

	response, err := suite.userService.AddQuickLink(userID, req)

	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), response)
	assert.Contains(suite.T(), err.Error(), "validation failed")
}

// TestAddQuickLink_ValidationError_MissingTitle tests validation error when title is missing
func (suite *UserServiceTestSuite) TestAddQuickLink_ValidationError_MissingTitle() {
	userID := uuid.New()

	req := &service.AddQuickLinkRequest{
		URL:      "https://github.com/user/repo",
		Icon:     "github",
		Category: "repository",
	}

	response, err := suite.userService.AddQuickLink(userID, req)

	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), response)
	assert.Contains(suite.T(), err.Error(), "validation failed")
}

// TestAddQuickLink_UserNotFound tests error when user is not found
func (suite *UserServiceTestSuite) TestAddQuickLink_UserNotFound() {
	userID := uuid.New()

	req := &service.AddQuickLinkRequest{
		URL:   "https://example.com",
		Title: "Example",
	}

	suite.mockUserRepo.EXPECT().
		GetByID(userID).
		Return(nil, apperrors.ErrUserNotFound).
		Times(1)

	response, err := suite.userService.AddQuickLink(userID, req)

	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), response)
	assert.Contains(suite.T(), err.Error(), "user not found")
}

// ===== Tests for RemoveQuickLink =====

// TestRemoveQuickLink_Success tests successfully removing a quick link (no-op, returns user unchanged)
func (suite *UserServiceTestSuite) TestRemoveQuickLink_Success() {
	userID := uuid.New()
	linkURL := "https://github.com/user/repo"

	existingUser := suite.factories.User.Create()
	existingUser.UserID = "I123456"

	suite.mockUserRepo.EXPECT().
		GetByID(userID).
		Return(existingUser, nil).
		Times(1)

	response, err := suite.userService.RemoveQuickLink(userID, linkURL)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), response)
	assert.Equal(suite.T(), existingUser.UserID, response.ID)
	assert.Equal(suite.T(), existingUser.FirstName, response.FirstName)
}

// TestRemoveQuickLink_EmptyURL tests error when URL is empty
func (suite *UserServiceTestSuite) TestRemoveQuickLink_EmptyURL() {
	userID := uuid.New()
	linkURL := ""

	response, err := suite.userService.RemoveQuickLink(userID, linkURL)

	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), response)
	assert.Contains(suite.T(), err.Error(), "link URL is required")
}

// TestRemoveQuickLink_UserNotFound tests error when user is not found
func (suite *UserServiceTestSuite) TestRemoveQuickLink_UserNotFound() {
	userID := uuid.New()
	linkURL := "https://github.com/user/repo"

	suite.mockUserRepo.EXPECT().
		GetByID(userID).
		Return(nil, apperrors.ErrUserNotFound).
		Times(1)

	response, err := suite.userService.RemoveQuickLink(userID, linkURL)

	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), response)
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
