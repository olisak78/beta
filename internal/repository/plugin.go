package repository

import (
	"developer-portal-backend/internal/database/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// PluginRepository handles database operations for plugins
type PluginRepository struct {
	db *gorm.DB
}

// NewPluginRepository creates a new plugin repository
func NewPluginRepository(db *gorm.DB) *PluginRepository {
	return &PluginRepository{db: db}
}

// Create creates a new plugin
func (r *PluginRepository) Create(plugin *models.Plugin) error {
	return r.db.Create(plugin).Error
}

// GetByID retrieves a plugin by ID
func (r *PluginRepository) GetByID(id uuid.UUID) (*models.Plugin, error) {
	var plugin models.Plugin
	err := r.db.First(&plugin, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &plugin, nil
}

// GetByName retrieves a plugin by name
func (r *PluginRepository) GetByName(name string) (*models.Plugin, error) {
	var plugin models.Plugin
	err := r.db.First(&plugin, "name = ?", name).Error
	if err != nil {
		return nil, err
	}
	return &plugin, nil
}

// GetAll retrieves all plugins with pagination
func (r *PluginRepository) GetAll(limit, offset int) ([]models.Plugin, int64, error) {
	var plugins []models.Plugin
	var total int64

	// Get total count
	if err := r.db.Model(&models.Plugin{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Get paginated results
	if err := r.db.Model(&models.Plugin{}).Limit(limit).Offset(offset).Find(&plugins).Error; err != nil {
		return nil, 0, err
	}

	return plugins, total, nil
}

// Update updates a plugin
func (r *PluginRepository) Update(plugin *models.Plugin) error {
	return r.db.Save(plugin).Error
}

// Delete deletes a plugin
func (r *PluginRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&models.Plugin{}, "id = ?", id).Error
}
