package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

// CacheKeyBuilder helps construct consistent cache keys
type CacheKeyBuilder struct {
	namespace string
	parts     []string
}

// NewCacheKeyBuilder creates a new key builder with a namespace
func NewCacheKeyBuilder(namespace string) *CacheKeyBuilder {
	return &CacheKeyBuilder{
		namespace: namespace,
		parts:     make([]string, 0),
	}
}

// Add adds a part to the cache key
func (b *CacheKeyBuilder) Add(part string) *CacheKeyBuilder {
	b.parts = append(b.parts, part)
	return b
}

// AddParams adds URL parameters to the key
func (b *CacheKeyBuilder) AddParams(params map[string]string) *CacheKeyBuilder {
	if len(params) == 0 {
		return b
	}

	paramParts := make([]string, 0, len(params))
	for k, v := range params {
		paramParts = append(paramParts, fmt.Sprintf("%s=%s", k, v))
	}

	b.parts = append(b.parts, strings.Join(paramParts, "&"))
	return b
}

// Build constructs the final cache key
func (b *CacheKeyBuilder) Build() string {
	if b.namespace == "" {
		return strings.Join(b.parts, ":")
	}
	return b.namespace + ":" + strings.Join(b.parts, ":")
}

// Hash builds a hashed version of the key
func (b *CacheKeyBuilder) Hash() string {
	key := b.Build()
	hash := sha256.Sum256([]byte(key))
	return b.namespace + ":hash:" + hex.EncodeToString(hash[:])
}

// SonarCacheKey generates a cache key for Sonar API calls
func SonarCacheKey(projectID string, metrics []string) string {
	return NewCacheKeyBuilder("sonar").
		Add("measures").
		Add(projectID).
		Add(strings.Join(metrics, ",")).
		Build()
}

// JiraCacheKey generates a cache key for Jira API calls
func JiraCacheKey(endpoint, userID string, params map[string]string) string {
	return NewCacheKeyBuilder("jira").
		Add(endpoint).
		Add(userID).
		AddParams(params).
		Build()
}

// GitHubCacheKey generates a cache key for GitHub API calls
func GitHubCacheKey(endpoint, userID string, params map[string]string) string {
	return NewCacheKeyBuilder("github").
		Add(endpoint).
		Add(userID).
		AddParams(params).
		Build()
}