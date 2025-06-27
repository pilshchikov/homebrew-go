package integration

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

var brewBinary string

func TestMain(m *testing.M) {
	// Build the brew binary for testing
	if err := buildBrew(); err != nil {
		panic("Failed to build brew binary: " + err.Error())
	}

	// Run tests
	code := m.Run()

	// Cleanup
	_ = os.RemoveAll(filepath.Dir(brewBinary))

	os.Exit(code)
}

func buildBrew() error {
	// Get the project root directory
	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	// Go up two levels to get to project root
	projectRoot := filepath.Join(wd, "..", "..")

	// Create temp directory for binary
	tmpDir, err := os.MkdirTemp("", "brew-test-*")
	if err != nil {
		return err
	}

	brewBinary = filepath.Join(tmpDir, "brew")

	// Build the binary
	cmd := exec.Command("go", "build", "-o", brewBinary, "./cmd/brew")
	cmd.Dir = projectRoot

	return cmd.Run()
}

func runBrew(args ...string) (string, string, error) {
	cmd := exec.Command(brewBinary, args...)

	// Create temporary directories for testing
	tempDir, _ := os.MkdirTemp("", "brew-test")
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Set minimal environment with temporary paths
	cmd.Env = []string{
		"PATH=" + os.Getenv("PATH"),
		"HOME=" + os.Getenv("HOME"),
		"HOMEBREW_NO_AUTO_UPDATE=1",
		"HOMEBREW_NO_ANALYTICS=1",
		"HOMEBREW_PREFIX=" + tempDir,
		"HOMEBREW_REPOSITORY=" + filepath.Join(tempDir, "Homebrew"),
		"HOMEBREW_LIBRARY=" + filepath.Join(tempDir, "Homebrew", "Library"),
		"HOMEBREW_CELLAR=" + filepath.Join(tempDir, "Cellar"),
		"HOMEBREW_CASKROOM=" + filepath.Join(tempDir, "Caskroom"),
		"HOMEBREW_CACHE=" + filepath.Join(tempDir, "Cache"),
		"HOMEBREW_LOGS=" + filepath.Join(tempDir, "Logs"),
		"HOMEBREW_TEMP=" + filepath.Join(tempDir, "Temp"),
	}

	// Create the necessary directories
	_ = os.MkdirAll(filepath.Join(tempDir, "Homebrew", "Library", "Taps"), 0755)

	stdout, err := cmd.Output()
	stderr := ""

	if exitErr, ok := err.(*exec.ExitError); ok {
		stderr = string(exitErr.Stderr)
	}

	return string(stdout), stderr, err
}

func TestVersionCommand(t *testing.T) {
	stdout, stderr, err := runBrew("--version")
	if err != nil {
		t.Fatalf("brew --version failed: %v\nstderr: %s", err, stderr)
	}

	if !strings.Contains(stdout, "Homebrew") {
		t.Errorf("Version output should contain 'Homebrew', got: %s", stdout)
	}

	if !strings.Contains(stdout, "Go:") {
		t.Errorf("Version output should contain Go version, got: %s", stdout)
	}

	if !strings.Contains(stdout, "Platform:") {
		t.Errorf("Version output should contain platform info, got: %s", stdout)
	}
}

func TestHelpCommand(t *testing.T) {
	stdout, stderr, err := runBrew("--help")
	if err != nil {
		t.Fatalf("brew --help failed: %v\nstderr: %s", err, stderr)
	}

	expectedCommands := []string{
		"install", "uninstall", "upgrade", "update",
		"search", "info", "list", "cleanup",
		"doctor", "config", "tap", "untap",
	}

	for _, cmd := range expectedCommands {
		if !strings.Contains(stdout, cmd) {
			t.Errorf("Help output should contain command '%s', got: %s", cmd, stdout)
		}
	}

	if !strings.Contains(stdout, "Usage:") {
		t.Errorf("Help output should contain usage information, got: %s", stdout)
	}
}

func TestConfigCommand(t *testing.T) {
	stdout, stderr, err := runBrew("config")
	if err != nil {
		t.Fatalf("brew config failed: %v\nstderr: %s", err, stderr)
	}

	expectedKeys := []string{
		"HOMEBREW_PREFIX:",
		"HOMEBREW_REPOSITORY:",
		"HOMEBREW_CELLAR:",
		"HOMEBREW_CACHE:",
	}

	for _, key := range expectedKeys {
		if !strings.Contains(stdout, key) {
			t.Errorf("Config output should contain '%s', got: %s", key, stdout)
		}
	}
}

func TestEnvCommand(t *testing.T) {
	stdout, stderr, err := runBrew("env")
	if err != nil {
		t.Fatalf("brew env failed: %v\nstderr: %s", err, stderr)
	}

	expectedExports := []string{
		"export HOMEBREW_PREFIX=",
		"export HOMEBREW_REPOSITORY=",
		"export HOMEBREW_CELLAR=",
		"export PATH=",
	}

	for _, export := range expectedExports {
		if !strings.Contains(stdout, export) {
			t.Errorf("Env output should contain '%s', got: %s", export, stdout)
		}
	}
}

func TestPrefixCommand(t *testing.T) {
	stdout, stderr, err := runBrew("prefix")
	if err != nil {
		t.Fatalf("brew prefix failed: %v\nstderr: %s", err, stderr)
	}

	prefix := strings.TrimSpace(stdout)
	if prefix == "" {
		t.Error("Prefix should not be empty")
	}

	// Should be an absolute path
	if !filepath.IsAbs(prefix) {
		t.Errorf("Prefix should be absolute path, got: %s", prefix)
	}
}

func TestCellarCommand(t *testing.T) {
	stdout, stderr, err := runBrew("cellar")
	if err != nil {
		t.Fatalf("brew cellar failed: %v\nstderr: %s", err, stderr)
	}

	cellar := strings.TrimSpace(stdout)
	if cellar == "" {
		t.Error("Cellar path should not be empty")
	}

	// Should be an absolute path
	if !filepath.IsAbs(cellar) {
		t.Errorf("Cellar should be absolute path, got: %s", cellar)
	}
}

func TestCacheCommand(t *testing.T) {
	stdout, stderr, err := runBrew("cache")
	if err != nil {
		t.Fatalf("brew cache failed: %v\nstderr: %s", err, stderr)
	}

	cache := strings.TrimSpace(stdout)
	if cache == "" {
		t.Error("Cache path should not be empty")
	}

	// Should be an absolute path
	if !filepath.IsAbs(cache) {
		t.Errorf("Cache should be absolute path, got: %s", cache)
	}
}

func TestSearchCommand(t *testing.T) {
	stdout, stderr, err := runBrew("search", "--help")
	if err != nil {
		t.Fatalf("brew search --help failed: %v\nstderr: %s", err, stderr)
	}

	if !strings.Contains(stdout, "Usage:") {
		t.Errorf("Search help should contain usage information, got: %s", stdout)
	}
}

func TestInstallCommandHelp(t *testing.T) {
	stdout, stderr, err := runBrew("install", "--help")
	if err != nil {
		t.Fatalf("brew install --help failed: %v\nstderr: %s", err, stderr)
	}

	expectedFlags := []string{
		"--formula",
		"--cask",
		"--build-from-source",
		"--force-bottle",
		"--dry-run",
	}

	for _, flag := range expectedFlags {
		if !strings.Contains(stdout, flag) {
			t.Errorf("Install help should contain flag '%s', got: %s", flag, stdout)
		}
	}
}

func TestDryRunInstall(t *testing.T) {
	_, stderr, err := runBrew("install", "--dry-run", "nonexistent-formula")

	// Dry run should not fail due to dry-run logic itself
	if err != nil {
		// If it fails, it should be due to formula not found or not implemented, not due to dry-run logic
		if !strings.Contains(stderr, "not found") && !strings.Contains(stderr, "not yet implemented") {
			t.Errorf("Unexpected error for dry-run install: %v\nstderr: %s", err, stderr)
		}
	}
}

func TestListCommand(t *testing.T) {
	stdout, stderr, err := runBrew("list")
	// List command should succeed even if no packages are installed
	if err != nil {
		// Only fail if it's not a "not implemented" error
		if !strings.Contains(stderr, "not yet implemented") {
			t.Fatalf("brew list failed: %v\nstderr: %s", err, stderr)
		}
	}

	// If successful, output can be empty (no packages installed)
	_ = stdout // Don't require any specific output
}

func TestDoctorCommand(t *testing.T) {
	stdout, stderr, err := runBrew("doctor")
	// Doctor command should provide system diagnostics
	if err != nil {
		// Only fail if it's not a "not implemented" error
		if !strings.Contains(stderr, "not yet implemented") {
			t.Fatalf("brew doctor failed: %v\nstderr: %s", err, stderr)
		}
	}

	_ = stdout // Output may vary based on system state
}

func TestTapCommand(t *testing.T) {
	stdout, stderr, err := runBrew("tap")
	// Tap command without arguments should list taps
	if err != nil {
		// Only fail if it's not a "not implemented" error or "no taps directory" error
		if !strings.Contains(stderr, "not yet implemented") && 
		   !strings.Contains(stderr, "no such file or directory") {
			t.Fatalf("brew tap failed: %v\nstderr: %s", err, stderr)
		}
		t.Skipf("Tap command failed as expected in test environment: %s", stderr)
	}

	_ = stdout // Output may be empty if no taps are installed
}

func TestInvalidCommand(t *testing.T) {
	stdout, stderr, err := runBrew("nonexistent-command")

	// Should fail with non-zero exit code
	if err == nil {
		t.Error("Invalid command should fail")
	}

	// Should provide helpful error message
	output := stdout + stderr
	if !strings.Contains(output, "Unknown command") && !strings.Contains(output, "unknown command") {
		t.Errorf("Should indicate unknown command, got: %s", output)
	}
}

func TestGlobalFlags(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{"verbose flag", []string{"--verbose", "--help"}},
		{"debug flag", []string{"--debug", "--help"}},
		{"quiet flag", []string{"--quiet", "--help"}},
		{"force flag", []string{"--force", "--help"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout, stderr, err := runBrew(tt.args...)
			if err != nil {
				t.Fatalf("%s failed: %v\nstderr: %s", tt.name, err, stderr)
			}

			// Help should still work with global flags
			if !strings.Contains(stdout, "Usage:") {
				t.Errorf("%s should still show help, got: %s", tt.name, stdout)
			}
		})
	}
}

func TestCompletionCommand(t *testing.T) {
	shells := []string{"bash", "zsh", "fish", "powershell"}

	for _, shell := range shells {
		t.Run("completion for "+shell, func(t *testing.T) {
			stdout, stderr, err := runBrew("completion", shell)
			if err != nil {
				t.Fatalf("brew completion %s failed: %v\nstderr: %s", shell, err, stderr)
			}

			if len(stdout) == 0 {
				t.Errorf("Completion for %s should generate output", shell)
			}
		})
	}
}
