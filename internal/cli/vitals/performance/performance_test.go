package performance

import (
	"bytes"
	"context"
	"flag"
	"strings"
	"testing"
)

func TestPerformanceCommand_Structure(t *testing.T) {
	cmd := PerformanceCommand()

	if cmd.Name != "performance" {
		t.Errorf("expected command name 'performance', got %q", cmd.Name)
	}

	expected := map[string]bool{
		"startup":   false,
		"rendering": false,
		"battery":   false,
	}

	for _, sub := range cmd.Subcommands {
		if _, ok := expected[sub.Name]; !ok {
			t.Errorf("unexpected subcommand: %q", sub.Name)
		}
		expected[sub.Name] = true
	}

	for name, found := range expected {
		if !found {
			t.Errorf("missing subcommand: %q", name)
		}
	}
}

func TestStartupCommand_Name(t *testing.T) {
	cmd := StartupCommand()
	if cmd.Name != "startup" {
		t.Errorf("expected command name 'startup', got %q", cmd.Name)
	}
}

func TestRenderingCommand_Name(t *testing.T) {
	cmd := RenderingCommand()
	if cmd.Name != "rendering" {
		t.Errorf("expected command name 'rendering', got %q", cmd.Name)
	}
}

func TestBatteryCommand_Name(t *testing.T) {
	cmd := BatteryCommand()
	if cmd.Name != "battery" {
		t.Errorf("expected command name 'battery', got %q", cmd.Name)
	}
}

func TestStartupCommand_PackageRequired(t *testing.T) {
	cmd := StartupCommand()
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error when --package is missing")
	}
	if !strings.Contains(err.Error(), "--package is required") {
		t.Errorf("expected '--package is required' error, got: %v", err)
	}
}

func TestRenderingCommand_PackageRequired(t *testing.T) {
	cmd := RenderingCommand()
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error when --package is missing")
	}
	if !strings.Contains(err.Error(), "--package is required") {
		t.Errorf("expected '--package is required' error, got: %v", err)
	}
}

func TestBatteryCommand_PackageRequired(t *testing.T) {
	cmd := BatteryCommand()
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error when --package is missing")
	}
	if !strings.Contains(err.Error(), "--package is required") {
		t.Errorf("expected '--package is required' error, got: %v", err)
	}
}

func TestBatteryCommand_InvalidType(t *testing.T) {
	cmd := BatteryCommand()
	if err := cmd.FlagSet.Parse([]string{"--package", "com.example.app", "--type", "invalid"}); err != nil {
		t.Fatalf("flag parse error: %v", err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for invalid --type")
	}
	if !strings.Contains(err.Error(), "--type must be 'wakeup' or 'wakelock'") {
		t.Errorf("expected type validation error, got: %v", err)
	}
}

func TestBatteryCommand_ValidType(t *testing.T) {
	for _, typ := range []string{"wakeup", "wakelock"} {
		t.Run(typ, func(t *testing.T) {
			cmd := BatteryCommand()
			if err := cmd.FlagSet.Parse([]string{"--package", "com.example.app", "--type", typ}); err != nil {
				t.Fatalf("flag parse error: %v", err)
			}
			err := cmd.Exec(context.Background(), nil)
			if err != nil {
				t.Errorf("expected no error for valid --type %q, got: %v", typ, err)
			}
		})
	}
}

func TestStartupCommand_HelpOutput(t *testing.T) {
	cmd := StartupCommand()
	if cmd.ShortHelp == "" {
		t.Error("expected non-empty ShortHelp")
	}
	if cmd.ShortUsage == "" {
		t.Error("expected non-empty ShortUsage")
	}
	if !strings.Contains(cmd.ShortUsage, "--package") {
		t.Errorf("expected ShortUsage to mention --package, got: %s", cmd.ShortUsage)
	}
}

func TestPerformanceCommand_HelpOutput(t *testing.T) {
	cmd := PerformanceCommand()
	if cmd.ShortHelp == "" {
		t.Error("expected non-empty ShortHelp")
	}
	if !strings.Contains(cmd.ShortUsage, "gplay vitals performance") {
		t.Errorf("expected ShortUsage to contain 'gplay vitals performance', got: %s", cmd.ShortUsage)
	}
}

func TestPerformanceCommand_NoArgsReturnsHelp(t *testing.T) {
	cmd := PerformanceCommand()
	err := cmd.Exec(context.Background(), nil)
	if err != flag.ErrHelp {
		t.Errorf("expected flag.ErrHelp with no args, got: %v", err)
	}
}

func TestPerformanceCommand_UnknownSubcommand(t *testing.T) {
	cmd := PerformanceCommand()
	var stderr bytes.Buffer
	// Redirect stderr to capture output - can't easily do this without os.Pipe,
	// but we can at least verify the error type.
	_ = stderr
	err := cmd.Exec(context.Background(), []string{"unknown"})
	if err != flag.ErrHelp {
		t.Errorf("expected flag.ErrHelp for unknown subcommand, got: %v", err)
	}
}

func TestStartupCommand_Flags(t *testing.T) {
	cmd := StartupCommand()
	expectedFlags := []string{"package", "from", "to", "dimension", "output", "pretty", "paginate"}
	for _, name := range expectedFlags {
		if cmd.FlagSet.Lookup(name) == nil {
			t.Errorf("expected flag --%s to be defined", name)
		}
	}
}

func TestRenderingCommand_Flags(t *testing.T) {
	cmd := RenderingCommand()
	expectedFlags := []string{"package", "from", "to", "dimension", "output", "pretty", "paginate"}
	for _, name := range expectedFlags {
		if cmd.FlagSet.Lookup(name) == nil {
			t.Errorf("expected flag --%s to be defined", name)
		}
	}
}

func TestBatteryCommand_Flags(t *testing.T) {
	cmd := BatteryCommand()
	expectedFlags := []string{"package", "from", "to", "dimension", "type", "output", "pretty", "paginate"}
	for _, name := range expectedFlags {
		if cmd.FlagSet.Lookup(name) == nil {
			t.Errorf("expected flag --%s to be defined", name)
		}
	}
}

func TestStartupCommand_PrettyWithTableOutputInvalid(t *testing.T) {
	cmd := StartupCommand()
	if err := cmd.FlagSet.Parse([]string{"--package", "com.example.app", "--output", "table", "--pretty"}); err != nil {
		t.Fatalf("flag parse error: %v", err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for --pretty with table output")
	}
	if !strings.Contains(err.Error(), "--pretty is only valid with JSON output") {
		t.Errorf("expected pretty validation error, got: %v", err)
	}
}

func TestStartupCommand_StubExecution(t *testing.T) {
	cmd := StartupCommand()
	if err := cmd.FlagSet.Parse([]string{"--package", "com.example.app"}); err != nil {
		t.Fatalf("flag parse error: %v", err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err != nil {
		t.Errorf("expected no error for stub execution, got: %v", err)
	}
}

func TestRenderingCommand_StubExecution(t *testing.T) {
	cmd := RenderingCommand()
	if err := cmd.FlagSet.Parse([]string{"--package", "com.example.app"}); err != nil {
		t.Fatalf("flag parse error: %v", err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err != nil {
		t.Errorf("expected no error for stub execution, got: %v", err)
	}
}
