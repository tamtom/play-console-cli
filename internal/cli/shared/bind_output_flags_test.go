package shared

import (
	"flag"
	"os"
	"testing"
)

func TestBindOutputFlags_DefaultsToJSON_WhenNotTerminal(t *testing.T) {
	// In test/CI environments, stdout is not a terminal, so the default should be "json"
	os.Unsetenv("GPLAY_DEFAULT_OUTPUT")
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	of := BindOutputFlags(fs)

	// When not a terminal and no env var, default should be "json"
	if of.Format() != "json" {
		t.Errorf("expected default format to be 'json' when not a terminal, got %q", of.Format())
	}
}

func TestBindOutputFlags_RespectsEnvVar(t *testing.T) {
	t.Setenv("GPLAY_DEFAULT_OUTPUT", "table")
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	of := BindOutputFlags(fs)

	if of.Format() != "table" {
		t.Errorf("expected format from env var to be 'table', got %q", of.Format())
	}
}

func TestOutputFlags_Format_ReturnsDefault(t *testing.T) {
	// Test with nil Output pointer
	of := &OutputFlags{}
	if of.Format() != "json" {
		t.Errorf("expected nil Output to default to 'json', got %q", of.Format())
	}

	// Test with set value
	val := "markdown"
	of2 := &OutputFlags{Output: &val}
	if of2.Format() != "markdown" {
		t.Errorf("expected format to be 'markdown', got %q", of2.Format())
	}
}

func TestOutputFlags_IsPretty_DefaultFalse(t *testing.T) {
	// Test with nil Pretty pointer
	of := &OutputFlags{}
	if of.IsPretty() {
		t.Error("expected nil Pretty to return false")
	}

	// Test with set value
	val := true
	of2 := &OutputFlags{Pretty: &val}
	if !of2.IsPretty() {
		t.Error("expected IsPretty to return true when set to true")
	}

	// Test with false
	val2 := false
	of3 := &OutputFlags{Pretty: &val2}
	if of3.IsPretty() {
		t.Error("expected IsPretty to return false when set to false")
	}
}
