package covgate

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const manifestPath = "../../docs/testing/command-coverage.json"

// TestCommandCoverageManifest is the CI gate: every command must carry a
// coverage class, and the manifest must not list removed commands.
func TestCommandCoverageManifest(t *testing.T) {
	m, err := LoadManifest(manifestPath)
	if err != nil {
		t.Fatalf("loading manifest: %v\nRegenerate with: go run ./tools/covmanifest", err)
	}

	missing, stale := Diff(m)
	if len(missing) > 0 {
		t.Errorf("commands without a coverage class (add them to %s):\n  %s",
			manifestPath, strings.Join(missing, "\n  "))
	}
	if len(stale) > 0 {
		t.Errorf("manifest entries for removed commands (delete them from %s):\n  %s",
			manifestPath, strings.Join(stale, "\n  "))
	}
	if t.Failed() {
		t.Log("Regenerate with: go run ./tools/covmanifest")
	}
}

func TestLeafCommands_NotEmptyAndSorted(t *testing.T) {
	leaves := LeafCommands()
	if len(leaves) < 50 {
		t.Fatalf("expected a substantial command tree, got %d leaves", len(leaves))
	}
	for i := 1; i < len(leaves); i++ {
		if leaves[i-1] >= leaves[i] {
			t.Fatalf("leaves must be sorted and unique: %q >= %q", leaves[i-1], leaves[i])
		}
	}
}

func TestLoadManifest_RejectsUnknownClass(t *testing.T) {
	path := filepath.Join(t.TempDir(), "m.json")
	bad, _ := json.Marshal(Manifest{"tracks list": {Class: "vibes"}})
	if err := os.WriteFile(path, bad, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadManifest(path); err == nil {
		t.Fatal("unknown class must be rejected")
	}
}
