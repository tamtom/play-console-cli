package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGlobalDryRunReachesCommandsWithLocalDryRun(t *testing.T) {
	t.Run("workflow run", func(t *testing.T) {
		dir := t.TempDir()
		t.Chdir(dir)
		marker := filepath.Join(dir, "marker")
		def := `{"workflows":{"ship":{"steps":[{"name":"touch","command":"touch ` + marker + `"}]}}}`
		if err := os.WriteFile("wf.json", []byte(def), 0o600); err != nil {
			t.Fatal(err)
		}
		code, _, stderr := runReleaseCommand(t, []string{"--dry-run", "workflow", "run", "wf.json"}, nil)
		if code != ExitSuccess {
			t.Fatalf("exit %d, stderr %q", code, stderr)
		}
		if _, err := os.Stat(marker); !os.IsNotExist(err) {
			t.Fatalf("global --dry-run executed a workflow step (marker stat err=%v)", err)
		}
		if _, err := os.Stat(filepath.Join(dir, ".gplay")); !os.IsNotExist(err) {
			t.Fatalf("global --dry-run wrote workflow state (stat err=%v)", err)
		}
	})

	t.Run("migrate fastlane", func(t *testing.T) {
		dir := t.TempDir()
		locale := filepath.Join(dir, "src", "en-US")
		if err := os.MkdirAll(locale, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(locale, "title.txt"), []byte("My App"), 0o600); err != nil {
			t.Fatal(err)
		}
		out := filepath.Join(dir, "out")
		code, stdout, stderr := runReleaseCommand(t, []string{"--dry-run", "migrate", "fastlane", "--source", filepath.Join(dir, "src"), "--output-dir", out}, nil)
		if code != ExitSuccess {
			t.Fatalf("exit %d, stderr %q", code, stderr)
		}
		if _, err := os.Stat(out); !os.IsNotExist(err) {
			t.Fatalf("global --dry-run wrote migrated files (stat err=%v)", err)
		}
		if !strings.Contains(stdout, `"dryRun":true`) {
			t.Fatalf("summary does not report dry run: %s", stdout)
		}
	})
}
