package shared

import (
	"os"
	"testing"
)

func TestResolveOutputFormat_FlagOverridesEnv(t *testing.T) {
	t.Setenv("GPLAY_DEFAULT_OUTPUT", "table")
	got := ResolveOutputFormat("markdown", "json")
	if got != "markdown" {
		t.Errorf("got %q, want %q", got, "markdown")
	}
}

func TestResolveOutputFormat_EnvUsedWhenFlagDefault(t *testing.T) {
	t.Setenv("GPLAY_DEFAULT_OUTPUT", "table")
	got := ResolveOutputFormat("json", "json")
	if got != "table" {
		t.Errorf("got %q, want %q", got, "table")
	}
}

func TestResolveOutputFormat_DefaultJson(t *testing.T) {
	os.Unsetenv("GPLAY_DEFAULT_OUTPUT")
	got := ResolveOutputFormat("json", "json")
	if got != "json" {
		t.Errorf("got %q, want %q", got, "json")
	}
}

func TestResolveOutputFormat_InvalidEnv(t *testing.T) {
	t.Setenv("GPLAY_DEFAULT_OUTPUT", "invalid")
	got := ResolveOutputFormat("json", "json")
	if got != "json" {
		t.Errorf("got %q, want %q", got, "json")
	}
}

func TestResolveOutputFormat_EmptyEnv(t *testing.T) {
	t.Setenv("GPLAY_DEFAULT_OUTPUT", "")
	got := ResolveOutputFormat("json", "json")
	if got != "json" {
		t.Errorf("got %q, want %q", got, "json")
	}
}
