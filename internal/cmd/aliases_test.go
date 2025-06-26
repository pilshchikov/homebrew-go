package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pilshchikov/homebrew-go/internal/config"
	"github.com/pilshchikov/homebrew-go/internal/logger"
)

func TestNewHomeCmd(t *testing.T) {
	logger.Init(false, false, true)
	cfg := &config.Config{}

	cmd := NewHomeCmd(cfg)

	if cmd.Use != "home [FORMULA...]" {
		t.Errorf("Expected Use to be 'home [FORMULA...]', got %s", cmd.Use)
	}

	if cmd.Short != "Open a formula or cask's homepage in a browser" {
		t.Errorf("Expected correct Short description, got %s", cmd.Short)
	}

	// Check aliases
	aliases := cmd.Aliases
	if len(aliases) != 1 || aliases[0] != "homepage" {
		t.Errorf("Expected alias 'homepage', got %v", aliases)
	}
}

func TestNewUsesCmd(t *testing.T) {
	logger.Init(false, false, true)
	cfg := &config.Config{}

	cmd := NewUsesCmd(cfg)

	if cmd.Use != "uses [OPTIONS] FORMULA" {
		t.Errorf("Expected Use to be 'uses [OPTIONS] FORMULA', got %s", cmd.Use)
	}

	if cmd.Short != "Show formulae and casks that specify formula as a dependency" {
		t.Errorf("Expected correct Short description, got %s", cmd.Short)
	}

	// Test flags exist
	flags := []string{"installed", "recursive", "include-test", "include-build"}
	for _, flag := range flags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("Expected flag %s to exist", flag)
		}
	}
}

func TestNewDescCmd(t *testing.T) {
	logger.Init(false, false, true)
	cfg := &config.Config{}

	cmd := NewDescCmd(cfg)

	if cmd.Use != "desc [OPTIONS] FORMULA|TEXT" {
		t.Errorf("Expected Use to be 'desc [OPTIONS] FORMULA|TEXT', got %s", cmd.Use)
	}

	if cmd.Short != "Display a formula's name and one-line description" {
		t.Errorf("Expected correct Short description, got %s", cmd.Short)
	}

	// Test flags exist
	flags := []string{"search", "name", "eval-all"}
	for _, flag := range flags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("Expected flag %s to exist", flag)
		}
	}
}

func TestNewOptionsCmd(t *testing.T) {
	logger.Init(false, false, true)
	cfg := &config.Config{}

	cmd := NewOptionsCmd(cfg)

	if cmd.Use != "options [OPTIONS] [FORMULA...]" {
		t.Errorf("Expected Use to be 'options [OPTIONS] [FORMULA...]', got %s", cmd.Use)
	}

	if cmd.Short != "Show install options specific to formula" {
		t.Errorf("Expected correct Short description, got %s", cmd.Short)
	}

	// Test flags exist
	flags := []string{"compact", "installed", "all"}
	for _, flag := range flags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("Expected flag %s to exist", flag)
		}
	}
}

func TestNewMissingCmd(t *testing.T) {
	logger.Init(false, false, true)
	cfg := &config.Config{}

	cmd := NewMissingCmd(cfg)

	if cmd.Use != "missing [OPTIONS] [FORMULA...]" {
		t.Errorf("Expected Use to be 'missing [OPTIONS] [FORMULA...]', got %s", cmd.Use)
	}

	if cmd.Short != "Check the given formulae for missing dependencies" {
		t.Errorf("Expected correct Short description, got %s", cmd.Short)
	}

	// Test flags exist
	if cmd.Flags().Lookup("hide") == nil {
		t.Error("Expected flag 'hide' to exist")
	}
}

func TestOpenURL(t *testing.T) {
	logger.Init(false, false, true)

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := openURL("https://example.com")

	// Restore stdout
	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Errorf("openURL failed: %v", err)
	}

	// Read captured output
	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if !strings.Contains(output, "https://example.com") {
		t.Error("Expected output to contain the URL")
	}
}

func TestOpenFormulaHomepages(t *testing.T) {
	logger.Init(false, false, true)
	cfg := &config.Config{}

	formulaNames := []string{"git", "node", "python"}
	err := openFormulaHomepages(cfg, formulaNames)
	if err != nil {
		t.Errorf("openFormulaHomepages failed: %v", err)
	}
}

func TestRunUses(t *testing.T) {
	logger.Init(false, false, true)
	cfg := &config.Config{}

	opts := &usesOptions{
		installed:    false,
		recursive:    false,
		includeTest:  false,
		includeBuild: false,
	}

	err := runUses(cfg, "test-formula", opts)
	if err != nil {
		t.Errorf("runUses failed: %v", err)
	}
}

func TestRunUsesWithOptions(t *testing.T) {
	logger.Init(false, false, true)
	cfg := &config.Config{}

	opts := &usesOptions{
		installed:    true,
		recursive:    true,
		includeTest:  true,
		includeBuild: true,
	}

	err := runUses(cfg, "test-formula", opts)
	if err != nil {
		t.Errorf("runUses with options failed: %v", err)
	}
}

func TestRunDesc(t *testing.T) {
	logger.Init(false, false, true)
	cfg := &config.Config{}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	opts := &descOptions{
		searchDesc: false,
		name:       false,
		eval:       false,
	}

	err := runDesc(cfg, []string{"test-formula"}, opts)

	// Restore stdout
	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Errorf("runDesc failed: %v", err)
	}

	// Read captured output
	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if !strings.Contains(output, "test-formula") {
		t.Error("Expected output to contain formula name")
	}
}

func TestRunDescSearch(t *testing.T) {
	logger.Init(false, false, true)
	cfg := &config.Config{}

	opts := &descOptions{
		searchDesc: true,
		name:       false,
		eval:       false,
	}

	err := runDesc(cfg, []string{"search-term"}, opts)
	if err != nil {
		t.Errorf("runDesc with search failed: %v", err)
	}
}

func TestSearchDescriptions(t *testing.T) {
	logger.Init(false, false, true)
	cfg := &config.Config{}
	opts := &descOptions{}

	err := searchDescriptions(cfg, []string{"test", "query"}, opts)
	if err != nil {
		t.Errorf("searchDescriptions failed: %v", err)
	}
}

func TestGetFormulaDescription(t *testing.T) {
	logger.Init(false, false, true)
	cfg := &config.Config{}

	desc, err := getFormulaDescription(cfg, "test-formula")
	if err != nil {
		t.Errorf("getFormulaDescription failed: %v", err)
	}

	if desc != "Formula description" {
		t.Errorf("Expected 'Formula description', got %s", desc)
	}
}

func TestRunOptions(t *testing.T) {
	logger.Init(false, false, true)
	cfg := &config.Config{}

	// Test with no args and no flags (should error)
	opts := &optionsOptions{
		compact:   false,
		installed: false,
		all:       false,
	}

	err := runOptions(cfg, []string{}, opts)
	if err == nil {
		t.Error("Expected error when running options with no arguments or flags")
	}
}

func TestRunOptionsWithFormulae(t *testing.T) {
	logger.Init(false, false, true)
	cfg := &config.Config{}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	opts := &optionsOptions{
		compact:   false,
		installed: false,
		all:       false,
	}

	err := runOptions(cfg, []string{"test-formula"}, opts)

	// Restore stdout
	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Errorf("runOptions with formulae failed: %v", err)
	}

	// Read captured output
	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if !strings.Contains(output, "no options available") {
		t.Error("Expected output to contain 'no options available'")
	}
}

func TestRunOptionsAll(t *testing.T) {
	logger.Init(false, false, true)
	cfg := &config.Config{}

	opts := &optionsOptions{
		all: true,
	}

	err := runOptions(cfg, []string{}, opts)
	if err != nil {
		t.Errorf("runOptions with --all failed: %v", err)
	}
}

func TestRunOptionsInstalled(t *testing.T) {
	logger.Init(false, false, true)
	cfg := &config.Config{}

	opts := &optionsOptions{
		installed: true,
	}

	err := runOptions(cfg, []string{}, opts)
	if err != nil {
		t.Errorf("runOptions with --installed failed: %v", err)
	}
}

func TestRunMissingNoArgs(t *testing.T) {
	logger.Init(false, false, true)

	// Create temp directory
	tempDir, err := os.MkdirTemp("", "missing-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cfg := &config.Config{
		HomebrewCellar: filepath.Join(tempDir, "Cellar"),
	}

	// Create empty cellar
	os.MkdirAll(cfg.HomebrewCellar, 0755)

	err = runMissing(cfg, []string{}, []string{})
	if err != nil {
		t.Errorf("runMissing with no args failed: %v", err)
	}
}

func TestRunMissingWithArgs(t *testing.T) {
	logger.Init(false, false, true)
	cfg := &config.Config{}

	err := runMissing(cfg, []string{"test-formula"}, []string{})
	if err != nil {
		t.Errorf("runMissing with args failed: %v", err)
	}
}

func TestRunMissingWithHide(t *testing.T) {
	logger.Init(false, false, true)
	cfg := &config.Config{}

	err := runMissing(cfg, []string{"formula1", "formula2"}, []string{"formula1"})
	if err != nil {
		t.Errorf("runMissing with hide failed: %v", err)
	}
}

func TestUsesOptions(t *testing.T) {
	opts := &usesOptions{
		installed:    true,
		recursive:    true,
		includeTest:  true,
		includeBuild: true,
	}

	if !opts.installed {
		t.Error("Expected installed to be true")
	}
	if !opts.recursive {
		t.Error("Expected recursive to be true")
	}
	if !opts.includeTest {
		t.Error("Expected includeTest to be true")
	}
	if !opts.includeBuild {
		t.Error("Expected includeBuild to be true")
	}
}

func TestDescOptions(t *testing.T) {
	opts := &descOptions{
		searchDesc: true,
		name:       true,
		eval:       true,
	}

	if !opts.searchDesc {
		t.Error("Expected searchDesc to be true")
	}
	if !opts.name {
		t.Error("Expected name to be true")
	}
	if !opts.eval {
		t.Error("Expected eval to be true")
	}
}

func TestOptionsOptions(t *testing.T) {
	opts := &optionsOptions{
		compact:   true,
		installed: true,
		all:       true,
	}

	if !opts.compact {
		t.Error("Expected compact to be true")
	}
	if !opts.installed {
		t.Error("Expected installed to be true")
	}
	if !opts.all {
		t.Error("Expected all to be true")
	}
}

func TestHomeCommandExecution(t *testing.T) {
	logger.Init(false, false, true)
	cfg := &config.Config{}
	cmd := NewHomeCmd(cfg)

	// Test command execution with no args (should open homebrew homepage)
	err := cmd.RunE(cmd, []string{})
	if err != nil {
		t.Errorf("home command with no args failed: %v", err)
	}

	// Test command execution with args
	err = cmd.RunE(cmd, []string{"git"})
	if err != nil {
		t.Errorf("home command with args failed: %v", err)
	}
}

func TestUsesCommandExecution(t *testing.T) {
	logger.Init(false, false, true)
	cfg := &config.Config{}
	cmd := NewUsesCmd(cfg)

	// Test command execution with args
	err := cmd.RunE(cmd, []string{"git"})
	if err != nil {
		t.Errorf("uses command failed: %v", err)
	}
}

func TestDescCommandExecution(t *testing.T) {
	logger.Init(false, false, true)
	cfg := &config.Config{}
	cmd := NewDescCmd(cfg)

	// Test command execution with args
	err := cmd.RunE(cmd, []string{"git"})
	if err != nil {
		t.Errorf("desc command failed: %v", err)
	}
}

func TestOptionsCommandExecution(t *testing.T) {
	logger.Init(false, false, true)
	cfg := &config.Config{}
	cmd := NewOptionsCmd(cfg)

	// Test command execution with args
	err := cmd.RunE(cmd, []string{"git"})
	if err != nil {
		t.Errorf("options command failed: %v", err)
	}
}

func TestMissingCommandExecution(t *testing.T) {
	logger.Init(false, false, true)

	// Create temp directory
	tempDir, err := os.MkdirTemp("", "missing-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cfg := &config.Config{
		HomebrewCellar: filepath.Join(tempDir, "Cellar"),
	}

	os.MkdirAll(cfg.HomebrewCellar, 0755)

	cmd := NewMissingCmd(cfg)

	// Test command execution with no args
	err = cmd.RunE(cmd, []string{})
	if err != nil {
		t.Errorf("missing command failed: %v", err)
	}
}
