//go:build integration

package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/tamtom/play-console-cli/internal/config"
)

func skipUnlessIntegration(t *testing.T) {
	t.Helper()
	if v := os.Getenv("GPLAY_INTEGRATION_TEST"); v != "1" && v != "true" {
		t.Skip("skipping integration test; set GPLAY_INTEGRATION_TEST=1")
	}
}

func requireServiceAccount(t *testing.T) string {
	t.Helper()
	path := os.Getenv("GPLAY_SERVICE_ACCOUNT")
	if path == "" {
		t.Skip("skipping: GPLAY_SERVICE_ACCOUNT not set")
	}
	return path
}

func isolateConfig(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	t.Setenv("GPLAY_CONFIG_PATH", configPath)
	return configPath
}

// --- Auth Init ---

func TestIntegration_AuthInit_CreatesConfig(t *testing.T) {
	skipUnlessIntegration(t)

	// auth init writes to GlobalPath (~/.gplay/config.json), not GPLAY_CONFIG_PATH.
	// Use --force since the global config likely already exists.
	cmd := AuthInitCommand()
	err := cmd.ParseAndRun(context.Background(), []string{"--force"})
	if err != nil {
		t.Fatalf("auth init failed: %v", err)
	}

	// Verify global config file exists and is valid JSON
	globalPath, err := config.GlobalPath()
	if err != nil {
		t.Fatalf("getting global path: %v", err)
	}

	data, err := os.ReadFile(globalPath)
	if err != nil {
		t.Fatalf("reading config: %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("config is not valid JSON: %v", err)
	}
	t.Logf("config created: %s (%d bytes)", globalPath, len(data))
}

// --- Auth Login ---

func TestIntegration_AuthLogin_StoresCredentials(t *testing.T) {
	skipUnlessIntegration(t)
	saPath := requireServiceAccount(t)
	configPath := isolateConfig(t)

	// Init first
	initCmd := AuthInitCommand()
	if err := initCmd.ParseAndRun(context.Background(), []string{"--force"}); err != nil {
		t.Fatalf("auth init: %v", err)
	}

	// Login
	loginCmd := AuthLoginCommand()
	err := loginCmd.ParseAndRun(context.Background(), []string{
		"--service-account", saPath,
		"--profile", "test-profile",
	})
	if err != nil {
		t.Fatalf("auth login: %v", err)
	}

	// Verify profile was stored
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("reading config: %v", err)
	}

	var cfg map[string]interface{}
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("parsing config: %v", err)
	}

	profiles, ok := cfg["profiles"].([]interface{})
	if !ok || len(profiles) == 0 {
		t.Fatal("expected at least one profile in config")
	}

	t.Logf("stored %d profile(s)", len(profiles))
}

// --- Auth Status ---

func TestIntegration_AuthStatus_ShowsProfile(t *testing.T) {
	skipUnlessIntegration(t)
	saPath := requireServiceAccount(t)
	isolateConfig(t)

	// Setup: init + login
	initCmd := AuthInitCommand()
	if err := initCmd.ParseAndRun(context.Background(), []string{"--force"}); err != nil {
		t.Fatalf("auth init: %v", err)
	}
	loginCmd := AuthLoginCommand()
	if err := loginCmd.ParseAndRun(context.Background(), []string{
		"--service-account", saPath,
	}); err != nil {
		t.Fatalf("auth login: %v", err)
	}

	// Capture status output
	statusCmd := AuthStatusCommand()
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := statusCmd.ParseAndRun(context.Background(), []string{"--output", "json"})

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatalf("auth status: %v", err)
	}

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(r); err != nil {
		t.Fatalf("reading output: %v", err)
	}

	var status map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &status); err != nil {
		t.Fatalf("parsing status JSON: %v\noutput: %s", err, buf.String())
	}

	if status["config_path"] == nil || status["config_path"] == "" {
		t.Error("expected config_path in status output")
	}

	// Only log that status returned valid JSON, not the contents (may include key paths)
	t.Logf("status returned valid JSON (%d bytes)", buf.Len())
}

// --- Auth Doctor ---

func TestIntegration_AuthDoctor_Passes(t *testing.T) {
	skipUnlessIntegration(t)
	saPath := requireServiceAccount(t)
	isolateConfig(t)

	// Setup: init + login
	initCmd := AuthInitCommand()
	if err := initCmd.ParseAndRun(context.Background(), []string{"--force"}); err != nil {
		t.Fatalf("auth init: %v", err)
	}
	loginCmd := AuthLoginCommand()
	if err := loginCmd.ParseAndRun(context.Background(), []string{
		"--service-account", saPath,
	}); err != nil {
		t.Fatalf("auth login: %v", err)
	}

	// Doctor should pass
	doctorCmd := AuthDoctorCommand()
	err := doctorCmd.ParseAndRun(context.Background(), []string{})
	if err != nil {
		t.Fatalf("auth doctor failed: %v", err)
	}
}

// --- Auth Logout ---

func TestIntegration_AuthLogout_RemovesProfile(t *testing.T) {
	skipUnlessIntegration(t)
	saPath := requireServiceAccount(t)
	configPath := isolateConfig(t)

	// Setup: init + login
	initCmd := AuthInitCommand()
	if err := initCmd.ParseAndRun(context.Background(), []string{"--force"}); err != nil {
		t.Fatalf("auth init: %v", err)
	}
	loginCmd := AuthLoginCommand()
	if err := loginCmd.ParseAndRun(context.Background(), []string{
		"--service-account", saPath,
		"--profile", "to-remove",
	}); err != nil {
		t.Fatalf("auth login: %v", err)
	}

	// Logout
	logoutCmd := AuthLogoutCommand()
	err := logoutCmd.ParseAndRun(context.Background(), []string{
		"--profile", "to-remove",
		"--confirm",
	})
	if err != nil {
		t.Fatalf("auth logout: %v", err)
	}

	// Verify profile removed
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("reading config: %v", err)
	}

	var cfg map[string]interface{}
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("parsing config: %v", err)
	}

	profiles, _ := cfg["profiles"].([]interface{})
	for _, p := range profiles {
		profile, _ := p.(map[string]interface{})
		if profile["name"] == "to-remove" {
			t.Error("profile 'to-remove' should have been deleted")
		}
	}
}
