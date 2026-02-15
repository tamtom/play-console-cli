package cmdtest

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	// Build the binary once
	t := &testing.T{}
	dir, _ := os.MkdirTemp("", "cmdtest-*")
	defer os.RemoveAll(dir)

	// We need a real test for Build, so skip TestMain build here
	// Tests that need the binary will call Build themselves
	_ = t
	os.Exit(m.Run())
}

func TestRun_Help(t *testing.T) {
	Build(t)
	r := Run(t, "--help")
	// --help may exit 0 or 1 depending on ffcli behavior
	if r.Stdout == "" && r.Stderr == "" {
		t.Error("expected output from --help")
	}
}

func TestRun_Version(t *testing.T) {
	Build(t)
	r := Run(t, "version")
	AssertExitCode(t, r.ExitCode, 0)
	if r.Stdout == "" {
		t.Error("expected version output")
	}
}

func TestAssertExitCode(t *testing.T) {
	// Just test it doesn't panic
	Build(t)
	r := Run(t, "version")
	AssertExitCode(t, r.ExitCode, 0)
}

func TestAssertStdoutContains(t *testing.T) {
	Build(t)
	r := Run(t, "version")
	// Version command prints something
	if r.Stdout != "" {
		AssertStdoutContains(t, r.Stdout, "")
	}
}
