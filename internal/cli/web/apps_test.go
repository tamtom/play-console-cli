package web

import (
	"context"
	"errors"
	"io"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/tamtom/play-console-cli/internal/cli/shared"
	"github.com/tamtom/play-console-cli/internal/testutil"
	"github.com/tamtom/play-console-cli/internal/webdriver"
	"github.com/tamtom/play-console-cli/internal/websession"
)

// saveWebSession stores a minimal valid session for the test account.
func saveWebSession(t *testing.T) {
	const email = "me@example.com"
	t.Helper()
	if err := websession.Save(&websession.Session{
		UserEmail: email,
		Cookies:   map[string][]websession.Cookie{playOrigin: {{Name: "SAPISID", Value: "v"}}},
	}); err != nil {
		t.Fatal(err)
	}
}

// appsListMock serves developer discovery plus the app-summaries endpoint.
func appsListMock(t *testing.T, payload string) *testutil.MockAPI {
	t.Helper()
	return testutil.NewMockAPI(t, map[string]http.HandlerFunc{
		"GET /console": func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, authConsolePath, http.StatusFound)
		},
		"GET " + authConsolePath: func(w http.ResponseWriter, r *http.Request) {
			_, _ = io.WriteString(w, authConsoleHTML)
		},
		"GET /v1/developers/" + authDeveloperID + "/appSummaries": func(w http.ResponseWriter, r *http.Request) {
			_, _ = io.WriteString(w, payload)
		},
	})
}

func TestWebAppsList_Executes(t *testing.T) {
	useTempSessionDir(t)
	saveWebSession(t)
	mockWebClient(t, appsListMock(t, `{"1":[{"1":{"1":{"1":"`+authDeveloperID+`"},"2":{"1":"555"}},"2":"Aérocoach","5":"com.example.demo","16":"en-US"}]}`))

	cmd := WebAppsListCommand()
	if err := cmd.FlagSet.Parse(nil); err != nil {
		t.Fatal(err)
	}
	stdout, err := captureWebStdout(func() error { return cmd.Exec(context.Background(), nil) })
	if err != nil {
		t.Fatalf("apps list: %v", err)
	}
	for _, want := range []string{"com.example.demo", "Aérocoach", "555"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("output missing %q: %s", want, stdout)
		}
	}
}

func TestWebAppsList_NoSession(t *testing.T) {
	useTempSessionDir(t)
	cmd := WebAppsListCommand()
	if err := cmd.FlagSet.Parse(nil); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil || !strings.Contains(err.Error(), "no web session") {
		t.Errorf("err = %v, want no-session error", err)
	}
}

func TestWebAppsCommands_RegisteredInGroup(t *testing.T) {
	var names []string
	for _, sub := range WebAppsCommand().Subcommands {
		names = append(names, sub.Name)
	}
	for _, want := range []string{"list", "create", "update", "status", "availability", "pricing", "review", "rollout", "declarations", "policy", "publish", "distribution", "promo-codes", "rating"} {
		if !slices.Contains(names, want) {
			t.Errorf("web apps subcommands = %v, want %q", names, want)
		}
	}
}

func TestWebAppsList_TableOutput(t *testing.T) {
	useTempSessionDir(t)
	saveWebSession(t)
	mockWebClient(t, appsListMock(t, `{"1":[{"1":{"1":{"1":"`+authDeveloperID+`"},"2":{"1":"555"}},"2":"Aérocoach","5":"com.example.demo","16":"en-US"}]}`))

	cmd := WebAppsListCommand()
	if err := cmd.FlagSet.Parse([]string{"--output", "table"}); err != nil {
		t.Fatal(err)
	}
	stdout, err := captureWebStdout(func() error { return cmd.Exec(context.Background(), nil) })
	if err != nil {
		t.Fatalf("apps list: %v", err)
	}
	if strings.HasPrefix(strings.TrimSpace(stdout), "[") || strings.Contains(stdout, `"packageName"`) {
		t.Errorf("table output fell back to JSON: %s", stdout)
	}
	for _, want := range []string{"PACKAGE", "com.example.demo"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("table output missing %q: %s", want, stdout)
		}
	}
}

// expiringAppsMock rejects the first app-list request with 403 and serves the
// second, modeling a session that expires between runs.
func expiringAppsMock(t *testing.T, payload string) *testutil.MockAPI {
	t.Helper()
	calls := 0
	return testutil.NewMockAPI(t, map[string]http.HandlerFunc{
		// The refresh path revalidates the recovered cookies against /console.
		"GET /console": func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, authConsolePath, http.StatusFound)
		},
		"GET " + authConsolePath: func(w http.ResponseWriter, r *http.Request) {
			_, _ = io.WriteString(w, authConsoleHTML)
		},
		"GET /v1/developers/" + authDeveloperID + "/appSummaries": func(w http.ResponseWriter, r *http.Request) {
			calls++
			if calls == 1 {
				w.WriteHeader(http.StatusForbidden)
				return
			}
			_, _ = io.WriteString(w, payload)
		},
	})
}

func TestWebAppsList_RefreshesExpiredSessionFromBrowserProfile(t *testing.T) {
	useTempSessionDir(t)
	saveWebSession(t)
	mockWebClient(t, expiringAppsMock(t, `{"1":[{"1":{"2":{"1":"555"}},"2":"Aérocoach","5":"com.example.demo"}]}`))
	launches, _ := stubBrowserSeams(t, validBrowserCookies())

	cmd := WebAppsListCommand()
	if err := cmd.FlagSet.Parse([]string{"--developer", authDeveloperID}); err != nil {
		t.Fatal(err)
	}
	stdout, err := captureWebStdout(func() error { return cmd.Exec(context.Background(), nil) })
	if err != nil {
		t.Fatalf("apps list should recover from an expired session: %v", err)
	}
	if !strings.Contains(stdout, "com.example.demo") {
		t.Errorf("output = %s", stdout)
	}
	if *launches != 0 {
		t.Errorf("launches = %d, want 0 (must not pop a browser window mid-command)", *launches)
	}
}

func TestWebAppsList_ExpiredWithNoBrowserProfileReportsAuthError(t *testing.T) {
	useTempSessionDir(t)
	saveWebSession(t)
	mockWebClient(t, expiringAppsMock(t, `{"1":[]}`))
	stubBrowserSeams(t, nil) // profile unusable

	cmd := WebAppsListCommand()
	if err := cmd.FlagSet.Parse([]string{"--developer", authDeveloperID}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil || !strings.Contains(err.Error(), "web auth login") {
		t.Errorf("err = %v, want actionable auth error", err)
	}
}

// createMock serves discovery, the availability check, and the create call,
// recording the order so the test can prove the check precedes the write.
func createMock(t *testing.T, availability string, order *[]string) *testutil.MockAPI {
	t.Helper()
	return testutil.NewMockAPI(t, map[string]http.HandlerFunc{
		"GET /console": func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, authConsolePath, http.StatusFound)
		},
		"GET " + authConsolePath: func(w http.ResponseWriter, r *http.Request) {
			_, _ = io.WriteString(w, authConsoleHTML)
		},
		"GET /v1/developers/" + authDeveloperID + ":checkPackageNameAvailability": func(w http.ResponseWriter, r *http.Request) {
			*order = append(*order, "check")
			_, _ = io.WriteString(w, availability)
		},
		"POST /v1/developers/" + authDeveloperID + ":createAppV2": func(w http.ResponseWriter, r *http.Request) {
			*order = append(*order, "create")
			_, _ = io.WriteString(w, `{"1":{"1":{"1":"`+authDeveloperID+`"},"2":{"1":"999888777"}}}`)
		},
	})
}

func createArgs() []string {
	return []string{
		"--name", "Matisse", "--package", "com.example.matisse",
		"--kind", "app", "--pricing", "free",
		"--accept-policies", "--accept-us-export-laws", "--confirm",
	}
}

func TestWebAppsCreate_AbortsWhenPackageTaken(t *testing.T) {
	useTempSessionDir(t)
	saveWebSession(t)
	var order []string
	mockWebClient(t, createMock(t, `{"1":4}`, &order))

	cmd := WebAppsCreateCommand()
	if err := cmd.FlagSet.Parse(createArgs()); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil || !strings.Contains(err.Error(), "taken") {
		t.Errorf("err = %v, want unavailable error", err)
	}
	for _, c := range order {
		if c == "create" {
			t.Fatal("must not create an app when the package name is unavailable")
		}
	}
}

func TestWebAppsCreate_RequiresDeclarations(t *testing.T) {
	useTempSessionDir(t)
	saveWebSession(t)
	cmd := WebAppsCreateCommand()
	args := []string{"--name", "Matisse", "--package", "com.x", "--kind", "app", "--pricing", "free", "--confirm"}
	if err := cmd.FlagSet.Parse(args); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil || !strings.Contains(err.Error(), "--accept-policies") {
		t.Errorf("err = %v, want declaration flags required", err)
	}
}

func TestWebAppsCreate_RequiresConfirm(t *testing.T) {
	useTempSessionDir(t)
	cmd := WebAppsCreateCommand()
	args := []string{
		"--name", "M", "--package", "com.x", "--kind", "app", "--pricing", "free",
		"--accept-policies", "--accept-us-export-laws",
	}
	if err := cmd.FlagSet.Parse(args); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil || !strings.Contains(err.Error(), "--confirm") {
		t.Errorf("err = %v, want --confirm required", err)
	}
}

func TestWebAppsCreate_DryRunSkipsNetwork(t *testing.T) {
	useTempSessionDir(t)
	// No session saved: success proves dry-run short-circuits before any I/O.
	cmd := WebAppsCreateCommand()
	if err := cmd.FlagSet.Parse(createArgs()); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Exec(shared.ContextWithDryRun(context.Background(), true), nil); err != nil {
		t.Fatalf("dry-run: %v", err)
	}
}

func TestWebAppsCreate_ValidatesPackageName(t *testing.T) {
	useTempSessionDir(t)
	cmd := WebAppsCreateCommand()
	if err := cmd.FlagSet.Parse([]string{
		"--name", "M", "--kind", "app", "--pricing", "free",
		"--accept-policies", "--accept-us-export-laws", "--confirm",
	}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil || !strings.Contains(err.Error(), "--package") {
		t.Errorf("err = %v, want --package required", err)
	}
}

// fakeCreator records the browser-driven create steps.
type fakeCreator struct {
	steps  []string
	form   webdriver.AppForm
	state  webdriver.FormState
	appID  string
	failAt string
}

func (f *fakeCreator) Fill(_ context.Context, _ string, form webdriver.AppForm) error {
	f.steps = append(f.steps, "fill")
	f.form = form
	if f.failAt == "fill" {
		return errors.New("fill failed")
	}
	return nil
}

func (f *fakeCreator) Read(context.Context) (*webdriver.FormState, error) {
	f.steps = append(f.steps, "read")
	s := f.state
	return &s, nil
}

func (f *fakeCreator) Submit(context.Context, time.Duration) (string, error) {
	f.steps = append(f.steps, "submit")
	return f.appID, nil
}

func (f *fakeCreator) Close() error { return nil }

// stubCreator installs the fake browser driver.
func stubCreator(t *testing.T, f *fakeCreator) {
	t.Helper()
	orig := newAppCreator
	newAppCreator = func(context.Context, string, time.Duration) (appCreator, error) { return f, nil }
	t.Cleanup(func() { newAppCreator = orig })
}

// goodState is a form the console would accept.
func goodState() webdriver.FormState {
	return webdriver.FormState{
		Title: "Matisse", PackageName: "com.example.matisse",
		Game: false, Paid: false, Policies: true, Export: true, CanSubmit: true,
	}
}

func TestWebAppsCreate_DrivesBrowserForm(t *testing.T) {
	useTempSessionDir(t)
	saveWebSession(t)
	var order []string
	mockWebClient(t, createMock(t, `{"1":1}`, &order))
	f := &fakeCreator{state: goodState(), appID: "9876543210987654"}
	stubCreator(t, f)

	cmd := WebAppsCreateCommand()
	if err := cmd.FlagSet.Parse(createArgs()); err != nil {
		t.Fatal(err)
	}
	stdout, err := captureWebStdout(func() error { return cmd.Exec(context.Background(), nil) })
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if strings.Join(f.steps, ",") != "fill,read,submit" {
		t.Errorf("steps = %v, want fill,read,submit", f.steps)
	}
	if f.form.Title != "Matisse" || f.form.PackageName != "com.example.matisse" || f.form.Game || f.form.Paid {
		t.Errorf("form = %+v", f.form)
	}
	if !strings.Contains(stdout, "9876543210987654") {
		t.Errorf("output = %s", stdout)
	}
	// The availability check must still run against the API first.
	if len(order) == 0 || order[0] != "check" {
		t.Errorf("api calls = %v, want the availability check first", order)
	}
}

func TestWebAppsCreate_RefusesToSubmitMismatchedForm(t *testing.T) {
	useTempSessionDir(t)
	saveWebSession(t)
	var order []string
	mockWebClient(t, createMock(t, `{"1":1}`, &order))
	bad := goodState()
	bad.PackageName = "com.something.else" // page did not take our value
	f := &fakeCreator{state: bad, appID: "x"}
	stubCreator(t, f)

	cmd := WebAppsCreateCommand()
	if err := cmd.FlagSet.Parse(createArgs()); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil || !strings.Contains(err.Error(), "does not match") {
		t.Errorf("err = %v, want mismatch error", err)
	}
	for _, s := range f.steps {
		if s == "submit" {
			t.Fatal("must not submit a form that does not match the request")
		}
	}
}

func TestWebAppsCreate_RefusesWhenSubmitDisabled(t *testing.T) {
	useTempSessionDir(t)
	saveWebSession(t)
	var order []string
	mockWebClient(t, createMock(t, `{"1":1}`, &order))
	st := goodState()
	st.CanSubmit = false
	f := &fakeCreator{state: st}
	stubCreator(t, f)

	cmd := WebAppsCreateCommand()
	if err := cmd.FlagSet.Parse(createArgs()); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil || !strings.Contains(err.Error(), "not ready") {
		t.Errorf("err = %v, want not-ready error", err)
	}
}

func updateArgs() []string {
	return []string{
		"--package", "com.example.demo",
		"--kind", "game",
		"--category", "Educational",
		"--confirm",
	}
}

func TestWebAppsUpdate_ValidatesFlags(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "package", args: []string{"--kind", "game", "--category", "Educational", "--confirm"}, want: "--package"},
		{name: "kind", args: []string{"--package", "com.example", "--kind", "tool", "--category", "Educational", "--confirm"}, want: "--kind must be app or game"},
		{name: "category", args: []string{"--package", "com.example", "--kind", "app", "--confirm"}, want: "--category"},
		{name: "confirm", args: []string{"--package", "com.example", "--kind", "app", "--category", "Education"}, want: "--confirm"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := WebAppsUpdateCommand()
			if err := cmd.FlagSet.Parse(tt.args); err != nil {
				t.Fatal(err)
			}
			err := cmd.Exec(context.Background(), nil)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Errorf("err = %v, want %q", err, tt.want)
			}
		})
	}
}

func TestWebAppsUpdate_DryRunSkipsBrowser(t *testing.T) {
	f := &fakeUpdater{failAt: "open"}
	stubUpdater(t, f)

	cmd := WebAppsUpdateCommand()
	if err := cmd.FlagSet.Parse(updateArgs()); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Exec(shared.ContextWithDryRun(context.Background(), true), nil); err != nil {
		t.Fatalf("dry-run: %v", err)
	}
	if len(f.steps) != 0 {
		t.Errorf("browser steps = %v, want none", f.steps)
	}
}

type fakeUpdater struct {
	steps       []string
	states      []webdriver.AppSettingsState
	developerID string
	appID       string
	account     string
	setKind     string
	setCategory string
	failAt      string
}

func (f *fakeUpdater) Open(_ context.Context, developerID, appID, account string) error {
	f.steps = append(f.steps, "open")
	f.developerID = developerID
	f.appID = appID
	f.account = account
	if f.failAt == "open" {
		return errors.New("open failed")
	}
	return nil
}

func (f *fakeUpdater) Read(context.Context) (*webdriver.AppSettingsState, error) {
	f.steps = append(f.steps, "read")
	if f.failAt == "read" {
		return nil, errors.New("read failed")
	}
	if len(f.states) == 0 {
		return nil, errors.New("no fake state")
	}
	state := f.states[0]
	f.states = f.states[1:]
	return &state, nil
}

func (f *fakeUpdater) SetClassification(_ context.Context, kind, category string) error {
	f.steps = append(f.steps, "set")
	f.setKind = kind
	f.setCategory = category
	if f.failAt == "set" {
		return errors.New("set failed")
	}
	return nil
}

func (f *fakeUpdater) Submit(context.Context, time.Duration) error {
	f.steps = append(f.steps, "submit")
	if f.failAt == "submit" {
		return errors.New("submit failed")
	}
	return nil
}

func (f *fakeUpdater) Close() error { return nil }

func stubUpdater(t *testing.T, f *fakeUpdater) {
	t.Helper()
	orig := newAppUpdater
	newAppUpdater = func(context.Context, string, time.Duration) (appUpdater, error) { return f, nil }
	t.Cleanup(func() { newAppUpdater = orig })
}

func setupUpdate(t *testing.T, f *fakeUpdater) {
	t.Helper()
	useTempSessionDir(t)
	saveWebSession(t)
	mockWebClient(t, appsListMock(t, `{"1":[{"1":{"1":{"1":"`+authDeveloperID+`"},"2":{"1":"555"}},"2":"Aérocoach","5":"com.example.demo","16":"en-US"}]}`))
	stubUpdater(t, f)
}

func TestWebAppsUpdate_NoOpSkipsWrite(t *testing.T) {
	f := &fakeUpdater{states: []webdriver.AppSettingsState{{Kind: "game", Category: "Educational"}}}
	setupUpdate(t, f)

	cmd := WebAppsUpdateCommand()
	if err := cmd.FlagSet.Parse(updateArgs()); err != nil {
		t.Fatal(err)
	}
	stdout, err := captureWebStdout(func() error { return cmd.Exec(context.Background(), nil) })
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if got := strings.Join(f.steps, ","); got != "open,read" {
		t.Errorf("steps = %s, want open,read", got)
	}
	if !strings.Contains(stdout, `"changed":false`) {
		t.Errorf("output = %s, want unchanged result", stdout)
	}
}

func TestWebAppsUpdate_DrivesAndVerifiesForm(t *testing.T) {
	f := &fakeUpdater{states: []webdriver.AppSettingsState{
		{Kind: "app", Category: "Education"},
		{Kind: "game", Category: "Educational", CanSubmit: true},
		{Kind: "game", Category: "Educational"},
	}}
	setupUpdate(t, f)

	cmd := WebAppsUpdateCommand()
	if err := cmd.FlagSet.Parse(updateArgs()); err != nil {
		t.Fatal(err)
	}
	stdout, err := captureWebStdout(func() error { return cmd.Exec(context.Background(), nil) })
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if got := strings.Join(f.steps, ","); got != "open,read,set,read,submit,open,read" {
		t.Errorf("steps = %s", got)
	}
	if f.developerID != authDeveloperID || f.appID != "555" {
		t.Errorf("opened developer/app = %s/%s", f.developerID, f.appID)
	}
	if f.account != "me@example.com" {
		t.Errorf("opened account = %q", f.account)
	}
	if f.setKind != "game" || f.setCategory != "Educational" {
		t.Errorf("set classification = %q/%q", f.setKind, f.setCategory)
	}
	for _, want := range []string{`"kind":"game"`, `"category":"Educational"`, `"changed":true`} {
		if !strings.Contains(stdout, want) {
			t.Errorf("output = %s, want %s", stdout, want)
		}
	}
}

func TestWebAppsUpdate_RefusesUnsafeSubmit(t *testing.T) {
	tests := []struct {
		name  string
		state webdriver.AppSettingsState
		want  string
	}{
		{name: "mismatch", state: webdriver.AppSettingsState{Kind: "app", Category: "Educational", CanSubmit: true}, want: "does not match"},
		{name: "disabled", state: webdriver.AppSettingsState{Kind: "game", Category: "Educational"}, want: "not ready"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &fakeUpdater{states: []webdriver.AppSettingsState{{Kind: "app", Category: "Education"}, tt.state}}
			setupUpdate(t, f)

			cmd := WebAppsUpdateCommand()
			if err := cmd.FlagSet.Parse(updateArgs()); err != nil {
				t.Fatal(err)
			}
			err := cmd.Exec(context.Background(), nil)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Errorf("err = %v, want %q", err, tt.want)
			}
			if slices.Contains(f.steps, "submit") {
				t.Fatal("must not submit an unverified form")
			}
		})
	}
}

func TestWebAppsUpdate_VerifiesSavedValue(t *testing.T) {
	f := &fakeUpdater{states: []webdriver.AppSettingsState{
		{Kind: "app", Category: "Education"},
		{Kind: "game", Category: "Educational", CanSubmit: true},
		{Kind: "app", Category: "Education"},
	}}
	setupUpdate(t, f)

	cmd := WebAppsUpdateCommand()
	if err := cmd.FlagSet.Parse(updateArgs()); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil || !strings.Contains(err.Error(), "was not saved") {
		t.Errorf("err = %v, want post-save verification error", err)
	}
}

func TestWebAppsUpdate_RejectsUnknownPackageBeforeBrowser(t *testing.T) {
	useTempSessionDir(t)
	saveWebSession(t)
	mockWebClient(t, appsListMock(t, `{"1":[]}`))
	f := &fakeUpdater{failAt: "open"}
	stubUpdater(t, f)

	cmd := WebAppsUpdateCommand()
	if err := cmd.FlagSet.Parse(updateArgs()); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Errorf("err = %v, want package-not-found error", err)
	}
	if len(f.steps) != 0 {
		t.Errorf("browser steps = %v, want none", f.steps)
	}
}
