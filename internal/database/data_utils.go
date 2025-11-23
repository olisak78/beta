package database

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
)

// loadFromYAMLFile is a generic YAML loader for a single filename in a directory.
// It returns an empty slice if the file is not found or empty.
func loadFromYAMLFile[T any](dataDir, filename string, decode func([]byte) ([]T, error)) ([]T, error) {
	path := filepath.Join(dataDir, filename)

	data, err := os.ReadFile(path)
	if err != nil {
		// If the file is missing, return empty slice (no-op)
		if os.IsNotExist(err) {
			return []T{}, nil
		}
		return nil, err
	}

	if len(data) == 0 {
		return []T{}, nil
	}

	items, err := decode(data)
	if err != nil {
		return nil, err
	}
	return items, nil
}

// jsonEqual compares two JSON payloads semantically (ignoring key order)
func jsonEqual(a json.RawMessage, b []byte) bool {
	var ja interface{}
	var jb interface{}
	// Normalize nil/empty cases
	if len(a) == 0 && len(b) == 0 {
		return true
	}
	if err := json.Unmarshal(a, &ja); err != nil {
		// If existing DB value is empty or invalid, treat as unequal unless b is also empty
		if len(a) == 0 && len(b) == 0 {
			return true
		}
		return false
	}
	if err := json.Unmarshal(b, &jb); err != nil {
		// If YAML metadata failed to unmarshal, treat as unequal
		return false
	}
	return reflect.DeepEqual(ja, jb)
}

/*
mergeJSON merges two JSON object payloads from raw bytes.

- a: existing JSON (from DB), provided as json.RawMessage
- b: overriding JSON (from YAML or input), provided as []byte

Behavior:
- If both are JSON objects, fields are merged recursively with b taking precedence.
- If either payload is invalid JSON or not an object, returns an error.
- Empty payloads are treated as empty objects.

Returns the merged JSON as []byte or an error.
*/
func mergeJSON(a json.RawMessage, b []byte) ([]byte, error) {
	var ma map[string]interface{}
	var mb map[string]interface{}

	// Unmarshal 'a'
	if len(a) == 0 {
		ma = map[string]interface{}{}
	} else {
		if err := json.Unmarshal(a, &ma); err != nil {
			return nil, fmt.Errorf("mergeJSON: invalid JSON 'a': %w", err)
		}
		if ma == nil {
			ma = map[string]interface{}{}
		}
	}

	// Unmarshal 'b'
	if len(b) == 0 {
		mb = map[string]interface{}{}
	} else {
		if err := json.Unmarshal(b, &mb); err != nil {
			return nil, fmt.Errorf("mergeJSON: invalid JSON 'b': %w", err)
		}
		if mb == nil {
			mb = map[string]interface{}{}
		}
	}

	merged := mergeMaps(ma, mb)

	out, err := json.Marshal(merged)
	if err != nil {
		return nil, fmt.Errorf("mergeJSON: marshal error: %w", err)
	}
	return out, nil
}

// mergeMaps performs a deep merge of two map[string]interface{} with b overriding a
func mergeMaps(a, b map[string]interface{}) map[string]interface{} {
	if a == nil {
		a = map[string]interface{}{}
	}
	for k, v := range b {
		if bv, ok := v.(map[string]interface{}); ok {
			if av, ok2 := a[k].(map[string]interface{}); ok2 {
				a[k] = mergeMaps(av, bv)
			} else {
				a[k] = mergeMaps(map[string]interface{}{}, bv)
			}
		} else {
			a[k] = v
		}
	}
	return a
}

// slugifyTitle creates a URL-friendly name from a title
func slugifyTitle(s string) string {
	s = strings.ToLower(s)
	var b strings.Builder
	prevHyphen := false
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			prevHyphen = false
		} else {
			if !prevHyphen {
				b.WriteByte('-')
				prevHyphen = true
			}
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		out = "link"
	}
	return out
}

// normalizeTagsCSV converts various YAML tag formats into a canonical CSV string
func normalizeTagsCSV(raw interface{}) string {
	if raw == nil {
		return ""
	}
	switch v := raw.(type) {
	case string:
		parts := strings.Split(v, ",")
		out := make([]string, 0, len(parts))
		for _, p := range parts {
			t := strings.TrimSpace(p)
			if t != "" {
				out = append(out, t)
			}
		}
		return strings.Join(out, ",")
	case []interface{}:
		out := make([]string, 0, len(v))
		for _, it := range v {
			if s, ok := it.(string); ok {
				s = strings.TrimSpace(s)
				if s != "" {
					out = append(out, s)
				}
			}
		}
		return strings.Join(out, ",")
	default:
		return ""
	}
}
