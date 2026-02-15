package shared

import (
	"fmt"
	"os"
	"strings"
)

const defaultOutputEnvVar = "GPLAY_DEFAULT_OUTPUT"

var validOutputFormats = map[string]bool{
	"json":     true,
	"table":    true,
	"markdown": true,
}

// ResolveOutputFormat returns the output format to use based on flag and env var.
// If flagValue is non-empty and not the default, it takes precedence.
// Otherwise falls back to GPLAY_DEFAULT_OUTPUT, then "json".
func ResolveOutputFormat(flagValue string, flagDefault string) string {
	// If the flag was explicitly set (different from default), use it
	if flagValue != "" && flagValue != flagDefault {
		return flagValue
	}

	// Check env var
	envVal := strings.ToLower(strings.TrimSpace(os.Getenv(defaultOutputEnvVar)))
	if envVal != "" {
		if validOutputFormats[envVal] {
			return envVal
		}
		fmt.Fprintf(os.Stderr, "Warning: invalid %s value %q, falling back to json\n", defaultOutputEnvVar, envVal)
	}

	// If flag was set to something (even the default), return it
	if flagValue != "" {
		return flagValue
	}

	return "json"
}
