package main

import (
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestReleaseBuildEmbedsVersionMetadata(t *testing.T) {
	binary := filepath.Join(t.TempDir(), "gplay")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}

	const (
		wantVersion = "v9.8.7"
		wantCommit  = "abc123"
		wantDate    = "2026-08-24T18:00:00Z"
	)
	ldflags := strings.Join([]string{
		"-X github.com/tamtom/play-console-cli/internal/version.Version=" + wantVersion,
		"-X github.com/tamtom/play-console-cli/internal/version.Commit=" + wantCommit,
		"-X github.com/tamtom/play-console-cli/internal/version.BuildDate=" + wantDate,
	}, " ")

	build := exec.Command("go", "build", "-trimpath", "-ldflags", ldflags, "-o", binary, ".") // #nosec G204 -- fixed build arguments exercise the release boundary
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build release binary: %v\n%s", err, output)
	}

	output, err := exec.Command(binary, "--version").CombinedOutput() // #nosec G204 -- binary is built into t.TempDir
	if err != nil {
		t.Fatalf("run release binary: %v\n%s", err, output)
	}
	got := strings.TrimSpace(string(output))
	want := wantVersion + " (commit: " + wantCommit + ", date: " + wantDate + ")"
	if got != want {
		t.Fatalf("release binary version = %q, want %q", got, want)
	}
}
