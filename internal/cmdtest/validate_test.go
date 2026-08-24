package cmdtest_test

import (
	"context"
	"strings"
	"testing"
)

func TestValidate_Help(t *testing.T) {
	root := RootCommand("test")
	stdout, stderr := captureOutput(t, func() {
		_ = root.Parse([]string{"validate"})
		_ = root.Run(context.Background())
	})
	combined := stdout + stderr
	if !strings.Contains(combined, "validate") {
		t.Fatalf("help should mention validate, got %q", combined)
	}
	for _, token := range []string{"bundle", "listing", "screenshots", "submission", "app-content", "--track", "--release-notes", "--offline"} {
		if !strings.Contains(combined, token) {
			t.Fatalf("help should mention %q, got %q", token, combined)
		}
	}
}

func TestValidate_RootFlagsExist(t *testing.T) {
	root := RootCommand("test")
	var parseErr error
	captureOutput(t, func() {
		parseErr = root.Parse([]string{"validate", "--package", "com.example.app"})
	})
	if parseErr != nil {
		t.Fatalf("validate root flags should parse, got %v", parseErr)
	}
}

func TestValidate_RootRejectsConflictingArtifactFlags(t *testing.T) {
	_, _, err := runCommand(
		t,
		"validate",
		"--package", "com.example.app",
		"--bundle", "app.aab",
		"--apk", "app.apk",
	)
	if err == nil {
		t.Fatal("expected conflicting artifact error")
	}
	if !strings.Contains(err.Error(), "use either --bundle or --apk") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateOfflineAppContentNeedsNoPackageOrAuthentication(t *testing.T) {
	input := `{"privacyPolicyUrl":"https://example.com/privacy","supportEmail":"support@example.com","ads":"no","appAccess":"all-accessible","targetAudience":["18+"],"contentRatingStatus":"complete","dataSafetyStatus":"complete","category":"APPLICATION","initialCountries":["US"],"policyDeclarationsReviewed":true,"declarations":{"financial-features":"not-applicable","health":"not-applicable","news":"not-applicable"},"sensitivePermissionsReviewed":true}`
	stdout, stderr, err := runCommand(
		t,
		"validate", "--offline", "--app-content", input,
		"--release-notes", "Bug fixes and stability improvements.",
	)
	if err != nil {
		t.Fatalf("offline validate: %v\nstderr: %s", err, stderr)
	}
	for _, want := range []string{`"offline":true`, `"id":"remote-checks-skipped"`, `"id":"app-content-inventory-loaded"`} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("offline output missing %q: %s", want, stdout)
		}
	}
}
