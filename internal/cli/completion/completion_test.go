package completion

import (
	"bytes"
	"context"
	"flag"
	"os"
	"strings"
	"testing"
)

func TestCompletionCommand_Name(t *testing.T) {
	cmd := CompletionCommand()
	if cmd.Name != "completion" {
		t.Errorf("Name = %q, want %q", cmd.Name, "completion")
	}
}

func TestCompletionCommand_HasSubcommands(t *testing.T) {
	cmd := CompletionCommand()
	names := map[string]bool{}
	for _, sub := range cmd.Subcommands {
		names[sub.Name] = true
	}
	for _, shell := range []string{"bash", "zsh", "fish", "powershell"} {
		if !names[shell] {
			t.Errorf("expected %q subcommand", shell)
		}
	}
}

func TestCompletionCommand_NoArgs_PrintsSetup(t *testing.T) {
	cmd := CompletionCommand()

	// Capture stderr
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	err := cmd.Exec(context.Background(), nil)

	w.Close()
	os.Stderr = oldStderr

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if err != flag.ErrHelp {
		t.Errorf("expected flag.ErrHelp, got %v", err)
	}
	if !strings.Contains(output, "gplay completion bash") {
		t.Error("expected setup instructions to mention bash")
	}
	if !strings.Contains(output, "gplay completion zsh") {
		t.Error("expected setup instructions to mention zsh")
	}
	if !strings.Contains(output, "gplay completion fish") {
		t.Error("expected setup instructions to mention fish")
	}
}

func TestBashCommand_Output(t *testing.T) {
	cmd := BashCommand()

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := cmd.Exec(context.Background(), nil)

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if !strings.Contains(output, "_gplay_completions") {
		t.Error("expected bash completion function")
	}
}

func TestZshCommand_Output(t *testing.T) {
	cmd := ZshCommand()

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := cmd.Exec(context.Background(), nil)

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if !strings.Contains(output, "#compdef gplay") {
		t.Error("expected zsh completion header")
	}
}

func TestFishCommand_Output(t *testing.T) {
	cmd := FishCommand()

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := cmd.Exec(context.Background(), nil)

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if !strings.Contains(output, "complete -c gplay") {
		t.Error("expected fish completion commands")
	}
}

func TestPowerShellCommand_Output(t *testing.T) {
	cmd := PowerShellCommand()

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := cmd.Exec(context.Background(), nil)

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if !strings.Contains(output, "Register-ArgumentCompleter") {
		t.Error("expected PowerShell completion commands")
	}
}
