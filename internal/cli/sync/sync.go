package sync

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/peterbourgon/ff/v3/ffcli"
	"google.golang.org/api/androidpublisher/v3"

	"github.com/tamtom/play-console-cli/internal/cli/shared"
	"github.com/tamtom/play-console-cli/internal/playclient"
)

// FastLane metadata file names
const (
	titleFile            = "title.txt"
	shortDescFile        = "short_description.txt"
	fullDescFile         = "full_description.txt"
	videoFile            = "video.txt"
	changelogsDir        = "changelogs"
	imagesDir            = "images"
	phoneScreenshotsDir  = "phoneScreenshots"
	tablet7ScreensDir    = "sevenInchScreenshots"
	tablet10ScreensDir   = "tenInchScreenshots"
	tvScreenshotsDir     = "tvScreenshots"
	wearScreenshotsDir   = "wearScreenshots"
	featureGraphicFile   = "featureGraphic.png"
	iconFile             = "icon.png"
	promoGraphicFile     = "promoGraphic.png"
	tvBannerFile         = "tvBanner.png"
)

func SyncCommand() *ffcli.Command {
	fs := flag.NewFlagSet("sync", flag.ExitOnError)
	return &ffcli.Command{
		Name:       "sync",
		ShortUsage: "gplay sync <subcommand> [flags]",
		ShortHelp:  "Sync metadata between local directory and Play Store.",
		LongHelp: `Sync metadata in FastLane-compatible format.

Directory structure (FastLane format):
  metadata/
    en-US/
      title.txt
      short_description.txt
      full_description.txt
      video.txt
      changelogs/
        default.txt
        100.txt
      images/
        phoneScreenshots/
        sevenInchScreenshots/
        tenInchScreenshots/
        tvScreenshots/
        wearScreenshots/
        featureGraphic.png
        icon.png
        promoGraphic.png
        tvBanner.png
    de-DE/
      ...`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			ExportListingsCommand(),
			ImportListingsCommand(),
			ExportImagesCommand(),
			ImportImagesCommand(),
			DiffListingsCommand(),
		},
		Exec: func(ctx context.Context, args []string) error {
			if len(args) == 0 {
				return flag.ErrHelp
			}
			return flag.ErrHelp
		},
	}
}

func ExportListingsCommand() *ffcli.Command {
	fs := flag.NewFlagSet("sync export-listings", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	editID := fs.String("edit", "", "Edit ID (optional, creates temporary edit if not provided)")
	outputDir := fs.String("dir", "./metadata", "Output directory for metadata")
	format := fs.String("format", "fastlane", "Output format: fastlane (default), json")

	return &ffcli.Command{
		Name:       "export-listings",
		ShortUsage: "gplay sync export-listings --package <name> --dir <path> [--edit <id>]",
		ShortHelp:  "Export store listings to local directory.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			service, err := playclient.NewService(ctx)
			if err != nil {
				return err
			}
			pkg := shared.ResolvePackageName(*packageName, service.Cfg)
			if strings.TrimSpace(pkg) == "" {
				return fmt.Errorf("--package is required")
			}

			ctx, cancel := shared.ContextWithTimeout(ctx, service.Cfg)
			defer cancel()

			// Create or use edit
			var edit *androidpublisher.AppEdit
			var tempEdit bool
			if strings.TrimSpace(*editID) != "" {
				edit, err = service.API.Edits.Get(pkg, *editID).Context(ctx).Do()
				if err != nil {
					return fmt.Errorf("failed to get edit: %w", err)
				}
			} else {
				edit, err = service.API.Edits.Insert(pkg, &androidpublisher.AppEdit{}).Context(ctx).Do()
				if err != nil {
					return fmt.Errorf("failed to create edit: %w", err)
				}
				tempEdit = true
				defer func() {
					_ = service.API.Edits.Delete(pkg, edit.Id).Context(context.Background()).Do()
				}()
			}

			// Get all listings
			listingsResp, err := service.API.Edits.Listings.List(pkg, edit.Id).Context(ctx).Do()
			if err != nil {
				return fmt.Errorf("failed to list listings: %w", err)
			}

			// Create output directory
			if err := os.MkdirAll(*outputDir, 0755); err != nil {
				return fmt.Errorf("failed to create output directory: %w", err)
			}

			// Export each listing
			for _, listing := range listingsResp.Listings {
				localeDir := filepath.Join(*outputDir, listing.Language)
				if err := os.MkdirAll(localeDir, 0755); err != nil {
					return fmt.Errorf("failed to create locale directory: %w", err)
				}

				if *format == "json" {
					// Export as JSON
					data, err := json.MarshalIndent(listing, "", "  ")
					if err != nil {
						return fmt.Errorf("failed to marshal listing: %w", err)
					}
					if err := os.WriteFile(filepath.Join(localeDir, "listing.json"), data, 0644); err != nil {
						return fmt.Errorf("failed to write listing.json: %w", err)
					}
				} else {
					// Export as FastLane format
					if listing.Title != "" {
						if err := os.WriteFile(filepath.Join(localeDir, titleFile), []byte(listing.Title), 0644); err != nil {
							return fmt.Errorf("failed to write title: %w", err)
						}
					}
					if listing.ShortDescription != "" {
						if err := os.WriteFile(filepath.Join(localeDir, shortDescFile), []byte(listing.ShortDescription), 0644); err != nil {
							return fmt.Errorf("failed to write short description: %w", err)
						}
					}
					if listing.FullDescription != "" {
						if err := os.WriteFile(filepath.Join(localeDir, fullDescFile), []byte(listing.FullDescription), 0644); err != nil {
							return fmt.Errorf("failed to write full description: %w", err)
						}
					}
					if listing.Video != "" {
						if err := os.WriteFile(filepath.Join(localeDir, videoFile), []byte(listing.Video), 0644); err != nil {
							return fmt.Errorf("failed to write video: %w", err)
						}
					}
				}

				fmt.Fprintf(os.Stderr, "Exported: %s\n", listing.Language)
			}

			if tempEdit {
				fmt.Fprintf(os.Stderr, "Note: Used temporary edit (deleted automatically)\n")
			}

			fmt.Fprintf(os.Stderr, "Exported %d listings to %s\n", len(listingsResp.Listings), *outputDir)
			return nil
		},
	}
}

func ImportListingsCommand() *ffcli.Command {
	fs := flag.NewFlagSet("sync import-listings", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	editID := fs.String("edit", "", "Edit ID (required)")
	inputDir := fs.String("dir", "./metadata", "Input directory with metadata")
	format := fs.String("format", "fastlane", "Input format: fastlane (default), json")
	dryRun := fs.Bool("dry-run", false, "Show what would be imported without making changes")

	return &ffcli.Command{
		Name:       "import-listings",
		ShortUsage: "gplay sync import-listings --package <name> --edit <id> --dir <path> [--dry-run]",
		ShortHelp:  "Import store listings from local directory.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if strings.TrimSpace(*editID) == "" {
				return fmt.Errorf("--edit is required")
			}

			service, err := playclient.NewService(ctx)
			if err != nil {
				return err
			}
			pkg := shared.ResolvePackageName(*packageName, service.Cfg)
			if strings.TrimSpace(pkg) == "" {
				return fmt.Errorf("--package is required")
			}

			// Read locale directories
			entries, err := os.ReadDir(*inputDir)
			if err != nil {
				return fmt.Errorf("failed to read input directory: %w", err)
			}

			ctx, cancel := shared.ContextWithTimeout(ctx, service.Cfg)
			defer cancel()

			imported := 0
			for _, entry := range entries {
				if !entry.IsDir() {
					continue
				}
				locale := entry.Name()
				localeDir := filepath.Join(*inputDir, locale)

				var listing *androidpublisher.Listing

				if *format == "json" {
					// Read from JSON
					data, err := os.ReadFile(filepath.Join(localeDir, "listing.json"))
					if err != nil {
						if os.IsNotExist(err) {
							continue
						}
						return fmt.Errorf("failed to read listing.json for %s: %w", locale, err)
					}
					listing = &androidpublisher.Listing{}
					if err := json.Unmarshal(data, listing); err != nil {
						return fmt.Errorf("failed to parse listing.json for %s: %w", locale, err)
					}
				} else {
					// Read from FastLane format
					listing = &androidpublisher.Listing{}

					if data, err := os.ReadFile(filepath.Join(localeDir, titleFile)); err == nil {
						listing.Title = strings.TrimSpace(string(data))
					}
					if data, err := os.ReadFile(filepath.Join(localeDir, shortDescFile)); err == nil {
						listing.ShortDescription = strings.TrimSpace(string(data))
					}
					if data, err := os.ReadFile(filepath.Join(localeDir, fullDescFile)); err == nil {
						listing.FullDescription = strings.TrimSpace(string(data))
					}
					if data, err := os.ReadFile(filepath.Join(localeDir, videoFile)); err == nil {
						listing.Video = strings.TrimSpace(string(data))
					}

					// Skip if no content
					if listing.Title == "" && listing.ShortDescription == "" && listing.FullDescription == "" {
						continue
					}
				}

				if *dryRun {
					fmt.Fprintf(os.Stderr, "Would import: %s (title: %q)\n", locale, truncate(listing.Title, 30))
				} else {
					_, err := service.API.Edits.Listings.Update(pkg, *editID, locale, listing).Context(ctx).Do()
					if err != nil {
						return fmt.Errorf("failed to update listing for %s: %w", locale, err)
					}
					fmt.Fprintf(os.Stderr, "Imported: %s\n", locale)
				}
				imported++
			}

			if *dryRun {
				fmt.Fprintf(os.Stderr, "Dry run: would import %d listings\n", imported)
			} else {
				fmt.Fprintf(os.Stderr, "Imported %d listings\n", imported)
			}
			return nil
		},
	}
}

func ExportImagesCommand() *ffcli.Command {
	fs := flag.NewFlagSet("sync export-images", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	editID := fs.String("edit", "", "Edit ID (optional, creates temporary edit if not provided)")
	outputDir := fs.String("dir", "./metadata", "Output directory for images")
	locale := fs.String("locale", "", "Specific locale to export (optional, exports all if not specified)")

	return &ffcli.Command{
		Name:       "export-images",
		ShortUsage: "gplay sync export-images --package <name> --dir <path> [--edit <id>] [--locale <lang>]",
		ShortHelp:  "Export listing images to local directory.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			service, err := playclient.NewService(ctx)
			if err != nil {
				return err
			}
			pkg := shared.ResolvePackageName(*packageName, service.Cfg)
			if strings.TrimSpace(pkg) == "" {
				return fmt.Errorf("--package is required")
			}

			ctx, cancel := shared.ContextWithTimeout(ctx, service.Cfg)
			defer cancel()

			// Create or use edit
			var edit *androidpublisher.AppEdit
			var tempEdit bool
			if strings.TrimSpace(*editID) != "" {
				edit, err = service.API.Edits.Get(pkg, *editID).Context(ctx).Do()
				if err != nil {
					return fmt.Errorf("failed to get edit: %w", err)
				}
			} else {
				edit, err = service.API.Edits.Insert(pkg, &androidpublisher.AppEdit{}).Context(ctx).Do()
				if err != nil {
					return fmt.Errorf("failed to create edit: %w", err)
				}
				tempEdit = true
				defer func() {
					_ = service.API.Edits.Delete(pkg, edit.Id).Context(context.Background()).Do()
				}()
			}

			// Get locales to export
			var locales []string
			if strings.TrimSpace(*locale) != "" {
				locales = []string{*locale}
			} else {
				listingsResp, err := service.API.Edits.Listings.List(pkg, edit.Id).Context(ctx).Do()
				if err != nil {
					return fmt.Errorf("failed to list listings: %w", err)
				}
				for _, l := range listingsResp.Listings {
					locales = append(locales, l.Language)
				}
			}

			imageTypes := []string{
				"featureGraphic",
				"icon",
				"phoneScreenshots",
				"promoGraphic",
				"sevenInchScreenshots",
				"tenInchScreenshots",
				"tvBanner",
				"tvScreenshots",
				"wearScreenshots",
			}

			exported := 0
			for _, loc := range locales {
				for _, imageType := range imageTypes {
					images, err := service.API.Edits.Images.List(pkg, edit.Id, loc, imageType).Context(ctx).Do()
					if err != nil {
						continue // Skip if no images
					}

					if len(images.Images) == 0 {
						continue
					}

					// Create directory structure
					var targetDir string
					switch imageType {
					case "phoneScreenshots", "sevenInchScreenshots", "tenInchScreenshots", "tvScreenshots", "wearScreenshots":
						targetDir = filepath.Join(*outputDir, loc, imagesDir, imageType)
					default:
						targetDir = filepath.Join(*outputDir, loc, imagesDir)
					}

					if err := os.MkdirAll(targetDir, 0755); err != nil {
						return fmt.Errorf("failed to create directory: %w", err)
					}

					// Note: The API doesn't provide direct download URLs in the images list
					// We output metadata about the images instead
					metaFile := filepath.Join(targetDir, imageType+"_meta.json")
					data, err := json.MarshalIndent(images.Images, "", "  ")
					if err != nil {
						return fmt.Errorf("failed to marshal image metadata: %w", err)
					}
					if err := os.WriteFile(metaFile, data, 0644); err != nil {
						return fmt.Errorf("failed to write image metadata: %w", err)
					}

					exported += len(images.Images)
					fmt.Fprintf(os.Stderr, "Exported metadata for %d %s images in %s\n", len(images.Images), imageType, loc)
				}
			}

			if tempEdit {
				fmt.Fprintf(os.Stderr, "Note: Used temporary edit (deleted automatically)\n")
			}

			fmt.Fprintf(os.Stderr, "Exported metadata for %d images to %s\n", exported, *outputDir)
			fmt.Fprintf(os.Stderr, "Note: Image files must be downloaded manually from the Play Console\n")
			return nil
		},
	}
}

func ImportImagesCommand() *ffcli.Command {
	fs := flag.NewFlagSet("sync import-images", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	editID := fs.String("edit", "", "Edit ID (required)")
	inputDir := fs.String("dir", "./metadata", "Input directory with images")
	locale := fs.String("locale", "", "Specific locale to import (optional, imports all if not specified)")
	dryRun := fs.Bool("dry-run", false, "Show what would be imported without making changes")

	return &ffcli.Command{
		Name:       "import-images",
		ShortUsage: "gplay sync import-images --package <name> --edit <id> --dir <path> [--locale <lang>] [--dry-run]",
		ShortHelp:  "Import listing images from local directory.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if strings.TrimSpace(*editID) == "" {
				return fmt.Errorf("--edit is required")
			}

			service, err := playclient.NewService(ctx)
			if err != nil {
				return err
			}
			pkg := shared.ResolvePackageName(*packageName, service.Cfg)
			if strings.TrimSpace(pkg) == "" {
				return fmt.Errorf("--package is required")
			}

			// Get locales to import
			var locales []string
			if strings.TrimSpace(*locale) != "" {
				locales = []string{*locale}
			} else {
				entries, err := os.ReadDir(*inputDir)
				if err != nil {
					return fmt.Errorf("failed to read input directory: %w", err)
				}
				for _, entry := range entries {
					if entry.IsDir() {
						locales = append(locales, entry.Name())
					}
				}
			}

			ctx, cancel := shared.ContextWithUploadTimeout(ctx, service.Cfg)
			defer cancel()

			// Map FastLane directory names to API image types
			imageTypeMappings := map[string]string{
				phoneScreenshotsDir: "phoneScreenshots",
				tablet7ScreensDir:   "sevenInchScreenshots",
				tablet10ScreensDir:  "tenInchScreenshots",
				tvScreenshotsDir:    "tvScreenshots",
				wearScreenshotsDir:  "wearScreenshots",
			}

			singleImageMappings := map[string]string{
				featureGraphicFile: "featureGraphic",
				iconFile:           "icon",
				promoGraphicFile:   "promoGraphic",
				tvBannerFile:       "tvBanner",
			}

			imported := 0
			for _, loc := range locales {
				imagesPath := filepath.Join(*inputDir, loc, imagesDir)
				if _, err := os.Stat(imagesPath); os.IsNotExist(err) {
					continue
				}

				// Import screenshot directories
				for dirName, imageType := range imageTypeMappings {
					screenshotDir := filepath.Join(imagesPath, dirName)
					files, err := os.ReadDir(screenshotDir)
					if err != nil {
						continue
					}

					for _, file := range files {
						if file.IsDir() {
							continue
						}
						ext := strings.ToLower(filepath.Ext(file.Name()))
						if ext != ".png" && ext != ".jpg" && ext != ".jpeg" {
							continue
						}

						filePath := filepath.Join(screenshotDir, file.Name())
						if *dryRun {
							fmt.Fprintf(os.Stderr, "Would upload: %s -> %s/%s\n", filePath, loc, imageType)
						} else {
							if err := uploadImage(ctx, service, pkg, *editID, loc, imageType, filePath); err != nil {
								fmt.Fprintf(os.Stderr, "Warning: failed to upload %s: %v\n", filePath, err)
								continue
							}
							fmt.Fprintf(os.Stderr, "Uploaded: %s -> %s/%s\n", file.Name(), loc, imageType)
						}
						imported++
					}
				}

				// Import single images
				for fileName, imageType := range singleImageMappings {
					filePath := filepath.Join(imagesPath, fileName)
					if _, err := os.Stat(filePath); os.IsNotExist(err) {
						continue
					}

					if *dryRun {
						fmt.Fprintf(os.Stderr, "Would upload: %s -> %s/%s\n", filePath, loc, imageType)
					} else {
						if err := uploadImage(ctx, service, pkg, *editID, loc, imageType, filePath); err != nil {
							fmt.Fprintf(os.Stderr, "Warning: failed to upload %s: %v\n", filePath, err)
							continue
						}
						fmt.Fprintf(os.Stderr, "Uploaded: %s -> %s/%s\n", fileName, loc, imageType)
					}
					imported++
				}
			}

			if *dryRun {
				fmt.Fprintf(os.Stderr, "Dry run: would upload %d images\n", imported)
			} else {
				fmt.Fprintf(os.Stderr, "Uploaded %d images\n", imported)
			}
			return nil
		},
	}
}

func DiffListingsCommand() *ffcli.Command {
	fs := flag.NewFlagSet("sync diff-listings", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	editID := fs.String("edit", "", "Edit ID (optional, creates temporary edit if not provided)")
	localDir := fs.String("dir", "./metadata", "Local metadata directory")
	format := fs.String("format", "fastlane", "Local format: fastlane (default), json")

	return &ffcli.Command{
		Name:       "diff-listings",
		ShortUsage: "gplay sync diff-listings --package <name> --dir <path> [--edit <id>]",
		ShortHelp:  "Show differences between local and remote listings.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			service, err := playclient.NewService(ctx)
			if err != nil {
				return err
			}
			pkg := shared.ResolvePackageName(*packageName, service.Cfg)
			if strings.TrimSpace(pkg) == "" {
				return fmt.Errorf("--package is required")
			}

			ctx, cancel := shared.ContextWithTimeout(ctx, service.Cfg)
			defer cancel()

			// Create or use edit
			var edit *androidpublisher.AppEdit
			var tempEdit bool
			if strings.TrimSpace(*editID) != "" {
				edit, err = service.API.Edits.Get(pkg, *editID).Context(ctx).Do()
				if err != nil {
					return fmt.Errorf("failed to get edit: %w", err)
				}
			} else {
				edit, err = service.API.Edits.Insert(pkg, &androidpublisher.AppEdit{}).Context(ctx).Do()
				if err != nil {
					return fmt.Errorf("failed to create edit: %w", err)
				}
				tempEdit = true
				defer func() {
					_ = service.API.Edits.Delete(pkg, edit.Id).Context(context.Background()).Do()
				}()
			}

			// Get remote listings
			listingsResp, err := service.API.Edits.Listings.List(pkg, edit.Id).Context(ctx).Do()
			if err != nil {
				return fmt.Errorf("failed to list listings: %w", err)
			}

			remoteListings := make(map[string]*androidpublisher.Listing)
			for _, l := range listingsResp.Listings {
				remoteListings[l.Language] = l
			}

			// Read local listings
			localListings := make(map[string]*androidpublisher.Listing)
			entries, err := os.ReadDir(*localDir)
			if err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("failed to read local directory: %w", err)
			}

			for _, entry := range entries {
				if !entry.IsDir() {
					continue
				}
				locale := entry.Name()
				localeDir := filepath.Join(*localDir, locale)

				var listing *androidpublisher.Listing
				if *format == "json" {
					data, err := os.ReadFile(filepath.Join(localeDir, "listing.json"))
					if err != nil {
						continue
					}
					listing = &androidpublisher.Listing{}
					if err := json.Unmarshal(data, listing); err != nil {
						continue
					}
				} else {
					listing = &androidpublisher.Listing{Language: locale}
					if data, err := os.ReadFile(filepath.Join(localeDir, titleFile)); err == nil {
						listing.Title = strings.TrimSpace(string(data))
					}
					if data, err := os.ReadFile(filepath.Join(localeDir, shortDescFile)); err == nil {
						listing.ShortDescription = strings.TrimSpace(string(data))
					}
					if data, err := os.ReadFile(filepath.Join(localeDir, fullDescFile)); err == nil {
						listing.FullDescription = strings.TrimSpace(string(data))
					}
					if data, err := os.ReadFile(filepath.Join(localeDir, videoFile)); err == nil {
						listing.Video = strings.TrimSpace(string(data))
					}
				}
				localListings[locale] = listing
			}

			// Compare
			hasDiff := false

			// Check for locales only in remote
			for locale := range remoteListings {
				if _, ok := localListings[locale]; !ok {
					fmt.Printf("- %s (only in remote)\n", locale)
					hasDiff = true
				}
			}

			// Check for locales only in local
			for locale := range localListings {
				if _, ok := remoteListings[locale]; !ok {
					fmt.Printf("+ %s (only in local)\n", locale)
					hasDiff = true
				}
			}

			// Check for differences in shared locales
			for locale, local := range localListings {
				remote, ok := remoteListings[locale]
				if !ok {
					continue
				}

				diffs := []string{}
				if local.Title != remote.Title {
					diffs = append(diffs, fmt.Sprintf("title: %q -> %q", truncate(remote.Title, 20), truncate(local.Title, 20)))
				}
				if local.ShortDescription != remote.ShortDescription {
					diffs = append(diffs, "short_description changed")
				}
				if local.FullDescription != remote.FullDescription {
					diffs = append(diffs, "full_description changed")
				}
				if local.Video != remote.Video {
					diffs = append(diffs, fmt.Sprintf("video: %q -> %q", remote.Video, local.Video))
				}

				if len(diffs) > 0 {
					fmt.Printf("~ %s: %s\n", locale, strings.Join(diffs, ", "))
					hasDiff = true
				}
			}

			if !hasDiff {
				fmt.Println("No differences found")
			}

			if tempEdit {
				fmt.Fprintf(os.Stderr, "\nNote: Used temporary edit (deleted automatically)\n")
			}

			return nil
		},
	}
}

func uploadImage(ctx context.Context, service *playclient.Service, pkg, editID, locale, imageType, filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return shared.WrapActionable(err, "failed to open image file", "Check that the file exists and is readable.")
	}
	defer file.Close()

	call := service.API.Edits.Images.Upload(pkg, editID, locale, imageType)
	call.Media(file)
	_, err = call.Context(ctx).Do()
	if err != nil {
		return shared.WrapGoogleAPIError("failed to upload image", err)
	}
	return nil
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
