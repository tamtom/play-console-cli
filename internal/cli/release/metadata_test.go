package release

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseListingsDir(t *testing.T) {
	t.Run("parses valid listings directory", func(t *testing.T) {
		dir := t.TempDir()

		// Create en-US locale
		enDir := filepath.Join(dir, "en-US")
		must(t, os.MkdirAll(enDir, 0o755))
		must(t, os.WriteFile(filepath.Join(enDir, "title.txt"), []byte("My App"), 0o644))
		must(t, os.WriteFile(filepath.Join(enDir, "short_description.txt"), []byte("A great app"), 0o644))
		must(t, os.WriteFile(filepath.Join(enDir, "full_description.txt"), []byte("Full description here"), 0o644))
		must(t, os.WriteFile(filepath.Join(enDir, "video.txt"), []byte("https://youtube.com/watch?v=abc"), 0o644))

		// Create ja locale with partial data
		jaDir := filepath.Join(dir, "ja")
		must(t, os.MkdirAll(jaDir, 0o755))
		must(t, os.WriteFile(filepath.Join(jaDir, "title.txt"), []byte("私のアプリ"), 0o644))

		result, err := ParseListingsDir(dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result) != 2 {
			t.Fatalf("expected 2 locales, got %d", len(result))
		}

		en := result["en-US"]
		if en.Title != "My App" {
			t.Errorf("expected title 'My App', got %q", en.Title)
		}
		if en.ShortDescription != "A great app" {
			t.Errorf("expected short description 'A great app', got %q", en.ShortDescription)
		}
		if en.FullDescription != "Full description here" {
			t.Errorf("expected full description, got %q", en.FullDescription)
		}
		if en.Video != "https://youtube.com/watch?v=abc" {
			t.Errorf("expected video URL, got %q", en.Video)
		}

		ja := result["ja"]
		if ja.Title != "私のアプリ" {
			t.Errorf("expected Japanese title, got %q", ja.Title)
		}
		if ja.ShortDescription != "" {
			t.Errorf("expected empty short description, got %q", ja.ShortDescription)
		}
	})

	t.Run("trims whitespace from file contents", func(t *testing.T) {
		dir := t.TempDir()
		enDir := filepath.Join(dir, "en-US")
		must(t, os.MkdirAll(enDir, 0o755))
		must(t, os.WriteFile(filepath.Join(enDir, "title.txt"), []byte("  My App  \n"), 0o644))

		result, err := ParseListingsDir(dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result["en-US"].Title != "My App" {
			t.Errorf("expected trimmed title, got %q", result["en-US"].Title)
		}
	})

	t.Run("errors on nonexistent directory", func(t *testing.T) {
		_, err := ParseListingsDir("/nonexistent/path")
		if err == nil {
			t.Fatal("expected error for nonexistent directory")
		}
	})

	t.Run("errors on file instead of directory", func(t *testing.T) {
		f := filepath.Join(t.TempDir(), "file.txt")
		must(t, os.WriteFile(f, []byte("not a dir"), 0o644))

		_, err := ParseListingsDir(f)
		if err == nil {
			t.Fatal("expected error for file path")
		}
	})

	t.Run("errors on empty directory", func(t *testing.T) {
		dir := t.TempDir()
		_, err := ParseListingsDir(dir)
		if err == nil {
			t.Fatal("expected error for empty directory")
		}
	})

	t.Run("skips non-directory entries in root", func(t *testing.T) {
		dir := t.TempDir()
		// Create a regular file at root level (should be skipped)
		must(t, os.WriteFile(filepath.Join(dir, "README.md"), []byte("ignore me"), 0o644))
		// Create a valid locale
		enDir := filepath.Join(dir, "en-US")
		must(t, os.MkdirAll(enDir, 0o755))
		must(t, os.WriteFile(filepath.Join(enDir, "title.txt"), []byte("App"), 0o644))

		result, err := ParseListingsDir(dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 1 {
			t.Fatalf("expected 1 locale, got %d", len(result))
		}
	})

	t.Run("skips locale dirs with all empty files", func(t *testing.T) {
		dir := t.TempDir()
		enDir := filepath.Join(dir, "en-US")
		must(t, os.MkdirAll(enDir, 0o755))
		must(t, os.WriteFile(filepath.Join(enDir, "title.txt"), []byte("App"), 0o644))

		// Create locale with only whitespace content
		emptyDir := filepath.Join(dir, "fr")
		must(t, os.MkdirAll(emptyDir, 0o755))
		must(t, os.WriteFile(filepath.Join(emptyDir, "title.txt"), []byte("   \n  "), 0o644))

		result, err := ParseListingsDir(dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 1 {
			t.Fatalf("expected 1 locale (fr should be skipped), got %d", len(result))
		}
	})
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
