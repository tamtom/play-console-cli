package cmdtest_test

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestSearch_FindsCanonicalInitialAppWorkflowOffline(t *testing.T) {
	stdout, stderr, err := runCommand(t, "search", "initial", "app", "record")
	if err != nil {
		t.Fatalf("search failed: %v\nstderr: %s", err, stderr)
	}

	var response struct {
		Query   string `json:"query"`
		Count   int    `json:"count"`
		Results []struct {
			Command string   `json:"command"`
			Matched []string `json:"matched"`
		} `json:"results"`
	}
	if err := json.Unmarshal([]byte(stdout), &response); err != nil {
		t.Fatalf("decode search output: %v\nstdout: %s", err, stdout)
	}
	if response.Query != "initial app record" {
		t.Fatalf("query = %q, want %q", response.Query, "initial app record")
	}
	if response.Count == 0 || len(response.Results) == 0 {
		t.Fatal("expected at least one search result")
	}
	if response.Results[0].Command != "gplay bootstrap plan" {
		t.Fatalf("first command = %q, want %q", response.Results[0].Command, "gplay bootstrap plan")
	}
	if len(response.Results[0].Matched) == 0 {
		t.Fatal("expected search result to explain why it matched")
	}
}

func TestSearch_FindsFlagsAndExamples(t *testing.T) {
	stdout, stderr, err := runCommand(t, "search", "staged", "rollout", "fraction")
	if err != nil {
		t.Fatalf("search failed: %v\nstderr: %s", err, stderr)
	}
	if !strings.Contains(stdout, `"command":"gplay rollout`) {
		t.Fatalf("expected rollout command in search results, got: %s", stdout)
	}
}

func TestSearch_RequiresQuery(t *testing.T) {
	_, stderr, err := runCommand(t, "search")
	if err == nil {
		t.Fatal("expected missing query error")
	}
	if !strings.Contains(stderr, "search query is required") {
		t.Fatalf("unexpected stderr: %s", stderr)
	}
}
