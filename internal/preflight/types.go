package preflight

import "strings"

// Severity of a preflight finding.
type Severity string

const (
	SeverityInfo    Severity = "info"
	SeverityWarning Severity = "warning"
	SeverityError   Severity = "error"
)

// Finding is a single preflight check result.
type Finding struct {
	// Scanner is the ID of the scanner that produced this finding.
	Scanner string `json:"scanner"`
	// Check is the stable identifier for the specific rule.
	Check    string   `json:"check"`
	Severity Severity `json:"severity"`
	Message  string   `json:"message"`
	// Entry is the bundle entry or file the finding refers to.
	Entry string `json:"entry,omitempty"`
	// Hint explains how to fix the finding.
	Hint string `json:"hint,omitempty"`
	// Ref links to the relevant Play policy or developer documentation.
	Ref string `json:"ref,omitempty"`
}

// ScannerRun records what a single scanner did.
type ScannerRun struct {
	ID       string `json:"id"`
	Findings int    `json:"findings"`
	Skipped  bool   `json:"skipped,omitempty"`
	Reason   string `json:"reason,omitempty"`
}

// Report aggregates all findings from a scan.
type Report struct {
	Path   string `json:"path"`
	Format string `json:"format,omitempty"`

	Package     string `json:"package,omitempty"`
	VersionCode int64  `json:"version_code,omitempty"`
	VersionName string `json:"version_name,omitempty"`
	MinSdk      int    `json:"min_sdk,omitempty"`
	TargetSdk   int    `json:"target_sdk,omitempty"`

	Findings []Finding `json:"findings"`
	Infos    int       `json:"infos"`
	Warnings int       `json:"warnings"`
	Errors   int       `json:"errors"`

	Scanners  []ScannerRun `json:"scanners"`
	Checks    []string     `json:"checks_run"`
	TotalSize int64        `json:"total_size_bytes"`
}

// Options tunes the scan.
type Options struct {
	// MaxBundleBytes is the maximum allowed total compressed size
	// (0 = default 200MB, matching the Play download-size limit).
	MaxBundleBytes int64
	// MaxDexBytes is the per-dex warning threshold (0 = default 64MB).
	MaxDexBytes int64
	// SkipSecretScan disables secret-pattern matching.
	SkipSecretScan bool
	// MaxScanBytesPerEntry caps how many bytes are read per entry
	// (0 = default 5MB).
	MaxScanBytesPerEntry int64
	// MaxDexScanBytes caps how many bytes of each dex are searched for SDK
	// signatures (0 = default 96MB).
	MaxDexScanBytes int64
	// ListingsDir enables the metadata scanner against a Fastlane-style
	// listings directory. When empty, the metadata scanner is skipped.
	ListingsDir string
	// MinTargetSDK is the minimum target SDK Play accepts
	// (0 = default currentMinTargetSDK).
	MinTargetSDK int
	// Only restricts the run to these scanner IDs. Empty means all.
	Only []string
	// Skip excludes these scanner IDs.
	Skip []string
}

// HasErrors returns true if the report contains any error-severity findings.
func (r *Report) HasErrors() bool { return r.Errors > 0 }

// severityRank orders severities for threshold comparisons.
func severityRank(s Severity) int {
	switch s {
	case SeverityError:
		return 3
	case SeverityWarning:
		return 2
	case SeverityInfo:
		return 1
	}
	return 0
}

// AtOrAbove reports whether the report contains a finding at least as severe
// as min.
func (r *Report) AtOrAbove(min Severity) bool {
	want := severityRank(min)
	for _, f := range r.Findings {
		if severityRank(f.Severity) >= want {
			return true
		}
	}
	return false
}

// FindingsFor returns the findings produced by a given scanner ID.
func (r *Report) FindingsFor(scannerID string) []Finding {
	var out []Finding
	for _, f := range r.Findings {
		if f.Scanner == scannerID {
			out = append(out, f)
		}
	}
	return out
}

// normalizeIDs lowercases and trims a list of scanner IDs, also splitting
// comma-separated entries so `--only manifest,secrets` works.
func normalizeIDs(in []string) []string {
	var out []string
	for _, s := range in {
		for _, part := range strings.Split(s, ",") {
			if p := strings.ToLower(strings.TrimSpace(part)); p != "" {
				out = append(out, p)
			}
		}
	}
	return out
}

func containsID(list []string, id string) bool {
	for _, s := range list {
		if s == id {
			return true
		}
	}
	return false
}
