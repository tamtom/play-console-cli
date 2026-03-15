package shared

import (
	"errors"
	"flag"
	"os"
	"strings"
	"testing"
)

func TestUsageError_ReturnsFlagErrHelp(t *testing.T) {
	err := UsageError("missing --package flag")
	if !errors.Is(err, flag.ErrHelp) {
		t.Errorf("expected flag.ErrHelp, got %v", err)
	}
}

func TestUsageError_PrintsToStderr(t *testing.T) {
	// Capture stderr
	oldStderr := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	os.Stderr = w

	_ = UsageError("missing --package flag")

	w.Close()
	os.Stderr = oldStderr

	buf := make([]byte, 1024)
	n, _ := r.Read(buf)
	r.Close()
	output := string(buf[:n])

	if !strings.Contains(output, "Error: missing --package flag") {
		t.Errorf("expected stderr to contain 'Error: missing --package flag', got %q", output)
	}
}

func TestUsageErrorf_FormatsMessage(t *testing.T) {
	// Capture stderr
	oldStderr := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	os.Stderr = w

	result := UsageErrorf("flag %q is required (got %d args)", "--package", 0)

	w.Close()
	os.Stderr = oldStderr

	buf := make([]byte, 1024)
	n, _ := r.Read(buf)
	r.Close()
	output := string(buf[:n])

	if !errors.Is(result, flag.ErrHelp) {
		t.Errorf("expected flag.ErrHelp, got %v", result)
	}

	expected := `Error: flag "--package" is required (got 0 args)`
	if !strings.Contains(output, expected) {
		t.Errorf("expected stderr to contain %q, got %q", expected, output)
	}
}
