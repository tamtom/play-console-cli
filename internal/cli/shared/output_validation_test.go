package shared

import (
	"context"
	"flag"
	"strings"
	"testing"

	"github.com/peterbourgon/ff/v3/ffcli"
)

func TestWrapCommandOutputValidation_ValidJSON(t *testing.T) {
	executed := false
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	fs.String("output", "json", "Output format")

	cmd := &ffcli.Command{
		Name:    "test",
		FlagSet: fs,
		Exec: func(ctx context.Context, args []string) error {
			executed = true
			return nil
		},
	}

	WrapCommandOutputValidation(cmd)

	if err := fs.Parse([]string{"--output", "json"}); err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if err := cmd.Exec(context.Background(), nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !executed {
		t.Error("original Exec should have been called")
	}
}

func TestWrapCommandOutputValidation_InvalidXML(t *testing.T) {
	executed := false
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	fs.String("output", "json", "Output format")

	cmd := &ffcli.Command{
		Name:    "test",
		FlagSet: fs,
		Exec: func(ctx context.Context, args []string) error {
			executed = true
			return nil
		},
	}

	WrapCommandOutputValidation(cmd)

	if err := fs.Parse([]string{"--output", "xml"}); err != nil {
		t.Fatalf("parse error: %v", err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for invalid output format")
	}
	if !strings.Contains(err.Error(), "unsupported output format") {
		t.Errorf("error should mention unsupported output format, got: %v", err)
	}
	if executed {
		t.Error("original Exec should NOT have been called for invalid format")
	}
}

func TestWrapCommandOutputValidation_PrettyWithTable(t *testing.T) {
	executed := false
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	fs.String("output", "json", "Output format")
	fs.Bool("pretty", false, "Pretty-print")

	cmd := &ffcli.Command{
		Name:    "test",
		FlagSet: fs,
		Exec: func(ctx context.Context, args []string) error {
			executed = true
			return nil
		},
	}

	WrapCommandOutputValidation(cmd)

	if err := fs.Parse([]string{"--output", "table", "--pretty"}); err != nil {
		t.Fatalf("parse error: %v", err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for --pretty with table output")
	}
	if !strings.Contains(err.Error(), "--pretty is only valid with JSON output") {
		t.Errorf("error should mention --pretty incompatibility, got: %v", err)
	}
	if executed {
		t.Error("original Exec should NOT have been called")
	}
}

func TestWrapCommandOutputValidation_NilCommand(t *testing.T) {
	// Should not panic
	WrapCommandOutputValidation(nil)
}

func TestWrapCommandOutputValidation_RecursiveSubcommands(t *testing.T) {
	parentExecuted := false
	childExecuted := false

	parentFS := flag.NewFlagSet("parent", flag.ContinueOnError)
	childFS := flag.NewFlagSet("child", flag.ContinueOnError)
	childFS.String("output", "json", "Output format")

	child := &ffcli.Command{
		Name:    "child",
		FlagSet: childFS,
		Exec: func(ctx context.Context, args []string) error {
			childExecuted = true
			return nil
		},
	}

	parent := &ffcli.Command{
		Name:        "parent",
		FlagSet:     parentFS,
		Subcommands: []*ffcli.Command{child},
		Exec: func(ctx context.Context, args []string) error {
			parentExecuted = true
			return nil
		},
	}

	WrapCommandOutputValidation(parent)

	// Test that child with invalid output is caught
	if err := childFS.Parse([]string{"--output", "xml"}); err != nil {
		t.Fatalf("parse error: %v", err)
	}
	err := child.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for invalid output format on child command")
	}
	if childExecuted {
		t.Error("child Exec should NOT have been called")
	}

	// Reset and test parent still works
	if err := parent.Exec(context.Background(), nil); err != nil {
		t.Fatalf("unexpected error on parent: %v", err)
	}
	if !parentExecuted {
		t.Error("parent Exec should have been called")
	}
}
