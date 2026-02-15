package listings

import (
	"strings"
	"testing"
)

func TestValidateLocale_Valid(t *testing.T) {
	valid := []string{
		"en-US", "en-GB", "fr-FR", "de-DE", "ja-JP", "zh-CN",
		"es-419", "fil", "af", "am", "ar", "id", "zu",
	}
	for _, code := range valid {
		if err := ValidateLocale(code); err != nil {
			t.Errorf("ValidateLocale(%q) = %v, want nil", code, err)
		}
	}
}

func TestValidateLocale_Empty(t *testing.T) {
	err := ValidateLocale("")
	if err == nil {
		t.Fatal("ValidateLocale(\"\") = nil, want error")
	}
	if !strings.Contains(err.Error(), "empty") {
		t.Errorf("error = %q, want to contain 'empty'", err.Error())
	}
}

func TestValidateLocale_Whitespace(t *testing.T) {
	err := ValidateLocale("   ")
	if err == nil {
		t.Fatal("ValidateLocale(\"   \") = nil, want error")
	}
	if !strings.Contains(err.Error(), "empty") {
		t.Errorf("error = %q, want to contain 'empty'", err.Error())
	}
}

func TestValidateLocale_UnderscoreSuggestion(t *testing.T) {
	err := ValidateLocale("en_US")
	if err == nil {
		t.Fatal("ValidateLocale(\"en_US\") = nil, want error")
	}
	if !strings.Contains(err.Error(), "en-US") {
		t.Errorf("error = %q, want suggestion for en-US", err.Error())
	}
	if !strings.Contains(err.Error(), "hyphens") {
		t.Errorf("error = %q, want hint about hyphens", err.Error())
	}
}

func TestValidateLocale_CaseMismatch(t *testing.T) {
	err := ValidateLocale("en-us")
	if err == nil {
		t.Fatal("ValidateLocale(\"en-us\") = nil, want error")
	}
	if !strings.Contains(err.Error(), "en-US") {
		t.Errorf("error = %q, want suggestion for en-US", err.Error())
	}
	if !strings.Contains(err.Error(), "capitalization") {
		t.Errorf("error = %q, want hint about capitalization", err.Error())
	}
}

func TestValidateLocale_UnderscoreAndCaseMismatch(t *testing.T) {
	err := ValidateLocale("en_us")
	if err == nil {
		t.Fatal("ValidateLocale(\"en_us\") = nil, want error")
	}
	if !strings.Contains(err.Error(), "en-US") {
		t.Errorf("error = %q, want suggestion for en-US", err.Error())
	}
}

func TestValidateLocale_UnknownLocale(t *testing.T) {
	err := ValidateLocale("xx-YY")
	if err == nil {
		t.Fatal("ValidateLocale(\"xx-YY\") = nil, want error")
	}
	if !strings.Contains(err.Error(), "not a supported") {
		t.Errorf("error = %q, want 'not a supported' message", err.Error())
	}
	if !strings.Contains(err.Error(), "gplay listings locales") {
		t.Errorf("error = %q, want reference to locales command", err.Error())
	}
}

func TestValidateLocale_TrimWhitespace(t *testing.T) {
	err := ValidateLocale("  en-US  ")
	if err != nil {
		t.Errorf("ValidateLocale(\"  en-US  \") = %v, want nil", err)
	}
}

func TestSortedLocales(t *testing.T) {
	locales := SortedLocales()
	if len(locales) != len(SupportedLocales) {
		t.Errorf("SortedLocales() returned %d items, want %d", len(locales), len(SupportedLocales))
	}
	for i := 1; i < len(locales); i++ {
		if locales[i] < locales[i-1] {
			t.Errorf("SortedLocales() not sorted: %q comes after %q", locales[i], locales[i-1])
		}
	}
}

func TestSortedLocales_ContainsKnownLocales(t *testing.T) {
	locales := SortedLocales()
	localeSet := make(map[string]bool)
	for _, l := range locales {
		localeSet[l] = true
	}
	expected := []string{"en-US", "ja-JP", "es-419", "fil", "af", "zu"}
	for _, e := range expected {
		if !localeSet[e] {
			t.Errorf("SortedLocales() missing %q", e)
		}
	}
}
