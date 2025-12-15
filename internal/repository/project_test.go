package repository

import (
	"testing"

	"developer-portal-backend/internal/testutils"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
)

// ProjectRepositoryTestSuite tests the ProjectRepository
type ProjectRepositoryTestSuite struct {
	suite.Suite
	baseTestSuite *testutils.BaseTestSuite
	repo          *ProjectRepository
	factories     *testutils.FactorySet
}

// SetupSuite runs before all tests in the suite
func (suite *ProjectRepositoryTestSuite) SetupSuite() {
	suite.baseTestSuite = testutils.SetupTestSuite(suite.T())

	suite.repo = NewProjectRepository(suite.baseTestSuite.DB)
	suite.factories = testutils.NewFactorySet()
}

// TearDownSuite runs after all tests in the suite
func (suite *ProjectRepositoryTestSuite) TearDownSuite() {
	suite.baseTestSuite.TeardownTestSuite()
}

// SetupTest runs before each test
func (suite *ProjectRepositoryTestSuite) SetupTest() {
	suite.baseTestSuite.SetupTest()
}

// TearDownTest runs after each test
func (suite *ProjectRepositoryTestSuite) TearDownTest() {
	suite.baseTestSuite.TearDownTest()
}

// TestCreate tests creating a new project
func (suite *ProjectRepositoryTestSuite) TestCreate() {
	// Create organization first
	org := suite.factories.Organization.Create()
	orgRepo := NewOrganizationRepository(suite.baseTestSuite.DB)
	err := orgRepo.Create(org)
	suite.NoError(err)

	// Create test project
	project := suite.factories.Project.Create()
	// NOTE: OrganizationID removed from Project model in new schema

	// Create the project
	err = suite.repo.Create(project)

	// Assertions
	suite.NoError(err)
	suite.NotEqual(uuid.Nil, project.ID)
	suite.NotZero(project.CreatedAt)
	suite.NotZero(project.UpdatedAt)
}

// TestCreateDuplicateName tests creating a project with duplicate name in same organization
func (suite *ProjectRepositoryTestSuite) TestCreateDuplicateName() {
	// Create organization first
	org := suite.factories.Organization.Create()
	orgRepo := NewOrganizationRepository(suite.baseTestSuite.DB)
	err := orgRepo.Create(org)
	suite.NoError(err)

	// Create first project
	project1 := suite.factories.Project.WithName("duplicate-project")
	// NOTE: OrganizationID removed from Project model in new schema
	err = suite.repo.Create(project1)
	suite.NoError(err)

	// Create second project with same name - should fail due to unique constraint on name
	project2 := suite.factories.Project.WithName("duplicate-project")
	// NOTE: OrganizationID removed from Project model in new schema

	err = suite.repo.Create(project2)
	suite.Error(err)
	suite.Contains(err.Error(), "projects_name_unique")
}

// TestCreateSameNameDifferentOrg tests creating projects with same name (REMOVED - OrganizationID no longer on Project)
// func (suite *ProjectRepositoryTestSuite) TestCreateSameNameDifferentOrg() {
// 	// NOTE: OrganizationID removed from Project model in new schema
// }

// TestGetByID tests retrieving a project by ID
func (suite *ProjectRepositoryTestSuite) TestGetByID() {
	// Create organization first
	org := suite.factories.Organization.Create()
	orgRepo := NewOrganizationRepository(suite.baseTestSuite.DB)
	err := orgRepo.Create(org)
	suite.NoError(err)

	// Create test project
	project := suite.factories.Project.Create()
	// NOTE: OrganizationID removed from Project model in new schema
	err = suite.repo.Create(project)
	suite.NoError(err)

	// Retrieve the project
	retrievedProject, err := suite.repo.GetByID(project.ID)

	// Assertions
	suite.NoError(err)
	suite.NotNil(retrievedProject)
	suite.Equal(project.ID, retrievedProject.ID)
	suite.Equal(project.Name, retrievedProject.Name)
	suite.Equal(project.Title, retrievedProject.Title)
}

// TestGetByIDNotFound tests retrieving a non-existent project
func (suite *ProjectRepositoryTestSuite) TestGetByIDNotFound() {
	nonExistentID := uuid.New()

	project, err := suite.repo.GetByID(nonExistentID)

	suite.Error(err)
	suite.Equal(gorm.ErrRecordNotFound, err)
	suite.Nil(project)
}

// TestGetByOrganizationID tests listing projects by organization (REMOVED - OrganizationID no longer on Project)
// func (suite *ProjectRepositoryTestSuite) TestGetByOrganizationID() {
// 	// NOTE: OrganizationID removed from Project model in new schema
// }

// TestGetByOrganizationIDWithPagination tests listing projects with pagination (REMOVED - OrganizationID no longer on Project)
// func (suite *ProjectRepositoryTestSuite) TestGetByOrganizationIDWithPagination() {
// 	// NOTE: OrganizationID removed from Project model in new schema
// }

// TestUpdate tests updating a project
func (suite *ProjectRepositoryTestSuite) TestUpdate() {
	// Create organization first
	org := suite.factories.Organization.Create()
	orgRepo := NewOrganizationRepository(suite.baseTestSuite.DB)
	err := orgRepo.Create(org)
	suite.NoError(err)

	// Create test project
	project := suite.factories.Project.Create()
	// NOTE: OrganizationID removed from Project model in new schema
	err = suite.repo.Create(project)
	suite.NoError(err)

	// Update the project
	project.Title = "Updated Project Title"
	project.Description = "Updated project description"

	err = suite.repo.Update(project)

	// Assertions
	suite.NoError(err)

	// Retrieve updated project
	updatedProject, err := suite.repo.GetByID(project.ID)
	suite.NoError(err)
	suite.Equal("Updated Project Title", updatedProject.Title)
	suite.Equal("Updated project description", updatedProject.Description)
	suite.True(updatedProject.UpdatedAt.After(updatedProject.CreatedAt))
}

// TestDelete tests deleting a project
func (suite *ProjectRepositoryTestSuite) TestDelete() {
	// Create organization first
	org := suite.factories.Organization.Create()
	orgRepo := NewOrganizationRepository(suite.baseTestSuite.DB)
	err := orgRepo.Create(org)
	suite.NoError(err)

	// Create test project
	project := suite.factories.Project.Create()
	// NOTE: OrganizationID removed from Project model in new schema
	err = suite.repo.Create(project)
	suite.NoError(err)

	// Delete the project
	err = suite.repo.Delete(project.ID)
	suite.NoError(err)

	// Verify project is deleted
	_, err = suite.repo.GetByID(project.ID)
	suite.Error(err)
	suite.Equal(gorm.ErrRecordNotFound, err)
}

// TestDeleteNotFound tests deleting a non-existent project
func (suite *ProjectRepositoryTestSuite) TestDeleteNotFound() {
	nonExistentID := uuid.New()

	err := suite.repo.Delete(nonExistentID)

	// Should not error when deleting non-existent record
	suite.NoError(err)
}

// TestGetByName tests retrieving a project by name
func (suite *ProjectRepositoryTestSuite) TestGetByName() {
	// Create test project
	project := suite.factories.Project.WithName("unique-project-name")
	// NOTE: OrganizationID removed from Project model in new schema
	err := suite.repo.Create(project)
	suite.NoError(err)

	// Retrieve the project by name
	retrievedProject, err := suite.repo.GetByName("unique-project-name")

	// Assertions
	suite.NoError(err)
	suite.NotNil(retrievedProject)
	suite.Equal(project.ID, retrievedProject.ID)
	suite.Equal("unique-project-name", retrievedProject.Name)
}

// TestGetByNameNotFound tests retrieving a non-existent project by name
func (suite *ProjectRepositoryTestSuite) TestGetByNameNotFound() {
	project, err := suite.repo.GetByName("nonexistent-project")

	suite.Error(err)
	suite.Equal(gorm.ErrRecordNotFound, err)
	suite.Nil(project)
}

// TestGetByStatus tests retrieving projects by status (REMOVED - Status field no longer on Project)
// func (suite *ProjectRepositoryTestSuite) TestGetByStatus() {
// 	// NOTE: Status field removed from Project model in new schema
// }

// TestGetActiveProjects tests retrieving active projects (REMOVED - Status field no longer on Project)
// func (suite *ProjectRepositoryTestSuite) TestGetActiveProjects() {
// 	// NOTE: Status field removed from Project model in new schema
// }

func (suite *ProjectRepositoryTestSuite) TestGetHealthURL_WithHealth() {
	// Create project with metadata.health template
	project := suite.factories.Project.WithName("project-health-1")
	project.Metadata = []byte(`{"health":"https://health.ingress.{landscape_domain}{health_suffix}"}`)

	err := suite.repo.Create(project)
	suite.NoError(err)

	url, err := suite.repo.GetHealthURL(project.ID)
	suite.NoError(err)
	suite.Equal("https://health.ingress.{landscape_domain}{health_suffix}", url)
}

func (suite *ProjectRepositoryTestSuite) TestGetHealthURL_NoMetadata() {
	// Create project with nil metadata
	project := suite.factories.Project.WithName("project-health-2")
	project.Metadata = nil

	err := suite.repo.Create(project)
	suite.NoError(err)

	url, err := suite.repo.GetHealthURL(project.ID)
	suite.NoError(err)
	suite.Equal("", url)
}

func (suite *ProjectRepositoryTestSuite) TestGetHealthURL_NoHealthField() {
	// Create project with metadata but without health field
	project := suite.factories.Project.WithName("project-health-3")
	project.Metadata = []byte(`{"views":["grid"]}`)

	err := suite.repo.Create(project)
	suite.NoError(err)

	url, err := suite.repo.GetHealthURL(project.ID)
	suite.NoError(err)
	suite.Equal("", url)
}

func (suite *ProjectRepositoryTestSuite) TestGetHealthURL_ProjectNotFound() {
	// Call with non-existent project ID
	_, err := suite.repo.GetHealthURL(uuid.New())
	suite.Error(err)
	suite.Equal(gorm.ErrRecordNotFound, err)
}

// Run the test suite
func TestProjectRepositoryTestSuite(t *testing.T) {
	suite.Run(t, new(ProjectRepositoryTestSuite))
}
