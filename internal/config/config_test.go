package config

import (
	"os"
	"testing"
)

func TestNew(t *testing.T) {
	// Save original environment
	originalEnv := make(map[string]string)
	envVars := []string{
		"HOMEBREW_PREFIX", "HOMEBREW_REPOSITORY", "HOMEBREW_CELLAR",
		"HOMEBREW_DEBUG", "HOMEBREW_VERBOSE", "HOMEBREW_QUIET",
	}

	for _, env := range envVars {
		if val := os.Getenv(env); val != "" {
			originalEnv[env] = val
		}
		os.Unsetenv(env)
	}

	// Restore environment after test
	defer func() {
		for _, env := range envVars {
			os.Unsetenv(env)
		}
		for env, val := range originalEnv {
			os.Setenv(env, val)
		}
	}()

	cfg, err := New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Test default values
	if cfg.HomebrewPrefix == "" {
		t.Error("HomebrewPrefix should not be empty")
	}

	if cfg.HomebrewRepository == "" {
		t.Error("HomebrewRepository should not be empty")
	}

	if cfg.HomebrewLibrary == "" {
		t.Error("HomebrewLibrary should not be empty")
	}

	// Test that defaults are reasonable
	if cfg.CurlRetries != 3 {
		t.Errorf("CurlRetries = %v, want 3", cfg.CurlRetries)
	}

	if !cfg.AutoUpdate {
		t.Error("AutoUpdate should be true by default")
	}
}

func TestEnvironmentOverrides(t *testing.T) {
	// Set test environment variables
	testEnv := map[string]string{
		"HOMEBREW_PREFIX":     "/test/prefix",
		"HOMEBREW_DEBUG":      "1",
		"HOMEBREW_VERBOSE":    "true",
		"HOMEBREW_QUIET":      "false",
		"HOMEBREW_AUTO_UPDATE": "0",
	}

	// Save and set test environment
	originalEnv := make(map[string]string)
	for key, val := range testEnv {
		if original := os.Getenv(key); original != "" {
			originalEnv[key] = original
		}
		os.Setenv(key, val)
	}

	// Restore environment after test
	defer func() {
		for key := range testEnv {
			os.Unsetenv(key)
		}
		for key, val := range originalEnv {
			os.Setenv(key, val)
		}
	}()

	cfg, err := New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Test environment overrides
	if cfg.HomebrewPrefix != "/test/prefix" {
		t.Errorf("HomebrewPrefix = %v, want /test/prefix", cfg.HomebrewPrefix)
	}

	if !cfg.Debug {
		t.Error("Debug should be true when HOMEBREW_DEBUG=1")
	}

	if !cfg.Verbose {
		t.Error("Verbose should be true when HOMEBREW_VERBOSE=true")
	}

	if cfg.Quiet {
		t.Error("Quiet should be false when HOMEBREW_QUIET=false")
	}

	if cfg.AutoUpdate {
		t.Error("AutoUpdate should be false when HOMEBREW_AUTO_UPDATE=0")
	}
}

func TestGetBoolEnv(t *testing.T) {
	tests := []struct {
		name         string
		envValue     string
		defaultValue bool
		expected     bool
	}{
		{"true string", "true", false, true},
		{"false string", "false", true, false},
		{"1 value", "1", false, true},
		{"0 value", "0", true, false},
		{"empty value", "", true, true},
		{"invalid value", "invalid", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := "TEST_BOOL_ENV"
			if tt.envValue != "" {
				os.Setenv(key, tt.envValue)
			} else {
				os.Unsetenv(key)
			}
			defer os.Unsetenv(key)

			result := getBoolEnv(key, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("getBoolEnv() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGetIntEnv(t *testing.T) {
	tests := []struct {
		name         string
		envValue     string
		defaultValue int
		expected     int
	}{
		{"valid int", "42", 0, 42},
		{"empty value", "", 10, 10},
		{"invalid value", "invalid", 5, 5},
		{"zero value", "0", 10, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := "TEST_INT_ENV"
			if tt.envValue != "" {
				os.Setenv(key, tt.envValue)
			} else {
				os.Unsetenv(key)
			}
			defer os.Unsetenv(key)

			result := getIntEnv(key, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("getIntEnv() = %v, want %v", result, tt.expected)
			}
		})
	}
}