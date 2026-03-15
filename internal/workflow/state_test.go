package workflow

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSaveAndLoadState(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".gplay", "workflow-state.json")

	original := &ExecutionResult{
		Workflow: "deploy",
		Success:  true,
		Steps: []StepResult{
			{Name: "build", Command: "make build", ExitCode: 0, Stdout: "ok\n"},
			{Name: "test", Command: "make test", ExitCode: 0, Stdout: "passed\n"},
		},
		ElapsedTime: 5 * time.Second,
	}

	if err := SaveState(path, original); err != nil {
		t.Fatalf("SaveState: %v", err)
	}

	loaded, err := LoadState(path)
	if err != nil {
		t.Fatalf("LoadState: %v", err)
	}
	if loaded == nil {
		t.Fatal("LoadState returned nil")
	}

	if loaded.Workflow != original.Workflow {
		t.Errorf("Workflow = %q, want %q", loaded.Workflow, original.Workflow)
	}
	if loaded.Success != original.Success {
		t.Errorf("Success = %v, want %v", loaded.Success, original.Success)
	}
	if len(loaded.Steps) != len(original.Steps) {
		t.Fatalf("Steps count = %d, want %d", len(loaded.Steps), len(original.Steps))
	}
	for i, step := range loaded.Steps {
		if step.Name != original.Steps[i].Name {
			t.Errorf("Step[%d].Name = %q, want %q", i, step.Name, original.Steps[i].Name)
		}
		if step.ExitCode != original.Steps[i].ExitCode {
			t.Errorf("Step[%d].ExitCode = %d, want %d", i, step.ExitCode, original.Steps[i].ExitCode)
		}
	}
}

func TestLoadState_FileNotFound(t *testing.T) {
	result, err := LoadState(filepath.Join(t.TempDir(), "nonexistent.json"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Error("expected nil result for nonexistent file")
	}
}

func TestLoadState_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(path, []byte("{invalid}"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	_, err := LoadState(path)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestSaveState_NilResult(t *testing.T) {
	err := SaveState(filepath.Join(t.TempDir(), "state.json"), nil)
	if err == nil {
		t.Fatal("expected error for nil result")
	}
}

func TestSaveState_CreatesDirectory(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "deep", "state.json")

	result := &ExecutionResult{
		Workflow: "test",
		Success:  true,
		Steps:    []StepResult{},
	}

	if err := SaveState(path, result); err != nil {
		t.Fatalf("SaveState: %v", err)
	}

	// Verify file exists.
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("state file not created: %v", err)
	}
}

func TestSaveState_AtomicWrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")

	result := &ExecutionResult{
		Workflow: "atomic-test",
		Success:  true,
		Steps:    []StepResult{},
	}

	if err := SaveState(path, result); err != nil {
		t.Fatalf("SaveState: %v", err)
	}

	// Ensure no .tmp file is left behind.
	tmpPath := path + ".tmp"
	if _, err := os.Stat(tmpPath); !os.IsNotExist(err) {
		t.Error("temp file should not exist after successful save")
	}
}
