package cmd

import (
	"encoding/json"
	"fmt"
	"runtime"

	"github.com/pilshchikov/homebrew-go/internal/config"
	"github.com/spf13/cobra"
)

// NewPrefixCmd creates the --prefix command
func NewPrefixCmd(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:    "prefix",
		Hidden: true,
		Short:  "Display Homebrew's install path",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintln(cmd.OutOrStdout(), cfg.HomebrewPrefix)
			return nil
		},
	}

	return cmd
}

// NewCellarCmd creates the --cellar command
func NewCellarCmd(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:    "cellar",
		Hidden: true,
		Short:  "Display Homebrew's Cellar path",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintln(cmd.OutOrStdout(), cfg.HomebrewCellar)
			return nil
		},
	}

	return cmd
}

// NewCacheCmd creates the --cache command
func NewCacheCmd(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:    "cache",
		Hidden: true,
		Short:  "Display Homebrew's download cache",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintln(cmd.OutOrStdout(), cfg.HomebrewCache)
			return nil
		},
	}

	return cmd
}

// NewEnvCmd creates the env command
func NewEnvCmd(cfg *config.Config) *cobra.Command {
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "env",
		Short: "Show a summary of the Homebrew build environment",
		RunE: func(cmd *cobra.Command, args []string) error {
			return showEnv(cfg, jsonOutput)
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")

	return cmd
}

// EnvironmentInfo represents Homebrew environment information
type EnvironmentInfo struct {
	HomebrewPrefix     string `json:"HOMEBREW_PREFIX"`
	HomebrewRepository string `json:"HOMEBREW_REPOSITORY"`
	HomebrewCellar     string `json:"HOMEBREW_CELLAR"`
	HomebrewCaskroom   string `json:"HOMEBREW_CASKROOM"`
	HomebrewCache      string `json:"HOMEBREW_CACHE"`
	HomebrewLogs       string `json:"HOMEBREW_LOGS"`
	HomebrewTemp       string `json:"HOMEBREW_TEMP"`
	Path               string `json:"PATH"`
	Platform           string `json:"platform"`
	GoVersion          string `json:"go_version"`
}

func showEnv(cfg *config.Config, jsonOutput bool) error {
	pathValue := fmt.Sprintf("%s/bin:%s/sbin:$PATH", cfg.HomebrewPrefix, cfg.HomebrewPrefix)

	if jsonOutput {
		env := EnvironmentInfo{
			HomebrewPrefix:     cfg.HomebrewPrefix,
			HomebrewRepository: cfg.HomebrewRepository,
			HomebrewCellar:     cfg.HomebrewCellar,
			HomebrewCaskroom:   cfg.HomebrewCaskroom,
			HomebrewCache:      cfg.HomebrewCache,
			HomebrewLogs:       cfg.HomebrewLogs,
			HomebrewTemp:       cfg.HomebrewTemp,
			Path:               pathValue,
			Platform:           runtime.GOOS + "/" + runtime.GOARCH,
			GoVersion:          runtime.Version(),
		}

		data, err := json.MarshalIndent(env, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal environment to JSON: %w", err)
		}
		fmt.Println(string(data))
	} else {
		// Traditional shell export format
		fmt.Printf("export HOMEBREW_PREFIX=%s\n", cfg.HomebrewPrefix)
		fmt.Printf("export HOMEBREW_REPOSITORY=%s\n", cfg.HomebrewRepository)
		fmt.Printf("export HOMEBREW_CELLAR=%s\n", cfg.HomebrewCellar)
		fmt.Printf("export HOMEBREW_CASKROOM=%s\n", cfg.HomebrewCaskroom)
		fmt.Printf("export PATH=%s\n", pathValue)
	}

	return nil
}
