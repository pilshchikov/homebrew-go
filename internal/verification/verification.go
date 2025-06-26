package verification

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/homebrew/brew/internal/errors"
	"github.com/homebrew/brew/internal/logger"
)

// ChecksumType represents different checksum algorithms
type ChecksumType string

const (
	SHA256 ChecksumType = "sha256"
	SHA512 ChecksumType = "sha512"
	SHA1   ChecksumType = "sha1"
	MD5    ChecksumType = "md5"
)

// Checksum represents a checksum with its type and value
type Checksum struct {
	Type  ChecksumType
	Value string
}

// FileInfo contains metadata about a file for verification
type FileInfo struct {
	Path         string
	Size         int64
	ModTime      time.Time
	Checksums    []Checksum
	ExpectedSize int64 // 0 if unknown
}

// VerificationResult contains the results of verification checks
type VerificationResult struct {
	FilePath       string
	ChecksumsPassed map[ChecksumType]bool
	SizeMatches    bool
	FileExists     bool
	Errors         []error
	Warnings       []string
}

// Verifier handles package verification and integrity checks
type Verifier struct {
	enableSizeCheck bool
	enableTimeCheck bool
	strictMode      bool
}

// NewVerifier creates a new package verifier
func NewVerifier(strict bool) *Verifier {
	return &Verifier{
		enableSizeCheck: true,
		enableTimeCheck: false, // Time checks can be unreliable for downloads
		strictMode:      strict,
	}
}

// VerifyFile performs comprehensive verification of a file
func (v *Verifier) VerifyFile(fileInfo *FileInfo) *VerificationResult {
	result := &VerificationResult{
		FilePath:        fileInfo.Path,
		ChecksumsPassed: make(map[ChecksumType]bool),
		Errors:          []error{},
		Warnings:        []string{},
	}

	// Check if file exists
	stat, err := os.Stat(fileInfo.Path)
	if err != nil {
		result.FileExists = false
		result.Errors = append(result.Errors, 
			errors.NewPermissionError("file access", fileInfo.Path, err))
		return result
	}
	result.FileExists = true

	// Verify file size if expected size is provided
	if fileInfo.ExpectedSize > 0 && v.enableSizeCheck {
		result.SizeMatches = stat.Size() == fileInfo.ExpectedSize
		if !result.SizeMatches {
			if v.strictMode {
				result.Errors = append(result.Errors, fmt.Errorf(
					"file size mismatch: expected %d bytes, got %d bytes", 
					fileInfo.ExpectedSize, stat.Size()))
			} else {
				result.Warnings = append(result.Warnings, fmt.Sprintf(
					"File size mismatch: expected %d bytes, got %d bytes", 
					fileInfo.ExpectedSize, stat.Size()))
			}
		}
	} else {
		result.SizeMatches = true // Unknown size, assume OK
	}

	// Verify checksums
	for _, checksum := range fileInfo.Checksums {
		passed, err := v.verifyChecksum(fileInfo.Path, checksum)
		result.ChecksumsPassed[checksum.Type] = passed
		if err != nil {
			result.Errors = append(result.Errors, err)
		}
	}

	return result
}

// verifyChecksum verifies a single checksum
func (v *Verifier) verifyChecksum(filePath string, checksum Checksum) (bool, error) {
	logger.Debug("Verifying %s checksum for %s", checksum.Type, filepath.Base(filePath))

	hasher, err := v.getHasher(checksum.Type)
	if err != nil {
		return false, err
	}

	file, err := os.Open(filePath)
	if err != nil {
		return false, errors.NewPermissionError("read file for checksum", filePath, err)
	}
	defer file.Close()

	if _, err := io.Copy(hasher, file); err != nil {
		return false, fmt.Errorf("failed to compute %s checksum: %w", checksum.Type, err)
	}

	actualChecksum := hex.EncodeToString(hasher.Sum(nil))
	expectedChecksum := strings.ToLower(checksum.Value)
	actualChecksum = strings.ToLower(actualChecksum)

	if actualChecksum != expectedChecksum {
		// Extract formula info from path for better error reporting
		filename := filepath.Base(filePath)
		parts := strings.Split(filename, "-")
		formula := ""
		version := ""
		if len(parts) >= 2 {
			formula = parts[0]
			version = parts[1]
		}
		return false, errors.NewChecksumError(formula, version, expectedChecksum, actualChecksum)
	}

	logger.Debug("%s checksum verified successfully", checksum.Type)
	return true, nil
}

// getHasher returns the appropriate hash function for the checksum type
func (v *Verifier) getHasher(checksumType ChecksumType) (hash.Hash, error) {
	switch checksumType {
	case SHA256:
		return sha256.New(), nil
	case SHA512:
		return sha512.New(), nil
	case SHA1:
		return sha1.New(), nil
	case MD5:
		return md5.New(), nil
	default:
		return nil, fmt.Errorf("unsupported checksum type: %s", checksumType)
	}
}

// ComputeChecksum computes a checksum for a file
func (v *Verifier) ComputeChecksum(filePath string, checksumType ChecksumType) (string, error) {
	hasher, err := v.getHasher(checksumType)
	if err != nil {
		return "", err
	}

	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// VerifyMultipleFiles verifies multiple files concurrently
func (v *Verifier) VerifyMultipleFiles(files []*FileInfo) []*VerificationResult {
	results := make([]*VerificationResult, len(files))
	
	// For small numbers of files, verify sequentially
	// For larger numbers, we could implement concurrent verification
	for i, fileInfo := range files {
		results[i] = v.VerifyFile(fileInfo)
	}
	
	return results
}

// IsVerificationSuccessful checks if verification passed all critical checks
func (result *VerificationResult) IsVerificationSuccessful() bool {
	if !result.FileExists {
		return false
	}
	
	// Check if any checksums failed
	for _, passed := range result.ChecksumsPassed {
		if !passed {
			return false
		}
	}
	
	// In strict mode, size must also match
	// (We don't check this in non-strict mode since download sizes can vary)
	
	return len(result.Errors) == 0
}

// GetSummary returns a human-readable summary of verification results
func (result *VerificationResult) GetSummary() string {
	if result.IsVerificationSuccessful() {
		checksumCount := len(result.ChecksumsPassed)
		return fmt.Sprintf("✓ Verification passed (%d checksums verified)", checksumCount)
	}
	
	var issues []string
	if !result.FileExists {
		issues = append(issues, "file does not exist")
	}
	
	for checksumType, passed := range result.ChecksumsPassed {
		if !passed {
			issues = append(issues, fmt.Sprintf("%s checksum failed", checksumType))
		}
	}
	
	if !result.SizeMatches {
		issues = append(issues, "size mismatch")
	}
	
	if len(result.Errors) > 0 {
		issues = append(issues, fmt.Sprintf("%d errors", len(result.Errors)))
	}
	
	return fmt.Sprintf("✗ Verification failed: %s", strings.Join(issues, ", "))
}

// LogResults logs the verification results with appropriate log levels
func (result *VerificationResult) LogResults() {
	if result.IsVerificationSuccessful() {
		logger.Success("Package verification: %s", result.GetSummary())
	} else {
		logger.Error("Package verification failed for %s", filepath.Base(result.FilePath))
		for _, err := range result.Errors {
			logger.Error("  - %v", err)
		}
		for _, warning := range result.Warnings {
			logger.Warn("  - %s", warning)
		}
	}
}

// PackageVerifier provides high-level verification for Homebrew packages
type PackageVerifier struct {
	verifier *Verifier
}

// NewPackageVerifier creates a new package verifier
func NewPackageVerifier(strict bool) *PackageVerifier {
	return &PackageVerifier{
		verifier: NewVerifier(strict),
	}
}

// VerifyBottle verifies a downloaded bottle file
func (pv *PackageVerifier) VerifyBottle(bottlePath, expectedSHA256 string, expectedSize int64) error {
	fileInfo := &FileInfo{
		Path:         bottlePath,
		ExpectedSize: expectedSize,
		Checksums: []Checksum{
			{Type: SHA256, Value: expectedSHA256},
		},
	}
	
	result := pv.verifier.VerifyFile(fileInfo)
	result.LogResults()
	
	if !result.IsVerificationSuccessful() {
		return fmt.Errorf("bottle verification failed: %s", result.GetSummary())
	}
	
	return nil
}

// VerifySource verifies a downloaded source archive
func (pv *PackageVerifier) VerifySource(sourcePath, expectedSHA256 string, expectedSize int64) error {
	fileInfo := &FileInfo{
		Path:         sourcePath,
		ExpectedSize: expectedSize,
		Checksums: []Checksum{
			{Type: SHA256, Value: expectedSHA256},
		},
	}
	
	result := pv.verifier.VerifyFile(fileInfo)
	result.LogResults()
	
	if !result.IsVerificationSuccessful() {
		return fmt.Errorf("source verification failed: %s", result.GetSummary())
	}
	
	return nil
}

// VerifyInstallation verifies an installed package's integrity
func (pv *PackageVerifier) VerifyInstallation(installPath string) *VerificationResult {
	// For installed packages, we primarily check if files exist and have reasonable sizes
	// We can't verify checksums since files may have been modified during installation
	
	stat, err := os.Stat(installPath)
	if err != nil {
		return &VerificationResult{
			FilePath:   installPath,
			FileExists: false,
			Errors:     []error{err},
		}
	}
	
	result := &VerificationResult{
		FilePath:        installPath,
		FileExists:      true,
		SizeMatches:     true, // We don't have expected size for installations
		ChecksumsPassed: make(map[ChecksumType]bool),
	}
	
	// Check if it's a directory (typical for installations)
	if stat.IsDir() {
		// Count files in installation
		fileCount := 0
		err := filepath.Walk(installPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil // Skip errors, don't fail the whole verification
			}
			if !info.IsDir() {
				fileCount++
			}
			return nil
		})
		
		if err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("Could not walk installation directory: %v", err))
		} else if fileCount == 0 {
			result.Warnings = append(result.Warnings, "Installation directory appears to be empty")
		} else {
			logger.Debug("Installation contains %d files", fileCount)
		}
	}
	
	return result
}