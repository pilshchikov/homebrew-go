package cmd

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/homebrew/brew/internal/api"
	"github.com/homebrew/brew/internal/config"
	"github.com/homebrew/brew/internal/errors"
	"github.com/homebrew/brew/internal/formula"
	"github.com/homebrew/brew/internal/logger"
	"github.com/spf13/cobra"
)

// NewInfoCmd creates the info command
func NewInfoCmd(cfg *config.Config) *cobra.Command {
	var (
		json      bool
		installed bool
		analytics bool
	)

	cmd := &cobra.Command{
		Use:     "info [OPTIONS] [FORMULA|CASK...]",
		Aliases: []string{"abv"},
		Short:   "Display brief statistics for your Homebrew installation",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return showSystemInfo(cfg, json)
			}

			apiClient := api.NewClient(cfg)
			for _, name := range args {
				logger.Step("Getting info for %s", name)

				if formula, err := apiClient.GetFormula(name); err == nil {
					showFormulaInfo(formula, json)
				} else {
					formErr := errors.NewFormulaNotFoundError(name)
					logger.LogDetailedError(logger.ErrorContext{
						Operation:   "formula lookup",
						Formula:     name,
						Error:       formErr,
						Suggestions: formErr.Suggestions,
					})
				}
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&json, "json", false, "Print output in JSON format")
	cmd.Flags().BoolVar(&installed, "installed", false, "Print installed versions only")
	cmd.Flags().BoolVar(&analytics, "analytics", false, "List analytics data")

	return cmd
}

func showSystemInfo(cfg *config.Config, jsonOutput bool) error {
	if jsonOutput {
		// TODO: Implement JSON output
		logger.Info("JSON output not yet implemented")
		return nil
	}

	// Count installed formulae and casks
	formulaeCount := 0
	if files, err := os.ReadDir(cfg.HomebrewCellar); err == nil {
		for _, file := range files {
			if file.IsDir() {
				formulaeCount++
			}
		}
	}

	casksCount := 0
	if files, err := os.ReadDir(cfg.HomebrewCaskroom); err == nil {
		for _, file := range files {
			if file.IsDir() {
				casksCount++
			}
		}
	}

	// Display in Homebrew format
	fmt.Printf("==> Homebrew 3.0.0-go\n")
	fmt.Printf("Homebrew/brew (go revision %s)\n", "unknown")
	fmt.Printf("Homebrew/homebrew-core N/A\n")
	fmt.Printf("Go: %s\n", runtime.Version())
	fmt.Printf("\n")
	fmt.Printf("==> Configuration\n")
	fmt.Printf("HOMEBREW_PREFIX: %s\n", cfg.HomebrewPrefix)
	fmt.Printf("HOMEBREW_REPOSITORY: %s\n", cfg.HomebrewRepository)
	fmt.Printf("HOMEBREW_CELLAR: %s\n", cfg.HomebrewCellar)
	fmt.Printf("HOMEBREW_CASKROOM: %s\n", cfg.HomebrewCaskroom)
	fmt.Printf("\n")
	fmt.Printf("==> Installation\n")

	if formulaeCount > 0 || casksCount > 0 {
		if formulaeCount > 0 {
			fmt.Printf("%d formulae installed\n", formulaeCount)
		}
		if casksCount > 0 {
			fmt.Printf("%d casks installed\n", casksCount)
		}
	} else {
		fmt.Printf("No formulae or casks installed\n")
	}

	return nil
}

func showFormulaInfo(formula *formula.Formula, jsonOutput bool) {
	if jsonOutput {
		// TODO: Implement JSON output
		fmt.Printf("JSON output not yet implemented\n")
		return
	}

	fmt.Printf("==> %s: %s\n", formula.Name, formula.Description)
	fmt.Printf("%s\n", formula.Homepage)
	if formula.License != "" {
		fmt.Printf("License: %s\n", formula.License)
	}
	if formula.Version != "" {
		fmt.Printf("Version: %s\n", formula.Version)
	}

	if len(formula.Dependencies) > 0 {
		fmt.Printf("Dependencies: %s\n", strings.Join(formula.Dependencies, ", "))
	}

	if formula.KegOnly {
		fmt.Printf("This formula is keg-only.\n")
	}

	if formula.Deprecated {
		fmt.Printf("This formula is deprecated.\n")
	}

	if formula.Disabled {
		fmt.Printf("This formula is disabled.\n")
	}

	if formula.Caveats != "" {
		fmt.Printf("\n==> Caveats\n%s\n", formula.Caveats)
	}

	fmt.Println()
}
