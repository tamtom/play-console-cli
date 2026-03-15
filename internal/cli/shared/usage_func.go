package shared

import (
	"flag"
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/peterbourgon/ff/v3/ffcli"
)

// DefaultUsageFunc renders help output for any command with formatted sections.
func DefaultUsageFunc(cmd *ffcli.Command) string {
	var b strings.Builder

	// USAGE section
	if cmd.ShortUsage != "" {
		fmt.Fprintf(&b, "USAGE\n  %s\n\n", cmd.ShortUsage)
	}

	// DESCRIPTION
	if cmd.LongHelp != "" {
		fmt.Fprintf(&b, "%s\n\n", cmd.LongHelp)
	} else if cmd.ShortHelp != "" {
		fmt.Fprintf(&b, "%s\n\n", cmd.ShortHelp)
	}

	// SUBCOMMANDS section
	if len(cmd.Subcommands) > 0 {
		fmt.Fprintf(&b, "SUBCOMMANDS\n")
		tw := tabwriter.NewWriter(&b, 2, 4, 2, ' ', 0)
		for _, sub := range cmd.Subcommands {
			// Hide deprecated commands
			if strings.HasPrefix(sub.ShortHelp, "DEPRECATED:") {
				continue
			}
			fmt.Fprintf(tw, "  %s\t%s\n", sub.Name, sub.ShortHelp)
		}
		tw.Flush()
		fmt.Fprintln(&b)
	}

	// FLAGS section
	if cmd.FlagSet != nil {
		var hasFlags bool
		cmd.FlagSet.VisitAll(func(f *flag.Flag) { hasFlags = true })
		if hasFlags {
			fmt.Fprintf(&b, "FLAGS\n")
			tw := tabwriter.NewWriter(&b, 2, 4, 2, ' ', 0)
			cmd.FlagSet.VisitAll(func(f *flag.Flag) {
				def := f.DefValue
				if def != "" && def != "false" && def != "0" {
					fmt.Fprintf(tw, "  --%s\t%s (default: %s)\n", f.Name, f.Usage, def)
				} else {
					fmt.Fprintf(tw, "  --%s\t%s\n", f.Name, f.Usage)
				}
			})
			tw.Flush()
			fmt.Fprintln(&b)
		}
	}

	return b.String()
}
