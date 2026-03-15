package validation

import (
	"strings"
	"testing"
)

func TestValidateTitle_Empty(t *testing.T) {
	result := ValidateTitle("en-US", "")
	if result == nil {
		t.Fatal("expected non-nil result for empty title")
	}
	if result.Severity != SeverityError {
		t.Errorf("expected error severity, got %s", result.Severity)
	}
	if result.Locale != "en-US" {
		t.Errorf("expected locale en-US, got %s", result.Locale)
	}
	if result.Field != "title" {
		t.Errorf("expected field title, got %s", result.Field)
	}
}

func TestValidateTitle_Valid(t *testing.T) {
	result := ValidateTitle("en-US", "My App")
	if result != nil {
		t.Errorf("expected nil for valid title, got: %+v", result)
	}
}

func TestValidateTitle_ExactlyAtLimit(t *testing.T) {
	title := strings.Repeat("a", 30)
	result := ValidateTitle("en-US", title)
	if result != nil {
		t.Errorf("expected nil for title at exactly 30 chars, got: %+v", result)
	}
}

func TestValidateTitle_OverLimit(t *testing.T) {
	title := strings.Repeat("a", 31)
	result := ValidateTitle("en-US", title)
	if result == nil {
		t.Fatal("expected non-nil result for title over limit")
	}
	if result.Severity != SeverityError {
		t.Errorf("expected error severity, got %s", result.Severity)
	}
	if !strings.Contains(result.Message, "30") {
		t.Errorf("message should mention 30 char limit, got: %s", result.Message)
	}
}

func TestValidateTitle_Unicode(t *testing.T) {
	// Japanese characters count as 1 rune each
	title := strings.Repeat("\u3042", 30) // 30 hiragana 'a' characters
	result := ValidateTitle("ja-JP", title)
	if result != nil {
		t.Errorf("expected nil for title at exactly 30 unicode chars, got: %+v", result)
	}
}

func TestValidateTitle_UnicodeOverLimit(t *testing.T) {
	title := strings.Repeat("\u3042", 31)
	result := ValidateTitle("ja-JP", title)
	if result == nil {
		t.Fatal("expected non-nil result for unicode title over limit")
	}
}

func TestValidateShortDescription_Empty(t *testing.T) {
	result := ValidateShortDescription("en-US", "")
	if result == nil {
		t.Fatal("expected non-nil result for empty short description")
	}
	if result.Severity != SeverityError {
		t.Errorf("expected error severity, got %s", result.Severity)
	}
	if result.Field != "short_description" {
		t.Errorf("expected field short_description, got %s", result.Field)
	}
}

func TestValidateShortDescription_Valid(t *testing.T) {
	result := ValidateShortDescription("en-US", "A short description of my app")
	if result != nil {
		t.Errorf("expected nil for valid short description, got: %+v", result)
	}
}

func TestValidateShortDescription_ExactlyAtLimit(t *testing.T) {
	desc := strings.Repeat("a", 80)
	result := ValidateShortDescription("en-US", desc)
	if result != nil {
		t.Errorf("expected nil for short desc at exactly 80 chars, got: %+v", result)
	}
}

func TestValidateShortDescription_OverLimit(t *testing.T) {
	desc := strings.Repeat("a", 81)
	result := ValidateShortDescription("en-US", desc)
	if result == nil {
		t.Fatal("expected non-nil result for short description over limit")
	}
	if result.Severity != SeverityError {
		t.Errorf("expected error severity, got %s", result.Severity)
	}
	if !strings.Contains(result.Message, "80") {
		t.Errorf("message should mention 80 char limit, got: %s", result.Message)
	}
}

func TestValidateFullDescription_Empty(t *testing.T) {
	result := ValidateFullDescription("en-US", "")
	if result == nil {
		t.Fatal("expected non-nil result for empty full description")
	}
	if result.Severity != SeverityError {
		t.Errorf("expected error severity, got %s", result.Severity)
	}
	if result.Field != "full_description" {
		t.Errorf("expected field full_description, got %s", result.Field)
	}
}

func TestValidateFullDescription_Valid(t *testing.T) {
	result := ValidateFullDescription("en-US", "A longer description of my app with more details.")
	if result != nil {
		t.Errorf("expected nil for valid full description, got: %+v", result)
	}
}

func TestValidateFullDescription_ExactlyAtLimit(t *testing.T) {
	desc := strings.Repeat("a", 4000)
	result := ValidateFullDescription("en-US", desc)
	if result != nil {
		t.Errorf("expected nil for full desc at exactly 4000 chars, got: %+v", result)
	}
}

func TestValidateFullDescription_OverLimit(t *testing.T) {
	desc := strings.Repeat("a", 4001)
	result := ValidateFullDescription("en-US", desc)
	if result == nil {
		t.Fatal("expected non-nil result for full description over limit")
	}
	if result.Severity != SeverityError {
		t.Errorf("expected error severity, got %s", result.Severity)
	}
	if !strings.Contains(result.Message, "4000") {
		t.Errorf("message should mention 4000 char limit, got: %s", result.Message)
	}
}

func TestValidateReleaseNotes_Empty(t *testing.T) {
	result := ValidateReleaseNotes("en-US", "")
	if result == nil {
		t.Fatal("expected non-nil result for empty release notes")
	}
	if result.Severity != SeverityWarning {
		t.Errorf("expected warning severity for empty release notes, got %s", result.Severity)
	}
}

func TestValidateReleaseNotes_Valid(t *testing.T) {
	result := ValidateReleaseNotes("en-US", "Bug fixes and improvements")
	if result != nil {
		t.Errorf("expected nil for valid release notes, got: %+v", result)
	}
}

func TestValidateReleaseNotes_ExactlyAtLimit(t *testing.T) {
	notes := strings.Repeat("a", 500)
	result := ValidateReleaseNotes("en-US", notes)
	if result != nil {
		t.Errorf("expected nil for release notes at exactly 500 chars, got: %+v", result)
	}
}

func TestValidateReleaseNotes_OverLimit(t *testing.T) {
	notes := strings.Repeat("a", 501)
	result := ValidateReleaseNotes("en-US", notes)
	if result == nil {
		t.Fatal("expected non-nil result for release notes over limit")
	}
	if result.Severity != SeverityError {
		t.Errorf("expected error severity, got %s", result.Severity)
	}
	if !strings.Contains(result.Message, "500") {
		t.Errorf("message should mention 500 char limit, got: %s", result.Message)
	}
}

func TestValidateTitle_HasRemediation(t *testing.T) {
	title := strings.Repeat("a", 31)
	result := ValidateTitle("en-US", title)
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Remediation == "" {
		t.Error("expected non-empty remediation for title over limit")
	}
}
