package runtime

import (
	"context"
	"flag"
	"io"

	"github.com/tamtom/play-console-cli/internal/appsigningclient"
	"github.com/tamtom/play-console-cli/internal/audit"
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

// Runtime owns cross-cutting CLI wiring that should not be spread across
// commands and shared helpers.
type Runtime struct {
	RootFlags *shared.RootFlags

	newPlayService        playclient.ServiceFactory
	newReportingService   reportingclient.ServiceFactory
	newGamesService       gamesclient.ServiceFactory
	newCustomAppsService  customappsclient.ServiceFactory
	newGCSService         gcsclient.ServiceFactory
	newChecksService      checksclient.ServiceFactory
	newIntegrityService   integrityclient.ServiceFactory
	newAppSigningService  appsigningclient.ServiceFactory
	newDeveloperIDService developeridclient.ServiceFactory
	clock                 shared.Clock
	stdout                io.Writer
	stderr                io.Writer
	auditSink             AuditSink
	filesystem            shared.Filesystem
}

// AuditSink receives completed command audit entries.
type AuditSink interface {
	Enabled() bool
	Write(audit.Entry) error
}

type productionAuditSink struct{}

func (productionAuditSink) Enabled() bool                 { return audit.Enabled() }
func (productionAuditSink) Write(entry audit.Entry) error { return audit.Write(entry) }

// NewDetached constructs a runtime for command packages that do not need root
// flag binding but still want shared client factories.
func NewDetached() *Runtime {
	return &Runtime{auditSink: productionAuditSink{}}
}

// WithPlayServiceFactory returns rt configured with a context-scoped Android
// Publisher service factory. It is primarily used by black-box command tests.
func (rt *Runtime) WithPlayServiceFactory(factory playclient.ServiceFactory) *Runtime {
	rt = Ensure(rt)
	rt.newPlayService = factory
	return rt
}

func (rt *Runtime) WithReportingServiceFactory(factory reportingclient.ServiceFactory) *Runtime {
	rt = Ensure(rt)
	rt.newReportingService = factory
	return rt
}

func (rt *Runtime) WithGamesServiceFactory(factory gamesclient.ServiceFactory) *Runtime {
	rt = Ensure(rt)
	rt.newGamesService = factory
	return rt
}

func (rt *Runtime) WithCustomAppsServiceFactory(factory customappsclient.ServiceFactory) *Runtime {
	rt = Ensure(rt)
	rt.newCustomAppsService = factory
	return rt
}

func (rt *Runtime) WithGCSServiceFactory(factory gcsclient.ServiceFactory) *Runtime {
	rt = Ensure(rt)
	rt.newGCSService = factory
	return rt
}

func (rt *Runtime) WithChecksServiceFactory(factory checksclient.ServiceFactory) *Runtime {
	rt = Ensure(rt)
	rt.newChecksService = factory
	return rt
}

func (rt *Runtime) WithIntegrityServiceFactory(factory integrityclient.ServiceFactory) *Runtime {
	rt = Ensure(rt)
	rt.newIntegrityService = factory
	return rt
}

// WithAppSigningServiceFactory installs the enterprise App Signing boundary.
func (rt *Runtime) WithAppSigningServiceFactory(factory appsigningclient.ServiceFactory) *Runtime {
	rt = Ensure(rt)
	rt.newAppSigningService = factory
	return rt
}

// WithDeveloperIDServiceFactory installs the official Developer ID Status boundary.
func (rt *Runtime) WithDeveloperIDServiceFactory(factory developeridclient.ServiceFactory) *Runtime {
	rt = Ensure(rt)
	rt.newDeveloperIDService = factory
	return rt
}

// WithClock returns rt configured with a command-scoped time source.
func (rt *Runtime) WithClock(clock shared.Clock) *Runtime {
	rt = Ensure(rt)
	rt.clock = clock
	return rt
}

// WithIO returns rt configured with command-scoped output streams.
func (rt *Runtime) WithIO(stdout, stderr io.Writer) *Runtime {
	rt = Ensure(rt)
	rt.stdout = stdout
	rt.stderr = stderr
	return rt
}

// WithAuditSink returns rt configured with a command audit destination.
func (rt *Runtime) WithAuditSink(sink AuditSink) *Runtime {
	rt = Ensure(rt)
	rt.auditSink = sink
	return rt
}

// WithFilesystem returns rt configured with a command-scoped durable file
// boundary.
func (rt *Runtime) WithFilesystem(filesystem shared.Filesystem) *Runtime {
	rt = Ensure(rt)
	rt.filesystem = filesystem
	return rt
}

// AuditSink returns the configured audit destination.
func (rt *Runtime) AuditSink() AuditSink {
	rt = Ensure(rt)
	return rt.auditSink
}

// NewRoot constructs a runtime and binds root-level flags to the provided
// FlagSet.
func NewRoot(fs *flag.FlagSet) *Runtime {
	rt := NewDetached()
	if fs != nil {
		rt.RootFlags = shared.BindRootFlags(fs)
	}
	return rt
}

// Ensure returns rt when non-nil and otherwise creates a detached runtime.
func Ensure(rt *Runtime) *Runtime {
	if rt != nil {
		return rt
	}
	return NewDetached()
}

// ApplyRootContext applies root flag side effects and returns the derived
// execution context.
func (rt *Runtime) ApplyRootContext(ctx context.Context) (context.Context, error) {
	if rt != nil && rt.clock != nil {
		ctx = shared.ContextWithClock(ctx, rt.clock)
	}
	if rt != nil && (rt.stdout != nil || rt.stderr != nil) {
		ctx = shared.ContextWithIO(ctx, rt.stdout, rt.stderr)
	}
	if rt != nil && rt.filesystem != nil {
		ctx = shared.ContextWithFilesystem(ctx, rt.filesystem)
	}
	if rt != nil && rt.newPlayService != nil {
		ctx = playclient.ContextWithServiceFactory(ctx, rt.newPlayService)
	}
	if rt != nil && rt.newReportingService != nil {
		ctx = reportingclient.ContextWithServiceFactory(ctx, rt.newReportingService)
	}
	if rt != nil && rt.newGamesService != nil {
		ctx = gamesclient.ContextWithServiceFactory(ctx, rt.newGamesService)
	}
	if rt != nil && rt.newCustomAppsService != nil {
		ctx = customappsclient.ContextWithServiceFactory(ctx, rt.newCustomAppsService)
	}
	if rt != nil && rt.newGCSService != nil {
		ctx = gcsclient.ContextWithServiceFactory(ctx, rt.newGCSService)
	}
	if rt != nil && rt.newChecksService != nil {
		ctx = checksclient.ContextWithServiceFactory(ctx, rt.newChecksService)
	}
	if rt != nil && rt.newIntegrityService != nil {
		ctx = integrityclient.ContextWithServiceFactory(ctx, rt.newIntegrityService)
	}
	if rt != nil && rt.newAppSigningService != nil {
		ctx = appsigningclient.ContextWithServiceFactory(ctx, rt.newAppSigningService)
	}
	if rt != nil && rt.newDeveloperIDService != nil {
		ctx = developeridclient.ContextWithServiceFactory(ctx, rt.newDeveloperIDService)
	}
	if rt == nil || rt.RootFlags == nil {
		return ctx, nil
	}

	rt.RootFlags.Apply()
	if err := rt.RootFlags.ValidateReportFlags(); err != nil {
		return ctx, err
	}
	if rt.RootFlags.DryRun != nil && *rt.RootFlags.DryRun {
		ctx = shared.ContextWithDryRun(ctx, true)
	}

	return ctx, nil
}

// NewPlayService creates an authenticated Android Publisher service.
func (rt *Runtime) NewPlayService(ctx context.Context) (*playclient.Service, error) {
	rt = Ensure(rt)
	if rt.newPlayService == nil {
		return playclient.NewService(ctx)
	}
	return rt.newPlayService(ctx)
}
