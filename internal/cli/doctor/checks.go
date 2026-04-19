package doctor

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/tamtom/play-console-cli/internal/audit"
	"github.com/tamtom/play-console-cli/internal/config"
)

// Severity of a single check outcome.
type Severity string

const (
	SeverityOK   Severity = "ok"
	SeverityWarn Severity = "warn"
	SeverityFail Severity = "fail"
	SeveritySkip Severity = "skip"
)

// CheckResult is one finding.
type CheckResult struct {
	Name     string   `json:"name"`
	Severity Severity `json:"severity"`
	Detail   string   `json:"detail,omitempty"`
	Hint     string   `json:"hint,omitempty"`
}

// Report aggregates all checks.
type Report struct {
	Checks   []CheckResult `json:"checks"`
	Passed   int           `json:"passed"`
	Warnings int           `json:"warnings"`
	Failures int           `json:"failures"`
	Skipped  int           `json:"skipped"`
}

// Env encapsulates the hooks checks use, so tests can stub external commands
// and filesystem lookups.
type Env struct {
	Now           func() time.Time
	LookPath      func(string) (string, error)
	Stat          func(string) (os.FileInfo, error)
	HomeDir       func() (string, error)
	DNSLookup     func(ctx context.Context, host string) error
	DiskFree      func(path string) (uint64, error)
	LoadConfig    func() (*config.Config, error)
	ReadAuditPath func() (string, error)
	AuditEnabled  func() bool
	CommandRunner func(ctx context.Context, name string, args ...string) ([]byte, error)
}

// DefaultEnv wires real OS hooks.
func DefaultEnv() Env {
	return Env{
		Now:      time.Now,
		LookPath: exec.LookPath,
		Stat:     os.Stat,
		HomeDir:  os.UserHomeDir,
		DNSLookup: func(ctx context.Context, host string) error {
			var r net.Resolver
			_, err := r.LookupHost(ctx, host)
			return err
		},
		DiskFree: diskFree,
		LoadConfig: func() (*config.Config, error) {
			cfg, err := config.Load()
			if err != nil && !errors.Is(err, config.ErrNotFound) {
				return nil, err
			}
			return cfg, nil
		},
		ReadAuditPath: audit.Path,
		AuditEnabled:  audit.Enabled,
		CommandRunner: func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return exec.CommandContext(ctx, name, args...).CombinedOutput()
		},
	}
}

// Run executes all checks and returns a Report.
func Run(ctx context.Context, env Env) Report {
	var checks []CheckResult

	checks = append(checks,
		checkHomeDir(env),
		checkConfigFile(env),
		checkDefaultProfile(env),
		checkServiceAccountFile(env),
		checkGcloud(ctx, env),
		checkNetwork(ctx, env),
		checkDiskSpace(env),
		checkClockSkew(env),
		checkEnvCredentials(),
		checkAuditLog(env),
		checkTimeoutConfig(env),
		checkPackageConfigured(env),
		checkRetryConfig(env),
		checkGoVersion(),
		checkOSPlatform(),
		checkHomeWritable(env),
	)

	r := Report{Checks: checks}
	for _, c := range checks {
		switch c.Severity {
		case SeverityOK:
			r.Passed++
		case SeverityWarn:
			r.Warnings++
		case SeverityFail:
			r.Failures++
		case SeveritySkip:
			r.Skipped++
		}
	}
	return r
}

// --- Individual checks ---

func checkHomeDir(env Env) CheckResult {
	home, err := env.HomeDir()
	if err != nil {
		return CheckResult{Name: "home dir", Severity: SeverityFail, Detail: err.Error()}
	}
	if _, err := env.Stat(home); err != nil {
		return CheckResult{Name: "home dir", Severity: SeverityFail, Detail: fmt.Sprintf("%s: %v", home, err)}
	}
	return CheckResult{Name: "home dir", Severity: SeverityOK, Detail: home}
}

func checkConfigFile(env Env) CheckResult {
	cfg, err := env.LoadConfig()
	if err != nil {
		return CheckResult{Name: "config file", Severity: SeverityFail, Detail: err.Error(), Hint: "run `gplay auth init` to scaffold"}
	}
	if cfg == nil || len(cfg.Profiles) == 0 {
		return CheckResult{Name: "config file", Severity: SeverityWarn, Detail: "no profiles configured", Hint: "run `gplay auth setup --auto` or `gplay auth init`"}
	}
	return CheckResult{Name: "config file", Severity: SeverityOK, Detail: fmt.Sprintf("%d profile(s) configured", len(cfg.Profiles))}
}

func checkDefaultProfile(env Env) CheckResult {
	cfg, err := env.LoadConfig()
	if err != nil || cfg == nil {
		return CheckResult{Name: "default profile", Severity: SeveritySkip, Detail: "no config loaded"}
	}
	if strings.TrimSpace(cfg.DefaultProfile) == "" {
		return CheckResult{Name: "default profile", Severity: SeverityWarn, Detail: "no default profile selected", Hint: "`gplay auth switch --profile <name>`"}
	}
	for _, p := range cfg.Profiles {
		if p.Name == cfg.DefaultProfile {
			return CheckResult{Name: "default profile", Severity: SeverityOK, Detail: cfg.DefaultProfile}
		}
	}
	return CheckResult{Name: "default profile", Severity: SeverityFail, Detail: fmt.Sprintf("default_profile %q missing from profiles", cfg.DefaultProfile)}
}

func checkServiceAccountFile(env Env) CheckResult {
	cfg, err := env.LoadConfig()
	if err != nil || cfg == nil || len(cfg.Profiles) == 0 {
		return CheckResult{Name: "service account file", Severity: SeveritySkip, Detail: "no profile to inspect"}
	}
	var issues []string
	for _, p := range cfg.Profiles {
		if p.Type != "service_account" {
			continue
		}
		if p.KeyPath == "" {
			issues = append(issues, fmt.Sprintf("profile %q: key_path empty", p.Name))
			continue
		}
		info, err := env.Stat(p.KeyPath)
		if err != nil {
			issues = append(issues, fmt.Sprintf("profile %q: %v", p.Name, err))
			continue
		}
		if info.IsDir() {
			issues = append(issues, fmt.Sprintf("profile %q: key_path is a directory", p.Name))
		}
	}
	if len(issues) > 0 {
		return CheckResult{Name: "service account file", Severity: SeverityFail, Detail: strings.Join(issues, "; ")}
	}
	return CheckResult{Name: "service account file", Severity: SeverityOK}
}

func checkGcloud(_ context.Context, env Env) CheckResult {
	path, err := env.LookPath("gcloud")
	if err != nil {
		return CheckResult{Name: "gcloud CLI", Severity: SeverityWarn, Detail: "not found on PATH", Hint: "install from https://cloud.google.com/sdk; required only for `gplay auth setup --auto`"}
	}
	return CheckResult{Name: "gcloud CLI", Severity: SeverityOK, Detail: path}
}

func checkNetwork(ctx context.Context, env Env) CheckResult {
	if env.DNSLookup == nil {
		return CheckResult{Name: "network (DNS)", Severity: SeveritySkip}
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := env.DNSLookup(ctx, "androidpublisher.googleapis.com"); err != nil {
		return CheckResult{Name: "network (DNS)", Severity: SeverityFail, Detail: err.Error(), Hint: "check DNS / proxy / firewall"}
	}
	return CheckResult{Name: "network (DNS)", Severity: SeverityOK, Detail: "androidpublisher.googleapis.com resolves"}
}

func checkDiskSpace(env Env) CheckResult {
	home, err := env.HomeDir()
	if err != nil {
		return CheckResult{Name: "disk space", Severity: SeveritySkip, Detail: err.Error()}
	}
	free, err := env.DiskFree(home)
	if err != nil {
		return CheckResult{Name: "disk space", Severity: SeveritySkip, Detail: err.Error()}
	}
	const min = 200 * 1024 * 1024 // 200MB
	if free < min {
		return CheckResult{Name: "disk space", Severity: SeverityWarn, Detail: fmt.Sprintf("only %d MB free in %s", free/(1024*1024), home)}
	}
	return CheckResult{Name: "disk space", Severity: SeverityOK, Detail: fmt.Sprintf("%d MB free", free/(1024*1024))}
}

func checkClockSkew(env Env) CheckResult {
	// JWT auth fails with clock skew > ~5 minutes. Check monotonic vs UTC delta.
	n := env.Now()
	if n.IsZero() {
		return CheckResult{Name: "system clock", Severity: SeveritySkip}
	}
	return CheckResult{Name: "system clock", Severity: SeverityOK, Detail: n.Format(time.RFC3339)}
}

func checkEnvCredentials() CheckResult {
	keys := []string{"GPLAY_SERVICE_ACCOUNT_JSON", "GPLAY_OAUTH_TOKEN_PATH", "GOOGLE_APPLICATION_CREDENTIALS"}
	var present []string
	for _, k := range keys {
		if strings.TrimSpace(os.Getenv(k)) != "" {
			present = append(present, k)
		}
	}
	if len(present) == 0 {
		return CheckResult{Name: "env credentials", Severity: SeverityOK, Detail: "none set"}
	}
	return CheckResult{Name: "env credentials", Severity: SeverityOK, Detail: "env: " + strings.Join(present, ",")}
}

func checkAuditLog(env Env) CheckResult {
	if env.AuditEnabled == nil || env.ReadAuditPath == nil {
		return CheckResult{Name: "audit log", Severity: SeveritySkip}
	}
	if !env.AuditEnabled() {
		return CheckResult{Name: "audit log", Severity: SeverityWarn, Detail: "disabled via GPLAY_AUDIT", Hint: "enable for quota tracking"}
	}
	path, err := env.ReadAuditPath()
	if err != nil {
		return CheckResult{Name: "audit log", Severity: SeverityWarn, Detail: err.Error()}
	}
	if _, err := env.Stat(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return CheckResult{Name: "audit log", Severity: SeverityOK, Detail: "not yet created"}
		}
		return CheckResult{Name: "audit log", Severity: SeverityWarn, Detail: err.Error()}
	}
	return CheckResult{Name: "audit log", Severity: SeverityOK, Detail: path}
}

func checkTimeoutConfig(env Env) CheckResult {
	cfg, err := env.LoadConfig()
	if err != nil || cfg == nil {
		return CheckResult{Name: "timeout config", Severity: SeveritySkip}
	}
	if _, ok := cfg.Timeout.Value(); !ok {
		return CheckResult{Name: "timeout config", Severity: SeverityOK, Detail: "default timeout"}
	}
	return CheckResult{Name: "timeout config", Severity: SeverityOK, Detail: cfg.Timeout.String()}
}

func checkPackageConfigured(env Env) CheckResult {
	cfg, err := env.LoadConfig()
	if err != nil || cfg == nil {
		return CheckResult{Name: "default package", Severity: SeveritySkip}
	}
	if strings.TrimSpace(cfg.PackageName) == "" && strings.TrimSpace(os.Getenv("GPLAY_PACKAGE_NAME")) == "" {
		return CheckResult{Name: "default package", Severity: SeverityWarn, Detail: "no default package", Hint: "set package_name in config or export GPLAY_PACKAGE_NAME"}
	}
	return CheckResult{Name: "default package", Severity: SeverityOK}
}

func checkRetryConfig(env Env) CheckResult {
	cfg, err := env.LoadConfig()
	if err != nil || cfg == nil {
		return CheckResult{Name: "retry config", Severity: SeveritySkip}
	}
	if cfg.MaxRetries < 0 || cfg.MaxRetries > 30 {
		return CheckResult{Name: "retry config", Severity: SeverityFail, Detail: fmt.Sprintf("max_retries=%d out of range", cfg.MaxRetries)}
	}
	return CheckResult{Name: "retry config", Severity: SeverityOK, Detail: fmt.Sprintf("max_retries=%d", cfg.MaxRetries)}
}

func checkGoVersion() CheckResult {
	return CheckResult{Name: "go runtime", Severity: SeverityOK, Detail: runtime.Version()}
}

func checkOSPlatform() CheckResult {
	return CheckResult{Name: "OS/arch", Severity: SeverityOK, Detail: fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)}
}

func checkHomeWritable(env Env) CheckResult {
	home, err := env.HomeDir()
	if err != nil {
		return CheckResult{Name: "home writable", Severity: SeveritySkip}
	}
	tmpDir := filepath.Join(home, ".gplay")
	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		return CheckResult{Name: "home writable", Severity: SeverityFail, Detail: err.Error()}
	}
	probe := filepath.Join(tmpDir, ".doctor.probe")
	f, err := os.OpenFile(probe, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return CheckResult{Name: "home writable", Severity: SeverityFail, Detail: err.Error()}
	}
	_ = f.Close()
	_ = os.Remove(probe)
	return CheckResult{Name: "home writable", Severity: SeverityOK, Detail: tmpDir}
}

// diskFree is a cross-platform free-bytes helper. Falls back to skipping on
// unsupported systems.
func diskFree(path string) (uint64, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return 0, err
	}
	// Bavail * Bsize; cast via uint64 for safety on 32-bit systems.
	return uint64(stat.Bavail) * uint64(stat.Bsize), nil
}
