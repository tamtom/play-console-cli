package reviews

import (
	"context"
	"flag"
	"fmt"
	"strings"

	"github.com/peterbourgon/ff/v3/ffcli"
	"google.golang.org/api/androidpublisher/v3"

	"github.com/tamtom/play-console-cli/internal/cli/shared"
	"github.com/tamtom/play-console-cli/internal/playclient"
)

func ReviewsCommand() *ffcli.Command {
	fs := flag.NewFlagSet("reviews", flag.ExitOnError)
	return &ffcli.Command{
		Name:       "reviews",
		ShortUsage: "gplay reviews <subcommand> [flags]",
		ShortHelp:  "Manage app reviews.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			ListCommand(),
			GetCommand(),
			ReplyCommand(),
		},
		Exec: func(ctx context.Context, args []string) error {
			if len(args) == 0 {
				return flag.ErrHelp
			}
			return flag.ErrHelp
		},
	}
}

func ListCommand() *ffcli.Command {
	fs := flag.NewFlagSet("reviews list", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	startIndex := fs.Int64("start-index", 0, "Start index")
	maxResults := fs.Int64("max-results", 50, "Max results per page")
	translation := fs.String("translation-language", "", "Translation language (e.g. en-US)")
	paginate := fs.Bool("paginate", false, "Fetch all pages")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "list",
		ShortUsage: "gplay reviews list --package <name> [flags]",
		ShortHelp:  "List reviews.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			service, err := playclient.NewService(ctx)
			if err != nil {
				return err
			}
			pkg := shared.ResolvePackageName(*packageName, service.Cfg)
			if strings.TrimSpace(pkg) == "" {
				return fmt.Errorf("--package is required")
			}

			ctx, cancel := shared.ContextWithTimeout(ctx, service.Cfg)
			defer cancel()

			if !*paginate {
				call := service.API.Reviews.List(pkg).Context(ctx).MaxResults(*maxResults)
				if *startIndex > 0 {
					call.StartIndex(*startIndex)
				}
				if strings.TrimSpace(*translation) != "" {
					call.TranslationLanguage(*translation)
				}
				resp, err := call.Do()
				if err != nil {
					return err
				}
				return shared.PrintOutput(resp, *outputFlag, *pretty)
			}

			var all []*androidpublisher.Review
			index := *startIndex
			for {
				call := service.API.Reviews.List(pkg).Context(ctx).MaxResults(*maxResults).StartIndex(index)
				if strings.TrimSpace(*translation) != "" {
					call.TranslationLanguage(*translation)
				}
				resp, err := call.Do()
				if err != nil {
					return err
				}
				all = append(all, resp.Reviews...)
				if len(resp.Reviews) == 0 || int64(len(resp.Reviews)) < *maxResults {
					break
				}
				index += int64(len(resp.Reviews))
			}
			return shared.PrintOutput(all, *outputFlag, *pretty)
		},
	}
}

func GetCommand() *ffcli.Command {
	fs := flag.NewFlagSet("reviews get", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	reviewID := fs.String("review", "", "Review ID")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "get",
		ShortUsage: "gplay reviews get --package <name> --review <id>",
		ShortHelp:  "Get a review.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			if strings.TrimSpace(*reviewID) == "" {
				return fmt.Errorf("--review is required")
			}
			service, err := playclient.NewService(ctx)
			if err != nil {
				return err
			}
			pkg := shared.ResolvePackageName(*packageName, service.Cfg)
			if strings.TrimSpace(pkg) == "" {
				return fmt.Errorf("--package is required")
			}
			ctx, cancel := shared.ContextWithTimeout(ctx, service.Cfg)
			defer cancel()
			resp, err := service.API.Reviews.Get(pkg, *reviewID).Context(ctx).Do()
			if err != nil {
				return err
			}
			return shared.PrintOutput(resp, *outputFlag, *pretty)
		},
	}
}

func ReplyCommand() *ffcli.Command {
	fs := flag.NewFlagSet("reviews reply", flag.ExitOnError)
	packageName := fs.String("package", "", "Package name (applicationId)")
	reviewID := fs.String("review", "", "Review ID")
	replyText := fs.String("text", "", "Reply text")
	outputFlag := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "reply",
		ShortUsage: "gplay reviews reply --package <name> --review <id> --text <reply>",
		ShortHelp:  "Reply to a review.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := shared.ValidateOutputFlags(*outputFlag, *pretty); err != nil {
				return err
			}
			if strings.TrimSpace(*reviewID) == "" {
				return fmt.Errorf("--review is required")
			}
			if strings.TrimSpace(*replyText) == "" {
				return fmt.Errorf("--text is required")
			}
			service, err := playclient.NewService(ctx)
			if err != nil {
				return err
			}
			pkg := shared.ResolvePackageName(*packageName, service.Cfg)
			if strings.TrimSpace(pkg) == "" {
				return fmt.Errorf("--package is required")
			}
			ctx, cancel := shared.ContextWithTimeout(ctx, service.Cfg)
			defer cancel()
			req := &androidpublisher.ReviewsReplyRequest{ReplyText: *replyText}
			resp, err := service.API.Reviews.Reply(pkg, *reviewID, req).Context(ctx).Do()
			if err != nil {
				return err
			}
			return shared.PrintOutput(resp, *outputFlag, *pretty)
		},
	}
}
