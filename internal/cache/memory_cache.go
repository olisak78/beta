package cache

import (
	"errors"
	"time"

	gocache "github.com/patrickmn/go-cache"
)

var (
	ErrKeyNotFound  = errors.New("key not found in cache")
	ErrInvalidValue = errors.New("invalid cached value type")
)

// InMemoryCache implements CacheService using go-cache
type InMemoryCache struct {
	cache *gocache.Cache
}

// NewInMemoryCache creates a new in-memory cache instance
func NewInMemoryCache(defaultTTL, cleanupInterval time.Duration) *InMemoryCache {
	return &InMemoryCache{
		cache: gocache.New(defaultTTL, cleanupInterval),
	}
}

// Get retrieves a value from cache by key
func (c *InMemoryCache) Get(key string) ([]byte, error) {
	value, found := c.cache.Get(key)
	if !found {
		return nil, ErrKeyNotFound
	}

	bytes, ok := value.([]byte)
	if !ok {
		return nil, ErrInvalidValue
	}

	return bytes, nil
}

// Set stores a value in cache with the specified TTL
func (c *InMemoryCache) Set(key string, value []byte, ttl time.Duration) error {
	c.cache.Set(key, value, ttl)
	return nil
}

// Delete removes a key from cache
func (c *InMemoryCache) Delete(key string) error {
	c.cache.Delete(key)
	return nil
}

// Exists checks if a key exists in cache
func (c *InMemoryCache) Exists(key string) bool {
	_, found := c.cache.Get(key)
	return found
}

// Clear removes all keys from cache
func (c *InMemoryCache) Clear() error {
	c.cache.Flush()
	return nil
}

// GetWithRefresh retrieves a value and extends its TTL
func (c *InMemoryCache) GetWithRefresh(key string, ttl time.Duration) ([]byte, error) {
	value, _, found := c.cache.GetWithExpiration(key)
	if !found {
		return nil, ErrKeyNotFound
	}

	bytes, ok := value.([]byte)
	if !ok {
		return nil, ErrInvalidValue
	}

	// Extend the TTL
	c.cache.Set(key, value, ttl)

	return bytes, nil
}

// ItemCount returns the number of items in the cache
func (c *InMemoryCache) ItemCount() int {
	return c.cache.ItemCount()
}