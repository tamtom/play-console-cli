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

// localeListing holds the metadata files read from disk for a single locale.
type localeListing struct {
	locale           string
	title            string
	shortDescription string
	fullDescription  string
	video            string
}

// PushCommand returns the metadata push subcommand.
func PushCommand() *ffcli.Command {
	fs := flag.NewFlagSet("metadata push", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	dir := fs.String("dir", "", "Metadata directory to read from (required)")
	locales := fs.String("locales", "", "Comma-separated list of locales to push (optional, pushes all if omitted)")
	confirm := fs.Bool("confirm", false, "Confirm push (required for safety)")
	dryRun := fs.Bool("dry-run", false, "Show what would be updated without calling API")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "push",
		ShortUsage: "gplay metadata push --package <name> --dir <path> [--locales en-US] [--confirm] [--dry-run]",
		ShortHelp:  "Push local metadata files to Google Play.",
		LongHelp: `Push store listing metadata from local files to Google Play.

Reads files from the metadata directory and updates listings for each locale.
Creates an edit, updates each listing, and commits the edit.

Requires --confirm for safety. Use --dry-run to preview changes.

Examples:
  gplay metadata push --package com.example --dir ./metadata --confirm
  gplay metadata push --package com.example --dir ./metadata --dry-run
  gplay metadata push --package com.example --dir ./metadata --locales en-US --confirm`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}

			if !*dryRun && !*confirm {
				return fmt.Errorf("--confirm is required (or use --dry-run to preview)")
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

			err = executePush(ctx, service.API, pkg, dirValue, localeFilter, *dryRun)
			if err != nil {
				return err
			}

			result := map[string]interface{}{
				"package": pkg,
				"dir":     dirValue,
				"dryRun":  *dryRun,
				"status":  "pushed",
			}
			return shared.PrintOutput(result, *outputFlag, *pretty)
		},
	}
}

// executePush performs the actual push operation.
// This is separated for testability with mock API services.
func executePush(ctx context.Context, api *androidpublisher.Service, packageName, dir string, localeFilter []string, dryRun bool) error {
	listings, err := readLocalMetadata(dir, localeFilter)
	if err != nil {
		return err
	}

	if len(listings) == 0 {
		return fmt.Errorf("no metadata files found in %s", dir)
	}

	if dryRun {
		for _, l := range listings {
			fmt.Fprintf(os.Stderr, "[DRY RUN] Would update %s: title=%q, short_description=%q, full_description=%q\n",
				l.locale, l.title, truncate(l.shortDescription, 40), truncate(l.fullDescription, 40))
		}
		fmt.Fprintf(os.Stderr, "[DRY RUN] Would push %d locales (no changes made)\n", len(listings))
		return nil
	}

	// Insert a new edit
	edit, err := api.Edits.Insert(packageName, nil).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("failed to create edit: %w", err)
	}

	// Update each listing
	for _, l := range listings {
		listing := &androidpublisher.Listing{
			Title:            l.title,
			ShortDescription: l.shortDescription,
			FullDescription:  l.fullDescription,
			Video:            l.video,
		}
		_, err := api.Edits.Listings.Update(packageName, edit.Id, l.locale, listing).Context(ctx).Do()
		if err != nil {
			return fmt.Errorf("failed to update listing for %s: %w", l.locale, err)
		}
	}

	// Commit the edit
	_, err = api.Edits.Commit(packageName, edit.Id).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("failed to commit edit: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Pushed %d locales\n", len(listings))
	return nil
}

// readLocalMetadata reads metadata files from the given directory.
func readLocalMetadata(dir string, localeFilter []string) ([]localeListing, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory %s: %w", dir, err)
	}

	filterSet := make(map[string]bool)
	for _, l := range localeFilter {
		filterSet[l] = true
	}

	var listings []localeListing
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		locale := entry.Name()
		if len(filterSet) > 0 && !filterSet[locale] {
			continue
		}

		localeDir := filepath.Join(dir, locale)
		l, err := readLocaleListing(locale, localeDir)
		if err != nil {
			return nil, err
		}
		if l != nil {
			listings = append(listings, *l)
		}
	}

	sort.Slice(listings, func(i, j int) bool {
		return listings[i].locale < listings[j].locale
	})

	return listings, nil
}

// readLocaleListing reads all metadata files for a single locale.
func readLocaleListing(locale, localeDir string) (*localeListing, error) {
	l := &localeListing{locale: locale}
	hasContent := false

	if data, err := os.ReadFile(filepath.Join(localeDir, "title.txt")); err == nil {
		l.title = string(data)
		hasContent = true
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to read title.txt for %s: %w", locale, err)
	}

	if data, err := os.ReadFile(filepath.Join(localeDir, "short_description.txt")); err == nil {
		l.shortDescription = string(data)
		hasContent = true
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to read short_description.txt for %s: %w", locale, err)
	}

	if data, err := os.ReadFile(filepath.Join(localeDir, "full_description.txt")); err == nil {
		l.fullDescription = string(data)
		hasContent = true
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to read full_description.txt for %s: %w", locale, err)
	}

	if data, err := os.ReadFile(filepath.Join(localeDir, "video_url.txt")); err == nil {
		l.video = strings.TrimSpace(string(data))
		hasContent = true
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to read video_url.txt for %s: %w", locale, err)
	}

	if !hasContent {
		return nil, nil
	}
	return l, nil
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
