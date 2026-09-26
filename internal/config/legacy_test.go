package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLegacyLocalConfigDoesNotFallBackToGlobal(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("GPLAY_CONFIG_PATH", "")
	if err := os.Mkdir(".gplay", 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(".gplay", "config.yaml"), []byte("default_package: com.example.legacy\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "gplay init --force") {
		t.Fatalf("expected migration guidance, got %v", err)
	}
}
