package validation

import (
	"fmt"
	"unicode/utf8"
)

// Google Play metadata character limits.
const (
	MaxTitleLength            = 30
	MaxShortDescriptionLength = 80
	MaxFullDescriptionLength  = 4000
	MaxReleaseNotesLength     = 500
)

// ValidateTitle checks that a title does not exceed the 30-character limit.
// Returns nil if the title is valid.
func ValidateTitle(locale, title string) *CheckResult {
	if title == "" {
		return &CheckResult{
			ID:          "title-empty",
			Severity:    SeverityError,
			Locale:      locale,
			Field:       "title",
			Message:     "Title is empty",
			Remediation: "Provide a title (max 30 characters)",
		}
	}
	length := utf8.RuneCountInString(title)
	if length > MaxTitleLength {
		return &CheckResult{
			ID:          "title-too-long",
			Severity:    SeverityError,
			Locale:      locale,
			Field:       "title",
			Message:     fmt.Sprintf("Title is %d characters, exceeds %d character limit", length, MaxTitleLength),
			Remediation: fmt.Sprintf("Shorten the title to %d characters or fewer", MaxTitleLength),
		}
	}
	return nil
}

// ValidateShortDescription checks that a short description does not exceed the 80-character limit.
// Returns nil if the short description is valid.
func ValidateShortDescription(locale, desc string) *CheckResult {
	if desc == "" {
		return &CheckResult{
			ID:          "short-description-empty",
			Severity:    SeverityError,
			Locale:      locale,
			Field:       "short_description",
			Message:     "Short description is empty",
			Remediation: "Provide a short description (max 80 characters)",
		}
	}
	length := utf8.RuneCountInString(desc)
	if length > MaxShortDescriptionLength {
		return &CheckResult{
			ID:          "short-description-too-long",
			Severity:    SeverityError,
			Locale:      locale,
			Field:       "short_description",
			Message:     fmt.Sprintf("Short description is %d characters, exceeds %d character limit", length, MaxShortDescriptionLength),
			Remediation: fmt.Sprintf("Shorten the short description to %d characters or fewer", MaxShortDescriptionLength),
		}
	}
	return nil
}

// ValidateFullDescription checks that a full description does not exceed the 4000-character limit.
// Returns nil if the full description is valid.
func ValidateFullDescription(locale, desc string) *CheckResult {
	if desc == "" {
		return &CheckResult{
			ID:          "full-description-empty",
			Severity:    SeverityError,
			Locale:      locale,
			Field:       "full_description",
			Message:     "Full description is empty",
			Remediation: "Provide a full description (max 4000 characters)",
		}
	}
	length := utf8.RuneCountInString(desc)
	if length > MaxFullDescriptionLength {
		return &CheckResult{
			ID:          "full-description-too-long",
			Severity:    SeverityError,
			Locale:      locale,
			Field:       "full_description",
			Message:     fmt.Sprintf("Full description is %d characters, exceeds %d character limit", length, MaxFullDescriptionLength),
			Remediation: fmt.Sprintf("Shorten the full description to %d characters or fewer", MaxFullDescriptionLength),
		}
	}
	return nil
}

// ValidateReleaseNotes checks that release notes do not exceed the 500-character limit.
// Returns nil if the release notes are valid.
func ValidateReleaseNotes(locale, notes string) *CheckResult {
	if notes == "" {
		return &CheckResult{
			ID:          "release-notes-empty",
			Severity:    SeverityWarning,
			Locale:      locale,
			Field:       "release_notes",
			Message:     "Release notes are empty",
			Remediation: "Provide release notes for this release (max 500 characters)",
		}
	}
	length := utf8.RuneCountInString(notes)
	if length > MaxReleaseNotesLength {
		return &CheckResult{
			ID:          "release-notes-too-long",
			Severity:    SeverityError,
			Locale:      locale,
			Field:       "release_notes",
			Message:     fmt.Sprintf("Release notes are %d characters, exceeds %d character limit", length, MaxReleaseNotesLength),
			Remediation: fmt.Sprintf("Shorten the release notes to %d characters or fewer", MaxReleaseNotesLength),
		}
	}
	return nil
}
