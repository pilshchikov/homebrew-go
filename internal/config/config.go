package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

// Config holds all Homebrew configuration
type Config struct {
	// Core paths
	HomebrewPrefix     string
	HomebrewRepository string
	HomebrewLibrary    string
	HomebrewCellar     string
	HomebrewCaskroom   string
	HomebrewCache      string
	HomebrewLogs       string
	HomebrewTemp       string

	// Behavior flags
	Debug                      bool
	Verbose                    bool
	Quiet                      bool
	AutoUpdate                 bool
	InstallCleanup             bool
	NoInstallUpgrade           bool
	NoInstalledDependentsCheck bool
	DisplayInstallTimes        bool
	ForceBottle                bool
	BuildFromSource            bool
	KeepTmp                    bool
	Force                      bool
	DryRun                     bool

	// Development flags
	Developer              bool
	NoAutoUpdate           bool
	SystemEnvTakesPriority bool

	// Network settings
	CurlRetries        int
	CurlConnectTimeout int
	CurlMaxTime        int
	APIAllowlist       []string
	APIBlocklist       []string

	// Analytics
	NoAnalytics       bool
	NoGoogleAnalytics bool

	// CI/Testing
	CI                 bool
	GithubHostedRunner bool
}

// New creates a new Config with default values and environment overrides
func New() (*Config, error) {
	cfg := &Config{
		// Default values
		AutoUpdate:         true,
		InstallCleanup:     true,
		CurlRetries:        3,
		CurlConnectTimeout: 5,
		CurlMaxTime:        0,
	}

	// Set paths based on OS and architecture
	if err := cfg.setPaths(); err != nil {
		return nil, fmt.Errorf("failed to set paths: %w", err)
	}

	// Load environment variables
	cfg.loadFromEnv()

	return cfg, nil
}

func (c *Config) setPaths() error {
	// Set default prefix based on OS/architecture
	if c.HomebrewPrefix == "" {
		if prefix := os.Getenv("HOMEBREW_PREFIX"); prefix != "" {
			c.HomebrewPrefix = prefix
		} else if runtime.GOOS == "darwin" && runtime.GOARCH == "amd64" {
			c.HomebrewPrefix = "/usr/local"
		} else if runtime.GOOS == "darwin" {
			c.HomebrewPrefix = "/opt/homebrew"
		} else {
			c.HomebrewPrefix = "/home/linuxbrew/.linuxbrew"
		}
	}

	// Set repository path
	if c.HomebrewRepository == "" {
		if repo := os.Getenv("HOMEBREW_REPOSITORY"); repo != "" {
			c.HomebrewRepository = repo
		} else {
			c.HomebrewRepository = c.HomebrewPrefix
		}
	}

	// Set library path
	if c.HomebrewLibrary == "" {
		c.HomebrewLibrary = filepath.Join(c.HomebrewRepository, "Library")
	}

	// Set cellar path
	if c.HomebrewCellar == "" {
		if cellar := os.Getenv("HOMEBREW_CELLAR"); cellar != "" {
			c.HomebrewCellar = cellar
		} else {
			c.HomebrewCellar = filepath.Join(c.HomebrewPrefix, "Cellar")
		}
	}

	// Set caskroom path
	if c.HomebrewCaskroom == "" {
		if caskroom := os.Getenv("HOMEBREW_CASKROOM"); caskroom != "" {
			c.HomebrewCaskroom = caskroom
		} else {
			c.HomebrewCaskroom = filepath.Join(c.HomebrewPrefix, "Caskroom")
		}
	}

	// Set cache path
	if c.HomebrewCache == "" {
		if cache := os.Getenv("HOMEBREW_CACHE"); cache != "" {
			c.HomebrewCache = cache
		} else {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("failed to get user home directory: %w", err)
			}
			c.HomebrewCache = filepath.Join(homeDir, "Library", "Caches", "Homebrew")
		}
	}

	// Set logs path
	if c.HomebrewLogs == "" {
		if logs := os.Getenv("HOMEBREW_LOGS"); logs != "" {
			c.HomebrewLogs = logs
		} else {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("failed to get user home directory: %w", err)
			}
			c.HomebrewLogs = filepath.Join(homeDir, "Library", "Logs", "Homebrew")
		}
	}

	// Set temp path
	if c.HomebrewTemp == "" {
		if temp := os.Getenv("HOMEBREW_TEMP"); temp != "" {
			c.HomebrewTemp = temp
		} else {
			c.HomebrewTemp = os.TempDir()
		}
	}

	return nil
}

func (c *Config) loadFromEnv() {
	// Behavior flags
	c.Debug = getBoolEnv("HOMEBREW_DEBUG", c.Debug)
	c.Verbose = getBoolEnv("HOMEBREW_VERBOSE", c.Verbose)
	c.Quiet = getBoolEnv("HOMEBREW_QUIET", c.Quiet)
	c.AutoUpdate = getBoolEnv("HOMEBREW_AUTO_UPDATE", c.AutoUpdate)
	c.InstallCleanup = getBoolEnv("HOMEBREW_INSTALL_CLEANUP", c.InstallCleanup)
	c.NoInstallUpgrade = getBoolEnv("HOMEBREW_NO_INSTALL_UPGRADE", c.NoInstallUpgrade)
	c.NoInstalledDependentsCheck = getBoolEnv("HOMEBREW_NO_INSTALLED_DEPENDENTS_CHECK", c.NoInstalledDependentsCheck)
	c.DisplayInstallTimes = getBoolEnv("HOMEBREW_DISPLAY_INSTALL_TIMES", c.DisplayInstallTimes)
	c.ForceBottle = getBoolEnv("HOMEBREW_FORCE_BOTTLE", c.ForceBottle)
	c.BuildFromSource = getBoolEnv("HOMEBREW_BUILD_FROM_SOURCE", c.BuildFromSource)
	c.KeepTmp = getBoolEnv("HOMEBREW_KEEP_TMP", c.KeepTmp)
	c.Force = getBoolEnv("HOMEBREW_FORCE", c.Force)

	// Development flags
	c.Developer = getBoolEnv("HOMEBREW_DEVELOPER", c.Developer)
	c.NoAutoUpdate = getBoolEnv("HOMEBREW_NO_AUTO_UPDATE", c.NoAutoUpdate)
	c.SystemEnvTakesPriority = getBoolEnv("HOMEBREW_SYSTEM_ENV_TAKES_PRIORITY", c.SystemEnvTakesPriority)

	// Network settings
	c.CurlRetries = getIntEnv("HOMEBREW_CURL_RETRIES", c.CurlRetries)
	c.CurlConnectTimeout = getIntEnv("HOMEBREW_CURL_CONNECT_TIMEOUT", c.CurlConnectTimeout)
	c.CurlMaxTime = getIntEnv("HOMEBREW_CURL_MAX_TIME", c.CurlMaxTime)

	// API settings
	if allowlist := os.Getenv("HOMEBREW_API_ALLOWLIST"); allowlist != "" {
		c.APIAllowlist = strings.Split(allowlist, ",")
	}
	if blocklist := os.Getenv("HOMEBREW_API_BLOCKLIST"); blocklist != "" {
		c.APIBlocklist = strings.Split(blocklist, ",")
	}

	// Analytics
	c.NoAnalytics = getBoolEnv("HOMEBREW_NO_ANALYTICS", c.NoAnalytics)
	c.NoGoogleAnalytics = getBoolEnv("HOMEBREW_NO_GOOGLE_ANALYTICS", c.NoGoogleAnalytics)

	// CI/Testing
	c.CI = getBoolEnv("CI", c.CI)
	c.GithubHostedRunner = getBoolEnv("HOMEBREW_GITHUB_HOSTED_RUNNER", c.GithubHostedRunner)
}

func getBoolEnv(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.ParseBool(value); err == nil {
			return parsed
		}
		// Handle "1" and "0" as boolean values
		if value == "1" {
			return true
		} else if value == "0" {
			return false
		}
	}
	return defaultValue
}

func getIntEnv(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

// EnsureDirectories creates necessary directories
func (c *Config) EnsureDirectories() error {
	dirs := []string{
		c.HomebrewCellar,
		c.HomebrewCaskroom,
		c.HomebrewCache,
		c.HomebrewLogs,
		c.HomebrewTemp,
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return nil
}
