// Package preflight runs offline compliance and hygiene checks against an
// AAB/APK before upload. It does NOT call the Play API.
//
// Checks are intentionally pragmatic: they operate on ZIP entries and raw
// bytes without decoding the protobuf manifest, so they run in CI on any
// machine. Checks can produce Info, Warning, or Error findings; Errors cause
// a non-zero exit in the CLI.
package preflight

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
)

// Severity of a preflight finding.
type Severity string

const (
	SeverityInfo    Severity = "info"
	SeverityWarning Severity = "warning"
	SeverityError   Severity = "error"
)

// Finding is a single preflight check result.
type Finding struct {
	Check    string   `json:"check"`
	Severity Severity `json:"severity"`
	Message  string   `json:"message"`
	Entry    string   `json:"entry,omitempty"`
	Hint     string   `json:"hint,omitempty"`
}

// Report aggregates all findings.
type Report struct {
	Path      string    `json:"path"`
	Findings  []Finding `json:"findings"`
	Infos     int       `json:"infos"`
	Warnings  int       `json:"warnings"`
	Errors    int       `json:"errors"`
	Checks    []string  `json:"checks_run"`
	TotalSize int64     `json:"total_size_bytes"`
}

// Options tunes the scanner.
type Options struct {
	// MaxBundleBytes is the maximum allowed total compressed size (0 = default 150MB).
	MaxBundleBytes int64
	// MaxDexBytes is per-dex warning threshold (0 = default 64MB).
	MaxDexBytes int64
	// SkipSecretScan disables secret-pattern matching.
	SkipSecretScan bool
	// MaxScanBytesPerEntry caps how many bytes we scan per entry (0 = default 5MB).
	MaxScanBytesPerEntry int64
}

// Scan runs all checks and returns a Report.
func Scan(path string, opts Options) (*Report, error) {
	if opts.MaxBundleBytes == 0 {
		opts.MaxBundleBytes = 150 * 1024 * 1024
	}
	if opts.MaxDexBytes == 0 {
		opts.MaxDexBytes = 64 * 1024 * 1024
	}
	if opts.MaxScanBytesPerEntry == 0 {
		opts.MaxScanBytesPerEntry = 5 * 1024 * 1024
	}

	r, err := zip.OpenReader(path) // #nosec G304 -- user-supplied bundle
	if err != nil {
		return nil, fmt.Errorf("open zip: %w", err)
	}
	defer func() { _ = r.Close() }()

	report := &Report{Path: path, Checks: allCheckNames()}

	// Index entries for cross-checks.
	entryNames := make(map[string]*zip.File, len(r.File))
	var total int64
	for _, f := range r.File {
		entryNames[f.Name] = f
		total += int64(f.CompressedSize64) // #nosec G115
	}
	report.TotalSize = total

	// Aggregate manifest bytes (useful for string-based checks).
	manifestBytes := readEntry(entryNames["base/manifest/AndroidManifest.xml"], opts.MaxScanBytesPerEntry)
	if len(manifestBytes) == 0 {
		manifestBytes = readEntry(entryNames["AndroidManifest.xml"], opts.MaxScanBytesPerEntry)
	}

	addAll := func(findings ...Finding) {
		report.Findings = append(report.Findings, findings...)
	}

	addAll(checkManifestPresent(entryNames)...)
	addAll(checkBundleSize(total, opts.MaxBundleBytes)...)
	addAll(checkResourcesPresent(entryNames)...)
	addAll(checkNativeLibArchitectures(entryNames)...)
	addAll(checkDexCount(entryNames, opts.MaxDexBytes)...)
	addAll(checkDebuggableFlag(manifestBytes)...)
	addAll(checkTestOnly(manifestBytes)...)
	addAll(checkCleartextTraffic(manifestBytes)...)
	addAll(checkDangerousPermissions(manifestBytes)...)
	if !opts.SkipSecretScan {
		addAll(scanForSecrets(r.File, opts.MaxScanBytesPerEntry)...)
	}
	addAll(checkMisplacedFiles(entryNames)...)

	for _, f := range report.Findings {
		switch f.Severity {
		case SeverityInfo:
			report.Infos++
		case SeverityWarning:
			report.Warnings++
		case SeverityError:
			report.Errors++
		}
	}

	return report, nil
}

func allCheckNames() []string {
	return []string{
		"manifest", "bundle_size", "resources", "native_libs",
		"dex", "debuggable", "test_only", "cleartext_traffic",
		"dangerous_permissions", "secrets", "misplaced_files",
	}
}

func readEntry(f *zip.File, maxBytes int64) []byte {
	if f == nil {
		return nil
	}
	rc, err := f.Open()
	if err != nil {
		return nil
	}
	defer rc.Close()
	lim := io.LimitReader(rc, maxBytes)
	data, err := io.ReadAll(lim)
	if err != nil {
		return nil
	}
	return data
}

// --- individual checks ---

func checkManifestPresent(entries map[string]*zip.File) []Finding {
	_, aabOK := entries["base/manifest/AndroidManifest.xml"]
	_, apkOK := entries["AndroidManifest.xml"]
	if !aabOK && !apkOK {
		return []Finding{{
			Check:    "manifest",
			Severity: SeverityError,
			Message:  "AndroidManifest.xml not found",
			Hint:     "the bundle must contain base/manifest/AndroidManifest.xml (AAB) or AndroidManifest.xml (APK)",
		}}
	}
	return nil
}

func checkBundleSize(total, limit int64) []Finding {
	if total > limit {
		return []Finding{{
			Check:    "bundle_size",
			Severity: SeverityWarning,
			Message:  fmt.Sprintf("bundle compressed size %d bytes exceeds limit %d", total, limit),
			Hint:     "consider Play Asset Delivery or dynamic features to reduce base module size",
		}}
	}
	return nil
}

func checkResourcesPresent(entries map[string]*zip.File) []Finding {
	if _, ok := entries["base/resources.pb"]; ok {
		return nil
	}
	if _, ok := entries["resources.arsc"]; ok {
		return nil
	}
	return []Finding{{
		Check:    "resources",
		Severity: SeverityError,
		Message:  "resources table not found (base/resources.pb or resources.arsc)",
	}}
}

func checkNativeLibArchitectures(entries map[string]*zip.File) []Finding {
	abis := map[string]bool{}
	for name := range entries {
		// Native libs live under lib/<abi>/ or base/lib/<abi>/
		prefix := ""
		switch {
		case strings.HasPrefix(name, "lib/"):
			prefix = "lib/"
		case strings.HasPrefix(name, "base/lib/"):
			prefix = "base/lib/"
		default:
			continue
		}
		rest := name[len(prefix):]
		slash := strings.Index(rest, "/")
		if slash <= 0 {
			continue
		}
		abis[rest[:slash]] = true
	}
	if len(abis) == 0 {
		return nil
	}
	if !abis["arm64-v8a"] {
		return []Finding{{
			Check:    "native_libs",
			Severity: SeverityError,
			Message:  "native libs present but arm64-v8a missing",
			Hint:     "Google Play requires 64-bit support; add arm64-v8a",
		}}
	}
	return nil
}

func checkDexCount(entries map[string]*zip.File, maxDex int64) []Finding {
	var dexCount int
	var findings []Finding
	for name, f := range entries {
		if strings.HasSuffix(name, ".dex") {
			dexCount++
			if int64(f.UncompressedSize64) > maxDex { // #nosec G115
				findings = append(findings, Finding{
					Check:    "dex",
					Severity: SeverityWarning,
					Message:  fmt.Sprintf("%s is %d bytes (>%d)", name, f.UncompressedSize64, maxDex),
					Entry:    name,
				})
			}
		}
	}
	if dexCount > 20 {
		findings = append(findings, Finding{
			Check:    "dex",
			Severity: SeverityWarning,
			Message:  fmt.Sprintf("%d dex files; consider R8 optimization", dexCount),
		})
	}
	return findings
}

// The binary AndroidManifest is a protobuf in AABs (not decoded here), but
// attribute names and booleans appear as UTF-8 substrings, which is enough for
// simple presence checks.
func checkDebuggableFlag(manifest []byte) []Finding {
	if len(manifest) == 0 {
		return nil
	}
	if bytes.Contains(manifest, []byte("debuggable")) {
		// Heuristic: true flag will appear close to the attribute. This is a
		// best-effort check — report as warning, not error.
		if bytes.Contains(manifest, []byte("android:debuggable=\"true\"")) ||
			containsNearby(manifest, []byte("debuggable"), []byte{0x01}, 4) {
			return []Finding{{
				Check:    "debuggable",
				Severity: SeverityError,
				Message:  "android:debuggable appears to be true",
				Hint:     "release builds must not be debuggable",
			}}
		}
	}
	return nil
}

func checkTestOnly(manifest []byte) []Finding {
	if len(manifest) == 0 {
		return nil
	}
	if bytes.Contains(manifest, []byte("testOnly")) &&
		(bytes.Contains(manifest, []byte("testOnly=\"true\"")) ||
			containsNearby(manifest, []byte("testOnly"), []byte{0x01}, 4)) {
		return []Finding{{
			Check:    "test_only",
			Severity: SeverityError,
			Message:  "manifest sets android:testOnly=true",
			Hint:     "remove testOnly before uploading to Play Console",
		}}
	}
	return nil
}

func checkCleartextTraffic(manifest []byte) []Finding {
	if len(manifest) == 0 {
		return nil
	}
	if bytes.Contains(manifest, []byte("usesCleartextTraffic")) &&
		(bytes.Contains(manifest, []byte("usesCleartextTraffic=\"true\"")) ||
			containsNearby(manifest, []byte("usesCleartextTraffic"), []byte{0x01}, 4)) {
		return []Finding{{
			Check:    "cleartext_traffic",
			Severity: SeverityWarning,
			Message:  "android:usesCleartextTraffic=true",
			Hint:     "prefer HTTPS; Play policy flags cleartext traffic in new apps",
		}}
	}
	return nil
}

var dangerousPermissions = []string{
	"android.permission.READ_SMS",
	"android.permission.SEND_SMS",
	"android.permission.RECEIVE_SMS",
	"android.permission.READ_CALL_LOG",
	"android.permission.WRITE_CALL_LOG",
	"android.permission.PROCESS_OUTGOING_CALLS",
	"android.permission.MANAGE_EXTERNAL_STORAGE",
	"android.permission.ACCESS_BACKGROUND_LOCATION",
	"android.permission.REQUEST_INSTALL_PACKAGES",
}

func checkDangerousPermissions(manifest []byte) []Finding {
	if len(manifest) == 0 {
		return nil
	}
	var findings []Finding
	for _, p := range dangerousPermissions {
		if bytes.Contains(manifest, []byte(p)) {
			findings = append(findings, Finding{
				Check:    "dangerous_permissions",
				Severity: SeverityInfo,
				Message:  "uses sensitive permission " + p,
				Hint:     "expect an extended Play policy review",
			})
		}
	}
	return findings
}

// Secret regex patterns. These are intentionally conservative to minimize
// false positives; they still catch the most common leaks.
var secretPatterns = map[string]*regexp.Regexp{
	"google_api_key":    regexp.MustCompile(`AIza[0-9A-Za-z_\-]{35}`),
	"aws_access_key":    regexp.MustCompile(`AKIA[0-9A-Z]{16}`),
	"slack_token":       regexp.MustCompile(`xox[baprs]-[0-9A-Za-z\-]+`),
	"stripe_secret":     regexp.MustCompile(`sk_live_[0-9A-Za-z]{24,}`),
	"private_key_block": regexp.MustCompile(`-----BEGIN (?:RSA |EC |OPENSSH )?PRIVATE KEY-----`),
	"firebase_url":      regexp.MustCompile(`https://[a-z0-9\-]+\.firebaseio\.com`),
	"jwt_token":         regexp.MustCompile(`eyJ[A-Za-z0-9_=\-]{10,}\.[A-Za-z0-9_=\-]{10,}\.[A-Za-z0-9_\.=\-]{10,}`),
}

func scanForSecrets(files []*zip.File, maxBytes int64) []Finding {
	var findings []Finding
	for _, f := range files {
		// Only scan likely-text and resource files; skip .dex/.so (large and
		// noisy — secrets there are very rare and expensive to scan well).
		lower := strings.ToLower(f.Name)
		if strings.HasSuffix(lower, ".dex") ||
			strings.HasSuffix(lower, ".so") ||
			strings.HasSuffix(lower, ".kotlin_module") ||
			strings.HasSuffix(lower, ".png") ||
			strings.HasSuffix(lower, ".jpg") ||
			strings.HasSuffix(lower, ".webp") ||
			strings.HasSuffix(lower, ".mp3") ||
			strings.HasSuffix(lower, ".mp4") ||
			strings.HasSuffix(lower, ".otf") ||
			strings.HasSuffix(lower, ".ttf") {
			continue
		}
		if int64(f.UncompressedSize64) > maxBytes { // #nosec G115
			continue
		}
		data := readEntry(f, maxBytes)
		for name, pat := range secretPatterns {
			if loc := pat.FindIndex(data); loc != nil {
				findings = append(findings, Finding{
					Check:    "secrets",
					Severity: SeverityError,
					Message:  fmt.Sprintf("%s matched in bundle", name),
					Entry:    f.Name,
					Hint:     "remove secrets from shipped code; load via secure storage at runtime",
				})
				break
			}
		}
	}
	return findings
}

func checkMisplacedFiles(entries map[string]*zip.File) []Finding {
	var findings []Finding
	suspicious := []string{".DS_Store", "__MACOSX/", ".git/", ".gitignore", "Thumbs.db"}
	for name := range entries {
		for _, sus := range suspicious {
			if strings.Contains(name, sus) {
				findings = append(findings, Finding{
					Check:    "misplaced_files",
					Severity: SeverityWarning,
					Message:  "bundle contains developer-environment artifact",
					Entry:    name,
				})
				break
			}
		}
	}
	return findings
}

// containsNearby returns true if `needle` appears within `window` bytes of a
// matching `context` marker. Used to make string-based manifest checks a
// little less noisy.
func containsNearby(haystack, context, needle []byte, window int) bool {
	if len(context) == 0 || len(needle) == 0 {
		return false
	}
	idx := bytes.Index(haystack, context)
	if idx < 0 {
		return false
	}
	end := idx + len(context) + window + len(needle)
	if end > len(haystack) {
		end = len(haystack)
	}
	return bytes.Contains(haystack[idx:end], needle)
}

// HasErrors returns true if the report contains any error-severity findings.
func (r *Report) HasErrors() bool { return r.Errors > 0 }

// ErrNoErrors is sentinel for CLI to detect clean reports.
var ErrNoErrors = errors.New("no errors")
