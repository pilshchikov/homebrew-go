package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"github.com/pilshchikov/homebrew-go/internal/api"
	"github.com/pilshchikov/homebrew-go/internal/cask"
	"github.com/pilshchikov/homebrew-go/internal/config"
	"github.com/pilshchikov/homebrew-go/internal/logger"
	"github.com/spf13/cobra"
)

// NewCasksCmd creates the casks command
func NewCasksCmd(cfg *config.Config) *cobra.Command {
	var (
		eval       bool
		jsonOutput bool
		onePerLine bool
	)

	cmd := &cobra.Command{
		Use:   "casks [OPTIONS]",
		Short: "List all locally available casks",
		Long: `List all locally available casks. By default, casks from all installed
taps are listed.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCasks(cfg, &casksOptions{
				eval:       eval,
				jsonOutput: jsonOutput,
				onePerLine: onePerLine,
			})
		},
	}

	cmd.Flags().BoolVar(&eval, "eval-all", false, "Evaluate all casks")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output cask information in JSON format")
	cmd.Flags().BoolVar(&onePerLine, "1", false, "List one cask per line")

	return cmd
}

// NewFormulaeCmd creates the formulae command
func NewFormulaeCmd(cfg *config.Config) *cobra.Command {
	var (
		eval       bool
		jsonOutput bool
		onePerLine bool
	)

	cmd := &cobra.Command{
		Use:   "formulae [OPTIONS]",
		Short: "List all locally available formulae",
		Long: `List all locally available formulae. By default, formulae from all 
installed taps are listed.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runFormulae(cfg, &formulaeOptions{
				eval:       eval,
				jsonOutput: jsonOutput,
				onePerLine: onePerLine,
			})
		},
	}

	cmd.Flags().BoolVar(&eval, "eval-all", false, "Evaluate all formulae")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output formula information in JSON format")
	cmd.Flags().BoolVar(&onePerLine, "1", false, "List one formula per line")

	return cmd
}

// NewCommandsCmd creates the commands command
func NewCommandsCmd(cfg *config.Config) *cobra.Command {
	var (
		quiet    bool
		include  []string
		builtin  bool
		external bool
	)

	cmd := &cobra.Command{
		Use:   "commands [OPTIONS]",
		Short: "Show lists of built-in and external commands",
		Long: `Show lists of built-in and external commands. Built-in commands are
part of Homebrew itself, while external commands are scripts in the PATH
that start with 'brew-'.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCommands(cfg, &commandsOptions{
				quiet:    quiet,
				include:  include,
				builtin:  builtin,
				external: external,
			})
		},
	}

	cmd.Flags().BoolVarP(&quiet, "quiet", "q", false, "List only the names of commands")
	cmd.Flags().StringSliceVar(&include, "include-aliases", nil, "Include command aliases")
	cmd.Flags().BoolVar(&builtin, "builtin", false, "Show only built-in commands")
	cmd.Flags().BoolVar(&external, "external", false, "Show only external commands")

	return cmd
}

type casksOptions struct {
	eval       bool
	jsonOutput bool
	onePerLine bool
}

type formulaeOptions struct {
	eval       bool
	jsonOutput bool
	onePerLine bool
}

type commandsOptions struct {
	quiet    bool
	include  []string
	builtin  bool
	external bool
}

func runCasks(cfg *config.Config, opts *casksOptions) error {
	logger.Debug("Listing available casks...")

	client := api.NewClient(cfg)

	// Get all available casks - this is a simplified implementation
	// In practice, this would scan all tapped repositories
	casks, err := client.SearchCasks("")
	if err != nil {
		return fmt.Errorf("failed to get casks: %w", err)
	}

	if opts.jsonOutput {
		return outputCasksJSON(casks)
	}

	// Sort casks by name
	sort.Slice(casks, func(i, j int) bool {
		return casks[i].Token < casks[j].Token
	})

	if opts.onePerLine {
		for _, cask := range casks {
			fmt.Println(cask.Token)
		}
	} else {
		// Print in columns
		var names []string
		for _, cask := range casks {
			names = append(names, cask.Token)
		}
		printColumns(names, 80)
	}

	return nil
}

func runFormulae(cfg *config.Config, opts *formulaeOptions) error {
	logger.Debug("Listing available formulae...")

	client := api.NewClient(cfg)

	// Get all available formulae
	formulae, err := client.SearchFormulae("")
	if err != nil {
		return fmt.Errorf("failed to get formulae: %w", err)
	}

	if opts.jsonOutput {
		return outputFormulaeJSON(formulae)
	}

	// Sort formulae by name
	sort.Slice(formulae, func(i, j int) bool {
		return formulae[i].Name < formulae[j].Name
	})

	if opts.onePerLine {
		for _, formula := range formulae {
			fmt.Println(formula.Name)
		}
	} else {
		// Print in columns
		var names []string
		for _, formula := range formulae {
			names = append(names, formula.Name)
		}
		printColumns(names, 80)
	}

	return nil
}

func runCommands(cfg *config.Config, opts *commandsOptions) error {
	builtinCommands := getBuiltinCommands()
	externalCommands := getExternalCommands()

	if opts.builtin && !opts.external {
		return printCommands(builtinCommands, "Built-in commands", opts.quiet)
	}

	if opts.external && !opts.builtin {
		return printCommands(externalCommands, "External commands", opts.quiet)
	}

	// Show both by default
	if err := printCommands(builtinCommands, "Built-in commands", opts.quiet); err != nil {
		return err
	}

	if len(externalCommands) > 0 {
		if !opts.quiet {
			fmt.Println()
		}
		return printCommands(externalCommands, "External commands", opts.quiet)
	}

	return nil
}

func getBuiltinCommands() []string {
	// List of built-in commands
	return []string{
		"analytics",
		"autoremove",
		"cleanup",
		"commands",
		"config",
		"deps",
		"desc",
		"doctor",
		"env",
		"home",
		"info",
		"install",
		"leaves",
		"link",
		"list",
		"missing",
		"options",
		"outdated",
		"pin",
		"reinstall",
		"search",
		"services",
		"tap",
		"tap-info",
		"uninstall",
		"unlink",
		"unpin",
		"untap",
		"update",
		"upgrade",
		"uses",
		"--cache",
		"--cellar",
		"--env",
		"--prefix",
		"--repository",
		"--version",
	}
}

func getExternalCommands() []string {
	// In a real implementation, this would scan PATH for brew-* scripts
	// For now, return empty list
	return []string{}
}

func printCommands(commands []string, title string, quiet bool) error {
	if !quiet {
		fmt.Printf("%s:\n", title)
	}

	sort.Strings(commands)

	if quiet {
		for _, cmd := range commands {
			fmt.Println(cmd)
		}
	} else {
		printColumns(commands, 80)
	}

	return nil
}

func outputCasksJSON(casks []*cask.Cask) error {
	// Convert to a simpler structure for JSON output
	var jsonCasks []map[string]interface{}

	for _, cask := range casks {
		jsonCask := map[string]interface{}{
			"token":       cask.Token,
			"name":        cask.Name,
			"homepage":    cask.Homepage,
			"description": cask.Description,
			"version":     cask.Version,
		}
		jsonCasks = append(jsonCasks, jsonCask)
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(jsonCasks)
}

func outputFormulaeJSON(formulae []api.SearchResult) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(formulae)
}

func printColumns(items []string, maxWidth int) {
	if len(items) == 0 {
		return
	}

	// Calculate column width
	maxLen := 0
	for _, item := range items {
		if len(item) > maxLen {
			maxLen = len(item)
		}
	}

	colWidth := maxLen + 2 // Add padding
	cols := maxWidth / colWidth
	if cols < 1 {
		cols = 1
	}

	// Print items in columns
	for i, item := range items {
		fmt.Printf("%-*s", colWidth, item)
		if (i+1)%cols == 0 || i == len(items)-1 {
			fmt.Println()
		}
	}
}
