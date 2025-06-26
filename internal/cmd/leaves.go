package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/homebrew/brew/internal/config"
	"github.com/homebrew/brew/internal/logger"
	"github.com/spf13/cobra"
)

// NewLeavesCmd creates the leaves command
func NewLeavesCmd(cfg *config.Config) *cobra.Command {
	var (
		installedOnRequest bool
		installedAsDep     bool
	)

	cmd := &cobra.Command{
		Use:   "leaves [OPTIONS]",
		Short: "List installed formulae that are not dependencies of other installed formulae",
		Long: `List installed formulae that are not dependencies of other installed formulae
and were not installed as dependencies.

These are considered "leaves" in the dependency tree - they are the top-level
packages that you explicitly installed.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLeaves(cfg, &leavesOptions{
				installedOnRequest: installedOnRequest,
				installedAsDep:     installedAsDep,
			})
		},
	}

	cmd.Flags().BoolVar(&installedOnRequest, "installed-on-request", false, "Show only formulae installed on request")
	cmd.Flags().BoolVar(&installedAsDep, "installed-as-dependency", false, "Show only formulae installed as dependencies")

	return cmd
}

type leavesOptions struct {
	installedOnRequest bool
	installedAsDep     bool
}

func runLeaves(cfg *config.Config, opts *leavesOptions) error {
	// Get all installed formulae
	installedFormulae, err := getInstalledFormulae(cfg)
	if err != nil {
		return fmt.Errorf("failed to get installed formulae: %w", err)
	}

	if len(installedFormulae) == 0 {
		logger.Info("No formulae installed")
		return nil
	}

	// Build dependency map
	dependencyMap, err := buildDependencyMap(cfg, installedFormulae)
	if err != nil {
		return fmt.Errorf("failed to build dependency map: %w", err)
	}

	// Find leaves (formulae that are not dependencies of others)
	var leaves []string

	for _, formula := range installedFormulae {
		isLeaf := true

		// Check if this formula is a dependency of any other installed formula
		for _, deps := range dependencyMap {
			for _, dep := range deps {
				if dep == formula {
					isLeaf = false
					break
				}
			}
			if !isLeaf {
				break
			}
		}

		if isLeaf {
			// Apply filters
			if opts.installedOnRequest && !isInstalledOnRequest(cfg, formula) {
				continue
			}
			if opts.installedAsDep && !isInstalledAsDependency(cfg, formula) {
				continue
			}

			leaves = append(leaves, formula)
		}
	}

	// Sort and display results
	sort.Strings(leaves)
	for _, leaf := range leaves {
		fmt.Println(leaf)
	}

	if len(leaves) == 0 && (opts.installedOnRequest || opts.installedAsDep) {
		logger.Info("No formulae match the specified criteria")
	}

	return nil
}

func getInstalledFormulae(cfg *config.Config) ([]string, error) {
	cellarPath := cfg.HomebrewCellar

	entries, err := os.ReadDir(cellarPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}

	var installed []string
	for _, entry := range entries {
		if entry.IsDir() {
			// Check if there are any version directories
			formulaPath := filepath.Join(cellarPath, entry.Name())
			versionEntries, err := os.ReadDir(formulaPath)
			if err != nil {
				continue
			}

			hasVersions := false
			for _, versionEntry := range versionEntries {
				if versionEntry.IsDir() {
					hasVersions = true
					break
				}
			}

			if hasVersions {
				installed = append(installed, entry.Name())
			}
		}
	}

	return installed, nil
}

func buildDependencyMap(cfg *config.Config, formulae []string) (map[string][]string, error) {
	dependencyMap := make(map[string][]string)

	// For each installed formula, get its dependencies
	for _, formulaName := range formulae {
		deps, err := getFormulaDependencies(cfg, formulaName)
		if err != nil {
			logger.Debug("Failed to get dependencies for %s: %v", formulaName, err)
			continue
		}
		dependencyMap[formulaName] = deps
	}

	return dependencyMap, nil
}

func getFormulaDependencies(cfg *config.Config, formulaName string) ([]string, error) {
	// Try to read install receipt for dependency information
	// receiptPath := filepath.Join(cfg.HomebrewCellar, formulaName, "INSTALL_RECEIPT.json")

	// For now, return empty dependencies as we don't have receipt parsing implemented
	// In a full implementation, this would parse the JSON receipt file
	return []string{}, nil
}

func isInstalledOnRequest(cfg *config.Config, formulaName string) bool {
	// Check if formula was installed on request (not as a dependency)
	// This would typically be stored in install receipt or tab file
	// For now, assume all installed formulae were requested
	return true
}

func isInstalledAsDependency(cfg *config.Config, formulaName string) bool {
	// Check if formula was installed as a dependency
	// This is the opposite of isInstalledOnRequest
	return !isInstalledOnRequest(cfg, formulaName)
}
