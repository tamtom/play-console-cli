package output

import (
	"bytes"
	"strings"
	"testing"
)

func TestRenderTableTo_Basic(t *testing.T) {
	var buf bytes.Buffer
	headers := []string{"Name", "Value"}
	rows := [][]string{
		{"alpha", "1"},
		{"beta", "2"},
	}
	RenderTableTo(&buf, headers, rows)
	out := buf.String()

	// Should contain headers (tablewriter auto-formats to uppercase)
	if !strings.Contains(strings.ToUpper(out), "NAME") {
		t.Error("expected headers in output")
	}
	// Should contain row data
	if !strings.Contains(out, "alpha") {
		t.Error("expected row data 'alpha' in output")
	}
	if !strings.Contains(out, "beta") {
		t.Error("expected row data 'beta' in output")
	}
	// Should use bordered table format (tablewriter uses Unicode box-drawing)
	if !strings.Contains(out, "│") && !strings.Contains(out, "|") {
		t.Error("expected column separator in output")
	}
}

func TestRenderTableTo_EmptyRows(t *testing.T) {
	var buf bytes.Buffer
	RenderTableTo(&buf, []string{"Col"}, nil)
	out := buf.String()
	// Even with no rows, headers should still render
	if !strings.Contains(strings.ToUpper(out), "COL") {
		t.Error("expected headers in output even with empty rows")
	}
}

func TestRenderTableTo_SpecialCharacters(t *testing.T) {
	var buf bytes.Buffer
	headers := []string{"Key", "Value"}
	rows := [][]string{
		{"emoji", "\U0001F600"},
		{"ansi", "\x1b[31mred\x1b[0m"},
		{"control", "hello\x00world"},
	}
	RenderTableTo(&buf, headers, rows)
	out := buf.String()

	// ANSI codes should be sanitized
	if strings.Contains(out, "\x1b[") {
		t.Error("expected ANSI codes to be stripped from output")
	}
	// Control characters should be sanitized
	if strings.Contains(out, "\x00") {
		t.Error("expected control characters to be stripped from output")
	}
	// Emoji should be preserved
	if !strings.Contains(out, "\U0001F600") {
		t.Error("expected emoji to be preserved in output")
	}
	// "red" text should remain after stripping ANSI
	if !strings.Contains(out, "red") {
		t.Error("expected 'red' text to remain after ANSI stripping")
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		input  string
		maxLen int
		want   string
	}{
		{"hello", 10, "hello"},         // shorter than maxLen
		{"hello", 5, "hello"},          // equal to maxLen
		{"hello world", 8, "hello..."}, // longer than maxLen
		{"hi", 2, "hi"},                // equal, no truncation
		{"hello", 3, "hel"},            // maxLen <= 3, no ellipsis
		{"abcd", 1, "a"},               // maxLen = 1
	}
	for _, tt := range tests {
		got := Truncate(tt.input, tt.maxLen)
		if got != tt.want {
			t.Errorf("Truncate(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
		}
	}
}
