package cmdtest_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tamtom/play-console-cli/internal/cmdtest"
)

var testConfigPath string

func TestMain(m *testing.M) {
	// Create isolated temp directory for all cmdtest tests
	tempDir, err := os.MkdirTemp("", "gplay-cmdtest-*")
	if err != nil {
		panic("failed to create temp dir: " + err.Error())
	}

	testConfigPath = filepath.Join(tempDir, "config.json")

	// Isolate from real credentials, resource defaults, and integration gates.
	// Individual tests may set explicit mock values after TestMain starts.
	for _, key := range []string{
		"GOOGLE_APPLICATION_CREDENTIALS",
		"GPLAY_ANDROID_DEVELOPER_ID_API_KEY",
		"GPLAY_CHECKS_ACCOUNT",
		"GPLAY_GAMES_APP_ID",
		"GPLAY_INTEGRATION_PACKAGE",
		"GPLAY_INTEGRATION_TEST",
		"GPLAY_MUTATING_INTEGRATION_TEST",
		"GPLAY_OAUTH_CLIENT_ID",
		"GPLAY_OAUTH_CLIENT_SECRET",
		"GPLAY_OAUTH_TOKEN_PATH",
		"GPLAY_PACKAGE_NAME",
		"GPLAY_PROFILE",
		"GPLAY_SERVICE_ACCOUNT",
		"GPLAY_SERVICE_ACCOUNT_JSON",
	} {
		_ = os.Unsetenv(key)
	}

	// Isolate from real user config and state.
	os.Setenv("GPLAY_CONFIG_PATH", testConfigPath)
	os.Setenv("GPLAY_NO_UPDATE", "1")
	os.Setenv("HOME", tempDir)

	code := m.Run()

	// Cleanup
	cmdtest.Cleanup()
	_ = os.RemoveAll(tempDir)
	os.Exit(code)
}
