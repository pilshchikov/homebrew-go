package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/pilshchikov/homebrew-go/internal/cmd"
	"github.com/pilshchikov/homebrew-go/internal/config"
	"github.com/pilshchikov/homebrew-go/internal/logger"
)

var (
	Version   = "dev"
	GitCommit = "unknown"
	BuildDate = "unknown"
)

func main() {
	if err := run(); err != nil {
		logger.Error("brew failed: %v", err)
		os.Exit(1)
	}
}

func run() error {
	// Initialize configuration
	cfg, err := config.New()
	if err != nil {
		return fmt.Errorf("failed to initialize config: %w", err)
	}

	// Set up homebrew paths
	if cfg.HomebrewPrefix == "" {
		if runtime.GOOS == "darwin" && runtime.GOARCH == "amd64" {
			cfg.HomebrewPrefix = "/usr/local"
		} else {
			cfg.HomebrewPrefix = "/opt/homebrew"
		}
	}

	if cfg.HomebrewRepository == "" {
		cfg.HomebrewRepository = cfg.HomebrewPrefix
	}

	if cfg.HomebrewLibrary == "" {
		cfg.HomebrewLibrary = filepath.Join(cfg.HomebrewRepository, "Library")
	}

	// Initialize logger with config
	logger.Init(cfg.Debug, cfg.Verbose, cfg.Quiet)

	// Create and execute root command
	rootCmd := cmd.NewRootCmd(cfg, Version, GitCommit, BuildDate)
	return rootCmd.Execute()
}
