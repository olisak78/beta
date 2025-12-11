package cache

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewInMemoryCache(t *testing.T) {
	cache := NewInMemoryCache(5*time.Minute, 10*time.Minute)
	assert.NotNil(t, cache)
	assert.NotNil(t, cache.cache)
}

func TestInMemoryCache_SetAndGet(t *testing.T) {
	cache := NewInMemoryCache(5*time.Minute, 10*time.Minute)

	key := "test:key"
	value := []byte("test value")

	// Set value
	err := cache.Set(key, value, 1*time.Minute)
	require.NoError(t, err)

	// Get value
	retrieved, err := cache.Get(key)
	require.NoError(t, err)
	assert.Equal(t, value, retrieved)
}

func TestInMemoryCache_GetNonExistent(t *testing.T) {
	cache := NewInMemoryCache(5*time.Minute, 10*time.Minute)

	// Try to get non-existent key
	_, err := cache.Get("non:existent")
	assert.Error(t, err)
	assert.Equal(t, ErrKeyNotFound, err)
}

func TestInMemoryCache_Delete(t *testing.T) {
	cache := NewInMemoryCache(5*time.Minute, 10*time.Minute)

	key := "test:key"
	value := []byte("test value")

	// Set and verify
	err := cache.Set(key, value, 1*time.Minute)
	require.NoError(t, err)

	exists := cache.Exists(key)
	assert.True(t, exists)

	// Delete
	err = cache.Delete(key)
	require.NoError(t, err)

	// Verify deletion
	exists = cache.Exists(key)
	assert.False(t, exists)

	_, err = cache.Get(key)
	assert.Error(t, err)
}

func TestInMemoryCache_Exists(t *testing.T) {
	cache := NewInMemoryCache(5*time.Minute, 10*time.Minute)

	key := "test:key"
	value := []byte("test value")

	// Should not exist initially
	exists := cache.Exists(key)
	assert.False(t, exists)

	// Set value
	err := cache.Set(key, value, 1*time.Minute)
	require.NoError(t, err)

	// Should exist now
	exists = cache.Exists(key)
	assert.True(t, exists)
}

func TestInMemoryCache_Clear(t *testing.T) {
	cache := NewInMemoryCache(5*time.Minute, 10*time.Minute)

	// Set multiple values
	keys := []string{"key1", "key2", "key3"}
	for _, key := range keys {
		err := cache.Set(key, []byte("value"), 1*time.Minute)
		require.NoError(t, err)
	}

	// Verify all exist
	for _, key := range keys {
		assert.True(t, cache.Exists(key))
	}

	// Clear cache
	err := cache.Clear()
	require.NoError(t, err)

	// Verify all are gone
	for _, key := range keys {
		assert.False(t, cache.Exists(key))
	}
}

func TestInMemoryCache_Expiration(t *testing.T) {
	cache := NewInMemoryCache(5*time.Minute, 10*time.Millisecond)

	key := "test:key"
	value := []byte("test value")

	// Set with short TTL
	err := cache.Set(key, value, 50*time.Millisecond)
	require.NoError(t, err)

	// Should exist immediately
	retrieved, err := cache.Get(key)
	require.NoError(t, err)
	assert.Equal(t, value, retrieved)

	// Wait for expiration
	time.Sleep(100 * time.Millisecond)

	// Should be expired now
	_, err = cache.Get(key)
	assert.Error(t, err)
	assert.Equal(t, ErrKeyNotFound, err)
}

func TestInMemoryCache_GetWithRefresh(t *testing.T) {
	cache := NewInMemoryCache(5*time.Minute, 10*time.Minute)

	key := "test:key"
	value := []byte("test value")

	// Set with short TTL
	err := cache.Set(key, value, 100*time.Millisecond)
	require.NoError(t, err)

	// Wait a bit
	time.Sleep(50 * time.Millisecond)

	// Get with refresh - extend TTL to 1 minute
	retrieved, err := cache.GetWithRefresh(key, 1*time.Minute)
	require.NoError(t, err)
	assert.Equal(t, value, retrieved)

	// Wait past original TTL
	time.Sleep(100 * time.Millisecond)

	// Should still exist because we refreshed
	retrieved, err = cache.Get(key)
	require.NoError(t, err)
	assert.Equal(t, value, retrieved)
}

func TestInMemoryCache_ItemCount(t *testing.T) {
	cache := NewInMemoryCache(5*time.Minute, 10*time.Minute)

	// Initially empty
	assert.Equal(t, 0, cache.ItemCount())

	// Add items
	for i := 0; i < 5; i++ {
		key := string(rune('a' + i))
		err := cache.Set(key, []byte("value"), 1*time.Minute)
		require.NoError(t, err)
	}

	// Should have 5 items
	assert.Equal(t, 5, cache.ItemCount())

	// Delete one
	err := cache.Delete("a")
	require.NoError(t, err)

	// Should have 4 items
	assert.Equal(t, 4, cache.ItemCount())
}

func TestInMemoryCache_ConcurrentAccess(t *testing.T) {
	cache := NewInMemoryCache(5*time.Minute, 10*time.Minute)

	// Test concurrent writes and reads
	done := make(chan bool)
	iterations := 100

	// Writer goroutine
	go func() {
		for i := 0; i < iterations; i++ {
			key := "concurrent:key"
			value := []byte("value")
			_ = cache.Set(key, value, 1*time.Minute)
		}
		done <- true
	}()

	// Reader goroutine
	go func() {
		for i := 0; i < iterations; i++ {
			_, _ = cache.Get("concurrent:key")
		}
		done <- true
	}()

	// Wait for both goroutines
	<-done
	<-done

	// Test should complete without panic
	assert.True(t, true)
}

func TestInMemoryCache_InvalidValueType(t *testing.T) {
	cache := NewInMemoryCache(5*time.Minute, 10*time.Minute)

	// Directly set a non-[]byte value (this shouldn't happen in normal usage)
	// but we test the error handling
	cache.cache.Set("invalid", "string value", 1*time.Minute)

	_, err := cache.Get("invalid")
	assert.Error(t, err)
	assert.Equal(t, ErrInvalidValue, err)
}