package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/homebrew/brew/internal/api"
	"github.com/homebrew/brew/internal/config"
	"github.com/homebrew/brew/internal/installer"
	"github.com/homebrew/brew/internal/logger"
	"github.com/spf13/cobra"
)

// NewInstallCmd creates the install command
func NewInstallCmd(cfg *config.Config) *cobra.Command {
	var (
		formulaOnly        bool
		caskOnly           bool
		buildFromSource    bool
		forceBottle        bool
		ignoreDependencies bool
		onlyDependencies   bool
		includeTest        bool
		headOnly           bool
		keepTmp            bool
		debugSymbols       bool
		displayTimes       bool
		ask                bool
		cc                 string
	)

	cmd := &cobra.Command{
		Use:   "install [OPTIONS] FORMULA|CASK...",
		Short: "Install a formula or cask",
		Long: `Install one or more formulae or casks.

Unless HOMEBREW_NO_INSTALLED_DEPENDENTS_CHECK is set, brew upgrade or brew reinstall 
will be run for outdated dependents and dependents with broken linkage, respectively.

Unless HOMEBREW_NO_INSTALL_CLEANUP is set, brew cleanup will then be run for 
the installed formulae or, every 30 days, for all formulae.

Unless HOMEBREW_NO_INSTALL_UPGRADE is set, brew install <formula> will upgrade 
<formula> if it is already installed but outdated.`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInstall(cfg, args, &installOptions{
				FormulaOnly:        formulaOnly,
				CaskOnly:           caskOnly,
				BuildFromSource:    buildFromSource,
				ForceBottle:        forceBottle,
				IgnoreDependencies: ignoreDependencies,
				OnlyDependencies:   onlyDependencies,
				IncludeTest:        includeTest,
				HeadOnly:           headOnly,
				KeepTmp:            keepTmp,
				DebugSymbols:       debugSymbols,
				DisplayTimes:       displayTimes,
				Ask:                ask,
				CC:                 cc,
				Force:              cfg.Force,
				DryRun:             cfg.DryRun,
				Verbose:            cfg.Verbose,
			})
		},
	}

	// Add flags
	cmd.Flags().BoolVar(&formulaOnly, "formula", false, "Treat all named arguments as formulae")
	cmd.Flags().BoolVar(&formulaOnly, "formulae", false, "Treat all named arguments as formulae")
	cmd.Flags().BoolVar(&caskOnly, "cask", false, "Treat all named arguments as casks")
	cmd.Flags().BoolVar(&caskOnly, "casks", false, "Treat all named arguments as casks")
	cmd.Flags().BoolVarP(&buildFromSource, "build-from-source", "s", false, "Compile formula from source even if a bottle is provided")
	cmd.Flags().BoolVar(&forceBottle, "force-bottle", false, "Install from a bottle if it exists")
	cmd.Flags().BoolVar(&ignoreDependencies, "ignore-dependencies", false, "Skip installing any dependencies")
	cmd.Flags().BoolVar(&onlyDependencies, "only-dependencies", false, "Install dependencies but not the formula itself")
	cmd.Flags().BoolVar(&includeTest, "include-test", false, "Install testing dependencies")
	cmd.Flags().BoolVar(&headOnly, "HEAD", false, "Install the HEAD version")
	cmd.Flags().BoolVar(&keepTmp, "keep-tmp", false, "Retain the temporary files created during installation")
	cmd.Flags().BoolVar(&debugSymbols, "debug-symbols", false, "Generate debug symbols on build")
	cmd.Flags().BoolVar(&displayTimes, "display-times", false, "Print install times for each package")
	cmd.Flags().BoolVar(&ask, "ask", false, "Ask for confirmation before downloading and installing")
	cmd.Flags().StringVar(&cc, "cc", "", "Attempt to compile using the specified compiler")

	return cmd
}

type installOptions struct {
	FormulaOnly        bool
	CaskOnly           bool
	BuildFromSource    bool
	ForceBottle        bool
	IgnoreDependencies bool
	OnlyDependencies   bool
	IncludeTest        bool
	HeadOnly           bool
	KeepTmp            bool
	DebugSymbols       bool
	DisplayTimes       bool
	Ask                bool
	CC                 string
	Force              bool
	DryRun             bool
	Verbose            bool
}

func runInstall(cfg *config.Config, args []string, opts *installOptions) error {
	timer := logger.NewTimer("Total install time")
	defer timer.Stop()

	// Parse arguments to separate formulae/casks from options
	formulae, casks, err := parseInstallArgs(args, opts)
	if err != nil {
		return fmt.Errorf("failed to parse arguments: %w", err)
	}

	if len(formulae) == 0 && len(casks) == 0 {
		return fmt.Errorf("no formulae or casks specified")
	}

	// Initialize installer
	inst := installer.New(cfg, &installer.Options{
		BuildFromSource:    opts.BuildFromSource || cfg.BuildFromSource,
		ForceBottle:        opts.ForceBottle || cfg.ForceBottle,
		IgnoreDependencies: opts.IgnoreDependencies,
		OnlyDependencies:   opts.OnlyDependencies,
		IncludeTest:        opts.IncludeTest,
		HeadOnly:           opts.HeadOnly,
		KeepTmp:            opts.KeepTmp || cfg.KeepTmp,
		DebugSymbols:       opts.DebugSymbols,
		Force:              opts.Force,
		DryRun:             opts.DryRun,
		Verbose:            opts.Verbose,
		CC:                 opts.CC,
	})

	// Install formulae
	var installTimes []installer.InstallResult

	for _, formulaName := range formulae {
		logger.Progress("Installing formula: %s", formulaName)

		if opts.DryRun {
			logger.Info("Would install formula: %s", formulaName)
			continue
		}

		// Check if formula is already installed
		if installed, err := isFormulaInstalled(cfg, formulaName); err != nil {
			logger.Warn("Failed to check if %s is installed: %v", formulaName, err)
		} else if installed && !opts.Force {
			if !cfg.NoInstallUpgrade {
				logger.Info("Formula %s is already installed, checking for updates...", formulaName)

				// Check if upgrade is needed
				needsUpgrade, err := checkIfUpgradeNeeded(cfg, formulaName)
				if err != nil {
					logger.Warn("Failed to check for updates for %s: %v", formulaName, err)
					logger.Info("Proceeding with reinstall...")
				} else if needsUpgrade {
					logger.Info("Formula %s has an update available, proceeding with upgrade...", formulaName)
				} else {
					logger.Info("Formula %s is already up-to-date", formulaName)
					continue
				}
			} else {
				logger.Info("Formula %s is already installed", formulaName)
				continue
			}
		}

		// Ask for confirmation if needed
		if opts.Ask {
			if !askForConfirmation(formulaName, "formula") {
				logger.Info("Skipping %s", formulaName)
				continue
			}
		}

		// Install the formula
		result, err := inst.InstallFormula(formulaName)
		if err != nil {
			return fmt.Errorf("failed to install formula %s: %w", formulaName, err)
		}

		installTimes = append(installTimes, *result)
		logger.Success("Successfully installed %s", formulaName)
	}

	// Install casks
	for _, caskName := range casks {
		logger.Progress("Installing cask: %s", caskName)

		if opts.DryRun {
			logger.Info("Would install cask: %s", caskName)
			continue
		}

		// Ask for confirmation if needed
		if opts.Ask {
			if !askForConfirmation(caskName, "cask") {
				logger.Info("Skipping %s", caskName)
				continue
			}
		}

		// Install the cask
		result, err := inst.InstallCask(caskName)
		if err != nil {
			return fmt.Errorf("failed to install cask %s: %w", caskName, err)
		}

		installTimes = append(installTimes, *result)
		logger.Success("Successfully installed %s", caskName)
	}

	// Display install times if requested
	if opts.DisplayTimes && len(installTimes) > 0 {
		logger.PrintDivider()
		logger.PrintHeader("Install Times")
		for _, result := range installTimes {
			logger.Info("  %-20s %v", result.Name, result.Duration)
		}
	}

	// Run cleanup if enabled
	if !cfg.NoInstallUpgrade && cfg.InstallCleanup {
		logger.Progress("Running cleanup...")
		if err := runCleanup(cfg, false); err != nil {
			logger.Warn("Cleanup failed: %v", err)
		}
	}

	return nil
}

func parseInstallArgs(args []string, opts *installOptions) ([]string, []string, error) {
	var formulae []string
	var casks []string

	for _, arg := range args {
		if strings.HasPrefix(arg, "--") {
			continue // Skip options
		}

		if opts.CaskOnly {
			casks = append(casks, arg)
		} else if opts.FormulaOnly {
			formulae = append(formulae, arg)
		} else {
			// Auto-detect based on name or check both
			if strings.Contains(arg, "/") {
				// Tap-qualified name, assume formula
				formulae = append(formulae, arg)
			} else if isCaskName(arg) {
				casks = append(casks, arg)
			} else {
				formulae = append(formulae, arg)
			}
		}
	}

	return formulae, casks, nil
}

func isCaskName(name string) bool {
	// Simple heuristic: casks often have different naming patterns
	// This is a placeholder - in practice, we'd check the cask repository
	return strings.Contains(name, "-") &&
		(strings.Contains(name, "app") ||
			strings.Contains(name, "desktop") ||
			strings.HasSuffix(name, ".app"))
}

func isFormulaInstalled(cfg *config.Config, name string) (bool, error) {
	formulaPath := filepath.Join(cfg.HomebrewCellar, name)
	_, err := os.Stat(formulaPath)
	if os.IsNotExist(err) {
		return false, nil
	}
	return err == nil, err
}

func askForConfirmation(name, typ string) bool {
	return logger.Confirm("Install %s %s?", typ, name)
}

// getPlatform returns the current platform identifier for bottles
func getPlatform() string {
	switch runtime.GOOS {
	case "darwin":
		switch runtime.GOARCH {
		case "amd64":
			return "monterey" // Default to recent macOS version
		case "arm64":
			return "arm64_monterey"
		}
	case "linux":
		return "x86_64_linux"
	}
	return "unknown"
}

// checkIfUpgradeNeeded checks if a formula needs to be upgraded
func checkIfUpgradeNeeded(cfg *config.Config, formulaName string) (bool, error) {
	// Get current installed version
	currentVersion, err := getInstalledVersion(cfg, formulaName)
	if err != nil {
		return false, fmt.Errorf("failed to get installed version: %w", err)
	}

	// Get latest version from API
	apiClient := api.NewClient(cfg)
	latestFormula, err := apiClient.GetFormula(formulaName)
	if err != nil {
		return false, fmt.Errorf("failed to get latest formula info: %w", err)
	}

	// Compare versions
	return currentVersion != latestFormula.Version, nil
}
