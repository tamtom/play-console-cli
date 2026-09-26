package update

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tamtom/play-console-cli/internal/version"
)

type testTransport func(*http.Request) (*http.Response, error)

func (f testTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestDownloadRequiresReleaseChecksum(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, "untrusted bytes") }))
	defer server.Close()
	path, err := DownloadUpdate(context.Background(), &UpdateInfo{DownloadURL: server.URL, LatestVersion: "1.0.0"})
	if path != "" {
		t.Cleanup(func() { _ = os.Remove(path) })
	}
	if err == nil {
		t.Fatal("accepted download without checksum")
	}
}

func TestDownloadVerifiesSelectedReleaseAsset(t *testing.T) {
	for _, checksum := range []string{"valid", "mismatch", "missing", "unavailable", "duplicate", "malformed"} {
		t.Run(checksum, func(t *testing.T) {
			t.Setenv("HOME", t.TempDir())
			t.Setenv("USERPROFILE", t.TempDir())
			body := "candidate executable bytes"
			digest := fmt.Sprintf("%x", sha256.Sum256([]byte(body)))
			asset := getBinaryName()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/checksums.txt" {
					switch checksum {
					case "valid":
						fmt.Fprintf(w, "%s  %s\n", digest, asset)
					case "mismatch":
						fmt.Fprintf(w, "%s  %s\n", strings.Repeat("0", 64), asset)
					case "missing":
						fmt.Fprintf(w, "%s  other-asset\n", digest)
					case "unavailable":
						http.Error(w, "missing", 404)
					case "duplicate":
						fmt.Fprintf(w, "%s  %s\n%s  %s\n", digest, asset, digest, asset)
					case "malformed":
						fmt.Fprintf(w, "invalid  %s\n", asset)
					}
					return
				}
				fmt.Fprint(w, body)
			}))
			defer server.Close()
			base := http.DefaultTransport
			http.DefaultTransport = testTransport(func(r *http.Request) (*http.Response, error) {
				if r.URL.Host == "api.github.com" {
					response := fmt.Sprintf(`{"tag_name":"v1.0.0","assets":[{"name":%q,"browser_download_url":%q},{"name":"checksums.txt","browser_download_url":%q}]}`, asset, server.URL+"/"+asset, server.URL+"/checksums.txt")
					return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(response))}, nil
				}
				return base.RoundTrip(r)
			})
			t.Cleanup(func() { http.DefaultTransport = base })
			info, err := CheckForUpdate(context.Background(), Options{ForceCheck: true})
			if err != nil {
				t.Fatal(err)
			}
			path, err := DownloadUpdate(context.Background(), info)
			if path != "" {
				defer func() { _ = os.Remove(path) }()
			}
			if checksum == "valid" {
				if err != nil {
					t.Fatal(err)
				}
				got, err := os.ReadFile(path)
				if err != nil || string(got) != body {
					t.Fatalf("download=%q err=%v", got, err)
				}
			} else if err == nil || path != "" {
				t.Fatalf("accepted %s checksum: path=%s err=%v", checksum, path, err)
			}
		})
	}
}

func TestReleaseVersionComparison(t *testing.T) {
	for _, tc := range []struct {
		current string
		newer   bool
	}{
		{"0.9.0", true},
		{"1.0.0-rc.1", true},
		{"1.0.0", false},
		{"1.0.0+build.5", false},
		{"1.1.0", false},
		{"dev", false},
		// git describe versions are development builds, not releases.
		{"0.10.0-35-gc896cf6", false},
		{"0.10.0-35-gc896cf6-dirty", false},
		{"0.9.0-dirty", false},
	} {
		t.Run(tc.current, func(t *testing.T) {
			t.Setenv("GPLAY_NO_UPDATE", "")
			t.Setenv("HOME", t.TempDir())
			t.Setenv("USERPROFILE", t.TempDir())
			base := http.DefaultTransport
			http.DefaultTransport = testTransport(func(*http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"tag_name":"v1.0.0"}`))}, nil
			})
			t.Cleanup(func() { http.DefaultTransport = base })
			info, err := CheckForUpdate(context.Background(), Options{CurrentVersion: tc.current, ForceCheck: true})
			if err != nil || info == nil || info.IsNewer != tc.newer {
				t.Fatalf("info=%+v err=%v wantNewer=%v", info, err, tc.newer)
			}
		})
	}
}

func TestCheckCachesSuccessAndFailure(t *testing.T) {
	for _, fail := range []bool{false, true} {
		t.Run(fmt.Sprint(fail), func(t *testing.T) {
			t.Setenv("GPLAY_NO_UPDATE", "")
			t.Setenv("HOME", t.TempDir())
			t.Setenv("USERPROFILE", t.TempDir())
			previous := version.Version
			version.Version = "0.9.0"
			t.Cleanup(func() { version.Version = previous })
			calls := 0
			base := http.DefaultTransport
			http.DefaultTransport = testTransport(func(r *http.Request) (*http.Response, error) {
				calls++
				if fail {
					return nil, fmt.Errorf("offline")
				}
				return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(`{"tag_name":"v1.0.0"}`))}, nil
			})
			t.Cleanup(func() { http.DefaultTransport = base })
			for i := range 2 {
				info, err := CheckForUpdate(context.Background(), Options{})
				switch {
				case fail && i == 0 && err == nil:
					t.Fatal("the failed check returned no error")
				case fail && info != nil:
					t.Fatalf("failed check cached as success: %+v", info)
				case !fail && (err != nil || info == nil || !info.IsNewer):
					t.Fatalf("cached update unavailable: info=%v err=%v", info, err)
				}
			}
			// A failed check is also cached, for FailedCheckInterval, so that
			// an offline machine does not wait for the network on each command.
			if calls != 1 {
				t.Fatalf("requests=%d want=1", calls)
			}
		})
	}
}

func TestCheckSkipsDevelopmentBuildsAndDisabledChecks(t *testing.T) {
	for _, tc := range []struct{ current, noUpdate string }{
		{"0.10.0-35-gc896cf6", ""},
		{"dev", ""},
		{"0.9.0", "true"},
		{"0.9.0", "YES"},
	} {
		t.Run(tc.current+"_"+tc.noUpdate, func(t *testing.T) {
			t.Setenv("GPLAY_NO_UPDATE", tc.noUpdate)
			t.Setenv("HOME", t.TempDir())
			t.Setenv("USERPROFILE", t.TempDir())
			calls := 0
			base := http.DefaultTransport
			http.DefaultTransport = testTransport(func(*http.Request) (*http.Response, error) {
				calls++
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"tag_name":"v1.0.0"}`))}, nil
			})
			t.Cleanup(func() { http.DefaultTransport = base })
			info, err := CheckForUpdate(context.Background(), Options{CurrentVersion: tc.current})
			if info != nil || err != nil || calls != 0 {
				t.Fatalf("info=%+v err=%v requests=%d, want no check", info, err, calls)
			}
		})
	}
}

func TestIsReleaseVersion(t *testing.T) {
	for v, want := range map[string]bool{
		"1.0.0": true, "v1.0.0": true, "1.0.0-rc.1": true,
		"0.10.0-35-gc896cf6": false, "0.10.0-35-gc896cf6-dirty": false, "1.0.0-dirty": false, "dev": false, "c896cf6": false,
	} {
		if got := IsReleaseVersion(v); got != want {
			t.Errorf("IsReleaseVersion(%q) = %v, want %v", v, got, want)
		}
	}
}

func TestRemoveStaleBackupsKeepsOtherFiles(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"gplay.exe", "gplay.exe.old-A1", "gplay.exe.old-B2", "gplay.exe.new-C3", "other.exe.old-D4"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = root.Close() }()

	removeStaleBackups(root, "gplay.exe")

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	if got := strings.Join(names, ","); got != "gplay.exe,gplay.exe.new-C3,other.exe.old-D4" {
		t.Fatalf("files after cleanup = %s", got)
	}
}
