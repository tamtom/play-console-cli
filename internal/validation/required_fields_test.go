package validation

import (
	"testing"
)

func TestValidateRequiredListingFields_AllPresent(t *testing.T) {
	listing := map[string]string{
		"title":             "My App",
		"short_description": "A great app",
		"full_description":  "This is a great app with many features.",
	}
	results := ValidateRequiredListingFields("en-US", listing)
	if len(results) != 0 {
		t.Errorf("expected 0 results for complete listing, got %d", len(results))
		for _, r := range results {
			t.Logf("  %s: %s", r.Field, r.Message)
		}
	}
}

func TestValidateRequiredListingFields_MissingTitle(t *testing.T) {
	listing := map[string]string{
		"short_description": "A great app",
		"full_description":  "This is a great app with many features.",
	}
	results := ValidateRequiredListingFields("en-US", listing)
	if len(results) != 1 {
		t.Fatalf("expected 1 result for missing title, got %d", len(results))
	}
	if results[0].Field != "title" {
		t.Errorf("expected field 'title', got %q", results[0].Field)
	}
	if results[0].Severity != SeverityError {
		t.Errorf("expected error severity, got %s", results[0].Severity)
	}
	if results[0].Locale != "en-US" {
		t.Errorf("expected locale en-US, got %s", results[0].Locale)
	}
}

func TestValidateRequiredListingFields_MissingAll(t *testing.T) {
	listing := map[string]string{}
	results := ValidateRequiredListingFields("en-US", listing)
	if len(results) != 3 {
		t.Errorf("expected 3 results for empty listing, got %d", len(results))
	}
}

func TestValidateRequiredListingFields_EmptyValues(t *testing.T) {
	listing := map[string]string{
		"title":             "",
		"short_description": "",
		"full_description":  "",
	}
	results := ValidateRequiredListingFields("en-US", listing)
	if len(results) != 3 {
		t.Errorf("expected 3 results for empty values, got %d", len(results))
	}
}

func TestValidateRequiredListingFields_WhitespaceOnly(t *testing.T) {
	listing := map[string]string{
		"title":             "   ",
		"short_description": "\t\n",
		"full_description":  "  \t  ",
	}
	results := ValidateRequiredListingFields("en-US", listing)
	if len(results) != 3 {
		t.Errorf("expected 3 results for whitespace-only values, got %d", len(results))
	}
}

func TestValidateRequiredListingFields_MissingShortDescription(t *testing.T) {
	listing := map[string]string{
		"title":            "My App",
		"full_description": "This is a great app with many features.",
	}
	results := ValidateRequiredListingFields("en-US", listing)
	if len(results) != 1 {
		t.Fatalf("expected 1 result for missing short_description, got %d", len(results))
	}
	if results[0].Field != "short_description" {
		t.Errorf("expected field 'short_description', got %q", results[0].Field)
	}
}

func TestValidateRequiredListingFields_MissingFullDescription(t *testing.T) {
	listing := map[string]string{
		"title":             "My App",
		"short_description": "A great app",
	}
	results := ValidateRequiredListingFields("en-US", listing)
	if len(results) != 1 {
		t.Fatalf("expected 1 result for missing full_description, got %d", len(results))
	}
	if results[0].Field != "full_description" {
		t.Errorf("expected field 'full_description', got %q", results[0].Field)
	}
}

func TestValidateRequiredListingFields_DifferentLocale(t *testing.T) {
	listing := map[string]string{
		"short_description": "A great app",
		"full_description":  "This is a great app.",
	}
	results := ValidateRequiredListingFields("ja-JP", listing)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Locale != "ja-JP" {
		t.Errorf("expected locale ja-JP, got %s", results[0].Locale)
	}
}

func TestValidateRequiredListingFields_NilMap(t *testing.T) {
	results := ValidateRequiredListingFields("en-US", nil)
	if len(results) != 3 {
		t.Errorf("expected 3 results for nil map, got %d", len(results))
	}
}
