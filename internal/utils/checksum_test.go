package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVerifySHA256(t *testing.T) {
	// Create a temporary file
	tempDir, err := os.MkdirTemp("", "utils-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	testFile := filepath.Join(tempDir, "test.txt")
	content := "Hello, World!"
	
	err = os.WriteFile(testFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Calculate expected SHA256
	hasher := sha256.New()
	hasher.Write([]byte(content))
	expectedSHA256 := hex.EncodeToString(hasher.Sum(nil))

	// Test with correct checksum
	err = VerifySHA256(testFile, expectedSHA256)
	if err != nil {
		t.Errorf("VerifySHA256 failed with correct checksum: %v", err)
	}

	// Test with incorrect checksum
	incorrectSHA256 := "incorrect_checksum"
	err = VerifySHA256(testFile, incorrectSHA256)
	if err == nil {
		t.Error("VerifySHA256 should fail with incorrect checksum")
	}

	if err != nil && !contains(err.Error(), "checksum mismatch") {
		t.Errorf("Expected 'checksum mismatch' error, got: %v", err)
	}

	// Test with non-existent file
	nonExistentFile := filepath.Join(tempDir, "nonexistent.txt")
	err = VerifySHA256(nonExistentFile, expectedSHA256)
	if err == nil {
		t.Error("VerifySHA256 should fail with non-existent file")
	}

	if err != nil && !contains(err.Error(), "failed to open file") {
		t.Errorf("Expected 'failed to open file' error, got: %v", err)
	}
}

func TestComputeSHA256(t *testing.T) {
	// Create a temporary file
	tempDir, err := os.MkdirTemp("", "utils-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	testFile := filepath.Join(tempDir, "test.txt")
	content := "Hello, World!"
	
	err = os.WriteFile(testFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Calculate expected SHA256
	hasher := sha256.New()
	hasher.Write([]byte(content))
	expectedSHA256 := hex.EncodeToString(hasher.Sum(nil))

	// Test ComputeSHA256
	actualSHA256, err := ComputeSHA256(testFile)
	if err != nil {
		t.Fatalf("ComputeSHA256 failed: %v", err)
	}

	if actualSHA256 != expectedSHA256 {
		t.Errorf("Expected SHA256 %s, got %s", expectedSHA256, actualSHA256)
	}

	// Test with non-existent file
	nonExistentFile := filepath.Join(tempDir, "nonexistent.txt")
	_, err = ComputeSHA256(nonExistentFile)
	if err == nil {
		t.Error("ComputeSHA256 should fail with non-existent file")
	}

	if err != nil && !contains(err.Error(), "failed to open file") {
		t.Errorf("Expected 'failed to open file' error, got: %v", err)
	}
}

func TestVerifySHA256EdgeCases(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "utils-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Test with empty file
	emptyFile := filepath.Join(tempDir, "empty.txt")
	err = os.WriteFile(emptyFile, []byte(""), 0644)
	if err != nil {
		t.Fatalf("Failed to write empty file: %v", err)
	}

	// SHA256 of empty string
	emptyHasher := sha256.New()
	emptySHA256 := hex.EncodeToString(emptyHasher.Sum(nil))

	err = VerifySHA256(emptyFile, emptySHA256)
	if err != nil {
		t.Errorf("VerifySHA256 failed with empty file: %v", err)
	}

	// Test with large file
	largeFile := filepath.Join(tempDir, "large.txt")
	largeContent := make([]byte, 1024*1024) // 1MB
	for i := range largeContent {
		largeContent[i] = byte(i % 256)
	}

	err = os.WriteFile(largeFile, largeContent, 0644)
	if err != nil {
		t.Fatalf("Failed to write large file: %v", err)
	}

	largeHasher := sha256.New()
	largeHasher.Write(largeContent)
	largeSHA256 := hex.EncodeToString(largeHasher.Sum(nil))

	err = VerifySHA256(largeFile, largeSHA256)
	if err != nil {
		t.Errorf("VerifySHA256 failed with large file: %v", err)
	}

	// Test ComputeSHA256 with large file
	computedSHA256, err := ComputeSHA256(largeFile)
	if err != nil {
		t.Fatalf("ComputeSHA256 failed with large file: %v", err)
	}

	if computedSHA256 != largeSHA256 {
		t.Errorf("ComputeSHA256 result mismatch for large file")
	}
}

func TestVerifySHA256WithSpecialCharacters(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "utils-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Test with file containing special characters
	specialFile := filepath.Join(tempDir, "special.txt")
	specialContent := "Hello, 世界! 🌍 Special chars: \n\t\r"
	
	err = os.WriteFile(specialFile, []byte(specialContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write special file: %v", err)
	}

	// Calculate expected SHA256
	specialHasher := sha256.New()
	specialHasher.Write([]byte(specialContent))
	specialSHA256 := hex.EncodeToString(specialHasher.Sum(nil))

	// Test verification
	err = VerifySHA256(specialFile, specialSHA256)
	if err != nil {
		t.Errorf("VerifySHA256 failed with special characters: %v", err)
	}

	// Test computation
	computedSHA256, err := ComputeSHA256(specialFile)
	if err != nil {
		t.Fatalf("ComputeSHA256 failed with special characters: %v", err)
	}

	if computedSHA256 != specialSHA256 {
		t.Errorf("ComputeSHA256 result mismatch for special characters")
	}
}

func TestSHA256CaseInsensitive(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "utils-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	testFile := filepath.Join(tempDir, "test.txt")
	content := "Case test"
	
	err = os.WriteFile(testFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Calculate expected SHA256
	hasher := sha256.New()
	hasher.Write([]byte(content))
	expectedSHA256 := hex.EncodeToString(hasher.Sum(nil))

	// Test with uppercase checksum
	uppercaseSHA256 := strings.ToUpper(expectedSHA256)
	err = VerifySHA256(testFile, uppercaseSHA256)
	if err == nil {
		t.Error("VerifySHA256 should be case sensitive and fail with uppercase checksum")
	}

	// Test with correct lowercase checksum
	err = VerifySHA256(testFile, expectedSHA256)
	if err != nil {
		t.Errorf("VerifySHA256 failed with correct lowercase checksum: %v", err)
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || strings.Contains(s, substr))
}

