package shared

import (
	"context"
	"testing"
)

func TestContextWithTimeout_NilConfigDoesNotPanic(t *testing.T) {
	ctx, cancel := ContextWithTimeout(context.Background(), nil)
	defer cancel()

	if ctx == nil {
		t.Fatal("expected context")
	}
}

func TestContextWithUploadTimeout_NilConfigDoesNotPanic(t *testing.T) {
	ctx, cancel := ContextWithUploadTimeout(context.Background(), nil)
	defer cancel()

	if ctx == nil {
		t.Fatal("expected context")
	}
}
