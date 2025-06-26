package installer

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/homebrew/brew/internal/api"
	"github.com/homebrew/brew/internal/cask"
	"github.com/homebrew/brew/internal/config"
	"github.com/homebrew/brew/internal/errors"
	"github.com/homebrew/brew/internal/formula"
	"github.com/homebrew/brew/internal/logger"
	"github.com/homebrew/brew/internal/tap"
	"github.com/homebrew/brew/internal/verification"
)

// progressReader wraps an io.Reader to show download progress
type progressReader struct {
	reader   io.Reader
	total    int64
	current  int64
	filename string
	lastUpdate time.Time
}

func (pr *progressReader) Read(p []byte) (n int, err error) {
	n, err = pr.reader.Read(p)
	pr.current += int64(n)
	
	// Update progress every 100ms to avoid flooding the terminal
	now := time.Now()
	if now.Sub(pr.lastUpdate) > 100*time.Millisecond || err == io.EOF {
		pr.lastUpdate = now
		percent := float64(pr.current) / float64(pr.total) * 100
		
		// Format file size
		currentMB := float64(pr.current) / 1024 / 1024
		totalMB := float64(pr.total) / 1024 / 1024
		
		if err == io.EOF {
			fmt.Printf("\r    Downloaded %s (%.1f MB) - 100%%\n", pr.filename, totalMB)
		} else {
			fmt.Printf("\r    Downloading %s (%.1f/%.1f MB) - %.1f%%", 
				pr.filename, currentMB, totalMB, percent)
		}
	}
	
	return n, err
}

// Installer handles formula and cask installation
type Installer struct {
	cfg       *config.Config
	opts      *Options
	apiClient *api.Client
	verifier  *verification.PackageVerifier
}

// Options contains installation options
type Options struct {
	BuildFromSource    bool
	ForceBottle        bool
	IgnoreDependencies bool
	OnlyDependencies   bool
	IncludeTest        bool
	HeadOnly           bool
	KeepTmp           bool
	DebugSymbols      bool
	Force             bool
	DryRun            bool
	Verbose           bool
	CC                string
	StrictVerification bool
}

// InstallResult contains the result of an installation
type InstallResult struct {
	Name     string
	Version  string
	Duration time.Duration
	Source   string // "bottle" or "source"
	Success  bool
	Error    error
}

// InstallReceipt contains installation metadata
type InstallReceipt struct {
	Name              string            `json:"name"`
	Version           string            `json:"version"`
	InstalledOn       time.Time         `json:"installed_on"`
	InstalledBy       string            `json:"installed_by"`
	Source            string            `json:"source"`
	BuildDependencies []string          `json:"build_dependencies,omitempty"`
	Dependencies      []string          `json:"dependencies,omitempty"`
	Options           []string          `json:"options,omitempty"`
	BuildOptions      map[string]string `json:"build_options,omitempty"`
	Compiler          string            `json:"compiler,omitempty"`
	Platform          string            `json:"platform"`
}

// New creates a new installer
func New(cfg *config.Config, opts *Options) *Installer {
	return &Installer{
		cfg:       cfg,
		opts:      opts,
		apiClient: api.NewClient(cfg),
		verifier:  verification.NewPackageVerifier(opts.StrictVerification),
	}
}

// InstallFormula installs a formula
func (i *Installer) InstallFormula(name string) (*InstallResult, error) {
	start := time.Now()
	result := &InstallResult{
		Name: name,
	}

	logger.Progress("Installing formula: %s", name)

	// Resolve formula
	f, err := i.resolveFormula(name)
	if err != nil {
		result.Error = err
		if strings.Contains(err.Error(), "not found") {
			br := errors.NewFormulaNotFoundError(name)
			logger.LogDetailedError(logger.ErrorContext{
				Operation:   br.Operation,
				Formula:     br.Formula,
				Error:       br,
				Suggestions: br.Suggestions,
			})
			return result, br
		}
		return result, errors.Wrap(err, "formula resolution", name)
	}

	result.Version = f.Version

	// Check dependencies first
	if !i.opts.IgnoreDependencies {
		logger.Step("Checking dependencies for %s", f.Name)
		if err := i.installDependencies(f); err != nil {
			result.Error = err
			if brewErr, ok := err.(*errors.BrewError); ok {
				logger.LogDetailedError(logger.ErrorContext{
					Operation:   brewErr.Operation,
					Formula:     brewErr.Formula,
					Version:     brewErr.Version,
					Platform:    brewErr.Platform,
					Error:       brewErr,
					Suggestions: brewErr.Suggestions,
				})
				return result, brewErr
			}
			return result, errors.NewDependencyError(name, "", err)
		}
	}

	// If only installing dependencies, stop here
	if i.opts.OnlyDependencies {
		result.Duration = time.Since(start)
		result.Success = true
		return result, nil
	}

	// Determine installation method
	var installErr error
	if i.shouldUseBottle(f) {
		logger.Step("Installing from bottle")
		result.Source = "bottle"
		installErr = i.installFromBottle(f)
		
		// If bottle installation fails, fall back to source
		if installErr != nil {
			// Only show warning for unexpected errors, not missing bottles
			if i.isBottleExpected(f, installErr) {
				logger.Warn("Bottle installation failed: %v", installErr)
			} else {
				logger.Debug("No bottle available, building from source: %v", installErr)
			}
			logger.Step("Falling back to building from source")
			result.Source = "source"
			installErr = i.installFromSource(f)
		}
	} else {
		logger.Step("Building from source")
		result.Source = "source"
		installErr = i.installFromSource(f)
	}

	if installErr != nil {
		result.Error = installErr
		// Provide detailed error context
		if brewErr, ok := installErr.(*errors.BrewError); ok {
			logger.LogDetailedError(logger.ErrorContext{
				Operation:   brewErr.Operation,
				Formula:     brewErr.Formula,
				Version:     brewErr.Version,
				Platform:    brewErr.Platform,
				Error:       brewErr,
				Suggestions: brewErr.Suggestions,
			})
		} else {
			// Wrap generic error
			installErr = errors.NewInstallationError(f.Name, f.Version, installErr)
			logger.LogDetailedError(logger.ErrorContext{
				Operation:   "installation",
				Formula:     f.Name,
				Version:     f.Version,
				Error:       installErr,
				Suggestions: installErr.(*errors.BrewError).Suggestions,
			})
		}
		return result, installErr
	}

	// Write install receipt
	if err := i.writeInstallReceipt(f, result.Source); err != nil {
		logger.Warn("Failed to write install receipt: %v", err)
	}

	// Link formula if needed
	if !f.KegOnly {
		if err := i.linkFormula(f); err != nil {
			logger.Warn("Failed to link formula: %v", err)
		}
	}

	result.Duration = time.Since(start)
	result.Success = true
	return result, nil
}

// InstallCask installs a cask
func (i *Installer) InstallCask(name string) (*InstallResult, error) {
	start := time.Now()
	result := &InstallResult{
		Name:   name,
		Source: "cask",
	}

	logger.Debug("Installing cask: %s", name)

	// Fetch cask information from API
	caskData, err := i.apiClient.GetCask(name)
	if err != nil {
		result.Error = fmt.Errorf("failed to fetch cask '%s': %w", name, err)
		return result, result.Error
	}

	// Create cask installer
	caskInstaller := cask.NewCaskInstaller(i.cfg)
	
	// Set up install options
	opts := &cask.CaskInstallOptions{
		Force:        i.opts.Force,
		RequireSHA:   true,
		Verbose:      i.opts.Verbose,
		DryRun:       i.opts.DryRun,
		NoQuarantine: false, // Could be made configurable
	}

	// Install the cask
	caskResult, err := caskInstaller.InstallCask(caskData, opts)
	if err != nil {
		result.Error = err
		return result, err
	}

	// Update result with cask-specific information
	result.Version = caskResult.Version
	result.Success = caskResult.Success
	result.Error = caskResult.Error
	result.Duration = time.Since(start)

	if caskResult.Caveats != "" {
		logger.Info("Caveats:")
		logger.Info(caskResult.Caveats)
	}

	return result, nil
}

func (i *Installer) resolveFormula(name string) (*formula.Formula, error) {
	// First try the API for faster resolution
	if f, err := i.apiClient.GetFormula(name); err == nil {
		logger.Debug("Resolved formula %s from API", name)
		return f, nil
	} else {
		logger.Debug("API resolution failed for %s: %v", name, err)
	}
	
	// Fallback to tap resolution
	tapManager := tap.NewManager(i.cfg)
	
	// Check if it's a tap-qualified name (e.g., user/repo/formula)
	parts := strings.Split(name, "/")
	if len(parts) == 3 {
		tapName := parts[0] + "/" + parts[1]
		formulaName := parts[2]
		
		t, err := tapManager.GetTap(tapName)
		if err != nil {
			return nil, fmt.Errorf("tap %s not found: %w", tapName, err)
		}
		
		return t.GetFormula(formulaName)
	}
	
	// Check core tap first
	coreTap, err := tapManager.GetTap("homebrew/core")
	if err == nil {
		if f, err := coreTap.GetFormula(name); err == nil {
			return f, nil
		}
	}
	
	// Search all taps
	taps, err := tapManager.ListTaps()
	if err != nil {
		return nil, fmt.Errorf("failed to list taps: %w", err)
	}
	
	for _, t := range taps {
		if f, err := t.GetFormula(name); err == nil {
			return f, nil
		}
	}
	
	return nil, fmt.Errorf("formula %s not found", name)
}

func (i *Installer) installDependencies(f *formula.Formula) error {
	deps := f.GetDependencies(i.opts.IncludeTest)
	if len(deps) == 0 {
		logger.Debug("No dependencies to install")
		return nil
	}

	logger.Step("Installing %d dependencies: %s", len(deps), strings.Join(deps, ", "))

	for idx, dep := range deps {
		logger.Progress("Installing dependency %d/%d: %s", idx+1, len(deps), dep)
		
		// Check if already installed
		if installed, err := i.isFormulaInstalled(dep); err != nil {
			return errors.NewDependencyError(f.Name, dep, 
				fmt.Errorf("failed to check if %s is installed: %w", dep, err))
		} else if installed {
			logger.Step("Dependency %s already installed", dep)
			continue
		}

		// Recursively install dependency
		if _, err := i.InstallFormula(dep); err != nil {
			// Wrap the error with dependency context
			if brewErr, ok := err.(*errors.BrewError); ok {
				return errors.NewDependencyError(f.Name, dep, brewErr)
			}
			return errors.NewDependencyError(f.Name, dep, err)
		}
		
		logger.Success("Dependency %s installed successfully", dep)
	}

	logger.Success("All dependencies installed successfully")
	return nil
}

func (i *Installer) shouldUseBottle(f *formula.Formula) bool {
	if i.opts.BuildFromSource && !i.opts.ForceBottle {
		return false
	}

	if i.opts.HeadOnly && !f.IsStable() {
		return false
	}

	platform := i.apiClient.GetPlatformTag()
	return f.HasBottle(platform)
}

func (i *Installer) isBottleExpected(f *formula.Formula, err error) bool {
	// Check if this looks like a formula that should have bottles
	// vs one that's expected to be source-only
	
	// If the formula explicitly claims to have bottles, then failure is unexpected
	platform := i.apiClient.GetPlatformTag()
	if f.HasBottle(platform) && f.Bottle != nil && f.Bottle.Stable != nil {
		// Check if it's just a 401/403 auth error (expected for missing bottles)
		errStr := err.Error()
		if strings.Contains(errStr, "401") || strings.Contains(errStr, "403") || 
		   strings.Contains(errStr, "not found") || strings.Contains(errStr, "no bottle available") {
			return false // This is expected for formulae without actual bottles
		}
		return true // Other errors are unexpected
	}
	
	// For formulae without bottle metadata, failure is expected
	return false
}

func (i *Installer) installFromBottle(f *formula.Formula) error {
	platform := i.apiClient.GetPlatformTag()
	
	// Try to download bottle using API client
	bottlePath, err := i.apiClient.DownloadBottle(f, platform)
	if err != nil {
		// Fallback to manual bottle handling
		bottleURL := f.GetBottleURL(platform)
		if bottleURL == "" {
			return fmt.Errorf("no bottle available for platform %s", platform)
		}

		// Download bottle manually
		bottlePath = filepath.Join(i.cfg.HomebrewCache, f.Name+"-"+f.Version+"."+platform+".bottle.tar.gz")
		if err := i.downloadFile(bottleURL, bottlePath); err != nil {
			return fmt.Errorf("failed to download bottle: %w", err)
		}

		// Verify SHA256
		expectedSHA := f.GetBottleSHA256(platform)
		if expectedSHA != "" {
			if err := i.verifier.VerifyBottle(bottlePath, expectedSHA, 0); err != nil {
				return fmt.Errorf("bottle verification failed: %w", err)
			}
		}
	}

	// Extract bottle
	cellarPath := f.GetCellarPath(i.cfg.HomebrewCellar)
	if err := os.MkdirAll(filepath.Dir(cellarPath), 0755); err != nil {
		return fmt.Errorf("failed to create cellar directory: %w", err)
	}

	if err := i.extractTarGz(bottlePath, cellarPath); err != nil {
		return fmt.Errorf("failed to extract bottle: %w", err)
	}

	// Clean up bottle file unless keeping temp files
	if !i.opts.KeepTmp {
		os.Remove(bottlePath)
	}

	return nil
}

func (i *Installer) installFromSource(f *formula.Formula) error {
	// Create temporary build directory
	buildDir := filepath.Join(i.cfg.HomebrewTemp, f.Name+"-"+f.Version)
	if err := os.MkdirAll(buildDir, 0755); err != nil {
		return fmt.Errorf("failed to create build directory: %w", err)
	}

	defer func() {
		if !i.opts.KeepTmp {
			os.RemoveAll(buildDir)
		}
	}()

	// Download source
	var sourceURL string
	if i.opts.HeadOnly && f.Head != nil {
		sourceURL = f.Head.URL
	} else {
		sourceURL = f.URL
	}

	logger.Debug("Downloading source from: %s", sourceURL)
	sourcePath := filepath.Join(buildDir, "source.tar.gz")
	if err := i.downloadFile(sourceURL, sourcePath); err != nil {
		return fmt.Errorf("failed to download source: %w", err)
	}
	logger.Debug("Downloaded source to: %s", sourcePath)

	// Verify checksum for stable version
	if !i.opts.HeadOnly && f.SHA256 != "" {
		if err := i.verifier.VerifySource(sourcePath, f.SHA256, 0); err != nil {
			return fmt.Errorf("source verification failed: %w", err)
		}
	}

	// Extract source
	sourceExtractDir := filepath.Join(buildDir, "extracted")
	logger.Debug("Extracting source to: %s", sourceExtractDir)
	if err := i.extractTarGz(sourcePath, sourceExtractDir); err != nil {
		return fmt.Errorf("failed to extract source: %w", err)
	}
	
	// Find the actual source directory (usually contains the project files)
	sourceDir, err := i.findSourceDirectory(sourceExtractDir)
	if err != nil {
		return fmt.Errorf("failed to find source directory: %w", err)
	}
	logger.Debug("Found source directory: %s", sourceDir)

	// Apply patches
	for _, patch := range f.Patches {
		if err := i.applyPatch(sourceDir, &patch); err != nil {
			return fmt.Errorf("failed to apply patch: %w", err)
		}
	}

	// Build and install
	cellarPath := f.GetCellarPath(i.cfg.HomebrewCellar)
	logger.Debug("Building in directory: %s", sourceDir)
	logger.Debug("Installing to: %s", cellarPath)
	if err := i.buildAndInstall(f, sourceDir, cellarPath); err != nil {
		return fmt.Errorf("failed to build and install: %w", err)
	}

	return nil
}

func (i *Installer) downloadFile(url, path string) error {
	filename := filepath.Base(url)
	logger.Step("Downloading %s", filename)

	// Create directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return errors.NewPermissionError("create download directory", filepath.Dir(path), err)
	}

	resp, err := http.Get(url)
	if err != nil {
		return errors.NewNetworkError("download", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		err := fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
		return errors.NewDownloadError("download", url, err)
	}

	// Get content length for verification
	contentLength := resp.ContentLength

	file, err := os.Create(path)
	if err != nil {
		return errors.NewPermissionError("create file", path, err)
	}
	defer file.Close()

	// Show download progress if content length is available
	var reader io.Reader = resp.Body
	if resp.ContentLength > 0 && !logger.IsQuiet() {
		reader = &progressReader{
			reader: resp.Body,
			total:  resp.ContentLength,
			filename: filename,
		}
	}

	bytesWritten, err := io.Copy(file, reader)
	if err != nil {
		return errors.NewDownloadError("save file", url, err)
	}

	// Verify downloaded size if content length was provided
	if contentLength > 0 && bytesWritten != contentLength {
		logger.Warn("Downloaded size (%d bytes) differs from expected size (%d bytes)", bytesWritten, contentLength)
	}

	logger.Success("Downloaded %s (%d bytes)", filename, bytesWritten)
	return nil
}

// VerifyInstallation verifies the integrity of an installed package
func (i *Installer) VerifyInstallation(formulaName string) (*verification.VerificationResult, error) {
	cellarPath := filepath.Join(i.cfg.HomebrewCellar, formulaName)
	return i.verifier.VerifyInstallation(cellarPath), nil
}

func (i *Installer) extractTarGz(tarPath, destDir string) error {
	file, err := os.Open(tarPath)
	if err != nil {
		return err
	}
	defer file.Close()

	gzr, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		target := filepath.Join(destDir, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, os.FileMode(header.Mode)); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}
			
			f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
			
			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return err
			}
			f.Close()
		}
	}

	return nil
}

func (i *Installer) findSourceDirectory(extractDir string) (string, error) {
	// List contents of extract directory
	files, err := os.ReadDir(extractDir)
	if err != nil {
		return "", err
	}
	
	// If there's only one directory, use it as the source directory
	if len(files) == 1 && files[0].IsDir() {
		return filepath.Join(extractDir, files[0].Name()), nil
	}
	
	// Look for a directory with configure script
	for _, file := range files {
		if file.IsDir() {
			dirPath := filepath.Join(extractDir, file.Name())
			configurePath := filepath.Join(dirPath, "configure")
			if _, err := os.Stat(configurePath); err == nil {
				return dirPath, nil
			}
		}
	}
	
	// If no subdirectory with configure, check if configure is in extract dir
	configurePath := filepath.Join(extractDir, "configure")
	if _, err := os.Stat(configurePath); err == nil {
		return extractDir, nil
	}
	
	// If no configure script found, look for Makefile
	for _, file := range files {
		if file.IsDir() {
			dirPath := filepath.Join(extractDir, file.Name())
			makefilePath := filepath.Join(dirPath, "Makefile")
			if _, err := os.Stat(makefilePath); err == nil {
				return dirPath, nil
			}
		}
	}
	
	// Fallback: use the extract directory itself
	return extractDir, nil
}

func (i *Installer) detectBuildSystem(sourceDir, cellarPath string) ([][]string, string, error) {
	// Check for configure script (autotools)
	configurePath := filepath.Join(sourceDir, "configure")
	if _, err := os.Stat(configurePath); err == nil {
		logger.Debug("Found configure script, using autotools")
		return i.buildAutotoolsCommands(sourceDir, cellarPath)
	}
	
	// Check for configure.ac/configure.in without configure script (needs autoreconf)
	configureAcPath := filepath.Join(sourceDir, "configure.ac")
	configureInPath := filepath.Join(sourceDir, "configure.in")
	if _, err := os.Stat(configureAcPath); err == nil {
		logger.Debug("Found configure.ac, using autotools with autoreconf")
		return i.buildAutotoolsWithAutoreconf(sourceDir, cellarPath)
	}
	if _, err := os.Stat(configureInPath); err == nil {
		logger.Debug("Found configure.in, using autotools with autoreconf")
		return i.buildAutotoolsWithAutoreconf(sourceDir, cellarPath)
	}
	
	// Check for CMake
	cmakeListsPath := filepath.Join(sourceDir, "CMakeLists.txt")
	if _, err := os.Stat(cmakeListsPath); err == nil {
		logger.Debug("Found CMakeLists.txt, using CMake")
		buildDir := filepath.Join(sourceDir, "build")
		os.MkdirAll(buildDir, 0755)
		return [][]string{
			{"cmake", "-S", ".", "-B", "build", "-DCMAKE_INSTALL_PREFIX=" + cellarPath, "-DCMAKE_BUILD_TYPE=Release"},
			{"cmake", "--build", "build", "--parallel"},
			{"cmake", "--install", "build"},
		}, "cmake", nil
	}
	
	// Check for Meson
	mesonBuildPath := filepath.Join(sourceDir, "meson.build")
	if _, err := os.Stat(mesonBuildPath); err == nil {
		logger.Debug("Found meson.build, using Meson")
		return [][]string{
			{"meson", "setup", "builddir", "--prefix=" + cellarPath, "--buildtype=release"},
			{"meson", "compile", "-C", "builddir"},
			{"meson", "install", "-C", "builddir"},
		}, "meson", nil
	}
	
	// Check for Python setup.py
	setupPyPath := filepath.Join(sourceDir, "setup.py")
	if _, err := os.Stat(setupPyPath); err == nil {
		logger.Debug("Found setup.py, using Python setuptools")
		return [][]string{
			{"python3", "setup.py", "build"},
			{"python3", "setup.py", "install", "--prefix=" + cellarPath},
		}, "python-setuptools", nil
	}
	
	// Check for Python pyproject.toml (modern Python packaging)
	pyprojectPath := filepath.Join(sourceDir, "pyproject.toml")
	if _, err := os.Stat(pyprojectPath); err == nil {
		logger.Debug("Found pyproject.toml, using pip")
		return [][]string{
			{"pip3", "install", ".", "--prefix=" + cellarPath, "--no-deps"},
		}, "python-pip", nil
	}
	
	// Check for Rust Cargo.toml
	cargoTomlPath := filepath.Join(sourceDir, "Cargo.toml")
	if _, err := os.Stat(cargoTomlPath); err == nil {
		logger.Debug("Found Cargo.toml, using Rust cargo")
		return [][]string{
			{"cargo", "build", "--release"},
			{"cargo", "install", "--path", ".", "--root", cellarPath},
		}, "rust-cargo", nil
	}
	
	// Check for Go modules
	goModPath := filepath.Join(sourceDir, "go.mod")
	if _, err := os.Stat(goModPath); err == nil {
		logger.Debug("Found go.mod, using Go modules")
		binDir := filepath.Join(cellarPath, "bin")
		os.MkdirAll(binDir, 0755)
		return [][]string{
			{"go", "build", "-o", binDir + "/", "./..."},
		}, "go-modules", nil
	}
	
	// Check for Node.js package.json
	packageJsonPath := filepath.Join(sourceDir, "package.json")
	if _, err := os.Stat(packageJsonPath); err == nil {
		logger.Debug("Found package.json, using npm")
		return [][]string{
			{"npm", "install"},
			{"npm", "run", "build"},
			{"npm", "install", "--prefix", cellarPath, "--global"},
		}, "npm", nil
	}
	
	// Check for Ninja build files
	buildNinjaPath := filepath.Join(sourceDir, "build.ninja")
	if _, err := os.Stat(buildNinjaPath); err == nil {
		logger.Debug("Found build.ninja, using Ninja")
		return [][]string{
			{"ninja"},
			{"ninja", "install"},
		}, "ninja", nil
	}
	
	// Check for Bazel BUILD files
	buildBazelPath := filepath.Join(sourceDir, "BUILD")
	buildBazelBazelPath := filepath.Join(sourceDir, "BUILD.bazel")
	workspacePath := filepath.Join(sourceDir, "WORKSPACE")
	_, err1 := os.Stat(buildBazelPath)
	_, err2 := os.Stat(buildBazelBazelPath)
	_, err3 := os.Stat(workspacePath)
	if err1 == nil || err2 == nil || err3 == nil {
		logger.Debug("Found Bazel build files, using Bazel")
		return [][]string{
			{"bazel", "build", "//..."},
			{"bazel", "run", "//install", "--", "--prefix=" + cellarPath},
		}, "bazel", nil
	}
	
	// Check for standard Makefile
	makefilePath := filepath.Join(sourceDir, "Makefile")
	if _, err := os.Stat(makefilePath); err == nil {
		logger.Debug("Found Makefile, using make")
		return [][]string{
			{"make", "PREFIX=" + cellarPath},
			{"make", "install", "PREFIX=" + cellarPath},
		}, "makefile", nil
	}
	
	
	// No recognized build system found
	buildFiles := []string{}
	possibleFiles := []string{
		"CMakeLists.txt", "meson.build", "setup.py", "pyproject.toml", 
		"Cargo.toml", "go.mod", "package.json", "build.ninja", 
		"BUILD", "BUILD.bazel", "WORKSPACE", "Makefile", "makefile",
		"configure.in", "configure.ac", "Makefile.am", "Makefile.in",
	}
	
	for _, file := range possibleFiles {
		if _, err := os.Stat(filepath.Join(sourceDir, file)); err == nil {
			buildFiles = append(buildFiles, file)
		}
	}
	
	buildErr := errors.NewBuildError("", "", fmt.Errorf("no supported build system found"))
	buildErr.Suggestions = []string{
		"This formula uses an unsupported or unrecognized build system",
		"Supported: autotools, CMake, Meson, Python (setuptools/pip), Rust (cargo), Go, Node.js (npm), Ninja, Bazel, Make",
	}
	
	if len(buildFiles) > 0 {
		buildErr.Suggestions = append(buildErr.Suggestions,
			fmt.Sprintf("Found build files: %s", strings.Join(buildFiles, ", ")),
			"These may indicate an unsupported build system variant")
	} else {
		buildErr.Suggestions = append(buildErr.Suggestions,
			"No recognized build files found in source directory",
			"This may be a library or data-only package")
	}
	
	return nil, "", buildErr
}

func (i *Installer) buildAutotoolsCommands(sourceDir, cellarPath string) ([][]string, string, error) {
	// Standard autotools build with existing configure script
	return [][]string{
		{"./configure", "--prefix=" + cellarPath, "--disable-dependency-tracking"},
		{"make"},
		{"make", "install"},
	}, "autotools", nil
}

func (i *Installer) buildAutotoolsWithAutoreconf(sourceDir, cellarPath string) ([][]string, string, error) {
	// Check if we need to install autotools first
	if err := i.ensureAutotoolsAvailable(); err != nil {
		return nil, "", fmt.Errorf("autotools not available: %w", err)
	}
	
	return [][]string{
		{"autoreconf", "-fiv"},
		{"./configure", "--prefix=" + cellarPath, "--disable-dependency-tracking"},
		{"make"},
		{"make", "install"},
	}, "autotools-generate", nil
}

func (i *Installer) ensureAutotoolsAvailable() error {
	// Check for required autotools commands
	requiredTools := []string{"autoreconf", "autoconf", "automake", "aclocal"}
	
	for _, tool := range requiredTools {
		if _, err := exec.LookPath(tool); err != nil {
			logger.Warn("Required tool '%s' not found", tool)
			
			// Try to install autotools using the system's package manager
			if err := i.installAutotools(); err != nil {
				return fmt.Errorf("autotools installation failed: %w", err)
			}
			break
		}
	}
	
	return nil
}

func (i *Installer) installAutotools() error {
	logger.Step("Installing autotools dependencies")
	
	// Try different package managers based on the system
	if runtime.GOOS == "darwin" {
		// Try to use Homebrew to install autotools
		cmd := exec.Command("brew", "install", "autoconf", "automake", "libtool")
		if err := cmd.Run(); err == nil {
			logger.Success("Installed autotools via Homebrew")
			return nil
		}
		
		// Try MacPorts as fallback
		cmd = exec.Command("port", "install", "autoconf", "automake", "libtool")
		if err := cmd.Run(); err == nil {
			logger.Success("Installed autotools via MacPorts")
			return nil
		}
	} else if runtime.GOOS == "linux" {
		// Try different Linux package managers
		managers := [][]string{
			{"apt-get", "update", "&&", "apt-get", "install", "-y", "autoconf", "automake", "libtool"},
			{"yum", "install", "-y", "autoconf", "automake", "libtool"},
			{"dnf", "install", "-y", "autoconf", "automake", "libtool"},
			{"pacman", "-S", "--noconfirm", "autoconf", "automake", "libtool"},
		}
		
		for _, mgr := range managers {
			cmd := exec.Command(mgr[0], mgr[1:]...)
			if err := cmd.Run(); err == nil {
				logger.Success("Installed autotools via %s", mgr[0])
				return nil
			}
		}
	}
	
	return fmt.Errorf("could not install autotools automatically - please install autoconf, automake, and libtool manually")
}

func (i *Installer) getBuildSystemSuggestions(buildSystem, command string) []string {
	suggestions := []string{}
	
	switch buildSystem {
	case "autotools", "autotools-generate":
		if strings.Contains(command, "configure") {
			suggestions = append(suggestions,
				"Check if all build dependencies are installed",
				"Review the configure output above for missing dependencies",
				"Try: brew install autoconf automake libtool",
				"Ensure pkg-config is installed for dependency detection")
		} else if strings.Contains(command, "make") {
			suggestions = append(suggestions,
				"Check for compilation errors in the output above",
				"Ensure you have the required development tools installed",
				"Try: xcode-select --install (on macOS)")
		} else if strings.Contains(command, "autoreconf") {
			suggestions = append(suggestions,
				"Install autotools if missing: brew install autoconf automake libtool",
				"Check if configure.ac or configure.in syntax is correct")
		}
		
	case "cmake":
		if strings.Contains(command, "cmake") && strings.Contains(command, "-S") {
			suggestions = append(suggestions,
				"Ensure CMake is installed: brew install cmake",
				"Check if all CMake dependencies are available",
				"Review CMakeLists.txt for missing required packages",
				"Try adding -DCMAKE_VERBOSE_MAKEFILE=ON for more details")
		} else if strings.Contains(command, "--build") {
			suggestions = append(suggestions,
				"Check for compilation errors in the output above",
				"Ensure all required libraries and headers are installed",
				"Try: cmake --build build --verbose for detailed output")
		} else if strings.Contains(command, "--install") {
			suggestions = append(suggestions,
				"Check if the build completed successfully",
				"Ensure you have write permissions to the install directory")
		}
		
	case "meson":
		if strings.Contains(command, "setup") {
			suggestions = append(suggestions,
				"Ensure Meson is installed: brew install meson",
				"Check if all Meson dependencies are available",
				"Review meson.build for missing required packages",
				"Try: pip3 install meson if brew version doesn't work")
		} else if strings.Contains(command, "compile") {
			suggestions = append(suggestions,
				"Check for compilation errors in the output above",
				"Ensure all required libraries and headers are installed",
				"Try: meson compile -C builddir --verbose for detailed output")
		} else if strings.Contains(command, "install") {
			suggestions = append(suggestions,
				"Check if the build completed successfully",
				"Ensure you have write permissions to the install directory")
		}
		
	case "python-setuptools":
		suggestions = append(suggestions,
			"Ensure Python 3 and setuptools are installed",
			"Try: pip3 install setuptools wheel",
			"Check if all Python dependencies are available",
			"Review setup.py for missing required packages")
			
	case "python-pip":
		suggestions = append(suggestions,
			"Ensure Python 3 and pip are installed",
			"Try: python3 -m pip install --upgrade pip",
			"Check if all Python dependencies are available",
			"Review pyproject.toml for build system requirements")
			
	case "rust-cargo":
		if strings.Contains(command, "build") {
			suggestions = append(suggestions,
				"Ensure Rust is installed: brew install rust",
				"Check if all Cargo dependencies can be downloaded",
				"Try: cargo build --verbose for detailed output",
				"Ensure you have internet access for crate downloads")
		} else if strings.Contains(command, "install") {
			suggestions = append(suggestions,
				"Check if the build completed successfully",
				"Ensure the binary was built correctly")
		}
		
	case "go-modules":
		suggestions = append(suggestions,
			"Ensure Go is installed: brew install go",
			"Check if all Go dependencies can be downloaded",
			"Try: go build -v for verbose output",
			"Ensure you have internet access for module downloads",
			"Check if go.mod and go.sum are valid")
			
	case "npm":
		if strings.Contains(command, "install") && !strings.Contains(command, "global") {
			suggestions = append(suggestions,
				"Ensure Node.js and npm are installed: brew install node",
				"Check if all npm dependencies can be downloaded",
				"Try: npm install --verbose for detailed output",
				"Clear npm cache: npm cache clean --force")
		} else if strings.Contains(command, "build") {
			suggestions = append(suggestions,
				"Check if the build script is defined in package.json",
				"Ensure all build dependencies are installed",
				"Try: npm run build --verbose for detailed output")
		} else if strings.Contains(command, "global") {
			suggestions = append(suggestions,
				"Check if the package was built successfully",
				"Ensure you have write permissions to the global directory")
		}
		
	case "ninja":
		suggestions = append(suggestions,
			"Ensure Ninja is installed: brew install ninja",
			"Check if build.ninja was generated correctly",
			"Try: ninja -v for verbose output",
			"Verify all dependencies for the build targets")
			
	case "bazel":
		suggestions = append(suggestions,
			"Ensure Bazel is installed: brew install bazel",
			"Check if WORKSPACE and BUILD files are valid",
			"Try: bazel build --verbose_failures //...",
			"Ensure all external dependencies can be downloaded")
			
	case "makefile":
		suggestions = append(suggestions,
			"Check for compilation errors in the output above",
			"Ensure you have the required development tools installed",
			"Try: make -j1 for sequential build to isolate errors",
			"Review the Makefile for proper PREFIX handling")
			
	default:
		suggestions = append(suggestions,
			"Check the build system documentation for troubleshooting",
			"Ensure all required build tools are installed",
			"Review the project's README for build instructions")
	}
	
	return suggestions
}

func (i *Installer) applyPatch(sourceDir string, patch *formula.Patch) error {
	logger.Step("Applying patch")
	
	var patchContent []byte
	var err error
	
	// Get patch content either from URL or inline data
	if patch.URL != "" {
		// Download patch from URL
		logger.Debug("Downloading patch from: %s", patch.URL)
		patchPath := filepath.Join(i.cfg.HomebrewTemp, "patch-"+filepath.Base(patch.URL))
		if err := i.downloadFile(patch.URL, patchPath); err != nil {
			return fmt.Errorf("failed to download patch: %w", err)
		}
		
		patchContent, err = os.ReadFile(patchPath)
		if err != nil {
			return fmt.Errorf("failed to read patch file: %w", err)
		}
		
		// Cleanup patch file
		defer os.Remove(patchPath)
	} else if patch.Data != "" {
		// Use inline patch data
		patchContent = []byte(patch.Data)
	} else {
		return fmt.Errorf("patch has no URL or inline data")
	}
	
	// Apply the patch using the patch command
	cmd := exec.Command("patch", fmt.Sprintf("-p%d", patch.Strip))
	cmd.Dir = sourceDir
	cmd.Stdin = strings.NewReader(string(patchContent))
	
	// Capture output for debugging
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	
	if err := cmd.Run(); err != nil {
		logger.Error("Patch application failed:")
		logger.Error("stdout: %s", stdout.String())
		logger.Error("stderr: %s", stderr.String())
		return fmt.Errorf("failed to apply patch: %w", err)
	}
	
	logger.Success("Patch applied successfully")
	return nil
}

func (i *Installer) buildAndInstall(f *formula.Formula, sourceDir, cellarPath string) error {
	logger.Progress("Building and installing %s", f.Name)

	// Create cellar directory
	if err := os.MkdirAll(cellarPath, 0755); err != nil {
		return errors.NewPermissionError("create cellar directory", cellarPath, err)
	}

	// Simple build process - in practice, this would be much more complex
	// and would need to handle different build systems (autotools, cmake, etc.)
	
	// Set environment variables
	env := os.Environ()
	env = append(env, "PREFIX="+cellarPath)
	env = append(env, "HOMEBREW_PREFIX="+i.cfg.HomebrewPrefix)
	
	if i.opts.CC != "" {
		env = append(env, "CC="+i.opts.CC)
	}

	// Detect build system and build accordingly
	commands, buildSystem, err := i.detectBuildSystem(sourceDir, cellarPath)
	if err != nil {
		return err
	}
	
	logger.Debug("Using build system: %s", buildSystem)

	for _, cmdArgs := range commands {
		cmdName := strings.Join(cmdArgs, " ")
		logger.Step("Running: %s", cmdName)
		
		cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
		cmd.Dir = sourceDir
		cmd.Env = env
		
		// Always show live output to match original Homebrew behavior
		// Capture output for error reporting while streaming live
		var stdout, stderr strings.Builder
		
		// Create multi-writers to both capture and display live output
		cmd.Stdout = io.MultiWriter(&stdout, os.Stdout)
		cmd.Stderr = io.MultiWriter(&stderr, os.Stderr)
		
		// In quiet mode, only capture without live display
		if logger.IsQuiet() {
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr
		}

		if err := cmd.Run(); err != nil {
			// Create detailed build error
			buildErr := errors.NewBuildError(f.Name, f.Version, err)
			
			// Add build system and command-specific suggestions
			buildErr.Suggestions = append(buildErr.Suggestions, i.getBuildSystemSuggestions(buildSystem, cmdArgs[0])...)
			
			// In quiet mode, show the captured output since it wasn't displayed live
			if logger.IsQuiet() && stderr.Len() > 0 {
				logger.Error("Build stderr output:")
				logger.Error(stderr.String())
			}
			
			return buildErr
		}
		
		// Show successful step completion
		logger.Success("Completed: %s", cmdName)
	}

	return nil
}

func (i *Installer) linkFormula(f *formula.Formula) error {
	logger.Debug("Linking formula %s", f.Name)
	
	cellarPath := f.GetCellarPath(i.cfg.HomebrewCellar)
	binDir := filepath.Join(cellarPath, "bin")
	
	// Link binaries
	if _, err := os.Stat(binDir); err == nil {
		files, err := os.ReadDir(binDir)
		if err != nil {
			return err
		}
		
		linkDir := filepath.Join(i.cfg.HomebrewPrefix, "bin")
		if err := os.MkdirAll(linkDir, 0755); err != nil {
			return err
		}
		
		for _, file := range files {
			if file.IsDir() {
				continue
			}
			
			src := filepath.Join(binDir, file.Name())
			dst := filepath.Join(linkDir, file.Name())
			
			// Remove existing link
			os.Remove(dst)
			
			// Create symlink
			if err := os.Symlink(src, dst); err != nil {
				return err
			}
		}
	}
	
	return nil
}

func (i *Installer) writeInstallReceipt(f *formula.Formula, source string) error {
	receipt := InstallReceipt{
		Name:        f.Name,
		Version:     f.Version,
		InstalledOn: time.Now(),
		InstalledBy: "brew-go",
		Source:      source,
		Dependencies: f.Dependencies,
		BuildDependencies: f.BuildDependencies,
		Platform:    i.apiClient.GetPlatformTag(),
	}

	if i.opts.CC != "" {
		receipt.Compiler = i.opts.CC
	}

	receiptPath := f.GetInstallReceipt(i.cfg.HomebrewCellar)
	if err := os.MkdirAll(filepath.Dir(receiptPath), 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(receipt, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(receiptPath, data, 0644)
}

func (i *Installer) isFormulaInstalled(name string) (bool, error) {
	formulaPath := filepath.Join(i.cfg.HomebrewCellar, name)
	_, err := os.Stat(formulaPath)
	if os.IsNotExist(err) {
		return false, nil
	}
	return err == nil, err
}

func getPlatform() string {
	switch runtime.GOOS {
	case "darwin":
		switch runtime.GOARCH {
		case "amd64":
			return "monterey"
		case "arm64":
			return "arm64_monterey"
		}
	case "linux":
		return "x86_64_linux"
	}
	return "unknown"
}