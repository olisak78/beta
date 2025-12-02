package cache

import "time"

// CacheService defines the interface for cache operations
type CacheService interface {
	Get(key string) ([]byte, error)
	Set(key string, value []byte, ttl time.Duration) error
	Delete(key string) error
	Exists(key string) bool
	Clear() error
	GetWithRefresh(key string, ttl time.Duration) ([]byte, error)
}

// CacheConfig holds configuration for cache implementations
type CacheConfig struct {
	DefaultTTL      time.Duration
	CleanupInterval time.Duration
	MaxItems        int
}

// EndpointCacheConfig defines TTL configuration per endpoint/service
type EndpointCacheConfig struct {
	SonarMeasures        time.Duration
	JiraIssues           time.Duration
	GitHubPullRequests   time.Duration
	GitHubContributions  time.Duration
	GitHubRepositoryInfo time.Duration
	ComponentHealth      time.Duration
	TeamData             time.Duration
	DefaultTTL           time.Duration
}

// DefaultEndpointConfig returns sensible default TTL values
func DefaultEndpointConfig() *EndpointCacheConfig {
	return &EndpointCacheConfig{
		SonarMeasures:        15 * time.Minute,
		JiraIssues:           5 * time.Minute,
		GitHubPullRequests:   3 * time.Minute,
		GitHubContributions:  30 * time.Minute,
		GitHubRepositoryInfo: 10 * time.Minute,
		ComponentHealth:      2 * time.Minute,
		TeamData:             10 * time.Minute,
		DefaultTTL:           5 * time.Minute,
	}
}