package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/homebrew/brew/internal/api"
	"github.com/homebrew/brew/internal/config"
	"github.com/homebrew/brew/internal/logger"
	"github.com/spf13/cobra"
)

// NewOutdatedCmd creates the outdated command
func NewOutdatedCmd(cfg *config.Config) *cobra.Command {
	var (
		jsonOutput bool
		greedy     bool
		verbose    bool
		fetchHead  bool
		quiet      bool
		cask       bool
	)

	cmd := &cobra.Command{
		Use:   "outdated [OPTIONS] [FORMULA|CASK...]",
		Short: "List installed formulae and casks that have a more recent version available",
		Long: `List installed formulae and casks that have a more recent version available.

By default, version information is displayed in interactive shells, and
suppressed otherwise.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runOutdated(cfg, args, &outdatedOptions{
				jsonOutput: jsonOutput,
				greedy:     greedy,
				verbose:    verbose,
				fetchHead:  fetchHead,
				quiet:      quiet,
				cask:       cask,
			})
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Print a JSON representation of the outdated formulae")
	cmd.Flags().BoolVar(&greedy, "greedy", false, "Check for outdated casks even if auto_updates is true")
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Include detailed version information")
	cmd.Flags().BoolVar(&fetchHead, "fetch-HEAD", false, "Detect if the HEAD installation is outdated")
	cmd.Flags().BoolVarP(&quiet, "quiet", "q", false, "List only the names of outdated kegs")
	cmd.Flags().BoolVar(&cask, "cask", false, "List only outdated casks")

	return cmd
}

type outdatedOptions struct {
	jsonOutput bool
	greedy     bool
	verbose    bool
	fetchHead  bool
	quiet      bool
	cask       bool
}

type OutdatedInfo struct {
	Name              string   `json:"name"`
	InstalledVersions []string `json:"installed_versions"`
	CurrentVersion    string   `json:"current_version"`
	PinnedVersion     string   `json:"pinned_version,omitempty"`
	Outdated          bool     `json:"outdated"`
}

func runOutdated(cfg *config.Config, formulaNames []string, opts *outdatedOptions) error {
	var outdatedItems []OutdatedInfo

	if opts.cask {
		// Handle casks
		outdatedCasks, err := getOutdatedCasks(cfg, formulaNames, opts)
		if err != nil {
			return fmt.Errorf("failed to get outdated casks: %w", err)
		}
		outdatedItems = append(outdatedItems, outdatedCasks...)
	} else {
		// Handle formulae
		outdatedFormulae, err := getOutdatedFormulae(cfg, formulaNames, opts)
		if err != nil {
			return fmt.Errorf("failed to get outdated formulae: %w", err)
		}
		outdatedItems = append(outdatedItems, outdatedFormulae...)
	}

	if opts.jsonOutput {
		return outputJSON(outdatedItems)
	}

	return outputText(outdatedItems, opts)
}

func getOutdatedFormulae(cfg *config.Config, formulaNames []string, opts *outdatedOptions) ([]OutdatedInfo, error) {
	var outdatedFormulae []OutdatedInfo

	// Get list of installed formulae
	var formulaeToCheck []string
	if len(formulaNames) > 0 {
		formulaeToCheck = formulaNames
	} else {
		installed, err := getInstalledFormulae(cfg)
		if err != nil {
			return nil, err
		}
		formulaeToCheck = installed
	}

	client := api.NewClient(cfg)

	for _, formulaName := range formulaeToCheck {
		// Get installed versions
		installedVersions, err := getInstalledVersions(cfg, formulaName)
		if err != nil {
			logger.Debug("Failed to get installed versions for %s: %v", formulaName, err)
			continue
		}

		if len(installedVersions) == 0 {
			continue // Not installed
		}

		// Get current version from API
		currentFormula, err := client.GetFormula(formulaName)
		if err != nil {
			logger.Debug("Failed to get current version for %s: %v", formulaName, err)
			continue
		}

		// Check if outdated
		latestInstalled := getLatestVersion(installedVersions)
		isOutdated := isVersionOutdated(latestInstalled, currentFormula.Version)

		if isOutdated || opts.verbose {
			info := OutdatedInfo{
				Name:              formulaName,
				InstalledVersions: installedVersions,
				CurrentVersion:    currentFormula.Version,
				Outdated:          isOutdated,
			}

			// Check if pinned
			if isPinned(cfg, formulaName) {
				info.PinnedVersion = latestInstalled
			}

			if isOutdated || opts.verbose {
				outdatedFormulae = append(outdatedFormulae, info)
			}
		}
	}

	return outdatedFormulae, nil
}

func getOutdatedCasks(cfg *config.Config, caskNames []string, opts *outdatedOptions) ([]OutdatedInfo, error) {
	var outdatedCasks []OutdatedInfo

	// For now, return empty list as cask outdated detection is complex
	logger.Debug("Cask outdated detection not fully implemented")

	return outdatedCasks, nil
}

func getInstalledVersions(cfg *config.Config, formulaName string) ([]string, error) {
	formulaPath := filepath.Join(cfg.HomebrewCellar, formulaName)

	entries, err := os.ReadDir(formulaPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}

	var versions []string
	for _, entry := range entries {
		if entry.IsDir() {
			versions = append(versions, entry.Name())
		}
	}

	return versions, nil
}

func getLatestVersion(versions []string) string {
	if len(versions) == 0 {
		return ""
	}

	// Simple string comparison - in a full implementation,
	// this would use proper version comparison
	sort.Strings(versions)
	return versions[len(versions)-1]
}

func isVersionOutdated(installed, current string) bool {
	// Simplified version comparison
	// In a full implementation, this would use semantic version comparison
	return installed != current && current != ""
}

func isPinned(cfg *config.Config, formulaName string) bool {
	// Check if formula is pinned
	pinFile := filepath.Join(cfg.HomebrewLibrary, "PinnedKegs", formulaName)
	_, err := os.Stat(pinFile)
	return err == nil
}

func outputJSON(outdatedItems []OutdatedInfo) error {
	// Filter to only outdated items for JSON output
	var outdated []OutdatedInfo
	for _, item := range outdatedItems {
		if item.Outdated {
			outdated = append(outdated, item)
		}
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(outdated)
}

func outputText(outdatedItems []OutdatedInfo, opts *outdatedOptions) error {
	// Filter to only outdated items
	var outdated []OutdatedInfo
	for _, item := range outdatedItems {
		if item.Outdated {
			outdated = append(outdated, item)
		}
	}

	if len(outdated) == 0 {
		if !opts.quiet {
			logger.Info("No outdated formulae")
		}
		return nil
	}

	// Sort by name
	sort.Slice(outdated, func(i, j int) bool {
		return outdated[i].Name < outdated[j].Name
	})

	for _, item := range outdated {
		if opts.quiet {
			fmt.Println(item.Name)
		} else if opts.verbose {
			installedVersionsStr := strings.Join(item.InstalledVersions, ", ")
			fmt.Printf("%s (%s) < %s", item.Name, installedVersionsStr, item.CurrentVersion)
			if item.PinnedVersion != "" {
				fmt.Printf(" [pinned at %s]", item.PinnedVersion)
			}
			fmt.Println()
		} else {
			fmt.Printf("%s (%s) < %s\n", item.Name,
				item.InstalledVersions[len(item.InstalledVersions)-1],
				item.CurrentVersion)
		}
	}

	return nil
}
