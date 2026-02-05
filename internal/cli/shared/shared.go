package shared

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/tamtom/play-console-cli/internal/config"
	"github.com/tamtom/play-console-cli/internal/output"
)

const (
	strictAuthEnvVar = "GPLAY_STRICT_AUTH"
	profileEnvVar    = "GPLAY_PROFILE"
	packageEnvVar    = "GPLAY_PACKAGE_NAME"
	timeoutEnvVar    = "GPLAY_TIMEOUT"
	timeoutSecondsEnvVar = "GPLAY_TIMEOUT_SECONDS"
	uploadTimeoutEnvVar = "GPLAY_UPLOAD_TIMEOUT"
	uploadTimeoutSecondsEnvVar = "GPLAY_UPLOAD_TIMEOUT_SECONDS"
)

// DefaultUsageFunc matches ffcli's default usage format.
func DefaultUsageFunc(cmd *ffcli.Command) string {
	return ffcli.DefaultUsageFunc(cmd)
}

// PrintOutput renders output in the requested format.
func PrintOutput(data interface{}, format string, pretty bool) error {
	format = strings.ToLower(strings.TrimSpace(format))
	switch format {
	case "json", "":
		if pretty {
			return output.PrintPrettyJSON(data)
		}
		return output.PrintJSON(data)
	case "markdown", "md":
		if pretty {
			return fmt.Errorf("--pretty is only valid with JSON output")
		}
		return output.PrintMarkdown(data)
	case "table":
		if pretty {
			return fmt.Errorf("--pretty is only valid with JSON output")
		}
		return output.PrintTable(data)
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
}

// ResolveProfileName returns the selected profile name.
func ResolveProfileName(cfg *config.Config) string {
	if env := strings.TrimSpace(os.Getenv(profileEnvVar)); env != "" {
		return env
	}
	if cfg != nil && strings.TrimSpace(cfg.DefaultProfile) != "" {
		return strings.TrimSpace(cfg.DefaultProfile)
	}
	if cfg != nil && len(cfg.Profiles) == 1 {
		return cfg.Profiles[0].Name
	}
	return ""
}

// ResolvePackageName returns a package name from flags/env/config.
func ResolvePackageName(flagValue string, cfg *config.Config) string {
	if strings.TrimSpace(flagValue) != "" {
		return strings.TrimSpace(flagValue)
	}
	if env := strings.TrimSpace(os.Getenv(packageEnvVar)); env != "" {
		return env
	}
	if cfg != nil && strings.TrimSpace(cfg.PackageName) != "" {
		return strings.TrimSpace(cfg.PackageName)
	}
	return ""
}

func StrictAuthEnabled() bool {
	value := strings.TrimSpace(os.Getenv(strictAuthEnvVar))
	if value == "" {
		return false
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false
	}
	return parsed
}

func ParseTimeouts(cfg *config.Config) (time.Duration, time.Duration) {
	return parseTimeout(timeoutEnvVar, timeoutSecondsEnvVar, cfg.Timeout, cfg.TimeoutSeconds),
		parseTimeout(uploadTimeoutEnvVar, uploadTimeoutSecondsEnvVar, cfg.UploadTimeout, cfg.UploadTimeoutSeconds)
}

func parseTimeout(envVar, envSecondsVar string, value config.DurationValue, secondsValue config.DurationValue) time.Duration {
	if env := strings.TrimSpace(os.Getenv(envVar)); env != "" {
		if parsed, err := time.ParseDuration(env); err == nil {
			return parsed
		}
	}
	if env := strings.TrimSpace(os.Getenv(envSecondsVar)); env != "" {
		if parsed, err := strconv.Atoi(env); err == nil {
			return time.Duration(parsed) * time.Second
		}
	}
	if v, ok := value.Value(); ok {
		return v
	}
	if v, ok := secondsValue.Value(); ok {
		return v
	}
	return 0
}

// ContextWithTimeout applies request timeouts.
func ContextWithTimeout(ctx context.Context, cfg *config.Config) (context.Context, context.CancelFunc) {
	requestTimeout, _ := ParseTimeouts(cfg)
	if requestTimeout <= 0 {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, requestTimeout)
}

// ContextWithUploadTimeout applies upload timeouts.
func ContextWithUploadTimeout(ctx context.Context, cfg *config.Config) (context.Context, context.CancelFunc) {
	_, uploadTimeout := ParseTimeouts(cfg)
	if uploadTimeout <= 0 {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, uploadTimeout)
}

// RequireFlags ensures the required flags are provided.
func RequireFlags(flagSet *flag.FlagSet, required ...string) error {
	var missing []string
	for _, name := range required {
		flag := flagSet.Lookup(name)
		if flag == nil {
			missing = append(missing, name)
			continue
		}
		if strings.TrimSpace(flag.Value.String()) == "" {
			missing = append(missing, name)
		}
	}
	if len(missing) == 0 {
		return nil
	}
	return fmt.Errorf("missing required flags: %s", strings.Join(missing, ", "))
}

// ValidateOutputFlags enforces output/pretty compatibility.
func ValidateOutputFlags(output string, pretty bool) error {
	normalized := strings.ToLower(strings.TrimSpace(output))
	if (normalized == "table" || normalized == "markdown" || normalized == "md") && pretty {
		return fmt.Errorf("--pretty is only valid with JSON output")
	}
	return nil
}

// ReportedError wraps errors that already have user-facing output.
type ReportedError struct{ Err error }

func (r ReportedError) Error() string { return r.Err.Error() }
func (r ReportedError) Unwrap() error { return r.Err }

// NewReportedError wraps err as ReportedError.
func NewReportedError(err error) error {
	if err == nil {
		return nil
	}
	return ReportedError{Err: err}
}

// IsReportedError returns true if err is a ReportedError.
func IsReportedError(err error) bool {
	var r ReportedError
	return errors.As(err, &r)
}
