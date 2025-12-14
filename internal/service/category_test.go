package service_test

import (
	"errors"
	"testing"

	"developer-portal-backend/internal/database/models"
	"developer-portal-backend/internal/mocks"
	"developer-portal-backend/internal/service"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

type CategoryServiceTestSuite struct {
	suite.Suite
	ctrl            *gomock.Controller
	mockCategoryRepo *mocks.MockCategoryRepositoryInterface
	categoryService *service.CategoryService
	validator       *validator.Validate
}

func (suite *CategoryServiceTestSuite) SetupTest() {
	suite.ctrl = gomock.NewController(suite.T())
	suite.mockCategoryRepo = mocks.NewMockCategoryRepositoryInterface(suite.ctrl)
	suite.validator = validator.New()
	suite.categoryService = service.NewCategoryService(suite.mockCategoryRepo, suite.validator)
}

func (suite *CategoryServiceTestSuite) TearDownTest() {
	suite.ctrl.Finish()
}

func (suite *CategoryServiceTestSuite) TestListCategories_DefaultPagination_Success() {
	// page < 1 and pageSize invalid should normalize to page=1, pageSize=1000
	// Expect: repo.GetAll(limit=1000, offset=0)
	cats := []models.Category{
		{
			BaseModel: models.BaseModel{
				ID:          uuid.New(),
				Name:        "build",
				Title:       "Build",
				Description: "Build related links",
			},
			Icon:  "wrench",
			Color: "blue",
		},
		{
			BaseModel: models.BaseModel{
				ID:          uuid.New(),
				Name:        "monitoring",
				Title:       "Monitoring",
				Description: "Monitoring links",
			},
			Icon:  "eye",
			Color: "green",
		},
	}
	suite.mockCategoryRepo.EXPECT().GetAll(1000, 0).Return(cats, int64(2), nil)

	resp, err := suite.categoryService.GetAll(0, 0)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), resp)
	assert.Equal(suite.T(), int64(2), resp.Total)
	assert.Equal(suite.T(), 1, resp.Page)
	assert.Equal(suite.T(), 1000, resp.PageSize)
	assert.Len(suite.T(), resp.Categories, 2)
	assert.Equal(suite.T(), "build", resp.Categories[0].Name)
	assert.Equal(suite.T(), "wrench", resp.Categories[0].Icon)
	assert.Equal(suite.T(), "blue", resp.Categories[0].Color)
}

func (suite *CategoryServiceTestSuite) TestListCategories_CustomPagination_Success() {
	// page=2, pageSize=10 => offset=10
	cats := []models.Category{
		{
			BaseModel: models.BaseModel{
				ID:          uuid.New(),
				Name:        "security",
				Title:       "Security",
				Description: "Security tools",
			},
			Icon:  "shield",
			Color: "red",
		},
	}
	suite.mockCategoryRepo.EXPECT().GetAll(10, 10).Return(cats, int64(11), nil)

	resp, err := suite.categoryService.GetAll(2, 10)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), resp)
	assert.Equal(suite.T(), int64(11), resp.Total)
	assert.Equal(suite.T(), 2, resp.Page)
	assert.Equal(suite.T(), 10, resp.PageSize)
	assert.Len(suite.T(), resp.Categories, 1)
	assert.Equal(suite.T(), "security", resp.Categories[0].Name)
}

func (suite *CategoryServiceTestSuite) TestListCategories_BoundsNormalization_Success() {
	// page negative should normalize to 1; pageSize > 1000 should normalize to 1000
	cats := []models.Category{}
	// For page=-5, pageSize=5000 => limit=1000, offset=(1-1)*1000=0
	suite.mockCategoryRepo.EXPECT().GetAll(1000, 0).Return(cats, int64(0), nil)

	resp, err := suite.categoryService.GetAll(-5, 5000)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), resp)
	assert.Equal(suite.T(), 1, resp.Page)
	assert.Equal(suite.T(), 1000, resp.PageSize)
	assert.Len(suite.T(), resp.Categories, 0)
}

func (suite *CategoryServiceTestSuite) TestListCategories_ServiceError() {
	suite.mockCategoryRepo.EXPECT().GetAll(1000, 0).Return(nil, int64(0), errors.New("db failed"))

	resp, err := suite.categoryService.GetAll(0, 0)

	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), resp)
	assert.Contains(suite.T(), err.Error(), "failed to get categories")
}

func TestCategoryServiceTestSuite(t *testing.T) {
	suite.Run(t, new(CategoryServiceTestSuite))
}
