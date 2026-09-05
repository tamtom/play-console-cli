package cmdtest_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/tamtom/play-console-cli/internal/cmdtest"
	"github.com/tamtom/play-console-cli/internal/sandbox"
)

func setupStarSuggestion(t *testing.T) string {
	t.Helper()
	cmdtest.Build(t)
	startSandbox(t)
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("GPLAY_NO_STAR_PROMPT", "")
	bin := t.TempDir()
	name := "gh"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	if err := os.WriteFile(filepath.Join(bin, name), []byte("must not execute gh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin)
	return home
}

func newStarSuggestionEdit(t *testing.T) string {
	t.Helper()
	created := sandboxJSON(t, "edits", "create", "--package", sandbox.Package)
	return created["id"].(string)
}

func TestStarSuggestion_MetadataCommitSuggestsOnceAcrossProcesses(t *testing.T) {
	setupStarSuggestion(t)
	for attempt := 0; attempt < 2; attempt++ {
		edit := newStarSuggestionEdit(t)
		sandboxJSON(t, "listings", "update", "--package", sandbox.Package, "--edit", edit,
			"--locale", "en-US", "--title", "Updated title")
		result := cmdtest.Run(t, "edits", "commit", "--package", sandbox.Package, "--edit", edit)
		if result.ExitCode != 0 || !json.Valid([]byte(result.Stdout)) {
			t.Fatalf("commit failed or stdout is not JSON: %+v", result)
		}
		if attempt == 0 {
			for _, want := range []string{"Would you like to star", "Only after an explicit yes", "gh api --hostname github.com --method PUT /user/starred/tamtom/play-console-cli"} {
				if !strings.Contains(result.Stderr, want) {
					t.Errorf("first commit stderr must contain %q, got %q", want, result.Stderr)
				}
			}
		} else if strings.Contains(result.Stderr, "star") {
			t.Errorf("suggestion repeated in another process: %q", result.Stderr)
		}
	}
}

func TestStarSuggestion_ReleaseSharesOneTimeStateWithCommit(t *testing.T) {
	setupStarSuggestion(t)
	bundle := filepath.Join(t.TempDir(), "app.aab")
	if err := os.WriteFile(bundle, []byte("sandbox bundle"), 0o600); err != nil {
		t.Fatal(err)
	}
	result := cmdtest.Run(t, "release", "--package", sandbox.Package, "--bundle", bundle)
	if result.ExitCode != 0 || !json.Valid([]byte(result.Stdout)) {
		t.Fatalf("release failed or stdout is not JSON: %+v", result)
	}
	if !strings.Contains(result.Stderr, "Would you like to star") {
		t.Fatalf("successful release must suggest starring: %q", result.Stderr)
	}
	edit := newStarSuggestionEdit(t)
	result = cmdtest.Run(t, "edits", "commit", "--package", sandbox.Package, "--edit", edit)
	if result.ExitCode != 0 || strings.Contains(result.Stderr, "Would you like to star") {
		t.Fatalf("commit after a release must succeed without another suggestion: %+v", result)
	}
}

func TestStarSuggestion_MetadataPushSuggestsOnceAfterRealSubmission(t *testing.T) {
	setupStarSuggestion(t)
	dir := t.TempDir()
	localeDir := filepath.Join(dir, "en-US")
	if err := os.Mkdir(localeDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(localeDir, "title.txt"), []byte("Updated title"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, attempt := range []string{"dry run", "first push", "second push"} {
		args := []string{"metadata", "push", "--package", sandbox.Package, "--dir", dir, "--confirm"}
		if attempt == "dry run" {
			args = append(args, "--dry-run")
		}
		result := cmdtest.Run(t, args...)
		if result.ExitCode != 0 || !json.Valid([]byte(result.Stdout)) {
			t.Fatalf("%s failed or stdout is not JSON: %+v", attempt, result)
		}
		wantSuggestion := attempt == "first push"
		if got := strings.Contains(result.Stderr, "Would you like to star"); got != wantSuggestion {
			t.Errorf("%s suggestion = %t, want %t: %q", attempt, got, wantSuggestion, result.Stderr)
		}
	}
}

func TestStarSuggestion_SkippedAttemptsDoNotConsumeSuggestion(t *testing.T) {
	for _, scenario := range []string{"no gh", "dry run", "failure", "read", "opt out"} {
		t.Run(scenario, func(t *testing.T) {
			setupStarSuggestion(t)
			bin := os.Getenv("PATH")
			edit := newStarSuggestionEdit(t)
			args := []string{"edits", "commit", "--package", sandbox.Package, "--edit", edit}
			switch scenario {
			case "no gh":
				t.Setenv("PATH", t.TempDir())
			case "dry run":
				args = append([]string{"--dry-run"}, args...)
			case "failure":
				args[len(args)-1] = "missing-edit"
			case "read":
				args[1] = "get"
			case "opt out":
				t.Setenv("GPLAY_NO_STAR_PROMPT", "1")
			}
			result := cmdtest.Run(t, args...)
			if scenario == "failure" {
				if result.ExitCode == 0 || result.Stderr == "" {
					t.Fatalf("failed commit must report an error: %+v", result)
				}
			} else if result.ExitCode != 0 {
				t.Fatalf("command failed: %+v", result)
			}
			if strings.Contains(result.Stderr, "Would you like to star") {
				t.Fatalf("unexpected star suggestion: %q", result.Stderr)
			}
			t.Setenv("PATH", bin)
			t.Setenv("GPLAY_NO_STAR_PROMPT", "")
			edit = newStarSuggestionEdit(t)
			result = cmdtest.Run(t, "edits", "commit", "--package", sandbox.Package, "--edit", edit)
			if result.ExitCode != 0 || !strings.Contains(result.Stderr, "Would you like to star") {
				t.Fatalf("next successful commit must still suggest starring: %+v", result)
			}
		})
	}
}

func TestStarSuggestion_UnwritableStateDoesNotFailCommit(t *testing.T) {
	home := setupStarSuggestion(t)
	if err := os.WriteFile(filepath.Join(home, ".gplay"), []byte("not a directory"), 0o600); err != nil {
		t.Fatal(err)
	}
	edit := newStarSuggestionEdit(t)
	result := cmdtest.Run(t, "edits", "commit", "--package", sandbox.Package, "--edit", edit)
	if result.ExitCode != 0 || !json.Valid([]byte(result.Stdout)) || strings.Contains(result.Stderr, "Would you like to star") {
		t.Fatalf("state errors must silently skip the suggestion: %+v", result)
	}
}

func TestStarSuggestion_ConcurrentCommitsSuggestOnlyOnce(t *testing.T) {
	setupStarSuggestion(t)
	var edits []string
	for i := 0; i < 4; i++ {
		edits = append(edits, newStarSuggestionEdit(t))
	}
	results := make(chan cmdtest.Result, len(edits))
	for _, edit := range edits {
		go func() {
			results <- cmdtest.Run(t, "edits", "commit", "--package", sandbox.Package, "--edit", edit)
		}()
	}
	count := 0
	for range edits {
		result := <-results
		if result.ExitCode != 0 {
			t.Errorf("commit failed: %+v", result)
		}
		count += strings.Count(result.Stderr, "Would you like to star")
	}
	if count != 1 {
		t.Errorf("concurrent commits emitted %d suggestions, want 1", count)
	}
}
