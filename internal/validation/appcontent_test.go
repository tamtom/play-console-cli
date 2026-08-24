package validation

import (
	"strings"
	"testing"
)

func TestValidateAppContentCompleteInventory(t *testing.T) {
	inventory := AppContentInventory{
		PrivacyPolicyURL:    "https://example.com/privacy",
		SupportEmail:        "support@example.com",
		Ads:                 "no",
		AppAccess:           "all-accessible",
		TargetAudience:      []string{"18+"},
		ContentRatingStatus: "complete",
		DataSafetyStatus:    "complete",
		Category:            "APPLICATION",
		Tags:                []string{"productivity"},
		InitialCountries:    []string{"US"},
		Declarations: map[string]string{
			"financial": "not-applicable",
			"health":    "not-applicable",
			"news":      "not-applicable",
		},
		SensitivePermissions: []PermissionDeclaration{
			{Name: "android.permission.CAMERA", Status: "complete"},
		},
	}
	results := ValidateAppContent(inventory)
	for _, result := range results {
		if result.Severity == SeverityError || result.Severity == SeverityWarning {
			t.Fatalf("unexpected finding: %#v", result)
		}
	}
}

func TestValidateAppContentReportsStableUnresolvedChecks(t *testing.T) {
	results := ValidateAppContent(AppContentInventory{
		PrivacyPolicyURL:    "javascript:alert(1)",
		SupportEmail:        "not-an-email",
		Ads:                 "unknown",
		AppAccess:           "restricted",
		ContentRatingStatus: "pending",
		DataSafetyStatus:    "pending",
		Declarations:        map[string]string{"health": "pending"},
		SensitivePermissions: []PermissionDeclaration{
			{Name: "android.permission.READ_SMS", Status: "pending"},
		},
	})
	want := []string{
		"app-content-privacy-policy-url-invalid",
		"app-content-support-email-invalid",
		"app-content-ads-unresolved",
		"app-content-reviewer-instructions-missing",
		"app-content-target-audience-missing",
		"app-content-content-rating-pending",
		"app-content-data-safety-pending",
		"app-content-declaration-pending",
		"app-content-sensitive-permission-pending",
	}
	for _, id := range want {
		if !containsCheckID(results, id) {
			t.Fatalf("missing check %q in %#v", id, results)
		}
	}
}

func TestValidateListingQualityDetectsPlaceholderAndWeakCopy(t *testing.T) {
	results := ValidateListingQuality("en-US", map[string]string{
		"title":             "Test App",
		"short_description": "TODO",
		"full_description":  "Lorem ipsum",
		"video":             "javascript:alert(1)",
	})
	for _, id := range []string{
		"metadata-placeholder",
		"short-description-too-short",
		"full-description-too-short",
		"video-url-invalid",
	} {
		if !containsCheckID(results, id) {
			t.Fatalf("missing check %q in %#v", id, results)
		}
	}
}

func containsCheckID(results []CheckResult, id string) bool {
	for _, result := range results {
		if strings.EqualFold(result.ID, id) {
			return true
		}
	}
	return false
}
