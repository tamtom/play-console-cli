// Package testutil provides httptest-based API mocking utilities for testing
// commands that interact with the Google Play Android Publisher API.
package testutil

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

// RequestEntry records a single HTTP request received by the mock server.
type RequestEntry struct {
	Method string
	Path   string
	Body   string
}

// MockAPI wraps an httptest.Server with request logging and convenience helpers.
type MockAPI struct {
	Server *httptest.Server

	mu  sync.Mutex
	log []RequestEntry
}

// NewMockAPI creates a new test server routing requests through the given
// handler map. Keys are "METHOD /path" (e.g. "GET /edits"). Unmatched
// requests receive a 404 response. Every request is recorded in the log.
// The server is automatically closed via t.Cleanup.
func NewMockAPI(t *testing.T, handlers map[string]http.HandlerFunc) *MockAPI {
	t.Helper()

	m := &MockAPI{}

	mux := http.NewServeMux()

	// Register a catch-all handler that does routing + logging.
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		defer r.Body.Close()

		m.mu.Lock()
		m.log = append(m.log, RequestEntry{
			Method: r.Method,
			Path:   r.URL.Path,
			Body:   string(body),
		})
		m.mu.Unlock()

		key := r.Method + " " + r.URL.Path
		if h, ok := handlers[key]; ok {
			h(w, r)
			return
		}

		// Fallback: 404
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]interface{}{
				"code":    404,
				"message": "no mock handler for " + key,
			},
		})
	})

	m.Server = httptest.NewServer(mux)
	t.Cleanup(m.Server.Close)

	return m
}

// BaseURL returns the test server's URL (e.g. "http://127.0.0.1:PORT").
func (m *MockAPI) BaseURL() string {
	return m.Server.URL
}

// RequestLog returns a copy of all recorded requests.
func (m *MockAPI) RequestLog() []RequestEntry {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]RequestEntry, len(m.log))
	copy(out, m.log)
	return out
}

// HandleJSON returns an http.HandlerFunc that responds with the given status
// code and JSON-encodes body into the response.
func HandleJSON(statusCode int, body interface{}) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		_ = json.NewEncoder(w).Encode(body)
	}
}

// HandleError returns an http.HandlerFunc that responds with a Google-style
// JSON error body.
func HandleError(statusCode int, message string) http.HandlerFunc {
	return HandleJSON(statusCode, map[string]interface{}{
		"error": map[string]interface{}{
			"code":    statusCode,
			"message": message,
		},
	})
}
