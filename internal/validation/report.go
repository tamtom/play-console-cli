package validation

import "fmt"

// Report aggregates validation check results.
type Report struct {
	Results  []CheckResult `json:"results"`
	Errors   int           `json:"errors"`
	Warnings int           `json:"warnings"`
}

// Add appends a check result to the report and updates counters.
func (r *Report) Add(result CheckResult) {
	r.Results = append(r.Results, result)
	switch result.Severity {
	case SeverityError:
		r.Errors++
	case SeverityWarning:
		r.Warnings++
	}
}

// HasErrors returns true if the report contains any error-level results.
func (r *Report) HasErrors() bool {
	return r.Errors > 0
}

// Summary returns a human-readable summary of the report.
func (r *Report) Summary() string {
	return fmt.Sprintf("Validation complete: %d error(s), %d warning(s)", r.Errors, r.Warnings)
}
