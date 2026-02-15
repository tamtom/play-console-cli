package release

import (
	"context"
	"strings"
	"testing"
)

func TestReleaseCommand_Name(t *testing.T) {
	cmd := ReleaseCommand()
	if cmd.Name != "release" {
		t.Errorf("expected name %q, got %q", "release", cmd.Name)
	}
}

func TestReleaseCommand_ShortHelp(t *testing.T) {
	cmd := ReleaseCommand()
	if cmd.ShortHelp == "" {
		t.Error("expected non-empty ShortHelp")
	}
}

func TestReleaseCommand_LongHelp(t *testing.T) {
	cmd := ReleaseCommand()
	if cmd.LongHelp == "" {
		t.Error("expected non-empty LongHelp")
	}
}

func TestReleaseCommand_UsageFunc(t *testing.T) {
	cmd := ReleaseCommand()
	if cmd.UsageFunc == nil {
		t.Error("expected UsageFunc to be set")
	}
}

func TestReleaseCommand_NoSubcommands(t *testing.T) {
	cmd := ReleaseCommand()
	if len(cmd.Subcommands) != 0 {
		t.Errorf("expected no subcommands, got %d", len(cmd.Subcommands))
	}
}

func TestReleaseCommand_MissingBundleAndApk(t *testing.T) {
	cmd := ReleaseCommand()
	if err := cmd.FlagSet.Parse([]string{"--package", "com.example.app"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error when neither --bundle nor --apk is provided")
	}
	if !strings.Contains(err.Error(), "--bundle") && !strings.Contains(err.Error(), "--apk") {
		t.Errorf("error should mention --bundle or --apk, got: %s", err.Error())
	}
}

func TestReleaseCommand_BothBundleAndApk(t *testing.T) {
	cmd := ReleaseCommand()
	if err := cmd.FlagSet.Parse([]string{"--bundle", "app.aab", "--apk", "app.apk"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error when both --bundle and --apk are provided")
	}
	if !strings.Contains(err.Error(), "not both") {
		t.Errorf("error should mention 'not both', got: %s", err.Error())
	}
}

func TestReleaseCommand_RolloutTooLow(t *testing.T) {
	cmd := ReleaseCommand()
	if err := cmd.FlagSet.Parse([]string{"--bundle", "app.aab", "--rollout", "-0.5"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for rollout < 0")
	}
	if !strings.Contains(err.Error(), "--rollout") {
		t.Errorf("error should mention --rollout, got: %s", err.Error())
	}
}

func TestReleaseCommand_RolloutTooHigh(t *testing.T) {
	cmd := ReleaseCommand()
	if err := cmd.FlagSet.Parse([]string{"--bundle", "app.aab", "--rollout", "1.5"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for rollout > 1")
	}
	if !strings.Contains(err.Error(), "--rollout") {
		t.Errorf("error should mention --rollout, got: %s", err.Error())
	}
}

func TestReleaseCommand_InvalidOutputFormat(t *testing.T) {
	cmd := ReleaseCommand()
	if err := cmd.FlagSet.Parse([]string{"--output", "xml"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for invalid output format")
	}
}

func TestReleaseCommand_PrettyWithTable(t *testing.T) {
	cmd := ReleaseCommand()
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

func TestReleaseCommand_WhitespaceBundle(t *testing.T) {
	cmd := ReleaseCommand()
	if err := cmd.FlagSet.Parse([]string{"--bundle", "   "}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for whitespace-only --bundle")
	}
	// Should fall through to missing bundle/apk validation
	if !strings.Contains(err.Error(), "--bundle") && !strings.Contains(err.Error(), "--apk") {
		t.Errorf("error should mention --bundle or --apk, got: %s", err.Error())
	}
}

func TestReleaseCommand_WhitespaceApk(t *testing.T) {
	cmd := ReleaseCommand()
	if err := cmd.FlagSet.Parse([]string{"--apk", "   "}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for whitespace-only --apk")
	}
	if !strings.Contains(err.Error(), "--bundle") && !strings.Contains(err.Error(), "--apk") {
		t.Errorf("error should mention --bundle or --apk, got: %s", err.Error())
	}
}

func TestReleaseCommand_RolloutBoundary_Zero(t *testing.T) {
	cmd := ReleaseCommand()
	if err := cmd.FlagSet.Parse([]string{"--bundle", "app.aab", "--rollout", "0"}); err != nil {
		t.Fatal(err)
	}
	// rollout=0 is valid (0.0-1.0 inclusive); should proceed past validation
	// Will fail on NewService (no credentials), not on rollout validation
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error (no credentials), but should not be rollout error")
	}
	if strings.Contains(err.Error(), "--rollout") {
		t.Errorf("rollout=0 should be valid, got: %s", err.Error())
	}
}

func TestReleaseCommand_RolloutBoundary_One(t *testing.T) {
	cmd := ReleaseCommand()
	if err := cmd.FlagSet.Parse([]string{"--bundle", "app.aab", "--rollout", "1"}); err != nil {
		t.Fatal(err)
	}
	// rollout=1.0 is the default and valid; should proceed past validation
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error (no credentials), but should not be rollout error")
	}
	if strings.Contains(err.Error(), "--rollout") {
		t.Errorf("rollout=1.0 should be valid, got: %s", err.Error())
	}
}
