package errors

import (
	"fmt"
	"strings"
	"testing"
)

func TestBrewError_Error(t *testing.T) {
	tests := []struct {
		name     string
		brewErr  *BrewError
		expected []string // substrings that should be present
	}{
		{
			name: "network error with all fields",
			brewErr: &BrewError{
				Type:      NetworkError,
				Operation: "download",
				Formula:   "hello",
				Version:   "2.12.2",
				Cause:     fmt.Errorf("connection timeout"),
			},
			expected: []string{"operation 'download' failed", "for formula 'hello'", "version '2.12.2'", "connection timeout"},
		},
		{
			name: "minimal error",
			brewErr: &BrewError{
				Type:      BuildError,
				Operation: "compilation",
				Cause:     fmt.Errorf("make failed"),
			},
			expected: []string{"operation 'compilation' failed", "make failed"},
		},
		{
			name: "formula not found",
			brewErr: &BrewError{
				Type:    FormulaNotFoundError,
				Formula: "nonexistent",
			},
			expected: []string{"for formula 'nonexistent'"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.brewErr.Error()
			for _, expected := range tt.expected {
				if !strings.Contains(result, expected) {
					t.Errorf("BrewError.Error() = %q, should contain %q", result, expected)
				}
			}
		})
	}
}

func TestBrewError_Unwrap(t *testing.T) {
	cause := fmt.Errorf("underlying error")
	brewErr := &BrewError{
		Type:  NetworkError,
		Cause: cause,
	}

	if brewErr.Unwrap() != cause {
		t.Errorf("BrewError.Unwrap() should return the underlying error")
	}
}

func TestBrewError_Is(t *testing.T) {
	err1 := &BrewError{Type: NetworkError}
	err2 := &BrewError{Type: NetworkError}
	err3 := &BrewError{Type: BuildError}
	genericErr := fmt.Errorf("generic error")

	if !err1.Is(err2) {
		t.Errorf("BrewError.Is() should return true for same error type")
	}

	if err1.Is(err3) {
		t.Errorf("BrewError.Is() should return false for different error type")
	}

	if err1.Is(genericErr) {
		t.Errorf("BrewError.Is() should return false for non-BrewError")
	}
}

func TestNewNetworkError(t *testing.T) {
	operation := "download"
	url := "https://github.com/example/repo"
	cause := fmt.Errorf("connection timeout")

	err := NewNetworkError(operation, url, cause)

	if err.Type != NetworkError {
		t.Errorf("NewNetworkError() Type = %v, want %v", err.Type, NetworkError)
	}

	if err.Operation != operation {
		t.Errorf("NewNetworkError() Operation = %v, want %v", err.Operation, operation)
	}

	if err.Cause != cause {
		t.Errorf("NewNetworkError() Cause = %v, want %v", err.Cause, cause)
	}

	if !err.Recoverable {
		t.Errorf("NewNetworkError() should be recoverable")
	}

	if len(err.Suggestions) == 0 {
		t.Errorf("NewNetworkError() should have suggestions")
	}

	// Test GitHub-specific suggestions
	if !strings.Contains(strings.Join(err.Suggestions, " "), "GitHub") {
		t.Errorf("NewNetworkError() should include GitHub-specific suggestions for GitHub URLs")
	}
}

func TestNewDependencyError(t *testing.T) {
	formula := "main-formula"
	dependency := "dep-formula"
	cause := fmt.Errorf("dependency not found")

	err := NewDependencyError(formula, dependency, cause)

	if err.Type != DependencyError {
		t.Errorf("NewDependencyError() Type = %v, want %v", err.Type, DependencyError)
	}

	if err.Formula != formula {
		t.Errorf("NewDependencyError() Formula = %v, want %v", err.Formula, formula)
	}

	if !err.Recoverable {
		t.Errorf("NewDependencyError() should be recoverable")
	}

	// Check for dependency-specific suggestions
	hasDepSuggestion := false
	for _, suggestion := range err.Suggestions {
		if strings.Contains(suggestion, dependency) {
			hasDepSuggestion = true
			break
		}
	}
	if !hasDepSuggestion {
		t.Errorf("NewDependencyError() should include dependency-specific suggestions")
	}
}

func TestNewBuildError(t *testing.T) {
	formula := "test-formula"
	version := "1.0.0"
	cause := fmt.Errorf("compilation failed")

	err := NewBuildError(formula, version, cause)

	if err.Type != BuildError {
		t.Errorf("NewBuildError() Type = %v, want %v", err.Type, BuildError)
	}

	if err.Formula != formula {
		t.Errorf("NewBuildError() Formula = %v, want %v", err.Formula, formula)
	}

	if err.Version != version {
		t.Errorf("NewBuildError() Version = %v, want %v", err.Version, version)
	}

	if err.Recoverable {
		t.Errorf("NewBuildError() should not be recoverable")
	}
}

func TestNewFormulaNotFoundError(t *testing.T) {
	formula := "nonexistent-formula"

	err := NewFormulaNotFoundError(formula)

	if err.Type != FormulaNotFoundError {
		t.Errorf("NewFormulaNotFoundError() Type = %v, want %v", err.Type, FormulaNotFoundError)
	}

	if err.Formula != formula {
		t.Errorf("NewFormulaNotFoundError() Formula = %v, want %v", err.Formula, formula)
	}

	if err.Recoverable {
		t.Errorf("NewFormulaNotFoundError() should not be recoverable")
	}

	// Check for search suggestion
	hasSearchSuggestion := false
	for _, suggestion := range err.Suggestions {
		if strings.Contains(suggestion, "brew search") && strings.Contains(suggestion, formula) {
			hasSearchSuggestion = true
			break
		}
	}
	if !hasSearchSuggestion {
		t.Errorf("NewFormulaNotFoundError() should include search suggestion")
	}
}

func TestNewChecksumError(t *testing.T) {
	formula := "test-formula"
	version := "1.0.0"
	expected := "abc123"
	actual := "def456"

	err := NewChecksumError(formula, version, expected, actual)

	if err.Type != ChecksumError {
		t.Errorf("NewChecksumError() Type = %v, want %v", err.Type, ChecksumError)
	}

	if err.Formula != formula {
		t.Errorf("NewChecksumError() Formula = %v, want %v", err.Formula, formula)
	}

	if err.Version != version {
		t.Errorf("NewChecksumError() Version = %v, want %v", err.Version, version)
	}

	if !err.Recoverable {
		t.Errorf("NewChecksumError() should be recoverable")
	}

	// Check that the error message contains both checksums
	errMsg := err.Error()
	if !strings.Contains(errMsg, expected) || !strings.Contains(errMsg, actual) {
		t.Errorf("NewChecksumError() error message should contain both checksums")
	}
}

func TestGetRecoveryOptions(t *testing.T) {
	tests := []struct {
		name        string
		errorType   ErrorType
		expectRetry bool
		expectIgnore bool
		maxRetries  int
	}{
		{
			name:        "network error",
			errorType:   NetworkError,
			expectRetry: true,
			expectIgnore: false,
			maxRetries:  3,
		},
		{
			name:        "dependency error",
			errorType:   DependencyError,
			expectRetry: true,
			expectIgnore: true,
			maxRetries:  1,
		},
		{
			name:        "build error",
			errorType:   BuildError,
			expectRetry: false,
			expectIgnore: false,
			maxRetries:  0,
		},
		{
			name:        "checksum error",
			errorType:   ChecksumError,
			expectRetry: true,
			expectIgnore: false,
			maxRetries:  2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := &BrewError{Type: tt.errorType}
			recovery := GetRecoveryOptions(err)

			if recovery.CanRetry != tt.expectRetry {
				t.Errorf("GetRecoveryOptions() CanRetry = %v, want %v", recovery.CanRetry, tt.expectRetry)
			}

			if recovery.CanIgnore != tt.expectIgnore {
				t.Errorf("GetRecoveryOptions() CanIgnore = %v, want %v", recovery.CanIgnore, tt.expectIgnore)
			}

			if recovery.MaxRetries != tt.maxRetries {
				t.Errorf("GetRecoveryOptions() MaxRetries = %v, want %v", recovery.MaxRetries, tt.maxRetries)
			}
		})
	}
}

func TestWrap(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		operation string
		formula   string
		expectNil bool
		expectType ErrorType
	}{
		{
			name:      "nil error",
			err:       nil,
			operation: "test",
			formula:   "test",
			expectNil: true,
		},
		{
			name:       "existing BrewError",
			err:        &BrewError{Type: NetworkError, Formula: "original"},
			operation:  "new-operation",
			formula:    "new-formula",
			expectNil:  false,
			expectType: NetworkError,
		},
		{
			name:       "generic error",
			err:        fmt.Errorf("generic error"),
			operation:  "test-operation",
			formula:    "test-formula",
			expectNil:  false,
			expectType: InstallationError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Wrap(tt.err, tt.operation, tt.formula)

			if tt.expectNil {
				if result != nil {
					t.Errorf("Wrap() should return nil for nil error")
				}
				return
			}

			brewErr, ok := result.(*BrewError)
			if !ok {
				t.Errorf("Wrap() should return BrewError")
				return
			}

			if brewErr.Type != tt.expectType {
				t.Errorf("Wrap() Type = %v, want %v", brewErr.Type, tt.expectType)
			}

			if brewErr.Operation != tt.operation {
				t.Errorf("Wrap() Operation = %v, want %v", brewErr.Operation, tt.operation)
			}
		})
	}
}

func TestIsRecoverable(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "recoverable BrewError",
			err:      &BrewError{Recoverable: true},
			expected: true,
		},
		{
			name:     "non-recoverable BrewError",
			err:      &BrewError{Recoverable: false},
			expected: false,
		},
		{
			name:     "generic error",
			err:      fmt.Errorf("generic error"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsRecoverable(tt.err)
			if result != tt.expected {
				t.Errorf("IsRecoverable() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGetErrorType(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected ErrorType
	}{
		{
			name:     "BrewError",
			err:      &BrewError{Type: NetworkError},
			expected: NetworkError,
		},
		{
			name:     "generic error",
			err:      fmt.Errorf("generic error"),
			expected: InstallationError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetErrorType(tt.err)
			if result != tt.expected {
				t.Errorf("GetErrorType() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestNewDownloadError(t *testing.T) {
	tests := []struct {
		name          string
		operation     string
		url           string
		cause         error
		expectedSuggs []string
	}{
		{
			name:      "404 error",
			operation: "download",
			url:       "https://example.com/file.tar.gz",
			cause:     fmt.Errorf("HTTP 404: Not Found"),
			expectedSuggs: []string{"moved or deleted"},
		},
		{
			name:      "timeout error",
			operation: "download",
			url:       "https://example.com/file.tar.gz",
			cause:     fmt.Errorf("context deadline exceeded"),
			expectedSuggs: []string{"slow", "try again later"},
		},
		{
			name:      "generic error",
			operation: "download",
			url:       "https://example.com/file.tar.gz",
			cause:     fmt.Errorf("connection refused"),
			expectedSuggs: []string{"internet connection"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewDownloadError(tt.operation, tt.url, tt.cause)

			if err.Type != DownloadError {
				t.Errorf("NewDownloadError() Type = %v, want %v", err.Type, DownloadError)
			}

			if !err.Recoverable {
				t.Errorf("NewDownloadError() should be recoverable")
			}

			// Check for expected suggestions
			suggestions := strings.Join(err.Suggestions, " ")
			for _, expectedSugg := range tt.expectedSuggs {
				if !strings.Contains(suggestions, expectedSugg) {
					t.Errorf("NewDownloadError() suggestions should contain %q, got: %v", expectedSugg, err.Suggestions)
				}
			}
		})
	}
}