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
