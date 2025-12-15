package service_test

import (
	"encoding/json"
	"errors"
	"testing"

	"developer-portal-backend/internal/database/models"
	apperrors "developer-portal-backend/internal/errors"
	"developer-portal-backend/internal/mocks"
	"developer-portal-backend/internal/service"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

// teamRepoStub is a lightweight stub that satisfies TeamRepositoryInterface.
// We only implement the methods actually used by LinkService in these tests (GetByID, GetByNameGlobal).
// The rest return default values to satisfy the interface.
type teamRepoStub struct {
	GetByIDFunc         func(id uuid.UUID) (*models.Team, error)
	GetByNameGlobalFunc func(name string) (*models.Team, error)
}

func (s *teamRepoStub) Create(team *models.Team) error { return errors.New("not implemented") }
func (s *teamRepoStub) GetByID(id uuid.UUID) (*models.Team, error) {
	if s.GetByIDFunc != nil {
		return s.GetByIDFunc(id)
	}
	return nil, errors.New("not implemented")
}
func (s *teamRepoStub) GetByName(groupID uuid.UUID, name string) (*models.Team, error) {
	return nil, errors.New("not implemented")
}
func (s *teamRepoStub) GetByOrganizationID(orgID uuid.UUID, limit, offset int) ([]models.Team, int64, error) {
	return nil, 0, errors.New("not implemented")
}
func (s *teamRepoStub) GetByGroupID(groupID uuid.UUID, limit, offset int) ([]models.Team, int64, error) {
	return nil, 0, errors.New("not implemented")
}
func (s *teamRepoStub) GetAll() ([]models.Team, error) { return nil, errors.New("not implemented") }
func (s *teamRepoStub) GetByNameGlobal(name string) (*models.Team, error) {
	if s.GetByNameGlobalFunc != nil {
		return s.GetByNameGlobalFunc(name)
	}
	return nil, errors.New("not implemented")
}
func (s *teamRepoStub) GetWithMembers(id uuid.UUID) (*models.Team, error) {
	return nil, errors.New("not implemented")
}
func (s *teamRepoStub) Update(team *models.Team) error { return errors.New("not implemented") }
func (s *teamRepoStub) Delete(id uuid.UUID) error      { return errors.New("not implemented") }

type LinkServiceTestSuite struct {
	suite.Suite
	ctrl             *gomock.Controller
	mockLinkRepo     *mocks.MockLinkRepositoryInterface
	mockUserRepo     *mocks.MockUserRepositoryInterface
	mockCategoryRepo *mocks.MockCategoryRepositoryInterface
	teamRepo         *teamRepoStub
	linkService      *service.LinkService
	validator        *validator.Validate
}

func (suite *LinkServiceTestSuite) SetupTest() {
	suite.ctrl = gomock.NewController(suite.T())
	suite.mockLinkRepo = mocks.NewMockLinkRepositoryInterface(suite.ctrl)
	suite.mockUserRepo = mocks.NewMockUserRepositoryInterface(suite.ctrl)
	suite.mockCategoryRepo = mocks.NewMockCategoryRepositoryInterface(suite.ctrl)
	suite.teamRepo = &teamRepoStub{}
	suite.validator = validator.New()

	suite.linkService = service.NewLinkService(
		suite.mockLinkRepo,
		suite.mockUserRepo,
		suite.teamRepo, // use stub instead of gomock for team repo to satisfy full interface
		suite.mockCategoryRepo,
		suite.validator,
	)
}

func (suite *LinkServiceTestSuite) TearDownTest() {
	suite.ctrl.Finish()
}

func (suite *LinkServiceTestSuite) TestCreateLink_Success_UserOwner() {
	ownerID := uuid.New()
	categoryID := uuid.New()
	createdBy := "user.created"
	req := &service.CreateLinkRequest{
		Name:        "my-link",
		Description: "some desc",
		Owner:       ownerID.String(),
		URL:         "https://example.com",
		CategoryID:  categoryID.String(),
		Tags:        "tag1, tag2",
		CreatedBy:   createdBy,
	}

	// created_by validation: found as user
	suite.mockUserRepo.EXPECT().GetByUserID(createdBy).Return(&models.User{UserID: createdBy}, nil)
	// owner validation: found as user by ID
	suite.mockUserRepo.EXPECT().GetByID(ownerID).Return(&models.User{BaseModel: models.BaseModel{ID: ownerID}}, nil)
	// category validation: found
	suite.mockCategoryRepo.EXPECT().GetByID(categoryID).Return(&models.Category{BaseModel: models.BaseModel{ID: categoryID}}, nil)
	// create: set ID on the entity
	suite.mockLinkRepo.EXPECT().Create(gomock.Any()).DoAndReturn(func(l *models.Link) error {
		l.ID = uuid.New()
		return nil
	})

	resp, err := suite.linkService.CreateLink(req)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), resp)
	assert.Equal(suite.T(), "my-link", resp.Name)
	assert.Equal(suite.T(), "my-link", resp.Title)
	assert.Equal(suite.T(), "some desc", resp.Description)
	assert.Equal(suite.T(), "https://example.com", resp.URL)
	assert.Equal(suite.T(), categoryID.String(), resp.CategoryID)
	assert.Equal(suite.T(), []string{"tag1", "tag2"}, resp.Tags)
}

func (suite *LinkServiceTestSuite) TestCreateLink_ValidationError() {
	req := &service.CreateLinkRequest{
		Name:       "",                  // required
		Owner:      uuid.New().String(), // valid
		URL:        "not-a-url",         // invalid
		CategoryID: uuid.New().String(),
		CreatedBy:  "someone",
	}
	resp, err := suite.linkService.CreateLink(req)

	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), resp)
	assert.Contains(suite.T(), err.Error(), "validation failed")
}

func (suite *LinkServiceTestSuite) TestCreateLink_URLMaxLength() {
	// Create a URL with exactly 2000 characters (should be accepted)
	baseURL := "https://example.com/path?"
	padding := ""
	for i := 0; i < 2000-len(baseURL); i++ {
		padding += "a"
	}
	maxLengthURL := baseURL + padding

	ownerID := uuid.New()
	categoryID := uuid.New()
	createdBy := "test.user"

	req := &service.CreateLinkRequest{
		Name:        "test-link",
		Description: "test description",
		Owner:       ownerID.String(),
		URL:         maxLengthURL, // URL with exactly 2000 characters
		CategoryID:  categoryID.String(),
		CreatedBy:   createdBy,
	}

	// Verify URL is exactly 2000 characters
	assert.Equal(suite.T(), 2000, len(req.URL))

	// Setup mocks for successful creation
	suite.mockUserRepo.EXPECT().GetByUserID(createdBy).Return(&models.User{UserID: createdBy}, nil)
	suite.mockUserRepo.EXPECT().GetByID(ownerID).Return(&models.User{BaseModel: models.BaseModel{ID: ownerID}}, nil)
	suite.mockCategoryRepo.EXPECT().GetByID(categoryID).Return(&models.Category{BaseModel: models.BaseModel{ID: categoryID}}, nil)
	suite.mockLinkRepo.EXPECT().Create(gomock.Any()).DoAndReturn(func(l *models.Link) error {
		l.ID = uuid.New()
		return nil
	})

	resp, err := suite.linkService.CreateLink(req)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), resp)
}

func (suite *LinkServiceTestSuite) TestCreateLink_URLTooLong() {
	// Create a URL that exceeds 2000 characters
	baseURL := "https://example.com/path?"
	// Add query parameters to make the URL exceed 2000 characters
	padding := ""
	for i := 0; i < 2001-len(baseURL); i++ {
		padding += "a"
	}
	longURL := baseURL + padding

	req := &service.CreateLinkRequest{
		Name:        "test-link",
		Description: "test description",
		Owner:       uuid.New().String(),
		URL:         longURL, // URL longer than 2000 characters
		CategoryID:  uuid.New().String(),
		CreatedBy:   "someone",
	}

	// Verify URL is indeed longer than 2000 characters
	assert.Greater(suite.T(), len(req.URL), 2000)

	resp, err := suite.linkService.CreateLink(req)

	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), resp)
	assert.Contains(suite.T(), err.Error(), "validation failed")
}

func (suite *LinkServiceTestSuite) TestCreateLink_CreatedByMissing() {
	req := &service.CreateLinkRequest{
		Name:       "ok",
		Owner:      uuid.New().String(),
		URL:        "https://example.com",
		CategoryID: uuid.New().String(),
		CreatedBy:  "", // missing
	}
	resp, err := suite.linkService.CreateLink(req)

	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), resp)
	assert.Contains(suite.T(), err.Error(), "created_by is required")
}

func (suite *LinkServiceTestSuite) TestCreateLink_CreatedByNotFound() {
	req := &service.CreateLinkRequest{
		Name:       "ok",
		Owner:      uuid.New().String(),
		URL:        "https://example.com",
		CategoryID: uuid.New().String(),
		CreatedBy:  "unknown_user_or_team",
	}
	// created_by: neither user nor team found
	suite.mockUserRepo.EXPECT().GetByUserID("unknown_user_or_team").Return(nil, errors.New("not found"))
	suite.teamRepo.GetByNameGlobalFunc = func(name string) (*models.Team, error) {
		return nil, errors.New("not found")
	}

	resp, err := suite.linkService.CreateLink(req)
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), resp)
	assert.Contains(suite.T(), err.Error(), "created_by user or team not found")
}

func (suite *LinkServiceTestSuite) TestCreateLink_InvalidOwnerUUID() {
	req := &service.CreateLinkRequest{
		Name:       "ok",
		Owner:      "bad-uuid",
		URL:        "https://example.com",
		CategoryID: uuid.New().String(),
		CreatedBy:  "creator",
	}

	resp, err := suite.linkService.CreateLink(req)
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), resp)
	assert.Contains(suite.T(), err.Error(), "validation failed")
}

func (suite *LinkServiceTestSuite) TestCreateLink_OwnerNotFound() {
	ownerID := uuid.New()
	req := &service.CreateLinkRequest{
		Name:       "ok",
		Owner:      ownerID.String(),
		URL:        "https://example.com",
		CategoryID: uuid.New().String(),
		CreatedBy:  "creator",
	}
	suite.mockUserRepo.EXPECT().GetByUserID("creator").Return(&models.User{UserID: "creator"}, nil)
	// owner not found as user nor team
	suite.mockUserRepo.EXPECT().GetByID(ownerID).Return(nil, errors.New("not found"))
	suite.teamRepo.GetByIDFunc = func(id uuid.UUID) (*models.Team, error) {
		return nil, errors.New("not found")
	}

	resp, err := suite.linkService.CreateLink(req)
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), resp)
	assert.Contains(suite.T(), err.Error(), "owner not found as user or team")
}

func (suite *LinkServiceTestSuite) TestCreateLink_CategoryNotFound() {
	ownerID := uuid.New()
	categoryID := uuid.New()
	req := &service.CreateLinkRequest{
		Name:       "ok",
		Owner:      ownerID.String(),
		URL:        "https://example.com",
		CategoryID: categoryID.String(),
		CreatedBy:  "creator",
	}
	suite.mockUserRepo.EXPECT().GetByUserID("creator").Return(&models.User{UserID: "creator"}, nil)
	suite.mockUserRepo.EXPECT().GetByID(ownerID).Return(&models.User{BaseModel: models.BaseModel{ID: ownerID}}, nil)
	// category not found
	suite.mockCategoryRepo.EXPECT().GetByID(categoryID).Return(nil, errors.New("not found"))

	resp, err := suite.linkService.CreateLink(req)
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), resp)
	assert.Contains(suite.T(), err.Error(), "category not found")
}

func (suite *LinkServiceTestSuite) TestCreateLink_RepoError() {
	ownerID := uuid.New()
	categoryID := uuid.New()
	req := &service.CreateLinkRequest{
		Name:        "ok",
		Description: "desc",
		Owner:       ownerID.String(),
		URL:         "https://example.com",
		CategoryID:  categoryID.String(),
		Tags:        "t1,t2",
		CreatedBy:   "creator",
	}
	suite.mockUserRepo.EXPECT().GetByUserID("creator").Return(&models.User{UserID: "creator"}, nil)
	suite.mockUserRepo.EXPECT().GetByID(ownerID).Return(&models.User{BaseModel: models.BaseModel{ID: ownerID}}, nil)
	suite.mockCategoryRepo.EXPECT().GetByID(categoryID).Return(&models.Category{BaseModel: models.BaseModel{ID: categoryID}}, nil)
	suite.mockLinkRepo.EXPECT().Create(gomock.Any()).Return(errors.New("db error"))

	resp, err := suite.linkService.CreateLink(req)
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), resp)
	assert.Contains(suite.T(), err.Error(), "failed to create link")
}

func (suite *LinkServiceTestSuite) TestGetByOwnerUserID_Success() {
	ownerUserID := "u123"
	ownerID := uuid.New()

	suite.mockUserRepo.EXPECT().GetByUserID(ownerUserID).Return(&models.User{BaseModel: models.BaseModel{ID: ownerID}, UserID: ownerUserID}, nil)

	catA := uuid.New()
	catB := uuid.New()
	link1 := models.Link{
		BaseModel: models.BaseModel{
			ID:          uuid.New(),
			Name:        "link-a",
			Title:       "link-a",
			Description: "d1",
		},
		Owner:      ownerID,
		URL:        "https://a.example.com",
		CategoryID: catA,
		Tags:       "a, b",
	}
	link2 := models.Link{
		BaseModel: models.BaseModel{
			ID:          uuid.New(),
			Name:        "link-b",
			Title:       "link-b",
			Description: "d2",
		},
		Owner:      ownerID,
		URL:        "https://b.example.com",
		CategoryID: catB,
		Tags:       "",
	}
	suite.mockLinkRepo.EXPECT().GetByOwner(ownerID).Return([]models.Link{link1, link2}, nil)

	res, err := suite.linkService.GetByOwnerUserID(ownerUserID)
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), res, 2)

	assert.Equal(suite.T(), "link-a", res[0].Name)
	assert.Equal(suite.T(), "https://a.example.com", res[0].URL)
	assert.Equal(suite.T(), catA.String(), res[0].CategoryID)
	assert.Equal(suite.T(), []string{"a", "b"}, res[0].Tags)

	assert.Equal(suite.T(), "link-b", res[1].Name)
	assert.Equal(suite.T(), "https://b.example.com", res[1].URL)
	assert.Equal(suite.T(), catB.String(), res[1].CategoryID)
	assert.Empty(suite.T(), res[1].Tags)
}

func (suite *LinkServiceTestSuite) TestGetByOwnerUserID_EmptyOwner() {
	res, err := suite.linkService.GetByOwnerUserID("")

	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), res)
	assert.Contains(suite.T(), err.Error(), "owner user_id is required")
}

func (suite *LinkServiceTestSuite) TestGetByOwnerUserID_UserNotFound() {
	suite.mockUserRepo.EXPECT().GetByUserID("missing").Return(nil, errors.New("not found"))

	res, err := suite.linkService.GetByOwnerUserID("missing")
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), res)
	assert.Contains(suite.T(), err.Error(), "owner user with user_id \"missing\" not found")
}

func (suite *LinkServiceTestSuite) TestGetByOwnerUserID_RepoError() {
	ownerID := uuid.New()
	suite.mockUserRepo.EXPECT().GetByUserID("u1").Return(&models.User{BaseModel: models.BaseModel{ID: ownerID}, UserID: "u1"}, nil)
	suite.mockLinkRepo.EXPECT().GetByOwner(ownerID).Return(nil, errors.New("db error"))

	res, err := suite.linkService.GetByOwnerUserID("u1")
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), res)
	assert.Contains(suite.T(), err.Error(), "failed to get links by owner")
}

func (suite *LinkServiceTestSuite) TestGetByOwnerUserIDWithViewer_FavoritesMarked() {
	ownerUserID := "owner1"
	viewerName := "viewer1"

	ownerID := uuid.New()
	viewerID := uuid.New()
	linkFavID := uuid.New()
	linkOtherID := uuid.New()

	// owner found
	suite.mockUserRepo.EXPECT().GetByUserID(ownerUserID).Return(&models.User{BaseModel: models.BaseModel{ID: ownerID}, UserID: ownerUserID}, nil)
	// viewer found with favorites metadata
	favs := map[string]interface{}{
		"favorites": []string{linkFavID.String()},
	}
	favBytes, _ := json.Marshal(favs)
	suite.mockUserRepo.EXPECT().GetByName(viewerName).Return(&models.User{
		BaseModel: models.BaseModel{ID: viewerID, Name: viewerName},
		Metadata:  favBytes,
	}, nil)

	links := []models.Link{
		{
			BaseModel:  models.BaseModel{ID: linkFavID, Name: "fav", Title: "fav"},
			Owner:      ownerID,
			URL:        "https://fav.example.com",
			CategoryID: uuid.New(),
			Tags:       "",
		},
		{
			BaseModel:  models.BaseModel{ID: linkOtherID, Name: "other", Title: "other"},
			Owner:      ownerID,
			URL:        "https://o.example.com",
			CategoryID: uuid.New(),
			Tags:       "",
		},
	}
	suite.mockLinkRepo.EXPECT().GetByOwner(ownerID).Return(links, nil)

	res, err := suite.linkService.GetByOwnerUserIDWithViewer(ownerUserID, viewerName)
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), res, 2)

	// Identify which is favorite
	var favFound, otherFound bool
	for _, r := range res {
		if r.Name == "fav" {
			assert.True(suite.T(), r.Favorite, "favorite flag should be true for favorite link")
			favFound = true
		}
		if r.Name == "other" {
			assert.False(suite.T(), r.Favorite, "favorite flag should be false for non-favorite link")
			otherFound = true
		}
	}
	assert.True(suite.T(), favFound)
	assert.True(suite.T(), otherFound)
}

func (suite *LinkServiceTestSuite) TestGetByOwnerUserIDWithViewer_EmptyViewer_Fallback() {
	ownerUserID := "owner1"
	ownerID := uuid.New()

	// Fallback to GetByOwnerUserID path
	// This results in a call to GetByUserID and GetByOwner
	suite.mockUserRepo.EXPECT().GetByUserID(ownerUserID).Return(&models.User{BaseModel: models.BaseModel{ID: ownerID}, UserID: ownerUserID}, nil)
	suite.mockLinkRepo.EXPECT().GetByOwner(ownerID).Return([]models.Link{}, nil)

	res, err := suite.linkService.GetByOwnerUserIDWithViewer(ownerUserID, "")
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), res)
	assert.Len(suite.T(), res, 0)
}

func (suite *LinkServiceTestSuite) TestGetByOwnerUserIDWithViewer_OwnerNotFound() {
	suite.mockUserRepo.EXPECT().GetByUserID("missing").Return(nil, errors.New("not found"))

	res, err := suite.linkService.GetByOwnerUserIDWithViewer("missing", "viewer")
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), res)
	assert.Contains(suite.T(), err.Error(), "owner user with user_id \"missing\" not found")
}

func (suite *LinkServiceTestSuite) TestGetByOwnerUserIDWithViewer_ViewerNotFound_Fallback() {
	ownerUserID := "owner2"
	ownerID := uuid.New()

	// Owner ok
	suite.mockUserRepo.EXPECT().GetByUserID(ownerUserID).Return(&models.User{BaseModel: models.BaseModel{ID: ownerID}, UserID: ownerUserID}, nil)
	// Viewer not found -> fallback to GetByOwnerUserID(ownerUserID)
	suite.mockUserRepo.EXPECT().GetByName("viewerX").Return(nil, errors.New("no viewer"))
	// Fallback path expects a second GetByUserID + GetByOwner call
	suite.mockUserRepo.EXPECT().GetByUserID(ownerUserID).Return(&models.User{BaseModel: models.BaseModel{ID: ownerID}, UserID: ownerUserID}, nil)
	suite.mockLinkRepo.EXPECT().GetByOwner(ownerID).Return([]models.Link{}, nil)

	res, err := suite.linkService.GetByOwnerUserIDWithViewer(ownerUserID, "viewerX")
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), res)
	assert.Len(suite.T(), res, 0)
}

func (suite *LinkServiceTestSuite) TestDeleteLink_Success() {
	id := uuid.New()
	suite.mockLinkRepo.EXPECT().Delete(id).Return(nil)

	err := suite.linkService.DeleteLink(id)
	assert.NoError(suite.T(), err)
}

func (suite *LinkServiceTestSuite) TestDeleteLink_Error() {
	id := uuid.New()
	suite.mockLinkRepo.EXPECT().Delete(id).Return(errors.New("db error"))

	err := suite.linkService.DeleteLink(id)
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "failed to delete link")
}

// UpdateLink Tests

func (suite *LinkServiceTestSuite) TestUpdateLink_Success() {
	linkID := uuid.New()
	categoryID := uuid.New()
	ownerID := uuid.New()
	updatedBy := "user.updated"

	req := &service.UpdateLinkRequest{
		Name:        "updated-link",
		Description: "updated description",
		URL:         "https://updated.example.com",
		CategoryID:  categoryID.String(),
		Tags:        "tag1, tag2, tag3",
		UpdatedBy:   updatedBy,
	}

	existingLink := &models.Link{
		BaseModel: models.BaseModel{
			ID:          linkID,
			Name:        "old-link",
			Title:       "old-link",
			Description: "old description",
		},
		Owner:      ownerID,
		URL:        "https://old.example.com",
		CategoryID: uuid.New(),
		Tags:       "old-tag",
	}

	// Validate updated_by user exists
	suite.mockUserRepo.EXPECT().GetByUserID(updatedBy).Return(&models.User{UserID: updatedBy}, nil)
	// Get existing link
	suite.mockLinkRepo.EXPECT().GetByID(linkID).Return(existingLink, nil)
	// Validate category exists
	suite.mockCategoryRepo.EXPECT().GetByID(categoryID).Return(&models.Category{BaseModel: models.BaseModel{ID: categoryID}}, nil)
	// Update link
	suite.mockLinkRepo.EXPECT().Update(gomock.Any()).DoAndReturn(func(l *models.Link) error {
		// Verify the link was updated correctly
		assert.Equal(suite.T(), "updated-link", l.Name)
		assert.Equal(suite.T(), "updated-link", l.Title)
		assert.Equal(suite.T(), "updated description", l.Description)
		assert.Equal(suite.T(), "https://updated.example.com", l.URL)
		assert.Equal(suite.T(), categoryID, l.CategoryID)
		assert.Equal(suite.T(), "tag1, tag2, tag3", l.Tags)
		assert.Equal(suite.T(), updatedBy, l.UpdatedBy)
		// Verify owner was NOT changed
		assert.Equal(suite.T(), ownerID, l.Owner)
		return nil
	})

	resp, err := suite.linkService.UpdateLink(linkID, req)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), resp)
	assert.Equal(suite.T(), linkID.String(), resp.ID)
	assert.Equal(suite.T(), "updated-link", resp.Name)
	assert.Equal(suite.T(), "updated-link", resp.Title)
	assert.Equal(suite.T(), "updated description", resp.Description)
	assert.Equal(suite.T(), "https://updated.example.com", resp.URL)
	assert.Equal(suite.T(), categoryID.String(), resp.CategoryID)
	assert.Equal(suite.T(), []string{"tag1", "tag2", "tag3"}, resp.Tags)
}

func (suite *LinkServiceTestSuite) TestUpdateLink_Success_UpdatedByTeam() {
	linkID := uuid.New()
	categoryID := uuid.New()
	ownerID := uuid.New()
	updatedBy := "team-name"

	req := &service.UpdateLinkRequest{
		Name:        "updated-link",
		Description: "updated by team",
		URL:         "https://updated.example.com",
		CategoryID:  categoryID.String(),
		Tags:        "tag1",
		UpdatedBy:   updatedBy,
	}

	existingLink := &models.Link{
		BaseModel: models.BaseModel{
			ID:          linkID,
			Name:        "old-link",
			Title:       "old-link",
			Description: "old description",
		},
		Owner:      ownerID,
		URL:        "https://old.example.com",
		CategoryID: uuid.New(),
		Tags:       "old-tag",
	}

	// Validate updated_by: not found as user, but found as team
	suite.mockUserRepo.EXPECT().GetByUserID(updatedBy).Return(nil, errors.New("user not found"))
	suite.teamRepo.GetByNameGlobalFunc = func(name string) (*models.Team, error) {
		if name == updatedBy {
			return &models.Team{BaseModel: models.BaseModel{Name: updatedBy}}, nil
		}
		return nil, errors.New("team not found")
	}
	// Get existing link
	suite.mockLinkRepo.EXPECT().GetByID(linkID).Return(existingLink, nil)
	// Validate category exists
	suite.mockCategoryRepo.EXPECT().GetByID(categoryID).Return(&models.Category{BaseModel: models.BaseModel{ID: categoryID}}, nil)
	// Update link
	suite.mockLinkRepo.EXPECT().Update(gomock.Any()).Return(nil)

	resp, err := suite.linkService.UpdateLink(linkID, req)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), resp)
	assert.Equal(suite.T(), "updated-link", resp.Name)
	assert.Equal(suite.T(), "updated-link", resp.Title)
	assert.Equal(suite.T(), "updated by team", resp.Description)
	assert.Equal(suite.T(), "https://updated.example.com", resp.URL)
	assert.Equal(suite.T(), categoryID.String(), resp.CategoryID)
	assert.Equal(suite.T(), []string{"tag1"}, resp.Tags)
}

func (suite *LinkServiceTestSuite) TestUpdateLink_ValidationError_InvalidFields() {
	testCases := []struct {
		name        string
		requestFunc func() *service.UpdateLinkRequest
		description string
	}{
		{
			name: "InvalidName",
			requestFunc: func() *service.UpdateLinkRequest {
				return &service.UpdateLinkRequest{
					Name:        "", // required, min=1
					Description: "desc",
					URL:         "https://example.com",
					CategoryID:  uuid.New().String(),
					UpdatedBy:   "user",
				}
			},
			description: "empty name should fail validation",
		},
		{
			name: "InvalidURL",
			requestFunc: func() *service.UpdateLinkRequest {
				return &service.UpdateLinkRequest{
					Name:        "valid-name",
					Description: "desc",
					URL:         "not-a-valid-url",
					CategoryID:  uuid.New().String(),
					UpdatedBy:   "user",
				}
			},
			description: "invalid URL format should fail validation",
		},
		{
			name: "InvalidCategoryUUID",
			requestFunc: func() *service.UpdateLinkRequest {
				return &service.UpdateLinkRequest{
					Name:        "valid-name",
					Description: "desc",
					URL:         "https://example.com",
					CategoryID:  "not-a-uuid",
					UpdatedBy:   "user",
				}
			},
			description: "invalid category UUID should fail validation",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			linkID := uuid.New()
			req := tc.requestFunc()

			resp, err := suite.linkService.UpdateLink(linkID, req)

			assert.Error(suite.T(), err, tc.description)
			assert.Nil(suite.T(), resp)
			assert.Contains(suite.T(), err.Error(), "validation error")
		})
	}
}

func (suite *LinkServiceTestSuite) TestUpdateLink_ValidationError_FieldsTooLong() {
	testCases := []struct {
		name        string
		requestFunc func() *service.UpdateLinkRequest
		description string
	}{
		{
			name: "NameTooLong",
			requestFunc: func() *service.UpdateLinkRequest {
				return &service.UpdateLinkRequest{
					Name:        "this-name-is-way-too-long-and-exceeds-the-maximum-allowed-length-of-40-characters",
					Description: "desc",
					URL:         "https://example.com",
					CategoryID:  uuid.New().String(),
					UpdatedBy:   "user",
				}
			},
			description: "name exceeding 40 characters should fail validation",
		},
		{
			name: "DescriptionTooLong",
			requestFunc: func() *service.UpdateLinkRequest {
				longDesc := ""
				for i := 0; i < 201; i++ {
					longDesc += "a"
				}
				return &service.UpdateLinkRequest{
					Name:        "valid-name",
					Description: longDesc, // max=200
					URL:         "https://example.com",
					CategoryID:  uuid.New().String(),
					UpdatedBy:   "user",
				}
			},
			description: "description exceeding 200 characters should fail validation",
		},
		{
			name: "URLTooLong",
			requestFunc: func() *service.UpdateLinkRequest {
				baseURL := "https://example.com/path?"
				padding := ""
				for i := 0; i < 2001-len(baseURL); i++ {
					padding += "a"
				}
				return &service.UpdateLinkRequest{
					Name:        "valid-name",
					Description: "desc",
					URL:         baseURL + padding, // max=2000
					CategoryID:  uuid.New().String(),
					UpdatedBy:   "user",
				}
			},
			description: "URL exceeding 2000 characters should fail validation",
		},
		{
			name: "TagsTooLong",
			requestFunc: func() *service.UpdateLinkRequest {
				longTags := ""
				for i := 0; i < 201; i++ {
					longTags += "a"
				}
				return &service.UpdateLinkRequest{
					Name:        "valid-name",
					Description: "desc",
					URL:         "https://example.com",
					CategoryID:  uuid.New().String(),
					Tags:        longTags, // max=200
					UpdatedBy:   "user",
				}
			},
			description: "tags exceeding 200 characters should fail validation",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			linkID := uuid.New()
			req := tc.requestFunc()

			resp, err := suite.linkService.UpdateLink(linkID, req)

			assert.Error(suite.T(), err, tc.description)
			assert.Nil(suite.T(), resp)
			assert.Contains(suite.T(), err.Error(), "validation error")
		})
	}
}

func (suite *LinkServiceTestSuite) TestUpdateLink_ValidationError_UpdatedByEmpty() {
	linkID := uuid.New()
	req := &service.UpdateLinkRequest{
		Name:        "valid-name",
		Description: "desc",
		URL:         "https://example.com",
		CategoryID:  uuid.New().String(),
		UpdatedBy:   "", // required
	}

	resp, err := suite.linkService.UpdateLink(linkID, req)

	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), resp)
	assert.Contains(suite.T(), err.Error(), "updated_by is required")
}

func (suite *LinkServiceTestSuite) TestUpdateLink_ValidationError_UpdatedByWhitespace() {
	linkID := uuid.New()
	req := &service.UpdateLinkRequest{
		Name:        "valid-name",
		Description: "desc",
		URL:         "https://example.com",
		CategoryID:  uuid.New().String(),
		UpdatedBy:   "   ", // only whitespace
	}

	resp, err := suite.linkService.UpdateLink(linkID, req)

	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), resp)
	assert.Contains(suite.T(), err.Error(), "updated_by is required")
}

func (suite *LinkServiceTestSuite) TestUpdateLink_Error_UpdatedByNotFound() {
	linkID := uuid.New()
	req := &service.UpdateLinkRequest{
		Name:        "valid-name",
		Description: "desc",
		URL:         "https://example.com",
		CategoryID:  uuid.New().String(),
		UpdatedBy:   "unknown-user",
	}

	// updated_by not found as user nor team
	suite.mockUserRepo.EXPECT().GetByUserID("unknown-user").Return(nil, errors.New("user not found"))
	suite.teamRepo.GetByNameGlobalFunc = func(name string) (*models.Team, error) {
		return nil, errors.New("team not found")
	}

	resp, err := suite.linkService.UpdateLink(linkID, req)

	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), resp)
	// Should return ErrUserNotFound
	assert.ErrorIs(suite.T(), err, apperrors.ErrUserOrTeamNotFound)
}

func (suite *LinkServiceTestSuite) TestUpdateLink_Error_LinkNotFound() {
	linkID := uuid.New()
	categoryID := uuid.New()
	req := &service.UpdateLinkRequest{
		Name:        "valid-name",
		Description: "desc",
		URL:         "https://example.com",
		CategoryID:  categoryID.String(),
		UpdatedBy:   "user",
	}

	// updated_by validation passes
	suite.mockUserRepo.EXPECT().GetByUserID("user").Return(&models.User{UserID: "user"}, nil)
	// Link not found
	suite.mockLinkRepo.EXPECT().GetByID(linkID).Return(nil, errors.New("not found"))

	resp, err := suite.linkService.UpdateLink(linkID, req)

	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), resp)
	assert.ErrorIs(suite.T(), err, apperrors.ErrLinkNotFound)
}

func (suite *LinkServiceTestSuite) TestUpdateLink_Error_CategoryNotFound() {
	linkID := uuid.New()
	categoryID := uuid.New()
	ownerID := uuid.New()
	req := &service.UpdateLinkRequest{
		Name:        "valid-name",
		Description: "desc",
		URL:         "https://example.com",
		CategoryID:  categoryID.String(),
		UpdatedBy:   "user",
	}

	existingLink := &models.Link{
		BaseModel: models.BaseModel{
			ID:          linkID,
			Name:        "old-link",
			Title:       "old-link",
			Description: "old description",
		},
		Owner:      ownerID,
		URL:        "https://old.example.com",
		CategoryID: uuid.New(),
		Tags:       "old-tag",
	}

	// updated_by validation passes
	suite.mockUserRepo.EXPECT().GetByUserID("user").Return(&models.User{UserID: "user"}, nil)
	// Link found
	suite.mockLinkRepo.EXPECT().GetByID(linkID).Return(existingLink, nil)
	// Category not found
	suite.mockCategoryRepo.EXPECT().GetByID(categoryID).Return(nil, errors.New("category not found"))

	resp, err := suite.linkService.UpdateLink(linkID, req)

	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), resp)
	assert.ErrorIs(suite.T(), err, apperrors.ErrCategoryNotFound)
}

func (suite *LinkServiceTestSuite) TestUpdateLink_Error_RepositoryUpdateFails() {
	linkID := uuid.New()
	categoryID := uuid.New()
	ownerID := uuid.New()
	req := &service.UpdateLinkRequest{
		Name:        "valid-name",
		Description: "desc",
		URL:         "https://example.com",
		CategoryID:  categoryID.String(),
		UpdatedBy:   "user",
	}

	existingLink := &models.Link{
		BaseModel: models.BaseModel{
			ID:          linkID,
			Name:        "old-link",
			Title:       "old-link",
			Description: "old description",
		},
		Owner:      ownerID,
		URL:        "https://old.example.com",
		CategoryID: uuid.New(),
		Tags:       "old-tag",
	}

	// updated_by validation passes
	suite.mockUserRepo.EXPECT().GetByUserID("user").Return(&models.User{UserID: "user"}, nil)
	// Link found
	suite.mockLinkRepo.EXPECT().GetByID(linkID).Return(existingLink, nil)
	// Category found
	suite.mockCategoryRepo.EXPECT().GetByID(categoryID).Return(&models.Category{BaseModel: models.BaseModel{ID: categoryID}}, nil)
	// Update fails
	suite.mockLinkRepo.EXPECT().Update(gomock.Any()).Return(errors.New("database error"))

	resp, err := suite.linkService.UpdateLink(linkID, req)

	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), resp)
	assert.Contains(suite.T(), err.Error(), "failed to update link")
}

func (suite *LinkServiceTestSuite) TestUpdateLink_EmptyTags() {
	linkID := uuid.New()
	categoryID := uuid.New()
	ownerID := uuid.New()
	req := &service.UpdateLinkRequest{
		Name:        "link-name",
		Description: "desc",
		URL:         "https://example.com",
		CategoryID:  categoryID.String(),
		Tags:        "", // empty tags
		UpdatedBy:   "user",
	}

	existingLink := &models.Link{
		BaseModel: models.BaseModel{
			ID:          linkID,
			Name:        "old-name",
			Title:       "old-title",
			Description: "old description",
		},
		Owner:      ownerID,
		URL:        "https://old.example.com",
		CategoryID: uuid.New(),
		Tags:       "old-tag1, old-tag2",
	}

	// updated_by validation passes
	suite.mockUserRepo.EXPECT().GetByUserID("user").Return(&models.User{UserID: "user"}, nil)
	// Link found
	suite.mockLinkRepo.EXPECT().GetByID(linkID).Return(existingLink, nil)
	// Category found
	suite.mockCategoryRepo.EXPECT().GetByID(categoryID).Return(&models.Category{BaseModel: models.BaseModel{ID: categoryID}}, nil)
	// Update
	suite.mockLinkRepo.EXPECT().Update(gomock.Any()).DoAndReturn(func(l *models.Link) error {
		assert.Equal(suite.T(), "", l.Tags)
		return nil
	})

	resp, err := suite.linkService.UpdateLink(linkID, req)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), resp)
	assert.Empty(suite.T(), resp.Tags)
}

func (suite *LinkServiceTestSuite) TestUpdateLink_ParseCategoryUUIDError() {
	linkID := uuid.New()
	req := &service.UpdateLinkRequest{
		Name:        "valid-name",
		Description: "desc",
		URL:         "https://example.com",
		CategoryID:  "invalid-uuid-format",
		UpdatedBy:   "user",
	}

	// The validator will catch the invalid UUID format before we even get to parsing
	resp, err := suite.linkService.UpdateLink(linkID, req)

	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), resp)
	assert.Contains(suite.T(), err.Error(), "validation error")
}

func TestLinkServiceTestSuite(t *testing.T) {
	suite.Run(t, new(LinkServiceTestSuite))
}
