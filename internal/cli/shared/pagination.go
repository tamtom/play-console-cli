package shared

import (
	"fmt"
	"strings"
)

const DefaultMaxPaginationPages = 1000

// PaginationGuard prevents auto-pagination from looping forever when an API
// repeats a token or continues beyond a defensible upper bound.
type PaginationGuard struct {
	seen     map[string]struct{}
	pages    int
	maxPages int
}

// NewPaginationGuard creates a guard. Values below one use the default limit.
func NewPaginationGuard(maxPages int) *PaginationGuard {
	if maxPages < 1 {
		maxPages = DefaultMaxPaginationPages
	}
	return &PaginationGuard{
		seen:     make(map[string]struct{}),
		maxPages: maxPages,
	}
}

// Advance records one completed page and its next token. done is true when the
// token is empty. A non-empty token may be used for the next request only when
// err is nil.
func (g *PaginationGuard) Advance(nextPageToken string) (done bool, err error) {
	g.pages++
	token := strings.TrimSpace(nextPageToken)
	if token == "" {
		return true, nil
	}
	if _, exists := g.seen[token]; exists {
		return false, fmt.Errorf("pagination stopped: repeated page token after %d pages", g.pages)
	}
	if g.pages >= g.maxPages {
		return false, fmt.Errorf("pagination stopped after reaching the %d-page limit", g.maxPages)
	}
	g.seen[token] = struct{}{}
	return false, nil
}
