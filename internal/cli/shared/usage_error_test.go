package shared

import (
	"errors"
	"flag"
	"testing"
)

func TestUsageError_ReturnsFlagErrHelp(t *testing.T) {
	err := UsageError("missing --package flag")
	if !errors.Is(err, flag.ErrHelp) {
		t.Errorf("expected flag.ErrHelp, got %v", err)
	}
}

func TestUsageError_CarriesMessage(t *testing.T) {
	err := UsageError("missing --package flag")
	if got, want := err.Error(), "Error: missing --package flag"; got != want {
		t.Fatalf("error = %q, want %q", got, want)
	}
}

func TestUsageErrorf_FormatsMessage(t *testing.T) {
	result := UsageErrorf("flag %q is required (got %d args)", "--package", 0)

	if !errors.Is(result, flag.ErrHelp) {
		t.Errorf("expected flag.ErrHelp, got %v", result)
	}

	expected := `Error: flag "--package" is required (got 0 args)`
	if result.Error() != expected {
		t.Errorf("error = %q, want %q", result.Error(), expected)
	}
}
