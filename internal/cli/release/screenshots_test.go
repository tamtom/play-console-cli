package release

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseScreenshotsDir(t *testing.T) {
	t.Run("parses valid screenshots directory", func(t *testing.T) {
		dir := t.TempDir()

		// en-US/phoneScreenshots
		phoneDir := filepath.Join(dir, "en-US", "phoneScreenshots")
		mustT(t, os.MkdirAll(phoneDir, 0o755))
		mustT(t, os.WriteFile(filepath.Join(phoneDir, "01.png"), []byte("img"), 0o644))
		mustT(t, os.WriteFile(filepath.Join(phoneDir, "02.png"), []byte("img"), 0o644))

		// en-US/tvScreenshots
		tvDir := filepath.Join(dir, "en-US", "tvScreenshots")
		mustT(t, os.MkdirAll(tvDir, 0o755))
		mustT(t, os.WriteFile(filepath.Join(tvDir, "tv1.jpg"), []byte("img"), 0o644))

		result, err := ParseScreenshotsDir(dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result) != 1 {
			t.Fatalf("expected 1 locale, got %d", len(result))
		}

		enUS := result["en-US"]
		if len(enUS) != 2 {
			t.Fatalf("expected 2 device types, got %d", len(enUS))
		}

		phonePaths := enUS["phoneScreenshots"]
		if len(phonePaths) != 2 {
			t.Fatalf("expected 2 phone screenshots, got %d", len(phonePaths))
		}
		// Verify sorted
		if !strings.HasSuffix(phonePaths[0], "01.png") {
			t.Errorf("expected first file to be 01.png, got %s", phonePaths[0])
		}

		tvPaths := enUS["tvScreenshots"]
		if len(tvPaths) != 1 {
			t.Fatalf("expected 1 tv screenshot, got %d", len(tvPaths))
		}
	})

	t.Run("supports multiple image extensions", func(t *testing.T) {
		dir := t.TempDir()
		phoneDir := filepath.Join(dir, "en-US", "phoneScreenshots")
		mustT(t, os.MkdirAll(phoneDir, 0o755))
		mustT(t, os.WriteFile(filepath.Join(phoneDir, "a.png"), []byte("img"), 0o644))
		mustT(t, os.WriteFile(filepath.Join(phoneDir, "b.jpg"), []byte("img"), 0o644))
		mustT(t, os.WriteFile(filepath.Join(phoneDir, "c.jpeg"), []byte("img"), 0o644))
		mustT(t, os.WriteFile(filepath.Join(phoneDir, "d.webp"), []byte("img"), 0o644))
		mustT(t, os.WriteFile(filepath.Join(phoneDir, "e.txt"), []byte("not an image"), 0o644))

		result, err := ParseScreenshotsDir(dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		paths := result["en-US"]["phoneScreenshots"]
		if len(paths) != 4 {
			t.Fatalf("expected 4 images (txt excluded), got %d", len(paths))
		}
	})

	t.Run("errors on unknown device type", func(t *testing.T) {
		dir := t.TempDir()
		badDir := filepath.Join(dir, "en-US", "invalidType")
		mustT(t, os.MkdirAll(badDir, 0o755))
		mustT(t, os.WriteFile(filepath.Join(badDir, "img.png"), []byte("img"), 0o644))

		_, err := ParseScreenshotsDir(dir)
		if err == nil {
			t.Fatal("expected error for unknown device type")
		}
		if !strings.Contains(err.Error(), "unknown screenshot type") {
			t.Errorf("expected 'unknown screenshot type' in error, got: %v", err)
		}
	})

	t.Run("errors on nonexistent directory", func(t *testing.T) {
		_, err := ParseScreenshotsDir("/nonexistent/path")
		if err == nil {
			t.Fatal("expected error for nonexistent directory")
		}
	})

	t.Run("errors on empty directory", func(t *testing.T) {
		dir := t.TempDir()
		_, err := ParseScreenshotsDir(dir)
		if err == nil {
			t.Fatal("expected error for empty directory")
		}
	})

	t.Run("skips subdirectories inside device type dirs", func(t *testing.T) {
		dir := t.TempDir()
		phoneDir := filepath.Join(dir, "en-US", "phoneScreenshots")
		mustT(t, os.MkdirAll(filepath.Join(phoneDir, "subdir"), 0o755))
		mustT(t, os.WriteFile(filepath.Join(phoneDir, "01.png"), []byte("img"), 0o644))

		result, err := ParseScreenshotsDir(dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		paths := result["en-US"]["phoneScreenshots"]
		if len(paths) != 1 {
			t.Fatalf("expected 1 screenshot (subdir excluded), got %d", len(paths))
		}
	})

	t.Run("multiple locales", func(t *testing.T) {
		dir := t.TempDir()

		enDir := filepath.Join(dir, "en-US", "phoneScreenshots")
		mustT(t, os.MkdirAll(enDir, 0o755))
		mustT(t, os.WriteFile(filepath.Join(enDir, "01.png"), []byte("img"), 0o644))

		jaDir := filepath.Join(dir, "ja", "phoneScreenshots")
		mustT(t, os.MkdirAll(jaDir, 0o755))
		mustT(t, os.WriteFile(filepath.Join(jaDir, "01.png"), []byte("img"), 0o644))

		result, err := ParseScreenshotsDir(dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result) != 2 {
			t.Fatalf("expected 2 locales, got %d", len(result))
		}
	})

	t.Run("skips device type dirs with no image files", func(t *testing.T) {
		dir := t.TempDir()
		phoneDir := filepath.Join(dir, "en-US", "phoneScreenshots")
		mustT(t, os.MkdirAll(phoneDir, 0o755))
		mustT(t, os.WriteFile(filepath.Join(phoneDir, "readme.txt"), []byte("not an image"), 0o644))

		tvDir := filepath.Join(dir, "en-US", "tvScreenshots")
		mustT(t, os.MkdirAll(tvDir, 0o755))
		mustT(t, os.WriteFile(filepath.Join(tvDir, "01.png"), []byte("img"), 0o644))

		result, err := ParseScreenshotsDir(dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		enUS := result["en-US"]
		if len(enUS) != 1 {
			t.Fatalf("expected 1 device type (phone skipped due to no images), got %d", len(enUS))
		}
		if _, ok := enUS["tvScreenshots"]; !ok {
			t.Error("expected tvScreenshots to be present")
		}
	})
}

func mustT(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
