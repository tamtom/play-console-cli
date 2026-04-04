package images

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"google.golang.org/api/androidpublisher/v3"

	"github.com/tamtom/play-console-cli/internal/config"
)

type fakeMediaBackend struct {
	locales []string
	images  map[string]map[string][]remoteImage
	uploads []string
	cfg     *config.Config
}

func (f *fakeMediaBackend) Config() *config.Config {
	if f.cfg == nil {
		return &config.Config{}
	}
	return f.cfg
}

func (f *fakeMediaBackend) ListLocales(ctx context.Context, packageName, editID string) ([]string, error) {
	return append([]string(nil), f.locales...), nil
}

func (f *fakeMediaBackend) ListImages(ctx context.Context, packageName, editID, locale, imageType string) ([]remoteImage, error) {
	if f.images == nil {
		return nil, nil
	}
	return append([]remoteImage(nil), f.images[locale][imageType]...), nil
}

func (f *fakeMediaBackend) UploadImage(ctx context.Context, packageName, editID, locale, imageType, filePath string) (*androidpublisher.Image, error) {
	f.uploads = append(f.uploads, filepath.Join(locale, imageType, filepath.Base(filePath)))
	return &androidpublisher.Image{Id: filepath.Base(filePath)}, nil
}

func TestScanLocaleMedia_ParsesSupportedLayout(t *testing.T) {
	root := t.TempDir()
	localeDir := filepath.Join(root, "en-US")
	if err := os.MkdirAll(filepath.Join(localeDir, "images", "phoneScreenshots"), 0o755); err != nil {
		t.Fatalf("mkdir screenshots: %v", err)
	}
	if err := os.WriteFile(filepath.Join(localeDir, "images", "phoneScreenshots", "1.png"), []byte("shot-1"), 0o644); err != nil {
		t.Fatalf("write screenshot: %v", err)
	}
	if err := os.WriteFile(filepath.Join(localeDir, "images", "featureGraphic.png"), []byte("feature"), 0o644); err != nil {
		t.Fatalf("write feature graphic: %v", err)
	}

	assets, err := scanLocaleMedia(localeDir)
	if err != nil {
		t.Fatalf("scanLocaleMedia: %v", err)
	}
	if got := len(assets["phoneScreenshots"]); got != 1 {
		t.Fatalf("expected 1 screenshot, got %d", got)
	}
	if got := len(assets["featureGraphic"]); got != 1 {
		t.Fatalf("expected 1 feature graphic, got %d", got)
	}
}

func TestScanLocaleMedia_InvalidDeviceType(t *testing.T) {
	root := t.TempDir()
	localeDir := filepath.Join(root, "en-US")
	if err := os.MkdirAll(filepath.Join(localeDir, "images", "bogusScreenshots"), 0o755); err != nil {
		t.Fatalf("mkdir bogus screenshot dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(localeDir, "images", "bogusScreenshots", "1.png"), []byte("shot-1"), 0o644); err != nil {
		t.Fatalf("write bogus screenshot: %v", err)
	}

	_, err := scanLocaleMedia(localeDir)
	if err == nil {
		t.Fatal("expected error for invalid screenshot type")
	}
	if !strings.Contains(err.Error(), "unknown screenshot type") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBuildMediaPlan_HappyPath(t *testing.T) {
	root := t.TempDir()
	localeDir := filepath.Join(root, "en-US", "images")
	if err := os.MkdirAll(filepath.Join(localeDir, "phoneScreenshots"), 0o755); err != nil {
		t.Fatalf("mkdir screenshots: %v", err)
	}
	if err := os.WriteFile(filepath.Join(localeDir, "phoneScreenshots", "1.png"), []byte("same"), 0o644); err != nil {
		t.Fatalf("write screenshot: %v", err)
	}
	if err := os.WriteFile(filepath.Join(localeDir, "featureGraphic.png"), []byte("local-feature"), 0o644); err != nil {
		t.Fatalf("write feature graphic: %v", err)
	}

	backend := &fakeMediaBackend{
		locales: []string{"en-US"},
		images: map[string]map[string][]remoteImage{
			"en-US": {
				"phoneScreenshots": {
					{ID: "same", Sha256: sha256Hex(t, []byte("same")), URL: "https://example.com/same.png"},
				},
				"featureGraphic": {
					{ID: "remote-feature", Sha256: sha256Hex(t, []byte("remote-feature")), URL: "https://example.com/remote-feature.png"},
				},
			},
		},
	}

	plan, err := buildMediaPlan(context.Background(), backend, "com.example.app", "edit-1", root, "")
	if err != nil {
		t.Fatalf("buildMediaPlan: %v", err)
	}
	if plan.Summary.Keep != 1 {
		t.Fatalf("expected 1 kept asset, got %d", plan.Summary.Keep)
	}
	if plan.Summary.Upload != 1 {
		t.Fatalf("expected 1 upload, got %d", plan.Summary.Upload)
	}
	if plan.Summary.RemoteOnly != 1 {
		t.Fatalf("expected 1 remote-only asset, got %d", plan.Summary.RemoteOnly)
	}
}

func TestPullMedia_WritesRemoteAssets(t *testing.T) {
	root := t.TempDir()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("pulled-image"))
	}))
	t.Cleanup(server.Close)

	backend := &fakeMediaBackend{
		locales: []string{"en-US"},
		images: map[string]map[string][]remoteImage{
			"en-US": {
				"featureGraphic": {
					{ID: "feature-1", URL: server.URL + "/feature.png"},
				},
			},
		},
	}

	result, err := pullMedia(context.Background(), backend, "com.example.app", "edit-1", root, "")
	if err != nil {
		t.Fatalf("pullMedia: %v", err)
	}
	if len(result.Files) != 1 {
		t.Fatalf("expected 1 downloaded file, got %d", len(result.Files))
	}
	data, err := os.ReadFile(result.Files[0])
	if err != nil {
		t.Fatalf("read downloaded file: %v", err)
	}
	if string(data) != "pulled-image" {
		t.Fatalf("unexpected downloaded content: %q", string(data))
	}
}

func TestSyncCommand_HappyPath(t *testing.T) {
	root := t.TempDir()
	localeDir := filepath.Join(root, "en-US", "images")
	if err := os.MkdirAll(filepath.Join(localeDir, "phoneScreenshots"), 0o755); err != nil {
		t.Fatalf("mkdir screenshots: %v", err)
	}
	if err := os.WriteFile(filepath.Join(localeDir, "phoneScreenshots", "1.png"), []byte("same"), 0o644); err != nil {
		t.Fatalf("write screenshot: %v", err)
	}

	fake := &fakeMediaBackend{
		locales: []string{"en-US"},
		images: map[string]map[string][]remoteImage{
			"en-US": {
				"phoneScreenshots": {
					{ID: "same", Sha256: sha256Hex(t, []byte("same")), URL: "https://example.com/same.png"},
				},
			},
		},
	}

	originalFactory := newMediaBackend
	newMediaBackend = func(ctx context.Context) (mediaBackend, error) { return fake, nil }
	t.Cleanup(func() { newMediaBackend = originalFactory })

	cmd := SyncCommand()
	if err := cmd.FlagSet.Parse([]string{"--package", "com.example.app", "--edit", "edit-1", "--dir", root}); err != nil {
		t.Fatalf("parse flags: %v", err)
	}
	out, err := captureCommandOutput(func() error {
		return cmd.Exec(context.Background(), nil)
	})
	if err != nil {
		t.Fatalf("sync exec: %v", err)
	}
	if len(fake.uploads) != 0 {
		t.Fatalf("expected no uploads for identical media, got %v", fake.uploads)
	}
	var summary syncResult
	if err := json.Unmarshal([]byte(out), &summary); err != nil {
		t.Fatalf("decode sync summary: %v", err)
	}
	if summary.Kept != 1 {
		t.Fatalf("expected 1 kept asset, got %d", summary.Kept)
	}
}

func captureCommandOutput(fn func() error) (string, error) {
	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		return "", err
	}
	os.Stdout = w

	var buf bytes.Buffer
	done := make(chan struct{})
	go func() {
		defer close(done)
		_, _ = io.Copy(&buf, r)
	}()

	runErr := fn()
	_ = w.Close()
	os.Stdout = orig
	<-done
	_ = r.Close()
	return buf.String(), runErr
}

func sha256Hex(t *testing.T, data []byte) string {
	t.Helper()
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
