package iap

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/tamtom/play-console-cli/internal/playclient"
)

func TestIAPCommand_Name(t *testing.T) {
	cmd := IAPCommand()
	if cmd.Name != "iap" {
		t.Errorf("expected name %q, got %q", "iap", cmd.Name)
	}
}

func TestIAPCommand_ShortHelp(t *testing.T) {
	cmd := IAPCommand()
	if cmd.ShortHelp == "" {
		t.Error("expected non-empty ShortHelp")
	}
}

func TestIAPCommand_LongHelp(t *testing.T) {
	cmd := IAPCommand()
	if cmd.LongHelp == "" {
		t.Error("expected non-empty LongHelp")
	}
}

func TestIAPCommand_UsageFunc(t *testing.T) {
	cmd := IAPCommand()
	if cmd.UsageFunc == nil {
		t.Error("expected UsageFunc to be set")
	}
}

func TestIAPCommand_HasSubcommands(t *testing.T) {
	cmd := IAPCommand()
	if len(cmd.Subcommands) == 0 {
		t.Error("expected subcommands")
	}
}

func TestIAPCommand_SubcommandNames(t *testing.T) {
	cmd := IAPCommand()
	expected := map[string]bool{
		"list":         false,
		"get":          false,
		"create":       false,
		"update":       false,
		"patch":        false,
		"delete":       false,
		"batch-get":    false,
		"batch-update": false,
		"batch-delete": false,
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

func TestIAPCommand_SubcommandsHaveUsageFunc(t *testing.T) {
	cmd := IAPCommand()
	for _, sub := range cmd.Subcommands {
		if sub.UsageFunc == nil {
			t.Errorf("subcommand %q missing UsageFunc", sub.Name)
		}
	}
}

func TestIAPCommand_NoArgs_ReturnsHelp(t *testing.T) {
	cmd := IAPCommand()
	err := cmd.Exec(context.Background(), nil)
	if !errors.Is(err, flag.ErrHelp) {
		t.Errorf("expected flag.ErrHelp, got %v", err)
	}
}

// --- iap get ---

func TestIAPGetCommand_Name(t *testing.T) {
	cmd := GetCommand()
	if cmd.Name != "get" {
		t.Errorf("expected name %q, got %q", "get", cmd.Name)
	}
}

func TestIAPGetCommand_MissingSku(t *testing.T) {
	cmd := GetCommand()
	if err := cmd.FlagSet.Parse([]string{}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for missing --sku")
	}
	if !strings.Contains(err.Error(), "--sku") {
		t.Errorf("error should mention --sku, got: %s", err.Error())
	}
}

func TestIAPGetCommand_WhitespaceSku(t *testing.T) {
	cmd := GetCommand()
	if err := cmd.FlagSet.Parse([]string{"--sku", "   "}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for whitespace-only --sku")
	}
	if !strings.Contains(err.Error(), "--sku") {
		t.Errorf("error should mention --sku, got: %s", err.Error())
	}
}

func TestIAPGetCommand_InvalidOutputFormat(t *testing.T) {
	cmd := GetCommand()
	if err := cmd.FlagSet.Parse([]string{"--output", "xml"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for invalid output format")
	}
}

func TestIAPGetCommand_PrettyWithTable(t *testing.T) {
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

// --- iap create ---

func TestIAPCreateCommand_Name(t *testing.T) {
	cmd := CreateCommand()
	if cmd.Name != "create" {
		t.Errorf("expected name %q, got %q", "create", cmd.Name)
	}
}

func TestIAPCreateCommand_MissingJson(t *testing.T) {
	cmd := CreateCommand()
	if err := cmd.FlagSet.Parse([]string{}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for missing --json")
	}
	if !strings.Contains(err.Error(), "--json") {
		t.Errorf("error should mention --json, got: %s", err.Error())
	}
}

func TestIAPCreateCommand_WhitespaceJson(t *testing.T) {
	cmd := CreateCommand()
	if err := cmd.FlagSet.Parse([]string{"--json", "   "}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for whitespace-only --json")
	}
	if !strings.Contains(err.Error(), "--json") {
		t.Errorf("error should mention --json, got: %s", err.Error())
	}
}

// --- iap update ---

func TestIAPUpdateCommand_Name(t *testing.T) {
	cmd := UpdateCommand()
	if cmd.Name != "update" {
		t.Errorf("expected name %q, got %q", "update", cmd.Name)
	}
}

func TestIAPUpdateCommand_MissingSku(t *testing.T) {
	cmd := UpdateCommand()
	if err := cmd.FlagSet.Parse([]string{"--json", `{"sku":"test"}`}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for missing --sku")
	}
	if !strings.Contains(err.Error(), "--sku") {
		t.Errorf("error should mention --sku, got: %s", err.Error())
	}
}

func TestIAPUpdateCommand_MissingJson(t *testing.T) {
	cmd := UpdateCommand()
	if err := cmd.FlagSet.Parse([]string{"--sku", "test_sku"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for missing --json")
	}
	if !strings.Contains(err.Error(), "--json") {
		t.Errorf("error should mention --json, got: %s", err.Error())
	}
}

// --- iap patch ---

func TestIAPPatchCommand_Name(t *testing.T) {
	cmd := PatchCommand()
	if cmd.Name != "patch" {
		t.Errorf("expected name %q, got %q", "patch", cmd.Name)
	}
}

func TestIAPPatchCommand_CallsAPI(t *testing.T) {
	var gotMethod, gotPath, gotBody string
	installMockIAPPlayService(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		body, _ := io.ReadAll(r.Body)
		gotBody = string(body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"packageName":"com.example.app","sku":"coins_100","status":"active"}`)
	})

	cmd := PatchCommand()
	if err := cmd.FlagSet.Parse([]string{
		"--package", "com.example.app",
		"--sku", "coins_100",
		"--json", `{"status":"active"}`,
		"--auto-convert-prices",
	}); err != nil {
		t.Fatalf("parse flags: %v", err)
	}
	stdout, err := captureIAPStdout(func() error {
		return cmd.Exec(context.Background(), nil)
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if gotMethod != http.MethodPatch {
		t.Fatalf("method = %s, want PATCH", gotMethod)
	}
	if gotPath != "/androidpublisher/v3/applications/com.example.app/inappproducts/coins_100" {
		t.Fatalf("unexpected path: %s", gotPath)
	}
	if !strings.Contains(gotBody, `"sku":"coins_100"`) {
		t.Fatalf("expected SKU to be set in body, got %s", gotBody)
	}
	if !strings.Contains(stdout, "coins_100") {
		t.Fatalf("expected patched product in output, got %s", stdout)
	}
}

// --- iap delete ---

func TestIAPDeleteCommand_Name(t *testing.T) {
	cmd := DeleteCommand()
	if cmd.Name != "delete" {
		t.Errorf("expected name %q, got %q", "delete", cmd.Name)
	}
}

func TestIAPDeleteCommand_MissingSku(t *testing.T) {
	cmd := DeleteCommand()
	if err := cmd.FlagSet.Parse([]string{"--confirm"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for missing --sku")
	}
	if !strings.Contains(err.Error(), "--sku") {
		t.Errorf("error should mention --sku, got: %s", err.Error())
	}
}

func TestIAPDeleteCommand_MissingConfirm(t *testing.T) {
	cmd := DeleteCommand()
	if err := cmd.FlagSet.Parse([]string{"--sku", "test_sku"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for missing --confirm")
	}
	if !strings.Contains(err.Error(), "--confirm") {
		t.Errorf("error should mention --confirm, got: %s", err.Error())
	}
}

// --- iap batch-get ---

func TestIAPBatchGetCommand_Name(t *testing.T) {
	cmd := BatchGetCommand()
	if cmd.Name != "batch-get" {
		t.Errorf("expected name %q, got %q", "batch-get", cmd.Name)
	}
}

func TestIAPBatchGetCommand_MissingSkus(t *testing.T) {
	cmd := BatchGetCommand()
	if err := cmd.FlagSet.Parse([]string{}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for missing --skus")
	}
	if !strings.Contains(err.Error(), "--skus") {
		t.Errorf("error should mention --skus, got: %s", err.Error())
	}
}

func installMockIAPPlayService(t *testing.T, handler http.HandlerFunc) {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	original := newPlayService
	newPlayService = func(ctx context.Context) (*playclient.Service, error) {
		return playclient.NewServiceWithClient(ctx, server.Client(), server.URL+"/")
	}
	t.Cleanup(func() {
		newPlayService = original
	})
}

func captureIAPStdout(fn func() error) (string, error) {
	origStdout := os.Stdout
	rOut, wOut, err := os.Pipe()
	if err != nil {
		return "", err
	}

	os.Stdout = wOut

	var buf bytes.Buffer
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_, _ = io.Copy(&buf, rOut)
	}()

	runErr := fn()

	_ = wOut.Close()
	os.Stdout = origStdout
	wg.Wait()
	_ = rOut.Close()

	return buf.String(), runErr
}

func TestIAPBatchGetCommand_WhitespaceSkus(t *testing.T) {
	cmd := BatchGetCommand()
	if err := cmd.FlagSet.Parse([]string{"--skus", "   "}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for whitespace-only --skus")
	}
	if !strings.Contains(err.Error(), "--skus") {
		t.Errorf("error should mention --skus, got: %s", err.Error())
	}
}

// --- iap batch-update ---

func TestIAPBatchUpdateCommand_Name(t *testing.T) {
	cmd := BatchUpdateCommand()
	if cmd.Name != "batch-update" {
		t.Errorf("expected name %q, got %q", "batch-update", cmd.Name)
	}
}

func TestIAPBatchUpdateCommand_MissingJson(t *testing.T) {
	cmd := BatchUpdateCommand()
	if err := cmd.FlagSet.Parse([]string{}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for missing --json")
	}
	if !strings.Contains(err.Error(), "--json") {
		t.Errorf("error should mention --json, got: %s", err.Error())
	}
}

// --- iap batch-delete ---

func TestIAPBatchDeleteCommand_Name(t *testing.T) {
	cmd := BatchDeleteCommand()
	if cmd.Name != "batch-delete" {
		t.Errorf("expected name %q, got %q", "batch-delete", cmd.Name)
	}
}

func TestIAPBatchDeleteCommand_MissingSkus(t *testing.T) {
	cmd := BatchDeleteCommand()
	if err := cmd.FlagSet.Parse([]string{"--confirm"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for missing --skus")
	}
	if !strings.Contains(err.Error(), "--skus") {
		t.Errorf("error should mention --skus, got: %s", err.Error())
	}
}

func TestIAPBatchDeleteCommand_MissingConfirm(t *testing.T) {
	cmd := BatchDeleteCommand()
	if err := cmd.FlagSet.Parse([]string{"--skus", "sku1,sku2"}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for missing --confirm")
	}
	if !strings.Contains(err.Error(), "--confirm") {
		t.Errorf("error should mention --confirm, got: %s", err.Error())
	}
}
