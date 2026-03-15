package output

import (
	"bytes"
	"strings"
	"testing"
)

func TestRenderMarkdownTable_Basic(t *testing.T) {
	var buf bytes.Buffer
	headers := []string{"Name", "Value"}
	rows := [][]string{
		{"alpha", "1"},
		{"beta", "2"},
	}
	err := RenderMarkdownTable(&buf, headers, rows)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := buf.String()
	lines := strings.Split(strings.TrimSpace(out), "\n")

	if len(lines) != 4 {
		t.Fatalf("expected 4 lines (header, separator, 2 rows), got %d: %q", len(lines), out)
	}

	// Header row
	if !strings.Contains(lines[0], "Name") || !strings.Contains(lines[0], "Value") {
		t.Errorf("header row should contain column names, got: %q", lines[0])
	}

	// Separator row
	if !strings.Contains(lines[1], "---") {
		t.Errorf("separator row should contain ---, got: %q", lines[1])
	}

	// Data rows
	if !strings.Contains(lines[2], "alpha") || !strings.Contains(lines[2], "1") {
		t.Errorf("first data row should contain alpha and 1, got: %q", lines[2])
	}
	if !strings.Contains(lines[3], "beta") || !strings.Contains(lines[3], "2") {
		t.Errorf("second data row should contain beta and 2, got: %q", lines[3])
	}
}

func TestRenderMarkdownTable_EscapesPipes(t *testing.T) {
	var buf bytes.Buffer
	headers := []string{"Expression"}
	rows := [][]string{
		{"a | b"},
		{"x|y"},
	}
	err := RenderMarkdownTable(&buf, headers, rows)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := buf.String()

	// Pipes inside cell content should be escaped
	if !strings.Contains(out, `a \| b`) {
		t.Errorf("expected pipes in cell content to be escaped, got: %q", out)
	}
	if !strings.Contains(out, `x\|y`) {
		t.Errorf("expected pipes in cell content to be escaped, got: %q", out)
	}
}

func TestRenderMarkdownTable_EmptyRows(t *testing.T) {
	var buf bytes.Buffer
	headers := []string{"Col1", "Col2"}
	err := RenderMarkdownTable(&buf, headers, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := buf.String()
	lines := strings.Split(strings.TrimSpace(out), "\n")

	// Should have header + separator only
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines (header + separator), got %d: %q", len(lines), out)
	}
	if !strings.Contains(lines[0], "Col1") {
		t.Errorf("header should contain Col1, got: %q", lines[0])
	}
}

func TestRenderMarkdownTable_PadsShortRows(t *testing.T) {
	var buf bytes.Buffer
	headers := []string{"A", "B", "C"}
	rows := [][]string{
		{"1"}, // only 1 cell, should be padded to 3
	}
	err := RenderMarkdownTable(&buf, headers, rows)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := buf.String()
	lines := strings.Split(strings.TrimSpace(out), "\n")

	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d: %q", len(lines), out)
	}

	// Data row should have the right number of pipe separators
	dataRow := lines[2]
	pipeCount := strings.Count(dataRow, "|") - strings.Count(dataRow, `\|`)
	// A markdown row "| 1 |  |  |" has 4 unescaped pipes for 3 columns
	if pipeCount < 4 {
		t.Errorf("expected padded row with 3 columns (4 pipes), got pipe count %d in: %q", pipeCount, dataRow)
	}
}

func TestRenderMarkdownTable_EmptyHeaders(t *testing.T) {
	var buf bytes.Buffer
	err := RenderMarkdownTable(&buf, nil, [][]string{{"data"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := buf.String()
	if out != "" {
		t.Errorf("expected empty output for empty headers, got: %q", out)
	}
}
