package handlers_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"developer-portal-backend/internal/api/handlers"
	apperrors "developer-portal-backend/internal/errors"
	"developer-portal-backend/internal/mocks"
	"developer-portal-backend/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

type DocumentationHandlerTestSuite struct {
	suite.Suite
	ctrl       *gomock.Controller
	mockDocSvc *mocks.MockDocumentationServiceInterface
	handler    *handlers.DocumentationHandler
}

func (suite *DocumentationHandlerTestSuite) SetupTest() {
	gin.SetMode(gin.TestMode)
	suite.ctrl = gomock.NewController(suite.T())
	suite.mockDocSvc = mocks.NewMockDocumentationServiceInterface(suite.ctrl)
	suite.handler = handlers.NewDocumentationHandler(suite.mockDocSvc)
}

func (suite *DocumentationHandlerTestSuite) TearDownTest() {
	suite.ctrl.Finish()
}

// Helper to create a router with optional username in context
func (suite *DocumentationHandlerTestSuite) newRouter(withUsername bool, username string) *gin.Engine {
	r := gin.New()
	if withUsername {
		r.Use(func(c *gin.Context) {
			c.Set("username", username)
			c.Next()
		})
	}
	r.POST("/documentations", suite.handler.CreateDocumentation)
	r.GET("/documentations/:id", suite.handler.GetDocumentationByID)
	r.GET("/teams/:id/documentations", suite.handler.GetDocumentationsByTeamID)
	r.PATCH("/documentations/:id", suite.handler.UpdateDocumentation)
	r.DELETE("/documentations/:id", suite.handler.DeleteDocumentation)
	return r
}

// TestCreateDocumentation_Success tests successful documentation creation
func (suite *DocumentationHandlerTestSuite) TestCreateDocumentation_Success() {
	router := suite.newRouter(true, "john.doe")
	teamID := uuid.New()
	docID := uuid.New()

	reqBody := map[string]interface{}{
		"team_id":     teamID.String(),
		"url":         "https://github.tools.sap/owner/repo/tree/main/docs",
		"title":       "API Documentation",
		"description": "REST API docs",
	}
	bodyBytes, _ := json.Marshal(reqBody)

	expectedResp := &service.DocumentationResponse{
		ID:          docID.String(),
		TeamID:      teamID.String(),
		Owner:       "owner",
		Repo:        "repo",
		Branch:      "main",
		DocsPath:    "docs",
		Title:       "API Documentation",
		Description: "REST API docs",
		CreatedBy:   "john.doe",
	}

	suite.mockDocSvc.EXPECT().CreateDocumentation(gomock.Any()).Return(expectedResp, nil)

	req := httptest.NewRequest(http.MethodPost, "/documentations", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusCreated, w.Code)

	var got service.DocumentationResponse
	err := json.Unmarshal(w.Body.Bytes(), &got)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), docID.String(), got.ID)
	assert.Equal(suite.T(), "API Documentation", got.Title)
}

// TestCreateDocumentation_InvalidJSON tests invalid JSON payload
func (suite *DocumentationHandlerTestSuite) TestCreateDocumentation_InvalidJSON() {
	router := suite.newRouter(true, "john.doe")

	req := httptest.NewRequest(http.MethodPost, "/documentations", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)
	assert.Contains(suite.T(), w.Body.String(), "error")
}

// TestCreateDocumentation_MissingUsername tests missing username in token
func (suite *DocumentationHandlerTestSuite) TestCreateDocumentation_MissingUsername() {
	router := suite.newRouter(false, "")
	teamID := uuid.New()

	reqBody := map[string]interface{}{
		"team_id":     teamID.String(),
		"url":         "https://github.tools.sap/owner/repo/tree/main/docs",
		"title":       "API Documentation",
		"description": "REST API docs",
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/documentations", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusUnauthorized, w.Code)
	assert.Contains(suite.T(), w.Body.String(), apperrors.ErrMissingUsernameInToken.Error())
}

// TestCreateDocumentation_ServiceError tests service layer error
func (suite *DocumentationHandlerTestSuite) TestCreateDocumentation_ServiceError() {
	router := suite.newRouter(true, "john.doe")
	teamID := uuid.New()

	reqBody := map[string]interface{}{
		"team_id":     teamID.String(),
		"url":         "https://github.tools.sap/owner/repo/tree/main/docs",
		"title":       "API Documentation",
		"description": "REST API docs",
	}
	bodyBytes, _ := json.Marshal(reqBody)

	suite.mockDocSvc.EXPECT().CreateDocumentation(gomock.Any()).Return(nil, errors.New("database error"))

	req := httptest.NewRequest(http.MethodPost, "/documentations", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)
	assert.Contains(suite.T(), w.Body.String(), "database error")
}

// TestGetDocumentationByID_Success tests successful retrieval by ID
func (suite *DocumentationHandlerTestSuite) TestGetDocumentationByID_Success() {
	router := suite.newRouter(false, "")
	docID := uuid.New()
	teamID := uuid.New()

	expectedResp := &service.DocumentationResponse{
		ID:          docID.String(),
		TeamID:      teamID.String(),
		Owner:       "owner",
		Repo:        "repo",
		Branch:      "main",
		DocsPath:    "docs",
		Title:       "API Documentation",
		Description: "REST API docs",
	}

	suite.mockDocSvc.EXPECT().GetDocumentationByID(docID).Return(expectedResp, nil)

	req := httptest.NewRequest(http.MethodGet, "/documentations/"+docID.String(), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var got service.DocumentationResponse
	err := json.Unmarshal(w.Body.Bytes(), &got)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), docID.String(), got.ID)
	assert.Equal(suite.T(), "API Documentation", got.Title)
}

// TestGetDocumentationByID_InvalidUUID tests invalid UUID format
func (suite *DocumentationHandlerTestSuite) TestGetDocumentationByID_InvalidUUID() {
	router := suite.newRouter(false, "")

	req := httptest.NewRequest(http.MethodGet, "/documentations/invalid-uuid", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)
	assert.Contains(suite.T(), w.Body.String(), apperrors.ErrInvalidDocumentationID.Error())
}

// TestGetDocumentationByID_NotFound tests documentation not found
func (suite *DocumentationHandlerTestSuite) TestGetDocumentationByID_NotFound() {
	router := suite.newRouter(false, "")
	docID := uuid.New()

	suite.mockDocSvc.EXPECT().
		GetDocumentationByID(docID).
		Return(nil, apperrors.ErrDocumentationNotFound)

	req := httptest.NewRequest(http.MethodGet, "/documentations/"+docID.String(), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusNotFound, w.Code)
	assert.Contains(suite.T(), w.Body.String(), apperrors.ErrDocumentationNotFound.Error())
}

// TestGetDocumentationsByTeamID_Success tests successful retrieval by team ID
func (suite *DocumentationHandlerTestSuite) TestGetDocumentationsByTeamID_Success() {
	router := suite.newRouter(false, "")
	teamID := uuid.New()
	docID1 := uuid.New()
	docID2 := uuid.New()

	expectedDocs := []service.DocumentationResponse{
		{
			ID:       docID1.String(),
			TeamID:   teamID.String(),
			Title:    "Doc 1",
			Owner:    "owner1",
			Repo:     "repo1",
			Branch:   "main",
			DocsPath: "docs",
		},
		{
			ID:       docID2.String(),
			TeamID:   teamID.String(),
			Title:    "Doc 2",
			Owner:    "owner2",
			Repo:     "repo2",
			Branch:   "main",
			DocsPath: "docs",
		},
	}

	suite.mockDocSvc.EXPECT().
		GetDocumentationsByTeamID(teamID).
		Return(expectedDocs, nil)

	req := httptest.NewRequest(http.MethodGet, "/teams/"+teamID.String()+"/documentations", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var got []service.DocumentationResponse
	err := json.Unmarshal(w.Body.Bytes(), &got)
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), got, 2)
	assert.Equal(suite.T(), "Doc 1", got[0].Title)
	assert.Equal(suite.T(), "Doc 2", got[1].Title)
}

// TestGetDocumentationsByTeamID_InvalidTeamID tests invalid team UUID
func (suite *DocumentationHandlerTestSuite) TestGetDocumentationsByTeamID_InvalidTeamID() {
	router := suite.newRouter(false, "")

	req := httptest.NewRequest(http.MethodGet, "/teams/invalid-uuid/documentations", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)
	assert.Contains(suite.T(), w.Body.String(), apperrors.ErrInvalidTeamID.Error())
}

// TestGetDocumentationsByTeamID_ServiceError tests service error returns 404
func (suite *DocumentationHandlerTestSuite) TestGetDocumentationsByTeamID_ServiceError() {
	router := suite.newRouter(false, "")
	teamID := uuid.New()

	suite.mockDocSvc.EXPECT().
		GetDocumentationsByTeamID(teamID).
		Return(nil, errors.New("service error"))

	req := httptest.NewRequest(http.MethodGet, "/teams/"+teamID.String()+"/documentations", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusNotFound, w.Code)
	assert.Contains(suite.T(), w.Body.String(), "service error")
}

// TestUpdateDocumentation_Success tests successful documentation update
func (suite *DocumentationHandlerTestSuite) TestUpdateDocumentation_Success() {
	router := suite.newRouter(true, "jane.doe")
	docID := uuid.New()
	teamID := uuid.New()

	newTitle := "Updated Title"
	reqBody := map[string]interface{}{
		"title": newTitle,
	}
	bodyBytes, _ := json.Marshal(reqBody)

	expectedResp := &service.DocumentationResponse{
		ID:        docID.String(),
		TeamID:    teamID.String(),
		Title:     newTitle,
		UpdatedBy: "jane.doe",
	}

	suite.mockDocSvc.EXPECT().
		UpdateDocumentation(docID, gomock.Any()).
		Return(expectedResp, nil)

	req := httptest.NewRequest(http.MethodPatch, "/documentations/"+docID.String(), bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var got service.DocumentationResponse
	err := json.Unmarshal(w.Body.Bytes(), &got)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), newTitle, got.Title)
}

// TestUpdateDocumentation_InvalidDocID tests invalid documentation ID
func (suite *DocumentationHandlerTestSuite) TestUpdateDocumentation_InvalidDocID() {
	router := suite.newRouter(true, "jane.doe")

	reqBody := map[string]interface{}{
		"title": "Updated Title",
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPatch, "/documentations/invalid-uuid", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)
	assert.Contains(suite.T(), w.Body.String(), apperrors.ErrInvalidDocumentationID.Error())
}

// TestUpdateDocumentation_InvalidJSON tests invalid JSON payload
func (suite *DocumentationHandlerTestSuite) TestUpdateDocumentation_InvalidJSON() {
	router := suite.newRouter(true, "jane.doe")
	docID := uuid.New()

	req := httptest.NewRequest(http.MethodPatch, "/documentations/"+docID.String(), bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)
}

// TestUpdateDocumentation_MissingUsername tests missing username in token
func (suite *DocumentationHandlerTestSuite) TestUpdateDocumentation_MissingUsername() {
	router := suite.newRouter(false, "")
	docID := uuid.New()

	reqBody := map[string]interface{}{
		"title": "Updated Title",
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPatch, "/documentations/"+docID.String(), bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusUnauthorized, w.Code)
	assert.Contains(suite.T(), w.Body.String(), apperrors.ErrMissingUsernameInToken.Error())
}

// TestUpdateDocumentation_ServiceError tests service layer error
func (suite *DocumentationHandlerTestSuite) TestUpdateDocumentation_ServiceError() {
	router := suite.newRouter(true, "jane.doe")
	docID := uuid.New()

	reqBody := map[string]interface{}{
		"title": "Updated Title",
	}
	bodyBytes, _ := json.Marshal(reqBody)

	suite.mockDocSvc.EXPECT().UpdateDocumentation(docID, gomock.Any()).Return(nil, errors.New("update failed"))

	req := httptest.NewRequest(http.MethodPatch, "/documentations/"+docID.String(), bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)
	assert.Contains(suite.T(), w.Body.String(), "update failed")
}

// TestDeleteDocumentation_Success tests successful documentation deletion
func (suite *DocumentationHandlerTestSuite) TestDeleteDocumentation_Success() {
	router := suite.newRouter(false, "")
	docID := uuid.New()

	suite.mockDocSvc.EXPECT().DeleteDocumentation(docID).Return(nil)

	req := httptest.NewRequest(http.MethodDelete, "/documentations/"+docID.String(), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusNoContent, w.Code)
}

// TestDeleteDocumentation_InvalidDocID tests invalid documentation ID
func (suite *DocumentationHandlerTestSuite) TestDeleteDocumentation_InvalidDocID() {
	router := suite.newRouter(false, "")

	req := httptest.NewRequest(http.MethodDelete, "/documentations/invalid-uuid", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)
	assert.Contains(suite.T(), w.Body.String(), apperrors.ErrInvalidDocumentationID.Error())
}

// TestDeleteDocumentation_ServiceError tests service layer error
func (suite *DocumentationHandlerTestSuite) TestDeleteDocumentation_ServiceError() {
	router := suite.newRouter(false, "")
	docID := uuid.New()

	suite.mockDocSvc.EXPECT().DeleteDocumentation(docID).Return(apperrors.ErrFailedToDeleteDocumentation)

	req := httptest.NewRequest(http.MethodDelete, "/documentations/"+docID.String(), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusInternalServerError, w.Code)
	assert.Contains(suite.T(), w.Body.String(), apperrors.ErrFailedToDeleteDocumentation.Error())
}

func TestDocumentationHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(DocumentationHandlerTestSuite))
}
