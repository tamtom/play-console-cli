package updatecmd

import (
	"testing"
)

func TestUpdateCommandName(t *testing.T) {
	cmd := UpdateCommand()
	if cmd.Name != "update" {
		t.Errorf("expected command name %q, got %q", "update", cmd.Name)
	}
}

func TestUpdateCommandShortHelp(t *testing.T) {
	cmd := UpdateCommand()
	if cmd.ShortHelp == "" {
		t.Error("expected non-empty ShortHelp")
	}
}

func TestUpdateCommandFlags(t *testing.T) {
	cmd := UpdateCommand()

	checkFlag := cmd.FlagSet.Lookup("check")
	if checkFlag == nil {
		t.Fatal("expected --check flag to be registered")
	}
	if checkFlag.DefValue != "false" {
		t.Errorf("expected --check default %q, got %q", "false", checkFlag.DefValue)
	}

	forceFlag := cmd.FlagSet.Lookup("force")
	if forceFlag == nil {
		t.Fatal("expected --force flag to be registered")
	}
	if forceFlag.DefValue != "false" {
		t.Errorf("expected --force default %q, got %q", "false", forceFlag.DefValue)
	}
}

func TestUpdateCommandUsageFunc(t *testing.T) {
	cmd := UpdateCommand()
	if cmd.UsageFunc == nil {
		t.Error("expected UsageFunc to be set")
	}
}

func TestDetectInstallMethod(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "homebrew cellar path",
			path:     "/opt/homebrew/Cellar/gplay/1.0.0/bin/gplay",
			expected: "homebrew",
		},
		{
			name:     "homebrew path",
			path:     "/usr/local/homebrew/bin/gplay",
			expected: "homebrew",
		},
		{
			name:     "linuxbrew path",
			path:     "/home/user/.linuxbrew/bin/gplay",
			expected: "homebrew",
		},
		{
			name:     "go install default GOPATH",
			path:     "/Users/testuser/go/bin/gplay",
			expected: "goinstall",
		},
		{
			name:     "binary direct path",
			path:     "/usr/local/bin/gplay",
			expected: "binary",
		},
		{
			name:     "binary in opt",
			path:     "/opt/gplay/bin/gplay",
			expected: "binary",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// For the go install test, we need HOME set correctly
			if tt.expected == "goinstall" {
				t.Setenv("HOME", "/Users/testuser")
				t.Setenv("GOPATH", "")
			}
			got := detectInstallMethod(tt.path)
			if got != tt.expected {
				t.Errorf("detectInstallMethod(%q) = %q, want %q", tt.path, got, tt.expected)
			}
		})
	}
}

func TestDetectInstallMethodCustomGOPATH(t *testing.T) {
	t.Setenv("GOPATH", "/custom/gopath")
	got := detectInstallMethod("/custom/gopath/bin/gplay")
	if got != "goinstall" {
		t.Errorf("expected %q for custom GOPATH, got %q", "goinstall", got)
	}
}
