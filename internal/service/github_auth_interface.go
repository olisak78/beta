package service

import (
	"fmt"

	"developer-portal-backend/internal/auth"
)

//go:generate mockgen -source=github_auth_interface.go -destination=../mocks/github_auth_mocks.go -package=mocks

// GitHubAuthService defines the interface for auth service methods needed by GitHub service
type GitHubAuthService interface {
	GetGitHubClient(provider string) (*auth.GitHubClient, error)
	GetGitHubAccessToken(userUUID, provider string) (string, error)
}

// authServiceAdapter adapts auth.AuthService to implement GitHubAuthService interface
type authServiceAdapter struct {
	authService *auth.AuthService
}

func (a *authServiceAdapter) GetGitHubAccessToken(userUUID, provider string) (string, error) {
	if a.authService == nil {
		return "", fmt.Errorf("auth service is not initialized")
	}
	return a.authService.GetGitHubAccessToken(userUUID, provider)
}

// NewAuthServiceAdapter creates an adapter for auth.AuthService
func NewAuthServiceAdapter(authService *auth.AuthService) GitHubAuthService {
	if authService == nil {
		return &authServiceAdapter{authService: nil}
	}
	return &authServiceAdapter{authService: authService}
}

func (a *authServiceAdapter) GetGitHubClient(provider string) (*auth.GitHubClient, error) {
	if a.authService == nil {
		return nil, fmt.Errorf("auth service is not initialized")
	}
	return a.authService.GetGitHubClient(provider)
}
