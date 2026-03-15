package workflow

import (
	"strings"
	"testing"
)

func TestValidate_ValidWorkflow(t *testing.T) {
	w := &Workflow{
		Name: "deploy",
		Steps: []Step{
			{Name: "build", Command: "make build"},
			{Name: "test", Command: "make test"},
		},
	}

	errs := Validate(w)
	if len(errs) != 0 {
		t.Errorf("expected no errors, got %d: %v", len(errs), errs)
	}
}

func TestValidate_EmptyName(t *testing.T) {
	w := &Workflow{
		Name:  "",
		Steps: []Step{{Name: "s1", Command: "echo hi"}},
	}

	errs := Validate(w)
	if len(errs) == 0 {
		t.Fatal("expected error for empty name")
	}

	found := false
	for _, err := range errs {
		if strings.Contains(err.Error(), "name must not be empty") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'name must not be empty' error, got %v", errs)
	}
}

func TestValidate_NoSteps(t *testing.T) {
	w := &Workflow{
		Name:  "empty",
		Steps: nil,
	}

	errs := Validate(w)
	if len(errs) == 0 {
		t.Fatal("expected error for no steps")
	}

	found := false
	for _, err := range errs {
		if strings.Contains(err.Error(), "at least one step") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'at least one step' error, got %v", errs)
	}
}

func TestValidate_DuplicateStepNames(t *testing.T) {
	w := &Workflow{
		Name: "dup",
		Steps: []Step{
			{Name: "build", Command: "echo a"},
			{Name: "build", Command: "echo b"},
		},
	}

	errs := Validate(w)
	if len(errs) == 0 {
		t.Fatal("expected error for duplicate step names")
	}

	found := false
	for _, err := range errs {
		if strings.Contains(err.Error(), "duplicate step name") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'duplicate step name' error, got %v", errs)
	}
}

func TestValidate_EmptyCommand(t *testing.T) {
	w := &Workflow{
		Name: "bad",
		Steps: []Step{
			{Name: "empty-cmd", Command: ""},
		},
	}

	errs := Validate(w)
	if len(errs) == 0 {
		t.Fatal("expected error for empty command")
	}

	found := false
	for _, err := range errs {
		if strings.Contains(err.Error(), "command must not be empty") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'command must not be empty' error, got %v", errs)
	}
}

func TestValidate_WhitespaceOnlyCommand(t *testing.T) {
	w := &Workflow{
		Name: "ws",
		Steps: []Step{
			{Name: "ws-cmd", Command: "   "},
		},
	}

	errs := Validate(w)
	if len(errs) == 0 {
		t.Fatal("expected error for whitespace-only command")
	}
}

func TestValidate_MultipleErrors(t *testing.T) {
	w := &Workflow{
		Name: "",
		Steps: []Step{
			{Name: "a", Command: ""},
			{Name: "a", Command: "echo hi"},
		},
	}

	errs := Validate(w)
	// Should have: empty name, empty command, duplicate name
	if len(errs) < 3 {
		t.Errorf("expected at least 3 errors, got %d: %v", len(errs), errs)
	}
}

func TestValidate_DuplicateParamNames(t *testing.T) {
	w := &Workflow{
		Name: "params",
		Steps: []Step{
			{Name: "s1", Command: "echo hi"},
		},
		Params: []Param{
			{Name: "VERSION", Required: true},
			{Name: "VERSION", Required: false},
		},
	}

	errs := Validate(w)
	found := false
	for _, err := range errs {
		if strings.Contains(err.Error(), "duplicate param name") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'duplicate param name' error, got %v", errs)
	}
}

func TestValidate_EmptyParamName(t *testing.T) {
	w := &Workflow{
		Name: "params",
		Steps: []Step{
			{Name: "s1", Command: "echo hi"},
		},
		Params: []Param{
			{Name: "", Required: true},
		},
	}

	errs := Validate(w)
	found := false
	for _, err := range errs {
		if strings.Contains(err.Error(), "name must not be empty") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected param 'name must not be empty' error, got %v", errs)
	}
}

func TestValidate_StepsWithoutNames(t *testing.T) {
	// Steps without names should not trigger duplicate name errors.
	w := &Workflow{
		Name: "ok",
		Steps: []Step{
			{Name: "", Command: "echo a"},
			{Name: "", Command: "echo b"},
		},
	}

	errs := Validate(w)
	if len(errs) != 0 {
		t.Errorf("expected no errors, got %d: %v", len(errs), errs)
	}
}
