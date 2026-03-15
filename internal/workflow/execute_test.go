package workflow

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestExecute_SimpleEchoStep(t *testing.T) {
	w := &Workflow{
		Name: "test-echo",
		Steps: []Step{
			{Name: "greet", Command: "echo hello"},
		},
	}

	result, err := Execute(context.Background(), w, nil, ExecuteOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Error("expected success")
	}
	if len(result.Steps) != 1 {
		t.Fatalf("expected 1 step result, got %d", len(result.Steps))
	}
	if got := strings.TrimSpace(result.Steps[0].Stdout); got != "hello" {
		t.Errorf("stdout = %q, want %q", got, "hello")
	}
}

func TestExecute_FailingStep(t *testing.T) {
	w := &Workflow{
		Name: "test-fail",
		Steps: []Step{
			{Name: "succeed", Command: "true"},
			{Name: "fail", Command: "false"},
			{Name: "never", Command: "echo should-not-run"},
		},
	}

	result, err := Execute(context.Background(), w, nil, ExecuteOptions{})
	if err == nil {
		t.Fatal("expected error for failing step")
	}
	if result.Success {
		t.Error("expected failure")
	}
	// The third step should not have been executed.
	if len(result.Steps) != 2 {
		t.Errorf("expected 2 step results, got %d", len(result.Steps))
	}
}

func TestExecute_ContinueOnError(t *testing.T) {
	w := &Workflow{
		Name: "test-continue",
		Steps: []Step{
			{Name: "fail", Command: "false", ContinueOn: "error"},
			{Name: "runs-anyway", Command: "echo still-running"},
		},
	}

	result, err := Execute(context.Background(), w, nil, ExecuteOptions{})
	// ContinueOn=error means the workflow continues but still reports failure.
	if err == nil {
		t.Fatal("expected error from failing step")
	}
	if len(result.Steps) != 2 {
		t.Fatalf("expected 2 step results, got %d", len(result.Steps))
	}
	if result.Steps[1].Skipped {
		t.Error("second step should not be skipped")
	}
	if got := strings.TrimSpace(result.Steps[1].Stdout); got != "still-running" {
		t.Errorf("stdout = %q, want %q", got, "still-running")
	}
}

func TestExecute_DryRun(t *testing.T) {
	var stderr bytes.Buffer
	w := &Workflow{
		Name: "test-dry",
		Steps: []Step{
			{Name: "build", Command: "make build"},
			{Name: "deploy", Command: "make deploy"},
		},
	}

	result, err := Execute(context.Background(), w, nil, ExecuteOptions{
		DryRun: true,
		Stderr: &stderr,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Error("expected success in dry-run mode")
	}
	// All steps should be marked as skipped in dry-run.
	for _, sr := range result.Steps {
		if !sr.Skipped {
			t.Errorf("step %q should be skipped in dry-run", sr.Name)
		}
	}
	// Stderr should contain dry-run output.
	if !strings.Contains(stderr.String(), "[dry-run]") {
		t.Errorf("stderr should contain dry-run messages, got: %s", stderr.String())
	}
}

func TestExecute_ConditionSkipped(t *testing.T) {
	w := &Workflow{
		Name: "test-cond",
		Steps: []Step{
			{Name: "conditional", Command: "echo should-not-run", Condition: "DEPLOY_ENABLED"},
		},
	}

	result, err := Execute(context.Background(), w, map[string]string{}, ExecuteOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Steps) != 1 {
		t.Fatalf("expected 1 step result, got %d", len(result.Steps))
	}
	if !result.Steps[0].Skipped {
		t.Error("step should be skipped when condition is falsy")
	}
}

func TestExecute_ConditionMet(t *testing.T) {
	w := &Workflow{
		Name: "test-cond-true",
		Steps: []Step{
			{Name: "conditional", Command: "echo ran", Condition: "DEPLOY_ENABLED"},
		},
	}

	result, err := Execute(context.Background(), w, map[string]string{"DEPLOY_ENABLED": "true"}, ExecuteOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Steps[0].Skipped {
		t.Error("step should not be skipped when condition is truthy")
	}
	if got := strings.TrimSpace(result.Steps[0].Stdout); got != "ran" {
		t.Errorf("stdout = %q, want %q", got, "ran")
	}
}

func TestExecute_VariableInterpolation(t *testing.T) {
	w := &Workflow{
		Name: "test-vars",
		Steps: []Step{
			{Name: "greet", Command: "echo {{ .GREETING }}"},
		},
	}

	params := map[string]string{"GREETING": "hello-world"}
	result, err := Execute(context.Background(), w, params, ExecuteOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := strings.TrimSpace(result.Steps[0].Stdout); got != "hello-world" {
		t.Errorf("stdout = %q, want %q", got, "hello-world")
	}
}

func TestExecute_BeforeAll(t *testing.T) {
	w := &Workflow{
		Name: "test-hooks",
		BeforeAll: []Step{
			{Name: "setup", Command: "echo before"},
		},
		Steps: []Step{
			{Name: "main", Command: "echo main"},
		},
	}

	result, err := Execute(context.Background(), w, nil, ExecuteOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Steps) != 2 {
		t.Fatalf("expected 2 step results, got %d", len(result.Steps))
	}
	if got := strings.TrimSpace(result.Steps[0].Stdout); got != "before" {
		t.Errorf("before_all stdout = %q, want %q", got, "before")
	}
}

func TestExecute_AfterAllRunsOnFailure(t *testing.T) {
	w := &Workflow{
		Name: "test-afterall",
		Steps: []Step{
			{Name: "fail", Command: "false"},
		},
		AfterAll: []Step{
			{Name: "cleanup", Command: "echo cleanup"},
		},
	}

	result, err := Execute(context.Background(), w, nil, ExecuteOptions{})
	if err == nil {
		t.Fatal("expected error")
	}
	// AfterAll should still have run.
	found := false
	for _, sr := range result.Steps {
		if sr.Name == "cleanup" {
			found = true
			if got := strings.TrimSpace(sr.Stdout); got != "cleanup" {
				t.Errorf("after_all stdout = %q, want %q", got, "cleanup")
			}
		}
	}
	if !found {
		t.Error("after_all step should have run even on failure")
	}
}

func TestExecute_OnErrorRunsOnFailure(t *testing.T) {
	w := &Workflow{
		Name: "test-onerror",
		Steps: []Step{
			{Name: "fail", Command: "false"},
		},
		OnError: []Step{
			{Name: "notify", Command: "echo error-handler"},
		},
	}

	result, err := Execute(context.Background(), w, nil, ExecuteOptions{})
	if err == nil {
		t.Fatal("expected error")
	}
	found := false
	for _, sr := range result.Steps {
		if sr.Name == "notify" {
			found = true
			if got := strings.TrimSpace(sr.Stdout); got != "error-handler" {
				t.Errorf("on_error stdout = %q, want %q", got, "error-handler")
			}
		}
	}
	if !found {
		t.Error("on_error step should have run on failure")
	}
}

func TestExecute_RequiredParamMissing(t *testing.T) {
	w := &Workflow{
		Name: "test-params",
		Params: []Param{
			{Name: "VERSION", Required: true},
		},
		Steps: []Step{
			{Name: "deploy", Command: "echo {{ .VERSION }}"},
		},
	}

	_, err := Execute(context.Background(), w, map[string]string{}, ExecuteOptions{})
	if err == nil {
		t.Fatal("expected error for missing required param")
	}
	if !strings.Contains(err.Error(), "required parameter") {
		t.Errorf("error should mention required parameter, got: %v", err)
	}
}

func TestExecute_ParamDefaults(t *testing.T) {
	w := &Workflow{
		Name: "test-defaults",
		Params: []Param{
			{Name: "ENV", Required: false, Default: "staging"},
		},
		Steps: []Step{
			{Name: "deploy", Command: "echo {{ .ENV }}"},
		},
	}

	result, err := Execute(context.Background(), w, map[string]string{}, ExecuteOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := strings.TrimSpace(result.Steps[0].Stdout); got != "staging" {
		t.Errorf("stdout = %q, want %q", got, "staging")
	}
}

func TestExecute_ParamOverridesDefault(t *testing.T) {
	w := &Workflow{
		Name: "test-override",
		Params: []Param{
			{Name: "ENV", Required: false, Default: "staging"},
		},
		Steps: []Step{
			{Name: "deploy", Command: "echo {{ .ENV }}"},
		},
	}

	result, err := Execute(context.Background(), w, map[string]string{"ENV": "production"}, ExecuteOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := strings.TrimSpace(result.Steps[0].Stdout); got != "production" {
		t.Errorf("stdout = %q, want %q", got, "production")
	}
}

func TestExecute_EnvMerge(t *testing.T) {
	w := &Workflow{
		Name: "test-env",
		Env:  map[string]string{"GREETING": "hello"},
		Steps: []Step{
			{Name: "greet", Command: "echo {{ .GREETING }}"},
		},
	}

	result, err := Execute(context.Background(), w, nil, ExecuteOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := strings.TrimSpace(result.Steps[0].Stdout); got != "hello" {
		t.Errorf("stdout = %q, want %q", got, "hello")
	}
}

func TestExecute_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	w := &Workflow{
		Name: "test-cancel",
		Steps: []Step{
			{Name: "slow", Command: "sleep 60"},
		},
	}

	result, err := Execute(ctx, w, nil, ExecuteOptions{})
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
	if result.Success {
		t.Error("expected failure")
	}
}
