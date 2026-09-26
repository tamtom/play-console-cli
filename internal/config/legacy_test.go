package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLegacyLocalConfigDoesNotFallBackToGlobal(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", os.Getenv("HOME"))
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
	if got := IgnoredLegacyGlobalPath(); got != "" {
		t.Fatalf("IgnoredLegacyGlobalPath = %q for a local YAML file, want empty", got)
	}
}

func TestGlobalLegacyYAMLIsIgnored(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("GPLAY_CONFIG_PATH", "")
	legacy := filepath.Join(home, ".gplay", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(legacy), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(legacy, []byte("default_package: com.example.legacy\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	project := filepath.Join(home, "project")
	if err := os.Mkdir(project, 0o700); err != nil {
		t.Fatal(err)
	}

	for _, cwd := range []string{home, project} {
		t.Run(filepath.Base(cwd), func(t *testing.T) {
			t.Chdir(cwd)
			path, err := Path()
			if err != nil {
				t.Fatal(err)
			}
			if filepath.Base(path) != "config.json" {
				t.Fatalf("Path = %q, want the global config.json", path)
			}
			if _, err := Load(); !errors.Is(err, ErrNotFound) {
				t.Fatalf("Load error = %v, want ErrNotFound", err)
			}
			if got := IgnoredLegacyGlobalPath(); filepath.Base(got) != "config.yaml" {
				t.Fatalf("IgnoredLegacyGlobalPath = %q, want the global config.yaml", got)
			}
		})
	}
}
