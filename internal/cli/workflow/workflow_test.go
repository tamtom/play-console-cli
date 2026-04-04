package workflow

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/peterbourgon/ff/v3/ffcli"
)

func findSubcmd(cmd *ffcli.Command, name string) *ffcli.Command {
	for _, sub := range cmd.Subcommands {
		if sub.Name == name {
			return sub
		}
	}
	return nil
}

func TestWorkflowCommand_HasSubcommands(t *testing.T) {
	cmd := WorkflowCommand()
	if cmd.Name != "workflow" {
		t.Errorf("Name = %q, want %q", cmd.Name, "workflow")
	}
	if len(cmd.Subcommands) != 3 {
		t.Errorf("expected 3 subcommands, got %d", len(cmd.Subcommands))
	}

	for _, want := range []string{"run", "validate", "list"} {
		if findSubcmd(cmd, want) == nil {
			t.Errorf("missing subcommand %q", want)
		}
	}
}

func TestWorkflowCommand_NoArgsReturnsHelp(t *testing.T) {
	cmd := WorkflowCommand()
	err := cmd.Exec(context.Background(), nil)
	if !errors.Is(err, flag.ErrHelp) {
		t.Errorf("expected flag.ErrHelp, got %v", err)
	}
}

func TestWorkflowRunCommand_NoArgsError(t *testing.T) {
	cmd := WorkflowCommand()
	runCmd := findSubcmd(cmd, "run")
	if runCmd == nil {
		t.Fatal("run subcommand not found")
	}

	err := runCmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for missing args")
	}
}

func TestWorkflowValidateCommand_NoArgsError(t *testing.T) {
	cmd := WorkflowCommand()
	validateCmd := findSubcmd(cmd, "validate")
	if validateCmd == nil {
		t.Fatal("validate subcommand not found")
	}

	err := validateCmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for missing args")
	}
}

func TestWorkflowListCommand_EmptyDir(t *testing.T) {
	dir := t.TempDir()

	cmd := WorkflowCommand()
	listCmd := findSubcmd(cmd, "list")
	if listCmd == nil {
		t.Fatal("list subcommand not found")
	}
	if err := listCmd.FlagSet.Parse([]string{"--dir", dir}); err != nil {
		t.Fatalf("parse flags: %v", err)
	}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := listCmd.Exec(context.Background(), nil)

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf [4096]byte
	n, _ := r.Read(buf[:])
	output := string(buf[:n])

	var workflows []any
	if err := json.Unmarshal([]byte(output), &workflows); err != nil {
		t.Fatalf("invalid JSON output: %v (output: %s)", err, output)
	}
	if len(workflows) != 0 {
		t.Errorf("expected empty list, got %d items", len(workflows))
	}
}

func TestWorkflowListCommand_NonexistentDir(t *testing.T) {
	cmd := WorkflowCommand()
	listCmd := findSubcmd(cmd, "list")
	if listCmd == nil {
		t.Fatal("list subcommand not found")
	}

	dir := filepath.Join(t.TempDir(), "nonexistent")
	if err := listCmd.FlagSet.Parse([]string{"--dir", dir}); err != nil {
		t.Fatalf("parse flags: %v", err)
	}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := listCmd.Exec(context.Background(), nil)

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf [4096]byte
	n, _ := r.Read(buf[:])
	output := string(buf[:n])

	var workflows []any
	if err := json.Unmarshal([]byte(output), &workflows); err != nil {
		t.Fatalf("invalid JSON output: %v (output: %s)", err, output)
	}
	if len(workflows) != 0 {
		t.Errorf("expected empty list, got %d items", len(workflows))
	}
}

func TestWorkflowListCommand_WithWorkflows(t *testing.T) {
	dir := t.TempDir()
	workflowJSON := `{
		"workflows": {
			"deploy": {
				"description": "Deploy the app",
				"steps": [
					{"name": "build", "command": "make build"},
					{"name": "test", "command": "make test"}
				]
			},
			"preflight": {
				"steps": [
					{"name": "validate", "command": "gplay validate --package com.example.app"}
				]
			}
		}
	}`
	if err := os.WriteFile(filepath.Join(dir, "deploy.json"), []byte(workflowJSON), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	cmd := WorkflowCommand()
	listCmd := findSubcmd(cmd, "list")
	if listCmd == nil {
		t.Fatal("list subcommand not found")
	}
	if err := listCmd.FlagSet.Parse([]string{"--dir", dir}); err != nil {
		t.Fatalf("parse flags: %v", err)
	}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := listCmd.Exec(context.Background(), nil)

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf [4096]byte
	n, _ := r.Read(buf[:])
	output := string(buf[:n])

	type info struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		StepCount   int    `json:"step_count"`
	}
	var workflows []info
	if err := json.Unmarshal([]byte(output), &workflows); err != nil {
		t.Fatalf("invalid JSON output: %v (output: %s)", err, output)
	}
	if len(workflows) != 2 {
		t.Fatalf("expected 2 workflows, got %d", len(workflows))
	}
	if workflows[0].Name != "deploy" {
		t.Errorf("Name = %q, want %q", workflows[0].Name, "deploy")
	}
	if workflows[0].StepCount != 2 {
		t.Errorf("StepCount = %d, want %d", workflows[0].StepCount, 2)
	}
	if workflows[1].Name != "preflight" {
		t.Errorf("Name = %q, want %q", workflows[1].Name, "preflight")
	}
}

func TestResolveWorkflowPath_JSONExtension(t *testing.T) {
	path, err := resolveWorkflowPath("./deploy.json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasSuffix(path, "deploy.json") {
		t.Errorf("path should end with deploy.json, got %q", path)
	}
}

func TestResolveWorkflowPath_NameOnly(t *testing.T) {
	path, err := resolveWorkflowPath("deploy")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(path, ".gplay/workflows/deploy.json") {
		t.Errorf("path should contain .gplay/workflows/deploy.json, got %q", path)
	}
}

func TestParseParams_Valid(t *testing.T) {
	params, err := parseParams([]string{"VERSION=1.0.0", "ENV=staging"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if params["VERSION"] != "1.0.0" {
		t.Errorf("VERSION = %q, want %q", params["VERSION"], "1.0.0")
	}
	if params["ENV"] != "staging" {
		t.Errorf("ENV = %q, want %q", params["ENV"], "staging")
	}
}

func TestParseParams_InvalidFormat(t *testing.T) {
	_, err := parseParams([]string{"NOEQUALS"})
	if err == nil {
		t.Fatal("expected error for invalid format")
	}
}

func TestParseParams_EmptyKey(t *testing.T) {
	_, err := parseParams([]string{"=value"})
	if err == nil {
		t.Fatal("expected error for empty key")
	}
}

func TestParseParams_Empty(t *testing.T) {
	params, err := parseParams(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(params) != 0 {
		t.Errorf("expected empty map, got %v", params)
	}
}

func TestWorkflowListCommand_UnexpectedArgs(t *testing.T) {
	cmd := WorkflowCommand()
	listCmd := findSubcmd(cmd, "list")
	if listCmd == nil {
		t.Fatal("list subcommand not found")
	}
	if err := listCmd.FlagSet.Parse([]string{}); err != nil {
		t.Fatalf("parse flags: %v", err)
	}

	err := listCmd.Exec(context.Background(), []string{"extra"})
	if err == nil {
		t.Fatal("expected error for unexpected args")
	}
}

func TestWorkflowValidateCommand_ValidFile(t *testing.T) {
	dir := t.TempDir()
	workflowJSON := `{
		"workflows": {
			"test": {
				"steps": [{"name": "s1", "command": "echo hi"}]
			}
		}
	}`
	filePath := filepath.Join(dir, "test.json")
	if err := os.WriteFile(filePath, []byte(workflowJSON), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	cmd := WorkflowCommand()
	validateCmd := findSubcmd(cmd, "validate")
	if validateCmd == nil {
		t.Fatal("validate subcommand not found")
	}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := validateCmd.Exec(context.Background(), []string{filePath})

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf [4096]byte
	n, _ := r.Read(buf[:])
	output := string(buf[:n])

	type result struct {
		Valid bool `json:"valid"`
	}
	var res result
	if err := json.Unmarshal([]byte(output), &res); err != nil {
		t.Fatalf("invalid JSON: %v (output: %s)", err, output)
	}
	if !res.Valid {
		t.Error("expected valid workflow")
	}
}

func TestWorkflowRunCommand_MultiWorkflowRequiresSelection(t *testing.T) {
	dir := t.TempDir()
	workflowJSON := `{
		"workflows": {
			"preflight": {"steps": [{"name": "validate", "command": "echo ok"}]},
			"publish": {"steps": [{"name": "release", "command": "echo ok"}]}
		}
	}`
	filePath := filepath.Join(dir, "release.json")
	if err := os.WriteFile(filePath, []byte(workflowJSON), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	cmd := WorkflowCommand()
	runCmd := findSubcmd(cmd, "run")
	if runCmd == nil {
		t.Fatal("run subcommand not found")
	}

	err := runCmd.Exec(context.Background(), []string{filePath})
	if err == nil {
		t.Fatal("expected selection error for multi-workflow file")
	}
	if !strings.Contains(err.Error(), "workflow name is required") {
		t.Fatalf("expected selection error, got %v", err)
	}
}

func TestWorkflowRunCommand_ExplicitWorkflowSelection(t *testing.T) {
	dir := t.TempDir()
	workflowJSON := `{
		"workflows": {
			"publish": {
				"params": [{"name": "TRACK", "required": true}],
				"steps": [{"name": "release", "run": "echo {{ .TRACK }}"}]
			}
		}
	}`
	filePath := filepath.Join(dir, "release.json")
	if err := os.WriteFile(filePath, []byte(workflowJSON), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	cmd := WorkflowCommand()
	runCmd := findSubcmd(cmd, "run")
	if runCmd == nil {
		t.Fatal("run subcommand not found")
	}
	if err := runCmd.FlagSet.Parse([]string{"--workflow", "publish", "--param", "TRACK=internal"}); err != nil {
		t.Fatalf("parse flags: %v", err)
	}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runCmd.Exec(context.Background(), []string{filePath})

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf [4096]byte
	n, _ := r.Read(buf[:])
	output := string(buf[:n])

	type result struct {
		Workflow string `json:"workflow"`
		Success  bool   `json:"success"`
	}
	var res result
	if err := json.Unmarshal([]byte(output), &res); err != nil {
		t.Fatalf("invalid JSON output: %v (output: %s)", err, output)
	}
	if res.Workflow != "publish" || !res.Success {
		t.Fatalf("unexpected result: %#v", res)
	}
}
