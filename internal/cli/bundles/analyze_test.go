package bundles

import (
	"archive/zip"
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeZip(t *testing.T, path string, entries map[string][]byte) {
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

func TestAnalyzeCommandRequiresFile(t *testing.T) {
	cmd := AnalyzeCommand()
	err := cmd.Exec(context.Background(), nil)
	if err == nil || !strings.Contains(err.Error(), "--file") {
		t.Fatalf("expected --file error, got %v", err)
	}
}

func TestAnalyzeCommandValidatesTop(t *testing.T) {
	cmd := AnalyzeCommand()
	_ = cmd.FlagSet.Parse([]string{"--file", "x.aab", "--top-files", "-1"})
	err := cmd.Exec(context.Background(), nil)
	if err == nil || !strings.Contains(err.Error(), "--top-files") {
		t.Fatalf("expected --top-files error, got %v", err)
	}
}

func TestAnalyzeCommandRuns(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app.aab")
	writeZip(t, path, map[string][]byte{
		"base/dex/classes.dex": bytes.Repeat([]byte("a"), 100),
	})
	cmd := AnalyzeCommand()
	_ = cmd.FlagSet.Parse([]string{"--file", path, "--output", "json"})
	if err := cmd.Exec(context.Background(), nil); err != nil {
		t.Fatalf("analyze: %v", err)
	}
}

func TestCompareCommandRequiresBoth(t *testing.T) {
	cmd := CompareCommand()
	_ = cmd.FlagSet.Parse([]string{"--base", "x.aab"})
	err := cmd.Exec(context.Background(), nil)
	if err == nil || !strings.Contains(err.Error(), "--candidate") {
		t.Fatalf("expected --candidate error, got %v", err)
	}
}

func TestCompareCommandRegressionExits(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "base.aab")
	cand := filepath.Join(dir, "cand.aab")
	writeZip(t, base, map[string][]byte{"base/dex/a.dex": []byte("a")})
	writeZip(t, cand, map[string][]byte{"base/dex/a.dex": bytes.Repeat([]byte("a"), 1000)})

	cmd := CompareCommand()
	_ = cmd.FlagSet.Parse([]string{"--base", base, "--candidate", cand, "--threshold", "100", "--output", "json"})
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected regression error")
	}
	if !strings.Contains(err.Error(), "regression") {
		t.Errorf("expected regression message, got %v", err)
	}
}

func TestCompareCommandNoRegression(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "base.aab")
	cand := filepath.Join(dir, "cand.aab")
	writeZip(t, base, map[string][]byte{"base/dex/a.dex": []byte("a")})
	writeZip(t, cand, map[string][]byte{"base/dex/a.dex": []byte("a")})
	cmd := CompareCommand()
	_ = cmd.FlagSet.Parse([]string{"--base", base, "--candidate", cand, "--output", "json"})
	if err := cmd.Exec(context.Background(), nil); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestBundlesSubcommandsIncludeAnalyze(t *testing.T) {
	cmd := BundlesCommand()
	have := map[string]bool{}
	for _, sub := range cmd.Subcommands {
		have[sub.Name] = true
	}
	for _, want := range []string{"upload", "list", "analyze", "compare"} {
		if !have[want] {
			t.Errorf("missing subcommand: %s", want)
		}
	}
}
