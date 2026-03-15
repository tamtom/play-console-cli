package workflow

import (
	"os"
	"testing"
)

func TestInterpolate_BasicSubstitution(t *testing.T) {
	vars := map[string]string{
		"NAME":    "myapp",
		"VERSION": "1.0.0",
	}

	result, err := Interpolate("gplay release --package {{ .NAME }} --version {{ .VERSION }}", vars)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "gplay release --package myapp --version 1.0.0"
	if result != want {
		t.Errorf("got %q, want %q", result, want)
	}
}

func TestInterpolate_NoVariables(t *testing.T) {
	result, err := Interpolate("echo hello", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "echo hello" {
		t.Errorf("got %q, want %q", result, "echo hello")
	}
}

func TestInterpolate_UndefinedVariableError(t *testing.T) {
	_, err := Interpolate("echo {{ .MISSING }}", map[string]string{})
	if err == nil {
		t.Fatal("expected error for undefined variable")
	}
	if got := err.Error(); got != `undefined variable "MISSING"` {
		t.Errorf("got error %q, want %q", got, `undefined variable "MISSING"`)
	}
}

func TestInterpolate_DefaultValue(t *testing.T) {
	result, err := Interpolate(`echo {{ .HOST | default "localhost" }}`, map[string]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "echo localhost" {
		t.Errorf("got %q, want %q", result, "echo localhost")
	}
}

func TestInterpolate_DefaultOverriddenByVar(t *testing.T) {
	vars := map[string]string{"HOST": "production.example.com"}
	result, err := Interpolate(`echo {{ .HOST | default "localhost" }}`, vars)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "echo production.example.com" {
		t.Errorf("got %q, want %q", result, "echo production.example.com")
	}
}

func TestInterpolate_EmptyDefault(t *testing.T) {
	result, err := Interpolate(`echo "{{ .OPT | default "" }}"`, map[string]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != `echo ""` {
		t.Errorf("got %q, want %q", result, `echo ""`)
	}
}

func TestInterpolate_EnvVarFallback(t *testing.T) {
	t.Setenv("TEST_WORKFLOW_VAR", "from-env")

	result, err := Interpolate("echo {{ .TEST_WORKFLOW_VAR }}", map[string]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "echo from-env" {
		t.Errorf("got %q, want %q", result, "echo from-env")
	}
}

func TestInterpolate_VarOverridesEnv(t *testing.T) {
	os.Setenv("TEST_WORKFLOW_VAR2", "from-env")
	t.Cleanup(func() { os.Unsetenv("TEST_WORKFLOW_VAR2") })

	vars := map[string]string{"TEST_WORKFLOW_VAR2": "from-vars"}
	result, err := Interpolate("echo {{ .TEST_WORKFLOW_VAR2 }}", vars)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "echo from-vars" {
		t.Errorf("got %q, want %q", result, "echo from-vars")
	}
}

func TestInterpolate_MultipleVariables(t *testing.T) {
	vars := map[string]string{
		"A": "alpha",
		"B": "beta",
		"C": "gamma",
	}
	result, err := Interpolate("{{ .A }}-{{ .B }}-{{ .C }}", vars)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "alpha-beta-gamma" {
		t.Errorf("got %q, want %q", result, "alpha-beta-gamma")
	}
}

func TestInterpolate_WhitespaceFlexibility(t *testing.T) {
	vars := map[string]string{"X": "value"}

	tests := []struct {
		template string
	}{
		{"{{.X}}"},
		{"{{ .X}}"},
		{"{{.X }}"},
		{"{{ .X }}"},
		{"{{  .X  }}"},
	}

	for _, tt := range tests {
		result, err := Interpolate(tt.template, vars)
		if err != nil {
			t.Errorf("Interpolate(%q): unexpected error: %v", tt.template, err)
			continue
		}
		if result != "value" {
			t.Errorf("Interpolate(%q) = %q, want %q", tt.template, result, "value")
		}
	}
}
