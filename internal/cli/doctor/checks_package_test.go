package doctor

import (
	"testing"

	"github.com/tamtom/play-console-cli/internal/config"
)

func TestCheckPackageConfiguredAcceptsEnvironment(t *testing.T) {
	env := Env{LoadConfig: func() (*config.Config, error) { return &config.Config{}, nil }}
	for _, key := range []string{"GPLAY_PACKAGE", "GPLAY_PACKAGE_NAME"} {
		t.Run(key, func(t *testing.T) {
			t.Setenv("GPLAY_PACKAGE", "")
			t.Setenv("GPLAY_PACKAGE_NAME", "")
			t.Setenv(key, "com.example.app")
			if got := checkPackageConfigured(env); got.Severity != SeverityOK {
				t.Fatalf("severity = %v, want OK: %+v", got.Severity, got)
			}
		})
	}
	t.Setenv("GPLAY_PACKAGE", "")
	t.Setenv("GPLAY_PACKAGE_NAME", "")
	if got := checkPackageConfigured(env); got.Severity != SeverityWarn || got.Hint != "set package_name in config or export GPLAY_PACKAGE" {
		t.Fatalf("no package: %+v", got)
	}
}
