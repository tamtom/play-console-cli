package web

import (
	"context"
	"strings"
	"testing"

	"github.com/tamtom/play-console-cli/internal/webdriver"
)

func TestWebAppsRating_ReadsDetailedStatus(t *testing.T) {
	useTempSessionDir(t)
	saveWebSession(t)
	mockWebClient(t, appsListMock(t, `{"1":[{"1":{"1":{"1":"`+authDeveloperID+`"},"2":{"1":"555"}},"2":"Aérocoach","5":"com.example.demo","16":"en-US"}]}`))
	original := runContentRating
	runContentRating = func(_ context.Context, _, developerID, appID, account string) (*webdriver.ContentRatingState, error) {
		if developerID != authDeveloperID || appID != "555" || account != "me@example.com" {
			t.Fatalf("unexpected runner args: %q %q %q", developerID, appID, account)
		}
		return &webdriver.ContentRatingState{
			IARCStatus:    "Completed",
			EmailAddress:  "info@example.com",
			CertificateID: "cert-123",
			SubmittedAt:   "July 29, 2026, 2:20 PM",
			DraftStatus:   "In progress",
			Ratings:       []webdriver.AppliedRating{{Authority: "ESRB", Rating: "Everyone"}},
		}, nil
	}
	t.Cleanup(func() { runContentRating = original })

	cmd := WebAppsRatingCommand()
	if err := cmd.FlagSet.Parse(availabilityArgs()); err != nil {
		t.Fatal(err)
	}
	stdout, err := captureWebStdout(func() error { return cmd.Exec(context.Background(), nil) })
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"com.example.demo", "Completed", "cert-123", "In progress", "Everyone"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("output = %s, want %q", stdout, want)
		}
	}
}

func TestWebAppsRating_ValidatesPackage(t *testing.T) {
	cmd := WebAppsRatingCommand()
	if err := cmd.FlagSet.Parse(nil); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil || !strings.Contains(err.Error(), "--package") {
		t.Errorf("err = %v, want --package error", err)
	}
}
