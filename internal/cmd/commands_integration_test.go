package cmd

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pilshchikov/homebrew-go/internal/config"
	"github.com/pilshchikov/homebrew-go/internal/logger"
)

func TestRunCleanup(t *testing.T) {
	// Initialize logger for tests
	logger.Init(false, false, true)

	// Create temporary directories for testing
	tempDir, err := os.MkdirTemp("", "brew-test-cleanup")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	cfg := &config.Config{
		HomebrewCache:  filepath.Join(tempDir, "cache"),
		HomebrewCellar: filepath.Join(tempDir, "cellar"),
		HomebrewPrefix: tempDir,
	}

	// Create test directories
	_ = os.MkdirAll(cfg.HomebrewCache, 0755)
	_ = os.MkdirAll(cfg.HomebrewCellar, 0755)

	// Test dry run
	err = runCleanup(cfg, true)
	if err != nil {
		t.Errorf("runCleanup dry run failed: %v", err)
	}

	// Test actual cleanup
	err = runCleanup(cfg, false)
	if err != nil {
		t.Errorf("runCleanup failed: %v", err)
	}
}

func TestPrintColumns(t *testing.T) {
	tests := []struct {
		name     string
		items    []string
		columns  int
		expected string
	}{
		{
			name:     "empty list",
			items:    []string{},
			columns:  4,
			expected: "",
		},
		{
			name:     "single item",
			items:    []string{"item1"},
			columns:  4,
			expected: "item1",
		},
		{
			name:     "multiple items",
			items:    []string{"item1", "item2", "item3"},
			columns:  2,
			expected: "item1  item2  \nitem3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			// Capture stdout
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			printColumns(tt.items, tt.columns)

			_ = w.Close()
			os.Stdout = oldStdout

			_, _ = buf.ReadFrom(r)
			output := strings.TrimSpace(buf.String())

			if tt.expected == "" && output != "" {
				t.Errorf("Expected empty output, got %q", output)
			} else if tt.expected != "" && !strings.Contains(output, strings.Split(tt.expected, "\n")[0]) {
				t.Errorf("Expected output containing %q, got %q", tt.expected, output)
			}
		})
	}
}

func TestFormatFileSize(t *testing.T) {
	tests := []struct {
		size     int64
		expected string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1073741824, "1.0 GB"},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("size_%d", tt.size), func(t *testing.T) {
			result := formatFileSize(tt.size)
			if result != tt.expected {
				t.Errorf("formatFileSize(%d) = %s, want %s", tt.size, result, tt.expected)
			}
		})
	}
}

func TestCountDirItems(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "brew-test-count")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Test empty directory
	count := countDirItems(tempDir)
	if count != 0 {
		t.Errorf("Expected 0 items in empty directory, got %d", count)
	}

	// Create some files
	for i := 0; i < 3; i++ {
		filename := filepath.Join(tempDir, fmt.Sprintf("file%d.txt", i))
		err = os.WriteFile(filename, []byte("test"), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	count = countDirItems(tempDir)
	if count != 3 {
		t.Errorf("Expected 3 items in directory, got %d", count)
	}

	// Test non-existent directory
	count = countDirItems("/nonexistent/path")
	if count != 0 {
		t.Errorf("Expected 0 items for non-existent directory, got %d", count)
	}
}

func TestShowConfigIntegration(t *testing.T) {
	cfg := &config.Config{
		HomebrewPrefix:     "/test/prefix",
		HomebrewRepository: "/test/repository",
		HomebrewLibrary:    "/test/library",
		HomebrewCellar:     "/test/cellar",
		HomebrewCaskroom:   "/test/caskroom",
		HomebrewCache:      "/test/cache",
		HomebrewLogs:       "/test/logs",
		HomebrewTemp:       "/test/temp",
		Debug:              true,
		Verbose:            false,
		AutoUpdate:         true,
		InstallCleanup:     false,
	}

	err := showConfig(cfg)
	if err != nil {
		t.Errorf("showConfig failed: %v", err)
	}
}

func TestShowEnvIntegration(t *testing.T) {
	cfg := &config.Config{
		HomebrewPrefix:     "/test/prefix",
		HomebrewRepository: "/test/repository",
		HomebrewCellar:     "/test/cellar",
		HomebrewCaskroom:   "/test/caskroom",
		HomebrewCache:      "/test/cache",
		HomebrewLogs:       "/test/logs",
		HomebrewTemp:       "/test/temp",
	}

	// Test regular output
	err := showEnv(cfg, false)
	if err != nil {
		t.Errorf("showEnv failed: %v", err)
	}

	// Test JSON output
	err = showEnv(cfg, true)
	if err != nil {
		t.Errorf("showEnv JSON failed: %v", err)
	}
}

func TestListInstalled(t *testing.T) {
	// Initialize logger for tests
	logger.Init(false, false, true)

	tempDir, err := os.MkdirTemp("", "brew-test-list")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	cfg := &config.Config{
		HomebrewCellar:   filepath.Join(tempDir, "cellar"),
		HomebrewCaskroom: filepath.Join(tempDir, "caskroom"),
	}

	// Create test directories
	_ = os.MkdirAll(cfg.HomebrewCellar, 0755)
	_ = os.MkdirAll(cfg.HomebrewCaskroom, 0755)

	// Create some test formulae
	formulaDir := filepath.Join(cfg.HomebrewCellar, "test-formula")
	_ = os.MkdirAll(filepath.Join(formulaDir, "1.0.0"), 0755)

	// Create some test casks
	caskDir := filepath.Join(cfg.HomebrewCaskroom, "test-cask")
	_ = os.MkdirAll(caskDir, 0755)

	// Test listing all
	err = listInstalled(cfg, false, false, false, false)
	if err != nil {
		t.Errorf("listInstalled failed: %v", err)
	}

	// Test listing only formulae
	err = listInstalled(cfg, false, true, false, false)
	if err != nil {
		t.Errorf("listInstalled formulae only failed: %v", err)
	}

	// Test listing only casks
	err = listInstalled(cfg, true, false, false, false)
	if err != nil {
		t.Errorf("listInstalled casks only failed: %v", err)
	}

	// Test with versions
	err = listInstalled(cfg, false, false, true, false)
	if err != nil {
		t.Errorf("listInstalled with versions failed: %v", err)
	}

	// Test with full names
	err = listInstalled(cfg, false, false, false, true)
	if err != nil {
		t.Errorf("listInstalled with full names failed: %v", err)
	}
}

func TestListFormulaFiles(t *testing.T) {
	// Initialize logger for tests
	logger.Init(false, false, true)

	tempDir, err := os.MkdirTemp("", "brew-test-files")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	cfg := &config.Config{
		HomebrewCellar: tempDir,
	}

	// Test non-existent formula
	err = listFormulaFiles(cfg, "nonexistent")
	if err == nil {
		t.Error("Expected error for non-existent formula")
	}

	// Create test formula structure
	formulaDir := filepath.Join(tempDir, "test-formula", "1.0.0")
	_ = os.MkdirAll(filepath.Join(formulaDir, "bin"), 0755)
	_ = os.WriteFile(filepath.Join(formulaDir, "bin", "test-binary"), []byte("test"), 0755)
	_ = os.WriteFile(filepath.Join(formulaDir, "README.txt"), []byte("readme"), 0644)

	// Test listing files
	err = listFormulaFiles(cfg, "test-formula")
	if err != nil {
		t.Errorf("listFormulaFiles failed: %v", err)
	}

	// Test formula with no versions
	emptyFormulaDir := filepath.Join(tempDir, "empty-formula")
	_ = os.MkdirAll(emptyFormulaDir, 0755)

	err = listFormulaFiles(cfg, "empty-formula")
	if err == nil {
		t.Error("Expected error for formula with no versions")
	}
}

func TestDirSize(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "brew-test-dirsize")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Test empty directory
	size, err := dirSize(tempDir)
	if err != nil {
		t.Errorf("dirSize failed for empty directory: %v", err)
	}
	if size != 0 {
		t.Errorf("Expected 0 size for empty directory, got %d", size)
	}

	// Create some test files
	testFile := filepath.Join(tempDir, "test.txt")
	content := "Hello, World!"
	err = os.WriteFile(testFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	size, err = dirSize(tempDir)
	if err != nil {
		t.Errorf("dirSize failed: %v", err)
	}

	expectedSize := int64(len(content))
	if size != expectedSize {
		t.Errorf("Expected size %d, got %d", expectedSize, size)
	}

	// Test non-existent directory
	_, err = dirSize("/nonexistent/path")
	if err == nil {
		t.Error("Expected error for non-existent directory")
	}
}

func TestCleanupCache(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "brew-test-cache")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Test non-existent cache directory
	size, count, err := cleanupCache("/nonexistent", false)
	if err != nil {
		t.Errorf("cleanupCache should handle non-existent directory gracefully: %v", err)
	}
	if size != 0 || count != 0 {
		t.Errorf("Expected 0 size and count for non-existent directory, got %d, %d", size, count)
	}

	// Create cache directory with some files
	_ = os.MkdirAll(tempDir, 0755)

	// Create a recent file (should not be cleaned)
	recentFile := filepath.Join(tempDir, "recent.txt")
	_ = os.WriteFile(recentFile, []byte("recent"), 0644)

	// Test cleanup (should not remove recent files)
	size, count, err = cleanupCache(tempDir, false)
	if err != nil {
		t.Errorf("cleanupCache failed: %v", err)
	}

	// Should be no cleanup for recent files
	if size != 0 || count != 0 {
		t.Errorf("Expected no cleanup for recent files, got size=%d, count=%d", size, count)
	}
}

func TestCleanupCellar(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "brew-test-cellar")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Test non-existent cellar directory
	size, count, err := cleanupCellar("/nonexistent", false)
	if err != nil {
		t.Errorf("cleanupCellar should handle non-existent directory gracefully: %v", err)
	}
	if size != 0 || count != 0 {
		t.Errorf("Expected 0 size and count for non-existent directory, got %d, %d", size, count)
	}

	// Create cellar directory structure
	_ = os.MkdirAll(tempDir, 0755)

	formulaDir := filepath.Join(tempDir, "test-formula")
	_ = os.MkdirAll(formulaDir, 0755)

	// Create multiple versions (only 2 should be kept)
	for i := 1; i <= 4; i++ {
		versionDir := filepath.Join(formulaDir, fmt.Sprintf("1.%d.0", i))
		_ = os.MkdirAll(versionDir, 0755)
		_ = os.WriteFile(filepath.Join(versionDir, "test.txt"), []byte("test"), 0644)
	}

	// Test cleanup (should remove old versions)
	_, count, err = cleanupCellar(tempDir, false)
	if err != nil {
		t.Errorf("cleanupCellar failed: %v", err)
	}

	// Should cleanup 2 old versions (keeping latest 2)
	if count != 2 {
		t.Errorf("Expected cleanup of 2 versions, got count=%d", count)
	}
}

func TestCleanupLockFiles(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "brew-test-locks")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	cfg := &config.Config{
		HomebrewPrefix: tempDir,
		HomebrewCache:  tempDir,
	}

	// Create some lock files
	lockFile := filepath.Join(tempDir, "test.lock")
	_ = os.WriteFile(lockFile, []byte("lock"), 0644)

	tmpFile := filepath.Join(tempDir, "test.tmp")
	_ = os.WriteFile(tmpFile, []byte("tmp"), 0644)

	// Test cleanup (recent files should not be removed)
	count, err := cleanupLockFiles(cfg, false)
	if err != nil {
		t.Errorf("cleanupLockFiles failed: %v", err)
	}

	// Files are recent, so should not be cleaned
	if count != 0 {
		t.Errorf("Expected no cleanup for recent lock files, got count=%d", count)
	}
}
