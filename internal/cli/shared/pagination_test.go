package shared

import (
	"strings"
	"testing"
)

func TestPaginationGuard_StopsAtPageLimit(t *testing.T) {
	guard := NewPaginationGuard(2)
	if done, err := guard.Advance("page-2"); done || err != nil {
		t.Fatalf("first page = (done %t, err %v), want continuation", done, err)
	}
	done, err := guard.Advance("page-3")
	if done || err == nil {
		t.Fatalf("second page = (done %t, err %v), want page-limit error", done, err)
	}
	if !strings.Contains(err.Error(), "2-page limit") {
		t.Fatalf("error = %q, want 2-page limit", err)
	}
}

func TestPaginationGuard_AcceptsFinalEmptyTokenAtLimit(t *testing.T) {
	guard := NewPaginationGuard(2)
	if done, err := guard.Advance("page-2"); done || err != nil {
		t.Fatalf("first page = (done %t, err %v), want continuation", done, err)
	}
	if done, err := guard.Advance(""); !done || err != nil {
		t.Fatalf("final page = (done %t, err %v), want completion", done, err)
	}
}

func TestPaginationGuard_RejectsRepeatedToken(t *testing.T) {
	guard := NewPaginationGuard(0)
	if done, err := guard.Advance("same"); done || err != nil {
		t.Fatalf("first page = (done %t, err %v), want continuation", done, err)
	}
	if done, err := guard.Advance("same"); done || err == nil || !strings.Contains(err.Error(), "repeated page token") {
		t.Fatalf("second page = (done %t, err %v), want repeated-token error", done, err)
	}
}
