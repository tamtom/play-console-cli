package livesmoke

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLedger_AppendAndLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ledger.jsonl")
	l := NewLedger(path)

	first := Resource{Kind: "edit", ID: "edit-1", Package: AllowedMutationPackage, RunID: "run-1"}
	second := Resource{Kind: "product", ID: "ls-run-1-product", Package: AllowedMutationPackage, RunID: "run-1"}

	if err := l.Append(first); err != nil {
		t.Fatalf("append first: %v", err)
	}
	if err := l.Append(second); err != nil {
		t.Fatalf("append second: %v", err)
	}

	got, err := l.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("want 2 resources, got %d", len(got))
	}
	if got[0].ID != "edit-1" || got[1].ID != "ls-run-1-product" {
		t.Fatalf("unexpected resources: %+v", got)
	}
	if got[0].CreatedAt == "" {
		t.Fatal("append must stamp CreatedAt")
	}
}

func TestLedger_LoadMissingFileReturnsEmpty(t *testing.T) {
	l := NewLedger(filepath.Join(t.TempDir(), "missing.jsonl"))
	got, err := l.Load()
	if err != nil {
		t.Fatalf("load of a missing file must not fail: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("want empty, got %+v", got)
	}
}

func TestLedger_AppendRejectsForeignPackage(t *testing.T) {
	l := NewLedger(filepath.Join(t.TempDir(), "ledger.jsonl"))
	err := l.Append(Resource{Kind: "edit", ID: "e", Package: "com.example.other", RunID: "r"})
	if err == nil {
		t.Fatal("append must reject a resource for a non-fixture package")
	}
}

func TestLedger_SkipsCorruptLines(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ledger.jsonl")
	content := "{\"kind\":\"edit\",\"id\":\"e1\",\"package\":\"" + AllowedMutationPackage + "\",\"run_id\":\"r\",\"created_at\":\"x\"}\nnot-json\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := NewLedger(path).Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(got) != 1 || got[0].ID != "e1" {
		t.Fatalf("want the one valid resource, got %+v", got)
	}
}
