package repository

import (
	"developer-portal-backend/internal/database/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// LinkRepository handles database operations for links
type LinkRepository struct {
	db *gorm.DB
}

// Ensure LinkRepository implements LinkRepositoryInterface
var _ LinkRepositoryInterface = (*LinkRepository)(nil)

// NewLinkRepository creates a new link repository
func NewLinkRepository(db *gorm.DB) *LinkRepository {
	return &LinkRepository{db: db}
}

// GetByOwner retrieves all links owned by the specified owner (user/team) UUID
func (r *LinkRepository) GetByOwner(owner uuid.UUID) ([]models.Link, error) {
	var links []models.Link
	if err := r.db.Where("owner = ?", owner).Order("title ASC").Find(&links).Error; err != nil {
		return nil, err
	}
	return links, nil
}

// GetByIDs retrieves links by a set of UUID IDs
func (r *LinkRepository) GetByIDs(ids []uuid.UUID) ([]models.Link, error) {
	if len(ids) == 0 {
		return []models.Link{}, nil
	}
	var links []models.Link
	if err := r.db.Where("id IN ?", ids).Order("title ASC").Find(&links).Error; err != nil {
		return nil, err
	}
	return links, nil
}

 // Create inserts a new link
func (r *LinkRepository) Create(link *models.Link) error {
	return r.db.Create(link).Error
}

// Delete removes a link by ID
func (r *LinkRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&models.Link{}, "id = ?", id).Error
}

// GetByID retrieves a link by its UUID
func (r *LinkRepository) GetByID(id uuid.UUID) (*models.Link, error) {
	var link models.Link
	if err := r.db.Where("id = ?", id).First(&link).Error; err != nil {
		return nil, err
	}
	return &link, nil
}

// Update updates an existing link
func (r *LinkRepository) Update(link *models.Link) error {
	return r.db.Save(link).Error
}
