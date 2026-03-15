package workflow

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

// interpolatePattern matches {{ .varname }} and {{ .varname | default "fallback" }}.
var interpolatePattern = regexp.MustCompile(`\{\{\s*\.(\w+)(?:\s*\|\s*default\s+"([^"]*)")?\s*\}\}`)

// Interpolate performs variable substitution on a template string.
// It supports {{ .varname }} syntax and {{ .VAR | default "fallback" }} for defaults.
// Returns an error if a required variable (no default) is undefined.
func Interpolate(template string, vars map[string]string) (string, error) {
	var errResult error

	result := interpolatePattern.ReplaceAllStringFunc(template, func(match string) string {
		if errResult != nil {
			return match
		}

		submatch := interpolatePattern.FindStringSubmatch(match)
		if len(submatch) < 2 {
			return match
		}

		varName := submatch[1]
		hasDefault := submatch[2] != "" || strings.Contains(match, `| default ""`)

		// Check vars map first, then environment.
		if value, ok := vars[varName]; ok {
			return value
		}
		if envValue := os.Getenv(varName); envValue != "" {
			return envValue
		}

		// Check for default value.
		if hasDefault {
			return submatch[2]
		}

		errResult = fmt.Errorf("undefined variable %q", varName)
		return match
	})

	if errResult != nil {
		return "", errResult
	}
	return result, nil
}
