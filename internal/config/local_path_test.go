package config

import (
	"os"
	"path/filepath"
	"testing"
)

// resolvedTempDir returns a temp directory with symlinks resolved.
// On macOS, /var is a symlink to /private/var, which causes path mismatches.
func resolvedTempDir(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	resolved, err := filepath.EvalSymlinks(tmpDir)
	if err != nil {
		t.Fatal(err)
	}
	return resolved
}

// chdirCleanup changes to dir and returns a cleanup that restores the original cwd.
func chdirCleanup(t *testing.T, dir string) {
	t.Helper()
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(origDir); err != nil {
			t.Fatal(err)
		}
	})
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
}

func TestLocalPath_FindsInCurrentDir(t *testing.T) {
	tmpDir := resolvedTempDir(t)
	gplayDir := filepath.Join(tmpDir, configDirName)
	if err := os.MkdirAll(gplayDir, 0o755); err != nil {
		t.Fatal(err)
	}
	configFile := filepath.Join(gplayDir, configFileName)
	if err := os.WriteFile(configFile, []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}

	chdirCleanup(t, tmpDir)

	got, err := LocalPath()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != configFile {
		t.Errorf("expected %q, got %q", configFile, got)
	}
}

func TestLocalPath_WalksUpToParent(t *testing.T) {
	tmpDir := resolvedTempDir(t)
	gplayDir := filepath.Join(tmpDir, configDirName)
	if err := os.MkdirAll(gplayDir, 0o755); err != nil {
		t.Fatal(err)
	}
	configFile := filepath.Join(gplayDir, configFileName)
	if err := os.WriteFile(configFile, []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}

	childDir := filepath.Join(tmpDir, "sub", "deep")
	if err := os.MkdirAll(childDir, 0o755); err != nil {
		t.Fatal(err)
	}

	chdirCleanup(t, childDir)

	got, err := LocalPath()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != configFile {
		t.Errorf("expected %q, got %q", configFile, got)
	}
}

func TestLocalPath_StopsAtGitBoundary(t *testing.T) {
	tmpDir := resolvedTempDir(t)

	// Parent has .gplay config (should NOT be found due to .git boundary)
	parentGplayDir := filepath.Join(tmpDir, configDirName)
	if err := os.MkdirAll(parentGplayDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(parentGplayDir, configFileName), []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}

	// Create a .git dir as boundary
	repoDir := filepath.Join(tmpDir, "repo")
	gitDir := filepath.Join(repoDir, ".git")
	if err := os.MkdirAll(gitDir, 0o755); err != nil {
		t.Fatal(err)
	}

	subDir := filepath.Join(repoDir, "sub")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatal(err)
	}

	chdirCleanup(t, subDir)

	got, err := LocalPath()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should fall back to cwd/.gplay/config.json, not find parent's config
	expected := filepath.Join(subDir, configDirName, configFileName)
	if got != expected {
		t.Errorf("expected fallback %q, got %q", expected, got)
	}
}

func TestLocalPath_FallsBackToCwd(t *testing.T) {
	tmpDir := resolvedTempDir(t)

	chdirCleanup(t, tmpDir)

	got, err := LocalPath()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := filepath.Join(tmpDir, configDirName, configFileName)
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}
