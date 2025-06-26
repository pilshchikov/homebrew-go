package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/homebrew/brew/internal/config"
	"github.com/homebrew/brew/internal/logger"
	"github.com/spf13/cobra"
)

// NewUninstallCmd creates the uninstall command
func NewUninstallCmd(cfg *config.Config) *cobra.Command {
	var (
		force      bool
		ignoreDeps bool
		zap        bool
	)

	cmd := &cobra.Command{
		Use:     "uninstall [OPTIONS] FORMULA|CASK...",
		Aliases: []string{"remove", "rm"},
		Short:   "Uninstall a formula or cask",
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUninstall(cfg, args, &uninstallOptions{
				Force:      force,
				IgnoreDeps: ignoreDeps,
				Zap:        zap,
			})
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Delete all installed versions")
	cmd.Flags().BoolVar(&ignoreDeps, "ignore-dependencies", false, "Don't fail uninstall if dependencies would be left")
	cmd.Flags().BoolVar(&zap, "zap", false, "Remove all files associated with a cask")

	return cmd
}

type uninstallOptions struct {
	Force      bool
	IgnoreDeps bool
	Zap        bool
}

func runUninstall(cfg *config.Config, args []string, opts *uninstallOptions) error {
	if len(args) == 0 {
		return fmt.Errorf("no formulae specified for uninstall")
	}

	for _, formulaName := range args {
		logger.PrintHeader(fmt.Sprintf("Uninstalling: %s", formulaName))

		// Check if formula is installed
		logger.Step("Checking if %s is installed", formulaName)
		installed, err := isFormulaInstalled(cfg, formulaName)
		if err != nil {
			return fmt.Errorf("failed to check if %s is installed: %w", formulaName, err)
		}

		if !installed {
			if opts.Force {
				logger.Warn("Formula %s is not installed", formulaName)
				continue
			} else {
				return fmt.Errorf("formula %s is not installed", formulaName)
			}
		}

		// Get installed version info
		version, err := getInstalledVersion(cfg, formulaName)
		if err == nil && version != "" {
			logger.Info("Found installed version: %s", version)
		}

		// Check for dependents if not ignoring dependencies
		if !opts.IgnoreDeps {
			logger.Step("Checking for dependents")
			dependents, err := findDependents(cfg, formulaName)
			if err != nil {
				return fmt.Errorf("failed to find dependents of %s: %w", formulaName, err)
			}

			if len(dependents) > 0 {
				logger.Warn("Formula %s is required by: %s", formulaName, strings.Join(dependents, ", "))
				return fmt.Errorf("cannot uninstall %s because it is required by: %s",
					formulaName, strings.Join(dependents, ", "))
			} else {
				logger.Debug("No dependents found")
			}
		}

		// Unlink formula
		logger.Step("Unlinking %s", formulaName)
		if err := unlinkFormulaUninstall(cfg, formulaName); err != nil {
			logger.Warn("Failed to unlink %s: %v", formulaName, err)
		} else {
			logger.Debug("Successfully unlinked %s", formulaName)
		}

		// Remove formula directory
		logger.Step("Removing %s files", formulaName)
		if err := removeFormula(cfg, formulaName); err != nil {
			return fmt.Errorf("failed to remove %s: %w", formulaName, err)
		}
		logger.Debug("Removed installation directory")

		logger.Success("Successfully uninstalled %s", formulaName)
	}

	return nil
}

func getInstalledVersion(cfg *config.Config, formulaName string) (string, error) {
	formulaDir := filepath.Join(cfg.HomebrewCellar, formulaName)
	entries, err := os.ReadDir(formulaDir)
	if err != nil {
		return "", err
	}

	// Find the first directory (should be the version)
	for _, entry := range entries {
		if entry.IsDir() {
			return entry.Name(), nil
		}
	}

	return "", fmt.Errorf("no version directory found")
}

func findDependents(cfg *config.Config, formulaName string) ([]string, error) {
	var dependents []string

	// Read all installed formulae
	files, err := os.ReadDir(cfg.HomebrewCellar)
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		if !file.IsDir() || file.Name() == formulaName {
			continue
		}

		// Check if this formula depends on the one we want to uninstall
		if depends, err := checkDependency(cfg, file.Name(), formulaName); err == nil && depends {
			dependents = append(dependents, file.Name())
		}
	}

	return dependents, nil
}

func checkDependency(cfg *config.Config, installedFormula, dependency string) (bool, error) {
	// Read install receipt to check dependencies
	receiptPath := filepath.Join(cfg.HomebrewCellar, installedFormula)

	// Find version directory
	versionDirs, err := os.ReadDir(receiptPath)
	if err != nil {
		return false, err
	}

	for _, versionDir := range versionDirs {
		if versionDir.IsDir() {
			receiptFile := filepath.Join(receiptPath, versionDir.Name(), "INSTALL_RECEIPT.json")
			if data, err := os.ReadFile(receiptFile); err == nil {
				var receipt struct {
					Dependencies []string `json:"dependencies"`
				}
				if json.Unmarshal(data, &receipt) == nil {
					for _, dep := range receipt.Dependencies {
						if dep == dependency {
							return true, nil
						}
					}
				}
			}
			break // Only check first version directory
		}
	}

	return false, nil
}

func unlinkFormulaUninstall(cfg *config.Config, formulaName string) error {
	linkDir := filepath.Join(cfg.HomebrewPrefix, "bin")

	// Find and remove symlinks for this formula
	files, err := os.ReadDir(linkDir)
	if err != nil {
		return err
	}

	formulaPrefix := filepath.Join(cfg.HomebrewCellar, formulaName)

	for _, file := range files {
		if file.Type()&os.ModeSymlink != 0 {
			linkPath := filepath.Join(linkDir, file.Name())
			target, err := os.Readlink(linkPath)
			if err == nil && strings.HasPrefix(target, formulaPrefix) {
				logger.Debug("Removing symlink: %s -> %s", linkPath, target)
				os.Remove(linkPath)
			}
		}
	}

	return nil
}

func removeFormula(cfg *config.Config, formulaName string) error {
	formulaPath := filepath.Join(cfg.HomebrewCellar, formulaName)

	// Check what we're about to remove
	if info, err := os.Stat(formulaPath); err == nil {
		if info.IsDir() {
			// Count files/directories being removed
			var fileCount, dirCount int
			err := filepath.WalkDir(formulaPath, func(path string, d os.DirEntry, err error) error {
				if err != nil {
					return nil // Continue on errors
				}
				if d.IsDir() {
					dirCount++
				} else {
					fileCount++
				}
				return nil
			})

			if err == nil && (fileCount > 0 || dirCount > 0) {
				logger.Debug("Removing %d files and %d directories from %s", fileCount, dirCount, formulaPath)
			}
		}
	}

	logger.Debug("Removing formula directory: %s", formulaPath)
	err := os.RemoveAll(formulaPath)
	if err != nil {
		return fmt.Errorf("failed to remove %s: %w", formulaPath, err)
	}

	logger.Debug("Successfully removed %s", formulaPath)
	return nil
}
