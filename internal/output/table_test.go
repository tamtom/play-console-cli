package output

import (
	"bytes"
	"strings"
	"testing"
)

func TestRenderTableTo(t *testing.T) {
	var buf bytes.Buffer
	headers := []string{"Name", "Value"}
	rows := [][]string{
		{"alpha", "1"},
		{"beta", "2"},
	}
	RenderTableTo(&buf, headers, rows)
	out := buf.String()
	if !strings.Contains(out, "Name") {
		t.Error("expected headers in output")
	}
	if !strings.Contains(out, "alpha") {
		t.Error("expected row data in output")
	}
	if !strings.Contains(out, "---") {
		t.Error("expected separator in output")
	}
}

func TestRenderTableTo_Empty(t *testing.T) {
	var buf bytes.Buffer
	RenderTableTo(&buf, []string{"Col"}, nil)
	if !strings.Contains(buf.String(), "No results found") {
		t.Error("expected empty message")
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		input  string
		maxLen int
		want   string
	}{
		{"hello", 10, "hello"},
		{"hello world", 8, "hello..."},
		{"hi", 2, "hi"},
		{"hello", 3, "hel"},
	}
	for _, tt := range tests {
		got := Truncate(tt.input, tt.maxLen)
		if got != tt.want {
			t.Errorf("Truncate(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
		}
	}
}
