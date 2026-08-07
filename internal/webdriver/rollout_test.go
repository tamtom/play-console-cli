package webdriver

import (
	"context"
	"strings"
	"testing"
	"time"
)

func withRolloutHelpers(script string) string {
	return formHelpers + ` && ` + rolloutHelpers + ` && ` + script
}

func TestProductionTrackURL(t *testing.T) {
	got := productionTrackURL(publishingTestDeveloper, publishingTestApp, testAccount)
	want := "https://play.google.com/console/developers/1234567890/app/9876543210/tracks/production?authuser=me%40example.com&hl=en"
	if got != want {
		t.Errorf("url = %q, want %q", got, want)
	}
}

func TestOpenProductionReleases_NoDraft(t *testing.T) {
	f := newFakeChrome(t)
	f.setReply(releasesTabReadyScript, true)
	f.setReply(openReleasesTabScript, true)
	// No draft: edit button never appears.
	f.setReply(draftReleaseReadyScript, false)
	b := connectFake(t, f)

	orig := rolloutDraftTimeout
	rolloutDraftTimeout = 300 * time.Millisecond
	t.Cleanup(func() { rolloutDraftTimeout = orig })

	err := OpenProductionReleases(context.Background(), b, publishingTestDeveloper, publishingTestApp, testAccount)
	if err == nil || !strings.Contains(err.Error(), "no draft release") {
		t.Errorf("err = %v, want no-draft error", err)
	}
}

func TestOpenProductionReleases_FindsDraft(t *testing.T) {
	f := newFakeChrome(t)
	f.setReply(releasesTabReadyScript, true)
	f.setReply(openReleasesTabScript, true)
	f.setReply(draftReleaseReadyScript, true)
	b := connectFake(t, f)

	if err := OpenProductionReleases(context.Background(), b, publishingTestDeveloper, publishingTestApp, testAccount); err != nil {
		t.Fatalf("OpenProductionReleases: %v", err)
	}
}

func scriptReleaseWizard(t *testing.T, f *fakeChrome) {
	t.Helper()
	f.setReply(withRolloutHelpers(editDraftClickScript), true)
	f.setReply(preparePageReadyScript, true)
	f.setReply(readPrepareScript, map[string]any{"releaseName": "1.2.1"})
	f.setReply(withRolloutHelpers(nextClickScript), true)
	f.setReply(reviewPageReadyScript, true)
	f.setReply(readReviewScript, map[string]any{
		"warnings": []string{"There is no deobfuscation file associated with this App Bundle."},
		"canSave":  true,
	})
	f.setReply(withRolloutHelpers(saveReviewClickScript), true)
	f.setReply(publishingOverviewPresentScript, true)
}

func TestReleaseWizard_HappyPath(t *testing.T) {
	f := newFakeChrome(t)
	scriptReleaseWizard(t, f)
	b := connectFake(t, f)

	prepare, err := OpenDraftRelease(context.Background(), b)
	if err != nil {
		t.Fatalf("OpenDraftRelease: %v", err)
	}
	if prepare.ReleaseName != "1.2.1" {
		t.Errorf("releaseName = %q", prepare.ReleaseName)
	}
	review, err := ReviewDraftRelease(context.Background(), b)
	if err != nil {
		t.Fatalf("ReviewDraftRelease: %v", err)
	}
	if len(review.Warnings) != 1 || !review.CanSave {
		t.Errorf("review = %+v", review)
	}
	if err := SaveReleaseReview(context.Background(), b, 2*time.Second); err != nil {
		t.Fatalf("SaveReleaseReview: %v", err)
	}
}

func TestOpenDraftRelease_RefusesDisabledEdit(t *testing.T) {
	f := newFakeChrome(t)
	scriptReleaseWizard(t, f)
	f.setReply(withRolloutHelpers(editDraftClickScript), false)
	b := connectFake(t, f)

	_, err := OpenDraftRelease(context.Background(), b)
	if err == nil || !strings.Contains(err.Error(), "Edit release") {
		t.Errorf("err = %v, want edit-release error", err)
	}
}

func TestSaveReleaseReview_RefusesDisabledSave(t *testing.T) {
	f := newFakeChrome(t)
	f.setReply(withRolloutHelpers(saveReviewClickScript), false)
	b := connectFake(t, f)

	err := SaveReleaseReview(context.Background(), b, 300*time.Millisecond)
	if err == nil || !strings.Contains(err.Error(), "missing or disabled") {
		t.Errorf("err = %v, want disabled-save error", err)
	}
}
