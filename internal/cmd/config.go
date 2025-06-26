package cmd

import (
	"fmt"

	"github.com/homebrew/brew/internal/config"
	"github.com/spf13/cobra"
)

// NewConfigCmd creates the config command
func NewConfigCmd(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Show Homebrew and system configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			return showConfig(cfg)
		},
	}

	return cmd
}

func showConfig(cfg *config.Config) error {
	fmt.Printf("HOMEBREW_PREFIX: %s\n", cfg.HomebrewPrefix)
	fmt.Printf("HOMEBREW_REPOSITORY: %s\n", cfg.HomebrewRepository)
	fmt.Printf("HOMEBREW_LIBRARY: %s\n", cfg.HomebrewLibrary)
	fmt.Printf("HOMEBREW_CELLAR: %s\n", cfg.HomebrewCellar)
	fmt.Printf("HOMEBREW_CASKROOM: %s\n", cfg.HomebrewCaskroom)
	fmt.Printf("HOMEBREW_CACHE: %s\n", cfg.HomebrewCache)
	fmt.Printf("HOMEBREW_LOGS: %s\n", cfg.HomebrewLogs)
	fmt.Printf("HOMEBREW_TEMP: %s\n", cfg.HomebrewTemp)

	fmt.Printf("\nBehavior flags:\n")
	fmt.Printf("  Debug: %t\n", cfg.Debug)
	fmt.Printf("  Verbose: %t\n", cfg.Verbose)
	fmt.Printf("  Auto-update: %t\n", cfg.AutoUpdate)
	fmt.Printf("  Install cleanup: %t\n", cfg.InstallCleanup)

	return nil
}
