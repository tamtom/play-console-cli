package androidtools

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/tamtom/play-console-cli/internal/cli/shared"
	"github.com/tamtom/play-console-cli/internal/localexec"
)

type signingResult struct {
	Mode     string        `json:"mode"`
	Tool     string        `json:"tool"`
	Args     []string      `json:"args"`
	Artifact localArtifact `json:"artifact"`
	Verified bool          `json:"verified"`
	Details  string        `json:"details,omitempty"`
}

// SigningCommand returns local signing inspection commands.
func SigningCommand() *ffcli.Command {
	fs := flag.NewFlagSet("android signing", flag.ExitOnError)
	return &ffcli.Command{
		Name:        "signing",
		ShortUsage:  "gplay android signing <verify|inspect> [flags]",
		ShortHelp:   "Verify release signatures and inspect upload certificates locally.",
		FlagSet:     fs,
		UsageFunc:   shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{VerifySignatureCommand(), InspectKeystoreCommand()},
		Exec:        func(context.Context, []string) error { return flag.ErrHelp },
	}
}

// VerifySignatureCommand verifies APK/AAB signing with local SDK/JDK tools.
func VerifySignatureCommand() *ffcli.Command {
	fs := flag.NewFlagSet("android signing verify", flag.ExitOnError)
	file := fs.String("file", "", "AAB or APK to verify")
	output := fs.String("output", "json", "Output format: json, table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")
	return &ffcli.Command{
		Name:       "verify",
		ShortUsage: "gplay android signing verify --file <app.aab|app.apk>",
		ShortHelp:  "Verify an APK/AAB signature with apksigner or jarsigner.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if len(args) != 0 || strings.TrimSpace(*file) == "" {
				return shared.UsageError("--file is required and positional arguments are not accepted")
			}
			if err := shared.ValidateOutputFlags(*output, *pretty); err != nil {
				return err
			}
			result, err := verifySignature(ctx, *file)
			if err != nil {
				return err
			}
			return shared.PrintOutputContext(ctx, result, *output, *pretty)
		},
	}
}

func verifySignature(ctx context.Context, path string) (signingResult, error) {
	artifact, err := inspectLocalArtifact(strings.TrimSpace(path))
	if err != nil {
		return signingResult{}, fmt.Errorf("inspect signed artifact: %w", err)
	}
	extension := strings.ToLower(filepath.Ext(artifact.Path))
	var toolName string
	var args []string
	switch extension {
	case ".apk":
		toolName = "apksigner"
		args = []string{"verify", "--verbose", "--print-certs", artifact.Path}
	case ".aab":
		toolName = "jarsigner"
		args = []string{"-verify", "-strict", "-certs", artifact.Path}
	default:
		return signingResult{}, fmt.Errorf("--file must end in .aab or .apk")
	}
	tool, err := resolvedExecutable(toolName)
	if err != nil {
		return signingResult{}, err
	}
	result := signingResult{Mode: "planned", Tool: tool, Args: args, Artifact: artifact}
	if shared.IsDryRun(ctx) {
		return result, nil
	}
	var stdout, stderr bytes.Buffer
	execCtx, cancel := shared.ContextWithTimeout(ctx, nil)
	defer cancel()
	err = localexec.RunnerFrom(ctx).Run(execCtx, localexec.Request{
		Executable: tool, Args: args, Stdout: &stdout, Stderr: &stderr,
	})
	if err != nil {
		return signingResult{}, fmt.Errorf("%s verification failed: %w: %s", toolName, err, diagnostic(stdout.String()+"\n"+stderr.String()))
	}
	result.Mode = "verified"
	result.Verified = true
	result.Details = diagnostic(stdout.String() + "\n" + stderr.String())
	return result, nil
}

var environmentNamePattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// InspectKeystoreCommand inspects a certificate without placing its password
// in process arguments.
func InspectKeystoreCommand() *ffcli.Command {
	fs := flag.NewFlagSet("android signing inspect", flag.ExitOnError)
	keystore := fs.String("keystore", "", "Upload-key keystore path")
	alias := fs.String("alias", "", "Key alias")
	passwordEnv := fs.String("password-env", "KEYSTORE_PASSWORD", "Environment variable containing the keystore password")
	output := fs.String("output", "json", "Output format: json, table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")
	return &ffcli.Command{
		Name:       "inspect",
		ShortUsage: "gplay android signing inspect --keystore <path> --alias <name> [--password-env KEYSTORE_PASSWORD]",
		ShortHelp:  "Inspect an upload certificate without exposing its password in argv.",
		LongHelp:   "The password is read by keytool directly from the named environment variable. It is never accepted as a flag or printed.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if len(args) != 0 || strings.TrimSpace(*keystore) == "" || strings.TrimSpace(*alias) == "" {
				return shared.UsageError("--keystore and --alias are required; positional arguments are not accepted")
			}
			if err := shared.ValidateOutputFlags(*output, *pretty); err != nil {
				return err
			}
			result, err := inspectKeystore(ctx, *keystore, *alias, *passwordEnv)
			if err != nil {
				return err
			}
			return shared.PrintOutputContext(ctx, result, *output, *pretty)
		},
	}
}

func inspectKeystore(ctx context.Context, path, alias, passwordEnv string) (signingResult, error) {
	artifact, err := inspectLocalArtifact(strings.TrimSpace(path))
	if err != nil {
		return signingResult{}, fmt.Errorf("inspect keystore: %w", err)
	}
	passwordEnv = strings.TrimSpace(passwordEnv)
	if !environmentNamePattern.MatchString(passwordEnv) {
		return signingResult{}, fmt.Errorf("--password-env must be an environment variable name")
	}
	if value, ok := os.LookupEnv(passwordEnv); !ok || value == "" {
		return signingResult{}, fmt.Errorf("environment variable %s is required so keytool cannot prompt interactively", passwordEnv)
	}
	tool, err := resolvedExecutable("keytool")
	if err != nil {
		return signingResult{}, err
	}
	args := []string{"-list", "-v", "-keystore", artifact.Path, "-alias", strings.TrimSpace(alias), "-storepass:env", passwordEnv}
	result := signingResult{Mode: "planned", Tool: tool, Args: args, Artifact: artifact}
	if shared.IsDryRun(ctx) {
		return result, nil
	}
	var stdout, stderr bytes.Buffer
	execCtx, cancel := shared.ContextWithTimeout(ctx, nil)
	defer cancel()
	err = localexec.RunnerFrom(ctx).Run(execCtx, localexec.Request{Executable: tool, Args: args, Stdout: &stdout, Stderr: &stderr})
	if err != nil {
		return signingResult{}, fmt.Errorf("keytool inspection failed: %w: %s", err, diagnostic(stdout.String()+"\n"+stderr.String()))
	}
	result.Mode = "inspected"
	result.Verified = true
	result.Details = diagnostic(stdout.String() + "\n" + stderr.String())
	return result, nil
}
