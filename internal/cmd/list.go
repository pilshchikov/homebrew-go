package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/pilshchikov/homebrew-go/internal/config"
	"github.com/pilshchikov/homebrew-go/internal/logger"
	"github.com/spf13/cobra"
)

// NewListCmd creates the list command
func NewListCmd(cfg *config.Config) *cobra.Command {
	var (
		formulae bool
		casks    bool
		versions bool
		full     bool
	)

	cmd := &cobra.Command{
		Use:     "list [OPTIONS] [FORMULA|CASK...]",
		Aliases: []string{"ls"},
		Short:   "List installed formulae and casks",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return listInstalled(cfg, formulae, casks, versions, full)
			}

			for _, name := range args {
				logger.Progress("Listing files for %s", name)
				if err := listFormulaFiles(cfg, name); err != nil {
					logger.Error("Failed to list files for %s: %v", name, err)
				}
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&formulae, "formulae", false, "List formulae only")
	cmd.Flags().BoolVar(&casks, "casks", false, "List casks only")
	cmd.Flags().BoolVar(&versions, "versions", false, "Show version numbers")
	cmd.Flags().BoolVar(&full, "full-name", false, "Print fully-qualified names")

	return cmd
}

func listInstalled(cfg *config.Config, formulae, casks, versions, full bool) error {
	var formulaeList []string
	var casksList []string

	// Get installed formulae
	if !casks {
		files, err := os.ReadDir(cfg.HomebrewCellar)
		if err != nil {
			return fmt.Errorf("failed to read cellar: %w", err)
		}

		for _, file := range files {
			if file.IsDir() {
				name := file.Name()
				if full {
					name = "homebrew/core/" + name
				}

				if versions {
					// List versions
					versionDirs, err := os.ReadDir(filepath.Join(cfg.HomebrewCellar, file.Name()))
					if err == nil {
						for _, versionDir := range versionDirs {
							if versionDir.IsDir() {
								formulaeList = append(formulaeList, fmt.Sprintf("%s %s", name, versionDir.Name()))
							}
						}
					}
				} else {
					formulaeList = append(formulaeList, name)
				}
			}
		}
	}

	// Get installed casks
	if !formulae {
		if files, err := os.ReadDir(cfg.HomebrewCaskroom); err == nil {
			for _, file := range files {
				if file.IsDir() {
					name := file.Name()
					if full {
						name = "homebrew/cask/" + name
					}
					casksList = append(casksList, name)
				}
			}
		}
	}

	// Sort lists
	sort.Strings(formulaeList)
	sort.Strings(casksList)

	// Output in Homebrew format
	if len(formulaeList) > 0 {
		fmt.Printf("==> Formulae\n")
		printColumns(formulaeList, 4) // 4 columns like original
	}

	if len(casksList) > 0 {
		if len(formulaeList) > 0 {
			fmt.Println() // Empty line between sections
		}
		fmt.Printf("==> Casks\n")
		printColumns(casksList, 4)
	}

	if len(formulaeList) == 0 && len(casksList) == 0 {
		fmt.Println("No formulae or casks installed.")
	}

	return nil
}

// listFormulaFiles lists all files installed by a specific formula
func listFormulaFiles(cfg *config.Config, name string) error {
	// Check if formula is installed
	formulaDir := filepath.Join(cfg.HomebrewCellar, name)
	if _, err := os.Stat(formulaDir); os.IsNotExist(err) {
		return fmt.Errorf("formula %s is not installed", name)
	}

	// Find all versions
	versions, err := os.ReadDir(formulaDir)
	if err != nil {
		return fmt.Errorf("failed to read formula directory: %w", err)
	}

	if len(versions) == 0 {
		return fmt.Errorf("no versions found for formula %s", name)
	}

	// Use the latest version (last alphabetically sorted)
	var latestVersion string
	for _, version := range versions {
		if version.IsDir() {
			latestVersion = version.Name()
		}
	}

	if latestVersion == "" {
		return fmt.Errorf("no valid version found for formula %s", name)
	}

	versionDir := filepath.Join(formulaDir, latestVersion)
	logger.Info("%s/%s:", name, latestVersion)

	// Walk through all files and directories
	return filepath.Walk(versionDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Get relative path from version directory
		relPath, err := filepath.Rel(versionDir, path)
		if err != nil {
			return err
		}

		// Skip the root directory
		if relPath == "." {
			return nil
		}

		// Format output based on file type
		if info.IsDir() {
			fmt.Printf("  %s/ (%d items)\n", relPath, countDirItems(path))
		} else {
			fmt.Printf("  %s (%s)\n", relPath, formatFileSize(info.Size()))
		}

		return nil
	})
}

// countDirItems counts the number of items in a directory
func countDirItems(dirPath string) int {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return 0
	}
	return len(entries)
}

// formatFileSize formats file size in human-readable format
func formatFileSize(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(size)/float64(div), "KMGTPE"[exp])
}
