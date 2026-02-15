package output

import (
	"os"
	"strings"
	"testing"
)

func TestColorsEnabledRespectsNO_COLOR(t *testing.T) {
	// Save and restore original value.
	orig, hadOrig := os.LookupEnv("NO_COLOR")
	defer func() {
		if hadOrig {
			os.Setenv("NO_COLOR", orig)
		} else {
			os.Unsetenv("NO_COLOR")
		}
	}()

	os.Setenv("NO_COLOR", "1")
	initColors()
	if ColorsEnabled() {
		t.Error("expected colors disabled when NO_COLOR is set")
	}

	os.Unsetenv("NO_COLOR")
	// Force re-init; in a non-TTY test env colors will still be off due to
	// TTY detection, so we test the NO_COLOR path explicitly above.
	initColors()
}

func TestColorsEnabledWhenNO_COLOREmpty(t *testing.T) {
	orig, hadOrig := os.LookupEnv("NO_COLOR")
	defer func() {
		if hadOrig {
			os.Setenv("NO_COLOR", orig)
		} else {
			os.Unsetenv("NO_COLOR")
		}
	}()

	// NO_COLOR spec: any non-empty value disables colors. Empty string means
	// the variable is set but empty -- spec says "when set", so we treat empty
	// as NOT disabling colors to match the most common interpretation.
	// However, the spec at no-color.org says "command-line software which
	// outputs text with ANSI color added should check for the presence of a
	// NO_COLOR environment variable that, when present, prevents ANSI color."
	// "present" means os.LookupEnv returns true, regardless of value.
	os.Setenv("NO_COLOR", "")
	initColors()
	if ColorsEnabled() {
		// Even an empty NO_COLOR disables colors per the spec ("when present").
		t.Error("expected colors disabled when NO_COLOR is set (even empty)")
	}
}

func TestColorFunctionsWithColorsEnabled(t *testing.T) {
	// Force colors on for this test.
	colorEnabled = true
	defer func() { initColors() }()

	tests := []struct {
		name     string
		fn       func(string) string
		input    string
		contains string
	}{
		{"Green", Green, "ok", "\033[32mok\033[0m"},
		{"Yellow", Yellow, "warn", "\033[33mwarn\033[0m"},
		{"Red", Red, "err", "\033[31merr\033[0m"},
		{"Bold", Bold, "title", "\033[1mtitle\033[0m"},
		{"Cyan", Cyan, "info", "\033[36minfo\033[0m"},
		{"Dim", Dim, "faint", "\033[2mfaint\033[0m"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.fn(tt.input)
			if got != tt.contains {
				t.Errorf("%s(%q) = %q, want %q", tt.name, tt.input, got, tt.contains)
			}
		})
	}
}

func TestColorFunctionsWithColorsDisabled(t *testing.T) {
	colorEnabled = false
	defer func() { initColors() }()

	tests := []struct {
		name  string
		fn    func(string) string
		input string
	}{
		{"Green", Green, "ok"},
		{"Yellow", Yellow, "warn"},
		{"Red", Red, "err"},
		{"Bold", Bold, "title"},
		{"Cyan", Cyan, "info"},
		{"Dim", Dim, "faint"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.fn(tt.input)
			if got != tt.input {
				t.Errorf("%s(%q) = %q, want plain %q", tt.name, tt.input, got, tt.input)
			}
			if strings.Contains(got, "\033[") {
				t.Errorf("%s(%q) contains ANSI codes when colors are disabled", tt.name, tt.input)
			}
		})
	}
}

func TestStatusColor(t *testing.T) {
	colorEnabled = true
	defer func() { initColors() }()

	tests := []struct {
		status    string
		wantCode  string
		wantPlain string
	}{
		{"completed", "\033[32m", "completed"},
		{"active", "\033[32m", "active"},
		{"draft", "\033[33m", "draft"},
		{"inProgress", "\033[33m", "inProgress"},
		{"halted", "\033[31m", "halted"},
		{"failed", "\033[31m", "failed"},
		{"unknown", "", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			got := StatusColor(tt.status)
			if tt.wantCode != "" {
				if !strings.HasPrefix(got, tt.wantCode) {
					t.Errorf("StatusColor(%q) = %q, want prefix %q", tt.status, got, tt.wantCode)
				}
				if !strings.HasSuffix(got, "\033[0m") {
					t.Errorf("StatusColor(%q) = %q, want suffix \\033[0m", tt.status, got)
				}
			} else {
				// Unknown status returns plain string.
				if got != tt.wantPlain {
					t.Errorf("StatusColor(%q) = %q, want %q", tt.status, got, tt.wantPlain)
				}
			}
		})
	}
}

func TestStatusColorDisabled(t *testing.T) {
	colorEnabled = false
	defer func() { initColors() }()

	statuses := []string{"completed", "active", "draft", "inProgress", "halted", "failed", "unknown"}
	for _, s := range statuses {
		t.Run(s, func(t *testing.T) {
			got := StatusColor(s)
			if got != s {
				t.Errorf("StatusColor(%q) = %q, want plain %q when disabled", s, got, s)
			}
		})
	}
}

func TestEmptyStringColorFunctions(t *testing.T) {
	colorEnabled = true
	defer func() { initColors() }()

	// Even empty strings get wrapped.
	if got := Green(""); got != "\033[32m\033[0m" {
		t.Errorf("Green(\"\") = %q, want wrapped empty", got)
	}
}
