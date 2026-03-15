package shared

import (
	"flag"
	"os"

	"golang.org/x/term"
)

// OutputFlags holds the parsed output format flags.
type OutputFlags struct {
	Output *string
	Pretty *bool
}

// BindOutputFlags registers --output and --pretty flags on the given FlagSet.
// The default for --output is TTY-aware: "table" if stdout is a terminal, "json" otherwise.
// The GPLAY_DEFAULT_OUTPUT env var overrides the default.
func BindOutputFlags(fs *flag.FlagSet) *OutputFlags {
	defaultFormat := defaultOutputFormat()
	output := fs.String("output", defaultFormat, "Output format: json, table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")
	return &OutputFlags{Output: output, Pretty: pretty}
}

// Format returns the resolved output format string.
func (o *OutputFlags) Format() string {
	if o.Output == nil {
		return "json"
	}
	return *o.Output
}

// IsPretty returns whether pretty-printing is enabled.
func (o *OutputFlags) IsPretty() bool {
	if o.Pretty == nil {
		return false
	}
	return *o.Pretty
}

func defaultOutputFormat() string {
	if env := os.Getenv("GPLAY_DEFAULT_OUTPUT"); env != "" {
		return env
	}
	if term.IsTerminal(int(os.Stdout.Fd())) {
		return "table"
	}
	return "json"
}
