package release

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseReleaseNotes(t *testing.T) {
	t.Run("plain text defaults to en-US", func(t *testing.T) {
		notes, err := ParseReleaseNotes("Bug fixes and improvements")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(notes) != 1 {
			t.Fatalf("expected 1 note, got %d", len(notes))
		}
		if notes[0].Language != "en-US" {
			t.Errorf("expected language en-US, got %q", notes[0].Language)
		}
		if notes[0].Text != "Bug fixes and improvements" {
			t.Errorf("expected text, got %q", notes[0].Text)
		}
	})

	t.Run("JSON array inline", func(t *testing.T) {
		input := `[{"language":"en-US","text":"English notes"},{"language":"ja","text":"日本語のノート"}]`
		notes, err := ParseReleaseNotes(input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(notes) != 2 {
			t.Fatalf("expected 2 notes, got %d", len(notes))
		}
		if notes[0].Language != "en-US" || notes[0].Text != "English notes" {
			t.Errorf("unexpected first note: %+v", notes[0])
		}
		if notes[1].Language != "ja" || notes[1].Text != "日本語のノート" {
			t.Errorf("unexpected second note: %+v", notes[1])
		}
	})

	t.Run("JSON from @file", func(t *testing.T) {
		dir := t.TempDir()
		f := filepath.Join(dir, "notes.json")
		content := `[{"language":"de","text":"Deutsche Anmerkungen"}]`
		if err := os.WriteFile(f, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}

		notes, err := ParseReleaseNotes("@" + f)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(notes) != 1 {
			t.Fatalf("expected 1 note, got %d", len(notes))
		}
		if notes[0].Language != "de" {
			t.Errorf("expected language de, got %q", notes[0].Language)
		}
	})

	t.Run("errors on empty input", func(t *testing.T) {
		_, err := ParseReleaseNotes("")
		if err == nil {
			t.Fatal("expected error for empty input")
		}
	})

	t.Run("errors on whitespace-only input", func(t *testing.T) {
		_, err := ParseReleaseNotes("   \n  ")
		if err == nil {
			t.Fatal("expected error for whitespace-only input")
		}
	})

	t.Run("errors on invalid JSON", func(t *testing.T) {
		_, err := ParseReleaseNotes("[invalid json")
		if err == nil {
			t.Fatal("expected error for invalid JSON")
		}
	})

	t.Run("errors on empty JSON array", func(t *testing.T) {
		_, err := ParseReleaseNotes("[]")
		if err == nil {
			t.Fatal("expected error for empty array")
		}
	})

	t.Run("errors on missing language field", func(t *testing.T) {
		_, err := ParseReleaseNotes(`[{"language":"","text":"some text"}]`)
		if err == nil {
			t.Fatal("expected error for missing language")
		}
		if !strings.Contains(err.Error(), "missing language") {
			t.Errorf("expected 'missing language' in error, got: %v", err)
		}
	})

	t.Run("errors on missing text field", func(t *testing.T) {
		_, err := ParseReleaseNotes(`[{"language":"en-US","text":""}]`)
		if err == nil {
			t.Fatal("expected error for missing text")
		}
		if !strings.Contains(err.Error(), "missing text") {
			t.Errorf("expected 'missing text' in error, got: %v", err)
		}
	})

	t.Run("errors on invalid @file path", func(t *testing.T) {
		_, err := ParseReleaseNotes("@")
		if err == nil {
			t.Fatal("expected error for empty @file path")
		}
	})

	t.Run("errors on nonexistent @file", func(t *testing.T) {
		_, err := ParseReleaseNotes("@/nonexistent/file.json")
		if err == nil {
			t.Fatal("expected error for nonexistent file")
		}
	})

	t.Run("trims whitespace around plain text", func(t *testing.T) {
		notes, err := ParseReleaseNotes("  Bug fixes  ")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if notes[0].Text != "Bug fixes" {
			t.Errorf("expected trimmed text, got %q", notes[0].Text)
		}
	})
}
