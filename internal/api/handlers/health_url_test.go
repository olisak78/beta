package handlers

import (
	"encoding/json"
	"testing"

	"developer-portal-backend/internal/database/models"
	apperrors "developer-portal-backend/internal/errors"
	"developer-portal-backend/internal/mocks"
	"developer-portal-backend/internal/service"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

// stubProjectService implements ProjectServiceInterface for template-based tests
type stubProjectService struct {
	tmpl  string
	regex string
}

func (s stubProjectService) GetAllProjects() ([]models.Project, error) { return nil, nil }
func (s stubProjectService) GetHealthMetadata(projectID uuid.UUID) (string, string, error) {
	return s.tmpl, s.regex, nil
}

func TestBuildComponentHealthURL_TemplateWithSubdomainAndSuffix(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	componentSvc := mocks.NewMockComponentServiceInterface(ctrl)
	landscapeSvc := mocks.NewMockLandscapeServiceInterface(ctrl)
	projectSvc := stubProjectService{tmpl: "https://{subdomain}.{component_name}.cfapps.{landscape_domain}/health{health_suffix}", regex: "OK"}

	componentID := uuid.New()
	projectID := uuid.New()
	landscapeID := uuid.New()

	metadata := json.RawMessage(`{"subdomain":"api","health_suffix":"/tenant/x","health":true}`)
	component := &models.Component{
		BaseModel: models.BaseModel{
			ID:       componentID,
			Name:     "comp",
			Metadata: metadata,
		},
		ProjectID: projectID,
	}
	componentSvc.EXPECT().GetByID(componentID).Return(component, nil)

	landscape := &service.LandscapeResponse{ID: landscapeID, Domain: "example.com"}
	landscapeSvc.EXPECT().GetLandscapeByID(landscapeID).Return(landscape, nil)

	url, successRegex, err := BuildComponentHealthURL(componentSvc, landscapeSvc, projectSvc, componentID, landscapeID)
	assert.NoError(t, err)
	assert.Equal(t, "https://api.comp.cfapps.example.com/health/tenant/x", url)
	assert.Equal(t, "OK", successRegex)
}

func TestBuildComponentHealthURL_TemplateWithoutSubdomain(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	componentSvc := mocks.NewMockComponentServiceInterface(ctrl)
	landscapeSvc := mocks.NewMockLandscapeServiceInterface(ctrl)
	projectSvc := stubProjectService{tmpl: "https://{subdomain}.{component_name}.cfapps.{landscape_domain}/health", regex: "OK"}

	componentID := uuid.New()
	projectID := uuid.New()
	landscapeID := uuid.New()

	metadata := json.RawMessage(`{"health":true}`)
	component := &models.Component{
		BaseModel: models.BaseModel{
			ID:       componentID,
			Name:     "comp",
			Metadata: metadata,
		},
		ProjectID: projectID,
	}
	componentSvc.EXPECT().GetByID(componentID).Return(component, nil)

	landscape := &service.LandscapeResponse{ID: landscapeID, Domain: "example.com"}
	landscapeSvc.EXPECT().GetLandscapeByID(landscapeID).Return(landscape, nil)

	url, successRegex, err := BuildComponentHealthURL(componentSvc, landscapeSvc, projectSvc, componentID, landscapeID)
	assert.NoError(t, err)
	assert.Equal(t, "https://comp.cfapps.example.com/health", url)
	assert.Equal(t, "OK", successRegex)
}

func TestBuildComponentHealthURL_TemplateIngressSuffixPresent(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	componentSvc := mocks.NewMockComponentServiceInterface(ctrl)
	landscapeSvc := mocks.NewMockLandscapeServiceInterface(ctrl)
	projectSvc := stubProjectService{tmpl: "https://health.ingress.{landscape_domain}{health_suffix}", regex: "OK"}

	componentID := uuid.New()
	projectID := uuid.New()
	landscapeID := uuid.New()

	metadata := json.RawMessage(`{"health_suffix":"/tenant/test","health":true}`)
	component := &models.Component{
		BaseModel: models.BaseModel{
			ID:       componentID,
			Name:     "comp",
			Metadata: metadata,
		},
		ProjectID: projectID,
	}
	componentSvc.EXPECT().GetByID(componentID).Return(component, nil)

	landscape := &service.LandscapeResponse{ID: landscapeID, Domain: "example.com"}
	landscapeSvc.EXPECT().GetLandscapeByID(landscapeID).Return(landscape, nil)

	url, successRegex, err := BuildComponentHealthURL(componentSvc, landscapeSvc, projectSvc, componentID, landscapeID)
	assert.NoError(t, err)
	assert.Equal(t, "https://health.ingress.example.com/tenant/test", url)
	assert.Equal(t, "OK", successRegex)
}

func TestBuildComponentHealthURL_TemplateIngressSuffixMissing(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	componentSvc := mocks.NewMockComponentServiceInterface(ctrl)
	landscapeSvc := mocks.NewMockLandscapeServiceInterface(ctrl)
	projectSvc := stubProjectService{tmpl: "https://health.ingress.{landscape_domain}{health_suffix}", regex: "OK"}

	componentID := uuid.New()
	projectID := uuid.New()
	landscapeID := uuid.New()

	metadata := json.RawMessage(`{"health":true}`)
	component := &models.Component{
		BaseModel: models.BaseModel{
			ID:       componentID,
			Name:     "comp",
			Metadata: metadata,
		},
		ProjectID: projectID,
	}
	componentSvc.EXPECT().GetByID(componentID).Return(component, nil)

	landscape := &service.LandscapeResponse{ID: landscapeID, Domain: "example.com"}
	landscapeSvc.EXPECT().GetLandscapeByID(landscapeID).Return(landscape, nil)

	url, successRegex, err := BuildComponentHealthURL(componentSvc, landscapeSvc, projectSvc, componentID, landscapeID)
	assert.NoError(t, err)
	assert.Equal(t, "https://health.ingress.example.com", url)
	assert.Equal(t, "OK", successRegex)
}

func TestBuildComponentHealthURL_FallbackWithSubdomain(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	componentSvc := mocks.NewMockComponentServiceInterface(ctrl)
	landscapeSvc := mocks.NewMockLandscapeServiceInterface(ctrl)

	componentID := uuid.New()
	projectID := uuid.New()
	landscapeID := uuid.New()

	metadata := json.RawMessage(`{"subdomain":"api","health":true}`)
	component := &models.Component{
		BaseModel: models.BaseModel{
			ID:       componentID,
			Name:     "comp",
			Metadata: metadata,
		},
		ProjectID: projectID,
	}
	componentSvc.EXPECT().GetByID(componentID).Return(component, nil)

	landscape := &service.LandscapeResponse{ID: landscapeID, Domain: "example.com"}
	landscapeSvc.EXPECT().GetLandscapeByID(landscapeID).Return(landscape, nil)

	url, successRegex, err := BuildComponentHealthURL(componentSvc, landscapeSvc, nil, componentID, landscapeID)
	assert.Error(t, err)
	assert.Equal(t, "", url)
	assert.Equal(t, "", successRegex)
	assert.Equal(t, ErrComponentHealthBadConfig, err)
}

func TestBuildComponentHealthURL_FallbackNoSubdomain(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	componentSvc := mocks.NewMockComponentServiceInterface(ctrl)
	landscapeSvc := mocks.NewMockLandscapeServiceInterface(ctrl)

	componentID := uuid.New()
	projectID := uuid.New()
	landscapeID := uuid.New()

	metadata := json.RawMessage(`{"health":true}`)
	component := &models.Component{
		BaseModel: models.BaseModel{
			ID:       componentID,
			Name:     "comp",
			Metadata: metadata,
		},
		ProjectID: projectID,
	}
	componentSvc.EXPECT().GetByID(componentID).Return(component, nil)

	landscape := &service.LandscapeResponse{ID: landscapeID, Domain: "example.com"}
	landscapeSvc.EXPECT().GetLandscapeByID(landscapeID).Return(landscape, nil)

	url, successRegex, err := BuildComponentHealthURL(componentSvc, landscapeSvc, nil, componentID, landscapeID)
	assert.Error(t, err)
	assert.Equal(t, "", url)
	assert.Equal(t, "", successRegex)
	assert.Equal(t, ErrComponentHealthBadConfig, err)
}

func TestBuildComponentHealthURL_HealthDisabled(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	componentSvc := mocks.NewMockComponentServiceInterface(ctrl)
	landscapeSvc := mocks.NewMockLandscapeServiceInterface(ctrl)

	componentID := uuid.New()
	projectID := uuid.New()
	landscapeID := uuid.New()

	metadata := json.RawMessage(`{"health":false}`)
	component := &models.Component{
		BaseModel: models.BaseModel{
			ID:       componentID,
			Name:     "comp",
			Metadata: metadata,
		},
		ProjectID: projectID,
	}
	componentSvc.EXPECT().GetByID(componentID).Return(component, nil)

	landscape := &service.LandscapeResponse{ID: landscapeID, Domain: "example.com"}
	landscapeSvc.EXPECT().GetLandscapeByID(landscapeID).Return(landscape, nil)

	url, successRegex, err := BuildComponentHealthURL(componentSvc, landscapeSvc, nil, componentID, landscapeID)
	assert.Error(t, err)
	assert.Equal(t, "", url)
	assert.Equal(t, "", successRegex)
	assert.Equal(t, ErrComponentHealthDisabled, err)
}

func TestBuildComponentHealthURL_LandscapeNotConfigured(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	componentSvc := mocks.NewMockComponentServiceInterface(ctrl)

	componentID := uuid.New()
	projectID := uuid.New()
	landscapeID := uuid.New()

	metadata := json.RawMessage(`{"health":true}`)
	component := &models.Component{
		BaseModel: models.BaseModel{
			ID:       componentID,
			Name:     "comp",
			Metadata: metadata,
		},
		ProjectID: projectID,
	}
	componentSvc.EXPECT().GetByID(componentID).Return(component, nil)

	url, successRegex, err := BuildComponentHealthURL(componentSvc, nil, nil, componentID, landscapeID)
	assert.Error(t, err)
	assert.Equal(t, "", url)
	assert.Equal(t, "", successRegex)
	assert.Equal(t, apperrors.ErrLandscapeNotConfigured, err)
}

func TestBuildComponentHealthURL_ComponentNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	componentSvc := mocks.NewMockComponentServiceInterface(ctrl)
	landscapeSvc := mocks.NewMockLandscapeServiceInterface(ctrl)

	componentID := uuid.New()
	landscapeID := uuid.New()

	componentSvc.EXPECT().GetByID(componentID).Return(nil, apperrors.ErrComponentNotFound)

	url, successRegex, err := BuildComponentHealthURL(componentSvc, landscapeSvc, nil, componentID, landscapeID)
	assert.Error(t, err)
	assert.Equal(t, "", url)
	assert.Equal(t, "", successRegex)
	assert.Equal(t, apperrors.ErrComponentNotFound, err)
}

func TestBuildComponentHealthURL_LandscapeNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	componentSvc := mocks.NewMockComponentServiceInterface(ctrl)
	landscapeSvc := mocks.NewMockLandscapeServiceInterface(ctrl)

	componentID := uuid.New()
	projectID := uuid.New()
	landscapeID := uuid.New()

	metadata := json.RawMessage(`{"health":true}`)
	component := &models.Component{
		BaseModel: models.BaseModel{
			ID:       componentID,
			Name:     "comp",
			Metadata: metadata,
		},
		ProjectID: projectID,
	}
	componentSvc.EXPECT().GetByID(componentID).Return(component, nil)
	landscapeSvc.EXPECT().GetLandscapeByID(landscapeID).Return(nil, apperrors.ErrLandscapeNotFound)

	url, successRegex, err := BuildComponentHealthURL(componentSvc, landscapeSvc, nil, componentID, landscapeID)
	assert.Error(t, err)
	assert.Equal(t, "", url)
	assert.Equal(t, "", successRegex)
	assert.Equal(t, apperrors.ErrLandscapeNotFound, err)
}
