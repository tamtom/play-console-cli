package vitals

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/tamtom/play-console-cli/internal/cli/shared"
	"github.com/tamtom/play-console-cli/internal/reportingclient"
)

var newMetricReportingService = reportingclient.NewService

var metricSetResources = map[string]string{
	"anr":                  "anrRateMetricSet",
	"crash":                "crashRateMetricSet",
	"error-count":          "errorCountMetricSet",
	"excessive-wakeup":     "excessiveWakeupRateMetricSet",
	"lmk":                  "lmkRateMetricSet",
	"slow-rendering":       "slowRenderingRateMetricSet",
	"slow-start":           "slowStartRateMetricSet",
	"stuck-wakelock":       "stuckBackgroundWakelockRateMetricSet",
	"anon-rss-swap-memory": "anonRssAndSwapMemoryUsageMetricSet",
	"bitmap-memory":        "bitmapMemoryUsageMetricSet",
}

func MetricSetsCommand() *ffcli.Command {
	fs := flag.NewFlagSet("vitals metric-sets", flag.ExitOnError)
	return &ffcli.Command{Name: "metric-sets", ShortUsage: "gplay vitals metric-sets <subcommand> [flags]", ShortHelp: "Describe or query any official Play Developer Reporting metric set.", FlagSet: fs, UsageFunc: shared.DefaultUsageFunc, Subcommands: []*ffcli.Command{MetricSetGetCommand(), MetricSetQueryCommand()}, Exec: func(context.Context, []string) error { return flag.ErrHelp }}
}

type metricFlags struct {
	packageName *string
	metricSet   *string
	output      *string
	pretty      *bool
}

func addMetricFlags(fs *flag.FlagSet) metricFlags {
	return metricFlags{
		packageName: fs.String("package", "", "Package name (applicationId)"),
		metricSet:   fs.String("metric-set", "", "Metric set: anr, crash, error-count, excessive-wakeup, lmk, slow-rendering, slow-start, stuck-wakelock, anon-rss-swap-memory, bitmap-memory"),
		output:      fs.String("output", "json", "Output format: json (default), table, markdown"),
		pretty:      fs.Bool("pretty", false, "Pretty-print JSON output"),
	}
}

func (f metricFlags) validate() (string, error) {
	if err := shared.ValidateOutputFlags(*f.output, *f.pretty); err != nil {
		return "", err
	}
	resource, ok := metricSetResources[strings.ToLower(strings.TrimSpace(*f.metricSet))]
	if !ok {
		return "", fmt.Errorf("--metric-set must be one of the documented values")
	}
	return resource, nil
}

func MetricSetGetCommand() *ffcli.Command {
	fs := flag.NewFlagSet("vitals metric-sets get", flag.ExitOnError)
	f := addMetricFlags(fs)
	return &ffcli.Command{
		Name: "get", ShortUsage: "gplay vitals metric-sets get --package <pkg> --metric-set <name>", ShortHelp: "Get dimensions, metrics, and freshness metadata for a metric set.", FlagSet: fs, UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			resource, err := f.validate()
			if err != nil {
				return err
			}
			service, err := newMetricReportingService(ctx)
			if err != nil {
				return err
			}
			pkg := shared.ResolvePackageName(*f.packageName, service.Cfg)
			if strings.TrimSpace(pkg) == "" {
				return fmt.Errorf("--package is required")
			}
			ctx, cancel := shared.ContextWithTimeout(ctx, service.Cfg)
			defer cancel()
			result, err := reportingJSON(ctx, service, http.MethodGet, metricSetURL(service, pkg, resource), nil)
			if err != nil {
				return err
			}
			return shared.PrintOutput(result, *f.output, *f.pretty)
		},
	}
}

func MetricSetQueryCommand() *ffcli.Command {
	fs := flag.NewFlagSet("vitals metric-sets query", flag.ExitOnError)
	f := addMetricFlags(fs)
	jsonArg := fs.String("json", "", "Official metric-set query request JSON (or @file)")
	return &ffcli.Command{
		Name: "query", ShortUsage: "gplay vitals metric-sets query --package <pkg> --metric-set <name> --json @query.json", ShortHelp: "Query any official metric set with its typed REST request body.", FlagSet: fs, UsageFunc: shared.DefaultUsageFunc,
		LongHelp: `Query any current Play Developer Reporting metric set using its official request schema.

JSON example: {"metrics":["distinctUsers"],"timelineSpec":{"aggregationPeriod":"DAILY"},"pageSize":1000}`,
		Exec: func(ctx context.Context, _ []string) error {
			resource, err := f.validate()
			if err != nil {
				return err
			}
			if strings.TrimSpace(*jsonArg) == "" {
				return fmt.Errorf("--json is required")
			}
			var body map[string]any
			if err := shared.LoadJSONArg(*jsonArg, &body); err != nil {
				return fmt.Errorf("invalid --json: %w", err)
			}
			service, err := newMetricReportingService(ctx)
			if err != nil {
				return err
			}
			pkg := shared.ResolvePackageName(*f.packageName, service.Cfg)
			if strings.TrimSpace(pkg) == "" {
				return fmt.Errorf("--package is required")
			}
			ctx, cancel := shared.ContextWithTimeout(ctx, service.Cfg)
			defer cancel()
			result, err := reportingJSON(ctx, service, http.MethodPost, metricSetURL(service, pkg, resource)+":query", body)
			if err != nil {
				return err
			}
			return shared.PrintOutput(result, *f.output, *f.pretty)
		},
	}
}

func ReleaseFiltersCommand() *ffcli.Command {
	fs := flag.NewFlagSet("vitals release-filters", flag.ExitOnError)
	pkgFlag := fs.String("package", "", "Package name (applicationId)")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")
	return &ffcli.Command{
		Name: "release-filters", ShortUsage: "gplay vitals release-filters --package <pkg>", ShortHelp: "Fetch valid release filters for reporting queries.", FlagSet: fs, UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			service, err := newMetricReportingService(ctx)
			if err != nil {
				return err
			}
			pkg := shared.ResolvePackageName(*pkgFlag, service.Cfg)
			if strings.TrimSpace(pkg) == "" {
				return fmt.Errorf("--package is required")
			}
			ctx, cancel := shared.ContextWithTimeout(ctx, service.Cfg)
			defer cancel()
			endpoint := strings.TrimRight(service.BasePath, "/") + "/v1beta1/apps/" + url.PathEscape(pkg) + ":fetchReleaseFilterOptions"
			result, err := reportingJSON(ctx, service, http.MethodGet, endpoint, nil)
			if err != nil {
				return err
			}
			return shared.PrintOutput(result, *outputFlag, *pretty)
		},
	}
}

func metricSetURL(service *reportingclient.Service, pkg, resource string) string {
	return strings.TrimRight(service.BasePath, "/") + "/v1beta1/apps/" + url.PathEscape(pkg) + "/" + resource
}

func reportingJSON(ctx context.Context, service *reportingclient.Service, method, endpoint string, body any) (any, error) {
	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reader = bytes.NewReader(encoded)
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint, reader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	res, err := service.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	responseBody, err := io.ReadAll(io.LimitReader(res.Body, 16<<20))
	if err != nil {
		return nil, err
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("reporting request failed: %s: %s", res.Status, strings.TrimSpace(string(responseBody)))
	}
	var result any
	if err := json.Unmarshal(responseBody, &result); err != nil {
		return nil, fmt.Errorf("decode reporting response: %w", err)
	}
	return result, nil
}
