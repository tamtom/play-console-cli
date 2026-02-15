package shared

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestContextDryRun(t *testing.T) {
	t.Run("default context is not dry-run", func(t *testing.T) {
		ctx := context.Background()
		if IsDryRun(ctx) {
			t.Fatal("expected IsDryRun to be false on background context")
		}
	})

	t.Run("context with dry-run true", func(t *testing.T) {
		ctx := ContextWithDryRun(context.Background(), true)
		if !IsDryRun(ctx) {
			t.Fatal("expected IsDryRun to be true")
		}
	})

	t.Run("context with dry-run false", func(t *testing.T) {
		ctx := ContextWithDryRun(context.Background(), false)
		if IsDryRun(ctx) {
			t.Fatal("expected IsDryRun to be false when explicitly set to false")
		}
	})
}

// fakeTransport records whether RoundTrip was called.
type fakeTransport struct {
	called  bool
	lastReq *http.Request
}

func (f *fakeTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	f.called = true
	f.lastReq = req
	return &http.Response{
		StatusCode: http.StatusOK,
		Status:     "200 OK",
		Header:     http.Header{},
		Body:       io.NopCloser(strings.NewReader(`{"real":"response"}`)),
		Request:    req,
	}, nil
}

func TestDryRunTransport_InterceptsWriteMethods(t *testing.T) {
	methods := []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			base := &fakeTransport{}
			var buf bytes.Buffer
			transport := &DryRunTransport{Base: base, Writer: &buf}

			body := `{"track":"production"}`
			req, err := http.NewRequest(method, "https://androidpublisher.googleapis.com/v3/test", strings.NewReader(body))
			if err != nil {
				t.Fatal(err)
			}

			resp, err := transport.RoundTrip(req)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if base.called {
				t.Fatal("expected base transport NOT to be called for write method")
			}

			if resp.StatusCode != http.StatusOK {
				t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
			}

			output := buf.String()
			if !strings.Contains(output, "[DRY RUN]") {
				t.Fatalf("expected [DRY RUN] prefix in output, got: %s", output)
			}
			if !strings.Contains(output, method) {
				t.Fatalf("expected method %s in output, got: %s", method, output)
			}
			if !strings.Contains(output, "https://androidpublisher.googleapis.com/v3/test") {
				t.Fatalf("expected URL in output, got: %s", output)
			}
			if !strings.Contains(output, `"track":"production"`) {
				t.Fatalf("expected body in output, got: %s", output)
			}
			if !strings.Contains(output, "No changes were made.") {
				t.Fatalf("expected 'No changes were made.' in output, got: %s", output)
			}
		})
	}
}

func TestDryRunTransport_PassesThroughGET(t *testing.T) {
	base := &fakeTransport{}
	var buf bytes.Buffer
	transport := &DryRunTransport{Base: base, Writer: &buf}

	req, err := http.NewRequest(http.MethodGet, "https://androidpublisher.googleapis.com/v3/test", nil)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := transport.RoundTrip(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !base.called {
		t.Fatal("expected base transport to be called for GET")
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}

	if buf.Len() != 0 {
		t.Fatalf("expected no dry-run output for GET, got: %s", buf.String())
	}
}

func TestDryRunTransport_PassesThroughHEAD(t *testing.T) {
	base := &fakeTransport{}
	var buf bytes.Buffer
	transport := &DryRunTransport{Base: base, Writer: &buf}

	req, err := http.NewRequest(http.MethodHead, "https://example.com/test", nil)
	if err != nil {
		t.Fatal(err)
	}

	_, err = transport.RoundTrip(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !base.called {
		t.Fatal("expected base transport to be called for HEAD")
	}

	if buf.Len() != 0 {
		t.Fatalf("expected no dry-run output for HEAD, got: %s", buf.String())
	}
}

func TestDryRunTransport_NoBody(t *testing.T) {
	base := &fakeTransport{}
	var buf bytes.Buffer
	transport := &DryRunTransport{Base: base, Writer: &buf}

	req, err := http.NewRequest(http.MethodDelete, "https://androidpublisher.googleapis.com/v3/resource/123", nil)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := transport.RoundTrip(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if base.called {
		t.Fatal("expected base transport NOT to be called")
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}

	output := buf.String()
	if !strings.Contains(output, "[DRY RUN] DELETE") {
		t.Fatalf("expected DELETE in output, got: %s", output)
	}
	// Should not contain "Body:" since there was no body
	if strings.Contains(output, "Body:") {
		t.Fatalf("unexpected Body line in output for bodyless request: %s", output)
	}
	if !strings.Contains(output, "No changes were made.") {
		t.Fatalf("expected 'No changes were made.' in output, got: %s", output)
	}
}

func TestDryRunTransport_LargeBodyTruncated(t *testing.T) {
	base := &fakeTransport{}
	var buf bytes.Buffer
	transport := &DryRunTransport{Base: base, Writer: &buf}

	// Create a body larger than 2048 bytes.
	largeBody := strings.Repeat("x", 3000)
	req, err := http.NewRequest(http.MethodPost, "https://example.com/upload", strings.NewReader(largeBody))
	if err != nil {
		t.Fatal(err)
	}

	_, err = transport.RoundTrip(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "... (truncated)") {
		t.Fatalf("expected truncation marker in output, got: %s", output)
	}
}

func TestDryRunTransport_SyntheticResponseBody(t *testing.T) {
	base := &fakeTransport{}
	var buf bytes.Buffer
	transport := &DryRunTransport{Base: base, Writer: &buf}

	req, err := http.NewRequest(http.MethodPost, "https://example.com/test", strings.NewReader(`{}`))
	if err != nil {
		t.Fatal(err)
	}

	resp, err := transport.RoundTrip(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}
	if string(body) != "{}" {
		t.Fatalf("expected empty JSON body '{}', got: %s", string(body))
	}

	if resp.Header.Get("Content-Type") != "application/json" {
		t.Fatalf("expected application/json content type, got: %s", resp.Header.Get("Content-Type"))
	}
}

func TestDryRunTransport_OutputFormat(t *testing.T) {
	base := &fakeTransport{}
	var buf bytes.Buffer
	transport := &DryRunTransport{Base: base, Writer: &buf}

	body := `{"releases":[{"status":"completed","versionCodes":["42"]}]}`
	req, err := http.NewRequest(http.MethodPut, "https://androidpublisher.googleapis.com/androidpublisher/v3/applications/com.example.app/edits/123/tracks/production", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}

	_, err = transport.RoundTrip(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines of output, got %d: %s", len(lines), output)
	}

	if !strings.HasPrefix(lines[0], "[DRY RUN] PUT ") {
		t.Fatalf("line 0 should start with '[DRY RUN] PUT ', got: %s", lines[0])
	}
	if !strings.HasPrefix(lines[1], "[DRY RUN] Body: ") {
		t.Fatalf("line 1 should start with '[DRY RUN] Body: ', got: %s", lines[1])
	}
	if lines[2] != "[DRY RUN] No changes were made." {
		t.Fatalf("line 2 should be '[DRY RUN] No changes were made.', got: %s", lines[2])
	}
}
