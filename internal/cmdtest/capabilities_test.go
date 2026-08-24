package cmdtest_test

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestCapabilities_ReportsPolicySafeBoundaries(t *testing.T) {
	stdout, stderr, err := runCommand(t, "capabilities")
	if err != nil {
		t.Fatalf("capabilities failed: %v\nstderr: %s", err, stderr)
	}

	var capabilities []struct {
		ID       string `json:"id"`
		Status   string `json:"status"`
		Provider string `json:"provider"`
	}
	if err := json.Unmarshal([]byte(stdout), &capabilities); err != nil {
		t.Fatalf("decode capabilities output: %v\nstdout: %s", err, stdout)
	}

	want := map[string]string{
		"app.create":                 "manual",
		"console.private_automation": "unsupported",
	}
	for _, capability := range capabilities {
		if status, ok := want[capability.ID]; ok {
			if capability.Status != status {
				t.Errorf("%s status = %q, want %q", capability.ID, capability.Status, status)
			}
			delete(want, capability.ID)
		}
		if capability.Provider == "private" || capability.Provider == "web-experimental" {
			t.Errorf("unsafe provider exposed by %s: %q", capability.ID, capability.Provider)
		}
	}
	for id := range want {
		t.Errorf("missing capability %q", id)
	}
}

func TestCapabilities_FiltersManualWorkflows(t *testing.T) {
	stdout, stderr, err := runCommand(t, "capabilities", "--status", "manual")
	if err != nil {
		t.Fatalf("capabilities failed: %v\nstderr: %s", err, stderr)
	}

	var capabilities []struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal([]byte(stdout), &capabilities); err != nil {
		t.Fatalf("decode capabilities output: %v", err)
	}
	if len(capabilities) == 0 {
		t.Fatal("expected manual capabilities")
	}
	for _, capability := range capabilities {
		if capability.Status != "manual" {
			t.Fatalf("unexpected status %q", capability.Status)
		}
	}
}

func TestCapabilities_RejectsUnexpectedArguments(t *testing.T) {
	_, _, err := runCommand(t, "capabilities", "extra")
	if err == nil {
		t.Fatal("expected unexpected argument error")
	}
	if !strings.Contains(err.Error(), "unexpected arguments") {
		t.Fatalf("unexpected error: %v", err)
	}
}
