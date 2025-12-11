package repository

import (
	"bytes"
	"encoding/base64"
	"strings"
	"testing"
	"time"

	"developer-portal-backend/internal/auth"
	"developer-portal-backend/internal/database/models"
	"developer-portal-backend/internal/testutils"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
)

type TokenRepositoryTestSuite struct {
	suite.Suite
	baseTestSuite *testutils.BaseTestSuite
	repo          *TokenRepository
}

// SetupSuite runs once for the suite
func (suite *TokenRepositoryTestSuite) SetupSuite() {
	// Initialize shared DB and repository
	suite.baseTestSuite = testutils.SetupTestSuite(suite.T())
	suite.repo = NewTokenRepository(suite.baseTestSuite.DB)

	// Set a deterministic TOKEN_SECRET for crypto tests
	secretBytes := bytes.Repeat([]byte{1}, 32)
	secret := base64.StdEncoding.EncodeToString(secretBytes)
	suite.Require().NoError(auth.SetTokenSecret(secret))
}

// TearDownSuite runs once after the suite
func (suite *TokenRepositoryTestSuite) TearDownSuite() {
	suite.baseTestSuite.TeardownTestSuite()
}

// SetupTest/TearDownTest run before/after each test
func (suite *TokenRepositoryTestSuite) SetupTest() {
	suite.baseTestSuite.SetupTest()
	suite.truncateTokens()
}

func (suite *TokenRepositoryTestSuite) TearDownTest() {
	suite.truncateTokens()
	suite.baseTestSuite.TearDownTest()
}

// truncateTokens ensures a clean tokens table
func (suite *TokenRepositoryTestSuite) truncateTokens() {
	suite.baseTestSuite.DB.Exec(`TRUNCATE TABLE "tokens" RESTART IDENTITY CASCADE;`)
}

// helper to insert a token directly via gorm
func (suite *TokenRepositoryTestSuite) insertRawToken(user uuid.UUID, provider, token string, expiresAt time.Time) {
	t := &models.Token{
		UserUUID:  user,
		Provider:  provider,
		Token:     token,
		ExpiresAt: expiresAt,
	}
	suite.Require().NoError(suite.baseTestSuite.DB.Create(t).Error)
}

// Test UpsertToken creates a new record and then updates existing one
func (suite *TokenRepositoryTestSuite) TestUpsertToken_CreateAndUpdate() {
	user := uuid.New()
	provider := "github"
	expires1 := time.Now().Add(1 * time.Hour)

	// Create new
	err := suite.repo.UpsertToken(user, provider, "tok-1", expires1)
	suite.NoError(err)

	// Verify DB state (encrypted value stored)
	var rec models.Token
	err = suite.baseTestSuite.DB.Where("user_uuid = ? AND provider = ?", user, provider).First(&rec).Error
	suite.NoError(err)
	suite.True(strings.HasPrefix(rec.Token, "enc:v1:"), "token should be encrypted with enc prefix")
	plain, derr := auth.DecryptToken(rec.Token)
	suite.NoError(derr)
	suite.Equal("tok-1", plain)
	suite.WithinDuration(expires1, rec.ExpiresAt, 2*time.Second)

	// Update existing
	expires2 := time.Now().Add(2 * time.Hour)
	err = suite.repo.UpsertToken(user, provider, "tok-2", expires2)
	suite.NoError(err)

	// Verify updated record
	var rec2 models.Token
	err = suite.baseTestSuite.DB.Where("user_uuid = ? AND provider = ?", user, provider).First(&rec2).Error
	suite.NoError(err)
	plain2, derr2 := auth.DecryptToken(rec2.Token)
	suite.NoError(derr2)
	suite.Equal("tok-2", plain2)
	suite.WithinDuration(expires2, rec2.ExpiresAt, 2*time.Second)
}

// Test GetValidToken returns decrypted token when not expired
func (suite *TokenRepositoryTestSuite) TestGetValidToken_Success() {
	user := uuid.New()
	provider := "jira"
	expires := time.Now().Add(30 * time.Minute)

	enc, err := auth.EncryptToken("my-secret-token")
	suite.Require().NoError(err)
	suite.insertRawToken(user, provider, enc, expires)

	tok, err := suite.repo.GetValidToken(user, provider)
	suite.NoError(err)
	suite.NotNil(tok)
	suite.Equal("my-secret-token", tok.Token) // should be decrypted
	suite.Equal(user, tok.UserUUID)
	suite.Equal(provider, tok.Provider)
}

// Test GetValidToken returns not found when expired
func (suite *TokenRepositoryTestSuite) TestGetValidToken_Expired() {
	user := uuid.New()
	provider := "slack"
	expired := time.Now().Add(-5 * time.Minute)

	enc, err := auth.EncryptToken("expired-token")
	suite.Require().NoError(err)
	suite.insertRawToken(user, provider, enc, expired)

	tok, err := suite.repo.GetValidToken(user, provider)
	suite.Error(err)
	suite.Equal(gorm.ErrRecordNotFound, err)
	suite.Nil(tok)
}

// Test GetValidToken fails when decryption fails for an encrypted-looking payload
func (suite *TokenRepositoryTestSuite) TestGetValidToken_DecryptError() {
	user := uuid.New()
	provider := "gitlab"
	expires := time.Now().Add(45 * time.Minute)

	// Malformed encrypted token (bad base64 after prefix)
	malformed := "enc:v1:invalid%%%base64"
	suite.insertRawToken(user, provider, malformed, expires)

	tok, err := suite.repo.GetValidToken(user, provider)
	suite.Error(err)
	suite.Nil(tok)
	// We expect a decryption-related error; message may vary, but should contain a hint
	suite.Contains(err.Error(), "base64", "expected base64 decode error for malformed payload")
}

// Test DeleteToken removes the record
func (suite *TokenRepositoryTestSuite) TestDeleteToken() {
	user := uuid.New()
	provider := "github"
	expires := time.Now().Add(1 * time.Hour)

	enc, err := auth.EncryptToken("to-delete")
	suite.Require().NoError(err)
	suite.insertRawToken(user, provider, enc, expires)

	// Ensure it exists
	var count int64
	suite.baseTestSuite.DB.Model(&models.Token{}).Where("user_uuid = ? AND provider = ?", user, provider).Count(&count)
	suite.Equal(int64(1), count)

	// Delete
	err = suite.repo.DeleteToken(user, provider)
	suite.NoError(err)

	// Verify deletion
	suite.baseTestSuite.DB.Model(&models.Token{}).Where("user_uuid = ? AND provider = ?", user, provider).Count(&count)
	suite.Equal(int64(0), count)
}

// Test CleanupExpiredTokens deletes only expired tokens
func (suite *TokenRepositoryTestSuite) TestCleanupExpiredTokens() {
	user1 := uuid.New()
	user2 := uuid.New()
	provider := "azure"

	enc1, err := auth.EncryptToken("expired-1")
	suite.Require().NoError(err)
	enc2, err := auth.EncryptToken("valid-1")
	suite.Require().NoError(err)

	// Insert one expired and one valid token
	suite.insertRawToken(user1, provider, enc1, time.Now().Add(-1*time.Hour))
	suite.insertRawToken(user2, provider, enc2, time.Now().Add(1*time.Hour))

	// Run cleanup
	err = suite.repo.CleanupExpiredTokens()
	suite.NoError(err)

	// Verify only valid remains
	var tokens []models.Token
	err = suite.baseTestSuite.DB.Find(&tokens).Error
	suite.NoError(err)
	suite.Len(tokens, 1)
	plain, derr := auth.DecryptToken(tokens[0].Token)
	suite.NoError(derr)
	suite.Equal("valid-1", plain)
}

// Run the test suite
func TestTokenRepositoryTestSuite(t *testing.T) {
	suite.Run(t, new(TokenRepositoryTestSuite))
}
