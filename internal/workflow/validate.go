package workflow

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
)

// ValidationCode classifies workflow definition errors.
type ValidationCode string

const (
	ErrNoWorkflows                 ValidationCode = "no_workflows"
	ErrInvalidWorkflowName         ValidationCode = "invalid_workflow_name"
	ErrEmptySteps                  ValidationCode = "empty_steps"
	ErrDuplicateStepName           ValidationCode = "duplicate_step_name"
	ErrStepNoAction                ValidationCode = "step_no_action"
	ErrStepEmptyRun                ValidationCode = "step_empty_run"
	ErrStepConflict                ValidationCode = "step_run_and_workflow"
	ErrWorkflowNotFound            ValidationCode = "workflow_not_found"
	ErrCyclicReference             ValidationCode = "cyclic_reference"
	ErrStepWithOnRun               ValidationCode = "step_with_on_run"
	ErrStepOutputsOnWorkflow       ValidationCode = "step_outputs_on_workflow"
	ErrStepOutputsRequireName      ValidationCode = "step_outputs_require_name"
	ErrDuplicateOutputProducerName ValidationCode = "duplicate_output_producer_name"
	ErrInvalidOutputName           ValidationCode = "invalid_output_name"
	ErrInvalidOutputExpr           ValidationCode = "invalid_output_expr"
	ErrDuplicateParamName          ValidationCode = "duplicate_param_name"
	ErrEmptyParamName              ValidationCode = "empty_param_name"
)

// ValidationError describes one structured workflow validation failure.
type ValidationError struct {
	Code     ValidationCode `json:"code"`
	Workflow string         `json:"workflow,omitempty"`
	Step     int            `json:"step,omitempty"`
	Message  string         `json:"message"`
}

func (e *ValidationError) Error() string {
	return e.Message
}

var (
	validWorkflowName = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_-]*$`)
	validOutputName   = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]*$`)
	validOutputExpr   = regexp.MustCompile(`^\$\.[a-zA-Z0-9_]+(?:\.[a-zA-Z0-9_]+)*$`)
)

// Validate checks a workflow definition for structural errors.
// It returns all detected errors for stable CLI reporting.
func Validate(def *Definition) []*ValidationError {
	var errs []*ValidationError

	if def == nil || len(def.Workflows) == 0 {
		return []*ValidationError{{
			Code:    ErrNoWorkflows,
			Message: "workflow file must define at least one workflow",
		}}
	}

	names := mapsKeys(def.Workflows)
	slices.Sort(names)
	outputProducers := map[string]string{}

	for _, name := range names {
		if !validWorkflowName.MatchString(strings.TrimSpace(name)) {
			errs = append(errs, &ValidationError{
				Code:     ErrInvalidWorkflowName,
				Workflow: name,
				Message:  fmt.Sprintf("workflow name %q must start with a letter and contain only letters, digits, hyphens, or underscores", name),
			})
		}

		workflow := def.Workflows[name]
		if len(workflow.Steps) == 0 {
			errs = append(errs, &ValidationError{
				Code:     ErrEmptySteps,
				Workflow: name,
				Message:  fmt.Sprintf("workflow %q must have at least one step", name),
			})
			continue
		}

		stepSeen := map[string]bool{}
		for idx, step := range workflow.Steps {
			stepNumber := idx + 1
			stepName := strings.TrimSpace(step.Name)
			if stepName != "" {
				if stepSeen[stepName] {
					errs = append(errs, &ValidationError{
						Code:     ErrDuplicateStepName,
						Workflow: name,
						Step:     stepNumber,
						Message:  fmt.Sprintf("workflow %q step %d reuses step name %q", name, stepNumber, stepName),
					})
				}
				stepSeen[stepName] = true
			}

			action := strings.TrimSpace(step.Action())
			hasAction := action != ""
			hasWorkflow := strings.TrimSpace(step.Workflow) != ""
			hasDeclaredAction := step.Run != "" || step.Command != ""

			switch {
			case hasAction && hasWorkflow:
				errs = append(errs, &ValidationError{
					Code:     ErrStepConflict,
					Workflow: name,
					Step:     stepNumber,
					Message:  fmt.Sprintf("workflow %q step %d has both run and workflow; only one is allowed", name, stepNumber),
				})
			case !hasAction && !hasWorkflow && hasDeclaredAction:
				errs = append(errs, &ValidationError{
					Code:     ErrStepEmptyRun,
					Workflow: name,
					Step:     stepNumber,
					Message:  fmt.Sprintf("workflow %q step %d has empty run command", name, stepNumber),
				})
			case !hasAction && !hasWorkflow:
				errs = append(errs, &ValidationError{
					Code:     ErrStepNoAction,
					Workflow: name,
					Step:     stepNumber,
					Message:  fmt.Sprintf("workflow %q step %d must define run or workflow", name, stepNumber),
				})
			}

			if hasAction && len(step.With) > 0 {
				errs = append(errs, &ValidationError{
					Code:     ErrStepWithOnRun,
					Workflow: name,
					Step:     stepNumber,
					Message:  fmt.Sprintf("workflow %q step %d uses 'with' on a run step; 'with' is only valid for workflow references", name, stepNumber),
				})
			}

			if len(step.Outputs) > 0 {
				if hasWorkflow {
					errs = append(errs, &ValidationError{
						Code:     ErrStepOutputsOnWorkflow,
						Workflow: name,
						Step:     stepNumber,
						Message:  fmt.Sprintf("workflow %q step %d declares outputs on a workflow step; outputs are only valid for run steps", name, stepNumber),
					})
				}

				if stepName == "" || !validWorkflowName.MatchString(stepName) {
					errs = append(errs, &ValidationError{
						Code:     ErrStepOutputsRequireName,
						Workflow: name,
						Step:     stepNumber,
						Message:  fmt.Sprintf("workflow %q step %d must use a reference-safe step name when declaring outputs", name, stepNumber),
					})
				} else if previous, exists := outputProducers[stepName]; exists {
					errs = append(errs, &ValidationError{
						Code:     ErrDuplicateOutputProducerName,
						Workflow: name,
						Step:     stepNumber,
						Message:  fmt.Sprintf("workflow %q step %d reuses output-producing step name %q already declared in workflow %q", name, stepNumber, stepName, previous),
					})
				} else {
					outputProducers[stepName] = name
				}

				outputNames := mapsKeys(step.Outputs)
				slices.Sort(outputNames)
				for _, outputName := range outputNames {
					if !validOutputName.MatchString(outputName) {
						errs = append(errs, &ValidationError{
							Code:     ErrInvalidOutputName,
							Workflow: name,
							Step:     stepNumber,
							Message:  fmt.Sprintf("workflow %q step %d has invalid output name %q", name, stepNumber, outputName),
						})
					}
					if !validOutputExpr.MatchString(strings.TrimSpace(step.Outputs[outputName])) {
						errs = append(errs, &ValidationError{
							Code:     ErrInvalidOutputExpr,
							Workflow: name,
							Step:     stepNumber,
							Message:  fmt.Sprintf("workflow %q step %d output %q must use a JSON path like $.field", name, stepNumber, outputName),
						})
					}
				}
			}

			if hasWorkflow {
				ref := strings.TrimSpace(step.Workflow)
				if _, ok := def.Workflows[ref]; !ok {
					errs = append(errs, &ValidationError{
						Code:     ErrWorkflowNotFound,
						Workflow: name,
						Step:     stepNumber,
						Message:  fmt.Sprintf("workflow %q step %d references unknown workflow %q", name, stepNumber, ref),
					})
				}
			}
		}

		paramSeen := map[string]bool{}
		for idx, param := range workflow.Params {
			paramNumber := idx + 1
			paramName := strings.TrimSpace(param.Name)
			if paramName == "" {
				errs = append(errs, &ValidationError{
					Code:     ErrEmptyParamName,
					Workflow: name,
					Message:  fmt.Sprintf("workflow %q param %d must not have an empty name", name, paramNumber),
				})
				continue
			}
			if paramSeen[paramName] {
				errs = append(errs, &ValidationError{
					Code:     ErrDuplicateParamName,
					Workflow: name,
					Message:  fmt.Sprintf("workflow %q param %d reuses param name %q", name, paramNumber, paramName),
				})
			}
			paramSeen[paramName] = true
		}
	}

	if cycleErr := detectCycles(def); cycleErr != nil {
		errs = append(errs, cycleErr)
	}

	return errs
}

func detectCycles(def *Definition) *ValidationError {
	const (
		white = 0
		gray  = 1
		black = 2
	)

	colors := make(map[string]int, len(def.Workflows))
	var path []string

	var dfs func(name string) *ValidationError
	dfs = func(name string) *ValidationError {
		colors[name] = gray
		path = append(path, name)

		current, ok := def.Workflows[name]
		if !ok {
			path = path[:len(path)-1]
			colors[name] = black
			return nil
		}

		for _, step := range current.Steps {
			ref := strings.TrimSpace(step.Workflow)
			if ref == "" {
				continue
			}
			switch colors[ref] {
			case gray:
				cycleStart := 0
				for i, entry := range path {
					if entry == ref {
						cycleStart = i
						break
					}
				}
				cycle := append(slices.Clone(path[cycleStart:]), ref)
				return &ValidationError{
					Code:     ErrCyclicReference,
					Workflow: name,
					Message:  fmt.Sprintf("cyclic workflow reference: %s", strings.Join(cycle, " -> ")),
				}
			case white:
				if err := dfs(ref); err != nil {
					return err
				}
			}
		}

		path = path[:len(path)-1]
		colors[name] = black
		return nil
	}

	names := mapsKeys(def.Workflows)
	slices.Sort(names)
	for _, name := range names {
		if colors[name] == white {
			if err := dfs(name); err != nil {
				return err
			}
		}
	}

	return nil
}
