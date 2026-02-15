package initcmd

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInitCommand_CreatesConfig(t *testing.T) {
	dir := t.TempDir()
	origDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(dir))
	t.Cleanup(func() { require.NoError(t, os.Chdir(origDir)) })

	cmd := InitCommand()
	var buf bytes.Buffer

	err = cmd.ParseAndRun(context.Background(), []string{"--package", "com.test.app"})
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
	origDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(dir))
	t.Cleanup(func() { require.NoError(t, os.Chdir(origDir)) })

	require.NoError(t, os.MkdirAll(".gplay", 0700))
	require.NoError(t, os.WriteFile(".gplay/config.yaml", []byte("existing"), 0600))

	cmd := InitCommand()
	err = cmd.ParseAndRun(context.Background(), []string{})
	if err == nil {
		t.Error("expected error for existing config")
	}
}

func TestInitCommand_Force(t *testing.T) {
	dir := t.TempDir()
	origDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(dir))
	t.Cleanup(func() { require.NoError(t, os.Chdir(origDir)) })

	require.NoError(t, os.MkdirAll(".gplay", 0700))
	require.NoError(t, os.WriteFile(".gplay/config.yaml", []byte("existing"), 0600))

	cmd := InitCommand()
	err = cmd.ParseAndRun(context.Background(), []string{"--force"})
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
