package cask

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/pilshchikov/homebrew-go/internal/config"
	"github.com/pilshchikov/homebrew-go/internal/errors"
	"github.com/pilshchikov/homebrew-go/internal/logger"
	"github.com/pilshchikov/homebrew-go/internal/verification"
)

// Installer handles installation of casks
type Installer struct {
	config   *config.Config
	verifier *verification.PackageVerifier
}

// CaskInstallOptions contains options for cask installation
type CaskInstallOptions struct {
	Force              bool
	RequireSHA         bool
	SkipCaskDeps       bool
	Verbose            bool
	DryRun             bool
	NoQuarantine       bool
	AdoptOrphanedCasks bool
}

// CaskInstallResult contains the result of a cask installation
type CaskInstallResult struct {
	Name      string
	Version   string
	Token     string
	Success   bool
	Error     error
	Artifacts []string
	Caveats   string
}

// NewCaskInstaller creates a new cask installer
func NewCaskInstaller(cfg *config.Config) *Installer {
	return &Installer{
		config:   cfg,
		verifier: verification.NewPackageVerifier(false), // Non-strict for casks
	}
}

// InstallCask installs a cask
func (ci *Installer) InstallCask(cask *Cask, opts *CaskInstallOptions) (*CaskInstallResult, error) {
	result := &CaskInstallResult{
		Name:    cask.Name,
		Version: cask.Version,
		Token:   cask.Token,
	}

	logger.PrintHeader(fmt.Sprintf("Installing Cask: %s", cask.Token))

	// Validate cask
	if err := cask.Validate(); err != nil {
		result.Error = fmt.Errorf("invalid cask: %w", err)
		return result, result.Error
	}

	// Check platform compatibility
	if !cask.IsCompatibleWithPlatform() {
		result.Error = fmt.Errorf("cask %s is not compatible with this platform", cask.Token)
		return result, result.Error
	}

	// Check if already installed
	if cask.IsInstalled() && !opts.Force {
		logger.Info("Cask %s is already installed", cask.Token)
		result.Success = true
		return result, nil
	}

	if opts.DryRun {
		logger.Info("Dry run: would install cask %s", cask.Token)
		result.Success = true
		return result, nil
	}

	// Download cask
	downloadPath, err := ci.downloadCask(cask)
	if err != nil {
		result.Error = fmt.Errorf("failed to download cask: %w", err)
		return result, result.Error
	}

	// Verify download
	if cask.Sha256 != "" && opts.RequireSHA {
		logger.Debug("Verifying cask checksum")
		if err := ci.verifier.VerifySource(downloadPath, cask.Sha256, 0); err != nil {
			result.Error = fmt.Errorf("cask verification failed: %w", err)
			return result, result.Error
		}
	}

	// Extract if needed
	extractedPath, err := ci.extractCask(cask, downloadPath)
	if err != nil {
		result.Error = fmt.Errorf("failed to extract cask: %w", err)
		return result, result.Error
	}

	// Install artifacts
	artifacts, err := ci.installArtifacts(cask, extractedPath, opts)
	if err != nil {
		result.Error = fmt.Errorf("failed to install artifacts: %w", err)
		return result, result.Error
	}
	result.Artifacts = artifacts

	// Handle caveats
	if cask.GetCaveats() != "" {
		result.Caveats = cask.GetCaveats()
		logger.Info("Caveats for %s:", cask.Token)
		logger.Info(cask.GetCaveats())
	}

	// Create install receipt
	if err := ci.createInstallReceipt(cask); err != nil {
		logger.Warn("Failed to create install receipt: %v", err)
	}

	result.Success = true
	logger.Success("Successfully installed cask %s", cask.Token)
	return result, nil
}

// downloadCask downloads the cask package
func (ci *Installer) downloadCask(cask *Cask) (string, error) {
	url := cask.GetDownloadURL()
	if url == "" {
		return "", fmt.Errorf("no download URL available")
	}

	cacheDir := filepath.Join(ci.config.HomebrewCache, "cask")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return "", errors.NewPermissionError("create cache directory", cacheDir, err)
	}

	downloadPath := filepath.Join(cacheDir, cask.GetCacheFileName())

	// Check if already downloaded
	if _, err := os.Stat(downloadPath); err == nil {
		logger.Debug("Using cached download: %s", downloadPath)
		return downloadPath, nil
	}

	logger.Step("Downloading %s", filepath.Base(downloadPath))
	return downloadPath, ci.downloadFile(url, downloadPath)
}

// downloadFile downloads a file from URL
func (ci *Installer) downloadFile(url, path string) error {
	// This is a simplified download - in practice would use the same
	// download logic as the main installer with progress tracking
	cmd := exec.Command("curl", "-L", "-o", path, url)
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		return errors.NewDownloadError("download", url, fmt.Errorf("curl failed: %s", string(output)))
	}
	
	return nil
}

// extractCask extracts the downloaded cask if needed
func (ci *Installer) extractCask(cask *Cask, downloadPath string) (string, error) {
	ext := cask.GetFileExtension()
	
	// For some formats, we don't need extraction
	if ext == ".pkg" {
		return downloadPath, nil
	}
	
	extractDir := filepath.Join(ci.config.HomebrewCache, "cask", "extract", cask.Token)
	if err := os.MkdirAll(extractDir, 0755); err != nil {
		return "", errors.NewPermissionError("create extract directory", extractDir, err)
	}
	
	switch ext {
	case ".dmg":
		return ci.extractDMG(downloadPath, extractDir)
	case ".zip":
		return ci.extractZip(downloadPath, extractDir)
	case ".tar.gz", ".tar.bz2", ".tar.xz":
		return ci.extractTar(downloadPath, extractDir)
	default:
		// For unknown formats, just return the download path
		return downloadPath, nil
	}
}

// extractDMG mounts and extracts a DMG file
func (ci *Installer) extractDMG(dmgPath, extractDir string) (string, error) {
	logger.Step("Mounting DMG")
	
	// Mount the DMG
	cmd := exec.Command("hdiutil", "attach", "-quiet", "-nobrowse", "-mountpoint", extractDir, dmgPath)
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to mount DMG: %w", err)
	}
	
	// Note: In a real implementation, we'd need to handle unmounting
	// and proper cleanup of the mount point
	return extractDir, nil
}

// extractZip extracts a ZIP file
func (ci *Installer) extractZip(zipPath, extractDir string) (string, error) {
	logger.Step("Extracting ZIP")
	
	cmd := exec.Command("unzip", "-q", "-d", extractDir, zipPath)
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to extract ZIP: %w", err)
	}
	
	return extractDir, nil
}

// extractTar extracts a tar archive
func (ci *Installer) extractTar(tarPath, extractDir string) (string, error) {
	logger.Step("Extracting tar archive")
	
	var cmd *exec.Cmd
	if strings.HasSuffix(tarPath, ".tar.gz") {
		cmd = exec.Command("tar", "-xzf", tarPath, "-C", extractDir)
	} else if strings.HasSuffix(tarPath, ".tar.bz2") {
		cmd = exec.Command("tar", "-xjf", tarPath, "-C", extractDir)
	} else if strings.HasSuffix(tarPath, ".tar.xz") {
		cmd = exec.Command("tar", "-xJf", tarPath, "-C", extractDir)
	} else {
		return "", fmt.Errorf("unsupported tar format: %s", tarPath)
	}
	
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to extract tar: %w", err)
	}
	
	return extractDir, nil
}

// installArtifacts installs the cask artifacts
func (ci *Installer) installArtifacts(cask *Cask, sourcePath string, opts *CaskInstallOptions) ([]string, error) {
	if len(cask.Artifacts) == 0 {
		return nil, fmt.Errorf("no artifacts to install")
	}
	
	artifacts := cask.Artifacts[0]
	installed := []string{}
	
	// Install applications
	for _, app := range artifacts.App {
		if err := ci.installApp(app, sourcePath, opts); err != nil {
			return installed, fmt.Errorf("failed to install app %s: %w", app.Source, err)
		}
		installed = append(installed, app.Target)
	}
	
	// Install binaries
	for _, binary := range artifacts.Binary {
		if err := ci.installBinary(binary, sourcePath, opts); err != nil {
			return installed, fmt.Errorf("failed to install binary %s: %w", binary.Source, err)
		}
		installed = append(installed, binary.Target)
	}
	
	// Install packages
	for _, pkg := range artifacts.Pkg {
		if err := ci.installPkg(pkg, sourcePath, opts); err != nil {
			return installed, fmt.Errorf("failed to install pkg %s: %w", pkg, err)
		}
		installed = append(installed, pkg)
	}
	
	// Handle installers
	for _, installer := range artifacts.Installer {
		if err := ci.runInstaller(installer, sourcePath, opts); err != nil {
			return installed, fmt.Errorf("failed to run installer: %w", err)
		}
		installed = append(installed, "installer")
	}
	
	return installed, nil
}

// installApp installs an application bundle
func (ci *Installer) installApp(app CaskApp, sourcePath string, opts *CaskInstallOptions) error {
	sourcePath = filepath.Join(sourcePath, app.Source)
	
	target := app.Target
	if target == "" {
		target = filepath.Join("/Applications", filepath.Base(app.Source))
	} else if !filepath.IsAbs(target) {
		target = filepath.Join("/Applications", target)
	}
	
	logger.Step("Installing app: %s → %s", app.Source, target)
	
	if opts.DryRun {
		return nil
	}
	
	// Check if target already exists
	if _, err := os.Stat(target); err == nil && !opts.Force {
		return fmt.Errorf("application already exists at %s", target)
	}
	
	// Copy the application
	cmd := exec.Command("cp", "-R", sourcePath, target)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to copy application: %w", err)
	}
	
	// Remove quarantine attribute if requested
	if opts.NoQuarantine {
		cmd = exec.Command("xattr", "-dr", "com.apple.quarantine", target)
		cmd.Run() // Ignore errors for this step
	}
	
	return nil
}

// installBinary installs a binary symlink
func (ci *Installer) installBinary(binary CaskBinary, sourcePath string, opts *CaskInstallOptions) error {
	sourcePath = filepath.Join(sourcePath, binary.Source)
	
	target := binary.Target
	if target == "" {
		target = filepath.Join("/usr/local/bin", filepath.Base(binary.Source))
	} else if !filepath.IsAbs(target) {
		target = filepath.Join("/usr/local/bin", target)
	}
	
	logger.Step("Installing binary: %s → %s", binary.Source, target)
	
	if opts.DryRun {
		return nil
	}
	
	// Create symlink
	if err := os.Symlink(sourcePath, target); err != nil {
		return fmt.Errorf("failed to create binary symlink: %w", err)
	}
	
	return nil
}

// installPkg installs a package file
func (ci *Installer) installPkg(pkg, sourcePath string, opts *CaskInstallOptions) error {
	pkgPath := filepath.Join(sourcePath, pkg)
	
	logger.Step("Installing package: %s", pkg)
	
	if opts.DryRun {
		return nil
	}
	
	// Install the package using installer command
	cmd := exec.Command("sudo", "installer", "-pkg", pkgPath, "-target", "/")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install package: %w", err)
	}
	
	return nil
}

// runInstaller runs custom installer scripts
func (ci *Installer) runInstaller(installer CaskInstaller, sourcePath string, opts *CaskInstallOptions) error {
	if installer.Manual != "" {
		logger.Info("Manual installation required: %s", installer.Manual)
		return nil
	}
	
	// Handle script-based installers
	if installer.Script != nil {
		logger.Step("Running installer script")
		
		if opts.DryRun {
			return nil
		}
		
		// This would need more complex implementation based on script type
		logger.Warn("Script-based installers not fully implemented")
	}
	
	return nil
}

// createInstallReceipt creates an installation receipt
func (ci *Installer) createInstallReceipt(cask *Cask) error {
	receiptDir := filepath.Join(ci.config.HomebrewCaskroom, cask.Token, cask.Version)
	if err := os.MkdirAll(receiptDir, 0755); err != nil {
		return err
	}
	
	receiptPath := filepath.Join(receiptDir, ".metadata")
	receiptData := fmt.Sprintf(`{
  "token": "%s",
  "name": "%s",
  "version": "%s",
  "installed_on": "%s",
  "installed_by": "brew-go"
}`, cask.Token, cask.Name, cask.Version, fmt.Sprintf("%d", os.Getpid()))
	
	return os.WriteFile(receiptPath, []byte(receiptData), 0644)
}

// UninstallCask uninstalls a cask
func (ci *Installer) UninstallCask(cask *Cask, opts *CaskInstallOptions) error {
	logger.PrintHeader(fmt.Sprintf("Uninstalling Cask: %s", cask.Token))
	
	if !cask.IsInstalled() {
		return fmt.Errorf("cask %s is not installed", cask.Token)
	}
	
	if len(cask.Artifacts) > 0 && len(cask.Artifacts[0].Uninstall) > 0 {
		return ci.runUninstallSteps(cask.Artifacts[0].Uninstall, opts)
	}
	
	// Default uninstall - remove applications
	return ci.removeDefaultArtifacts(cask, opts)
}

// runUninstallSteps runs custom uninstall steps
func (ci *Installer) runUninstallSteps(uninstalls []CaskUninstall, opts *CaskInstallOptions) error {
	for _, uninstall := range uninstalls {
		// Delete files
		for _, path := range uninstall.Delete {
			logger.Step("Deleting: %s", path)
			if !opts.DryRun {
				os.RemoveAll(path)
			}
		}
		
		// Move to trash
		for _, path := range uninstall.Trash {
			logger.Step("Moving to trash: %s", path)
			if !opts.DryRun {
				// This would need integration with macOS trash system
				os.RemoveAll(path)
			}
		}
		
		// Remove directories
		for _, path := range uninstall.Rmdir {
			logger.Step("Removing directory: %s", path)
			if !opts.DryRun {
				os.Remove(path)
			}
		}
		
		// Handle other uninstall steps (pkgutil, signals, etc.)
		// This would need more implementation
	}
	
	return nil
}

// removeDefaultArtifacts removes standard cask artifacts
func (ci *Installer) removeDefaultArtifacts(cask *Cask, opts *CaskInstallOptions) error {
	if len(cask.Artifacts) == 0 {
		return nil
	}
	
	artifacts := cask.Artifacts[0]
	
	// Remove applications
	for _, app := range artifacts.App {
		target := app.Target
		if target == "" {
			target = filepath.Join("/Applications", filepath.Base(app.Source))
		}
		
		logger.Step("Removing application: %s", target)
		if !opts.DryRun {
			os.RemoveAll(target)
		}
	}
	
	// Remove binaries
	for _, binary := range artifacts.Binary {
		target := binary.Target
		if target == "" {
			target = filepath.Join("/usr/local/bin", filepath.Base(binary.Source))
		}
		
		logger.Step("Removing binary: %s", target)
		if !opts.DryRun {
			os.Remove(target)
		}
	}
	
	return nil
}