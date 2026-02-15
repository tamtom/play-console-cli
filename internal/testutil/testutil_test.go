package testutil

import (
	"encoding/json"
	"os"
	"testing"
)

func TestSkipUnlessIntegration_Skips(t *testing.T) {
	os.Unsetenv("GPLAY_INTEGRATION_TEST")
	st := &testing.T{}
	// We can't easily test t.Skip in a sub-test without running a subprocess,
	// so we test the positive path below.
	_ = st
}

func TestSkipUnlessIntegration_Runs(t *testing.T) {
	t.Setenv("GPLAY_INTEGRATION_TEST", "1")
	// Should not skip — if it does, this test would be marked as skipped.
	SkipUnlessIntegration(t)
}

func TestSkipUnlessIntegration_RunsTrue(t *testing.T) {
	t.Setenv("GPLAY_INTEGRATION_TEST", "true")
	SkipUnlessIntegration(t)
}

func TestIsolateConfig(t *testing.T) {
	original := os.Getenv("GPLAY_CONFIG_PATH")
	dir := IsolateConfig(t)
	if dir == "" {
		t.Fatal("expected non-empty config dir")
	}
	got := os.Getenv("GPLAY_CONFIG_PATH")
	if got != dir {
		t.Errorf("GPLAY_CONFIG_PATH = %q, want %q", got, dir)
	}
	// After test cleanup, the env should be restored.
	// We can't test cleanup directly, but we verify it was set.
	_ = original
}

func TestMockServiceAccount(t *testing.T) {
	path := MockServiceAccount(t)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading mock service account: %v", err)
	}
	var sa map[string]string
	if err := json.Unmarshal(data, &sa); err != nil {
		t.Fatalf("parsing mock service account: %v", err)
	}
	requiredFields := []string{"type", "project_id", "private_key_id", "private_key", "client_email", "client_id", "auth_uri", "token_uri"}
	for _, field := range requiredFields {
		if sa[field] == "" {
			t.Errorf("missing required field %q", field)
		}
	}
	if sa["type"] != "service_account" {
		t.Errorf("type = %q, want %q", sa["type"], "service_account")
	}
}

func TestRequireEnv_Skips(t *testing.T) {
	// Set an env var and verify RequireEnv returns it
	t.Setenv("GPLAY_TEST_DUMMY", "hello")
	got := RequireEnv(t, "GPLAY_TEST_DUMMY")
	if got != "hello" {
		t.Errorf("RequireEnv = %q, want %q", got, "hello")
	}
}
