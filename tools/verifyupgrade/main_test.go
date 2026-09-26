package main

import (
	"net/http"
	"testing"
)

type recordingTransport struct{ requests []*http.Request }

func (t *recordingTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	t.requests = append(t.requests, r)
	return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody, Request: r}, nil
}

func TestGitHubAuthSendsTokenOnlyToTheAPIHost(t *testing.T) {
	base := &recordingTransport{}
	transport := githubAuth{base: base, token: "test-token"}
	for _, url := range []string{
		"https://api.github.com/repos/tamtom/play-console-cli/releases/latest",
		"https://github.com/tamtom/play-console-cli/releases/download/v0.10.0/gplay-linux-amd64",
		"http://api.github.com/insecure",
	} {
		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := transport.RoundTrip(req); err != nil {
			t.Fatal(err)
		}
		if req.Header.Get("Authorization") != "" {
			t.Fatalf("the caller's request was changed: %s", url)
		}
	}
	want := []string{"Bearer test-token", "", ""}
	for i, r := range base.requests {
		if got := r.Header.Get("Authorization"); got != want[i] {
			t.Errorf("%s: Authorization = %q, want %q", r.URL, got, want[i])
		}
	}
}
