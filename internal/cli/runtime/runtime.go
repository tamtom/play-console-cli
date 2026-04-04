package runtime

import (
	"context"
	"flag"

	"github.com/tamtom/play-console-cli/internal/cli/shared"
	"github.com/tamtom/play-console-cli/internal/playclient"
)

// Runtime owns cross-cutting CLI wiring that should not be spread across
// commands and shared helpers.
type Runtime struct {
	RootFlags *shared.RootFlags

	newPlayService func(context.Context) (*playclient.Service, error)
}

// NewDetached constructs a runtime for command packages that do not need root
// flag binding but still want shared client factories.
func NewDetached() *Runtime {
	return &Runtime{
		newPlayService: playclient.NewService,
	}
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
	return rt.newPlayService(ctx)
}
