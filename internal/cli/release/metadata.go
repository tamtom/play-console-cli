package release

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ListingData holds the parsed listing metadata for a single locale.
type ListingData struct {
	Title                   string
	TitlePresent            bool
	ShortDescription        string
	ShortDescriptionPresent bool
	FullDescription         string
	FullDescriptionPresent  bool
	Video                   string
	VideoPresent            bool
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

		if listing.hasFields() {
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

	title, present, err := readFileIfExists(filepath.Join(dir, "title.txt"))
	if err != nil {
		return listing, err
	}
	listing.Title = title
	listing.TitlePresent = present

	shortDesc, present, err := readFileIfExists(filepath.Join(dir, "short_description.txt"))
	if err != nil {
		return listing, err
	}
	listing.ShortDescription = shortDesc
	listing.ShortDescriptionPresent = present

	fullDesc, present, err := readFileIfExists(filepath.Join(dir, "full_description.txt"))
	if err != nil {
		return listing, err
	}
	listing.FullDescription = fullDesc
	listing.FullDescriptionPresent = present

	video, present, err := readFileIfExists(filepath.Join(dir, "video.txt"))
	if err != nil {
		return listing, err
	}
	listing.Video = video
	listing.VideoPresent = present

	return listing, nil
}

func (l ListingData) hasFields() bool {
	return l.TitlePresent || l.ShortDescriptionPresent || l.FullDescriptionPresent || l.VideoPresent
}

func (l ListingData) forceSendFields() []string {
	fields := make([]string, 0, 4)
	if l.TitlePresent {
		fields = append(fields, "Title")
	}
	if l.ShortDescriptionPresent {
		fields = append(fields, "ShortDescription")
	}
	if l.FullDescriptionPresent {
		fields = append(fields, "FullDescription")
	}
	if l.VideoPresent {
		fields = append(fields, "Video")
	}
	return fields
}

func readFileIfExists(path string) (string, bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", false, nil
		}
		return "", false, fmt.Errorf("failed to read %s: %w", filepath.Base(path), err)
	}
	return strings.TrimSpace(string(data)), true, nil
}
