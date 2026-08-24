package runtime

import (
	"bytes"
	"context"
	"flag"
	"os"
	"testing"
	"time"

	"github.com/tamtom/play-console-cli/internal/appsigningclient"
	"github.com/tamtom/play-console-cli/internal/checksclient"
	"github.com/tamtom/play-console-cli/internal/cli/shared"
	"github.com/tamtom/play-console-cli/internal/customappsclient"
	"github.com/tamtom/play-console-cli/internal/developeridclient"
	"github.com/tamtom/play-console-cli/internal/gamesclient"
	"github.com/tamtom/play-console-cli/internal/gcsclient"
	"github.com/tamtom/play-console-cli/internal/integrityclient"
	"github.com/tamtom/play-console-cli/internal/playclient"
	"github.com/tamtom/play-console-cli/internal/reportingclient"
)

type testFilesystem struct{}

func (*testFilesystem) ReadFile(path string) ([]byte, error) { return []byte(path), nil }

func (*testFilesystem) AtomicWriteFile(string, []byte, os.FileMode, os.FileMode) error { return nil }

func (*testFilesystem) CreateExclusiveFile(string, []byte, os.FileMode, os.FileMode) error {
	return nil
}

func TestNewRoot_BindsRootFlags(t *testing.T) {
	fs := flag.NewFlagSet("gplay", flag.ContinueOnError)
	rt := NewRoot(fs)

	if rt == nil {
		t.Fatal("expected runtime")
		return
	}
	if rt.RootFlags == nil {
		t.Fatal("expected bound root flags")
	}
}

func TestEnsure_ReturnsDetachedRuntime(t *testing.T) {
	rt := Ensure(nil)
	if rt == nil {
		t.Fatal("expected detached runtime")
		return
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

func TestApplyRootContextInjectsPlayServiceForEveryCommandPackage(t *testing.T) {
	want := &playclient.Service{}
	calls := 0
	rt := NewDetached().WithPlayServiceFactory(func(context.Context) (*playclient.Service, error) {
		calls++
		return want, nil
	})
	ctx, err := rt.ApplyRootContext(context.Background())
	if err != nil {
		t.Fatalf("ApplyRootContext: %v", err)
	}
	got, err := playclient.NewService(ctx)
	if err != nil {
		t.Fatalf("playclient.NewService: %v", err)
	}
	if got != want || calls != 1 {
		t.Fatalf("injected service = %p calls=%d, want %p calls=1", got, calls, want)
	}
}

func TestApplyRootContextInjectsClock(t *testing.T) {
	want := time.Date(2042, time.March, 4, 5, 6, 7, 0, time.UTC)
	rt := NewDetached().WithClock(shared.ClockFunc(func() time.Time { return want }))
	ctx, err := rt.ApplyRootContext(context.Background())
	if err != nil {
		t.Fatalf("ApplyRootContext: %v", err)
	}
	if got := shared.Now(ctx); !got.Equal(want) {
		t.Fatalf("injected clock = %s, want %s", got, want)
	}
}

func TestApplyRootContextInjectsIO(t *testing.T) {
	var stdout, stderr bytes.Buffer
	rt := NewDetached().WithIO(&stdout, &stderr)
	ctx, err := rt.ApplyRootContext(context.Background())
	if err != nil {
		t.Fatalf("ApplyRootContext: %v", err)
	}
	_, _ = shared.Stdout(ctx).Write([]byte("out"))
	_, _ = shared.Stderr(ctx).Write([]byte("err"))
	if stdout.String() != "out" || stderr.String() != "err" {
		t.Fatalf("injected streams = %q, %q", stdout.String(), stderr.String())
	}
}

func TestApplyRootContextInjectsFilesystem(t *testing.T) {
	want := &testFilesystem{}
	rt := NewDetached().WithFilesystem(want)
	ctx, err := rt.ApplyRootContext(context.Background())
	if err != nil {
		t.Fatalf("ApplyRootContext: %v", err)
	}
	if got := shared.FilesystemFrom(ctx); got != want {
		t.Fatalf("filesystem = %#v, want %#v", got, want)
	}
}

func TestApplyRootContextInjectsEveryOfficialServiceFamily(t *testing.T) {
	wantReporting := &reportingclient.Service{}
	wantGames := &gamesclient.Service{}
	wantCustomApps := &customappsclient.Service{}
	wantGCS := &gcsclient.Service{}
	wantChecks := &checksclient.Service{}
	wantIntegrity := &integrityclient.Service{}
	wantAppSigning := &appsigningclient.Service{}
	wantDeveloperID := &developeridclient.Service{}
	rt := NewDetached().
		WithReportingServiceFactory(func(context.Context) (*reportingclient.Service, error) { return wantReporting, nil }).
		WithGamesServiceFactory(func(context.Context) (*gamesclient.Service, error) { return wantGames, nil }).
		WithCustomAppsServiceFactory(func(context.Context) (*customappsclient.Service, error) { return wantCustomApps, nil }).
		WithGCSServiceFactory(func(context.Context) (*gcsclient.Service, error) { return wantGCS, nil }).
		WithChecksServiceFactory(func(context.Context) (*checksclient.Service, error) { return wantChecks, nil }).
		WithIntegrityServiceFactory(func(context.Context) (*integrityclient.Service, error) { return wantIntegrity, nil }).
		WithAppSigningServiceFactory(func(context.Context) (*appsigningclient.Service, error) { return wantAppSigning, nil }).
		WithDeveloperIDServiceFactory(func(context.Context, string) (*developeridclient.Service, error) { return wantDeveloperID, nil })
	ctx, err := rt.ApplyRootContext(context.Background())
	if err != nil {
		t.Fatalf("ApplyRootContext: %v", err)
	}
	assertSameService(t, "reporting", wantReporting, func() (any, error) { return reportingclient.NewService(ctx) })
	assertSameService(t, "games", wantGames, func() (any, error) { return gamesclient.NewService(ctx) })
	assertSameService(t, "custom apps", wantCustomApps, func() (any, error) { return customappsclient.NewService(ctx) })
	assertSameService(t, "gcs", wantGCS, func() (any, error) { return gcsclient.NewService(ctx) })
	assertSameService(t, "checks", wantChecks, func() (any, error) { return checksclient.NewService(ctx) })
	assertSameService(t, "integrity", wantIntegrity, func() (any, error) { return integrityclient.NewService(ctx) })
	assertSameService(t, "app signing", wantAppSigning, func() (any, error) { return appsigningclient.NewService(ctx) })
	assertSameService(t, "developer ID", wantDeveloperID, func() (any, error) { return developeridclient.NewService(ctx, "test-key") })
}

func assertSameService(t *testing.T, name string, want any, factory func() (any, error)) {
	t.Helper()
	got, err := factory()
	if err != nil {
		t.Fatalf("%s service: %v", name, err)
	}
	if got != want {
		t.Fatalf("%s service = %p, want %p", name, got, want)
	}
}
