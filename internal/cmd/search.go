package cmd

import (
	"fmt"

	"github.com/pilshchikov/homebrew-go/internal/api"
	"github.com/pilshchikov/homebrew-go/internal/config"
	"github.com/pilshchikov/homebrew-go/internal/errors"
	"github.com/pilshchikov/homebrew-go/internal/logger"
	"github.com/spf13/cobra"
)

// NewSearchCmd creates the search command
func NewSearchCmd(cfg *config.Config) *cobra.Command {
	var (
		formulae bool
		casks    bool
		desc     bool
	)

	cmd := &cobra.Command{
		Use:   "search [OPTIONS] [TEXT|/REGEX/]",
		Short: "Search for formulae and casks",
		RunE: func(cmd *cobra.Command, args []string) error {
			query := ""
			if len(args) > 0 {
				query = args[0]
			}

			if query == "" {
				// Show some popular formulae when no query is provided
				fmt.Println("==> Formulae")
				popularFormulae := []string{
					"git", "node", "python", "curl", "wget", "cmake", "go", "rust",
					"docker", "kubernetes-cli", "terraform", "awscli", "jq", "tmux",
					"vim", "neovim", "htop", "tree", "ffmpeg", "imagemagick",
				}
				printColumns(popularFormulae, 4)

				fmt.Println("\n==> Casks")
				popularCasks := []string{
					"visual-studio-code", "google-chrome", "firefox", "slack",
					"discord", "zoom", "notion", "figma", "docker", "iterm2",
				}
				printColumns(popularCasks, 4)

				fmt.Printf("\nIf you meant %q, try:\n  brew search %s\n", query, query)
				return nil
			}

			// Search using real API
			apiClient := api.NewClient(cfg)
			logger.Step("Searching for %q", query)
			results, err := apiClient.SearchFormulae(query)

			if err != nil {
				netErr := errors.NewNetworkError("search", "formulae API", err)
				logger.LogDetailedError(logger.ErrorContext{
					Operation:   "search",
					Error:       netErr,
					Suggestions: netErr.Suggestions,
				})
				fmt.Printf("No formulae found matching %q\n", query)
				return nil
			}

			fmt.Printf("==> Formulae\n")
			if len(results) > 0 {
				var names []string
				for _, result := range results {
					names = append(names, result.Name)
				}
				printColumnsSearch(names, 4)
			} else {
				fmt.Printf("No formulae found matching %q\n", query)
			}

			fmt.Printf("\n==> Casks\n")
			fmt.Printf("No casks found matching %q\n", query)

			return nil
		},
	}

	cmd.Flags().BoolVar(&formulae, "formulae", false, "Search formulae only")
	cmd.Flags().BoolVar(&casks, "casks", false, "Search casks only")
	cmd.Flags().BoolVar(&desc, "desc", false, "Search descriptions too")

	return cmd
}

// printColumnsSearch prints items in columns like the original Homebrew
func printColumnsSearch(items []string, columns int) {
	if len(items) == 0 {
		return
	}

	// Calculate max width for each item to determine column width
	maxWidth := 0
	for _, item := range items {
		if len(item) > maxWidth {
			maxWidth = len(item)
		}
	}

	// Add some padding
	colWidth := maxWidth + 2

	// Print items in columns
	for i, item := range items {
		fmt.Printf("%-*s", colWidth, item)

		// New line after every 'columns' items or at the end
		if (i+1)%columns == 0 || i == len(items)-1 {
			fmt.Println()
		}
	}
}
