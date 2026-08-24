package main

import (
	"fmt"
	"os"

	"github.com/tamtom/play-console-cli/cmd"
	buildversion "github.com/tamtom/play-console-cli/internal/version"
)

func versionInfo() string {
	return fmt.Sprintf("%s (commit: %s, date: %s)", buildversion.Version, buildversion.Commit, buildversion.BuildDate)
}

func main() {
	os.Exit(cmd.Run(os.Args[1:], versionInfo()))
}
