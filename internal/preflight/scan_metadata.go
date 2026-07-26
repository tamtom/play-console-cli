package preflight

import (
	"fmt"
	"image"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"

	// Registered for image.DecodeConfig so screenshot dimensions can be read
	// without decoding whole images.
	_ "image/jpeg"
	_ "image/png"

	"github.com/tamtom/play-console-cli/internal/validation"
)

// Play store listing asset rules.
const (
	minScreenshotSide  = 320
	maxScreenshotSide  = 3840
	maxScreenshotRatio = 2.0
	minPhoneScreenshot = 2
	maxScreenshotCount = 8
	maxScreenshotBytes = 8 * 1024 * 1024
	maxIconBytes       = 1024 * 1024
	maxGraphicBytes    = 15 * 1024 * 1024
)

// fixedSizeImages maps a single-image filename to its required dimensions.
var fixedSizeImages = map[string]struct {
	W, H     int
	MaxBytes int64
}{
	"icon.png":           {512, 512, maxIconBytes},
	"featureGraphic.png": {1024, 500, maxGraphicBytes},
	"promoGraphic.png":   {180, 120, maxGraphicBytes},
	"tvBanner.png":       {1280, 720, maxGraphicBytes},
}

// screenshotDirs are the per-form-factor screenshot directories.
var screenshotDirs = []string{
	"phoneScreenshots",
	"sevenInchScreenshots",
	"tenInchScreenshots",
	"tvScreenshots",
	"wearScreenshots",
}

// scanMetadata validates a Fastlane-style listings directory offline: text
// field lengths and, uniquely, real screenshot pixel dimensions.
func scanMetadata(c *scanContext) []Finding {
	dir := c.opts.ListingsDir
	entries, err := os.ReadDir(dir)
	if err != nil {
		return []Finding{{
			Check:    "listings_dir",
			Severity: SeverityError,
			Message:  fmt.Sprintf("cannot read listings directory: %v", err),
			Entry:    dir,
		}}
	}

	var locales []string
	for _, e := range entries {
		if e.IsDir() {
			locales = append(locales, e.Name())
		}
	}
	sort.Strings(locales)

	if len(locales) == 0 {
		return []Finding{{
			Check:    "listings_dir",
			Severity: SeverityError,
			Message:  "listings directory contains no locale subdirectories",
			Entry:    dir,
			Hint:     "expected a layout like <dir>/en-US/title.txt",
		}}
	}

	var out []Finding
	for _, locale := range locales {
		localeDir := filepath.Join(dir, locale)
		out = append(out, checkListingText(locale, localeDir)...)
		out = append(out, checkListingImages(locale, filepath.Join(localeDir, "images"))...)
	}
	return out
}

// listingTextField describes one validated text file in a locale directory.
type listingTextField struct {
	file     string
	label    string
	max      int
	required bool
}

// checkListingText validates the text assets for one locale.
func checkListingText(locale, localeDir string) []Finding {
	fields := []listingTextField{
		{"title.txt", "title", validation.MaxTitleLength, true},
		{"short_description.txt", "short description", validation.MaxShortDescriptionLength, true},
		{"full_description.txt", "full description", validation.MaxFullDescriptionLength, true},
	}

	var out []Finding
	for _, f := range fields {
		path := filepath.Join(localeDir, f.file)
		data, err := os.ReadFile(path) // #nosec G304 -- path derived from the user-supplied listings dir
		if err != nil {
			if os.IsNotExist(err) && f.required {
				out = append(out, Finding{
					Check:    "listing_text_missing",
					Severity: SeverityError,
					Message:  fmt.Sprintf("[%s] %s is missing", locale, f.file),
					Entry:    path,
				})
			}
			continue
		}
		text := strings.TrimSpace(string(data))
		if text == "" {
			out = append(out, Finding{
				Check:    "listing_text_empty",
				Severity: SeverityError,
				Message:  fmt.Sprintf("[%s] %s is empty", locale, f.label),
				Entry:    path,
			})
			continue
		}
		if n := utf8.RuneCountInString(text); n > f.max {
			out = append(out, Finding{
				Check:    "listing_text_length",
				Severity: SeverityError,
				Message:  fmt.Sprintf("[%s] %s is %d characters, over the %d limit", locale, f.label, n, f.max),
				Entry:    path,
				Hint:     fmt.Sprintf("trim to %d characters or fewer", f.max),
			})
		}
	}

	out = append(out, checkChangelogs(locale, filepath.Join(localeDir, "changelogs"))...)
	return out
}

// checkChangelogs validates release-note files for one locale.
func checkChangelogs(locale, dir string) []Finding {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil // changelogs are optional
	}
	var out []Finding
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".txt") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		data, err := os.ReadFile(path) // #nosec G304 -- path derived from the user-supplied listings dir
		if err != nil {
			continue
		}
		if n := utf8.RuneCountInString(strings.TrimSpace(string(data))); n > validation.MaxReleaseNotesLength {
			out = append(out, Finding{
				Check:    "listing_release_notes",
				Severity: SeverityError,
				Message:  fmt.Sprintf("[%s] release notes are %d characters, over the %d limit", locale, n, validation.MaxReleaseNotesLength),
				Entry:    path,
			})
		}
	}
	return out
}

// checkListingImages validates icons, graphics, and screenshots for a locale.
func checkListingImages(locale, imagesDir string) []Finding {
	if _, err := os.Stat(imagesDir); err != nil {
		return nil // a locale may legitimately ship text only
	}

	var out []Finding
	for name, spec := range fixedSizeImages {
		path := filepath.Join(imagesDir, name)
		info, err := os.Stat(path)
		if err != nil {
			continue
		}
		cfg, format, err := decodeImageConfig(path)
		if err != nil {
			out = append(out, Finding{
				Check:    "image_unreadable",
				Severity: SeverityError,
				Message:  fmt.Sprintf("[%s] %s could not be decoded: %v", locale, name, err),
				Entry:    path,
				Hint:     "Play accepts 24-bit PNG or JPEG assets",
			})
			continue
		}
		if cfg.Width != spec.W || cfg.Height != spec.H {
			out = append(out, Finding{
				Check:    "image_dimensions",
				Severity: SeverityError,
				Message:  fmt.Sprintf("[%s] %s is %dx%d, must be %dx%d", locale, name, cfg.Width, cfg.Height, spec.W, spec.H),
				Entry:    path,
			})
		}
		if info.Size() > spec.MaxBytes {
			out = append(out, Finding{
				Check:    "image_size",
				Severity: SeverityError,
				Message:  fmt.Sprintf("[%s] %s is %d bytes, over the %d limit", locale, name, info.Size(), spec.MaxBytes),
				Entry:    path,
			})
		}
		if name == "icon.png" && format != "png" {
			out = append(out, Finding{
				Check:    "image_format",
				Severity: SeverityError,
				Message:  fmt.Sprintf("[%s] icon must be a PNG, found %s", locale, format),
				Entry:    path,
			})
		}
	}

	for _, sd := range screenshotDirs {
		out = append(out, checkScreenshotDir(locale, sd, filepath.Join(imagesDir, sd))...)
	}
	return out
}

// checkScreenshotDir validates one screenshot form factor.
func checkScreenshotDir(locale, kind, dir string) []Finding {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if kind == "phoneScreenshots" {
			return []Finding{{
				Check:    "screenshot_count",
				Severity: SeverityError,
				Message:  fmt.Sprintf("[%s] no phone screenshots found", locale),
				Entry:    dir,
				Hint:     fmt.Sprintf("Play requires at least %d phone screenshots", minPhoneScreenshot),
			}}
		}
		return nil
	}

	var out []Finding
	var images []string
	for _, e := range entries {
		if e.IsDir() || !isImageFile(e.Name()) {
			continue
		}
		images = append(images, filepath.Join(dir, e.Name()))
	}
	sort.Strings(images)

	if kind == "phoneScreenshots" && len(images) < minPhoneScreenshot {
		out = append(out, Finding{
			Check:    "screenshot_count",
			Severity: SeverityError,
			Message:  fmt.Sprintf("[%s] %d phone screenshot(s); Play requires at least %d", locale, len(images), minPhoneScreenshot),
			Entry:    dir,
		})
	}
	if len(images) > maxScreenshotCount {
		out = append(out, Finding{
			Check:    "screenshot_count",
			Severity: SeverityWarning,
			Message:  fmt.Sprintf("[%s] %d %s exceed the %d Play accepts; extras are ignored", locale, len(images), kind, maxScreenshotCount),
			Entry:    dir,
		})
	}

	for _, path := range images {
		out = append(out, checkScreenshotFile(locale, path)...)
	}
	return out
}

// checkScreenshotFile validates one screenshot's dimensions and size.
func checkScreenshotFile(locale, path string) []Finding {
	var out []Finding

	info, err := os.Stat(path)
	if err == nil && info.Size() > maxScreenshotBytes {
		out = append(out, Finding{
			Check:    "image_size",
			Severity: SeverityError,
			Message:  fmt.Sprintf("[%s] %s is %d bytes, over the %d limit", locale, filepath.Base(path), info.Size(), maxScreenshotBytes),
			Entry:    path,
		})
	}

	cfg, _, err := decodeImageConfig(path)
	if err != nil {
		return append(out, Finding{
			Check:    "image_unreadable",
			Severity: SeverityError,
			Message:  fmt.Sprintf("[%s] %s could not be decoded: %v", locale, filepath.Base(path), err),
			Entry:    path,
			Hint:     "Play accepts PNG and JPEG screenshots",
		})
	}

	short, long := cfg.Width, cfg.Height
	if short > long {
		short, long = long, short
	}
	if short < minScreenshotSide {
		out = append(out, Finding{
			Check:    "image_dimensions",
			Severity: SeverityError,
			Message:  fmt.Sprintf("[%s] %s is %dx%d; each side must be at least %dpx", locale, filepath.Base(path), cfg.Width, cfg.Height, minScreenshotSide),
			Entry:    path,
		})
	}
	if long > maxScreenshotSide {
		out = append(out, Finding{
			Check:    "image_dimensions",
			Severity: SeverityError,
			Message:  fmt.Sprintf("[%s] %s is %dx%d; no side may exceed %dpx", locale, filepath.Base(path), cfg.Width, cfg.Height, maxScreenshotSide),
			Entry:    path,
		})
	}
	if short > 0 && float64(long)/float64(short) > maxScreenshotRatio {
		out = append(out, Finding{
			Check:    "image_ratio",
			Severity: SeverityError,
			Message: fmt.Sprintf("[%s] %s has a %.2f:1 aspect ratio; Play allows at most %.0f:1",
				locale, filepath.Base(path), float64(long)/float64(short), maxScreenshotRatio),
			Entry: path,
		})
	}
	return out
}

// decodeImageConfig reads image dimensions without decoding pixel data.
func decodeImageConfig(path string) (image.Config, string, error) {
	f, err := os.Open(path) // #nosec G304 -- path derived from the user-supplied listings dir
	if err != nil {
		return image.Config{}, "", err
	}
	defer func() { _ = f.Close() }()
	return image.DecodeConfig(f)
}

// isImageFile reports whether a filename has a Play-supported image extension.
func isImageFile(name string) bool {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".png", ".jpg", ".jpeg":
		return true
	}
	return false
}
