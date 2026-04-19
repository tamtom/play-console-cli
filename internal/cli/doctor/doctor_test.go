package doctor

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/tamtom/play-console-cli/internal/config"
)

func stubEnv(t *testing.T) Env {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	return Env{
		Now:      func() time.Time { return time.Unix(1_700_000_000, 0).UTC() },
		LookPath: func(name string) (string, error) { return "/usr/bin/" + name, nil },
		Stat:     os.Stat,
		HomeDir:  func() (string, error) { return home, nil },
		DNSLookup: func(ctx context.Context, host string) error {
			return nil
		},
		DiskFree:      func(path string) (uint64, error) { return 5 * 1024 * 1024 * 1024, nil },
		LoadConfig:    func() (*config.Config, error) { return nil, nil },
		ReadAuditPath: func() (string, error) { return filepath.Join(home, "audit.log"), nil },
		AuditEnabled:  func() bool { return true },
		CommandRunner: func(ctx context.Context, name string, args ...string) ([]byte, error) { return nil, nil },
	}
}

func TestRunHasMinimumChecks(t *testing.T) {
	env := stubEnv(t)
	r := Run(context.Background(), env)
	if len(r.Checks) < 15 {
		t.Errorf("expected >=15 checks, got %d", len(r.Checks))
	}
}

func TestRunCountsSeverities(t *testing.T) {
	env := stubEnv(t)
	r := Run(context.Background(), env)
	total := r.Passed + r.Warnings + r.Failures + r.Skipped
	if total != len(r.Checks) {
		t.Errorf("severity totals (%d) != checks (%d)", total, len(r.Checks))
	}
}

func TestCheckGcloudMissing(t *testing.T) {
	env := stubEnv(t)
	env.LookPath = func(name string) (string, error) { return "", errors.New("not found") }
	res := checkGcloud(context.Background(), env)
	if res.Severity != SeverityWarn {
		t.Errorf("expected warn, got %s", res.Severity)
	}
	if res.Hint == "" {
		t.Error("expected hint for missing gcloud")
	}
}

func TestCheckNetworkFailure(t *testing.T) {
	env := stubEnv(t)
	env.DNSLookup = func(ctx context.Context, host string) error { return errors.New("no dns") }
	res := checkNetwork(context.Background(), env)
	if res.Severity != SeverityFail {
		t.Errorf("expected fail, got %s", res.Severity)
	}
}

func TestCheckDiskSpaceLow(t *testing.T) {
	env := stubEnv(t)
	env.DiskFree = func(path string) (uint64, error) { return 10 * 1024 * 1024, nil } // 10MB
	res := checkDiskSpace(env)
	if res.Severity != SeverityWarn {
		t.Errorf("expected warn, got %s", res.Severity)
	}
}

func TestCheckServiceAccountMissing(t *testing.T) {
	env := stubEnv(t)
	env.LoadConfig = func() (*config.Config, error) {
		return &config.Config{
			DefaultProfile: "default",
			Profiles: []config.Profile{
				{Name: "default", Type: "service_account", KeyPath: "/does/not/exist.json"},
			},
		}, nil
	}
	res := checkServiceAccountFile(env)
	if res.Severity != SeverityFail {
		t.Errorf("expected fail, got %s (%s)", res.Severity, res.Detail)
	}
}

func TestCheckServiceAccountOK(t *testing.T) {
	env := stubEnv(t)
	f := filepath.Join(t.TempDir(), "sa.json")
	if err := os.WriteFile(f, []byte(`{}`), 0o600); err != nil {
		t.Fatal(err)
	}
	env.LoadConfig = func() (*config.Config, error) {
		return &config.Config{
			DefaultProfile: "default",
			Profiles: []config.Profile{
				{Name: "default", Type: "service_account", KeyPath: f},
			},
		}, nil
	}
	res := checkServiceAccountFile(env)
	if res.Severity != SeverityOK {
		t.Errorf("expected ok, got %s (%s)", res.Severity, res.Detail)
	}
}

func TestCheckDefaultProfileMissing(t *testing.T) {
	env := stubEnv(t)
	env.LoadConfig = func() (*config.Config, error) {
		return &config.Config{
			DefaultProfile: "ghost",
			Profiles:       []config.Profile{{Name: "real", Type: "service_account"}},
		}, nil
	}
	res := checkDefaultProfile(env)
	if res.Severity != SeverityFail {
		t.Errorf("expected fail, got %s", res.Severity)
	}
}

func TestCheckConfigNoProfiles(t *testing.T) {
	env := stubEnv(t)
	res := checkConfigFile(env)
	// nil cfg -> warn
	if res.Severity != SeverityWarn {
		t.Errorf("expected warn, got %s", res.Severity)
	}
}

func TestCheckAuditDisabled(t *testing.T) {
	env := stubEnv(t)
	env.AuditEnabled = func() bool { return false }
	res := checkAuditLog(env)
	if res.Severity != SeverityWarn {
		t.Errorf("expected warn, got %s", res.Severity)
	}
}

func TestDoctorCommandSmoke(t *testing.T) {
	cmd := DoctorCommand()
	if cmd.Name != "doctor" {
		t.Errorf("name = %q", cmd.Name)
	}
	if cmd.FlagSet == nil {
		t.Error("expected flagset")
	}
}
