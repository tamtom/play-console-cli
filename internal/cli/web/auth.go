package web

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/peterbourgon/ff/v3/ffcli"
	"github.com/tamtom/play-console-cli/internal/cli/shared"
	"github.com/tamtom/play-console-cli/internal/webclient"
	"github.com/tamtom/play-console-cli/internal/websession"
)

// playOrigin is the origin key under which Play Console cookies are stored.
const playOrigin = "https://play.google.com/"

// webRPCClient is the subset of webclient.Client used by the web commands.
// It exists so tests can inject a client backed by httptest servers.
type webRPCClient interface {
	DiscoverDeveloperID(ctx context.Context) (string, error)
	ListApps(ctx context.Context, developerID string) ([]webclient.App, error)
	PromoCodesTermsAccepted(ctx context.Context, developerID string) (bool, error)
	CheckPackageName(ctx context.Context, developerID, packageName string) (webclient.PackageAvailability, error)
	CreateApp(ctx context.Context, req webclient.CreateAppRequest) (*webclient.App, error)
}

// Test seams (same pattern as newReportingService in internal/cli/apps).
var (
	newWebClient = func(sess *websession.Session) webRPCClient {
		return webclient.New(sess)
	}
	sessionSave             = websession.Save
	sessionLoad             = websession.Load
	sessionDelete           = websession.Delete
	sessionDeleteAll        = websession.DeleteAll
	sessionList             = websession.List
	sessionStore            = websession.Store
	importChromeCookies     = websession.ImportChromeCookies
	importChromeCookiesFrom = websession.ImportChromeCookiesFrom
	chromeLauncher          = websession.LaunchChrome
	interactiveLauncher     = websession.LaunchInteractiveChrome
	readStdin               = func() ([]byte, error) { return io.ReadAll(os.Stdin) }
)

// browserPollInterval is how often --browser re-reads the dedicated profile
// while waiting for the user to finish signing in. Overridden in tests.
var browserPollInterval = 2 * time.Second

// consoleLoginURL is where the dedicated browser window starts.
const consoleLoginURL = "https://play.google.com/console"

// AuthCommand returns the `gplay web auth` command group.
func AuthCommand() *ffcli.Command {
	fs := flag.NewFlagSet("web auth", flag.ExitOnError)
	return &ffcli.Command{
		Name:       "auth",
		ShortUsage: "gplay web auth <subcommand> [flags]",
		ShortHelp:  "Manage Play Console browser sessions (cookies) for web RPCs.",
		LongHelp: `Manage Play Console browser sessions.

Web commands talk to Play Console's internal web APIs using cookies captured
from a signed-in browser. This is SEPARATE from "gplay auth" service accounts
and is required only for commands the official Android Publisher API does not
support (such as listing every app in a developer account).

Sessions are stored per account: in the macOS Keychain on macOS (cookies
never touch disk), or as files with 0600 permissions under ~/.gplay/web/
when GPLAY_WEB_SESSION_DIR is set or on other platforms.
Cookies are secrets: treat them like passwords and never commit them.`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			AuthLoginCommand(),
			AuthStatusCommand(),
			AuthLogoutCommand(),
		},
		Exec: func(ctx context.Context, args []string) error {
			if len(args) == 0 {
				return flag.ErrHelp
			}
			fmt.Fprintf(os.Stderr, "Unknown subcommand: %s\n", args[0])
			return flag.ErrHelp
		},
	}
}

// AuthLoginCommand returns the `gplay web auth login` subcommand.
func AuthLoginCommand() *ffcli.Command {
	fs := flag.NewFlagSet("web auth login", flag.ExitOnError)
	email := fs.String("email", "", "Google account email the cookies belong to")
	cookies := fs.String("cookies", "", "Raw Cookie header value copied from a play.google.com request")
	cookiesFile := fs.String("cookies-file", "", "File with cookies (raw header or exporter JSON; '-' for stdin)")
	browser := fs.Bool("browser", false, "Sign in via a gplay-controlled Chrome window with its own persistent profile")
	browserTimeout := fs.Duration("browser-timeout", 5*time.Minute, "How long to wait for the browser sign-in to complete")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "login",
		ShortUsage: "gplay web auth login --email <email> [--browser | --cookies <header> | --cookies-file <path|->]",
		ShortHelp:  "Store browser cookies as a Play Console web session.",
		LongHelp: `Store browser cookies as a Play Console web session.

By default, gplay imports the active Play Console cookies from Google Chrome
on macOS. macOS may ask you to allow access to "Chrome Safe Storage".

With --browser, gplay instead opens a Chrome window it controls, backed by its
own profile under ~/.gplay/web/browser. Sign in there once: the profile keeps
that login, so later runs reuse it without opening a window and without
touching your everyday Chrome profiles. If the profile is still signed in,
--browser refreshes the session silently. The window opens with no DevTools
port and closes itself once sign-in completes.

To provide cookies manually instead:
  1. Sign in to https://play.google.com/console in your browser.
  2. Open DevTools -> Network and select any request to play.google.com.
  3. Copy the full "Cookie:" request header value and pass it via --cookies,
     or save it to a file and use --cookies-file (use '-' to read from stdin).

Alternatively paste a JSON export from a cookie manager extension
(Cookie-Editor, EditThisCookie); the format is auto-detected.

The session is validated against the real Play Console before it is saved.
On macOS it is stored in the macOS Keychain (service "gplay web session");
with GPLAY_WEB_SESSION_DIR set, or on other platforms, as a 0600 file under
~/.gplay/web/ instead.

Examples:
  gplay web auth login --email me@example.com
  gplay web auth login --email me@example.com --browser
  gplay web auth login --email me@example.com --cookies "SID=...; SAPISID=...; ..."
  pbpaste | gplay web auth login --email me@example.com --cookies-file -`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			email := strings.TrimSpace(*email)
			if email == "" {
				return fmt.Errorf("--email is required")
			}
			hasInline := strings.TrimSpace(*cookies) != ""
			hasFile := strings.TrimSpace(*cookiesFile) != ""
			if hasInline && hasFile {
				return fmt.Errorf("--cookies and --cookies-file are mutually exclusive")
			}
			if *browser && (hasInline || hasFile) {
				return fmt.Errorf("--browser cannot be combined with --cookies or --cookies-file")
			}

			var (
				byOrigin    map[string][]websession.Cookie
				developerID string
				sess        *websession.Session
				err         error
			)
			if *browser {
				sess, byOrigin, developerID, err = browserLogin(ctx, email, *browserTimeout)
			} else {
				var candidates []map[string][]websession.Cookie
				if !hasInline && !hasFile {
					candidates, err = importChromeCookies(ctx)
					if err != nil {
						return fmt.Errorf("automatic Chrome cookie import failed: %w; alternatively use --browser, --cookies or --cookies-file", err)
					}
				} else {
					source := strings.TrimSpace(*cookies)
					if hasFile {
						source, err = readCookieFile(strings.TrimSpace(*cookiesFile))
						if err != nil {
							return err
						}
					}
					parsed, perr := parseCookieSource(source)
					if perr != nil {
						return perr
					}
					candidates = []map[string][]websession.Cookie{parsed}
				}
				sess, byOrigin, developerID, err = pickValidCandidate(ctx, email, candidates)
			}
			if err != nil {
				return err
			}

			if err := sessionSave(sess); err != nil {
				return err
			}

			cookieCount := 0
			origins := make([]string, 0, len(byOrigin))
			for origin, list := range byOrigin {
				origins = append(origins, origin)
				cookieCount += len(list)
			}
			sort.Strings(origins)

			return shared.PrintOutput(map[string]any{
				"email":         email,
				"developer_id":  developerID,
				"cookie_count":  cookieCount,
				"origins":       origins,
				"session_store": sessionStore(email),
				"validated":     true,
				"validated_utc": time.Now().UTC().Format(time.RFC3339),
			}, *outputFlag, *pretty)
		},
	}
}

// AuthStatusCommand returns the `gplay web auth status` subcommand.
func AuthStatusCommand() *ffcli.Command {
	fs := flag.NewFlagSet("web auth status", flag.ExitOnError)
	check := fs.Bool("check", false, "Validate each stored session against the real Play Console")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "status",
		ShortUsage: "gplay web auth status [--check]",
		ShortHelp:  "List stored web sessions and optionally validate them.",
		LongHelp: `List the stored Play Console web sessions, sorted by account email.

Each entry reports the account email, when the session file was last written,
and its path. Cookies are never printed. With --check, every session is used
against the real Play Console and reported as "valid" or "expired"; this costs
one request per session, so it is off by default.

Examples:
  gplay web auth status
  gplay web auth status --check --output table`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			emails, err := sessionList()
			if err != nil {
				return err
			}

			type entry struct {
				Email        string `json:"email"`
				UpdatedAt    string `json:"updated_at,omitempty"`
				SessionStore string `json:"session_store,omitempty"`
				Status       string `json:"status,omitempty"`
			}
			entries := make([]entry, 0, len(emails))
			for _, email := range emails {
				e := entry{Email: email}
				if sess, err := sessionLoad(email); err == nil {
					e.UpdatedAt = sess.UpdatedAt.UTC().Format(time.RFC3339)
					e.SessionStore = sessionStore(email)
					if *check {
						client := newWebClient(sess)
						cctx, cancel := shared.ContextWithTimeout(ctx, nil)
						_, err := client.DiscoverDeveloperID(cctx)
						cancel()
						if err != nil {
							e.Status = "expired"
						} else {
							e.Status = "valid"
						}
					}
				}
				entries = append(entries, e)
			}
			return shared.PrintOutput(entries, *outputFlag, *pretty)
		},
	}
}

// AuthLogoutCommand returns the `gplay web auth logout` subcommand.
func AuthLogoutCommand() *ffcli.Command {
	fs := flag.NewFlagSet("web auth logout", flag.ExitOnError)
	account := fs.String("account", "", "Account email to remove (default: last used session)")
	all := fs.Bool("all", false, "Remove all stored web sessions")
	confirm := fs.Bool("confirm", false, "Confirm removal")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "logout",
		ShortUsage: "gplay web auth logout [--account <email> | --all] --confirm",
		ShortHelp:  "Delete stored web sessions.",
		LongHelp: `Delete stored Play Console web sessions.

Without --account the last used session is removed; --all removes every stored
session. --confirm is required either way.

This only deletes gplay's saved cookies. The Chrome profile created by
"gplay web auth login --browser" stays signed in under ~/.gplay/web/browser, so
a later "login --browser" restores a session without any interaction. Delete
that directory too for a full sign-out.

Examples:
  gplay web auth logout --confirm
  gplay web auth logout --account me@example.com --confirm
  gplay web auth logout --all --confirm`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			if !*confirm {
				return fmt.Errorf("--confirm is required to delete web sessions")
			}
			if *all {
				if err := sessionDeleteAll(); err != nil {
					return err
				}
				return shared.PrintOutput(map[string]any{"deleted": "all"}, *outputFlag, *pretty)
			}
			account := strings.TrimSpace(*account)
			if err := sessionDelete(account); err != nil {
				return err
			}
			if account == "" {
				account = "last"
			}
			return shared.PrintOutput(map[string]any{"deleted": account}, *outputFlag, *pretty)
		},
	}
}

// pickValidCandidate validates cookie candidates against the real console and
// returns the first that authenticates. Validating before saving keeps a bad
// paste or a stale profile from clobbering a working session.
func pickValidCandidate(ctx context.Context, email string, candidates []map[string][]websession.Cookie) (*websession.Session, map[string][]websession.Cookie, string, error) {
	var validateErr error
	for _, candidate := range candidates {
		if !hasSAPISID(candidate[playOrigin]) {
			continue
		}
		current := &websession.Session{UserEmail: email, Cookies: candidate}
		cctx, cancel := shared.ContextWithTimeout(ctx, nil)
		developerID, err := newWebClient(current).DiscoverDeveloperID(cctx)
		cancel()
		if err == nil {
			return current, candidate, developerID, nil
		}
		validateErr = err
	}
	if validateErr != nil {
		return nil, nil, "", fmt.Errorf("web session validation failed: %w", validateErr)
	}
	return nil, nil, "", fmt.Errorf("no SAPISID cookie found for %s — sign in to Play Console and retry", playOrigin)
}

// browserLogin drives the dedicated Chrome profile: reuse it when it is still
// signed in, otherwise open a visible window and poll the profile until the
// sign-in completes. The window opens without a DevTools port — it holds a
// live Google session, and DevTools has no authentication — and is closed once
// the session lands. The profile on disk keeps the sign-in for later runs.
func browserLogin(ctx context.Context, email string, timeout time.Duration) (*websession.Session, map[string][]websession.Cookie, string, error) {
	dir := websession.BrowserProfileDir()
	if sess, byOrigin, developerID, err := browserAttempt(ctx, email, dir); err == nil {
		return sess, byOrigin, developerID, nil
	}

	terminate, err := interactiveLauncher(ctx, dir, consoleLoginURL)
	if err != nil {
		return nil, nil, "", err
	}
	fmt.Fprintf(os.Stderr, "Opened a gplay-controlled Chrome window; sign in as %s. The window closes when sign-in completes.\n", email)
	fmt.Fprintf(os.Stderr, "Profile: %s (kept signed in, so this is a one-time step)\n", dir)

	deadline := time.Now().Add(timeout)
	for {
		select {
		case <-ctx.Done():
			return nil, nil, "", ctx.Err()
		case <-time.After(browserPollInterval):
		}
		if sess, byOrigin, developerID, err := browserAttempt(ctx, email, dir); err == nil {
			terminate() // close the window we opened: it now holds a live session
			return sess, byOrigin, developerID, nil
		}
		if time.Now().After(deadline) {
			return nil, nil, "", fmt.Errorf("timed out after %s waiting for the Play Console sign-in; leave the window signed in and rerun, or raise --browser-timeout", timeout)
		}
	}
}

// refreshFromBrowserProfile silently re-imports the dedicated Chrome profile
// and stores the session when it still authenticates, so routine commands keep
// working across the roughly daily cookie expiry. It deliberately never opens a
// window: this runs on the failure path of an ordinary command, where a
// surprise browser is worse than a clear error telling the user to log in.
func refreshFromBrowserProfile(ctx context.Context, email string) *websession.Session {
	if strings.TrimSpace(email) == "" {
		return nil
	}
	sess, _, _, err := browserAttempt(ctx, email, websession.BrowserProfileDir())
	if err != nil {
		return nil
	}
	if err := sessionSave(sess); err != nil {
		return nil
	}
	return sess
}

// browserAttempt reads the dedicated profile once and validates what it finds.
func browserAttempt(ctx context.Context, email, dir string) (*websession.Session, map[string][]websession.Cookie, string, error) {
	candidates, err := importChromeCookiesFrom(ctx, dir)
	if err != nil {
		return nil, nil, "", err
	}
	return pickValidCandidate(ctx, email, candidates)
}

// readCookieFile reads cookie data from a file path, or stdin when path is "-".
func readCookieFile(path string) (string, error) {
	if path == "-" {
		data, err := readStdin()
		if err != nil {
			return "", fmt.Errorf("reading cookies from stdin: %w", err)
		}
		return string(data), nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", shared.WrapActionable(err, "failed to read cookies file", "Check that the file exists and is readable.")
	}
	return string(data), nil
}

// parseCookieSource auto-detects the cookie format: JSON exporter dumps start
// with '[' or '{'; anything else is treated as a raw Cookie header for the
// Play Console origin.
func parseCookieSource(source string) (map[string][]websession.Cookie, error) {
	trimmed := strings.TrimSpace(source)
	if trimmed == "" {
		return nil, fmt.Errorf("cookie data is empty")
	}
	if strings.HasPrefix(trimmed, "[") || strings.HasPrefix(trimmed, "{") {
		byOrigin, err := websession.ParseCookieExportJSON([]byte(trimmed))
		if err != nil {
			return nil, err
		}
		return projectCookiesForOrigin(byOrigin, playOrigin)
	}
	cookies, err := websession.ParseCookieHeader(playOrigin, trimmed)
	if err != nil {
		return nil, err
	}
	return map[string][]websession.Cookie{playOrigin: cookies}, nil
}

// projectCookiesForOrigin collects cookies whose exported domain applies to
// the target host. For example, .google.com cookies apply to play.google.com.
func projectCookiesForOrigin(byOrigin map[string][]websession.Cookie, target string) (map[string][]websession.Cookie, error) {
	targetURL, err := url.Parse(target)
	if err != nil {
		return nil, fmt.Errorf("parsing target origin: %w", err)
	}
	var cookies []websession.Cookie
	for origin, list := range byOrigin {
		originURL, err := url.Parse(origin)
		if err != nil {
			continue
		}
		if targetURL.Host == originURL.Host || strings.HasSuffix(targetURL.Host, "."+originURL.Host) {
			cookies = append(cookies, list...)
		}
	}
	return map[string][]websession.Cookie{target: cookies}, nil
}

// hasSAPISID reports whether the cookie list carries a SAPISID-class cookie
// usable for SAPISIDHASH auth.
func hasSAPISID(cookies []websession.Cookie) bool {
	for _, c := range cookies {
		switch c.Name {
		case "SAPISID", "__Secure-1PAPISID", "__Secure-3PAPISID":
			if c.Value != "" {
				return true
			}
		}
	}
	return false
}
