// Package androidtools provides optional local Android build, signing, and
// screenshot helpers. It has no Google Play network surface.
package androidtools

import (
	"context"
	"flag"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/tamtom/play-console-cli/internal/cli/shared"
)

// AndroidCommand returns the local Android toolchain command group.
func AndroidCommand() *ffcli.Command {
	fs := flag.NewFlagSet("android", flag.ExitOnError)
	return &ffcli.Command{
		Name:       "android",
		ShortUsage: "gplay android <build|signing|screenshots> [flags]",
		ShortHelp:  "Run optional local Android build, signing, and screenshot helpers.",
		LongHelp: `Run local Android toolchain helpers without contacting Google Play.

Gradle wrappers, Android SDK tools, JDK signing tools, and adb are invoked
directly without a shell. Normal gplay API commands never depend on these
tools.`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			BuildCommand(),
			SigningCommand(),
			ScreenshotsCommand(),
		},
		Exec: func(context.Context, []string) error { return flag.ErrHelp },
	}
}
