package migrate

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/tamtom/play-console-cli/internal/cli/shared"
)

// Known Fastlane text files that map to store listing fields.
var fastlaneTextFiles = []string{
	"title.txt",
	"short_description.txt",
	"full_description.txt",
	"video.txt",
}

// Known Fastlane image directories.
var fastlaneImageDirs = []string{
	"phoneScreenshots",
	"sevenInchScreenshots",
	"tenInchScreenshots",
	"tvScreenshots",
	"wearScreenshots",
}

// Known Fastlane single-image files.
var fastlaneSingleImages = []string{
	"featureGraphic.png",
	"icon.png",
	"promoGraphic.png",
	"tvBanner.png",
}

// FastlaneSummary is the JSON-serialisable result of a migration.
type FastlaneSummary struct {
	Source      string           `json:"source"`
	OutputDir   string           `json:"outputDir,omitempty"`
	DryRun      bool             `json:"dryRun"`
	Locales     []LocaleSummary  `json:"locales"`
	TotalFiles  int              `json:"totalFiles"`
	TotalImages int              `json:"totalImages"`
	Warnings    []string         `json:"warnings,omitempty"`
}

// LocaleSummary describes what was found/imported for one locale.
type LocaleSummary struct {
	Locale     string   `json:"locale"`
	TextFiles  []string `json:"textFiles,omitempty"`
	Changelogs []string `json:"changelogs,omitempty"`
	Images     []string `json:"images,omitempty"`
}

// FastlaneCommand returns the "migrate fastlane" subcommand.
func FastlaneCommand() *ffcli.Command {
	fs := flag.NewFlagSet("migrate fastlane", flag.ExitOnError)
	source := fs.String("source", "", "Path to Fastlane metadata/android/ directory (required)")
	outputDir := fs.String("output-dir", ".gplay/metadata/", "Output directory for imported metadata")
	dryRun := fs.Bool("dry-run", false, "Preview what would be imported without writing files")
	locales := fs.String("locales", "", "Comma-separated list of locales to import (default: all)")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "fastlane",
		ShortUsage: "gplay migrate fastlane --source <path> [--output-dir <path>] [--dry-run] [--locales <list>]",
		ShortHelp:  "Import metadata from Fastlane directory structure.",
		LongHelp: `Import metadata from a Fastlane metadata/android/ directory.

Reads the standard Fastlane directory layout and copies text files,
changelogs, and images into the gplay metadata directory.

Fastlane directory structure:
  metadata/android/
    en-US/
      title.txt
      short_description.txt
      full_description.txt
      video.txt
      changelogs/
        100.txt
      images/
        phoneScreenshots/
        featureGraphic.png
    de-DE/
      ...`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			if strings.TrimSpace(*source) == "" {
				return fmt.Errorf("--source is required")
			}

			var localeFilter map[string]bool
			if strings.TrimSpace(*locales) != "" {
				localeFilter = make(map[string]bool)
				for _, l := range strings.Split(*locales, ",") {
					l = strings.TrimSpace(l)
					if l != "" {
						localeFilter[l] = true
					}
				}
			}

			summary, err := runFastlaneMigration(*source, *outputDir, *dryRun, localeFilter)
			if err != nil {
				return err
			}

			return shared.PrintOutput(summary, *outputFlag, *pretty)
		},
	}
}

// runFastlaneMigration scans the Fastlane source directory and optionally
// copies files to the output directory. It returns a summary of what was
// found or imported.
func runFastlaneMigration(source, outputDir string, dryRun bool, localeFilter map[string]bool) (*FastlaneSummary, error) {
	// Validate source exists and is a directory.
	info, err := os.Stat(source)
	if err != nil {
		return nil, fmt.Errorf("cannot access source directory: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("source is not a directory: %s", source)
	}

	entries, err := os.ReadDir(source)
	if err != nil {
		return nil, fmt.Errorf("cannot read source directory: %w", err)
	}

	summary := &FastlaneSummary{
		Source:  source,
		DryRun:  dryRun,
	}
	if !dryRun {
		summary.OutputDir = outputDir
	}

	// Collect locale dirs, sorted for deterministic output.
	var localeDirs []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if localeFilter != nil && !localeFilter[name] {
			continue
		}
		localeDirs = append(localeDirs, name)
	}
	sort.Strings(localeDirs)

	if len(localeDirs) == 0 {
		summary.Warnings = append(summary.Warnings, "no locale directories found in source")
		return summary, nil
	}

	for _, locale := range localeDirs {
		localeSrc := filepath.Join(source, locale)
		localeDst := filepath.Join(outputDir, locale)

		ls := LocaleSummary{Locale: locale}

		// --- Text files ---
		for _, tf := range fastlaneTextFiles {
			srcPath := filepath.Join(localeSrc, tf)
			if _, err := os.Stat(srcPath); err != nil {
				continue
			}
			ls.TextFiles = append(ls.TextFiles, tf)
			summary.TotalFiles++

			if !dryRun {
				if err := copyFile(srcPath, filepath.Join(localeDst, tf)); err != nil {
					summary.Warnings = append(summary.Warnings,
						fmt.Sprintf("[%s] failed to copy %s: %v", locale, tf, err))
				}
			}
		}

		// --- Changelogs ---
		changelogSrc := filepath.Join(localeSrc, "changelogs")
		if clEntries, err := os.ReadDir(changelogSrc); err == nil {
			for _, cl := range clEntries {
				if cl.IsDir() || !strings.HasSuffix(cl.Name(), ".txt") {
					continue
				}
				ls.Changelogs = append(ls.Changelogs, cl.Name())
				summary.TotalFiles++

				if !dryRun {
					src := filepath.Join(changelogSrc, cl.Name())
					dst := filepath.Join(localeDst, "changelogs", cl.Name())
					if err := copyFile(src, dst); err != nil {
						summary.Warnings = append(summary.Warnings,
							fmt.Sprintf("[%s] failed to copy changelog %s: %v", locale, cl.Name(), err))
					}
				}
			}
			sort.Strings(ls.Changelogs)
		}

		// --- Images ---
		imagesSrc := filepath.Join(localeSrc, "images")
		if _, err := os.Stat(imagesSrc); err == nil {
			// Screenshot directories
			for _, imgDir := range fastlaneImageDirs {
				dirPath := filepath.Join(imagesSrc, imgDir)
				imgEntries, err := os.ReadDir(dirPath)
				if err != nil {
					continue
				}
				for _, img := range imgEntries {
					if img.IsDir() {
						continue
					}
					if !isImageFile(img.Name()) {
						continue
					}
					rel := filepath.Join("images", imgDir, img.Name())
					ls.Images = append(ls.Images, rel)
					summary.TotalImages++

					if !dryRun {
						src := filepath.Join(dirPath, img.Name())
						dst := filepath.Join(localeDst, rel)
						if err := copyFile(src, dst); err != nil {
							summary.Warnings = append(summary.Warnings,
								fmt.Sprintf("[%s] failed to copy %s: %v", locale, rel, err))
						}
					}
				}
			}

			// Single image files (featureGraphic.png, etc.)
			for _, singleImg := range fastlaneSingleImages {
				srcPath := filepath.Join(imagesSrc, singleImg)
				if _, err := os.Stat(srcPath); err != nil {
					continue
				}
				rel := filepath.Join("images", singleImg)
				ls.Images = append(ls.Images, rel)
				summary.TotalImages++

				if !dryRun {
					dst := filepath.Join(localeDst, rel)
					if err := copyFile(srcPath, dst); err != nil {
						summary.Warnings = append(summary.Warnings,
							fmt.Sprintf("[%s] failed to copy %s: %v", locale, rel, err))
					}
				}
			}
			sort.Strings(ls.Images)
		}

		summary.Locales = append(summary.Locales, ls)
	}

	return summary, nil
}

// copyFile copies a single file, creating parent directories as needed.
func copyFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}

// isImageFile checks if a filename has a common image extension.
func isImageFile(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	switch ext {
	case ".png", ".jpg", ".jpeg", ".webp":
		return true
	}
	return false
}
