package cache

import (
	"encoding/json"
	"time"
	"fmt"

	"developer-portal-backend/internal/logger"
)

// CacheWrapper provides higher-level caching abstractions with automatic logging and error handling
type CacheWrapper struct {
	cache      CacheService
	defaultTTL time.Duration
}

// NewCacheWrapper creates a new cache wrapper
func NewCacheWrapper(cache CacheService, defaultTTL time.Duration) *CacheWrapper {
	return &CacheWrapper{
		cache:      cache,
		defaultTTL: defaultTTL,
	}
}

// GetOrSet retrieves from cache or executes fetcher function
// This is the low-level method that returns []byte
func (w *CacheWrapper) GetOrSet(key string, ttl time.Duration, fetcher func() (interface{}, error)) ([]byte, error) {
	// Try to get from cache first
	data, err := w.cache.Get(key)
	if err == nil {
		// Validate that cached data is valid JSON before returning
		var temp interface{}
		if err := json.Unmarshal(data, &temp); err != nil {
			logger.New().WithField("cache_key", key).WithError(err).Warn("Cached data is corrupted, treating as cache miss")
			// Fall through to fetcher
		} else {
			logger.New().WithField("cache_key", key).Debug("Cache hit")
			return data, nil
		}
	}

	// Cache miss or corrupted data - call fetcher
	logger.New().WithField("cache_key", key).Debug("Cache miss")

	result, err := fetcher()
	if err != nil {
		return nil, fmt.Errorf("fetcher failed: %w", err)
	}

	// Marshal the result
	data, err = json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	// Store in cache
	if err := w.cache.Set(key, data, ttl); err != nil {
		logger.New().WithField("cache_key", key).WithError(err).Warn("Failed to cache response")
	} else {
		logger.New().WithField("cache_key", key).Debug("Cached response")
	}

	return data, nil
}

// GetOrSetTyped remains the same - it's fine!
func (w *CacheWrapper) GetOrSetTyped(key string, ttl time.Duration, out interface{}, fetcher func() (interface{}, error)) error {
	data, err := w.GetOrSet(key, ttl, fetcher)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(data, out); err != nil {
		logger.New().WithField("cache_key", key).WithError(err).Warn("Failed to unmarshal cached data")
		return err
	}

	return nil
}

// GetJSON retrieves and unmarshals JSON from cache (without fetcher)
// Use this when you only want to check cache, not fetch on miss
func (w *CacheWrapper) GetJSON(key string, out interface{}) error {
	data, err := w.cache.Get(key)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(data, out); err != nil {
		logger.New().WithField("cache_key", key).WithError(err).Debug("Cache data corrupted")
		return err
	}

	logger.New().WithField("cache_key", key).Debug("Cache hit")
	return nil
}

// SetJSON marshals and stores JSON in cache
// Use this when you want to manually set cache without Get
func (w *CacheWrapper) SetJSON(key string, value interface{}, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		logger.New().WithField("cache_key", key).WithError(err).Error("Failed to marshal for cache")
		return err
	}

	if err := w.cache.Set(key, data, ttl); err != nil {
		logger.New().WithField("cache_key", key).WithError(err).Warn("Failed to cache response")
		return err
	}

	logger.New().WithField("cache_key", key).Debug("Cached response")
	return nil
}

// Delete removes a key from cache
func (w *CacheWrapper) Delete(key string) error {
	return w.cache.Delete(key)
}

// Exists checks if a key exists in cache
func (w *CacheWrapper) Exists(key string) bool {
	return w.cache.Exists(key)
}