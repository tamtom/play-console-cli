package release

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"google.golang.org/api/androidpublisher/v3"
	"google.golang.org/api/option"

	"github.com/tamtom/play-console-cli/internal/cli/shared"
	"github.com/tamtom/play-console-cli/internal/config"
	"github.com/tamtom/play-console-cli/internal/playclient"
)

func TestReleaseCommand_Name(t *testing.T) {
	cmd := ReleaseCommand()
	if cmd.Name != "release" {
		t.Errorf("expected name %q, got %q", "release", cmd.Name)
	}
}

func TestReleaseCommand_ShortHelp(t *testing.T) {
	cmd := ReleaseCommand()
	if cmd.ShortHelp == "" {
		t.Error("expected non-empty ShortHelp")
	}
}

func TestReleaseCommand_LongHelp(t *testing.T) {
	cmd := ReleaseCommand()
	if cmd.LongHelp == "" {
		t.Error("expected non-empty LongHelp")
	}
}

func TestReleaseCommand_UsageFunc(t *testing.T) {
	cmd := ReleaseCommand()
	if cmd.UsageFunc == nil {
		t.Error("expected UsageFunc to be set")
	}
}

func TestReleaseCommand_NoSubcommands(t *testing.T) {
	cmd := ReleaseCommand()
	if len(cmd.Subcommands) != 0 {
		t.Errorf("expected no subcommands, got %d", len(cmd.Subcommands))
	}
}

func TestReleaseCommand_MissingBundleAndApk(t *testing.T) {
	cmd := ReleaseCommand()
	if err := cmd.FlagSet.Parse([]string{"--package", "com.example.app"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error when neither --bundle nor --apk is provided")
	}
	if !strings.Contains(err.Error(), "--bundle") && !strings.Contains(err.Error(), "--apk") {
		t.Errorf("error should mention --bundle or --apk, got: %s", err.Error())
	}
}

func TestReleaseCommand_BothBundleAndApk(t *testing.T) {
	cmd := ReleaseCommand()
	if err := cmd.FlagSet.Parse([]string{"--bundle", "app.aab", "--apk", "app.apk"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error when both --bundle and --apk are provided")
	}
	if !strings.Contains(err.Error(), "not both") {
		t.Errorf("error should mention 'not both', got: %s", err.Error())
	}
}

func TestReleaseCommand_RolloutTooLow(t *testing.T) {
	cmd := ReleaseCommand()
	if err := cmd.FlagSet.Parse([]string{"--bundle", "app.aab", "--rollout", "-0.5"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for rollout < 0")
	}
	if !strings.Contains(err.Error(), "--rollout") {
		t.Errorf("error should mention --rollout, got: %s", err.Error())
	}
}

func TestReleaseCommand_RolloutTooHigh(t *testing.T) {
	cmd := ReleaseCommand()
	if err := cmd.FlagSet.Parse([]string{"--bundle", "app.aab", "--rollout", "1.5"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for rollout > 1")
	}
	if !strings.Contains(err.Error(), "--rollout") {
		t.Errorf("error should mention --rollout, got: %s", err.Error())
	}
}

func TestReleaseCommand_InvalidOutputFormat(t *testing.T) {
	cmd := ReleaseCommand()
	if err := cmd.FlagSet.Parse([]string{"--output", "xml"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for invalid output format")
	}
}

func TestReleaseCommand_PrettyWithTable(t *testing.T) {
	cmd := ReleaseCommand()
	if err := cmd.FlagSet.Parse([]string{"--output", "table", "--pretty"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for --pretty with table output")
	}
	if !strings.Contains(err.Error(), "--pretty") {
		t.Errorf("error should mention --pretty, got: %s", err.Error())
	}
}

func TestReleaseCommand_WhitespaceBundle(t *testing.T) {
	cmd := ReleaseCommand()
	if err := cmd.FlagSet.Parse([]string{"--bundle", "   "}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for whitespace-only --bundle")
	}
	// Should fall through to missing bundle/apk validation
	if !strings.Contains(err.Error(), "--bundle") && !strings.Contains(err.Error(), "--apk") {
		t.Errorf("error should mention --bundle or --apk, got: %s", err.Error())
	}
}

func TestReleaseCommand_WhitespaceApk(t *testing.T) {
	cmd := ReleaseCommand()
	if err := cmd.FlagSet.Parse([]string{"--apk", "   "}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for whitespace-only --apk")
	}
	if !strings.Contains(err.Error(), "--bundle") && !strings.Contains(err.Error(), "--apk") {
		t.Errorf("error should mention --bundle or --apk, got: %s", err.Error())
	}
}

func TestReleaseCommand_RolloutBoundary_Zero(t *testing.T) {
	cmd := ReleaseCommand()
	if err := cmd.FlagSet.Parse([]string{"--bundle", "app.aab", "--rollout", "0"}); err != nil {
		t.Fatal(err)
	}
	// rollout=0 is valid (0.0-1.0 inclusive); should proceed past validation
	// Will fail on NewService (no credentials), not on rollout validation
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error (no credentials), but should not be rollout error")
	}
	if strings.Contains(err.Error(), "--rollout") {
		t.Errorf("rollout=0 should be valid, got: %s", err.Error())
	}
}

func TestReleaseCommand_RolloutBoundary_One(t *testing.T) {
	cmd := ReleaseCommand()
	if err := cmd.FlagSet.Parse([]string{"--bundle", "app.aab", "--rollout", "1"}); err != nil {
		t.Fatal(err)
	}
	// rollout=1.0 is the default and valid; should proceed past validation
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error (no credentials), but should not be rollout error")
	}
	if strings.Contains(err.Error(), "--rollout") {
		t.Errorf("rollout=1.0 should be valid, got: %s", err.Error())
	}
}

func TestExecute_DryRunBuildsPlanWithoutService(t *testing.T) {
	origNewService := newServiceFn
	newServiceFn = func(context.Context) (*playclient.Service, error) {
		t.Fatal("dry-run should not create an authenticated Play service")
		return nil, nil
	}
	t.Cleanup(func() { newServiceFn = origNewService })

	bundlePath := filepath.Join(t.TempDir(), "app.aab")
	if err := os.WriteFile(bundlePath, []byte("bundle"), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := Execute(shared.ContextWithDryRun(context.Background(), true), Options{
		PackageName:  "com.example",
		Track:        "internal",
		BundlePath:   bundlePath,
		ReleaseNotes: "Bug fixes",
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result["dryRun"] != true {
		t.Fatalf("dryRun = %#v, want true", result["dryRun"])
	}
	steps, ok := result["steps"].([]string)
	if !ok || len(steps) == 0 {
		t.Fatalf("expected dry-run steps, got %#v", result["steps"])
	}
	if !containsString(steps, "commit edit") {
		t.Fatalf("expected commit step in dry-run plan, got %#v", steps)
	}
}

func TestExecute_UpdatesListingsAndReplacesScreenshots(t *testing.T) {
	bundlePath := filepath.Join(t.TempDir(), "app.aab")
	if err := os.WriteFile(bundlePath, []byte("bundle"), 0o644); err != nil {
		t.Fatal(err)
	}

	listingsDir := t.TempDir()
	localeDir := filepath.Join(listingsDir, "en-US")
	if err := os.MkdirAll(localeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(localeDir, "title.txt"), []byte("Updated Title"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(localeDir, "short_description.txt"), []byte("Updated Short"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(localeDir, "full_description.txt"), []byte("Updated Full"), 0o644); err != nil {
		t.Fatal(err)
	}

	screenshotsDir := t.TempDir()
	phoneDir := filepath.Join(screenshotsDir, "en-US", "phoneScreenshots")
	if err := os.MkdirAll(phoneDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(phoneDir, "01.png"), []byte("png"), 0o644); err != nil {
		t.Fatal(err)
	}

	var listingUpdates atomic.Int32
	var imageDeletes atomic.Int32
	var imageUploads atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		path := r.URL.Path
		switch {
		case r.Method == http.MethodPost && strings.HasSuffix(path, "/androidpublisher/v3/applications/com.example/edits"):
			_ = json.NewEncoder(w).Encode(androidpublisher.AppEdit{Id: "edit-123"})
		case r.Method == http.MethodPost && strings.Contains(path, "/upload/androidpublisher/v3/applications/com.example/edits/edit-123/bundles"):
			_ = json.NewEncoder(w).Encode(androidpublisher.Bundle{VersionCode: 42})
		case r.Method == http.MethodPut && strings.Contains(path, "/androidpublisher/v3/applications/com.example/edits/edit-123/listings/en-US"):
			listingUpdates.Add(1)
			var listing androidpublisher.Listing
			if err := json.NewDecoder(r.Body).Decode(&listing); err != nil {
				t.Errorf("decode listing: %v", err)
			}
			if listing.Title != "Updated Title" {
				t.Errorf("Title = %q, want Updated Title", listing.Title)
			}
			_ = json.NewEncoder(w).Encode(listing)
		case r.Method == http.MethodDelete && strings.Contains(path, "/androidpublisher/v3/applications/com.example/edits/edit-123/listings/en-US/phoneScreenshots"):
			imageDeletes.Add(1)
			_ = json.NewEncoder(w).Encode(androidpublisher.ImagesDeleteAllResponse{})
		case r.Method == http.MethodPost && strings.Contains(path, "/upload/androidpublisher/v3/applications/com.example/edits/edit-123/listings/en-US/phoneScreenshots"):
			imageUploads.Add(1)
			_ = json.NewEncoder(w).Encode(androidpublisher.ImagesUploadResponse{Image: &androidpublisher.Image{Id: "img-1"}})
		case r.Method == http.MethodPut && strings.Contains(path, "/androidpublisher/v3/applications/com.example/edits/edit-123/tracks/internal"):
			_ = json.NewEncoder(w).Encode(androidpublisher.Track{Track: "internal"})
		case r.Method == http.MethodPost && strings.Contains(path, "/androidpublisher/v3/applications/com.example/edits/edit-123:validate"):
			_ = json.NewEncoder(w).Encode(androidpublisher.AppEdit{Id: "edit-123"})
		case r.Method == http.MethodPost && strings.Contains(path, "/androidpublisher/v3/applications/com.example/edits/edit-123:commit"):
			_ = json.NewEncoder(w).Encode(androidpublisher.AppEdit{Id: "edit-123"})
		default:
			t.Errorf("unexpected request: %s %s", r.Method, path)
			http.Error(w, "unexpected request", http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)

	api, err := androidpublisher.NewService(context.Background(), option.WithHTTPClient(server.Client()), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatal(err)
	}
	origNewService := newServiceFn
	newServiceFn = func(context.Context) (*playclient.Service, error) {
		return &playclient.Service{API: api, Cfg: &config.Config{}}, nil
	}
	t.Cleanup(func() { newServiceFn = origNewService })

	result, err := Execute(context.Background(), Options{
		PackageName:    "com.example",
		Track:          "internal",
		BundlePath:     bundlePath,
		ListingsDir:    listingsDir,
		ScreenshotsDir: screenshotsDir,
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if listingUpdates.Load() != 1 {
		t.Fatalf("listing updates = %d, want 1", listingUpdates.Load())
	}
	if imageDeletes.Load() != 1 {
		t.Fatalf("image deletes = %d, want 1", imageDeletes.Load())
	}
	if imageUploads.Load() != 1 {
		t.Fatalf("image uploads = %d, want 1", imageUploads.Load())
	}
	if result["metadataLocales"] != 1 {
		t.Fatalf("metadataLocales = %#v, want 1", result["metadataLocales"])
	}
	if result["screenshotImages"] != 1 {
		t.Fatalf("screenshotImages = %#v, want 1", result["screenshotImages"])
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
