// Command covmanifest refreshes docs/testing/command-coverage.json.
// Existing classifications are kept. New commands get the class "offline",
// because every command has at least offline cmdtest coverage. Entries for
// removed commands are dropped.
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/tamtom/play-console-cli/internal/covgate"
)

const manifestPath = "docs/testing/command-coverage.json"

func main() {
	existing, err := covgate.LoadManifest(manifestPath)
	if errors.Is(err, os.ErrNotExist) {
		existing = covgate.Manifest{}
	} else if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	fresh := covgate.Manifest{}
	added := 0
	for _, cmd := range covgate.LeafCommands() {
		if entry, ok := existing[cmd]; ok {
			fresh[cmd] = entry
			continue
		}
		fresh[cmd] = covgate.Entry{Class: covgate.ClassOffline}
		added++
	}

	data, err := json.MarshalIndent(fresh, "", "  ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := os.WriteFile(manifestPath, append(data, '\n'), 0o644); err != nil { // #nosec G306 -- checked-in docs file
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("wrote %s: %d commands (%d new, %d removed)\n",
		manifestPath, len(fresh), added, len(existing)-(len(fresh)-added))
}
