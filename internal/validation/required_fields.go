package validation

import (
	"fmt"
	"strings"
)

// requiredListingFields lists the fields that must be non-empty in a listing.
var requiredListingFields = []string{
	"title",
	"short_description",
	"full_description",
}

// ValidateRequiredListingFields checks that all required listing fields
// are present and non-empty. Returns a slice of CheckResults for any
// missing or empty fields.
func ValidateRequiredListingFields(locale string, listing map[string]string) []CheckResult {
	var results []CheckResult
	for _, field := range requiredListingFields {
		value := strings.TrimSpace(listing[field])
		if value == "" {
			results = append(results, CheckResult{
				ID:          fmt.Sprintf("required-%s", field),
				Severity:    SeverityError,
				Locale:      locale,
				Field:       field,
				Message:     fmt.Sprintf("Required field %q is missing or empty", field),
				Remediation: fmt.Sprintf("Provide a non-empty value for %s", field),
			})
		}
	}
	return results
}
