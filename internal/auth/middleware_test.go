package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupMiddlewareTestService(t *testing.T) *AuthService {
	t.Helper()
	config := &AuthConfig{
		JWTSecret:   "test-signing-key-mw",
		RedirectURL: "http://localhost:3000",
		Providers: map[string]ProviderConfig{
			"githubtools": {
				ClientID:     "id",
				ClientSecret: "secret",
			},
		},
	}
	service, err := NewAuthService(config, nil, &noopTokenStore{})
	require.NoError(t, err)
	return service
}

func TestRequireAuth_NotExpiredToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	service := setupMiddlewareTestService(t)
	// set expiration to 1 minute from  now
	now := time.Now()
	claims := &AuthClaims{
		Username: "user58",
		Email:    "user58@example.com",
		UUID:     "uuid-58",
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(1 * time.Minute)),
			Issuer:    "developer-portal",
		},
	}
	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token, err := jwtToken.SignedString([]byte(service.config.JWTSecret))
	require.NoError(t, err)
	require.NotEmpty(t, token)

	// Setup router with middleware
	mw := NewAuthMiddleware(service)
	r := gin.New()
	r.GET("/protected", mw.RequireAuth(), func(c *gin.Context) {
		// Ensure claims were placed in context
		if claims, ok := GetAuthClaims(c); ok && claims != nil {
			c.String(http.StatusOK, "ok")
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "missing claims"})
	})

	// Execute request with Bearer token
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "token with < 1h expiry should pass middleware")
	assert.Equal(t, "ok", w.Body.String())
}

func TestRequireAuth_ExpiredToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	service := setupMiddlewareTestService(t)

	// set expiration to 1 minute ago
	claims := &AuthClaims{
		Username: "expired-user",
		Email:    "expired@example.com",
		UUID:     "uuid-expired",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Minute)),
			Issuer:    "developer-portal",
		},
	}
	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	expiredToken, err := jwtToken.SignedString([]byte(service.config.JWTSecret))
	require.NoError(t, err)

	// Setup router with middleware
	mw := NewAuthMiddleware(service)
	r := gin.New()
	r.GET("/protected", mw.RequireAuth(), func(c *gin.Context) {
		c.String(http.StatusOK, "ok-should-not-reach")
	})

	// Execute request with expired Bearer token
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+expiredToken)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code, "expired token (> 1h) should be rejected by middleware")

	// Validate error response shape
	var resp map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	_, hasErr := resp["error"]
	assert.True(t, hasErr, "response should include error message")
}
