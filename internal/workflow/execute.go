package workflow

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"slices"
	"strconv"
	"strings"
	"time"
)

// ExecuteOptions configures workflow execution.
type ExecuteOptions struct {
	DryRun    bool
	Resume    bool
	Stdout    io.Writer
	Stderr    io.Writer
	StatePath string
}

// ExecutionResult is the structured output of a workflow execution.
type ExecutionResult struct {
	Workflow    string            `json:"workflow"`
	Steps       []StepResult      `json:"steps"`
	Success     bool              `json:"success"`
	ElapsedTime time.Duration     `json:"elapsed_time"`
	Outputs     map[string]string `json:"outputs,omitempty"`
}

// StepResult records one executed step.
type StepResult struct {
	Path     string            `json:"path,omitempty"`
	Workflow string            `json:"workflow,omitempty"`
	Name     string            `json:"name"`
	Command  string            `json:"command"`
	Stdout   string            `json:"stdout"`
	Stderr   string            `json:"stderr"`
	ExitCode int               `json:"exit_code"`
	Skipped  bool              `json:"skipped"`
	Outputs  map[string]string `json:"outputs,omitempty"`
}

// Execute runs a single workflow using the legacy API.
func Execute(ctx context.Context, workflow *Workflow, params map[string]string, opts ExecuteOptions) (*ExecutionResult, error) {
	if workflow == nil {
		return nil, fmt.Errorf("workflow is required")
	}

	name := strings.TrimSpace(workflow.Name)
	if name == "" {
		name = "workflow"
	}

	return ExecuteDefinition(ctx, &Definition{
		Workflows: map[string]Workflow{name: *workflow},
	}, name, params, opts)
}

// ExecuteDefinition runs one named workflow from a definition, including any
// referenced child workflows.
func ExecuteDefinition(ctx context.Context, def *Definition, workflowName string, params map[string]string, opts ExecuteOptions) (*ExecutionResult, error) {
	if opts.Stdout == nil {
		opts.Stdout = os.Stdout
	}
	if opts.Stderr == nil {
		opts.Stderr = os.Stderr
	}
	if opts.StatePath == "" {
		opts.StatePath = ".gplay/workflow-state.json"
	}

	root, ok := def.Workflows[workflowName]
	if !ok {
		return nil, fmt.Errorf("workflow %q not found", workflowName)
	}
	root.Name = workflowName

	start := time.Now()
	result := &ExecutionResult{
		Workflow: workflowName,
		Steps:    make([]StepResult, 0),
		Success:  true,
	}

	completed := map[string]bool{}
	if opts.Resume {
		state, err := LoadState(opts.StatePath)
		if err == nil && state != nil && state.Workflow == workflowName {
			for _, step := range state.Steps {
				if step.ExitCode == 0 && !step.Skipped {
					key := step.Path
					if key == "" {
						key = step.Name
					}
					completed[key] = true
				}
			}
		}
	}

	runner := executor{
		ctx:       ctx,
		def:       def,
		opts:      opts,
		result:    result,
		completed: completed,
	}

	_, mainErr := runner.executeWorkflow(root, params, workflowName)
	result.ElapsedTime = time.Since(start)
	result.Outputs = collectOutputs(result.Steps)

	if !opts.DryRun {
		_ = SaveState(opts.StatePath, result)
	}

	if mainErr != nil {
		result.Success = false
		return result, mainErr
	}

	return result, nil
}

type executor struct {
	ctx       context.Context
	def       *Definition
	opts      ExecuteOptions
	result    *ExecutionResult
	completed map[string]bool
}

func (e *executor) executeWorkflow(workflow Workflow, incomingVars map[string]string, pathPrefix string) (map[string]string, error) {
	vars, err := prepareWorkflowVars(workflow, incomingVars)
	if err != nil {
		return nil, err
	}

	if err := e.executeStepList(workflow.Name, workflow.BeforeAll, vars, pathPrefix+".before_all"); err != nil {
		_ = e.executeStepList(workflow.Name, workflow.OnError, vars, pathPrefix+".on_error")
		_ = e.executeStepList(workflow.Name, workflow.AfterAll, vars, pathPrefix+".after_all")
		return vars, err
	}

	mainErr := e.executeStepList(workflow.Name, workflow.Steps, vars, pathPrefix)
	if mainErr != nil {
		_ = e.executeStepList(workflow.Name, workflow.OnError, vars, pathPrefix+".on_error")
	}

	afterErr := e.executeStepList(workflow.Name, workflow.AfterAll, vars, pathPrefix+".after_all")
	if mainErr != nil {
		return vars, mainErr
	}
	return vars, afterErr
}

func (e *executor) executeStepList(currentWorkflow string, steps []Step, vars map[string]string, pathPrefix string) error {
	var firstErr error
	for index, step := range steps {
		stepID := buildStepID(step, index)
		stepPath := pathPrefix + "." + stepID
		if err := e.executeStep(currentWorkflow, step, vars, stepPath); err != nil {
			if firstErr == nil {
				firstErr = err
			}
			if step.ContinueOn == "error" {
				continue
			}
			return err
		}
	}
	return firstErr
}

func (e *executor) executeStep(currentWorkflow string, step Step, vars map[string]string, path string) error {
	result := StepResult{
		Path:     path,
		Workflow: currentWorkflow,
		Name:     strings.TrimSpace(step.Name),
	}
	if result.Name == "" {
		result.Name = pathSegment(path)
	}

	predicate := strings.TrimSpace(step.Predicate())
	if predicate != "" {
		condValue, ok := vars[predicate]
		if !ok {
			condValue = os.Getenv(predicate)
		}
		if !isTruthy(condValue) {
			result.Skipped = true
			e.result.Steps = append(e.result.Steps, result)
			return nil
		}
	}

	if ref := strings.TrimSpace(step.Workflow); ref != "" {
		result.Command = "workflow:" + ref
		e.result.Steps = append(e.result.Steps, result)

		workflow, ok := e.def.Workflows[ref]
		if !ok {
			e.result.Steps[len(e.result.Steps)-1].ExitCode = 1
			e.result.Steps[len(e.result.Steps)-1].Stderr = fmt.Sprintf("workflow %q not found", ref)
			return fmt.Errorf("step %q failed: workflow %q not found", result.Name, ref)
		}
		workflow.Name = ref

		childVars, err := buildWorkflowStepVars(vars, step.With)
		if err != nil {
			e.result.Steps[len(e.result.Steps)-1].ExitCode = 1
			e.result.Steps[len(e.result.Steps)-1].Stderr = err.Error()
			return fmt.Errorf("step %q failed: %w", result.Name, err)
		}

		updatedVars, err := e.executeWorkflow(workflow, childVars, path)
		if err != nil {
			e.result.Steps[len(e.result.Steps)-1].ExitCode = 1
			e.result.Steps[len(e.result.Steps)-1].Stderr = err.Error()
			return fmt.Errorf("step %q failed: %w", result.Name, err)
		}
		for key, value := range updatedVars {
			if strings.Contains(key, ".") {
				vars[key] = value
			}
		}

		return nil
	}

	if e.completed[path] {
		result.Skipped = true
		e.result.Steps = append(e.result.Steps, result)
		return nil
	}

	command, err := Interpolate(step.Action(), vars)
	if err != nil {
		result.Command = step.Action()
		result.ExitCode = 1
		result.Stderr = err.Error()
		e.result.Steps = append(e.result.Steps, result)
		return fmt.Errorf("step %q failed: %w", result.Name, err)
	}
	result.Command = command

	if e.opts.DryRun {
		fmt.Fprintf(e.opts.Stderr, "[dry-run] step %s: %s\n", path, command)
		result.Skipped = true
		e.result.Steps = append(e.result.Steps, result)
		return nil
	}

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd := exec.CommandContext(e.ctx, "sh", "-c", command)
	cmd.Env = buildEnvSlice(vars)
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	runErr := cmd.Run()
	if runErr != nil {
		result.ExitCode = 1
		var exitErr *exec.ExitError
		if errors.As(runErr, &exitErr) {
			result.ExitCode = exitErr.ExitCode()
		}
	}
	result.Stdout = stdoutBuf.String()
	result.Stderr = stderrBuf.String()

	if runErr == nil && len(step.Outputs) > 0 {
		outputs, extractErr := extractStepOutputs(result.Stdout, step.Outputs)
		if extractErr != nil {
			result.ExitCode = 1
			result.Stderr = strings.TrimSpace(strings.TrimSpace(result.Stderr) + "\n" + extractErr.Error())
			runErr = extractErr
		} else {
			result.Outputs = outputs
			outputPrefix := sanitizePathSegment(result.Name)
			for key, value := range outputs {
				vars[outputPrefix+"."+key] = value
			}
		}
	}

	e.result.Steps = append(e.result.Steps, result)
	if runErr != nil {
		return fmt.Errorf("step %q failed with exit code %d", result.Name, result.ExitCode)
	}

	return nil
}

func prepareWorkflowVars(workflow Workflow, incoming map[string]string) (map[string]string, error) {
	vars := mergeVars(workflow.Env, incoming)

	for _, param := range workflow.Params {
		if _, ok := vars[param.Name]; !ok && param.Default != "" {
			vars[param.Name] = param.Default
		}
	}

	for _, param := range workflow.Params {
		if param.Required {
			if value, ok := vars[param.Name]; !ok || strings.TrimSpace(value) == "" {
				return nil, fmt.Errorf("required parameter %q is missing", param.Name)
			}
		}
	}

	return vars, nil
}

func buildWorkflowStepVars(parentVars map[string]string, with map[string]string) (map[string]string, error) {
	vars := mergeVars(parentVars)
	for key, value := range with {
		interpolated, err := Interpolate(value, vars)
		if err != nil {
			return nil, fmt.Errorf("interpolate workflow param %q: %w", key, err)
		}
		vars[key] = interpolated
	}
	return vars, nil
}

func buildStepID(step Step, index int) string {
	if name := strings.TrimSpace(step.Name); name != "" {
		return sanitizePathSegment(name)
	}
	if ref := strings.TrimSpace(step.Workflow); ref != "" {
		return sanitizePathSegment(ref)
	}
	return "step_" + strconv.Itoa(index+1)
}

func sanitizePathSegment(value string) string {
	replacer := strings.NewReplacer(" ", "_", "/", "_", "\\", "_", ":", "_")
	value = replacer.Replace(strings.TrimSpace(value))
	if value == "" {
		return "step"
	}
	return value
}

func pathSegment(path string) string {
	if idx := strings.LastIndex(path, "."); idx >= 0 {
		return path[idx+1:]
	}
	return path
}

func mergeVars(maps ...map[string]string) map[string]string {
	result := make(map[string]string)
	for _, current := range maps {
		for key, value := range current {
			result[key] = value
		}
	}
	return result
}

func isTruthy(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "y", "on":
		return true
	default:
		return false
	}
}

func buildEnvSlice(env map[string]string) []string {
	base := os.Environ()
	if len(env) == 0 {
		return base
	}
	for key, value := range env {
		base = append(base, key+"="+value)
	}
	return base
}

func extractStepOutputs(stdout string, declared map[string]string) (map[string]string, error) {
	var decoded any
	dec := json.NewDecoder(strings.NewReader(stdout))
	dec.UseNumber()
	if err := dec.Decode(&decoded); err != nil {
		return nil, fmt.Errorf("parse step stdout for outputs: %w", err)
	}

	outputs := make(map[string]string, len(declared))
	names := mapsKeys(declared)
	slices.Sort(names)
	for _, name := range names {
		value, err := readJSONPath(decoded, declared[name])
		if err != nil {
			return nil, fmt.Errorf("extract output %q: %w", name, err)
		}
		outputs[name] = value
	}
	return outputs, nil
}

func readJSONPath(value any, expr string) (string, error) {
	if !strings.HasPrefix(expr, "$.") {
		return "", fmt.Errorf("unsupported output path %q", expr)
	}

	current := value
	for _, part := range strings.Split(strings.TrimPrefix(expr, "$."), ".") {
		object, ok := current.(map[string]any)
		if !ok {
			return "", fmt.Errorf("path %q does not resolve to an object at %q", expr, part)
		}
		next, ok := object[part]
		if !ok {
			return "", fmt.Errorf("path %q is missing field %q", expr, part)
		}
		current = next
	}

	switch typed := current.(type) {
	case nil:
		return "", nil
	case string:
		return typed, nil
	case json.Number:
		return typed.String(), nil
	case bool:
		if typed {
			return "true", nil
		}
		return "false", nil
	case float64:
		return strconv.FormatFloat(typed, 'f', -1, 64), nil
	default:
		encoded, err := json.Marshal(typed)
		if err != nil {
			return "", fmt.Errorf("marshal output value: %w", err)
		}
		return string(encoded), nil
	}
}

func collectOutputs(steps []StepResult) map[string]string {
	outputs := map[string]string{}
	for _, step := range steps {
		if len(step.Outputs) == 0 {
			continue
		}
		stepName := sanitizePathSegment(strings.TrimSpace(step.Name))
		for key, value := range step.Outputs {
			outputs[stepName+"."+key] = value
		}
	}
	if len(outputs) == 0 {
		return nil
	}
	return outputs
}
