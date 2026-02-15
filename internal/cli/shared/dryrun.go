package shared

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// dryRunKey is the context key for the dry-run flag.
type dryRunKey struct{}

// ContextWithDryRun returns a context with the dry-run flag set.
func ContextWithDryRun(ctx context.Context, dryRun bool) context.Context {
	return context.WithValue(ctx, dryRunKey{}, dryRun)
}

// IsDryRun returns true if the context has dry-run enabled.
func IsDryRun(ctx context.Context) bool {
	v, ok := ctx.Value(dryRunKey{}).(bool)
	return ok && v
}

// writeMethods are HTTP methods that mutate state.
var writeMethods = map[string]bool{
	http.MethodPost:   true,
	http.MethodPut:    true,
	http.MethodPatch:  true,
	http.MethodDelete: true,
}

// DryRunTransport wraps an http.RoundTripper and intercepts write requests
// when dry-run mode is active. GET/HEAD requests pass through normally.
// Write requests (POST, PUT, PATCH, DELETE) are logged to stderr and return
// a synthetic 200 OK response without making any actual API call.
type DryRunTransport struct {
	Base   http.RoundTripper
	Writer io.Writer // output destination (typically os.Stderr)
}

// RoundTrip implements http.RoundTripper.
func (t *DryRunTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if !writeMethods[req.Method] {
		return t.Base.RoundTrip(req)
	}

	w := t.Writer
	if w == nil {
		return t.Base.RoundTrip(req)
	}

	// Log the intercepted request.
	fmt.Fprintf(w, "[DRY RUN] %s %s\n", req.Method, req.URL.String())

	if req.Body != nil && req.Body != http.NoBody {
		body, err := io.ReadAll(req.Body)
		req.Body.Close()
		if err == nil && len(body) > 0 {
			// Truncate very large bodies (e.g., binary uploads) for readability.
			const maxBody = 2048
			display := string(body)
			if len(display) > maxBody {
				display = display[:maxBody] + "... (truncated)"
			}
			fmt.Fprintf(w, "[DRY RUN] Body: %s\n", strings.TrimSpace(display))
		}
	}

	fmt.Fprintf(w, "[DRY RUN] No changes were made.\n")

	// Return a synthetic successful response.
	return &http.Response{
		StatusCode: http.StatusOK,
		Status:     "200 OK",
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header:     http.Header{"Content-Type": {"application/json"}},
		Body:       io.NopCloser(bytes.NewBufferString("{}")),
		Request:    req,
	}, nil
}
