package main

import (
	"fmt"
	"os"

	"github.com/tamtom/play-console-cli/cmd"
)

var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

func versionInfo() string {
	return fmt.Sprintf("%s (commit: %s, date: %s)", version, commit, date)
}

func main() {
	os.Exit(cmd.Run(os.Args[1:], versionInfo()))
}
