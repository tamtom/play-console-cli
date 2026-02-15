package release

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ListingData holds the parsed listing metadata for a single locale.
type ListingData struct {
	Title            string
	ShortDescription string
	FullDescription  string
	Video            string
}

// ParseListingsDir reads a directory structured as:
//
//	<dir>/<locale>/title.txt
//	<dir>/<locale>/short_description.txt
//	<dir>/<locale>/full_description.txt
//	<dir>/<locale>/video.txt
//
// It returns a map from locale code to ListingData.
func ParseListingsDir(dir string) (map[string]ListingData, error) {
	info, err := os.Stat(dir)
	if err != nil {
		return nil, fmt.Errorf("listings directory not found: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("listings path is not a directory: %s", dir)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read listings directory: %w", err)
	}

	result := make(map[string]ListingData)

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		locale := entry.Name()
		localeDir := filepath.Join(dir, locale)

		listing, err := parseLocaleDir(localeDir)
		if err != nil {
			return nil, fmt.Errorf("failed to parse locale %s: %w", locale, err)
		}

		// Only include locales that have at least one non-empty field
		if listing.Title != "" || listing.ShortDescription != "" ||
			listing.FullDescription != "" || listing.Video != "" {
			result[locale] = listing
		}
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("no listing data found in directory: %s", dir)
	}

	return result, nil
}

func parseLocaleDir(dir string) (ListingData, error) {
	var listing ListingData

	title, err := readFileIfExists(filepath.Join(dir, "title.txt"))
	if err != nil {
		return listing, err
	}
	listing.Title = title

	shortDesc, err := readFileIfExists(filepath.Join(dir, "short_description.txt"))
	if err != nil {
		return listing, err
	}
	listing.ShortDescription = shortDesc

	fullDesc, err := readFileIfExists(filepath.Join(dir, "full_description.txt"))
	if err != nil {
		return listing, err
	}
	listing.FullDescription = fullDesc

	video, err := readFileIfExists(filepath.Join(dir, "video.txt"))
	if err != nil {
		return listing, err
	}
	listing.Video = video

	return listing, nil
}

func readFileIfExists(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("failed to read %s: %w", filepath.Base(path), err)
	}
	return strings.TrimSpace(string(data)), nil
}
