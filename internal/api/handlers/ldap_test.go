package handlers_test

import (
	apperrors "developer-portal-backend/internal/errors"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"developer-portal-backend/internal/api/handlers"
	"developer-portal-backend/internal/mocks"
	"developer-portal-backend/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

// LDAPHandlerTestSuite tests LDAP handler with mocked service
type LDAPHandlerTestSuite struct {
	suite.Suite
	ctrl            *gomock.Controller
	mockLDAPService *mocks.MockLDAPServiceInterface
	mockUserRepo    *mocks.MockUserRepositoryInterface
	handler         *handlers.LDAPHandler
}

func (suite *LDAPHandlerTestSuite) SetupTest() {
	gin.SetMode(gin.TestMode)
	suite.ctrl = gomock.NewController(suite.T())
	suite.mockLDAPService = mocks.NewMockLDAPServiceInterface(suite.ctrl)
	suite.mockUserRepo = mocks.NewMockUserRepositoryInterface(suite.ctrl)
	suite.handler = handlers.NewLDAPHandler(suite.mockLDAPService, suite.mockUserRepo)
}

func (suite *LDAPHandlerTestSuite) TearDownTest() {
	suite.ctrl.Finish()
}

// newRouter creates a test router with LDAP handler routes
func (suite *LDAPHandlerTestSuite) newRouter() *gin.Engine {
	r := gin.New()
	r.GET("/users/search/new", suite.handler.UserSearch)
	return r
}

/*************** UserSearch ***************/

func (suite *LDAPHandlerTestSuite) TestUserSearch_MissingNameParameter() {
	router := suite.newRouter()

	req := httptest.NewRequest(http.MethodGet, "/users/search/new", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)
	assert.Contains(suite.T(), w.Body.String(), apperrors.NewMissingQueryParam("name").Error())
}

func (suite *LDAPHandlerTestSuite) TestUserSearch_EmptyNameParameter() {
	router := suite.newRouter()

	req := httptest.NewRequest(http.MethodGet, "/users/search/new?name=", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)
	assert.Contains(suite.T(), w.Body.String(), apperrors.NewMissingQueryParam("name").Error())
}

func (suite *LDAPHandlerTestSuite) TestUserSearch_SuccessWithNewUsers() {
	router := suite.newRouter()

	// Mock LDAP search results
	ldapUsers := []service.LDAPUser{
		{
			Name:      "i123456",
			GivenName: "John",
			SN:        "Doe",
			Mail:      "john.doe@example.com",
			Mobile:    "1234567890",
		},
		{
			Name:      "i789012",
			GivenName: "Jane",
			SN:        "Smith",
			Mail:      "jane.smith@example.com",
			Mobile:    "0987654321",
		},
	}

	suite.mockLDAPService.EXPECT().SearchUsersByCN("john").Return(ldapUsers, nil)
	suite.mockUserRepo.EXPECT().GetExistingUserIDs([]string{"i123456", "i789012"}).Return([]string{"i123456"}, nil)

	req := httptest.NewRequest(http.MethodGet, "/users/search/new?name=john", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)
	assert.Contains(suite.T(), w.Body.String(), "i123456")
	assert.Contains(suite.T(), w.Body.String(), "i789012")
	assert.Contains(suite.T(), w.Body.String(), "John")
	assert.Contains(suite.T(), w.Body.String(), "Jane")
	assert.Contains(suite.T(), w.Body.String(), "john.doe@example.com")
	assert.Contains(suite.T(), w.Body.String(), "jane.smith@example.com")
}

func (suite *LDAPHandlerTestSuite) TestUserSearch_SuccessAllNewUsers() {
	router := suite.newRouter()

	ldapUsers := []service.LDAPUser{
		{
			Name:      "i111111",
			GivenName: "Alice",
			SN:        "Wonder",
			Mail:      "alice@example.com",
			Mobile:    "1111111111",
		},
	}

	suite.mockLDAPService.EXPECT().SearchUsersByCN("alice").Return(ldapUsers, nil)
	suite.mockUserRepo.EXPECT().GetExistingUserIDs([]string{"i111111"}).Return([]string{}, nil)

	req := httptest.NewRequest(http.MethodGet, "/users/search/new?name=alice", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)
	assert.Contains(suite.T(), w.Body.String(), "i111111")
	assert.Contains(suite.T(), w.Body.String(), "Alice")
	assert.Contains(suite.T(), w.Body.String(), "alice@example.com")
}

func (suite *LDAPHandlerTestSuite) TestUserSearch_LDAPError() {
	router := suite.newRouter()

	suite.mockLDAPService.EXPECT().SearchUsersByCN("test").Return(nil, errors.New("connection failed"))

	req := httptest.NewRequest(http.MethodGet, "/users/search/new?name=test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusBadGateway, w.Code)
	assert.Contains(suite.T(), w.Body.String(), "ldap search failed")
	assert.Contains(suite.T(), w.Body.String(), "connection failed")
}

func (suite *LDAPHandlerTestSuite) TestUserSearch_EmptyLDAPResults() {
	router := suite.newRouter()

	suite.mockLDAPService.EXPECT().SearchUsersByCN("nonexistent").Return([]service.LDAPUser{}, nil)
	suite.mockUserRepo.EXPECT().GetExistingUserIDs([]string{}).Return([]string{}, nil)

	req := httptest.NewRequest(http.MethodGet, "/users/search/new?name=nonexistent", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)
	assert.Contains(suite.T(), w.Body.String(), `"result":[]`)
}

func (suite *LDAPHandlerTestSuite) TestUserSearch_RepoErrorIgnored() {
	router := suite.newRouter()

	ldapUsers := []service.LDAPUser{
		{
			Name:      "i999999",
			GivenName: "Bob",
			SN:        "Builder",
			Mail:      "bob@example.com",
			Mobile:    "9999999999",
		},
	}

	suite.mockLDAPService.EXPECT().SearchUsersByCN("bob").Return(ldapUsers, nil)
	suite.mockUserRepo.EXPECT().GetExistingUserIDs([]string{"i999999"}).Return(nil, errors.New("db error"))

	req := httptest.NewRequest(http.MethodGet, "/users/search/new?name=bob", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should still succeed, treating all users as new
	assert.Equal(suite.T(), http.StatusOK, w.Code)
	assert.Contains(suite.T(), w.Body.String(), "i999999")
	assert.Contains(suite.T(), w.Body.String(), "Bob")
}

func TestLDAPHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(LDAPHandlerTestSuite))
}
