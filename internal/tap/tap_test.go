package tap

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/homebrew/brew/internal/config"
)

func TestNewManager(t *testing.T) {
	cfg := &config.Config{}
	manager := NewManager(cfg)

	if manager.cfg != cfg {
		t.Error("Manager should store config reference")
	}
}

func TestValidateTapName(t *testing.T) {
	cfg := &config.Config{}
	manager := NewManager(cfg)

	tests := []struct {
		name    string
		tapName string
		wantErr bool
	}{
		{"valid tap name", "user/repo", false},
		{"valid short name", "myrepo", false},
		{"empty name", "", true},
		{"name with spaces", "user name/repo", true},
		{"name with special chars", "user@example/repo", false}, // This might be valid in some contexts
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := manager.validateTapName(tt.tapName)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateTapName() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetDefaultRemote(t *testing.T) {
	cfg := &config.Config{}
	manager := NewManager(cfg)

	tests := []struct {
		name     string
		tapName  string
		expected string
	}{
		{
			name:     "full tap name",
			tapName:  "user/repo",
			expected: "https://github.com/user/homebrew-repo.git",
		},
		{
			name:     "short tap name",
			tapName:  "myrepo",
			expected: "https://github.com/homebrew/homebrew-myrepo.git",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := manager.getDefaultRemote(tt.tapName)
			if result != tt.expected {
				t.Errorf("getDefaultRemote() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGetTapPath(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		HomebrewRepository: tmpDir,
	}
	manager := NewManager(cfg)

	tests := []struct {
		name     string
		tapName  string
		expected string
	}{
		{
			name:     "full tap name",
			tapName:  "user/repo",
			expected: filepath.Join(tmpDir, "Library", "Taps", "user", "homebrew-repo"),
		},
		{
			name:     "short tap name",
			tapName:  "myrepo",
			expected: filepath.Join(tmpDir, "Library", "Taps", "homebrew", "homebrew-myrepo"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := manager.getTapPath(tt.tapName)
			if result != tt.expected {
				t.Errorf("getTapPath() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestIsTapDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{}
	manager := NewManager(cfg)

	// Test empty directory
	emptyDir := filepath.Join(tmpDir, "empty")
	err := os.MkdirAll(emptyDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create empty directory: %v", err)
	}

	if manager.isTapDirectory(emptyDir) {
		t.Error("Empty directory should not be a tap directory")
	}

	// Test directory with Formula subdirectory
	formulaDir := filepath.Join(tmpDir, "with-formula", "Formula")
	err = os.MkdirAll(formulaDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create formula directory: %v", err)
	}

	if !manager.isTapDirectory(filepath.Join(tmpDir, "with-formula")) {
		t.Error("Directory with Formula subdirectory should be a tap directory")
	}

	// Test directory with Casks subdirectory
	casksDir := filepath.Join(tmpDir, "with-casks", "Casks")
	err = os.MkdirAll(casksDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create casks directory: %v", err)
	}

	if !manager.isTapDirectory(filepath.Join(tmpDir, "with-casks")) {
		t.Error("Directory with Casks subdirectory should be a tap directory")
	}

	// Test non-existent directory
	if manager.isTapDirectory("/non/existent/directory") {
		t.Error("Non-existent directory should not be a tap directory")
	}
}

func TestCountFormulae(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{}
	manager := NewManager(cfg)

	// Create Formula directory with test formulae
	formulaDir := filepath.Join(tmpDir, "Formula")
	err := os.MkdirAll(formulaDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create formula directory: %v", err)
	}

	// Create test formula files
	testFormulae := []string{"wget.rb", "curl.rb", "python.rb", "not-a-formula.txt"}
	for _, formula := range testFormulae {
		filePath := filepath.Join(formulaDir, formula)
		err := os.WriteFile(filePath, []byte("# Test formula"), 0644)
		if err != nil {
			t.Fatalf("Failed to create test formula %s: %v", formula, err)
		}
	}

	count := manager.countFormulae(tmpDir)
	expectedCount := 3 // Only .rb files should be counted

	if count != expectedCount {
		t.Errorf("countFormulae() = %v, want %v", count, expectedCount)
	}

	// Test directory without Formula subdirectory
	emptyDir := filepath.Join(tmpDir, "empty")
	err = os.MkdirAll(emptyDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create empty directory: %v", err)
	}

	count = manager.countFormulae(emptyDir)
	if count != 0 {
		t.Errorf("countFormulae() for directory without Formula = %v, want 0", count)
	}
}

func TestCountCasks(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{}
	manager := NewManager(cfg)

	// Create Casks directory with test casks
	casksDir := filepath.Join(tmpDir, "Casks")
	err := os.MkdirAll(casksDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create casks directory: %v", err)
	}

	// Create test cask files
	testCasks := []string{"firefox.rb", "chrome.rb", "not-a-cask.txt"}
	for _, cask := range testCasks {
		filePath := filepath.Join(casksDir, cask)
		err := os.WriteFile(filePath, []byte("# Test cask"), 0644)
		if err != nil {
			t.Fatalf("Failed to create test cask %s: %v", cask, err)
		}
	}

	count := manager.countCasks(tmpDir)
	expectedCount := 2 // Only .rb files should be counted

	if count != expectedCount {
		t.Errorf("countCasks() = %v, want %v", count, expectedCount)
	}
}

func TestLoadTap(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		HomebrewRepository: tmpDir,
	}
	manager := NewManager(cfg)

	// Create a test tap directory structure
	tapPath := filepath.Join(tmpDir, "Library", "Taps", "testuser", "homebrew-testrepo")
	formulaDir := filepath.Join(tapPath, "Formula")
	err := os.MkdirAll(formulaDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create tap directory: %v", err)
	}

	// Create some test formulae
	testFormulae := []string{"formula1.rb", "formula2.rb"}
	for _, formula := range testFormulae {
		filePath := filepath.Join(formulaDir, formula)
		err := os.WriteFile(filePath, []byte("# Test formula"), 0644)
		if err != nil {
			t.Fatalf("Failed to create test formula: %v", err)
		}
	}

	tap, err := manager.loadTap(tapPath)
	if err != nil {
		t.Fatalf("loadTap() error = %v", err)
	}

	expectedName := "testuser/testrepo"
	if tap.Name != expectedName {
		t.Errorf("Tap name = %v, want %v", tap.Name, expectedName)
	}

	if tap.User != "testuser" {
		t.Errorf("Tap user = %v, want testuser", tap.User)
	}

	if tap.Repository != "testrepo" {
		t.Errorf("Tap repository = %v, want testrepo", tap.Repository)
	}

	if !tap.Installed {
		t.Error("Loaded tap should be marked as installed")
	}

	if tap.Formulae != 2 {
		t.Errorf("Tap formulae count = %v, want 2", tap.Formulae)
	}

	if tap.Official {
		t.Error("Test user tap should not be marked as official")
	}
}

func TestVerifyTap(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{}
	manager := NewManager(cfg)

	// Test empty directory (should fail)
	emptyDir := filepath.Join(tmpDir, "empty")
	err := os.MkdirAll(emptyDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create empty directory: %v", err)
	}

	err = manager.verifyTap(emptyDir)
	if err == nil {
		t.Error("verifyTap() should fail for empty directory")
	}

	// Test directory with Formula (should pass)
	formulaDir := filepath.Join(tmpDir, "with-formula")
	err = os.MkdirAll(filepath.Join(formulaDir, "Formula"), 0755)
	if err != nil {
		t.Fatalf("Failed to create formula directory: %v", err)
	}

	err = manager.verifyTap(formulaDir)
	if err != nil {
		t.Errorf("verifyTap() should pass for directory with Formula: %v", err)
	}

	// Test directory with Casks (should pass)
	casksDir := filepath.Join(tmpDir, "with-casks")
	err = os.MkdirAll(filepath.Join(casksDir, "Casks"), 0755)
	if err != nil {
		t.Fatalf("Failed to create casks directory: %v", err)
	}

	err = manager.verifyTap(casksDir)
	if err != nil {
		t.Errorf("verifyTap() should pass for directory with Casks: %v", err)
	}
}

func TestTapOptions(t *testing.T) {
	opts := &TapOptions{
		Force:   true,
		Quiet:   false,
		Shallow: true,
		Branch:  "main",
	}

	if !opts.Force {
		t.Error("Force option should be true")
	}

	if opts.Quiet {
		t.Error("Quiet option should be false")
	}

	if !opts.Shallow {
		t.Error("Shallow option should be true")
	}

	if opts.Branch != "main" {
		t.Errorf("Branch option = %v, want main", opts.Branch)
	}
}

func TestTapStruct(t *testing.T) {
	tap := &Tap{
		Name:        "user/repo",
		FullName:    "homebrew/repo",
		User:        "user",
		Repository:  "repo",
		Remote:      "https://github.com/user/homebrew-repo.git",
		Path:        "/path/to/tap",
		Installed:   true,
		Official:    false,
		Formulae:    10,
		Casks:       5,
	}

	if tap.Name != "user/repo" {
		t.Errorf("Name = %v, want user/repo", tap.Name)
	}

	if !tap.Installed {
		t.Error("Installed should be true")
	}

	if tap.Official {
		t.Error("Official should be false for user tap")
	}

	if tap.Formulae != 10 {
		t.Errorf("Formulae count = %v, want 10", tap.Formulae)
	}

	if tap.Casks != 5 {
		t.Errorf("Casks count = %v, want 5", tap.Casks)
	}
}

func TestTapListFormulaeOriginal(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a test tap
	tap := &Tap{
		Name: "test/tap",
		Path: tmpDir,
	}

	// Create Formula directory with test formulae
	formulaDir := filepath.Join(tmpDir, "Formula")
	err := os.MkdirAll(formulaDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create formula directory: %v", err)
	}

	// Create test formula files
	testFormulae := []string{"wget.rb", "curl.rb", "python.yaml", "not-a-formula.txt"}
	for _, formula := range testFormulae {
		filePath := filepath.Join(formulaDir, formula)
		err := os.WriteFile(filePath, []byte("# Test formula"), 0644)
		if err != nil {
			t.Fatalf("Failed to create test formula %s: %v", formula, err)
		}
	}

	formulae, err := tap.ListFormulae()
	if err != nil {
		t.Fatalf("ListFormulae() error = %v", err)
	}

	expectedCount := 3 // .rb and .yaml files
	if len(formulae) != expectedCount {
		t.Errorf("ListFormulae() count = %v, want %v", len(formulae), expectedCount)
	}

	// Check that formulae are sorted
	expectedFormulae := []string{"curl", "python", "wget"}
	for i, expected := range expectedFormulae {
		if i < len(formulae) && formulae[i] != expected {
			t.Errorf("Formula[%d] = %v, want %v", i, formulae[i], expected)
		}
	}

	// Verify that .txt file is not included
	for _, formula := range formulae {
		if strings.Contains(formula, "not-a-formula") {
			t.Error("Non-formula files should not be included")
		}
	}
}