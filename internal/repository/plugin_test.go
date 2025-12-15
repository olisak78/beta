package repository

import (
	"testing"

	"developer-portal-backend/internal/database/models"
	"developer-portal-backend/internal/testutils"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type PluginRepositoryTestSuite struct {
	suite.Suite
	*testutils.BaseTestSuite
	repo *PluginRepository
}

func (suite *PluginRepositoryTestSuite) SetupSuite() {
	suite.BaseTestSuite = testutils.SetupTestSuite(suite.T())
	suite.repo = NewPluginRepository(suite.DB)
}

func (suite *PluginRepositoryTestSuite) TearDownTest() {
	// Clean up after each test - this is handled by BaseTestSuite
	suite.CleanTestDB()
}

func (suite *PluginRepositoryTestSuite) TestNewPluginRepository() {
	repo := NewPluginRepository(suite.DB)
	assert.NotNil(suite.T(), repo)
	assert.Equal(suite.T(), suite.DB, repo.db)
}

func (suite *PluginRepositoryTestSuite) TestCreate() {
	plugin := &models.Plugin{
		BaseModel: models.BaseModel{
			Name:        "test-plugin",
			Title:       "Test Plugin",
			Description: "A test plugin for testing",
		},
		Icon:               "TestIcon",
		ReactComponentPath: "/plugins/test/Test.jsx",
		BackendServerURL:   "http://localhost:3001",
		Owner:              "Test Team",
	}

	err := suite.repo.Create(plugin)
	assert.NoError(suite.T(), err)
	assert.NotEqual(suite.T(), uuid.Nil, plugin.ID)

	// Verify the plugin was created in the database
	var dbPlugin models.Plugin
	err = suite.DB.First(&dbPlugin, "id = ?", plugin.ID).Error
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), plugin.Name, dbPlugin.Name)
	assert.Equal(suite.T(), plugin.Title, dbPlugin.Title)
	assert.Equal(suite.T(), plugin.Description, dbPlugin.Description)
	assert.Equal(suite.T(), plugin.Icon, dbPlugin.Icon)
	assert.Equal(suite.T(), plugin.ReactComponentPath, dbPlugin.ReactComponentPath)
	assert.Equal(suite.T(), plugin.BackendServerURL, dbPlugin.BackendServerURL)
	assert.Equal(suite.T(), plugin.Owner, dbPlugin.Owner)
}

func (suite *PluginRepositoryTestSuite) TestCreate_DuplicateName() {
	plugin1 := &models.Plugin{
		BaseModel: models.BaseModel{
			Name:        "duplicate-plugin",
			Title:       "First Plugin",
			Description: "First plugin with duplicate name",
		},
		Icon:               "Icon1",
		ReactComponentPath: "/plugins/first/First.jsx",
		BackendServerURL:   "http://localhost:3001",
		Owner:              "Team 1",
	}

	plugin2 := &models.Plugin{
		BaseModel: models.BaseModel{
			Name:        "duplicate-plugin",
			Title:       "Second Plugin",
			Description: "Second plugin with duplicate name",
		},
		Icon:               "Icon2",
		ReactComponentPath: "/plugins/second/Second.jsx",
		BackendServerURL:   "http://localhost:3002",
		Owner:              "Team 2",
	}

	// First creation should succeed
	err := suite.repo.Create(plugin1)
	assert.NoError(suite.T(), err)

	// Second creation with same name should succeed since there's no unique constraint
	err = suite.repo.Create(plugin2)
	assert.NoError(suite.T(), err)
}

func (suite *PluginRepositoryTestSuite) TestGetByID() {
	// Create a plugin first
	plugin := &models.Plugin{
		BaseModel: models.BaseModel{
			Name:        "get-by-id-plugin",
			Title:       "Get By ID Plugin",
			Description: "Plugin for testing GetByID",
		},
		Icon:               "GetByIDIcon",
		ReactComponentPath: "/plugins/getbyid/GetByID.jsx",
		BackendServerURL:   "http://localhost:3003",
		Owner:              "GetByID Team",
	}

	err := suite.repo.Create(plugin)
	assert.NoError(suite.T(), err)

	// Test successful retrieval
	retrievedPlugin, err := suite.repo.GetByID(plugin.ID)
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), retrievedPlugin)
	assert.Equal(suite.T(), plugin.ID, retrievedPlugin.ID)
	assert.Equal(suite.T(), plugin.Name, retrievedPlugin.Name)
	assert.Equal(suite.T(), plugin.Title, retrievedPlugin.Title)

	// Test retrieval with non-existent ID
	nonExistentID := uuid.New()
	retrievedPlugin, err = suite.repo.GetByID(nonExistentID)
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), retrievedPlugin)
	assert.Contains(suite.T(), err.Error(), "record not found")
}

func (suite *PluginRepositoryTestSuite) TestGetByName() {
	// Create a plugin first
	plugin := &models.Plugin{
		BaseModel: models.BaseModel{
			Name:        "get-by-name-plugin",
			Title:       "Get By Name Plugin",
			Description: "Plugin for testing GetByName",
		},
		Icon:               "GetByNameIcon",
		ReactComponentPath: "/plugins/getbyname/GetByName.jsx",
		BackendServerURL:   "http://localhost:3004",
		Owner:              "GetByName Team",
	}

	err := suite.repo.Create(plugin)
	assert.NoError(suite.T(), err)

	// Test successful retrieval
	retrievedPlugin, err := suite.repo.GetByName(plugin.Name)
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), retrievedPlugin)
	assert.Equal(suite.T(), plugin.ID, retrievedPlugin.ID)
	assert.Equal(suite.T(), plugin.Name, retrievedPlugin.Name)
	assert.Equal(suite.T(), plugin.Title, retrievedPlugin.Title)

	// Test retrieval with non-existent name
	retrievedPlugin, err = suite.repo.GetByName("non-existent-plugin")
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), retrievedPlugin)
	assert.Contains(suite.T(), err.Error(), "record not found")
}

func (suite *PluginRepositoryTestSuite) TestGetAll() {
	// Create multiple plugins
	plugins := []*models.Plugin{
		{
			BaseModel: models.BaseModel{
				Name:        "plugin-1",
				Title:       "Plugin 1",
				Description: "First plugin",
			},
			Icon:               "Icon1",
			ReactComponentPath: "/plugins/1/Plugin1.jsx",
			BackendServerURL:   "http://localhost:3001",
			Owner:              "Team 1",
		},
		{
			BaseModel: models.BaseModel{
				Name:        "plugin-2",
				Title:       "Plugin 2",
				Description: "Second plugin",
			},
			Icon:               "Icon2",
			ReactComponentPath: "/plugins/2/Plugin2.jsx",
			BackendServerURL:   "http://localhost:3002",
			Owner:              "Team 2",
		},
		{
			BaseModel: models.BaseModel{
				Name:        "plugin-3",
				Title:       "Plugin 3",
				Description: "Third plugin",
			},
			Icon:               "Icon3",
			ReactComponentPath: "/plugins/3/Plugin3.jsx",
			BackendServerURL:   "http://localhost:3003",
			Owner:              "Team 3",
		},
	}

	for _, plugin := range plugins {
		err := suite.repo.Create(plugin)
		assert.NoError(suite.T(), err)
	}

	// Test getting all plugins without pagination
	retrievedPlugins, total, err := suite.repo.GetAll(10, 0)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), int64(3), total)
	assert.Len(suite.T(), retrievedPlugins, 3)

	// Test pagination - first page
	retrievedPlugins, total, err = suite.repo.GetAll(2, 0)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), int64(3), total)
	assert.Len(suite.T(), retrievedPlugins, 2)

	// Test pagination - second page
	retrievedPlugins, total, err = suite.repo.GetAll(2, 2)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), int64(3), total)
	assert.Len(suite.T(), retrievedPlugins, 1)

	// Test pagination - beyond available records
	retrievedPlugins, total, err = suite.repo.GetAll(10, 10)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), int64(3), total)
	assert.Len(suite.T(), retrievedPlugins, 0)

	// Test with empty database
	suite.DB.Exec("DELETE FROM plugins")
	retrievedPlugins, total, err = suite.repo.GetAll(10, 0)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), int64(0), total)
	assert.Len(suite.T(), retrievedPlugins, 0)
}

func (suite *PluginRepositoryTestSuite) TestUpdate() {
	// Create a plugin first
	plugin := &models.Plugin{
		BaseModel: models.BaseModel{
			Name:        "update-plugin",
			Title:       "Update Plugin",
			Description: "Plugin for testing update",
		},
		Icon:               "UpdateIcon",
		ReactComponentPath: "/plugins/update/Update.jsx",
		BackendServerURL:   "http://localhost:3005",
		Owner:              "Update Team",
	}

	err := suite.repo.Create(plugin)
	assert.NoError(suite.T(), err)

	// Update the plugin
	plugin.Title = "Updated Plugin Title"
	plugin.Description = "Updated plugin description"
	plugin.Icon = "UpdatedIcon"
	plugin.ReactComponentPath = "/plugins/updated/Updated.jsx"
	plugin.BackendServerURL = "http://localhost:3006"
	plugin.Owner = "Updated Team"

	err = suite.repo.Update(plugin)
	assert.NoError(suite.T(), err)

	// Verify the update
	retrievedPlugin, err := suite.repo.GetByID(plugin.ID)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "Updated Plugin Title", retrievedPlugin.Title)
	assert.Equal(suite.T(), "Updated plugin description", retrievedPlugin.Description)
	assert.Equal(suite.T(), "UpdatedIcon", retrievedPlugin.Icon)
	assert.Equal(suite.T(), "/plugins/updated/Updated.jsx", retrievedPlugin.ReactComponentPath)
	assert.Equal(suite.T(), "http://localhost:3006", retrievedPlugin.BackendServerURL)
	assert.Equal(suite.T(), "Updated Team", retrievedPlugin.Owner)
}

func (suite *PluginRepositoryTestSuite) TestUpdate_NonExistentPlugin() {
	// Try to update a plugin that doesn't exist
	nonExistentPlugin := &models.Plugin{
		BaseModel: models.BaseModel{
			ID:          uuid.New(),
			Name:        "non-existent",
			Title:       "Non Existent",
			Description: "This plugin doesn't exist",
		},
		Icon:               "NonExistentIcon",
		ReactComponentPath: "/plugins/nonexistent/NonExistent.jsx",
		BackendServerURL:   "http://localhost:3007",
		Owner:              "Non Existent Team",
	}

	err := suite.repo.Update(nonExistentPlugin)
	// GORM's Save method will create the record if it doesn't exist
	// So this should not error, but let's verify the behavior
	assert.NoError(suite.T(), err)

	// Verify it was created
	retrievedPlugin, err := suite.repo.GetByID(nonExistentPlugin.ID)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), nonExistentPlugin.Name, retrievedPlugin.Name)
}

func (suite *PluginRepositoryTestSuite) TestDelete() {
	// Create a plugin first
	plugin := &models.Plugin{
		BaseModel: models.BaseModel{
			Name:        "delete-plugin",
			Title:       "Delete Plugin",
			Description: "Plugin for testing delete",
		},
		Icon:               "DeleteIcon",
		ReactComponentPath: "/plugins/delete/Delete.jsx",
		BackendServerURL:   "http://localhost:3008",
		Owner:              "Delete Team",
	}

	err := suite.repo.Create(plugin)
	assert.NoError(suite.T(), err)

	// Verify the plugin exists
	retrievedPlugin, err := suite.repo.GetByID(plugin.ID)
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), retrievedPlugin)

	// Delete the plugin
	err = suite.repo.Delete(plugin.ID)
	assert.NoError(suite.T(), err)

	// Verify the plugin is deleted
	retrievedPlugin, err = suite.repo.GetByID(plugin.ID)
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), retrievedPlugin)
	assert.Contains(suite.T(), err.Error(), "record not found")
}

func (suite *PluginRepositoryTestSuite) TestDelete_NonExistentPlugin() {
	// Try to delete a plugin that doesn't exist
	nonExistentID := uuid.New()
	err := suite.repo.Delete(nonExistentID)

	// GORM's Delete method doesn't return an error if no records are affected
	// This is expected behavior
	assert.NoError(suite.T(), err)
}

func (suite *PluginRepositoryTestSuite) TestCRUDOperations_Integration() {
	// Test a complete CRUD cycle

	// Create
	plugin := &models.Plugin{
		BaseModel: models.BaseModel{
			Name:        "crud-plugin",
			Title:       "CRUD Plugin",
			Description: "Plugin for testing CRUD operations",
		},
		Icon:               "CRUDIcon",
		ReactComponentPath: "/plugins/crud/CRUD.jsx",
		BackendServerURL:   "http://localhost:3009",
		Owner:              "CRUD Team",
	}

	err := suite.repo.Create(plugin)
	assert.NoError(suite.T(), err)
	assert.NotEqual(suite.T(), uuid.Nil, plugin.ID)

	// Read
	retrievedPlugin, err := suite.repo.GetByID(plugin.ID)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), plugin.Name, retrievedPlugin.Name)

	retrievedPlugin, err = suite.repo.GetByName(plugin.Name)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), plugin.ID, retrievedPlugin.ID)

	// Update
	plugin.Title = "Updated CRUD Plugin"
	err = suite.repo.Update(plugin)
	assert.NoError(suite.T(), err)

	retrievedPlugin, err = suite.repo.GetByID(plugin.ID)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "Updated CRUD Plugin", retrievedPlugin.Title)

	// Delete
	err = suite.repo.Delete(plugin.ID)
	assert.NoError(suite.T(), err)

	retrievedPlugin, err = suite.repo.GetByID(plugin.ID)
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), retrievedPlugin)
}

func TestPluginRepositoryTestSuite(t *testing.T) {
	suite.Run(t, new(PluginRepositoryTestSuite))
}
