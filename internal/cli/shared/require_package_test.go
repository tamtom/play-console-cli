package shared

import (
	"os"
	"strings"
	"testing"

	"github.com/tamtom/play-console-cli/internal/config"
)

func TestRequirePackageName_FromFlag(t *testing.T) {
	pkg, err := RequirePackageName("com.example.flag", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pkg != "com.example.flag" {
		t.Errorf("got %q; want %q", pkg, "com.example.flag")
	}
}

func TestRequirePackageName_FromEnvVar(t *testing.T) {
	t.Setenv("GPLAY_PACKAGE_NAME", "com.example.env")

	pkg, err := RequirePackageName("", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pkg != "com.example.env" {
		t.Errorf("got %q; want %q", pkg, "com.example.env")
	}
}

func TestRequirePackageName_FromConfig(t *testing.T) {
	// Ensure env var is not set
	os.Unsetenv("GPLAY_PACKAGE_NAME")

	cfg := &config.Config{PackageName: "com.example.config"}
	pkg, err := RequirePackageName("", cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pkg != "com.example.config" {
		t.Errorf("got %q; want %q", pkg, "com.example.config")
	}
}

func TestRequirePackageName_ErrorWhenMissing(t *testing.T) {
	// Ensure env var is not set
	os.Unsetenv("GPLAY_PACKAGE_NAME")

	_, err := RequirePackageName("", nil)
	if err == nil {
		t.Fatal("expected error when package name is not found")
	}
	msg := err.Error()
	if !strings.Contains(msg, "--package") {
		t.Errorf("error message %q should mention --package", msg)
	}
	if !strings.Contains(msg, "GPLAY_PACKAGE_NAME") {
		t.Errorf("error message %q should mention GPLAY_PACKAGE_NAME", msg)
	}
	if !strings.Contains(msg, "config") {
		t.Errorf("error message %q should mention config", msg)
	}
}

func TestRequirePackageName_TrimsWhitespace(t *testing.T) {
	pkg, err := RequirePackageName("  com.example.trimmed  ", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pkg != "com.example.trimmed" {
		t.Errorf("got %q; want %q", pkg, "com.example.trimmed")
	}
}

func TestRequirePackageName_WhitespaceOnlyIsEmpty(t *testing.T) {
	os.Unsetenv("GPLAY_PACKAGE_NAME")

	_, err := RequirePackageName("   ", nil)
	if err == nil {
		t.Fatal("expected error when package name is whitespace-only")
	}
}
