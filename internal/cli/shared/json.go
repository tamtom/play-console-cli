package shared

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
)

// LoadJSONArg parses JSON from a literal string or @file path.
func LoadJSONArg(value string, out interface{}) error {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fmt.Errorf("empty json value")
	}
	if strings.HasPrefix(trimmed, "@") {
		path := strings.TrimSpace(strings.TrimPrefix(trimmed, "@"))
		if path == "" {
			return fmt.Errorf("invalid @file path")
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return json.Unmarshal(data, out)
	}
	return json.Unmarshal([]byte(trimmed), out)
}

// LoadJSONArgRaw returns the raw JSON bytes from a literal string or @file path
// without unmarshaling. Use this when you need to inspect the JSON keys before
// parsing into a typed struct.
func LoadJSONArgRaw(value string) ([]byte, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil, fmt.Errorf("empty json value")
	}
	if strings.HasPrefix(trimmed, "@") {
		path := strings.TrimSpace(strings.TrimPrefix(trimmed, "@"))
		if path == "" {
			return nil, fmt.Errorf("invalid @file path")
		}
		return os.ReadFile(path)
	}
	return []byte(trimmed), nil
}

// DeriveUpdateMask extracts top-level keys from raw JSON and returns a sorted,
// comma-separated update mask containing only keys that appear in mutableFields.
//
// We use an allowlist rather than a blocklist because the CLI unmarshals JSON
// into typed SDK structs before sending. The SDK drops unknown fields during
// unmarshal, so including an unknown key in the mask would create a mismatch
// (mask names a field the body doesn't contain) and cause a guaranteed 400.
func DeriveUpdateMask(raw []byte, mutableFields []string) (string, error) {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err != nil {
		return "", fmt.Errorf("invalid JSON object: %w", err)
	}

	allowed := make(map[string]bool, len(mutableFields))
	for _, f := range mutableFields {
		allowed[f] = true
	}

	var mask []string
	for key := range obj {
		if allowed[key] {
			mask = append(mask, key)
		}
	}

	if len(mask) == 0 {
		return "", fmt.Errorf("no updatable fields found in JSON; valid fields: %s", strings.Join(mutableFields, ", "))
	}

	sort.Strings(mask)
	return strings.Join(mask, ","), nil
}
