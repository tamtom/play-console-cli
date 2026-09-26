package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/tamtom/play-console-cli/internal/rootfs"
)

const (
	configDirName    = ".gplay"
	configFileName   = "config.json"
	configPathEnvVar = "GPLAY_CONFIG_PATH"
)

// DurationValue stores a duration with its raw string representation.
// It marshals to/from JSON as a string to preserve config compatibility.
type DurationValue struct {
	Duration time.Duration
	Raw      string
}

// ParseDurationValue parses a duration string or seconds value into a DurationValue.
func ParseDurationValue(raw string) (DurationValue, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return DurationValue{}, nil
	}
	parsed, err := parseDurationValue(raw)
	if err != nil {
		return DurationValue{}, err
	}
	return DurationValue{Duration: parsed, Raw: raw}, nil
}

// Value returns the parsed duration if it's positive.
func (d DurationValue) Value() (time.Duration, bool) {
	if d.Duration > 0 {
		return d.Duration, true
	}
	raw := strings.TrimSpace(d.Raw)
	if raw == "" {
		return 0, false
	}
	parsed, err := parseDurationValue(raw)
	if err != nil || parsed <= 0 {
		return 0, false
	}
	return parsed, true
}

// String returns the raw string when available, falling back to the duration value.
func (d DurationValue) String() string {
	if strings.TrimSpace(d.Raw) != "" {
		return d.Raw
	}
	if d.Duration == 0 {
		return ""
	}
	return d.Duration.String()
}

// MarshalJSON stores the raw string when available, preserving the config format.
func (d DurationValue) MarshalJSON() ([]byte, error) {
	raw := strings.TrimSpace(d.Raw)
	if raw == "" {
		if d.Duration == 0 {
			return json.Marshal("")
		}
		raw = d.Duration.String()
	}
	return json.Marshal(raw)
}

// UnmarshalJSON parses duration strings or seconds values from JSON.
func (d *DurationValue) UnmarshalJSON(data []byte) error {
	var raw string
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	raw = strings.TrimSpace(raw)
	d.Raw = raw
	if raw == "" {
		d.Duration = 0
		return nil
	}
	parsed, err := parseDurationValue(raw)
	if err != nil {
		d.Duration = 0
		return nil
	}
	d.Duration = parsed
	return nil
}

func parseDurationValue(raw string) (time.Duration, error) {
	if parsed, err := time.ParseDuration(raw); err == nil {
		return parsed, nil
	}
	seconds, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("invalid duration %q", raw)
	}
	return time.Duration(seconds) * time.Second, nil
}

// Profile stores a named auth profile in config.json.
type Profile struct {
	Name         string `json:"name"`
	Type         string `json:"type"`
	KeyPath      string `json:"key_path,omitempty"`
	TokenPath    string `json:"token_path,omitempty"`
	ClientID     string `json:"client_id,omitempty"`
	ClientSecret string `json:"client_secret,omitempty"`
}

// Config holds the application configuration.
type Config struct {
	DefaultProfile       string        `json:"default_profile"`
	Profiles             []Profile     `json:"profiles,omitempty"`
	PackageName          string        `json:"package_name,omitempty"`
	Timeout              DurationValue `json:"timeout"`
	TimeoutSeconds       DurationValue `json:"timeout_seconds"`
	UploadTimeout        DurationValue `json:"upload_timeout"`
	UploadTimeoutSeconds DurationValue `json:"upload_timeout_seconds"`
	MaxRetries           int           `json:"max_retries,omitempty"`
	RetryDelay           string        `json:"retry_delay,omitempty"`
	Debug                string        `json:"debug"`
	ChecksAccount        string        `json:"checks_account,omitempty"`
	GamesApplicationID   string        `json:"games_application_id,omitempty"`
	maxRetriesSet        bool
}

// MaxRetriesConfigured reports whether max_retries was explicitly present in
// the loaded config. This preserves the distinction between an omitted value
// (use the default) and zero (disable retries).
func (c *Config) MaxRetriesConfigured() bool {
	return c != nil && (c.maxRetriesSet || c.MaxRetries != 0)
}

// UnmarshalJSON records whether max_retries was explicitly configured.
func (c *Config) UnmarshalJSON(data []byte) error {
	type configAlias Config
	var decoded configAlias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	*c = Config(decoded)
	_, c.maxRetriesSet = fields["max_retries"]
	return nil
}

// MarshalJSON retains an explicitly configured zero max_retries value.
func (c Config) MarshalJSON() ([]byte, error) {
	type configAlias Config
	if !c.maxRetriesSet || c.MaxRetries != 0 {
		return json.Marshal(configAlias(c))
	}
	return json.Marshal(struct {
		configAlias
		MaxRetries int `json:"max_retries"`
	}{configAlias: configAlias(c), MaxRetries: c.MaxRetries})
}

const maxConfigRetries = 30

// Validate checks the configuration for invalid values.
func (c *Config) Validate() error {
	if c == nil {
		return nil
	}

	// Validate profile names are non-empty
	seen := make(map[string]bool)
	for i, p := range c.Profiles {
		name := strings.TrimSpace(p.Name)
		if name == "" {
			return fmt.Errorf("profile at index %d has empty name", i)
		}
		if seen[name] {
			return fmt.Errorf("duplicate profile name: %q", name)
		}
		seen[name] = true
	}

	// Validate default profile exists if set
	if dp := strings.TrimSpace(c.DefaultProfile); dp != "" && len(c.Profiles) > 0 {
		found := false
		for _, p := range c.Profiles {
			if p.Name == dp {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("default_profile %q not found in profiles", dp)
		}
	}

	// Validate retry bounds
	if c.MaxRetries < 0 || c.MaxRetries > maxConfigRetries {
		return fmt.Errorf("max_retries must be between 0 and %d, got %d", maxConfigRetries, c.MaxRetries)
	}

	return nil
}

// ErrNotFound is returned when the config file doesn't exist.
var ErrNotFound = fmt.Errorf("configuration not found")

// ErrInvalidPath is returned when the config path is invalid.
var ErrInvalidPath = errors.New("invalid config path")

// GlobalPath returns the global configuration file path.
func GlobalPath() (string, error) {
	return configPath()
}

// LocalPath returns the nearest local configuration file path.
// It walks up from the current directory to find the nearest .gplay/config.json,
// stopping at a .git boundary or the filesystem root.
func LocalPath() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	home, _ := os.UserHomeDir()
	dir := cwd
	// The .gplay directory in the home directory holds the global config,
	// so the search stops before it.
	for home == "" || !sameDir(dir, home) {
		candidate := filepath.Join(dir, configDirName, configFileName)
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
		legacy := filepath.Join(dir, configDirName, "config.yaml")
		if _, err := os.Stat(legacy); err == nil {
			return legacy, nil
		}

		// Stop at .git boundary
		gitDir := filepath.Join(dir, ".git")
		if _, err := os.Stat(gitDir); err == nil {
			break
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break // reached filesystem root
		}
		dir = parent
	}

	// Fall back to cwd/.gplay/config.json (original behavior)
	return filepath.Join(cwd, configDirName, configFileName), nil
}

// Path returns the active configuration file path.
func Path() (string, error) {
	return resolvePath()
}

func configDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(home, configDirName), nil
}

func configPath() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, configFileName), nil
}

func resolvePath() (string, error) {
	if envPath := strings.TrimSpace(os.Getenv(configPathEnvVar)); envPath != "" {
		return cleanConfigPath(envPath)
	}

	localPath, err := LocalPath()
	if err == nil {
		if _, err := os.Stat(localPath); err == nil {
			return localPath, nil
		}
	}

	return configPath()
}

// sameDir reports whether a and b name the same directory. It compares the
// files, not the strings, because a path can contain a symbolic link.
func sameDir(a, b string) bool {
	aInfo, err := os.Stat(a)
	if err != nil {
		return false
	}
	bInfo, err := os.Stat(b)
	if err != nil {
		return false
	}
	return os.SameFile(aInfo, bInfo)
}

// IgnoredLegacyGlobalPath returns the path of ~/.gplay/config.yaml when the
// active config is the global config.json and only the YAML file exists.
// gplay never read the global YAML file, so the caller shows a warning and
// the file stays ignored. It returns "" in all other cases.
func IgnoredLegacyGlobalPath() string {
	if strings.TrimSpace(os.Getenv(configPathEnvVar)) != "" {
		return ""
	}
	active, err := resolvePath()
	if err != nil {
		return ""
	}
	global, err := configPath()
	if err != nil || active != global {
		return ""
	}
	if _, err := os.Stat(global); err == nil {
		return ""
	}
	legacy := filepath.Join(filepath.Dir(global), "config.yaml")
	if _, err := os.Stat(legacy); err != nil {
		return ""
	}
	return legacy
}

func cleanConfigPath(path string) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", ErrInvalidPath
	}
	return filepath.Clean(path), nil
}

// Load reads the active configuration file.
func Load() (*Config, error) {
	path, err := Path()
	if err != nil {
		return nil, err
	}
	cfg, err := LoadAt(path)
	if err != nil {
		return nil, err
	}
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config at %s: %w", path, err)
	}
	return cfg, nil
}

// LoadAt reads configuration from a specific path.
func LoadAt(path string) (*Config, error) {
	if filepath.Ext(path) == ".yaml" || filepath.Ext(path) == ".yml" {
		return nil, fmt.Errorf("legacy YAML configuration at %s is unsupported; in %s, run gplay init --force with your --package, --service-account and --timeout values to create .gplay/config.json", path, filepath.Dir(filepath.Dir(path)))
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// SaveAt writes configuration to a specific path.
func SaveAt(path string, cfg *Config) error {
	if ext := strings.ToLower(filepath.Ext(path)); ext == ".yaml" || ext == ".yml" {
		return fmt.Errorf("legacy YAML configuration at %s is unsupported; recreate it as config.json using gplay init --force", path)
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	if err := rootfs.AtomicWriteFile(path, data, 0o600, 0o700); err != nil {
		return err
	}
	return nil
}
