package workflow

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestExecute_RetryEventuallySucceedsWithFreshAttemptOutput(t *testing.T) {
	counter := filepath.Join(t.TempDir(), "attempts")
	command := fmt.Sprintf(
		`count=$(cat %q 2>/dev/null || echo 0); count=$((count+1)); echo "$count" > %q; if [ "$count" -lt 3 ]; then echo "stale-$count"; exit 1; fi; echo fresh`,
		counter,
		counter,
	)
	w := &Workflow{
		Name: "retry",
		Steps: []Step{{
			Name:  "eventual",
			Run:   command,
			Retry: &RetryPolicy{MaxAttempts: 3, Delay: "1ms"},
		}},
	}
	var diagnostics bytes.Buffer

	result, err := Execute(context.Background(), w, nil, ExecuteOptions{
		StatePath: filepath.Join(t.TempDir(), "state.json"),
		Stderr:    &diagnostics,
	})
	if err != nil {
		t.Fatalf("retry workflow failed: %v", err)
	}
	step := result.Steps[0]
	if got := strings.TrimSpace(step.Stdout); got != "fresh" {
		t.Fatalf("final stdout = %q, want only successful-attempt output", got)
	}
	if len(step.Attempts) != 3 {
		t.Fatalf("attempts = %#v, want 3", step.Attempts)
	}
	if step.Attempts[0].Status != "error" || step.Attempts[1].Status != "error" || step.Attempts[2].Status != "ok" {
		t.Fatalf("unexpected attempt statuses: %#v", step.Attempts)
	}
	if !strings.Contains(diagnostics.String(), "attempt 2/3") {
		t.Fatalf("retry diagnostics missing attempt count: %s", diagnostics.String())
	}
}

func TestExecute_TimeoutHasStableStructuredFailure(t *testing.T) {
	timeout := "25ms"
	w := &Workflow{
		Name: "timeout",
		Steps: []Step{{
			Name:    "slow-release",
			Run:     "sleep 5",
			Timeout: &timeout,
		}},
	}
	started := time.Now()

	result, err := Execute(context.Background(), w, nil, ExecuteOptions{
		StatePath: filepath.Join(t.TempDir(), "state.json"),
	})
	if err == nil {
		t.Fatal("expected timeout failure")
	}
	if elapsed := time.Since(started); elapsed > time.Second {
		t.Fatalf("timeout took %s, process tree was not terminated", elapsed)
	}
	step := result.Steps[0]
	if step.FailureReason != "timeout" || len(step.Attempts) != 1 {
		t.Fatalf("unexpected timeout result: %#v", step)
	}
	if step.Attempts[0].Status != "timeout" || step.Attempts[0].Error != "step timed out after 25ms" {
		t.Fatalf("unexpected timeout attempt: %#v", step.Attempts[0])
	}
}

func TestExecute_TimeoutWithoutRetryCannotResumeAmbiguousMutation(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, "state.json")
	marker := filepath.Join(dir, "attempts")
	timeout := "25ms"
	command := fmt.Sprintf(`echo attempt >> %q; sleep 5`, marker)
	w := &Workflow{
		Name:  "ambiguous",
		Steps: []Step{{Name: "publish", Run: command, Timeout: &timeout}},
	}

	if _, err := Execute(context.Background(), w, nil, ExecuteOptions{StatePath: statePath}); err == nil {
		t.Fatal("expected first run to time out")
	}
	_, err := Execute(context.Background(), w, nil, ExecuteOptions{StatePath: statePath, Resume: true})
	if err == nil || !strings.Contains(err.Error(), "cannot be resumed safely") {
		t.Fatalf("expected ambiguity-safe resume rejection, got %v", err)
	}
	data, readErr := os.ReadFile(marker)
	if readErr != nil {
		t.Fatalf("read marker: %v", readErr)
	}
	if attempts := strings.Count(string(data), "attempt"); attempts != 1 {
		t.Fatalf("resume re-executed ambiguous command %d times", attempts)
	}
}

func TestExecute_RetryExhaustionCanResumeWithAttemptHistory(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, "state.json")
	counter := filepath.Join(dir, "attempts")
	command := fmt.Sprintf(
		`count=$(cat %q 2>/dev/null || echo 0); count=$((count+1)); echo "$count" > %q; if [ "$count" -lt 3 ]; then exit 1; fi; echo recovered`,
		counter,
		counter,
	)
	w := &Workflow{
		Name: "resume-retry",
		Steps: []Step{{
			Name:  "eventual",
			Run:   command,
			Retry: &RetryPolicy{MaxAttempts: 2, Delay: "1ms"},
		}},
	}
	if _, err := Execute(context.Background(), w, nil, ExecuteOptions{StatePath: statePath}); err == nil {
		t.Fatal("expected first invocation to exhaust retries")
	}

	result, err := Execute(context.Background(), w, nil, ExecuteOptions{StatePath: statePath, Resume: true})
	if err != nil {
		t.Fatalf("resume failed: %v", err)
	}
	step := result.Steps[0]
	if got := strings.TrimSpace(step.Stdout); got != "recovered" {
		t.Fatalf("stdout = %q", got)
	}
	if len(step.Attempts) != 3 {
		t.Fatalf("attempt history = %#v, want 3 attempts", step.Attempts)
	}
	if step.Attempts[0].Invocation != 1 || step.Attempts[1].Invocation != 1 || step.Attempts[2].Invocation != 2 {
		t.Fatalf("attempt invocations = %#v", step.Attempts)
	}
}

func TestExecute_OutputExtractionFailureIsTerminalEvenWithRetry(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, "state.json")
	marker := filepath.Join(dir, "attempts")
	w := &Workflow{
		Name: "output-terminal",
		Steps: []Step{{
			Name:    "publish",
			Run:     fmt.Sprintf(`echo attempt >> %q; echo not-json`, marker),
			Retry:   &RetryPolicy{MaxAttempts: 3, Delay: "1ms"},
			Outputs: map[string]string{"id": "$.id"},
		}},
	}

	result, err := Execute(context.Background(), w, nil, ExecuteOptions{StatePath: statePath})
	if err == nil {
		t.Fatal("expected output extraction failure")
	}
	step := result.Steps[0]
	if step.FailureReason != "output_error" || len(step.Attempts) != 1 || step.Attempts[0].FailureReason != "output_error" {
		t.Fatalf("unexpected terminal output failure: %#v", step)
	}
	if _, resumeErr := Execute(context.Background(), w, nil, ExecuteOptions{StatePath: statePath, Resume: true}); resumeErr == nil {
		t.Fatal("expected output-error resume rejection")
	}
	data, readErr := os.ReadFile(marker)
	if readErr != nil {
		t.Fatalf("read marker: %v", readErr)
	}
	if attempts := strings.Count(string(data), "attempt"); attempts != 1 {
		t.Fatalf("terminal output error executed command %d times", attempts)
	}
}

func TestExecute_StatePersistenceFailureIsReturned(t *testing.T) {
	dir := t.TempDir()
	parentFile := filepath.Join(dir, "not-a-directory")
	if err := os.WriteFile(parentFile, []byte("occupied"), 0o600); err != nil {
		t.Fatalf("write parent file: %v", err)
	}
	w := &Workflow{Name: "persist", Steps: []Step{{Name: "done", Run: "true"}}}

	result, err := Execute(context.Background(), w, nil, ExecuteOptions{
		StatePath: filepath.Join(parentFile, "state.json"),
	})
	if err == nil || !strings.Contains(err.Error(), "persist workflow state") {
		t.Fatalf("expected state persistence error, got %v", err)
	}
	if result == nil || result.Success {
		t.Fatalf("persistence failure must make the run unsuccessful: %#v", result)
	}
}

func TestExecute_ResumeRequiresExistingMatchingState(t *testing.T) {
	dir := t.TempDir()
	marker := filepath.Join(dir, "marker")
	w := &Workflow{Name: "resume", Steps: []Step{{Name: "publish", Run: fmt.Sprintf("touch %q", marker)}}}

	_, err := Execute(context.Background(), w, nil, ExecuteOptions{
		Resume:    true,
		StatePath: filepath.Join(dir, "missing-state.json"),
	})
	if err == nil || !strings.Contains(err.Error(), "no saved workflow state") {
		t.Fatalf("expected missing resume-state error, got %v", err)
	}
	if _, statErr := os.Stat(marker); !os.IsNotExist(statErr) {
		t.Fatalf("resume without state executed a command: %v", statErr)
	}
}

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
		t.Fatal("expected error for canceled context")
	}
	if result.Success {
		t.Error("expected failure")
	}
}

func TestExecuteDefinition_ReferencedWorkflowWithOutputs(t *testing.T) {
	def := &Definition{
		Workflows: map[string]Workflow{
			"prepare": {
				Name: "prepare",
				Steps: []Step{
					{
						Name:    "capture",
						Run:     `printf '{"version":"42"}'`,
						Outputs: map[string]string{"version": "$.version"},
					},
				},
			},
			"deploy": {
				Name: "deploy",
				Steps: []Step{
					{Name: "prepare", Workflow: "prepare"},
					{Name: "publish", Run: "echo {{ .capture.version }}"},
				},
			},
		},
	}

	result, err := ExecuteDefinition(context.Background(), def, "deploy", nil, ExecuteOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Steps) < 3 {
		t.Fatalf("expected nested workflow steps to be recorded, got %d", len(result.Steps))
	}
	if got := strings.TrimSpace(result.Steps[len(result.Steps)-1].Stdout); got != "42" {
		t.Fatalf("stdout = %q, want %q", got, "42")
	}
	if result.Outputs["capture.version"] != "42" {
		t.Fatalf("expected capture.version output, got %#v", result.Outputs)
	}
}

func TestExecuteDefinition_WorkflowWithInterpolatedParams(t *testing.T) {
	def := &Definition{
		Workflows: map[string]Workflow{
			"prepare": {
				Name: "prepare",
				Steps: []Step{
					{
						Name:    "capture",
						Run:     `printf '{"track":"beta"}'`,
						Outputs: map[string]string{"track": "$.track"},
					},
				},
			},
			"publish": {
				Name: "publish",
				Params: []Param{
					{Name: "TRACK", Required: true},
				},
				Steps: []Step{
					{Name: "release", Run: "echo {{ .TRACK }}"},
				},
			},
			"deploy": {
				Name: "deploy",
				Steps: []Step{
					{Name: "prepare", Workflow: "prepare"},
					{Name: "publish", Workflow: "publish", With: map[string]string{"TRACK": "{{ .capture.track }}"}},
				},
			},
		},
	}

	result, err := ExecuteDefinition(context.Background(), def, "deploy", nil, ExecuteOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found := false
	for _, step := range result.Steps {
		if step.Path == "deploy.publish.release" {
			found = true
			if got := strings.TrimSpace(step.Stdout); got != "beta" {
				t.Fatalf("stdout = %q, want %q", got, "beta")
			}
		}
	}
	if !found {
		t.Fatal("expected nested publish step result")
	}
}

func TestExecuteDefinition_ResumeUsesNestedPaths(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "workflow-state.json")
	def := &Definition{
		Workflows: map[string]Workflow{
			"child": {
				Name: "child",
				Steps: []Step{
					{Name: "first", Run: "echo child"},
				},
			},
			"root": {
				Name: "root",
				Steps: []Step{
					{Name: "child", Workflow: "child"},
					{Name: "final", Run: "echo final"},
				},
			},
		},
	}

	if _, err := ExecuteDefinition(context.Background(), def, "root", nil, ExecuteOptions{StatePath: statePath}); err != nil {
		t.Fatalf("unexpected first run error: %v", err)
	}

	result, err := ExecuteDefinition(context.Background(), def, "root", nil, ExecuteOptions{
		Resume:    true,
		StatePath: statePath,
	})
	if err != nil {
		t.Fatalf("unexpected resume error: %v", err)
	}

	skipped := map[string]bool{}
	for _, step := range result.Steps {
		if step.Skipped {
			skipped[step.Path] = true
		}
	}
	if !skipped["root.child.first"] || !skipped["root.final"] {
		t.Fatalf("expected nested resume skips, got %#v", skipped)
	}
}

func TestExecuteDefinition_ResumeRejectsChangedDefinitionBeforeExecution(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, "workflow-state.json")
	marker := filepath.Join(dir, "marker")
	def := &Definition{Workflows: map[string]Workflow{
		"release": {Steps: []Step{{Name: "publish", Run: fmt.Sprintf("echo original > %q", marker)}}},
	}}
	if _, err := ExecuteDefinition(context.Background(), def, "release", nil, ExecuteOptions{StatePath: statePath}); err != nil {
		t.Fatalf("first run failed: %v", err)
	}

	changed := &Definition{Workflows: map[string]Workflow{
		"release": {Steps: []Step{{Name: "publish", Run: fmt.Sprintf("echo changed > %q", marker)}}},
	}}
	_, err := ExecuteDefinition(context.Background(), changed, "release", nil, ExecuteOptions{
		Resume:    true,
		StatePath: statePath,
	})
	if err == nil || !strings.Contains(err.Error(), "definition changed") {
		t.Fatalf("expected definition fingerprint error, got %v", err)
	}
	data, readErr := os.ReadFile(marker)
	if readErr != nil {
		t.Fatalf("read marker: %v", readErr)
	}
	if got := strings.TrimSpace(string(data)); got != "original" {
		t.Fatalf("marker changed during rejected resume: %q", got)
	}
}
