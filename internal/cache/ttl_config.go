package cache

import "time"

// TTLConfig defines cache TTL durations for different endpoints/services
// These can be adjusted based on data freshness requirements
type TTLConfig struct {
	// Landscape service TTLs
	LandscapeList      time.Duration
	LandscapeByID      time.Duration
	LandscapeByName    time.Duration
	LandscapeByProject time.Duration
	LandscapeSearch    time.Duration

	// External API TTLs (for future use with Jira, GitHub, Sonar)
	JiraIssues          time.Duration
	JiraIssuesCount     time.Duration
	GitHubPullRequests  time.Duration
	GitHubContributions time.Duration
	SonarMeasures       time.Duration

	// Component service TTLs
	ComponentList   time.Duration
	ComponentByID   time.Duration
	ComponentHealth time.Duration

	// Default TTL for unspecified endpoints
	Default time.Duration
}

// DefaultTTLConfig returns default TTL configuration
// These values can be overridden via environment variables or config file
func DefaultTTLConfig() TTLConfig {
	return TTLConfig{
		// Landscape data changes infrequently - use longer TTLs
		LandscapeList:      5 * time.Minute,
		LandscapeByID:      5 * time.Minute,
		LandscapeByName:    5 * time.Minute,
		LandscapeByProject: 5 * time.Minute,
		LandscapeSearch:    2 * time.Minute,

		// External APIs - shorter TTLs for more dynamic data
		JiraIssues:          2 * time.Minute,
		JiraIssuesCount:     1 * time.Minute,
		GitHubPullRequests:  3 * time.Minute,
		GitHubContributions: 10 * time.Minute,
		SonarMeasures:       5 * time.Minute,

		// Component data
		ComponentList:   5 * time.Minute,
		ComponentByID:   5 * time.Minute,
		ComponentHealth: 30 * time.Second, // Health checks need to be fresh

		// Default
		Default: 5 * time.Minute,
	}
}

// CacheKeyPrefix defines prefixes for cache keys to organize cached data
type CacheKeyPrefix string

const (
	// Landscape cache key prefixes
	KeyPrefixLandscapeList      CacheKeyPrefix = "landscape:list"
	KeyPrefixLandscapeByID      CacheKeyPrefix = "landscape:id"
	KeyPrefixLandscapeByName    CacheKeyPrefix = "landscape:name"
	KeyPrefixLandscapeByProject CacheKeyPrefix = "landscape:project"
	KeyPrefixLandscapeSearch    CacheKeyPrefix = "landscape:search"

	// External API cache key prefixes
	KeyPrefixJiraIssues      CacheKeyPrefix = "jira:issues"
	KeyPrefixJiraIssuesCount CacheKeyPrefix = "jira:issues:count"
	KeyPrefixGitHubPRs       CacheKeyPrefix = "github:prs"
	KeyPrefixGitHubContrib   CacheKeyPrefix = "github:contributions"
	KeyPrefixSonarMeasures   CacheKeyPrefix = "sonar:measures"

	// Component cache key prefixes
	KeyPrefixComponentList   CacheKeyPrefix = "component:list"
	KeyPrefixComponentByID   CacheKeyPrefix = "component:id"
	KeyPrefixComponentHealth CacheKeyPrefix = "component:health"
)

// BuildKey constructs a cache key from prefix and identifiers
func BuildKey(prefix CacheKeyPrefix, parts ...string) string {
	key := string(prefix)
	for _, part := range parts {
		key += ":" + part
	}
	return key
}
