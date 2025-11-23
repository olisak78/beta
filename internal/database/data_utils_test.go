package database

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMergeJSON_PreservesExistingAttributesAndDeepMerges(t *testing.T) {
	a := json.RawMessage(`{"name":"A","age":10,"meta":{"x":1}}`)
	b := []byte(`{"age":20,"meta":{"y":2},"new":true}`)

	merged, err := mergeJSON(a, b)
	if err != nil {
		t.Fatalf("mergeJSON returned error: %v", err)
	}

	var got interface{}
	if err := json.Unmarshal(merged, &got); err != nil {
		t.Fatalf("failed to unmarshal merged JSON: %v", err)
	}

	var want interface{}
	expected := []byte(`{"age":20,"meta":{"x":1,"y":2},"name":"A","new":true}`)
	if err := json.Unmarshal(expected, &want); err != nil {
		t.Fatalf("failed to unmarshal expected JSON: %v", err)
	}

	if !jsonDeepEqual(got, want) {
		tg, _ := json.MarshalIndent(got, "", "  ")
		tw, _ := json.MarshalIndent(want, "", "  ")
		t.Fatalf("merged JSON mismatch\nGot:\n%s\nWant:\n%s", tg, tw)
	}
}

func TestMergeJSON_InvalidInputs(t *testing.T) {
	t.Run("invalid a json", func(t *testing.T) {
		a := json.RawMessage(`{"x":1`)
		b := []byte(`{"y":2}`)
		if _, err := mergeJSON(a, b); err == nil {
			t.Fatal("expected error for invalid a")
		}
	})

	t.Run("invalid b json", func(t *testing.T) {
		a := json.RawMessage(`{"x":1}`)
		b := []byte(`{"y":2`)
		if _, err := mergeJSON(a, b); err == nil {
			t.Fatal("expected error for invalid b")
		}
	})

	t.Run("non-object a (array)", func(t *testing.T) {
		a := json.RawMessage(`[1,2,3]`)
		b := []byte(`{"y":2}`)
		if _, err := mergeJSON(a, b); err == nil {
			t.Fatal("expected error for non-object a")
		}
	})

	t.Run("non-object b (array)", func(t *testing.T) {
		a := json.RawMessage(`{"x":1}`)
		b := []byte(`[1,2]`)
		if _, err := mergeJSON(a, b); err == nil {
			t.Fatal("expected error for non-object b")
		}
	})
}

func TestJSONEqual_Cases(t *testing.T) {
	t.Run("equal ignoring key order", func(t *testing.T) {
		a := json.RawMessage(`{"a":1,"b":{"x":1,"y":2}}`)
		b := []byte(`{"b":{"y":2,"x":1},"a":1}`)
		if !jsonEqual(a, b) {
			t.Fatal("expected jsonEqual to be true for equivalent objects")
		}
	})

	t.Run("unequal values", func(t *testing.T) {
		a := json.RawMessage(`{"a":1,"b":{"x":1,"y":2}}`)
		b := []byte(`{"b":{"y":2,"x":2},"a":1}`)
		if jsonEqual(a, b) {
			t.Fatal("expected jsonEqual to be false for different values")
		}
	})

	t.Run("both empty", func(t *testing.T) {
		a := json.RawMessage(nil)
		b := []byte(nil)
		if !jsonEqual(a, b) {
			t.Fatal("expected jsonEqual to be true for both empty")
		}
	})

	t.Run("empty a vs non-empty b", func(t *testing.T) {
		a := json.RawMessage(nil)
		b := []byte(`{"x":1}`)
		if jsonEqual(a, b) {
			t.Fatal("expected jsonEqual to be false (a empty, b non-empty)")
		}
	})

	t.Run("invalid a json", func(t *testing.T) {
		a := json.RawMessage(`{"x":1`)
		b := []byte(`{"x":1}`)
		if jsonEqual(a, b) {
			t.Fatal("expected jsonEqual to be false when a is invalid")
		}
	})

	t.Run("invalid b json", func(t *testing.T) {
		a := json.RawMessage(`{"x":1}`)
		b := []byte(`{"x":1`)
		if jsonEqual(a, b) {
			t.Fatal("expected jsonEqual to be false when b is invalid")
		}
	})
}

func TestSlugifyTitle(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"Hello, World!", "hello-world"},
		{"API   Gateway--Service", "api-gateway-service"},
		{"***", "link"},
		{"with_underscores", "with-underscores"},
		{"Trailing--", "trailing"},
		{"UPPER and lower 123", "upper-and-lower-123"},
		{"", "link"},
	}

	for _, c := range cases {
		if got := slugifyTitle(c.in); got != c.want {
			t.Fatalf("slugifyTitle(%q) = %q; want %q", c.in, got, c.want)
		}
	}
}

func TestNormalizeTagsCSV(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		if got := normalizeTagsCSV(nil); got != "" {
			t.Fatalf("normalizeTagsCSV(nil) = %q; want empty", got)
		}
	})

	t.Run("string with spaces and blanks", func(t *testing.T) {
		if got := normalizeTagsCSV(" a, b , , c "); got != "a,b,c" {
			t.Fatalf("normalizeTagsCSV(string) = %q; want %q", got, "a,b,c")
		}
	})

	t.Run("slice of interfaces with non-strings", func(t *testing.T) {
		in := []interface{}{" a ", "b", 3, "", "c "}
		if got := normalizeTagsCSV(in); got != "a,b,c" {
			t.Fatalf("normalizeTagsCSV(slice) = %q; want %q", got, "a,b,c")
		}
	})

	t.Run("unknown type", func(t *testing.T) {
		if got := normalizeTagsCSV(123); got != "" {
			t.Fatalf("normalizeTagsCSV(unknown) = %q; want empty", got)
		}
	})

	t.Run("single tag string", func(t *testing.T) {
		if got := normalizeTagsCSV("single"); got != "single" {
			t.Fatalf("normalizeTagsCSV(single) = %q; want %q", got, "single")
		}
	})
}

func TestLoadFromYAMLFile_Cases(t *testing.T) {
	decoder := func(data []byte) ([]string, error) {
		s := strings.TrimSpace(string(data))
		if s == "ERR" {
			return nil, errors.New("decode error")
		}
		if s == "" {
			return []string{}, nil
		}
		parts := strings.Split(s, "\n")
		return parts, nil
	}

	t.Run("file not found returns empty slice", func(t *testing.T) {
		dir := t.TempDir()
		items, err := loadFromYAMLFile[string](dir, "missing.yaml", decoder)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(items) != 0 {
			t.Fatalf("expected empty slice, got %v", items)
		}
	})

	t.Run("empty file returns empty slice", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "empty.yaml")
		if err := os.WriteFile(path, []byte(""), 0o644); err != nil {
			t.Fatalf("write empty file: %v", err)
		}
		items, err := loadFromYAMLFile[string](dir, "empty.yaml", decoder)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(items) != 0 {
			t.Fatalf("expected empty slice, got %v", items)
		}
	})

	t.Run("decoder error is propagated", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "err.yaml")
		if err := os.WriteFile(path, []byte("ERR"), 0o644); err != nil {
			t.Fatalf("write file: %v", err)
		}
		if _, err := loadFromYAMLFile[string](dir, "err.yaml", decoder); err == nil {
			t.Fatal("expected error from decoder")
		}
	})

	t.Run("successful decode", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "ok.yaml")
		if err := os.WriteFile(path, []byte("a\nb\n"), 0o644); err != nil {
			t.Fatalf("write file: %v", err)
		}
		items, err := loadFromYAMLFile[string](dir, "ok.yaml", decoder)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := []string{"a", "b"}
		if len(items) != len(want) || items[0] != "a" || items[1] != "b" {
			t.Fatalf("unexpected items: got %v want %v", items, want)
		}
	})
}

// jsonDeepEqual is a helper used only in tests for deep comparing decoded JSON
func jsonDeepEqual(a, b interface{}) bool {
	ab, _ := json.Marshal(a)
	bb, _ := json.Marshal(b)
	var av interface{}
	var bv interface{}
	_ = json.Unmarshal(ab, &av)
	_ = json.Unmarshal(bb, &bv)
	return deepEqual(av, bv)
}

func deepEqual(a, b interface{}) bool {
	switch at := a.(type) {
	case map[string]interface{}:
		bt, ok := b.(map[string]interface{})
		if !ok || len(at) != len(bt) {
			return false
		}
		for k, v := range at {
			if !deepEqual(v, bt[k]) {
				return false
			}
		}
		return true
	case []interface{}:
		bt, ok := b.([]interface{})
		if !ok || len(at) != len(bt) {
			return false
		}
		for i := range at {
			if !deepEqual(at[i], bt[i]) {
				return false
			}
		}
		return true
	default:
		return a == b
	}
}
