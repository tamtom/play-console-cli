package bundleanalysis

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func writeFakeAAB(t *testing.T, path string, entries map[string][]byte) {
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

func TestAnalyzeBasic(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app.aab")
	writeFakeAAB(t, path, map[string][]byte{
		"base/manifest/AndroidManifest.xml":      bytes.Repeat([]byte("a"), 1000),
		"base/dex/classes.dex":                   bytes.Repeat([]byte("b"), 5000),
		"base/res/layout/main.xml":               bytes.Repeat([]byte("c"), 2000),
		"base/lib/arm64-v8a/libapp.so":           bytes.Repeat([]byte("d"), 8000),
		"base/assets/images/cover.png":           bytes.Repeat([]byte("e"), 3000),
		"feature_payment/dex/classes.dex":        bytes.Repeat([]byte("f"), 1500),
		"feature_payment/res/values/strings.xml": bytes.Repeat([]byte("g"), 400),
		"META-INF/MANIFEST.MF":                   bytes.Repeat([]byte("h"), 100),
	})

	a, err := Analyze(path, Options{TopFiles: 3})
	if err != nil {
		t.Fatal(err)
	}
	if a.TotalFiles != 8 {
		t.Errorf("TotalFiles=%d want 8", a.TotalFiles)
	}
	if len(a.Modules) < 3 {
		t.Errorf("want >=3 modules, got %d (%v)", len(a.Modules), a.Modules)
	}
	if len(a.Buckets) == 0 {
		t.Error("want buckets populated")
	}
	if len(a.LargestFiles) != 3 {
		t.Errorf("want top 3 files, got %d", len(a.LargestFiles))
	}
	// Largest file should be the 8000-byte native lib.
	if a.LargestFiles[0].UncompressedB != 8000 {
		t.Errorf("unexpected largest file: %+v", a.LargestFiles[0])
	}
}

func TestAnalyzeMissingFile(t *testing.T) {
	_, err := Analyze(filepath.Join(t.TempDir(), "missing.aab"), Options{})
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestBucketForEntry(t *testing.T) {
	cases := map[string]Bucket{
		"base/dex/classes.dex":              BucketDex,
		"base/lib/arm64-v8a/libapp.so":      BucketNative,
		"lib/x86/libc.so":                   BucketNative,
		"base/res/layout/main.xml":          BucketResources,
		"base/resources.pb":                 BucketResources,
		"feature/assets/file.json":          BucketAssets,
		"META-INF/MANIFEST.MF":              BucketMeta,
		"base/manifest/AndroidManifest.xml": BucketManifest,
		"kotlin/reflect/something.bin":      BucketKotlin,
		"random.txt":                        BucketOther,
	}
	for name, want := range cases {
		if got := bucketForEntry(name); got != want {
			t.Errorf("bucketForEntry(%q)=%s want %s", name, got, want)
		}
	}
}

func TestModuleForEntry(t *testing.T) {
	cases := map[string]string{
		"base/dex/classes.dex":       "base",
		"feature_payments/dex/a.dex": "feature_payments",
		"META-INF/MANIFEST.MF":       "meta",
		"top-level-file.xml":         "apk",
	}
	for in, want := range cases {
		if got := moduleForEntry(in); got != want {
			t.Errorf("moduleForEntry(%q)=%q want %q", in, got, want)
		}
	}
}

func TestCompareDiffsAndRegression(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "base.aab")
	cand := filepath.Join(dir, "cand.aab")
	writeFakeAAB(t, base, map[string][]byte{
		"base/dex/classes.dex": bytes.Repeat([]byte("x"), 1000),
	})
	writeFakeAAB(t, cand, map[string][]byte{
		"base/dex/classes.dex":            bytes.Repeat([]byte("x"), 5000),
		"feature_payment/dex/classes.dex": bytes.Repeat([]byte("y"), 2000),
	})
	baseA, _ := Analyze(base, Options{})
	candA, _ := Analyze(cand, Options{})

	diff := Compare(baseA, candA, 3000)
	if diff.DeltaUncompr <= 0 {
		t.Errorf("expected positive delta, got %d", diff.DeltaUncompr)
	}
	if !diff.Regression {
		t.Errorf("expected regression flag when delta (%d) > threshold (3000)", diff.DeltaUncompr)
	}
	if len(diff.PerModule) == 0 {
		t.Error("expected per-module diff entries")
	}

	// Verify the candidate-only module is flagged.
	foundAdded := false
	for _, m := range diff.PerModule {
		if m.Module == "feature_payment" && m.AddedInCandidate {
			foundAdded = true
		}
	}
	if !foundAdded {
		t.Error("expected feature_payment marked AddedInCandidate")
	}
}

func TestCompareNoRegressionWithoutThreshold(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "base.aab")
	cand := filepath.Join(dir, "cand.aab")
	writeFakeAAB(t, base, map[string][]byte{"base/dex/classes.dex": []byte("aaaa")})
	writeFakeAAB(t, cand, map[string][]byte{"base/dex/classes.dex": bytes.Repeat([]byte("a"), 100000)})
	baseA, _ := Analyze(base, Options{})
	candA, _ := Analyze(cand, Options{})
	d := Compare(baseA, candA, 0)
	if d.Regression {
		t.Error("no threshold -> no regression flag")
	}
}

func TestParseSizeThreshold(t *testing.T) {
	cases := map[string]int64{
		"":     0,
		"100":  100,
		"500B": 500,
		"2K":   2048,
		"2KB":  2048,
		"3M":   3 * 1024 * 1024,
		"1GB":  1024 * 1024 * 1024,
	}
	for in, want := range cases {
		got, err := ParseSizeThreshold(in)
		if err != nil {
			t.Errorf("%q: unexpected error %v", in, err)
			continue
		}
		if got != want {
			t.Errorf("%q: got %d want %d", in, got, want)
		}
	}
	if _, err := ParseSizeThreshold("weird"); err == nil {
		t.Error("expected error for non-numeric")
	}
}
