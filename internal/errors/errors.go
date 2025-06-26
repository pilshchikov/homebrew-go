package errors

import (
	"fmt"
	"strings"
)

// ErrorType represents different categories of errors
type ErrorType int

const (
	// NetworkError represents network connectivity issues
	NetworkError ErrorType = iota
	// DependencyError represents dependency resolution issues
	DependencyError
	// BuildError represents compilation/build failures
	BuildError
	// PermissionError represents file system permission issues
	PermissionError
	// FormulaNotFoundError represents missing formula errors
	FormulaNotFoundError
	// ConfigurationError represents configuration issues
	ConfigurationError
	// InstallationError represents general installation failures
	InstallationError
	// DownloadError represents download failures
	DownloadError
	// ChecksumError represents checksum verification failures
	ChecksumError
)

// BrewError represents a structured error with context
type BrewError struct {
	Type        ErrorType
	Operation   string
	Formula     string
	Version     string
	Platform    string
	Cause       error
	Suggestions []string
	Recoverable bool
}

// Error implements the error interface
func (e *BrewError) Error() string {
	var parts []string
	
	if e.Operation != "" {
		parts = append(parts, fmt.Sprintf("operation '%s' failed", e.Operation))
	}
	
	if e.Formula != "" {
		parts = append(parts, fmt.Sprintf("for formula '%s'", e.Formula))
	}
	
	if e.Version != "" {
		parts = append(parts, fmt.Sprintf("version '%s'", e.Version))
	}
	
	if e.Cause != nil {
		parts = append(parts, fmt.Sprintf("reason: %v", e.Cause))
	}
	
	return strings.Join(parts, " ")
}

// Unwrap returns the underlying error
func (e *BrewError) Unwrap() error {
	return e.Cause
}

// Is checks if the error matches a specific type
func (e *BrewError) Is(target error) bool {
	if brewErr, ok := target.(*BrewError); ok {
		return e.Type == brewErr.Type
	}
	return false
}

// NewNetworkError creates a network-related error
func NewNetworkError(operation, url string, cause error) *BrewError {
	suggestions := []string{
		"Check your internet connection",
		"Verify that the URL is accessible",
		"Try again in a few minutes",
	}
	
	if strings.Contains(url, "github.com") {
		suggestions = append(suggestions, "Check GitHub's status at https://status.github.com")
	}
	
	return &BrewError{
		Type:        NetworkError,
		Operation:   operation,
		Cause:       cause,
		Suggestions: suggestions,
		Recoverable: true,
	}
}

// NewDependencyError creates a dependency-related error
func NewDependencyError(formula, dependency string, cause error) *BrewError {
	suggestions := []string{
		fmt.Sprintf("Try installing '%s' separately first", dependency),
		"Check if the dependency name is correct",
		"Use --ignore-dependencies to skip dependency checks",
	}
	
	return &BrewError{
		Type:        DependencyError,
		Operation:   "dependency resolution",
		Formula:     formula,
		Cause:       cause,
		Suggestions: suggestions,
		Recoverable: true,
	}
}

// NewBuildError creates a build-related error
func NewBuildError(formula, version string, cause error) *BrewError {
	suggestions := []string{
		"Try building from source with --build-from-source",
		"Check if you have the required build tools installed",
		"Look for error messages in the build output above",
		"Search for known issues with this formula",
	}
	
	return &BrewError{
		Type:        BuildError,
		Operation:   "build",
		Formula:     formula,
		Version:     version,
		Cause:       cause,
		Suggestions: suggestions,
		Recoverable: false,
	}
}

// NewPermissionError creates a permission-related error
func NewPermissionError(operation, path string, cause error) *BrewError {
	suggestions := []string{
		"Check file and directory permissions",
		"Ensure you have write access to the installation directory",
		"Try running with appropriate permissions",
	}
	
	return &BrewError{
		Type:        PermissionError,
		Operation:   operation,
		Cause:       cause,
		Suggestions: suggestions,
		Recoverable: true,
	}
}

// NewFormulaNotFoundError creates a formula not found error
func NewFormulaNotFoundError(formula string) *BrewError {
	suggestions := []string{
		fmt.Sprintf("Search for similar formulae with 'brew search %s'", formula),
		"Check if the formula name is spelled correctly",
		"Try updating your tap list with 'brew update'",
		"Check if the formula is in a tap that needs to be added",
	}
	
	return &BrewError{
		Type:        FormulaNotFoundError,
		Operation:   "formula lookup",
		Formula:     formula,
		Suggestions: suggestions,
		Recoverable: false,
	}
}

// NewDownloadError creates a download-related error
func NewDownloadError(operation, url string, cause error) *BrewError {
	suggestions := []string{
		"Check your internet connection",
		"Verify the download URL is correct",
		"Try downloading manually to test connectivity",
	}
	
	if strings.Contains(cause.Error(), "404") {
		suggestions = append(suggestions, "The file may have been moved or deleted")
	}
	
	if strings.Contains(cause.Error(), "timeout") || strings.Contains(cause.Error(), "deadline exceeded") {
		suggestions = append(suggestions, "The server may be slow, try again later")
	}
	
	return &BrewError{
		Type:        DownloadError,
		Operation:   operation,
		Cause:       cause,
		Suggestions: suggestions,
		Recoverable: true,
	}
}

// NewChecksumError creates a checksum verification error
func NewChecksumError(formula, version string, expected, actual string) *BrewError {
	cause := fmt.Errorf("checksum mismatch: expected %s, got %s", expected, actual)
	
	suggestions := []string{
		"The download may be corrupted, try downloading again",
		"Clear your cache and retry the installation",
		"Check if there's a newer version of the formula available",
		"Report this issue if it persists",
	}
	
	return &BrewError{
		Type:        ChecksumError,
		Operation:   "checksum verification",
		Formula:     formula,
		Version:     version,
		Cause:       cause,
		Suggestions: suggestions,
		Recoverable: true,
	}
}

// NewConfigurationError creates a configuration-related error
func NewConfigurationError(operation string, cause error) *BrewError {
	suggestions := []string{
		"Check your Homebrew configuration",
		"Verify environment variables are set correctly",
		"Try running 'brew doctor' to diagnose issues",
	}
	
	return &BrewError{
		Type:        ConfigurationError,
		Operation:   operation,
		Cause:       cause,
		Suggestions: suggestions,
		Recoverable: true,
	}
}

// NewInstallationError creates a general installation error
func NewInstallationError(formula, version string, cause error) *BrewError {
	suggestions := []string{
		"Check the installation logs for more details",
		"Try installing with --verbose for more information",
		"Search for known issues with this formula",
		"Consider using an alternative formula if available",
	}
	
	return &BrewError{
		Type:        InstallationError,
		Operation:   "installation",
		Formula:     formula,
		Version:     version,
		Cause:       cause,
		Suggestions: suggestions,
		Recoverable: false,
	}
}

// ErrorRecovery provides recovery suggestions and actions
type ErrorRecovery struct {
	CanRetry          bool
	CanIgnore         bool
	CanUseAlternative bool
	RetryDelay        int // seconds
	MaxRetries        int
}

// GetRecoveryOptions returns recovery options for a given error
func GetRecoveryOptions(err *BrewError) ErrorRecovery {
	switch err.Type {
	case NetworkError, DownloadError:
		return ErrorRecovery{
			CanRetry:   true,
			RetryDelay: 5,
			MaxRetries: 3,
		}
	case ChecksumError:
		return ErrorRecovery{
			CanRetry:   true,
			RetryDelay: 1,
			MaxRetries: 2,
		}
	case DependencyError:
		return ErrorRecovery{
			CanRetry:          true,
			CanIgnore:         true,
			CanUseAlternative: true,
			MaxRetries:        1,
		}
	case PermissionError, ConfigurationError:
		return ErrorRecovery{
			CanRetry:   true,
			MaxRetries: 1,
		}
	default:
		return ErrorRecovery{
			CanRetry:   false,
			MaxRetries: 0,
		}
	}
}

// Wrap wraps an error with additional context
func Wrap(err error, operation, formula string) error {
	if err == nil {
		return nil
	}
	
	if brewErr, ok := err.(*BrewError); ok {
		// Update existing BrewError with additional context
		brewErr.Operation = operation
		if brewErr.Formula == "" {
			brewErr.Formula = formula
		}
		return brewErr
	}
	
	// Create new BrewError from generic error
	return &BrewError{
		Type:      InstallationError,
		Operation: operation,
		Formula:   formula,
		Cause:     err,
	}
}

// IsRecoverable checks if an error can be recovered from
func IsRecoverable(err error) bool {
	if brewErr, ok := err.(*BrewError); ok {
		return brewErr.Recoverable
	}
	return false
}

// GetErrorType returns the error type for a given error
func GetErrorType(err error) ErrorType {
	if brewErr, ok := err.(*BrewError); ok {
		return brewErr.Type
	}
	return InstallationError
}