package metadata

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/peterbourgon/ff/v3/ffcli"
	"google.golang.org/api/androidpublisher/v3"

	"github.com/tamtom/play-console-cli/internal/cli/shared"
	"github.com/tamtom/play-console-cli/internal/playclient"
)

// PullCommand returns the metadata pull subcommand.
func PullCommand() *ffcli.Command {
	fs := flag.NewFlagSet("metadata pull", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	dir := fs.String("dir", "", "Output directory for metadata files (required)")
	locales := fs.String("locales", "", "Comma-separated list of locales to pull (optional, pulls all if omitted)")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "pull",
		ShortUsage: "gplay metadata pull --package <name> --dir <path> [--locales en-US,ja-JP]",
		ShortHelp:  "Pull store listing metadata into local files.",
		LongHelp: `Pull store listing metadata from Google Play into local files.

Creates a directory structure with one folder per locale:
  metadata/
    en-US/
      title.txt
      short_description.txt
      full_description.txt
      video_url.txt          (if present)
    ja-JP/
      ...

Examples:
  gplay metadata pull --package com.example --dir ./metadata
  gplay metadata pull --package com.example --dir ./metadata --locales en-US,ja-JP`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}

			service, err := playclient.NewService(ctx)
			if err != nil {
				return err
			}
			pkg := shared.ResolvePackageName(*packageName, service.Cfg)
			if strings.TrimSpace(pkg) == "" {
				return fmt.Errorf("--package is required")
			}
			dirValue := strings.TrimSpace(*dir)
			if dirValue == "" {
				return fmt.Errorf("--dir is required")
			}

			var localeFilter []string
			if strings.TrimSpace(*locales) != "" {
				for _, l := range strings.Split(*locales, ",") {
					trimmed := strings.TrimSpace(l)
					if trimmed != "" {
						localeFilter = append(localeFilter, trimmed)
					}
				}
			}

			ctx, cancel := shared.ContextWithTimeout(ctx, service.Cfg)
			defer cancel()

			err = executePull(ctx, service.API, pkg, dirValue, localeFilter)
			if err != nil {
				return err
			}

			result := map[string]interface{}{
				"package": pkg,
				"dir":     dirValue,
				"status":  "pulled",
			}
			return shared.PrintOutput(result, *outputFlag, *pretty)
		},
	}
}

// executePull performs the actual pull operation.
// This is separated for testability with mock API services.
func executePull(ctx context.Context, api *androidpublisher.Service, packageName, dir string, localeFilter []string) error {
	// Insert a new edit
	edit, err := api.Edits.Insert(packageName, nil).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("failed to create edit: %w", err)
	}

	// List all listings
	listResp, err := api.Edits.Listings.List(packageName, edit.Id).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("failed to list listings: %w", err)
	}

	// Delete the edit (we only needed read access)
	_ = api.Edits.Delete(packageName, edit.Id).Context(ctx).Do()

	if listResp == nil || len(listResp.Listings) == 0 {
		return fmt.Errorf("no listings found for package %s", packageName)
	}

	// Build locale filter set
	filterSet := make(map[string]bool)
	for _, l := range localeFilter {
		filterSet[l] = true
	}

	var pulledLocales []string
	for _, listing := range listResp.Listings {
		locale := listing.Language
		if len(filterSet) > 0 && !filterSet[locale] {
			continue
		}

		localeDir := filepath.Join(dir, locale)
		if err := os.MkdirAll(localeDir, 0o755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", localeDir, err)
		}

		if err := writeMetadataFile(filepath.Join(localeDir, "title.txt"), listing.Title); err != nil {
			return err
		}
		if err := writeMetadataFile(filepath.Join(localeDir, "short_description.txt"), listing.ShortDescription); err != nil {
			return err
		}
		if err := writeMetadataFile(filepath.Join(localeDir, "full_description.txt"), listing.FullDescription); err != nil {
			return err
		}
		if strings.TrimSpace(listing.Video) != "" {
			if err := writeMetadataFile(filepath.Join(localeDir, "video_url.txt"), listing.Video); err != nil {
				return err
			}
		}

		pulledLocales = append(pulledLocales, locale)
	}

	sort.Strings(pulledLocales)
	fmt.Fprintf(os.Stderr, "Pulled %d locales to %s\n", len(pulledLocales), dir)
	return nil
}

func writeMetadataFile(path, content string) error {
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return fmt.Errorf("failed to write %s: %w", path, err)
	}
	return nil
}
