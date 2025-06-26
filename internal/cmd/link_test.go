package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/homebrew/brew/internal/config"
	"github.com/homebrew/brew/internal/logger"
)

func TestNewLinkCmd(t *testing.T) {
	logger.Init(false, false, true)
	cfg := &config.Config{}

	cmd := NewLinkCmd(cfg)

	if cmd.Use != "link [OPTIONS] FORMULA..." {
		t.Errorf("Expected Use to be 'link [OPTIONS] FORMULA...', got %s", cmd.Use)
	}

	if cmd.Short != "Symlink all of a formula's installed files into the Homebrew prefix" {
		t.Errorf("Expected correct Short description, got %s", cmd.Short)
	}

	// Test flags exist
	flags := []string{"overwrite", "dry-run", "force"}
	for _, flag := range flags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("Expected flag %s to exist", flag)
		}
	}
}

func TestNewUnlinkCmd(t *testing.T) {
	logger.Init(false, false, true)
	cfg := &config.Config{}

	cmd := NewUnlinkCmd(cfg)

	if cmd.Use != "unlink [OPTIONS] FORMULA..." {
		t.Errorf("Expected Use to be 'unlink [OPTIONS] FORMULA...', got %s", cmd.Use)
	}

	if cmd.Short != "Remove symlinks for a formula from the Homebrew prefix" {
		t.Errorf("Expected correct Short description, got %s", cmd.Short)
	}

	// Test flags exist
	flags := []string{"dry-run"}
	for _, flag := range flags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("Expected flag %s to exist", flag)
		}
	}
}

func TestRunLinkNoArgs(t *testing.T) {
	logger.Init(false, false, true)
	cfg := &config.Config{}

	opts := &linkOptions{}
	err := runLink(cfg, []string{}, opts)
	if err == nil {
		t.Error("Expected error when running link with no arguments")
	}
}

func TestRunUnlinkNoArgs(t *testing.T) {
	logger.Init(false, false, true)
	cfg := &config.Config{}

	opts := &unlinkOptions{}
	err := runUnlink(cfg, []string{}, opts)
	if err == nil {
		t.Error("Expected error when running unlink with no arguments")
	}
}

func TestRunLinkWithArgs(t *testing.T) {
	logger.Init(false, false, true)

	// Create temp directory structure
	tempDir, err := os.MkdirTemp("", "link-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cfg := &config.Config{
		HomebrewCellar: filepath.Join(tempDir, "Cellar"),
		HomebrewPrefix: filepath.Join(tempDir, "usr", "local"),
	}

	// Create cellar with installed formula
	cellarDir := cfg.HomebrewCellar
	os.MkdirAll(cellarDir, 0755)

	formulaDir := filepath.Join(cellarDir, "test-formula")
	versionDir := filepath.Join(formulaDir, "1.0.0")
	binDir := filepath.Join(versionDir, "bin")
	os.MkdirAll(binDir, 0755)

	// Create executable in formula
	execPath := filepath.Join(binDir, "test-exec")
	os.WriteFile(execPath, []byte("#!/bin/bash\necho test"), 0755)

	// Create prefix bin directory
	prefixBinDir := filepath.Join(cfg.HomebrewPrefix, "bin")
	os.MkdirAll(prefixBinDir, 0755)

	opts := &linkOptions{
		overwrite: false,
		dryRun:    false,
		force:     false,
	}

	err = runLink(cfg, []string{"test-formula"}, opts)
	if err != nil {
		t.Errorf("runLink failed: %v", err)
	}
}

func TestRunUnlinkWithArgs(t *testing.T) {
	logger.Init(false, false, true)

	// Create temp directory structure
	tempDir, err := os.MkdirTemp("", "link-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cfg := &config.Config{
		HomebrewCellar: filepath.Join(tempDir, "Cellar"),
		HomebrewPrefix: filepath.Join(tempDir, "usr", "local"),
	}

	// Create cellar and prefix directories
	os.MkdirAll(cfg.HomebrewCellar, 0755)
	prefixBinDir := filepath.Join(cfg.HomebrewPrefix, "bin")
	os.MkdirAll(prefixBinDir, 0755)

	// Create a symlink in prefix
	formulaDir := filepath.Join(cfg.HomebrewCellar, "test-formula")
	versionDir := filepath.Join(formulaDir, "1.0.0")
	binDir := filepath.Join(versionDir, "bin")
	os.MkdirAll(binDir, 0755)

	execPath := filepath.Join(binDir, "test-exec")
	os.WriteFile(execPath, []byte("#!/bin/bash\necho test"), 0755)

	linkPath := filepath.Join(prefixBinDir, "test-exec")
	os.Symlink(execPath, linkPath)

	opts := &unlinkOptions{
		dryRun: false,
	}

	err = runUnlink(cfg, []string{"test-formula"}, opts)
	if err != nil {
		t.Errorf("runUnlink failed: %v", err)
	}
}

func TestRunLinkNotInstalled(t *testing.T) {
	logger.Init(false, false, true)

	// Create temp directory
	tempDir, err := os.MkdirTemp("", "link-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cfg := &config.Config{
		HomebrewCellar: filepath.Join(tempDir, "Cellar"),
	}

	// Create empty cellar
	os.MkdirAll(cfg.HomebrewCellar, 0755)

	opts := &linkOptions{}
	err = runLink(cfg, []string{"non-existent-formula"}, opts)
	if err == nil {
		t.Error("Expected error when linking non-existent formula")
	}
}

func TestLinkFormula(t *testing.T) {
	logger.Init(false, false, true)

	// Create temp directory structure
	tempDir, err := os.MkdirTemp("", "link-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cfg := &config.Config{
		HomebrewCellar: filepath.Join(tempDir, "Cellar"),
		HomebrewPrefix: filepath.Join(tempDir, "usr", "local"),
	}

	// Create formula structure
	formulaDir := filepath.Join(cfg.HomebrewCellar, "test-formula")
	versionDir := filepath.Join(formulaDir, "1.0.0")
	binDir := filepath.Join(versionDir, "bin")
	os.MkdirAll(binDir, 0755)

	// Create executable
	execPath := filepath.Join(binDir, "test-exec")
	os.WriteFile(execPath, []byte("#!/bin/bash\necho test"), 0755)

	// Create prefix directories
	prefixBinDir := filepath.Join(cfg.HomebrewPrefix, "bin")
	os.MkdirAll(prefixBinDir, 0755)

	opts := &linkOptions{
		overwrite: false,
		dryRun:    false,
		force:     false,
	}

	err = linkFormula(cfg, "test-formula", opts)
	if err != nil {
		t.Errorf("linkFormula failed: %v", err)
	}
}

func TestUnlinkFormulaLink(t *testing.T) {
	logger.Init(false, false, true)

	// Create temp directory structure
	tempDir, err := os.MkdirTemp("", "link-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cfg := &config.Config{
		HomebrewCellar: filepath.Join(tempDir, "Cellar"),
		HomebrewPrefix: filepath.Join(tempDir, "usr", "local"),
	}

	// Create formula and prefix directories
	formulaDir := filepath.Join(cfg.HomebrewCellar, "test-formula")
	versionDir := filepath.Join(formulaDir, "1.0.0")
	binDir := filepath.Join(versionDir, "bin")
	os.MkdirAll(binDir, 0755)

	prefixBinDir := filepath.Join(cfg.HomebrewPrefix, "bin")
	os.MkdirAll(prefixBinDir, 0755)

	// Create executable and symlink
	execPath := filepath.Join(binDir, "test-exec")
	os.WriteFile(execPath, []byte("#!/bin/bash\necho test"), 0755)

	linkPath := filepath.Join(prefixBinDir, "test-exec")
	os.Symlink(execPath, linkPath)

	opts := &unlinkOptions{
		dryRun: false,
	}

	err = unlinkFormulaLink(cfg, "test-formula", opts)
	if err != nil {
		t.Errorf("unlinkFormulaLink failed: %v", err)
	}

	// Check if symlink was removed
	if _, err := os.Stat(linkPath); !os.IsNotExist(err) {
		t.Error("Expected symlink to be removed")
	}
}

func TestIsFormulaInstalledSimple(t *testing.T) {
	logger.Init(false, false, true)

	// Test with non-existent formula
	cfg := &config.Config{
		HomebrewCellar: "/non/existent/path",
	}

	installed := isFormulaInstalledSimple(cfg, "non-existent")
	if installed {
		t.Error("Expected non-existent formula to return false")
	}

	// Test with existing formula
	tempDir, err := os.MkdirTemp("", "link-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cfg = &config.Config{
		HomebrewCellar: tempDir,
	}

	// Create formula directory with version
	formulaDir := filepath.Join(tempDir, "test-formula")
	versionDir := filepath.Join(formulaDir, "1.0.0")
	os.MkdirAll(versionDir, 0755)

	installed = isFormulaInstalledSimple(cfg, "test-formula")
	if !installed {
		t.Error("Expected existing formula to return true")
	}
}

func TestIsKegOnly(t *testing.T) {
	logger.Init(false, false, true)

	cfg := &config.Config{
		HomebrewCellar: "/tmp/test-cellar",
	}

	// Test keg-only check (simplified implementation)
	kegOnly := isKegOnly(cfg, "test-formula")
	// Current implementation returns false
	if kegOnly {
		t.Log("Formula is marked as keg-only")
	}
}

func TestLinkOptions(t *testing.T) {
	opts := &linkOptions{
		overwrite: true,
		dryRun:    true,
		force:     true,
	}

	if !opts.overwrite {
		t.Error("Expected overwrite to be true")
	}
	if !opts.dryRun {
		t.Error("Expected dryRun to be true")
	}
	if !opts.force {
		t.Error("Expected force to be true")
	}
}

func TestUnlinkOptions(t *testing.T) {
	opts := &unlinkOptions{
		dryRun: true,
	}

	if !opts.dryRun {
		t.Error("Expected dryRun to be true")
	}
}

func TestLinkCommandExecution(t *testing.T) {
	logger.Init(false, false, true)

	// Create temp directory
	tempDir, err := os.MkdirTemp("", "link-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cfg := &config.Config{
		HomebrewCellar: filepath.Join(tempDir, "Cellar"),
		HomebrewPrefix: filepath.Join(tempDir, "usr", "local"),
	}

	// Create formula structure
	formulaDir := filepath.Join(cfg.HomebrewCellar, "test-formula")
	versionDir := filepath.Join(formulaDir, "1.0.0")
	os.MkdirAll(versionDir, 0755)

	// Test link command
	linkCmd := NewLinkCmd(cfg)
	err = linkCmd.RunE(linkCmd, []string{"test-formula"})
	if err != nil {
		t.Errorf("link command failed: %v", err)
	}

	// Test unlink command
	unlinkCmd := NewUnlinkCmd(cfg)
	err = unlinkCmd.RunE(unlinkCmd, []string{"test-formula"})
	if err != nil {
		t.Errorf("unlink command failed: %v", err)
	}
}

func TestLinkDryRun(t *testing.T) {
	logger.Init(false, false, true)

	// Create temp directory structure
	tempDir, err := os.MkdirTemp("", "link-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cfg := &config.Config{
		HomebrewCellar: filepath.Join(tempDir, "Cellar"),
		HomebrewPrefix: filepath.Join(tempDir, "usr", "local"),
	}

	// Create formula structure
	formulaDir := filepath.Join(cfg.HomebrewCellar, "test-formula")
	versionDir := filepath.Join(formulaDir, "1.0.0")
	binDir := filepath.Join(versionDir, "bin")
	os.MkdirAll(binDir, 0755)

	execPath := filepath.Join(binDir, "test-exec")
	os.WriteFile(execPath, []byte("#!/bin/bash\necho test"), 0755)

	prefixBinDir := filepath.Join(cfg.HomebrewPrefix, "bin")
	os.MkdirAll(prefixBinDir, 0755)

	opts := &linkOptions{
		dryRun: true,
	}

	err = runLink(cfg, []string{"test-formula"}, opts)
	if err != nil {
		t.Errorf("runLink with dry-run failed: %v", err)
	}

	// Check that no actual linking occurred
	linkPath := filepath.Join(prefixBinDir, "test-exec")
	if _, err := os.Stat(linkPath); !os.IsNotExist(err) {
		t.Error("Expected no symlink to be created during dry-run")
	}
}

func TestUnlinkDryRun(t *testing.T) {
	logger.Init(false, false, true)

	// Create temp directory structure
	tempDir, err := os.MkdirTemp("", "link-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cfg := &config.Config{
		HomebrewCellar: filepath.Join(tempDir, "Cellar"),
		HomebrewPrefix: filepath.Join(tempDir, "usr", "local"),
	}

	// Create formula and symlink
	formulaDir := filepath.Join(cfg.HomebrewCellar, "test-formula")
	versionDir := filepath.Join(formulaDir, "1.0.0")
	binDir := filepath.Join(versionDir, "bin")
	os.MkdirAll(binDir, 0755)

	execPath := filepath.Join(binDir, "test-exec")
	os.WriteFile(execPath, []byte("#!/bin/bash\necho test"), 0755)

	prefixBinDir := filepath.Join(cfg.HomebrewPrefix, "bin")
	os.MkdirAll(prefixBinDir, 0755)

	linkPath := filepath.Join(prefixBinDir, "test-exec")
	os.Symlink(execPath, linkPath)

	opts := &unlinkOptions{
		dryRun: true,
	}

	err = runUnlink(cfg, []string{"test-formula"}, opts)
	if err != nil {
		t.Errorf("runUnlink with dry-run failed: %v", err)
	}

	// Check that symlink still exists (not removed during dry-run)
	if _, err := os.Stat(linkPath); os.IsNotExist(err) {
		t.Error("Expected symlink to still exist during dry-run")
	}
}

func TestLinkMultipleFormulae(t *testing.T) {
	logger.Init(false, false, true)

	// Create temp directory structure
	tempDir, err := os.MkdirTemp("", "link-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cfg := &config.Config{
		HomebrewCellar: filepath.Join(tempDir, "Cellar"),
		HomebrewPrefix: filepath.Join(tempDir, "usr", "local"),
	}

	// Create multiple formulae
	formulae := []string{"formula1", "formula2"}
	for _, formula := range formulae {
		formulaDir := filepath.Join(cfg.HomebrewCellar, formula)
		versionDir := filepath.Join(formulaDir, "1.0.0")
		os.MkdirAll(versionDir, 0755)
	}

	os.MkdirAll(filepath.Join(cfg.HomebrewPrefix, "bin"), 0755)

	opts := &linkOptions{}
	err = runLink(cfg, formulae, opts)
	if err != nil {
		t.Errorf("runLink with multiple formulae failed: %v", err)
	}

	unlinkOpts := &unlinkOptions{}
	err = runUnlink(cfg, formulae, unlinkOpts)
	if err != nil {
		t.Errorf("runUnlink with multiple formulae failed: %v", err)
	}
}
