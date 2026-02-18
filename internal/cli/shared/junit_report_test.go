package shared

import (
	"encoding/xml"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestJUnitXMLGeneration(t *testing.T) {
	t.Run("produces valid JUnit XML with header", func(t *testing.T) {
		suites := &JUnitTestSuites{
			Suites: []JUnitTestSuite{
				{
					Name:     "validation",
					Tests:    1,
					Failures: 0,
					Errors:   0,
					Time:     0.5,
					Cases: []JUnitTestCase{
						{
							Name:      "check-passed",
							ClassName: "validation",
							Time:      0.5,
						},
					},
				},
			},
		}

		dir := t.TempDir()
		path := filepath.Join(dir, "results.xml")
		if err := WriteJUnitReport(suites, path); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("failed to read file: %v", err)
		}

		content := string(data)

		// Must have XML header
		if !strings.HasPrefix(content, `<?xml version="1.0" encoding="UTF-8"?>`) {
			t.Fatalf("expected XML header, got: %s", content[:80])
		}

		// Must contain testsuites element
		if !strings.Contains(content, "<testsuites>") {
			t.Fatal("expected <testsuites> element")
		}

		// Must be valid XML
		var parsed JUnitTestSuites
		if err := xml.Unmarshal([]byte(strings.TrimPrefix(content, `<?xml version="1.0" encoding="UTF-8"?>`+"\n")), &parsed); err != nil {
			t.Fatalf("invalid XML: %v", err)
		}

		if len(parsed.Suites) != 1 {
			t.Fatalf("expected 1 suite, got %d", len(parsed.Suites))
		}
		if parsed.Suites[0].Name != "validation" {
			t.Fatalf("expected suite name %q, got %q", "validation", parsed.Suites[0].Name)
		}
	})

	t.Run("includes failure details", func(t *testing.T) {
		suites := &JUnitTestSuites{
			Suites: []JUnitTestSuite{
				{
					Name:     "bundle",
					Tests:    2,
					Failures: 1,
					Errors:   0,
					Time:     1.2,
					Cases: []JUnitTestCase{
						{
							Name:      "valid-package",
							ClassName: "bundle",
							Time:      0.6,
						},
						{
							Name:      "valid-signing-key",
							ClassName: "bundle",
							Time:      0.6,
							Failure: &JUnitFailure{
								Message: "signing key mismatch",
								Type:    "validation-error",
								Body:    "Expected SHA-256 fingerprint to match upload key",
							},
						},
					},
				},
			},
		}

		dir := t.TempDir()
		path := filepath.Join(dir, "results.xml")
		if err := WriteJUnitReport(suites, path); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("failed to read file: %v", err)
		}

		content := string(data)

		if !strings.Contains(content, `message="signing key mismatch"`) {
			t.Fatal("expected failure message in XML")
		}
		if !strings.Contains(content, `type="validation-error"`) {
			t.Fatal("expected failure type in XML")
		}
		if !strings.Contains(content, "Expected SHA-256 fingerprint to match upload key") {
			t.Fatal("expected failure body in XML")
		}
	})

	t.Run("uses indented output", func(t *testing.T) {
		suites := &JUnitTestSuites{
			Suites: []JUnitTestSuite{
				{
					Name:  "test",
					Tests: 1,
					Cases: []JUnitTestCase{
						{Name: "case1", ClassName: "test"},
					},
				},
			},
		}

		dir := t.TempDir()
		path := filepath.Join(dir, "results.xml")
		if err := WriteJUnitReport(suites, path); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("failed to read file: %v", err)
		}

		// Indented output should have lines starting with spaces
		lines := strings.Split(string(data), "\n")
		foundIndented := false
		for _, line := range lines {
			if strings.HasPrefix(line, "  ") {
				foundIndented = true
				break
			}
		}
		if !foundIndented {
			t.Fatal("expected indented XML output")
		}
	})

	t.Run("file has 0644 permissions", func(t *testing.T) {
		suites := &JUnitTestSuites{}

		dir := t.TempDir()
		path := filepath.Join(dir, "results.xml")
		if err := WriteJUnitReport(suites, path); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("failed to stat file: %v", err)
		}
		perm := info.Mode().Perm()
		if perm != 0644 {
			t.Fatalf("expected permissions 0644, got %o", perm)
		}
	})
}

func TestWriteJUnitReport_Errors(t *testing.T) {
	t.Run("returns error for invalid path", func(t *testing.T) {
		suites := &JUnitTestSuites{}
		err := WriteJUnitReport(suites, "/nonexistent/dir/results.xml")
		if err == nil {
			t.Fatal("expected error for invalid path")
		}
	})
}

func TestNewJUnitFromValidation(t *testing.T) {
	t.Run("errors become failures", func(t *testing.T) {
		errors := []string{"missing icon", "invalid package name"}
		warnings := []string{"large APK size"}
		result := NewJUnitFromValidation("bundle", errors, warnings, 1.5)

		if len(result.Suites) != 1 {
			t.Fatalf("expected 1 suite, got %d", len(result.Suites))
		}

		suite := result.Suites[0]
		if suite.Name != "bundle" {
			t.Fatalf("expected suite name %q, got %q", "bundle", suite.Name)
		}
		if suite.Tests != 3 {
			t.Fatalf("expected 3 tests, got %d", suite.Tests)
		}
		if suite.Failures != 2 {
			t.Fatalf("expected 2 failures, got %d", suite.Failures)
		}
		if suite.Errors != 0 {
			t.Fatalf("expected 0 errors, got %d", suite.Errors)
		}
		if suite.Time != 1.5 {
			t.Fatalf("expected time 1.5, got %f", suite.Time)
		}

		// Check error test cases
		if suite.Cases[0].Name != "missing icon" {
			t.Fatalf("expected case name %q, got %q", "missing icon", suite.Cases[0].Name)
		}
		if suite.Cases[0].ClassName != "bundle" {
			t.Fatalf("expected classname %q, got %q", "bundle", suite.Cases[0].ClassName)
		}
		if suite.Cases[0].Failure == nil {
			t.Fatal("expected failure for error case")
		}
		if suite.Cases[0].Failure.Message != "missing icon" {
			t.Fatalf("expected failure message %q, got %q", "missing icon", suite.Cases[0].Failure.Message)
		}
		if suite.Cases[0].Failure.Type != "validation-error" {
			t.Fatalf("expected failure type %q, got %q", "validation-error", suite.Cases[0].Failure.Type)
		}

		if suite.Cases[1].Name != "invalid package name" {
			t.Fatalf("expected case name %q, got %q", "invalid package name", suite.Cases[1].Name)
		}
		if suite.Cases[1].Failure == nil {
			t.Fatal("expected failure for second error case")
		}

		// Check warning test case (should pass, no failure)
		if suite.Cases[2].Name != "large APK size" {
			t.Fatalf("expected case name %q, got %q", "large APK size", suite.Cases[2].Name)
		}
		if suite.Cases[2].ClassName != "bundle" {
			t.Fatalf("expected classname %q, got %q", "bundle", suite.Cases[2].ClassName)
		}
		if suite.Cases[2].Failure != nil {
			t.Fatal("expected no failure for warning case")
		}
	})

	t.Run("empty errors and warnings produce passing suite", func(t *testing.T) {
		result := NewJUnitFromValidation("listing", nil, nil, 0.3)

		if len(result.Suites) != 1 {
			t.Fatalf("expected 1 suite, got %d", len(result.Suites))
		}

		suite := result.Suites[0]
		if suite.Name != "listing" {
			t.Fatalf("expected suite name %q, got %q", "listing", suite.Name)
		}
		if suite.Tests != 1 {
			t.Fatalf("expected 1 test, got %d", suite.Tests)
		}
		if suite.Failures != 0 {
			t.Fatalf("expected 0 failures, got %d", suite.Failures)
		}
		if suite.Cases[0].Name != "listing-validation" {
			t.Fatalf("expected case name %q, got %q", "listing-validation", suite.Cases[0].Name)
		}
		if suite.Cases[0].Failure != nil {
			t.Fatal("expected no failure for passing suite")
		}
	})

	t.Run("only warnings produce passing cases", func(t *testing.T) {
		warnings := []string{"warning1", "warning2"}
		result := NewJUnitFromValidation("screenshots", nil, warnings, 2.0)

		suite := result.Suites[0]
		if suite.Tests != 2 {
			t.Fatalf("expected 2 tests, got %d", suite.Tests)
		}
		if suite.Failures != 0 {
			t.Fatalf("expected 0 failures, got %d", suite.Failures)
		}
		for i, c := range suite.Cases {
			if c.Failure != nil {
				t.Fatalf("expected no failure for warning case %d", i)
			}
		}
	})

	t.Run("only errors produce all failures", func(t *testing.T) {
		errors := []string{"error1", "error2", "error3"}
		result := NewJUnitFromValidation("bundle", errors, nil, 0.8)

		suite := result.Suites[0]
		if suite.Tests != 3 {
			t.Fatalf("expected 3 tests, got %d", suite.Tests)
		}
		if suite.Failures != 3 {
			t.Fatalf("expected 3 failures, got %d", suite.Failures)
		}
		for i, c := range suite.Cases {
			if c.Failure == nil {
				t.Fatalf("expected failure for error case %d", i)
			}
		}
	})

	t.Run("XML roundtrip of validation result", func(t *testing.T) {
		errors := []string{"bad version code"}
		warnings := []string{"missing translation"}
		result := NewJUnitFromValidation("release", errors, warnings, 0.5)

		dir := t.TempDir()
		path := filepath.Join(dir, "test.xml")
		if err := WriteJUnitReport(result, path); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("failed to read file: %v", err)
		}

		content := string(data)
		if !strings.Contains(content, `name="release"`) {
			t.Fatal("expected suite name in XML")
		}
		if !strings.Contains(content, `tests="2"`) {
			t.Fatal("expected tests count in XML")
		}
		if !strings.Contains(content, `failures="1"`) {
			t.Fatal("expected failures count in XML")
		}
		if !strings.Contains(content, `message="bad version code"`) {
			t.Fatal("expected failure message in XML")
		}
		if !strings.Contains(content, `name="missing translation"`) {
			t.Fatal("expected warning case in XML")
		}
	})
}
