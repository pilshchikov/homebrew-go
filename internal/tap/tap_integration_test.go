package tap

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"strings"

	"github.com/pilshchikov/homebrew-go/internal/config"
	"github.com/pilshchikov/homebrew-go/internal/logger"
)

func TestManagerOperations(t *testing.T) {
	// Initialize logger for tests
	logger.Init(false, false, true)

	tempDir, err := os.MkdirTemp("", "tap-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	cfg := &config.Config{
		HomebrewRepository: tempDir,
	}

	manager := NewManager(cfg)
	if manager == nil {
		t.Error("NewManager should not return nil")
		return
	}

	if manager.cfg != cfg {
		t.Error("Manager config not set correctly")
	}
}

func TestListTapsEmpty(t *testing.T) {
	logger.Init(false, false, true)

	tempDir, err := os.MkdirTemp("", "tap-test-empty")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	cfg := &config.Config{
		HomebrewRepository: tempDir,
	}

	manager := NewManager(cfg)

	// Create the Taps directory but leave it empty
	tapsDir := filepath.Join(tempDir, "Library", "Taps")
	_ = os.MkdirAll(tapsDir, 0755)

	taps, err := manager.ListTaps()
	if err != nil {
		t.Errorf("ListTaps failed: %v", err)
	}

	if len(taps) != 0 {
		t.Errorf("Expected 0 taps in empty directory, got %d", len(taps))
	}
}

func TestGetTapNonExistent(t *testing.T) {
	logger.Init(false, false, true)

	tempDir, err := os.MkdirTemp("", "tap-test-get")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	cfg := &config.Config{
		HomebrewRepository: tempDir,
	}

	manager := NewManager(cfg)

	_, err = manager.GetTap("nonexistent/tap")
	if err == nil {
		t.Error("Expected error for non-existent tap")
	}
}

func TestGetTapPathIntegration(t *testing.T) {
	cfg := &config.Config{
		HomebrewRepository: "/test/repo",
	}

	manager := NewManager(cfg)

	tests := []struct {
		name     string
		expected string
	}{
		{
			name:     "user/repo",
			expected: "/test/repo/Library/Taps/user/homebrew-repo",
		},
		{
			name:     "simple-name",
			expected: "/test/repo/Library/Taps/homebrew/homebrew-simple-name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := manager.getTapPath(tt.name)
			if path != tt.expected {
				t.Errorf("Expected path %s, got %s", tt.expected, path)
			}
		})
	}
}

func TestValidateTapNameIntegration(t *testing.T) {
	manager := &Manager{}

	tests := []struct {
		name        string
		expectError bool
	}{
		{"valid-name", false},
		{"user/repo", false},
		{"", true},
		{"name with spaces", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := manager.validateTapName(tt.name)
			hasError := err != nil

			if tt.expectError && !hasError {
				t.Errorf("Expected error for tap name %q", tt.name)
			}
			if !tt.expectError && hasError {
				t.Errorf("Unexpected error for tap name %q: %v", tt.name, err)
			}
		})
	}
}

func TestGetDefaultRemoteIntegration(t *testing.T) {
	manager := &Manager{}

	tests := []struct {
		name     string
		expected string
	}{
		{
			name:     "user/repo",
			expected: "https://github.com/user/homebrew-repo.git",
		},
		{
			name:     "simple-name",
			expected: "https://github.com/homebrew/homebrew-simple-name.git",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			remote := manager.getDefaultRemote(tt.name)
			if remote != tt.expected {
				t.Errorf("Expected remote %s, got %s", tt.expected, remote)
			}
		})
	}
}

func TestIsTapDirectoryIntegration(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "tap-test-isdir")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	manager := &Manager{}

	// Test non-existent directory
	if manager.isTapDirectory("/nonexistent/path") {
		t.Error("Non-existent directory should not be a tap directory")
	}

	// Test directory without Formula or Casks
	emptyDir := filepath.Join(tempDir, "empty")
	_ = os.MkdirAll(emptyDir, 0755)
	if manager.isTapDirectory(emptyDir) {
		t.Error("Empty directory should not be a tap directory")
	}

	// Test directory with Formula subdirectory
	formulaDir := filepath.Join(tempDir, "with-formula")
	_ = os.MkdirAll(filepath.Join(formulaDir, "Formula"), 0755)
	if !manager.isTapDirectory(formulaDir) {
		t.Error("Directory with Formula subdirectory should be a tap directory")
	}

	// Test directory with Casks subdirectory
	casksDir := filepath.Join(tempDir, "with-casks")
	_ = os.MkdirAll(filepath.Join(casksDir, "Casks"), 0755)
	if !manager.isTapDirectory(casksDir) {
		t.Error("Directory with Casks subdirectory should be a tap directory")
	}
}

func TestCountFormulaeAndCasks(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "tap-test-count")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	manager := &Manager{}

	// Create test tap structure
	formulaDir := filepath.Join(tempDir, "Formula")
	casksDir := filepath.Join(tempDir, "Casks")
	_ = os.MkdirAll(formulaDir, 0755)
	_ = os.MkdirAll(casksDir, 0755)

	// Create some formula files
	for i := 0; i < 3; i++ {
		filename := filepath.Join(formulaDir, fmt.Sprintf("formula%d.rb", i))
		_ = os.WriteFile(filename, []byte("# formula"), 0644)
	}

	// Create some cask files
	for i := 0; i < 2; i++ {
		filename := filepath.Join(casksDir, fmt.Sprintf("cask%d.rb", i))
		_ = os.WriteFile(filename, []byte("# cask"), 0644)
	}

	// Create non-Ruby files (should be ignored)
	_ = os.WriteFile(filepath.Join(formulaDir, "readme.txt"), []byte("readme"), 0644)
	_ = os.WriteFile(filepath.Join(casksDir, "info.md"), []byte("info"), 0644)

	// Test counting
	formulaeCount := manager.countFormulae(tempDir)
	if formulaeCount != 3 {
		t.Errorf("Expected 3 formulae, got %d", formulaeCount)
	}

	casksCount := manager.countCasks(tempDir)
	if casksCount != 2 {
		t.Errorf("Expected 2 casks, got %d", casksCount)
	}

	// Test with non-existent directories
	formulaeCount = manager.countFormulae("/nonexistent")
	if formulaeCount != 0 {
		t.Errorf("Expected 0 formulae for non-existent directory, got %d", formulaeCount)
	}

	casksCount = manager.countCasks("/nonexistent")
	if casksCount != 0 {
		t.Errorf("Expected 0 casks for non-existent directory, got %d", casksCount)
	}
}

func TestVerifyTapIntegration(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "tap-test-verify")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	manager := &Manager{}

	// Test invalid tap (no Formula or Casks)
	emptyDir := filepath.Join(tempDir, "empty")
	_ = os.MkdirAll(emptyDir, 0755)

	err = manager.verifyTap(emptyDir)
	if err == nil {
		t.Error("Expected error for invalid tap")
	}

	// Test valid tap with Formula
	validDir := filepath.Join(tempDir, "valid")
	_ = os.MkdirAll(filepath.Join(validDir, "Formula"), 0755)

	err = manager.verifyTap(validDir)
	if err != nil {
		t.Errorf("Expected no error for valid tap: %v", err)
	}

	// Test valid tap with Casks
	validCasksDir := filepath.Join(tempDir, "valid-casks")
	_ = os.MkdirAll(filepath.Join(validCasksDir, "Casks"), 0755)

	err = manager.verifyTap(validCasksDir)
	if err != nil {
		t.Errorf("Expected no error for valid casks tap: %v", err)
	}
}

func TestProgressWriter(t *testing.T) {
	writer := &ProgressWriter{prefix: "test"}

	// Test writing some data
	data := []byte("test progress message\n")
	n, err := writer.Write(data)

	if err != nil {
		t.Errorf("ProgressWriter.Write failed: %v", err)
	}

	if n != len(data) {
		t.Errorf("Expected to write %d bytes, wrote %d", len(data), n)
	}

	// Test writing empty data
	n, err = writer.Write([]byte(""))
	if err != nil {
		t.Errorf("ProgressWriter.Write failed for empty data: %v", err)
	}

	if n != 0 {
		t.Errorf("Expected to write 0 bytes for empty data, wrote %d", n)
	}
}

func TestAddTap(t *testing.T) {
	logger.Init(false, false, true)

	tempDir, err := os.MkdirTemp("", "tap-test-add")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	cfg := &config.Config{
		HomebrewRepository: tempDir,
	}

	manager := NewManager(cfg)

	// Test validation - empty name
	err = manager.AddTap("", "", nil)
	if err == nil {
		t.Error("Expected error for empty tap name")
	}
	if !strings.Contains(err.Error(), "invalid tap name") {
		t.Errorf("Expected validation error, got: %v", err)
	}

	// Test validation - name with spaces
	err = manager.AddTap("invalid name", "", nil)
	if err == nil {
		t.Error("Expected error for tap name with spaces")
	}
	if !strings.Contains(err.Error(), "cannot contain spaces") {
		t.Errorf("Expected spaces error, got: %v", err)
	}

	// Test with invalid remote (will fail to clone)
	err = manager.AddTap("test/invalid", "https://github.com/nonexistent/repo.git", nil)
	if err == nil {
		t.Error("Expected error for invalid remote")
	}
	if !strings.Contains(err.Error(), "failed to clone") {
		t.Errorf("Expected clone error, got: %v", err)
	}

	// Test default remote generation
	defaultRemote := manager.getDefaultRemote("test/example")
	expectedRemote := "https://github.com/test/homebrew-example.git"
	if defaultRemote != expectedRemote {
		t.Errorf("Expected default remote %s, got %s", expectedRemote, defaultRemote)
	}

	// Test simple name default remote
	simpleRemote := manager.getDefaultRemote("example")
	expectedSimple := "https://github.com/homebrew/homebrew-example.git"
	if simpleRemote != expectedSimple {
		t.Errorf("Expected simple remote %s, got %s", expectedSimple, simpleRemote)
	}
}

func TestRemoveTap(t *testing.T) {
	logger.Init(false, false, true)

	tempDir, err := os.MkdirTemp("", "tap-test-remove")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	cfg := &config.Config{
		HomebrewRepository: tempDir,
		HomebrewCellar:     filepath.Join(tempDir, "Cellar"),
	}

	manager := NewManager(cfg)

	// Test removing non-existent tap
	err = manager.RemoveTap("nonexistent/tap", nil)
	if err == nil {
		t.Error("Expected error for non-existent tap")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Expected 'not found' error, got: %v", err)
	}

	// Create a mock tap structure
	tapPath := filepath.Join(tempDir, "Library", "Taps", "test", "homebrew-example")
	_ = os.MkdirAll(filepath.Join(tapPath, "Formula"), 0755)

	// Create a formula file
	formulaFile := filepath.Join(tapPath, "Formula", "testformula.rb")
	_ = os.WriteFile(formulaFile, []byte("# test formula"), 0644)

	// Test removing tap without installed formulae
	err = manager.RemoveTap("test/example", nil)
	if err != nil {
		t.Errorf("Expected successful removal, got: %v", err)
	}

	// Verify tap was removed
	if _, err := os.Stat(tapPath); !os.IsNotExist(err) {
		t.Error("Expected tap directory to be removed")
	}
}

func TestUpdateTap(t *testing.T) {
	logger.Init(false, false, true)

	tempDir, err := os.MkdirTemp("", "tap-test-update")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	cfg := &config.Config{
		HomebrewRepository: tempDir,
	}

	manager := NewManager(cfg)

	// Test updating non-existent tap
	err = manager.UpdateTap("nonexistent/tap")
	if err == nil {
		t.Error("Expected error for non-existent tap")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Expected 'not found' error, got: %v", err)
	}

	// Create a mock tap structure without git repo
	tapPath := filepath.Join(tempDir, "Library", "Taps", "test", "homebrew-example")
	_ = os.MkdirAll(filepath.Join(tapPath, "Formula"), 0755)

	// Test updating tap without git repository
	err = manager.UpdateTap("test/example")
	if err == nil {
		t.Error("Expected error for tap without git repository")
	}
	if !strings.Contains(err.Error(), "failed to open tap repository") {
		t.Errorf("Expected git error, got: %v", err)
	}
}

func TestGetInstalledFormulaeFromTap(t *testing.T) {
	logger.Init(false, false, true)

	tempDir, err := os.MkdirTemp("", "tap-test-installed")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	cfg := &config.Config{
		HomebrewRepository: tempDir,
		HomebrewCellar:     filepath.Join(tempDir, "Cellar"),
	}

	manager := NewManager(cfg)

	// Create a mock tap structure
	tapPath := filepath.Join(tempDir, "Library", "Taps", "test", "homebrew-example")
	formulaDir := filepath.Join(tapPath, "Formula")
	_ = os.MkdirAll(formulaDir, 0755)

	// Create some formula files
	formulas := []string{"formula1", "formula2", "formula3"}
	for _, formula := range formulas {
		formulaFile := filepath.Join(formulaDir, formula+".rb")
		_ = os.WriteFile(formulaFile, []byte("# "+formula), 0644)
	}

	// Create cellar structure with one installed formula
	cellarDir := filepath.Join(tempDir, "Cellar", "formula1")
	_ = os.MkdirAll(cellarDir, 0755)

	tap := &Tap{
		Name: "test/example",
		Path: tapPath,
	}

	installedFormulae, err := manager.getInstalledFormulaeFromTap(tap)
	if err != nil {
		t.Fatalf("getInstalledFormulaeFromTap failed: %v", err)
	}

	// Should find formula1 as installed
	if len(installedFormulae) != 1 {
		t.Errorf("Expected 1 installed formula, got %d", len(installedFormulae))
	}

	if len(installedFormulae) > 0 && installedFormulae[0] != "formula1" {
		t.Errorf("Expected 'formula1' to be installed, got %v", installedFormulae)
	}
}

func TestIsFormulaFromTap(t *testing.T) {
	logger.Init(false, false, true)

	tempDir, err := os.MkdirTemp("", "tap-test-formula")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	cfg := &config.Config{
		HomebrewRepository: tempDir,
	}

	manager := NewManager(cfg)

	// Create a mock tap structure
	tapPath := filepath.Join(tempDir, "Library", "Taps", "test", "homebrew-example")
	formulaDir := filepath.Join(tapPath, "Formula")
	_ = os.MkdirAll(formulaDir, 0755)

	// Create a formula file
	formulaFile := filepath.Join(formulaDir, "testformula.rb")
	_ = os.WriteFile(formulaFile, []byte("# test formula"), 0644)

	// Test formula that exists in tap
	if !manager.isFormulaFromTap("testformula", "test/example") {
		t.Error("Expected testformula to be from test/example tap")
	}

	// Test formula that doesn't exist in tap
	if manager.isFormulaFromTap("nonexistent", "test/example") {
		t.Error("Expected nonexistent formula to not be from tap")
	}

	// Test YAML formula
	yamlFile := filepath.Join(formulaDir, "yamlformula.yaml")
	_ = os.WriteFile(yamlFile, []byte("# yaml formula"), 0644)

	if !manager.isFormulaFromTap("yamlformula", "test/example") {
		t.Error("Expected yamlformula to be from test/example tap")
	}
}

func TestTapGetFormula(t *testing.T) {
	logger.Init(false, false, true)

	tempDir, err := os.MkdirTemp("", "tap-test-getformula")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	tap := &Tap{
		Name: "test/example",
		Path: tempDir,
	}

	formulaDir := filepath.Join(tempDir, "Formula")
	_ = os.MkdirAll(formulaDir, 0755)

	// Test formula not found
	_, err = tap.GetFormula("nonexistent")
	if err == nil {
		t.Error("Expected error for non-existent formula")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Expected 'not found' error, got: %v", err)
	}

	// Create a Ruby formula file (should indicate not implemented)
	rubyFile := filepath.Join(formulaDir, "rubyformula.rb")
	_ = os.WriteFile(rubyFile, []byte("# ruby formula"), 0644)

	_, err = tap.GetFormula("rubyformula")
	if err == nil {
		t.Error("Expected error for Ruby DSL not implemented")
	}
	if !strings.Contains(err.Error(), "Ruby DSL parsing not implemented") {
		t.Errorf("Expected Ruby DSL error, got: %v", err)
	}

	// Test YAML formula (would need actual YAML content for parsing)
	yamlFile := filepath.Join(formulaDir, "yamlformula.yaml")
	invalidYaml := []byte("invalid: yaml: content")
	_ = os.WriteFile(yamlFile, invalidYaml, 0644)

	_, err = tap.GetFormula("yamlformula")
	if err == nil {
		t.Error("Expected error for invalid YAML")
	}
	if !strings.Contains(err.Error(), "failed to parse formula") {
		t.Errorf("Expected parse error, got: %v", err)
	}
}

func TestTapListFormulae(t *testing.T) {
	logger.Init(false, false, true)

	tempDir, err := os.MkdirTemp("", "tap-test-listformulae")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	tap := &Tap{
		Name: "test/example",
		Path: tempDir,
	}

	formulaDir := filepath.Join(tempDir, "Formula")
	_ = os.MkdirAll(formulaDir, 0755)

	// Test empty formula directory
	formulae, err := tap.ListFormulae()
	if err != nil {
		t.Fatalf("ListFormulae failed: %v", err)
	}

	if len(formulae) != 0 {
		t.Errorf("Expected 0 formulae in empty directory, got %d", len(formulae))
	}

	// Create some formula files
	formulaFiles := []string{
		"formula1.rb",
		"formula2.yaml",
		"formula3.rb",
		"readme.txt", // Should be ignored
		"subdir",     // Directory, should be ignored
	}

	for _, filename := range formulaFiles {
		filePath := filepath.Join(formulaDir, filename)
		if filename == "subdir" {
			_ = os.MkdirAll(filePath, 0755)
		} else {
			_ = os.WriteFile(filePath, []byte("# "+filename), 0644)
		}
	}

	formulae, err = tap.ListFormulae()
	if err != nil {
		t.Fatalf("ListFormulae failed: %v", err)
	}

	// Should find 3 formulae (2 .rb + 1 .yaml, excluding .txt and directory)
	expected := []string{"formula1", "formula2", "formula3"}
	if len(formulae) != len(expected) {
		t.Errorf("Expected %d formulae, got %d", len(expected), len(formulae))
	}

	// Check that results are sorted
	for i, formula := range formulae {
		if i < len(expected) && formula != expected[i] {
			t.Errorf("Expected formula %s at index %d, got %s", expected[i], i, formula)
		}
	}

	// Test non-existent directory
	nonExistentTap := &Tap{
		Name: "nonexistent/tap",
		Path: "/nonexistent/path",
	}

	_, err = nonExistentTap.ListFormulae()
	if err == nil {
		t.Error("Expected error for non-existent formula directory")
	}
}
