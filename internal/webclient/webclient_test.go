package webclient

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/tamtom/play-console-cli/internal/testutil"
	"github.com/tamtom/play-console-cli/internal/websession"
)

func testSession() *websession.Session {
	return &websession.Session{
		Version:   1,
		UpdatedAt: time.Now(),
		UserEmail: "user@example.com",
		Cookies: map[string][]websession.Cookie{
			"https://play.google.com/": {
				{Name: "SID", Value: "sid-value"},
				{Name: "SAPISID", Value: "testSAPISIDvalue"},
				{Name: "__Secure-1PAPISID", Value: "onePAPISIDvalue"},
				{Name: "__Secure-3PAPISID", Value: "threePAPISIDvalue"},
			},
		},
	}
}

func TestAuthHeader_KnownVector(t *testing.T) {
	c := NewWithClient(testSession(), nil, DefaultBaseURL)
	c.now = func() time.Time { return time.Unix(1700000000, 0) }

	// Expected hashes computed by hand:
	//   sha1("1700000000 testSAPISIDvalue https://play.google.com")
	//   sha1("1700000000 onePAPISIDvalue https://play.google.com")
	//   sha1("1700000000 threePAPISIDvalue https://play.google.com")
	want := "SAPISIDHASH 1700000000_72f7de5144c401d0f339f04c9a11ee3f9bc21b13" +
		" SAPISID1PHASH 1700000000_681f5891e016967ed9160b14276c9a740e1e7ad5" +
		" SAPISID3PHASH 1700000000_97499670fc17144a52e479960c7f0599bac4bbe8"
	if got := c.authHeader(); got != want {
		t.Errorf("authHeader() = %q\nwant %q", got, want)
	}
}

func TestAuthHeader_SAPISIDOnly(t *testing.T) {
	sess := testSession()
	sess.Cookies["https://play.google.com/"] = sess.Cookies["https://play.google.com/"][:2]
	c := NewWithClient(sess, nil, DefaultBaseURL)
	c.now = func() time.Time { return time.Unix(1700000000, 0) }

	want := "SAPISIDHASH 1700000000_72f7de5144c401d0f339f04c9a11ee3f9bc21b13"
	if got := c.authHeader(); got != want {
		t.Errorf("authHeader() = %q, want %q", got, want)
	}
}

func TestAuthHeader_NoCookies(t *testing.T) {
	c := NewWithClient(&websession.Session{}, nil, DefaultBaseURL)
	if got := c.authHeader(); got != "" {
		t.Errorf("authHeader() = %q, want empty", got)
	}
}

func TestDiscoverDeveloperID(t *testing.T) {
	const developerID = "1234567890123456789"
	const path = "/console/u/2/developers"
	mock := testutil.NewMockAPI(t, map[string]http.HandlerFunc{
		"GET /console": func(w http.ResponseWriter, r *http.Request) {
			if got := r.URL.Query().Get("authuser"); got != "user@example.com" {
				t.Errorf("authuser = %q, want user@example.com", got)
			}
			http.Redirect(w, r, path, http.StatusFound)
		},
		"GET " + path: func(w http.ResponseWriter, r *http.Request) {
			// Real shape: field 1.1 is the console path, 2.1.1 the developer.
			fmt.Fprintf(w, `<html><script>window.serializedInitialChunks['startupData'] = `+
				`"\x7b\x221\x22:\x7b\x221\x22:\x22\/console\/u\/2\/developers\x22\x7d,`+
				`\x222\x22:\x7b\x221\x22:\x7b\x221\x22:\x22%s\x22\x7d\x7d\x7d";</script></html>`, developerID)
		},
	})
	c := NewWithClient(testSession(), nil, mock.BaseURL())
	id, err := c.DiscoverDeveloperID(context.Background())
	if err != nil {
		t.Fatalf("DiscoverDeveloperID: %v", err)
	}
	if id != developerID {
		t.Errorf("id = %q, want %s", id, developerID)
	}
}

func TestDiscoverDeveloperID_NotFound(t *testing.T) {
	mock := testutil.NewMockAPI(t, map[string]http.HandlerFunc{
		"GET /console": func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, `<html><body>nothing here</body></html>`)
		},
	})
	c := NewWithClient(testSession(), nil, mock.BaseURL())
	if _, err := c.DiscoverDeveloperID(context.Background()); !errors.Is(err, ErrAuth) {
		t.Errorf("err = %v, want ErrAuth", err)
	}
}

// realAppSummariesPayload is the actual json+protobuf response captured from
// playconsoleapps-pa for a single-app developer account.
const realAppSummariesPayload = `{"1":[{"1":{"1":{"1":"6901885972034847549"},"2":{"1":"4974539508825146246"}},"2":"Aérocoach","4":{"1":[1],"2":1},"5":"com.unifiedsense.aerocoach","8":1,"16":"en-US"}]}`

func TestListApps(t *testing.T) {
	t.Setenv(appsAPIKeyEnv, "")
	var gotPath, gotAuthUser, gotKey string
	mock := testutil.NewMockAPI(t, map[string]http.HandlerFunc{
		"GET /console": func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, `<script>window.serializedInitialChunks['startupData'] = `+
				`"\x7b\x221\x22:\x7b\x228\x22:\x22runtime-test-key\x22\x7d\x7d";</script>`)
		},
		"GET /v1/developers/6901885972034847549/appSummaries": func(w http.ResponseWriter, r *http.Request) {
			gotPath = r.URL.Path
			gotAuthUser = r.Header.Get("X-Goog-AuthUser")
			gotKey = r.Header.Get("X-Goog-Api-Key")
			fmt.Fprint(w, realAppSummariesPayload)
		},
	})
	c := NewWithClient(testSession(), nil, mock.BaseURL())
	apps, err := c.ListApps(context.Background(), "6901885972034847549")
	if err != nil {
		t.Fatalf("ListApps: %v", err)
	}
	if len(apps) != 1 {
		t.Fatalf("got %d apps, want 1", len(apps))
	}
	if apps[0].PackageName != "com.unifiedsense.aerocoach" || apps[0].DisplayName != "Aérocoach" {
		t.Errorf("app = %+v", apps[0])
	}
	if apps[0].AppID != "4974539508825146246" {
		t.Errorf("appID = %q", apps[0].AppID)
	}
	if gotKey != "runtime-test-key" {
		t.Errorf("X-Goog-Api-Key = %q, want key discovered from console HTML", gotKey)
	}
	// Sending X-Goog-AuthUser makes the real backend reject the credentials.
	if gotAuthUser != "" {
		t.Errorf("X-Goog-AuthUser = %q, want it never sent", gotAuthUser)
	}
	_ = gotPath
}

func TestListApps_MissingAPIKeyIsActionable(t *testing.T) {
	t.Setenv(appsAPIKeyEnv, "")
	mock := testutil.NewMockAPI(t, map[string]http.HandlerFunc{
		"GET /console": func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, `<script>window.serializedInitialChunks['startupData'] = `+
				`"\x7b\x221\x22:\x7b\x7d\x7d";</script>`)
		},
		"GET /v1/developers/dev1/appSummaries": func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, `{"1":[]}`)
		},
	})
	c := NewWithClient(testSession(), nil, mock.BaseURL())

	_, err := c.ListApps(context.Background(), "dev1")
	if err == nil {
		t.Error("ListApps succeeded, want missing API key error")
	} else {
		for _, want := range []string{"Play Console API key", appsAPIKeyEnv} {
			if !strings.Contains(err.Error(), want) {
				t.Errorf("error %q does not contain %q", err, want)
			}
		}
	}
	for _, req := range mock.RequestLog() {
		if req.Path == "/v1/developers/dev1/appSummaries" {
			t.Error("apps backend called without a discovered API key")
		}
	}
}

func TestAPIKey_ConcurrentDiscovery(t *testing.T) {
	t.Setenv(appsAPIKeyEnv, "")
	var consoleCalls atomic.Int32
	mock := testutil.NewMockAPI(t, map[string]http.HandlerFunc{
		"GET /console": func(w http.ResponseWriter, r *http.Request) {
			consoleCalls.Add(1)
			time.Sleep(10 * time.Millisecond)
			fmt.Fprint(w, `<script>window.serializedInitialChunks['startupData'] = `+
				`"\x7b\x221\x22:\x7b\x228\x22:\x22runtime-test-key\x22\x7d\x7d";</script>`)
		},
	})
	c := NewWithClient(testSession(), nil, mock.BaseURL())

	const workers = 16
	start := make(chan struct{})
	errs := make(chan error, workers)
	for range workers {
		go func() {
			<-start
			key, err := c.apiKey(context.Background())
			if err == nil && key != "runtime-test-key" {
				err = fmt.Errorf("apiKey = %q, want runtime-test-key", key)
			}
			errs <- err
		}()
	}
	close(start)
	for range workers {
		if err := <-errs; err != nil {
			t.Error(err)
		}
	}
	if got := consoleCalls.Load(); got != 1 {
		t.Errorf("console requests = %d, want one cached discovery", got)
	}
}

func TestListApps_FollowsPagination(t *testing.T) {
	t.Setenv(appsAPIKeyEnv, "test-key")
	var tokens []string
	mock := testutil.NewMockAPI(t, map[string]http.HandlerFunc{
		"GET /v1/developers/dev1/appSummaries": func(w http.ResponseWriter, r *http.Request) {
			token := r.URL.Query().Get("pageToken")
			tokens = append(tokens, token)
			if token == "" {
				fmt.Fprint(w, `{"1":[{"1":{"2":{"1":"a1"}},"2":"First","5":"com.first"}],"2":"tok2"}`)
				return
			}
			fmt.Fprint(w, `{"1":[{"1":{"2":{"1":"a2"}},"2":"Second","5":"com.second"}]}`)
		},
	})
	c := NewWithClient(testSession(), nil, mock.BaseURL())
	apps, err := c.ListApps(context.Background(), "dev1")
	if err != nil {
		t.Fatalf("ListApps: %v", err)
	}
	if len(apps) != 2 {
		t.Fatalf("got %d apps, want 2 across pages", len(apps))
	}
	if len(tokens) != 2 || tokens[1] != "tok2" {
		t.Errorf("page tokens = %v, want second request to carry tok2", tokens)
	}
}

func TestListApps_AuthError(t *testing.T) {
	t.Setenv(appsAPIKeyEnv, "test-key")
	mock := testutil.NewMockAPI(t, map[string]http.HandlerFunc{
		"GET /v1/developers/dev1/appSummaries": func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusForbidden)
		},
	})
	c := NewWithClient(testSession(), nil, mock.BaseURL())
	if _, err := c.ListApps(context.Background(), "dev1"); !errors.Is(err, ErrAuth) {
		t.Errorf("err = %v, want ErrAuth", err)
	}
}

// realStartupData mirrors the console's escaped startup blob: field 2.1.1 is
// the developer account currently in scope.
const realStartupData = `<html><script>window.serializedInitialChunks['startupData'] = ` +
	`"\x7b\x221\x22:\x7b\x221\x22:\x22\/console\/u\/2\/developers\x22\x7d,` +
	`\x222\x22:\x7b\x221\x22:\x7b\x221\x22:\x226901885972034847549\x22\x7d,\x222\x22:3\x7d\x7d";</script></html>`

func TestParseStartupDeveloperID(t *testing.T) {
	id, err := parseStartupDeveloperID([]byte(realStartupData))
	if err != nil {
		t.Fatalf("parseStartupDeveloperID: %v", err)
	}
	if id != "6901885972034847549" {
		t.Errorf("id = %q, want 6901885972034847549", id)
	}
}

// Unrelated 19-digit numbers in the page must not be mistaken for developer
// accounts: the old regex scan flagged them and broke discovery.
func TestDiscoverDeveloperID_IgnoresUnrelated19DigitNumbers(t *testing.T) {
	mock := testutil.NewMockAPI(t, map[string]http.HandlerFunc{
		"GET /console": func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, `<html><script>var noise = "9117404620569054295";</script>`+realStartupData+`</html>`)
		},
	})
	c := NewWithClient(testSession(), nil, mock.BaseURL())
	id, err := c.DiscoverDeveloperID(context.Background())
	if err != nil {
		t.Fatalf("DiscoverDeveloperID: %v", err)
	}
	if id != "6901885972034847549" {
		t.Errorf("id = %q, want the structurally parsed developer", id)
	}
}

func TestCheckPackageName(t *testing.T) {
	t.Setenv(appsAPIKeyEnv, "test-key")
	var gotQuery, gotMethod string
	mock := testutil.NewMockAPI(t, map[string]http.HandlerFunc{
		"GET /v1/developers/dev1:checkPackageNameAvailability": func(w http.ResponseWriter, r *http.Request) {
			gotMethod, gotQuery = r.Method, r.URL.Query().Get("packageName")
			fmt.Fprint(w, `{"1":1}`) // AVAILABILITY_AVAILABLE, as captured live
		},
	})
	c := NewWithClient(testSession(), nil, mock.BaseURL())
	avail, err := c.CheckPackageName(context.Background(), "dev1", "com.unifiedsense.matisse")
	if err != nil {
		t.Fatalf("CheckPackageName: %v", err)
	}
	if !avail.Available() {
		t.Errorf("availability = %v, want available", avail)
	}
	if gotMethod != http.MethodGet {
		t.Errorf("method = %s, want GET", gotMethod)
	}
	if gotQuery != "com.unifiedsense.matisse" {
		t.Errorf("packageName param = %q", gotQuery)
	}
}

func TestCheckPackageName_Taken(t *testing.T) {
	t.Setenv(appsAPIKeyEnv, "test-key")
	mock := testutil.NewMockAPI(t, map[string]http.HandlerFunc{
		"GET /v1/developers/dev1:checkPackageNameAvailability": func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, `{"1":4}`) // AVAILABILITY_UNAVAILABLE
		},
	})
	c := NewWithClient(testSession(), nil, mock.BaseURL())
	avail, err := c.CheckPackageName(context.Background(), "dev1", "com.taken")
	if err != nil {
		t.Fatalf("CheckPackageName: %v", err)
	}
	if avail.Available() {
		t.Error("expected unavailable")
	}
}

func TestCreateApp(t *testing.T) {
	t.Setenv(appsAPIKeyEnv, "test-key")
	var gotBody, gotMethod string
	mock := testutil.NewMockAPI(t, map[string]http.HandlerFunc{
		"POST /v1/developers/dev1:createAppV2": func(w http.ResponseWriter, r *http.Request) {
			b, _ := io.ReadAll(r.Body)
			gotBody, gotMethod = string(b), r.Method
			fmt.Fprint(w, `{"1":{"1":{"1":"dev1"},"2":{"1":"999888777"}}}`)
		},
	})
	c := NewWithClient(testSession(), nil, mock.BaseURL())
	app, err := c.CreateApp(context.Background(), CreateAppRequest{
		DeveloperID: "dev1", Title: "Matisse", DefaultLanguage: "en-US",
		Kind: AppKindApp, Paid: false, PackageName: "com.unifiedsense.matisse",
		MeetsGuidelines: true, USLawCompliant: true,
	})
	if err != nil {
		t.Fatalf("CreateApp: %v", err)
	}
	if app.AppID != "999888777" {
		t.Errorf("appID = %q, want 999888777", app.AppID)
	}
	if gotMethod != http.MethodPost {
		t.Errorf("method = %s, want POST", gotMethod)
	}
	// Wire contract: numeric field keys from the console's own schema.
	for _, want := range []string{`"1":{"1":"dev1"}`, `"2":"Matisse"`, `"3":"en-US"`, `"4":1`, `"5":false`, `"6":true`, `"7":true`, `"11":"com.unifiedsense.matisse"`} {
		if !strings.Contains(gotBody, want) {
			t.Errorf("body missing %s: %s", want, gotBody)
		}
	}
}

func TestCreateApp_RequiresDeclarations(t *testing.T) {
	c := NewWithClient(testSession(), nil, DefaultBaseURL)
	_, err := c.CreateApp(context.Background(), CreateAppRequest{
		DeveloperID: "dev1", Title: "X", DefaultLanguage: "en-US", Kind: AppKindApp,
		PackageName: "com.x", MeetsGuidelines: false, USLawCompliant: true,
	})
	if err == nil || !strings.Contains(err.Error(), "declaration") {
		t.Errorf("err = %v, want declaration error", err)
	}
}

func TestCreateApp_GameKind(t *testing.T) {
	t.Setenv(appsAPIKeyEnv, "test-key")
	var gotBody string
	mock := testutil.NewMockAPI(t, map[string]http.HandlerFunc{
		"POST /v1/developers/dev1:createAppV2": func(w http.ResponseWriter, r *http.Request) {
			b, _ := io.ReadAll(r.Body)
			gotBody = string(b)
			fmt.Fprint(w, `{"1":{"2":{"1":"1"}}}`)
		},
	})
	c := NewWithClient(testSession(), nil, mock.BaseURL())
	if _, err := c.CreateApp(context.Background(), CreateAppRequest{
		DeveloperID: "dev1", Title: "G", DefaultLanguage: "en-US", Kind: AppKindGame,
		Paid: true, PackageName: "com.g", MeetsGuidelines: true, USLawCompliant: true,
	}); err != nil {
		t.Fatalf("CreateApp: %v", err)
	}
	if !strings.Contains(gotBody, `"4":2`) || !strings.Contains(gotBody, `"5":true`) {
		t.Errorf("game/paid encoding wrong: %s", gotBody)
	}
}
