package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pilshchikov/homebrew-go/internal/config"
	"github.com/pilshchikov/homebrew-go/internal/logger"
)

func TestNewPinCmd(t *testing.T) {
	logger.Init(false, false, true)
	cfg := &config.Config{}

	cmd := NewPinCmd(cfg)

	if cmd.Use != "pin FORMULA..." {
		t.Errorf("Expected Use to be 'pin FORMULA...', got %s", cmd.Use)
	}

	if cmd.Short != "Pin specified formulae to their current versions" {
		t.Errorf("Expected correct Short description, got %s", cmd.Short)
	}
}

func TestNewUnpinCmd(t *testing.T) {
	logger.Init(false, false, true)
	cfg := &config.Config{}

	cmd := NewUnpinCmd(cfg)

	if cmd.Use != "unpin FORMULA..." {
		t.Errorf("Expected Use to be 'unpin FORMULA...', got %s", cmd.Use)
	}

	if cmd.Short != "Unpin specified formulae, allowing them to be upgraded" {
		t.Errorf("Expected correct Short description, got %s", cmd.Short)
	}
}

func TestRunPinNoArgs(t *testing.T) {
	logger.Init(false, false, true)
	cfg := &config.Config{}

	err := runPin(cfg, []string{})
	if err == nil {
		t.Error("Expected error when running pin with no arguments")
	}
}

func TestRunUnpinNoArgs(t *testing.T) {
	logger.Init(false, false, true)
	cfg := &config.Config{}

	err := runUnpin(cfg, []string{})
	if err == nil {
		t.Error("Expected error when running unpin with no arguments")
	}
}

func TestRunPinWithArgs(t *testing.T) {
	logger.Init(false, false, true)

	// Create temp directory structure
	tempDir, err := os.MkdirTemp("", "pin-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cfg := &config.Config{
		HomebrewCellar:  filepath.Join(tempDir, "Cellar"),
		HomebrewLibrary: filepath.Join(tempDir, "Library"),
	}

	// Create cellar with installed formula
	cellarDir := cfg.HomebrewCellar
	os.MkdirAll(cellarDir, 0755)

	formulaDir := filepath.Join(cellarDir, "test-formula")
	versionDir := filepath.Join(formulaDir, "1.0.0")
	os.MkdirAll(versionDir, 0755)

	// Create library directory
	os.MkdirAll(cfg.HomebrewLibrary, 0755)

	err = runPin(cfg, []string{"test-formula"})
	if err != nil {
		t.Errorf("runPin failed: %v", err)
	}

	// Check if pin file was created
	pinDir := filepath.Join(cfg.HomebrewLibrary, "PinnedKegs")
	pinFile := filepath.Join(pinDir, "test-formula")

	if _, err := os.Stat(pinFile); os.IsNotExist(err) {
		t.Error("Expected pin file to be created")
	}
}

func TestRunUnpinWithArgs(t *testing.T) {
	logger.Init(false, false, true)

	// Create temp directory structure
	tempDir, err := os.MkdirTemp("", "pin-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cfg := &config.Config{
		HomebrewLibrary: filepath.Join(tempDir, "Library"),
	}

	// Create library and pin directories
	pinDir := filepath.Join(cfg.HomebrewLibrary, "PinnedKegs")
	os.MkdirAll(pinDir, 0755)

	// Create pin file
	pinFile := filepath.Join(pinDir, "test-formula")
	os.WriteFile(pinFile, []byte("pinned"), 0644)

	err = runUnpin(cfg, []string{"test-formula"})
	if err != nil {
		t.Errorf("runUnpin failed: %v", err)
	}

	// Check if pin file was removed
	if _, err := os.Stat(pinFile); !os.IsNotExist(err) {
		t.Error("Expected pin file to be removed")
	}
}

func TestRunPinNotInstalled(t *testing.T) {
	logger.Init(false, false, true)

	// Create temp directory
	tempDir, err := os.MkdirTemp("", "pin-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cfg := &config.Config{
		HomebrewCellar: filepath.Join(tempDir, "Cellar"),
	}

	// Create empty cellar
	os.MkdirAll(cfg.HomebrewCellar, 0755)

	err = runPin(cfg, []string{"non-existent-formula"})
	if err == nil {
		t.Error("Expected error when pinning non-existent formula")
	}
}

func TestRunUnpinNotPinned(t *testing.T) {
	logger.Init(false, false, true)

	// Create temp directory
	tempDir, err := os.MkdirTemp("", "pin-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cfg := &config.Config{
		HomebrewLibrary: filepath.Join(tempDir, "Library"),
	}

	// Create library directory but no pin
	os.MkdirAll(cfg.HomebrewLibrary, 0755)

	err = runUnpin(cfg, []string{"unpinned-formula"})
	if err == nil {
		t.Error("Expected error when unpinning non-pinned formula")
	}
}

func TestCreatePinFile(t *testing.T) {
	logger.Init(false, false, true)

	// Create temp directory
	tempDir, err := os.MkdirTemp("", "pin-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	pinFile := filepath.Join(tempDir, "test-formula")

	err = createPinFile(pinFile, "test-formula")
	if err != nil {
		t.Errorf("createPinFile failed: %v", err)
	}

	// Check if pin file was created
	if _, err := os.Stat(pinFile); os.IsNotExist(err) {
		t.Error("Expected pin file to be created")
	}
}

func TestRemovePinFile(t *testing.T) {
	logger.Init(false, false, true)

	// Create temp directory structure
	tempDir, err := os.MkdirTemp("", "pin-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create pin file
	pinFile := filepath.Join(tempDir, "test-formula")
	os.WriteFile(pinFile, []byte("pinned"), 0644)

	err = os.Remove(pinFile)
	if err != nil {
		t.Errorf("Failed to remove pin file: %v", err)
	}

	// Check if pin file was removed
	if _, err := os.Stat(pinFile); !os.IsNotExist(err) {
		t.Error("Expected pin file to be removed")
	}
}

func TestIsFormulaInstalledPin(t *testing.T) {
	logger.Init(false, false, true)

	// Test with non-existent formula
	cfg := &config.Config{
		HomebrewCellar: "/non/existent/path",
	}

	installed := isFormulaInstalledPin(cfg, "non-existent")
	if installed {
		t.Error("Expected non-existent formula to return false")
	}

	// Test with existing formula
	tempDir, err := os.MkdirTemp("", "pin-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cfg = &config.Config{
		HomebrewCellar: tempDir,
	}

	// Create formula directory
	formulaDir := filepath.Join(tempDir, "test-formula")
	versionDir := filepath.Join(formulaDir, "1.0.0")
	os.MkdirAll(versionDir, 0755)

	installed = isFormulaInstalledPin(cfg, "test-formula")
	if !installed {
		t.Error("Expected existing formula to return true")
	}
}

func TestPinCommandExecution(t *testing.T) {
	logger.Init(false, false, true)

	// Create temp directory
	tempDir, err := os.MkdirTemp("", "pin-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cfg := &config.Config{
		HomebrewCellar:  filepath.Join(tempDir, "Cellar"),
		HomebrewLibrary: filepath.Join(tempDir, "Library"),
	}

	// Create cellar with installed formula
	cellarDir := cfg.HomebrewCellar
	os.MkdirAll(cellarDir, 0755)

	formulaDir := filepath.Join(cellarDir, "test-formula")
	versionDir := filepath.Join(formulaDir, "1.0.0")
	os.MkdirAll(versionDir, 0755)

	// Test pin command
	pinCmd := NewPinCmd(cfg)
	err = pinCmd.RunE(pinCmd, []string{"test-formula"})
	if err != nil {
		t.Errorf("pin command failed: %v", err)
	}

	// Test unpin command
	unpinCmd := NewUnpinCmd(cfg)
	err = unpinCmd.RunE(unpinCmd, []string{"test-formula"})
	if err != nil {
		t.Errorf("unpin command failed: %v", err)
	}
}

func TestPinMultipleFormulae(t *testing.T) {
	logger.Init(false, false, true)

	// Create temp directory structure
	tempDir, err := os.MkdirTemp("", "pin-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cfg := &config.Config{
		HomebrewCellar:  filepath.Join(tempDir, "Cellar"),
		HomebrewLibrary: filepath.Join(tempDir, "Library"),
	}

	// Create cellar with multiple installed formulae
	cellarDir := cfg.HomebrewCellar
	os.MkdirAll(cellarDir, 0755)

	formulae := []string{"formula1", "formula2", "formula3"}
	for _, formula := range formulae {
		formulaDir := filepath.Join(cellarDir, formula)
		versionDir := filepath.Join(formulaDir, "1.0.0")
		os.MkdirAll(versionDir, 0755)
	}

	// Create library directory
	os.MkdirAll(cfg.HomebrewLibrary, 0755)

	// Pin all formulae
	err = runPin(cfg, formulae)
	if err != nil {
		t.Errorf("runPin with multiple formulae failed: %v", err)
	}

	// Check if all pin files were created
	pinDir := filepath.Join(cfg.HomebrewLibrary, "PinnedKegs")
	for _, formula := range formulae {
		pinFile := filepath.Join(pinDir, formula)
		if _, err := os.Stat(pinFile); os.IsNotExist(err) {
			t.Errorf("Expected pin file for %s to be created", formula)
		}
	}

	// Unpin all formulae
	err = runUnpin(cfg, formulae)
	if err != nil {
		t.Errorf("runUnpin with multiple formulae failed: %v", err)
	}

	// Check if all pin files were removed
	for _, formula := range formulae {
		pinFile := filepath.Join(pinDir, formula)
		if _, err := os.Stat(pinFile); !os.IsNotExist(err) {
			t.Errorf("Expected pin file for %s to be removed", formula)
		}
	}
}
