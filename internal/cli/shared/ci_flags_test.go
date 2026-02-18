package shared

import (
	"flag"
	"testing"
)

func TestRegisterCIFlags(t *testing.T) {
	t.Run("registers both flags with defaults", func(t *testing.T) {
		fs := flag.NewFlagSet("test", flag.ContinueOnError)
		var cf CIFlags
		RegisterCIFlags(fs, &cf)

		reportFlag := fs.Lookup("report")
		if reportFlag == nil {
			t.Fatal("expected --report flag to be registered")
		}
		if reportFlag.DefValue != "" {
			t.Fatalf("expected default value empty, got %q", reportFlag.DefValue)
		}

		reportFileFlag := fs.Lookup("report-file")
		if reportFileFlag == nil {
			t.Fatal("expected --report-file flag to be registered")
		}
		if reportFileFlag.DefValue != "results.xml" {
			t.Fatalf("expected default value %q, got %q", "results.xml", reportFileFlag.DefValue)
		}
	})

	t.Run("parses junit report flag", func(t *testing.T) {
		fs := flag.NewFlagSet("test", flag.ContinueOnError)
		var cf CIFlags
		RegisterCIFlags(fs, &cf)

		err := fs.Parse([]string{"--report", "junit"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cf.Report != "junit" {
			t.Fatalf("expected report %q, got %q", "junit", cf.Report)
		}
		if cf.ReportFile != "results.xml" {
			t.Fatalf("expected report-file %q, got %q", "results.xml", cf.ReportFile)
		}
	})

	t.Run("parses custom report file", func(t *testing.T) {
		fs := flag.NewFlagSet("test", flag.ContinueOnError)
		var cf CIFlags
		RegisterCIFlags(fs, &cf)

		err := fs.Parse([]string{"--report", "junit", "--report-file", "custom.xml"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cf.Report != "junit" {
			t.Fatalf("expected report %q, got %q", "junit", cf.Report)
		}
		if cf.ReportFile != "custom.xml" {
			t.Fatalf("expected report-file %q, got %q", "custom.xml", cf.ReportFile)
		}
	})

	t.Run("empty report is valid", func(t *testing.T) {
		fs := flag.NewFlagSet("test", flag.ContinueOnError)
		var cf CIFlags
		RegisterCIFlags(fs, &cf)

		err := fs.Parse([]string{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cf.Report != "" {
			t.Fatalf("expected empty report, got %q", cf.Report)
		}
	})
}

func TestValidateCIFlags(t *testing.T) {
	t.Run("valid junit report", func(t *testing.T) {
		cf := &CIFlags{Report: "junit", ReportFile: "results.xml"}
		if err := ValidateCIFlags(cf); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("empty report is valid", func(t *testing.T) {
		cf := &CIFlags{Report: "", ReportFile: "results.xml"}
		if err := ValidateCIFlags(cf); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("invalid report format", func(t *testing.T) {
		cf := &CIFlags{Report: "html", ReportFile: "results.xml"}
		err := ValidateCIFlags(cf)
		if err == nil {
			t.Fatal("expected error for invalid report format")
		}
		expected := `unsupported report format "html": valid values are: junit`
		if err.Error() != expected {
			t.Fatalf("expected error %q, got %q", expected, err.Error())
		}
	})

	t.Run("empty report file when report set", func(t *testing.T) {
		cf := &CIFlags{Report: "junit", ReportFile: ""}
		err := ValidateCIFlags(cf)
		if err == nil {
			t.Fatal("expected error for empty report file")
		}
		expected := "--report-file is required when --report is set"
		if err.Error() != expected {
			t.Fatalf("expected error %q, got %q", expected, err.Error())
		}
	})

	t.Run("whitespace-only report file when report set", func(t *testing.T) {
		cf := &CIFlags{Report: "junit", ReportFile: "   "}
		err := ValidateCIFlags(cf)
		if err == nil {
			t.Fatal("expected error for whitespace report file")
		}
		expected := "--report-file is required when --report is set"
		if err.Error() != expected {
			t.Fatalf("expected error %q, got %q", expected, err.Error())
		}
	})

	t.Run("case insensitive report format", func(t *testing.T) {
		cf := &CIFlags{Report: "JUnit", ReportFile: "results.xml"}
		if err := ValidateCIFlags(cf); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}
