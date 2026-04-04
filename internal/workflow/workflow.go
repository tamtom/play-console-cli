// Package workflow provides a standalone workflow runner for .gplay/workflows/*.json files.
// It has zero imports from the rest of the codebase, depending only on Go stdlib.
package workflow

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"slices"
	"strings"
)

// Definition is the normalized workflow definition format.
// New files should use the `workflows` map. Legacy single-workflow files are
// still supported and are wrapped into a one-entry Definition at load time.
type Definition struct {
	Workflows map[string]Workflow `json:"workflows"`
}

// Workflow is one named automation sequence.
type Workflow struct {
	Name        string            `json:"name,omitempty"`
	Description string            `json:"description,omitempty"`
	Private     bool              `json:"private,omitempty"`
	Params      []Param           `json:"params,omitempty"`
	Env         map[string]string `json:"env,omitempty"`
	BeforeAll   []Step            `json:"before_all,omitempty"`
	Steps       []Step            `json:"steps"`
	AfterAll    []Step            `json:"after_all,omitempty"`
	OnError     []Step            `json:"on_error,omitempty"`
}

// Step is one executable action in a workflow.
// Bare JSON strings unmarshal to Step{Run: "..."}.
//
// `command` and `condition` remain supported as legacy aliases for `run` and
// `if`, so existing workflow files continue to work.
type Step struct {
	Name       string            `json:"name,omitempty"`
	Run        string            `json:"run,omitempty"`
	Command    string            `json:"command,omitempty"`
	Workflow   string            `json:"workflow,omitempty"`
	ContinueOn string            `json:"continue_on,omitempty"` // "error" to continue on failure
	If         string            `json:"if,omitempty"`          // skip if evaluates to false
	Condition  string            `json:"condition,omitempty"`
	With       map[string]string `json:"with,omitempty"`
	Outputs    map[string]string `json:"outputs,omitempty"`
}

// Param declares a workflow parameter.
type Param struct {
	Name     string `json:"name"`
	Required bool   `json:"required"`
	Default  string `json:"default,omitempty"`
}

// Action returns the normalized shell action for a step.
func (s Step) Action() string {
	if strings.TrimSpace(s.Run) != "" {
		return s.Run
	}
	return s.Command
}

// Predicate returns the normalized conditional variable for a step.
func (s Step) Predicate() string {
	if strings.TrimSpace(s.If) != "" {
		return s.If
	}
	return s.Condition
}

// UnmarshalJSON handles the flexible step format:
//   - bare string -> Step{Run: "..."}
//   - object -> normal unmarshal
func (s *Step) UnmarshalJSON(data []byte) error {
	if bytes.Equal(data, []byte("null")) {
		return fmt.Errorf("step must be a string or object, not null")
	}

	var raw string
	if err := json.Unmarshal(data, &raw); err == nil {
		*s = Step{Run: raw}
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
	if strings.TrimSpace(alias.Run) == "" && alias.Command != "" {
		alias.Run = alias.Command
	}
	if strings.TrimSpace(alias.If) == "" && alias.Condition != "" {
		alias.If = alias.Condition
	}
	*s = Step(alias)
	return nil
}

// LoadDefinition reads and parses a workflow definition from a JSON file.
// It supports both the new `workflows` schema and the legacy single-workflow
// schema used by earlier gplay versions.
func LoadDefinition(path string) (*Definition, error) {
	data, err := readFile(path)
	if err != nil {
		return nil, fmt.Errorf("load workflow: %w", err)
	}

	type definitionEnvelope struct {
		Workflows map[string]Workflow `json:"workflows"`
	}

	var env definitionEnvelope
	if err := decodeStrict(data, &env); err == nil && len(env.Workflows) > 0 {
		def := &Definition{Workflows: make(map[string]Workflow, len(env.Workflows))}
		for name, workflow := range env.Workflows {
			workflow.Name = strings.TrimSpace(name)
			def.Workflows[name] = workflow
		}
		return def, nil
	}

	var workflow Workflow
	if err := decodeStrict(data, &workflow); err != nil {
		return nil, fmt.Errorf("parse workflow JSON: %w", err)
	}

	name := strings.TrimSpace(workflow.Name)
	if name == "" {
		name = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	}
	workflow.Name = name

	return &Definition{
		Workflows: map[string]Workflow{name: workflow},
	}, nil
}

// Load reads a workflow file and returns a single workflow when the selection
// is unambiguous. It is kept for backward compatibility with the original
// single-workflow API.
func Load(path string) (*Workflow, error) {
	def, err := LoadDefinition(path)
	if err != nil {
		return nil, err
	}
	name, workflow, err := SelectWorkflow(def, strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)), "")
	if err != nil {
		return nil, err
	}
	workflow.Name = name
	return &workflow, nil
}

// SelectWorkflow chooses the workflow to run from a definition.
// If explicitName is set, it wins. Otherwise an implicitName match wins.
// If there is only one workflow, it is selected automatically.
func SelectWorkflow(def *Definition, implicitName, explicitName string) (string, Workflow, error) {
	if def == nil || len(def.Workflows) == 0 {
		return "", Workflow{}, fmt.Errorf("workflow file must define at least one workflow")
	}

	if explicitName != "" {
		workflow, ok := def.Workflows[explicitName]
		if !ok {
			return "", Workflow{}, fmt.Errorf("workflow %q not found", explicitName)
		}
		workflow.Name = explicitName
		return explicitName, workflow, nil
	}

	if implicitName != "" {
		if workflow, ok := def.Workflows[implicitName]; ok {
			workflow.Name = implicitName
			return implicitName, workflow, nil
		}
	}

	if len(def.Workflows) == 1 {
		names := mapsKeys(def.Workflows)
		slices.Sort(names)
		name := names[0]
		workflow := def.Workflows[name]
		workflow.Name = name
		return name, workflow, nil
	}

	names := mapsKeys(def.Workflows)
	slices.Sort(names)
	return "", Workflow{}, fmt.Errorf(
		"workflow name is required; available workflows: %s",
		strings.Join(names, ", "),
	)
}

func decodeStrict(data []byte, dst any) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return err
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return fmt.Errorf("trailing data")
	}
	return nil
}

func mapsKeys[K comparable, V any](m map[K]V) []K {
	keys := make([]K, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	return keys
}
