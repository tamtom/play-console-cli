package release

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/tamtom/play-console-cli/internal/cli/shared"
	"github.com/tamtom/play-console-cli/internal/playclient"
)

const (
	fakePackage = "com.example.app"
	fakeEditID  = "edit-1"
	fakeEdits   = "/androidpublisher/v3/applications/" + fakePackage + "/edits"
	fakeEdit    = fakeEdits + "/" + fakeEditID
	fakeUpload  = "/upload/androidpublisher/v3/applications/" + fakePackage + "/edits/" + fakeEditID
)

// fakePlay is a minimal Edits API server for release tests. remoteImages maps
// "locale/imageType" to the SHA-256 values of images already in the edit.
type fakePlay struct {
	t            *testing.T
	locales      []string
	remoteImages map[string][]string
	onEditInsert func()

	mu            sync.Mutex
	requests      []string
	listingBodies map[string]map[string]any
	imageUploads  map[string][][]byte
}

func newFakePlay(t *testing.T, locales []string, remoteImages map[string][]string) *fakePlay {
	t.Helper()
	return &fakePlay{
		t:             t,
		locales:       locales,
		remoteImages:  remoteImages,
		listingBodies: map[string]map[string]any{},
		imageUploads:  map[string][][]byte{},
	}
}

func (f *fakePlay) context(t *testing.T) context.Context {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(f.serveHTTP))
	t.Cleanup(server.Close)
	ctx := playclient.ContextWithServiceFactory(context.Background(), func(ctx context.Context) (*playclient.Service, error) {
		return playclient.NewServiceWithClient(ctx, server.Client(), server.URL+"/")
	})
	return shared.ContextWithIO(ctx, &bytes.Buffer{}, &bytes.Buffer{})
}

func (f *fakePlay) serveHTTP(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		f.t.Errorf("read request body: %v", err)
	}
	f.mu.Lock()
	f.requests = append(f.requests, r.Method+" "+r.URL.Path)
	f.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	path := r.URL.Path
	switch {
	case r.Method == http.MethodPost && path == fakeEdits:
		if f.onEditInsert != nil {
			f.onEditInsert()
		}
		writeJSON(w, map[string]any{"id": fakeEditID})
	case r.Method == http.MethodPatch && strings.HasPrefix(path, fakeEdit+"/listings/"):
		locale := strings.TrimPrefix(path, fakeEdit+"/listings/")
		var listing map[string]any
		if err := json.Unmarshal(body, &listing); err != nil {
			f.t.Errorf("decode listing body: %v", err)
		}
		f.mu.Lock()
		f.listingBodies[locale] = listing
		f.mu.Unlock()
		writeJSON(w, map[string]any{"language": locale})
	case r.Method == http.MethodGet && path == fakeEdit+"/listings":
		listings := make([]map[string]any, 0, len(f.locales))
		for _, locale := range f.locales {
			listings = append(listings, map[string]any{"language": locale})
		}
		writeJSON(w, map[string]any{"listings": listings})
	case r.Method == http.MethodGet && strings.HasPrefix(path, fakeEdit+"/listings/"):
		key := strings.TrimPrefix(path, fakeEdit+"/listings/")
		images := make([]map[string]any, 0)
		for i, sum := range f.remoteImages[key] {
			images = append(images, map[string]any{"id": fmt.Sprintf("remote-%d", i), "sha256": sum})
		}
		writeJSON(w, map[string]any{"images": images})
	case r.Method == http.MethodPost && path == fakeUpload+"/bundles":
		writeJSON(w, map[string]any{"versionCode": 42})
	case r.Method == http.MethodPost && strings.HasPrefix(path, fakeUpload+"/listings/"):
		key := strings.TrimPrefix(path, fakeUpload+"/listings/")
		f.mu.Lock()
		f.imageUploads[key] = append(f.imageUploads[key], body)
		f.mu.Unlock()
		writeJSON(w, map[string]any{"image": map[string]any{"id": "uploaded"}})
	case r.Method == http.MethodPut && path == fakeEdit+"/tracks/internal":
		writeJSON(w, map[string]any{"track": "internal"})
	case r.Method == http.MethodPost && (path == fakeEdit+":validate" || path == fakeEdit+":commit"):
		writeJSON(w, map[string]any{"id": fakeEditID})
	default:
		http.Error(w, fmt.Sprintf("unexpected request: %s %s", r.Method, path), http.StatusNotFound)
	}
}

func (f *fakePlay) requestLog() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.requests...)
}

func (f *fakePlay) uploadsFor(key string) [][]byte {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.imageUploads[key]
}

func (f *fakePlay) assertNoDeletes(t *testing.T) {
	t.Helper()
	for _, request := range f.requestLog() {
		if strings.HasPrefix(request, http.MethodDelete+" ") {
			t.Fatalf("release sent %s; release must never delete screenshots", request)
		}
	}
}

func writeJSON(w http.ResponseWriter, value any) {
	_ = json.NewEncoder(w).Encode(value)
}

type releaseFixture struct {
	bundle      string
	listings    string
	screenshots string
}

func newReleaseFixture(t *testing.T) releaseFixture {
	t.Helper()
	dir := t.TempDir()
	fixture := releaseFixture{
		bundle:      filepath.Join(dir, "app.aab"),
		listings:    filepath.Join(dir, "listings"),
		screenshots: filepath.Join(dir, "screenshots"),
	}
	if err := os.WriteFile(fixture.bundle, []byte("not-empty-aab"), 0o600); err != nil {
		t.Fatal(err)
	}
	return fixture
}

func (f releaseFixture) writeListing(t *testing.T, locale, name, content string) {
	t.Helper()
	dir := filepath.Join(f.listings, locale)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func (f releaseFixture) writeScreenshot(t *testing.T, locale, imageType, name string, fill color.RGBA) []byte {
	t.Helper()
	dir := filepath.Join(f.screenshots, locale, imageType)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	data := testPNG(t, fill)
	if err := os.WriteFile(filepath.Join(dir, name), data, 0o600); err != nil {
		t.Fatal(err)
	}
	return data
}

func (f releaseFixture) options() Options {
	return Options{
		PackageName:     fakePackage,
		Track:           "internal",
		BundlePath:      f.bundle,
		RolloutFraction: 1,
		Status:          "completed",
	}
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func testPNG(t *testing.T, fill color.RGBA) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 320, 640))
	for y := 0; y < 640; y++ {
		for x := 0; x < 320; x++ {
			img.SetRGBA(x, y, fill)
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

var (
	red   = color.RGBA{R: 0xff, A: 0xff}
	green = color.RGBA{G: 0xff, A: 0xff}
	blue  = color.RGBA{B: 0xff, A: 0xff}
)

func TestExecute_AppliesListingsAndScreenshotsBeforeCommit(t *testing.T) {
	fixture := newReleaseFixture(t)
	fixture.writeListing(t, "en-US", "title.txt", "Example App")
	fixture.writeListing(t, "en-US", "video.txt", "")
	fixture.writeScreenshot(t, "en-US", "phoneScreenshots", "01.png", red)
	fixture.writeScreenshot(t, "en-US", "phoneScreenshots", "02.png", green)

	play := newFakePlay(t, []string{"en-US"}, nil)
	opts := fixture.options()
	opts.ListingsDir = fixture.listings
	opts.ScreenshotsDir = fixture.screenshots
	result, err := Execute(play.context(t), opts)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	want := []string{
		"POST " + fakeEdits,
		"PATCH " + fakeEdit + "/listings/en-US",
		"GET " + fakeEdit + "/listings",
		"GET " + fakeEdit + "/listings/en-US/phoneScreenshots",
		"POST " + fakeUpload + "/bundles",
		"POST " + fakeUpload + "/listings/en-US/phoneScreenshots",
		"POST " + fakeUpload + "/listings/en-US/phoneScreenshots",
		"PUT " + fakeEdit + "/tracks/internal",
		"POST " + fakeEdit + ":validate",
		"POST " + fakeEdit + ":commit",
	}
	if got := play.requestLog(); strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("requests:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	listing := play.listingBodies["en-US"]
	if listing["title"] != "Example App" || listing["video"] != "" {
		t.Fatalf("listing body = %#v, want title and explicit empty video", listing)
	}
	if _, ok := listing["shortDescription"]; ok {
		t.Fatalf("listing body = %#v, a missing file must not overwrite its field", listing)
	}
	if result["screenshotsUploaded"] != 2 || result["screenshotsSkipped"] != 0 {
		t.Fatalf("result = %#v, want 2 uploaded and 0 skipped", result)
	}
	if locales, ok := result["listingsUpdated"].([]string); !ok || strings.Join(locales, ",") != "en-US" {
		t.Fatalf("result listingsUpdated = %#v, want [en-US]", result["listingsUpdated"])
	}
}

func TestExecute_SkipsScreenshotsAlreadyOnPlay(t *testing.T) {
	fixture := newReleaseFixture(t)
	onPlay := fixture.writeScreenshot(t, "en-US", "phoneScreenshots", "01.png", red)
	newShot := fixture.writeScreenshot(t, "en-US", "phoneScreenshots", "02.png", green)
	fixture.writeScreenshot(t, "en-US", "phoneScreenshots", "03.png", green) // same content as 02.png

	play := newFakePlay(t, []string{"en-US"}, map[string][]string{
		"en-US/phoneScreenshots": {strings.ToUpper(sha256Hex(onPlay)), sha256Hex([]byte("other"))},
	})
	opts := fixture.options()
	opts.ScreenshotsDir = fixture.screenshots
	result, err := Execute(play.context(t), opts)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	uploads := play.uploadsFor("en-US/phoneScreenshots")
	if len(uploads) != 1 || !bytes.Contains(uploads[0], newShot) {
		t.Fatalf("uploads = %d, want only 02.png", len(uploads))
	}
	if result["screenshotsUploaded"] != 1 || result["screenshotsSkipped"] != 2 {
		t.Fatalf("result = %#v, want 1 uploaded and 2 skipped", result)
	}
	play.assertNoDeletes(t)
}

func TestExecute_RerunWithSameScreenshotsUploadsNothing(t *testing.T) {
	fixture := newReleaseFixture(t)
	first := fixture.writeScreenshot(t, "en-US", "phoneScreenshots", "01.png", red)
	second := fixture.writeScreenshot(t, "en-US", "phoneScreenshots", "02.png", green)

	play := newFakePlay(t, []string{"en-US"}, map[string][]string{
		"en-US/phoneScreenshots": {sha256Hex(first), sha256Hex(second)},
	})
	opts := fixture.options()
	opts.ScreenshotsDir = fixture.screenshots
	result, err := Execute(play.context(t), opts)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if uploads := play.uploadsFor("en-US/phoneScreenshots"); len(uploads) != 0 {
		t.Fatalf("uploads = %d, want 0 on a re-run", len(uploads))
	}
	if result["screenshotsUploaded"] != 0 || result["screenshotsSkipped"] != 2 {
		t.Fatalf("result = %#v, want 0 uploaded and 2 skipped", result)
	}
	play.assertNoDeletes(t)
}

func TestExecute_StopsBeforeBundleUploadWhenScreenshotLimitIsExceeded(t *testing.T) {
	fixture := newReleaseFixture(t)
	fixture.writeScreenshot(t, "en-US", "phoneScreenshots", "01.png", red)
	fixture.writeScreenshot(t, "en-US", "phoneScreenshots", "02.png", green)

	remote := make([]string, 7)
	for i := range remote {
		remote[i] = sha256Hex([]byte(fmt.Sprintf("old-%d", i)))
	}
	play := newFakePlay(t, []string{"en-US"}, map[string][]string{"en-US/phoneScreenshots": remote})
	opts := fixture.options()
	opts.ScreenshotsDir = fixture.screenshots
	_, err := Execute(play.context(t), opts)
	if err == nil {
		t.Fatal("expected screenshot limit error")
	}
	for _, want := range []string{"en-US", "phoneScreenshots", "9", "8", "gplay images delete"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %q, want it to contain %q", err, want)
		}
	}
	for _, request := range play.requestLog() {
		if strings.HasPrefix(request, "POST "+fakeUpload) || strings.Contains(request, "/tracks/") || strings.HasSuffix(request, ":commit") {
			t.Fatalf("release sent %s after the limit check failed", request)
		}
	}
	play.assertNoDeletes(t)
}

func TestExecute_PhoneScreenshotMinimumCountsImagesOnPlay(t *testing.T) {
	tests := []struct {
		name    string
		remote  []string
		wantErr bool
	}{
		{name: "one local and none on Play", remote: nil, wantErr: true},
		{name: "one local and one on Play", remote: []string{sha256Hex([]byte("old"))}, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fixture := newReleaseFixture(t)
			fixture.writeScreenshot(t, "en-US", "phoneScreenshots", "01.png", red)
			play := newFakePlay(t, []string{"en-US"}, map[string][]string{"en-US/phoneScreenshots": tt.remote})
			opts := fixture.options()
			opts.ScreenshotsDir = fixture.screenshots
			_, err := Execute(play.context(t), opts)
			if tt.wantErr {
				if err == nil || !strings.Contains(err.Error(), "at least 2") {
					t.Fatalf("error = %v, want phone minimum error", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("Execute: %v", err)
			}
		})
	}
}

func TestExecute_TreatsLocaleWithoutListingAsEmpty(t *testing.T) {
	fixture := newReleaseFixture(t)
	fixture.writeScreenshot(t, "de-DE", "phoneScreenshots", "01.png", red)
	fixture.writeScreenshot(t, "de-DE", "phoneScreenshots", "02.png", green)

	play := newFakePlay(t, []string{"en-US"}, nil)
	opts := fixture.options()
	opts.ScreenshotsDir = fixture.screenshots
	if _, err := Execute(play.context(t), opts); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	for _, request := range play.requestLog() {
		if request == "GET "+fakeEdit+"/listings/de-DE/phoneScreenshots" {
			t.Fatal("release listed images for a locale that has no listing in the edit")
		}
	}
	if uploads := play.uploadsFor("de-DE/phoneScreenshots"); len(uploads) != 2 {
		t.Fatalf("uploads = %d, want 2", len(uploads))
	}
}

func TestExecute_DryRunDoesNotReadRemoteScreenshots(t *testing.T) {
	fixture := newReleaseFixture(t)
	fixture.writeScreenshot(t, "en-US", "phoneScreenshots", "01.png", red)
	fixture.writeScreenshot(t, "en-US", "phoneScreenshots", "02.png", green)

	play := newFakePlay(t, []string{"en-US"}, nil)
	opts := fixture.options()
	opts.ScreenshotsDir = fixture.screenshots
	ctx := shared.ContextWithDryRun(play.context(t), true)
	if _, err := Execute(ctx, opts); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	for _, request := range play.requestLog() {
		if strings.HasPrefix(request, "GET "+fakeEdit+"/listings") {
			t.Fatalf("dry run sent %s; the dry-run edit has no ID", request)
		}
	}
}

func TestExecute_RejectsInvalidScreenshotBeforeServiceCreation(t *testing.T) {
	fixture := newReleaseFixture(t)
	fixture.writeScreenshot(t, "en-US", "phoneScreenshots", "02.png", green)
	if err := os.WriteFile(filepath.Join(fixture.screenshots, "en-US", "phoneScreenshots", "01.png"), []byte("not-an-image"), 0o600); err != nil {
		t.Fatal(err)
	}

	serviceCreated := false
	ctx := playclient.ContextWithServiceFactory(context.Background(), func(context.Context) (*playclient.Service, error) {
		serviceCreated = true
		return nil, errors.New("service should not be created")
	})
	_, err := Execute(ctx, Options{BundlePath: fixture.bundle, ScreenshotsDir: fixture.screenshots})
	if err == nil || !strings.Contains(err.Error(), "could not be decoded") {
		t.Fatalf("error = %v, want invalid screenshot error", err)
	}
	if serviceCreated {
		t.Fatal("service was created before rejecting invalid screenshot")
	}
}

func TestExecute_UploadsPrevalidatedScreenshotDescriptor(t *testing.T) {
	fixture := newReleaseFixture(t)
	original := fixture.writeScreenshot(t, "en-US", "phoneScreenshots", "01.png", red)
	fixture.writeScreenshot(t, "en-US", "phoneScreenshots", "02.png", green)
	firstPath := filepath.Join(fixture.screenshots, "en-US", "phoneScreenshots", "01.png")
	replacement := testPNG(t, blue)

	play := newFakePlay(t, []string{"en-US"}, nil)
	play.onEditInsert = func() {
		// Replace the file after validation. The upload must still send the
		// descriptor that passed validation.
		tmp := firstPath + ".tmp"
		if err := os.WriteFile(tmp, replacement, 0o600); err != nil {
			t.Errorf("write replacement: %v", err)
		}
		if err := os.Rename(tmp, firstPath); err != nil {
			t.Errorf("replace screenshot: %v", err)
		}
	}
	opts := fixture.options()
	opts.ScreenshotsDir = fixture.screenshots
	if _, err := Execute(play.context(t), opts); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	uploads := play.uploadsFor("en-US/phoneScreenshots")
	if len(uploads) != 2 {
		t.Fatalf("uploads = %d, want 2", len(uploads))
	}
	if !bytes.Contains(uploads[0], original) || bytes.Contains(uploads[0], replacement) {
		t.Fatal("first upload did not use the prevalidated file descriptor")
	}
}

func TestExecute_ScreenshotLimitCountsDuplicateImagesOnPlay(t *testing.T) {
	fixture := newReleaseFixture(t)
	for i := 0; i < 7; i++ {
		fixture.writeScreenshot(t, "en-US", "phoneScreenshots", fmt.Sprintf("%02d.png", i), color.RGBA{R: uint8(i * 30), A: 0xff})
	}
	duplicate := sha256Hex([]byte("old"))
	play := newFakePlay(t, []string{"en-US"}, map[string][]string{
		"en-US/phoneScreenshots": {duplicate, duplicate},
	})
	opts := fixture.options()
	opts.ScreenshotsDir = fixture.screenshots
	_, err := Execute(play.context(t), opts)
	if err == nil || !strings.Contains(err.Error(), "would have 9 screenshots (2 already on Play, 7 new)") {
		t.Fatalf("error = %v, want limit error that counts both images on Play", err)
	}
}

func TestExecute_MinimumAppliesOnlyToPhoneScreenshots(t *testing.T) {
	fixture := newReleaseFixture(t)
	fixture.writeScreenshot(t, "en-US", "tenInchScreenshots", "01.png", red)

	play := newFakePlay(t, []string{"en-US"}, nil)
	opts := fixture.options()
	opts.ScreenshotsDir = fixture.screenshots
	if _, err := Execute(play.context(t), opts); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if uploads := play.uploadsFor("en-US/tenInchScreenshots"); len(uploads) != 1 {
		t.Fatalf("uploads = %d, want 1 tablet screenshot", len(uploads))
	}
}
