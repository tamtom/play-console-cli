package snitch

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/tamtom/play-console-cli/internal/cli/shared"
)

const (
	githubTokenEnvVar    = "GITHUB_TOKEN"
	githubTokenGHEnvVar  = "GH_TOKEN"
	defaultOwner         = "tamtom"
	defaultRepo          = "play-console-cli"
	maxSearchResults     = 5
	maxResponseBodyBytes = 8192
	maxLabelPages        = 10
	labelsPerPage        = 100
)

// githubAPIBase is a variable so tests can override it with httptest servers.
var githubAPIBase = "https://api.github.com"

// setGitHubAPIBase is used by tests to point at httptest servers.
func setGitHubAPIBase(base string) {
	githubAPIBase = base
}

var validSeverities = []string{"bug", "enhancement"}

// githubHTTPClient is a package-level var for testability.
var githubHTTPClient = func() *http.Client {
	return &http.Client{Timeout: 30 * time.Second}
}

type stringListFlag []string

func (f *stringListFlag) String() string {
	if f == nil {
		return ""
	}
	return strings.Join(*f, ",")
}

func (f *stringListFlag) Set(value string) error {
	label := strings.TrimSpace(value)
	if label == "" {
		return fmt.Errorf("label must not be empty")
	}
	*f = append(*f, label)
	return nil
}

// SnitchCommand returns the top-level snitch command.
func SnitchCommand(version string) *ffcli.Command {
	fs := flag.NewFlagSet("snitch", flag.ContinueOnError)

	repro := fs.String("repro", "", "Reproduction steps (required)")
	expected := fs.String("expected", "", "Expected behavior (required)")
	actual := fs.String("actual", "", "Actual behavior (required)")
	severity := fs.String("severity", "bug", "Severity: bug or enhancement")
	dryRun := fs.Bool("dry-run", false, "Show what would be filed without filing")
	local := fs.Bool("local", false, "Log to .gplay/snitch.log instead of filing on GitHub")
	confirm := fs.Bool("confirm", false, "Create the GitHub issue after duplicate search")
	var labels stringListFlag
	fs.Var(&labels, "label", "Existing repo label to attach (repeatable)")

	return &ffcli.Command{
		Name:       "snitch",
		ShortUsage: "gplay snitch [flags]",
		ShortHelp:  "Report CLI friction as a GitHub issue.",
		LongHelp: `Report CLI friction directly from the terminal.

Searches for duplicate issues when GITHUB_TOKEN or GH_TOKEN is available.
Without --confirm, snitch prints a preview only. Use --local to log friction
offline for later review with "gplay snitch flush".

Examples:
  gplay snitch --repro 'gplay vitals crashes --package "com.example"' --expected "Should show crashes" --actual "Error: package not found" --confirm
  gplay snitch --repro "gplay tracks list" --expected "tracks listed" --actual "timeout" --dry-run
  gplay snitch --repro "gplay tracks list" --expected "tracks listed" --actual "timeout" --local
  gplay snitch flush
  gplay snitch flush --file .gplay/snitch.log`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			flushCommand(),
		},
		Exec: func(ctx context.Context, args []string) error {
			// Validate required flags.
			if strings.TrimSpace(*repro) == "" {
				return shared.UsageError("--repro is required")
			}
			if strings.TrimSpace(*expected) == "" {
				return shared.UsageError("--expected is required")
			}
			if strings.TrimSpace(*actual) == "" {
				return shared.UsageError("--actual is required")
			}

			sev := strings.TrimSpace(strings.ToLower(*severity))
			if !isValidSeverity(sev) {
				return shared.UsageErrorf("--severity must be one of: %s", strings.Join(validSeverities, ", "))
			}

			entry := LogEntry{
				Description:  strings.TrimSpace(*repro),
				Repro:        strings.TrimSpace(*repro),
				Expected:     strings.TrimSpace(*expected),
				Actual:       strings.TrimSpace(*actual),
				Labels:       append([]string(nil), labels...),
				Severity:     sev,
				Timestamp:    time.Now().UTC(),
				GplayVersion: version,
				OS:           fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
				GoVersion:    runtime.Version(),
			}

			token := resolveGitHubToken()

			if len(entry.Labels) > 0 && !(*local && !*dryRun) {
				validatedLabels, err := validateRequestedLabels(ctx, token, entry.Labels)
				if err != nil {
					if strings.Contains(err.Error(), "flag") {
						return err
					}
					fmt.Fprintf(os.Stderr, "Warning: %v; continuing without preflight label validation.\n", err)
					entry.Labels = dedupeLabels(entry.Labels)
				} else {
					entry.Labels = validatedLabels
				}
			}

			if *local && !*dryRun {
				return writeLocalLog(entry)
			}

			var duplicates []GitHubIssue
			if token != "" {
				var err error
				duplicates, err = searchIssues(ctx, token, issueTitle(entry))
				if err != nil {
					fmt.Fprintf(os.Stderr, "Warning: duplicate search failed: %v\n", err)
				}
			} else {
				fmt.Fprintln(os.Stderr, "Note: skipping duplicate search because GITHUB_TOKEN or GH_TOKEN is not set.")
			}

			printPotentialDuplicates(duplicates)

			if *dryRun || !*confirm {
				printPreview(entry, *dryRun)
				return nil
			}

			if token == "" {
				return fmt.Errorf("snitch: GITHUB_TOKEN or GH_TOKEN is required to create issues")
			}

			issue, err := createIssue(ctx, token, entry)
			if err != nil {
				return fmt.Errorf("snitch: failed to create issue: %w", err)
			}
			if labels := issueLabels(entry); len(labels) > 0 {
				if err := addIssueLabels(ctx, token, issue.Number, labels); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: issue created, but labels could not be applied: %v\n", err)
				}
			}

			fmt.Fprintf(os.Stderr, "Issue created: #%d %s\n", issue.Number, issue.HTMLURL)
			result := map[string]any{
				"number":   issue.Number,
				"html_url": issue.HTMLURL,
				"title":    issue.Title,
			}
			return json.NewEncoder(os.Stdout).Encode(result)
		},
	}
}

func flushCommand() *ffcli.Command {
	fs := flag.NewFlagSet("snitch flush", flag.ContinueOnError)
	logFile := fs.String("file", "", "Path to snitch log file (default: .gplay/snitch.log)")

	return &ffcli.Command{
		Name:       "flush",
		ShortUsage: "gplay snitch flush [--file PATH]",
		ShortHelp:  "Review locally logged friction entries.",
		LongHelp: `Review friction entries logged with --local.

Prints all entries from .gplay/snitch.log (or --file path) in a readable format.
Filing from flush is manual: copy the description and rerun "gplay snitch"
with --confirm when you're ready to create the issue.

Examples:
  gplay snitch flush
  gplay snitch flush --file .gplay/snitch.log`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if len(args) > 0 {
				return shared.UsageError("snitch flush does not accept positional arguments; use --file PATH to specify a log file")
			}

			path := strings.TrimSpace(*logFile)
			if path == "" {
				path = filepath.Join(".gplay", "snitch.log")
			}

			entries, err := readLocalLog(path)
			if os.IsNotExist(err) {
				fmt.Fprintln(os.Stderr, "No local snitch entries found.")
				return nil
			}
			if err != nil {
				return fmt.Errorf("snitch flush: %w", err)
			}

			if len(entries) == 0 {
				fmt.Fprintln(os.Stderr, "No local snitch entries found.")
				return nil
			}

			fmt.Fprint(os.Stdout, formatLocalEntries(entries))
			return nil
		},
	}
}

// LogEntry represents a friction report.
type LogEntry struct {
	Description  string    `json:"description"`
	Repro        string    `json:"repro,omitempty"`
	Expected     string    `json:"expected,omitempty"`
	Actual       string    `json:"actual,omitempty"`
	Labels       []string  `json:"labels,omitempty"`
	Severity     string    `json:"severity"`
	Timestamp    time.Time `json:"timestamp"`
	GplayVersion string    `json:"gplay_version"`
	OS           string    `json:"os"`
	GoVersion    string    `json:"go_version,omitempty"`
}

// GitHubIssue represents a GitHub issue (search result or creation response).
type GitHubIssue struct {
	Number  int    `json:"number"`
	Title   string `json:"title"`
	HTMLURL string `json:"html_url"`
	State   string `json:"state"`
}

func isValidSeverity(s string) bool {
	for _, v := range validSeverities {
		if s == v {
			return true
		}
	}
	return false
}

func resolveGitHubToken() string {
	if v := strings.TrimSpace(os.Getenv(githubTokenEnvVar)); v != "" {
		return v
	}
	if v := strings.TrimSpace(os.Getenv(githubTokenGHEnvVar)); v != "" {
		return v
	}
	return ""
}

func issueTitle(e LogEntry) string {
	prefix := ""
	switch e.Severity {
	case "enhancement":
		prefix = "Enhancement: "
	}
	return prefix + e.Description
}

func issueBody(e LogEntry) string {
	var b strings.Builder

	if e.Repro != "" {
		b.WriteString("## Reproduction Steps\n")
		b.WriteString(e.Repro)
		b.WriteString("\n\n")
	}

	if e.Expected != "" {
		b.WriteString("## Expected Behavior\n")
		b.WriteString(e.Expected)
		b.WriteString("\n\n")
	}

	if e.Actual != "" {
		b.WriteString("## Actual Behavior\n")
		b.WriteString(e.Actual)
		b.WriteString("\n\n")
	}

	b.WriteString("## Environment\n")
	b.WriteString(fmt.Sprintf("- gplay version: %s\n", e.GplayVersion))
	b.WriteString(fmt.Sprintf("- OS: %s\n", e.OS))
	if e.GoVersion != "" {
		b.WriteString(fmt.Sprintf("- Go: %s\n", e.GoVersion))
	}

	return b.String()
}

func issueLabels(e LogEntry) []string {
	labels := []string{"gplay-snitch"}
	switch e.Severity {
	case "bug":
		labels = append(labels, "bug")
	case "enhancement":
		labels = append(labels, "enhancement")
	}
	labels = append(labels, e.Labels...)
	return dedupeLabels(labels)
}

func printPotentialDuplicates(duplicates []GitHubIssue) {
	if len(duplicates) == 0 {
		return
	}

	fmt.Fprintf(os.Stderr, "Potentially related issues (%d):\n", len(duplicates))
	for _, dup := range duplicates {
		fmt.Fprintf(os.Stderr, "  #%d %s\n       %s\n", dup.Number, dup.Title, dup.HTMLURL)
	}
	fmt.Fprintln(os.Stderr)
}

func printPreview(entry LogEntry, dryRun bool) {
	if dryRun {
		fmt.Fprintln(os.Stderr, "--- Dry run: would create issue ---")
	} else {
		fmt.Fprintln(os.Stderr, "--- Preview only: rerun with --confirm to create issue ---")
	}
	fmt.Fprintf(os.Stderr, "Title: %s\n", issueTitle(entry))
	if labels := issueLabels(entry); len(labels) > 0 {
		fmt.Fprintf(os.Stderr, "Labels: %s\n", strings.Join(labels, ", "))
	}
	fmt.Fprintf(os.Stderr, "Body:\n%s\n", issueBody(entry))
}

func dedupeLabels(labels []string) []string {
	if len(labels) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(labels))
	deduped := make([]string, 0, len(labels))
	for _, raw := range labels {
		label := strings.TrimSpace(raw)
		if label == "" {
			continue
		}
		key := strings.ToLower(label)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		deduped = append(deduped, label)
	}

	return deduped
}

func listRepoLabels(ctx context.Context, token string) ([]string, error) {
	seen := make(map[string]struct{})
	labels := make([]string, 0, labelsPerPage)

	for page := 1; page <= maxLabelPages; page++ {
		labelsURL := fmt.Sprintf(
			"%s/repos/%s/%s/labels?per_page=%d&page=%d",
			githubAPIBase,
			defaultOwner,
			defaultRepo,
			labelsPerPage,
			page,
		)

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, labelsURL, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Accept", "application/vnd.github+json")
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}

		resp, err := githubHTTPClient().Do(req)
		if err != nil {
			return nil, err
		}

		var pageLabels []struct {
			Name string `json:"name"`
		}
		if resp.StatusCode == http.StatusOK {
			err = json.NewDecoder(resp.Body).Decode(&pageLabels)
		} else {
			err = readGitHubAPIError(resp)
		}
		resp.Body.Close()
		if err != nil {
			return nil, err
		}

		for _, label := range pageLabels {
			name := strings.TrimSpace(label.Name)
			if name == "" {
				continue
			}
			key := strings.ToLower(name)
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			labels = append(labels, name)
		}

		if len(pageLabels) < labelsPerPage {
			break
		}
	}

	sort.Strings(labels)
	return labels, nil
}

func validateRequestedLabels(ctx context.Context, token string, requested []string) ([]string, error) {
	requested = dedupeLabels(requested)
	if len(requested) == 0 {
		return nil, nil
	}

	repoLabels, err := listRepoLabels(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("failed to validate --label values: %w", err)
	}

	labelIndex := make(map[string]string, len(repoLabels))
	for _, label := range repoLabels {
		labelIndex[strings.ToLower(label)] = label
	}

	validated := make([]string, 0, len(requested))
	unknown := make([]string, 0, len(requested))
	for _, label := range requested {
		canonical, ok := labelIndex[strings.ToLower(label)]
		if !ok {
			unknown = append(unknown, label)
			continue
		}
		validated = append(validated, canonical)
	}

	if len(unknown) > 0 {
		sort.Strings(unknown)
		return nil, shared.UsageErrorf("--label must reference existing repo labels; unknown label(s): %s", strings.Join(unknown, ", "))
	}

	return validated, nil
}

func readLocalLog(path string) ([]LogEntry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" {
		return nil, nil
	}

	lines := strings.Split(trimmed, "\n")
	entries := make([]LogEntry, 0, len(lines))
	for i, rawLine := range lines {
		line := strings.TrimSpace(rawLine)
		if line == "" {
			continue
		}

		var entry LogEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			return nil, fmt.Errorf("invalid log entry on line %d: %w", i+1, err)
		}
		entries = append(entries, entry)
	}

	return entries, nil
}

func formatLocalEntries(entries []LogEntry) string {
	var b strings.Builder

	for i, entry := range entries {
		fmt.Fprintf(&b, "[%d] %s: %s\n", i+1, entry.Severity, entry.Description)
		if !entry.Timestamp.IsZero() {
			fmt.Fprintf(&b, "Timestamp: %s\n", entry.Timestamp.Format(time.RFC3339))
		}
		if entry.GplayVersion != "" {
			fmt.Fprintf(&b, "gplay version: %s\n", entry.GplayVersion)
		}
		if entry.OS != "" {
			fmt.Fprintf(&b, "OS: %s\n", entry.OS)
		}
		if entry.Repro != "" {
			fmt.Fprintf(&b, "Reproduction:\n%s\n", entry.Repro)
		}
		if entry.Expected != "" {
			fmt.Fprintf(&b, "Expected:\n%s\n", entry.Expected)
		}
		if entry.Actual != "" {
			fmt.Fprintf(&b, "Actual:\n%s\n", entry.Actual)
		}
		if len(entry.Labels) > 0 {
			fmt.Fprintf(&b, "Labels: %s\n", strings.Join(entry.Labels, ", "))
		}
		if i < len(entries)-1 {
			b.WriteString("\n")
		}
	}

	return b.String()
}

func searchIssues(ctx context.Context, token string, query string) ([]GitHubIssue, error) {
	q := fmt.Sprintf("repo:%s/%s is:issue is:open in:title %q", defaultOwner, defaultRepo, strings.TrimSpace(query))
	searchURL := fmt.Sprintf("%s/search/issues?q=%s&per_page=%d",
		githubAPIBase, url.QueryEscape(q), maxSearchResults)

	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	client := githubHTTPClient()
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		limited := io.LimitReader(resp.Body, maxResponseBodyBytes)
		body, _ := io.ReadAll(limited)
		return nil, fmt.Errorf("GitHub search returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var result struct {
		Items []GitHubIssue `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode search results: %w", err)
	}

	return result.Items, nil
}

func createIssue(ctx context.Context, token string, entry LogEntry) (*GitHubIssue, error) {
	issueURL := fmt.Sprintf("%s/repos/%s/%s/issues", githubAPIBase, defaultOwner, defaultRepo)

	payload := map[string]any{
		"title": issueTitle(entry),
		"body":  issueBody(entry),
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", issueURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	client := githubHTTPClient()
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, readGitHubAPIError(resp)
	}

	var issue GitHubIssue
	if err := json.NewDecoder(resp.Body).Decode(&issue); err != nil {
		return nil, fmt.Errorf("failed to decode issue response: %w", err)
	}

	return &issue, nil
}

func addIssueLabels(ctx context.Context, token string, issueNumber int, labels []string) error {
	if len(labels) == 0 {
		return nil
	}

	labelsURL := fmt.Sprintf(
		"%s/repos/%s/%s/issues/%d/labels",
		githubAPIBase,
		defaultOwner,
		defaultRepo,
		issueNumber,
	)

	payload := map[string]any{
		"labels": labels,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", labelsURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	client := githubHTTPClient()
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return readGitHubAPIError(resp)
	}

	return nil
}

func readGitHubAPIError(resp *http.Response) error {
	limited := io.LimitReader(resp.Body, maxResponseBodyBytes)
	respBody, _ := io.ReadAll(limited)
	return fmt.Errorf("GitHub returned %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
}

func writeLocalLog(entry LogEntry) error {
	dir := ".gplay"
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("snitch: failed to create %s: %w", dir, err)
	}

	path := filepath.Join(dir, "snitch.log")

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("snitch: failed to marshal entry: %w", err)
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("snitch: failed to open %s: %w", path, err)
	}
	defer f.Close()

	if err := f.Chmod(0o600); err != nil {
		return fmt.Errorf("snitch: failed to set secure permissions on %s: %w", path, err)
	}

	if _, err := f.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("snitch: failed to write entry: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Friction logged to %s\n", path)
	return nil
}
