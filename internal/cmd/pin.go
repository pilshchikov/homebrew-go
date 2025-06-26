package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pilshchikov/homebrew-go/internal/config"
	"github.com/pilshchikov/homebrew-go/internal/logger"
	"github.com/spf13/cobra"
)

// NewPinCmd creates the pin command
func NewPinCmd(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pin FORMULA...",
		Short: "Pin the specified formulae to their current versions",
		Long: `Pin the specified formulae to their current versions, preventing them from
being upgraded when running 'brew upgrade'. This is useful for keeping
specific versions of formulae that you don't want to upgrade automatically.

Pinned formulae will be ignored by 'brew upgrade' and 'brew upgrade --all'.`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPin(cfg, args)
		},
	}

	return cmd
}

// NewUnpinCmd creates the unpin command
func NewUnpinCmd(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "unpin FORMULA...",
		Short: "Unpin specified formulae, allowing them to be upgraded again",
		Long: `Unpin the specified formulae, allowing them to be upgraded when running
'brew upgrade'. This removes the pin that was set by 'brew pin'.`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUnpin(cfg, args)
		},
	}

	return cmd
}

func runPin(cfg *config.Config, formulaNames []string) error {
	pinnedKegsDir := filepath.Join(cfg.HomebrewLibrary, "PinnedKegs")

	// Ensure PinnedKegs directory exists
	if err := os.MkdirAll(pinnedKegsDir, 0755); err != nil {
		return fmt.Errorf("failed to create PinnedKegs directory: %w", err)
	}

	var pinned []string
	var errors []string

	for _, formulaName := range formulaNames {
		// Check if formula is installed
		if !isFormulaInstalledPin(cfg, formulaName) {
			errors = append(errors, fmt.Sprintf("Formula %s is not installed", formulaName))
			continue
		}

		// Check if already pinned
		pinFile := filepath.Join(pinnedKegsDir, formulaName)
		if _, err := os.Stat(pinFile); err == nil {
			logger.Info("Formula %s is already pinned", formulaName)
			continue
		}

		// Create pin file
		if err := createPinFile(pinFile, formulaName); err != nil {
			errors = append(errors, fmt.Sprintf("Failed to pin %s: %v", formulaName, err))
			continue
		}

		pinned = append(pinned, formulaName)
		logger.Success("Pinned %s", formulaName)
	}

	// Report results
	if len(pinned) > 0 {
		logger.Info("Pinned formulae: %v", pinned)
	}

	if len(errors) > 0 {
		for _, errMsg := range errors {
			logger.Error(errMsg)
		}
		return fmt.Errorf("failed to pin some formulae")
	}

	return nil
}

func runUnpin(cfg *config.Config, formulaNames []string) error {
	pinnedKegsDir := filepath.Join(cfg.HomebrewLibrary, "PinnedKegs")

	var unpinned []string
	var errors []string

	for _, formulaName := range formulaNames {
		pinFile := filepath.Join(pinnedKegsDir, formulaName)

		// Check if pinned
		if _, err := os.Stat(pinFile); os.IsNotExist(err) {
			logger.Info("Formula %s is not pinned", formulaName)
			continue
		}

		// Remove pin file
		if err := os.Remove(pinFile); err != nil {
			errors = append(errors, fmt.Sprintf("Failed to unpin %s: %v", formulaName, err))
			continue
		}

		unpinned = append(unpinned, formulaName)
		logger.Success("Unpinned %s", formulaName)
	}

	// Report results
	if len(unpinned) > 0 {
		logger.Info("Unpinned formulae: %v", unpinned)
	}

	if len(errors) > 0 {
		for _, errMsg := range errors {
			logger.Error(errMsg)
		}
		return fmt.Errorf("failed to unpin some formulae")
	}

	return nil
}

func createPinFile(pinFile, formulaName string) error {
	// Get current installed version
	installedVersions, err := getInstalledVersions(nil, formulaName)
	if err != nil {
		return err
	}

	if len(installedVersions) == 0 {
		return fmt.Errorf("no installed versions found")
	}

	// Use the latest installed version
	pinnedVersion := getLatestVersion(installedVersions)

	// Create pin file with version info
	content := fmt.Sprintf("# Pin file for %s\n# Pinned at version: %s\n", formulaName, pinnedVersion)

	return os.WriteFile(pinFile, []byte(content), 0644)
}

func isFormulaInstalledPin(cfg *config.Config, formulaName string) bool {
	formulaPath := filepath.Join(cfg.HomebrewCellar, formulaName)

	entries, err := os.ReadDir(formulaPath)
	if err != nil {
		return false
	}

	// Check if there are any version directories
	for _, entry := range entries {
		if entry.IsDir() {
			return true
		}
	}

	return false
}
