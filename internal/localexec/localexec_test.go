package localexec

import (
	"context"
	"testing"
)

func TestContextRunnerIsUsed(t *testing.T) {
	called := false
	runner := RunnerFunc(func(context.Context, Request) error {
		called = true
		return nil
	})
	ctx := ContextWithRunner(context.Background(), runner)
	if err := RunnerFrom(ctx).Run(ctx, Request{Executable: "ignored"}); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("injected runner was not called")
	}
}

func TestSystemRunnerRejectsEmptyExecutable(t *testing.T) {
	err := (systemRunner{}).Run(context.Background(), Request{})
	if err == nil {
		t.Fatal("expected empty executable error")
	}
}
