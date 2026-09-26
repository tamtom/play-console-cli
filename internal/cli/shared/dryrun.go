package shared

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"unicode"
)

// dryRunKey is the context key for the dry-run flag.
type dryRunKey struct{}

// ContextWithDryRun returns a context with the dry-run flag set.
func ContextWithDryRun(ctx context.Context, dryRun bool) context.Context {
	return context.WithValue(ctx, dryRunKey{}, dryRun)
}

// IsDryRun returns true if the context has dry-run enabled.
func IsDryRun(ctx context.Context) bool {
	v, ok := ctx.Value(dryRunKey{}).(bool)
	return ok && v
}

// writeMethods are HTTP methods that mutate state.
var writeMethods = map[string]bool{
	http.MethodPost:   true,
	http.MethodPut:    true,
	http.MethodPatch:  true,
	http.MethodDelete: true,
}

// DryRunTransport wraps an http.RoundTripper and intercepts write requests
// when dry-run mode is active. GET/HEAD requests pass through normally.
// Write requests (POST, PUT, PATCH, DELETE) are logged to stderr and return
// a synthetic 200 OK response without making any actual API call.
type DryRunTransport struct {
	Base   http.RoundTripper
	Writer io.Writer // output destination (typically os.Stderr)
}

// RoundTrip implements http.RoundTripper.
func (t *DryRunTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if !writeMethods[req.Method] {
		base := t.Base
		if base == nil {
			base = http.DefaultTransport
		}
		return base.RoundTrip(req)
	}

	w := t.Writer
	if w == nil {
		w = io.Discard
	}

	// Log the intercepted request.
	fmt.Fprintf(w, "[DRY RUN] %s %s\n", req.Method, dryRunURL(req)) // #nosec G705 -- writing to stderr, not a web response

	if req.Body != nil && req.Body != http.NoBody {
		// Parse and redact the full JSON body first. Only then cut the
		// redacted text, so that a cut can never show a secret.
		const readLimit, displayLimit = 1 << 20, 2048
		body, err := io.ReadAll(io.LimitReader(req.Body, readLimit+1))
		req.Body.Close()
		if err == nil && len(body) > 0 {
			display := "[omitted: non-JSON body]"
			if len(body) > readLimit {
				display = "[omitted: large body] ... (truncated)"
			} else {
				var value any
				if json.Unmarshal(body, &value) == nil {
					redactDryRunJSON(value)
					if safe, err := json.Marshal(value); err == nil {
						display = string(safe)
						if len(display) > displayLimit {
							display = strings.ToValidUTF8(display[:displayLimit], "") + fmt.Sprintf(" ... (truncated, %d bytes)", len(safe))
						}
					}
				}
			}
			fmt.Fprintf(w, "[DRY RUN] Body: %s\n", display)
		}
	}

	fmt.Fprintf(w, "[DRY RUN] No changes were made.\n")

	// Return a synthetic successful response.
	return &http.Response{
		StatusCode: http.StatusOK,
		Status:     "200 OK",
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header:     http.Header{"Content-Type": {"application/json"}},
		Body:       io.NopCloser(bytes.NewBufferString("{}")),
		Request:    req,
	}, nil
}

func dryRunURL(req *http.Request) string {
	return redactURL(*req.URL)
}

// redactURL removes user info, the fragment, sensitive query values and the
// path segment after "tokens" from a copy of u.
func redactURL(u url.URL) string {
	u.User = nil
	query := u.Query()
	for key := range query {
		if dryRunSensitiveName(key) {
			query.Set(key, "<redacted>")
		}
	}
	u.RawQuery = query.Encode()
	segments := strings.Split(u.Path, "/")
	for i := 1; i < len(segments); i++ {
		if strings.EqualFold(segments[i-1], "tokens") {
			segments[i] = "<redacted>"
		}
	}
	u.Path = strings.Join(segments, "/")
	u.RawPath = ""
	u.Fragment = ""
	return u.String()
}

func redactDryRunJSON(value any) {
	switch v := value.(type) {
	case map[string]any:
		for key, child := range v {
			if dryRunSensitiveName(key) {
				v[key] = "<redacted>"
			} else {
				redactDryRunJSON(child)
			}
		}
	case []any:
		for _, child := range v {
			redactDryRunJSON(child)
		}
	}
}

var (
	// sensitiveNameWords are words that mark a field or query name as secret.
	sensitiveNameWords = map[string]bool{
		"token": true, "tokens": true, "secret": true, "secrets": true, "password": true, "passwords": true,
		"passphrase": true, "credential": true, "credentials": true, "key": true, "keys": true,
		"authorization": true, "webhook": true, "payload": true,
	}
	// sensitiveNameSuffixes also match names that have no word separators,
	// for example IDTOKEN or apikey.
	sensitiveNameSuffixes = []string{"token", "secret", "password", "credential", "credentials", "key", "payload"}
)

// dryRunSensitiveName reports whether a JSON field or a query parameter can
// hold a secret. It compares words, not substrings, so that names such as
// metadata or keywords stay visible. A name whose last word is "data" is
// sensitive, because RTDN messages carry the notification in "data".
func dryRunSensitiveName(key string) bool {
	lower := strings.ToLower(key)
	for _, suffix := range sensitiveNameSuffixes {
		if strings.HasSuffix(lower, suffix) {
			return true
		}
	}
	words := nameWords(key)
	for _, word := range words {
		if sensitiveNameWords[word] {
			return true
		}
	}
	return len(words) > 0 && words[len(words)-1] == "data"
}

// nameWords splits a camelCase, snake_case or kebab-case name into
// lower-case words.
func nameWords(name string) []string {
	var words []string
	var current []rune
	flush := func() {
		if len(current) > 0 {
			words = append(words, strings.ToLower(string(current)))
			current = current[:0]
		}
	}
	runes := []rune(name)
	for i, r := range runes {
		switch {
		case r == '_' || r == '-' || r == '.' || r == ' ':
			flush()
		case unicode.IsUpper(r) && i > 0 && (unicode.IsLower(runes[i-1]) || (i+1 < len(runes) && unicode.IsLower(runes[i+1]))):
			flush()
			current = append(current, r)
		default:
			current = append(current, r)
		}
	}
	flush()
	return words
}
