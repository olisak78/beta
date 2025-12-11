package cache

import (
	"testing"

	"github.com/stretchr/testify/assert"
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

func TestCacheKeyBuilder_EmptyParts(t *testing.T) {
	builder := NewCacheKeyBuilder("test")
	key := builder.Build()

	assert.Equal(t, "test:", key)
}

func TestCacheKeyBuilder_SinglePart(t *testing.T) {
	builder := NewCacheKeyBuilder("api")
	key := builder.Add("users").Build()

	assert.Equal(t, "api:users", key)
}

func TestCacheKeyBuilder_MultipleParts(t *testing.T) {
	builder := NewCacheKeyBuilder("api")
	key := builder.Add("users").Add("123").Add("profile").Build()

	assert.Equal(t, "api:users:123:profile", key)
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

func TestCacheKeyBuilder_WithParams_OrderIndependent(t *testing.T) {
	// Test that params are included regardless of order
	builder1 := NewCacheKeyBuilder("api")
	params1 := map[string]string{
		"a": "1",
		"b": "2",
	}
	key1 := builder1.Add("endpoint").AddParams(params1).Build()

	// Should contain both params
	assert.Contains(t, key1, "a=1")
	assert.Contains(t, key1, "b=2")
	assert.Contains(t, key1, "api:endpoint:")
}

func TestCacheKeyBuilder_EmptyParams(t *testing.T) {
	builder := NewCacheKeyBuilder("api")
	params := map[string]string{}
	key := builder.Add("prs").AddParams(params).Build()

	assert.Equal(t, "api:prs", key)
}

func TestCacheKeyBuilder_NilParams(t *testing.T) {
	builder := NewCacheKeyBuilder("api")
	key := builder.Add("prs").AddParams(nil).Build()

	assert.Equal(t, "api:prs", key)
}

func TestCacheKeyBuilder_Hash(t *testing.T) {
	builder := NewCacheKeyBuilder("test")
	key := builder.Add("very").Add("long").Add("key").Add("with").Add("many").Add("parts").Hash()

	assert.Contains(t, key, "test:hash:")
	assert.Len(t, key, len("test:hash:")+64) // SHA256 produces 64 hex chars
}

func TestCacheKeyBuilder_Hash_Consistency(t *testing.T) {
	// Same input should produce same hash
	builder1 := NewCacheKeyBuilder("test")
	hash1 := builder1.Add("user").Add("123").Hash()

	builder2 := NewCacheKeyBuilder("test")
	hash2 := builder2.Add("user").Add("123").Hash()

	assert.Equal(t, hash1, hash2)
}

func TestCacheKeyBuilder_Hash_Uniqueness(t *testing.T) {
	// Different inputs should produce different hashes
	builder1 := NewCacheKeyBuilder("test")
	hash1 := builder1.Add("user").Add("123").Hash()

	builder2 := NewCacheKeyBuilder("test")
	hash2 := builder2.Add("user").Add("456").Hash()

	assert.NotEqual(t, hash1, hash2)
}

func TestCacheKeyBuilder_Chaining(t *testing.T) {
	// Test that method chaining works properly
	key := NewCacheKeyBuilder("api").
		Add("users").
		Add("123").
		Add("posts").
		Build()

	assert.Equal(t, "api:users:123:posts", key)
}

func TestSonarCacheKey(t *testing.T) {
	key := SonarCacheKey("project123", []string{"coverage", "bugs", "vulnerabilities"})
	expected := "sonar:measures:project123:coverage,bugs,vulnerabilities"

	assert.Equal(t, expected, key)
}

func TestSonarCacheKey_SingleMetric(t *testing.T) {
	key := SonarCacheKey("project456", []string{"coverage"})
	expected := "sonar:measures:project456:coverage"

	assert.Equal(t, expected, key)
}

func TestSonarCacheKey_EmptyMetrics(t *testing.T) {
	key := SonarCacheKey("project789", []string{})
	expected := "sonar:measures:project789:"

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

func TestJiraCacheKey_NoParams(t *testing.T) {
	key := JiraCacheKey("issues", "user123", nil)
	expected := "jira:issues:user123"

	assert.Equal(t, expected, key)
}

func TestJiraCacheKey_EmptyParams(t *testing.T) {
	key := JiraCacheKey("issues", "user456", map[string]string{})
	expected := "jira:issues:user456"

	assert.Equal(t, expected, key)
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

func TestGitHubCacheKey_NoParams(t *testing.T) {
	key := GitHubCacheKey("repos", "user789", nil)
	expected := "github:repos:user789"

	assert.Equal(t, expected, key)
}

func TestGitHubCacheKey_ComplexParams(t *testing.T) {
	params := map[string]string{
		"state":     "all",
		"sort":      "updated",
		"direction": "desc",
		"per_page":  "100",
	}
	key := GitHubCacheKey("pull-requests", "user999", params)

	assert.Contains(t, key, "github:pull-requests:user999:")
	assert.Contains(t, key, "state=all")
	assert.Contains(t, key, "sort=updated")
	assert.Contains(t, key, "direction=desc")
	assert.Contains(t, key, "per_page=100")
}

// Benchmark tests
func BenchmarkCacheKeyBuilder_Simple(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NewCacheKeyBuilder("test").
			Add("user").
			Add("123").
			Build()
	}
}

func BenchmarkCacheKeyBuilder_WithParams(b *testing.B) {
	params := map[string]string{
		"status": "open",
		"limit":  "10",
		"sort":   "created",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		NewCacheKeyBuilder("api").
			Add("prs").
			AddParams(params).
			Build()
	}
}

func BenchmarkCacheKeyBuilder_Hash(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NewCacheKeyBuilder("test").
			Add("very").
			Add("long").
			Add("key").
			Add("with").
			Add("many").
			Add("parts").
			Hash()
	}
}

func BenchmarkSonarCacheKey(b *testing.B) {
	metrics := []string{"coverage", "bugs", "vulnerabilities"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		SonarCacheKey("project123", metrics)
	}
}

func BenchmarkJiraCacheKey(b *testing.B) {
	params := map[string]string{
		"status": "open",
		"limit":  "50",
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		JiraCacheKey("issues", "user123", params)
	}
}

func BenchmarkGitHubCacheKey(b *testing.B) {
	params := map[string]string{
		"state": "open",
		"sort":  "created",
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GitHubCacheKey("pull-requests", "user456", params)
	}
}