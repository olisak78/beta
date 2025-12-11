package auth

import (
	"context"
	"fmt"
	"net/http"

	"github.com/google/go-github/v57/github"
	"golang.org/x/oauth2"
)

// GitHubClient wraps the GitHub API client with authentication support
type GitHubClient struct {
	config *ProviderConfig
	client *github.Client
}

// UserProfile represents a GitHub user profile
type UserProfile struct {
	ID        int64  `json:"id"`
	Username  string `json:"username"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatarUrl"`
	UUID      string `json:"uuid"` // User UUID from database table
}

// NewGitHubClient creates a new GitHub API client
func NewGitHubClient(config *ProviderConfig) *GitHubClient {
	var client *github.Client

	if config.EnterpriseBaseURL != "" {
		// GitHub Enterprise Server
		client, _ = github.NewEnterpriseClient(config.EnterpriseBaseURL, config.EnterpriseBaseURL, nil)
	} else {
		// GitHub.com
		client = github.NewClient(nil)
	}

	return &GitHubClient{
		config: config,
		client: client,
	}
}

// GetUserProfile fetches user profile information from GitHub API
func (c *GitHubClient) GetUserProfile(ctx context.Context, accessToken string) (*UserProfile, error) {
	// Create OAuth2 client with access token
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: accessToken},
	)
	tc := oauth2.NewClient(ctx, ts)

	// Create GitHub client with authenticated HTTP client
	var client *github.Client
	if c.config.EnterpriseBaseURL != "" {
		client, _ = github.NewEnterpriseClient(c.config.EnterpriseBaseURL, c.config.EnterpriseBaseURL, tc)
	} else {
		client = github.NewClient(tc)
	}

	// Get authenticated user
	user, resp, err := client.Users.Get(ctx, "")
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusUnauthorized {
			return nil, fmt.Errorf("invalid access token")
		}
		return nil, fmt.Errorf("failed to get user profile: %w", err)
	}

	// Get user emails
	emails, _, err := client.Users.ListEmails(ctx, nil)
	if err != nil {
		// Don't fail if we can't get emails, just log it
		emails = []*github.UserEmail{}
	}

	// Find primary email
	primaryEmail := ""
	for _, email := range emails {
		if email.GetPrimary() {
			primaryEmail = email.GetEmail()
			break
		}
	}

	// If no primary email found, use the first verified email
	if primaryEmail == "" {
		for _, email := range emails {
			if email.GetVerified() {
				primaryEmail = email.GetEmail()
				break
			}
		}
	}

	// Fallback to user email from profile if available
	if primaryEmail == "" && user.GetEmail() != "" {
		primaryEmail = user.GetEmail()
	}

	profile := &UserProfile{
		ID:        user.GetID(),
		Username:  user.GetLogin(),
		Email:     primaryEmail,
		Name:      user.GetName(),
		AvatarURL: user.GetAvatarURL(),
	}

	return profile, nil
}

// GetOAuth2Config returns the OAuth2 configuration for this GitHub client
func (c *GitHubClient) GetOAuth2Config(redirectURL string) *oauth2.Config {
	var endpoint oauth2.Endpoint

	if c.config.EnterpriseBaseURL != "" {
		// GitHub Enterprise Server endpoints
		endpoint = oauth2.Endpoint{
			AuthURL:  fmt.Sprintf("%s/login/oauth/authorize", c.config.EnterpriseBaseURL),
			TokenURL: fmt.Sprintf("%s/login/oauth/access_token", c.config.EnterpriseBaseURL),
		}
	} else {
		// GitHub.com endpoints
		endpoint = oauth2.Endpoint{
			AuthURL:  "https://github.com/login/oauth/authorize",
			TokenURL: "https://github.com/login/oauth/access_token",
		}
	}

	return &oauth2.Config{
		ClientID:     c.config.ClientID,
		ClientSecret: c.config.ClientSecret,
		RedirectURL:  redirectURL,
		Scopes:       []string{"user:email", "read:user", "repo"},
		Endpoint:     endpoint,
	}
}

// ValidateConfig validates the GitHub client configuration
func (c *GitHubClient) ValidateConfig() error {
	if c.config.ClientID == "" {
		return fmt.Errorf("client ID is required")
	}
	if c.config.ClientSecret == "" {
		return fmt.Errorf("client secret is required")
	}
	return nil
}

// GetEnterpriseBaseURL returns the enterprise base URL if configured
func (c *GitHubClient) GetEnterpriseBaseURL() string {
	if c.config == nil {
		return ""
	}
	return c.config.EnterpriseBaseURL
}
