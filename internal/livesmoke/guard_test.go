package livesmoke

import (
	"strings"
	"testing"
)

func TestEnsureMutationAllowed_ExactFixturePackage(t *testing.T) {
	if err := EnsureMutationAllowed("com.itdeveapps.stepsshare"); err != nil {
		t.Fatalf("fixture package must be allowed, got error: %v", err)
	}
}

func TestEnsureMutationAllowed_RejectsOtherPackages(t *testing.T) {
	cases := []string{
		"",
		"com.example.app",
		"com.itdeveapps.stepsshare2",
		"com.itdeveapps.stepsshareX",
		"com.itdeveapps.STEPSSHARE",
		" com.itdeveapps.stepsshare",
		"com.itdeveapps.stepsshare ",
		"com.itdeveapps.stepsshare\n",
	}
	for _, pkg := range cases {
		err := EnsureMutationAllowed(pkg)
		if err == nil {
			t.Fatalf("package %q must be rejected", pkg)
		}
		if !strings.Contains(err.Error(), "mutation") {
			t.Fatalf("error for %q must name the mutation guard, got: %v", pkg, err)
		}
	}
}

func TestResourceName_ContainsPrefixAndRunID(t *testing.T) {
	name := ResourceName("someid", "product")
	if !strings.HasPrefix(name, NamePrefix) {
		t.Fatalf("resource name %q must start with %q", name, NamePrefix)
	}
	if !strings.Contains(name, "someid") {
		t.Fatalf("resource name %q must contain the run ID", name)
	}
	if !strings.Contains(name, "product") {
		t.Fatalf("resource name %q must contain the kind", name)
	}
}

func TestIsManagedResourceName(t *testing.T) {
	if !IsManagedResourceName(ResourceName("123", "product")) {
		t.Fatal("generated names must be recognized as managed")
	}
	for _, name := range []string{"", "premium_upgrade", "coins_100", "sku_livesmoke"} {
		if IsManagedResourceName(name) {
			t.Fatalf("foreign name %q must not be recognized as managed", name)
		}
	}
}
