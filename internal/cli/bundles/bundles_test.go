package bundles

import (
	"context"
	"errors"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tamtom/play-console-cli/internal/playclient"
)

func TestBundlesCommand_Name(t *testing.T) {
	cmd := BundlesCommand()
	if cmd.Name != "bundles" {
		t.Errorf("expected name %q, got %q", "bundles", cmd.Name)
	}
}

func TestBundlesCommand_ShortHelp(t *testing.T) {
	cmd := BundlesCommand()
	if cmd.ShortHelp == "" {
		t.Error("expected non-empty ShortHelp")
	}
}

func TestBundlesCommand_UsageFunc(t *testing.T) {
	cmd := BundlesCommand()
	if cmd.UsageFunc == nil {
		t.Error("expected UsageFunc to be set")
	}
}

func TestBundlesCommand_HasSubcommands(t *testing.T) {
	cmd := BundlesCommand()
	if len(cmd.Subcommands) == 0 {
		t.Error("expected subcommands")
	}
}

func TestBundlesCommand_SubcommandNames(t *testing.T) {
	cmd := BundlesCommand()
	expected := map[string]bool{
		"upload":  false,
		"list":    false,
		"analyze": false,
		"compare": false,
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

func TestBundlesCommand_SubcommandsHaveUsageFunc(t *testing.T) {
	cmd := BundlesCommand()
	for _, sub := range cmd.Subcommands {
		if sub.UsageFunc == nil {
			t.Errorf("subcommand %q missing UsageFunc", sub.Name)
		}
	}
}

func TestBundlesCommand_SubcommandsHaveShortHelp(t *testing.T) {
	cmd := BundlesCommand()
	for _, sub := range cmd.Subcommands {
		if sub.ShortHelp == "" {
			t.Errorf("subcommand %q missing ShortHelp", sub.Name)
		}
	}
}

func TestBundlesCommand_NoArgs_ReturnsHelp(t *testing.T) {
	cmd := BundlesCommand()
	err := cmd.Exec(context.Background(), nil)
	if !errors.Is(err, flag.ErrHelp) {
		t.Errorf("expected flag.ErrHelp, got %v", err)
	}
}

// --- bundles upload ---

func TestBundlesUploadCommand_Name(t *testing.T) {
	cmd := UploadCommand()
	if cmd.Name != "upload" {
		t.Errorf("expected name %q, got %q", "upload", cmd.Name)
	}
}

func TestBundlesUploadCommand_MissingFile(t *testing.T) {
	cmd := UploadCommand()
	if err := cmd.FlagSet.Parse([]string{}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for missing --file")
	}
	if !strings.Contains(err.Error(), "--file") {
		t.Errorf("error should mention --file, got: %s", err.Error())
	}
}

func TestBundlesUploadCommand_WhitespaceFile(t *testing.T) {
	cmd := UploadCommand()
	if err := cmd.FlagSet.Parse([]string{"--file", "   "}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for whitespace-only --file")
	}
	if !strings.Contains(err.Error(), "--file") {
		t.Errorf("error should mention --file, got: %s", err.Error())
	}
}

func TestBundlesUploadCommand_InvalidOutputFormat(t *testing.T) {
	cmd := UploadCommand()
	if err := cmd.FlagSet.Parse([]string{"--output", "csv"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for invalid output format")
	}
}

func TestBundlesUploadCommand_PrettyWithTable(t *testing.T) {
	cmd := UploadCommand()
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

func TestBundlesUploadCommand_RejectsEmptyFileBeforeServiceCreation(t *testing.T) {
	emptyBundle := filepath.Join(t.TempDir(), "empty.aab")
	if err := os.WriteFile(emptyBundle, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	cmd := UploadCommand()
	if err := cmd.FlagSet.Parse([]string{
		"--package", "com.example.app",
		"--edit", "edit-1",
		"--file", emptyBundle,
	}); err != nil {
		t.Fatal(err)
	}

	serviceCreated := false
	ctx := playclient.ContextWithServiceFactory(context.Background(), func(context.Context) (*playclient.Service, error) {
		serviceCreated = true
		return nil, errors.New("service should not be created")
	})

	err := cmd.Exec(ctx, nil)
	if err == nil {
		t.Fatal("expected empty upload error")
	}
	if !strings.Contains(err.Error(), "empty") {
		t.Fatalf("error = %q, want empty upload error", err)
	}
	if serviceCreated {
		t.Fatal("service was created before rejecting empty upload")
	}
}

// --- bundles list ---

func TestBundlesListCommand_Name(t *testing.T) {
	cmd := ListCommand()
	if cmd.Name != "list" {
		t.Errorf("expected name %q, got %q", "list", cmd.Name)
	}
}

func TestBundlesListCommand_InvalidOutputFormat(t *testing.T) {
	cmd := ListCommand()
	if err := cmd.FlagSet.Parse([]string{"--output", "xml"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for invalid output format")
	}
}

func TestBundlesListCommand_PrettyWithMarkdown(t *testing.T) {
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
