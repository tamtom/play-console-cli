// verifyupgrade checks a release candidate over an isolated installation of
// the latest published stable binary. It never changes the operator's install.
package main

import (
	"context"
	"crypto/sha256"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/tamtom/play-console-cli/internal/update"
)

func main() {
	candidate := flag.String("candidate", "", "Native release candidate executable")
	expected := flag.String("version", "", "Expected candidate version")
	flag.Parse()
	if err := verify(*candidate, *expected); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func verify(candidate, expected string) error {
	if candidate == "" || expected == "" {
		return fmt.Errorf("--candidate and --version are required")
	}
	candidate, err := filepath.Abs(candidate)
	if err != nil {
		return err
	}
	work, err := os.MkdirTemp("", "gplay-upgrade-check-")
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(work) }()
	// The release checker caches results under HOME; contain those writes too.
	for _, key := range []string{"HOME", "USERPROFILE"} {
		if err := os.Setenv(key, work); err != nil {
			return err
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	stable, err := update.CheckForUpdate(ctx, update.Options{ForceCheck: true})
	if err != nil {
		return err
	}
	if stable == nil {
		return fmt.Errorf("no previous stable release available")
	}
	previous, err := update.DownloadUpdate(ctx, stable)
	if err != nil {
		return fmt.Errorf("verify previous stable release: %w", err)
	}
	defer func() { _ = os.Remove(previous) }()
	name := "gplay"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	target := filepath.Join(work, name)
	if err := os.Rename(previous, target); err != nil {
		return err
	}
	if err := os.Chmod(target, 0o755); err != nil {
		return err
	}
	if err := checkVersion(ctx, target, stable.LatestVersion); err != nil {
		return err
	}

	file, err := os.Open(candidate)
	if err != nil {
		return err
	}
	hash := sha256.New()
	_, copyErr := io.Copy(hash, file)
	closeErr := file.Close()
	if copyErr != nil {
		return copyErr
	}
	if closeErr != nil {
		return closeErr
	}
	checksum := fmt.Sprintf("%x", hash.Sum(nil))
	asset := fmt.Sprintf("gplay-%s-%s", runtime.GOOS, runtime.GOARCH)
	if runtime.GOOS == "windows" {
		asset += ".exe"
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/checksums.txt" {
			fmt.Fprintf(w, "%s  %s\n", checksum, asset)
			return
		}
		if r.URL.Path == "/"+asset {
			http.ServeFile(w, r, candidate)
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()
	verified, err := update.DownloadUpdate(ctx, &update.UpdateInfo{AssetName: asset, DownloadURL: server.URL + "/" + asset, ChecksumURL: server.URL + "/checksums.txt"})
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(verified) }()
	if err := update.ApplyUpdate(verified, target); err != nil {
		return err
	}
	if err := checkVersion(ctx, target, expected); err != nil {
		return err
	}
	fmt.Printf("Verified native upgrade %s -> %s on %s/%s\n", stable.LatestVersion, expected, runtime.GOOS, runtime.GOARCH)
	return nil
}

func checkVersion(ctx context.Context, path, expected string) error {
	out, err := exec.CommandContext(ctx, path, "--version").CombinedOutput()
	if err != nil {
		return fmt.Errorf("run installed binary: %w: %s", err, out)
	}
	fields := strings.Fields(string(out))
	if len(fields) == 0 || strings.TrimPrefix(fields[0], "v") != strings.TrimPrefix(expected, "v") {
		return fmt.Errorf("installed version %q does not match %s", out, expected)
	}
	return nil
}
