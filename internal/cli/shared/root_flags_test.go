package shared

import (
	"flag"
	"os"
	"testing"
)

func TestBindRootFlags_RegistersAllFlags(t *testing.T) {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	rf := BindRootFlags(fs)

	if rf.Profile == nil {
		t.Error("expected Profile to be non-nil")
	}
	if rf.Debug == nil {
		t.Error("expected Debug to be non-nil")
	}
	if rf.DryRun == nil {
		t.Error("expected DryRun to be non-nil")
	}
	if rf.Report == nil {
		t.Error("expected Report to be non-nil")
	}
	if rf.ReportFile == nil {
		t.Error("expected ReportFile to be non-nil")
	}

	// Verify flags are registered on the FlagSet
	for _, name := range []string{"profile", "debug", "dry-run", "report", "report-file"} {
		if fs.Lookup(name) == nil {
			t.Errorf("expected flag %q to be registered", name)
		}
	}
}

func TestApply_SetsProfile(t *testing.T) {
	// Save and restore env
	orig := os.Getenv("GPLAY_PROFILE")
	defer os.Setenv("GPLAY_PROFILE", orig)

	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	rf := BindRootFlags(fs)
	if err := fs.Parse([]string{"--profile", "staging"}); err != nil {
		t.Fatal(err)
	}

	rf.Apply()

	if got := os.Getenv("GPLAY_PROFILE"); got != "staging" {
		t.Errorf("GPLAY_PROFILE = %q, want %q", got, "staging")
	}
}

func TestApply_SetsDebug(t *testing.T) {
	orig := os.Getenv("GPLAY_DEBUG")
	defer os.Setenv("GPLAY_DEBUG", orig)

	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	rf := BindRootFlags(fs)
	if err := fs.Parse([]string{"--debug"}); err != nil {
		t.Fatal(err)
	}

	rf.Apply()

	if got := os.Getenv("GPLAY_DEBUG"); got != "1" {
		t.Errorf("GPLAY_DEBUG = %q, want %q", got, "1")
	}
}

func TestApply_EmptyProfile_DoesNotSetEnv(t *testing.T) {
	orig := os.Getenv("GPLAY_PROFILE")
	os.Setenv("GPLAY_PROFILE", "original")
	defer os.Setenv("GPLAY_PROFILE", orig)

	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	rf := BindRootFlags(fs)
	if err := fs.Parse([]string{}); err != nil {
		t.Fatal(err)
	}

	rf.Apply()

	if got := os.Getenv("GPLAY_PROFILE"); got != "original" {
		t.Errorf("GPLAY_PROFILE = %q, want %q (should be unchanged)", got, "original")
	}
}

func TestValidateReportFlags_OnlyReport_Error(t *testing.T) {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	rf := BindRootFlags(fs)
	if err := fs.Parse([]string{"--report", "junit"}); err != nil {
		t.Fatal(err)
	}

	err := rf.ValidateReportFlags()
	if err == nil {
		t.Error("expected error when --report is set without --report-file")
	}
}

func TestValidateReportFlags_OnlyReportFile_Error(t *testing.T) {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	rf := BindRootFlags(fs)
	if err := fs.Parse([]string{"--report-file", "results.xml"}); err != nil {
		t.Fatal(err)
	}

	err := rf.ValidateReportFlags()
	if err == nil {
		t.Error("expected error when --report-file is set without --report")
	}
}

func TestValidateReportFlags_BothSet_Success(t *testing.T) {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	rf := BindRootFlags(fs)
	if err := fs.Parse([]string{"--report", "junit", "--report-file", "results.xml"}); err != nil {
		t.Fatal(err)
	}

	err := rf.ValidateReportFlags()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateReportFlags_InvalidFormat_Error(t *testing.T) {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	rf := BindRootFlags(fs)
	if err := fs.Parse([]string{"--report", "xml", "--report-file", "results.xml"}); err != nil {
		t.Fatal(err)
	}

	err := rf.ValidateReportFlags()
	if err == nil {
		t.Error("expected error for unsupported report format")
	}
}

func TestValidateReportFlags_NeitherSet_Success(t *testing.T) {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	rf := BindRootFlags(fs)
	if err := fs.Parse([]string{}); err != nil {
		t.Fatal(err)
	}

	err := rf.ValidateReportFlags()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
