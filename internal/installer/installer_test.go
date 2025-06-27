package installer

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/pilshchikov/homebrew-go/internal/config"
	"github.com/pilshchikov/homebrew-go/internal/formula"
	"github.com/pilshchikov/homebrew-go/internal/logger"
)

func TestNew(t *testing.T) {
	cfg := &config.Config{}
	opts := &Options{
		BuildFromSource: true,
		Verbose:         true,
	}

	installer := New(cfg, opts)

	if installer.cfg != cfg {
		t.Error("Installer should store config reference")
	}

	if installer.opts != opts {
		t.Error("Installer should store options reference")
	}
}

func TestGetPlatform(t *testing.T) {
	platform := getPlatform()

	switch runtime.GOOS {
	case "darwin":
		if !strings.Contains(platform, "monterey") {
			t.Errorf("Expected macOS platform to contain 'monterey', got %s", platform)
		}
	case "linux":
		if platform != "x86_64_linux" {
			t.Errorf("Expected Linux platform to be 'x86_64_linux', got %s", platform)
		}
	default:
		if platform != "unknown" {
			t.Errorf("Expected unknown platform for %s, got %s", runtime.GOOS, platform)
		}
	}
}

func TestShouldUseBottle(t *testing.T) {
	cfg := &config.Config{}
	installer := New(cfg, &Options{})

	// Get the actual platform tag that the API client would use
	platformTag := installer.apiClient.GetPlatformTag()

	tests := []struct {
		name     string
		formula  *formula.Formula
		opts     *Options
		expected bool
	}{
		{
			name: "build from source",
			formula: &formula.Formula{
				Name:    "test",
				Version: "1.0.0",
				Bottle: &formula.Bottle{
					Stable: &formula.BottleSpec{
						Files: map[string]formula.BottleFile{
							platformTag: {URL: "test.tar.gz", SHA256: "abc123"},
						},
					},
				},
			},
			opts:     &Options{BuildFromSource: true},
			expected: false,
		},
		{
			name: "force bottle",
			formula: &formula.Formula{
				Name:    "test",
				Version: "1.0.0",
				Bottle: &formula.Bottle{
					Stable: &formula.BottleSpec{
						Files: map[string]formula.BottleFile{
							platformTag: {URL: "test.tar.gz", SHA256: "abc123"},
						},
					},
				},
			},
			opts:     &Options{BuildFromSource: true, ForceBottle: true},
			expected: true,
		},
		{
			name: "has bottle",
			formula: &formula.Formula{
				Name:    "test",
				Version: "1.0.0",
				Bottle: &formula.Bottle{
					Stable: &formula.BottleSpec{
						Files: map[string]formula.BottleFile{
							platformTag: {URL: "test.tar.gz", SHA256: "abc123"},
						},
					},
				},
			},
			opts:     &Options{},
			expected: true,
		},
		{
			name: "no bottle",
			formula: &formula.Formula{
				Name:    "test",
				Version: "1.0.0",
			},
			opts:     &Options{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			installer.opts = tt.opts
			result := installer.shouldUseBottle(tt.formula)
			if result != tt.expected {
				t.Errorf("shouldUseBottle() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestVerifyChecksum(t *testing.T) {
	// Initialize logger for tests
	logger.Init(false, false, true) // quiet mode

	cfg := &config.Config{}
	installer := New(cfg, &Options{})

	// Create a temporary file with known content
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testContent := "Hello, World!"

	err := os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Calculate expected SHA256
	hasher := sha256.New()
	hasher.Write([]byte(testContent))
	expectedSHA := hex.EncodeToString(hasher.Sum(nil))

	// Test correct checksum using new verification system
	err = installer.verifier.VerifySource(testFile, expectedSHA, 0)
	if err != nil {
		t.Errorf("VerifySource() with correct checksum failed: %v", err)
	}

	// Test incorrect checksum
	err = installer.verifier.VerifySource(testFile, "incorrect_checksum", 0)
	if err == nil {
		t.Error("VerifySource() with incorrect checksum should fail")
	}

	// Test non-existent file
	err = installer.verifier.VerifySource("/non/existent/file", expectedSHA, 0)
	if err == nil {
		t.Error("VerifySource() with non-existent file should fail")
	}
}

func TestWriteInstallReceipt(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		HomebrewCellar: tmpDir,
	}
	installer := New(cfg, &Options{CC: "gcc"})

	testFormula := &formula.Formula{
		Name:              "test-formula",
		Version:           "1.0.0",
		Dependencies:      []string{"dep1", "dep2"},
		BuildDependencies: []string{"build-dep1"},
	}

	err := installer.writeInstallReceipt(testFormula, "bottle")
	if err != nil {
		t.Fatalf("writeInstallReceipt() failed: %v", err)
	}

	// Verify receipt file was created
	receiptPath := testFormula.GetInstallReceipt(cfg.HomebrewCellar)
	if _, err := os.Stat(receiptPath); os.IsNotExist(err) {
		t.Error("Install receipt file was not created")
	}

	// Read and verify receipt content
	content, err := os.ReadFile(receiptPath)
	if err != nil {
		t.Fatalf("Failed to read receipt file: %v", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "test-formula") {
		t.Error("Receipt should contain formula name")
	}
	if !strings.Contains(contentStr, "1.0.0") {
		t.Error("Receipt should contain version")
	}
	if !strings.Contains(contentStr, "bottle") {
		t.Error("Receipt should contain source")
	}
	if !strings.Contains(contentStr, "gcc") {
		t.Error("Receipt should contain compiler")
	}
}

func TestIsFormulaInstalled(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		HomebrewCellar: tmpDir,
	}
	installer := New(cfg, &Options{})

	// Test non-existent formula
	installed, err := installer.isFormulaInstalled("non-existent")
	if err != nil {
		t.Errorf("isFormulaInstalled() error = %v", err)
	}
	if installed {
		t.Error("Non-existent formula should not be installed")
	}

	// Create a formula directory
	formulaDir := filepath.Join(tmpDir, "test-formula")
	err = os.MkdirAll(formulaDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create formula directory: %v", err)
	}

	// Test existing formula
	installed, err = installer.isFormulaInstalled("test-formula")
	if err != nil {
		t.Errorf("isFormulaInstalled() error = %v", err)
	}
	if !installed {
		t.Error("Existing formula should be installed")
	}
}

func TestInstallResult(t *testing.T) {
	result := &InstallResult{
		Name:     "test-formula",
		Version:  "1.0.0",
		Duration: time.Second,
		Source:   "bottle",
		Success:  true,
		Error:    nil,
	}

	if result.Name != "test-formula" {
		t.Errorf("Name = %v, want test-formula", result.Name)
	}

	if result.Success != true {
		t.Error("Success should be true")
	}

	if result.Error != nil {
		t.Error("Error should be nil for successful install")
	}
}

func TestInstallReceipt(t *testing.T) {
	receipt := &InstallReceipt{
		Name:              "test-formula",
		Version:           "1.0.0",
		InstalledOn:       time.Now(),
		InstalledBy:       "brew-go",
		Source:            "bottle",
		Dependencies:      []string{"dep1", "dep2"},
		BuildDependencies: []string{"build-dep1"},
		Platform:          getPlatform(),
	}

	if receipt.Name != "test-formula" {
		t.Errorf("Name = %v, want test-formula", receipt.Name)
	}

	if len(receipt.Dependencies) != 2 {
		t.Errorf("Dependencies count = %v, want 2", len(receipt.Dependencies))
	}

	if len(receipt.BuildDependencies) != 1 {
		t.Errorf("BuildDependencies count = %v, want 1", len(receipt.BuildDependencies))
	}
}

func TestOptionsValidation(t *testing.T) {
	tests := []struct {
		name string
		opts *Options
	}{
		{
			name: "default options",
			opts: &Options{},
		},
		{
			name: "build from source",
			opts: &Options{
				BuildFromSource: true,
				KeepTmp:         true,
				DebugSymbols:    true,
			},
		},
		{
			name: "force bottle",
			opts: &Options{
				ForceBottle: true,
				Verbose:     true,
			},
		},
		{
			name: "dry run",
			opts: &Options{
				DryRun:  true,
				Verbose: true,
			},
		},
	}

	cfg := &config.Config{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			installer := New(cfg, tt.opts)
			if installer == nil {
				t.Error("New() should not return nil")
				return
			}
			if installer.opts != tt.opts {
				t.Error("Options not properly stored")
			}
		})
	}
}


func TestExtractTarGz(t *testing.T) {
	// This is a basic test - in a real implementation,
	// we'd create a real tar.gz file and test extraction
	cfg := &config.Config{}
	installer := New(cfg, &Options{})

	tmpDir := t.TempDir()
	nonExistentTar := filepath.Join(tmpDir, "non-existent.tar.gz")
	destDir := filepath.Join(tmpDir, "dest")

	err := installer.extractTarGz(nonExistentTar, destDir)
	if err == nil {
		t.Error("extractTarGz() should fail with non-existent file")
	}
}

func TestProgressReader(t *testing.T) {
	content := "Hello, World! This is test content for progress reader."
	reader := strings.NewReader(content)

	progressReader := &progressReader{
		reader:   reader,
		total:    int64(len(content)),
		filename: "test-file.txt",
	}

	// Read in chunks to test progress updates
	buffer := make([]byte, 10)
	totalRead := 0

	for {
		n, err := progressReader.Read(buffer)
		totalRead += n

		if err == io.EOF {
			break
		}

		if err != nil {
			t.Fatalf("progressReader.Read() failed: %v", err)
		}

		// Verify progress tracking
		if progressReader.current != int64(totalRead) {
			t.Errorf("progressReader.current = %d, want %d", progressReader.current, totalRead)
		}
	}

	// Verify total was read correctly
	if totalRead != len(content) {
		t.Errorf("Total read = %d, want %d", totalRead, len(content))
	}

	if progressReader.current != int64(len(content)) {
		t.Errorf("Final progress = %d, want %d", progressReader.current, len(content))
	}
}

func TestDownloadFileWithProgress(t *testing.T) {
	// Initialize logger for tests
	logger.Init(false, false, true) // quiet mode for tests

	// This test requires a mock HTTP server for full testing
	// For now, test the error cases

	cfg := &config.Config{}
	installer := New(cfg, &Options{})

	tmpDir := t.TempDir()
	destPath := filepath.Join(tmpDir, "downloaded-file")

	// Test with invalid URL
	err := installer.downloadFile("invalid://url", destPath)
	if err == nil {
		t.Error("downloadFile() should fail with invalid URL")
	}

	// Verify it returns the enhanced error type
	if !strings.Contains(err.Error(), "download") {
		t.Errorf("downloadFile() should return enhanced error, got: %v", err)
	}
}

func TestFindSourceDirectory(t *testing.T) {
	cfg := &config.Config{}
	installer := New(cfg, &Options{})

	tmpDir := t.TempDir()

	tests := []struct {
		name        string
		setupFunc   func() string
		expectError bool
		expectPath  string
	}{
		{
			name: "single directory with configure",
			setupFunc: func() string {
				subDir := filepath.Join(tmpDir, "test-1.0.0")
				_ = os.MkdirAll(subDir, 0755)
				configFile := filepath.Join(subDir, "configure")
				_ = os.WriteFile(configFile, []byte("#!/bin/bash\n"), 0755)
				return tmpDir
			},
			expectError: false,
			expectPath:  "test-1.0.0",
		},
		{
			name: "configure in root",
			setupFunc: func() string {
				extractDir := filepath.Join(tmpDir, "direct")
				_ = os.MkdirAll(extractDir, 0755)
				configFile := filepath.Join(extractDir, "configure")
				_ = os.WriteFile(configFile, []byte("#!/bin/bash\n"), 0755)
				return extractDir
			},
			expectError: false,
		},
		{
			name: "directory with Makefile",
			setupFunc: func() string {
				extractDir := filepath.Join(tmpDir, "makefile-test")
				subDir := filepath.Join(extractDir, "source")
				_ = os.MkdirAll(subDir, 0755)
				makeFile := filepath.Join(subDir, "Makefile")
				_ = os.WriteFile(makeFile, []byte("all:\n\techo 'building'\n"), 0644)
				return extractDir
			},
			expectError: false,
			expectPath:  "source",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			extractDir := tt.setupFunc()

			result, err := installer.findSourceDirectory(extractDir)

			if tt.expectError && err == nil {
				t.Error("findSourceDirectory() should have failed")
			}

			if !tt.expectError && err != nil {
				t.Errorf("findSourceDirectory() failed: %v", err)
			}

			if tt.expectPath != "" && !strings.HasSuffix(result, tt.expectPath) {
				t.Errorf("findSourceDirectory() = %s, should end with %s", result, tt.expectPath)
			}
		})
	}
}

func TestEnhancedErrorHandling(t *testing.T) {
	// Initialize logger for tests
	logger.Init(false, false, true) // quiet mode

	cfg := &config.Config{}
	installer := New(cfg, &Options{})

	// Test enhanced checksum verification
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "hello-1.0.0.tar.gz")
	_ = os.WriteFile(testFile, []byte("test content"), 0644)

	err := installer.verifier.VerifySource(testFile, "wrong_checksum", 0)
	if err == nil {
		t.Error("VerifySource() should fail with wrong checksum")
	}

	// Verify it's enhanced error with context
	errStr := err.Error()
	if !strings.Contains(errStr, "checksum") {
		t.Errorf("Enhanced checksum error should contain 'checksum', got: %v", err)
	}
}

func TestBuildAndInstallErrorHandling(t *testing.T) {
	cfg := &config.Config{
		HomebrewCellar: t.TempDir(),
	}
	installer := New(cfg, &Options{})

	// Create a test formula
	formula := &formula.Formula{
		Name:    "test-formula",
		Version: "1.0.0",
	}

	// Test with non-existent source directory
	nonExistentDir := "/non/existent/directory"
	cellarPath := filepath.Join(cfg.HomebrewCellar, "test-formula", "1.0.0")

	err := installer.buildAndInstall(formula, nonExistentDir, cellarPath)
	if err == nil {
		t.Error("buildAndInstall() should fail with non-existent source directory")
	}

	// Test with directory without build system
	tmpDir := t.TempDir()
	emptySourceDir := filepath.Join(tmpDir, "empty-source")
	_ = os.MkdirAll(emptySourceDir, 0755)

	err = installer.buildAndInstall(formula, emptySourceDir, cellarPath)
	if err == nil {
		t.Error("buildAndInstall() should fail with no build system")
	}

	// Should return enhanced error
	if !strings.Contains(err.Error(), "build") {
		t.Errorf("buildAndInstall() should return enhanced build error, got: %v", err)
	}
}

func TestInstallDependenciesProgress(t *testing.T) {
	cfg := &config.Config{
		HomebrewCellar: t.TempDir(),
	}
	installer := New(cfg, &Options{})

	// Create a formula with dependencies
	formula := &formula.Formula{
		Name:         "main-formula",
		Version:      "1.0.0",
		Dependencies: []string{"dep1", "dep2"},
	}

	// This test would need more complex mocking to fully test
	// For now, test the basic structure
	err := installer.installDependencies(formula)

	// Should fail because dependencies don't exist, but should return enhanced error
	if err == nil {
		t.Error("installDependencies() should fail with non-existent dependencies")
	}

	// Verify it returns structured error
	if !strings.Contains(err.Error(), "dependency") {
		t.Errorf("installDependencies() should return dependency error, got: %v", err)
	}
}

func TestDetectBuildSystem(t *testing.T) {
	cfg := &config.Config{
		HomebrewCellar: t.TempDir(),
	}
	installer := New(cfg, &Options{})

	tests := []struct {
		name             string
		setupFunc        func(dir string)
		expectedSystem   string
		expectError      bool
		expectedCommands int
	}{
		{
			name: "autotools with configure",
			setupFunc: func(dir string) {
				_ = os.WriteFile(filepath.Join(dir, "configure"), []byte("#!/bin/bash\n"), 0755)
			},
			expectedSystem:   "autotools",
			expectedCommands: 3,
		},
		{
			name: "cmake",
			setupFunc: func(dir string) {
				_ = os.WriteFile(filepath.Join(dir, "CMakeLists.txt"), []byte("cmake_minimum_required(VERSION 3.0)\n"), 0644)
			},
			expectedSystem:   "cmake",
			expectedCommands: 3,
		},
		{
			name: "meson",
			setupFunc: func(dir string) {
				_ = os.WriteFile(filepath.Join(dir, "meson.build"), []byte("project('test')\n"), 0644)
			},
			expectedSystem:   "meson",
			expectedCommands: 3,
		},
		{
			name: "python setuptools",
			setupFunc: func(dir string) {
				_ = os.WriteFile(filepath.Join(dir, "setup.py"), []byte("from setuptools import setup\nsetup()\n"), 0644)
			},
			expectedSystem:   "python-setuptools",
			expectedCommands: 2,
		},
		{
			name: "python pip",
			setupFunc: func(dir string) {
				_ = os.WriteFile(filepath.Join(dir, "pyproject.toml"), []byte("[build-system]\nrequires = ['setuptools']\n"), 0644)
			},
			expectedSystem:   "python-pip",
			expectedCommands: 1,
		},
		{
			name: "rust cargo",
			setupFunc: func(dir string) {
				_ = os.WriteFile(filepath.Join(dir, "Cargo.toml"), []byte("[package]\nname = 'test'\nversion = '0.1.0'\n"), 0644)
			},
			expectedSystem:   "rust-cargo",
			expectedCommands: 2,
		},
		{
			name: "go modules",
			setupFunc: func(dir string) {
				_ = os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test\ngo 1.21\n"), 0644)
			},
			expectedSystem:   "go-modules",
			expectedCommands: 1,
		},
		{
			name: "node.js npm",
			setupFunc: func(dir string) {
				_ = os.WriteFile(filepath.Join(dir, "package.json"), []byte("{\"name\": \"test\", \"version\": \"1.0.0\"}\n"), 0644)
			},
			expectedSystem:   "npm",
			expectedCommands: 3,
		},
		{
			name: "ninja",
			setupFunc: func(dir string) {
				_ = os.WriteFile(filepath.Join(dir, "build.ninja"), []byte("rule compile\n  command = gcc $in -o $out\n"), 0644)
			},
			expectedSystem:   "ninja",
			expectedCommands: 2,
		},
		{
			name: "bazel",
			setupFunc: func(dir string) {
				_ = os.WriteFile(filepath.Join(dir, "WORKSPACE"), []byte("workspace(name = 'test')\n"), 0644)
				_ = os.WriteFile(filepath.Join(dir, "BUILD"), []byte("cc_binary(name = 'test', srcs = ['main.c'])\n"), 0644)
			},
			expectedSystem:   "bazel",
			expectedCommands: 2,
		},
		{
			name: "makefile",
			setupFunc: func(dir string) {
				_ = os.WriteFile(filepath.Join(dir, "Makefile"), []byte("all:\n\techo 'building'\n"), 0644)
			},
			expectedSystem:   "makefile",
			expectedCommands: 2,
		},
		{
			name: "autotools generate",
			setupFunc: func(dir string) {
				_ = os.WriteFile(filepath.Join(dir, "configure.ac"), []byte("AC_INIT([test], [1.0])\n"), 0644)
				_ = os.WriteFile(filepath.Join(dir, "Makefile.am"), []byte("SUBDIRS = src\n"), 0644)
			},
			expectedSystem:   "autotools-generate",
			expectedCommands: 4,
		},
		{
			name: "no build system",
			setupFunc: func(dir string) {
				_ = os.WriteFile(filepath.Join(dir, "README.txt"), []byte("This is a readme\n"), 0644)
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			cellarPath := filepath.Join(tmpDir, "cellar")

			tt.setupFunc(tmpDir)

			commands, buildSystem, err := installer.detectBuildSystem(tmpDir, cellarPath)

			if tt.expectError {
				if err == nil {
					t.Error("detectBuildSystem() should have failed for no build system")
				}
				return
			}

			if err != nil {
				t.Fatalf("detectBuildSystem() failed: %v", err)
			}

			if buildSystem != tt.expectedSystem {
				t.Errorf("detectBuildSystem() buildSystem = %s, want %s", buildSystem, tt.expectedSystem)
			}

			if len(commands) != tt.expectedCommands {
				t.Errorf("detectBuildSystem() commands count = %d, want %d", len(commands), tt.expectedCommands)
			}

			// Verify commands are properly structured
			for i, cmd := range commands {
				if len(cmd) == 0 {
					t.Errorf("detectBuildSystem() command %d is empty", i)
				}
			}
		})
	}
}

func TestGetBuildSystemSuggestions(t *testing.T) {
	cfg := &config.Config{}
	installer := New(cfg, &Options{})

	tests := []struct {
		name        string
		buildSystem string
		command     string
		expectCount int // minimum number of suggestions expected
	}{
		{
			name:        "autotools configure",
			buildSystem: "autotools",
			command:     "./configure",
			expectCount: 3,
		},
		{
			name:        "cmake setup",
			buildSystem: "cmake",
			command:     "cmake -S",
			expectCount: 3,
		},
		{
			name:        "meson setup",
			buildSystem: "meson",
			command:     "meson setup",
			expectCount: 3,
		},
		{
			name:        "rust cargo build",
			buildSystem: "rust-cargo",
			command:     "cargo build",
			expectCount: 3,
		},
		{
			name:        "go modules",
			buildSystem: "go-modules",
			command:     "go build",
			expectCount: 4,
		},
		{
			name:        "npm install",
			buildSystem: "npm",
			command:     "npm install",
			expectCount: 3,
		},
		{
			name:        "unknown build system",
			buildSystem: "unknown",
			command:     "unknown-command",
			expectCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			suggestions := installer.getBuildSystemSuggestions(tt.buildSystem, tt.command)

			if len(suggestions) < tt.expectCount {
				t.Errorf("getBuildSystemSuggestions() suggestions count = %d, want at least %d", len(suggestions), tt.expectCount)
			}

			// Verify suggestions are not empty
			for i, suggestion := range suggestions {
				if strings.TrimSpace(suggestion) == "" {
					t.Errorf("getBuildSystemSuggestions() suggestion %d is empty", i)
				}
			}
		})
	}
}

func TestAdvancedBuildSystemIntegration(t *testing.T) {
	cfg := &config.Config{
		HomebrewCellar: t.TempDir(),
	}
	installer := New(cfg, &Options{})

	// Test that build system detection integrates properly with error suggestions
	tmpDir := t.TempDir()
	cellarPath := filepath.Join(tmpDir, "cellar")

	// Create a CMake project
	_ = os.WriteFile(filepath.Join(tmpDir, "CMakeLists.txt"),
		[]byte("cmake_minimum_required(VERSION 3.0)\nproject(test)\n"), 0644)

	commands, buildSystem, err := installer.detectBuildSystem(tmpDir, cellarPath)
	if err != nil {
		t.Fatalf("detectBuildSystem() failed: %v", err)
	}

	if buildSystem != "cmake" {
		t.Errorf("detectBuildSystem() buildSystem = %s, want cmake", buildSystem)
	}

	// Test that we get appropriate suggestions for cmake commands
	for _, cmd := range commands {
		if len(cmd) > 0 {
			// Use the full command string to match the suggestions logic
			cmdStr := strings.Join(cmd, " ")
			suggestions := installer.getBuildSystemSuggestions(buildSystem, cmdStr)
			if len(suggestions) == 0 {
				t.Errorf("getBuildSystemSuggestions() should return suggestions for cmake command %s", cmdStr)
			}
		}
	}
}

func TestVerificationIntegration(t *testing.T) {
	// Initialize logger for tests
	logger.Init(false, false, true) // quiet mode

	cfg := &config.Config{
		HomebrewCellar: t.TempDir(),
	}
	installer := New(cfg, &Options{StrictVerification: true})

	// Test that installer has verifier
	if installer.verifier == nil {
		t.Error("Installer should have verifier initialized")
	}

	// Test bottle verification
	tmpDir := t.TempDir()
	bottleFile := filepath.Join(tmpDir, "test-1.0.0.arm64_sequoia.bottle.tar.gz")
	bottleContent := "fake bottle content"

	err := os.WriteFile(bottleFile, []byte(bottleContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test bottle file: %v", err)
	}

	// Calculate expected checksum
	hasher := sha256.New()
	hasher.Write([]byte(bottleContent))
	expectedSHA := hex.EncodeToString(hasher.Sum(nil))

	// Test bottle verification
	err = installer.verifier.VerifyBottle(bottleFile, expectedSHA, int64(len(bottleContent)))
	if err != nil {
		t.Errorf("VerifyBottle() failed: %v", err)
	}

	// Test source verification
	sourceFile := filepath.Join(tmpDir, "test-1.0.0.tar.gz")
	sourceContent := "fake source content"

	err = os.WriteFile(sourceFile, []byte(sourceContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test source file: %v", err)
	}

	// Calculate expected checksum
	hasher = sha256.New()
	hasher.Write([]byte(sourceContent))
	expectedSourceSHA := hex.EncodeToString(hasher.Sum(nil))

	err = installer.verifier.VerifySource(sourceFile, expectedSourceSHA, int64(len(sourceContent)))
	if err != nil {
		t.Errorf("VerifySource() failed: %v", err)
	}

	// Test installation verification
	installDir := filepath.Join(cfg.HomebrewCellar, "test-formula")
	err = os.MkdirAll(installDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create installation directory: %v", err)
	}

	binDir := filepath.Join(installDir, "bin")
	err = os.MkdirAll(binDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create bin directory: %v", err)
	}

	err = os.WriteFile(filepath.Join(binDir, "test-binary"), []byte("fake binary"), 0755)
	if err != nil {
		t.Fatalf("Failed to create test binary: %v", err)
	}

	result, err := installer.VerifyInstallation("test-formula")
	if err != nil {
		t.Errorf("VerifyInstallation() failed: %v", err)
	}

	if !result.IsVerificationSuccessful() {
		t.Errorf("Installation verification should succeed: %s", result.GetSummary())
	}
}

func TestVerificationOptions(t *testing.T) {
	cfg := &config.Config{}

	// Test strict verification option
	strictInstaller := New(cfg, &Options{StrictVerification: true})
	if strictInstaller.verifier == nil {
		t.Error("Strict installer should have verifier")
	}

	// Test non-strict verification option
	nonStrictInstaller := New(cfg, &Options{StrictVerification: false})
	if nonStrictInstaller.verifier == nil {
		t.Error("Non-strict installer should have verifier")
	}
}
