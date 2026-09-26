package updatecmd

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/tamtom/play-console-cli/internal/cli/shared"
	"github.com/tamtom/play-console-cli/internal/version"
)

func TestSelfUpdateInstallation(t *testing.T) {
	dir := t.TempDir()
	candidate := filepath.Join(dir, "candidate")
	if runtime.GOOS == "windows" {
		candidate += ".exe"
	}
	build := exec.Command("go", "build", "-ldflags", "-X github.com/tamtom/play-console-cli/internal/version.Version=1.0.0", "-o", candidate, "../../..")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build candidate: %v\n%s", err, out)
	}
	body, err := os.ReadFile(candidate)
	if err != nil {
		t.Fatal(err)
	}
	oldPath, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	oldBody, err := os.ReadFile(oldPath)
	if err != nil {
		t.Fatal(err)
	}
	asset := fmt.Sprintf("gplay-%s-%s", runtime.GOOS, runtime.GOARCH)
	if runtime.GOOS == "windows" {
		asset += ".exe"
	}
	digest := fmt.Sprintf("%x", sha256.Sum256(body))
	for _, mode := range []string{"valid", "mismatch", "missing_asset", "missing_checksums", "check", "current"} {
		t.Run(mode, func(t *testing.T) {
			home := t.TempDir()
			target := filepath.Join(home, "gplay")
			if runtime.GOOS == "windows" {
				target += ".exe"
			}
			if err := os.WriteFile(target, oldBody, 0o700); err != nil {
				t.Fatal(err)
			}
			var downloads atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/latest":
					assets := []map[string]string{{"name": asset, "browser_download_url": "http://" + r.Host + "/binary"}, {"name": "checksums.txt", "browser_download_url": "http://" + r.Host + "/checksums.txt"}}
					if mode == "missing_asset" {
						assets = assets[1:]
					}
					if mode == "missing_checksums" {
						assets = assets[:1]
					}
					_ = json.NewEncoder(w).Encode(map[string]any{"tag_name": "v1.0.0", "assets": assets})
				case "/checksums.txt":
					checksum := digest
					if mode == "mismatch" {
						checksum = strings.Repeat("0", 64)
					}
					fmt.Fprintf(w, "%s  %s\n", checksum, asset)
				case "/binary":
					downloads.Add(1)
					_, _ = w.Write(body)
				default:
					http.NotFound(w, r)
				}
			}))
			defer server.Close()
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			child := exec.CommandContext(ctx, target, "-test.run=^TestUpdateInstallerProcess$")
			child.Env = append(os.Environ(), "HOME="+home, "USERPROFILE="+home, "GPLAY_TEST_UPDATE_SERVER="+server.URL, "GPLAY_TEST_UPDATE_MODE="+mode)
			out, err := child.CombinedOutput()
			if err != nil {
				t.Fatalf("installer process: %v\n%s", err, out)
			}
			installed, err := os.ReadFile(target)
			if err != nil {
				t.Fatal(err)
			}
			if (mode == "check" || mode == "current") && downloads.Load() != 0 {
				t.Fatal("read-only update downloaded an executable")
			}
			if mode != "valid" {
				if !bytes.Equal(installed, oldBody) {
					t.Fatal("read-only or failed update replaced the existing executable")
				}
				return
			}
			if !bytes.Equal(installed, body) {
				t.Fatal("verified candidate was not installed")
			}
			run := exec.CommandContext(ctx, target, "--version")
			out, err = run.CombinedOutput()
			if err != nil || !strings.HasPrefix(string(out), "1.0.0 ") {
				t.Fatalf("updated executable: %v\n%s", err, out)
			}
		})
	}
}

type installerHTTPTransport func(*http.Request) (*http.Response, error)

func (f installerHTTPTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestUpdateInstallerProcess(t *testing.T) {
	endpoint := os.Getenv("GPLAY_TEST_UPDATE_SERVER")
	if endpoint == "" {
		t.Skip("isolated installer process")
	}
	base := http.DefaultTransport
	http.DefaultTransport = installerHTTPTransport(func(r *http.Request) (*http.Response, error) {
		if r.URL.Host == "api.github.com" {
			replacement, err := http.NewRequestWithContext(r.Context(), "GET", endpoint+"/latest", nil)
			if err != nil {
				return nil, err
			}
			return base.RoundTrip(replacement)
		}
		return base.RoundTrip(r)
	})
	defer func() { http.DefaultTransport = base }()
	var stderr bytes.Buffer
	ctx := shared.ContextWithIO(context.Background(), io.Discard, &stderr)
	mode := os.Getenv("GPLAY_TEST_UPDATE_MODE")
	version.Version = "1.0.0"
	args := []string{"--force"}
	if mode == "check" {
		version.Version = "0.9.0"
		args = []string{"--check"}
	}
	if mode == "current" {
		args = nil
	}
	err := UpdateCommand().ParseAndRun(ctx, args)
	if mode == "check" && (!strings.Contains(stderr.String(), "Current: 0.9.0") || !strings.Contains(stderr.String(), "Latest:  1.0.0")) {
		t.Fatalf("check did not report versions: %s", stderr.String())
	}
	if mode == "valid" || mode == "check" || mode == "current" {
		if err != nil {
			t.Fatalf("update: %v\n%s", err, stderr.String())
		}
	} else if err == nil {
		t.Fatalf("unsafe update reported success: %s", stderr.String())
	}
}
