// Package preflight runs offline compliance and hygiene checks against an
// AAB/APK before upload. It does NOT call the Play API.
//
// The engine decodes AndroidManifest.xml for real — binary AXML for APKs and
// the aapt2 protobuf encoding for App Bundles — then runs nine independent
// scanners over the decoded manifest and the archive contents:
//
//	manifest     structural and security attributes of the manifest
//	permissions  restricted and sensitive permission audit
//	native_libs  64-bit coverage, ABI hygiene, 16 KB page alignment
//	metadata     store listing text and screenshot validation
//	secrets      credentials and developer artifacts shipped in the build
//	billing      Play Billing vs third-party payment SDKs
//	privacy      tracking/analytics SDKs and Data safety implications
//	policy       Play policy deadlines and restricted behaviors
//	size         download size, dex budget, and payload bloat
//
// Findings carry Info, Warning, or Error severity; the CLI decides which
// level fails the build.
package preflight

import (
	"archive/zip"
	"errors"
	"fmt"
)

// registeredScanner binds a scanner ID to its implementation.
type registeredScanner struct {
	ID  string
	Run func(*scanContext) []Finding
}

// scanners is the ordered registry. Order determines report ordering.
var scanners = []registeredScanner{
	{ID: "manifest", Run: scanManifest},
	{ID: "permissions", Run: scanPermissions},
	{ID: "native_libs", Run: scanNativeLibs},
	{ID: "metadata", Run: scanMetadata},
	{ID: "secrets", Run: scanSecrets},
	{ID: "billing", Run: scanBilling},
	{ID: "privacy", Run: scanPrivacy},
	{ID: "policy", Run: scanPolicy},
	{ID: "size", Run: scanSize},
}

// ScannerIDs returns the IDs of every registered scanner, in run order.
func ScannerIDs() []string {
	out := make([]string, 0, len(scanners))
	for _, s := range scanners {
		out = append(out, s.ID)
	}
	return out
}

// ValidateScannerIDs returns an error naming the first unknown ID.
func ValidateScannerIDs(ids []string) error {
	known := ScannerIDs()
	for _, id := range normalizeIDs(ids) {
		if !containsID(known, id) {
			return fmt.Errorf("unknown scanner %q (known: %v)", id, known)
		}
	}
	return nil
}

// Scan runs the selected scanners against an AAB or APK and returns a Report.
func Scan(path string, opts Options) (*Report, error) {
	if err := ValidateScannerIDs(opts.Only); err != nil {
		return nil, err
	}
	if err := ValidateScannerIDs(opts.Skip); err != nil {
		return nil, err
	}

	r, err := zip.OpenReader(path) // #nosec G304 -- user-supplied bundle
	if err != nil {
		return nil, fmt.Errorf("open bundle: %w", err)
	}
	defer func() { _ = r.Close() }()

	ctx := newScanContext(path, r, opts)

	report := &Report{
		Path:      path,
		Format:    ctx.format,
		TotalSize: ctx.totalCompressed,
	}
	if m := ctx.manifest; m != nil {
		report.Package = m.Package
		report.VersionCode = m.VersionCode
		report.VersionName = m.VersionName
		report.MinSdk = m.MinSdk
		report.TargetSdk = m.TargetSdk
	}

	only := normalizeIDs(opts.Only)
	skip := normalizeIDs(opts.Skip)

	for _, s := range scanners {
		run := ScannerRun{ID: s.ID}

		switch {
		case len(only) > 0 && !containsID(only, s.ID):
			run.Skipped, run.Reason = true, "not selected by --only"
		case containsID(skip, s.ID):
			run.Skipped, run.Reason = true, "excluded by --skip"
		case s.ID == "secrets" && opts.SkipSecretScan:
			run.Skipped, run.Reason = true, "--skip-secrets"
		case s.ID == "metadata" && opts.ListingsDir == "":
			run.Skipped, run.Reason = true, "no --listings-dir provided"
		}

		if !run.Skipped {
			findings := s.Run(ctx)
			for i := range findings {
				findings[i].Scanner = s.ID
			}
			report.Findings = append(report.Findings, findings...)
			run.Findings = len(findings)
		}

		report.Scanners = append(report.Scanners, run)
		if !run.Skipped {
			report.Checks = append(report.Checks, s.ID)
		}
	}

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

// ErrNoErrors is a sentinel for callers detecting clean reports.
var ErrNoErrors = errors.New("no errors")
