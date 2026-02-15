package reviews

import (
	"context"
	"flag"
	"strings"
	"testing"
)

func TestReviewsCommand_Name(t *testing.T) {
	cmd := ReviewsCommand()
	if cmd.Name != "reviews" {
		t.Errorf("expected name %q, got %q", "reviews", cmd.Name)
	}
}

func TestReviewsCommand_ShortHelp(t *testing.T) {
	cmd := ReviewsCommand()
	if cmd.ShortHelp == "" {
		t.Error("expected non-empty ShortHelp")
	}
}

func TestReviewsCommand_UsageFunc(t *testing.T) {
	cmd := ReviewsCommand()
	if cmd.UsageFunc == nil {
		t.Error("expected UsageFunc to be set")
	}
}

func TestReviewsCommand_HasSubcommands(t *testing.T) {
	cmd := ReviewsCommand()
	if len(cmd.Subcommands) == 0 {
		t.Error("expected subcommands")
	}
}

func TestReviewsCommand_SubcommandNames(t *testing.T) {
	cmd := ReviewsCommand()
	expected := map[string]bool{
		"list":  false,
		"get":   false,
		"reply": false,
	}
	for _, sub := range cmd.Subcommands {
		if _, ok := expected[sub.Name]; ok {
			expected[sub.Name] = true
		} else {
			t.Errorf("unexpected subcommand: %s", sub.Name)
		}
	}
	for name, found := range expected {
		if !found {
			t.Errorf("missing subcommand: %s", name)
		}
	}
}

func TestReviewsCommand_SubcommandsHaveUsageFunc(t *testing.T) {
	cmd := ReviewsCommand()
	for _, sub := range cmd.Subcommands {
		if sub.UsageFunc == nil {
			t.Errorf("subcommand %q missing UsageFunc", sub.Name)
		}
	}
}

func TestReviewsCommand_NoArgs_ReturnsHelp(t *testing.T) {
	cmd := ReviewsCommand()
	err := cmd.Exec(context.Background(), nil)
	if err != flag.ErrHelp {
		t.Errorf("expected flag.ErrHelp, got %v", err)
	}
}

// --- reviews get ---

func TestReviewsGetCommand_Name(t *testing.T) {
	cmd := GetCommand()
	if cmd.Name != "get" {
		t.Errorf("expected name %q, got %q", "get", cmd.Name)
	}
}

func TestReviewsGetCommand_MissingReview(t *testing.T) {
	cmd := GetCommand()
	if err := cmd.FlagSet.Parse([]string{}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for missing --review")
	}
	if !strings.Contains(err.Error(), "--review") {
		t.Errorf("error should mention --review, got: %s", err.Error())
	}
}

func TestReviewsGetCommand_WhitespaceReview(t *testing.T) {
	cmd := GetCommand()
	if err := cmd.FlagSet.Parse([]string{"--review", "   "}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for whitespace-only --review")
	}
	if !strings.Contains(err.Error(), "--review") {
		t.Errorf("error should mention --review, got: %s", err.Error())
	}
}

func TestReviewsGetCommand_InvalidOutputFormat(t *testing.T) {
	cmd := GetCommand()
	if err := cmd.FlagSet.Parse([]string{"--output", "xml"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for invalid output format")
	}
}

func TestReviewsGetCommand_PrettyWithTable(t *testing.T) {
	cmd := GetCommand()
	if err := cmd.FlagSet.Parse([]string{"--output", "table", "--pretty"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for --pretty with table output")
	}
	if !strings.Contains(err.Error(), "--pretty") {
		t.Errorf("error should mention --pretty, got: %s", err.Error())
	}
}

// --- reviews reply ---

func TestReviewsReplyCommand_Name(t *testing.T) {
	cmd := ReplyCommand()
	if cmd.Name != "reply" {
		t.Errorf("expected name %q, got %q", "reply", cmd.Name)
	}
}

func TestReviewsReplyCommand_MissingReview(t *testing.T) {
	cmd := ReplyCommand()
	if err := cmd.FlagSet.Parse([]string{"--text", "Thank you!"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for missing --review")
	}
	if !strings.Contains(err.Error(), "--review") {
		t.Errorf("error should mention --review, got: %s", err.Error())
	}
}

func TestReviewsReplyCommand_MissingText(t *testing.T) {
	cmd := ReplyCommand()
	if err := cmd.FlagSet.Parse([]string{"--review", "abc123"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for missing --text")
	}
	if !strings.Contains(err.Error(), "--text") {
		t.Errorf("error should mention --text, got: %s", err.Error())
	}
}

func TestReviewsReplyCommand_WhitespaceText(t *testing.T) {
	cmd := ReplyCommand()
	if err := cmd.FlagSet.Parse([]string{"--review", "abc123", "--text", "   "}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for whitespace-only --text")
	}
	if !strings.Contains(err.Error(), "--text") {
		t.Errorf("error should mention --text, got: %s", err.Error())
	}
}

// --- reviews list ---

func TestReviewsListCommand_Name(t *testing.T) {
	cmd := ListCommand()
	if cmd.Name != "list" {
		t.Errorf("expected name %q, got %q", "list", cmd.Name)
	}
}

func TestReviewsListCommand_InvalidOutputFormat(t *testing.T) {
	cmd := ListCommand()
	if err := cmd.FlagSet.Parse([]string{"--output", "csv"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for invalid output format")
	}
}

func TestReviewsListCommand_PrettyWithMarkdown(t *testing.T) {
	cmd := ListCommand()
	if err := cmd.FlagSet.Parse([]string{"--output", "markdown", "--pretty"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for --pretty with markdown output")
	}
	if !strings.Contains(err.Error(), "--pretty") {
		t.Errorf("error should mention --pretty, got: %s", err.Error())
	}
}
