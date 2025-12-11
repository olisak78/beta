package handlers_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"developer-portal-backend/internal/api/handlers"
	"developer-portal-backend/internal/mocks"
	"developer-portal-backend/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

type LinkHandlerTestSuite struct {
	suite.Suite
	ctrl     *gomock.Controller
	mockLink *mocks.MockLinkServiceInterface
	handler  *handlers.LinkHandler
}

func (suite *LinkHandlerTestSuite) SetupTest() {
	gin.SetMode(gin.TestMode)
	suite.ctrl = gomock.NewController(suite.T())
	suite.mockLink = mocks.NewMockLinkServiceInterface(suite.ctrl)
	suite.handler = handlers.NewLinkHandler(suite.mockLink)
}

func (suite *LinkHandlerTestSuite) TearDownTest() {
	suite.ctrl.Finish()
}

// helper to build a router and optionally inject a viewer (username) via middleware
func (suite *LinkHandlerTestSuite) newRouter(withViewer bool, viewerName string) *gin.Engine {
	r := gin.New()
	if withViewer {
		r.Use(func(c *gin.Context) {
			c.Set("username", viewerName)
			c.Next()
		})
	}
	r.GET("/links", suite.handler.ListLinks)
	r.POST("/links", suite.handler.CreateLink)
	r.PUT("/links/:id", suite.handler.UpdateLink)
	r.DELETE("/links/:id", suite.handler.DeleteLink)
	return r
}

func (suite *LinkHandlerTestSuite) TestListLinks_WithViewer_DefaultOwner() {
	router := suite.newRouter(true, "john.doe")

	link := service.LinkResponse{
		ID:          uuid.New().String(),
		Name:        "docs",
		Title:       "docs",
		Description: "Developer docs",
		URL:         "https://example.com/docs",
		CategoryID:  uuid.New().String(),
		Tags:        []string{"doc", "team"},
		Favorite:    true,
	}
	suite.mockLink.EXPECT().
		GetByOwnerUserIDWithViewer("cis.devops", "john.doe").
		Return([]service.LinkResponse{link}, nil)

	req := httptest.NewRequest(http.MethodGet, "/links", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)
	var got []service.LinkResponse
	err := json.Unmarshal(w.Body.Bytes(), &got)
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), got, 1)
	assert.Equal(suite.T(), link.ID, got[0].ID)
	assert.True(suite.T(), got[0].Favorite)
}

func (suite *LinkHandlerTestSuite) TestListLinks_WithoutViewer_DefaultOwner() {
	router := suite.newRouter(false, "")

	link := service.LinkResponse{
		ID:          uuid.New().String(),
		Name:        "docs",
		Title:       "docs",
		Description: "Developer docs",
		URL:         "https://example.com/docs",
		CategoryID:  uuid.New().String(),
		Tags:        []string{"doc"},
	}
	suite.mockLink.EXPECT().
		GetByOwnerUserID("cis.devops").
		Return([]service.LinkResponse{link}, nil)

	req := httptest.NewRequest(http.MethodGet, "/links", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)
	var got []service.LinkResponse
	err := json.Unmarshal(w.Body.Bytes(), &got)
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), got, 1)
	assert.Equal(suite.T(), link.ID, got[0].ID)
}

func (suite *LinkHandlerTestSuite) TestListLinks_NotFound_404() {
	router := suite.newRouter(true, "john.doe")

	suite.mockLink.EXPECT().
		GetByOwnerUserIDWithViewer("cis.devops", "john.doe").
		Return(nil, errors.New(`owner user with user_id "cis.devops" not found`))

	req := httptest.NewRequest(http.MethodGet, "/links", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusNotFound, w.Code)
	assert.Contains(suite.T(), w.Body.String(), "not found")
}

func (suite *LinkHandlerTestSuite) TestListLinks_ServiceError_500() {
	router := suite.newRouter(false, "")

	// To trigger 500 in handler, return non-nil slice and error
	suite.mockLink.EXPECT().
		GetByOwnerUserID("cis.devops").
		Return([]service.LinkResponse{}, errors.New("db failure"))

	req := httptest.NewRequest(http.MethodGet, "/links", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusInternalServerError, w.Code)
	assert.Contains(suite.T(), w.Body.String(), "Failed to get links")
	assert.Contains(suite.T(), w.Body.String(), "db failure")
}

func (suite *LinkHandlerTestSuite) TestCreateLink_Unauthorized_NoUsername() {
	router := suite.newRouter(false, "")

	body := `{
		"name":"Doc",
		"description":"Docs",
		"owner":"` + uuid.New().String() + `",
		"url":"https://example.com",
		"category_id":"` + uuid.New().String() + `",
		"tags":"a,b"
	}`
	req := httptest.NewRequest(http.MethodPost, "/links", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusUnauthorized, w.Code)
	assert.Contains(suite.T(), w.Body.String(), "missing username in token")
}

func (suite *LinkHandlerTestSuite) TestCreateLink_Success() {
	router := suite.newRouter(true, "cis.devops")

	ownerID := uuid.New().String()
	categoryID := uuid.New().String()
	body := `{
		"name":"Doc",
		"description":"Docs",
		"owner":"` + ownerID + `",
		"url":"https://example.com",
		"category_id":"` + categoryID + `",
		"tags":"a,b"
	}`

	suite.mockLink.EXPECT().
		CreateLink(gomock.Any()).
		DoAndReturn(func(req *service.CreateLinkRequest) (*service.LinkResponse, error) {
			// Validate CreatedBy is mapped from context username
			assert.Equal(suite.T(), "cis.devops", req.CreatedBy)
			assert.Equal(suite.T(), "Doc", req.Name)
			assert.Equal(suite.T(), "Docs", req.Description)
			assert.Equal(suite.T(), ownerID, req.Owner)
			assert.Equal(suite.T(), "https://example.com", req.URL)
			assert.Equal(suite.T(), categoryID, req.CategoryID)

			return &service.LinkResponse{
				ID:          uuid.New().String(),
				Name:        req.Name,
				Title:       req.Name,
				Description: req.Description,
				URL:         req.URL,
				CategoryID:  req.CategoryID,
				Tags:        []string{"a", "b"},
			}, nil
		})

	req := httptest.NewRequest(http.MethodPost, "/links", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusCreated, w.Code)
	var got service.LinkResponse
	err := json.Unmarshal(w.Body.Bytes(), &got)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "Doc", got.Name)
	assert.Equal(suite.T(), "Doc", got.Title) // mirrors name
	assert.Equal(suite.T(), "https://example.com", got.URL)
	assert.Equal(suite.T(), categoryID, got.CategoryID)
	assert.ElementsMatch(suite.T(), []string{"a", "b"}, got.Tags)
}

func (suite *LinkHandlerTestSuite) TestCreateLink_BadRequest_FromService() {
	router := suite.newRouter(true, "cis.devops")

	body := `{
		"name":"",
		"description":"Docs",
		"owner":"` + uuid.New().String() + `",
		"url":"https://example.com",
		"category_id":"` + uuid.New().String() + `",
		"tags":""
	}`

	suite.mockLink.EXPECT().
		CreateLink(gomock.Any()).
		Return(nil, errors.New("validation failed: name is required"))

	req := httptest.NewRequest(http.MethodPost, "/links", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)
	assert.Contains(suite.T(), w.Body.String(), "validation failed")
}

func (suite *LinkHandlerTestSuite) TestUpdateLink_InvalidUUID() {
	router := suite.newRouter(false, "")

	req := httptest.NewRequest(http.MethodPut, "/links/not-a-uuid", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)
	assert.Contains(suite.T(), w.Body.String(), "invalid link ID")
}

func (suite *LinkHandlerTestSuite) TestUpdateLink_Unauthorized_NoUsername() {
	router := suite.newRouter(false, "")

	id := uuid.New()
	categoryID := uuid.New().String()
	body := `{
		"name":"Updated Doc",
		"description":"Updated Docs",
		"url":"https://example.com/updated",
		"category_id":"` + categoryID + `",
		"tags":"c,d"
	}`
	req := httptest.NewRequest(http.MethodPut, "/links/"+id.String(), bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusUnauthorized, w.Code)
	assert.Contains(suite.T(), w.Body.String(), "missing username in token")
}

func (suite *LinkHandlerTestSuite) TestUpdateLink_Success() {
	router := suite.newRouter(true, "cis.devops")

	id := uuid.New()
	categoryID := uuid.New().String()
	body := `{
		"name":"Updated Doc",
		"description":"Updated Docs",
		"url":"https://example.com/updated",
		"category_id":"` + categoryID + `",
		"tags":"c,d"
	}`

	suite.mockLink.EXPECT().
		UpdateLink(id, gomock.Any()).
		DoAndReturn(func(linkID uuid.UUID, req *service.UpdateLinkRequest) (*service.LinkResponse, error) {
			// Validate UpdatedBy is mapped from context username
			assert.Equal(suite.T(), "cis.devops", req.UpdatedBy)
			assert.Equal(suite.T(), "Updated Doc", req.Name)
			assert.Equal(suite.T(), "Updated Docs", req.Description)
			assert.Equal(suite.T(), "https://example.com/updated", req.URL)
			assert.Equal(suite.T(), categoryID, req.CategoryID)

			return &service.LinkResponse{
				ID:          id.String(),
				Name:        req.Name,
				Title:       req.Name,
				Description: req.Description,
				URL:         req.URL,
				CategoryID:  req.CategoryID,
				Tags:        []string{"c", "d"},
			}, nil
		})

	req := httptest.NewRequest(http.MethodPut, "/links/"+id.String(), bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)
	var got service.LinkResponse
	err := json.Unmarshal(w.Body.Bytes(), &got)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "Updated Doc", got.Name)
	assert.Equal(suite.T(), "Updated Doc", got.Title) // mirrors name
	assert.Equal(suite.T(), "https://example.com/updated", got.URL)
	assert.Equal(suite.T(), categoryID, got.CategoryID)
	assert.ElementsMatch(suite.T(), []string{"c", "d"}, got.Tags)
}

func (suite *LinkHandlerTestSuite) TestUpdateLink_NotFound() {
	router := suite.newRouter(true, "cis.devops")

	id := uuid.New()
	categoryID := uuid.New().String()
	body := `{
		"name":"Updated Doc",
		"description":"Updated Docs",
		"url":"https://example.com/updated",
		"category_id":"` + categoryID + `",
		"tags":"c,d"
	}`

	suite.mockLink.EXPECT().
		UpdateLink(id, gomock.Any()).
		Return(nil, errors.New("link not found"))

	req := httptest.NewRequest(http.MethodPut, "/links/"+id.String(), bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusInternalServerError, w.Code)
	assert.Contains(suite.T(), w.Body.String(), "failed to update link")
}

func (suite *LinkHandlerTestSuite) TestUpdateLink_BadRequest_FromService() {
	router := suite.newRouter(true, "cis.devops")

	id := uuid.New()
	categoryID := uuid.New().String()
	body := `{
		"name":"",
		"description":"Updated Docs",
		"url":"https://example.com/updated",
		"category_id":"` + categoryID + `",
		"tags":""
	}`

	suite.mockLink.EXPECT().
		UpdateLink(id, gomock.Any()).
		Return(nil, errors.New("validation failed: name is required"))

	req := httptest.NewRequest(http.MethodPut, "/links/"+id.String(), bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusInternalServerError, w.Code)
	assert.Contains(suite.T(), w.Body.String(), "failed")
}

func (suite *LinkHandlerTestSuite) TestDeleteLink_InvalidUUID() {
	router := suite.newRouter(false, "")

	req := httptest.NewRequest(http.MethodDelete, "/links/not-a-uuid", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)
	assert.Contains(suite.T(), w.Body.String(), "invalid link ID")
}

func (suite *LinkHandlerTestSuite) TestDeleteLink_ServiceError() {
	router := suite.newRouter(false, "")

	id := uuid.New()
	suite.mockLink.EXPECT().
		DeleteLink(id).
		Return(errors.New("repo failure"))

	req := httptest.NewRequest(http.MethodDelete, "/links/"+id.String(), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusInternalServerError, w.Code)
	assert.Contains(suite.T(), w.Body.String(), "failed to delete link")
	assert.Contains(suite.T(), w.Body.String(), "repo failure")
}

func (suite *LinkHandlerTestSuite) TestDeleteLink_Success() {
	router := suite.newRouter(false, "")

	id := uuid.New()
	suite.mockLink.EXPECT().
		DeleteLink(id).
		Return(nil)

	req := httptest.NewRequest(http.MethodDelete, "/links/"+id.String(), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusNoContent, w.Code)
	assert.Equal(suite.T(), "", w.Body.String())
}

func TestLinkHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(LinkHandlerTestSuite))
}
