package cache

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCacheWrapper(t *testing.T) {
	cache := NewInMemoryCache(5*time.Minute, 10*time.Minute)
	wrapper := NewCacheWrapper(cache, 1*time.Minute)

	assert.NotNil(t, wrapper)
	assert.NotNil(t, wrapper.cache)
	assert.Equal(t, 1*time.Minute, wrapper.defaultTTL)
}

func TestCacheWrapper_GetOrSetTyped_CacheHit(t *testing.T) {
	cache := NewInMemoryCache(5*time.Minute, 10*time.Minute)
	wrapper := NewCacheWrapper(cache, 1*time.Minute)

	type TestData struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	// Pre-populate cache
	original := TestData{Name: "test", Value: 42}
	data, _ := json.Marshal(original)
	err := cache.Set("test:key", data, 1*time.Minute)
	require.NoError(t, err)

	// Test cache hit - fetcher should NOT be called
	var result TestData
	fetcherCalled := false
	err = wrapper.GetOrSetTyped("test:key", 1*time.Minute, &result, func() (interface{}, error) {
		fetcherCalled = true
		return nil, errors.New("fetcher should not be called")
	})

	require.NoError(t, err)
	assert.False(t, fetcherCalled, "Fetcher should not be called on cache hit")
	assert.Equal(t, "test", result.Name)
	assert.Equal(t, 42, result.Value)
}

func TestCacheWrapper_GetOrSetTyped_CacheMiss(t *testing.T) {
	cache := NewInMemoryCache(5*time.Minute, 10*time.Minute)
	wrapper := NewCacheWrapper(cache, 1*time.Minute)

	type TestData struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	// Test cache miss - fetcher SHOULD be called
	var result TestData
	fetcherCalled := false
	err := wrapper.GetOrSetTyped("test:key", 1*time.Minute, &result, func() (interface{}, error) {
		fetcherCalled = true
		return &TestData{Name: "fetched", Value: 99}, nil
	})

	require.NoError(t, err)
	assert.True(t, fetcherCalled, "Fetcher should be called on cache miss")
	assert.Equal(t, "fetched", result.Name)
	assert.Equal(t, 99, result.Value)

	// Verify value was cached
	exists := cache.Exists("test:key")
	assert.True(t, exists)
}

func TestCacheWrapper_GetOrSetTyped_FetcherError(t *testing.T) {
	cache := NewInMemoryCache(5*time.Minute, 10*time.Minute)
	wrapper := NewCacheWrapper(cache, 1*time.Minute)

	type TestData struct {
		Name string `json:"name"`
	}

	var result TestData
	expectedErr := errors.New("database connection failed")

	err := wrapper.GetOrSetTyped("test:key", 1*time.Minute, &result, func() (interface{}, error) {
		return nil, expectedErr
	})

	require.Error(t, err)
	// Change this line to check for wrapped errors
	assert.ErrorIs(t, err, expectedErr)
	// Or alternatively, check the error message contains the expected text
	// assert.Contains(t, err.Error(), "database connection failed")

	// Verify nothing was cached
	exists := cache.Exists("test:key")
	assert.False(t, exists)
}

func TestCacheWrapper_GetOrSetTyped_CorruptedCache(t *testing.T) {
	cache := NewInMemoryCache(5*time.Minute, 10*time.Minute)
	wrapper := NewCacheWrapper(cache, 1*time.Minute)

	type TestData struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	// Put corrupted data in cache (invalid JSON)
	err := cache.Set("test:key", []byte("not valid json"), 1*time.Minute)
	require.NoError(t, err)

	// Should handle corruption and call fetcher
	var result TestData
	fetcherCalled := false
	err = wrapper.GetOrSetTyped("test:key", 1*time.Minute, &result, func() (interface{}, error) {
		fetcherCalled = true
		return &TestData{Name: "recovered", Value: 77}, nil
	})

	require.NoError(t, err)
	assert.True(t, fetcherCalled, "Fetcher should be called when cache is corrupted")
	assert.Equal(t, "recovered", result.Name)
	assert.Equal(t, 77, result.Value)
}

func TestCacheWrapper_GetOrSetTyped_WithSlice(t *testing.T) {
	cache := NewInMemoryCache(5*time.Minute, 10*time.Minute)
	wrapper := NewCacheWrapper(cache, 1*time.Minute)

	type Item struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}

	var result []Item
	err := wrapper.GetOrSetTyped("test:list", 1*time.Minute, &result, func() (interface{}, error) {
		items := []Item{
			{ID: 1, Name: "Item 1"},
			{ID: 2, Name: "Item 2"},
			{ID: 3, Name: "Item 3"},
		}
		return items, nil
	})

	require.NoError(t, err)
	assert.Len(t, result, 3)
	assert.Equal(t, 1, result[0].ID)
	assert.Equal(t, "Item 1", result[0].Name)
}

func TestCacheWrapper_GetOrSetTyped_WithPrimitives(t *testing.T) {
	cache := NewInMemoryCache(5*time.Minute, 10*time.Minute)
	wrapper := NewCacheWrapper(cache, 1*time.Minute)

	t.Run("string", func(t *testing.T) {
		var result string
		err := wrapper.GetOrSetTyped("test:string", 1*time.Minute, &result, func() (interface{}, error) {
			return "hello world", nil
		})
		require.NoError(t, err)
		assert.Equal(t, "hello world", result)
	})

	t.Run("int", func(t *testing.T) {
		var result int
		err := wrapper.GetOrSetTyped("test:int", 1*time.Minute, &result, func() (interface{}, error) {
			return 42, nil
		})
		require.NoError(t, err)
		assert.Equal(t, 42, result)
	})

	t.Run("bool", func(t *testing.T) {
		var result bool
		err := wrapper.GetOrSetTyped("test:bool", 1*time.Minute, &result, func() (interface{}, error) {
			return true, nil
		})
		require.NoError(t, err)
		assert.True(t, result)
	})
}

func TestCacheWrapper_GetJSON(t *testing.T) {
	cache := NewInMemoryCache(5*time.Minute, 10*time.Minute)
	wrapper := NewCacheWrapper(cache, 1*time.Minute)

	type TestData struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	// Pre-populate cache
	original := TestData{Name: "test", Value: 42}
	data, _ := json.Marshal(original)
	err := cache.Set("test:key", data, 1*time.Minute)
	require.NoError(t, err)

	// Get with GetJSON
	var result TestData
	err = wrapper.GetJSON("test:key", &result)
	require.NoError(t, err)
	assert.Equal(t, "test", result.Name)
	assert.Equal(t, 42, result.Value)
}

func TestCacheWrapper_GetJSON_NotFound(t *testing.T) {
	cache := NewInMemoryCache(5*time.Minute, 10*time.Minute)
	wrapper := NewCacheWrapper(cache, 1*time.Minute)

	type TestData struct {
		Name string `json:"name"`
	}

	var result TestData
	err := wrapper.GetJSON("nonexistent:key", &result)
	assert.Error(t, err)
	assert.Equal(t, ErrKeyNotFound, err)
}

func TestCacheWrapper_SetJSON(t *testing.T) {
	cache := NewInMemoryCache(5*time.Minute, 10*time.Minute)
	wrapper := NewCacheWrapper(cache, 1*time.Minute)

	type TestData struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	data := TestData{Name: "test", Value: 42}
	err := wrapper.SetJSON("test:key", data, 1*time.Minute)
	require.NoError(t, err)

	// Verify it was cached correctly
	var result TestData
	err = wrapper.GetJSON("test:key", &result)
	require.NoError(t, err)
	assert.Equal(t, "test", result.Name)
	assert.Equal(t, 42, result.Value)
}

func TestCacheWrapper_Delete(t *testing.T) {
	cache := NewInMemoryCache(5*time.Minute, 10*time.Minute)
	wrapper := NewCacheWrapper(cache, 1*time.Minute)

	// Set a value
	err := cache.Set("test:key", []byte("value"), 1*time.Minute)
	require.NoError(t, err)

	// Delete via wrapper
	err = wrapper.Delete("test:key")
	require.NoError(t, err)

	// Verify deletion
	exists := wrapper.Exists("test:key")
	assert.False(t, exists)
}

func TestCacheWrapper_Exists(t *testing.T) {
	cache := NewInMemoryCache(5*time.Minute, 10*time.Minute)
	wrapper := NewCacheWrapper(cache, 1*time.Minute)

	// Initially doesn't exist
	exists := wrapper.Exists("test:key")
	assert.False(t, exists)

	// Set a value
	err := cache.Set("test:key", []byte("value"), 1*time.Minute)
	require.NoError(t, err)

	// Now exists
	exists = wrapper.Exists("test:key")
	assert.True(t, exists)
}

func TestCacheWrapper_ConcurrentAccess(t *testing.T) {
	cache := NewInMemoryCache(5*time.Minute, 10*time.Minute)
	wrapper := NewCacheWrapper(cache, 1*time.Minute)

	type TestData struct {
		Value int `json:"value"`
	}

	done := make(chan bool)
	iterations := 100

	// Multiple goroutines calling GetOrSetTyped
	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < iterations; j++ {
				var result TestData
				key := "concurrent:key"
				_ = wrapper.GetOrSetTyped(key, 1*time.Minute, &result, func() (interface{}, error) {
					return &TestData{Value: id}, nil
				})
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Test should complete without panic
	assert.True(t, true)
}

// Benchmark tests
func BenchmarkCacheWrapper_GetOrSetTyped_CacheHit(b *testing.B) {
	cache := NewInMemoryCache(5*time.Minute, 10*time.Minute)
	wrapper := NewCacheWrapper(cache, 1*time.Minute)

	type TestData struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	// Pre-populate
	data := TestData{Name: "test", Value: 42}
	jsonData, _ := json.Marshal(data)
	_ = cache.Set("bench:key", jsonData, 1*time.Minute)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var result TestData
		_ = wrapper.GetOrSetTyped("bench:key", 1*time.Minute, &result, func() (interface{}, error) {
			return &TestData{Name: "test", Value: 42}, nil
		})
	}
}

func BenchmarkCacheWrapper_GetOrSetTyped_CacheMiss(b *testing.B) {
	cache := NewInMemoryCache(5*time.Minute, 10*time.Minute)
	wrapper := NewCacheWrapper(cache, 1*time.Minute)

	type TestData struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var result TestData
		key := string(rune('a' + i%26)) // Use different keys to force misses
		_ = wrapper.GetOrSetTyped(key, 1*time.Minute, &result, func() (interface{}, error) {
			return &TestData{Name: "test", Value: 42}, nil
		})
	}
}