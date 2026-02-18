package releasenotes

import (
	"testing"
)

func TestFormat_EmptyCommits(t *testing.T) {
	result := Format(nil, 500)
	if result != "" {
		t.Errorf("Format(nil, 500) = %q, want empty string", result)
	}
}

func TestFormat_SingleCommit(t *testing.T) {
	commits := []GitCommit{
		{Hash: "abc1234", Subject: "Fix login crash"},
	}
	result := Format(commits, 500)
	want := "- Fix login crash"
	if result != want {
		t.Errorf("Format() = %q, want %q", result, want)
	}
}

func TestFormat_MultipleCommits(t *testing.T) {
	commits := []GitCommit{
		{Hash: "abc1234", Subject: "Fix login crash"},
		{Hash: "def5678", Subject: "Add dark mode"},
		{Hash: "ghi9012", Subject: "Update translations"},
	}
	result := Format(commits, 500)
	want := "- Fix login crash\n- Add dark mode\n- Update translations"
	if result != want {
		t.Errorf("Format() = %q, want %q", result, want)
	}
}

func TestFormat_NoTruncationWhenUnderLimit(t *testing.T) {
	commits := []GitCommit{
		{Hash: "abc1234", Subject: "Short"},
	}
	result := Format(commits, 500)
	if len(result) > 500 {
		t.Errorf("result length %d exceeds maxChars 500", len(result))
	}
}

func TestFormat_TruncationAtLineBoundary(t *testing.T) {
	commits := []GitCommit{
		{Hash: "a", Subject: "First commit message"},    // "- First commit message" = 22 chars
		{Hash: "b", Subject: "Second commit message"},   // "- Second commit message" = 23 chars
		{Hash: "c", Subject: "Third commit message"},    // "- Third commit message" = 22 chars
		{Hash: "d", Subject: "Fourth commit message"},   // "- Fourth commit message" = 23 chars
	}

	// "- First commit message\n- Second commit message" = 46 chars
	// + "\n..." = 50 chars
	// "- Third commit message" would make it 69 + "\n..." = 73 chars
	result := Format(commits, 55)

	// Should contain first two lines and ellipsis
	want := "- First commit message\n- Second commit message\n..."
	if result != want {
		t.Errorf("Format() = %q, want %q", result, want)
	}
}

func TestFormat_TruncationDoesNotCutMidLine(t *testing.T) {
	commits := []GitCommit{
		{Hash: "a", Subject: "First commit"},
		{Hash: "b", Subject: "Second commit that is longer"},
		{Hash: "c", Subject: "Third commit"},
	}

	// Set limit so that cutting mid-line of the second commit would be needed
	// "- First commit" = 14 chars
	// "\n- Second commit that is longer" = 31 chars more = 45 total
	// "\n..." = 4 more = 49
	// Let's set limit to 30 so only first line fits
	result := Format(commits, 30)
	want := "- First commit\n..."
	if result != want {
		t.Errorf("Format() = %q, want %q", result, want)
	}
}

func TestFormat_ResultWithinMaxChars(t *testing.T) {
	commits := []GitCommit{
		{Hash: "a", Subject: "Alpha"},
		{Hash: "b", Subject: "Beta"},
		{Hash: "c", Subject: "Gamma"},
		{Hash: "d", Subject: "Delta"},
		{Hash: "e", Subject: "Epsilon"},
	}
	maxChars := 40
	result := Format(commits, maxChars)
	if len(result) > maxChars {
		t.Errorf("result length %d exceeds maxChars %d: %q", len(result), maxChars, result)
	}
}

func TestFormat_ZeroMaxCharsNoTruncation(t *testing.T) {
	commits := []GitCommit{
		{Hash: "a", Subject: "First"},
		{Hash: "b", Subject: "Second"},
	}
	result := Format(commits, 0)
	want := "- First\n- Second"
	if result != want {
		t.Errorf("Format() = %q, want %q", result, want)
	}
}

func TestFormat_ExactFitNoEllipsis(t *testing.T) {
	commits := []GitCommit{
		{Hash: "a", Subject: "Hello"},
	}
	// "- Hello" = 7 chars
	result := Format(commits, 7)
	want := "- Hello"
	if result != want {
		t.Errorf("Format() = %q, want %q", result, want)
	}
}

func TestFormat_VerySmallLimit(t *testing.T) {
	commits := []GitCommit{
		{Hash: "a", Subject: "A long commit message that exceeds the limit"},
	}
	result := Format(commits, 10)
	// Budget is 10 - 4 ("\n...") = 6. No full line fits, so we truncate the first line.
	if len(result) > 10 {
		t.Errorf("result length %d exceeds maxChars 10: %q", len(result), result)
	}
	if result == "" {
		t.Error("expected non-empty result")
	}
}
