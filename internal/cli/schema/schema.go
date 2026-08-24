// Package schema exposes embedded official Google Play discovery contracts.
package schema

import (
	"context"
	"encoding/json"
	"flag"
	"strconv"
	"strings"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/tamtom/play-console-cli/internal/apischema"
	"github.com/tamtom/play-console-cli/internal/cli/shared"
	"github.com/tamtom/play-console-cli/internal/output"
)

// EndpointResponse is the machine-readable endpoint query result.
type EndpointResponse struct {
	Query   string               `json:"query,omitempty"`
	API     string               `json:"api,omitempty"`
	Method  string               `json:"method,omitempty"`
	Count   int                  `json:"count"`
	Results []apischema.Endpoint `json:"results"`
}

// TypeResponse is the machine-readable discovery-type query result.
type TypeResponse struct {
	Query   string           `json:"query,omitempty"`
	API     string           `json:"api,omitempty"`
	Count   int              `json:"count"`
	Results []apischema.Type `json:"results"`
}

func init() {
	output.RegisterType(EndpointResponse{}, []string{"API", "METHOD", "ID", "PATH", "REQUEST", "RESPONSE", "PARAMS"}, func(data any) [][]string {
		response := data.(EndpointResponse)
		rows := make([][]string, 0, len(response.Results))
		for _, endpoint := range response.Results {
			rows = append(rows, []string{
				endpoint.API,
				endpoint.HTTPMethod,
				endpoint.ID,
				endpoint.Path,
				endpoint.RequestType,
				endpoint.ResponseType,
				strconv.Itoa(len(endpoint.Parameters)),
			})
		}
		return rows
	})
	output.RegisterType(TypeResponse{}, []string{"API", "TYPE", "KIND", "DESCRIPTION"}, func(data any) [][]string {
		response := data.(TypeResponse)
		rows := make([][]string, 0, len(response.Results))
		for _, item := range response.Results {
			var details struct {
				Type        string `json:"type"`
				Description string `json:"description"`
			}
			_ = json.Unmarshal(item.Definition, &details)
			rows = append(rows, []string{item.API, item.Name, details.Type, details.Description})
		}
		return rows
	})
}

// Command returns the credential-free official API schema command.
func Command() *ffcli.Command {
	fs := flag.NewFlagSet("schema", flag.ExitOnError)
	api := fs.String("api", "", "Filter by API name")
	method := fs.String("method", "", "Filter endpoints by HTTP method")
	typeName := fs.String("type", "", "Inspect a request or response type by name")
	listEndpoints := fs.Bool("list", false, "List all matching endpoints without a query")
	listTypes := fs.Bool("list-types", false, "List all matching discovery types")
	outputFlags := shared.BindOutputFlags(fs)

	return &ffcli.Command{
		Name:       "schema",
		ShortUsage: "gplay schema [flags] [query]",
		ShortHelp:  "Inspect embedded official Google Play API endpoint and type schemas.",
		LongHelp: `Inspect the reviewed official Google Play discovery documents locally.

Endpoint results include the API and method ID, HTTP transport, path/query
parameters, request and response types, OAuth scopes, and media-upload paths.
Type results preserve Google's complete discovery definition, including nested
properties, refs, enums, formats, descriptions, and constraints.

This command never authenticates, contacts Google, or changes an account.

Examples:
  gplay schema androidpublisher.orders.batchget
  gplay schema --api playintegrity --method POST decodeIntegrityToken
  gplay schema --api androidpublisher --type OrdersReviewRefundRequest
  gplay schema --api checks --list --output table
  gplay schema --api playdeveloperreporting --list-types`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			_ = ctx
			if err := shared.ValidateOutputFlags(outputFlags.Format(), outputFlags.IsPretty()); err != nil {
				return err
			}
			if *listEndpoints && *listTypes {
				return shared.UsageError("--list and --list-types cannot be used together")
			}
			if strings.TrimSpace(*typeName) != "" && (*listEndpoints || *listTypes) {
				return shared.UsageError("--type cannot be combined with --list or --list-types")
			}
			if strings.TrimSpace(*typeName) != "" && strings.TrimSpace(*method) != "" {
				return shared.UsageError("--method cannot be used with --type")
			}

			index, err := apischema.Load()
			if err != nil {
				return err
			}
			if strings.TrimSpace(*typeName) != "" || *listTypes {
				if len(args) != 0 {
					return shared.UsageError("unexpected arguments with type schema query")
				}
				query := strings.TrimSpace(*typeName)
				items, err := index.FindTypes(apischema.Filter{API: *api, Query: query})
				if err != nil {
					return err
				}
				response := TypeResponse{Query: query, API: strings.TrimSpace(*api), Count: len(items), Results: items}
				return shared.PrintOutputContext(ctx, response, outputFlags.Format(), outputFlags.IsPretty())
			}

			query := strings.TrimSpace(strings.Join(args, " "))
			if query == "" && !*listEndpoints {
				return shared.UsageError("schema query is required (or use --list or --list-types)")
			}
			if *listEndpoints && query != "" {
				return shared.UsageError("unexpected query with --list")
			}
			items, err := index.FindEndpoints(apischema.Filter{API: *api, Method: *method, Query: query})
			if err != nil {
				return err
			}
			response := EndpointResponse{
				Query:   query,
				API:     strings.TrimSpace(*api),
				Method:  strings.ToUpper(strings.TrimSpace(*method)),
				Count:   len(items),
				Results: items,
			}
			return shared.PrintOutputContext(ctx, response, outputFlags.Format(), outputFlags.IsPretty())
		},
	}
}
