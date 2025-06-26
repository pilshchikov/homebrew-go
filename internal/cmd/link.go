package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pilshchikov/homebrew-go/internal/config"
	"github.com/pilshchikov/homebrew-go/internal/logger"
	"github.com/spf13/cobra"
)

// NewLinkCmd creates the link command
func NewLinkCmd(cfg *config.Config) *cobra.Command {
	var (
		overwrite bool
		dryRun    bool
		force     bool
	)

	cmd := &cobra.Command{
		Use:   "link [OPTIONS] FORMULA...",
		Short: "Symlink all of a formula's installed files into Homebrew's prefix",
		Long: `Symlink all of a formula's installed files into Homebrew's prefix. This is
done automatically when you install formulae but can be useful if you need to
re-link after certain changes.

If the formula is keg-only, its files will not be symlinked into the prefix.`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLink(cfg, args, &linkOptions{
				overwrite: overwrite,
				dryRun:    dryRun,
				force:     force,
			})
		},
	}

	cmd.Flags().BoolVar(&overwrite, "overwrite", false, "Overwrite existing symlinks")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be linked without actually linking")
	cmd.Flags().BoolVar(&force, "force", false, "Allow keg-only formulae to be linked")

	return cmd
}

// NewUnlinkCmd creates the unlink command
func NewUnlinkCmd(cfg *config.Config) *cobra.Command {
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "unlink [OPTIONS] FORMULA...",
		Short: "Remove symlinks for a formula's installed files from Homebrew's prefix",
		Long: `Remove symlinks for a formula's installed files from Homebrew's prefix.
This will not delete the installed files, only the symlinks to them.

This can be useful for temporarily disabling a formula or resolving conflicts.`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUnlink(cfg, args, &unlinkOptions{
				dryRun: dryRun,
			})
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be unlinked without actually unlinking")

	return cmd
}

type linkOptions struct {
	overwrite bool
	dryRun    bool
	force     bool
}

type unlinkOptions struct {
	dryRun bool
}

func runLink(cfg *config.Config, formulaNames []string, opts *linkOptions) error {
	var linked []string
	var errors []string

	for _, formulaName := range formulaNames {
		if !isFormulaInstalledSimple(cfg, formulaName) {
			errors = append(errors, fmt.Sprintf("Formula %s is not installed", formulaName))
			continue
		}

		// Check if formula is keg-only
		if isKegOnly(cfg, formulaName) && !opts.force {
			logger.Info("Formula %s is keg-only and won't be linked unless forced", formulaName)
			continue
		}

		if err := linkFormula(cfg, formulaName, opts); err != nil {
			errors = append(errors, fmt.Sprintf("Failed to link %s: %v", formulaName, err))
			continue
		}

		linked = append(linked, formulaName)
		if !opts.dryRun {
			logger.Success("Linked %s", formulaName)
		}
	}

	// Report results
	if len(linked) > 0 {
		if opts.dryRun {
			logger.Info("Would link formulae: %v", linked)
		} else {
			logger.Info("Linked formulae: %v", linked)
		}
	}

	if len(errors) > 0 {
		for _, errMsg := range errors {
			logger.Error(errMsg)
		}
		return fmt.Errorf("failed to link some formulae")
	}

	return nil
}

func runUnlink(cfg *config.Config, formulaNames []string, opts *unlinkOptions) error {
	var unlinked []string
	var errors []string

	for _, formulaName := range formulaNames {
		if !isFormulaInstalledSimple(cfg, formulaName) {
			errors = append(errors, fmt.Sprintf("Formula %s is not installed", formulaName))
			continue
		}

		if err := unlinkFormulaLink(cfg, formulaName, opts); err != nil {
			errors = append(errors, fmt.Sprintf("Failed to unlink %s: %v", formulaName, err))
			continue
		}

		unlinked = append(unlinked, formulaName)
		if !opts.dryRun {
			logger.Success("Unlinked %s", formulaName)
		}
	}

	// Report results
	if len(unlinked) > 0 {
		if opts.dryRun {
			logger.Info("Would unlink formulae: %v", unlinked)
		} else {
			logger.Info("Unlinked formulae: %v", unlinked)
		}
	}

	if len(errors) > 0 {
		for _, errMsg := range errors {
			logger.Error(errMsg)
		}
		return fmt.Errorf("failed to unlink some formulae")
	}

	return nil
}

func linkFormula(cfg *config.Config, formulaName string, opts *linkOptions) error {
	// Get the latest installed version
	installedVersions, err := getInstalledVersions(cfg, formulaName)
	if err != nil {
		return err
	}

	if len(installedVersions) == 0 {
		return fmt.Errorf("no installed versions found")
	}

	latestVersion := getLatestVersion(installedVersions)
	formulaPath := filepath.Join(cfg.HomebrewCellar, formulaName, latestVersion)

	logger.Debug("Linking %s from %s", formulaName, formulaPath)

	// Link common directories
	return linkDirectories(cfg, formulaPath, opts)
}

func unlinkFormulaLink(cfg *config.Config, formulaName string, opts *unlinkOptions) error {
	// Find symlinks in prefix that point to this formula
	symlinks, err := findFormulaSymlinks(cfg, formulaName)
	if err != nil {
		return err
	}

	if len(symlinks) == 0 {
		logger.Info("No symlinks found for %s", formulaName)
		return nil
	}

	logger.Debug("Found %d symlinks for %s", len(symlinks), formulaName)

	for _, symlinkPath := range symlinks {
		if opts.dryRun {
			logger.Info("Would remove: %s", symlinkPath)
		} else {
			logger.Debug("Removing symlink: %s", symlinkPath)
			if err := os.Remove(symlinkPath); err != nil {
				logger.Warn("Failed to remove symlink %s: %v", symlinkPath, err)
			}
		}
	}

	return nil
}

func linkDirectories(cfg *config.Config, sourcePath string, opts *linkOptions) error {
	// Common directories to link
	linkDirs := map[string]string{
		"bin":     "bin",
		"sbin":    "sbin",
		"lib":     "lib",
		"include": "include",
		"share":   "share",
		"etc":     "etc",
	}

	for sourceDir, targetDir := range linkDirs {
		sourceDirPath := filepath.Join(sourcePath, sourceDir)
		targetDirPath := filepath.Join(cfg.HomebrewPrefix, targetDir)

		if _, err := os.Stat(sourceDirPath); os.IsNotExist(err) {
			continue // Source directory doesn't exist, skip
		}

		if err := linkDirectory(sourceDirPath, targetDirPath, opts); err != nil {
			return fmt.Errorf("failed to link %s: %w", sourceDir, err)
		}
	}

	return nil
}

func linkDirectory(sourceDir, targetDir string, opts *linkOptions) error {
	// Ensure target directory exists
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return err
	}

	// Walk source directory and create symlinks
	return filepath.Walk(sourceDir, func(sourcePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip the root directory
		if sourcePath == sourceDir {
			return nil
		}

		// Calculate relative path and target path
		relPath, err := filepath.Rel(sourceDir, sourcePath)
		if err != nil {
			return err
		}

		targetPath := filepath.Join(targetDir, relPath)

		if info.IsDir() {
			// Create directory in target
			if !opts.dryRun {
				return os.MkdirAll(targetPath, info.Mode())
			}
			return nil
		}

		// Create symlink for files
		return createSymlink(sourcePath, targetPath, opts)
	})
}

func createSymlink(sourcePath, targetPath string, opts *linkOptions) error {
	// Check if target already exists
	if _, err := os.Lstat(targetPath); err == nil {
		if !opts.overwrite {
			logger.Debug("Skipping existing file: %s", targetPath)
			return nil
		}

		if !opts.dryRun {
			if err := os.Remove(targetPath); err != nil {
				return fmt.Errorf("failed to remove existing file %s: %w", targetPath, err)
			}
		}
	}

	if opts.dryRun {
		logger.Info("Would link: %s -> %s", targetPath, sourcePath)
		return nil
	}

	logger.Debug("Creating symlink: %s -> %s", targetPath, sourcePath)
	return os.Symlink(sourcePath, targetPath)
}

func findFormulaSymlinks(cfg *config.Config, formulaName string) ([]string, error) {
	var symlinks []string
	formulaCellarPath := filepath.Join(cfg.HomebrewCellar, formulaName)

	// Search common directories for symlinks pointing to this formula
	searchDirs := []string{
		filepath.Join(cfg.HomebrewPrefix, "bin"),
		filepath.Join(cfg.HomebrewPrefix, "sbin"),
		filepath.Join(cfg.HomebrewPrefix, "lib"),
		filepath.Join(cfg.HomebrewPrefix, "include"),
		filepath.Join(cfg.HomebrewPrefix, "share"),
		filepath.Join(cfg.HomebrewPrefix, "etc"),
	}

	for _, searchDir := range searchDirs {
		if _, err := os.Stat(searchDir); os.IsNotExist(err) {
			continue
		}

		err := filepath.Walk(searchDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil // Continue on errors
			}

			// Check if it's a symlink
			if info.Mode()&os.ModeSymlink != 0 {
				target, err := os.Readlink(path)
				if err != nil {
					return nil
				}

				// Check if symlink points to our formula
				if strings.Contains(target, formulaCellarPath) {
					symlinks = append(symlinks, path)
				}
			}

			return nil
		})

		if err != nil {
			logger.Debug("Error walking directory %s: %v", searchDir, err)
		}
	}

	return symlinks, nil
}

func isKegOnly(cfg *config.Config, formulaName string) bool {
	// Check if formula is keg-only
	// This would typically be determined by the formula definition
	// For now, return false as a placeholder
	return false
}

func isFormulaInstalledSimple(cfg *config.Config, formulaName string) bool {
	formulaPath := filepath.Join(cfg.HomebrewCellar, formulaName)
	_, err := os.Stat(formulaPath)
	return err == nil
}
