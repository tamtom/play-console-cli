package runtime

import (
	"context"
	"flag"

	"github.com/tamtom/play-console-cli/internal/checksclient"
	"github.com/tamtom/play-console-cli/internal/cli/shared"
	"github.com/tamtom/play-console-cli/internal/customappsclient"
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

	newPlayService       playclient.ServiceFactory
	newReportingService  reportingclient.ServiceFactory
	newGamesService      gamesclient.ServiceFactory
	newCustomAppsService customappsclient.ServiceFactory
	newGCSService        gcsclient.ServiceFactory
	newChecksService     checksclient.ServiceFactory
	newIntegrityService  integrityclient.ServiceFactory
}

// NewDetached constructs a runtime for command packages that do not need root
// flag binding but still want shared client factories.
func NewDetached() *Runtime {
	return &Runtime{}
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
