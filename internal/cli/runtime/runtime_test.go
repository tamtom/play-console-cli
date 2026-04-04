package runtime

import (
	"context"
	"flag"
	"os"
	"testing"

	"github.com/tamtom/play-console-cli/internal/cli/shared"
)

func TestNewRoot_BindsRootFlags(t *testing.T) {
	fs := flag.NewFlagSet("gplay", flag.ContinueOnError)
	rt := NewRoot(fs)

	if rt == nil {
		t.Fatal("expected runtime")
	}
	if rt.RootFlags == nil {
		t.Fatal("expected bound root flags")
	}
}

func TestEnsure_ReturnsDetachedRuntime(t *testing.T) {
	rt := Ensure(nil)
	if rt == nil {
		t.Fatal("expected detached runtime")
	}
	if rt.RootFlags != nil {
		t.Fatal("detached runtime should not bind root flags")
	}
}

func TestApplyRootContext_AppliesEnvAndDryRun(t *testing.T) {
	t.Setenv("GPLAY_PROFILE", "")
	t.Setenv("GPLAY_DEBUG", "")

	fs := flag.NewFlagSet("gplay", flag.ContinueOnError)
	rt := NewRoot(fs)
	if err := fs.Parse([]string{"--profile", "staging", "--debug", "--dry-run"}); err != nil {
		t.Fatalf("parse flags: %v", err)
	}

	ctx, err := rt.ApplyRootContext(context.Background())
	if err != nil {
		t.Fatalf("ApplyRootContext: %v", err)
	}

	if got := os.Getenv("GPLAY_PROFILE"); got != "staging" {
		t.Fatalf("GPLAY_PROFILE = %q, want %q", got, "staging")
	}
	if got := os.Getenv("GPLAY_DEBUG"); got != "1" {
		t.Fatalf("GPLAY_DEBUG = %q, want %q", got, "1")
	}
	if !shared.IsDryRun(ctx) {
		t.Fatal("expected dry-run context")
	}
}

func TestApplyRootContext_ValidatesReportFlags(t *testing.T) {
	fs := flag.NewFlagSet("gplay", flag.ContinueOnError)
	rt := NewRoot(fs)
	if err := fs.Parse([]string{"--report", "junit"}); err != nil {
		t.Fatalf("parse flags: %v", err)
	}

	if _, err := rt.ApplyRootContext(context.Background()); err == nil {
		t.Fatal("expected report flag validation error")
	}
}
