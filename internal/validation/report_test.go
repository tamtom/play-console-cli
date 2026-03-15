package validation

import (
	"strings"
	"testing"
)

func TestReport_Empty(t *testing.T) {
	r := &Report{}
	if r.HasErrors() {
		t.Error("empty report should not have errors")
	}
	if r.Errors != 0 {
		t.Errorf("expected 0 errors, got %d", r.Errors)
	}
	if r.Warnings != 0 {
		t.Errorf("expected 0 warnings, got %d", r.Warnings)
	}
	if len(r.Results) != 0 {
		t.Errorf("expected 0 results, got %d", len(r.Results))
	}
}

func TestReport_AddError(t *testing.T) {
	r := &Report{}
	r.Add(CheckResult{
		ID:       "test-error",
		Severity: SeverityError,
		Message:  "something is wrong",
	})
	if !r.HasErrors() {
		t.Error("report with error should have errors")
	}
	if r.Errors != 1 {
		t.Errorf("expected 1 error, got %d", r.Errors)
	}
	if r.Warnings != 0 {
		t.Errorf("expected 0 warnings, got %d", r.Warnings)
	}
	if len(r.Results) != 1 {
		t.Errorf("expected 1 result, got %d", len(r.Results))
	}
}

func TestReport_AddWarning(t *testing.T) {
	r := &Report{}
	r.Add(CheckResult{
		ID:       "test-warning",
		Severity: SeverityWarning,
		Message:  "something might be wrong",
	})
	if r.HasErrors() {
		t.Error("report with only warnings should not have errors")
	}
	if r.Errors != 0 {
		t.Errorf("expected 0 errors, got %d", r.Errors)
	}
	if r.Warnings != 1 {
		t.Errorf("expected 1 warning, got %d", r.Warnings)
	}
}

func TestReport_AddInfo(t *testing.T) {
	r := &Report{}
	r.Add(CheckResult{
		ID:       "test-info",
		Severity: SeverityInfo,
		Message:  "informational message",
	})
	if r.HasErrors() {
		t.Error("report with only info should not have errors")
	}
	if r.Errors != 0 {
		t.Errorf("expected 0 errors, got %d", r.Errors)
	}
	if r.Warnings != 0 {
		t.Errorf("expected 0 warnings, got %d", r.Warnings)
	}
	if len(r.Results) != 1 {
		t.Errorf("expected 1 result, got %d", len(r.Results))
	}
}

func TestReport_MixedSeverities(t *testing.T) {
	r := &Report{}
	r.Add(CheckResult{ID: "e1", Severity: SeverityError, Message: "error 1"})
	r.Add(CheckResult{ID: "w1", Severity: SeverityWarning, Message: "warn 1"})
	r.Add(CheckResult{ID: "e2", Severity: SeverityError, Message: "error 2"})
	r.Add(CheckResult{ID: "i1", Severity: SeverityInfo, Message: "info 1"})
	r.Add(CheckResult{ID: "w2", Severity: SeverityWarning, Message: "warn 2"})

	if !r.HasErrors() {
		t.Error("report with errors should have errors")
	}
	if r.Errors != 2 {
		t.Errorf("expected 2 errors, got %d", r.Errors)
	}
	if r.Warnings != 2 {
		t.Errorf("expected 2 warnings, got %d", r.Warnings)
	}
	if len(r.Results) != 5 {
		t.Errorf("expected 5 results, got %d", len(r.Results))
	}
}

func TestReport_Summary_NoIssues(t *testing.T) {
	r := &Report{}
	s := r.Summary()
	if !strings.Contains(s, "0 error") {
		t.Errorf("summary should mention 0 errors, got: %s", s)
	}
	if !strings.Contains(s, "0 warning") {
		t.Errorf("summary should mention 0 warnings, got: %s", s)
	}
}

func TestReport_Summary_WithIssues(t *testing.T) {
	r := &Report{}
	r.Add(CheckResult{ID: "e1", Severity: SeverityError, Message: "err"})
	r.Add(CheckResult{ID: "w1", Severity: SeverityWarning, Message: "warn"})
	s := r.Summary()
	if !strings.Contains(s, "1 error") {
		t.Errorf("summary should mention 1 error, got: %s", s)
	}
	if !strings.Contains(s, "1 warning") {
		t.Errorf("summary should mention 1 warning, got: %s", s)
	}
}
