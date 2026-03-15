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
