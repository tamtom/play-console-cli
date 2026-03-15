package docs

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"os"
	"strings"
	"testing"
)

func TestListCommand_Name(t *testing.T) {
	cmd := ListCommand()
	if cmd.Name != "list" {
		t.Errorf("Name = %q, want %q", cmd.Name, "list")
	}
}

func TestShowCommand_Name(t *testing.T) {
	cmd := ShowCommand()
	if cmd.Name != "show" {
		t.Errorf("Name = %q, want %q", cmd.Name, "show")
	}
}

func TestDocsCommand_HasShowAndListSubcommands(t *testing.T) {
	cmd := DocsCommand()
	names := map[string]bool{}
	for _, sub := range cmd.Subcommands {
		names[sub.Name] = true
	}
	if !names["show"] {
		t.Error("expected 'show' subcommand")
	}
	if !names["list"] {
		t.Error("expected 'list' subcommand")
	}
	if !names["generate"] {
		t.Error("expected 'generate' subcommand")
	}
}

func TestTopicSlugs(t *testing.T) {
	slugs := topicSlugs()
	if len(slugs) == 0 {
		t.Fatal("expected at least one topic")
	}

	expected := []string{"auth-setup", "release-workflow", "metadata-format", "tracks", "troubleshooting"}
	for _, want := range expected {
		found := false
		for _, got := range slugs {
			if got == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected topic slug %q", want)
		}
	}
}

func TestFindTopic_Exists(t *testing.T) {
	topic, ok := findTopic("auth-setup")
	if !ok {
		t.Fatal("expected to find auth-setup topic")
	}
	if topic.Slug != "auth-setup" {
		t.Errorf("Slug = %q, want %q", topic.Slug, "auth-setup")
	}
	if topic.Content == "" {
		t.Error("expected non-empty content")
	}
}

func TestFindTopic_NotExists(t *testing.T) {
	_, ok := findTopic("nonexistent-topic")
	if ok {
		t.Error("expected not to find nonexistent topic")
	}
}

func TestFindTopic_CaseInsensitive(t *testing.T) {
	_, ok := findTopic("Auth-Setup")
	if !ok {
		t.Error("expected case-insensitive lookup to work")
	}
}

func TestShowCommand_NoArgs(t *testing.T) {
	cmd := ShowCommand()
	err := cmd.Exec(context.Background(), nil)
	if !errors.Is(err, flag.ErrHelp) {
		t.Errorf("expected flag.ErrHelp, got %v", err)
	}
}

func TestShowCommand_TooManyArgs(t *testing.T) {
	cmd := ShowCommand()
	// Redirect stderr to capture error messages
	old := os.Stderr
	os.Stderr, _ = os.CreateTemp("", "stderr")
	defer func() { os.Stderr = old }()

	err := cmd.Exec(context.Background(), []string{"one", "two"})
	if !errors.Is(err, flag.ErrHelp) {
		t.Errorf("expected flag.ErrHelp, got %v", err)
	}
}

func TestShowCommand_UnknownTopic(t *testing.T) {
	cmd := ShowCommand()
	old := os.Stderr
	os.Stderr, _ = os.CreateTemp("", "stderr")
	defer func() { os.Stderr = old }()

	err := cmd.Exec(context.Background(), []string{"nonexistent"})
	if !errors.Is(err, flag.ErrHelp) {
		t.Errorf("expected flag.ErrHelp, got %v", err)
	}
}

func TestShowCommand_ValidTopic(t *testing.T) {
	cmd := ShowCommand()

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := cmd.Exec(context.Background(), []string{"auth-setup"})

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(r); err != nil {
		t.Fatalf("reading pipe: %v", err)
	}
	output := buf.String()

	if !strings.Contains(output, "Authentication Setup") {
		t.Error("expected output to contain topic content")
	}
}

func TestListCommand_Output(t *testing.T) {
	cmd := ListCommand()

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := cmd.Exec(context.Background(), nil)

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(r); err != nil {
		t.Fatalf("reading pipe: %v", err)
	}
	output := buf.String()

	if !strings.Contains(output, "auth-setup") {
		t.Error("expected output to contain auth-setup")
	}
	if !strings.Contains(output, "troubleshooting") {
		t.Error("expected output to contain troubleshooting")
	}
}

func TestTopicContent_NonEmpty(t *testing.T) {
	for _, slug := range topicSlugs() {
		topic, ok := findTopic(slug)
		if !ok {
			t.Errorf("topic %q not found", slug)
			continue
		}
		if strings.TrimSpace(topic.Content) == "" {
			t.Errorf("topic %q has empty content", slug)
		}
	}
}
