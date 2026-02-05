package shared

import (
	"encoding/json"
	"fmt"
	"os"
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
