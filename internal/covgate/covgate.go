// Package covgate enforces the command-coverage manifest: every CLI command
// must carry an explicit test-coverage class. CI fails when a command is
// added without classifying how it is tested.
package covgate

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/tamtom/play-console-cli/internal/cli/registry"
)

// Classes a command can declare. "sandbox" and "live" both imply the command
// also has offline coverage.
const (
	ClassSandbox = "sandbox" // black-box against the hermetic sandbox API
	ClassLive    = "live"    // exercised by the weekly live smoke suite
	ClassOffline = "offline" // offline cmdtest/unit coverage only
	ClassManual  = "manual"  // no automation possible or allowed; tested by hand
)

// Entry is one command's declared coverage.
type Entry struct {
	Class string `json:"class"`
	Note  string `json:"note,omitempty"`
}

// Manifest maps a full command path ("tracks list") to its coverage entry.
type Manifest map[string]Entry

// LoadManifest reads and validates the manifest file.
func LoadManifest(path string) (Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	for cmd, e := range m {
		switch e.Class {
		case ClassSandbox, ClassLive, ClassOffline, ClassManual:
		default:
			return nil, fmt.Errorf("command %q has unknown class %q", cmd, e.Class)
		}
	}
	return m, nil
}

// LeafCommands returns the sorted full paths of every runnable leaf command
// in the registry.
func LeafCommands() []string {
	var leaves []string
	for _, cmd := range registry.Subcommands("dev") {
		walk("", cmd, &leaves)
	}
	sort.Strings(leaves)
	return leaves
}

func walk(prefix string, cmd *ffcli.Command, out *[]string) {
	path := cmd.Name
	if prefix != "" {
		path = prefix + " " + cmd.Name
	}
	if len(cmd.Subcommands) == 0 {
		*out = append(*out, path)
		return
	}
	for _, sub := range cmd.Subcommands {
		walk(path, sub, out)
	}
}

// Diff compares the registry against the manifest. Missing holds commands
// without a manifest entry; stale holds manifest entries whose command no
// longer exists.
func Diff(m Manifest) (missing, stale []string) {
	leaves := LeafCommands()
	seen := map[string]bool{}
	for _, leaf := range leaves {
		seen[leaf] = true
		if _, ok := m[leaf]; !ok {
			missing = append(missing, leaf)
		}
	}
	for cmd := range m {
		if !seen[cmd] {
			stale = append(stale, cmd)
		}
	}
	sort.Strings(missing)
	sort.Strings(stale)
	return missing, stale
}
