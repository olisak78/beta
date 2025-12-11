package handlers_test

import (
	"bytes"
	"developer-portal-backend/internal/service"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-playground/validator/v10"

	"developer-portal-backend/internal/api/handlers"
	"developer-portal-backend/internal/database/models"
	"developer-portal-backend/internal/mocks"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

type teamRepoAdapter struct {
	inner *mocks.MockTeamRepositoryInterface
}

//func (a *teamRepoAdapter) Create(team *models.Team) error {
//	return a.inner.Create(team)
//}
//func (a *teamRepoAdapter) GetByID(id uuid.UUID) (*models.Team, error) {
//	return a.inner.GetByID(id)
//}
//func (a *teamRepoAdapter) GetByName(groupID uuid.UUID, name string) (*models.Team, error) {
//	return a.inner.GetByName(groupID, name)
//}
//func (a *teamRepoAdapter) GetByOrganizationID(orgID uuid.UUID, limit, offset int) ([]models.Team, int64, error) {
//	return a.inner.GetByOrganizationID(orgID, limit, offset)
//}
//func (a *teamRepoAdapter) GetByGroupID(groupID uuid.UUID, limit, offset int) ([]models.Team, int64, error) {
//	// not used in these tests
//	return []models.Team{}, 0, nil
//}
//func (a *teamRepoAdapter) GetAll() ([]models.Team, error) {
//	return a.inner.GetAll()
//}
//func (a *teamRepoAdapter) GetByNameGlobal(name string) (*models.Team, error) {
//	return a.inner.GetByNameGlobal(name)
//}
//func (a *teamRepoAdapter) GetWithMembers(id uuid.UUID) (*models.Team, error) {
//	return a.inner.GetWithMembers(id)
//}
//func (a *teamRepoAdapter) Update(team *models.Team) error {
//	return a.inner.Update(team)
//}
//func (a *teamRepoAdapter) Delete(id uuid.UUID) error {
//	return a.inner.Delete(id)
//}

type UserHandlerTestSuite struct {
	suite.Suite
	ctrl            *gomock.Controller
	mockUserRepo    *mocks.MockUserRepositoryInterface
	mockLinkRepo    *mocks.MockLinkRepositoryInterface
	mockPluginRepo  *mocks.MockPluginRepositoryInterface
	mockTeamRepo    *mocks.MockTeamRepositoryInterface
	mockTeamService *mocks.MockTeamServiceInterface
	userService     *service.UserService
	handler         *handlers.UserHandler
}

func (suite *UserHandlerTestSuite) SetupTest() {
	gin.SetMode(gin.TestMode)
	suite.ctrl = gomock.NewController(suite.T())
	suite.mockUserRepo = mocks.NewMockUserRepositoryInterface(suite.ctrl)
	suite.mockLinkRepo = mocks.NewMockLinkRepositoryInterface(suite.ctrl)
	suite.mockPluginRepo = mocks.NewMockPluginRepositoryInterface(suite.ctrl)
	suite.mockTeamRepo = mocks.NewMockTeamRepositoryInterface(suite.ctrl)
	suite.mockTeamService = mocks.NewMockTeamServiceInterface(suite.ctrl)

	v := validator.New()
	suite.userService = service.NewUserService(suite.mockUserRepo, suite.mockLinkRepo, suite.mockPluginRepo, v)
	suite.handler = handlers.NewUserHandler(suite.userService, suite.mockTeamService)
}

func (suite *UserHandlerTestSuite) TearDownTest() {
	suite.ctrl.Finish()
}

// helper to build router and optionally inject a username into context
func (suite *UserHandlerTestSuite) newRouter(withUsername bool, username string) *gin.Engine {
	r := gin.New()
	if withUsername {
		r.Use(func(c *gin.Context) {
			c.Set("username", username)
			c.Next()
		})
	}
	// routes
	r.POST("/users", suite.handler.CreateUser)
	r.GET("/users", suite.handler.ListUsers)
	r.GET("/users/me", suite.handler.GetCurrentUser)
	r.PUT("/users", suite.handler.UpdateUserTeam)
	r.GET("/users/:user_id", suite.handler.GetMemberByUserID)
	r.POST("/users/:user_id/favorites/:link_id", suite.handler.AddFavoriteLink)
	r.DELETE("/users/:user_id/favorites/:link_id", suite.handler.RemoveFavoriteLink)
	return r
}

/*************** CreateUser ***************/

func (suite *UserHandlerTestSuite) TestCreateUser_Success() {
	router := suite.newRouter(true, "creator.user")
	teamID := uuid.New()

	// team must exist (handler validates this)
	suite.mockTeamService.EXPECT().GetByID(teamID).Return(&service.TeamResponse{}, nil)

	// user service will check if email exists (should not exist)
	suite.mockUserRepo.EXPECT().GetByEmail("john.doe@example.com").Return(nil, errors.New("not found"))

	// user service will create the user
	suite.mockUserRepo.EXPECT().Create(gomock.Any()).Return(nil)

	body := map[string]interface{}{
		"id":          "i12345",
		"first_name":  "John",
		"last_name":   "Doe",
		"email":       "john.doe@example.com",
		"mobile":      "123456",
		"team_domain": "developer",
		"team_role":   "member",
		"team_id":     teamID,
	}
	data, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/users", bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusCreated, w.Code)
	var got map[string]interface{}
	assert.NoError(suite.T(), json.Unmarshal(w.Body.Bytes(), &got))
	assert.Equal(suite.T(), "i12345", got["id"])
	assert.Equal(suite.T(), "John", got["first_name"])
	assert.Equal(suite.T(), "Doe", got["last_name"])
}

func (suite *UserHandlerTestSuite) TestCreateUser_InvalidTeamID() {
	router := suite.newRouter(true, "creator.user")
	teamID := uuid.New()

	// team not found -> invalid team_id
	suite.mockTeamService.EXPECT().GetByID(teamID).Return(nil, errors.New("not found"))

	body := map[string]interface{}{
		"id":         "i12345",
		"first_name": "John",
		"last_name":  "Doe",
		"email":      "john.doe@example.com",
		"team_id":    teamID,
	}
	data, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/users", bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)
	assert.Contains(suite.T(), w.Body.String(), "invalid team_id")
}

func (suite *UserHandlerTestSuite) TestCreateUser_InvalidTeamDomain() {
	router := suite.newRouter(true, "creator.user")
	teamID := uuid.New()

	// team exists, but invalid domain value should fail
	suite.mockTeamService.EXPECT().GetByID(teamID).Return(&service.TeamResponse{}, nil)

	body := map[string]interface{}{
		"id":          "i12345",
		"first_name":  "John",
		"last_name":   "Doe",
		"email":       "john.doe@example.com",
		"team_id":     teamID,
		"team_domain": "invalid-domain",
	}
	data, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/users", bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)
	assert.Contains(suite.T(), w.Body.String(), "invalid team_domain")
}

func (suite *UserHandlerTestSuite) TestCreateUser_MissingUsername_Unauthorized() {
	router := suite.newRouter(false, "")
	teamID := uuid.New()

	// team exists
	suite.mockTeamService.EXPECT().GetByID(teamID).Return(&service.TeamResponse{}, nil)

	body := map[string]interface{}{
		"id":         "i12345",
		"first_name": "John",
		"last_name":  "Doe",
		"email":      "john.doe@example.com",
		"team_id":    teamID,
	}
	data, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/users", bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusUnauthorized, w.Code)
	assert.Contains(suite.T(), w.Body.String(), "missing username in token")
}

/*************** GetMemberByUserID ***************/

func (suite *UserHandlerTestSuite) TestGetMemberByUserID_Success() {
	router := suite.newRouter(false, "")
	userUUID := uuid.New()
	userID := "i123456"

	suite.mockUserRepo.EXPECT().GetByUserID(userID).Return(&models.User{
		BaseModel:  models.BaseModel{ID: userUUID},
		UserID:     userID,
		FirstName:  "Alice",
		LastName:   "Smith",
		Email:      "alice@example.com",
		TeamDomain: models.TeamDomainDeveloper,
		TeamRole:   models.TeamRoleMember,
	}, nil)
	suite.mockLinkRepo.EXPECT().GetByIDs(gomock.Any()).Return([]models.Link{}, nil)
	suite.mockLinkRepo.EXPECT().GetByOwner(userUUID).Return([]models.Link{}, nil)

	req := httptest.NewRequest(http.MethodGet, "/users/"+userID, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)
	var got map[string]interface{}
	assert.NoError(suite.T(), json.Unmarshal(w.Body.Bytes(), &got))
	assert.Equal(suite.T(), userID, got["id"])
	assert.Equal(suite.T(), "Alice", got["first_name"])
}

func (suite *UserHandlerTestSuite) TestGetMemberByUserID_NotFound() {
	router := suite.newRouter(false, "")

	// service maps any repo error to not found
	suite.mockUserRepo.EXPECT().GetByUserID("unknown").Return(nil, errors.New("missing"))

	req := httptest.NewRequest(http.MethodGet, "/users/unknown", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusNotFound, w.Code)
	assert.Contains(suite.T(), w.Body.String(), "User not found")
}

/*************** ListUsers ***************/

func (suite *UserHandlerTestSuite) TestListUsers_ByUserName_Success() {
	router := suite.newRouter(false, "")
	// First GetByName
	uID := uuid.New()
	userID := "iuser-1"
	suite.mockUserRepo.EXPECT().GetByName("john.doe").Return(&models.User{
		BaseModel:  models.BaseModel{ID: uID},
		UserID:     userID,
		FirstName:  "John",
		LastName:   "Doe",
		Email:      "john@example.com",
		TeamDomain: models.TeamDomainDeveloper,
		TeamRole:   models.TeamRoleMember,
	}, nil)
	// Then GetByUserID in GetUserByUserIDWithLinks
	suite.mockUserRepo.EXPECT().GetByUserID(userID).Return(&models.User{
		BaseModel:  models.BaseModel{ID: uID},
		UserID:     userID,
		FirstName:  "John",
		LastName:   "Doe",
		Email:      "john@example.com",
		TeamDomain: models.TeamDomainDeveloper,
		TeamRole:   models.TeamRoleMember,
	}, nil)
	suite.mockLinkRepo.EXPECT().GetByIDs(gomock.Any()).Return([]models.Link{}, nil)
	suite.mockLinkRepo.EXPECT().GetByOwner(uID).Return([]models.Link{}, nil)

	req := httptest.NewRequest(http.MethodGet, "/users?user-name=john.doe", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)
	var got map[string]interface{}
	assert.NoError(suite.T(), json.Unmarshal(w.Body.Bytes(), &got))
	assert.Equal(suite.T(), "John", got["first_name"])
}

func (suite *UserHandlerTestSuite) TestListUsers_SearchQ_Success() {
	router := suite.newRouter(false, "")

	users := []models.User{
		{BaseModel: models.BaseModel{ID: uuid.New()}, UserID: "i1", FirstName: "A", LastName: "B", Email: "a@b", TeamDomain: models.TeamDomainDeveloper, TeamRole: models.TeamRoleMember},
		{BaseModel: models.BaseModel{ID: uuid.New()}, UserID: "i2", FirstName: "C", LastName: "D", Email: "c@d", TeamDomain: models.TeamDomainDeveloper, TeamRole: models.TeamRoleMember},
	}
	suite.mockUserRepo.EXPECT().SearchByNameOrTitleGlobal("abc", 10, 5).Return(users, int64(2), nil)

	req := httptest.NewRequest(http.MethodGet, "/users?q=abc&limit=10&offset=5", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)
	var got map[string]interface{}
	assert.NoError(suite.T(), json.Unmarshal(w.Body.Bytes(), &got))
	assert.Equal(suite.T(), float64(2), got["total"])
}

func (suite *UserHandlerTestSuite) TestListUsers_Default_Success() {
	router := suite.newRouter(false, "")

	users := []models.User{
		{BaseModel: models.BaseModel{ID: uuid.New()}, UserID: "i1", FirstName: "A", LastName: "B", Email: "a@b", TeamDomain: models.TeamDomainDeveloper, TeamRole: models.TeamRoleMember},
	}
	suite.mockUserRepo.EXPECT().GetAll(20, 0).Return(users, int64(1), nil)

	req := httptest.NewRequest(http.MethodGet, "/users", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)
	var got map[string]interface{}
	assert.NoError(suite.T(), json.Unmarshal(w.Body.Bytes(), &got))
	assert.Equal(suite.T(), float64(1), got["total"])
}

/*************** GetCurrentUser ***************/

func (suite *UserHandlerTestSuite) TestGetCurrentUser_Success() {
	router := suite.newRouter(true, "john.doe")
	uID := uuid.New()
	userID := "i-123"

	// GetByName for current user
	suite.mockUserRepo.EXPECT().GetByName("john.doe").Return(&models.User{
		BaseModel:  models.BaseModel{ID: uID},
		UserID:     userID,
		FirstName:  "John",
		LastName:   "Doe",
		Email:      "john@example.com",
		TeamDomain: models.TeamDomainDeveloper,
		TeamRole:   models.TeamRoleMember,
	}, nil)
	// Then GetByUserID in WithLinks
	suite.mockUserRepo.EXPECT().GetByUserID(userID).Return(&models.User{
		BaseModel:  models.BaseModel{ID: uID},
		UserID:     userID,
		FirstName:  "John",
		LastName:   "Doe",
		Email:      "john@example.com",
		TeamDomain: models.TeamDomainDeveloper,
		TeamRole:   models.TeamRoleMember,
	}, nil)
	suite.mockLinkRepo.EXPECT().GetByIDs(gomock.Any()).Return([]models.Link{}, nil)
	suite.mockLinkRepo.EXPECT().GetByOwner(uID).Return([]models.Link{}, nil)

	req := httptest.NewRequest(http.MethodGet, "/users/me", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)
	var got map[string]interface{}
	assert.NoError(suite.T(), json.Unmarshal(w.Body.Bytes(), &got))
	assert.Equal(suite.T(), "John", got["first_name"])
}

func (suite *UserHandlerTestSuite) TestGetCurrentUser_Unauthorized() {
	router := suite.newRouter(false, "")

	req := httptest.NewRequest(http.MethodGet, "/users/me", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusUnauthorized, w.Code)
	assert.Contains(suite.T(), w.Body.String(), "missing username in token")
}

/*************** UpdateUserTeam ***************/

func (suite *UserHandlerTestSuite) TestUpdateUserTeam_Success() {
	router := suite.newRouter(true, "admin.user")
	userUUID := uuid.New()
	teamUUID := uuid.New()

	// team exists
	suite.mockTeamService.EXPECT().GetByID(teamUUID).Return(&service.TeamResponse{}, nil)
	// user exists and update ok
	suite.mockUserRepo.EXPECT().GetByID(userUUID).Return(&models.User{
		BaseModel: models.BaseModel{ID: userUUID},
		UserID:    "i123",
	}, nil)
	suite.mockUserRepo.EXPECT().Update(gomock.Any()).Return(nil)

	body := map[string]string{
		"user_uuid":     userUUID.String(),
		"new_team_uuid": teamUUID.String(),
	}
	data, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPut, "/users", bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)
}

func (suite *UserHandlerTestSuite) TestUpdateUserTeam_InvalidUserUUID() {
	router := suite.newRouter(true, "admin.user")
	body := map[string]string{
		"user_uuid":     "not-a-uuid",
		"new_team_uuid": uuid.New().String(),
	}
	data, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPut, "/users", bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)
	assert.Contains(suite.T(), w.Body.String(), "invalid user_uuid")
}

func (suite *UserHandlerTestSuite) TestUpdateUserTeam_InvalidTeamUUID_NotExists() {
	router := suite.newRouter(true, "admin.user")
	body := map[string]string{
		"user_uuid":     uuid.New().String(),
		"new_team_uuid": uuid.New().String(),
	}
	data, _ := json.Marshal(body)

	// team not found
	suite.mockTeamService.EXPECT().GetByID(gomock.Any()).Return(nil, errors.New("not found"))

	req := httptest.NewRequest(http.MethodPut, "/users", bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)
	assert.Contains(suite.T(), w.Body.String(), "invalid new_team_uuid")
}

func (suite *UserHandlerTestSuite) TestUpdateUserTeam_Unauthorized() {
	router := suite.newRouter(false, "")
	userUUID := uuid.New()
	teamUUID := uuid.New()

	// team exists, then unauthorized due to missing username
	suite.mockTeamService.EXPECT().GetByID(teamUUID).Return(&service.TeamResponse{}, nil)

	body := map[string]string{
		"user_uuid":     userUUID.String(),
		"new_team_uuid": teamUUID.String(),
	}
	data, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPut, "/users", bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusUnauthorized, w.Code)
	assert.Contains(suite.T(), w.Body.String(), "missing username in token")
}

func (suite *UserHandlerTestSuite) TestUpdateUserTeam_UserNotFound() {
	router := suite.newRouter(true, "admin.user")
	userUUID := uuid.New()
	teamUUID := uuid.New()

	suite.mockTeamService.EXPECT().GetByID(teamUUID).Return(&service.TeamResponse{}, nil)
	suite.mockUserRepo.EXPECT().GetByID(userUUID).Return(nil, errors.New("missing"))

	body := map[string]string{
		"user_uuid":     userUUID.String(),
		"new_team_uuid": teamUUID.String(),
	}
	data, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPut, "/users", bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusNotFound, w.Code)
	assert.Contains(suite.T(), w.Body.String(), "User not found")
}

/*************** Favorites ***************/

func (suite *UserHandlerTestSuite) TestAddFavoriteLink_Success() {
	router := suite.newRouter(false, "")
	linkID := uuid.New()
	uID := uuid.New()

	// Load by user_id then update
	suite.mockUserRepo.EXPECT().GetByUserID("iuser-1").Return(&models.User{
		BaseModel:  models.BaseModel{ID: uID},
		UserID:     "iuser-1",
		FirstName:  "A",
		LastName:   "B",
		Email:      "a@b",
		TeamDomain: models.TeamDomainDeveloper,
		TeamRole:   models.TeamRoleMember,
		Metadata:   json.RawMessage(nil),
	}, nil)
	suite.mockUserRepo.EXPECT().Update(gomock.Any()).Return(nil)

	req := httptest.NewRequest(http.MethodPost, "/users/iuser-1/favorites/"+linkID.String(), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)
}

func (suite *UserHandlerTestSuite) TestAddFavoriteLink_InvalidLinkID() {
	router := suite.newRouter(false, "")

	req := httptest.NewRequest(http.MethodPost, "/users/iuser-1/favorites/not-a-uuid", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)
	assert.Contains(suite.T(), w.Body.String(), "invalid link_id")
}

func (suite *UserHandlerTestSuite) TestAddFavoriteLink_UserNotFound() {
	router := suite.newRouter(false, "")
	linkID := uuid.New()

	suite.mockUserRepo.EXPECT().GetByUserID("missing").Return(nil, errors.New("nope"))

	req := httptest.NewRequest(http.MethodPost, "/users/missing/favorites/"+linkID.String(), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusNotFound, w.Code)
}

func (suite *UserHandlerTestSuite) TestRemoveFavoriteLink_Success() {
	router := suite.newRouter(false, "")
	linkID := uuid.New()
	uID := uuid.New()

	// Load by user_id then update
	suite.mockUserRepo.EXPECT().GetByUserID("iuser-2").Return(&models.User{
		BaseModel:  models.BaseModel{ID: uID},
		UserID:     "iuser-2",
		FirstName:  "A",
		LastName:   "B",
		Email:      "a@b",
		TeamDomain: models.TeamDomainDeveloper,
		TeamRole:   models.TeamRoleMember,
		Metadata:   json.RawMessage(`{"favorites":["` + linkID.String() + `"]}`),
	}, nil)
	suite.mockUserRepo.EXPECT().Update(gomock.Any()).Return(nil)

	req := httptest.NewRequest(http.MethodDelete, "/users/iuser-2/favorites/"+linkID.String(), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)
}

func TestUserHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(UserHandlerTestSuite))
}
