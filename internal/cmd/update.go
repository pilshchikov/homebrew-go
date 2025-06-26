package cmd

import (
	"fmt"

	"github.com/homebrew/brew/internal/config"
	"github.com/homebrew/brew/internal/logger"
	"github.com/homebrew/brew/internal/tap"
	"github.com/spf13/cobra"
)

// NewUpdateCmd creates the update command
func NewUpdateCmd(cfg *config.Config) *cobra.Command {
	var (
		merge      bool
		preinstall bool
	)

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Fetch the newest version of Homebrew and all formulae",
		RunE: func(cmd *cobra.Command, args []string) error {
			logger.Progress("Updating Homebrew")

			// Update taps
			tapManager := tap.NewManager(cfg)
			taps, err := tapManager.ListTaps()
			if err != nil {
				return fmt.Errorf("failed to list taps: %w", err)
			}

			for _, t := range taps {
				logger.Step("Updating tap %s", t.Name)
				if err := tapManager.UpdateTap(t.Name); err != nil {
					logger.Warn("Failed to update tap %s: %v", t.Name, err)
				}
			}

			logger.Success("Updated Homebrew")
			return nil
		},
	}

	cmd.Flags().BoolVar(&merge, "merge", false, "Use git merge to apply updates")
	cmd.Flags().BoolVar(&preinstall, "preinstall", false, "Run preinstall script")

	return cmd
}
