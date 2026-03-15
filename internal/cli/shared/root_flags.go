package shared

import (
	"flag"
	"os"
	"strings"
)

// RootFlags holds the parsed root-level flags.
type RootFlags struct {
	Profile    *string
	Debug      *bool
	DryRun     *bool
	Report     *string
	ReportFile *string
}

// BindRootFlags registers root-level flags on the given FlagSet.
func BindRootFlags(fs *flag.FlagSet) *RootFlags {
	return &RootFlags{
		Profile:    fs.String("profile", "", "Config profile to use (overrides GPLAY_PROFILE)"),
		Debug:      fs.Bool("debug", false, "Enable debug logging (overrides GPLAY_DEBUG)"),
		DryRun:     fs.Bool("dry-run", false, "Preview write operations without executing them"),
		Report:     fs.String("report", "", "CI report format (junit)"),
		ReportFile: fs.String("report-file", "", "CI report output file path"),
	}
}

// Apply sets environment variables based on parsed root flags.
// Call this after root.Parse() and before root.Run().
func (rf *RootFlags) Apply() {
	if rf.Profile != nil && strings.TrimSpace(*rf.Profile) != "" {
		os.Setenv("GPLAY_PROFILE", strings.TrimSpace(*rf.Profile))
	}
	if rf.Debug != nil && *rf.Debug {
		os.Setenv("GPLAY_DEBUG", "1")
	}
}

// ValidateReportFlags checks that --report and --report-file are used together.
func (rf *RootFlags) ValidateReportFlags() error {
	hasReport := rf.Report != nil && strings.TrimSpace(*rf.Report) != ""
	hasFile := rf.ReportFile != nil && strings.TrimSpace(*rf.ReportFile) != ""

	if hasReport && !hasFile {
		return UsageError("--report-file is required when --report is set")
	}
	if hasFile && !hasReport {
		return UsageError("--report is required when --report-file is set")
	}
	if hasReport {
		format := strings.ToLower(strings.TrimSpace(*rf.Report))
		if format != "junit" {
			return UsageErrorf("unsupported report format %q (supported: junit)", format)
		}
	}
	return nil
}
