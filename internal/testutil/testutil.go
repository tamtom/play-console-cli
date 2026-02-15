package testutil

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// SkipUnlessIntegration skips the test unless GPLAY_INTEGRATION_TEST is set to "1" or "true".
func SkipUnlessIntegration(t *testing.T) {
	t.Helper()
	v := os.Getenv("GPLAY_INTEGRATION_TEST")
	if v != "1" && v != "true" {
		t.Skip("skipping integration test; set GPLAY_INTEGRATION_TEST=1 to run")
	}
}

// IsolateConfig sets GPLAY_CONFIG_PATH to a fresh temp directory and registers cleanup.
func IsolateConfig(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	prev := os.Getenv("GPLAY_CONFIG_PATH")
	os.Setenv("GPLAY_CONFIG_PATH", dir)
	t.Cleanup(func() {
		if prev == "" {
			os.Unsetenv("GPLAY_CONFIG_PATH")
		} else {
			os.Setenv("GPLAY_CONFIG_PATH", prev)
		}
	})
	return dir
}

// MockServiceAccount creates a fake but structurally valid service account JSON file.
func MockServiceAccount(t *testing.T) string {
	t.Helper()
	sa := map[string]string{
		"type":           "service_account",
		"project_id":     "test-project",
		"private_key_id": "key123",
		"private_key":    "-----BEGIN RSA PRIVATE KEY-----\nMIIBogIBAAJBALRiMLAHudeSA/x3hB2f+2NRkJlBEBGoviKrswxMa0sNHBEJTGcb\nkz9/M7FjpSBiVkGBPIxZKylFSk693mUCAwEAAQJAZ6bUJeMhNgDJOuMFsGN2IyGX\nmEaSPLbSJMiJQBGHGYyhE0PBuSl7SgPgyEBJjRTDlHBuC6gyIa3FqxhyzNP7gQIh\nAOHqJBE1YMeitHv/GERNIKc0dCtMJPAAthGvjhSVFILRAiEAzCAuaJOIFpUm6NUM\nT3qlV8kJOEGzMbPFjGaVwPBuysUCIGbqMhE3T5Rj/MBFBiaNjSEt6JFj01l2g0Ev\nCEO+aNYRAiEAu4ETcKMiJp9BbCJHr0MBqEBb71FaDjX11YMq52wHSGkCIDRATuFg\ntgb2ArFe6t+vg0mJe0dCVHlGRv1jGkRoX+4q\n-----END RSA PRIVATE KEY-----\n",
		"client_email":   "test@test-project.iam.gserviceaccount.com",
		"client_id":      "123456789",
		"auth_uri":       "https://accounts.google.com/o/oauth2/auth",
		"token_uri":      "https://oauth2.googleapis.com/token",
	}
	data, err := json.MarshalIndent(sa, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "service-account.json")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// RequireEnv skips the test if the named env var is not set, otherwise returns its value.
func RequireEnv(t *testing.T, key string) string {
	t.Helper()
	v := os.Getenv(key)
	if v == "" {
		t.Skipf("skipping: %s not set", key)
	}
	return v
}
