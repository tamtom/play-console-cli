package sandbox

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func startServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv, _, err := Start()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(srv.Close)
	return srv
}

func do(t *testing.T, method, url string, body string) (int, map[string]any) {
	t.Helper()
	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	req, err := http.NewRequest(method, url, reader)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	parsed := map[string]any{}
	_ = json.Unmarshal(raw, &parsed)
	return resp.StatusCode, parsed
}

func TestSandbox_TokenEndpoint(t *testing.T) {
	srv := startServer(t)
	code, body := do(t, "POST", srv.URL+"/token", "grant_type=jwt")
	if code != 200 || body["access_token"] != "sandbox-token" {
		t.Fatalf("token endpoint: code=%d body=%v", code, body)
	}
}

func TestSandbox_UnknownPathIs404(t *testing.T) {
	srv := startServer(t)
	code, body := do(t, "GET", srv.URL+"/androidpublisher/v3/applications/"+Package+"/nonexistent", "")
	if code != 404 {
		t.Fatalf("unknown path: code=%d body=%v", code, body)
	}
	errObj, _ := body["error"].(map[string]any)
	msg, _ := errObj["message"].(string)
	if !strings.Contains(msg, "no official androidpublisher method") {
		t.Fatalf("unknown path must name the catalog check, got: %v", body)
	}
}

func TestSandbox_KnownButUnimplementedIs501(t *testing.T) {
	srv := startServer(t)
	// dataSafety is in the catalog but has no sandbox handler.
	code, body := do(t, "POST", srv.URL+"/androidpublisher/v3/applications/"+Package+"/dataSafety", "{}")
	if code != 501 {
		t.Fatalf("unimplemented method: code=%d body=%v", code, body)
	}
}

func TestSandbox_UnknownPackageIs404(t *testing.T) {
	srv := startServer(t)
	code, _ := do(t, "POST", srv.URL+"/androidpublisher/v3/applications/com.other.app/edits", "")
	if code != 404 {
		t.Fatalf("foreign package must 404, got %d", code)
	}
}

func TestSandbox_EditLifecycle(t *testing.T) {
	srv := startServer(t)
	base := srv.URL + "/androidpublisher/v3/applications/" + Package

	code, created := do(t, "POST", base+"/edits", "")
	if code != 200 || created["id"] == nil {
		t.Fatalf("insert: code=%d body=%v", code, created)
	}
	editID := created["id"].(string)

	code, tracks := do(t, "GET", base+"/edits/"+editID+"/tracks", "")
	if code != 200 {
		t.Fatalf("tracks list: code=%d body=%v", code, tracks)
	}
	if n := len(tracks["tracks"].([]any)); n != 2 {
		t.Fatalf("want 2 seeded tracks, got %d", n)
	}

	code, _ = do(t, "PUT", base+"/edits/"+editID+"/listings/en-US", `{"title":"New Title","language":"en-US"}`)
	if code != 200 {
		t.Fatalf("listing update: code=%d", code)
	}
	code, listing := do(t, "GET", base+"/edits/"+editID+"/listings/en-US", "")
	if code != 200 || listing["title"] != "New Title" {
		t.Fatalf("listing readback: code=%d body=%v", code, listing)
	}

	code, _ = do(t, "POST", base+"/edits/"+editID+":commit", "")
	if code != 200 {
		t.Fatalf("commit: code=%d", code)
	}

	// A committed edit is gone.
	code, _ = do(t, "GET", base+"/edits/"+editID, "")
	if code != 400 {
		t.Fatalf("get after commit must fail with 400, got %d", code)
	}
}

func TestSandbox_InvalidEditIs400(t *testing.T) {
	srv := startServer(t)
	code, _ := do(t, "GET", srv.URL+"/androidpublisher/v3/applications/"+Package+"/edits/bogus/tracks", "")
	if code != 400 {
		t.Fatalf("invalid edit must 400, got %d", code)
	}
}
