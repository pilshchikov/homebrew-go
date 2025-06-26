package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/pilshchikov/homebrew-go/internal/api"
	"github.com/pilshchikov/homebrew-go/internal/config"
	"github.com/pilshchikov/homebrew-go/internal/installer"
	"github.com/pilshchikov/homebrew-go/internal/logger"
	"github.com/spf13/cobra"
)

// NewUpgradeCmd creates the upgrade command
func NewUpgradeCmd(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "upgrade [FORMULA|CASK...]",
		Short: "Upgrade formulae and casks",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUpgrade(cfg, args)
		},
	}

	return cmd
}

func runUpgrade(cfg *config.Config, args []string) error {
	apiClient := api.NewClient(cfg)

	if len(args) == 0 {
		// Upgrade all outdated formulae
		logger.Progress("Checking for outdated formulae")
		outdated, err := findOutdatedFormulae(cfg, apiClient)
		if err != nil {
			return fmt.Errorf("failed to find outdated formulae: %w", err)
		}

		if len(outdated) == 0 {
			logger.Info("All formulae are up to date")
			return nil
		}

		logger.Info("Found %d outdated formulae: %s", len(outdated), strings.Join(outdated, ", "))
		args = outdated
	} else {
		logger.Progress("Upgrading specified formulae: %s", strings.Join(args, ", "))
	}

	// Upgrade each specified formula
	for _, formulaName := range args {
		logger.Progress("Upgrading %s", formulaName)

		// Check if formula is installed
		installed, err := isFormulaInstalled(cfg, formulaName)
		if err != nil {
			return fmt.Errorf("failed to check if %s is installed: %w", formulaName, err)
		}

		if !installed {
			logger.Warn("Formula %s is not installed, installing instead", formulaName)
			// Fall through to install
		} else {
			// Check if upgrade is needed
			currentVersion, err := getInstalledVersion(cfg, formulaName)
			if err != nil {
				return fmt.Errorf("failed to get current version of %s: %w", formulaName, err)
			}

			latestFormula, err := apiClient.GetFormula(formulaName)
			if err != nil {
				return fmt.Errorf("failed to get latest version of %s: %w", formulaName, err)
			}

			if currentVersion == latestFormula.Version {
				logger.Info("Formula %s is already up to date (%s)", formulaName, currentVersion)
				continue
			}

			logger.Info("Upgrading %s from %s to %s", formulaName, currentVersion, latestFormula.Version)

			// Uninstall old version
			if err := runUninstall(cfg, []string{formulaName}, &uninstallOptions{Force: true, IgnoreDeps: true}); err != nil {
				return fmt.Errorf("failed to uninstall old version of %s: %w", formulaName, err)
			}
		}

		// Install new version
		if err := installFormula(cfg, formulaName); err != nil {
			return fmt.Errorf("failed to install %s: %w", formulaName, err)
		}

		logger.Success("Successfully upgraded %s", formulaName)
	}

	return nil
}

func findOutdatedFormulae(cfg *config.Config, apiClient *api.Client) ([]string, error) {
	var outdated []string

	// Get list of installed formulae
	files, err := os.ReadDir(cfg.HomebrewCellar)
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		if !file.IsDir() {
			continue
		}

		formulaName := file.Name()

		// Get current installed version
		currentVersion, err := getInstalledVersion(cfg, formulaName)
		if err != nil {
			logger.Debug("Failed to get version for %s: %v", formulaName, err)
			continue
		}

		// Get latest version from API
		latestFormula, err := apiClient.GetFormula(formulaName)
		if err != nil {
			logger.Debug("Failed to get latest version for %s: %v", formulaName, err)
			continue
		}

		// Compare versions
		if currentVersion != latestFormula.Version {
			logger.Debug("Found outdated formula: %s (%s -> %s)", formulaName, currentVersion, latestFormula.Version)
			outdated = append(outdated, formulaName)
		}
	}

	return outdated, nil
}

func installFormula(cfg *config.Config, formulaName string) error {
	// Use the install command functionality
	opts := &installer.Options{
		BuildFromSource:    false,
		ForceBottle:        false,
		IgnoreDependencies: false,
		OnlyDependencies:   false,
		IncludeTest:        false,
		HeadOnly:           false,
		KeepTmp:            false,
		DebugSymbols:       false,
		Force:              false,
		DryRun:             false,
		Verbose:            cfg.Verbose,
	}

	inst := installer.New(cfg, opts)
	_, err := inst.InstallFormula(formulaName)
	return err
}
