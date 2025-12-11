package auth

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware provides JWT authentication middleware
type AuthMiddleware struct {
	service *AuthService
}

// NewAuthMiddleware creates a new authentication middleware
func NewAuthMiddleware(service *AuthService) *AuthMiddleware {
	return &AuthMiddleware{service: service}
}

// RequireAuth validates JWT tokens and sets user context
func (m *AuthMiddleware) RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header is required"})
			c.Abort()
			return
		}

		// Extract token from Bearer header
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization header format"})
			c.Abort()
			return
		}

		// Validate token
		claims, err := m.service.ValidateJWT(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token", "details": err.Error()})
			c.Abort()
			return
		}

		// Set user context
		c.Set("user_uuid", claims.UUID)
		c.Set("username", claims.Username)
		c.Set("email", claims.Email)
		c.Set("auth_claims", claims)

		c.Next()
	}
}

// RequireProvider validates that the request comes from a specific provider
func (m *AuthMiddleware) RequireProvider(allowedProviders ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		provider, exists := c.Get("provider")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
			c.Abort()
			return
		}

		providerStr, ok := provider.(string)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid provider context"})
			c.Abort()
			return
		}

		// Check if provider is allowed
		allowed := false
		for _, allowedProvider := range allowedProviders {
			if providerStr == allowedProvider {
				allowed = true
				break
			}
		}

		if !allowed {
			c.JSON(http.StatusForbidden, gin.H{"error": "Provider not allowed for this resource"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// GetUserID is a helper function to extract user ID from context
func GetUserID(c *gin.Context) (int64, bool) {
	userID, exists := c.Get("user_id")
	if !exists {
		return 0, false
	}

	id, ok := userID.(int64)
	return id, ok
}

// GetUsername is a helper function to extract username from context
func GetUsername(c *gin.Context) (string, bool) {
	username, exists := c.Get("username")
	if !exists {
		return "", false
	}

	name, ok := username.(string)
	return name, ok
}

// GetUserEmail is a helper function to extract user email from context
func GetUserEmail(c *gin.Context) (string, bool) {
	email, exists := c.Get("email")
	if !exists {
		return "", false
	}

	emailStr, ok := email.(string)
	return emailStr, ok
}

// GetProvider is a helper function to extract provider from context
func GetProvider(c *gin.Context) (string, bool) {
	provider, exists := c.Get("provider")
	if !exists {
		return "", false
	}

	providerStr, ok := provider.(string)
	return providerStr, ok
}

// GetEnvironment is a helper function to extract environment from context
func GetEnvironment(c *gin.Context) (string, bool) {
	environment, exists := c.Get("environment")
	if !exists {
		return "", false
	}

	envStr, ok := environment.(string)
	return envStr, ok
}

// GetAuthClaims is a helper function to extract full auth claims from context
func GetAuthClaims(c *gin.Context) (*AuthClaims, bool) {
	claims, exists := c.Get("auth_claims")
	if !exists {
		return nil, false
	}

	authClaims, ok := claims.(*AuthClaims)
	return authClaims, ok
}
