package validation

// Severity represents the severity level of a validation check result.
type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
	SeverityInfo    Severity = "info"
)

// CheckResult represents the outcome of a single validation check.
type CheckResult struct {
	ID          string   `json:"id"`
	Severity    Severity `json:"severity"`
	Locale      string   `json:"locale,omitempty"`
	Field       string   `json:"field,omitempty"`
	Message     string   `json:"message"`
	Remediation string   `json:"remediation,omitempty"`
}
