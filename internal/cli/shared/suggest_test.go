package shared

import "testing"

func TestLevenshteinDistance(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"tracks", "trakcs", 2},
		{"auth", "auth", 0},
		{"", "abc", 3},
		{"abc", "", 3},
		{"kitten", "sitting", 3},
	}
	for _, tt := range tests {
		got := LevenshteinDistance(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("LevenshteinDistance(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestSuggestCommand(t *testing.T) {
	commands := []string{"auth", "tracks", "listings", "reviews", "bundles", "images"}
	tests := []struct {
		input string
		want  string
	}{
		{"trakcs", "tracks"},
		{"atuh", "auth"},
		{"listing", "listings"},
		{"xyzabc123", ""},
	}
	for _, tt := range tests {
		got := SuggestCommand(tt.input, commands, 3)
		if got != tt.want {
			t.Errorf("SuggestCommand(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestFormatUnknownCommand(t *testing.T) {
	commands := []string{"auth", "tracks"}
	msg := FormatUnknownCommand("trakcs", commands)
	if msg == "" {
		t.Error("expected non-empty message")
	}
}
