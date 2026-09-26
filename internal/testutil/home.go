package testutil

import (
	"fmt"
	"os"
	"testing"
)

// RunWithIsolatedHome runs the package tests with HOME and USERPROFILE set to
// a temporary directory. Call it from TestMain in a package whose tests run
// commands that fall back to ~/.gplay (config, audit log, markers), so that
// no test can create or overwrite files in the real home directory.
func RunWithIsolatedHome(m *testing.M) int {
	home, err := os.MkdirTemp("", "gplay-test-home-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "creating isolated home: %v\n", err)
		return 1
	}
	defer func() { _ = os.RemoveAll(home) }()

	for _, key := range []string{"GPLAY_CONFIG_PATH", "GPLAY_AUDIT_LOG"} {
		_ = os.Unsetenv(key)
	}
	_ = os.Setenv("HOME", home)
	_ = os.Setenv("USERPROFILE", home)

	return m.Run()
}
