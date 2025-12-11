package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"developer-portal-backend/internal/database/models"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type noopTokenStore struct{}

func (n *noopTokenStore) UpsertToken(userUUID uuid.UUID, provider string, token string, expiresAt time.Time) error {
	return nil
}
func (n *noopTokenStore) GetValidToken(userUUID uuid.UUID, provider string) (*models.Token, error) {
	return nil, nil
}
func (n *noopTokenStore) DeleteToken(userUUID uuid.UUID, provider string) error { return nil }
func (n *noopTokenStore) CleanupExpiredTokens() error                           { return nil }

func TestAuthConfig(t *testing.T) {
	t.Run("valid config structure", func(t *testing.T) {
		// Test creating a valid config directly
		config := &AuthConfig{
			JWTSecret:   "test-signing-key",
			RedirectURL: "http://localhost:3000",
			Providers: map[string]ProviderConfig{
				"githubtools": {
					ClientID:          "dev-client-id",
					ClientSecret:      "dev-client-secret",
					EnterpriseBaseURL: "https://github.tools.sap",
				},
				"githubwdf": {
					ClientID:          "wdf-dev-client-id",
					ClientSecret:      "wdf-dev-client-secret",
					EnterpriseBaseURL: "https://github.wdf.sap.corp",
				},
			},
		}

		// Test validation
		err := config.ValidateConfig()
		assert.NoError(t, err)
		assert.NotEmpty(t, config.JWTSecret)
		assert.NotEmpty(t, config.RedirectURL)
	})

	t.Run("missing jwt secret", func(t *testing.T) {
		config := &AuthConfig{
			RedirectURL: "http://localhost:3000",
			Providers: map[string]ProviderConfig{
				"githubtools": {
					ClientID:     "dev-client-id",
					ClientSecret: "dev-client-secret",
				},
			},
		}

		err := config.ValidateConfig()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "JWT secret is required")
	})

	t.Run("missing redirect url", func(t *testing.T) {
		config := &AuthConfig{
			JWTSecret: "test-secret",
			Providers: map[string]ProviderConfig{
				"githubtools": {
					ClientID:     "dev-client-id",
					ClientSecret: "dev-client-secret",
				},
			},
		}

		err := config.ValidateConfig()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "redirect URL is required")
	})

	t.Run("missing client credentials", func(t *testing.T) {
		config := &AuthConfig{
			JWTSecret:   "test-secret",
			RedirectURL: "http://localhost:3000",
			Providers: map[string]ProviderConfig{
				"githubtools": {
					// Missing ClientID and ClientSecret
				},
			},
		}

		err := config.ValidateConfig()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "client_id is required")
	})
}

func TestGitHubClientConfig(t *testing.T) {
	config := &ProviderConfig{
		ClientID:          "test-client-id",
		ClientSecret:      "test-client-secret",
		EnterpriseBaseURL: "https://github.example.com",
	}

	client := NewGitHubClient(config)
	assert.NotNil(t, client)

	oauthConfig := client.GetOAuth2Config("http://localhost:8080/callback")
	assert.Equal(t, "test-client-id", oauthConfig.ClientID)
	assert.Equal(t, "test-client-secret", oauthConfig.ClientSecret)
	assert.Equal(t, "http://localhost:8080/callback", oauthConfig.RedirectURL)
	assert.Contains(t, oauthConfig.Scopes, "user:email")
}

func TestJWTOperations(t *testing.T) {
	config := &AuthConfig{
		JWTSecret:   "test-signing-key-for-jwt-operations",
		RedirectURL: "http://localhost:3000",
		Providers: map[string]ProviderConfig{
			"githubtools": {
				ClientID:          "test-client-id",
				ClientSecret:      "test-client-secret",
				EnterpriseBaseURL: "https://github.tools.sap",
			},
		},
	}

	service, err := NewAuthService(config, nil, &noopTokenStore{})
	require.NoError(t, err)

	userProfile := &UserProfile{
		ID:        12345,
		Username:  "testuser",
		Email:     "test@example.com",
		Name:      "Test User",
		AvatarURL: "https://avatars.githubusercontent.com/u/12345",
		UUID:      "test-uuid",
	}

	// Test token generation
	token, err := service.GenerateJWT(userProfile)
	assert.NoError(t, err)
	assert.NotEmpty(t, token)

	// Test token validation
	validatedClaims, err := service.ValidateJWT(token)
	assert.NoError(t, err)
	assert.Equal(t, userProfile.Username, validatedClaims.Username)
	assert.Equal(t, userProfile.Email, validatedClaims.Email)
	assert.Equal(t, userProfile.UUID, validatedClaims.UUID)

	// Test invalid token
	_, err = service.ValidateJWT("invalid-token")
	assert.Error(t, err)
}

func TestAuthHandlers(t *testing.T) {
	// Create test config
	config := &AuthConfig{
		Providers: map[string]ProviderConfig{
			"githubtools": {
				ClientID:          "test-client-id",
				ClientSecret:      "test-client-secret",
				EnterpriseBaseURL: "https://github.tools.sap",
			},
		},
		JWTSecret:   "test-signing-key",
		RedirectURL: "http://localhost:3000",
	}

	service, err := NewAuthService(config, nil, &noopTokenStore{})
	require.NoError(t, err)

	handler := NewAuthHandler(service)

	// Setup Gin in test mode
	gin.SetMode(gin.TestMode)

	t.Run("Start endpoint", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/auth/githubtools/start", nil)
		c.Params = gin.Params{{Key: "provider", Value: "githubtools"}}

		handler.Start(c)

		assert.Equal(t, http.StatusFound, w.Code)
		location := w.Header().Get("Location")
		assert.Contains(t, location, "github.tools.sap")
		assert.Contains(t, location, "oauth/authorize")
	})

	t.Run("Logout endpoint", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/api/auth/githubtools/logout", nil)
		c.Request.Header.Set("Content-Type", "application/json")
		c.Params = gin.Params{{Key: "provider", Value: "githubtools"}}

		handler.Logout(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "Logged out successfully", response["message"])
	})
}

func TestRefreshToken(t *testing.T) {
	config := &AuthConfig{
		JWTSecret:   "test-signing-key-for-refresh-test",
		RedirectURL: "http://localhost:3000",
		Providers: map[string]ProviderConfig{
			"githubtools": {
				ClientID:          "test-client-id",
				ClientSecret:      "test-client-secret",
				EnterpriseBaseURL: "https://github.tools.sap",
			},
		},
	}

	service, err := NewAuthService(config, nil, &noopTokenStore{})
	require.NoError(t, err)

	// Create a user profile
	userProfile := &UserProfile{
		ID:        12345,
		Username:  "testuser",
		Email:     "test@example.com",
		Name:      "Test User",
		AvatarURL: "https://avatars.githubusercontent.com/u/12345",
		UUID:      "test-uuid",
	}

	// Generate initial token
	token, err := service.GenerateJWT(userProfile)
	require.NoError(t, err)

	// Validate the token can be parsed
	claims, err := service.ValidateJWT(token)
	assert.NoError(t, err)
	assert.Equal(t, userProfile.Username, claims.Username)
	assert.Equal(t, userProfile.Email, claims.Email)
	assert.Equal(t, userProfile.UUID, claims.UUID)
}

func TestConfigValidation(t *testing.T) {
	t.Run("empty providers map", func(t *testing.T) {
		config := &AuthConfig{
			JWTSecret:   "test-secret",
			RedirectURL: "http://localhost:3000",
			Providers:   map[string]ProviderConfig{},
		}

		err := config.ValidateConfig()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "at least one provider")
	})

	t.Run("template strings are valid", func(t *testing.T) {
		config := &AuthConfig{
			JWTSecret:   "test-secret",
			RedirectURL: "http://localhost:3000",
			Providers: map[string]ProviderConfig{
				"githubtools": {
					ClientID:     "${GITHUB_CLIENT_ID}",
					ClientSecret: "${GITHUB_CLIENT_SECRET}",
				},
			},
		}

		// Template strings are valid (non-empty) during validation
		// They will be expanded by LoadAuthConfig from environment
		err := config.ValidateConfig()
		assert.NoError(t, err)
	})

	t.Run("mixed valid and template providers", func(t *testing.T) {
		config := &AuthConfig{
			JWTSecret:   "test-secret",
			RedirectURL: "http://localhost:3000",
			Providers: map[string]ProviderConfig{
				"githubtools": {
					ClientID:     "real-client-id",
					ClientSecret: "real-client-secret",
				},
				"githubwdf": {
					ClientID:     "${GITHUB_WDF_CLIENT_ID}",
					ClientSecret: "${GITHUB_WDF_CLIENT_SECRET}",
				},
			},
		}

		// Should pass because githubtools has valid credentials
		err := config.ValidateConfig()
		assert.NoError(t, err)
	})
}

func TestGetProvider(t *testing.T) {
	config := &AuthConfig{
		JWTSecret:   "test-secret",
		RedirectURL: "http://localhost:3000",
		Providers: map[string]ProviderConfig{
			"githubtools": {
				ClientID:          "test-client-id",
				ClientSecret:      "test-client-secret",
				EnterpriseBaseURL: "https://github.tools.sap",
			},
		},
	}

	t.Run("existing provider", func(t *testing.T) {
		provider, err := config.GetProvider("githubtools")
		assert.NoError(t, err)
		assert.NotNil(t, provider)
		assert.Equal(t, "test-client-id", provider.ClientID)
		assert.Equal(t, "https://github.tools.sap", provider.EnterpriseBaseURL)
	})

	t.Run("non-existing provider", func(t *testing.T) {
		_, err := config.GetProvider("nonexistent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "provider 'nonexistent' not found")
	})
}

func TestLoadAuthConfigFromFile(t *testing.T) {
	// Skip this test for now - config loading is working in the actual application
	// This test needs to be refactored to work with viper's environment variable expansion
	t.Skip("Config file loading tested via integration tests")
}

func TestEnvironmentVariableOverrides(t *testing.T) {
	// Skip this test for now - environment variable expansion tested via integration
	t.Skip("Environment variable expansion tested via integration tests")
}

func TestJWTExpiration(t *testing.T) {
	config := &AuthConfig{
		JWTSecret:   "test-signing-key-for-expiration-test",
		RedirectURL: "http://localhost:3000",
		Providers: map[string]ProviderConfig{
			"githubtools": {
				ClientID:     "test-client-id",
				ClientSecret: "test-client-secret",
			},
		},
	}

	service, err := NewAuthService(config, nil, &noopTokenStore{})
	require.NoError(t, err)

	userProfile := &UserProfile{
		ID:       12345,
		Username: "testuser",
		Email:    "test@example.com",
		UUID:     "test-uuid",
	}

	// Generate token
	token, err := service.GenerateJWT(userProfile)
	require.NoError(t, err)
	assert.NotEmpty(t, token, "Token should not be empty")

	// Token should be valid now
	claims, err := service.ValidateJWT(token)
	assert.NoError(t, err)
	assert.NotNil(t, claims)

	// Verify all basic claims are set
	assert.Equal(t, userProfile.Username, claims.Username)
	assert.Equal(t, userProfile.Email, claims.Email)
	assert.Equal(t, userProfile.UUID, claims.UUID)
}

func TestJWTExpiration_ExpiredTokenInvalid(t *testing.T) {
	// Config with known secret and minimal provider to init service
	config := &AuthConfig{
		JWTSecret:   "test-signing-key-for-expired-token",
		RedirectURL: "http://localhost:3000",
		Providers: map[string]ProviderConfig{
			"githubtools": {
				ClientID:     "test-client-id",
				ClientSecret: "test-client-secret",
			},
		},
	}

	service, err := NewAuthService(config, nil, &noopTokenStore{})
	require.NoError(t, err)

	// Manually mint an already-expired token (exp in the past)
	claims := &AuthClaims{
		Username: "testuser",
		Email:    "test@example.com",
		UUID:     "test-uuid",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Minute)),
			Issuer:    "developer-portal",
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	expiredToken, err := token.SignedString([]byte(config.JWTSecret))
	require.NoError(t, err)

	// Validate should fail due to expiration
	_, err = service.ValidateJWT(expiredToken)
	assert.Error(t, err, "expected an error for expired token")
}

func TestJWTExpiration_58MinutesValid(t *testing.T) {
	// Configure service to mint tokens with 58 minutes expiry
	config := &AuthConfig{
		JWTSecret:   "test-signing-key-for-58-minutes",
		RedirectURL: "http://localhost:3000",
		Providers: map[string]ProviderConfig{
			"githubtools": {
				ClientID:     "test-client-id",
				ClientSecret: "test-client-secret",
			},
		},
		JWTExpiresInSeconds: 58 * 60, // 58 minutes
	}

	service, err := NewAuthService(config, nil, &noopTokenStore{})
	require.NoError(t, err)

	// Create minimal profile
	userProfile := &UserProfile{
		ID:       1,
		Username: "user58",
		Email:    "user58@example.com",
		UUID:     "uuid-58",
	}

	// Generate and validate token
	token, err := service.GenerateJWT(userProfile)
	require.NoError(t, err)
	require.NotEmpty(t, token)

	claims, err := service.ValidateJWT(token)
	assert.NoError(t, err, "token with 58 minutes expiry should be valid")
	require.NotNil(t, claims)

	// Ensure the exp is ~58 minutes in the future (allow small processing skew)
	require.NotNil(t, claims.ExpiresAt)
	now := time.Now()
	exp := claims.ExpiresAt.Time
	assert.True(t, exp.After(now.Add(57*time.Minute)), "exp should be after ~57 minutes from now")
	assert.True(t, exp.Before(now.Add(59*time.Minute)), "exp should be before ~59 minutes from now")
}
