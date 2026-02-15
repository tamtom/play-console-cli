package initcmd

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestInitCommand_CreatesConfig(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Chdir(origDir); err != nil {
			t.Log(err)
		}
	}()

	cmd := InitCommand()
	var buf bytes.Buffer

	err := cmd.ParseAndRun(context.Background(), []string{"--package", "com.test.app"})
	if err != nil {
		t.Fatalf("init failed: %v", err)
	}
	_ = buf

	configPath := filepath.Join(dir, ".gplay", "config.yaml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("reading config: %v", err)
	}
	if !bytes.Contains(data, []byte("com.test.app")) {
		t.Error("config should contain package name")
	}
}

func TestInitCommand_ExistingConfig(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Chdir(origDir); err != nil {
			t.Log(err)
		}
	}()

	if err := os.MkdirAll(".gplay", 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(".gplay/config.yaml", []byte("existing"), 0600); err != nil {
		t.Fatal(err)
	}

	cmd := InitCommand()
	err := cmd.ParseAndRun(context.Background(), []string{})
	if err == nil {
		t.Error("expected error for existing config")
	}
}

func TestInitCommand_Force(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Chdir(origDir); err != nil {
			t.Log(err)
		}
	}()

	if err := os.MkdirAll(".gplay", 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(".gplay/config.yaml", []byte("existing"), 0600); err != nil {
		t.Fatal(err)
	}

	cmd := InitCommand()
	err := cmd.ParseAndRun(context.Background(), []string{"--force"})
	if err != nil {
		t.Fatalf("init --force failed: %v", err)
	}
}

func TestGenerateConfig(t *testing.T) {
	cfg := generateConfig("com.test.app", "/path/to/key.json", "30s")
	if cfg == "" {
		t.Error("expected non-empty config")
	}
}
