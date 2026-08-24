package bootstrap

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildPlan_IsDeterministic(t *testing.T) {
	artifactPath := writeArtifact(t)
	request := Request{
		PackageName:  "dev.acme.application",
		AppName:      "Acme",
		ArtifactPath: artifactPath,
	}

	first, err := BuildPlan(request)
	if err != nil {
		t.Fatal(err)
	}
	second, err := BuildPlan(request)
	if err != nil {
		t.Fatal(err)
	}
	if first.ID != second.ID {
		t.Fatalf("plan IDs differ: %q != %q", first.ID, second.ID)
	}
	if !strings.HasPrefix(first.ID, "sha256:") {
		t.Fatalf("unexpected plan ID: %q", first.ID)
	}
}

func TestBuildPlan_RejectsPlaceholderPackage(t *testing.T) {
	_, err := BuildPlan(Request{
		PackageName:  "com.example.application",
		AppName:      "Example",
		ArtifactPath: writeArtifact(t),
	})
	if err == nil {
		t.Fatal("expected placeholder package error")
	}
	if !strings.Contains(err.Error(), "placeholder") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func writeArtifact(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "app.aab")
	if err := os.WriteFile(path, []byte("test-aab"), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}
