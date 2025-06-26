package cmd

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/pilshchikov/homebrew-go/internal/config"
	"github.com/pilshchikov/homebrew-go/internal/logger"
	"github.com/spf13/cobra"
)

// NewCleanupCmd creates the cleanup command
func NewCleanupCmd(cfg *config.Config) *cobra.Command {
	var (
		dryRun bool
		prune  string
	)

	cmd := &cobra.Command{
		Use:   "cleanup [OPTIONS] [FORMULA|CASK...]",
		Short: "Remove stale lock files and outdated downloads",
		RunE: func(cmd *cobra.Command, args []string) error {
			logger.Progress("Running cleanup")
			return runCleanup(cfg, dryRun)
		},
	}

	cmd.Flags().BoolVarP(&dryRun, "dry-run", "n", false, "Show what would be removed")
	cmd.Flags().StringVar(&prune, "prune", "0", "Remove all cache files older than specified days")

	return cmd
}

// runCleanup removes old downloads, cache files, and outdated versions
func runCleanup(cfg *config.Config, dryRun bool) error {
	var totalFreed int64
	var itemsRemoved int

	logger.Step("Removing old downloads from cache")
	cacheFreed, cacheItems, err := cleanupCache(cfg.HomebrewCache, dryRun)
	if err != nil {
		logger.Warn("Failed to cleanup cache: %v", err)
	} else {
		totalFreed += cacheFreed
		itemsRemoved += cacheItems
	}

	logger.Step("Removing outdated versions")
	cellarFreed, cellarItems, err := cleanupCellar(cfg.HomebrewCellar, dryRun)
	if err != nil {
		logger.Warn("Failed to cleanup cellar: %v", err)
	} else {
		totalFreed += cellarFreed
		itemsRemoved += cellarItems
	}

	logger.Step("Removing lock files")
	lockItems, err := cleanupLockFiles(cfg, dryRun)
	if err != nil {
		logger.Warn("Failed to cleanup lock files: %v", err)
	} else {
		itemsRemoved += lockItems
	}

	if dryRun {
		logger.Info("Would remove %d items, freeing %s", itemsRemoved, formatFileSize(totalFreed))
	} else {
		logger.Success("Removed %d items, freed %s", itemsRemoved, formatFileSize(totalFreed))
	}

	return nil
}

// cleanupCache removes old cache files
func cleanupCache(cacheDir string, dryRun bool) (int64, int, error) {
	if _, err := os.Stat(cacheDir); os.IsNotExist(err) {
		return 0, 0, nil
	}

	var totalSize int64
	var itemCount int

	err := filepath.Walk(cacheDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		// Remove files older than 30 days
		if info.ModTime().AddDate(0, 0, 30).Before(time.Now()) {
			totalSize += info.Size()
			itemCount++

			if dryRun {
				logger.Debug("Would remove: %s (%s)", path, formatFileSize(info.Size()))
			} else {
				if err := os.Remove(path); err != nil {
					logger.Debug("Failed to remove %s: %v", path, err)
				} else {
					logger.Debug("Removed: %s (%s)", path, formatFileSize(info.Size()))
				}
			}
		}

		return nil
	})

	return totalSize, itemCount, err
}

// cleanupCellar removes outdated formula versions (keeps latest 2)
func cleanupCellar(cellarDir string, dryRun bool) (int64, int, error) {
	if _, err := os.Stat(cellarDir); os.IsNotExist(err) {
		return 0, 0, nil
	}

	var totalSize int64
	var itemCount int

	formulae, err := os.ReadDir(cellarDir)
	if err != nil {
		return 0, 0, err
	}

	for _, formula := range formulae {
		if !formula.IsDir() {
			continue
		}

		formulaPath := filepath.Join(cellarDir, formula.Name())
		versions, err := os.ReadDir(formulaPath)
		if err != nil {
			continue
		}

		// Sort versions and keep only the latest 2
		if len(versions) > 2 {
			sort.Slice(versions, func(i, j int) bool {
				return versions[i].Name() < versions[j].Name()
			})

			// Remove old versions (keep last 2)
			for i := 0; i < len(versions)-2; i++ {
				versionPath := filepath.Join(formulaPath, versions[i].Name())

				// Calculate size
				size, err := dirSize(versionPath)
				if err == nil {
					totalSize += size
					itemCount++

					if dryRun {
						logger.Debug("Would remove: %s/%s (%s)", formula.Name(), versions[i].Name(), formatFileSize(size))
					} else {
						if err := os.RemoveAll(versionPath); err != nil {
							logger.Debug("Failed to remove %s: %v", versionPath, err)
						} else {
							logger.Debug("Removed: %s/%s (%s)", formula.Name(), versions[i].Name(), formatFileSize(size))
						}
					}
				}
			}
		}
	}

	return totalSize, itemCount, nil
}

// cleanupLockFiles removes stale lock files
func cleanupLockFiles(cfg *config.Config, dryRun bool) (int, error) {
	lockDirs := []string{
		cfg.HomebrewPrefix,
		cfg.HomebrewCache,
		"/tmp",
	}

	var itemCount int

	for _, dir := range lockDirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			continue
		}

		err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil // Continue on error
			}

			if strings.HasSuffix(path, ".lock") || strings.HasSuffix(path, ".tmp") {
				// Check if lock file is stale (older than 1 hour)
				if info.ModTime().Add(time.Hour).Before(time.Now()) {
					itemCount++

					if dryRun {
						logger.Debug("Would remove stale lock: %s", path)
					} else {
						if err := os.Remove(path); err != nil {
							logger.Debug("Failed to remove lock file %s: %v", path, err)
						} else {
							logger.Debug("Removed stale lock: %s", path)
						}
					}
				}
			}

			return nil
		})

		if err != nil {
			logger.Debug("Error walking %s: %v", dir, err)
		}
	}

	return itemCount, nil
}

// dirSize calculates the total size of a directory
func dirSize(path string) (int64, error) {
	var size int64

	err := filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})

	return size, err
}
