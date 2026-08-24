// Package localexec is the injectable boundary for explicitly requested local
// Android toolchain processes. Commands are executed directly, never by a shell.
package localexec

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

// Request describes one local process invocation.
type Request struct {
	Executable string
	Args       []string
	Dir        string
	Env        []string
	Stdin      io.Reader
	Stdout     io.Writer
	Stderr     io.Writer
}

// Runner executes a local process without shell interpretation.
type Runner interface {
	Run(context.Context, Request) error
}

// RunnerFunc adapts a function into a Runner.
type RunnerFunc func(context.Context, Request) error

func (f RunnerFunc) Run(ctx context.Context, request Request) error { return f(ctx, request) }

type runnerContextKey struct{}

// ContextWithRunner installs a test or host-specific local process boundary.
func ContextWithRunner(ctx context.Context, runner Runner) context.Context {
	if runner == nil {
		return ctx
	}
	return context.WithValue(ctx, runnerContextKey{}, runner)
}

// RunnerFrom returns the injected runner or the production direct-exec runner.
func RunnerFrom(ctx context.Context) Runner {
	if ctx != nil {
		if runner, ok := ctx.Value(runnerContextKey{}).(Runner); ok && runner != nil {
			return runner
		}
	}
	return systemRunner{}
}

type systemRunner struct{}

func (systemRunner) Run(ctx context.Context, request Request) error {
	if strings.TrimSpace(request.Executable) == "" {
		return fmt.Errorf("local executable is required")
	}
	command := exec.CommandContext(ctx, request.Executable, request.Args...)
	command.Dir = request.Dir
	if request.Env != nil {
		command.Env = request.Env
	} else {
		command.Env = os.Environ()
	}
	command.Stdin = request.Stdin
	command.Stdout = request.Stdout
	command.Stderr = request.Stderr
	return command.Run()
}
