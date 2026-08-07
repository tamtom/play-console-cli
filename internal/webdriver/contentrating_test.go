package webdriver

import (
	"context"
	"testing"
)

func TestNormalizeIARCStatus(t *testing.T) {
	for input, want := range map[string]string{
		"check_circle View details": "Completed",
		"check_circle Completed":    "Completed",
		"timelapse In progress":     "In progress",
		"Pending View details":      "Pending",
	} {
		if got := normalizeIARCStatus(input); got != want {
			t.Errorf("normalizeIARCStatus(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestReadContentRating(t *testing.T) {
	f := newFakeChrome(t)
	f.setReply(contentRatingReadyExpr(), true)
	f.setReply(readContentRatingScript, map[string]any{
		"iarcStatus":    "Completed",
		"emailAddress":  "info@example.com",
		"certificateId": "cert-123",
		"submittedAt":   "July 29, 2026, 2:20 PM",
		"draftStatus":   "In progress",
		"canStart":      false,
		"ratings":       []map[string]any{{"authority": "ESRB", "rating": "Everyone"}},
	})
	b := connectFake(t, f)

	state, err := ReadContentRating(context.Background(), b, publishingTestDeveloper, publishingTestApp, testAccount)
	if err != nil {
		t.Fatal(err)
	}
	if state.IARCStatus != "Completed" || state.CertificateID != "cert-123" || state.DraftStatus != "In progress" || len(state.Ratings) != 1 {
		t.Errorf("state = %+v", state)
	}
}
