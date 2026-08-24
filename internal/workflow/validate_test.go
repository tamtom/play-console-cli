package workflow

import (
	"strings"
	"testing"
)

func definitionForValidation(workflow Workflow) *Definition {
	if strings.TrimSpace(workflow.Name) == "" {
		workflow.Name = "deploy"
	}
	return &Definition{
		Workflows: map[string]Workflow{
			workflow.Name: workflow,
		},
	}
}

func TestValidate_ValidWorkflow(t *testing.T) {
	errs := Validate(definitionForValidation(Workflow{
		Name: "deploy",
		Steps: []Step{
			{Name: "build", Command: "make build"},
			{Name: "test", Command: "make test"},
		},
	}))
	if len(errs) != 0 {
		t.Fatalf("expected no errors, got %d: %v", len(errs), errs)
	}
}

func TestValidate_NoWorkflows(t *testing.T) {
	errs := Validate(&Definition{})
	if len(errs) != 1 || errs[0].Code != ErrNoWorkflows {
		t.Fatalf("expected ErrNoWorkflows, got %#v", errs)
	}
}

func TestValidate_InvalidWorkflowName(t *testing.T) {
	errs := Validate(&Definition{
		Workflows: map[string]Workflow{
			"123deploy": {Steps: []Step{{Name: "build", Command: "make build"}}},
		},
	})
	if len(errs) == 0 || errs[0].Code != ErrInvalidWorkflowName {
		t.Fatalf("expected ErrInvalidWorkflowName, got %#v", errs)
	}
}

func TestValidate_NoSteps(t *testing.T) {
	errs := Validate(definitionForValidation(Workflow{Name: "empty"}))
	if len(errs) == 0 {
		t.Fatal("expected validation error")
	}
	if errs[0].Code != ErrEmptySteps {
		t.Fatalf("expected ErrEmptySteps, got %#v", errs[0])
	}
}

func TestValidate_DuplicateStepNames(t *testing.T) {
	errs := Validate(definitionForValidation(Workflow{
		Name: "dup",
		Steps: []Step{
			{Name: "build", Command: "echo a"},
			{Name: "build", Command: "echo b"},
		},
	}))
	found := false
	for _, err := range errs {
		if err.Code == ErrDuplicateStepName {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected ErrDuplicateStepName, got %#v", errs)
	}
}

func TestValidate_EmptyCommand(t *testing.T) {
	errs := Validate(definitionForValidation(Workflow{
		Name:  "bad",
		Steps: []Step{{Name: "empty-cmd", Run: "   "}},
	}))
	found := false
	for _, err := range errs {
		if err.Code == ErrStepEmptyRun {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected ErrStepEmptyRun, got %#v", errs)
	}
}

func TestValidate_WithOnRunStep(t *testing.T) {
	errs := Validate(definitionForValidation(Workflow{
		Name: "bad",
		Steps: []Step{{
			Name: "build",
			Run:  "make build",
			With: map[string]string{"TRACK": "internal"},
		}},
	}))
	found := false
	for _, err := range errs {
		if err.Code == ErrStepWithOnRun {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected ErrStepWithOnRun, got %#v", errs)
	}
}

func TestValidate_UnknownWorkflowReference(t *testing.T) {
	errs := Validate(definitionForValidation(Workflow{
		Name:  "deploy",
		Steps: []Step{{Name: "publish", Workflow: "missing"}},
	}))
	found := false
	for _, err := range errs {
		if err.Code == ErrWorkflowNotFound {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected ErrWorkflowNotFound, got %#v", errs)
	}
}

func TestValidate_CyclicWorkflowReference(t *testing.T) {
	errs := Validate(&Definition{
		Workflows: map[string]Workflow{
			"deploy":  {Steps: []Step{{Workflow: "publish"}}},
			"publish": {Steps: []Step{{Workflow: "deploy"}}},
		},
	})
	found := false
	for _, err := range errs {
		if err.Code == ErrCyclicReference {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected ErrCyclicReference, got %#v", errs)
	}
}

func TestValidate_OutputRules(t *testing.T) {
	errs := Validate(&Definition{
		Workflows: map[string]Workflow{
			"prepare": {
				Steps: []Step{
					{Name: "capture", Workflow: "publish", Outputs: map[string]string{"version": "$.version"}},
				},
			},
			"publish": {
				Steps: []Step{{Name: "release", Run: "echo ok"}},
			},
		},
	})
	found := false
	for _, err := range errs {
		if err.Code == ErrStepOutputsOnWorkflow {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected ErrStepOutputsOnWorkflow, got %#v", errs)
	}
}

func TestValidate_DuplicateOutputProducerNames(t *testing.T) {
	errs := Validate(&Definition{
		Workflows: map[string]Workflow{
			"prepare": {
				Steps: []Step{{Name: "capture", Run: "echo '{}'", Outputs: map[string]string{"version": "$.version"}}},
			},
			"publish": {
				Steps: []Step{{Name: "capture", Run: "echo '{}'", Outputs: map[string]string{"code": "$.code"}}},
			},
		},
	})
	found := false
	for _, err := range errs {
		if err.Code == ErrDuplicateOutputProducerName {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected ErrDuplicateOutputProducerName, got %#v", errs)
	}
}

func TestValidate_ParamRules(t *testing.T) {
	errs := Validate(definitionForValidation(Workflow{
		Name: "deploy",
		Params: []Param{
			{Name: "TRACK"},
			{Name: "TRACK"},
			{Name: ""},
		},
		Steps: []Step{{Name: "build", Command: "make build"}},
	}))

	foundDuplicate := false
	foundEmpty := false
	for _, err := range errs {
		if err.Code == ErrDuplicateParamName {
			foundDuplicate = true
		}
		if err.Code == ErrEmptyParamName {
			foundEmpty = true
		}
	}
	if !foundDuplicate || !foundEmpty {
		t.Fatalf("expected duplicate and empty param errors, got %#v", errs)
	}
}

func TestValidate_RetryPolicyIsExplicitAndBounded(t *testing.T) {
	tests := []struct {
		name string
		step Step
		code ValidationCode
	}{
		{
			name: "too few attempts",
			step: Step{Name: "release", Run: "false", Retry: &RetryPolicy{MaxAttempts: 1, Delay: "1s"}},
			code: ErrInvalidRetryMaxAttempts,
		},
		{
			name: "zero delay",
			step: Step{Name: "release", Run: "false", Retry: &RetryPolicy{MaxAttempts: 2, Delay: "0s"}},
			code: ErrInvalidRetryDelay,
		},
		{
			name: "workflow call",
			step: Step{Name: "child", Workflow: "child", Retry: &RetryPolicy{MaxAttempts: 2, Delay: "1s"}},
			code: ErrStepRetryOnWorkflow,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			def := &Definition{Workflows: map[string]Workflow{
				"main":  {Steps: []Step{tt.step}},
				"child": {Steps: []Step{{Name: "ok", Run: "true"}}},
			}}
			var found bool
			for _, validationErr := range Validate(def) {
				if validationErr.Code == tt.code {
					found = true
				}
			}
			if !found {
				t.Fatalf("expected %q, got %#v", tt.code, Validate(def))
			}
		})
	}
}

func TestValidate_TimeoutPolicyIsRunOnlyAndBounded(t *testing.T) {
	zero := "0s"
	oneSecond := "1s"
	tests := []struct {
		name string
		step Step
		code ValidationCode
	}{
		{
			name: "zero timeout",
			step: Step{Name: "release", Run: "sleep 1", Timeout: &zero},
			code: ErrInvalidStepTimeout,
		},
		{
			name: "workflow call",
			step: Step{Name: "child", Workflow: "child", Timeout: &oneSecond},
			code: ErrStepTimeoutOnWorkflow,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			def := &Definition{Workflows: map[string]Workflow{
				"main":  {Steps: []Step{tt.step}},
				"child": {Steps: []Step{{Name: "ok", Run: "true"}}},
			}}
			for _, validationErr := range Validate(def) {
				if validationErr.Code == tt.code {
					return
				}
			}
			t.Fatalf("expected %q, got %#v", tt.code, Validate(def))
		})
	}
}

func TestValidate_RetryAndTimeoutPoliciesInLifecycleHooks(t *testing.T) {
	zero := "0s"
	def := definitionForValidation(Workflow{
		BeforeAll: []Step{{Name: "prepare", Run: "true", Retry: &RetryPolicy{MaxAttempts: 1, Delay: "1s"}}},
		Steps:     []Step{{Name: "release", Run: "true"}},
		AfterAll:  []Step{{Name: "cleanup", Run: "true", Timeout: &zero}},
	})
	foundRetry := false
	foundTimeout := false
	for _, validationErr := range Validate(def) {
		foundRetry = foundRetry || validationErr.Code == ErrInvalidRetryMaxAttempts
		foundTimeout = foundTimeout || validationErr.Code == ErrInvalidStepTimeout
	}
	if !foundRetry || !foundTimeout {
		t.Fatalf("hook policies were not fully validated: %#v", Validate(def))
	}
}
