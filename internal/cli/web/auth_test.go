package web

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/tamtom/play-console-cli/internal/testutil"
	"github.com/tamtom/play-console-cli/internal/webclient"
	"github.com/tamtom/play-console-cli/internal/websession"
)

const (
	authDeveloperID = "1234567890123456789"
	authConsolePath = "/console/u/2/developers/" + authDeveloperID + "/app-list"
	authConsoleHTML = `<html><script>window.serializedInitialChunks['startupData'] = ` +
		`"\x7b\x221\x22:\x7b\x228\x22:\x22runtime-test-key\x22\x7d\x7d";</script>` +
		`<script src="https://www.gstatic.com/acx/play/console/brt/play_console_ui_20260727/main/main.dart.js"></script></html>`
	authWizHTML = `<html><script>window.WIZ_global_data = {"SNlM0e":"xsrf","FdrFJe":"-1","cfb2h":"bl"};</script><a href="/console/1234567890123456789/app-list">apps</a></html>`
)

// useTempSessionDir points web session storage at a temp dir.
func useTempSessionDir(t *testing.T) {
	t.Helper()
	t.Setenv("GPLAY_WEB_SESSION_DIR", t.TempDir())
}

// mockWebClient redirects the web client seam at the given mock server.
func mockWebClient(t *testing.T, mock *testutil.MockAPI) {
	t.Helper()
	orig := newWebClient
	newWebClient = func(sess *websession.Session) webRPCClient {
		return webclient.NewWithClient(sess, nil, mock.BaseURL())
	}
	t.Cleanup(func() { newWebClient = orig })
}

// consoleMock returns a mock that redirects to the current Play Console URL.
func consoleMock(t *testing.T) *testutil.MockAPI {
	t.Helper()
	return testutil.NewMockAPI(t, map[string]http.HandlerFunc{
		"GET /console": func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, authConsolePath, http.StatusFound)
		},
		"GET " + authConsolePath: func(w http.ResponseWriter, r *http.Request) {
			_, _ = io.WriteString(w, authConsoleHTML)
		},
	})
}

// captureWebStdout runs fn with os.Stdout redirected and returns the output.
func captureWebStdout(fn func() error) (string, error) {
	origStdout := os.Stdout
	rOut, wOut, err := os.Pipe()
	if err != nil {
		return "", err
	}
	os.Stdout = wOut

	var buf bytes.Buffer
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_, _ = io.Copy(&buf, rOut)
	}()

	runErr := fn()

	_ = wOut.Close()
	os.Stdout = origStdout
	wg.Wait()
	_ = rOut.Close()

	return buf.String(), runErr
}

func TestAuthLogin_RequiresEmail(t *testing.T) {
	useTempSessionDir(t)
	cmd := AuthLoginCommand()
	if err := cmd.FlagSet.Parse([]string{"--cookies", "SAPISID=x"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil || !strings.Contains(err.Error(), "--email") {
		t.Errorf("err = %v, want --email required", err)
	}
}

func TestAuthLogin_AutoImportsChromeCookies(t *testing.T) {
	useTempSessionDir(t)
	mockWebClient(t, consoleMock(t))
	orig := importChromeCookies
	importChromeCookies = func(context.Context) ([]map[string][]websession.Cookie, error) {
		return []map[string][]websession.Cookie{
			{
				playOrigin: {
					{Name: "SID", Value: "sid"},
					{Name: "SAPISID", Value: "sapisid"},
				},
			},
		}, nil
	}
	t.Cleanup(func() { importChromeCookies = orig })

	cmd := AuthLoginCommand()
	if err := cmd.FlagSet.Parse([]string{"--email", "me@example.com"}); err != nil {
		t.Fatal(err)
	}
	stdout, err := captureWebStdout(func() error {
		return cmd.Exec(context.Background(), nil)
	})
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if !strings.Contains(stdout, `"email":"me@example.com"`) || !strings.Contains(stdout, `"cookie_count":2`) {
		t.Fatalf("output = %s", stdout)
	}
}

func TestAuthLogin_AutoImportError(t *testing.T) {
	useTempSessionDir(t)
	orig := importChromeCookies
	importChromeCookies = func(context.Context) ([]map[string][]websession.Cookie, error) {
		return nil, errors.New("Chrome cookie database not found")
	}
	t.Cleanup(func() { importChromeCookies = orig })

	cmd := AuthLoginCommand()
	if err := cmd.FlagSet.Parse([]string{"--email", "me@example.com"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil || !strings.Contains(err.Error(), "Chrome cookie database not found") ||
		!strings.Contains(err.Error(), "--cookies") {
		t.Fatalf("err = %v, want import error with manual fallback", err)
	}
}

func TestAuthLogin_TriesChromeProfiles(t *testing.T) {
	useTempSessionDir(t)
	mock := testutil.NewMockAPI(t, map[string]http.HandlerFunc{
		"GET /console": func(w http.ResponseWriter, r *http.Request) {
			cookie, _ := r.Cookie("profile")
			if cookie == nil || cookie.Value != "matching-profile" {
				w.WriteHeader(http.StatusForbidden)
				return
			}
			http.Redirect(w, r, authConsolePath, http.StatusFound)
		},
		"GET " + authConsolePath: func(w http.ResponseWriter, r *http.Request) {
			_, _ = io.WriteString(w, authConsoleHTML)
		},
	})
	mockWebClient(t, mock)

	orig := importChromeCookies
	importChromeCookies = func(context.Context) ([]map[string][]websession.Cookie, error) {
		return []map[string][]websession.Cookie{
			{
				playOrigin:     {{Name: "SAPISID", Value: "wrong-profile"}},
				mock.BaseURL(): {{Name: "profile", Value: "wrong-profile"}},
			},
			{
				playOrigin:     {{Name: "SAPISID", Value: "matching-profile"}},
				mock.BaseURL(): {{Name: "profile", Value: "matching-profile"}},
			},
		}, nil
	}
	t.Cleanup(func() { importChromeCookies = orig })

	cmd := AuthLoginCommand()
	if err := cmd.FlagSet.Parse([]string{"--email", "me@example.com"}); err != nil {
		t.Fatal(err)
	}
	if _, err := captureWebStdout(func() error {
		return cmd.Exec(context.Background(), nil)
	}); err != nil {
		t.Fatalf("login: %v", err)
	}

	sess, err := websession.Load("me@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if got := sess.Cookies[playOrigin][0].Value; got != "matching-profile" {
		t.Fatalf("saved SAPISID = %q, want matching-profile", got)
	}
}

func TestAuthLogin_RejectsBothCookieSources(t *testing.T) {
	useTempSessionDir(t)
	cmd := AuthLoginCommand()
	if err := cmd.FlagSet.Parse([]string{"--email", "me@example.com", "--cookies", "SAPISID=x", "--cookies-file", "f"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil || !strings.Contains(err.Error(), "mutually exclusive") {
		t.Errorf("err = %v, want mutually exclusive error", err)
	}
}

func TestAuthLogin_RequiresSAPISID(t *testing.T) {
	useTempSessionDir(t)
	mockWebClient(t, consoleMock(t))
	cmd := AuthLoginCommand()
	if err := cmd.FlagSet.Parse([]string{"--email", "me@example.com", "--cookies", "SID=only-sid"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil || !strings.Contains(err.Error(), "SAPISID") {
		t.Errorf("err = %v, want SAPISID error", err)
	}
}

func TestAuthLogin_ReportsMalformedCookieExport(t *testing.T) {
	useTempSessionDir(t)
	cmd := AuthLoginCommand()
	if err := cmd.FlagSet.Parse([]string{"--email", "me@example.com", "--cookies", "{"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil || !strings.Contains(err.Error(), "parsing cookie export JSON") {
		t.Fatalf("err = %v, want cookie parse error", err)
	}
}

func TestAuthLogin_HeaderSuccess(t *testing.T) {
	useTempSessionDir(t)
	mockWebClient(t, consoleMock(t))
	orig := importChromeCookies
	importChromeCookies = func(context.Context) ([]map[string][]websession.Cookie, error) {
		t.Fatal("manual cookies must not invoke Chrome import")
		return nil, nil
	}
	t.Cleanup(func() { importChromeCookies = orig })
	cmd := AuthLoginCommand()
	if err := cmd.FlagSet.Parse([]string{"--email", "me@example.com", "--cookies", "SID=sid; SAPISID=sapisid"}); err != nil {
		t.Fatal(err)
	}
	stdout, err := captureWebStdout(func() error {
		return cmd.Exec(context.Background(), nil)
	})
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	for _, want := range []string{`"email":"me@example.com"`, `"developer_id":"` + authDeveloperID + `"`, `"cookie_count":2`, `"validated":true`} {
		if !strings.Contains(stdout, want) {
			t.Errorf("output missing %s: %s", want, stdout)
		}
	}
	// Session must be persisted.
	sess, err := websession.Load("me@example.com")
	if err != nil {
		t.Fatalf("Load after login: %v", err)
	}
	if sess.UserEmail != "me@example.com" {
		t.Errorf("UserEmail = %q", sess.UserEmail)
	}
}

func TestAuthLogin_JSONExportSuccess(t *testing.T) {
	useTempSessionDir(t)
	mockWebClient(t, consoleMock(t))
	export := `[{"name":"SAPISID","value":"sapisid","domain":".google.com","expirationDate":1800000000}]`
	cmd := AuthLoginCommand()
	if err := cmd.FlagSet.Parse([]string{"--email", "json@example.com", "--cookies", export}); err != nil {
		t.Fatal(err)
	}
	stdout, err := captureWebStdout(func() error {
		return cmd.Exec(context.Background(), nil)
	})
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if !strings.Contains(stdout, `"email":"json@example.com"`) {
		t.Errorf("output: %s", stdout)
	}
	sess, err := websession.Load("json@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if got := sess.Cookies[playOrigin]; len(got) != 1 || got[0].Name != "SAPISID" {
		t.Fatalf("Play Console cookies = %#v", got)
	}
}

func TestAuthLogin_CookiesFileStdin(t *testing.T) {
	useTempSessionDir(t)
	mockWebClient(t, consoleMock(t))
	orig := readStdin
	readStdin = func() ([]byte, error) { return []byte("SID=sid; SAPISID=from-stdin"), nil }
	t.Cleanup(func() { readStdin = orig })

	cmd := AuthLoginCommand()
	if err := cmd.FlagSet.Parse([]string{"--email", "stdin@example.com", "--cookies-file", "-"}); err != nil {
		t.Fatal(err)
	}
	stdout, err := captureWebStdout(func() error {
		return cmd.Exec(context.Background(), nil)
	})
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if !strings.Contains(stdout, `"email":"stdin@example.com"`) {
		t.Errorf("output: %s", stdout)
	}
}

func TestAuthLogin_InvalidSessionNotSaved(t *testing.T) {
	useTempSessionDir(t)
	mock := testutil.NewMockAPI(t, map[string]http.HandlerFunc{
		"GET /console": func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusForbidden)
		},
	})
	mockWebClient(t, mock)

	cmd := AuthLoginCommand()
	if err := cmd.FlagSet.Parse([]string{"--email", "bad@example.com", "--cookies", "SAPISID=stale"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil || !strings.Contains(err.Error(), "gplay web auth login") {
		t.Errorf("err = %v, want re-login hint", err)
	}
	emails, _ := websession.List()
	if len(emails) != 0 {
		t.Errorf("failed login must not save a session, got %v", emails)
	}
}

func TestAuthStatus_Empty(t *testing.T) {
	useTempSessionDir(t)
	cmd := AuthStatusCommand()
	if err := cmd.FlagSet.Parse(nil); err != nil {
		t.Fatal(err)
	}
	stdout, err := captureWebStdout(func() error {
		return cmd.Exec(context.Background(), nil)
	})
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if !strings.Contains(stdout, "[]") {
		t.Errorf("output = %q, want empty list", stdout)
	}
}

func TestAuthLogin_OutputsSessionStore(t *testing.T) {
	useTempSessionDir(t)
	mockWebClient(t, consoleMock(t))
	cmd := AuthLoginCommand()
	if err := cmd.FlagSet.Parse([]string{"--email", "me@example.com", "--cookies", "SID=sid; SAPISID=sapisid"}); err != nil {
		t.Fatal(err)
	}
	stdout, err := captureWebStdout(func() error {
		return cmd.Exec(context.Background(), nil)
	})
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	// With GPLAY_WEB_SESSION_DIR set, the file backend reports the path.
	wantPath := filepath.Join(os.Getenv("GPLAY_WEB_SESSION_DIR"),
		"session-"+websession.AccountKey("me@example.com")+".json")
	if !strings.Contains(stdout, `"session_store":"`+wantPath+`"`) {
		t.Errorf("output missing session_store %s: %s", wantPath, stdout)
	}
	if strings.Contains(stdout, "session_file") {
		t.Errorf("output should not contain session_file: %s", stdout)
	}
}

func TestAuthStatus_OutputsSessionStore(t *testing.T) {
	useTempSessionDir(t)
	if err := websession.Save(&websession.Session{
		UserEmail: "me@example.com",
		Cookies:   map[string][]websession.Cookie{playOrigin: {{Name: "SAPISID", Value: "v"}}},
	}); err != nil {
		t.Fatal(err)
	}
	cmd := AuthStatusCommand()
	if err := cmd.FlagSet.Parse(nil); err != nil {
		t.Fatal(err)
	}
	stdout, err := captureWebStdout(func() error {
		return cmd.Exec(context.Background(), nil)
	})
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if !strings.Contains(stdout, `"session_store":"`) {
		t.Errorf("output missing session_store: %s", stdout)
	}
	if strings.Contains(stdout, "session_file") {
		t.Errorf("output should not contain session_file: %s", stdout)
	}
}

func TestAuthStatus_Check(t *testing.T) {
	useTempSessionDir(t)
	mockWebClient(t, consoleMock(t))
	if err := websession.Save(&websession.Session{
		UserEmail: "me@example.com",
		Cookies:   map[string][]websession.Cookie{playOrigin: {{Name: "SAPISID", Value: "v"}}},
	}); err != nil {
		t.Fatal(err)
	}
	cmd := AuthStatusCommand()
	if err := cmd.FlagSet.Parse([]string{"--check"}); err != nil {
		t.Fatal(err)
	}
	stdout, err := captureWebStdout(func() error {
		return cmd.Exec(context.Background(), nil)
	})
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if !strings.Contains(stdout, `"email":"me@example.com"`) || !strings.Contains(stdout, `"status":"valid"`) {
		t.Errorf("output: %s", stdout)
	}
}

func TestAuthLogout_RequiresConfirm(t *testing.T) {
	useTempSessionDir(t)
	cmd := AuthLogoutCommand()
	if err := cmd.FlagSet.Parse([]string{"--account", "me@example.com"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil || !strings.Contains(err.Error(), "--confirm") {
		t.Errorf("err = %v, want --confirm required", err)
	}
}

func TestAuthLogout_DeletesAccount(t *testing.T) {
	useTempSessionDir(t)
	if err := websession.Save(&websession.Session{
		UserEmail: "me@example.com",
		Cookies:   map[string][]websession.Cookie{playOrigin: {{Name: "SAPISID", Value: "v"}}},
	}); err != nil {
		t.Fatal(err)
	}
	cmd := AuthLogoutCommand()
	if err := cmd.FlagSet.Parse([]string{"--account", "me@example.com", "--confirm"}); err != nil {
		t.Fatal(err)
	}
	stdout, err := captureWebStdout(func() error {
		return cmd.Exec(context.Background(), nil)
	})
	if err != nil {
		t.Fatalf("logout: %v", err)
	}
	if !strings.Contains(stdout, `"deleted":"me@example.com"`) {
		t.Errorf("output: %s", stdout)
	}
	emails, _ := websession.List()
	if len(emails) != 0 {
		t.Errorf("session should be deleted, got %v", emails)
	}
}

func TestAuthLogout_All(t *testing.T) {
	useTempSessionDir(t)
	for _, email := range []string{"a@example.com", "b@example.com"} {
		if err := websession.Save(&websession.Session{
			UserEmail: email,
			Cookies:   map[string][]websession.Cookie{playOrigin: {{Name: "SAPISID", Value: "v"}}},
		}); err != nil {
			t.Fatal(err)
		}
	}
	cmd := AuthLogoutCommand()
	if err := cmd.FlagSet.Parse([]string{"--all", "--confirm"}); err != nil {
		t.Fatal(err)
	}
	stdout, err := captureWebStdout(func() error {
		return cmd.Exec(context.Background(), nil)
	})
	if err != nil {
		t.Fatalf("logout: %v", err)
	}
	if !strings.Contains(stdout, `"deleted":"all"`) {
		t.Errorf("output: %s", stdout)
	}
	emails, _ := websession.List()
	if len(emails) != 0 {
		t.Errorf("all sessions should be deleted, got %v", emails)
	}
}

// validBrowserCookies is a cookie set that passes the SAPISID precheck.
func validBrowserCookies() []map[string][]websession.Cookie {
	return []map[string][]websession.Cookie{{playOrigin: {{Name: "SAPISID", Value: "v"}}}}
}

// stubBrowserSeams replaces the dedicated-profile seams. Each call to the
// import seam returns the next entry in results (the last entry repeats), so a
// test can model "cookies show up on the Nth poll". Returns the launch and
// terminate counters for the interactive window.
func stubBrowserSeams(t *testing.T, results ...[]map[string][]websession.Cookie) (launches, terminated *int) {
	t.Helper()
	origImport, origLaunch, origPoll := importChromeCookiesFrom, interactiveLauncher, browserPollInterval
	calls, launchCount, terminateCount := 0, 0, 0
	importChromeCookiesFrom = func(ctx context.Context, dir string) ([]map[string][]websession.Cookie, error) {
		r := results[min(calls, len(results)-1)]
		calls++
		if r == nil {
			return nil, errors.New("no cookies yet")
		}
		return r, nil
	}
	interactiveLauncher = func(ctx context.Context, dir, url string) (func(), error) {
		launchCount++
		return func() { terminateCount++ }, nil
	}
	browserPollInterval = time.Millisecond
	t.Cleanup(func() {
		importChromeCookiesFrom, interactiveLauncher, browserPollInterval = origImport, origLaunch, origPoll
	})
	return &launchCount, &terminateCount
}

func TestWebAuthLogin_BrowserExcludesManualCookies(t *testing.T) {
	useTempSessionDir(t)
	cmd := AuthLoginCommand()
	if err := cmd.FlagSet.Parse([]string{"--email", "me@example.com", "--browser", "--cookies", "SAPISID=v"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil || !strings.Contains(err.Error(), "--browser") {
		t.Errorf("err = %v, want --browser exclusivity error", err)
	}
}

func TestWebAuthLogin_BrowserReusesSignedInProfile(t *testing.T) {
	useTempSessionDir(t)
	mockWebClient(t, consoleMock(t))
	launches, terminated := stubBrowserSeams(t, validBrowserCookies())

	cmd := AuthLoginCommand()
	if err := cmd.FlagSet.Parse([]string{"--email", "me@example.com", "--browser"}); err != nil {
		t.Fatal(err)
	}
	if _, err := captureWebStdout(func() error { return cmd.Exec(context.Background(), nil) }); err != nil {
		t.Fatalf("browser login: %v", err)
	}
	if *launches != 0 {
		t.Errorf("launches = %d, want 0 (profile already signed in)", *launches)
	}
	if *terminated != 0 {
		t.Errorf("terminated = %d, want 0 (no window was opened)", *terminated)
	}
	if _, err := websession.Load("me@example.com"); err != nil {
		t.Errorf("session not saved: %v", err)
	}
}

func TestWebAuthLogin_BrowserLaunchesAndWaitsForLogin(t *testing.T) {
	useTempSessionDir(t)
	mockWebClient(t, consoleMock(t))
	// Empty profile, then still empty, then the user finishes signing in.
	launches, terminated := stubBrowserSeams(t, nil, nil, validBrowserCookies())

	cmd := AuthLoginCommand()
	if err := cmd.FlagSet.Parse([]string{"--email", "me@example.com", "--browser"}); err != nil {
		t.Fatal(err)
	}
	if _, err := captureWebStdout(func() error { return cmd.Exec(context.Background(), nil) }); err != nil {
		t.Fatalf("browser login: %v", err)
	}
	if *launches != 1 {
		t.Errorf("launches = %d, want exactly 1", *launches)
	}
	if *terminated != 1 {
		t.Errorf("terminated = %d, want 1 (window closed once the session landed)", *terminated)
	}
	if _, err := websession.Load("me@example.com"); err != nil {
		t.Errorf("session not saved: %v", err)
	}
}

func TestWebAuthLogin_BrowserTimesOut(t *testing.T) {
	useTempSessionDir(t)
	mockWebClient(t, consoleMock(t))
	_, terminated := stubBrowserSeams(t, nil)

	cmd := AuthLoginCommand()
	if err := cmd.FlagSet.Parse([]string{"--email", "me@example.com", "--browser", "--browser-timeout", "20ms"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil || !strings.Contains(err.Error(), "timed out") {
		t.Errorf("err = %v, want timeout error", err)
	}
	if *terminated != 0 {
		t.Errorf("terminated = %d, want 0 (window left for the user to finish signing in)", *terminated)
	}
}
