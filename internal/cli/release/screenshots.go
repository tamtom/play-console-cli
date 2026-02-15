package release

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ValidScreenshotTypes lists the image types accepted by the Google Play API.
var ValidScreenshotTypes = map[string]bool{
	"phoneScreenshots":        true,
	"sevenInchScreenshots":    true,
	"tenInchScreenshots":      true,
	"tvScreenshots":           true,
	"wearScreenshots":         true,
	"chromebookScreenshots":   true,
}

// ParseScreenshotsDir reads a directory structured as:
//
//	<dir>/<locale>/<deviceType>/<file.png>
//
// It returns map[locale]map[deviceType][]filePaths.
// File paths are sorted alphabetically within each device type.
func ParseScreenshotsDir(dir string) (map[string]map[string][]string, error) {
	info, err := os.Stat(dir)
	if err != nil {
		return nil, fmt.Errorf("screenshots directory not found: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("screenshots path is not a directory: %s", dir)
	}

	localeEntries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read screenshots directory: %w", err)
	}

	result := make(map[string]map[string][]string)

	for _, localeEntry := range localeEntries {
		if !localeEntry.IsDir() {
			continue
		}

		locale := localeEntry.Name()
		localeDir := filepath.Join(dir, locale)

		deviceEntries, err := os.ReadDir(localeDir)
		if err != nil {
			return nil, fmt.Errorf("failed to read locale directory %s: %w", locale, err)
		}

		deviceMap := make(map[string][]string)

		for _, deviceEntry := range deviceEntries {
			if !deviceEntry.IsDir() {
				continue
			}

			deviceType := deviceEntry.Name()
			if !ValidScreenshotTypes[deviceType] {
				return nil, fmt.Errorf("unknown screenshot type %q in locale %s; valid types: %s",
					deviceType, locale, validTypesString())
			}

			deviceDir := filepath.Join(localeDir, deviceType)
			files, err := os.ReadDir(deviceDir)
			if err != nil {
				return nil, fmt.Errorf("failed to read device directory %s/%s: %w", locale, deviceType, err)
			}

			var filePaths []string
			for _, f := range files {
				if f.IsDir() {
					continue
				}
				ext := strings.ToLower(filepath.Ext(f.Name()))
				if ext == ".png" || ext == ".jpg" || ext == ".jpeg" || ext == ".webp" {
					filePaths = append(filePaths, filepath.Join(deviceDir, f.Name()))
				}
			}

			sort.Strings(filePaths)

			if len(filePaths) > 0 {
				deviceMap[deviceType] = filePaths
			}
		}

		if len(deviceMap) > 0 {
			result[locale] = deviceMap
		}
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("no screenshots found in directory: %s", dir)
	}

	return result, nil
}

func validTypesString() string {
	types := make([]string, 0, len(ValidScreenshotTypes))
	for t := range ValidScreenshotTypes {
		types = append(types, t)
	}
	sort.Strings(types)
	return strings.Join(types, ", ")
}
