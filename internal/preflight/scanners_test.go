package preflight

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// encodePNG renders a blank PNG of the requested dimensions.
func encodePNG(t *testing.T, w, h int) []byte {
	t.Helper()
	var b bytes.Buffer
	if err := png.Encode(&b, image.NewRGBA(image.Rect(0, 0, w, h))); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

// --- fixtures ---------------------------------------------------------------

// cleanManifest is a baseline that produces no errors, so each test can
// change exactly one thing and attribute the finding to that change.
func cleanManifest() pbElem {
	return pbElem{
		name: "manifest",
		attrs: []pbAttr{
			{name: "package", value: "com.acme.app"},
			{ns: AndroidNS, name: "versionCode", compiled: pbPrimInt(10)},
			{ns: AndroidNS, name: "versionName", value: "1.0.0"},
		},
		children: []pbElem{
			{name: "uses-sdk", attrs: []pbAttr{
				{ns: AndroidNS, name: "minSdkVersion", compiled: pbPrimInt(24)},
				{ns: AndroidNS, name: "targetSdkVersion", compiled: pbPrimInt(35)},
			}},
			{name: "application", children: []pbElem{
				{name: "activity", attrs: []pbAttr{
					{ns: AndroidNS, name: "name", value: ".MainActivity"},
					{ns: AndroidNS, name: "exported", compiled: pbPrimBool(true)},
				}, children: []pbElem{
					{name: "intent-filter", children: []pbElem{
						{name: "action", attrs: []pbAttr{{ns: AndroidNS, name: "name", value: "android.intent.action.MAIN"}}},
						{name: "category", attrs: []pbAttr{{ns: AndroidNS, name: "name", value: "android.intent.category.LAUNCHER"}}},
					}},
				}},
			}},
		},
	}
}

// withPermissions returns a copy of the manifest with permissions prepended.
func withPermissions(m pbElem, names ...string) pbElem {
	var perms []pbElem
	for _, n := range names {
		perms = append(perms, pbElem{name: "uses-permission", attrs: []pbAttr{
			{ns: AndroidNS, name: "name", value: n},
		}})
	}
	m.children = append(perms, m.children...)
	return m
}

// appNode returns a pointer to the <application> child so tests can mutate it.
func appNode(m *pbElem) *pbElem {
	for i := range m.children {
		if m.children[i].name == "application" {
			return &m.children[i]
		}
	}
	panic("no application element")
}

// buildTestBundle writes an AAB containing the given manifest plus extras.
func buildTestBundle(t *testing.T, manifest []byte, extra map[string][]byte) string {
	t.Helper()
	entries := map[string][]byte{
		"base/manifest/AndroidManifest.xml": manifest,
		"base/resources.pb":                 []byte("res"),
		"base/dex/classes.dex":              bytes.Repeat([]byte("a"), 64),
	}
	for k, v := range extra {
		entries[k] = v
	}
	path := filepath.Join(t.TempDir(), "app.aab")
	buildAAB(t, path, entries)
	return path
}

// scanClean scans a bundle built from the given manifest and extras.
func scanFixture(t *testing.T, m pbElem, extra map[string][]byte, opts Options) *Report {
	t.Helper()
	r, err := Scan(buildTestBundle(t, pbNode(m), extra), opts)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

// findingFor returns the first finding matching a check name.
func findingFor(r *Report, check string) (Finding, bool) {
	for _, f := range r.Findings {
		if f.Check == check {
			return f, true
		}
	}
	return Finding{}, false
}

// requireFinding asserts a check fired at the expected severity.
func requireFinding(t *testing.T, r *Report, check string, sev Severity) Finding {
	t.Helper()
	f, ok := findingFor(r, check)
	if !ok {
		t.Fatalf("expected check %q, got %+v", check, r.Findings)
	}
	if f.Severity != sev {
		t.Fatalf("check %q severity = %s, want %s", check, f.Severity, sev)
	}
	return f
}

// requireNoFinding asserts a check did not fire.
func requireNoFinding(t *testing.T, r *Report, check string) {
	t.Helper()
	if f, ok := findingFor(r, check); ok {
		t.Fatalf("did not expect check %q, got %+v", check, f)
	}
}

// --- baseline ---------------------------------------------------------------

func TestCleanBundleHasNoErrors(t *testing.T) {
	r := scanFixture(t, cleanManifest(), nil, Options{})
	if r.Errors != 0 {
		t.Fatalf("expected no errors, got %d: %+v", r.Errors, r.Findings)
	}
	if r.Package != "com.acme.app" || r.TargetSdk != 35 || r.VersionCode != 10 {
		t.Errorf("report metadata = %+v", r)
	}
	if r.Format != formatAAB {
		t.Errorf("format = %q, want aab", r.Format)
	}
}

func TestReportRecordsEveryScanner(t *testing.T) {
	r := scanFixture(t, cleanManifest(), nil, Options{})
	if len(r.Scanners) != len(ScannerIDs()) {
		t.Fatalf("scanners recorded = %d, want %d", len(r.Scanners), len(ScannerIDs()))
	}
	byID := map[string]ScannerRun{}
	for _, s := range r.Scanners {
		byID[s.ID] = s
	}
	if !byID["metadata"].Skipped || byID["metadata"].Reason == "" {
		t.Errorf("metadata should be skipped without --listings-dir: %+v", byID["metadata"])
	}
	if byID["manifest"].Skipped {
		t.Error("manifest scanner should run")
	}
}

// --- scanner selection ------------------------------------------------------

func TestScannerSelection(t *testing.T) {
	m := cleanManifest()
	appNode(&m).attrs = append(appNode(&m).attrs, pbAttr{
		ns: AndroidNS, name: "debuggable", compiled: pbPrimBool(true),
	})

	only := scanFixture(t, m, nil, Options{Only: []string{"policy"}})
	requireNoFinding(t, only, "debuggable")

	skipped := scanFixture(t, m, nil, Options{Skip: []string{"manifest"}})
	requireNoFinding(t, skipped, "debuggable")

	both := scanFixture(t, m, nil, Options{})
	requireFinding(t, both, "debuggable", SeverityError)
}

func TestUnknownScannerIDRejected(t *testing.T) {
	path := buildTestBundle(t, pbNode(cleanManifest()), nil)
	if _, err := Scan(path, Options{Only: []string{"nope"}}); err == nil {
		t.Error("expected error for unknown --only scanner")
	}
	if _, err := Scan(path, Options{Skip: []string{"nope"}}); err == nil {
		t.Error("expected error for unknown --skip scanner")
	}
}

// --- 1. manifest ------------------------------------------------------------

func TestManifestDebuggableAndTestOnly(t *testing.T) {
	m := cleanManifest()
	m.attrs = append(m.attrs, pbAttr{ns: AndroidNS, name: "testOnly", compiled: pbPrimBool(true)})
	appNode(&m).attrs = append(appNode(&m).attrs, pbAttr{ns: AndroidNS, name: "debuggable", compiled: pbPrimBool(true)})

	r := scanFixture(t, m, nil, Options{})
	requireFinding(t, r, "debuggable", SeverityError)
	requireFinding(t, r, "test_only", SeverityError)
}

func TestManifestPlaceholderPackageRejected(t *testing.T) {
	m := cleanManifest()
	m.attrs[0] = pbAttr{name: "package", value: "com.example.myapp"}
	r := scanFixture(t, m, nil, Options{})
	requireFinding(t, r, "package_name", SeverityError)
}

func TestManifestExportedWithoutDeclaration(t *testing.T) {
	m := cleanManifest()
	// A filtered receiver with no android:exported fails to install on 12+.
	appNode(&m).children = append(appNode(&m).children, pbElem{
		name:  "receiver",
		attrs: []pbAttr{{ns: AndroidNS, name: "name", value: ".BootReceiver"}},
		children: []pbElem{{name: "intent-filter", children: []pbElem{
			{name: "action", attrs: []pbAttr{{ns: AndroidNS, name: "name", value: "android.intent.action.BOOT_COMPLETED"}}},
		}}},
	})
	r := scanFixture(t, m, nil, Options{})
	requireFinding(t, r, "exported_undeclared", SeverityError)
}

func TestManifestExportedProviderGrantingURIsIsError(t *testing.T) {
	m := cleanManifest()
	appNode(&m).children = append(appNode(&m).children, pbElem{
		name: "provider",
		attrs: []pbAttr{
			{ns: AndroidNS, name: "name", value: ".LeakyProvider"},
			{ns: AndroidNS, name: "exported", compiled: pbPrimBool(true)},
			{ns: AndroidNS, name: "grantUriPermissions", compiled: pbPrimBool(true)},
		},
	})
	r := scanFixture(t, m, nil, Options{})
	requireFinding(t, r, "exported_provider", SeverityError)
	requireFinding(t, r, "exported_component", SeverityWarning)
}

func TestManifestLauncherActivityIsNotFlagged(t *testing.T) {
	r := scanFixture(t, cleanManifest(), nil, Options{})
	requireNoFinding(t, r, "exported_component")
}

func TestManifestForegroundServiceTypeNeedsPermission(t *testing.T) {
	m := cleanManifest()
	appNode(&m).children = append(appNode(&m).children, pbElem{
		name: "service",
		attrs: []pbAttr{
			{ns: AndroidNS, name: "name", value: ".SyncService"},
			{ns: AndroidNS, name: "foregroundServiceType", value: "dataSync"},
		},
	})

	missing := scanFixture(t, m, nil, Options{})
	f := requireFinding(t, missing, "foreground_service_type", SeverityError)
	if !strings.Contains(f.Message, "FOREGROUND_SERVICE_DATA_SYNC") {
		t.Errorf("expected the required permission in the message, got %q", f.Message)
	}

	granted := scanFixture(t, withPermissions(m, "android.permission.FOREGROUND_SERVICE_DATA_SYNC"), nil, Options{})
	requireNoFinding(t, granted, "foreground_service_type")
}

func TestManifestUndecodableFallsBackToHeuristics(t *testing.T) {
	path := buildTestBundle(t, []byte(`<application android:debuggable="true"/>`), nil)
	r, err := Scan(path, Options{})
	if err != nil {
		t.Fatal(err)
	}
	requireFinding(t, r, "manifest_undecodable", SeverityWarning)
	requireFinding(t, r, "debuggable", SeverityError)
}

// --- 2. permissions ---------------------------------------------------------

func TestPermissionsRestrictedAndSensitive(t *testing.T) {
	m := withPermissions(
		cleanManifest(),
		"android.permission.READ_SMS",
		"android.permission.CAMERA",
	)
	r := scanFixture(t, m, nil, Options{})

	var restricted, sensitive bool
	for _, f := range r.FindingsFor("permissions") {
		if f.Entry == "android.permission.READ_SMS" && f.Severity == SeverityWarning {
			restricted = true
			if f.Ref == "" {
				t.Error("restricted permission finding should link to the policy")
			}
		}
		if f.Entry == "android.permission.CAMERA" && f.Severity == SeverityInfo {
			sensitive = true
		}
	}
	if !restricted || !sensitive {
		t.Errorf("restricted=%v sensitive=%v: %+v", restricted, sensitive, r.Findings)
	}
}

func TestPermissionsLegacyStorageOnModernTarget(t *testing.T) {
	m := withPermissions(cleanManifest(), "android.permission.WRITE_EXTERNAL_STORAGE")
	r := scanFixture(t, m, nil, Options{})
	requireFinding(t, r, "legacy_storage_permission", SeverityWarning)
}

func TestPermissionsDuplicatesAndDeprecated(t *testing.T) {
	m := withPermissions(
		cleanManifest(),
		"android.permission.INTERNET",
		"android.permission.INTERNET",
		"android.permission.GET_TASKS",
	)
	r := scanFixture(t, m, nil, Options{})
	requireFinding(t, r, "duplicate_permission", SeverityInfo)
	requireFinding(t, r, "deprecated_permission", SeverityInfo)
}

// --- 3. native_libs ---------------------------------------------------------

// elf64SharedObject builds a minimal AArch64 ELF with one PT_LOAD segment at
// the given alignment, which is all the page-size check inspects.
func elf64SharedObject(align uint64) []byte {
	const (
		ehsize    = 64
		phentsize = 56
	)
	b := &bytes.Buffer{}

	b.Write([]byte{0x7f, 'E', 'L', 'F'})
	b.WriteByte(2) // ELFCLASS64
	b.WriteByte(1) // ELFDATA2LSB
	b.WriteByte(1) // EV_CURRENT
	b.WriteByte(0) // ELFOSABI_NONE
	b.Write(make([]byte, 8))

	write16 := func(v uint16) { _ = binary.Write(b, binary.LittleEndian, v) }
	write32 := func(v uint32) { _ = binary.Write(b, binary.LittleEndian, v) }
	write64 := func(v uint64) { _ = binary.Write(b, binary.LittleEndian, v) }

	write16(3)   // ET_DYN
	write16(183) // EM_AARCH64
	write32(1)   // EV_CURRENT
	write64(0)   // e_entry
	write64(ehsize)
	write64(0) // e_shoff
	write32(0) // e_flags
	write16(ehsize)
	write16(phentsize)
	write16(1) // e_phnum
	write16(64)
	write16(0) // e_shnum
	write16(0) // e_shstrndx

	write32(1) // PT_LOAD
	write32(5) // PF_R|PF_X
	write64(0) // p_offset
	write64(0) // p_vaddr
	write64(0) // p_paddr
	write64(0x1000)
	write64(0x1000)
	write64(align)

	return b.Bytes()
}

func TestNativeLibsPageAlignment(t *testing.T) {
	misaligned := map[string][]byte{
		"base/lib/arm64-v8a/libapp.so": elf64SharedObject(4096),
	}
	r := scanFixture(t, cleanManifest(), misaligned, Options{})
	f := requireFinding(t, r, "page_alignment", SeverityError) // targetSdk 35
	if f.Entry != "base/lib/arm64-v8a/libapp.so" {
		t.Errorf("entry = %q", f.Entry)
	}

	aligned := map[string][]byte{
		"base/lib/arm64-v8a/libapp.so": elf64SharedObject(16384),
	}
	ok := scanFixture(t, cleanManifest(), aligned, Options{})
	requireNoFinding(t, ok, "page_alignment")
}

func TestNativeLibsPageAlignmentIsWarningBelowTarget35(t *testing.T) {
	m := cleanManifest()
	m.children[0] = pbElem{name: "uses-sdk", attrs: []pbAttr{
		{ns: AndroidNS, name: "minSdkVersion", compiled: pbPrimInt(24)},
		{ns: AndroidNS, name: "targetSdkVersion", compiled: pbPrimInt(34)},
	}}
	r := scanFixture(t, m, map[string][]byte{
		"base/lib/arm64-v8a/libapp.so": elf64SharedObject(4096),
	}, Options{})
	requireFinding(t, r, "page_alignment", SeverityWarning)
}

func TestNativeLibs32BitOnlyIsError(t *testing.T) {
	r := scanFixture(t, cleanManifest(), map[string][]byte{
		"base/lib/armeabi-v7a/libapp.so": elf64SharedObject(16384),
	}, Options{})
	requireFinding(t, r, "native_libs", SeverityError)
}

func TestNativeLibsExtractNativeLibs(t *testing.T) {
	m := cleanManifest()
	appNode(&m).attrs = append(appNode(&m).attrs, pbAttr{
		ns: AndroidNS, name: "extractNativeLibs", compiled: pbPrimBool(true),
	})
	r := scanFixture(t, m, map[string][]byte{
		"base/lib/arm64-v8a/libapp.so": elf64SharedObject(16384),
	}, Options{})
	requireFinding(t, r, "extract_native_libs", SeverityWarning)
}

// --- 4. metadata ------------------------------------------------------------

// writePNG writes a PNG of the requested dimensions.
func writePNG(t *testing.T, path string, w, h int) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, encodePNG(t, w, h), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeText(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// validListings builds a complete, passing listings tree.
func validListings(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	locale := filepath.Join(dir, "en-US")
	writeText(t, filepath.Join(locale, "title.txt"), "Acme App")
	writeText(t, filepath.Join(locale, "short_description.txt"), "A short description")
	writeText(t, filepath.Join(locale, "full_description.txt"), "A full description")
	writePNG(t, filepath.Join(locale, "images", "icon.png"), 512, 512)
	writePNG(t, filepath.Join(locale, "images", "featureGraphic.png"), 1024, 500)
	writePNG(t, filepath.Join(locale, "images", "phoneScreenshots", "1.png"), 1080, 1920)
	writePNG(t, filepath.Join(locale, "images", "phoneScreenshots", "2.png"), 1080, 1920)
	return dir
}

func TestMetadataValidListingsPasses(t *testing.T) {
	r := scanFixture(t, cleanManifest(), nil, Options{ListingsDir: validListings(t)})
	if got := r.FindingsFor("metadata"); len(got) != 0 {
		t.Fatalf("expected no metadata findings, got %+v", got)
	}
}

func TestMetadataTitleTooLong(t *testing.T) {
	dir := validListings(t)
	writeText(t, filepath.Join(dir, "en-US", "title.txt"), strings.Repeat("x", 31))
	r := scanFixture(t, cleanManifest(), nil, Options{ListingsDir: dir})
	requireFinding(t, r, "listing_text_length", SeverityError)
}

func TestMetadataIconWrongDimensions(t *testing.T) {
	dir := validListings(t)
	writePNG(t, filepath.Join(dir, "en-US", "images", "icon.png"), 256, 256)
	r := scanFixture(t, cleanManifest(), nil, Options{ListingsDir: dir})
	f := requireFinding(t, r, "image_dimensions", SeverityError)
	if !strings.Contains(f.Message, "256x256") {
		t.Errorf("message should report actual dimensions, got %q", f.Message)
	}
}

func TestMetadataScreenshotTooSmallAndTooTall(t *testing.T) {
	dir := validListings(t)
	writePNG(t, filepath.Join(dir, "en-US", "images", "phoneScreenshots", "1.png"), 100, 200)
	writePNG(t, filepath.Join(dir, "en-US", "images", "phoneScreenshots", "2.png"), 400, 1200)
	r := scanFixture(t, cleanManifest(), nil, Options{ListingsDir: dir})
	requireFinding(t, r, "image_dimensions", SeverityError)
	requireFinding(t, r, "image_ratio", SeverityError)
}

func TestMetadataTooFewPhoneScreenshots(t *testing.T) {
	dir := validListings(t)
	if err := os.Remove(filepath.Join(dir, "en-US", "images", "phoneScreenshots", "2.png")); err != nil {
		t.Fatal(err)
	}
	r := scanFixture(t, cleanManifest(), nil, Options{ListingsDir: dir})
	requireFinding(t, r, "screenshot_count", SeverityError)
}

func TestMetadataMissingDirIsReported(t *testing.T) {
	r := scanFixture(t, cleanManifest(), nil, Options{ListingsDir: filepath.Join(t.TempDir(), "nope")})
	requireFinding(t, r, "listings_dir", SeverityError)
}

// --- 6. billing -------------------------------------------------------------

// dexWith builds a fake dex entry containing the given type descriptors.
func dexWith(markers ...string) []byte {
	var b bytes.Buffer
	b.WriteString("dex\n035\x00")
	b.Write(bytes.Repeat([]byte{0}, 64))
	for _, m := range markers {
		b.WriteString(m)
		b.WriteString("Foo;")
	}
	return b.Bytes()
}

func TestBillingThirdPartyProcessorFlagged(t *testing.T) {
	r := scanFixture(t, cleanManifest(), map[string][]byte{
		"base/dex/classes.dex": dexWith("Lcom/stripe/android/"),
	}, Options{})
	f := requireFinding(t, r, "third_party_payments", SeverityWarning)
	if !strings.Contains(f.Message, "Stripe") {
		t.Errorf("message = %q", f.Message)
	}
}

func TestBillingThirdPartyAlongsidePlayBillingIsInfo(t *testing.T) {
	r := scanFixture(t, cleanManifest(), map[string][]byte{
		"base/dex/classes.dex": dexWith("Lcom/stripe/android/", "Lcom/android/billingclient/api/"),
	}, Options{})
	requireFinding(t, r, "third_party_payments", SeverityInfo)
	requireFinding(t, r, "play_billing", SeverityInfo)
}

func TestBillingPermissionWithoutImplementation(t *testing.T) {
	m := withPermissions(cleanManifest(), billingPermission)
	r := scanFixture(t, m, nil, Options{})
	requireFinding(t, r, "billing_permission", SeverityWarning)
}

// --- 7. privacy -------------------------------------------------------------

func TestPrivacyTrackingSDKRequiresAdID(t *testing.T) {
	r := scanFixture(t, cleanManifest(), map[string][]byte{
		"base/dex/classes.dex": dexWith("Lcom/appsflyer/"),
	}, Options{})
	f := requireFinding(t, r, "tracking_sdk", SeverityInfo)
	if !strings.Contains(f.Message, "AppsFlyer") {
		t.Errorf("message = %q", f.Message)
	}
	requireFinding(t, r, "advertising_id", SeverityWarning)
}

func TestPrivacyAdIDDeclaredWithoutSDK(t *testing.T) {
	m := withPermissions(cleanManifest(), adIDPermission)
	r := scanFixture(t, m, nil, Options{})
	requireFinding(t, r, "advertising_id", SeverityInfo)
}

func TestPrivacyCleartextWithTrackingSDK(t *testing.T) {
	m := cleanManifest()
	appNode(&m).attrs = append(appNode(&m).attrs, pbAttr{
		ns: AndroidNS, name: "usesCleartextTraffic", compiled: pbPrimBool(true),
	})
	r := scanFixture(t, m, map[string][]byte{
		"base/dex/classes.dex": dexWith("Lcom/appsflyer/"),
	}, Options{})
	requireFinding(t, r, "insecure_transit", SeverityWarning)
}

// --- 8. policy --------------------------------------------------------------

func TestPolicyTargetSDKBelowFloor(t *testing.T) {
	m := cleanManifest()
	m.children[0] = pbElem{name: "uses-sdk", attrs: []pbAttr{
		{ns: AndroidNS, name: "targetSdkVersion", compiled: pbPrimInt(30)},
	}}
	r := scanFixture(t, m, nil, Options{})
	f := requireFinding(t, r, "target_sdk", SeverityError)
	if !strings.Contains(f.Message, "30") {
		t.Errorf("message = %q", f.Message)
	}

	// The floor is configurable so the check survives Play's annual bump.
	relaxed := scanFixture(t, m, nil, Options{MinTargetSDK: 30})
	requireNoFinding(t, relaxed, "target_sdk")
}

func TestPolicyRestrictedService(t *testing.T) {
	m := cleanManifest()
	appNode(&m).children = append(appNode(&m).children, pbElem{
		name: "service",
		attrs: []pbAttr{
			{ns: AndroidNS, name: "name", value: ".A11yService"},
			{ns: AndroidNS, name: "permission", value: "android.permission.BIND_ACCESSIBILITY_SERVICE"},
		},
	})
	r := scanFixture(t, m, nil, Options{})
	requireFinding(t, r, "restricted_service", SeverityWarning)
}

func TestPolicyAPKFormatNoted(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app.apk")
	buildAAB(t, path, map[string][]byte{
		"AndroidManifest.xml": pbNode(cleanManifest()),
		"resources.arsc":      []byte("res"),
		"classes.dex":         []byte("dex"),
	})
	r, err := Scan(path, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if r.Format != formatAPK {
		t.Fatalf("format = %q, want apk", r.Format)
	}
	requireFinding(t, r, "upload_format", SeverityInfo)
}

func TestPolicyFamiliesNoteWhenAdsPresent(t *testing.T) {
	r := scanFixture(t, cleanManifest(), map[string][]byte{
		"base/dex/classes.dex": dexWith("Lcom/google/android/gms/ads/"),
	}, Options{})
	requireFinding(t, r, "families_policy", SeverityInfo)
	requireFinding(t, r, "ads_sdk", SeverityInfo)
}

// --- 9. size ----------------------------------------------------------------

func TestSizeOverLimitIsError(t *testing.T) {
	r := scanFixture(t, cleanManifest(), nil, Options{MaxBundleBytes: 1})
	requireFinding(t, r, "bundle_size", SeverityError)
}

func TestSizeDexCountWarning(t *testing.T) {
	extra := map[string][]byte{}
	for i := 0; i < 25; i++ {
		extra[filepath.Join("base/dex", "classes"+string(rune('a'+i))+".dex")] = []byte("dex")
	}
	r := scanFixture(t, cleanManifest(), extra, Options{})
	f := requireFinding(t, r, "dex", SeverityWarning)
	if !strings.Contains(f.Message, "dex files") {
		t.Errorf("message = %q", f.Message)
	}
}

func TestHumanBytes(t *testing.T) {
	cases := map[int64]string{
		512:                    "512 B",
		2048:                   "2.0 KB",
		5 * 1024 * 1024:        "5.0 MB",
		3 * 1024 * 1024 * 1024: "3.0 GB",
	}
	for in, want := range cases {
		if got := humanBytes(in); got != want {
			t.Errorf("humanBytes(%d) = %q, want %q", in, got, want)
		}
	}
}
