package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"developer-portal-backend/internal/auth"
	apperrors "developer-portal-backend/internal/errors"
	"developer-portal-backend/internal/logger"

	"github.com/google/go-github/v57/github"
	"golang.org/x/oauth2"
)

// GitHubService provides methods to interact with GitHub API
type GitHubService struct {
	authService GitHubAuthService
}

// NewGitHubService creates a new GitHub service
func NewGitHubService(authService *auth.AuthService) *GitHubService {
	return &GitHubService{
		authService: NewAuthServiceAdapter(authService),
	}
}

// NewGitHubServiceWithAdapter creates a new GitHub service with a custom auth service adapter
// This constructor is primarily for testing with mock auth services
func NewGitHubServiceWithAdapter(authService GitHubAuthService) *GitHubService {
	return &GitHubService{
		authService: authService,
	}
}

// PullRequest represents a GitHub pull request
type PullRequest struct {
	ID        int64      `json:"id" example:"1234567890"`
	Number    int        `json:"number" example:"42"`
	Title     string     `json:"title" example:"Add new feature"`
	State     string     `json:"state" example:"open"`
	CreatedAt time.Time  `json:"created_at" example:"2025-01-01T12:00:00Z"`
	UpdatedAt time.Time  `json:"updated_at" example:"2025-01-02T12:00:00Z"`
	HTMLURL   string     `json:"html_url" example:"https://github.com/owner/repo/pull/42"`
	User      GitHubUser `json:"user"`
	Repo      Repository `json:"repository"`
	Draft     bool       `json:"draft" example:"false"`
}

// GitHubUser represents a GitHub user
type GitHubUser struct {
	Login     string `json:"login" example:"johndoe"`
	ID        int64  `json:"id" example:"12345"`
	AvatarURL string `json:"avatar_url" example:"https://avatars.githubusercontent.com/u/12345"`
}

// Repository represents a GitHub repository
type Repository struct {
	Name     string `json:"name" example:"my-repo"`
	FullName string `json:"full_name" example:"owner/my-repo"`
	Owner    string `json:"owner" example:"owner"`
	Private  bool   `json:"private" example:"false"`
}

// PullRequestsResponse represents the response for pull requests
type PullRequestsResponse struct {
	PullRequests []PullRequest `json:"pull_requests"`
	Total        int           `json:"total"`
}

// TotalContributions Response represents the response for user contributions
type TotalContributionsResponse struct {
	TotalContributions int    `json:"total_contributions" example:"1234"`
	Period             string `json:"period" example:"365d"`
	From               string `json:"from" example:"2024-10-16T00:00:00Z"`
	To                 string `json:"to" example:"2025-10-16T23:59:59Z"`
}

// PRReviewCommentsResponse represents the response for PR review comments count
type PRReviewCommentsResponse struct {
	TotalComments int    `json:"total_comments" example:"42"`
	Period        string `json:"period" example:"30d"`
	From          string `json:"from" example:"2024-10-16T00:00:00Z"`
	To            string `json:"to" example:"2024-11-16T23:59:59Z"`
}

// ContributionDay represents contributions for a single day
type ContributionDay struct {
	Date              string `json:"date" example:"2025-01-15"`
	ContributionCount int    `json:"contribution_count" example:"5"`
	ContributionLevel string `json:"contribution_level" example:"SECOND_QUARTILE"`
	Color             string `json:"color" example:"#40c463"`
}

// ContributionWeek represents a week of contributions
type ContributionWeek struct {
	FirstDay         string            `json:"first_day" example:"2025-01-12"`
	ContributionDays []ContributionDay `json:"contribution_days"`
}

// ContributionsHeatmapResponse represents the response for contribution heatmap
type ContributionsHeatmapResponse struct {
	TotalContributions int                `json:"total_contributions" example:"1234"`
	Weeks              []ContributionWeek `json:"weeks"`
	From               string             `json:"from" example:"2024-10-16T00:00:00Z"`
	To                 string             `json:"to" example:"2025-10-16T23:59:59Z"`
}

// PRMergeTimeDataPoint represents a single data point for PR merge time metrics (weekly)
type PRMergeTimeDataPoint struct {
	WeekStart    string  `json:"week_start" example:"2024-10-15"`
	WeekEnd      string  `json:"week_end" example:"2024-10-21"`
	AverageHours float64 `json:"average_hours" example:"18.5"`
	PRCount      int     `json:"pr_count" example:"3"`
}

// AveragePRMergeTimeResponse represents the response for average PR merge time
type AveragePRMergeTimeResponse struct {
	AveragePRMergeTimeHours float64                `json:"average_pr_merge_time_hours" example:"24.5"`
	PRCount                 int                    `json:"pr_count" example:"15"`
	Period                  string                 `json:"period" example:"30d"`
	From                    string                 `json:"from" example:"2024-10-03T00:00:00Z"`
	To                      string                 `json:"to" example:"2024-11-02T23:59:59Z"`
	TimeSeries              []PRMergeTimeDataPoint `json:"time_series"`
}

// parseRepositoryFromURL extracts repository information from a GitHub URL
// Handles URLs like: https://github.com/owner/repo/pull/123
// or https://github.enterprise.com/owner/repo/pull/123
func parseRepositoryFromURL(urlStr string) (owner, repoName, fullName string) {
	if urlStr == "" {
		return "", "", ""
	}

	// Remove protocol and split by /
	urlStr = strings.TrimPrefix(urlStr, "https://")
	urlStr = strings.TrimPrefix(urlStr, "http://")
	parts := strings.Split(urlStr, "/")

	// We need at least: domain/owner/repo/...
	if len(parts) < 3 {
		return "", "", ""
	}

	// parts[0] is the domain (github.com or github.enterprise.com)
	// parts[1] is the owner
	// parts[2] is the repo name
	owner = parts[1]
	repoName = parts[2]
	fullName = owner + "/" + repoName

	return owner, repoName, fullName
}

// GetUserOpenPullRequests retrieves all open pull requests for the authenticated user
func (s *GitHubService) GetUserOpenPullRequests(ctx context.Context, claims *auth.AuthClaims, state, sort, direction string, perPage, page int) (*PullRequestsResponse, error) {
	if claims == nil {
		return nil, fmt.Errorf("authentication required")
	}

	// Get GitHub access token using validated JWT claims
	accessToken, err := s.authService.GetGitHubAccessTokenFromClaims(claims)
	if err != nil {
		return nil, fmt.Errorf("failed to get GitHub access token: %w", err)
	}

	// Get GitHub client configuration for the user's provider
	githubClientConfig, err := s.authService.GetGitHubClient(claims.Provider)
	if err != nil {
		return nil, fmt.Errorf("failed to get GitHub client: %w", err)
	}

	// Create OAuth2 client with access token
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: accessToken},
	)
	tc := oauth2.NewClient(ctx, ts)

	// Create authenticated GitHub client
	var client *github.Client
	if githubClientConfig != nil && githubClientConfig.GetEnterpriseBaseURL() != "" {
		client, err = github.NewEnterpriseClient(githubClientConfig.GetEnterpriseBaseURL(), githubClientConfig.GetEnterpriseBaseURL(), tc)
		if err != nil {
			return nil, fmt.Errorf("failed to create GitHub Enterprise client: %w", err)
		}
	} else {
		client = github.NewClient(tc)
	}

	// Set default values
	if state == "" {
		state = "open"
	}
	if sort == "" {
		sort = "created"
	}
	if direction == "" {
		direction = "desc"
	}
	if perPage <= 0 || perPage > 100 {
		perPage = 30
	}
	if page <= 0 {
		page = 1
	}

	// Search for pull requests created by the authenticated user
	// Using search API for better filtering capabilities
	// Note: GitHub Search API doesn't support state:all - omit state qualifier to get all PRs
	var query string
	if state == "all" {
		query = "is:pr author:@me"
	} else {
		query = fmt.Sprintf("is:pr author:@me state:%s", state)
	}

	searchOpts := &github.SearchOptions{
		Sort:  sort,
		Order: direction,
		ListOptions: github.ListOptions{
			PerPage: perPage,
			Page:    page,
		},
	}

	result, resp, err := client.Search.Issues(ctx, query, searchOpts)
	if err != nil {
		// Check if it's a rate limit error
		if resp != nil && resp.StatusCode == 403 {
			return nil, apperrors.ErrGitHubAPIRateLimitExceeded
		}
		return nil, fmt.Errorf("failed to search pull requests: %w", err)
	}

	// Convert GitHub issues (PRs are issues in GitHub API) to our PR structure
	pullRequests := make([]PullRequest, 0, len(result.Issues))
	for _, issue := range result.Issues {
		if issue.PullRequestLinks == nil {
			continue // Skip if it's not actually a PR
		}

		pr := PullRequest{
			ID:        issue.GetID(),
			Number:    issue.GetNumber(),
			Title:     issue.GetTitle(),
			State:     issue.GetState(),
			CreatedAt: issue.GetCreatedAt().Time,
			UpdatedAt: issue.GetUpdatedAt().Time,
			HTMLURL:   issue.GetHTMLURL(),
			Draft:     issue.GetDraft(),
			User: GitHubUser{
				Login:     issue.GetUser().GetLogin(),
				ID:        issue.GetUser().GetID(),
				AvatarURL: issue.GetUser().GetAvatarURL(),
			},
		}

		// Parse repository info from the issue
		if issue.Repository != nil {
			pr.Repo = Repository{
				Name:     issue.Repository.GetName(),
				FullName: issue.Repository.GetFullName(),
				Private:  issue.Repository.GetPrivate(),
			}
			if issue.Repository.Owner != nil {
				pr.Repo.Owner = issue.Repository.Owner.GetLogin()
			}
		} else {
			// Fallback: parse repository info from the HTML URL
			// GitHub Search API often doesn't include the Repository field
			owner, repoName, fullName := parseRepositoryFromURL(issue.GetHTMLURL())
			if owner != "" && repoName != "" {
				pr.Repo = Repository{
					Name:     repoName,
					FullName: fullName,
					Owner:    owner,
					// Note: We can't determine if the repo is private from the URL alone
					// Default to false, but this could be enhanced with an additional API call if needed
					Private: false,
				}
			}
		}

		pullRequests = append(pullRequests, pr)
	}

	response := &PullRequestsResponse{
		PullRequests: pullRequests,
		Total:        result.GetTotal(),
	}

	return response, nil
}

// GetUserTotalContributions retrieves the total contributions for the authenticated user over a specified period
func (s *GitHubService) GetUserTotalContributions(ctx context.Context, claims *auth.AuthClaims, period string) (*TotalContributionsResponse, error) {
	if claims == nil {
		return nil, fmt.Errorf("authentication required")
	}

	// Validate period format early (before making any API calls)
	var from, to time.Time
	var parsedPeriod string
	var query string

	if period == "" {
		// No period specified - use GitHub's default behavior
		parsedPeriod = "github_default"
		query = `{
			viewer {
				contributionsCollection {
					startedAt
					endedAt
					contributionCalendar {
						totalContributions
					}
				}
			}
		}`
	} else {
		// Validate period format before parsing
		if len(period) < 2 || period[len(period)-1] != 'd' {
			return nil, fmt.Errorf("%w: period must be in format '<number>d' (e.g., '30d', '90d', '365d')", apperrors.ErrInvalidPeriodFormat)
		}

		// Parse custom period and calculate date range
		var err error
		from, to, parsedPeriod, err = parsePeriod(period)
		if err != nil {
			return nil, fmt.Errorf("%w: %w", apperrors.ErrInvalidPeriodFormat, err)
		}

		query = fmt.Sprintf(`{
			viewer {
				contributionsCollection(from: "%s", to: "%s") {
					startedAt
					endedAt
					contributionCalendar {
						totalContributions
					}
				}
			}
		}`, from.Format(time.RFC3339), to.Format(time.RFC3339))
	}

	// Get GitHub access token using validated JWT claims
	accessToken, err := s.authService.GetGitHubAccessTokenFromClaims(claims)
	if err != nil {
		return nil, fmt.Errorf("failed to get GitHub access token: %w", err)
	}

	// Get GitHub client configuration for the user's provider
	githubClientConfig, err := s.authService.GetGitHubClient(claims.Provider)
	if err != nil {
		return nil, fmt.Errorf("failed to get GitHub client: %w", err)
	}

	// Execute GraphQL query
	var result struct {
		Viewer struct {
			ContributionsCollection struct {
				StartedAt            string `json:"startedAt"`
				EndedAt              string `json:"endedAt"`
				ContributionCalendar struct {
					TotalContributions int `json:"totalContributions"`
				} `json:"contributionCalendar"`
			} `json:"contributionsCollection"`
		} `json:"viewer"`
	}

	reqBody := struct {
		Query string `json:"query"`
	}{
		Query: query,
	}

	// Determine the correct GraphQL endpoint
	var graphqlURL string
	if githubClientConfig != nil && githubClientConfig.GetEnterpriseBaseURL() != "" {
		// GitHub Enterprise: Use /api/graphql (NOT /api/v3/graphql)
		graphqlURL = strings.TrimSuffix(githubClientConfig.GetEnterpriseBaseURL(), "/") + "/api/graphql"
	} else {
		// GitHub.com: Use standard GraphQL endpoint
		graphqlURL = "https://api.github.com/graphql"
	}

	// Create HTTP request manually for GraphQL
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal GraphQL query: %w", err)
	}

	ghReq, err := http.NewRequestWithContext(ctx, "POST", graphqlURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create GraphQL request: %w", err)
	}

	// Set headers
	ghReq.Header.Set("Authorization", "Bearer "+accessToken)
	ghReq.Header.Set("Content-Type", "application/json")
	ghReq.Header.Set("Accept", "application/json")

	// Execute request - respect context deadline if available
	httpClient := &http.Client{}
	if deadline, ok := ctx.Deadline(); ok {
		timeout := time.Until(deadline)
		if timeout > 0 {
			httpClient.Timeout = timeout
		} else {
			httpClient.Timeout = time.Second // Minimal timeout for expired contexts
		}
	} else {
		httpClient.Timeout = 30 * time.Second
	}
	resp, err := httpClient.Do(ghReq)
	if err != nil {
		return nil, fmt.Errorf("failed to execute GraphQL query: %w", err)
	}
	defer resp.Body.Close()

	// Check for rate limit
	if resp.StatusCode == 403 {
		return nil, apperrors.ErrGitHubAPIRateLimitExceeded
	}

	// Check for other HTTP errors
	if resp.StatusCode != 200 {
		bodyBytes, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return nil, fmt.Errorf("GraphQL query failed with status %d and failed to read response body: %w", resp.StatusCode, readErr)
		}
		return nil, fmt.Errorf("GraphQL query failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Parse response - GitHub GraphQL returns data in a wrapper
	var graphQLResponse struct {
		Data   json.RawMessage `json:"data"`
		Errors []struct {
			Message string   `json:"message"`
			Path    []string `json:"path,omitempty"`
		} `json:"errors,omitempty"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&graphQLResponse); err != nil {
		return nil, fmt.Errorf("failed to decode GraphQL response: %w", err)
	}

	// Check for GraphQL errors
	if len(graphQLResponse.Errors) > 0 {
		return nil, fmt.Errorf("GraphQL error: %s", graphQLResponse.Errors[0].Message)
	}

	// Parse the actual data
	if err := json.Unmarshal(graphQLResponse.Data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal result: %w", err)
	}

	// Use GitHub's actual date range from the response
	// This ensures we return what GitHub actually used, not just what we calculated
	fromStr := result.Viewer.ContributionsCollection.StartedAt
	toStr := result.Viewer.ContributionsCollection.EndedAt

	// If GitHub didn't return dates (shouldn't happen), fall back to calculated dates
	if fromStr == "" && !from.IsZero() {
		fromStr = from.Format(time.RFC3339)
	}
	if toStr == "" && !to.IsZero() {
		toStr = to.Format(time.RFC3339)
	}

	response := &TotalContributionsResponse{
		TotalContributions: result.Viewer.ContributionsCollection.ContributionCalendar.TotalContributions,
		Period:             parsedPeriod,
		From:               fromStr,
		To:                 toStr,
	}

	return response, nil
}

// GetContributionsHeatmap retrieves the contribution heatmap for the authenticated user
func (s *GitHubService) GetContributionsHeatmap(ctx context.Context, claims *auth.AuthClaims, period string) (*ContributionsHeatmapResponse, error) {
	if claims == nil {
		return nil, fmt.Errorf("authentication required")
	}

	log := logger.WithContext(ctx).WithFields(map[string]interface{}{
		"provider": claims.Provider,
		"period":   period,
	})

	log.Info("Fetching GitHub contribution heatmap")

	// Validate that the provider exists in configuration
	if _, err := s.authService.GetGitHubClient(claims.Provider); err != nil {
		log.Errorf("Provider '%s' not configured in auth.yaml", claims.Provider)
		return nil, fmt.Errorf("%w: provider '%s'. Please check available providers in auth.yaml", apperrors.ErrProviderNotConfigured, claims.Provider)
	}

	// Validate period format early (before making any API calls)
	var from, to time.Time
	var query string

	if period == "" {
		log.Debug("Using GitHub's default period for contribution heatmap")
		// No period specified - use GitHub's default behavior
		query = `{
			viewer {
				contributionsCollection {
					startedAt
					endedAt
					contributionCalendar {
						totalContributions
						weeks {
							firstDay
							contributionDays {
								date
								contributionCount
								contributionLevel
								color
							}
						}
					}
				}
			}
		}`
	} else {
		// Validate period format before parsing
		if len(period) < 2 || period[len(period)-1] != 'd' {
			log.Errorf("Invalid period format: %s", period)
			return nil, fmt.Errorf("%w: period must be in format '<number>d' (e.g., '30d', '90d', '365d')", apperrors.ErrInvalidPeriodFormat)
		}

		// Parse custom period and calculate date range
		var err error
		var parsedPeriod string
		from, to, parsedPeriod, err = parsePeriod(period)
		if err != nil {
			log.Errorf("Failed to parse period '%s': %v", period, err)
			return nil, fmt.Errorf("%w: %w", apperrors.ErrInvalidPeriodFormat, err)
		}
		_ = parsedPeriod // Used for validation, not needed in this response

		log.Debugf("Using custom period: from %s to %s", from.Format(time.RFC3339), to.Format(time.RFC3339))

		query = fmt.Sprintf(`{
			viewer {
				contributionsCollection(from: "%s", to: "%s") {
					startedAt
					endedAt
					contributionCalendar {
						totalContributions
						weeks {
							firstDay
							contributionDays {
								date
								contributionCount
								contributionLevel
								color
							}
						}
					}
				}
			}
		}`, from.Format(time.RFC3339), to.Format(time.RFC3339))
	}

	// Get GitHub access token using validated JWT claims
	accessToken, err := s.authService.GetGitHubAccessTokenFromClaims(claims)
	if err != nil {
		log.Errorf("Failed to get GitHub access token: %v", err)
		return nil, fmt.Errorf("failed to get GitHub access token: %w", err)
	}

	// Get GitHub client configuration for the user's provider
	githubClientConfig, err := s.authService.GetGitHubClient(claims.Provider)
	if err != nil {
		log.Errorf("Failed to get GitHub client for provider '%s': %v", claims.Provider, err)
		return nil, fmt.Errorf("failed to get GitHub client: %w", err)
	}

	// Execute GraphQL query
	var result struct {
		Viewer struct {
			ContributionsCollection struct {
				StartedAt            string `json:"startedAt"`
				EndedAt              string `json:"endedAt"`
				ContributionCalendar struct {
					TotalContributions int `json:"totalContributions"`
					Weeks              []struct {
						FirstDay         string `json:"firstDay"`
						ContributionDays []struct {
							Date              string `json:"date"`
							ContributionCount int    `json:"contributionCount"`
							ContributionLevel string `json:"contributionLevel"`
							Color             string `json:"color"`
						} `json:"contributionDays"`
					} `json:"weeks"`
				} `json:"contributionCalendar"`
			} `json:"contributionsCollection"`
		} `json:"viewer"`
	}

	reqBody := struct {
		Query string `json:"query"`
	}{
		Query: query,
	}

	// Determine the correct GraphQL endpoint
	var graphqlURL string
	if githubClientConfig != nil && githubClientConfig.GetEnterpriseBaseURL() != "" {
		// GitHub Enterprise: Use /api/graphql (NOT /api/v3/graphql)
		graphqlURL = strings.TrimSuffix(githubClientConfig.GetEnterpriseBaseURL(), "/") + "/api/graphql"
		log.Debugf("Using GitHub Enterprise GraphQL endpoint: %s", graphqlURL)
	} else {
		// GitHub.com: Use standard GraphQL endpoint
		graphqlURL = "https://api.github.com/graphql"
		log.Debug("Using GitHub.com GraphQL endpoint")
	}

	// Create HTTP request manually for GraphQL
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		log.Errorf("Failed to marshal GraphQL query: %v", err)
		return nil, fmt.Errorf("failed to marshal GraphQL query: %w", err)
	}

	log.Infof("Executing GraphQL query to %s", graphqlURL)

	ghReq, err := http.NewRequestWithContext(ctx, "POST", graphqlURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		log.Errorf("Failed to create GraphQL request: %v", err)
		return nil, fmt.Errorf("failed to create GraphQL request: %w", err)
	}

	// Set headers
	ghReq.Header.Set("Authorization", "Bearer "+accessToken)
	ghReq.Header.Set("Content-Type", "application/json")
	ghReq.Header.Set("Accept", "application/json")

	// Execute request - respect context deadline if available
	httpClient := &http.Client{}
	if deadline, ok := ctx.Deadline(); ok {
		timeout := time.Until(deadline)
		if timeout > 0 {
			httpClient.Timeout = timeout
		} else {
			httpClient.Timeout = time.Second // Minimal timeout for expired contexts
		}
	} else {
		httpClient.Timeout = 30 * time.Second
	}
	resp, err := httpClient.Do(ghReq)
	if err != nil {
		log.Errorf("Failed to execute GraphQL query: %v", err)
		return nil, fmt.Errorf("failed to execute GraphQL query: %w", err)
	}
	defer resp.Body.Close()

	log.Debugf("GitHub API response status: %d", resp.StatusCode)

	// Check for rate limit
	if resp.StatusCode == 403 {
		log.Warn("GitHub API rate limit exceeded")
		return nil, apperrors.ErrGitHubAPIRateLimitExceeded
	}

	// Check for other HTTP errors
	if resp.StatusCode != 200 {
		bodyBytes, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			log.Errorf("GraphQL query failed with status %d and failed to read response body", resp.StatusCode)
			return nil, fmt.Errorf("GraphQL query failed with status %d and failed to read response body: %w", resp.StatusCode, readErr)
		}
		log.Errorf("GraphQL query failed with status %d: %s", resp.StatusCode, string(bodyBytes))
		return nil, fmt.Errorf("GraphQL query failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Parse response - GitHub GraphQL returns data in a wrapper
	var graphQLResponse struct {
		Data   json.RawMessage `json:"data"`
		Errors []struct {
			Message string   `json:"message"`
			Path    []string `json:"path,omitempty"`
		} `json:"errors,omitempty"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&graphQLResponse); err != nil {
		log.Errorf("Failed to decode GraphQL response: %v", err)
		return nil, fmt.Errorf("failed to decode GraphQL response: %w", err)
	}

	// Check for GraphQL errors
	if len(graphQLResponse.Errors) > 0 {
		log.Errorf("GraphQL error: %s", graphQLResponse.Errors[0].Message)
		return nil, fmt.Errorf("GraphQL error: %s", graphQLResponse.Errors[0].Message)
	}

	// Parse the actual data
	if err := json.Unmarshal(graphQLResponse.Data, &result); err != nil {
		log.Errorf("Failed to unmarshal GraphQL result: %v", err)
		return nil, fmt.Errorf("failed to unmarshal result: %w", err)
	}

	log.Debugf("Successfully retrieved contribution heatmap with %d weeks", len(result.Viewer.ContributionsCollection.ContributionCalendar.Weeks))

	// Convert the result to our response format
	weeks := make([]ContributionWeek, 0, len(result.Viewer.ContributionsCollection.ContributionCalendar.Weeks))
	for _, week := range result.Viewer.ContributionsCollection.ContributionCalendar.Weeks {
		days := make([]ContributionDay, 0, len(week.ContributionDays))
		for _, day := range week.ContributionDays {
			days = append(days, ContributionDay{
				Date:              day.Date,
				ContributionCount: day.ContributionCount,
				ContributionLevel: day.ContributionLevel,
				Color:             day.Color,
			})
		}
		weeks = append(weeks, ContributionWeek{
			FirstDay:         week.FirstDay,
			ContributionDays: days,
		})
	}

	response := &ContributionsHeatmapResponse{
		TotalContributions: result.Viewer.ContributionsCollection.ContributionCalendar.TotalContributions,
		Weeks:              weeks,
		From:               result.Viewer.ContributionsCollection.StartedAt,
		To:                 result.Viewer.ContributionsCollection.EndedAt,
	}

	log.Infof("Successfully fetched contribution heatmap: %d total contributions from %s to %s",
		response.TotalContributions, response.From, response.To)

	return response, nil
}

// GetAveragePRMergeTime retrieves the average time to merge PRs for the authenticated user
func (s *GitHubService) GetAveragePRMergeTime(ctx context.Context, claims *auth.AuthClaims, period string) (*AveragePRMergeTimeResponse, error) {
	if claims == nil {
		return nil, fmt.Errorf("authentication required")
	}

	log := logger.WithContext(ctx).WithFields(map[string]interface{}{
		"provider": claims.Provider,
		"period":   period,
	})

	log.Info("Fetching average PR merge time")

	// Parse and validate period
	var from, to time.Time
	var parsedPeriod string
	var err error

	if period == "" {
		period = "30d"
	}

	from, to, parsedPeriod, err = parsePeriod(period)
	if err != nil {
		log.Errorf("Invalid period format: %s", period)
		return nil, fmt.Errorf("%w: %w", apperrors.ErrInvalidPeriodFormat, err)
	}

	log.Debugf("Querying merged PRs from %s to %s", from.Format(time.RFC3339), to.Format(time.RFC3339))

	// Get GitHub access token
	accessToken, err := s.authService.GetGitHubAccessTokenFromClaims(claims)
	if err != nil {
		log.Errorf("Failed to get GitHub access token: %v", err)
		return nil, fmt.Errorf("failed to get GitHub access token: %w", err)
	}

	// Get GitHub client configuration
	githubClientConfig, err := s.authService.GetGitHubClient(claims.Provider)
	if err != nil {
		log.Errorf("Failed to get GitHub client for provider '%s': %v", claims.Provider, err)
		return nil, fmt.Errorf("failed to get GitHub client: %w", err)
	}

	// Determine GraphQL endpoint
	var graphqlURL string
	if githubClientConfig != nil && githubClientConfig.GetEnterpriseBaseURL() != "" {
		graphqlURL = strings.TrimSuffix(githubClientConfig.GetEnterpriseBaseURL(), "/") + "/api/graphql"
		log.Debugf("Using GitHub Enterprise GraphQL endpoint: %s", graphqlURL)
	} else {
		graphqlURL = "https://api.github.com/graphql"
		log.Debug("Using GitHub.com GraphQL endpoint")
	}

	// Build search query for merged PRs
	searchQuery := fmt.Sprintf("is:pr author:@me is:merged merged:>=%s", from.Format("2006-01-02"))

	// Collect all PRs with pagination
	type prData struct {
		Number     int
		CreatedAt  string
		MergedAt   string
		Repository struct {
			Name  string
			Owner struct {
				Login string
			}
		}
	}

	allPRs := []prData{}
	hasNextPage := true
	cursor := ""

	for hasNextPage {
		// Build GraphQL query
		query := `query($q: String!, $first: Int!, $after: String) {
			search(query: $q, type: ISSUE, first: $first, after: $after) {
				pageInfo {
					hasNextPage
					endCursor
				}
				nodes {
					... on PullRequest {
						number
						createdAt
						mergedAt
						repository {
							name
							owner {
								login
							}
						}
					}
				}
			}
		}`

		variables := map[string]interface{}{
			"q":     searchQuery,
			"first": 100,
			"after": nil,
		}
		if cursor != "" {
			variables["after"] = cursor
		}

		reqBody := map[string]interface{}{
			"query":     query,
			"variables": variables,
		}

		jsonBody, err := json.Marshal(reqBody)
		if err != nil {
			log.Errorf("Failed to marshal GraphQL query: %v", err)
			return nil, fmt.Errorf("failed to marshal GraphQL query: %w", err)
		}

		ghReq, err := http.NewRequestWithContext(ctx, "POST", graphqlURL, bytes.NewBuffer(jsonBody))
		if err != nil {
			log.Errorf("Failed to create GraphQL request: %v", err)
			return nil, fmt.Errorf("failed to create GraphQL request: %w", err)
		}

		ghReq.Header.Set("Authorization", "Bearer "+accessToken)
		ghReq.Header.Set("Content-Type", "application/json")
		ghReq.Header.Set("Accept", "application/json")

		httpClient := &http.Client{}
		if deadline, ok := ctx.Deadline(); ok {
			timeout := time.Until(deadline)
			if timeout > 0 {
				httpClient.Timeout = timeout
			} else {
				httpClient.Timeout = time.Second
			}
		} else {
			httpClient.Timeout = 30 * time.Second
		}

		resp, err := httpClient.Do(ghReq)
		if err != nil {
			log.Errorf("Failed to execute GraphQL query: %v", err)
			return nil, fmt.Errorf("failed to execute GraphQL query: %w", err)
		}
		defer resp.Body.Close()

		log.Debugf("GitHub API response status: %d", resp.StatusCode)

		if resp.StatusCode == 403 {
			log.Warn("GitHub API rate limit exceeded")
			return nil, apperrors.ErrGitHubAPIRateLimitExceeded
		}

		if resp.StatusCode != 200 {
			bodyBytes, readErr := io.ReadAll(resp.Body)
			if readErr != nil {
				log.Errorf("GraphQL query failed with status %d and failed to read response body", resp.StatusCode)
				return nil, fmt.Errorf("GraphQL query failed with status %d and failed to read response body: %w", resp.StatusCode, readErr)
			}
			log.Errorf("GraphQL query failed with status %d: %s", resp.StatusCode, string(bodyBytes))
			return nil, fmt.Errorf("GraphQL query failed with status %d: %s", resp.StatusCode, string(bodyBytes))
		}

		var graphQLResponse struct {
			Data struct {
				Search struct {
					PageInfo struct {
						HasNextPage bool   `json:"hasNextPage"`
						EndCursor   string `json:"endCursor"`
					} `json:"pageInfo"`
					Nodes []prData `json:"nodes"`
				} `json:"search"`
			} `json:"data"`
			Errors []struct {
				Message string   `json:"message"`
				Path    []string `json:"path,omitempty"`
			} `json:"errors,omitempty"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&graphQLResponse); err != nil {
			log.Errorf("Failed to decode GraphQL response: %v", err)
			return nil, fmt.Errorf("failed to decode GraphQL response: %w", err)
		}

		if len(graphQLResponse.Errors) > 0 {
			log.Errorf("GraphQL error: %s", graphQLResponse.Errors[0].Message)
			return nil, fmt.Errorf("GraphQL error: %s", graphQLResponse.Errors[0].Message)
		}

		allPRs = append(allPRs, graphQLResponse.Data.Search.Nodes...)
		hasNextPage = graphQLResponse.Data.Search.PageInfo.HasNextPage
		cursor = graphQLResponse.Data.Search.PageInfo.EndCursor

		log.Debugf("Fetched %d PRs, hasNextPage: %v", len(graphQLResponse.Data.Search.Nodes), hasNextPage)
	}

	// Calculate merge times and group by week
	type weekData struct {
		totalHours float64
		count      int
		weekStart  time.Time
		weekEnd    time.Time
	}

	// Define 4 weeks going back from today
	now := time.Now().UTC()
	weeks := make([]*weekData, 4)
	for i := 0; i < 4; i++ {
		weekEnd := now.AddDate(0, 0, -7*i)
		weekStart := weekEnd.AddDate(0, 0, -7)
		weeks[i] = &weekData{
			weekStart: weekStart,
			weekEnd:   weekEnd,
		}
	}

	var totalHours float64
	var validPRCount int

	for _, pr := range allPRs {
		if pr.MergedAt == "" || pr.CreatedAt == "" {
			continue
		}

		createdAt, err := time.Parse(time.RFC3339, pr.CreatedAt)
		if err != nil {
			log.Warnf("Failed to parse createdAt for PR #%d: %v", pr.Number, err)
			continue
		}

		mergedAt, err := time.Parse(time.RFC3339, pr.MergedAt)
		if err != nil {
			log.Warnf("Failed to parse mergedAt for PR #%d: %v", pr.Number, err)
			continue
		}

		mergeTimeHours := mergedAt.Sub(createdAt).Hours()
		totalHours += mergeTimeHours
		validPRCount++

		// Assign PR to the appropriate week
		for _, week := range weeks {
			if (mergedAt.Equal(week.weekStart) || mergedAt.After(week.weekStart)) && mergedAt.Before(week.weekEnd) {
				week.totalHours += mergeTimeHours
				week.count++
				break
			}
		}
	}

	log.Infof("Total merged PRs found: %d, successfully processed: %d", len(allPRs), validPRCount)

	// Calculate overall average and round to 2 decimal places
	var averageHours float64
	if validPRCount > 0 {
		averageHours = roundTo2Decimals(totalHours / float64(validPRCount))
	}

	// Build time series (always 4 weeks, newest to oldest)
	timeSeries := make([]PRMergeTimeDataPoint, 4)
	for i := 0; i < 4; i++ {
		week := weeks[i]
		var avgForWeek float64
		if week.count > 0 {
			avgForWeek = roundTo2Decimals(week.totalHours / float64(week.count))
		}
		timeSeries[i] = PRMergeTimeDataPoint{
			WeekStart:    week.weekStart.Format("2006-01-02"),
			WeekEnd:      week.weekEnd.Format("2006-01-02"),
			AverageHours: avgForWeek,
			PRCount:      week.count,
		}
	}

	response := &AveragePRMergeTimeResponse{
		AveragePRMergeTimeHours: averageHours,
		PRCount:                 validPRCount,
		Period:                  parsedPeriod,
		From:                    from.Format(time.RFC3339),
		To:                      to.Format(time.RFC3339),
		TimeSeries:              timeSeries,
	}

	log.Infof("Successfully calculated average PR merge time: %.2f hours across %d PRs", averageHours, validPRCount)

	return response, nil
}

// roundTo2Decimals rounds a float64 to 2 decimal places
func roundTo2Decimals(num float64) float64 {
	return math.Round(num*100) / 100
}

// parsePeriod parses a period string (e.g., "30d", "90d", "365d") and returns the from/to dates
// Default period is 365 days if not specified or invalid
func parsePeriod(period string) (from, to time.Time, parsedPeriod string, err error) {
	to = time.Now().UTC()

	// Default to 365 days if empty or invalid
	days := 365
	parsedPeriod = "365d"

	if period != "" {
		// Parse period format like "30d", "90d", "365d"
		if len(period) < 2 || period[len(period)-1] != 'd' {
			return time.Time{}, time.Time{}, "", fmt.Errorf("period must be in format '<number>d' (e.g., '30d', '90d', '365d')")
		}

		var parseErr error
		days, parseErr = strconv.Atoi(period[:len(period)-1])
		if parseErr != nil || days <= 0 {
			return time.Time{}, time.Time{}, "", fmt.Errorf("period must contain a positive number of days")
		}

		// GitHub API supports max 1 year of contributions
		if days > 365 {
			days = 365
		}

		parsedPeriod = period
	}

	from = to.AddDate(0, 0, -days)

	// Set from to start of day and to to end of day
	from = time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, time.UTC)
	to = time.Date(to.Year(), to.Month(), to.Day(), 23, 59, 59, 0, time.UTC)

	return from, to, parsedPeriod, nil
}

// GetRepositoryContent fetches repository file or directory content from GitHub
func (s *GitHubService) GetRepositoryContent(ctx context.Context, claims *auth.AuthClaims, owner, repo, path, ref string) (interface{}, error) {
	// Get access token from auth service
	accessToken, err := s.authService.GetGitHubAccessTokenFromClaims(claims)
	if err != nil {
		return nil, fmt.Errorf("failed to get access token: %w", err)
	}

	// Get GitHub client configuration
	githubClientConfig, err := s.authService.GetGitHubClient(claims.Provider)
	if err != nil {
		return nil, err
	}

	// Create OAuth2 token source
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: accessToken},
	)
	tc := oauth2.NewClient(ctx, ts)

	// Create authenticated GitHub client
	var client *github.Client
	if githubClientConfig != nil && githubClientConfig.GetEnterpriseBaseURL() != "" {
		client, err = github.NewEnterpriseClient(githubClientConfig.GetEnterpriseBaseURL(), githubClientConfig.GetEnterpriseBaseURL(), tc)
		if err != nil {
			return nil, fmt.Errorf("failed to create GitHub Enterprise client: %w", err)
		}
	} else {
		client = github.NewClient(tc)
	}

	// Set default ref if not provided
	if ref == "" {
		ref = "main"
	}

	// Remove leading slash from path if present
	if len(path) > 0 && path[0] == '/' {
		path = path[1:]
	}

	// Fetch repository content
	fileContent, directoryContent, resp, err := client.Repositories.GetContents(
		ctx,
		owner,
		repo,
		path,
		&github.RepositoryContentGetOptions{
			Ref: ref,
		},
	)

	// Handle errors
	if err != nil {
		// Check for rate limit
		if resp != nil && resp.StatusCode == 403 {
			return nil, apperrors.ErrGitHubAPIRateLimitExceeded
		}
		// Check for not found
		if resp != nil && resp.StatusCode == 404 {
			return nil, apperrors.NewNotFoundError("repository content")
		}
		return nil, fmt.Errorf("failed to fetch repository content: %w", err)
	}

	// Return directory contents (array)
	if directoryContent != nil {
		result := make([]map[string]interface{}, len(directoryContent))
		for i, item := range directoryContent {
			result[i] = map[string]interface{}{
				"name":         item.GetName(),
				"path":         item.GetPath(),
				"sha":          item.GetSHA(),
				"size":         item.GetSize(),
				"url":          item.GetURL(),
				"html_url":     item.GetHTMLURL(),
				"git_url":      item.GetGitURL(),
				"download_url": item.GetDownloadURL(),
				"type":         item.GetType(),
				"_links": map[string]string{
					"self": item.GetURL(),
					"git":  item.GetGitURL(),
					"html": item.GetHTMLURL(),
				},
			}
		}
		return result, nil
	}

	// Return file content (object)
	if fileContent != nil {
		content, err := fileContent.GetContent()
		if err != nil {
			return nil, fmt.Errorf("failed to get file content: %w", err)
		}
		return map[string]interface{}{
			"name":         fileContent.GetName(),
			"path":         fileContent.GetPath(),
			"sha":          fileContent.GetSHA(),
			"size":         fileContent.GetSize(),
			"url":          fileContent.GetURL(),
			"html_url":     fileContent.GetHTMLURL(),
			"git_url":      fileContent.GetGitURL(),
			"download_url": fileContent.GetDownloadURL(),
			"type":         fileContent.GetType(),
			"content":      content,
			"encoding":     fileContent.GetEncoding(),
			"_links": map[string]string{
				"self": fileContent.GetURL(),
				"git":  fileContent.GetGitURL(),
				"html": fileContent.GetHTMLURL(),
			},
		}, nil
	}

	return nil, fmt.Errorf("unexpected response from GitHub API")
}

// UpdateRepositoryFile updates a file in a GitHub repository
func (s *GitHubService) UpdateRepositoryFile(ctx context.Context, claims *auth.AuthClaims, owner, repo, path, message, content, sha, branch string) (interface{}, error) {
	// Get access token from auth service
	accessToken, err := s.authService.GetGitHubAccessTokenFromClaims(claims)
	if err != nil {
		return nil, fmt.Errorf("failed to get access token: %w", err)
	}

	// Get GitHub client configuration
	githubClientConfig, err := s.authService.GetGitHubClient(claims.Provider)
	if err != nil {
		return nil, err
	}

	// Create OAuth2 token source
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: accessToken},
	)
	tc := oauth2.NewClient(ctx, ts)

	// Create authenticated GitHub client
	var client *github.Client
	if githubClientConfig != nil && githubClientConfig.GetEnterpriseBaseURL() != "" {
		client, err = github.NewEnterpriseClient(githubClientConfig.GetEnterpriseBaseURL(), githubClientConfig.GetEnterpriseBaseURL(), tc)
		if err != nil {
			return nil, fmt.Errorf("failed to create GitHub Enterprise client: %w", err)
		}
	} else {
		client = github.NewClient(tc)
	}

	// Remove leading slash from path if present
	if len(path) > 0 && path[0] == '/' {
		path = path[1:]
	}

	// Set default branch if not provided
	if branch == "" {
		branch = "main"
	}

	// Create update options
	opts := &github.RepositoryContentFileOptions{
		Message: github.String(message),
		Content: []byte(content),
		SHA:     github.String(sha),
		Branch:  github.String(branch),
	}

	// Update the file
	result, resp, err := client.Repositories.UpdateFile(ctx, owner, repo, path, opts)
	if err != nil {
		// Check for rate limit
		if resp != nil && resp.StatusCode == 403 {
			return nil, apperrors.ErrGitHubAPIRateLimitExceeded
		}
		// Check for not found
		if resp != nil && resp.StatusCode == 404 {
			return nil, apperrors.NewNotFoundError("repository or file")
		}
		return nil, fmt.Errorf("failed to update repository file: %w", err)
	}

	// Return the result
	return map[string]interface{}{
		"content": map[string]interface{}{
			"name":         result.Content.GetName(),
			"path":         result.Content.GetPath(),
			"sha":          result.Content.GetSHA(),
			"size":         result.Content.GetSize(),
			"url":          result.Content.GetURL(),
			"html_url":     result.Content.GetHTMLURL(),
			"git_url":      result.Content.GetGitURL(),
			"download_url": result.Content.GetDownloadURL(),
			"type":         result.Content.GetType(),
		},
		"commit": map[string]interface{}{
			"sha":      result.Commit.GetSHA(),
			"url":      result.Commit.GetURL(),
			"html_url": result.Commit.GetHTMLURL(),
			"message":  result.Commit.GetMessage(),
			"author": map[string]interface{}{
				"name":  result.Commit.Author.GetName(),
				"email": result.Commit.Author.GetEmail(),
				"date":  result.Commit.Author.GetDate(),
			},
			"committer": map[string]interface{}{
				"name":  result.Commit.Committer.GetName(),
				"email": result.Commit.Committer.GetEmail(),
				"date":  result.Commit.Committer.GetDate(),
			},
		},
	}, nil
}

// GetGitHubAsset fetches a GitHub asset (image, file, etc.) with authentication
func (s *GitHubService) GetGitHubAsset(ctx context.Context, claims *auth.AuthClaims, assetURL string) ([]byte, string, error) {
	log := logger.WithContext(ctx).WithFields(map[string]interface{}{
		"asset_url": assetURL,
		"provider":  claims.Provider,
		"user_id":   claims.UserID,
	})

	// Get access token from auth service
	accessToken, err := s.authService.GetGitHubAccessTokenFromClaims(claims)
	if err != nil {
		log.WithError(err).Error("Failed to get access token from claims")
		return nil, "", fmt.Errorf("failed to get access token: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "GET", assetURL, nil)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create request: %w", err)
	}

	// Add authorization header
	// GitHub asset URLs may require "token" prefix instead of "Bearer"
	req.Header.Set("Authorization", fmt.Sprintf("token %s", accessToken))
	req.Header.Set("Accept", "*/*")
	req.Header.Set("User-Agent", "Developer-Portal-Backend")

	// Make the request with redirect following
	// GitHub asset URLs redirect to media.github.tools.sap with a temporary token
	client := &http.Client{
		Timeout: 30 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// Follow up to 10 redirects (default)
			if len(via) >= 10 {
				return fmt.Errorf("stopped after 10 redirects")
			}
			// Preserve Authorization header only for same host
			// Don't send OAuth token to media.github.tools.sap (it uses query param token)
			if req.URL.Host != via[0].URL.Host {
				req.Header.Del("Authorization")
			}
			return nil
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		log.WithError(err).Error("Failed to fetch GitHub asset")
		return nil, "", fmt.Errorf("failed to fetch asset: %w", err)
	}
	defer resp.Body.Close()

	// Log response for debugging
	log.WithFields(map[string]interface{}{
		"status_code":    resp.StatusCode,
		"content_type":   resp.Header.Get("Content-Type"),
		"content_length": resp.Header.Get("Content-Length"),
	}).Debug("GitHub asset response received")

	// Check response status
	if resp.StatusCode == 403 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		log.WithFields(map[string]interface{}{
			"response_body": string(bodyBytes),
		}).Warn("GitHub API rate limit exceeded for asset")
		return nil, "", apperrors.ErrGitHubAPIRateLimitExceeded
	}
	if resp.StatusCode == 404 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		log.WithFields(map[string]interface{}{
			"response_body": string(bodyBytes),
		}).Warn("GitHub asset not found")
		return nil, "", fmt.Errorf("GitHub asset not found at URL: %s", assetURL)
	}
	if resp.StatusCode == 401 {
		// Read the error body to see what GitHub says
		bodyBytes, _ := io.ReadAll(resp.Body)
		log.WithFields(map[string]interface{}{
			"response_body": string(bodyBytes),
		}).Error("GitHub authentication failed with 'token' prefix")

		// Try with "Bearer" prefix instead of "token"
		req2, _ := http.NewRequestWithContext(ctx, "GET", assetURL, nil)
		req2.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
		req2.Header.Set("Accept", "*/*")
		req2.Header.Set("User-Agent", "Developer-Portal-Backend")

		resp2, err2 := client.Do(req2)
		if err2 != nil {
			return nil, "", fmt.Errorf("failed to fetch asset with Bearer auth: %w", err2)
		}
		defer resp2.Body.Close()

		if resp2.StatusCode != 200 {
			bodyBytes, _ := io.ReadAll(resp2.Body)
			log.WithFields(map[string]interface{}{
				"status_code": resp2.StatusCode,
				"body":        string(bodyBytes),
			}).Error("Authentication failed with both methods")
			return nil, "", fmt.Errorf("authentication failed with both token and Bearer: status %d", resp2.StatusCode)
		}

		resp = resp2
		log.Info("Successfully authenticated with 'Bearer' prefix")
	} else if resp.StatusCode != 200 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		log.WithFields(map[string]interface{}{
			"status_code": resp.StatusCode,
			"body":        string(bodyBytes),
		}).Error("Unexpected status code fetching asset")
		return nil, "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Read response body
	assetData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read asset data: %w", err)
	}

	// Get content type from response headers
	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	log.WithFields(map[string]interface{}{
		"size":         len(assetData),
		"content_type": contentType,
	}).Info("Successfully fetched GitHub asset")
	return assetData, contentType, nil
}


func (s *GitHubService) ClosePullRequest(ctx context.Context, claims *auth.AuthClaims, owner, repo string, prNumber int, deleteBranch bool) (*PullRequest, error) {
	if claims == nil {
		return nil, fmt.Errorf("authentication required")
	}
	if owner == "" || repo == "" {
		return nil, fmt.Errorf("owner and repo are required")
	}

	// GitHub access token using validated JWT claims
	accessToken, err := s.authService.GetGitHubAccessTokenFromClaims(claims)
	if err != nil {
		return nil, fmt.Errorf("failed to get GitHub access token: %w", err)
	}

	// Get GitHub client configuration for the user's provider
	githubClientConfig, err := s.authService.GetGitHubClient(claims.Provider)
	if err != nil {
		return nil, fmt.Errorf("failed to get GitHub client: %w", err)
	}

	// Create OAuth2 client with access token
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: accessToken},
	)
	tc := oauth2.NewClient(ctx, ts)

	// Create authenticated GitHub client
	var client *github.Client
	if githubClientConfig != nil && githubClientConfig.GetEnterpriseBaseURL() != "" {
		client, err = github.NewEnterpriseClient(githubClientConfig.GetEnterpriseBaseURL(), githubClientConfig.GetEnterpriseBaseURL(), tc)
		if err != nil {
			return nil, fmt.Errorf("failed to create GitHub Enterprise client: %w", err)
		}
	} else {
		client = github.NewClient(tc)
	}

	// Fetch PR to ensure it exists and check current state, also get head branch
	pr, resp, err := client.PullRequests.Get(ctx, owner, repo, prNumber)
	if err != nil {
		if resp != nil && resp.StatusCode == 403 {
			return nil, apperrors.ErrGitHubAPIRateLimitExceeded
		}
		if resp != nil && resp.StatusCode == 404 {
			return nil, apperrors.NewNotFoundError("pull request")
		}
		return nil, fmt.Errorf("failed to get pull request: %w", err)
	}

	// Only close open PRs
	if strings.EqualFold(pr.GetState(), "closed") {
		return nil, fmt.Errorf("%w: pull request is already closed", apperrors.ErrInvalidStatus)
	}

	// Close the PR
	updated, resp, err := client.PullRequests.Edit(ctx, owner, repo, prNumber, &github.PullRequest{
		State: github.String("closed"),
	})
	if err != nil {
		if resp != nil && resp.StatusCode == 403 {
			return nil, apperrors.ErrGitHubAPIRateLimitExceeded
		}
		if resp != nil && resp.StatusCode == 404 {
			return nil, apperrors.NewNotFoundError("pull request")
		}
		return nil, fmt.Errorf("failed to close pull request: %w", err)
	}

	// Optionally delete the PR branch
	if deleteBranch {
		head := updated.GetHead()
		branch := head.GetRef()
		headRepo := head.GetRepo()
		headRepoName := repo
		headOwner := owner
		if headRepo != nil {
			if headRepo.GetName() != "" {
				headRepoName = headRepo.GetName()
			}
			if headRepo.GetOwner() != nil && headRepo.GetOwner().GetLogin() != "" {
				headOwner = headRepo.GetOwner().GetLogin()
			}
		}
		if branch != "" && headRepoName != "" && headOwner != "" {
			ref := fmt.Sprintf("heads/%s", branch)
			delResp, delErr := client.Git.DeleteRef(ctx, headOwner, headRepoName, ref)
			if delErr != nil {
				if delResp != nil && delResp.StatusCode == 403 {
					return nil, apperrors.ErrGitHubAPIRateLimitExceeded
				}
				// Ignore 404 (branch already deleted or not found)
				if delResp == nil || delResp.StatusCode != 404 {
					return nil, fmt.Errorf("failed to delete branch '%s' in %s/%s: %w", branch, headOwner, headRepoName, delErr)
				}
			}
		}
	}

	// Convert to our PullRequest structure
	result := PullRequest{
		ID:        updated.GetID(),
		Number:    updated.GetNumber(),
		Title:     updated.GetTitle(),
		State:     updated.GetState(),
		CreatedAt: updated.GetCreatedAt().Time,
		UpdatedAt: updated.GetUpdatedAt().Time,
		HTMLURL:   updated.GetHTMLURL(),
		Draft:     updated.GetDraft(),
		User: GitHubUser{
			Login:     updated.GetUser().GetLogin(),
			ID:        updated.GetUser().GetID(),
			AvatarURL: updated.GetUser().GetAvatarURL(),
		},
		Repo: Repository{
			Name:     repo,
			FullName: owner + "/" + repo,
			Owner:    owner,
			Private:  false,
		},
	}

	return &result, nil
}

// GetUserPRReviewComments gets the total number of PR review comments made by the authenticated user
func (s *GitHubService) GetUserPRReviewComments(ctx context.Context, claims *auth.AuthClaims, period string) (*PRReviewCommentsResponse, error) {
	if claims == nil {
		return nil, fmt.Errorf("authentication required")
	}

	// Parse period (default to 30 days)
	var from, to time.Time
	var parsedPeriod string
	var err error

	if period == "" {
		period = "30d"
	}

	// Validate period format
	if len(period) < 2 || period[len(period)-1] != 'd' {
		return nil, fmt.Errorf("%w: period must be in format '<number>d' (e.g., '30d', '90d', '365d')", apperrors.ErrInvalidPeriodFormat)
	}

	// Parse custom period and calculate date range
	from, to, parsedPeriod, err = parsePeriod(period)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", apperrors.ErrInvalidPeriodFormat, err)
	}

	// Get GitHub access token using validated JWT claims
	accessToken, err := s.authService.GetGitHubAccessTokenFromClaims(claims)
	if err != nil {
		return nil, fmt.Errorf("failed to get GitHub access token: %w", err)
	}

	// Get GitHub client configuration for the user's provider
	githubClientConfig, err := s.authService.GetGitHubClient(claims.Provider)
	if err != nil {
		return nil, fmt.Errorf("failed to get GitHub client: %w", err)
	}

	// Create OAuth2 client with access token
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: accessToken},
	)
	tc := oauth2.NewClient(ctx, ts)

	// Create authenticated GitHub client
	var client *github.Client
	if githubClientConfig != nil && githubClientConfig.GetEnterpriseBaseURL() != "" {
		client, err = github.NewEnterpriseClient(githubClientConfig.GetEnterpriseBaseURL(), githubClientConfig.GetEnterpriseBaseURL(), tc)
		if err != nil {
			return nil, fmt.Errorf("failed to create GitHub Enterprise client: %w", err)
		}
	} else {
		client = github.NewClient(tc)
	}

	// Get the authenticated user
	user, resp, err := client.Users.Get(ctx, "")
	if err != nil {
		if resp != nil && resp.StatusCode == 403 {
			return nil, apperrors.ErrGitHubAPIRateLimitExceeded
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	username := user.GetLogin()

	// Search for pull request review comments by the user within the time period
	query := fmt.Sprintf("type:pr reviewed-by:%s created:%s..%s",
		username,
		from.Format("2006-01-02"),
		to.Format("2006-01-02"))

	searchOpts := &github.SearchOptions{
		ListOptions: github.ListOptions{
			PerPage: 100,
			Page:    1,
		},
	}

	totalComments := 0

	// Paginate through all results
	for {
		result, resp, err := client.Search.Issues(ctx, query, searchOpts)
		if err != nil {
			if resp != nil && resp.StatusCode == 403 {
				return nil, apperrors.ErrGitHubAPIRateLimitExceeded
			}
			return nil, fmt.Errorf("failed to search PR review comments: %w", err)
		}

		totalComments += result.GetTotal()

		// Check if there are more pages
		if resp.NextPage == 0 {
			break
		}
		searchOpts.Page = resp.NextPage
	}

	return &PRReviewCommentsResponse{
		TotalComments: totalComments,
		Period:        parsedPeriod,
		From:          from.Format(time.RFC3339),
		To:            to.Format(time.RFC3339),
	}, nil
}
