package cmd

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/pilshchikov/homebrew-go/internal/config"
	"github.com/pilshchikov/homebrew-go/internal/logger"
	"github.com/pilshchikov/homebrew-go/internal/tap"
	"github.com/spf13/cobra"
)

// NewRootCmd creates the root command for the brew CLI
func NewRootCmd(cfg *config.Config, version, gitCommit, buildDate string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "brew",
		Short: "The missing package manager for macOS (or Linux)",
		Long: `Homebrew is a package manager for macOS (or Linux) that allows you to install
and manage software packages with ease. It provides a simple command-line
interface for installing, updating, and removing packages.`,
		Version: version,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			// Ensure directories exist
			if err := cfg.EnsureDirectories(); err != nil {
				logger.Error("Failed to create directories: %v", err)
				os.Exit(1)
			}

			// Check for updates (if auto-update is enabled)
			checkForUpdates(cfg)
		},
		CompletionOptions: cobra.CompletionOptions{
			DisableDefaultCmd: false,
		},
	}

	// Add version template
	cmd.SetVersionTemplate(fmt.Sprintf(`Homebrew %s
Homebrew/brew (git revision %s; last commit %s)
Homebrew/homebrew-core (git revision N/A)
Go: %s
Platform: %s
`, version, gitCommit, buildDate, runtime.Version(), runtime.GOOS+"/"+runtime.GOARCH))

	// Global flags
	cmd.PersistentFlags().BoolVar(&cfg.Debug, "debug", cfg.Debug, "Enable debug mode")
	cmd.PersistentFlags().BoolVarP(&cfg.Verbose, "verbose", "v", cfg.Verbose, "Enable verbose output")
	cmd.PersistentFlags().BoolVarP(&cfg.Quiet, "quiet", "q", cfg.Quiet, "Suppress output")
	cmd.PersistentFlags().BoolVar(&cfg.Force, "force", cfg.Force, "Force the operation")
	cmd.PersistentFlags().BoolVar(&cfg.DryRun, "dry-run", cfg.DryRun, "Show what would be done without actually doing it")

	// Add subcommands
	cmd.AddCommand(NewInstallCmd(cfg))
	cmd.AddCommand(NewUninstallCmd(cfg))
	cmd.AddCommand(NewUpgradeCmd(cfg))
	cmd.AddCommand(NewUpdateCmd(cfg))
	cmd.AddCommand(NewSearchCmd(cfg))
	cmd.AddCommand(NewInfoCmd(cfg))
	cmd.AddCommand(NewListCmd(cfg))
	cmd.AddCommand(NewCleanupCmd(cfg))
	cmd.AddCommand(NewServicesCmd(cfg))
	cmd.AddCommand(NewTapCmd(cfg))
	cmd.AddCommand(NewUntapCmd(cfg))
	cmd.AddCommand(NewDoctorCmd(cfg))
	cmd.AddCommand(NewConfigCmd(cfg))
	cmd.AddCommand(NewVersionCmd(cfg, version, gitCommit, buildDate))

	// Dependency and status commands
	cmd.AddCommand(NewDepsCmd(cfg))
	cmd.AddCommand(NewLeavesCmd(cfg))
	cmd.AddCommand(NewOutdatedCmd(cfg))
	cmd.AddCommand(NewPinCmd(cfg))
	cmd.AddCommand(NewUnpinCmd(cfg))
	cmd.AddCommand(NewLinkCmd(cfg))
	cmd.AddCommand(NewUnlinkCmd(cfg))

	// Information and listing commands
	cmd.AddCommand(NewHomeCmd(cfg))
	cmd.AddCommand(NewUsesCmd(cfg))
	cmd.AddCommand(NewDescCmd(cfg))
	cmd.AddCommand(NewOptionsCmd(cfg))
	cmd.AddCommand(NewMissingCmd(cfg))
	cmd.AddCommand(NewCasksCmd(cfg))
	cmd.AddCommand(NewFormulaeCmd(cfg))
	cmd.AddCommand(NewCommandsCmd(cfg))

	// Environment commands
	cmd.AddCommand(NewPrefixCmd(cfg))
	cmd.AddCommand(NewCellarCmd(cfg))
	cmd.AddCommand(NewCacheCmd(cfg))
	cmd.AddCommand(NewEnvCmd(cfg))

	// Help customization
	cmd.SetHelpTemplate(getHelpTemplate())

	return cmd
}

func getHelpTemplate() string {
	return `{{.Long | trimTrailingWhitespaces}}

{{if or .Runnable .HasSubCommands}}{{.UsageString}}{{end}}`
}

// Execute runs the root command
func Execute() error {
	cfg, err := config.New()
	if err != nil {
		return fmt.Errorf("failed to initialize config: %w", err)
	}

	logger.Init(cfg.Debug, cfg.Verbose, cfg.Quiet)

	rootCmd := NewRootCmd(cfg, "3.0.0", "unknown", "unknown")
	return rootCmd.Execute()
}

// checkForUpdates checks if Homebrew should be updated
func checkForUpdates(cfg *config.Config) {
	if cfg.NoAutoUpdate || cfg.CI {
		return
	}

	logger.Debug("Checking for updates...")

	// Check if we should perform auto-update based on time
	if !shouldAutoUpdate(cfg) {
		logger.Debug("Auto-update not needed at this time")
		return
	}

	logger.Step("Auto-updating Homebrew...")

	// Update the main Homebrew repository (brew core)
	tapManager := tap.NewManager(cfg)
	if err := tapManager.UpdateTap("homebrew/core"); err != nil {
		logger.Debug("Failed to auto-update homebrew/core: %v", err)
		return
	}

	// Update homebrew/cask if installed
	taps, err := tapManager.ListTaps()
	if err == nil {
		for _, t := range taps {
			if t.Name == "homebrew/cask" {
				if err := tapManager.UpdateTap("homebrew/cask"); err != nil {
					logger.Debug("Failed to auto-update homebrew/cask: %v", err)
				}
				break
			}
		}
	}

	// Update timestamp for next auto-update check
	updateLastUpdateTime(cfg)
	logger.Debug("Auto-update completed")
}

// setupShellCompletion sets up shell completion
func setupShellCompletion(cmd *cobra.Command) {
	// Add completion command
	completionCmd := &cobra.Command{
		Use:       "completion [bash|zsh|fish|powershell]",
		Short:     "Generate completion script",
		Args:      cobra.ExactValidArgs(1),
		ValidArgs: []string{"bash", "zsh", "fish", "powershell"},
		Run: func(cmd *cobra.Command, args []string) {
			switch args[0] {
			case "bash":
				cmd.Root().GenBashCompletion(os.Stdout)
			case "zsh":
				cmd.Root().GenZshCompletion(os.Stdout)
			case "fish":
				cmd.Root().GenFishCompletion(os.Stdout, true)
			case "powershell":
				cmd.Root().GenPowerShellCompletion(os.Stdout)
			}
		},
	}

	cmd.AddCommand(completionCmd)
}

// validateArgs validates common argument patterns
func validateArgs(cmd *cobra.Command, args []string, minArgs int) error {
	if len(args) < minArgs {
		return fmt.Errorf("requires at least %d argument(s), only received %d", minArgs, len(args))
	}
	return nil
}

// parseFormulaArgs parses formula arguments and separates them from options
func parseFormulaArgs(args []string) ([]string, []string) {
	var formulas []string
	var options []string

	for _, arg := range args {
		if strings.HasPrefix(arg, "--") {
			options = append(options, arg)
		} else {
			formulas = append(formulas, arg)
		}
	}

	return formulas, options
}

// shouldAutoUpdate checks if auto-update should be performed based on time
func shouldAutoUpdate(cfg *config.Config) bool {
	// Check if auto-update interval has passed (default 24 hours)
	interval := 24 * time.Hour

	lastUpdate := getLastUpdateTime(cfg)
	return time.Since(lastUpdate) >= interval
}

// getLastUpdateTime gets the timestamp of the last update
func getLastUpdateTime(cfg *config.Config) time.Time {
	updateFile := cfg.HomebrewPrefix + "/.last_update"

	if info, err := os.Stat(updateFile); err == nil {
		return info.ModTime()
	}

	// If file doesn't exist, return zero time to trigger update
	return time.Time{}
}

// updateLastUpdateTime updates the timestamp file
func updateLastUpdateTime(cfg *config.Config) {
	updateFile := cfg.HomebrewPrefix + "/.last_update"

	// Create or touch the file
	if file, err := os.Create(updateFile); err == nil {
		file.Close()
	}
}
