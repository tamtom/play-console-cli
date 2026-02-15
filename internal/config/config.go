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
	Debug                string        `json:"debug"`
}

// ErrNotFound is returned when the config file doesn't exist.
var ErrNotFound = fmt.Errorf("configuration not found")

// ErrInvalidPath is returned when the config path is invalid.
var ErrInvalidPath = errors.New("invalid config path")

// GlobalPath returns the global configuration file path.
func GlobalPath() (string, error) {
	return configPath()
}

// LocalPath returns the local configuration file path.
func LocalPath() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
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
	return LoadAt(path)
}

// LoadAt reads configuration from a specific path.
func LoadAt(path string) (*Config, error) {
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
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return err
	}
	return nil
}
