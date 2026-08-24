package config

import (
	"os"
	"path/filepath"
	"testing"
)

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
