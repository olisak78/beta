package cache_test

import (
	"encoding/json"
	"testing"
	"time"

	"developer-portal-backend/internal/cache"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// CacheTestSuite defines the test suite for cache service
type CacheTestSuite struct {
	suite.Suite
	cache *cache.InMemoryCache
}

// SetupTest sets up the test suite
func (suite *CacheTestSuite) SetupTest() {
	config := cache.CacheConfig{
		DefaultTTL:      100 * time.Millisecond,
		CleanupInterval: 50 * time.Millisecond,
		Enabled:         true,
	}
	suite.cache = cache.NewInMemoryCache(config)
}

// TestGet_CacheMiss tests cache miss behavior
func (suite *CacheTestSuite) TestGet_CacheMiss() {
	_, err := suite.cache.Get("nonexistent-key")
	assert.ErrorIs(suite.T(), err, cache.ErrCacheMiss)
}

// TestSetAndGet tests basic set and get operations
func (suite *CacheTestSuite) TestSetAndGet() {
	key := "test-key"
	value := []byte("test-value")

	err := suite.cache.Set(key, value, 5*time.Minute)
	assert.NoError(suite.T(), err)

	retrieved, err := suite.cache.Get(key)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), value, retrieved)
}

// TestSetAndGet_WithDefaultTTL tests set with default TTL
func (suite *CacheTestSuite) TestSetAndGet_WithDefaultTTL() {
	key := "test-key-default-ttl"
	value := []byte("test-value")

	// Pass 0 to use default TTL
	err := suite.cache.Set(key, value, 0)
	assert.NoError(suite.T(), err)

	retrieved, err := suite.cache.Get(key)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), value, retrieved)
}

// TestDelete tests delete operation
func (suite *CacheTestSuite) TestDelete() {
	key := "test-key-delete"
	value := []byte("test-value")

	err := suite.cache.Set(key, value, 5*time.Minute)
	assert.NoError(suite.T(), err)

	err = suite.cache.Delete(key)
	assert.NoError(suite.T(), err)

	_, err = suite.cache.Get(key)
	assert.ErrorIs(suite.T(), err, cache.ErrCacheMiss)
}

// TestClear tests clear operation
func (suite *CacheTestSuite) TestClear() {
	// Set multiple values
	suite.cache.Set("key1", []byte("value1"), 5*time.Minute)
	suite.cache.Set("key2", []byte("value2"), 5*time.Minute)
	suite.cache.Set("key3", []byte("value3"), 5*time.Minute)

	suite.cache.Clear()

	// All should be gone
	_, err := suite.cache.Get("key1")
	assert.ErrorIs(suite.T(), err, cache.ErrCacheMiss)
	_, err = suite.cache.Get("key2")
	assert.ErrorIs(suite.T(), err, cache.ErrCacheMiss)
	_, err = suite.cache.Get("key3")
	assert.ErrorIs(suite.T(), err, cache.ErrCacheMiss)
}

// TestGetWithTTL tests getting value with TTL info
func (suite *CacheTestSuite) TestGetWithTTL() {
	key := "test-key-ttl"
	value := []byte("test-value")
	ttl := 5 * time.Minute

	err := suite.cache.Set(key, value, ttl)
	assert.NoError(suite.T(), err)

	retrieved, remainingTTL, found := suite.cache.GetWithTTL(key)
	assert.True(suite.T(), found)
	assert.Equal(suite.T(), value, retrieved)
	assert.True(suite.T(), remainingTTL > 0)
	assert.True(suite.T(), remainingTTL <= ttl)
}

// TestGetWithTTL_NotFound tests GetWithTTL for non-existent key
func (suite *CacheTestSuite) TestGetWithTTL_NotFound() {
	_, _, found := suite.cache.GetWithTTL("nonexistent")
	assert.False(suite.T(), found)
}

// TestExpiration tests that items expire after TTL
func (suite *CacheTestSuite) TestExpiration() {
	key := "test-key-expire"
	value := []byte("test-value")

	err := suite.cache.Set(key, value, 50*time.Millisecond)
	assert.NoError(suite.T(), err)

	// Should exist immediately
	retrieved, err := suite.cache.Get(key)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), value, retrieved)

	// Wait for expiration
	time.Sleep(100 * time.Millisecond)

	// Should be gone now
	_, err = suite.cache.Get(key)
	assert.ErrorIs(suite.T(), err, cache.ErrCacheMiss)
}

// TestDisabledCache tests cache behavior when disabled
func (suite *CacheTestSuite) TestDisabledCache() {
	suite.cache.SetEnabled(false)

	key := "test-key-disabled"
	value := []byte("test-value")

	// Set should succeed silently
	err := suite.cache.Set(key, value, 5*time.Minute)
	assert.NoError(suite.T(), err)

	// Get should return disabled error
	_, err = suite.cache.Get(key)
	assert.ErrorIs(suite.T(), err, cache.ErrCacheDisabled)

	// Re-enable for other tests
	suite.cache.SetEnabled(true)
}

// TestStats tests statistics retrieval
func (suite *CacheTestSuite) TestStats() {
	suite.cache.Set("key1", []byte("value1"), 5*time.Minute)
	suite.cache.Set("key2", []byte("value2"), 5*time.Minute)

	stats := suite.cache.Stats()
	assert.Equal(suite.T(), 2, stats["item_count"])
	assert.Equal(suite.T(), true, stats["enabled"])
}

// TestJSONValues tests caching JSON values
func (suite *CacheTestSuite) TestJSONValues() {
	type TestStruct struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}

	original := TestStruct{ID: "123", Name: "Test"}
	data, err := json.Marshal(original)
	assert.NoError(suite.T(), err)

	err = suite.cache.Set("json-key", data, 5*time.Minute)
	assert.NoError(suite.T(), err)

	retrieved, err := suite.cache.Get("json-key")
	assert.NoError(suite.T(), err)

	var decoded TestStruct
	err = json.Unmarshal(retrieved, &decoded)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), original, decoded)
}

// Run the test suite
func TestCacheTestSuite(t *testing.T) {
	suite.Run(t, new(CacheTestSuite))
}

// NoOpCacheTestSuite tests the no-op cache implementation
type NoOpCacheTestSuite struct {
	suite.Suite
	cache *cache.NoOpCache
}

func (suite *NoOpCacheTestSuite) SetupTest() {
	suite.cache = cache.NewNoOpCache()
}

func (suite *NoOpCacheTestSuite) TestGet_AlwaysDisabled() {
	_, err := suite.cache.Get("any-key")
	assert.ErrorIs(suite.T(), err, cache.ErrCacheDisabled)
}

func (suite *NoOpCacheTestSuite) TestSet_NoOp() {
	err := suite.cache.Set("any-key", []byte("value"), 5*time.Minute)
	assert.NoError(suite.T(), err)

	// Still returns disabled on get
	_, err = suite.cache.Get("any-key")
	assert.ErrorIs(suite.T(), err, cache.ErrCacheDisabled)
}

func (suite *NoOpCacheTestSuite) TestDelete_NoOp() {
	err := suite.cache.Delete("any-key")
	assert.NoError(suite.T(), err)
}

func (suite *NoOpCacheTestSuite) TestGetWithTTL_NotFound() {
	_, _, found := suite.cache.GetWithTTL("any-key")
	assert.False(suite.T(), found)
}

func TestNoOpCacheTestSuite(t *testing.T) {
	suite.Run(t, new(NoOpCacheTestSuite))
}

// CacheWrapperTestSuite tests the cache wrapper
type CacheWrapperTestSuite struct {
	suite.Suite
	cache   *cache.InMemoryCache
	wrapper *cache.CacheWrapper[string]
}

func (suite *CacheWrapperTestSuite) SetupTest() {
	config := cache.CacheConfig{
		DefaultTTL:      5 * time.Minute,
		CleanupInterval: 10 * time.Minute,
		Enabled:         true,
	}
	suite.cache = cache.NewInMemoryCache(config)
	suite.wrapper = cache.NewCacheWrapper[string](suite.cache)
}

func (suite *CacheWrapperTestSuite) TestGetOrFetch_CacheMiss_FetchesAndCaches() {
	fetchCalled := false
	expectedValue := "fetched-value"

	result, err := suite.wrapper.GetOrFetch("wrapper-key", 5*time.Minute, func() (string, error) {
		fetchCalled = true
		return expectedValue, nil
	})

	assert.NoError(suite.T(), err)
	assert.True(suite.T(), fetchCalled)
	assert.Equal(suite.T(), expectedValue, result)

	// Second call should use cache
	fetchCalled = false
	result, err = suite.wrapper.GetOrFetch("wrapper-key", 5*time.Minute, func() (string, error) {
		fetchCalled = true
		return "should-not-be-called", nil
	})

	assert.NoError(suite.T(), err)
	assert.False(suite.T(), fetchCalled)
	assert.Equal(suite.T(), expectedValue, result)
}

func (suite *CacheWrapperTestSuite) TestGetOrFetch_FetchError() {
	expectedError := assert.AnError

	result, err := suite.wrapper.GetOrFetch("error-key", 5*time.Minute, func() (string, error) {
		return "", expectedError
	})

	assert.ErrorIs(suite.T(), err, expectedError)
	assert.Empty(suite.T(), result)
}

func (suite *CacheWrapperTestSuite) TestInvalidate() {
	// First, populate the cache
	suite.wrapper.GetOrFetch("invalidate-key", 5*time.Minute, func() (string, error) {
		return "value", nil
	})

	// Invalidate
	err := suite.wrapper.Invalidate("invalidate-key")
	assert.NoError(suite.T(), err)

	// Should fetch again
	fetchCalled := false
	suite.wrapper.GetOrFetch("invalidate-key", 5*time.Minute, func() (string, error) {
		fetchCalled = true
		return "new-value", nil
	})

	assert.True(suite.T(), fetchCalled)
}

func TestCacheWrapperTestSuite(t *testing.T) {
	suite.Run(t, new(CacheWrapperTestSuite))
}

// TTLConfigTests tests TTL configuration
func TestDefaultTTLConfig(t *testing.T) {
	config := cache.DefaultTTLConfig()

	assert.Equal(t, 5*time.Minute, config.LandscapeList)
	assert.Equal(t, 5*time.Minute, config.LandscapeByID)
	assert.Equal(t, 5*time.Minute, config.LandscapeByName)
	assert.Equal(t, 5*time.Minute, config.LandscapeByProject)
	assert.Equal(t, 2*time.Minute, config.LandscapeSearch)
	assert.Equal(t, 2*time.Minute, config.JiraIssues)
	assert.Equal(t, 1*time.Minute, config.JiraIssuesCount)
	assert.Equal(t, 3*time.Minute, config.GitHubPullRequests)
	assert.Equal(t, 10*time.Minute, config.GitHubContributions)
	assert.Equal(t, 5*time.Minute, config.SonarMeasures)
	assert.Equal(t, 5*time.Minute, config.Default)
}

func TestBuildKey(t *testing.T) {
	tests := []struct {
		name     string
		prefix   cache.CacheKeyPrefix
		parts    []string
		expected string
	}{
		{
			name:     "single part",
			prefix:   cache.KeyPrefixLandscapeByID,
			parts:    []string{"123"},
			expected: "landscape:id:123",
		},
		{
			name:     "multiple parts",
			prefix:   cache.KeyPrefixLandscapeByProject,
			parts:    []string{"project1", "all"},
			expected: "landscape:project:project1:all",
		},
		{
			name:     "no parts",
			prefix:   cache.KeyPrefixLandscapeList,
			parts:    []string{},
			expected: "landscape:list",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := cache.BuildKey(tc.prefix, tc.parts...)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestDefaultCacheConfig(t *testing.T) {
	config := cache.DefaultCacheConfig()

	assert.Equal(t, 5*time.Minute, config.DefaultTTL)
	assert.Equal(t, 10*time.Minute, config.CleanupInterval)
	assert.True(t, config.Enabled)
}
