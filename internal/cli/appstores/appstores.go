// Package appstores exposes the official Android Publisher APIs for registered
// third-party app stores. These commands are intentionally separate from the
// normal Google Play developer workflow.
package appstores

import (
	"context"
	"flag"
	"fmt"
	"mime"
	"os"
	"path/filepath"
	"strings"

	"github.com/peterbourgon/ff/v3/ffcli"
	"google.golang.org/api/androidpublisher/v3"
	"google.golang.org/api/googleapi"

	"github.com/tamtom/play-console-cli/internal/cli/shared"
	"github.com/tamtom/play-console-cli/internal/playclient"
)

var newPlayService = playclient.NewService

type commonFlags struct {
	storePackage *string
	registered   *bool
	output       *string
	pretty       *bool
}

func addCommonFlags(fs *flag.FlagSet) commonFlags {
	return commonFlags{
		storePackage: fs.String("app-store-package", "", "Package name of the registered third-party app store"),
		registered:   fs.Bool("registered-third-party-store", false, "Acknowledge that this account is enrolled in Google's third-party app-store program"),
		output:       fs.String("output", "json", "Output format: json (default), table, markdown"),
		pretty:       fs.Bool("pretty", false, "Pretty-print JSON output"),
	}
}

func (f commonFlags) validate() error {
	if err := shared.ValidateOutputFlags(*f.output, *f.pretty); err != nil {
		return err
	}
	if strings.TrimSpace(*f.storePackage) == "" {
		return fmt.Errorf("--app-store-package is required")
	}
	if !*f.registered {
		return fmt.Errorf("--registered-third-party-store is required; these APIs are not for ordinary Google Play apps")
	}
	return nil
}

func Command() *ffcli.Command {
	fs := flag.NewFlagSet("app-stores", flag.ExitOnError)
	return &ffcli.Command{
		Name:       "app-stores",
		ShortUsage: "gplay app-stores <subcommand> [flags]",
		ShortHelp:  "Operate official APIs for registered third-party Android app stores.",
		LongHelp: `Operate the Android Publisher App Store Review and Play Catalog APIs.

This namespace is only for organizations registered in Google's third-party
app-store program. It does not create or manage a normal Google Play listing.
Every command requires --registered-third-party-store to prevent accidental use.`,
		FlagSet: fs, UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			CreateAppCommand(), UpdateAppCommand(), PublishStatusCommand(),
			UploadAPKCommand(), UploadPolicyFileCommand(), UploadImageCommand(),
			RecentAppViewCommand(), RecentUpdateEventsCommand(),
		},
		Exec: func(context.Context, []string) error { return flag.ErrHelp },
	}
}

func CreateAppCommand() *ffcli.Command {
	fs := flag.NewFlagSet("app-stores create-app", flag.ExitOnError)
	c := addCommonFlags(fs)
	pkg := fs.String("package", "", "Package name of the hosted app")
	confirm := fs.Bool("confirm", false, "Confirm creation of the hosted app record")
	return &ffcli.Command{
		Name: "create-app", ShortUsage: "gplay app-stores create-app --app-store-package <pkg> --package <pkg> --registered-third-party-store --confirm", ShortHelp: "Create a third-party-app-store hosted app record.", FlagSet: fs, UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			if err := c.validate(); err != nil {
				return err
			}
			if strings.TrimSpace(*pkg) == "" {
				return fmt.Errorf("--package is required")
			}
			if !*confirm {
				return fmt.Errorf("--confirm is required")
			}
			s, err := newPlayService(ctx)
			if err != nil {
				return err
			}
			ctx, cancel := shared.ContextWithTimeout(ctx, s.Cfg)
			defer cancel()
			_, err = s.API.Appstoreappsreview.Createappstorehostedapp(*c.storePackage, &androidpublisher.CreateAppStoreHostedAppRequest{PackageName: *pkg}).Context(ctx).Do()
			if err != nil {
				return err
			}
			return shared.PrintOutputContext(ctx, map[string]any{"created": true, "appStorePackageName": *c.storePackage, "packageName": *pkg}, *c.output, *c.pretty)
		},
	}
}

func UpdateAppCommand() *ffcli.Command {
	fs := flag.NewFlagSet("app-stores update-app", flag.ExitOnError)
	c := addCommonFlags(fs)
	jsonArg := fs.String("json", "", "UpdateAppStoreHostedAppRequest JSON (or @file)")
	confirm := fs.Bool("confirm", false, "Confirm submission for immediate review")
	return &ffcli.Command{
		Name: "update-app", ShortUsage: "gplay app-stores update-app --app-store-package <pkg> --json @app.json --registered-third-party-store --confirm", ShortHelp: "Update and submit a hosted app for review.", FlagSet: fs, UsageFunc: shared.DefaultUsageFunc,
		LongHelp: `Update a registered third-party-store hosted app and submit it for review.

JSON example: {"packageName":"app.example","activeApks":{},"activeLocalizedStoreListings":[],"appDetails":{},"policyDeclarations":[]}`,
		Exec: func(ctx context.Context, _ []string) error {
			if err := c.validate(); err != nil {
				return err
			}
			if strings.TrimSpace(*jsonArg) == "" {
				return fmt.Errorf("--json is required")
			}
			if !*confirm {
				return fmt.Errorf("--confirm is required")
			}
			var req androidpublisher.UpdateAppStoreHostedAppRequest
			if err := shared.LoadJSONArg(*jsonArg, &req); err != nil {
				return fmt.Errorf("invalid --json: %w", err)
			}
			if strings.TrimSpace(req.PackageName) == "" {
				return fmt.Errorf("packageName is required in --json")
			}
			s, err := newPlayService(ctx)
			if err != nil {
				return err
			}
			ctx, cancel := shared.ContextWithTimeout(ctx, s.Cfg)
			defer cancel()
			_, err = s.API.Appstoreappsreview.Updateappstorehostedapp(*c.storePackage, &req).Context(ctx).Do()
			if err != nil {
				return err
			}
			return shared.PrintOutputContext(ctx, map[string]any{"updated": true, "appStorePackageName": *c.storePackage, "packageName": req.PackageName}, *c.output, *c.pretty)
		},
	}
}

func PublishStatusCommand() *ffcli.Command {
	fs := flag.NewFlagSet("app-stores publish-status", flag.ExitOnError)
	c := addCommonFlags(fs)
	pkg := fs.String("package", "", "Package name of the hosted app")
	state := fs.String("state", "", "Publish state: PUBLISHED or UNPUBLISHED")
	confirm := fs.Bool("confirm", false, "Confirm the publish-state change")
	return &ffcli.Command{
		Name: "publish-status", ShortUsage: "gplay app-stores publish-status --app-store-package <pkg> --package <pkg> --state <state> --registered-third-party-store --confirm", ShortHelp: "Update a hosted app's publish status.", FlagSet: fs, UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			if err := c.validate(); err != nil {
				return err
			}
			if strings.TrimSpace(*pkg) == "" {
				return fmt.Errorf("--package is required")
			}
			stateValue := strings.ToUpper(strings.TrimSpace(*state))
			if stateValue != "PUBLISHED" && stateValue != "UNPUBLISHED" {
				return fmt.Errorf("--state must be PUBLISHED or UNPUBLISHED")
			}
			if !*confirm {
				return fmt.Errorf("--confirm is required")
			}
			s, err := newPlayService(ctx)
			if err != nil {
				return err
			}
			ctx, cancel := shared.ContextWithTimeout(ctx, s.Cfg)
			defer cancel()
			req := &androidpublisher.UpdateAppStoreHostedAppPublishStatusRequest{PublishState: "APP_STORE_APP_PUBLISH_STATE_" + stateValue}
			_, err = s.API.Appstoreappsreview.Updateappstorehostedapppublishstatus(*c.storePackage, *pkg, req).Context(ctx).Do()
			if err != nil {
				return err
			}
			return shared.PrintOutputContext(ctx, map[string]any{"updated": true, "packageName": *pkg, "publishState": stateValue}, *c.output, *c.pretty)
		},
	}
}

type uploadKind int

const (
	uploadAPK uploadKind = iota
	uploadPolicy
	uploadImage
)

func uploadCommand(kind uploadKind) *ffcli.Command {
	names := []string{"upload-apk", "upload-policy-file", "upload-image"}
	name := names[kind]
	fs := flag.NewFlagSet("app-stores "+name, flag.ExitOnError)
	c := addCommonFlags(fs)
	pkg := fs.String("package", "", "Package name of the hosted app")
	filePath := fs.String("file", "", "File to upload")
	confirm := fs.Bool("confirm", false, "Confirm upload to the third-party app-store review service")
	return &ffcli.Command{
		Name: name, ShortUsage: "gplay app-stores " + name + " --app-store-package <pkg> --package <pkg> --file <path> --registered-third-party-store --confirm", ShortHelp: "Upload review media for a hosted app.", FlagSet: fs, UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			if err := c.validate(); err != nil {
				return err
			}
			if strings.TrimSpace(*pkg) == "" {
				return fmt.Errorf("--package is required")
			}
			if strings.TrimSpace(*filePath) == "" {
				return fmt.Errorf("--file is required")
			}
			if !*confirm {
				return fmt.Errorf("--confirm is required")
			}
			mediaType, err := uploadMediaType(kind, *filePath)
			if err != nil {
				return err
			}
			file, err := os.Open(*filePath)
			if err != nil {
				return fmt.Errorf("open upload file: %w", err)
			}
			defer file.Close()
			s, err := newPlayService(ctx)
			if err != nil {
				return err
			}
			ctx, cancel := shared.ContextWithUploadTimeout(ctx, s.Cfg)
			defer cancel()
			var result any
			switch kind {
			case uploadAPK:
				result, err = s.API.Appstoreappsreview.Uploadapk(*c.storePackage, *pkg, &androidpublisher.UploadApkRequest{}).Media(file, googleapi.ContentType(mediaType)).Context(ctx).Do()
			case uploadPolicy:
				result, err = s.API.Appstoreappsreview.Uploadappstoreapppolicydeclarationfile(*c.storePackage, *pkg, &androidpublisher.UploadAppStoreAppPolicyDeclarationFileRequest{FileType: "DECLARATION_FILE_TYPE_DOCUMENT"}).Media(file, googleapi.ContentType(mediaType)).Context(ctx).Do()
			case uploadImage:
				result, err = s.API.Appstoreappsreview.Uploadimage(*c.storePackage, *pkg, &androidpublisher.UploadImageRequest{}).Media(file, googleapi.ContentType(mediaType)).Context(ctx).Do()
			}
			if err != nil {
				return err
			}
			return shared.PrintOutputContext(ctx, result, *c.output, *c.pretty)
		},
	}
}

func UploadAPKCommand() *ffcli.Command        { return uploadCommand(uploadAPK) }
func UploadPolicyFileCommand() *ffcli.Command { return uploadCommand(uploadPolicy) }
func UploadImageCommand() *ffcli.Command      { return uploadCommand(uploadImage) }

func uploadMediaType(kind uploadKind, path string) (string, error) {
	ext := strings.ToLower(filepath.Ext(path))
	if kind == uploadAPK {
		if ext != ".apk" {
			return "", fmt.Errorf("--file must have an .apk extension")
		}
		return "application/vnd.android.package-archive", nil
	}
	allowed := map[string]string{".png": "image/png", ".jpg": "image/jpeg", ".jpeg": "image/jpeg"}
	if kind == uploadPolicy {
		allowed[".pdf"] = "application/pdf"
	}
	if mediaType, ok := allowed[ext]; ok {
		return mediaType, nil
	}
	if detected := mime.TypeByExtension(ext); detected != "" {
		return "", fmt.Errorf("unsupported upload media type %q", detected)
	}
	return "", fmt.Errorf("unsupported file extension %q", ext)
}

func RecentAppViewCommand() *ffcli.Command {
	fs := flag.NewFlagSet("app-stores recent-app-view", flag.ExitOnError)
	c := addCommonFlags(fs)
	pkg := fs.String("package", "", "Google Play app package name")
	return &ffcli.Command{
		Name: "recent-app-view", ShortUsage: "gplay app-stores recent-app-view --app-store-package <pkg> --package <pkg> --registered-third-party-store", ShortHelp: "Get a recent Play Catalog app view.", FlagSet: fs, UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			if err := c.validate(); err != nil {
				return err
			}
			if strings.TrimSpace(*pkg) == "" {
				return fmt.Errorf("--package is required")
			}
			s, err := newPlayService(ctx)
			if err != nil {
				return err
			}
			ctx, cancel := shared.ContextWithTimeout(ctx, s.Cfg)
			defer cancel()
			result, err := s.API.Appstorecatalog.Recentappviews.Get(*c.storePackage, *pkg).Context(ctx).Do()
			if err != nil {
				return err
			}
			return shared.PrintOutputContext(ctx, result, *c.output, *c.pretty)
		},
	}
}

func RecentUpdateEventsCommand() *ffcli.Command {
	fs := flag.NewFlagSet("app-stores recent-update-events", flag.ExitOnError)
	c := addCommonFlags(fs)
	start := fs.String("start-time", "", "Inclusive RFC3339 start time")
	end := fs.String("end-time", "", "Exclusive RFC3339 end time")
	pageSize := fs.Int64("page-size", 0, "Maximum events per page")
	pageToken := fs.String("page-token", "", "Continuation token")
	paginate := fs.Bool("paginate", false, "Fetch every page")
	return &ffcli.Command{
		Name: "recent-update-events", ShortUsage: "gplay app-stores recent-update-events --app-store-package <pkg> --registered-third-party-store [flags]", ShortHelp: "List recent Play Catalog update events.", FlagSet: fs, UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, _ []string) error {
			if err := c.validate(); err != nil {
				return err
			}
			s, err := newPlayService(ctx)
			if err != nil {
				return err
			}
			ctx, cancel := shared.ContextWithTimeout(ctx, s.Cfg)
			defer cancel()
			call := s.API.Appstorecatalog.Recentupdateevents.List(*c.storePackage).Context(ctx)
			if *start != "" {
				call.StartTime(*start)
			}
			if *end != "" {
				call.EndTime(*end)
			}
			if *pageSize > 0 {
				call.PageSize(*pageSize)
			}
			if *pageToken != "" {
				call.PageToken(*pageToken)
			}
			if !*paginate {
				result, err := call.Do()
				if err != nil {
					return err
				}
				return shared.PrintOutputContext(ctx, result, *c.output, *c.pretty)
			}
			var events []*androidpublisher.RecentUpdateEvent
			err = call.Pages(ctx, func(page *androidpublisher.ListRecentUpdateEventsResponse) error {
				events = append(events, page.RecentUpdateEvents...)
				return nil
			})
			if err != nil {
				return err
			}
			return shared.PrintOutputContext(ctx, events, *c.output, *c.pretty)
		},
	}
}
