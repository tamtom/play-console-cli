package releasenotes

import (
	"context"
	"errors"
	"flag"
	"strings"
	"testing"
)

func TestReleaseNotesCommand_Name(t *testing.T) {
	cmd := ReleaseNotesCommand()
	if cmd.Name != "release-notes" {
		t.Errorf("Name = %q, want %q", cmd.Name, "release-notes")
	}
}

func TestReleaseNotesCommand_HasSubcommands(t *testing.T) {
	cmd := ReleaseNotesCommand()
	if len(cmd.Subcommands) == 0 {
		t.Fatal("expected subcommands")
	}
	found := false
	for _, sub := range cmd.Subcommands {
		if sub.Name == "generate" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'generate' subcommand")
	}
}

func TestReleaseNotesCommand_NoArgsReturnsHelp(t *testing.T) {
	cmd := ReleaseNotesCommand()
	err := cmd.Exec(context.Background(), nil)
	if !errors.Is(err, flag.ErrHelp) {
		t.Errorf("expected flag.ErrHelp, got %v", err)
	}
}

func TestGenerateCommand_Flags(t *testing.T) {
	cmd := GenerateCommand()
	if cmd.FlagSet == nil {
		t.Fatal("expected FlagSet")
	}

	tests := []struct {
		name     string
		defValue string
	}{
		{"since-tag", ""},
		{"since-ref", ""},
		{"until-ref", "HEAD"},
		{"max-chars", "500"},
		{"output", "json"},
	}

	for _, tt := range tests {
		f := cmd.FlagSet.Lookup(tt.name)
		if f == nil {
			t.Errorf("flag --%s not found", tt.name)
			continue
		}
		if f.DefValue != tt.defValue {
			t.Errorf("flag --%s default = %q, want %q", tt.name, f.DefValue, tt.defValue)
		}
	}
}

func TestRunGenerate_MutuallyExclusiveFlags(t *testing.T) {
	err := runGenerate(context.Background(), generateOpts{
		sinceTag:   "v1.0.0",
		sinceRef:   "abc1234",
		untilRef:   "HEAD",
		maxChars:   500,
		outputFlag: "json",
	})
	if !errors.Is(err, flag.ErrHelp) {
		t.Errorf("expected flag.ErrHelp for mutually exclusive flags, got %v", err)
	}
}

func TestRunGenerate_NeitherFlagProvided(t *testing.T) {
	err := runGenerate(context.Background(), generateOpts{
		sinceTag:   "",
		sinceRef:   "",
		untilRef:   "HEAD",
		maxChars:   500,
		outputFlag: "json",
	})
	if !errors.Is(err, flag.ErrHelp) {
		t.Errorf("expected flag.ErrHelp when neither flag provided, got %v", err)
	}
}

func TestRunGenerate_WhitespaceOnlyFlags(t *testing.T) {
	err := runGenerate(context.Background(), generateOpts{
		sinceTag:   "  ",
		sinceRef:   "  ",
		untilRef:   "HEAD",
		maxChars:   500,
		outputFlag: "json",
	})
	if !errors.Is(err, flag.ErrHelp) {
		t.Errorf("expected flag.ErrHelp for whitespace-only flags, got %v", err)
	}
}

func TestRunGenerate_SinceTagOnly(t *testing.T) {
	// This will fail with a git error since we're not in a real repo context
	// with the specified tag, but it should NOT fail with flag validation errors.
	err := runGenerate(context.Background(), generateOpts{
		sinceTag:   "v1.0.0",
		sinceRef:   "",
		untilRef:   "HEAD",
		maxChars:   500,
		outputFlag: "json",
	})
	// Should get a git error, not a flag validation error
	if errors.Is(err, flag.ErrHelp) {
		t.Error("did not expect flag.ErrHelp when --since-tag is provided")
	}
	if err != nil && !strings.Contains(err.Error(), "git") {
		t.Errorf("expected git-related error, got: %v", err)
	}
}

func TestRunGenerate_SinceRefOnly(t *testing.T) {
	err := runGenerate(context.Background(), generateOpts{
		sinceTag:   "",
		sinceRef:   "abc1234",
		untilRef:   "HEAD",
		maxChars:   500,
		outputFlag: "json",
	})
	// Should get a git error, not a flag validation error
	if errors.Is(err, flag.ErrHelp) {
		t.Error("did not expect flag.ErrHelp when --since-ref is provided")
	}
	if err != nil && !strings.Contains(err.Error(), "git") {
		t.Errorf("expected git-related error, got: %v", err)
	}
}
