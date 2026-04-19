package preflight

import (
	"archive/zip"
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	preflightpkg "github.com/tamtom/play-console-cli/internal/preflight"
)

func writeAAB(t *testing.T, path string, entries map[string][]byte) {
	t.Helper()
	buf := &bytes.Buffer{}
	w := zip.NewWriter(buf)
	for name, body := range entries {
		f, err := w.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.Write(body); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
}

func minimalAAB() map[string][]byte {
	return map[string][]byte{
		"base/manifest/AndroidManifest.xml": []byte("manifest"),
		"base/resources.pb":                 []byte("res"),
		"base/lib/arm64-v8a/libapp.so":      []byte("x"),
	}
}

func TestPreflightRequiresFile(t *testing.T) {
	cmd := PreflightCommand()
	err := cmd.Exec(context.Background(), nil)
	if err == nil || !strings.Contains(err.Error(), "--file") {
		t.Fatalf("expected --file error, got %v", err)
	}
}

func TestPreflightCleanRun(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app.aab")
	writeAAB(t, path, minimalAAB())
	cmd := PreflightCommand()
	_ = cmd.FlagSet.Parse([]string{"--file", path, "--output", "json"})
	if err := cmd.Exec(context.Background(), nil); err != nil {
		t.Fatalf("expected clean run, got %v", err)
	}
}

func TestPreflightFailsOnError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app.aab")
	// missing manifest => SeverityError
	writeAAB(t, path, map[string][]byte{"base/resources.pb": []byte("r")})
	cmd := PreflightCommand()
	_ = cmd.FlagSet.Parse([]string{"--file", path, "--output", "json"})
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected non-nil error from failing preflight")
	}
	if !strings.Contains(err.Error(), "error") {
		t.Errorf("unexpected error text: %v", err)
	}
}

func TestPreflightFailOnWarning(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app.aab")
	entries := minimalAAB()
	entries["base/assets/.DS_Store"] = []byte{0} // -> warning
	writeAAB(t, path, entries)
	cmd := PreflightCommand()
	_ = cmd.FlagSet.Parse([]string{"--file", path, "--output", "json", "--fail-on", "warning"})
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected warning-level failure")
	}
}

func TestPreflightIgnoresInfoWhenFailOnError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app.aab")
	entries := minimalAAB()
	entries["base/manifest/AndroidManifest.xml"] = []byte("android.permission.READ_SMS") // info finding
	writeAAB(t, path, entries)
	cmd := PreflightCommand()
	_ = cmd.FlagSet.Parse([]string{"--file", path, "--output", "json"})
	if err := cmd.Exec(context.Background(), nil); err != nil {
		t.Errorf("info-only findings should not fail --fail-on=error, got %v", err)
	}
}

func TestParseSeverity(t *testing.T) {
	cases := map[string]preflightpkg.Severity{
		"info":    preflightpkg.SeverityInfo,
		"warning": preflightpkg.SeverityWarning,
		"warn":    preflightpkg.SeverityWarning,
		"error":   preflightpkg.SeverityError,
	}
	for in, want := range cases {
		got, err := parseSeverity(in)
		if err != nil {
			t.Errorf("%q: %v", in, err)
			continue
		}
		if got != want {
			t.Errorf("%q -> %q want %q", in, got, want)
		}
	}
	if _, err := parseSeverity("bogus"); err == nil {
		t.Error("expected error")
	}
}

func TestPreflightInvalidMaxSize(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app.aab")
	writeAAB(t, path, minimalAAB())
	cmd := PreflightCommand()
	_ = cmd.FlagSet.Parse([]string{"--file", path, "--max-size", "weird"})
	err := cmd.Exec(context.Background(), nil)
	if err == nil || !strings.Contains(err.Error(), "--max-size") {
		t.Fatalf("expected --max-size error, got %v", err)
	}
}
