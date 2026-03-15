package cmd

import (
	"bytes"
	"os"
	"testing"
)

func TestRun_VersionFlag(t *testing.T) {
	code := Run([]string{"--version"}, "1.0.0")
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}
}

func TestRun_VersionFlagOutput(t *testing.T) {
	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	_ = Run([]string{"--version"}, "1.0.0 (commit: abc123, date: 2024-01-01)")

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if output == "" {
		t.Fatal("expected version output, got empty string")
	}
	if !bytes.Contains([]byte(output), []byte("1.0.0")) {
		t.Errorf("expected output to contain '1.0.0', got %q", output)
	}
}

func TestRun_NoArgs(t *testing.T) {
	code := Run([]string{}, "1.0.0")
	if code != 1 {
		t.Errorf("expected exit code 1, got %d", code)
	}
}

func TestRun_UnknownCommand(t *testing.T) {
	code := Run([]string{"nonexistent"}, "1.0.0")
	if code != 1 {
		t.Errorf("expected exit code 1, got %d", code)
	}
}

func TestIsVersionOnlyInvocation(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want bool
	}{
		{name: "double-dash version", args: []string{"--version"}, want: true},
		{name: "single-dash version", args: []string{"-version"}, want: true},
		{name: "no args", args: []string{}, want: false},
		{name: "version subcommand", args: []string{"version"}, want: false},
		{name: "version with extra args", args: []string{"--version", "extra"}, want: false},
		{name: "other flag", args: []string{"--help"}, want: false},
		{name: "version flag not first", args: []string{"apps", "--version"}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isVersionOnlyInvocation(tt.args)
			if got != tt.want {
				t.Errorf("isVersionOnlyInvocation(%v) = %v, want %v", tt.args, got, tt.want)
			}
		})
	}
}
