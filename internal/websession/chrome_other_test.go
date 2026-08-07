//go:build !darwin

package websession

import (
	"context"
	"strings"
	"testing"
)

func TestImportChromeCookies_UnsupportedPlatform(t *testing.T) {
	_, err := ImportChromeCookies(context.Background())
	if err == nil || !strings.Contains(err.Error(), "--cookies") {
		t.Fatalf("ImportChromeCookies() error = %v, want manual fallback", err)
	}
}

func TestChromeBinary_UnsupportedPlatform(t *testing.T) {
	t.Setenv(chromeBinaryEnv, "")
	_, err := chromeBinary()
	if err == nil || !strings.Contains(err.Error(), chromeBinaryEnv) || !strings.Contains(err.Error(), "macOS") {
		t.Fatalf("chromeBinary() error = %v, want macOS-only message naming %s", err, chromeBinaryEnv)
	}
}
