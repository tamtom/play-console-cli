package cmdtest_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWorkflow_Help(t *testing.T) {
	stdout, stderr, err := runCommand(t, "workflow")
	if err == nil {
		t.Fatal("expected help error for workflow root")
	}
	combined := stdout + stderr
	for _, token := range []string{"workflow", "run", "validate", "list"} {
		if !strings.Contains(combined, token) {
			t.Fatalf("workflow help should mention %q, got %q", token, combined)
		}
	}
}

func TestWorkflowValidate_PrintsStructuredErrors(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "release.json")
	content := `{
		"workflows": {
			"publish": {
				"steps": [
					{"name": "release", "workflow": "missing"}
				]
			}
		}
	}`
	if err := os.WriteFile(filePath, []byte(content), 0o600); err != nil {
		t.Fatalf("write workflow file: %v", err)
	}

	stdout, stderr, err := runCommand(t, "workflow", "validate", filePath)
	if err == nil {
		t.Fatal("expected validation error")
	}
	if stderr != "" {
		t.Fatalf("expected structured validation output on stdout only, stderr=%q", stderr)
	}

	var result struct {
		Valid  bool `json:"valid"`
		Errors []struct {
			Code string `json:"code"`
		} `json:"errors"`
	}
	if decodeErr := json.Unmarshal([]byte(stdout), &result); decodeErr != nil {
		t.Fatalf("failed to decode workflow validation output: %v\nstdout: %s", decodeErr, stdout)
	}
	if result.Valid {
		t.Fatal("expected invalid workflow")
	}
	if len(result.Errors) == 0 || result.Errors[0].Code != "workflow_not_found" {
		t.Fatalf("unexpected validation errors: %#v", result.Errors)
	}
}

func TestWorkflowRun_DryRunSeparatesStdoutAndStderr(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "release.json")
	content := `{
		"workflows": {
			"publish": {
				"params": [{"name": "TRACK", "required": true}],
				"steps": [
					{"name": "echo-track", "run": "echo {{ .TRACK }}"}
				]
			}
		}
	}`
	if err := os.WriteFile(filePath, []byte(content), 0o600); err != nil {
		t.Fatalf("write workflow file: %v", err)
	}

	stdout, stderr, err := runCommand(
		t,
		"workflow", "run",
		"--dry-run",
		"--workflow", "publish",
		"--param", "TRACK=internal",
		filePath,
	)
	if err != nil {
		t.Fatalf("unexpected dry-run error: %v", err)
	}
	if !strings.Contains(stderr, "[dry-run] step publish.echo-track") {
		t.Fatalf("expected dry-run log on stderr, got %q", stderr)
	}

	var result struct {
		Workflow string `json:"workflow"`
		Success  bool   `json:"success"`
		Steps    []struct {
			Path    string `json:"path"`
			Skipped bool   `json:"skipped"`
		} `json:"steps"`
	}
	if decodeErr := json.Unmarshal([]byte(stdout), &result); decodeErr != nil {
		t.Fatalf("failed to decode workflow run output: %v\nstdout: %s", decodeErr, stdout)
	}
	if result.Workflow != "publish" || !result.Success {
		t.Fatalf("unexpected workflow run result: %#v", result)
	}
	if len(result.Steps) != 1 || result.Steps[0].Path != "publish.echo-track" || !result.Steps[0].Skipped {
		t.Fatalf("unexpected dry-run steps: %#v", result.Steps)
	}
}

func TestWorkflowList_ListsNamedWorkflowsFromFile(t *testing.T) {
	dir := t.TempDir()
	content := `{
		"workflows": {
			"preflight": {"steps": [{"name": "validate", "run": "echo ok"}]},
			"publish": {"steps": [{"name": "release", "run": "echo ok"}]}
		}
	}`
	if err := os.WriteFile(filepath.Join(dir, "release.json"), []byte(content), 0o600); err != nil {
		t.Fatalf("write workflow file: %v", err)
	}

	stdout, stderr, err := runCommand(t, "workflow", "list", "--dir", dir)
	if err != nil {
		t.Fatalf("unexpected list error: %v", err)
	}
	if stderr != "" {
		t.Fatalf("expected clean stderr, got %q", stderr)
	}

	var result []struct {
		Name string `json:"name"`
		File string `json:"file"`
	}
	if decodeErr := json.Unmarshal([]byte(stdout), &result); decodeErr != nil {
		t.Fatalf("failed to decode workflow list output: %v\nstdout: %s", decodeErr, stdout)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 workflows, got %#v", result)
	}
	if result[0].File != "release.json" || result[0].Name != "preflight" || result[1].Name != "publish" {
		t.Fatalf("unexpected workflow list result: %#v", result)
	}
}
