package metadata

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/tamtom/play-console-cli/internal/cli/shared"
)

// Google Play metadata character limits.
const (
	maxTitleLen            = 30
	maxShortDescriptionLen = 80
	maxFullDescriptionLen  = 4000
)

// validationError represents a single metadata validation issue.
type validationError struct {
	Locale  string `json:"locale"`
	Field   string `json:"field"`
	Message string `json:"message"`
	Length  int    `json:"length,omitempty"`
	Limit   int    `json:"limit,omitempty"`
}

// validationResult holds all validation results.
type validationResult struct {
	Dir        string            `json:"dir"`
	Locales    int               `json:"locales"`
	Errors     []validationError `json:"errors,omitempty"`
	ErrorCount int               `json:"errorCount"`
	Valid      bool              `json:"valid"`
}

// ValidateCommand returns the metadata validate subcommand.
func ValidateCommand() *ffcli.Command {
	fs := flag.NewFlagSet("metadata validate", flag.ExitOnError)
	dir := fs.String("dir", "", "Metadata directory to validate (required)")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "validate",
		ShortUsage: "gplay metadata validate --dir <path>",
		ShortHelp:  "Validate metadata files offline.",
		LongHelp: `Validate local metadata files against Google Play character limits.

Checks:
  - title.txt: max 30 characters
  - short_description.txt: max 80 characters
  - full_description.txt: max 4000 characters

Reports errors per locale per field. Exits non-zero if validation errors found.

Examples:
  gplay metadata validate --dir ./metadata`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}

			dirValue := strings.TrimSpace(*dir)
			if dirValue == "" {
				return fmt.Errorf("--dir is required")
			}

			result, err := validateMetadata(dirValue)
			if err != nil {
				return err
			}

			if printErr := shared.PrintOutput(result, *outputFlag, *pretty); printErr != nil {
				return printErr
			}

			if !result.Valid {
				return fmt.Errorf("metadata validation failed: %d error(s) found", result.ErrorCount)
			}
			return nil
		},
	}
}

// validateMetadata validates all metadata files in the given directory.
func validateMetadata(dir string) (*validationResult, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory %s: %w", dir, err)
	}

	result := &validationResult{
		Dir:    dir,
		Errors: []validationError{},
		Valid:  true,
	}

	var locales []string
	for _, entry := range entries {
		if entry.IsDir() {
			locales = append(locales, entry.Name())
		}
	}
	sort.Strings(locales)

	if len(locales) == 0 {
		return nil, fmt.Errorf("no locale directories found in %s", dir)
	}

	result.Locales = len(locales)

	for _, locale := range locales {
		localeDir := filepath.Join(dir, locale)
		errors := validateLocaleMetadata(locale, localeDir)
		result.Errors = append(result.Errors, errors...)
	}

	result.ErrorCount = len(result.Errors)
	result.Valid = result.ErrorCount == 0
	return result, nil
}

// validateLocaleMetadata validates metadata files for a single locale.
func validateLocaleMetadata(locale, localeDir string) []validationError {
	var errors []validationError

	errors = append(errors, validateFieldLength(locale, localeDir, "title.txt", "title", maxTitleLen)...)
	errors = append(errors, validateFieldLength(locale, localeDir, "short_description.txt", "short_description", maxShortDescriptionLen)...)
	errors = append(errors, validateFieldLength(locale, localeDir, "full_description.txt", "full_description", maxFullDescriptionLen)...)

	return errors
}

// validateFieldLength checks that a metadata file does not exceed the character limit.
func validateFieldLength(locale, localeDir, filename, fieldName string, maxLen int) []validationError {
	path := filepath.Join(localeDir, filename)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return []validationError{{
			Locale:  locale,
			Field:   fieldName,
			Message: fmt.Sprintf("failed to read %s: %v", filename, err),
		}}
	}

	content := strings.TrimSpace(string(data))
	length := utf8.RuneCountInString(content)

	if length > maxLen {
		return []validationError{{
			Locale:  locale,
			Field:   fieldName,
			Message: fmt.Sprintf("%s exceeds %d characters (%d/%d)", fieldName, maxLen, length, maxLen),
			Length:  length,
			Limit:   maxLen,
		}}
	}

	return nil
}
