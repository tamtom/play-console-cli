package web

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/tamtom/play-console-cli/internal/cli/shared"
	"github.com/tamtom/play-console-cli/internal/webdriver"
)

type fakePublishBrowser struct {
	steps         []string
	overviews     []*webdriver.PublishingOverview
	setup         *webdriver.AppSetup
	countries     []webdriver.CountriesState
	pricing       []webdriver.AppPricingState
	prepare       *webdriver.PrepareState
	review        *webdriver.ReviewState
	declarations  *webdriver.DeclarationsState
	policy        *webdriver.PolicyStatus
	distribution  []*webdriver.DistributionState
	addedFactor   string
	questSteps    []webdriver.QuestionnaireStep
	questHasNext  bool
	setChoices    [][]string
	importedCSV   string
	setNames      []string
	unsetNames    []string
	setPrice      string
	setPrivacyURL string
	setRadioPage  string
	setRadioYes   bool
	setManagedOn  bool
	developerID   string
	appID         string
	account       string
	failAt        string
}

func (f *fakePublishBrowser) ReadOverview(_ context.Context, developerID, appID, account string) (*webdriver.PublishingOverview, error) {
	f.steps = append(f.steps, "read-overview")
	f.developerID, f.appID, f.account = developerID, appID, account
	if f.failAt == "read-overview" {
		return nil, errors.New("read overview failed")
	}
	if len(f.overviews) == 0 {
		return nil, errors.New("no fake overview")
	}
	o := f.overviews[0]
	f.overviews = f.overviews[1:]
	return o, nil
}

func (f *fakePublishBrowser) ReadSetup(context.Context, string, string, string) (*webdriver.AppSetup, error) {
	f.steps = append(f.steps, "read-setup")
	if f.setup == nil {
		return nil, errors.New("no fake setup")
	}
	return f.setup, nil
}

func (f *fakePublishBrowser) SendForReview(context.Context, time.Duration) error {
	f.steps = append(f.steps, "send")
	if f.failAt == "send" {
		return errors.New("send failed")
	}
	return nil
}

func (f *fakePublishBrowser) OpenCountries(_ context.Context, developerID, appID, account, track string) error {
	f.steps = append(f.steps, "open-countries")
	f.developerID, f.appID, f.account = developerID, appID, account
	if f.failAt == "open-countries" {
		return errors.New("open failed")
	}
	return nil
}

func (f *fakePublishBrowser) ReadCountries(context.Context) (*webdriver.CountriesState, error) {
	f.steps = append(f.steps, "read-countries")
	if f.failAt == "read-countries" {
		return nil, errors.New("read failed")
	}
	if len(f.countries) == 0 {
		return nil, errors.New("no fake countries state")
	}
	state := f.countries[0]
	f.countries = f.countries[1:]
	return &state, nil
}

func (f *fakePublishBrowser) SetCountries(_ context.Context, names []string) error {
	f.steps = append(f.steps, "set-countries")
	f.setNames = names
	if f.failAt == "set-countries" {
		return errors.New("set failed")
	}
	return nil
}

func (f *fakePublishBrowser) UnsetCountries(_ context.Context, names []string) error {
	f.steps = append(f.steps, "unset-countries")
	f.unsetNames = names
	if f.failAt == "unset-countries" {
		return errors.New("unset failed")
	}
	return nil
}

func (f *fakePublishBrowser) SetAllCountries(context.Context) error {
	f.steps = append(f.steps, "set-all")
	if f.failAt == "set-all" {
		return errors.New("set all failed")
	}
	return nil
}

func (f *fakePublishBrowser) SubmitCountries(context.Context, time.Duration) error {
	f.steps = append(f.steps, "submit-countries")
	if f.failAt == "submit-countries" {
		return errors.New("submit failed")
	}
	return nil
}

func (f *fakePublishBrowser) DiscardCountries(context.Context) error {
	f.steps = append(f.steps, "discard")
	return nil
}

func (f *fakePublishBrowser) OpenPricing(_ context.Context, developerID, appID, account string) error {
	f.steps = append(f.steps, "open-pricing")
	f.developerID, f.appID, f.account = developerID, appID, account
	if f.failAt == "open-pricing" {
		return errors.New("open pricing failed")
	}
	return nil
}

func (f *fakePublishBrowser) ReadPricing(context.Context) (*webdriver.AppPricingState, error) {
	f.steps = append(f.steps, "read-pricing")
	if f.failAt == "read-pricing" {
		return nil, errors.New("read pricing failed")
	}
	if len(f.pricing) == 0 {
		return nil, errors.New("no fake pricing state")
	}
	state := f.pricing[0]
	f.pricing = f.pricing[1:]
	return &state, nil
}

func (f *fakePublishBrowser) SetPrice(_ context.Context, price string) error {
	f.steps = append(f.steps, "set-price")
	f.setPrice = price
	if f.failAt == "set-price" {
		return errors.New("set price failed")
	}
	return nil
}

func (f *fakePublishBrowser) SubmitPricing(context.Context, time.Duration) error {
	f.steps = append(f.steps, "submit-pricing")
	if f.failAt == "submit-pricing" {
		return errors.New("submit pricing failed")
	}
	return nil
}

func (f *fakePublishBrowser) DiscardPricing(context.Context) error {
	f.steps = append(f.steps, "discard-pricing")
	return nil
}

func (f *fakePublishBrowser) OpenProductionReleases(_ context.Context, developerID, appID, account string) error {
	f.steps = append(f.steps, "open-releases")
	f.developerID, f.appID, f.account = developerID, appID, account
	if f.failAt == "open-releases" {
		return errors.New("open releases failed")
	}
	return nil
}

func (f *fakePublishBrowser) OpenDraftRelease(context.Context) (*webdriver.PrepareState, error) {
	f.steps = append(f.steps, "open-draft")
	if f.failAt == "open-draft" {
		return nil, errors.New("open draft failed")
	}
	if f.prepare == nil {
		return nil, errors.New("no fake prepare state")
	}
	return f.prepare, nil
}

func (f *fakePublishBrowser) ReviewDraftRelease(context.Context) (*webdriver.ReviewState, error) {
	f.steps = append(f.steps, "review-draft")
	if f.failAt == "review-draft" {
		return nil, errors.New("review draft failed")
	}
	if f.review == nil {
		return nil, errors.New("no fake review state")
	}
	return f.review, nil
}

func (f *fakePublishBrowser) SaveReleaseReview(context.Context, time.Duration) error {
	f.steps = append(f.steps, "save-review")
	if f.failAt == "save-review" {
		return errors.New("save review failed")
	}
	return nil
}

func (f *fakePublishBrowser) ReadDeclarations(_ context.Context, developerID, appID, account string) (*webdriver.DeclarationsState, error) {
	f.steps = append(f.steps, "read-declarations")
	f.developerID, f.appID, f.account = developerID, appID, account
	if f.declarations == nil {
		return nil, errors.New("no fake declarations")
	}
	return f.declarations, nil
}

func (f *fakePublishBrowser) SetPrivacyPolicyURL(_ context.Context, developerID, appID, account, url string) (bool, error) {
	f.steps = append(f.steps, "set-privacy-url")
	f.setPrivacyURL = url
	if f.failAt == "set-privacy-url" {
		return false, errors.New("set privacy url failed")
	}
	return true, nil
}

func (f *fakePublishBrowser) SetRadioDeclaration(_ context.Context, developerID, appID, account, page string, yes bool) (bool, error) {
	f.steps = append(f.steps, "set-radio")
	f.setRadioPage, f.setRadioYes = page, yes
	if f.failAt == "set-radio" {
		return false, errors.New("set radio failed")
	}
	return true, nil
}

func (f *fakePublishBrowser) ReadPolicyStatus(context.Context, string, string, string) (*webdriver.PolicyStatus, error) {
	f.steps = append(f.steps, "read-policy")
	if f.policy == nil {
		return nil, errors.New("no fake policy status")
	}
	return f.policy, nil
}

func (f *fakePublishBrowser) SetManagedPublishing(_ context.Context, developerID, appID, account string, on bool) error {
	f.steps = append(f.steps, "set-managed")
	f.setManagedOn = on
	if f.failAt == "set-managed" {
		return errors.New("set managed failed")
	}
	return nil
}

func (f *fakePublishBrowser) PublishApprovedChanges(context.Context, string, string, string, time.Duration) error {
	f.steps = append(f.steps, "publish-now")
	if f.failAt == "publish-now" {
		return errors.New("publish failed")
	}
	return nil
}

func (f *fakePublishBrowser) OpenQuestionnaire(_ context.Context, developerID, appID, account, page string) error {
	f.steps = append(f.steps, "open-questionnaire:"+page)
	f.developerID, f.appID, f.account = developerID, appID, account
	if f.failAt == "open-questionnaire" {
		return errors.New("open questionnaire failed")
	}
	return nil
}

func (f *fakePublishBrowser) ReadQuestionnaireStep(context.Context) (*webdriver.QuestionnaireStep, error) {
	f.steps = append(f.steps, "read-step")
	if len(f.questSteps) == 0 {
		return nil, errors.New("no fake questionnaire step")
	}
	step := f.questSteps[0]
	f.questSteps = f.questSteps[1:]
	return &step, nil
}

func (f *fakePublishBrowser) SetStepChoices(_ context.Context, ids []string) error {
	f.steps = append(f.steps, "set-choices")
	f.setChoices = append(f.setChoices, ids)
	if f.failAt == "set-choices" {
		return errors.New("set choices failed")
	}
	return nil
}

func (f *fakePublishBrowser) QuestionnaireNext(context.Context) (bool, error) {
	f.steps = append(f.steps, "quest-next")
	return f.questHasNext, nil
}

func (f *fakePublishBrowser) QuestionnaireSave(context.Context, time.Duration) error {
	f.steps = append(f.steps, "quest-save")
	if f.failAt == "quest-save" {
		return errors.New("quest save failed")
	}
	return nil
}

func (f *fakePublishBrowser) QuestionnaireDiscard(context.Context) error {
	f.steps = append(f.steps, "quest-discard")
	return nil
}

func (f *fakePublishBrowser) ReadDistribution(_ context.Context, developerID, appID, account string) (*webdriver.DistributionState, error) {
	f.steps = append(f.steps, "read-distribution")
	f.developerID, f.appID, f.account = developerID, appID, account
	if len(f.distribution) == 0 {
		return nil, errors.New("no fake distribution state")
	}
	state := f.distribution[0]
	f.distribution = f.distribution[1:]
	return state, nil
}

func (f *fakePublishBrowser) AddFormFactor(_ context.Context, factor string) error {
	f.steps = append(f.steps, "add-factor")
	f.addedFactor = factor
	if f.failAt == "add-factor" {
		return errors.New("add factor failed")
	}
	return nil
}

func (f *fakePublishBrowser) ImportDataSafetyCSV(_ context.Context, developerID, appID, account, filePath string) error {
	f.steps = append(f.steps, "import-csv")
	f.importedCSV = filePath
	if f.failAt == "import-csv" {
		return errors.New("import failed")
	}
	return nil
}

func (f *fakePublishBrowser) Close() error { return nil }

func stubPublishBrowser(t *testing.T, f *fakePublishBrowser) {
	t.Helper()
	orig := newPublishBrowser
	newPublishBrowser = func(context.Context, string, time.Duration) (publishBrowser, error) { return f, nil }
	t.Cleanup(func() { newPublishBrowser = orig })
}

func setupPublish(t *testing.T, f *fakePublishBrowser) {
	t.Helper()
	useTempSessionDir(t)
	saveWebSession(t)
	mockWebClient(t, appsListMock(t, `{"1":[{"1":{"1":{"1":"`+authDeveloperID+`"},"2":{"1":"555"}},"2":"Aérocoach","5":"com.example.demo","16":"en-US"}]}`))
	stubPublishBrowser(t, f)
}

func availabilityArgs(extra ...string) []string {
	return append([]string{"--package", "com.example.demo"}, extra...)
}

// --- status ---

func TestWebAppsStatus_ValidatesPackage(t *testing.T) {
	cmd := WebAppsStatusCommand()
	if err := cmd.FlagSet.Parse(nil); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil || !strings.Contains(err.Error(), "--package") {
		t.Errorf("err = %v, want --package error", err)
	}
}

func TestWebAppsStatus_ReportsSetupAndPublishing(t *testing.T) {
	f := &fakePublishBrowser{
		setup: &webdriver.AppSetup{
			AppState: "draft",
			Goals: []webdriver.SetupGoal{
				{
					ID: "prod-goal", Title: "Create and publish a release", Complete: 1, Total: 5,
					PendingTasks: []string{"Select countries and regions"},
				},
			},
		},
		overviews: []*webdriver.PublishingOverview{{
			PendingChanges: []webdriver.PendingChange{
				{Section: "Store settings", Item: "App category", Description: "Select app category (Education app)"},
			},
			CanSendForReview:  false,
			SendBlockedReason: "To send changes for review, complete the required steps in the app dashboard",
		}},
	}
	setupPublish(t, f)

	cmd := WebAppsStatusCommand()
	if err := cmd.FlagSet.Parse(availabilityArgs()); err != nil {
		t.Fatal(err)
	}
	stdout, err := captureWebStdout(func() error { return cmd.Exec(context.Background(), nil) })
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	for _, want := range []string{
		`"appState":"draft"`, `"prod-goal"`, `"complete":1`, `"total":5`,
		"Select countries and regions", `"canSendForReview":false`, "complete the required steps",
	} {
		if !strings.Contains(stdout, want) {
			t.Errorf("output missing %q: %s", want, stdout)
		}
	}
	if got := strings.Join(f.steps, ","); got != "read-setup,read-overview" {
		t.Errorf("steps = %s, want read-setup,read-overview", got)
	}
	if f.developerID != authDeveloperID || f.appID != "555" || f.account != "me@example.com" {
		t.Errorf("resolved target = %s/%s@%s", f.developerID, f.appID, f.account)
	}
}

func TestWebAppsStatus_RejectsUnknownPackageBeforeBrowser(t *testing.T) {
	useTempSessionDir(t)
	saveWebSession(t)
	mockWebClient(t, appsListMock(t, `{"1":[]}`))
	f := &fakePublishBrowser{failAt: "read-overview"}
	stubPublishBrowser(t, f)

	cmd := WebAppsStatusCommand()
	if err := cmd.FlagSet.Parse(availabilityArgs()); err != nil {
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

// --- availability ---

func TestWebAppsAvailability_ValidatesFlags(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "package", args: []string{"--all-countries", "--confirm"}, want: "--package"},
		{name: "conflict", args: availabilityArgs("--countries", "Slovenia", "--all-countries", "--confirm"), want: "cannot be used together"},
		{name: "confirm", args: availabilityArgs("--all-countries"), want: "--confirm"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := WebAppsAvailabilityCommand()
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

func TestWebAppsAvailability_DryRunSkipsBrowser(t *testing.T) {
	f := &fakePublishBrowser{failAt: "open-countries"}
	stubPublishBrowser(t, f)

	cmd := WebAppsAvailabilityCommand()
	if err := cmd.FlagSet.Parse(availabilityArgs("--all-countries", "--confirm")); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Exec(shared.ContextWithDryRun(context.Background(), true), nil); err != nil {
		t.Fatalf("dry-run: %v", err)
	}
	if len(f.steps) != 0 {
		t.Errorf("browser steps = %v, want none", f.steps)
	}
}

func TestWebAppsAvailability_ReadOnlyNeedsNoConfirm(t *testing.T) {
	f := &fakePublishBrowser{
		countries: []webdriver.CountriesState{{Selected: []string{"Slovenia"}}},
	}
	setupPublish(t, f)

	cmd := WebAppsAvailabilityCommand()
	if err := cmd.FlagSet.Parse(availabilityArgs()); err != nil {
		t.Fatal(err)
	}
	stdout, err := captureWebStdout(func() error { return cmd.Exec(context.Background(), nil) })
	if err != nil {
		t.Fatalf("availability: %v", err)
	}
	if got := strings.Join(f.steps, ","); got != "open-countries,read-countries,discard" {
		t.Errorf("steps = %s, want open,read,discard", got)
	}
	for _, want := range []string{`"selectedCount":1`, `"changed":false`, "Slovenia", `"track":"production"`} {
		if !strings.Contains(stdout, want) {
			t.Errorf("output missing %q: %s", want, stdout)
		}
	}
}

func TestWebAppsAvailability_WritesAndVerifies(t *testing.T) {
	f := &fakePublishBrowser{
		countries: []webdriver.CountriesState{
			{Selected: nil},
			{Selected: []string{"Austria", "Slovenia"}, CanSubmit: true},
			{Selected: []string{"Austria", "Slovenia"}},
		},
	}
	setupPublish(t, f)

	cmd := WebAppsAvailabilityCommand()
	if err := cmd.FlagSet.Parse(availabilityArgs("--countries", "Slovenia,Austria", "--confirm")); err != nil {
		t.Fatal(err)
	}
	stdout, err := captureWebStdout(func() error { return cmd.Exec(context.Background(), nil) })
	if err != nil {
		t.Fatalf("availability: %v", err)
	}
	want := "open-countries,read-countries,set-countries,read-countries,submit-countries,open-countries,read-countries,discard"
	if got := strings.Join(f.steps, ","); got != want {
		t.Errorf("steps = %s, want %s", got, want)
	}
	if !slices.Equal(f.setNames, []string{"Slovenia", "Austria"}) {
		t.Errorf("set names = %v", f.setNames)
	}
	for _, wantOut := range []string{`"selectedCount":2`, `"changed":true`} {
		if !strings.Contains(stdout, wantOut) {
			t.Errorf("output missing %q: %s", wantOut, stdout)
		}
	}
}

func TestWebAppsAvailability_AllCountries(t *testing.T) {
	f := &fakePublishBrowser{
		countries: []webdriver.CountriesState{
			{Selected: []string{"Slovenia"}},
			{Selected: []string{"Austria", "Slovenia", "United States"}, CanSubmit: true},
			{Selected: []string{"Austria", "Slovenia", "United States"}},
		},
	}
	setupPublish(t, f)

	cmd := WebAppsAvailabilityCommand()
	if err := cmd.FlagSet.Parse(availabilityArgs("--all-countries", "--confirm")); err != nil {
		t.Fatal(err)
	}
	stdout, err := captureWebStdout(func() error { return cmd.Exec(context.Background(), nil) })
	if err != nil {
		t.Fatalf("availability: %v", err)
	}
	if !slices.Contains(f.steps, "set-all") {
		t.Errorf("steps = %v, want set-all", f.steps)
	}
	if slices.Contains(f.steps, "set-countries") {
		t.Errorf("steps = %v, must not set individual countries", f.steps)
	}
	if !strings.Contains(stdout, `"changed":true`) {
		t.Errorf("output = %s, want changed", stdout)
	}
}

func TestWebAppsAvailability_NoOpSkipsSubmit(t *testing.T) {
	f := &fakePublishBrowser{
		countries: []webdriver.CountriesState{
			{Selected: []string{"Slovenia"}},
			{Selected: []string{"Slovenia"}},
		},
	}
	setupPublish(t, f)

	cmd := WebAppsAvailabilityCommand()
	if err := cmd.FlagSet.Parse(availabilityArgs("--countries", "Slovenia", "--confirm")); err != nil {
		t.Fatal(err)
	}
	stdout, err := captureWebStdout(func() error { return cmd.Exec(context.Background(), nil) })
	if err != nil {
		t.Fatalf("availability: %v", err)
	}
	if slices.Contains(f.steps, "submit-countries") {
		t.Errorf("steps = %v, must not submit a no-op", f.steps)
	}
	if !strings.Contains(stdout, `"changed":false`) {
		t.Errorf("output = %s, want unchanged", stdout)
	}
}

func TestWebAppsAvailability_RefusesUnsubmittableForm(t *testing.T) {
	f := &fakePublishBrowser{
		countries: []webdriver.CountriesState{
			{Selected: nil},
			{Selected: []string{"Slovenia"}, CanSubmit: false},
		},
	}
	setupPublish(t, f)

	cmd := WebAppsAvailabilityCommand()
	if err := cmd.FlagSet.Parse(availabilityArgs("--countries", "Slovenia", "--confirm")); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil || !strings.Contains(err.Error(), "not ready to save") {
		t.Errorf("err = %v, want not-ready error", err)
	}
	if slices.Contains(f.steps, "submit-countries") {
		t.Error("must not submit an unsubmittable form")
	}
}

func TestWebAppsAvailability_VerifiesSavedSelection(t *testing.T) {
	f := &fakePublishBrowser{
		countries: []webdriver.CountriesState{
			{Selected: nil},
			{Selected: []string{"Slovenia"}, CanSubmit: true},
			{Selected: nil},
		},
	}
	setupPublish(t, f)

	cmd := WebAppsAvailabilityCommand()
	if err := cmd.FlagSet.Parse(availabilityArgs("--countries", "Slovenia", "--confirm")); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil || !strings.Contains(err.Error(), "was not saved") {
		t.Errorf("err = %v, want post-save verification error", err)
	}
}

func TestWebAppsAvailability_RejectsUnknownPackageBeforeBrowser(t *testing.T) {
	useTempSessionDir(t)
	saveWebSession(t)
	mockWebClient(t, appsListMock(t, `{"1":[]}`))
	f := &fakePublishBrowser{failAt: "open-countries"}
	stubPublishBrowser(t, f)

	cmd := WebAppsAvailabilityCommand()
	if err := cmd.FlagSet.Parse(availabilityArgs("--all-countries", "--confirm")); err != nil {
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

// --- review ---

func reviewArgs() []string {
	return []string{"--package", "com.example.demo", "--confirm"}
}

func TestWebAppsReview_ValidatesFlags(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "package", args: []string{"--confirm"}, want: "--package"},
		{name: "confirm", args: []string{"--package", "com.example"}, want: "--confirm"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := WebAppsReviewCommand()
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

func TestWebAppsReview_DryRunSkipsBrowser(t *testing.T) {
	f := &fakePublishBrowser{failAt: "read-overview"}
	stubPublishBrowser(t, f)

	cmd := WebAppsReviewCommand()
	if err := cmd.FlagSet.Parse(reviewArgs()); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Exec(shared.ContextWithDryRun(context.Background(), true), nil); err != nil {
		t.Fatalf("dry-run: %v", err)
	}
	if len(f.steps) != 0 {
		t.Errorf("browser steps = %v, want none", f.steps)
	}
}

func TestWebAppsReview_RefusesWhenNothingPending(t *testing.T) {
	f := &fakePublishBrowser{overviews: []*webdriver.PublishingOverview{{}}}
	setupPublish(t, f)

	cmd := WebAppsReviewCommand()
	if err := cmd.FlagSet.Parse(reviewArgs()); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil || !strings.Contains(err.Error(), "no pending changes") {
		t.Errorf("err = %v, want nothing-pending error", err)
	}
	if slices.Contains(f.steps, "send") {
		t.Error("must not send when nothing is pending")
	}
}

func TestWebAppsReview_RefusesWhenConsoleNotReady(t *testing.T) {
	f := &fakePublishBrowser{overviews: []*webdriver.PublishingOverview{{
		PendingChanges:    []webdriver.PendingChange{{Item: "App category"}},
		CanSendForReview:  false,
		SendBlockedReason: "complete the required steps in the app dashboard",
	}}}
	setupPublish(t, f)

	cmd := WebAppsReviewCommand()
	if err := cmd.FlagSet.Parse(reviewArgs()); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil || !strings.Contains(err.Error(), "complete the required steps") {
		t.Errorf("err = %v, want the console's blocked reason", err)
	}
	if slices.Contains(f.steps, "send") {
		t.Error("must not send when the console blocks review")
	}
}

func TestWebAppsReview_SendsAndVerifies(t *testing.T) {
	f := &fakePublishBrowser{overviews: []*webdriver.PublishingOverview{
		{
			PendingChanges: []webdriver.PendingChange{
				{Section: "Store settings", Item: "App category"},
				{Section: "Countries / regions", Item: "Countries / regions"},
			},
			CanSendForReview: true,
		},
		{},
	}}
	setupPublish(t, f)

	cmd := WebAppsReviewCommand()
	if err := cmd.FlagSet.Parse(reviewArgs()); err != nil {
		t.Fatal(err)
	}
	stdout, err := captureWebStdout(func() error { return cmd.Exec(context.Background(), nil) })
	if err != nil {
		t.Fatalf("review: %v", err)
	}
	if got := strings.Join(f.steps, ","); got != "read-overview,send,read-overview" {
		t.Errorf("steps = %s, want read,send,read", got)
	}
	for _, want := range []string{`"sentCount":2`, "App category", "Countries / regions"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("output missing %q: %s", want, stdout)
		}
	}
}

func TestWebAppsReview_VerifiesSent(t *testing.T) {
	f := &fakePublishBrowser{overviews: []*webdriver.PublishingOverview{
		{
			PendingChanges:   []webdriver.PendingChange{{Item: "App category"}},
			CanSendForReview: true,
		},
		{PendingChanges: []webdriver.PendingChange{{Item: "App category"}}},
	}}
	setupPublish(t, f)

	cmd := WebAppsReviewCommand()
	if err := cmd.FlagSet.Parse(reviewArgs()); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil || !strings.Contains(err.Error(), "were not sent") {
		t.Errorf("err = %v, want post-send verification error", err)
	}
}

// --- pricing ---

func pricingArgs(extra ...string) []string {
	return append([]string{"--package", "com.example.demo"}, extra...)
}

func TestWebAppsPricing_ValidatesFlags(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "package", args: []string{"--price", "69.99", "--confirm"}, want: "--package"},
		{name: "price format", args: pricingArgs("--price", "€69.99", "--confirm"), want: "--price must be a plain amount"},
		{name: "confirm", args: pricingArgs("--price", "69.99"), want: "--confirm"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := WebAppsPricingCommand()
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

func TestWebAppsPricing_DryRunSkipsBrowser(t *testing.T) {
	f := &fakePublishBrowser{failAt: "open-pricing"}
	stubPublishBrowser(t, f)

	cmd := WebAppsPricingCommand()
	if err := cmd.FlagSet.Parse(pricingArgs("--price", "69.99", "--confirm")); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Exec(shared.ContextWithDryRun(context.Background(), true), nil); err != nil {
		t.Fatalf("dry-run: %v", err)
	}
	if len(f.steps) != 0 {
		t.Errorf("browser steps = %v, want none", f.steps)
	}
}

func TestWebAppsPricing_ReadOnlyNeedsNoConfirm(t *testing.T) {
	f := &fakePublishBrowser{
		pricing: []webdriver.AppPricingState{{PriceType: "paid", PricesSet: false}},
	}
	setupPublish(t, f)

	cmd := WebAppsPricingCommand()
	if err := cmd.FlagSet.Parse(pricingArgs()); err != nil {
		t.Fatal(err)
	}
	stdout, err := captureWebStdout(func() error { return cmd.Exec(context.Background(), nil) })
	if err != nil {
		t.Fatalf("pricing: %v", err)
	}
	if got := strings.Join(f.steps, ","); got != "open-pricing,read-pricing,discard-pricing" {
		t.Errorf("steps = %s, want open,read,discard", got)
	}
	for _, want := range []string{`"priceType":"paid"`, `"pricesSet":false`, `"changed":false`} {
		if !strings.Contains(stdout, want) {
			t.Errorf("output missing %q: %s", want, stdout)
		}
	}
}

func TestWebAppsPricing_WritesAndVerifies(t *testing.T) {
	f := &fakePublishBrowser{
		pricing: []webdriver.AppPricingState{
			{PriceType: "paid"},
			{PriceType: "paid", StagedChanges: true, CanSave: true},
			{PriceType: "paid", PricesSet: true},
		},
	}
	setupPublish(t, f)

	cmd := WebAppsPricingCommand()
	if err := cmd.FlagSet.Parse(pricingArgs("--price", "69.99", "--confirm")); err != nil {
		t.Fatal(err)
	}
	stdout, err := captureWebStdout(func() error { return cmd.Exec(context.Background(), nil) })
	if err != nil {
		t.Fatalf("pricing: %v", err)
	}
	want := "open-pricing,read-pricing,set-price,read-pricing,submit-pricing,open-pricing,read-pricing,discard-pricing"
	if got := strings.Join(f.steps, ","); got != want {
		t.Errorf("steps = %s, want %s", got, want)
	}
	if f.setPrice != "69.99" {
		t.Errorf("set price = %q", f.setPrice)
	}
	if !strings.Contains(stdout, `"changed":true`) {
		t.Errorf("output = %s, want changed", stdout)
	}
}

func TestWebAppsPricing_RefusesFreeApp(t *testing.T) {
	f := &fakePublishBrowser{
		pricing: []webdriver.AppPricingState{{PriceType: "free"}},
	}
	setupPublish(t, f)

	cmd := WebAppsPricingCommand()
	if err := cmd.FlagSet.Parse(pricingArgs("--price", "69.99", "--confirm")); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil || !strings.Contains(err.Error(), "not set to Paid") {
		t.Errorf("err = %v, want not-paid error", err)
	}
	if slices.Contains(f.steps, "set-price") {
		t.Error("must not run the pricing wizard for a free app")
	}
}

func TestWebAppsPricing_RefusesUnsavableForm(t *testing.T) {
	f := &fakePublishBrowser{
		pricing: []webdriver.AppPricingState{
			{PriceType: "paid"},
			{PriceType: "paid", StagedChanges: true, CanSave: false},
		},
	}
	setupPublish(t, f)

	cmd := WebAppsPricingCommand()
	if err := cmd.FlagSet.Parse(pricingArgs("--price", "69.99", "--confirm")); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil || !strings.Contains(err.Error(), "not ready to save") {
		t.Errorf("err = %v, want not-ready error", err)
	}
	if slices.Contains(f.steps, "submit-pricing") {
		t.Error("must not submit an unsavable form")
	}
}

func TestWebAppsPricing_VerifiesSavedPrice(t *testing.T) {
	f := &fakePublishBrowser{
		pricing: []webdriver.AppPricingState{
			{PriceType: "paid"},
			{PriceType: "paid", StagedChanges: true, CanSave: true},
			{PriceType: "paid", StagedChanges: true},
		},
	}
	setupPublish(t, f)

	cmd := WebAppsPricingCommand()
	if err := cmd.FlagSet.Parse(pricingArgs("--price", "69.99", "--confirm")); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil || !strings.Contains(err.Error(), "was not saved") {
		t.Errorf("err = %v, want post-save verification error", err)
	}
}

func TestWebAppsPricing_RejectsUnknownPackageBeforeBrowser(t *testing.T) {
	useTempSessionDir(t)
	saveWebSession(t)
	mockWebClient(t, appsListMock(t, `{"1":[]}`))
	f := &fakePublishBrowser{failAt: "open-pricing"}
	stubPublishBrowser(t, f)

	cmd := WebAppsPricingCommand()
	if err := cmd.FlagSet.Parse(pricingArgs("--price", "69.99", "--confirm")); err != nil {
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

func TestWebAppsAvailability_RemovesUndesiredCountries(t *testing.T) {
	f := &fakePublishBrowser{
		countries: []webdriver.CountriesState{
			{Selected: []string{"Slovenia", "Sudan"}},
			{Selected: []string{"Slovenia"}, CanSubmit: true},
			{Selected: []string{"Slovenia"}},
		},
	}
	setupPublish(t, f)

	cmd := WebAppsAvailabilityCommand()
	if err := cmd.FlagSet.Parse(availabilityArgs("--countries", "Slovenia", "--confirm")); err != nil {
		t.Fatal(err)
	}
	stdout, err := captureWebStdout(func() error { return cmd.Exec(context.Background(), nil) })
	if err != nil {
		t.Fatalf("availability: %v", err)
	}
	if slices.Contains(f.steps, "set-countries") {
		t.Errorf("steps = %v, nothing should be added", f.steps)
	}
	if !slices.Equal(f.unsetNames, []string{"Sudan"}) {
		t.Errorf("unset names = %v, want [Sudan]", f.unsetNames)
	}
	if !strings.Contains(stdout, `"changed":true`) {
		t.Errorf("output = %s, want changed", stdout)
	}
}

func TestWebAppsAvailability_ErrorsWhenRemovalDoesNotStick(t *testing.T) {
	f := &fakePublishBrowser{
		countries: []webdriver.CountriesState{
			{Selected: []string{"Slovenia", "Sudan"}},
			{Selected: []string{"Slovenia", "Sudan"}},
		},
	}
	setupPublish(t, f)

	cmd := WebAppsAvailabilityCommand()
	if err := cmd.FlagSet.Parse(availabilityArgs("--countries", "Slovenia", "--confirm")); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil || !strings.Contains(err.Error(), "still targeted") {
		t.Errorf("err = %v, want still-targeted error", err)
	}
	if slices.Contains(f.steps, "submit-countries") {
		t.Error("must not submit an unverified selection")
	}
}

// --- rollout ---

func rolloutArgs() []string {
	return []string{"--package", "com.example.demo", "--confirm"}
}

func TestWebAppsRollout_ValidatesFlags(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "package", args: []string{"--confirm"}, want: "--package"},
		{name: "confirm", args: []string{"--package", "com.example"}, want: "--confirm"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := WebAppsRolloutCommand()
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

func TestWebAppsRollout_DryRunSkipsBrowser(t *testing.T) {
	f := &fakePublishBrowser{failAt: "open-releases"}
	stubPublishBrowser(t, f)

	cmd := WebAppsRolloutCommand()
	if err := cmd.FlagSet.Parse(rolloutArgs()); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Exec(shared.ContextWithDryRun(context.Background(), true), nil); err != nil {
		t.Fatalf("dry-run: %v", err)
	}
	if len(f.steps) != 0 {
		t.Errorf("browser steps = %v, want none", f.steps)
	}
}

func TestWebAppsRollout_RollsOutAndSendsForReview(t *testing.T) {
	f := &fakePublishBrowser{
		prepare: &webdriver.PrepareState{ReleaseName: "1.2.1"},
		review:  &webdriver.ReviewState{Warnings: []string{"no deobfuscation file"}, CanSave: true},
		overviews: []*webdriver.PublishingOverview{
			{PendingChanges: []webdriver.PendingChange{{Item: "Releases", Description: "1.2.1"}}, CanSendForReview: true},
			{},
		},
	}
	setupPublish(t, f)

	cmd := WebAppsRolloutCommand()
	if err := cmd.FlagSet.Parse(rolloutArgs()); err != nil {
		t.Fatal(err)
	}
	stdout, err := captureWebStdout(func() error { return cmd.Exec(context.Background(), nil) })
	if err != nil {
		t.Fatalf("rollout: %v", err)
	}
	want := "open-releases,open-draft,review-draft,save-review,read-overview,send,read-overview"
	if got := strings.Join(f.steps, ","); got != want {
		t.Errorf("steps = %s, want %s", got, want)
	}
	for _, wantOut := range []string{`"releaseName":"1.2.1"`, `"sentForReview":true`, "deobfuscation", `"pendingChanges":0`} {
		if !strings.Contains(stdout, wantOut) {
			t.Errorf("output missing %q: %s", wantOut, stdout)
		}
	}
}

func TestWebAppsRollout_RefusesUnsavableReview(t *testing.T) {
	f := &fakePublishBrowser{
		prepare: &webdriver.PrepareState{ReleaseName: "1.2.1"},
		review:  &webdriver.ReviewState{CanSave: false},
	}
	setupPublish(t, f)

	cmd := WebAppsRolloutCommand()
	if err := cmd.FlagSet.Parse(rolloutArgs()); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil || !strings.Contains(err.Error(), "not ready to save") {
		t.Errorf("err = %v, want not-ready error", err)
	}
	if slices.Contains(f.steps, "save-review") {
		t.Error("must not save an unready review page")
	}
}

func TestWebAppsRollout_ReportsBlockedSend(t *testing.T) {
	f := &fakePublishBrowser{
		prepare: &webdriver.PrepareState{ReleaseName: "1.2.1"},
		review:  &webdriver.ReviewState{CanSave: true},
		overviews: []*webdriver.PublishingOverview{{
			PendingChanges:    []webdriver.PendingChange{{Item: "Releases"}},
			CanSendForReview:  false,
			SendBlockedReason: "complete the required steps in the app dashboard",
		}},
	}
	setupPublish(t, f)

	cmd := WebAppsRolloutCommand()
	if err := cmd.FlagSet.Parse(rolloutArgs()); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil || !strings.Contains(err.Error(), "staged in the Publishing overview") {
		t.Errorf("err = %v, want staged-but-blocked error", err)
	}
	if slices.Contains(f.steps, "send") {
		t.Error("must not send when the console blocks review")
	}
}

func TestWebAppsRollout_RejectsUnknownPackageBeforeBrowser(t *testing.T) {
	useTempSessionDir(t)
	saveWebSession(t)
	mockWebClient(t, appsListMock(t, `{"1":[]}`))
	f := &fakePublishBrowser{failAt: "open-releases"}
	stubPublishBrowser(t, f)

	cmd := WebAppsRolloutCommand()
	if err := cmd.FlagSet.Parse(rolloutArgs()); err != nil {
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

func TestWebAppsReview_UsesSummaryCountWhenTableEmpty(t *testing.T) {
	f := &fakePublishBrowser{overviews: []*webdriver.PublishingOverview{
		{CanSendForReview: true, SummaryPendingCount: 12},
		{},
	}}
	setupPublish(t, f)

	cmd := WebAppsReviewCommand()
	if err := cmd.FlagSet.Parse(reviewArgs()); err != nil {
		t.Fatal(err)
	}
	stdout, err := captureWebStdout(func() error { return cmd.Exec(context.Background(), nil) })
	if err != nil {
		t.Fatalf("review: %v", err)
	}
	if !slices.Contains(f.steps, "send") {
		t.Errorf("steps = %v, want send", f.steps)
	}
	if !strings.Contains(stdout, `"sentCount":12`) {
		t.Errorf("output = %s, want sentCount 12", stdout)
	}
}
