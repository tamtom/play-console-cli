package migrate

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// setupFastlaneFixture creates a realistic Fastlane metadata/android/ tree
// inside a temp directory and returns the path to the root locale directory.
func setupFastlaneFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()

	// en-US locale with all file types
	enUS := filepath.Join(root, "en-US")
	mkFile(t, filepath.Join(enUS, "title.txt"), "My App")
	mkFile(t, filepath.Join(enUS, "short_description.txt"), "A short description")
	mkFile(t, filepath.Join(enUS, "full_description.txt"), "A full description of the app.")
	mkFile(t, filepath.Join(enUS, "video.txt"), "https://youtu.be/abc123")
	mkFile(t, filepath.Join(enUS, "changelogs", "100.txt"), "Bug fixes")
	mkFile(t, filepath.Join(enUS, "changelogs", "200.txt"), "New feature")
	mkFile(t, filepath.Join(enUS, "images", "phoneScreenshots", "1.png"), "png-data")
	mkFile(t, filepath.Join(enUS, "images", "phoneScreenshots", "2.png"), "png-data")
	mkFile(t, filepath.Join(enUS, "images", "featureGraphic.png"), "feature-data")

	// de-DE locale with partial files
	deDE := filepath.Join(root, "de-DE")
	mkFile(t, filepath.Join(deDE, "title.txt"), "Meine App")
	mkFile(t, filepath.Join(deDE, "short_description.txt"), "Eine kurze Beschreibung")

	return root
}

// mkFile creates a file with the given content, creating parent dirs.
func mkFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

// ─── CLI-level tests ───────────────────────────────────────────────────────

func TestFastlaneCommand_SourceRequired(t *testing.T) {
	cmd := FastlaneCommand()
	err := cmd.ParseAndRun(context.Background(), []string{})
	if err == nil {
		t.Fatal("expected error when --source is missing")
	}
	if got := err.Error(); got != "--source is required" {
		t.Fatalf("unexpected error: %s", got)
	}
}

func TestFastlaneCommand_InvalidOutputFlags(t *testing.T) {
	cmd := FastlaneCommand()
	err := cmd.ParseAndRun(context.Background(), []string{
		"--source", "/tmp/does-not-matter",
		"--output", "table",
		"--pretty",
	})
	if err == nil {
		t.Fatal("expected error for --pretty with table output")
	}
}

func TestFastlaneCommand_SourceNotFound(t *testing.T) {
	cmd := FastlaneCommand()
	err := cmd.ParseAndRun(context.Background(), []string{
		"--source", "/tmp/nonexistent-dir-xyz",
	})
	if err == nil {
		t.Fatal("expected error for missing source directory")
	}
}

func TestFastlaneCommand_DryRunProducesJSON(t *testing.T) {
	src := setupFastlaneFixture(t)

	// Capture stdout.
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cmd := FastlaneCommand()
	err := cmd.ParseAndRun(context.Background(), []string{
		"--source", src,
		"--dry-run",
	})

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf [8192]byte
	n, _ := r.Read(buf[:])
	output := string(buf[:n])

	var summary FastlaneSummary
	if err := json.Unmarshal([]byte(output), &summary); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, output)
	}

	if !summary.DryRun {
		t.Error("expected dryRun=true")
	}
	if len(summary.Locales) != 2 {
		t.Fatalf("expected 2 locales, got %d", len(summary.Locales))
	}
}

func TestFastlaneCommand_LocaleFilter(t *testing.T) {
	src := setupFastlaneFixture(t)

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cmd := FastlaneCommand()
	err := cmd.ParseAndRun(context.Background(), []string{
		"--source", src,
		"--dry-run",
		"--locales", "de-DE",
	})

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf [8192]byte
	n, _ := r.Read(buf[:])

	var summary FastlaneSummary
	if err := json.Unmarshal(buf[:n], &summary); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}

	if len(summary.Locales) != 1 {
		t.Fatalf("expected 1 locale, got %d", len(summary.Locales))
	}
	if summary.Locales[0].Locale != "de-DE" {
		t.Errorf("expected de-DE, got %s", summary.Locales[0].Locale)
	}
}

// ─── Unit tests for core migration logic ───────────────────────────────────

func TestRunFastlaneMigration_DryRun(t *testing.T) {
	src := setupFastlaneFixture(t)

	summary, err := runFastlaneMigration(src, "/unused", true, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !summary.DryRun {
		t.Error("expected dryRun=true")
	}
	if summary.OutputDir != "" {
		t.Errorf("expected empty outputDir in dry-run, got %q", summary.OutputDir)
	}
	if len(summary.Locales) != 2 {
		t.Fatalf("expected 2 locales, got %d", len(summary.Locales))
	}

	// en-US assertions
	en := findLocale(t, summary, "en-US")
	assertContains(t, en.TextFiles, "title.txt")
	assertContains(t, en.TextFiles, "short_description.txt")
	assertContains(t, en.TextFiles, "full_description.txt")
	assertContains(t, en.TextFiles, "video.txt")
	if len(en.Changelogs) != 2 {
		t.Errorf("expected 2 changelogs, got %d", len(en.Changelogs))
	}
	if len(en.Images) != 3 {
		t.Errorf("expected 3 images, got %d: %v", len(en.Images), en.Images)
	}

	// de-DE assertions
	de := findLocale(t, summary, "de-DE")
	if len(de.TextFiles) != 2 {
		t.Errorf("expected 2 text files, got %d", len(de.TextFiles))
	}
	if len(de.Changelogs) != 0 {
		t.Errorf("expected 0 changelogs, got %d", len(de.Changelogs))
	}

	// Totals
	if summary.TotalFiles != 8 { // 4 text + 2 changelog (en) + 2 text (de)
		t.Errorf("expected 8 total files, got %d", summary.TotalFiles)
	}
	if summary.TotalImages != 3 { // 2 phone + 1 feature
		t.Errorf("expected 3 total images, got %d", summary.TotalImages)
	}
}

func TestRunFastlaneMigration_CopiesFiles(t *testing.T) {
	src := setupFastlaneFixture(t)
	dst := filepath.Join(t.TempDir(), "output")

	summary, err := runFastlaneMigration(src, dst, false, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if summary.DryRun {
		t.Error("expected dryRun=false")
	}
	if summary.OutputDir != dst {
		t.Errorf("expected outputDir=%q, got %q", dst, summary.OutputDir)
	}

	// Verify files were actually written.
	assertFileContent(t, filepath.Join(dst, "en-US", "title.txt"), "My App")
	assertFileContent(t, filepath.Join(dst, "en-US", "short_description.txt"), "A short description")
	assertFileContent(t, filepath.Join(dst, "en-US", "changelogs", "100.txt"), "Bug fixes")
	assertFileContent(t, filepath.Join(dst, "en-US", "changelogs", "200.txt"), "New feature")
	assertFileExists(t, filepath.Join(dst, "en-US", "images", "phoneScreenshots", "1.png"))
	assertFileExists(t, filepath.Join(dst, "en-US", "images", "phoneScreenshots", "2.png"))
	assertFileExists(t, filepath.Join(dst, "en-US", "images", "featureGraphic.png"))
	assertFileContent(t, filepath.Join(dst, "de-DE", "title.txt"), "Meine App")
}

func TestRunFastlaneMigration_LocaleFilter(t *testing.T) {
	src := setupFastlaneFixture(t)

	filter := map[string]bool{"en-US": true}
	summary, err := runFastlaneMigration(src, "/unused", true, filter)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(summary.Locales) != 1 {
		t.Fatalf("expected 1 locale, got %d", len(summary.Locales))
	}
	if summary.Locales[0].Locale != "en-US" {
		t.Errorf("expected en-US, got %s", summary.Locales[0].Locale)
	}
}

func TestRunFastlaneMigration_EmptySource(t *testing.T) {
	src := t.TempDir() // empty directory

	summary, err := runFastlaneMigration(src, "/unused", true, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(summary.Locales) != 0 {
		t.Errorf("expected 0 locales, got %d", len(summary.Locales))
	}
	if len(summary.Warnings) == 0 {
		t.Error("expected warning about no locale directories")
	}
}

func TestRunFastlaneMigration_SourceNotDir(t *testing.T) {
	tmp := t.TempDir()
	file := filepath.Join(tmp, "notadir.txt")
	if err := os.WriteFile(file, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := runFastlaneMigration(file, "/unused", true, nil)
	if err == nil {
		t.Fatal("expected error for non-directory source")
	}
}

func TestRunFastlaneMigration_SourceMissing(t *testing.T) {
	_, err := runFastlaneMigration("/tmp/nonexistent-migrate-test", "/unused", true, nil)
	if err == nil {
		t.Fatal("expected error for missing source")
	}
}

func TestMigrateCommand_ShowsHelp(t *testing.T) {
	cmd := MigrateCommand()
	err := cmd.ParseAndRun(context.Background(), []string{})
	if err == nil {
		t.Fatal("expected ErrHelp from parent command")
	}
}

func TestIsImageFile(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{"photo.png", true},
		{"photo.PNG", true},
		{"photo.jpg", true},
		{"photo.jpeg", true},
		{"photo.webp", true},
		{"readme.txt", false},
		{"data.json", false},
		{".gitkeep", false},
	}
	for _, tt := range tests {
		if got := isImageFile(tt.name); got != tt.want {
			t.Errorf("isImageFile(%q) = %v, want %v", tt.name, got, tt.want)
		}
	}
}

// ─── Test helpers ──────────────────────────────────────────────────────────

func findLocale(t *testing.T, s *FastlaneSummary, locale string) LocaleSummary {
	t.Helper()
	for _, ls := range s.Locales {
		if ls.Locale == locale {
			return ls
		}
	}
	t.Fatalf("locale %q not found in summary", locale)
	return LocaleSummary{}
}

func assertContains(t *testing.T, slice []string, item string) {
	t.Helper()
	for _, s := range slice {
		if s == item {
			return
		}
	}
	t.Errorf("expected %v to contain %q", slice, item)
}

func assertFileContent(t *testing.T, path, want string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("cannot read %s: %v", path, err)
	}
	if got := string(data); got != want {
		t.Errorf("%s: got %q, want %q", path, got, want)
	}
}

func assertFileExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Errorf("expected file to exist: %s", path)
	}
}
