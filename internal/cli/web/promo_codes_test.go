package web

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/tamtom/play-console-cli/internal/cli/shared"
	"github.com/tamtom/play-console-cli/internal/testutil"
	"github.com/tamtom/play-console-cli/internal/webdriver"
)

func stubPromoCodes(t *testing.T, fn promoCodesRunner) {
	t.Helper()
	original := runPromoCodes
	runPromoCodes = fn
	t.Cleanup(func() { runPromoCodes = original })
}

func setupPromoCodes(t *testing.T, fn promoCodesRunner) {
	t.Helper()
	useTempSessionDir(t)
	saveWebSession(t)
	mockWebClient(t, promoCodesMock(t, true))
	stubPromoCodes(t, fn)
}

func promoCodesMock(t *testing.T, accepted bool) *testutil.MockAPI {
	t.Helper()
	termsPayload := `{"7":[{"1":9,"2":0,"3":1}]}`
	if accepted {
		termsPayload = `{"7":[{"1":9,"2":1,"3":1}]}`
	}
	return testutil.NewMockAPI(t, map[string]http.HandlerFunc{
		"GET /console": func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, authConsolePath, http.StatusFound)
		},
		"GET " + authConsolePath: func(w http.ResponseWriter, _ *http.Request) {
			_, _ = io.WriteString(w, authConsoleHTML)
		},
		"GET /v1/developers/" + authDeveloperID + "/appSummaries": func(w http.ResponseWriter, _ *http.Request) {
			_, _ = io.WriteString(w, `{"1":[{"1":{"1":{"1":"`+authDeveloperID+`"},"2":{"1":"555"}},"2":"Aérocoach","5":"com.example.demo","16":"en-US"}]}`)
		},
		"GET /v1/developers/" + authDeveloperID + "/developersummaries": func(w http.ResponseWriter, _ *http.Request) {
			_, _ = io.WriteString(w, termsPayload)
		},
	})
}

func TestWebAppsPromoCodes_ValidatesCreate(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "package", args: nil, want: "--package"},
		{name: "name", args: availabilityArgs("--create", "--start-date", "2026-08-02", "--end-date", "2026-08-03", "--code-count", "10", "--confirm"), want: "--name"},
		{name: "name length", args: availabilityArgs("--create", "--name", strings.Repeat("x", 61), "--start-date", "2026-08-02", "--end-date", "2026-08-03", "--code-count", "10", "--confirm"), want: "60 characters"},
		{name: "date format", args: availabilityArgs("--create", "--name", "Launch", "--start-date", "08/02/2026", "--end-date", "2026-08-03", "--code-count", "10", "--confirm"), want: "YYYY-MM-DD"},
		{name: "date order", args: availabilityArgs("--create", "--name", "Launch", "--start-date", "2026-08-03", "--end-date", "2026-08-02", "--code-count", "10", "--confirm"), want: "before"},
		{name: "one year", args: availabilityArgs("--create", "--name", "Launch", "--start-date", "2026-08-02", "--end-date", "2027-08-03", "--code-count", "10", "--confirm"), want: "one year"},
		{name: "count", args: availabilityArgs("--create", "--name", "Launch", "--start-date", "2026-08-02", "--end-date", "2026-08-03", "--code-count", "501", "--confirm"), want: "1 and 500"},
		{name: "confirm", args: availabilityArgs("--create", "--name", "Launch", "--start-date", "2026-08-02", "--end-date", "2026-08-03", "--code-count", "10"), want: "--confirm"},
		{name: "create required", args: availabilityArgs("--name", "Launch"), want: "--create"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := WebAppsPromoCodesCommand()
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

func TestWebAppsPromoCodes_ReadsEmptyList(t *testing.T) {
	setupPromoCodes(t, func(_ context.Context, _, developerID, appID, account string, termsAccepted bool, form *webdriver.PaidAppPromotionForm) (*webdriver.PromotionsState, error) {
		if developerID != authDeveloperID || appID != "555" || account != "me@example.com" || !termsAccepted || form != nil {
			t.Fatalf("unexpected runner args: developer=%q app=%q account=%q accepted=%v form=%+v", developerID, appID, account, termsAccepted, form)
		}
		return &webdriver.PromotionsState{Promotions: []webdriver.Promotion{}}, nil
	})

	cmd := WebAppsPromoCodesCommand()
	if err := cmd.FlagSet.Parse(availabilityArgs()); err != nil {
		t.Fatal(err)
	}
	stdout, err := captureWebStdout(func() error { return cmd.Exec(context.Background(), nil) })
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"promotions":[]`, `"termsAcceptanceRequired":false`, `"changed":false`} {
		if !strings.Contains(stdout, want) {
			t.Errorf("output = %s, want %s", stdout, want)
		}
	}
}

func TestWebAppsPromoCodes_CreatesPaidAppPromotion(t *testing.T) {
	setupPromoCodes(t, func(_ context.Context, _, _, _, _ string, termsAccepted bool, form *webdriver.PaidAppPromotionForm) (*webdriver.PromotionsState, error) {
		if !termsAccepted || form == nil || form.Name != "Launch" || form.StartDate != "2026-08-02" || form.EndDate != "2026-08-03" || form.CodeCount != 10 {
			t.Fatalf("form = %+v", form)
		}
		return &webdriver.PromotionsState{Promotions: []webdriver.Promotion{{Name: "Launch", Type: "Paid app", Status: "Scheduled"}}}, nil
	})

	cmd := WebAppsPromoCodesCommand()
	args := availabilityArgs("--create", "--name", "Launch", "--start-date", "2026-08-02", "--end-date", "2026-08-03", "--code-count", "10", "--confirm")
	if err := cmd.FlagSet.Parse(args); err != nil {
		t.Fatal(err)
	}
	stdout, err := captureWebStdout(func() error { return cmd.Exec(context.Background(), nil) })
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout, `"created":"Launch"`) || !strings.Contains(stdout, `"changed":true`) {
		t.Errorf("output = %s", stdout)
	}
}

func TestWebAppsPromoCodes_FailsClosedWithoutAcceptedTerms(t *testing.T) {
	useTempSessionDir(t)
	saveWebSession(t)
	mockWebClient(t, promoCodesMock(t, false))
	called := false
	stubPromoCodes(t, func(context.Context, string, string, string, string, bool, *webdriver.PaidAppPromotionForm) (*webdriver.PromotionsState, error) {
		called = true
		return nil, errors.New("should not run")
	})

	cmd := WebAppsPromoCodesCommand()
	args := availabilityArgs("--create", "--name", "Launch", "--start-date", "2026-08-02", "--end-date", "2026-08-03", "--code-count", "10", "--confirm")
	if err := cmd.FlagSet.Parse(args); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil || !strings.Contains(err.Error(), "accepted manually") {
		t.Fatalf("err = %v", err)
	}
	if called {
		t.Fatal("browser runner called without positive Terms acceptance")
	}
}

func TestWebAppsPromoCodes_DryRunDoesNotOpenBrowser(t *testing.T) {
	called := false
	stubPromoCodes(t, func(context.Context, string, string, string, string, bool, *webdriver.PaidAppPromotionForm) (*webdriver.PromotionsState, error) {
		called = true
		return nil, errors.New("should not run")
	})
	cmd := WebAppsPromoCodesCommand()
	args := availabilityArgs("--create", "--name", "Launch", "--start-date", "2026-08-02", "--end-date", "2026-08-03", "--code-count", "10", "--confirm")
	if err := cmd.FlagSet.Parse(args); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Exec(shared.ContextWithDryRun(context.Background(), true), nil); err != nil {
		t.Fatal(err)
	}
	if called {
		t.Fatal("browser runner called during dry run")
	}
}
