package verification

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pilshchikov/homebrew-go/internal/logger"
)

func TestNewVerifier(t *testing.T) {
	verifier := NewVerifier(true)
	if verifier == nil {
		t.Error("NewVerifier() should not return nil")
		return
	}

	if !verifier.strictMode {
		t.Error("NewVerifier(true) should create verifier in strict mode")
	}

	verifier = NewVerifier(false)
	if verifier.strictMode {
		t.Error("NewVerifier(false) should create verifier in non-strict mode")
	}
}

func TestVerifyFile(t *testing.T) {
	// Create a temporary test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testContent := "Hello, World!"

	err := os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Calculate expected checksum
	hasher := sha256.New()
	hasher.Write([]byte(testContent))
	expectedSHA256 := hex.EncodeToString(hasher.Sum(nil))

	verifier := NewVerifier(false)

	tests := []struct {
		name            string
		fileInfo        *FileInfo
		expectSuccess   bool
		expectExists    bool
		expectSizeMatch bool
	}{
		{
			name: "valid file with correct checksum",
			fileInfo: &FileInfo{
				Path:         testFile,
				ExpectedSize: int64(len(testContent)),
				Checksums: []Checksum{
					{Type: SHA256, Value: expectedSHA256},
				},
			},
			expectSuccess:   true,
			expectExists:    true,
			expectSizeMatch: true,
		},
		{
			name: "file with wrong checksum",
			fileInfo: &FileInfo{
				Path:         testFile,
				ExpectedSize: int64(len(testContent)),
				Checksums: []Checksum{
					{Type: SHA256, Value: "wrong_checksum"},
				},
			},
			expectSuccess:   false,
			expectExists:    true,
			expectSizeMatch: true,
		},
		{
			name: "file with wrong size",
			fileInfo: &FileInfo{
				Path:         testFile,
				ExpectedSize: 999,
				Checksums: []Checksum{
					{Type: SHA256, Value: expectedSHA256},
				},
			},
			expectSuccess:   true, // Non-strict mode doesn't fail on size mismatch
			expectExists:    true,
			expectSizeMatch: false,
		},
		{
			name: "non-existent file",
			fileInfo: &FileInfo{
				Path: filepath.Join(tmpDir, "nonexistent.txt"),
				Checksums: []Checksum{
					{Type: SHA256, Value: expectedSHA256},
				},
			},
			expectSuccess:   false,
			expectExists:    false,
			expectSizeMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := verifier.VerifyFile(tt.fileInfo)

			if result.FileExists != tt.expectExists {
				t.Errorf("FileExists = %v, want %v", result.FileExists, tt.expectExists)
			}

			if result.SizeMatches != tt.expectSizeMatch {
				t.Errorf("SizeMatches = %v, want %v", result.SizeMatches, tt.expectSizeMatch)
			}

			if result.IsVerificationSuccessful() != tt.expectSuccess {
				t.Errorf("IsVerificationSuccessful() = %v, want %v", result.IsVerificationSuccessful(), tt.expectSuccess)
			}
		})
	}
}

func TestVerifyFileStrictMode(t *testing.T) {
	// Create a temporary test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testContent := "Hello, World!"

	err := os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Calculate expected checksum
	hasher := sha256.New()
	hasher.Write([]byte(testContent))
	expectedSHA256 := hex.EncodeToString(hasher.Sum(nil))

	verifier := NewVerifier(true) // Strict mode

	// Test with wrong size in strict mode
	fileInfo := &FileInfo{
		Path:         testFile,
		ExpectedSize: 999, // Wrong size
		Checksums: []Checksum{
			{Type: SHA256, Value: expectedSHA256},
		},
	}

	result := verifier.VerifyFile(fileInfo)

	// In strict mode, size mismatch should cause errors
	if len(result.Errors) == 0 {
		t.Error("Expected errors in strict mode with size mismatch")
	}

	if result.SizeMatches {
		t.Error("SizeMatches should be false with wrong expected size")
	}
}

func TestGetHasher(t *testing.T) {
	verifier := NewVerifier(false)

	tests := []struct {
		checksumType ChecksumType
		expectError  bool
	}{
		{SHA256, false},
		{SHA512, false},
		{SHA1, false},
		{MD5, false},
		{ChecksumType("unsupported"), true},
	}

	for _, tt := range tests {
		t.Run(string(tt.checksumType), func(t *testing.T) {
			hasher, err := verifier.getHasher(tt.checksumType)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error for unsupported checksum type")
				}
				if hasher != nil {
					t.Error("Expected nil hasher for unsupported checksum type")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if hasher == nil {
					t.Error("Expected non-nil hasher for supported checksum type")
				}
			}
		})
	}
}

func TestComputeChecksum(t *testing.T) {
	// Create a test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testContent := "Hello, World!"

	err := os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	verifier := NewVerifier(false)

	// Test SHA256 computation
	checksum, err := verifier.ComputeChecksum(testFile, SHA256)
	if err != nil {
		t.Fatalf("ComputeChecksum() failed: %v", err)
	}

	// Verify the computed checksum matches manual calculation
	hasher := sha256.New()
	hasher.Write([]byte(testContent))
	expectedChecksum := hex.EncodeToString(hasher.Sum(nil))

	if checksum != expectedChecksum {
		t.Errorf("ComputeChecksum() = %s, want %s", checksum, expectedChecksum)
	}

	// Test with non-existent file
	_, err = verifier.ComputeChecksum("/nonexistent/file.txt", SHA256)
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}

func TestVerifyMultipleFiles(t *testing.T) {
	tmpDir := t.TempDir()
	verifier := NewVerifier(false)

	// Create multiple test files
	files := []*FileInfo{}
	for i := 0; i < 3; i++ {
		filename := filepath.Join(tmpDir, fmt.Sprintf("test%d.txt", i))
		content := fmt.Sprintf("Test file %d", i)

		err := os.WriteFile(filename, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file %d: %v", i, err)
		}

		// Calculate checksum
		hasher := sha256.New()
		hasher.Write([]byte(content))
		checksum := hex.EncodeToString(hasher.Sum(nil))

		files = append(files, &FileInfo{
			Path: filename,
			Checksums: []Checksum{
				{Type: SHA256, Value: checksum},
			},
		})
	}

	results := verifier.VerifyMultipleFiles(files)

	if len(results) != len(files) {
		t.Errorf("Expected %d results, got %d", len(files), len(results))
	}

	for i, result := range results {
		if !result.IsVerificationSuccessful() {
			t.Errorf("File %d verification failed: %s", i, result.GetSummary())
		}
	}
}

func TestVerificationResult_IsVerificationSuccessful(t *testing.T) {
	tests := []struct {
		name     string
		result   *VerificationResult
		expected bool
	}{
		{
			name: "successful verification",
			result: &VerificationResult{
				FileExists:      true,
				ChecksumsPassed: map[ChecksumType]bool{SHA256: true},
				SizeMatches:     true,
				Errors:          []error{},
			},
			expected: true,
		},
		{
			name: "file does not exist",
			result: &VerificationResult{
				FileExists:      false,
				ChecksumsPassed: map[ChecksumType]bool{SHA256: true},
				SizeMatches:     true,
				Errors:          []error{},
			},
			expected: false,
		},
		{
			name: "checksum failed",
			result: &VerificationResult{
				FileExists:      true,
				ChecksumsPassed: map[ChecksumType]bool{SHA256: false},
				SizeMatches:     true,
				Errors:          []error{},
			},
			expected: false,
		},
		{
			name: "has errors",
			result: &VerificationResult{
				FileExists:      true,
				ChecksumsPassed: map[ChecksumType]bool{SHA256: true},
				SizeMatches:     true,
				Errors:          []error{fmt.Errorf("test error")},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.result.IsVerificationSuccessful() != tt.expected {
				t.Errorf("IsVerificationSuccessful() = %v, want %v", tt.result.IsVerificationSuccessful(), tt.expected)
			}
		})
	}
}

func TestVerificationResult_GetSummary(t *testing.T) {
	tests := []struct {
		name     string
		result   *VerificationResult
		contains string
	}{
		{
			name: "successful verification",
			result: &VerificationResult{
				FileExists:      true,
				ChecksumsPassed: map[ChecksumType]bool{SHA256: true, SHA1: true},
				SizeMatches:     true,
				Errors:          []error{},
			},
			contains: "✓ Verification passed",
		},
		{
			name: "file does not exist",
			result: &VerificationResult{
				FileExists:      false,
				ChecksumsPassed: map[ChecksumType]bool{},
				SizeMatches:     true,
				Errors:          []error{},
			},
			contains: "file does not exist",
		},
		{
			name: "checksum failed",
			result: &VerificationResult{
				FileExists:      true,
				ChecksumsPassed: map[ChecksumType]bool{SHA256: false},
				SizeMatches:     true,
				Errors:          []error{},
			},
			contains: "sha256 checksum failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			summary := tt.result.GetSummary()
			if !strings.Contains(summary, tt.contains) {
				t.Errorf("GetSummary() = %q, should contain %q", summary, tt.contains)
			}
		})
	}
}

func TestPackageVerifier(t *testing.T) {
	// Initialize logger for tests
	logger.Init(false, false, true) // quiet mode

	tmpDir := t.TempDir()
	pv := NewPackageVerifier(false)

	// Create a test file
	testFile := filepath.Join(tmpDir, "test-bottle.tar.gz")
	testContent := "fake bottle content"

	err := os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Calculate expected checksum
	hasher := sha256.New()
	hasher.Write([]byte(testContent))
	expectedSHA256 := hex.EncodeToString(hasher.Sum(nil))

	// Test bottle verification
	err = pv.VerifyBottle(testFile, expectedSHA256, int64(len(testContent)))
	if err != nil {
		t.Errorf("VerifyBottle() failed: %v", err)
	}

	// Test with wrong checksum
	err = pv.VerifyBottle(testFile, "wrong_checksum", int64(len(testContent)))
	if err == nil {
		t.Error("VerifyBottle() should fail with wrong checksum")
	}

	// Test source verification
	err = pv.VerifySource(testFile, expectedSHA256, int64(len(testContent)))
	if err != nil {
		t.Errorf("VerifySource() failed: %v", err)
	}
}

func TestVerifyInstallation(t *testing.T) {
	// Initialize logger for tests
	logger.Init(false, false, true) // quiet mode

	tmpDir := t.TempDir()
	pv := NewPackageVerifier(false)

	// Create a fake installation directory
	installDir := filepath.Join(tmpDir, "installation")
	err := os.MkdirAll(installDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create installation directory: %v", err)
	}

	// Add some files to the installation
	binDir := filepath.Join(installDir, "bin")
	err = os.MkdirAll(binDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create bin directory: %v", err)
	}

	testBinary := filepath.Join(binDir, "test-binary")
	err = os.WriteFile(testBinary, []byte("fake binary"), 0755)
	if err != nil {
		t.Fatalf("Failed to create test binary: %v", err)
	}

	// Test installation verification
	result := pv.VerifyInstallation(installDir)
	if !result.FileExists {
		t.Error("Installation should exist")
	}

	if !result.IsVerificationSuccessful() {
		t.Errorf("Installation verification should succeed: %s", result.GetSummary())
	}

	// Test with non-existent installation
	result = pv.VerifyInstallation(filepath.Join(tmpDir, "nonexistent"))
	if result.FileExists {
		t.Error("Non-existent installation should not exist")
	}

	if result.IsVerificationSuccessful() {
		t.Error("Non-existent installation verification should fail")
	}
}
