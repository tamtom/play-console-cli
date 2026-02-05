package validate

import (
	"archive/zip"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/tamtom/play-console-cli/internal/cli/shared"
)

// Validation limits from Google Play
const (
	maxTitleLength            = 30
	maxShortDescriptionLength = 80
	maxFullDescriptionLength  = 4000
	maxPhoneScreenshots       = 8
	maxTabletScreenshots      = 8
	maxTVScreenshots          = 8
	maxWearScreenshots        = 8
	minScreenshots            = 2
)

func ValidateCommand() *ffcli.Command {
	fs := flag.NewFlagSet("validate", flag.ExitOnError)
	return &ffcli.Command{
		Name:       "validate",
		ShortUsage: "gplay validate <subcommand> [flags]",
		ShortHelp:  "Pre-flight validation commands.",
		LongHelp: `Validate resources before uploading to Google Play.

These commands perform local validation to catch common issues
before making API calls, saving time and reducing errors.`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			BundleCommand(),
			ListingCommand(),
			ScreenshotsCommand(),
		},
		Exec: func(ctx context.Context, args []string) error {
			if len(args) == 0 {
				return flag.ErrHelp
			}
			return flag.ErrHelp
		},
	}
}

func BundleCommand() *ffcli.Command {
	fs := flag.NewFlagSet("validate bundle", flag.ExitOnError)
	filePath := fs.String("file", "", "Path to .aab bundle file")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "bundle",
		ShortUsage: "gplay validate bundle --file <path>",
		ShortHelp:  "Validate an app bundle before upload.",
		LongHelp: `Validate an Android App Bundle (.aab) file.

Checks:
- File exists and is readable
- File has .aab extension
- File is a valid ZIP archive
- Contains required bundle components`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			if strings.TrimSpace(*filePath) == "" {
				return fmt.Errorf("--file is required")
			}

			result := validateBundle(*filePath)
			return shared.PrintOutput(result, *outputFlag, *pretty)
		},
	}
}

func ListingCommand() *ffcli.Command {
	fs := flag.NewFlagSet("validate listing", flag.ExitOnError)
	dir := fs.String("dir", "./metadata", "Directory containing listing metadata")
	locale := fs.String("locale", "", "Specific locale to validate (optional)")
	format := fs.String("format", "fastlane", "Metadata format: fastlane (default), json")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "listing",
		ShortUsage: "gplay validate listing --dir <path> [--locale <lang>]",
		ShortHelp:  "Validate store listing metadata.",
		LongHelp: `Validate store listing metadata files.

Checks:
- Title length (max 30 characters)
- Short description length (max 80 characters)
- Full description length (max 4000 characters)
- Required fields present
- Valid UTF-8 encoding`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}

			result := validateListings(*dir, *locale, *format)
			return shared.PrintOutput(result, *outputFlag, *pretty)
		},
	}
}

func ScreenshotsCommand() *ffcli.Command {
	fs := flag.NewFlagSet("validate screenshots", flag.ExitOnError)
	dir := fs.String("dir", "./metadata", "Directory containing screenshots")
	locale := fs.String("locale", "", "Specific locale to validate (optional)")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "screenshots",
		ShortUsage: "gplay validate screenshots --dir <path> [--locale <lang>]",
		ShortHelp:  "Validate screenshot images.",
		LongHelp: `Validate screenshot images for store listings.

Checks:
- Minimum 2 screenshots required per device type
- Maximum 8 screenshots per device type
- Valid image formats (PNG, JPEG)
- File is readable`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}

			result := validateScreenshots(*dir, *locale)
			return shared.PrintOutput(result, *outputFlag, *pretty)
		},
	}
}

type ValidationResult struct {
	Valid    bool              `json:"valid"`
	Errors   []string          `json:"errors,omitempty"`
	Warnings []string          `json:"warnings,omitempty"`
	Details  map[string]interface{} `json:"details,omitempty"`
}

func validateBundle(filePath string) *ValidationResult {
	result := &ValidationResult{
		Valid:   true,
		Details: make(map[string]interface{}),
	}

	// Check file exists
	info, err := os.Stat(filePath)
	if err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("File not found: %s", filePath))
		return result
	}

	result.Details["fileName"] = filepath.Base(filePath)
	result.Details["fileSize"] = info.Size()
	result.Details["fileSizeHuman"] = formatBytes(info.Size())

	// Check extension
	if !strings.HasSuffix(strings.ToLower(filePath), ".aab") {
		result.Warnings = append(result.Warnings, "File does not have .aab extension")
	}

	// Check it's a valid ZIP
	reader, err := zip.OpenReader(filePath)
	if err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("Not a valid ZIP archive: %v", err))
		return result
	}
	defer reader.Close()

	// Check for required bundle components
	requiredFiles := map[string]bool{
		"BundleConfig.pb": false,
		"base/":           false,
	}

	for _, file := range reader.File {
		for required := range requiredFiles {
			if strings.HasPrefix(file.Name, required) || file.Name == required {
				requiredFiles[required] = true
			}
		}
	}

	for required, found := range requiredFiles {
		if !found {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("Missing required component: %s", required))
		}
	}

	result.Details["fileCount"] = len(reader.File)

	return result
}

func validateListings(dir, locale, format string) *ValidationResult {
	result := &ValidationResult{
		Valid:   true,
		Details: make(map[string]interface{}),
	}

	localeResults := make(map[string]interface{})

	var locales []string
	if locale != "" {
		locales = []string{locale}
	} else {
		entries, err := os.ReadDir(dir)
		if err != nil {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("Cannot read directory: %v", err))
			return result
		}
		for _, entry := range entries {
			if entry.IsDir() {
				locales = append(locales, entry.Name())
			}
		}
	}

	for _, loc := range locales {
		localeDir := filepath.Join(dir, loc)
		locResult := validateLocaleListing(localeDir, format)
		localeResults[loc] = locResult

		if !locResult.Valid {
			result.Valid = false
			for _, err := range locResult.Errors {
				result.Errors = append(result.Errors, fmt.Sprintf("[%s] %s", loc, err))
			}
		}
		for _, warn := range locResult.Warnings {
			result.Warnings = append(result.Warnings, fmt.Sprintf("[%s] %s", loc, warn))
		}
	}

	result.Details["locales"] = localeResults
	result.Details["localeCount"] = len(locales)

	return result
}

type LocaleValidationResult struct {
	Valid    bool     `json:"valid"`
	Errors   []string `json:"errors,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
	Title    *FieldValidation `json:"title,omitempty"`
	ShortDescription *FieldValidation `json:"shortDescription,omitempty"`
	FullDescription  *FieldValidation `json:"fullDescription,omitempty"`
}

type FieldValidation struct {
	Present bool `json:"present"`
	Length  int  `json:"length"`
	MaxLength int `json:"maxLength"`
	Valid   bool `json:"valid"`
}

func validateLocaleListing(localeDir, format string) *LocaleValidationResult {
	result := &LocaleValidationResult{Valid: true}

	if format == "json" {
		// Validate JSON format
		data, err := os.ReadFile(filepath.Join(localeDir, "listing.json"))
		if err != nil {
			if !os.IsNotExist(err) {
				result.Valid = false
				result.Errors = append(result.Errors, fmt.Sprintf("Cannot read listing.json: %v", err))
			}
			return result
		}

		var listing map[string]interface{}
		if err := json.Unmarshal(data, &listing); err != nil {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("Invalid JSON: %v", err))
			return result
		}

		// Validate fields from JSON
		if title, ok := listing["title"].(string); ok {
			result.Title = validateField(title, maxTitleLength, true)
			if !result.Title.Valid {
				result.Valid = false
				result.Errors = append(result.Errors, fmt.Sprintf("Title too long: %d/%d characters", result.Title.Length, maxTitleLength))
			}
		}

		if shortDesc, ok := listing["shortDescription"].(string); ok {
			result.ShortDescription = validateField(shortDesc, maxShortDescriptionLength, true)
			if !result.ShortDescription.Valid {
				result.Valid = false
				result.Errors = append(result.Errors, fmt.Sprintf("Short description too long: %d/%d characters", result.ShortDescription.Length, maxShortDescriptionLength))
			}
		}

		if fullDesc, ok := listing["fullDescription"].(string); ok {
			result.FullDescription = validateField(fullDesc, maxFullDescriptionLength, false)
			if !result.FullDescription.Valid {
				result.Valid = false
				result.Errors = append(result.Errors, fmt.Sprintf("Full description too long: %d/%d characters", result.FullDescription.Length, maxFullDescriptionLength))
			}
		}

		return result
	}

	// Validate FastLane format
	titlePath := filepath.Join(localeDir, "title.txt")
	shortDescPath := filepath.Join(localeDir, "short_description.txt")
	fullDescPath := filepath.Join(localeDir, "full_description.txt")

	// Title
	if titleData, err := os.ReadFile(titlePath); err == nil {
		title := strings.TrimSpace(string(titleData))
		if !utf8.ValidString(title) {
			result.Valid = false
			result.Errors = append(result.Errors, "Title contains invalid UTF-8")
		}
		result.Title = validateField(title, maxTitleLength, true)
		if !result.Title.Valid {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("Title too long: %d/%d characters", result.Title.Length, maxTitleLength))
		}
	} else if !os.IsNotExist(err) {
		result.Warnings = append(result.Warnings, "Cannot read title.txt")
	}

	// Short description
	if shortData, err := os.ReadFile(shortDescPath); err == nil {
		shortDesc := strings.TrimSpace(string(shortData))
		if !utf8.ValidString(shortDesc) {
			result.Valid = false
			result.Errors = append(result.Errors, "Short description contains invalid UTF-8")
		}
		result.ShortDescription = validateField(shortDesc, maxShortDescriptionLength, true)
		if !result.ShortDescription.Valid {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("Short description too long: %d/%d characters", result.ShortDescription.Length, maxShortDescriptionLength))
		}
	} else if !os.IsNotExist(err) {
		result.Warnings = append(result.Warnings, "Cannot read short_description.txt")
	}

	// Full description
	if fullData, err := os.ReadFile(fullDescPath); err == nil {
		fullDesc := strings.TrimSpace(string(fullData))
		if !utf8.ValidString(fullDesc) {
			result.Valid = false
			result.Errors = append(result.Errors, "Full description contains invalid UTF-8")
		}
		result.FullDescription = validateField(fullDesc, maxFullDescriptionLength, false)
		if !result.FullDescription.Valid {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("Full description too long: %d/%d characters", result.FullDescription.Length, maxFullDescriptionLength))
		}
	} else if !os.IsNotExist(err) {
		result.Warnings = append(result.Warnings, "Cannot read full_description.txt")
	}

	return result
}

func validateField(content string, maxLength int, required bool) *FieldValidation {
	length := utf8.RuneCountInString(content)
	return &FieldValidation{
		Present:   content != "",
		Length:    length,
		MaxLength: maxLength,
		Valid:     length <= maxLength && (!required || content != ""),
	}
}

func validateScreenshots(dir, locale string) *ValidationResult {
	result := &ValidationResult{
		Valid:   true,
		Details: make(map[string]interface{}),
	}

	var locales []string
	if locale != "" {
		locales = []string{locale}
	} else {
		entries, err := os.ReadDir(dir)
		if err != nil {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("Cannot read directory: %v", err))
			return result
		}
		for _, entry := range entries {
			if entry.IsDir() {
				locales = append(locales, entry.Name())
			}
		}
	}

	localeResults := make(map[string]interface{})

	screenshotDirs := map[string]int{
		"phoneScreenshots":      maxPhoneScreenshots,
		"sevenInchScreenshots":  maxTabletScreenshots,
		"tenInchScreenshots":    maxTabletScreenshots,
		"tvScreenshots":         maxTVScreenshots,
		"wearScreenshots":       maxWearScreenshots,
	}

	for _, loc := range locales {
		locResult := map[string]interface{}{}
		imagesDir := filepath.Join(dir, loc, "images")

		for screenshotDir, maxCount := range screenshotDirs {
			fullPath := filepath.Join(imagesDir, screenshotDir)
			files, err := os.ReadDir(fullPath)
			if err != nil {
				if !os.IsNotExist(err) {
					result.Warnings = append(result.Warnings, fmt.Sprintf("[%s] Cannot read %s: %v", loc, screenshotDir, err))
				}
				continue
			}

			// Count valid image files
			imageCount := 0
			for _, file := range files {
				if file.IsDir() {
					continue
				}
				ext := strings.ToLower(filepath.Ext(file.Name()))
				if ext == ".png" || ext == ".jpg" || ext == ".jpeg" {
					imageCount++
				}
			}

			dirResult := map[string]interface{}{
				"count":    imageCount,
				"maxCount": maxCount,
			}

			if imageCount > 0 && imageCount < minScreenshots {
				result.Warnings = append(result.Warnings, fmt.Sprintf("[%s] %s has %d screenshots (minimum recommended: %d)", loc, screenshotDir, imageCount, minScreenshots))
			}

			if imageCount > maxCount {
				result.Valid = false
				result.Errors = append(result.Errors, fmt.Sprintf("[%s] %s has %d screenshots (maximum: %d)", loc, screenshotDir, imageCount, maxCount))
				dirResult["valid"] = false
			} else {
				dirResult["valid"] = true
			}

			locResult[screenshotDir] = dirResult
		}

		localeResults[loc] = locResult
	}

	result.Details["locales"] = localeResults

	return result
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
