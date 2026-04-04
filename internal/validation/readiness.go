package validation

import "fmt"

// ReadinessState describes how a readiness check should be interpreted.
type ReadinessState string

const (
	ReadinessBlocking ReadinessState = "blocking"
	ReadinessWarning  ReadinessState = "warning"
	ReadinessInfo     ReadinessState = "info"
	ReadinessManual   ReadinessState = "manual"
)

// ReadinessCheck is a single readiness finding grouped by section.
type ReadinessCheck struct {
	ID          string                 `json:"id"`
	Section     string                 `json:"section"`
	State       ReadinessState         `json:"state"`
	Locale      string                 `json:"locale,omitempty"`
	Field       string                 `json:"field,omitempty"`
	Message     string                 `json:"message"`
	Remediation string                 `json:"remediation,omitempty"`
	Details     map[string]interface{} `json:"details,omitempty"`
}

// ReadinessSummary aggregates the overall readiness state.
type ReadinessSummary struct {
	Blocking int  `json:"blocking"`
	Warnings int  `json:"warnings"`
	Info     int  `json:"info"`
	Manual   int  `json:"manual"`
	Ready    bool `json:"ready"`
}

// ReadinessReport is the canonical release-readiness payload for Play.
type ReadinessReport struct {
	PackageName string           `json:"packageName"`
	Track       string           `json:"track,omitempty"`
	Artifact    string           `json:"artifact,omitempty"`
	Checks      []ReadinessCheck `json:"checks"`
	Summary     ReadinessSummary `json:"summary"`
}

// AddCheck appends a readiness check and updates summary counters.
func (r *ReadinessReport) AddCheck(check ReadinessCheck) {
	r.Checks = append(r.Checks, check)

	switch check.State {
	case ReadinessBlocking:
		r.Summary.Blocking++
	case ReadinessWarning:
		r.Summary.Warnings++
	case ReadinessInfo:
		r.Summary.Info++
	case ReadinessManual:
		r.Summary.Manual++
	}

	r.Summary.Ready = r.Summary.Blocking == 0
}

// SummaryLine returns a short human-readable summary.
func (r *ReadinessReport) SummaryLine() string {
	return fmt.Sprintf(
		"Readiness complete: %d blocking, %d warning, %d info, %d manual follow-up",
		r.Summary.Blocking,
		r.Summary.Warnings,
		r.Summary.Info,
		r.Summary.Manual,
	)
}
