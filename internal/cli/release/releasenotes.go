package release

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// ReleaseNote holds a localized release note entry.
type ReleaseNote struct {
	Language string `json:"language"`
	Text     string `json:"text"`
}

// ParseReleaseNotes parses release notes from either:
//   - A plain text string (assigned to en-US locale)
//   - A JSON array literal: [{"language":"en-US","text":"..."}]
//   - A @file path pointing to a JSON file with the same array format
//
// Returns a slice of ReleaseNote entries.
func ParseReleaseNotes(input string) ([]ReleaseNote, error) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return nil, fmt.Errorf("release notes input is empty")
	}

	// @file reference
	if strings.HasPrefix(trimmed, "@") {
		path := strings.TrimSpace(strings.TrimPrefix(trimmed, "@"))
		if path == "" {
			return nil, fmt.Errorf("invalid @file path for release notes")
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to read release notes file: %w", err)
		}
		return parseReleaseNotesJSON(data)
	}

	// Try JSON array
	if strings.HasPrefix(trimmed, "[") {
		return parseReleaseNotesJSON([]byte(trimmed))
	}

	// Plain text -- default to en-US
	return []ReleaseNote{
		{Language: "en-US", Text: trimmed},
	}, nil
}

func parseReleaseNotesJSON(data []byte) ([]ReleaseNote, error) {
	var notes []ReleaseNote
	if err := json.Unmarshal(data, &notes); err != nil {
		return nil, fmt.Errorf("invalid release notes JSON: %w", err)
	}

	for i, n := range notes {
		if strings.TrimSpace(n.Language) == "" {
			return nil, fmt.Errorf("release note at index %d is missing language", i)
		}
		if strings.TrimSpace(n.Text) == "" {
			return nil, fmt.Errorf("release note at index %d is missing text", i)
		}
	}

	if len(notes) == 0 {
		return nil, fmt.Errorf("release notes JSON array is empty")
	}

	return notes, nil
}
