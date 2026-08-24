package experiments

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/tamtom/play-console-cli/internal/cli/shared"
	"github.com/tamtom/play-console-cli/internal/cli/sync"
)

func TestSupportReportsMissingOfficialLifecycleWithoutPrivateFallback(t *testing.T) {
	command := SupportCommand()
	if err := command.Parse(nil); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	ctx := shared.ContextWithIO(context.Background(), &stdout, &bytes.Buffer{})
	if err := command.Run(ctx); err != nil {
		t.Fatal(err)
	}
	output := stdout.String()
	for _, want := range []string{
		`"officialLifecycleApi":false`, `"officialResultsApi":false`,
		`"officialApplyWinnerApi":true`, `"privateInterfacesUsed":false`,
		`"discoveryMethodsFound":[]`,
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("support output missing %q: %s", want, output)
		}
	}
}

func TestApplyWinnerRequiresExactConfirmationBeforeSync(t *testing.T) {
	original := runWinnerSync
	runWinnerSync = func(context.Context, string, string, string, string) (*sync.RunResult, error) {
		t.Fatal("mismatched confirmation reached official sync")
		return nil, nil
	}
	t.Cleanup(func() { runWinnerSync = original })
	command := ApplyWinnerCommand()
	if err := command.Parse([]string{
		"--package", "dev.example.app", "--edit", "edit-1", "--winner", "variant-b",
		"--confirm-winner", "variant-a", "--dir", "/tmp/winner",
	}); err != nil {
		t.Fatal(err)
	}
	err := command.Run(context.Background())
	if err == nil || !strings.Contains(err.Error(), "must exactly match") {
		t.Fatalf("error = %v", err)
	}
}

func TestApplyWinnerDelegatesToOfficialResumableSync(t *testing.T) {
	original := runWinnerSync
	called := false
	runWinnerSync = func(_ context.Context, pkg, edit, dir, stateDir string) (*sync.RunResult, error) {
		called = true
		if pkg != "dev.example.app" || edit != "edit-1" || dir != "/tmp/winner" || stateDir != "/tmp/state" {
			t.Fatalf("sync args = %q %q %q %q", pkg, edit, dir, stateDir)
		}
		return &sync.RunResult{PlanFile: "plan.json", ReceiptFile: "receipt.json"}, nil
	}
	t.Cleanup(func() { runWinnerSync = original })
	command := ApplyWinnerCommand()
	if err := command.Parse([]string{
		"--package", "dev.example.app", "--edit", "edit-1", "--winner", "variant-b",
		"--confirm-winner", "variant-b", "--dir", "/tmp/winner", "--state-dir", "/tmp/state",
	}); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	ctx := shared.ContextWithIO(context.Background(), &stdout, &bytes.Buffer{})
	if err := command.Run(ctx); err != nil {
		t.Fatal(err)
	}
	if !called || !strings.Contains(stdout.String(), `"selectionSource":"manual"`) || !strings.Contains(stdout.String(), `"provider":"official-api"`) {
		t.Fatalf("output = %s", stdout.String())
	}
}
