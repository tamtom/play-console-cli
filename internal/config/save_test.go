package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfig_ExplicitZeroMaxRetriesRoundTrips(t *testing.T) {
	var cfg Config
	if err := json.Unmarshal([]byte(`{"package_name":"com.example.app","max_retries":0}`), &cfg); err != nil {
		t.Fatal(err)
	}
	if !cfg.MaxRetriesConfigured() {
		t.Fatal("explicit max_retries=0 was not retained")
	}
	encoded, err := json.Marshal(&cfg)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(encoded), `"max_retries":0`) {
		t.Fatalf("encoded config = %s, want explicit max_retries=0", encoded)
	}
}

func TestSaveAtIsAtomicAndRefusesSymlinkDestination(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "config.json")
	if err := SaveAt(path, &Config{PackageName: "com.example.first"}); err != nil {
		t.Fatalf("SaveAt first: %v", err)
	}
	if err := SaveAt(path, &Config{PackageName: "com.example.second"}); err != nil {
		t.Fatalf("SaveAt second: %v", err)
	}
	loaded, err := LoadAt(path)
	if err != nil {
		t.Fatalf("LoadAt: %v", err)
	}
	if loaded.PackageName != "com.example.second" {
		t.Fatalf("PackageName = %q, want com.example.second", loaded.PackageName)
	}

	sentinel := filepath.Join(dir, "sentinel")
	if err := os.WriteFile(sentinel, []byte("original"), 0o600); err != nil {
		t.Fatalf("write sentinel: %v", err)
	}
	linked := filepath.Join(dir, "linked-config.json")
	if err := os.Symlink(sentinel, linked); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if err := SaveAt(linked, &Config{PackageName: "com.example.unsafe"}); err == nil {
		t.Fatal("SaveAt unexpectedly accepted a symlink destination")
	}
	got, err := os.ReadFile(sentinel)
	if err != nil || string(got) != "original" {
		t.Fatalf("sentinel changed: content=%q error=%v", got, err)
	}
}
