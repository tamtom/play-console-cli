package sync

import (
	"os"
	"path/filepath"
	"testing"

	"google.golang.org/api/androidpublisher/v3"

	"github.com/tamtom/play-console-cli/internal/rootfs"
)

func TestExportListingsToRootRejectsEscapingLocale(t *testing.T) {
	parent := t.TempDir()
	root, err := rootfs.OpenOrCreate(filepath.Join(parent, "metadata"), 0o755)
	if err != nil {
		t.Fatalf("OpenOrCreate: %v", err)
	}
	defer func() { _ = root.Close() }()

	err = exportListingsToRoot(root, "fastlane", []*androidpublisher.Listing{{
		Language: "../../escape", Title: "unsafe",
	}})
	if err == nil {
		t.Fatal("exportListingsToRoot unexpectedly accepted an escaping locale")
	}
	if _, err := os.Stat(filepath.Join(parent, "escape", titleFile)); !os.IsNotExist(err) {
		t.Fatal("listing export wrote outside the selected output directory")
	}
}

func TestExportListingsToRootWritesFastlaneAndJSON(t *testing.T) {
	for _, format := range []string{"fastlane", "json"} {
		t.Run(format, func(t *testing.T) {
			root, err := rootfs.OpenOrCreate(t.TempDir(), 0o755)
			if err != nil {
				t.Fatalf("OpenOrCreate: %v", err)
			}
			defer func() { _ = root.Close() }()
			listing := &androidpublisher.Listing{Language: "en-US", Title: "Example", ShortDescription: "Short", FullDescription: "Full"}
			if err := exportListingsToRoot(root, format, []*androidpublisher.Listing{listing}); err != nil {
				t.Fatalf("exportListingsToRoot: %v", err)
			}
			name := filepath.Join("en-US", titleFile)
			if format == "json" {
				name = filepath.Join("en-US", "listing.json")
			}
			if _, err := root.ReadFile(name); err != nil {
				t.Fatalf("read exported file: %v", err)
			}
		})
	}
}
