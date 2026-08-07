// Package webclient talks to Play Console's internal web APIs using a stored
// browser session (cookies). Requests carry SAPISIDHASH auth derived from the
// SAPISID cookie, exactly as the console's own frontend does. None of this is
// covered by the official Android Publisher API.
package webclient

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/tamtom/play-console-cli/internal/websession"
)

// DefaultBaseURL is the Play Console Boq frontend origin.
const DefaultBaseURL = "https://play.google.com"

// authOrigin is the origin used as the third input of the SAPISIDHASH.
const authOrigin = "https://play.google.com"

// playOrigin is the session cookie key holding Play Console cookies.
const playOrigin = "https://play.google.com/"

// ErrAuth marks authentication-class failures: the session is expired or
// invalid and the user must run "gplay web auth login" again.
var ErrAuth = errors.New("web session invalid or expired")

// errAccountsRedirect is returned by CheckRedirect when the console bounces
// to accounts.google.com, which means the session cookies no longer
// authenticate.
var errAccountsRedirect = errors.New("redirected to accounts.google.com")

// Client is a Play Console web-session RPC client.
type Client struct {
	sess         *websession.Session
	http         *http.Client
	baseURL      string
	appsBaseURL  string
	appsAPIKeyMu sync.Mutex
	appsAPIKey   string
	now          func() time.Time // overridable in tests
}

// New returns a Client for the given session talking to the real Play
// Console.
func New(sess *websession.Session) *Client {
	return NewWithClient(sess, nil, DefaultBaseURL)
}

// NewWithClient is the test seam: it allows injecting an http.Client and a
// different base URL. When httpClient is nil a default one is built with a
// cookie jar seeded from the session and redirect detection for
// accounts.google.com.
func NewWithClient(sess *websession.Session, httpClient *http.Client, baseURL string) *Client {
	jar, _ := cookiejar.New(nil)
	seedCookies(jar, sess)

	if httpClient == nil {
		httpClient = &http.Client{Jar: jar}
	} else if httpClient.Jar == nil {
		httpClient.Jar = jar
	}
	if httpClient.CheckRedirect == nil {
		httpClient.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			if strings.Contains(req.URL.Host, "accounts.google.com") {
				return errAccountsRedirect
			}
			return nil
		}
	}

	// When pointed at a test server, route the apps API there too so mocks
	// cover both hosts.
	trimmed := strings.TrimRight(baseURL, "/")
	appsBase := DefaultAppsBaseURL
	if trimmed != DefaultBaseURL {
		appsBase = trimmed
	}

	return &Client{
		sess:        sess,
		http:        httpClient,
		baseURL:     trimmed,
		appsBaseURL: appsBase,
		now:         time.Now,
	}
}

// seedCookies loads all session cookies into the jar under their origins.
func seedCookies(jar *cookiejar.Jar, sess *websession.Session) {
	if sess == nil {
		return
	}
	for origin, cookies := range sess.Cookies {
		u, err := url.Parse(origin)
		if err != nil {
			continue
		}
		var hc []*http.Cookie
		for _, c := range cookies {
			ck := &http.Cookie{Name: c.Name, Value: c.Value, Path: "/"}
			if !c.Expires.IsZero() {
				ck.Expires = c.Expires
			}
			hc = append(hc, ck)
		}
		jar.SetCookies(u, hc)
	}
}

// cookieValue returns a session cookie value from the play.google.com
// origin (accepting the key with or without trailing slash).
func (c *Client) cookieValue(name string) string {
	if c.sess == nil {
		return ""
	}
	for _, origin := range []string{playOrigin, strings.TrimRight(playOrigin, "/")} {
		for _, ck := range c.sess.Cookies[origin] {
			if ck.Name == name {
				return ck.Value
			}
		}
	}
	return ""
}

// cookieHeader renders the stored Play Console cookies as a Cookie header.
// The jar keys them to play.google.com, so it will not serve these
// .google.com account cookies to other Google API hosts.
func (c *Client) cookieHeader() string {
	if c.sess == nil {
		return ""
	}
	for _, origin := range []string{playOrigin, strings.TrimRight(playOrigin, "/")} {
		cookies := c.sess.Cookies[origin]
		if len(cookies) == 0 {
			continue
		}
		parts := make([]string, 0, len(cookies))
		for _, ck := range cookies {
			parts = append(parts, ck.Name+"="+ck.Value)
		}
		return strings.Join(parts, "; ")
	}
	return ""
}

// sapisidHash computes "<ts>_<sha1(ts value origin)>" for one cookie.
func sapisidHash(kind, value string, ts int64) string {
	sum := sha1.Sum([]byte(fmt.Sprintf("%d %s %s", ts, value, authOrigin)))
	return fmt.Sprintf("%s %d_%s", kind, ts, hex.EncodeToString(sum[:]))
}

// authHeader builds the Authorization header with SAPISIDHASH plus the
// 1P/3P variants when those cookies exist.
func (c *Client) authHeader() string {
	ts := c.now().Unix()
	var parts []string
	if v := c.cookieValue("SAPISID"); v != "" {
		parts = append(parts, sapisidHash("SAPISIDHASH", v, ts))
	}
	if v := c.cookieValue("__Secure-1PAPISID"); v != "" {
		parts = append(parts, sapisidHash("SAPISID1PHASH", v, ts))
	}
	if v := c.cookieValue("__Secure-3PAPISID"); v != "" {
		parts = append(parts, sapisidHash("SAPISID3PHASH", v, ts))
	}
	return strings.Join(parts, " ")
}

func (c *Client) getWithURL(ctx context.Context, path string) ([]byte, *url.URL, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return nil, nil, err
	}
	if h := c.authHeader(); h != "" {
		req.Header.Set("Authorization", h)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		if errors.Is(err, errAccountsRedirect) || errors.Is(errors.Unwrap(err), errAccountsRedirect) {
			return nil, nil, fmt.Errorf("%w: redirected to accounts.google.com (run gplay web auth login again)", ErrAuth)
		}
		return nil, nil, fmt.Errorf("GET %s: %w", path, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil, nil, fmt.Errorf("%w: GET %s returned HTTP %d (run gplay web auth login again)", ErrAuth, path, resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil, nil, fmt.Errorf("GET %s returned HTTP %d", path, resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, err
	}
	return body, resp.Request.URL, nil
}

func (c *Client) getConsole(ctx context.Context) ([]byte, *url.URL, error) {
	path := "/console"
	if c.sess != nil && strings.TrimSpace(c.sess.UserEmail) != "" {
		path += "?authuser=" + url.QueryEscape(strings.TrimSpace(c.sess.UserEmail))
	}
	return c.getWithURL(ctx, path)
}

// developerIDRe accepts both the current account-scoped URL and the legacy
// /console/<developer-id>/ shape.
var developerIDRe = regexp.MustCompile(`/console/(?:u/\d+/developers/)?(\d+)(?:/|$)`)

// startupDataRe extracts the console's serialized startup blob, a JS string
// literal holding JSON with numeric field keys.
var startupDataRe = regexp.MustCompile(`serializedInitialChunks\['startupData'\]\s*=\s*("(?:[^"\\]|\\.)*")`)

// jsHexEscapeRe matches the \xNN escapes the blob uses for punctuation.
var jsHexEscapeRe = regexp.MustCompile(`\\x([0-9a-fA-F]{2})`)

// startupData is the console's serialized bootstrap blob. Field numbers come
// from the page itself: 1.8 is the public web API key, 2.1.1 the developer
// account currently in scope.
type startupData struct {
	Boot struct {
		APIKey string `json:"8"`
	} `json:"1"`
	Scope struct {
		Developer struct {
			Value string `json:"1"`
		} `json:"1"`
	} `json:"2"`
}

// parseStartupData decodes the console's bootstrap blob.
func parseStartupData(body []byte) (*startupData, error) {
	m := startupDataRe.FindSubmatch(body)
	if m == nil {
		return nil, errors.New("startupData not found in console HTML")
	}
	// \xNN is not valid JSON; \uXXXX is. Rewriting lets the JSON decoder
	// unescape the literal, then the result is itself JSON.
	literal := jsHexEscapeRe.ReplaceAll(m[1], []byte(`\u00$1`))

	var inner string
	if err := json.Unmarshal(literal, &inner); err != nil {
		return nil, fmt.Errorf("decoding startupData string: %w", err)
	}
	var data startupData
	if err := json.Unmarshal([]byte(inner), &data); err != nil {
		return nil, fmt.Errorf("parsing startupData: %w", err)
	}
	return &data, nil
}

// parseStartupDeveloperID reads the developer account currently in scope.
// Scanning the page for 19-digit numbers instead would match unrelated IDs and
// silently scope a write to the wrong developer account.
func parseStartupDeveloperID(body []byte) (string, error) {
	data, err := parseStartupData(body)
	if err != nil {
		return "", err
	}
	if data.Scope.Developer.Value == "" {
		return "", errors.New("no developer account in startupData")
	}
	return data.Scope.Developer.Value, nil
}

// DiscoverDeveloperID best-effort extracts the developer account ID from the
// console root page.
func (c *Client) DiscoverDeveloperID(ctx context.Context) (string, error) {
	body, finalURL, err := c.getConsole(ctx)
	if err != nil {
		return "", err
	}
	data, _ := parseStartupData(body)
	if data != nil {
		c.appsAPIKeyMu.Lock()
		c.appsAPIKey = strings.TrimSpace(data.Boot.APIKey)
		c.appsAPIKeyMu.Unlock()
	}
	for _, candidate := range [][]byte{[]byte(finalURL.Path), body} {
		if m := developerIDRe.FindSubmatch(candidate); m != nil {
			return string(m[1]), nil
		}
	}
	if data != nil && data.Scope.Developer.Value != "" {
		return data.Scope.Developer.Value, nil
	}
	return "", fmt.Errorf("%w: no developer account found in Play Console (run gplay web auth login again)", ErrAuth)
}
