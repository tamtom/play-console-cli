package shared

import (
	"flag"
	"fmt"
	"strings"
)

// validReportFormats lists the supported CI report formats.
var validReportFormats = map[string]bool{
	"junit": true,
}

// CIFlags holds CI report configuration.
type CIFlags struct {
	Report     string // "junit" or empty
	ReportFile string // output file path (default: "results.xml")
}

// RegisterCIFlags adds --report and --report-file flags to a flag set.
func RegisterCIFlags(fs *flag.FlagSet, cf *CIFlags) {
	fs.StringVar(&cf.Report, "report", "", "CI report format (junit)")
	fs.StringVar(&cf.ReportFile, "report-file", "results.xml", "CI report output file path")
}

// ValidateCIFlags checks that the flag values are valid.
func ValidateCIFlags(cf *CIFlags) error {
	report := strings.ToLower(strings.TrimSpace(cf.Report))

	if report == "" {
		return nil
	}

	if !validReportFormats[report] {
		formats := make([]string, 0, len(validReportFormats))
		for f := range validReportFormats {
			formats = append(formats, f)
		}
		return fmt.Errorf("unsupported report format %q: valid values are: %s", cf.Report, strings.Join(formats, ", "))
	}

	if strings.TrimSpace(cf.ReportFile) == "" {
		return fmt.Errorf("--report-file is required when --report is set")
	}

	return nil
}
