package androidtools

import (
	"context"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/tamtom/play-console-cli/internal/cli/shared"
	"github.com/tamtom/play-console-cli/internal/localexec"
)

var (
	gradleNamePattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_-]*$`)
	moduleNamePattern = regexp.MustCompile(`^[A-Za-z0-9_.-]+(?:/[A-Za-z0-9_.-]+)*$`)
)

type buildResult struct {
	Mode         string          `json:"mode"`
	Project      string          `json:"project"`
	Wrapper      string          `json:"wrapper"`
	Tasks        []string        `json:"tasks"`
	ArtifactType string          `json:"artifactType"`
	Artifacts    []localArtifact `json:"artifacts"`
}

// BuildCommand builds a tested release artifact in one Gradle invocation.
func BuildCommand() *ffcli.Command {
	fs := flag.NewFlagSet("android build", flag.ExitOnError)
	project := fs.String("project", ".", "Android project directory")
	module := fs.String("module", "app", "Android application module path")
	variant := fs.String("variant", "release", "Build variant name")
	artifactType := fs.String("artifact-type", "aab", "Release artifact: aab or apk")
	skipTests := fs.Bool("skip-tests", false, "Skip release unit tests")
	skipLint := fs.Bool("skip-lint", false, "Skip release lint")
	clean := fs.Bool("clean", false, "Run the module clean task first")
	output := fs.String("output", "json", "Output format: json, table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")
	return &ffcli.Command{
		Name:       "build",
		ShortUsage: "gplay android build [--project <dir>] [--variant release] [flags]",
		ShortHelp:  "Test, lint, and build an Android release with the project Gradle wrapper.",
		LongHelp: `Run the project-owned Gradle wrapper directly, without a shell.

By default one invocation runs release unit tests, release lint, and produces
an Android App Bundle. Signing values are inherited from the environment and
are never accepted as CLI flags, logged, or copied into process arguments.

Global --dry-run validates the wrapper and prints the exact plan without
executing Gradle.`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if len(args) != 0 {
				return shared.UsageErrorf("unexpected arguments: %s", strings.Join(args, " "))
			}
			if err := shared.ValidateOutputFlags(*output, *pretty); err != nil {
				return err
			}
			result, err := runBuild(ctx, buildOptions{
				Project: *project, Module: *module, Variant: *variant, ArtifactType: *artifactType,
				SkipTests: *skipTests, SkipLint: *skipLint, Clean: *clean,
			})
			if err != nil {
				return err
			}
			return shared.PrintOutputContext(ctx, result, *output, *pretty)
		},
	}
}

type buildOptions struct {
	Project      string
	Module       string
	Variant      string
	ArtifactType string
	SkipTests    bool
	SkipLint     bool
	Clean        bool
}

func runBuild(ctx context.Context, opts buildOptions) (buildResult, error) {
	project, err := filepath.Abs(strings.TrimSpace(opts.Project))
	if err != nil {
		return buildResult{}, fmt.Errorf("resolve --project: %w", err)
	}
	info, err := os.Stat(project)
	if err != nil || !info.IsDir() {
		return buildResult{}, fmt.Errorf("--project must be a readable directory: %s", project)
	}
	module := strings.TrimSpace(opts.Module)
	variant := strings.TrimSpace(opts.Variant)
	if !moduleNamePattern.MatchString(module) {
		return buildResult{}, fmt.Errorf("--module contains unsupported characters")
	}
	if !gradleNamePattern.MatchString(variant) {
		return buildResult{}, fmt.Errorf("--variant contains unsupported characters")
	}
	typeName := strings.ToLower(strings.TrimSpace(opts.ArtifactType))
	if typeName != "aab" && typeName != "apk" {
		return buildResult{}, fmt.Errorf("--artifact-type must be aab or apk")
	}
	wrapperName := "gradlew"
	if runtime.GOOS == "windows" {
		wrapperName = "gradlew.bat"
	}
	wrapper := filepath.Join(project, wrapperName)
	wrapperInfo, err := os.Lstat(wrapper)
	if err != nil || wrapperInfo.Mode()&os.ModeSymlink != 0 || !wrapperInfo.Mode().IsRegular() {
		return buildResult{}, fmt.Errorf("project Gradle wrapper must be a regular, non-symlink file: %s", wrapper)
	}
	if runtime.GOOS != "windows" && wrapperInfo.Mode().Perm()&0o111 == 0 {
		return buildResult{}, fmt.Errorf("project Gradle wrapper is not executable: %s", wrapper)
	}

	qualified := func(task string) string { return ":" + strings.ReplaceAll(module, "/", ":") + ":" + task }
	variantTask := strings.ToUpper(variant[:1]) + variant[1:]
	var tasks []string
	if opts.Clean {
		tasks = append(tasks, qualified("clean"))
	}
	if !opts.SkipTests {
		tasks = append(tasks, qualified("test"+variantTask+"UnitTest"))
	}
	if !opts.SkipLint {
		tasks = append(tasks, qualified("lint"+variantTask))
	}
	buildTask := "bundle" + variantTask
	if typeName == "apk" {
		buildTask = "assemble" + variantTask
	}
	tasks = append(tasks, qualified(buildTask))
	result := buildResult{Mode: "planned", Project: project, Wrapper: wrapper, Tasks: tasks, ArtifactType: typeName, Artifacts: []localArtifact{}}
	if shared.IsDryRun(ctx) {
		return result, nil
	}

	execCtx, cancel := shared.ContextWithTimeout(ctx, nil)
	defer cancel()
	gradleArgs := append(append([]string(nil), tasks...), "--no-daemon", "--console=plain", "--stacktrace")
	err = localexec.RunnerFrom(ctx).Run(execCtx, localexec.Request{
		Executable: wrapper, Args: gradleArgs, Dir: project,
		Stdout: shared.Stderr(ctx), Stderr: shared.Stderr(ctx),
	})
	if err != nil {
		return buildResult{}, fmt.Errorf("gradle release build failed: %w", err)
	}

	extension := ".aab"
	if typeName == "apk" {
		extension = ".apk"
	}
	artifacts, err := discoverBuildArtifacts(project, module, extension)
	if err != nil {
		return buildResult{}, err
	}
	if len(artifacts) == 0 {
		return buildResult{}, fmt.Errorf("gradle succeeded but no %s artifact was found under %s/build/outputs", extension, module)
	}
	result.Mode = "built"
	result.Artifacts = artifacts
	return result, nil
}

func discoverBuildArtifacts(project, module, extension string) ([]localArtifact, error) {
	root := filepath.Join(project, filepath.FromSlash(module), "build", "outputs")
	if _, err := os.Stat(root); errorsIsNotExist(err) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	var paths []string
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type()&os.ModeSymlink != 0 {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if !entry.IsDir() && strings.EqualFold(filepath.Ext(entry.Name()), extension) {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("scan Gradle outputs: %w", err)
	}
	sort.Strings(paths)
	artifacts := make([]localArtifact, 0, len(paths))
	for _, path := range paths {
		artifact, err := inspectLocalArtifact(path)
		if err != nil {
			return nil, err
		}
		artifacts = append(artifacts, artifact)
	}
	return artifacts, nil
}

func errorsIsNotExist(err error) bool { return err != nil && os.IsNotExist(err) }
