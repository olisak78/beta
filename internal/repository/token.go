package repository

import (
	"developer-portal-backend/internal/auth"
	"developer-portal-backend/internal/database/models"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// TokenRepository handles database operations for provider access tokens (refresh/session tokens)
type TokenRepository struct {
	db *gorm.DB
}

// NewTokenRepository creates a new token repository
func NewTokenRepository(db *gorm.DB) *TokenRepository {
	return &TokenRepository{db: db}
}

// UpsertToken creates or updates a token record for a given user and provider.
// Implemented as a single-statement UPSERT to avoid race conditions.
func (r *TokenRepository) UpsertToken(userUUID uuid.UUID, provider string, token string, expiresAt time.Time) error {
	encTok, err := auth.EncryptToken(token)
	if err != nil {
		return err
	}
	tok := &models.Token{
		UserUUID:  userUUID,
		Provider:  provider,
		Token:     encTok,
		ExpiresAt: expiresAt,
	}
	return r.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_uuid"}, {Name: "provider"}},
		DoUpdates: clause.Assignments(map[string]interface{}{"token": tok.Token, "expires_at": tok.ExpiresAt}),
	}).Create(tok).Error
}

// GetValidToken returns a non-expired token for the given user and provider.
func (r *TokenRepository) GetValidToken(userUUID uuid.UUID, provider string) (*models.Token, error) {
	var tok models.Token
	if err := r.db.Where("user_uuid = ? AND provider = ? AND expires_at > ?", userUUID, provider, time.Now()).
		First(&tok).Error; err != nil {
		// Mitigate timing side-channel by performing a dummy decrypt attempt even when record is not found
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if dummy, encErr := auth.EncryptToken("dummy"); encErr == nil {
				_, _ = auth.DecryptToken(dummy)
			}
		}
		return nil, err
	}
	// Decrypt token before returning; tokens must be stored encrypted
	plain, decErr := auth.DecryptToken(tok.Token)
	if decErr != nil {
		// Perform dummy encrypt+decrypt to reduce timing side-channel differences
		if dummy, encErr := auth.EncryptToken("dummy"); encErr == nil {
			_, _ = auth.DecryptToken(dummy)
		}
		return nil, decErr
	}
	// Perform dummy encrypt+decrypt to reduce timing side-channel differences
	if dummy, encErr := auth.EncryptToken("dummy"); encErr == nil {
		_, _ = auth.DecryptToken(dummy)
	}
	tok.Token = plain
	return &tok, nil
}

// DeleteToken removes a token record for the given user and provider.
func (r *TokenRepository) DeleteToken(userUUID uuid.UUID, provider string) error {
	return r.db.Where("user_uuid = ? AND provider = ?", userUUID, provider).
		Delete(&models.Token{}).Error
}

// CleanupExpiredTokens deletes all expired tokens from the table.
func (r *TokenRepository) CleanupExpiredTokens() error {
	return r.db.Where("expires_at <= ?", time.Now()).
		Delete(&models.Token{}).Error
}
