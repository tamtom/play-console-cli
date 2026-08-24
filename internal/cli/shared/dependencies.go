package shared

import (
	"context"
	"io"
	"os"
	"time"

	"github.com/tamtom/play-console-cli/internal/rootfs"
)

// Clock is the runtime time source used by commands that create or validate
// durable artifacts.
type Clock interface {
	Now() time.Time
}

// ClockFunc adapts a function into a Clock.
type ClockFunc func() time.Time

// Now implements Clock.
func (f ClockFunc) Now() time.Time { return f() }

type (
	clockContextKey      struct{}
	ioContextKey         struct{}
	filesystemContextKey struct{}
)

// Filesystem is the durable file boundary carried by the CLI runtime.
type Filesystem interface {
	ReadFile(path string) ([]byte, error)
	AtomicWriteFile(path string, data []byte, fileMode, dirMode os.FileMode) error
}

type systemFilesystem struct{}

func (systemFilesystem) ReadFile(path string) ([]byte, error) { return rootfs.ReadFile(path) }
func (systemFilesystem) AtomicWriteFile(path string, data []byte, fileMode, dirMode os.FileMode) error {
	return rootfs.AtomicWriteFile(path, data, fileMode, dirMode)
}

// ContextWithFilesystem installs a command-scoped durable file boundary.
func ContextWithFilesystem(ctx context.Context, filesystem Filesystem) context.Context {
	if filesystem == nil {
		return ctx
	}
	return context.WithValue(ctx, filesystemContextKey{}, filesystem)
}

// FilesystemFrom returns the command-scoped filesystem or the rooted
// production implementation.
func FilesystemFrom(ctx context.Context) Filesystem {
	if ctx != nil {
		if filesystem, ok := ctx.Value(filesystemContextKey{}).(Filesystem); ok && filesystem != nil {
			return filesystem
		}
	}
	return systemFilesystem{}
}

type commandIO struct {
	stdout io.Writer
	stderr io.Writer
}

// ContextWithIO installs command-scoped output streams.
func ContextWithIO(ctx context.Context, stdout, stderr io.Writer) context.Context {
	if stdout == nil {
		stdout = io.Discard
	}
	if stderr == nil {
		stderr = io.Discard
	}
	return context.WithValue(ctx, ioContextKey{}, commandIO{stdout: stdout, stderr: stderr})
}

// Stdout returns the command-scoped standard output stream.
func Stdout(ctx context.Context) io.Writer {
	if ctx != nil {
		if streams, ok := ctx.Value(ioContextKey{}).(commandIO); ok {
			return streams.stdout
		}
	}
	return os.Stdout
}

// Stderr returns the command-scoped standard error stream.
func Stderr(ctx context.Context) io.Writer {
	if ctx != nil {
		if streams, ok := ctx.Value(ioContextKey{}).(commandIO); ok {
			return streams.stderr
		}
	}
	return os.Stderr
}

// ContextWithClock installs a command-scoped clock.
func ContextWithClock(ctx context.Context, clock Clock) context.Context {
	if clock == nil {
		return ctx
	}
	return context.WithValue(ctx, clockContextKey{}, clock)
}

// Now reads the command-scoped clock or the production wall clock.
func Now(ctx context.Context) time.Time {
	if ctx != nil {
		if clock, ok := ctx.Value(clockContextKey{}).(Clock); ok && clock != nil {
			return clock.Now()
		}
	}
	return time.Now()
}
