package metadata

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateCommand_Name(t *testing.T) {
	cmd := ValidateCommand()
	if cmd.Name != "validate" {
		t.Errorf("expected name %q, got %q", "validate", cmd.Name)
	}
}

func TestValidateCommand_MissingDir(t *testing.T) {
	cmd := ValidateCommand()
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for missing --dir")
	}
}

func TestValidateCommand_EmptyDir(t *testing.T) {
	cmd := ValidateCommand()
	if err := cmd.FlagSet.Parse([]string{"--dir", "   "}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for empty --dir")
	}
}

func TestValidateCommand_NonExistentDir(t *testing.T) {
	cmd := ValidateCommand()
	if err := cmd.FlagSet.Parse([]string{"--dir", "/nonexistent/path"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for non-existent directory")
	}
}

func TestValidateCommand_ValidMetadata(t *testing.T) {
	dir := t.TempDir()
	localeDir := filepath.Join(dir, "en-US")
	if err := os.MkdirAll(localeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(localeDir, "title.txt"), "My App")
	writeFile(t, filepath.Join(localeDir, "short_description.txt"), "A short description")
	writeFile(t, filepath.Join(localeDir, "full_description.txt"), "A full description of the app.")

	cmd := ValidateCommand()
	if err := cmd.FlagSet.Parse([]string{"--dir", dir}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateCommand_TitleTooLong(t *testing.T) {
	dir := t.TempDir()
	localeDir := filepath.Join(dir, "en-US")
	if err := os.MkdirAll(localeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(localeDir, "title.txt"), strings.Repeat("a", 31))
	writeFile(t, filepath.Join(localeDir, "short_description.txt"), "Short")
	writeFile(t, filepath.Join(localeDir, "full_description.txt"), "Full")

	cmd := ValidateCommand()
	if err := cmd.FlagSet.Parse([]string{"--dir", dir}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for title exceeding 30 chars")
	}
}

func TestValidateCommand_TitleExactlyAtLimit(t *testing.T) {
	dir := t.TempDir()
	localeDir := filepath.Join(dir, "en-US")
	if err := os.MkdirAll(localeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(localeDir, "title.txt"), strings.Repeat("a", 30))
	writeFile(t, filepath.Join(localeDir, "short_description.txt"), "Short")
	writeFile(t, filepath.Join(localeDir, "full_description.txt"), "Full")

	cmd := ValidateCommand()
	if err := cmd.FlagSet.Parse([]string{"--dir", dir}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error for title at exactly 30 chars: %v", err)
	}
}

func TestValidateCommand_ShortDescriptionTooLong(t *testing.T) {
	dir := t.TempDir()
	localeDir := filepath.Join(dir, "en-US")
	if err := os.MkdirAll(localeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(localeDir, "title.txt"), "Title")
	writeFile(t, filepath.Join(localeDir, "short_description.txt"), strings.Repeat("b", 81))
	writeFile(t, filepath.Join(localeDir, "full_description.txt"), "Full")

	cmd := ValidateCommand()
	if err := cmd.FlagSet.Parse([]string{"--dir", dir}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for short description exceeding 80 chars")
	}
}

func TestValidateCommand_ShortDescriptionExactlyAtLimit(t *testing.T) {
	dir := t.TempDir()
	localeDir := filepath.Join(dir, "en-US")
	if err := os.MkdirAll(localeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(localeDir, "title.txt"), "Title")
	writeFile(t, filepath.Join(localeDir, "short_description.txt"), strings.Repeat("b", 80))
	writeFile(t, filepath.Join(localeDir, "full_description.txt"), "Full")

	cmd := ValidateCommand()
	if err := cmd.FlagSet.Parse([]string{"--dir", dir}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error for short description at exactly 80 chars: %v", err)
	}
}

func TestValidateCommand_FullDescriptionTooLong(t *testing.T) {
	dir := t.TempDir()
	localeDir := filepath.Join(dir, "en-US")
	if err := os.MkdirAll(localeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(localeDir, "title.txt"), "Title")
	writeFile(t, filepath.Join(localeDir, "short_description.txt"), "Short")
	writeFile(t, filepath.Join(localeDir, "full_description.txt"), strings.Repeat("c", 4001))

	cmd := ValidateCommand()
	if err := cmd.FlagSet.Parse([]string{"--dir", dir}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for full description exceeding 4000 chars")
	}
}

func TestValidateCommand_FullDescriptionExactlyAtLimit(t *testing.T) {
	dir := t.TempDir()
	localeDir := filepath.Join(dir, "en-US")
	if err := os.MkdirAll(localeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(localeDir, "title.txt"), "Title")
	writeFile(t, filepath.Join(localeDir, "short_description.txt"), "Short")
	writeFile(t, filepath.Join(localeDir, "full_description.txt"), strings.Repeat("c", 4000))

	cmd := ValidateCommand()
	if err := cmd.FlagSet.Parse([]string{"--dir", dir}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error for full description at exactly 4000 chars: %v", err)
	}
}

func TestValidateCommand_MultipleLocales(t *testing.T) {
	dir := t.TempDir()
	for _, locale := range []string{"en-US", "ja-JP"} {
		localeDir := filepath.Join(dir, locale)
		if err := os.MkdirAll(localeDir, 0o755); err != nil {
			t.Fatal(err)
		}
		writeFile(t, filepath.Join(localeDir, "title.txt"), "Title")
		writeFile(t, filepath.Join(localeDir, "short_description.txt"), "Short")
		writeFile(t, filepath.Join(localeDir, "full_description.txt"), "Full")
	}

	cmd := ValidateCommand()
	if err := cmd.FlagSet.Parse([]string{"--dir", dir}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateCommand_MultipleLocalesOneInvalid(t *testing.T) {
	dir := t.TempDir()
	// en-US valid
	enDir := filepath.Join(dir, "en-US")
	if err := os.MkdirAll(enDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(enDir, "title.txt"), "Title")
	writeFile(t, filepath.Join(enDir, "short_description.txt"), "Short")
	writeFile(t, filepath.Join(enDir, "full_description.txt"), "Full")

	// ja-JP with too-long title
	jaDir := filepath.Join(dir, "ja-JP")
	if err := os.MkdirAll(jaDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(jaDir, "title.txt"), strings.Repeat("x", 31))
	writeFile(t, filepath.Join(jaDir, "short_description.txt"), "Short")
	writeFile(t, filepath.Join(jaDir, "full_description.txt"), "Full")

	cmd := ValidateCommand()
	if err := cmd.FlagSet.Parse([]string{"--dir", dir}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error when one locale has validation errors")
	}
}

func TestValidateCommand_EmptyMetadataDir(t *testing.T) {
	dir := t.TempDir()
	cmd := ValidateCommand()
	if err := cmd.FlagSet.Parse([]string{"--dir", dir}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for empty metadata directory")
	}
}

func TestValidateCommand_VideoURLPresent(t *testing.T) {
	dir := t.TempDir()
	localeDir := filepath.Join(dir, "en-US")
	if err := os.MkdirAll(localeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(localeDir, "title.txt"), "My App")
	writeFile(t, filepath.Join(localeDir, "short_description.txt"), "Short")
	writeFile(t, filepath.Join(localeDir, "full_description.txt"), "Full")
	writeFile(t, filepath.Join(localeDir, "video_url.txt"), "https://youtube.com/watch?v=abc")

	cmd := ValidateCommand()
	if err := cmd.FlagSet.Parse([]string{"--dir", dir}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
