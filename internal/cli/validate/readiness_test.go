package validate

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"google.golang.org/api/androidpublisher/v3"

	"github.com/tamtom/play-console-cli/internal/cli/shared"
	"github.com/tamtom/play-console-cli/internal/validation"
)

func TestBuildReadinessReport_Success(t *testing.T) {
	original := fetchRemoteReadinessStateFn
	fetchRemoteReadinessStateFn = func(context.Context, string, string) (*remoteReadinessState, error) {
		return &remoteReadinessState{
			Tracks: []*androidpublisher.Track{
				{
					Track: "production",
					Releases: []*androidpublisher.TrackRelease{
						{
							Status:       "completed",
							VersionCodes: []int64{42},
							ReleaseNotes: []*androidpublisher.LocalizedText{{Language: "en-US", Text: "Bug fixes"}},
						},
					},
				},
			},
			TargetTrack: &androidpublisher.Track{
				Track: "production",
				Releases: []*androidpublisher.TrackRelease{
					{
						Status:       "completed",
						VersionCodes: []int64{42},
						ReleaseNotes: []*androidpublisher.LocalizedText{{Language: "en-US", Text: "Bug fixes"}},
					},
				},
			},
			Listings: []*androidpublisher.Listing{{Language: "en-US", Title: "Example"}},
		}, nil
	}
	t.Cleanup(func() {
		fetchRemoteReadinessStateFn = original
	})

	listingsDir := filepath.Join(t.TempDir(), "metadata")
	if err := os.MkdirAll(filepath.Join(listingsDir, "en-US"), 0o755); err != nil {
		t.Fatalf("mkdir listings: %v", err)
	}
	writeTextFile(t, filepath.Join(listingsDir, "en-US", "title.txt"), "Example App")
	writeTextFile(t, filepath.Join(listingsDir, "en-US", "short_description.txt"), "Short description")
	writeTextFile(t, filepath.Join(listingsDir, "en-US", "full_description.txt"), "Full description")

	screenshotsDir := filepath.Join(t.TempDir(), "screenshots")
	phoneDir := filepath.Join(screenshotsDir, "en-US", "phoneScreenshots")
	if err := os.MkdirAll(phoneDir, 0o755); err != nil {
		t.Fatalf("mkdir screenshots: %v", err)
	}
	writeTextFile(t, filepath.Join(phoneDir, "1.png"), "png")
	writeTextFile(t, filepath.Join(phoneDir, "2.png"), "png")

	report := buildReadinessReport(context.Background(), readinessOptions{
		PackageName:    "com.example.app",
		Track:          "production",
		ListingsDir:    listingsDir,
		ScreenshotsDir: screenshotsDir,
		ReleaseNotes:   "Bug fixes",
	})

	if report.Summary.Blocking != 0 {
		t.Fatalf("blocking = %d, want 0", report.Summary.Blocking)
	}
	if report.Summary.Manual == 0 {
		t.Fatal("expected manual follow-up checks")
	}
	assertCheckState(t, report, "local-listings-found", validation.ReadinessInfo)
	assertCheckState(t, report, "remote-listings-found", validation.ReadinessInfo)
	assertCheckState(t, report, "target-track-active-release", validation.ReadinessInfo)
}

func TestBuildReadinessReport_RemoteFailureAddsBlocking(t *testing.T) {
	original := fetchRemoteReadinessStateFn
	fetchRemoteReadinessStateFn = func(context.Context, string, string) (*remoteReadinessState, error) {
		return nil, context.DeadlineExceeded
	}
	t.Cleanup(func() {
		fetchRemoteReadinessStateFn = original
	})

	report := buildReadinessReport(context.Background(), readinessOptions{
		PackageName: "com.example.app",
		Track:       "production",
	})

	assertCheckState(t, report, "remote-play-state-unavailable", validation.ReadinessBlocking)
}

func TestValidateCommand_RootRequiresPackageWhenFlagsProvided(t *testing.T) {
	cmd := ValidateCommand()
	if err := cmd.FlagSet.Parse([]string{"--release-notes", "Bug fixes"}); err != nil {
		t.Fatalf("parse flags: %v", err)
	}

	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected missing package error")
	}
	if !strings.Contains(err.Error(), "package") {
		t.Fatalf("expected package error, got: %v", err)
	}
}

func TestRunReadinessCommand_StrictWarningsFail(t *testing.T) {
	original := fetchRemoteReadinessStateFn
	fetchRemoteReadinessStateFn = func(context.Context, string, string) (*remoteReadinessState, error) {
		return &remoteReadinessState{}, nil
	}
	t.Cleanup(func() {
		fetchRemoteReadinessStateFn = original
	})

	err := runReadinessCommand(context.Background(), readinessOptions{
		PackageName: "com.example.app",
		Track:       "production",
		Strict:      true,
		Output:      "json",
	})
	if err == nil {
		t.Fatal("expected strict-mode failure")
	}
	if !shared.IsReportedError(err) {
		t.Fatalf("expected reported error, got %T", err)
	}
}

func writeTextFile(t *testing.T, path, value string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(value), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func assertCheckState(t *testing.T, report *validation.ReadinessReport, id string, state validation.ReadinessState) {
	t.Helper()
	for _, check := range report.Checks {
		if check.ID == id {
			if check.State != state {
				t.Fatalf("check %q state = %q, want %q", id, check.State, state)
			}
			return
		}
	}
	t.Fatalf("check %q not found", id)
}
