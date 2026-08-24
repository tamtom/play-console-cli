package androidtools

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"image/png"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/tamtom/play-console-cli/internal/cli/shared"
	"github.com/tamtom/play-console-cli/internal/localexec"
	"github.com/tamtom/play-console-cli/internal/rootfs"
	"github.com/tamtom/play-console-cli/internal/validation"
)

const maxScreenshotBytes = 50 * 1024 * 1024

var (
	localePattern         = regexp.MustCompile(`^[A-Za-z]{2,3}(?:-[A-Za-z0-9]{2,8})*$`)
	screenshotNamePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)
)

var screenshotTypes = map[string]string{
	"phoneScreenshots":      "phone",
	"sevenInchScreenshots":  "tablet7",
	"tenInchScreenshots":    "tablet10",
	"tvScreenshots":         "tv",
	"wearScreenshots":       "wear",
	"chromebookScreenshots": "chromebook",
}

type screenshotResult struct {
	Mode       string `json:"mode"`
	Serial     string `json:"serial"`
	Locale     string `json:"locale"`
	ImageType  string `json:"imageType"`
	Path       string `json:"path"`
	Width      int    `json:"width,omitempty"`
	Height     int    `json:"height,omitempty"`
	Size       int    `json:"sizeBytes,omitempty"`
	SHA256     string `json:"sha256,omitempty"`
	Executable string `json:"executable"`
}

// ScreenshotsCommand returns local adb screenshot helpers.
func ScreenshotsCommand() *ffcli.Command {
	fs := flag.NewFlagSet("android screenshots", flag.ExitOnError)
	return &ffcli.Command{
		Name:        "screenshots",
		ShortUsage:  "gplay android screenshots capture [flags]",
		ShortHelp:   "Capture validated Play listing screenshots from Android devices.",
		FlagSet:     fs,
		UsageFunc:   shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{CaptureScreenshotCommand()},
		Exec:        func(context.Context, []string) error { return flag.ErrHelp },
	}
}

// CaptureScreenshotCommand captures the currently visible device screen.
func CaptureScreenshotCommand() *ffcli.Command {
	fs := flag.NewFlagSet("android screenshots capture", flag.ExitOnError)
	serial := fs.String("serial", "", "adb device/emulator serial")
	locale := fs.String("locale", "en-US", "Locale label used in the output path")
	imageType := fs.String("type", "phoneScreenshots", "Play screenshot type")
	name := fs.String("name", "screenshot", "Screenshot filename without .png")
	outputDir := fs.String("output-dir", "./metadata", "Root output directory")
	layout := fs.String("layout", "metadata", "Directory layout: metadata or release")
	output := fs.String("output", "json", "Output format: json, table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")
	return &ffcli.Command{
		Name:       "capture",
		ShortUsage: "gplay android screenshots capture --serial <id> --locale <tag> --type <type> --name <name>",
		ShortHelp:  "Capture the current screen through adb exec-out and validate the PNG.",
		LongHelp: `Capture the currently visible device screen directly over adb; no temporary
file is created on the device. The locale is an output label only—the command
does not mutate device locale or app state.

The default metadata layout writes <dir>/<locale>/images/<type>/<name>.png for
gplay sync. --layout release writes <dir>/<locale>/<type>/<name>.png for
--screenshots-dir release workflows.`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if len(args) != 0 || strings.TrimSpace(*serial) == "" {
				return shared.UsageError("--serial is required and positional arguments are not accepted")
			}
			if err := shared.ValidateOutputFlags(*output, *pretty); err != nil {
				return err
			}
			result, err := captureScreenshot(ctx, screenshotOptions{
				Serial: *serial, Locale: *locale, ImageType: *imageType, Name: *name,
				OutputDir: *outputDir, Layout: *layout,
			})
			if err != nil {
				return err
			}
			return shared.PrintOutputContext(ctx, result, *output, *pretty)
		},
	}
}

type screenshotOptions struct {
	Serial    string
	Locale    string
	ImageType string
	Name      string
	OutputDir string
	Layout    string
}

func captureScreenshot(ctx context.Context, opts screenshotOptions) (screenshotResult, error) {
	serial := strings.TrimSpace(opts.Serial)
	locale := strings.TrimSpace(opts.Locale)
	imageType := strings.TrimSpace(opts.ImageType)
	name := strings.TrimSuffix(strings.TrimSpace(opts.Name), ".png")
	deviceType, ok := screenshotTypes[imageType]
	if !ok {
		return screenshotResult{}, fmt.Errorf("--type is not a supported Play screenshot type")
	}
	if !localePattern.MatchString(locale) {
		return screenshotResult{}, fmt.Errorf("--locale must be a BCP-47-style language tag")
	}
	if !screenshotNamePattern.MatchString(name) || name == "." || name == ".." {
		return screenshotResult{}, fmt.Errorf("--name contains unsupported characters")
	}
	layout := strings.ToLower(strings.TrimSpace(opts.Layout))
	var relative string
	switch layout {
	case "metadata":
		relative = filepath.Join(locale, "images", imageType, name+".png")
	case "release":
		relative = filepath.Join(locale, imageType, name+".png")
	default:
		return screenshotResult{}, fmt.Errorf("--layout must be metadata or release")
	}
	rootDir, err := filepath.Abs(strings.TrimSpace(opts.OutputDir))
	if err != nil {
		return screenshotResult{}, fmt.Errorf("resolve --output-dir: %w", err)
	}
	destination := filepath.Join(rootDir, relative)
	adb, err := resolvedExecutable("adb")
	if err != nil {
		return screenshotResult{}, err
	}
	result := screenshotResult{
		Mode: "planned", Serial: serial, Locale: locale, ImageType: imageType,
		Path: destination, Executable: adb,
	}
	if shared.IsDryRun(ctx) {
		return result, nil
	}

	var screenshot bytes.Buffer
	limited := &limitedWriter{Writer: &screenshot, Remaining: maxScreenshotBytes}
	var stderr bytes.Buffer
	execCtx, cancel := shared.ContextWithTimeout(ctx, nil)
	defer cancel()
	err = localexec.RunnerFrom(ctx).Run(execCtx, localexec.Request{
		Executable: adb, Args: []string{"-s", serial, "exec-out", "screencap", "-p"},
		Stdout: limited, Stderr: &stderr,
	})
	if err != nil {
		return screenshotResult{}, fmt.Errorf("adb screenshot failed: %w: %s", err, diagnostic(stderr.String()))
	}
	data := screenshot.Bytes()
	config, err := png.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return screenshotResult{}, fmt.Errorf("adb did not return a valid PNG: %w", err)
	}
	if finding := validation.ValidateScreenshotDimensions(deviceType, config.Width, config.Height); finding != nil {
		return screenshotResult{}, fmt.Errorf("captured screenshot is not Play-ready: %s", finding.Message)
	}
	outputRoot, err := rootfs.OpenOrCreate(rootDir, 0o755)
	if err != nil {
		return screenshotResult{}, fmt.Errorf("open screenshot output root: %w", err)
	}
	defer func() { _ = outputRoot.Close() }()
	if err := outputRoot.AtomicWrite(relative, data, 0o644); err != nil {
		return screenshotResult{}, fmt.Errorf("write screenshot: %w", err)
	}
	hash := sha256.Sum256(data)
	result.Mode = "captured"
	result.Width = config.Width
	result.Height = config.Height
	result.Size = len(data)
	result.SHA256 = hex.EncodeToString(hash[:])
	return result, nil
}

type limitedWriter struct {
	Writer    *bytes.Buffer
	Remaining int
}

func (w *limitedWriter) Write(data []byte) (int, error) {
	if len(data) > w.Remaining {
		return 0, fmt.Errorf("screenshot exceeds %d bytes", maxScreenshotBytes)
	}
	n, err := w.Writer.Write(data)
	w.Remaining -= n
	return n, err
}
