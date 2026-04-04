package validation

import "testing"

func TestReadinessReportAddCheck_UpdatesSummary(t *testing.T) {
	report := &ReadinessReport{PackageName: "com.example.app"}

	report.AddCheck(ReadinessCheck{ID: "b1", State: ReadinessBlocking})
	report.AddCheck(ReadinessCheck{ID: "w1", State: ReadinessWarning})
	report.AddCheck(ReadinessCheck{ID: "i1", State: ReadinessInfo})
	report.AddCheck(ReadinessCheck{ID: "m1", State: ReadinessManual})

	if report.Summary.Blocking != 1 {
		t.Fatalf("blocking = %d, want 1", report.Summary.Blocking)
	}
	if report.Summary.Warnings != 1 {
		t.Fatalf("warnings = %d, want 1", report.Summary.Warnings)
	}
	if report.Summary.Info != 1 {
		t.Fatalf("info = %d, want 1", report.Summary.Info)
	}
	if report.Summary.Manual != 1 {
		t.Fatalf("manual = %d, want 1", report.Summary.Manual)
	}
	if report.Summary.Ready {
		t.Fatal("report should not be ready when blocking issues exist")
	}
}

func TestReadinessReportSummaryLine(t *testing.T) {
	report := &ReadinessReport{}
	report.AddCheck(ReadinessCheck{State: ReadinessWarning})

	got := report.SummaryLine()
	want := "Readiness complete: 0 blocking, 1 warning, 0 info, 0 manual follow-up"
	if got != want {
		t.Fatalf("SummaryLine() = %q, want %q", got, want)
	}
}
