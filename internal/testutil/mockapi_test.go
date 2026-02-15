package testutil_test

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/tamtom/play-console-cli/internal/testutil"
)

func TestNewMockAPI_BaseURL(t *testing.T) {
	mock := testutil.NewMockAPI(t, nil)
	url := mock.BaseURL()
	if url == "" {
		t.Fatal("BaseURL() returned empty string")
	}
	if url[:4] != "http" {
		t.Fatalf("BaseURL() should start with http, got %q", url)
	}
}

func TestMockAPI_HandlerRouting(t *testing.T) {
	handlers := map[string]http.HandlerFunc{
		"GET /hello": testutil.HandleJSON(http.StatusOK, map[string]string{"msg": "hi"}),
	}
	mock := testutil.NewMockAPI(t, handlers)

	resp, err := http.Get(mock.BaseURL() + "/hello")
	if err != nil {
		t.Fatalf("GET /hello failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var body map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if body["msg"] != "hi" {
		t.Fatalf("expected msg=hi, got %q", body["msg"])
	}
}

func TestMockAPI_UnmatchedRoute_Returns404(t *testing.T) {
	mock := testutil.NewMockAPI(t, nil)

	resp, err := http.Get(mock.BaseURL() + "/nonexistent")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

func TestMockAPI_RequestLog(t *testing.T) {
	handlers := map[string]http.HandlerFunc{
		"POST /submit": testutil.HandleJSON(http.StatusCreated, map[string]string{"ok": "true"}),
	}
	mock := testutil.NewMockAPI(t, handlers)

	// Make a request with a body.
	resp, err := http.Post(mock.BaseURL()+"/submit", "application/json", nil)
	if err != nil {
		t.Fatalf("POST /submit failed: %v", err)
	}
	resp.Body.Close()

	log := mock.RequestLog()
	if len(log) != 1 {
		t.Fatalf("expected 1 log entry, got %d", len(log))
	}
	if log[0].Method != "POST" {
		t.Fatalf("expected POST, got %s", log[0].Method)
	}
	if log[0].Path != "/submit" {
		t.Fatalf("expected /submit, got %s", log[0].Path)
	}
}

func TestMockAPI_MultipleRequests(t *testing.T) {
	handlers := map[string]http.HandlerFunc{
		"GET /a": testutil.HandleJSON(http.StatusOK, "a"),
		"GET /b": testutil.HandleJSON(http.StatusOK, "b"),
	}
	mock := testutil.NewMockAPI(t, handlers)

	for _, path := range []string{"/a", "/b", "/a"} {
		resp, err := http.Get(mock.BaseURL() + path)
		if err != nil {
			t.Fatalf("GET %s failed: %v", path, err)
		}
		resp.Body.Close()
	}

	log := mock.RequestLog()
	if len(log) != 3 {
		t.Fatalf("expected 3 log entries, got %d", len(log))
	}
	if log[0].Path != "/a" || log[1].Path != "/b" || log[2].Path != "/a" {
		t.Fatalf("unexpected log paths: %v", log)
	}
}

func TestHandleError(t *testing.T) {
	handlers := map[string]http.HandlerFunc{
		"GET /fail": testutil.HandleError(http.StatusForbidden, "access denied"),
	}
	mock := testutil.NewMockAPI(t, handlers)

	resp, err := http.Get(mock.BaseURL() + "/fail")
	if err != nil {
		t.Fatalf("GET /fail failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", resp.StatusCode)
	}

	data, _ := io.ReadAll(resp.Body)
	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to parse error JSON: %v", err)
	}
	errObj, ok := parsed["error"].(map[string]interface{})
	if !ok {
		t.Fatal("expected error object in response")
	}
	if errObj["message"] != "access denied" {
		t.Fatalf("expected message 'access denied', got %q", errObj["message"])
	}
}

func TestFixtures_NotNil(t *testing.T) {
	fixtures := []struct {
		name string
		fn   func() map[string]interface{}
	}{
		{"EditFixture", testutil.EditFixture},
		{"TrackFixture", testutil.TrackFixture},
		{"TracksListFixture", testutil.TracksListFixture},
		{"ReviewFixture", testutil.ReviewFixture},
		{"ReviewsListFixture", testutil.ReviewsListFixture},
		{"ListingFixture", testutil.ListingFixture},
		{"ListingsListFixture", testutil.ListingsListFixture},
		{"ErrorFixture404", testutil.ErrorFixture404},
		{"ErrorFixture403", testutil.ErrorFixture403},
		{"ErrorFixture409", testutil.ErrorFixture409},
	}
	for _, f := range fixtures {
		t.Run(f.name, func(t *testing.T) {
			result := f.fn()
			if result == nil {
				t.Fatalf("%s returned nil", f.name)
			}
			// Ensure it marshals to valid JSON.
			data, err := json.Marshal(result)
			if err != nil {
				t.Fatalf("%s failed to marshal: %v", f.name, err)
			}
			if len(data) < 2 {
				t.Fatalf("%s produced empty JSON", f.name)
			}
		})
	}
}

func TestFixtures_WithHandleJSON(t *testing.T) {
	// Verify that fixtures integrate cleanly with HandleJSON in a MockAPI.
	handlers := map[string]http.HandlerFunc{
		"GET /edit":    testutil.HandleJSON(http.StatusOK, testutil.EditFixture()),
		"GET /tracks":  testutil.HandleJSON(http.StatusOK, testutil.TracksListFixture()),
		"GET /reviews": testutil.HandleJSON(http.StatusOK, testutil.ReviewsListFixture()),
		"GET /listing": testutil.HandleJSON(http.StatusOK, testutil.ListingFixture()),
		"GET /err404":  testutil.HandleJSON(http.StatusNotFound, testutil.ErrorFixture404()),
		"GET /err403":  testutil.HandleJSON(http.StatusForbidden, testutil.ErrorFixture403()),
		"GET /err409":  testutil.HandleJSON(http.StatusConflict, testutil.ErrorFixture409()),
	}
	mock := testutil.NewMockAPI(t, handlers)

	paths := []struct {
		path   string
		status int
	}{
		{"/edit", 200},
		{"/tracks", 200},
		{"/reviews", 200},
		{"/listing", 200},
		{"/err404", 404},
		{"/err403", 403},
		{"/err409", 409},
	}
	for _, p := range paths {
		resp, err := http.Get(mock.BaseURL() + p.path)
		if err != nil {
			t.Fatalf("GET %s failed: %v", p.path, err)
		}
		if resp.StatusCode != p.status {
			t.Fatalf("GET %s: expected %d, got %d", p.path, p.status, resp.StatusCode)
		}
		resp.Body.Close()
	}

	log := mock.RequestLog()
	if len(log) != len(paths) {
		t.Fatalf("expected %d log entries, got %d", len(paths), len(log))
	}
}
