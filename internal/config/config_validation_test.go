package config

import (
	"strings"
	"testing"
)

func TestValidate_ValidConfig(t *testing.T) {
	cfg := &Config{
		DefaultProfile: "default",
		Profiles: []Profile{
			{Name: "default", Type: "service_account", KeyPath: "/path/to/key.json"},
		},
		MaxRetries: 3,
		RetryDelay: "1s",
	}
	if err := cfg.Validate(); err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestValidate_EmptyProfileName(t *testing.T) {
	cfg := &Config{
		Profiles: []Profile{
			{Name: "", Type: "service_account"},
		},
	}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for empty profile name")
	}
	if !strings.Contains(err.Error(), "empty name") {
		t.Errorf("expected error about empty name, got: %s", err.Error())
	}
}

func TestValidate_DuplicateProfileNames(t *testing.T) {
	cfg := &Config{
		Profiles: []Profile{
			{Name: "default", Type: "service_account"},
			{Name: "default", Type: "service_account"},
		},
	}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for duplicate profile names")
	}
	if !strings.Contains(err.Error(), "duplicate profile name") {
		t.Errorf("expected error about duplicate profile name, got: %s", err.Error())
	}
}

func TestValidate_DefaultProfileNotInProfiles(t *testing.T) {
	cfg := &Config{
		DefaultProfile: "missing",
		Profiles: []Profile{
			{Name: "default", Type: "service_account"},
		},
	}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for default_profile not in profiles")
	}
	if !strings.Contains(err.Error(), "not found in profiles") {
		t.Errorf("expected error about default_profile not found, got: %s", err.Error())
	}
}

func TestValidate_MaxRetriesExceedsLimit(t *testing.T) {
	cfg := &Config{
		MaxRetries: 31,
	}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for max_retries > 30")
	}
	if !strings.Contains(err.Error(), "max_retries") {
		t.Errorf("expected error about max_retries, got: %s", err.Error())
	}
}

func TestValidate_MaxRetriesNegative(t *testing.T) {
	cfg := &Config{
		MaxRetries: -1,
	}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for negative max_retries")
	}
	if !strings.Contains(err.Error(), "max_retries") {
		t.Errorf("expected error about max_retries, got: %s", err.Error())
	}
}

func TestValidate_NilConfig(t *testing.T) {
	var cfg *Config
	if err := cfg.Validate(); err != nil {
		t.Errorf("expected no error for nil config, got %v", err)
	}
}

func TestValidate_MaxRetriesAtBoundary(t *testing.T) {
	cfg := &Config{
		MaxRetries: 30,
	}
	if err := cfg.Validate(); err != nil {
		t.Errorf("expected no error for max_retries=30, got %v", err)
	}
}

func TestValidate_MaxRetriesZero(t *testing.T) {
	cfg := &Config{
		MaxRetries: 0,
	}
	if err := cfg.Validate(); err != nil {
		t.Errorf("expected no error for max_retries=0, got %v", err)
	}
}
