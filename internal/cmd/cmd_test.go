package cmd

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/pilshchikov/homebrew-go/internal/config"
	"github.com/spf13/cobra"
)

func TestNewRootCmd(t *testing.T) {
	cfg := &config.Config{
		HomebrewPrefix: "/test/prefix",
	}

	rootCmd := NewRootCmd(cfg, "1.0.0", "abc123", "2023-01-01")

	if rootCmd.Use != "brew" {
		t.Errorf("Root command use = %v, want brew", rootCmd.Use)
	}

	if rootCmd.Version != "1.0.0" {
		t.Errorf("Root command version = %v, want 1.0.0", rootCmd.Version)
	}

	// Test that subcommands are added
	subcommands := []string{
		"install", "uninstall", "upgrade", "update", "search",
		"info", "list", "cleanup", "services", "tap", "untap",
		"doctor", "config", "version",
	}

	for _, subcmd := range subcommands {
		found := false
		for _, cmd := range rootCmd.Commands() {
			if cmd.Name() == subcmd {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Subcommand %s not found", subcmd)
		}
	}
}

func TestParseFormulaArgs(t *testing.T) {
	tests := []struct {
		name             string
		args             []string
		expectedFormulae []string
		expectedOptions  []string
	}{
		{
			name:             "only formulae",
			args:             []string{"wget", "curl", "python"},
			expectedFormulae: []string{"wget", "curl", "python"},
			expectedOptions:  []string{},
		},
		{
			name:             "only options",
			args:             []string{"--verbose", "--force", "--debug"},
			expectedFormulae: []string{},
			expectedOptions:  []string{"--verbose", "--force", "--debug"},
		},
		{
			name:             "mixed",
			args:             []string{"wget", "--verbose", "curl", "--force"},
			expectedFormulae: []string{"wget", "curl"},
			expectedOptions:  []string{"--verbose", "--force"},
		},
		{
			name:             "empty",
			args:             []string{},
			expectedFormulae: []string{},
			expectedOptions:  []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formulae, options := parseFormulaArgs(tt.args)

			if len(formulae) != len(tt.expectedFormulae) {
				t.Errorf("Formulae count = %v, want %v", len(formulae), len(tt.expectedFormulae))
			}

			for i, expected := range tt.expectedFormulae {
				if i < len(formulae) && formulae[i] != expected {
					t.Errorf("Formula[%d] = %v, want %v", i, formulae[i], expected)
				}
			}

			if len(options) != len(tt.expectedOptions) {
				t.Errorf("Options count = %v, want %v", len(options), len(tt.expectedOptions))
			}

			for i, expected := range tt.expectedOptions {
				if i < len(options) && options[i] != expected {
					t.Errorf("Option[%d] = %v, want %v", i, options[i], expected)
				}
			}
		})
	}
}

func TestValidateArgs(t *testing.T) {
	cmd := &cobra.Command{}

	tests := []struct {
		name    string
		args    []string
		minArgs int
		wantErr bool
	}{
		{"sufficient args", []string{"arg1", "arg2"}, 2, false},
		{"more than sufficient", []string{"arg1", "arg2", "arg3"}, 2, false},
		{"insufficient args", []string{"arg1"}, 2, true},
		{"no args required", []string{}, 0, false},
		{"no args but some required", []string{}, 1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateArgs(cmd, tt.args, tt.minArgs)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateArgs() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestIsCaskName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"app suffix with dash", "firefox-browser.app", true},
		{"desktop in name", "desktop-app", true},
		{"app in name", "myapp-tool", true},
		{"hyphenated", "multi-word-app", true},
		{"simple formula", "wget", false},
		{"no special patterns", "simple", false},
		{"underscore only", "my_tool", false},
		{"app suffix no dash", "firefox.app", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isCaskName(tt.input)
			if result != tt.expected {
				t.Errorf("isCaskName(%s) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestGetPlatform(t *testing.T) {
	platform := getPlatform()

	if platform == "" {
		t.Error("getPlatform() should not return empty string")
	}

	// Platform should be one of the known values or "unknown"
	validPlatforms := []string{"monterey", "arm64_monterey", "x86_64_linux", "unknown"}
	found := false
	for _, valid := range validPlatforms {
		if platform == valid {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("getPlatform() = %v, should be one of %v", platform, validPlatforms)
	}
}

func TestShowConfig(t *testing.T) {
	cfg := &config.Config{
		HomebrewPrefix:     "/test/prefix",
		HomebrewRepository: "/test/repo",
		HomebrewLibrary:    "/test/library",
		HomebrewCellar:     "/test/cellar",
		HomebrewCaskroom:   "/test/caskroom",
		HomebrewCache:      "/test/cache",
		HomebrewLogs:       "/test/logs",
		HomebrewTemp:       "/test/temp",
		Debug:              true,
		Verbose:            false,
		AutoUpdate:         true,
		InstallCleanup:     false,
	}

	// Capture stdout
	var buf bytes.Buffer
	originalStdout := func() {
		showConfig(cfg)
	}

	// This is a simplified test - in practice we'd need to redirect stdout
	// For now, just test that the function doesn't panic
	err := showConfig(cfg)
	if err != nil {
		t.Errorf("showConfig() error = %v", err)
	}

	_ = buf // Suppress unused variable warning
	_ = originalStdout
}

func TestShowEnv(t *testing.T) {
	cfg := &config.Config{
		HomebrewPrefix:     "/test/prefix",
		HomebrewRepository: "/test/repo",
		HomebrewCellar:     "/test/cellar",
		HomebrewCaskroom:   "/test/caskroom",
	}

	// Test that showEnv doesn't panic
	err := showEnv(cfg, false)
	if err != nil {
		t.Errorf("showEnv() error = %v", err)
	}
}

func TestAskForConfirmation(t *testing.T) {
	// This function reads from stdin, so we can't easily test it
	// without mocking stdin. For now, just ensure it exists and
	// has the right signature.

	// Test that the function exists and can be called
	// We can't compare functions to nil, so just test that it doesn't panic
	defer func() {
		if r := recover(); r != nil {
			t.Error("askForConfirmation function should not panic when called")
		}
	}()

	// Test with empty input (will return false but shouldn't panic)
	_ = askForConfirmation("test", "formula")
}

func TestInstallOptions(t *testing.T) {
	opts := &installOptions{
		FormulaOnly:        true,
		CaskOnly:           false,
		BuildFromSource:    true,
		ForceBottle:        false,
		IgnoreDependencies: true,
		OnlyDependencies:   false,
		IncludeTest:        true,
		HeadOnly:           false,
		KeepTmp:            true,
		DebugSymbols:       false,
		DisplayTimes:       true,
		Ask:                false,
		CC:                 "gcc",
		Force:              true,
		DryRun:             false,
		Verbose:            true,
	}

	if !opts.FormulaOnly {
		t.Error("FormulaOnly should be true")
	}

	if opts.CaskOnly {
		t.Error("CaskOnly should be false")
	}

	if !opts.BuildFromSource {
		t.Error("BuildFromSource should be true")
	}

	if opts.CC != "gcc" {
		t.Errorf("CC = %v, want gcc", opts.CC)
	}
}

func TestParseInstallArgs(t *testing.T) {
	tests := []struct {
		name             string
		args             []string
		opts             *installOptions
		expectedFormulae int
		expectedCasks    int
	}{
		{
			name:             "formula only mode",
			args:             []string{"wget", "curl", "firefox.app"},
			opts:             &installOptions{FormulaOnly: true},
			expectedFormulae: 3,
			expectedCasks:    0,
		},
		{
			name:             "cask only mode",
			args:             []string{"wget", "curl", "firefox.app"},
			opts:             &installOptions{CaskOnly: true},
			expectedFormulae: 0,
			expectedCasks:    3,
		},
		{
			name:             "auto detect mode",
			args:             []string{"wget", "curl", "firefox-app"},
			opts:             &installOptions{},
			expectedFormulae: 2, // wget, curl
			expectedCasks:    1, // firefox-app (contains hyphen and app)
		},
		{
			name:             "tap qualified formula",
			args:             []string{"user/tap/formula"},
			opts:             &installOptions{},
			expectedFormulae: 1,
			expectedCasks:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formulae, casks, err := parseInstallArgs(tt.args, tt.opts)
			if err != nil {
				t.Errorf("parseInstallArgs() error = %v", err)
			}

			if len(formulae) != tt.expectedFormulae {
				t.Errorf("Formulae count = %v, want %v", len(formulae), tt.expectedFormulae)
			}

			if len(casks) != tt.expectedCasks {
				t.Errorf("Casks count = %v, want %v", len(casks), tt.expectedCasks)
			}
		})
	}
}

func TestCommandCreation(t *testing.T) {
	cfg := &config.Config{}

	commands := []struct {
		name string
		fn   func(*config.Config) *cobra.Command
	}{
		{"install", NewInstallCmd},
		{"uninstall", NewUninstallCmd},
		{"upgrade", NewUpgradeCmd},
		{"update", NewUpdateCmd},
		{"search", NewSearchCmd},
		{"info", NewInfoCmd},
		{"list", NewListCmd},
		{"cleanup", NewCleanupCmd},
		{"services", NewServicesCmd},
		{"tap", NewTapCmd},
		{"untap", NewUntapCmd},
		{"doctor", NewDoctorCmd},
		{"config", NewConfigCmd},
	}

	for _, cmd := range commands {
		t.Run(cmd.name, func(t *testing.T) {
			command := cmd.fn(cfg)
			if command == nil {
				t.Errorf("%s command should not be nil", cmd.name)
			}
			if command.Name() != cmd.name {
				t.Errorf("%s command name = %v, want %v", cmd.name, command.Name(), cmd.name)
			}
		})
	}
}

func TestVersionCommand(t *testing.T) {
	cfg := &config.Config{}
	version := "1.0.0"
	gitCommit := "abc123"
	buildDate := "2023-01-01"

	cmd := NewVersionCmd(cfg, version, gitCommit, buildDate)

	if cmd == nil {
		t.Error("Version command should not be nil")
	}

	if cmd.Name() != "version" {
		t.Errorf("Version command name = %v, want version", cmd.Name())
	}

	// Test version command execution by capturing output
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	err := cmd.Execute()
	if err != nil {
		t.Errorf("Version command execution error = %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, version) {
		t.Errorf("Version output should contain version %s, but got: %s", version, output)
	}
}

func TestEnvironmentCommands(t *testing.T) {
	cfg := &config.Config{
		HomebrewPrefix: "/test/prefix",
		HomebrewCellar: "/test/cellar",
		HomebrewCache:  "/test/cache",
	}

	commands := []struct {
		name     string
		fn       func(*config.Config) *cobra.Command
		expected string
	}{
		{"prefix", NewPrefixCmd, cfg.HomebrewPrefix},
		{"cellar", NewCellarCmd, cfg.HomebrewCellar},
		{"cache", NewCacheCmd, cfg.HomebrewCache},
	}

	for _, cmd := range commands {
		t.Run(cmd.name, func(t *testing.T) {
			command := cmd.fn(cfg)
			if command == nil {
				t.Errorf("%s command should not be nil", cmd.name)
			}

			// Test command execution
			var buf bytes.Buffer
			command.SetOut(&buf)
			command.SetErr(&buf)

			err := command.Execute()
			if err != nil {
				t.Errorf("%s command execution error = %v", cmd.name, err)
			}

			output := strings.TrimSpace(buf.String())
			if output != cmd.expected {
				t.Errorf("%s command output = %v, want %v", cmd.name, output, cmd.expected)
			}
		})
	}
}

func TestUpdateCommand(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "brew-test-update")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cfg := &config.Config{
		HomebrewRepository: tempDir,
	}

	cmd := NewUpdateCmd(cfg)
	if cmd == nil {
		t.Error("Update command should not be nil")
	}

	if cmd.Name() != "update" {
		t.Errorf("Update command name = %v, want update", cmd.Name())
	}

	// Test flags
	if !cmd.Flags().HasFlags() {
		t.Error("Update command should have flags")
	}
}

func TestSearchCommandFlags(t *testing.T) {
	cfg := &config.Config{}
	cmd := NewSearchCmd(cfg)

	if cmd == nil {
		t.Error("Search command should not be nil")
	}

	// Test that flags exist
	flags := []string{"formulae", "casks", "desc"}
	for _, flag := range flags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("Search command should have --%s flag", flag)
		}
	}
}

func TestDoctorCommandExecution(t *testing.T) {
	cfg := &config.Config{
		HomebrewPrefix:   "/tmp/test-prefix",
		HomebrewCellar:   "/tmp/test-cellar",
		HomebrewCaskroom: "/tmp/test-caskroom",
		HomebrewCache:    "/tmp/test-cache",
	}

	cmd := NewDoctorCmd(cfg)
	if cmd == nil {
		t.Error("Doctor command should not be nil")
	}

	// Test JSON flag exists
	if cmd.Flags().Lookup("json") == nil {
		t.Error("Doctor command should have --json flag")
	}
}

func TestInfoCommandFlags(t *testing.T) {
	cfg := &config.Config{}
	cmd := NewInfoCmd(cfg)

	if cmd == nil {
		t.Error("Info command should not be nil")
	}

	// Test that flags exist
	flags := []string{"json", "installed", "analytics"}
	for _, flag := range flags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("Info command should have --%s flag", flag)
		}
	}
}

func TestListCommandFlags(t *testing.T) {
	cfg := &config.Config{}
	cmd := NewListCmd(cfg)

	if cmd == nil {
		t.Error("List command should not be nil")
	}

	// Test that flags exist
	flags := []string{"formulae", "casks", "versions", "full-name"}
	for _, flag := range flags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("List command should have --%s flag", flag)
		}
	}

	// Test aliases
	if len(cmd.Aliases) == 0 || cmd.Aliases[0] != "ls" {
		t.Error("List command should have 'ls' alias")
	}
}

func TestCleanupCommandFlags(t *testing.T) {
	cfg := &config.Config{}
	cmd := NewCleanupCmd(cfg)

	if cmd == nil {
		t.Error("Cleanup command should not be nil")
	}

	// Test that flags exist
	flags := []string{"dry-run", "prune"}
	for _, flag := range flags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("Cleanup command should have --%s flag", flag)
		}
	}
}

func TestUninstallCommandFlags(t *testing.T) {
	cfg := &config.Config{}
	cmd := NewUninstallCmd(cfg)

	if cmd == nil {
		t.Error("Uninstall command should not be nil")
	}

	// Test that flags exist
	flags := []string{"force", "ignore-dependencies", "zap"}
	for _, flag := range flags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("Uninstall command should have --%s flag", flag)
		}
	}

	// Test aliases
	expectedAliases := []string{"remove", "rm"}
	if len(cmd.Aliases) != len(expectedAliases) {
		t.Errorf("Expected %d aliases, got %d", len(expectedAliases), len(cmd.Aliases))
	}
	for i, alias := range expectedAliases {
		if i < len(cmd.Aliases) && cmd.Aliases[i] != alias {
			t.Errorf("Expected alias %s, got %s", alias, cmd.Aliases[i])
		}
	}
}

func TestTapCommandFlags(t *testing.T) {
	cfg := &config.Config{}
	cmd := NewTapCmd(cfg)

	if cmd == nil {
		t.Error("Tap command should not be nil")
	}

	// Test that flags exist
	flags := []string{"force", "shallow", "quiet", "branch"}
	for _, flag := range flags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("Tap command should have --%s flag", flag)
		}
	}
}

func TestUntapCommandFlags(t *testing.T) {
	cfg := &config.Config{}
	cmd := NewUntapCmd(cfg)

	if cmd == nil {
		t.Error("Untap command should not be nil")
	}

	// Test that flags exist
	if cmd.Flags().Lookup("force") == nil {
		t.Error("Untap command should have --force flag")
	}

	// Test args validation
	if cmd.Args == nil {
		t.Error("Untap command should have argument validation")
	}
}

func TestServicesCommandSubcommands(t *testing.T) {
	cfg := &config.Config{}
	cmd := NewServicesCmd(cfg)

	if cmd == nil {
		t.Error("Services command should not be nil")
	}

	// Test that subcommands exist
	subcommands := []string{"list", "start", "stop"}
	for _, subcmd := range subcommands {
		found := false
		for _, command := range cmd.Commands() {
			if command.Name() == subcmd {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Services command should have %s subcommand", subcmd)
		}
	}
}
