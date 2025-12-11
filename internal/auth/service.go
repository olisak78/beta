package auth

import (
	"context"
	"crypto/rand"
	apperrors "developer-portal-backend/internal/errors"
	"encoding/base64"
	"fmt"
	"reflect"
	"time"

	"developer-portal-backend/internal/database/models"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/oauth2"
)

// RefreshTokenData stores information about a refresh token
type RefreshTokenData struct {
	UserUUID    string    `json:"user_uuid"`
	Provider    string    `json:"provider"`
	AccessToken string    `json:"access_token"`
	ExpiresAt   time.Time `json:"expires_at"`
}

// MemberRepository defines the interface for member operations needed by auth service
type UserRepository interface {
	GetByEmail(email string) (interface{}, error)
}

// TokenStore defines persistence API for provider access tokens
type TokenStore interface {
	UpsertToken(userUUID uuid.UUID, provider string, token string, expiresAt time.Time) error
	GetValidToken(userUUID uuid.UUID, provider string) (*models.Token, error)
	DeleteToken(userUUID uuid.UUID, provider string) error
	CleanupExpiredTokens() error
}

// AuthService provides authentication functionality
type AuthService struct {
	config        *AuthConfig
	githubClients map[string]*GitHubClient
	tokenStore    TokenStore
	userRepo      UserRepository
}

// AuthClaims represents JWT token claims
type AuthClaims struct {
	Username string `json:"username" example:"I012345"`
	Email    string `json:"email" example:"john.doe@sap.com"`
	UUID     string `json:"user_uuid" example:"550e8400-e29b-41d4-a716-446655440000"`
	// Standard JWT fields
	jwt.RegisteredClaims `swaggerignore:"true"`
}

// AuthStartResponse represents the response for auth start endpoint
type AuthStartResponse struct {
	URL string `json:"url"`
}

// AuthHandlerResponse represents the response for auth handler endpoint
type AuthHandlerResponse struct {
	AccessToken string `json:"accessToken"`
	TokenType   string `json:"tokenType"`
	ExpiresIn   int64  `json:"expiresIn"`
}

// RefreshTokenRequest represents the request for token refresh
type RefreshTokenRequest struct {
	RefreshToken string `json:"refreshToken" binding:"required"`
}

// AuthRefreshResponse represents the response from the refresh endpoint
type AuthRefreshResponse struct {
	AccessToken      string      `json:"accessToken" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
	TokenType        string      `json:"tokenType" example:"bearer"`
	ExpiresInSeconds int64       `json:"expiresInSeconds" example:"3600"`
	Scope            string      `json:"scope" example:"user:email read:user"`
	Profile          UserProfile `json:"profile"`
	Valid            bool        `json:"valid,omitempty" example:"true"`
}

// AuthLogoutResponse represents the response from the logout endpoint
type AuthLogoutResponse struct {
	Message string `json:"message" example:"Logged out successfully"`
}

// AuthValidateResponse represents the response from the token validation endpoint
type AuthValidateResponse struct {
	Valid  bool        `json:"valid" example:"true"`
	Claims *AuthClaims `json:"claims"`
}

// NewAuthService creates a new authentication service
func NewAuthService(config *AuthConfig, userRepo UserRepository, tokenStore TokenStore) (*AuthService, error) {
	if err := config.ValidateConfig(); err != nil {
		return nil, fmt.Errorf("invalid auth config: %w", err)
	}

	// Initialize GitHub clients for each provider
	githubClients := make(map[string]*GitHubClient)
	for providerName, providerConfig := range config.Providers {
		githubClients[providerName] = NewGitHubClient(&providerConfig)
	}

	return &AuthService{
		config:        config,
		githubClients: githubClients,
		tokenStore:    tokenStore,
		userRepo:      userRepo,
	}, nil
}

// getMemberIDByEmail looks up a member by email and returns their ID as a string
// Returns empty string if member is not found or an error occurs
func (s *AuthService) getMemberIDByEmail(email string) string {
	if s.userRepo == nil || email == "" {
		return ""
	}

	member, err := s.userRepo.GetByEmail(email)
	if err != nil || member == nil {
		return ""
	}

	// Use reflection to access the ID field from the member struct
	// This works with models.Member which has an ID field of type uuid.UUID
	val := reflect.ValueOf(member)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() == reflect.Struct {
		idField := val.FieldByName("ID")
		if idField.IsValid() {
			// Convert UUID to string
			idStr := fmt.Sprintf("%v", idField.Interface())
			return idStr
		}
	}

	return ""
}

// GetAuthURL generates OAuth2 authorization URL
func (s *AuthService) GetAuthURL(provider, state string) (string, error) {
	_, err := s.config.GetProvider(provider)
	if err != nil {
		return "", err
	}

	githubClient, exists := s.githubClients[provider]
	if !exists {
		return "", fmt.Errorf("GitHub client not found for provider %s", provider)
	}

	// Generate callback URL
	callbackURL := fmt.Sprintf("%s/api/auth/%s/handler/frame", s.config.RedirectURL, provider)

	oauth2Config := githubClient.GetOAuth2Config(callbackURL)
	authURL := oauth2Config.AuthCodeURL(state, oauth2.AccessTypeOffline)

	return authURL, nil
}

// HandleCallback processes OAuth2 callback and returns user information
func (s *AuthService) HandleCallback(ctx context.Context, provider, code, state string) (*AuthHandlerResponse, error) {
	_, err := s.config.GetProvider(provider)
	if err != nil {
		return nil, err
	}

	githubClient, exists := s.githubClients[provider]
	if !exists {
		return nil, fmt.Errorf("GitHub client not found for provider %s", provider)
	}

	// Generate callback URL
	callbackURL := fmt.Sprintf("%s/api/auth/%s/handler/frame", s.config.RedirectURL, provider)

	oauth2Config := githubClient.GetOAuth2Config(callbackURL)

	// Exchange authorization code for access token
	token, err := oauth2Config.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code for token: %w", err)
	}

	// Get user profile from GitHub
	profile, err := githubClient.GetUserProfile(ctx, token.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to get user profile: %w", err)
	}

	// Look up member by email and populate UUID if found
	profile.UUID = s.getMemberIDByEmail(profile.Email)

	// Persist provider access token in DB-backed token store (if user has UUID)
	if s.tokenStore != nil && profile.UUID != "" {
		userID, parseErr := uuid.Parse(profile.UUID)
		if parseErr != nil {
			return nil, fmt.Errorf("invalid user UUID: %w", parseErr)
		}
		if upsertErr := s.tokenStore.UpsertToken(userID, provider, token.AccessToken, time.Now().AddDate(0, 0, s.config.AccessTokenExpiresInDays)); upsertErr != nil {
			return nil, fmt.Errorf("failed to store provider access token: %w", upsertErr)
		}
	}

	// Generate JWT token
	jwtToken, err := s.GenerateJWT(profile)
	if err != nil {
		return nil, fmt.Errorf("failed to generate JWT: %w", err)
	}

	response := &AuthHandlerResponse{
		AccessToken: jwtToken,
		TokenType:   "Bearer",
		ExpiresIn:   int64(s.config.JWTExpiresInSeconds),
	}

	return response, nil
}

// GenerateJWT creates a JWT token for the user (provider is deprecated hence ignored)
func (s *AuthService) GenerateJWT(userProfile *UserProfile) (string, error) {
	now := time.Now()
	claims := &AuthClaims{
		Username: userProfile.Username,
		Email:    userProfile.Email,
		UUID:     userProfile.UUID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(s.config.JWTExpiresInSeconds) * time.Second)),
			Issuer:    "developer-portal",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.config.JWTSecret))
}

// ValidateJWT validates and parses a JWT token
func (s *AuthService) ValidateJWT(tokenString string) (*AuthClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &AuthClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.config.JWTSecret), nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if claims, ok := token.Claims.(*AuthClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}

// generateRandomString generates a random base64 encoded string
func (s *AuthService) generateRandomString(length int) (string, error) {
	bytes := make([]byte, length)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

func (s *AuthService) GetGitHubAccessToken(userUUID, provider string) (string, error) {
	if s == nil {
		return "", apperrors.ErrAuthServiceNotInitialized
	}
	// check userUUID is not empty
	if userUUID == "" {
		return "", apperrors.ErrUserUUIDMissing
	}
	// check provider is not empty
	if provider == "" {
		return "", apperrors.ErrProviderMissing
	}

	// Use DB-backed token store
	if s.tokenStore == nil {
		return "", apperrors.ErrTokenStoreNotInitialized
	}
	uid, err := uuid.Parse(userUUID)
	if err != nil {
		return "", fmt.Errorf("invalid userUUID: %w", err)
	}
	tok, err := s.tokenStore.GetValidToken(uid, provider)
	if err != nil || tok == nil || time.Now().After(tok.ExpiresAt) {
		return "", fmt.Errorf("no valid GitHub token found for user %s with provider %s", userUUID, provider)
	}
	return tok.Token, nil
}

// GetGitHubClient retrieves the GitHub client for a specific provider
func (s *AuthService) GetGitHubClient(provider string) (*GitHubClient, error) {
	if s == nil {
		return nil, apperrors.ErrAuthServiceNotInitialized
	}

	client, exists := s.githubClients[provider]
	if !exists {
		return nil, fmt.Errorf("GitHub client not found for provider %s", provider)
	}
	return client, nil
}

// Logout handles user logout (stateless JWT tokens don't require server-side logout)
func (s *AuthService) Logout() error {
	// For JWT tokens, logout is typically handled client-side by removing the token
	// In a production system, you might maintain a blacklist of invalidated tokens
	return nil
}
