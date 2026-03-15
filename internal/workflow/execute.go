package workflow

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"
)

// ExecuteOptions configures workflow execution.
type ExecuteOptions struct {
	DryRun bool
	Resume bool
	Stdout io.Writer
	Stderr io.Writer
}

// ExecutionResult is the structured output of a workflow execution.
type ExecutionResult struct {
	Workflow    string        `json:"workflow"`
	Steps       []StepResult  `json:"steps"`
	Success     bool          `json:"success"`
	ElapsedTime time.Duration `json:"elapsed_time"`
}

// StepResult records one executed step.
type StepResult struct {
	Name     string `json:"name"`
	Command  string `json:"command"`
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	ExitCode int    `json:"exit_code"`
	Skipped  bool   `json:"skipped"`
}

// Execute runs a workflow with the given parameters.
// It runs each step via exec.CommandContext, capturing stdout/stderr per step.
// On first failure it stops (unless ContinueOn="error").
// BeforeAll steps run before main steps.
// AfterAll steps run after main steps (even on failure).
// OnError steps run when any main step fails.
func Execute(ctx context.Context, w *Workflow, params map[string]string, opts ExecuteOptions) (*ExecutionResult, error) {
	if opts.Stdout == nil {
		opts.Stdout = os.Stdout
	}
	if opts.Stderr == nil {
		opts.Stderr = os.Stderr
	}

	start := time.Now()
	result := &ExecutionResult{
		Workflow: w.Name,
		Steps:    make([]StepResult, 0),
		Success:  true,
	}

	// Merge env with params (params override).
	vars := mergeVars(w.Env, params)

	// Apply parameter defaults for any missing params.
	for _, p := range w.Params {
		if _, ok := vars[p.Name]; !ok {
			if p.Default != "" {
				vars[p.Name] = p.Default
			}
		}
	}

	// Validate required params.
	for _, p := range w.Params {
		if p.Required {
			if v, ok := vars[p.Name]; !ok || strings.TrimSpace(v) == "" {
				return nil, fmt.Errorf("required parameter %q is missing", p.Name)
			}
		}
	}

	// Load previously completed state if resuming.
	var completedSteps map[string]bool
	if opts.Resume {
		state, err := LoadState(".gplay/workflow-state.json")
		if err == nil && state != nil && state.Workflow == w.Name {
			completedSteps = make(map[string]bool)
			for _, sr := range state.Steps {
				if sr.ExitCode == 0 && !sr.Skipped {
					completedSteps[sr.Name] = true
				}
			}
		}
	}

	var mainErr error

	// Run BeforeAll steps.
	for _, step := range w.BeforeAll {
		sr := executeStep(ctx, step, vars, opts, completedSteps)
		result.Steps = append(result.Steps, sr)
		if sr.ExitCode != 0 && !sr.Skipped {
			result.Success = false
			mainErr = fmt.Errorf("before_all step %q failed with exit code %d", step.Name, sr.ExitCode)
			break
		}
	}

	// Run main steps if before_all succeeded.
	if mainErr == nil {
		for _, step := range w.Steps {
			sr := executeStep(ctx, step, vars, opts, completedSteps)
			result.Steps = append(result.Steps, sr)

			if sr.ExitCode != 0 && !sr.Skipped {
				result.Success = false
				mainErr = fmt.Errorf("step %q failed with exit code %d", step.Name, sr.ExitCode)
				if step.ContinueOn != "error" {
					break
				}
			}
		}
	}

	// Run OnError steps if there was a failure.
	if mainErr != nil {
		for _, step := range w.OnError {
			sr := executeStep(ctx, step, vars, opts, completedSteps)
			result.Steps = append(result.Steps, sr)
		}
	}

	// Run AfterAll steps (always, even on failure).
	for _, step := range w.AfterAll {
		sr := executeStep(ctx, step, vars, opts, completedSteps)
		result.Steps = append(result.Steps, sr)
	}

	result.ElapsedTime = time.Since(start)

	// Save state for resume.
	if !opts.DryRun {
		_ = SaveState(".gplay/workflow-state.json", result)
	}

	if mainErr != nil {
		return result, mainErr
	}
	return result, nil
}

func executeStep(ctx context.Context, step Step, vars map[string]string, opts ExecuteOptions, completedSteps map[string]bool) StepResult {
	sr := StepResult{
		Name:    step.Name,
		Command: step.Command,
	}

	// Check if already completed (resume).
	if completedSteps != nil && step.Name != "" && completedSteps[step.Name] {
		sr.Skipped = true
		return sr
	}

	// Evaluate condition.
	if step.Condition != "" {
		condValue, ok := vars[step.Condition]
		if !ok {
			condValue = os.Getenv(step.Condition)
		}
		if !isTruthy(condValue) {
			sr.Skipped = true
			return sr
		}
	}

	// Interpolate variables in command.
	command, err := Interpolate(step.Command, vars)
	if err != nil {
		sr.ExitCode = 1
		sr.Stderr = err.Error()
		return sr
	}
	sr.Command = command

	// Dry run: print command without executing.
	if opts.DryRun {
		fmt.Fprintf(opts.Stderr, "[dry-run] step %s: %s\n", step.Name, command)
		sr.Skipped = true
		return sr
	}

	// Execute the command.
	var stdoutBuf, stderrBuf bytes.Buffer

	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	cmd.Env = buildEnvSlice(vars)
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	if err := cmd.Run(); err != nil {
		sr.ExitCode = 1
		if exitErr, ok := err.(*exec.ExitError); ok {
			sr.ExitCode = exitErr.ExitCode()
		}
	}

	sr.Stdout = stdoutBuf.String()
	sr.Stderr = stderrBuf.String()

	return sr
}

// mergeVars merges variable maps in order. Later values override earlier.
func mergeVars(maps ...map[string]string) map[string]string {
	result := make(map[string]string)
	for _, m := range maps {
		for k, v := range m {
			result[k] = v
		}
	}
	return result
}

// isTruthy returns true if a value is explicitly truthy.
func isTruthy(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "y", "on":
		return true
	default:
		return false
	}
}

// buildEnvSlice creates a []string for exec.Cmd.Env by overlaying the
// vars map onto os.Environ().
func buildEnvSlice(env map[string]string) []string {
	base := os.Environ()
	if len(env) == 0 {
		return base
	}
	for k, v := range env {
		base = append(base, k+"="+v)
	}
	return base
}
