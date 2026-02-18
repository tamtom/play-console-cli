package shared

import (
	"encoding/xml"
	"fmt"
	"os"
)

// JUnitTestSuites is the top-level XML element.
type JUnitTestSuites struct {
	XMLName xml.Name         `xml:"testsuites"`
	Suites  []JUnitTestSuite `xml:"testsuite"`
}

// JUnitTestSuite represents a single test suite.
type JUnitTestSuite struct {
	XMLName  xml.Name        `xml:"testsuite"`
	Name     string          `xml:"name,attr"`
	Tests    int             `xml:"tests,attr"`
	Failures int             `xml:"failures,attr"`
	Errors   int             `xml:"errors,attr"`
	Time     float64         `xml:"time,attr"`
	Cases    []JUnitTestCase `xml:"testcase"`
}

// JUnitTestCase represents a single test case.
type JUnitTestCase struct {
	XMLName   xml.Name      `xml:"testcase"`
	Name      string        `xml:"name,attr"`
	ClassName string        `xml:"classname,attr"`
	Time      float64       `xml:"time,attr"`
	Failure   *JUnitFailure `xml:"failure,omitempty"`
}

// JUnitFailure represents a test failure.
type JUnitFailure struct {
	Message string `xml:"message,attr"`
	Type    string `xml:"type,attr"`
	Body    string `xml:",chardata"`
}

// WriteJUnitReport writes JUnit XML to the specified file.
func WriteJUnitReport(suites *JUnitTestSuites, filePath string) error {
	data, err := xml.MarshalIndent(suites, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal JUnit XML: %w", err)
	}

	content := xml.Header + string(data) + "\n"

	if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write JUnit report: %w", err)
	}

	return nil
}

// NewJUnitFromValidation converts validation results into JUnit format.
// validationName is the suite name (e.g., "bundle", "listing", "screenshots").
// errors are test failures, warnings are passed-with-warnings.
func NewJUnitFromValidation(validationName string, errors []string, warnings []string, durationSecs float64) *JUnitTestSuites {
	var cases []JUnitTestCase
	failures := 0

	for _, e := range errors {
		cases = append(cases, JUnitTestCase{
			Name:      e,
			ClassName: validationName,
			Failure: &JUnitFailure{
				Message: e,
				Type:    "validation-error",
				Body:    e,
			},
		})
		failures++
	}

	for _, w := range warnings {
		cases = append(cases, JUnitTestCase{
			Name:      w,
			ClassName: validationName,
		})
	}

	// If no errors and no warnings, create a single passing test case.
	if len(cases) == 0 {
		cases = append(cases, JUnitTestCase{
			Name:      validationName + "-validation",
			ClassName: validationName,
		})
	}

	suite := JUnitTestSuite{
		Name:     validationName,
		Tests:    len(cases),
		Failures: failures,
		Errors:   0,
		Time:     durationSecs,
		Cases:    cases,
	}

	return &JUnitTestSuites{
		Suites: []JUnitTestSuite{suite},
	}
}
