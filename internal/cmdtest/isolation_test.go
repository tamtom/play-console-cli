package cmdtest_test

import (
	"os"
	"testing"
)

func TestIsolation_ConfigPathSet(t *testing.T) {
	got := os.Getenv("GPLAY_CONFIG_PATH")
	if got != testConfigPath {
		t.Fatalf("GPLAY_CONFIG_PATH not isolated: got %q, want %q", got, testConfigPath)
	}
}

func TestIsolation_NoUpdateSet(t *testing.T) {
	got := os.Getenv("GPLAY_NO_UPDATE")
	if got != "1" {
		t.Fatalf("GPLAY_NO_UPDATE not set: got %q", got)
	}
}

func TestIsolation_RealCredentialsAndIntegrationGatesAreScrubbed(t *testing.T) {
	for _, key := range []string{
		"GOOGLE_APPLICATION_CREDENTIALS",
		"GPLAY_ANDROID_DEVELOPER_ID_API_KEY",
		"GPLAY_INTEGRATION_TEST",
		"GPLAY_MUTATING_INTEGRATION_TEST",
		"GPLAY_OAUTH_TOKEN_PATH",
		"GPLAY_SERVICE_ACCOUNT",
		"GPLAY_SERVICE_ACCOUNT_JSON",
	} {
		if value := os.Getenv(key); value != "" {
			t.Errorf("%s leaked into black-box tests", key)
		}
	}
}
