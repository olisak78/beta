package cache

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCacheKeyBuilder_Basic(t *testing.T) {
	builder := NewCacheKeyBuilder("test")
	key := builder.Add("user").Add("123").Build()

	assert.Equal(t, "test:user:123", key)
}

func TestCacheKeyBuilder_NoNamespace(t *testing.T) {
	builder := NewCacheKeyBuilder("")
	key := builder.Add("user").Add("123").Build()

	assert.Equal(t, "user:123", key)
}

func TestCacheKeyBuilder_WithParams(t *testing.T) {
	builder := NewCacheKeyBuilder("api")
	params := map[string]string{
		"status": "open",
		"limit":  "10",
	}
	key := builder.Add("prs").AddParams(params).Build()

	// The key should contain the params (order may vary due to map iteration)
	assert.Contains(t, key, "api:prs:")
	assert.Contains(t, key, "status=open")
	assert.Contains(t, key, "limit=10")
}

func TestCacheKeyBuilder_EmptyParams(t *testing.T) {
	builder := NewCacheKeyBuilder("api")
	params := map[string]string{}
	key := builder.Add("prs").AddParams(params).Build()

	assert.Equal(t, "api:prs", key)
}

func TestCacheKeyBuilder_Hash(t *testing.T) {
	builder := NewCacheKeyBuilder("test")
	key := builder.Add("very").Add("long").Add("key").Add("with").Add("many").Add("parts").Hash()

	assert.Contains(t, key, "test:hash:")
	assert.Len(t, key, len("test:hash:")+64) // SHA256 produces 64 hex chars
}

func TestSonarCacheKey(t *testing.T) {
	key := SonarCacheKey("project123", []string{"coverage", "bugs", "vulnerabilities"})
	expected := "sonar:measures:project123:coverage,bugs,vulnerabilities"

	assert.Equal(t, expected, key)
}

func TestJiraCacheKey(t *testing.T) {
	params := map[string]string{
		"status": "open",
		"limit":  "50",
	}
	key := JiraCacheKey("issues", "user123", params)

	assert.Contains(t, key, "jira:issues:user123:")
	assert.Contains(t, key, "status=open")
	assert.Contains(t, key, "limit=50")
}

func TestGitHubCacheKey(t *testing.T) {
	params := map[string]string{
		"state": "open",
		"sort":  "created",
	}
	key := GitHubCacheKey("pull-requests", "user456", params)

	assert.Contains(t, key, "github:pull-requests:user456:")
	assert.Contains(t, key, "state=open")
	assert.Contains(t, key, "sort=created")
}

func TestComponentHealthCacheKey(t *testing.T) {
	key := ComponentHealthCacheKey("component789")
	expected := "health:component:component789"

	assert.Equal(t, expected, key)
}

func TestTeamDataCacheKey(t *testing.T) {
	key := TeamDataCacheKey("team456")
	expected := "team:team456"

	assert.Equal(t, expected, key)
}

func TestCacheWrapper_GetOrSet_CacheHit(t *testing.T) {
	cache := NewInMemoryCache(5*time.Minute, 10*time.Minute)
	wrapper := NewCacheWrapper(cache, 5*time.Minute)

	key := "test:key"
	expectedValue := map[string]string{"name": "John", "role": "Developer"}

	// Pre-populate cache
	data, err := json.Marshal(expectedValue)
	require.NoError(t, err)
	err = cache.Set(key, data, 1*time.Minute)
	require.NoError(t, err)

	// Fetcher should not be called
	fetcherCalled := false
	fetcher := func() (interface{}, error) {
		fetcherCalled = true
		return nil, nil
	}

	result, err := wrapper.GetOrSet(key, 1*time.Minute, fetcher)
	require.NoError(t, err)
	assert.False(t, fetcherCalled, "Fetcher should not be called on cache hit")

	var retrieved map[string]string
	err = json.Unmarshal(result, &retrieved)
	require.NoError(t, err)
	assert.Equal(t, expectedValue, retrieved)
}

func TestCacheWrapper_GetOrSet_CacheMiss(t *testing.T) {
	cache := NewInMemoryCache(5*time.Minute, 10*time.Minute)
	wrapper := NewCacheWrapper(cache, 5*time.Minute)

	key := "test:key"
	expectedValue := map[string]string{"name": "Jane", "role": "Manager"}

	// Fetcher should be called
	fetcherCalled := false
	fetcher := func() (interface{}, error) {
		fetcherCalled = true
		return expectedValue, nil
	}

	result, err := wrapper.GetOrSet(key, 1*time.Minute, fetcher)
	require.NoError(t, err)
	assert.True(t, fetcherCalled, "Fetcher should be called on cache miss")

	var retrieved map[string]string
	err = json.Unmarshal(result, &retrieved)
	require.NoError(t, err)
	assert.Equal(t, expectedValue, retrieved)

	// Verify value was cached
	cachedData, err := cache.Get(key)
	require.NoError(t, err)
	var cached map[string]string
	err = json.Unmarshal(cachedData, &cached)
	require.NoError(t, err)
	assert.Equal(t, expectedValue, cached)
}

func TestCacheWrapper_GetOrSet_FetcherError(t *testing.T) {
	cache := NewInMemoryCache(5*time.Minute, 10*time.Minute)
	wrapper := NewCacheWrapper(cache, 5*time.Minute)

	key := "test:key"
	expectedError := errors.New("fetch failed")

	fetcher := func() (interface{}, error) {
		return nil, expectedError
	}

	_, err := wrapper.GetOrSet(key, 1*time.Minute, fetcher)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "fetcher failed")
}

func TestCacheWrapper_GetJSON(t *testing.T) {
	cache := NewInMemoryCache(5*time.Minute, 10*time.Minute)
	wrapper := NewCacheWrapper(cache, 5*time.Minute)

	key := "test:json"
	original := map[string]int{"count": 42, "total": 100}

	// Set JSON
	err := wrapper.SetJSON(key, original, 1*time.Minute)
	require.NoError(t, err)

	// Get JSON
	var retrieved map[string]int
	err = wrapper.GetJSON(key, &retrieved)
	require.NoError(t, err)
	assert.Equal(t, original, retrieved)
}

func TestCacheWrapper_SetJSON(t *testing.T) {
	cache := NewInMemoryCache(5*time.Minute, 10*time.Minute)
	wrapper := NewCacheWrapper(cache, 5*time.Minute)

	key := "test:json"
	value := struct {
		Name  string `json:"name"`
		Age   int    `json:"age"`
		Email string `json:"email"`
	}{
		Name:  "Alice",
		Age:   30,
		Email: "alice@example.com",
	}

	err := wrapper.SetJSON(key, value, 1*time.Minute)
	require.NoError(t, err)

	// Verify it was stored
	data, err := cache.Get(key)
	require.NoError(t, err)

	var retrieved struct {
		Name  string `json:"name"`
		Age   int    `json:"age"`
		Email string `json:"email"`
	}
	err = json.Unmarshal(data, &retrieved)
	require.NoError(t, err)
	assert.Equal(t, value, retrieved)
}

func TestCacheWrapper_GetJSON_NotFound(t *testing.T) {
	cache := NewInMemoryCache(5*time.Minute, 10*time.Minute)
	wrapper := NewCacheWrapper(cache, 5*time.Minute)

	var result map[string]string
	err := wrapper.GetJSON("non:existent", &result)
	assert.Error(t, err)
}

func TestInvalidatePattern(t *testing.T) {
	cache := NewInMemoryCache(5*time.Minute, 10*time.Minute)

	// This should return an error as pattern matching is not yet implemented
	err := InvalidatePattern(cache, "test:*")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not implemented")
}

func TestDefaultEndpointConfig(t *testing.T) {
	config := DefaultEndpointConfig()

	assert.NotNil(t, config)
	assert.Equal(t, 15*time.Minute, config.SonarMeasures)
	assert.Equal(t, 5*time.Minute, config.JiraIssues)
	assert.Equal(t, 3*time.Minute, config.GitHubPullRequests)
	assert.Equal(t, 30*time.Minute, config.GitHubContributions)
	assert.Equal(t, 10*time.Minute, config.GitHubRepositoryInfo)
	assert.Equal(t, 2*time.Minute, config.ComponentHealth)
	assert.Equal(t, 10*time.Minute, config.TeamData)
	assert.Equal(t, 5*time.Minute, config.DefaultTTL)
}