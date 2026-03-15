package cmdtest_test

import (
	"os"
	"path/filepath"
	"testing"
)

var testConfigPath string

func TestMain(m *testing.M) {
	// Create isolated temp directory for all cmdtest tests
	tempDir, err := os.MkdirTemp("", "gplay-cmdtest-*")
	if err != nil {
		panic("failed to create temp dir: " + err.Error())
	}

	testConfigPath = filepath.Join(tempDir, "config.json")

	// Isolate from real user config and state
	os.Setenv("GPLAY_CONFIG_PATH", testConfigPath)
	os.Setenv("GPLAY_NO_UPDATE", "1")
	os.Setenv("HOME", tempDir)

	code := m.Run()

	// Cleanup
	_ = os.RemoveAll(tempDir)
	os.Exit(code)
}
