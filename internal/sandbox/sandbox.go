// Package sandbox is a local, in-memory fake of the Google Play Android
// Publisher API for hermetic black-box tests. Every incoming request is
// matched against the embedded official discovery catalog: a request whose
// method and path do not exist in the catalog fails with 404, and a known
// method without a sandbox implementation fails with 501. This keeps the
// fake honest — it can never accept a call the real API would reject as
// nonexistent.
//
// Point the compiled CLI at a sandbox server with GPLAY_API_BASE_URL and a
// service-account file whose token_uri targets the server's /token endpoint
// (see testutil.SandboxServiceAccount).
package sandbox

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"sync"

	"github.com/tamtom/play-console-cli/internal/apischema"
)

// Package is the seeded application every sandbox server knows about.
const Package = "com.sandbox.app"

type route struct {
	id      string
	method  string
	pattern *regexp.Regexp
	params  []string
}

// Server is one in-memory sandbox instance. Create it with New and serve it
// with Start; state is per-instance, so parallel tests stay isolated.
type Server struct {
	mu       sync.Mutex
	routes   []route
	handlers map[string]func(s *Server, w http.ResponseWriter, r *http.Request, params map[string]string)
	edits    map[string]*editState
	nextEdit int
}

type editState struct {
	pkg      string
	tracks   map[string]map[string]any
	listings map[string]map[string]any
	bundles  []map[string]any
}

var paramPattern = regexp.MustCompile(`\{([a-zA-Z]+)\}`)

// New builds a sandbox with routes compiled from the embedded catalog.
func New() (*Server, error) {
	index, err := apischema.Load()
	if err != nil {
		return nil, err
	}
	s := &Server{
		edits:    map[string]*editState{},
		handlers: endpointHandlers(),
	}
	for _, e := range index.Endpoints {
		if e.API != "androidpublisher" {
			continue
		}
		s.routes = append(s.routes, compileRoute(e.ID, e.HTTPMethod, "/"+e.Path))
		if e.MediaUpload != nil && e.MediaUpload.SimplePath != "" {
			s.routes = append(s.routes, compileRoute(e.ID, e.HTTPMethod, e.MediaUpload.SimplePath))
		}
	}
	return s, nil
}

// Start returns a running test server. The caller owns Close; httptest
// binds to 127.0.0.1 only.
func Start() (*httptest.Server, *Server, error) {
	s, err := New()
	if err != nil {
		return nil, nil, err
	}
	return httptest.NewServer(s), s, nil
}

func compileRoute(id, method, path string) route {
	var params []string
	escaped := regexp.QuoteMeta(path)
	// QuoteMeta escapes { and } as \{ and \}.
	rx := regexp.MustCompile(`\\\{([a-zA-Z]+)\\\}`).ReplaceAllStringFunc(escaped, func(m string) string {
		params = append(params, paramPattern.FindStringSubmatch(strings.ReplaceAll(m, `\`, ""))[1])
		return `([^/:]+)`
	})
	return route{
		id:      id,
		method:  method,
		pattern: regexp.MustCompile("^" + rx + "$"),
		params:  params,
	}
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/token" {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"sandbox-token","token_type":"Bearer","expires_in":3600}`))
		return
	}

	for _, rt := range s.routes {
		if rt.method != r.Method {
			continue
		}
		m := rt.pattern.FindStringSubmatch(r.URL.Path)
		if m == nil {
			continue
		}
		params := map[string]string{}
		for i, name := range rt.params {
			params[name] = m[i+1]
		}
		handler, ok := s.handlers[rt.id]
		if !ok {
			writeError(w, http.StatusNotImplemented, "NOT_IMPLEMENTED",
				fmt.Sprintf("sandbox: method %s is in the official catalog but has no sandbox implementation", rt.id))
			return
		}
		s.mu.Lock()
		defer s.mu.Unlock()
		handler(s, w, r, params)
		return
	}

	writeError(w, http.StatusNotFound, "NOT_FOUND",
		fmt.Sprintf("sandbox: no official androidpublisher method matches %s %s", r.Method, r.URL.Path))
}

// writeError emits the googleapi error JSON shape so the generated client
// surfaces code and message exactly like the real API.
func writeError(w http.ResponseWriter, code int, status, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]any{
			"code":    code,
			"message": message,
			"status":  status,
			"errors": []map[string]any{
				{"message": message, "reason": strings.ToLower(status)},
			},
		},
	})
}

func writeJSON(w http.ResponseWriter, body any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(body)
}
