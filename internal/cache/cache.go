package cache

import (
	"encoding/json"
	"errors"
	"time"

	gocache "github.com/patrickmn/go-cache"
)

// Common cache errors
var (
	ErrCacheMiss     = errors.New("cache miss")
	ErrCacheDisabled = errors.New("cache is disabled")
)

// CacheService defines the interface for caching operations.
// This interface allows swapping in-memory implementation for Redis without major refactoring.
type CacheService interface {
	// Get retrieves a value from cache by key
	Get(key string) ([]byte, error)
	// Set stores a value in cache with the given TTL
	Set(key string, value []byte, ttl time.Duration) error
	// Delete removes a value from cache
	Delete(key string) error
	// Clear removes all items from cache
	Clear()
	// GetWithTTL retrieves a value and checks if it exists
	GetWithTTL(key string) ([]byte, time.Duration, bool)
}

// CacheConfig holds configuration for the cache service
type CacheConfig struct {
	// DefaultTTL is the default expiration time for cached items
	DefaultTTL time.Duration
	// CleanupInterval is how often expired items are cleaned up
	CleanupInterval time.Duration
	// Enabled determines if caching is active
	Enabled bool
}

// DefaultCacheConfig returns a sensible default configuration
func DefaultCacheConfig() CacheConfig {
	return CacheConfig{
		DefaultTTL:      5 * time.Minute,
		CleanupInterval: 10 * time.Minute,
		Enabled:         true,
	}
}

// InMemoryCache implements CacheService using go-cache
type InMemoryCache struct {
	cache   *gocache.Cache
	config  CacheConfig
	enabled bool
}

// NewInMemoryCache creates a new in-memory cache instance
func NewInMemoryCache(config CacheConfig) *InMemoryCache {
	return &InMemoryCache{
		cache:   gocache.New(config.DefaultTTL, config.CleanupInterval),
		config:  config,
		enabled: config.Enabled,
	}
}

// Get retrieves a value from the cache
func (c *InMemoryCache) Get(key string) ([]byte, error) {
	if !c.enabled {
		return nil, ErrCacheDisabled
	}

	value, found := c.cache.Get(key)
	if !found {
		return nil, ErrCacheMiss
	}

	data, ok := value.([]byte)
	if !ok {
		return nil, ErrCacheMiss
	}

	return data, nil
}

// Set stores a value in the cache with the given TTL
func (c *InMemoryCache) Set(key string, value []byte, ttl time.Duration) error {
	if !c.enabled {
		return nil
	}

	if ttl <= 0 {
		ttl = c.config.DefaultTTL
	}

	c.cache.Set(key, value, ttl)
	return nil
}

// Delete removes a value from the cache
func (c *InMemoryCache) Delete(key string) error {
	if !c.enabled {
		return nil
	}

	c.cache.Delete(key)
	return nil
}

// Clear removes all items from the cache
func (c *InMemoryCache) Clear() {
	if c.enabled {
		c.cache.Flush()
	}
}

// GetWithTTL retrieves a value and returns remaining TTL
func (c *InMemoryCache) GetWithTTL(key string) ([]byte, time.Duration, bool) {
	if !c.enabled {
		return nil, 0, false
	}

	value, expiration, found := c.cache.GetWithExpiration(key)
	if !found {
		return nil, 0, false
	}

	data, ok := value.([]byte)
	if !ok {
		return nil, 0, false
	}

	var remainingTTL time.Duration
	if !expiration.IsZero() {
		remainingTTL = time.Until(expiration)
	}

	return data, remainingTTL, true
}

// IsEnabled returns whether the cache is enabled
func (c *InMemoryCache) IsEnabled() bool {
	return c.enabled
}

// SetEnabled enables or disables the cache
func (c *InMemoryCache) SetEnabled(enabled bool) {
	c.enabled = enabled
}

// Stats returns cache statistics
func (c *InMemoryCache) Stats() map[string]interface{} {
	return map[string]interface{}{
		"item_count": c.cache.ItemCount(),
		"enabled":    c.enabled,
	}
}

// NoOpCache implements CacheService but does nothing (useful for testing or when cache is disabled)
type NoOpCache struct{}

// NewNoOpCache creates a new no-op cache instance
func NewNoOpCache() *NoOpCache {
	return &NoOpCache{}
}

// Get always returns cache miss
func (c *NoOpCache) Get(key string) ([]byte, error) {
	return nil, ErrCacheDisabled
}

// Set does nothing
func (c *NoOpCache) Set(key string, value []byte, ttl time.Duration) error {
	return nil
}

// Delete does nothing
func (c *NoOpCache) Delete(key string) error {
	return nil
}

// Clear does nothing
func (c *NoOpCache) Clear() {}

// GetWithTTL always returns not found
func (c *NoOpCache) GetWithTTL(key string) ([]byte, time.Duration, bool) {
	return nil, 0, false
}

// CacheWrapper provides helper methods for common caching patterns
type CacheWrapper[T any] struct {
	cache CacheService
}

// NewCacheWrapper creates a new cache wrapper for type T
func NewCacheWrapper[T any](cache CacheService) *CacheWrapper[T] {
	return &CacheWrapper[T]{cache: cache}
}

// GetOrFetch attempts to get from cache, or fetches and caches if not found
func (w *CacheWrapper[T]) GetOrFetch(key string, ttl time.Duration, fetchFn func() (T, error)) (T, error) {
	var result T

	// Try to get from cache
	data, err := w.cache.Get(key)
	if err == nil {
		if unmarshalErr := json.Unmarshal(data, &result); unmarshalErr == nil {
			return result, nil
		}
	}

	// Fetch fresh data
	result, err = fetchFn()
	if err != nil {
		return result, err
	}

	// Cache the result (ignore cache errors)
	if data, marshalErr := json.Marshal(result); marshalErr == nil {
		_ = w.cache.Set(key, data, ttl)
	}

	return result, nil
}

// Invalidate removes an item from the cache
func (w *CacheWrapper[T]) Invalidate(key string) error {
	return w.cache.Delete(key)
}

// InvalidatePattern removes all items matching a key prefix
// Note: This is a basic implementation. For Redis, this would use SCAN/KEYS
func (w *CacheWrapper[T]) InvalidatePattern(prefix string) {
	// In-memory cache doesn't support pattern-based deletion directly
	// This would need to be implemented differently for Redis
	// For now, we rely on individual key deletion
}
