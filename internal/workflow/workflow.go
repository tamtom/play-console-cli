// Package workflow provides a standalone workflow runner for .gplay/workflows/*.json files.
// It has zero imports from the rest of the codebase, depending only on Go stdlib.
package workflow

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

// Workflow is the top-level workflow definition.
type Workflow struct {
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Params      []Param           `json:"params,omitempty"`
	Env         map[string]string `json:"env,omitempty"`
	BeforeAll   []Step            `json:"before_all,omitempty"`
	Steps       []Step            `json:"steps"`
	AfterAll    []Step            `json:"after_all,omitempty"`
	OnError     []Step            `json:"on_error,omitempty"`
}

// Step is one executable action in a workflow.
type Step struct {
	Name       string `json:"name"`
	Command    string `json:"command"`
	ContinueOn string `json:"continue_on,omitempty"` // "error" to continue on failure
	Condition  string `json:"condition,omitempty"`    // skip if evaluates to false
}

// Param declares a workflow parameter.
type Param struct {
	Name     string `json:"name"`
	Required bool   `json:"required"`
	Default  string `json:"default,omitempty"`
}

// UnmarshalJSON handles the flexible step format:
//   - bare string -> Step{Command: "..."}
//   - object -> normal unmarshal
func (s *Step) UnmarshalJSON(data []byte) error {
	if bytes.Equal(data, []byte("null")) {
		return fmt.Errorf("step must be a string or object, not null")
	}

	var raw string
	if err := json.Unmarshal(data, &raw); err == nil {
		*s = Step{Command: raw}
		return nil
	}

	type stepAlias Step
	var alias stepAlias
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&alias); err != nil {
		return fmt.Errorf("step must be a string or object: %w", err)
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return fmt.Errorf("step must be a single JSON value: trailing data")
	}
	*s = Step(alias)
	return nil
}

// Load reads and parses a workflow definition from a JSON file.
func Load(path string) (*Workflow, error) {
	data, err := readFile(path)
	if err != nil {
		return nil, fmt.Errorf("load workflow: %w", err)
	}

	var w Workflow
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&w); err != nil {
		return nil, fmt.Errorf("parse workflow JSON: %w", err)
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return nil, fmt.Errorf("parse workflow JSON: trailing data")
	}

	return &w, nil
}
