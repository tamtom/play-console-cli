package workflow

import (
	"fmt"
	"strings"
)

// Validate checks a workflow definition for structural errors.
// Returns all validation errors found (not just the first).
func Validate(w *Workflow) []error {
	var errs []error

	if strings.TrimSpace(w.Name) == "" {
		errs = append(errs, fmt.Errorf("workflow name must not be empty"))
	}

	if len(w.Steps) == 0 {
		errs = append(errs, fmt.Errorf("workflow must have at least one step"))
	}

	// Check for duplicate step names.
	seen := make(map[string]bool)
	for i, step := range w.Steps {
		name := strings.TrimSpace(step.Name)
		if name == "" {
			continue
		}
		if seen[name] {
			errs = append(errs, fmt.Errorf("step %d: duplicate step name %q", i+1, name))
		}
		seen[name] = true
	}

	// Check commands are non-empty.
	for i, step := range w.Steps {
		if strings.TrimSpace(step.Command) == "" {
			errs = append(errs, fmt.Errorf("step %d: command must not be empty", i+1))
		}
	}

	// Validate params: names must be non-empty and unique.
	paramSeen := make(map[string]bool)
	for i, p := range w.Params {
		pname := strings.TrimSpace(p.Name)
		if pname == "" {
			errs = append(errs, fmt.Errorf("param %d: name must not be empty", i+1))
			continue
		}
		if paramSeen[pname] {
			errs = append(errs, fmt.Errorf("param %d: duplicate param name %q", i+1, pname))
		}
		paramSeen[pname] = true
	}

	return errs
}
