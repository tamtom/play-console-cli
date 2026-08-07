package webdriver

import (
	"context"
	"strings"
	"testing"
	"time"
)

// Placeholder identities shared by every test in this package; never real
// account data.
const (
	publishingTestDeveloper = "1234567890"
	publishingTestApp       = "9876543210"
	testAccount             = "me@example.com"
)

func TestPublishingOverviewURL(t *testing.T) {
	got := publishingOverviewURL(publishingTestDeveloper, publishingTestApp, testAccount)
	want := "https://play.google.com/console/developers/1234567890/app/9876543210/publishing?authuser=me%40example.com&hl=en"
	if got != want {
		t.Errorf("url = %q, want %q", got, want)
	}
}

func TestAppDashboardURL(t *testing.T) {
	got := appDashboardURL(publishingTestDeveloper, publishingTestApp, "")
	want := "https://play.google.com/console/developers/1234567890/app/9876543210/app-dashboard"
	if got != want {
		t.Errorf("url = %q, want %q", got, want)
	}
}

func connectFake(t *testing.T, f *fakeChrome) *Browser {
	t.Helper()
	b, err := Connect(context.Background(), portDir(t, f), 2*time.Second)
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	t.Cleanup(func() { _ = b.Close() }) //nolint:errcheck // test teardown
	return b
}

func TestReadPublishingOverview(t *testing.T) {
	f := newFakeChrome(t)
	f.setReply(publishingOverviewReadyWait, true)
	f.setReply(readPublishingOverviewScript, map[string]any{
		"pendingChanges": []map[string]any{
			{"section": "Store settings", "item": "App category", "description": "Select app category (Education app)"},
			{"section": "", "item": "Countries / regions", "description": "Add 176 countries / regions"},
		},
		"canSendForReview":  false,
		"sendBlockedReason": "To send changes for review, complete the required steps in the app dashboard",
		"managedPublishing": false,
	})
	b := connectFake(t, f)

	overview, err := ReadPublishingOverview(context.Background(), b, publishingTestDeveloper, publishingTestApp, testAccount)
	if err != nil {
		t.Fatalf("ReadPublishingOverview: %v", err)
	}
	if len(overview.PendingChanges) != 2 {
		t.Fatalf("pending changes = %+v", overview.PendingChanges)
	}
	first := overview.PendingChanges[0]
	if first.Section != "Store settings" || first.Item != "App category" || first.Description == "" {
		t.Errorf("first change = %+v", first)
	}
	if overview.CanSendForReview {
		t.Error("CanSendForReview = true, want false")
	}
	if overview.SendBlockedReason == "" {
		t.Error("SendBlockedReason empty, want the dashboard-steps message")
	}
	if reads := countSeen(f, readPublishingOverviewScript); reads != 1 {
		t.Errorf("read script ran %d times, want 1 for a settled page", reads)
	}
}

func TestReadPublishingOverview_ReadsInReviewState(t *testing.T) {
	f := newFakeChrome(t)
	f.setReply(publishingOverviewReadyWait, true)
	f.setReply(publishingOverviewSettledWait, true)
	f.setReply(readPublishingOverviewScript, map[string]any{
		"pendingChanges": []map[string]any{},
		"inReview":       true,
	})
	b := connectFake(t, f)

	overview, err := ReadPublishingOverview(context.Background(), b, publishingTestDeveloper, publishingTestApp, testAccount)
	if err != nil {
		t.Fatalf("ReadPublishingOverview: %v", err)
	}
	if !overview.InReview {
		t.Error("InReview = false, want true")
	}
}

func countSeen(f *fakeChrome, expr string) int {
	n := 0
	for _, seen := range f.seen {
		if seen == expr {
			n++
		}
	}
	return n
}

func TestReadPublishingOverview_ReReadsWhenSendStateUnsettled(t *testing.T) {
	f := newFakeChrome(t)
	f.setReply(publishingOverviewReadyWait, true)
	f.setReply(publishingOverviewSettledWait, true)
	// Both reads answer "disabled, no reason": the command must wait for the
	// settled state and read again rather than trusting the first answer.
	f.setReply(readPublishingOverviewScript, map[string]any{
		"pendingChanges":   []map[string]any{},
		"canSendForReview": false,
	})
	b := connectFake(t, f)

	if _, err := ReadPublishingOverview(context.Background(), b, publishingTestDeveloper, publishingTestApp, testAccount); err != nil {
		t.Fatalf("ReadPublishingOverview: %v", err)
	}
	if reads := countSeen(f, readPublishingOverviewScript); reads != 2 {
		t.Errorf("read script ran %d times, want 2 (initial + settled re-read)", reads)
	}
}

func TestReadPublishingOverview_RequiresIDs(t *testing.T) {
	if _, err := ReadPublishingOverview(context.Background(), nil, "", "app", ""); err == nil {
		t.Error("want an error when developer ID is empty")
	}
}

func TestReadAppSetup(t *testing.T) {
	f := newFakeChrome(t)
	f.setReply(appDashboardReadyScript, true)
	f.setReply(readAppSetupScript, map[string]any{
		"appState": "draft",
		"goals": []map[string]any{
			{
				"id": "basic-info-goal", "title": "Set up your app", "complete": 12, "total": 13,
				"pendingTasks": []string{"Set the price of your app"},
			},
			{
				"id": "prod-goal", "title": "Create and publish a release", "complete": 1, "total": 5,
				"pendingTasks": []string{"Select countries and regions", "Create a new release"},
			},
		},
	})
	b := connectFake(t, f)

	setup, err := ReadAppSetup(context.Background(), b, publishingTestDeveloper, publishingTestApp, testAccount)
	if err != nil {
		t.Fatalf("ReadAppSetup: %v", err)
	}
	if setup.AppState != "draft" {
		t.Errorf("appState = %q, want draft", setup.AppState)
	}
	if len(setup.Goals) != 2 {
		t.Fatalf("goals = %+v", setup.Goals)
	}
	goal := setup.Goals[1]
	if goal.ID != "prod-goal" || goal.Complete != 1 || goal.Total != 5 {
		t.Errorf("goal = %+v", goal)
	}
	if len(goal.PendingTasks) != 2 || goal.PendingTasks[0] != "Select countries and regions" {
		t.Errorf("pendingTasks = %v", goal.PendingTasks)
	}
}

func TestSendForReview_ConfirmsDialog(t *testing.T) {
	f := newFakeChrome(t)
	f.setReply(sendForReviewClickWait, true)
	f.setReply(sendForReviewDialogPresentScript, true)
	f.setReply(sendForReviewConfirmScript, "confirmed")
	f.setReply(sendForReviewSettledScript, true)
	b := connectFake(t, f)

	if err := SendForReview(context.Background(), b, 2*time.Second); err != nil {
		t.Fatalf("SendForReview: %v", err)
	}
}

func TestSendForReview_SettlesWithoutDialog(t *testing.T) {
	f := newFakeChrome(t)
	f.setReply(sendForReviewClickWait, true)
	// No dialog: the page settles, so the present-probe turns true on its own.
	f.setReply(sendForReviewDialogPresentScript, true)
	f.setReply(sendForReviewConfirmScript, "none")
	f.setReply(sendForReviewSettledScript, true)
	b := connectFake(t, f)

	if err := SendForReview(context.Background(), b, 2*time.Second); err != nil {
		t.Fatalf("SendForReview: %v", err)
	}
}

func TestSendForReview_RefusesDisabledButton(t *testing.T) {
	f := newFakeChrome(t)
	f.setReply(sendForReviewClickWait, false)
	b := connectFake(t, f)

	err := SendForReview(context.Background(), b, 300*time.Millisecond)
	if err == nil || !strings.Contains(err.Error(), "missing or disabled") {
		t.Errorf("err = %v, want disabled-button error", err)
	}
}

func TestSendForReview_ErrorsOnUnknownDialog(t *testing.T) {
	f := newFakeChrome(t)
	f.setReply(sendForReviewClickWait, true)
	f.setReply(sendForReviewDialogPresentScript, true)
	f.setReply(sendForReviewConfirmScript, "no-button")
	b := connectFake(t, f)

	err := SendForReview(context.Background(), b, 300*time.Millisecond)
	if err == nil || !strings.Contains(err.Error(), "no recognizable confirm button") {
		t.Errorf("err = %v, want unknown-dialog error", err)
	}
}
