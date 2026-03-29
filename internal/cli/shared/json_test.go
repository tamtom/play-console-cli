package shared

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadJSONArgRaw_InlineJSON(t *testing.T) {
	raw, err := LoadJSONArgRaw(`{"listings":[]}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(raw) != `{"listings":[]}` {
		t.Errorf("got %q, want %q", string(raw), `{"listings":[]}`)
	}
}

func TestLoadJSONArgRaw_FileJSON(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "test.json")
	if err := os.WriteFile(p, []byte(`{"key":"value"}`), 0644); err != nil {
		t.Fatal(err)
	}
	raw, err := LoadJSONArgRaw("@" + p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(raw) != `{"key":"value"}` {
		t.Errorf("got %q", string(raw))
	}
}

func TestLoadJSONArgRaw_Empty(t *testing.T) {
	_, err := LoadJSONArgRaw("")
	if err == nil {
		t.Fatal("expected error for empty input")
	}
}

func TestLoadJSONArgRaw_InvalidFilePath(t *testing.T) {
	_, err := LoadJSONArgRaw("@")
	if err == nil {
		t.Fatal("expected error for bare @")
	}
}

func TestDeriveUpdateMask_NormalFields(t *testing.T) {
	raw := []byte(`{"listings":[],"purchaseOptions":[]}`)
	mutable := []string{"listings", "offerTags", "purchaseOptions", "restrictedPaymentCountries", "taxAndComplianceSettings"}
	mask, err := DeriveUpdateMask(raw, mutable)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mask != "listings,purchaseOptions" {
		t.Errorf("got %q, want %q", mask, "listings,purchaseOptions")
	}
}

func TestDeriveUpdateMask_AllFields(t *testing.T) {
	raw := []byte(`{"listings":[],"offerTags":[],"purchaseOptions":[],"restrictedPaymentCountries":{},"taxAndComplianceSettings":{}}`)
	mutable := []string{"listings", "offerTags", "purchaseOptions", "restrictedPaymentCountries", "taxAndComplianceSettings"}
	mask, err := DeriveUpdateMask(raw, mutable)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mask != "listings,offerTags,purchaseOptions,restrictedPaymentCountries,taxAndComplianceSettings" {
		t.Errorf("got %q", mask)
	}
}

func TestDeriveUpdateMask_ImmutableFieldsFiltered(t *testing.T) {
	raw := []byte(`{"listings":[],"packageName":"com.example","productId":"test"}`)
	mutable := []string{"listings", "offerTags", "purchaseOptions"}
	mask, err := DeriveUpdateMask(raw, mutable)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mask != "listings" {
		t.Errorf("got %q, want %q", mask, "listings")
	}
}

func TestDeriveUpdateMask_EmptyJSON(t *testing.T) {
	raw := []byte(`{}`)
	mutable := []string{"listings", "purchaseOptions"}
	_, err := DeriveUpdateMask(raw, mutable)
	if err == nil {
		t.Fatal("expected error for empty JSON object")
	}
	if !strings.Contains(err.Error(), "no updatable fields") {
		t.Errorf("error should mention no updatable fields, got: %s", err.Error())
	}
}

func TestDeriveUpdateMask_OnlyImmutableFields(t *testing.T) {
	raw := []byte(`{"packageName":"com.example","productId":"test"}`)
	mutable := []string{"listings", "purchaseOptions"}
	_, err := DeriveUpdateMask(raw, mutable)
	if err == nil {
		t.Fatal("expected error when only immutable fields are present")
	}
	if !strings.Contains(err.Error(), "no updatable fields") {
		t.Errorf("error should mention no updatable fields, got: %s", err.Error())
	}
}

func TestDeriveUpdateMask_InvalidJSON(t *testing.T) {
	raw := []byte(`not json`)
	mutable := []string{"listings"}
	_, err := DeriveUpdateMask(raw, mutable)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestDeriveUpdateMask_SortOrder(t *testing.T) {
	raw := []byte(`{"taxAndComplianceSettings":{},"listings":[],"offerTags":[]}`)
	mutable := []string{"listings", "offerTags", "purchaseOptions", "taxAndComplianceSettings"}
	mask, err := DeriveUpdateMask(raw, mutable)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mask != "listings,offerTags,taxAndComplianceSettings" {
		t.Errorf("mask not sorted: got %q", mask)
	}
}

func TestDeriveUpdateMask_SingleField(t *testing.T) {
	raw := []byte(`{"listings":[{"languageCode":"en-US","title":"Test"}]}`)
	mutable := []string{"listings", "purchaseOptions"}
	mask, err := DeriveUpdateMask(raw, mutable)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mask != "listings" {
		t.Errorf("got %q, want %q", mask, "listings")
	}
}

func TestDeriveUpdateMask_UnknownFieldsIgnored(t *testing.T) {
	// Unknown fields are dropped by SDK unmarshal, so they must not appear
	// in the mask — that would cause a 400 (mask names a field the body
	// doesn't contain).
	raw := []byte(`{"listings":[],"unknownFutureField":"value"}`)
	mutable := []string{"listings", "purchaseOptions"}
	mask, err := DeriveUpdateMask(raw, mutable)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mask != "listings" {
		t.Errorf("got %q, want %q (unknown fields should be excluded)", mask, "listings")
	}
}
