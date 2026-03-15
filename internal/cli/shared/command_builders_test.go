package shared

import (
	"context"
	"flag"
	"testing"
)

func TestBuildPaginatedListCommand_RegistersAllFlags(t *testing.T) {
	cmd := BuildPaginatedListCommand(PaginatedListCommandConfig{
		Name:       "list",
		ShortUsage: "test list",
		ShortHelp:  "List items",
		Exec: func(ctx context.Context, pageSize int, pageToken string, paginate bool, output *OutputFlags) error {
			return nil
		},
	})

	expectedFlags := []string{"page-size", "paginate", "next", "output", "pretty"}
	for _, name := range expectedFlags {
		if cmd.FlagSet.Lookup(name) == nil {
			t.Errorf("expected flag --%s to be registered", name)
		}
	}
}

func TestBuildPaginatedListCommand_PassesCorrectValues(t *testing.T) {
	var gotPageSize int
	var gotPageToken string
	var gotPaginate bool
	var gotOutput *OutputFlags

	cmd := BuildPaginatedListCommand(PaginatedListCommandConfig{
		Name:       "list",
		ShortUsage: "test list",
		ShortHelp:  "List items",
		Exec: func(ctx context.Context, pageSize int, pageToken string, paginate bool, output *OutputFlags) error {
			gotPageSize = pageSize
			gotPageToken = pageToken
			gotPaginate = paginate
			gotOutput = output
			return nil
		},
	})

	if err := cmd.FlagSet.Parse([]string{"--page-size", "50", "--next", "abc123", "--paginate", "--output", "table"}); err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if err := cmd.Exec(context.Background(), nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if gotPageSize != 50 {
		t.Errorf("expected pageSize=50, got %d", gotPageSize)
	}
	if gotPageToken != "abc123" {
		t.Errorf("expected pageToken=abc123, got %q", gotPageToken)
	}
	if !gotPaginate {
		t.Error("expected paginate=true")
	}
	if gotOutput == nil {
		t.Fatal("expected output to be non-nil")
	}
	if gotOutput.Format() != "table" {
		t.Errorf("expected output format=table, got %q", gotOutput.Format())
	}
}

func TestBuildPaginatedListCommand_ExtraFlags(t *testing.T) {
	executed := false
	var customFlag *string

	cmd := BuildPaginatedListCommand(PaginatedListCommandConfig{
		Name:       "list",
		ShortUsage: "test list",
		ShortHelp:  "List items",
		ExtraFlags: func(fs *flag.FlagSet) {
			customFlag = fs.String("custom", "", "A custom flag")
		},
		Exec: func(ctx context.Context, pageSize int, pageToken string, paginate bool, output *OutputFlags) error {
			executed = true
			return nil
		},
	})

	if cmd.FlagSet.Lookup("custom") == nil {
		t.Fatal("expected --custom flag to be registered")
	}

	if err := cmd.FlagSet.Parse([]string{"--custom", "hello"}); err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if err := cmd.Exec(context.Background(), nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !executed {
		t.Error("Exec should have been called")
	}
	if customFlag == nil || *customFlag != "hello" {
		t.Errorf("expected custom=hello, got %q", *customFlag)
	}
}

func TestBuildConfirmDeleteCommand_RejectsWithoutConfirm(t *testing.T) {
	executed := false
	cmd := BuildConfirmDeleteCommand(ConfirmDeleteCommandConfig{
		Name:       "delete",
		ShortUsage: "test delete",
		ShortHelp:  "Delete something",
		Exec: func(ctx context.Context) error {
			executed = true
			return nil
		},
	})

	// Don't pass --confirm
	if err := cmd.FlagSet.Parse([]string{}); err != nil {
		t.Fatalf("parse error: %v", err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error when --confirm is not set")
	}
	if err != flag.ErrHelp {
		t.Errorf("expected flag.ErrHelp, got: %v", err)
	}
	if executed {
		t.Error("Exec should NOT have been called without --confirm")
	}
}

func TestBuildConfirmDeleteCommand_RunsWithConfirm(t *testing.T) {
	executed := false
	cmd := BuildConfirmDeleteCommand(ConfirmDeleteCommandConfig{
		Name:       "delete",
		ShortUsage: "test delete",
		ShortHelp:  "Delete something",
		Exec: func(ctx context.Context) error {
			executed = true
			return nil
		},
	})

	if err := cmd.FlagSet.Parse([]string{"--confirm"}); err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if err := cmd.Exec(context.Background(), nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !executed {
		t.Error("Exec should have been called with --confirm")
	}
}

func TestBuildConfirmDeleteCommand_ExtraFlags(t *testing.T) {
	cmd := BuildConfirmDeleteCommand(ConfirmDeleteCommandConfig{
		Name:       "delete",
		ShortUsage: "test delete",
		ShortHelp:  "Delete something",
		ExtraFlags: func(fs *flag.FlagSet) {
			fs.String("id", "", "Item ID")
		},
		Exec: func(ctx context.Context) error {
			return nil
		},
	})

	if cmd.FlagSet.Lookup("id") == nil {
		t.Fatal("expected --id flag to be registered")
	}
	if cmd.FlagSet.Lookup("confirm") == nil {
		t.Fatal("expected --confirm flag to be registered")
	}
}
