package preflight

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func buildAAB(t *testing.T, path string, entries map[string][]byte) {
	t.Helper()
	buf := &bytes.Buffer{}
	w := zip.NewWriter(buf)
	for name, body := range entries {
		f, err := w.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.Write(body); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
}

func minimalAAB() map[string][]byte {
	return map[string][]byte{
		"base/manifest/AndroidManifest.xml": []byte("manifest-bytes"),
		"base/resources.pb":                 []byte("res"),
		"base/dex/classes.dex":              bytes.Repeat([]byte("a"), 100),
		"base/lib/arm64-v8a/libapp.so":      bytes.Repeat([]byte("n"), 100),
	}
}

func TestScanMinimalClean(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app.aab")
	buildAAB(t, path, minimalAAB())
	r, err := Scan(path, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if r.Errors != 0 {
		t.Errorf("expected 0 errors, got %d: %+v", r.Errors, r.Findings)
	}
}

func TestScanDetectsMissingManifest(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app.aab")
	buildAAB(t, path, map[string][]byte{
		"base/resources.pb": []byte("res"),
	})
	r, _ := Scan(path, Options{})
	foundManifest := false
	foundResources := false
	for _, f := range r.Findings {
		if f.Check == "manifest" && f.Severity == SeverityError {
			foundManifest = true
		}
		if f.Check == "resources" {
			foundResources = true
		}
	}
	if !foundManifest {
		t.Error("expected manifest error")
	}
	_ = foundResources // resources exists so no error
}

func TestScanDetectsMissingResources(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app.aab")
	buildAAB(t, path, map[string][]byte{
		"base/manifest/AndroidManifest.xml": []byte("x"),
	})
	r, _ := Scan(path, Options{})
	has := false
	for _, f := range r.Findings {
		if f.Check == "resources" && f.Severity == SeverityError {
			has = true
		}
	}
	if !has {
		t.Error("expected resources error")
	}
}

func TestScanDetectsMissingArm64(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app.aab")
	entries := minimalAAB()
	delete(entries, "base/lib/arm64-v8a/libapp.so")
	entries["base/lib/armeabi-v7a/libapp.so"] = []byte("x")
	buildAAB(t, path, entries)
	r, _ := Scan(path, Options{})
	has := false
	for _, f := range r.Findings {
		if f.Check == "native_libs" && f.Severity == SeverityError {
			has = true
		}
	}
	if !has {
		t.Error("expected native_libs error for missing arm64")
	}
}

func TestScanDetectsBundleSize(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app.aab")
	buildAAB(t, path, minimalAAB())
	r, _ := Scan(path, Options{MaxBundleBytes: 1}) // force tiny limit
	has := false
	for _, f := range r.Findings {
		if f.Check == "bundle_size" {
			has = true
		}
	}
	if !has {
		t.Error("expected bundle_size warning")
	}
}

func TestScanSecretDetection(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app.aab")
	entries := minimalAAB()
	// Planting a Google API key in a resource-ish file.
	entries["base/res/values/secrets.xml"] = []byte(`<string name="k">AIza` + strings.Repeat("x", 35) + `</string>`)
	buildAAB(t, path, entries)
	r, _ := Scan(path, Options{})
	has := false
	for _, f := range r.Findings {
		if f.Check == "secrets" && f.Severity == SeverityError {
			has = true
		}
	}
	if !has {
		t.Error("expected secret detection")
	}
}

func TestScanSecretDetectionDisabled(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app.aab")
	entries := minimalAAB()
	entries["base/res/values/secrets.xml"] = []byte(`AIza` + strings.Repeat("x", 35))
	buildAAB(t, path, entries)
	r, _ := Scan(path, Options{SkipSecretScan: true})
	for _, f := range r.Findings {
		if f.Check == "secrets" {
			t.Errorf("secret scan should be skipped, got %+v", f)
		}
	}
}

func TestScanDebuggable(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app.aab")
	entries := minimalAAB()
	entries["base/manifest/AndroidManifest.xml"] = []byte(`android:debuggable="true" ...`)
	buildAAB(t, path, entries)
	r, _ := Scan(path, Options{})
	has := false
	for _, f := range r.Findings {
		if f.Check == "debuggable" && f.Severity == SeverityError {
			has = true
		}
	}
	if !has {
		t.Error("expected debuggable error")
	}
}

func TestScanDangerousPermissionLogs(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app.aab")
	entries := minimalAAB()
	entries["base/manifest/AndroidManifest.xml"] = []byte(`android.permission.READ_SMS`)
	buildAAB(t, path, entries)
	r, _ := Scan(path, Options{})
	has := false
	for _, f := range r.Findings {
		if f.Check == "dangerous_permissions" {
			has = true
		}
	}
	if !has {
		t.Error("expected dangerous permission info finding")
	}
}

func TestScanMisplacedFiles(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app.aab")
	entries := minimalAAB()
	entries["base/assets/.DS_Store"] = []byte{0}
	buildAAB(t, path, entries)
	r, _ := Scan(path, Options{})
	has := false
	for _, f := range r.Findings {
		if f.Check == "misplaced_files" {
			has = true
		}
	}
	if !has {
		t.Error("expected misplaced_files warning")
	}
}

func TestScanMissingFile(t *testing.T) {
	_, err := Scan(filepath.Join(t.TempDir(), "missing.aab"), Options{})
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestHasErrors(t *testing.T) {
	r := &Report{}
	if r.HasErrors() {
		t.Error("empty report should not have errors")
	}
	r.Errors = 1
	if !r.HasErrors() {
		t.Error("expected HasErrors=true")
	}
}
