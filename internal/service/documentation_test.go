package service_test

import (
	"errors"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"developer-portal-backend/internal/database/models"
	"developer-portal-backend/internal/mocks"
	"developer-portal-backend/internal/service"
)

// DocumentationServiceTestSuite defines the test suite for DocumentationService
type DocumentationServiceTestSuite struct {
	suite.Suite
	ctrl                 *gomock.Controller
	mockDocRepo          *mocks.MockDocumentationRepositoryInterface
	mockTeamRepo         *mocks.MockTeamRepositoryInterface
	documentationService *service.DocumentationService
	validator            *validator.Validate
	testTeamID           uuid.UUID
	testTeam             *models.Team
}

// SetupTest sets up the test suite
func (suite *DocumentationServiceTestSuite) SetupTest() {
	suite.ctrl = gomock.NewController(suite.T())
	suite.mockDocRepo = mocks.NewMockDocumentationRepositoryInterface(suite.ctrl)
	suite.mockTeamRepo = mocks.NewMockTeamRepositoryInterface(suite.ctrl)
	suite.validator = validator.New()

	suite.documentationService = service.NewDocumentationService(
		suite.mockDocRepo,
		suite.mockTeamRepo,
		suite.validator,
	)

	// Setup test data
	suite.testTeamID = uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	suite.testTeam = &models.Team{}
	suite.testTeam.ID = suite.testTeamID
}

// TearDownTest cleans up after each test
func (suite *DocumentationServiceTestSuite) TearDownTest() {
	suite.ctrl.Finish()
}

func (suite *DocumentationServiceTestSuite) TestCreateDocumentation_Success_FullGitHubURL() {
	req := &service.CreateDocumentationRequest{
		TeamID:      suite.testTeamID.String(),
		URL:         "https://github.tools.sap/cfs-platform-engineering/cfs-platform-docs/tree/main/docs/coe",
		Title:       "Platform Documentation",
		Description: "Complete platform documentation",
		CreatedBy:   "test-user",
	}

	suite.mockTeamRepo.EXPECT().
		GetByID(suite.testTeamID).
		Return(suite.testTeam, nil).
		Times(1)

	suite.mockDocRepo.EXPECT().
		Create(gomock.Any()).
		DoAndReturn(func(doc *models.Documentation) error {
			doc.ID = uuid.New()
			return nil
		}).
		Times(1)

	result, err := suite.documentationService.CreateDocumentation(req)

	suite.NoError(err)
	suite.NotNil(result)
	suite.Equal(suite.testTeamID.String(), result.TeamID)
	suite.Equal("cfs-platform-engineering", result.Owner)
	suite.Equal("cfs-platform-docs", result.Repo)
	suite.Equal("main", result.Branch)
	suite.Equal("docs/coe", result.DocsPath)
	suite.Equal("Platform Documentation", result.Title)
	suite.Equal("Complete platform documentation", result.Description)
	suite.Equal("test-user", result.CreatedBy)
}

func (suite *DocumentationServiceTestSuite) TestCreateDocumentation_Success_RepositoryRootURL() {
	req := &service.CreateDocumentationRequest{
		TeamID:      suite.testTeamID.String(),
		URL:         "https://github.tools.sap/cfs-platform-engineering/developer-portal-frontend",
		Title:       "Frontend Documentation",
		Description: "Frontend repository docs",
		CreatedBy:   "test-user",
	}

	suite.mockTeamRepo.EXPECT().
		GetByID(suite.testTeamID).
		Return(suite.testTeam, nil).
		Times(1)

	suite.mockDocRepo.EXPECT().
		Create(gomock.Any()).
		DoAndReturn(func(doc *models.Documentation) error {
			doc.ID = uuid.New()
			return nil
		}).
		Times(1)

	result, err := suite.documentationService.CreateDocumentation(req)

	suite.NoError(err)
	suite.NotNil(result)
	suite.Equal("cfs-platform-engineering", result.Owner)
	suite.Equal("developer-portal-frontend", result.Repo)
	suite.Equal("main", result.Branch)
	suite.Equal("/", result.DocsPath)
}

func (suite *DocumentationServiceTestSuite) TestCreateDocumentation_Success_GitHubComURL() {
	req := &service.CreateDocumentationRequest{
		TeamID:      suite.testTeamID.String(),
		URL:         "https://github.com/kubernetes/kubernetes/tree/master/docs",
		Title:       "Kubernetes Docs",
		Description: "Kubernetes documentation",
		CreatedBy:   "test-user",
	}

	suite.mockTeamRepo.EXPECT().
		GetByID(suite.testTeamID).
		Return(suite.testTeam, nil).
		Times(1)

	suite.mockDocRepo.EXPECT().
		Create(gomock.Any()).
		DoAndReturn(func(doc *models.Documentation) error {
			doc.ID = uuid.New()
			return nil
		}).
		Times(1)

	result, err := suite.documentationService.CreateDocumentation(req)

	suite.NoError(err)
	suite.NotNil(result)
	suite.Equal("kubernetes", result.Owner)
	suite.Equal("kubernetes", result.Repo)
	suite.Equal("master", result.Branch)
	suite.Equal("docs", result.DocsPath)
}

func (suite *DocumentationServiceTestSuite) TestCreateDocumentation_Success_BlobURL() {
	req := &service.CreateDocumentationRequest{
		TeamID:      suite.testTeamID.String(),
		URL:         "https://github.tools.sap/org/repo/blob/develop/README.md",
		Title:       "README Documentation",
		Description: "README file",
		CreatedBy:   "test-user",
	}

	suite.mockTeamRepo.EXPECT().
		GetByID(suite.testTeamID).
		Return(suite.testTeam, nil).
		Times(1)

	suite.mockDocRepo.EXPECT().
		Create(gomock.Any()).
		DoAndReturn(func(doc *models.Documentation) error {
			doc.ID = uuid.New()
			return nil
		}).
		Times(1)

	result, err := suite.documentationService.CreateDocumentation(req)

	suite.NoError(err)
	suite.NotNil(result)
	suite.Equal("org", result.Owner)
	suite.Equal("repo", result.Repo)
	suite.Equal("develop", result.Branch)
	suite.Equal("README.md", result.DocsPath)
}

func (suite *DocumentationServiceTestSuite) TestCreateDocumentation_Success_UnicodeCharacters() {
	req := &service.CreateDocumentationRequest{
		TeamID:      suite.testTeamID.String(),
		URL:         "https://github.tools.sap/org/repo",
		Title:       "文档 📚 Documentation",
		Description: "中文 العربية 🚀",
		CreatedBy:   "test-user",
	}

	suite.mockTeamRepo.EXPECT().
		GetByID(suite.testTeamID).
		Return(suite.testTeam, nil).
		Times(1)

	suite.mockDocRepo.EXPECT().
		Create(gomock.Any()).
		DoAndReturn(func(doc *models.Documentation) error {
			doc.ID = uuid.New()
			return nil
		}).
		Times(1)

	result, err := suite.documentationService.CreateDocumentation(req)

	suite.NoError(err)
	suite.NotNil(result)
	suite.Equal("文档 📚 Documentation", result.Title)
	suite.Equal("中文 العربية 🚀", result.Description)
}

func (suite *DocumentationServiceTestSuite) TestCreateDocumentation_Success_DeepNestedPath() {
	req := &service.CreateDocumentationRequest{
		TeamID:      suite.testTeamID.String(),
		URL:         "https://github.tools.sap/org/repo/tree/main/docs/api/v1/reference/guide",
		Title:       "Deep Path Doc",
		Description: "Documentation with deep path",
		CreatedBy:   "test-user",
	}

	suite.mockTeamRepo.EXPECT().
		GetByID(suite.testTeamID).
		Return(suite.testTeam, nil).
		Times(1)

	suite.mockDocRepo.EXPECT().
		Create(gomock.Any()).
		DoAndReturn(func(doc *models.Documentation) error {
			doc.ID = uuid.New()
			return nil
		}).
		Times(1)

	result, err := suite.documentationService.CreateDocumentation(req)

	suite.NoError(err)
	suite.NotNil(result)
	suite.Equal("docs/api/v1/reference/guide", result.DocsPath)
}

func (suite *DocumentationServiceTestSuite) TestCreateDocumentation_Success_MaxLength() {
	maxTitle := string(make([]byte, 100)) // Exactly 100 characters
	maxDesc := string(make([]byte, 200))
	req := &service.CreateDocumentationRequest{
		TeamID:      suite.testTeamID.String(),
		URL:         "https://github.tools.sap/org/repo",
		Title:       maxTitle,
		Description: maxDesc,
		CreatedBy:   "test-user",
	}

	suite.mockTeamRepo.EXPECT().
		GetByID(suite.testTeamID).
		Return(suite.testTeam, nil).
		Times(1)

	suite.mockDocRepo.EXPECT().
		Create(gomock.Any()).
		DoAndReturn(func(doc *models.Documentation) error {
			doc.ID = uuid.New()
			return nil
		}).
		Times(1)

	result, err := suite.documentationService.CreateDocumentation(req)

	suite.NoError(err)
	suite.NotNil(result)
	suite.Equal(100, len(result.Title))
	suite.Equal(200, len(result.Description))
}

func (suite *DocumentationServiceTestSuite) TestCreateDocumentation_Error_MissingTeamID() {
	req := &service.CreateDocumentationRequest{
		TeamID:      "",
		URL:         "https://github.tools.sap/org/repo",
		Title:       "Test Doc",
		Description: "Test",
		CreatedBy:   "test-user",
	}

	result, err := suite.documentationService.CreateDocumentation(req)

	suite.Error(err)
	suite.Nil(result)
	suite.Contains(err.Error(), "validation failed")
}

func (suite *DocumentationServiceTestSuite) TestCreateDocumentation_Error_InvalidTeamIDUUID() {
	req := &service.CreateDocumentationRequest{
		TeamID:      "not-a-uuid",
		URL:         "https://github.tools.sap/org/repo",
		Title:       "Test Doc",
		Description: "Test",
		CreatedBy:   "test-user",
	}

	result, err := suite.documentationService.CreateDocumentation(req)

	suite.Error(err)
	suite.Nil(result)
	suite.Contains(err.Error(), "validation failed")
}

func (suite *DocumentationServiceTestSuite) TestCreateDocumentation_Error_MissingURL() {
	req := &service.CreateDocumentationRequest{
		TeamID:      suite.testTeamID.String(),
		URL:         "",
		Title:       "Test Doc",
		Description: "Test",
		CreatedBy:   "test-user",
	}

	result, err := suite.documentationService.CreateDocumentation(req)

	suite.Error(err)
	suite.Nil(result)
	suite.Contains(err.Error(), "validation failed")
}

func (suite *DocumentationServiceTestSuite) TestCreateDocumentation_Error_InvalidURLFormat() {
	req := &service.CreateDocumentationRequest{
		TeamID:      suite.testTeamID.String(),
		URL:         "not-a-valid-url",
		Title:       "Test Doc",
		Description: "Test",
		CreatedBy:   "test-user",
	}

	result, err := suite.documentationService.CreateDocumentation(req)

	suite.Error(err)
	suite.Nil(result)
	suite.Contains(err.Error(), "validation failed")
}

func (suite *DocumentationServiceTestSuite) TestCreateDocumentation_Error_MissingTitle() {
	req := &service.CreateDocumentationRequest{
		TeamID:      suite.testTeamID.String(),
		URL:         "https://github.tools.sap/org/repo",
		Title:       "",
		Description: "Test",
		CreatedBy:   "test-user",
	}

	result, err := suite.documentationService.CreateDocumentation(req)

	suite.Error(err)
	suite.Nil(result)
	suite.Contains(err.Error(), "validation failed")
}

func (suite *DocumentationServiceTestSuite) TestCreateDocumentation_Error_TitleTooLong() {
	req := &service.CreateDocumentationRequest{
		TeamID:      suite.testTeamID.String(),
		URL:         "https://github.tools.sap/org/repo",
		Title:       string(make([]byte, 101)), // 101 characters, max is 100
		Description: "Test",
		CreatedBy:   "test-user",
	}

	result, err := suite.documentationService.CreateDocumentation(req)

	suite.Error(err)
	suite.Nil(result)
	suite.Contains(err.Error(), "validation failed")
}

func (suite *DocumentationServiceTestSuite) TestCreateDocumentation_Error_DescriptionTooLong() {
	req := &service.CreateDocumentationRequest{
		TeamID:      suite.testTeamID.String(),
		URL:         "https://github.tools.sap/org/repo",
		Title:       "Test Doc",
		Description: string(make([]byte, 201)), // 201 characters, max is 200
		CreatedBy:   "test-user",
	}

	result, err := suite.documentationService.CreateDocumentation(req)

	suite.Error(err)
	suite.Nil(result)
	suite.Contains(err.Error(), "validation failed")
}

func (suite *DocumentationServiceTestSuite) TestCreateDocumentation_Error_URLTooLong() {
	req := &service.CreateDocumentationRequest{
		TeamID:      suite.testTeamID.String(),
		URL:         "https://github.tools.sap/" + string(make([]byte, 1000)), // Over 1000 chars
		Title:       "Test Doc",
		Description: "Test",
		CreatedBy:   "test-user",
	}

	result, err := suite.documentationService.CreateDocumentation(req)

	suite.Error(err)
	suite.Nil(result)
	suite.Contains(err.Error(), "validation failed")
}

func (suite *DocumentationServiceTestSuite) TestCreateDocumentation_Error_MissingCreatedBy() {
	req := &service.CreateDocumentationRequest{
		TeamID:      suite.testTeamID.String(),
		URL:         "https://github.tools.sap/org/repo",
		Title:       "Test Doc",
		Description: "Test",
		CreatedBy:   "",
	}

	result, err := suite.documentationService.CreateDocumentation(req)

	suite.Error(err)
	suite.Nil(result)
	suite.Contains(err.Error(), "created_by is required")
}

func (suite *DocumentationServiceTestSuite) TestCreateDocumentation_Error_CreatedByWhitespaceOnly() {
	req := &service.CreateDocumentationRequest{
		TeamID:      suite.testTeamID.String(),
		URL:         "https://github.tools.sap/org/repo",
		Title:       "Test Doc",
		Description: "Test",
		CreatedBy:   "   ",
	}

	result, err := suite.documentationService.CreateDocumentation(req)

	suite.Error(err)
	suite.Nil(result)
	suite.Contains(err.Error(), "created_by is required")
}

func (suite *DocumentationServiceTestSuite) TestCreateDocumentation_Error_InvalidTeamIDUUIDParsing() {

	req := &service.CreateDocumentationRequest{
		TeamID:      "invalid-uuid-format",
		URL:         "https://github.tools.sap/org/repo",
		Title:       "Test Doc",
		Description: "Test",
		CreatedBy:   "test-user",
	}

	result, err := suite.documentationService.CreateDocumentation(req)

	suite.Error(err)
	suite.Nil(result)

	suite.Contains(err.Error(), "validation failed")
}

func (suite *DocumentationServiceTestSuite) TestCreateDocumentation_Error_TeamNotFound() {
	req := &service.CreateDocumentationRequest{
		TeamID:      suite.testTeamID.String(),
		URL:         "https://github.tools.sap/org/repo",
		Title:       "Test Doc",
		Description: "Test",
		CreatedBy:   "test-user",
	}

	suite.mockTeamRepo.EXPECT().
		GetByID(suite.testTeamID).
		Return(nil, errors.New("record not found")).
		Times(1)

	result, err := suite.documentationService.CreateDocumentation(req)

	suite.Error(err)
	suite.Nil(result)
	suite.Contains(err.Error(), "team not found")
}

func (suite *DocumentationServiceTestSuite) TestCreateDocumentation_Error_InvalidGitHubHost() {
	req := &service.CreateDocumentationRequest{
		TeamID:      suite.testTeamID.String(),
		URL:         "https://gitlab.com/org/repo",
		Title:       "Test Doc",
		Description: "Test",
		CreatedBy:   "test-user",
	}

	suite.mockTeamRepo.EXPECT().
		GetByID(suite.testTeamID).
		Return(suite.testTeam, nil).
		Times(1)

	result, err := suite.documentationService.CreateDocumentation(req)

	suite.Error(err)
	suite.Nil(result)
	suite.Contains(err.Error(), "invalid GitHub URL")
}

func (suite *DocumentationServiceTestSuite) TestCreateDocumentation_Error_InvalidGitHubURLMissingRepo() {
	req := &service.CreateDocumentationRequest{
		TeamID:      suite.testTeamID.String(),
		URL:         "https://github.tools.sap/org",
		Title:       "Test Doc",
		Description: "Test",
		CreatedBy:   "test-user",
	}

	suite.mockTeamRepo.EXPECT().
		GetByID(suite.testTeamID).
		Return(suite.testTeam, nil).
		Times(1)

	result, err := suite.documentationService.CreateDocumentation(req)

	suite.Error(err)
	suite.Nil(result)
	suite.Contains(err.Error(), "invalid GitHub URL")
}

func (suite *DocumentationServiceTestSuite) TestCreateDocumentation_Error_InvalidGitHubURLWrongPathStructure() {
	req := &service.CreateDocumentationRequest{
		TeamID:      suite.testTeamID.String(),
		URL:         "https://github.tools.sap/org/repo/invalid/path",
		Title:       "Test Doc",
		Description: "Test",
		CreatedBy:   "test-user",
	}

	suite.mockTeamRepo.EXPECT().
		GetByID(suite.testTeamID).
		Return(suite.testTeam, nil).
		Times(1)

	result, err := suite.documentationService.CreateDocumentation(req)

	suite.Error(err)
	suite.Nil(result)
	suite.Contains(err.Error(), "invalid GitHub URL")
}

func (suite *DocumentationServiceTestSuite) TestCreateDocumentation_Error_RepositoryCreateFails() {
	req := &service.CreateDocumentationRequest{
		TeamID:      suite.testTeamID.String(),
		URL:         "https://github.tools.sap/org/repo",
		Title:       "Test Doc",
		Description: "Test",
		CreatedBy:   "test-user",
	}

	suite.mockTeamRepo.EXPECT().
		GetByID(suite.testTeamID).
		Return(suite.testTeam, nil).
		Times(1)

	suite.mockDocRepo.EXPECT().
		Create(gomock.Any()).
		Return(errors.New("database connection failed")).
		Times(1)

	result, err := suite.documentationService.CreateDocumentation(req)

	suite.Error(err)
	suite.Nil(result)
	suite.Contains(err.Error(), "failed to create documentation")
}

func (suite *DocumentationServiceTestSuite) TestGetDocumentationByID_Success() {
	docID := uuid.New()
	doc := &models.Documentation{
		TeamID:      suite.testTeamID,
		Owner:       "test-owner",
		Repo:        "test-repo",
		Branch:      "main",
		DocsPath:    "docs/api",
		Title:       "API Documentation",
		Description: "Complete API documentation",
		CreatedBy:   "test-user",
		UpdatedBy:   "test-user",
	}
	doc.ID = docID

	suite.mockDocRepo.EXPECT().
		GetByID(docID).
		Return(doc, nil).
		Times(1)

	result, err := suite.documentationService.GetDocumentationByID(docID)

	suite.NoError(err)
	suite.NotNil(result)
	suite.Equal(docID.String(), result.ID)
	suite.Equal(suite.testTeamID.String(), result.TeamID)
	suite.Equal("test-owner", result.Owner)
	suite.Equal("test-repo", result.Repo)
	suite.Equal("main", result.Branch)
	suite.Equal("docs/api", result.DocsPath)
	suite.Equal("API Documentation", result.Title)
	suite.Equal("Complete API documentation", result.Description)
	suite.Equal("test-user", result.CreatedBy)
	suite.Equal("test-user", result.UpdatedBy)
}

func (suite *DocumentationServiceTestSuite) TestGetDocumentationByID_NotFound() {
	docID := uuid.New()

	suite.mockDocRepo.EXPECT().
		GetByID(docID).
		Return(nil, errors.New("record not found")).
		Times(1)

	result, err := suite.documentationService.GetDocumentationByID(docID)

	suite.Error(err)
	suite.Nil(result)
	suite.Contains(err.Error(), "documentation not found")
}

func (suite *DocumentationServiceTestSuite) TestGetDocumentationByID_RepositoryError() {
	docID := uuid.New()

	suite.mockDocRepo.EXPECT().
		GetByID(docID).
		Return(nil, errors.New("database connection error")).
		Times(1)

	result, err := suite.documentationService.GetDocumentationByID(docID)

	suite.Error(err)
	suite.Nil(result)
	suite.Contains(err.Error(), "documentation not found")
}

func (suite *DocumentationServiceTestSuite) TestGetDocumentationsByTeamID_Success() {
	docs := []models.Documentation{
		{
			TeamID:      suite.testTeamID,
			Owner:       "owner1",
			Repo:        "repo1",
			Branch:      "main",
			DocsPath:    "docs",
			Title:       "Documentation 1",
			Description: "First documentation",
			CreatedBy:   "user1",
			UpdatedBy:   "user1",
		},
		{
			TeamID:      suite.testTeamID,
			Owner:       "owner2",
			Repo:        "repo2",
			Branch:      "develop",
			DocsPath:    "api/docs",
			Title:       "Documentation 2",
			Description: "Second documentation",
			CreatedBy:   "user2",
			UpdatedBy:   "user2",
		},
	}
	docs[0].ID = uuid.New()
	docs[1].ID = uuid.New()

	suite.mockTeamRepo.EXPECT().
		GetByID(suite.testTeamID).
		Return(suite.testTeam, nil).
		Times(1)

	suite.mockDocRepo.EXPECT().
		GetByTeamID(suite.testTeamID).
		Return(docs, nil).
		Times(1)

	result, err := suite.documentationService.GetDocumentationsByTeamID(suite.testTeamID)

	suite.NoError(err)
	suite.NotNil(result)
	suite.Len(result, 2)
	suite.Equal("Documentation 1", result[0].Title)
	suite.Equal("owner1", result[0].Owner)
	suite.Equal("repo1", result[0].Repo)
	suite.Equal("Documentation 2", result[1].Title)
	suite.Equal("owner2", result[1].Owner)
	suite.Equal("repo2", result[1].Repo)
}

func (suite *DocumentationServiceTestSuite) TestGetDocumentationsByTeamID_EmptyResult() {
	suite.mockTeamRepo.EXPECT().
		GetByID(suite.testTeamID).
		Return(suite.testTeam, nil).
		Times(1)

	suite.mockDocRepo.EXPECT().
		GetByTeamID(suite.testTeamID).
		Return([]models.Documentation{}, nil).
		Times(1)

	result, err := suite.documentationService.GetDocumentationsByTeamID(suite.testTeamID)

	suite.NoError(err)
	suite.NotNil(result)
	suite.Empty(result)
}

func (suite *DocumentationServiceTestSuite) TestGetDocumentationsByTeamID_TeamNotFound() {
	suite.mockTeamRepo.EXPECT().
		GetByID(suite.testTeamID).
		Return(nil, errors.New("record not found")).
		Times(1)

	result, err := suite.documentationService.GetDocumentationsByTeamID(suite.testTeamID)

	suite.Error(err)
	suite.Nil(result)
	suite.Contains(err.Error(), "team not found")
}

func (suite *DocumentationServiceTestSuite) TestGetDocumentationsByTeamID_RepositoryError() {
	suite.mockTeamRepo.EXPECT().
		GetByID(suite.testTeamID).
		Return(suite.testTeam, nil).
		Times(1)

	suite.mockDocRepo.EXPECT().
		GetByTeamID(suite.testTeamID).
		Return(nil, errors.New("database connection error")).
		Times(1)

	result, err := suite.documentationService.GetDocumentationsByTeamID(suite.testTeamID)

	suite.Error(err)
	suite.Nil(result)
	suite.Contains(err.Error(), "failed to get documentations")
}

func (suite *DocumentationServiceTestSuite) TestUpdateDocumentation_Success_AllFields() {
	docID := uuid.New()
	existingDoc := &models.Documentation{
		TeamID:      suite.testTeamID,
		Owner:       "old-owner",
		Repo:        "old-repo",
		Branch:      "old-branch",
		DocsPath:    "old/path",
		Title:       "Old Title",
		Description: "Old description",
		CreatedBy:   "creator",
		UpdatedBy:   "old-updater",
	}
	existingDoc.ID = docID

	newURL := "https://github.tools.sap/new-owner/new-repo/tree/main/docs/api"
	newTitle := "New Title"
	newDesc := "New description"
	req := &service.UpdateDocumentationRequest{
		URL:         &newURL,
		Title:       &newTitle,
		Description: &newDesc,
		UpdatedBy:   "new-updater",
	}

	suite.mockDocRepo.EXPECT().
		GetByID(docID).
		Return(existingDoc, nil).
		Times(1)

	suite.mockDocRepo.EXPECT().
		Update(gomock.Any()).
		DoAndReturn(func(doc *models.Documentation) error {
			suite.Equal("new-owner", doc.Owner)
			suite.Equal("new-repo", doc.Repo)
			suite.Equal("main", doc.Branch)
			suite.Equal("docs/api", doc.DocsPath)
			suite.Equal("New Title", doc.Title)
			suite.Equal("New description", doc.Description)
			suite.Equal("new-updater", doc.UpdatedBy)
			return nil
		}).
		Times(1)

	result, err := suite.documentationService.UpdateDocumentation(docID, req)

	suite.NoError(err)
	suite.NotNil(result)
	suite.Equal("new-owner", result.Owner)
	suite.Equal("new-repo", result.Repo)
	suite.Equal("New Title", result.Title)
	suite.Equal("New description", result.Description)
}

func (suite *DocumentationServiceTestSuite) TestUpdateDocumentation_Success_OnlyTitle() {
	docID := uuid.New()
	existingDoc := &models.Documentation{
		TeamID:      suite.testTeamID,
		Owner:       "owner",
		Repo:        "repo",
		Branch:      "main",
		DocsPath:    "docs",
		Title:       "Old Title",
		Description: "Old description",
		CreatedBy:   "creator",
		UpdatedBy:   "old-updater",
	}
	existingDoc.ID = docID

	newTitle := "Updated Title Only"
	req := &service.UpdateDocumentationRequest{
		Title:     &newTitle,
		UpdatedBy: "updater",
	}

	suite.mockDocRepo.EXPECT().
		GetByID(docID).
		Return(existingDoc, nil).
		Times(1)

	suite.mockDocRepo.EXPECT().
		Update(gomock.Any()).
		DoAndReturn(func(doc *models.Documentation) error {
			suite.Equal("Updated Title Only", doc.Title)
			suite.Equal("Old description", doc.Description) // Unchanged
			suite.Equal("owner", doc.Owner)                 // Unchanged
			return nil
		}).
		Times(1)

	result, err := suite.documentationService.UpdateDocumentation(docID, req)

	suite.NoError(err)
	suite.NotNil(result)
	suite.Equal("Updated Title Only", result.Title)
	suite.Equal("Old description", result.Description)
}

func (suite *DocumentationServiceTestSuite) TestUpdateDocumentation_Error_ValidationFailed() {
	docID := uuid.New()

	testCases := []struct {
		name        string
		setupReq    func() *service.UpdateDocumentationRequest
		description string
	}{
		{
			name: "TitleTooLong",
			setupReq: func() *service.UpdateDocumentationRequest {
				longTitle := string(make([]byte, 101)) // 101 characters, max is 100
				return &service.UpdateDocumentationRequest{
					Title:     &longTitle,
					UpdatedBy: "updater",
				}
			},
			description: "Title exceeds 100 characters",
		},
		{
			name: "DescriptionTooLong",
			setupReq: func() *service.UpdateDocumentationRequest {
				longDesc := string(make([]byte, 201)) // 201 characters, max is 200
				return &service.UpdateDocumentationRequest{
					Description: &longDesc,
					UpdatedBy:   "updater",
				}
			},
			description: "Description exceeds 200 characters",
		},
		{
			name: "URLTooLong",
			setupReq: func() *service.UpdateDocumentationRequest {
				longURL := "https://github.tools.sap/" + string(make([]byte, 1000))
				return &service.UpdateDocumentationRequest{
					URL:       &longURL,
					UpdatedBy: "updater",
				}
			},
			description: "URL exceeds 1000 characters",
		},
		{
			name: "InvalidURLFormat",
			setupReq: func() *service.UpdateDocumentationRequest {
				invalidURL := "not-a-valid-url-format"
				return &service.UpdateDocumentationRequest{
					URL:       &invalidURL,
					UpdatedBy: "updater",
				}
			},
			description: "URL format is invalid",
		},
		{
			name: "EmptyURL",
			setupReq: func() *service.UpdateDocumentationRequest {
				emptyURL := ""
				return &service.UpdateDocumentationRequest{
					URL:       &emptyURL,
					UpdatedBy: "updater",
				}
			},
			description: "URL is empty string",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			req := tc.setupReq()

			result, err := suite.documentationService.UpdateDocumentation(docID, req)

			suite.Error(err, tc.description)
			suite.Nil(result, tc.description)
			suite.Contains(err.Error(), "validation failed", tc.description)
		})
	}
}

func (suite *DocumentationServiceTestSuite) TestUpdateDocumentation_Error_MissingUpdatedBy() {
	docID := uuid.New()
	newTitle := "New Title"
	req := &service.UpdateDocumentationRequest{
		Title:     &newTitle,
		UpdatedBy: "",
	}

	result, err := suite.documentationService.UpdateDocumentation(docID, req)

	suite.Error(err)
	suite.Nil(result)
	suite.Contains(err.Error(), "updated_by is required")
}

func (suite *DocumentationServiceTestSuite) TestUpdateDocumentation_Error_UpdatedByWhitespaceOnly() {
	docID := uuid.New()
	newTitle := "New Title"
	req := &service.UpdateDocumentationRequest{
		Title:     &newTitle,
		UpdatedBy: "   ",
	}

	result, err := suite.documentationService.UpdateDocumentation(docID, req)

	suite.Error(err)
	suite.Nil(result)
	suite.Contains(err.Error(), "updated_by is required")
}

func (suite *DocumentationServiceTestSuite) TestUpdateDocumentation_Error_DocumentationNotFound() {
	docID := uuid.New()
	newTitle := "New Title"
	req := &service.UpdateDocumentationRequest{
		Title:     &newTitle,
		UpdatedBy: "updater",
	}

	suite.mockDocRepo.EXPECT().
		GetByID(docID).
		Return(nil, errors.New("record not found")).
		Times(1)

	result, err := suite.documentationService.UpdateDocumentation(docID, req)

	suite.Error(err)
	suite.Nil(result)
	suite.Contains(err.Error(), "documentation not found")
}

func (suite *DocumentationServiceTestSuite) TestUpdateDocumentation_Error_InvalidGitHubHost() {
	docID := uuid.New()
	existingDoc := &models.Documentation{
		TeamID:    suite.testTeamID,
		Owner:     "owner",
		Repo:      "repo",
		Branch:    "main",
		DocsPath:  "docs",
		Title:     "Title",
		CreatedBy: "creator",
	}
	existingDoc.ID = docID

	invalidURL := "https://gitlab.com/owner/repo"
	req := &service.UpdateDocumentationRequest{
		URL:       &invalidURL,
		UpdatedBy: "updater",
	}

	suite.mockDocRepo.EXPECT().
		GetByID(docID).
		Return(existingDoc, nil).
		Times(1)

	result, err := suite.documentationService.UpdateDocumentation(docID, req)

	suite.Error(err)
	suite.Nil(result)
	suite.Contains(err.Error(), "invalid GitHub URL")
}

func (suite *DocumentationServiceTestSuite) TestUpdateDocumentation_Error_InvalidGitHubURL_WrongPathType() {
	docID := uuid.New()
	existingDoc := &models.Documentation{
		TeamID:    suite.testTeamID,
		Owner:     "owner",
		Repo:      "repo",
		Branch:    "main",
		DocsPath:  "docs",
		Title:     "Title",
		CreatedBy: "creator",
	}
	existingDoc.ID = docID

	// URL with /commits/ instead of /tree/ or /blob/
	invalidURL := "https://github.tools.sap/owner/repo/commits/main/docs"
	req := &service.UpdateDocumentationRequest{
		URL:       &invalidURL,
		UpdatedBy: "updater",
	}

	suite.mockDocRepo.EXPECT().
		GetByID(docID).
		Return(existingDoc, nil).
		Times(1)

	result, err := suite.documentationService.UpdateDocumentation(docID, req)

	suite.Error(err)
	suite.Nil(result)
	suite.Contains(err.Error(), "invalid GitHub URL")
	suite.Contains(err.Error(), "expected /tree/ or /blob/")
}

func (suite *DocumentationServiceTestSuite) TestUpdateDocumentation_Error_RepositoryUpdateFails() {
	docID := uuid.New()
	existingDoc := &models.Documentation{
		TeamID:      suite.testTeamID,
		Owner:       "owner",
		Repo:        "repo",
		Branch:      "main",
		DocsPath:    "docs",
		Title:       "Old Title",
		Description: "Old description",
		CreatedBy:   "creator",
	}
	existingDoc.ID = docID

	newTitle := "New Title"
	req := &service.UpdateDocumentationRequest{
		Title:     &newTitle,
		UpdatedBy: "updater",
	}

	suite.mockDocRepo.EXPECT().
		GetByID(docID).
		Return(existingDoc, nil).
		Times(1)

	suite.mockDocRepo.EXPECT().
		Update(gomock.Any()).
		Return(errors.New("database connection error")).
		Times(1)

	result, err := suite.documentationService.UpdateDocumentation(docID, req)

	suite.Error(err)
	suite.Nil(result)
	suite.Contains(err.Error(), "failed to update documentation")
}

func (suite *DocumentationServiceTestSuite) TestDeleteDocumentation_Success() {
	docID := uuid.New()
	existingDoc := &models.Documentation{
		TeamID:      suite.testTeamID,
		Owner:       "owner",
		Repo:        "repo",
		Branch:      "main",
		DocsPath:    "docs",
		Title:       "Documentation to Delete",
		Description: "This will be deleted",
		CreatedBy:   "creator",
	}
	existingDoc.ID = docID

	suite.mockDocRepo.EXPECT().
		GetByID(docID).
		Return(existingDoc, nil).
		Times(1)

	suite.mockDocRepo.EXPECT().
		Delete(docID).
		Return(nil).
		Times(1)

	err := suite.documentationService.DeleteDocumentation(docID)

	suite.NoError(err)
}

func (suite *DocumentationServiceTestSuite) TestDeleteDocumentation_Error_NotFound() {
	docID := uuid.New()

	suite.mockDocRepo.EXPECT().
		GetByID(docID).
		Return(nil, errors.New("record not found")).
		Times(1)

	err := suite.documentationService.DeleteDocumentation(docID)

	suite.Error(err)
	suite.Contains(err.Error(), "documentation not found")
}

func (suite *DocumentationServiceTestSuite) TestDeleteDocumentation_Error_DeleteFails() {
	docID := uuid.New()
	existingDoc := &models.Documentation{
		TeamID:      suite.testTeamID,
		Owner:       "owner",
		Repo:        "repo",
		Branch:      "main",
		DocsPath:    "docs",
		Title:       "Documentation",
		Description: "Test",
		CreatedBy:   "creator",
	}
	existingDoc.ID = docID

	suite.mockDocRepo.EXPECT().
		GetByID(docID).
		Return(existingDoc, nil).
		Times(1)

	suite.mockDocRepo.EXPECT().
		Delete(docID).
		Return(errors.New("database connection error")).
		Times(1)

	err := suite.documentationService.DeleteDocumentation(docID)

	suite.Error(err)
	suite.Contains(err.Error(), "failed to delete documentation")
}

// TestDocumentationServiceTestSuite runs the test suite
func TestDocumentationServiceTestSuite(t *testing.T) {
	suite.Run(t, new(DocumentationServiceTestSuite))
}
