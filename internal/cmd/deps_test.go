package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/pilshchikov/homebrew-go/internal/config"
	"github.com/pilshchikov/homebrew-go/internal/logger"
)

func TestNewDepsCmd(t *testing.T) {
	logger.Init(false, false, true)
	cfg := &config.Config{}

	cmd := NewDepsCmd(cfg)

	if cmd.Use != "deps [OPTIONS] FORMULA..." {
		t.Errorf("Expected Use to be 'deps [OPTIONS] FORMULA...', got %s", cmd.Use)
	}

	if cmd.Short != "Show dependencies for formulae" {
		t.Errorf("Expected Short to be 'Show dependencies for formulae', got %s", cmd.Short)
	}

	// Test flags exist
	flags := []string{"installed", "missing", "dependents", "include-optional", "include-build", "include-test", "tree", "top-level", "annotate"}
	for _, flag := range flags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("Expected flag %s to exist", flag)
		}
	}
}

func TestRunDeps(t *testing.T) {
	logger.Init(false, false, true)
	cfg := &config.Config{}

	// Test basic deps run
	opts := &depsOptions{
		showInstalled:   false,
		showMissing:     false,
		showDependents:  false,
		includeOptional: false,
		includeBuild:    false,
		includeTest:     false,
		tree:            false,
		topLevel:        false,
		annotate:        false,
	}

	err := runDeps(cfg, []string{"test-formula"}, opts)
	if err != nil {
		t.Errorf("runDeps failed: %v", err)
	}
}

func TestRunDepsTree(t *testing.T) {
	logger.Init(false, false, true)
	cfg := &config.Config{}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	opts := &depsOptions{tree: true}
	err := runDeps(cfg, []string{"test-formula"}, opts)

	// Restore stdout
	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Errorf("runDeps with tree failed: %v", err)
	}

	// Read captured output
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	if output == "" {
		t.Error("Expected tree output, got empty string")
	}
}

func TestRunDepsDependents(t *testing.T) {
	logger.Init(false, false, true)
	cfg := &config.Config{}

	opts := &depsOptions{showDependents: true}
	err := runDeps(cfg, []string{"test-formula"}, opts)

	if err != nil {
		t.Errorf("runDeps with dependents failed: %v", err)
	}
}

func TestIsFormulaInstalledDeps(t *testing.T) {
	logger.Init(false, false, true)
	cfg := &config.Config{
		HomebrewCellar: "/tmp/test-cellar",
	}

	// Test with non-existent formula
	installed := isFormulaInstalledDeps(cfg, "non-existent-formula")
	if installed {
		t.Error("Expected non-existent formula to return false")
	}
}

func TestShowDependents(t *testing.T) {
	logger.Init(false, false, true)
	cfg := &config.Config{}
	opts := &depsOptions{}

	err := showDependents(cfg, []string{"test-formula"}, opts)
	if err != nil {
		t.Errorf("showDependents failed: %v", err)
	}
}

func TestShowDepsTree(t *testing.T) {
	logger.Init(false, false, true)
	cfg := &config.Config{}
	opts := &depsOptions{}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := showDepsTree(cfg, []string{"test-formula", "another-formula"}, opts)

	// Restore stdout
	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Errorf("showDepsTree failed: %v", err)
	}

	// Read captured output
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	if output == "" {
		t.Error("Expected tree output, got empty string")
	}
}

// Test depsOptions struct
func TestDepsOptions(t *testing.T) {
	opts := &depsOptions{
		showInstalled:   true,
		showMissing:     true,
		showDependents:  true,
		includeOptional: true,
		includeBuild:    true,
		includeTest:     true,
		tree:            true,
		topLevel:        true,
		annotate:        true,
	}

	if !opts.showInstalled {
		t.Error("Expected showInstalled to be true")
	}
	if !opts.tree {
		t.Error("Expected tree to be true")
	}
}

func TestDepsCommandExecution(t *testing.T) {
	logger.Init(false, false, true)
	cfg := &config.Config{}
	cmd := NewDepsCmd(cfg)

	// Test command execution with no args (should fail)
	err := cmd.RunE(cmd, []string{})
	if err == nil {
		t.Error("Expected error when running deps with no arguments")
	}

	// Test command execution with args
	err = cmd.RunE(cmd, []string{"test-formula"})
	if err != nil {
		t.Errorf("deps command failed: %v", err)
	}
}

func TestDepsWithTempDir(t *testing.T) {
	logger.Init(false, false, true)

	// Create temporary directory structure
	tempDir, err := os.MkdirTemp("", "deps-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	cfg := &config.Config{
		HomebrewCellar: filepath.Join(tempDir, "Cellar"),
		HomebrewPrefix: filepath.Join(tempDir, "usr", "local"),
	}

	// Create cellar directory
	_ = os.MkdirAll(cfg.HomebrewCellar, 0755)

	opts := &depsOptions{}
	err = runDeps(cfg, []string{"test-formula"}, opts)
	if err != nil {
		t.Errorf("runDeps with temp dir failed: %v", err)
	}
}
