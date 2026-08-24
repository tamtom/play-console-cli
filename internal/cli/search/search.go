// Package search provides deterministic, credential-free CLI discovery.
package search

import (
	"context"
	"flag"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/tamtom/play-console-cli/internal/capability"
	"github.com/tamtom/play-console-cli/internal/cli/shared"
	"github.com/tamtom/play-console-cli/internal/output"
)

const defaultLimit = 10

// Response is the machine-readable result of a command search.
type Response struct {
	Query   string   `json:"query"`
	Count   int      `json:"count"`
	Results []Result `json:"results"`
}

// Result describes a matching public CLI command.
type Result struct {
	Command  string   `json:"command"`
	Summary  string   `json:"summary"`
	Usage    string   `json:"usage,omitempty"`
	Score    int      `json:"score"`
	Matched  []string `json:"matched"`
	Examples []string `json:"examples,omitempty"`
}

type document struct {
	command   string
	summary   string
	usage     string
	examples  []string
	depth     int
	path      tokenSet
	summaryT  tokenSet
	help      tokenSet
	flags     tokenSet
	canonical tokenSet
	phrases   []string
}

type tokenSet map[string]struct{}

func init() {
	output.RegisterType(Response{}, []string{"SCORE", "COMMAND", "SUMMARY", "MATCHED"}, func(data any) [][]string {
		response := data.(Response)
		rows := make([][]string, 0, len(response.Results))
		for _, result := range response.Results {
			rows = append(rows, []string{
				strconv.Itoa(result.Score),
				result.Command,
				result.Summary,
				strings.Join(result.Matched, ", "),
			})
		}
		return rows
	})
}

// Command returns the local command-discovery command. The command tree is
// supplied lazily so normal invocations never materialize it.
func Command(commands func() []*ffcli.Command) *ffcli.Command {
	fs := flag.NewFlagSet("search", flag.ExitOnError)
	limit := fs.Int("limit", defaultLimit, "Maximum number of results")
	outputFlags := shared.BindOutputFlags(fs)

	return &ffcli.Command{
		Name:       "search",
		ShortUsage: "gplay search [flags] <query>",
		ShortHelp:  "Search commands, examples, flags, capabilities, and canonical intents.",
		LongHelp: `Search the complete gplay command surface locally and deterministically.

The index contains command paths, summaries, usages, examples, flags, official
API resources, and policy-aware capability intents. Search never authenticates,
contacts Google, reads credentials, or changes an account.

Examples:
  gplay search "initial app record"
  gplay search "staged rollout fraction"
  gplay search --limit 5 "reply to reviews"
  gplay search --output table "upload bundle"`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			_ = ctx
			if len(args) == 0 || strings.TrimSpace(strings.Join(args, " ")) == "" {
				return shared.UsageError("search query is required")
			}
			if *limit <= 0 {
				return shared.UsageError("--limit must be greater than 0")
			}
			if commands == nil {
				return fmt.Errorf("search command catalog is unavailable")
			}
			if err := shared.ValidateOutputFlags(outputFlags.Format(), outputFlags.IsPretty()); err != nil {
				return err
			}

			response := Commands(commands(), strings.Join(args, " "), *limit)
			return shared.PrintOutput(response, outputFlags.Format(), outputFlags.IsPretty())
		},
	}
}

// Commands searches a materialized command tree. Results are ranked and
// ordered deterministically.
func Commands(commands []*ffcli.Command, query string, limit int) Response {
	normalized := strings.Join(strings.Fields(strings.ToLower(query)), " ")
	if limit <= 0 {
		limit = defaultLimit
	}
	documents := collectDocuments(commands)
	enrichWithCapabilities(documents)
	queryTokens := tokens(normalized)

	type scored struct {
		result Result
		depth  int
	}
	matches := make([]scored, 0)
	for _, doc := range documents {
		score, reasons := score(doc, queryTokens, normalized)
		if score == 0 {
			continue
		}
		matches = append(matches, scored{
			depth: doc.depth,
			result: Result{
				Command:  doc.command,
				Summary:  doc.summary,
				Usage:    doc.usage,
				Score:    score,
				Matched:  reasons,
				Examples: doc.examples,
			},
		})
	}

	sort.Slice(matches, func(i, j int) bool {
		if matches[i].result.Score != matches[j].result.Score {
			return matches[i].result.Score > matches[j].result.Score
		}
		if matches[i].depth != matches[j].depth {
			return matches[i].depth > matches[j].depth
		}
		return matches[i].result.Command < matches[j].result.Command
	})
	if len(matches) > limit {
		matches = matches[:limit]
	}

	results := make([]Result, 0, len(matches))
	for _, match := range matches {
		results = append(results, match.result)
	}
	return Response{Query: normalized, Count: len(results), Results: results}
}

func collectDocuments(commands []*ffcli.Command) []*document {
	var result []*document
	for _, command := range commands {
		collectDocument(&result, command, nil)
	}
	return result
}

func collectDocument(result *[]*document, command *ffcli.Command, parents []string) {
	if command == nil || strings.HasPrefix(command.ShortHelp, "DEPRECATED:") {
		return
	}
	parts := append(append([]string{"gplay"}, parents...), command.Name)
	path := strings.Join(parts, " ")
	usage := strings.TrimSpace(command.ShortUsage)
	if usage == "" {
		usage = path
	}
	longHelp := strings.TrimSpace(command.LongHelp)
	flags := make([]string, 0)
	if command.FlagSet != nil {
		command.FlagSet.VisitAll(func(item *flag.Flag) {
			flags = append(flags, item.Name+" "+item.Usage)
		})
	}

	*result = append(*result, &document{
		command:   path,
		summary:   strings.TrimSpace(command.ShortHelp),
		usage:     usage,
		examples:  examples(longHelp),
		depth:     len(parts) - 1,
		path:      tokenMap(strings.Join(parts[1:], " ")),
		summaryT:  tokenMap(command.ShortHelp + " " + usage),
		help:      tokenMap(longHelp),
		flags:     tokenMap(strings.Join(flags, " ")),
		canonical: tokenSet{},
	})

	nextParents := append(append([]string{}, parents...), command.Name)
	for _, subcommand := range command.Subcommands {
		collectDocument(result, subcommand, nextParents)
	}
}

func enrichWithCapabilities(documents []*document) {
	items, err := capability.List(capability.Filter{})
	if err != nil {
		return
	}
	for _, item := range items {
		command := strings.ToLower(strings.TrimSpace(item.Command))
		if command == "" {
			continue
		}
		canonicalText := strings.Join([]string{
			item.ID, item.Intent, item.Provider, item.APIResource, item.Notes, item.NextAction,
		}, " ")
		for _, doc := range documents {
			if strings.ToLower(doc.command) != command {
				continue
			}
			merge(doc.canonical, tokenMap(canonicalText))
			doc.phrases = append(doc.phrases, strings.ToLower(item.Intent))
		}
	}
}

func score(doc *document, queryTokens []string, query string) (int, []string) {
	if len(queryTokens) == 0 {
		return 0, nil
	}
	total := 0
	reasons := make(map[string]struct{})
	for _, token := range queryTokens {
		best := 0
		reason := ""
		for _, candidate := range []struct {
			set    tokenSet
			weight int
			name   string
		}{
			{doc.path, 120, "command"},
			{doc.canonical, 100, "capability"},
			{doc.summaryT, 70, "summary"},
			{doc.flags, 50, "flag"},
			{doc.help, 30, "help/example"},
		} {
			if _, ok := candidate.set[token]; ok && candidate.weight > best {
				best = candidate.weight
				reason = candidate.name
				continue
			}
			if hasPrefix(candidate.set, token) && candidate.weight/2 > best {
				best = candidate.weight / 2
				reason = candidate.name + "-prefix"
			}
		}
		if best == 0 {
			return 0, nil
		}
		total += best
		reasons[reason] = struct{}{}
	}
	for _, phrase := range doc.phrases {
		if query != "" && strings.Contains(phrase, query) {
			total += 250
			reasons["canonical-intent"] = struct{}{}
			break
		}
	}

	matched := make([]string, 0, len(reasons))
	for reason := range reasons {
		matched = append(matched, reason)
	}
	sort.Strings(matched)
	return total, matched
}

func examples(help string) []string {
	var result []string
	for _, line := range strings.Split(help, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "gplay ") {
			result = append(result, line)
		}
	}
	return result
}

func tokenMap(value string) tokenSet {
	result := tokenSet{}
	for _, token := range tokens(value) {
		result[token] = struct{}{}
	}
	return result
}

func tokens(value string) []string {
	return strings.FieldsFunc(strings.ToLower(value), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
}

func merge(destination, source tokenSet) {
	for token := range source {
		destination[token] = struct{}{}
	}
}

func hasPrefix(values tokenSet, prefix string) bool {
	if len(prefix) < 3 {
		return false
	}
	for value := range values {
		if strings.HasPrefix(value, prefix) {
			return true
		}
	}
	return false
}
