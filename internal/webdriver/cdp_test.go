package webdriver

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"golang.org/x/net/websocket"
)

// fakeChrome serves the /json/list target listing plus a DevTools websocket
// that answers Runtime.evaluate from a scripted table.
type fakeChrome struct {
	server *httptest.Server
	// mu guards the scripted tables: the websocket handler runs on its own
	// goroutine while tests mutate replies mid-flight.
	mu           sync.Mutex
	replies      map[string]any // expression -> value returned
	defaultReply any
	fail         map[string]string
	dom          map[string]json.RawMessage // DOM.* method -> result payload
	seen         []string
}

// reply records/returns the scripted answer for an expression.
func (f *fakeChrome) reply(expr string) (any, string, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.seen = append(f.seen, expr)
	if msg, bad := f.fail[expr]; bad {
		return nil, msg, true
	}
	value, ok := f.replies[expr]
	if !ok {
		value = f.defaultReply
	}
	return value, "", false
}

// setReply scripts an answer.
func (f *fakeChrome) setReply(expr string, value any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.replies[expr] = value
}

// setDOMReply scripts a DOM-domain method answer.
func (f *fakeChrome) setDOMReply(t *testing.T, method string, result any) {
	t.Helper()
	raw, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.dom[method] = raw
}

func newFakeChrome(t *testing.T) *fakeChrome {
	t.Helper()
	f := &fakeChrome{replies: map[string]any{}, fail: map[string]string{}, dom: map[string]json.RawMessage{}}
	mux := http.NewServeMux()
	mux.HandleFunc("/json/list", func(w http.ResponseWriter, r *http.Request) {
		ws := "ws" + strings.TrimPrefix(f.server.URL, "http") + "/devtools/page/1"
		_ = json.NewEncoder(w).Encode([]target{
			{Type: "other", URL: "chrome://x"},
			{Type: "page", URL: "about:blank", WebSocketDebuggerURL: ws},
		})
	})
	mux.Handle("/devtools/page/1", websocket.Handler(func(c *websocket.Conn) {
		for {
			var req cdpRequest
			if err := websocket.JSON.Receive(c, &req); err != nil {
				return
			}
			// Interleave an unsolicited event to prove the client skips it.
			_ = websocket.JSON.Send(c, map[string]any{"method": "Page.loadEventFired"})

			if strings.HasPrefix(req.Method, "DOM.") {
				f.mu.Lock()
				raw := f.dom[req.Method]
				f.mu.Unlock()
				_ = websocket.JSON.Send(c, map[string]any{"id": req.ID, "result": raw})
				continue
			}

			expr, _ := req.Params["expression"].(string)
			value, msg, bad := f.reply(expr)
			if bad {
				_ = websocket.JSON.Send(c, map[string]any{
					"id": req.ID,
					"result": map[string]any{
						"exceptionDetails": map[string]any{"text": msg},
					},
				})
				continue
			}
			_ = websocket.JSON.Send(c, map[string]any{
				"id":     req.ID,
				"result": map[string]any{"result": map[string]any{"value": value}},
			})
		}
	}))
	f.server = httptest.NewServer(mux)
	t.Cleanup(f.server.Close)
	return f
}

// portDir writes a DevToolsActivePort file pointing at the fake server.
func portDir(t *testing.T, f *fakeChrome) string {
	t.Helper()
	dir := t.TempDir()
	port := strings.TrimPrefix(f.server.URL, "http://127.0.0.1:")
	if err := os.WriteFile(filepath.Join(dir, devToolsPortFile), []byte(port+"\n/devtools/browser/x"), 0o600); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestConnectAndEval(t *testing.T) {
	f := newFakeChrome(t)
	f.setReply("1+1", 2)
	dir := portDir(t, f)

	b, err := Connect(context.Background(), dir, 2*time.Second)
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer func() { _ = b.Close() }() //nolint:errcheck // read-path close

	var got int
	if err := b.Eval(context.Background(), "1+1", &got); err != nil {
		t.Fatalf("Eval: %v", err)
	}
	if got != 2 {
		t.Errorf("got %d, want 2", got)
	}
}

func TestEval_SurfacesPageException(t *testing.T) {
	f := newFakeChrome(t)
	f.fail["boom()"] = "ReferenceError: boom is not defined"
	dir := portDir(t, f)

	b, err := Connect(context.Background(), dir, 2*time.Second)
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer func() { _ = b.Close() }() //nolint:errcheck // read-path close

	err = b.Eval(context.Background(), "boom()", nil)
	if err == nil || !strings.Contains(err.Error(), "ReferenceError") {
		t.Errorf("err = %v, want the page exception surfaced", err)
	}
}

func TestEvalUntil_WaitsForTrue(t *testing.T) {
	f := newFakeChrome(t)
	f.setReply("ready", false)
	dir := portDir(t, f)

	b, err := Connect(context.Background(), dir, 2*time.Second)
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer func() { _ = b.Close() }() //nolint:errcheck // read-path close

	// Flip the answer once polling has started.
	go func() {
		time.Sleep(300 * time.Millisecond)
		f.setReply("ready", true)
	}()
	if err := b.EvalUntil(context.Background(), "ready", 5*time.Second); err != nil {
		t.Fatalf("EvalUntil: %v", err)
	}
}

func TestEvalUntil_TimesOut(t *testing.T) {
	f := newFakeChrome(t)
	f.setReply("never", false)
	dir := portDir(t, f)

	b, err := Connect(context.Background(), dir, 2*time.Second)
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer func() { _ = b.Close() }() //nolint:errcheck // read-path close

	err = b.EvalUntil(context.Background(), "never", 400*time.Millisecond)
	if err == nil || !strings.Contains(err.Error(), "timed out") {
		t.Errorf("err = %v, want timeout", err)
	}
}

func TestConnect_NoProfile(t *testing.T) {
	_, err := Connect(context.Background(), t.TempDir(), 300*time.Millisecond)
	if err == nil {
		t.Fatal("expected an error when no Chrome is running for the profile")
	}
}

func TestLaunchArgs_IncludesDebugPort(t *testing.T) {
	args := LaunchArgs("/tmp/profile")
	joined := fmt.Sprint(args)
	for _, want := range []string{DebugPortArg, AllowOriginArg, "--user-data-dir=/tmp/profile"} {
		if !strings.Contains(joined, want) {
			t.Errorf("args %v missing %q", args, want)
		}
	}
}

func TestInteractiveArgs_ExcludeDebugPort(t *testing.T) {
	// An interactive window holds a live Google session; it must not expose a
	// DevTools endpoint for other local processes to drive it with.
	args := InteractiveArgs("/tmp/profile")
	joined := fmt.Sprint(args)
	for _, unwanted := range []string{"--remote-debugging-port", "--remote-allow-origins"} {
		if strings.Contains(joined, unwanted) {
			t.Errorf("interactive args %v must not contain %q", args, unwanted)
		}
	}
	for _, want := range []string{"--user-data-dir=/tmp/profile", "--no-first-run"} {
		if !strings.Contains(joined, want) {
			t.Errorf("interactive args %v missing %q", args, want)
		}
	}
}

func TestFillAppForm_RejectsNonDefaultLanguage(t *testing.T) {
	err := FillAppForm(context.Background(), nil, "dev1", AppForm{Language: "de-DE"})
	if err == nil || !strings.Contains(err.Error(), "en-US") {
		t.Errorf("err = %v, want a clear unsupported-language error", err)
	}
}

func TestCreateAppURL(t *testing.T) {
	got := createAppURL(publishingTestDeveloper)
	if !strings.HasSuffix(got, "/developers/1234567890/create-new-app") {
		t.Errorf("url = %s", got)
	}
}

func TestJSString_EscapesInjection(t *testing.T) {
	// App names are user input and land inside a script; they must be escaped.
	got := jsString(`a"); alert(1); //`)
	if strings.Contains(got, `a");`) && !strings.Contains(got, `\"`) {
		t.Errorf("jsString did not escape: %s", got)
	}
}

func TestJSString_EscapesSingleQuotes(t *testing.T) {
	// JSON escaping leaves single quotes bare, but values can land inside
	// single-quoted JS contexts (e.g. attribute selectors), where a bare '
	// would terminate the literal early. JavaScript accepts \' inside any
	// string literal, so it must be escaped explicitly.
	for _, tc := range []struct{ in, want string }{
		{"it's", `"it\'s"`},
		{`a\'b`, `"a\\\'b"`},
		{"plain", `"plain"`},
	} {
		if got := jsString(tc.in); got != tc.want {
			t.Errorf("jsString(%q) = %s, want %s", tc.in, got, tc.want)
		}
	}
}

func TestAppIDFromURL(t *testing.T) {
	m := appIDFromURL.FindStringSubmatch("/console/u/2/developers/690/app/9876543210/app-dashboard")
	if m == nil || m[1] != "9876543210" {
		t.Errorf("match = %v", m)
	}
}

func TestSubmitAppForm_ErrorsWhenButtonDisabled(t *testing.T) {
	f := newFakeChrome(t)
	dir := portDir(t, f)
	b, err := Connect(context.Background(), dir, 2*time.Second)
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer func() { _ = b.Close() }() //nolint:errcheck // read-path close
	// The fake returns nil (false) for the click expression.
	_, err = SubmitAppForm(context.Background(), b, 300*time.Millisecond)
	if err == nil || !strings.Contains(err.Error(), "Create app") {
		t.Errorf("err = %v, want disabled-button error", err)
	}
}
