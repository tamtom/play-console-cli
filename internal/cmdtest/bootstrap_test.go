package cmdtest_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBootstrapPlan_IsOfflineManualHandoff(t *testing.T) {
	artifactPath := filepath.Join(t.TempDir(), "app.aab")
	if err := os.WriteFile(artifactPath, []byte("offline-test-artifact"), 0o600); err != nil {
		t.Fatal(err)
	}

	stdout, stderr, err := runCommand(
		t,
		"bootstrap", "plan",
		"--package", "dev.example.safeapp",
		"--name", "Safe App",
		"--aab", artifactPath,
	)
	if err != nil {
		t.Fatalf("bootstrap plan failed: %v\nstderr: %s", err, stderr)
	}

	var plan struct {
		ID                    string `json:"id"`
		Status                string `json:"status"`
		ExecutesChanges       bool   `json:"executes_changes"`
		UsesPrivateInterfaces bool   `json:"uses_private_interfaces"`
		RequiresManualConsole bool   `json:"requires_manual_console"`
		Steps                 []struct {
			ID   string `json:"id"`
			Mode string `json:"mode"`
		} `json:"steps"`
	}
	if err := json.Unmarshal([]byte(stdout), &plan); err != nil {
		t.Fatalf("decode bootstrap plan: %v\nstdout: %s", err, stdout)
	}

	if plan.ID == "" {
		t.Error("plan ID must be populated")
	}
	if plan.Status != "manual_action_required" {
		t.Errorf("status = %q", plan.Status)
	}
	if plan.ExecutesChanges {
		t.Error("bootstrap plan must not execute changes")
	}
	if plan.UsesPrivateInterfaces {
		t.Error("bootstrap plan must not use private interfaces")
	}
	if !plan.RequiresManualConsole {
		t.Error("bootstrap plan must require a manual Console handoff")
	}

	wantManualSteps := map[string]bool{
		"create-app-record":         false,
		"upload-first-app-bundle":   false,
		"review-legal-declarations": false,
	}
	for _, step := range plan.Steps {
		if _, ok := wantManualSteps[step.ID]; ok {
			if step.Mode != "manual" {
				t.Errorf("%s mode = %q, want manual", step.ID, step.Mode)
			}
			wantManualSteps[step.ID] = true
		}
	}
	for id, found := range wantManualSteps {
		if !found {
			t.Errorf("missing manual step %q", id)
		}
	}
}

func TestBootstrapPlan_RequiresExistingAAB(t *testing.T) {
	_, _, err := runCommand(
		t,
		"bootstrap", "plan",
		"--package", "dev.example.safeapp",
		"--name", "Safe App",
		"--aab", filepath.Join(t.TempDir(), "missing.aab"),
	)
	if err == nil {
		t.Fatal("expected missing AAB error")
	}
	if !strings.Contains(err.Error(), "open --aab") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBootstrapPlan_TableOutput(t *testing.T) {
	artifactPath := filepath.Join(t.TempDir(), "app.aab")
	if err := os.WriteFile(artifactPath, []byte("offline-test-artifact"), 0o600); err != nil {
		t.Fatal(err)
	}

	stdout, stderr, err := runCommand(
		t,
		"bootstrap", "plan",
		"--package", "dev.example.safeapp",
		"--name", "Safe App",
		"--aab", artifactPath,
		"--output", "table",
	)
	if err != nil {
		t.Fatalf("bootstrap plan failed: %v\nstderr: %s", err, stderr)
	}
	if !strings.Contains(stdout, "create-app-record") || !strings.Contains(stdout, "manual") {
		t.Fatalf("table missing manual handoff: %s", stdout)
	}
}
