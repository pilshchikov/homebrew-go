package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pilshchikov/homebrew-go/internal/config"
	"github.com/pilshchikov/homebrew-go/internal/logger"
)

func TestNewOutdatedCmd(t *testing.T) {
	logger.Init(false, false, true)
	cfg := &config.Config{}

	cmd := NewOutdatedCmd(cfg)

	if cmd.Use != "outdated [OPTIONS] [FORMULA|CASK...]" {
		t.Errorf("Expected Use to be 'outdated [OPTIONS] [FORMULA|CASK...]', got %s", cmd.Use)
	}

	if cmd.Short != "List installed formulae and casks that have a more recent version available" {
		t.Errorf("Expected correct Short description, got %s", cmd.Short)
	}

	// Test flags exist
	flags := []string{"json", "greedy", "verbose", "fetch-HEAD", "quiet", "cask"}
	for _, flag := range flags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("Expected flag %s to exist", flag)
		}
	}
}

func TestRunOutdatedEmptyInstallation(t *testing.T) {
	logger.Init(false, false, true)

	// Create temp directory
	tempDir, err := os.MkdirTemp("", "outdated-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cfg := &config.Config{
		HomebrewCellar:  filepath.Join(tempDir, "Cellar"),
		HomebrewLibrary: filepath.Join(tempDir, "Library"),
	}

	// Create empty cellar directory
	os.MkdirAll(cfg.HomebrewCellar, 0755)
	os.MkdirAll(cfg.HomebrewLibrary, 0755)

	opts := &outdatedOptions{
		jsonOutput: false,
		greedy:     false,
		verbose:    false,
		fetchHead:  false,
		quiet:      false,
		cask:       false,
	}

	err = runOutdated(cfg, []string{}, opts)
	if err != nil {
		t.Errorf("runOutdated failed: %v", err)
	}
}

func TestRunOutdatedWithCask(t *testing.T) {
	logger.Init(false, false, true)

	cfg := &config.Config{
		HomebrewCellar:  "/tmp/test-cellar",
		HomebrewLibrary: "/tmp/test-library",
	}

	opts := &outdatedOptions{
		cask: true,
	}

	err := runOutdated(cfg, []string{}, opts)
	if err != nil {
		t.Errorf("runOutdated with cask failed: %v", err)
	}
}

func TestGetOutdatedFormulae(t *testing.T) {
	logger.Init(false, false, true)

	// Create temp directory structure
	tempDir, err := os.MkdirTemp("", "outdated-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cfg := &config.Config{
		HomebrewCellar:  filepath.Join(tempDir, "Cellar"),
		HomebrewLibrary: filepath.Join(tempDir, "Library"),
	}

	// Create cellar and library directories
	os.MkdirAll(cfg.HomebrewCellar, 0755)
	os.MkdirAll(cfg.HomebrewLibrary, 0755)

	opts := &outdatedOptions{
		verbose: true,
	}

	// Test with specific formula names
	outdated, err := getOutdatedFormulae(cfg, []string{"test-formula"}, opts)
	if err != nil {
		t.Errorf("getOutdatedFormulae failed: %v", err)
	}

	if outdated == nil {
		t.Error("Expected non-nil outdated list")
	}
}

func TestGetOutdatedCasks(t *testing.T) {
	logger.Init(false, false, true)
	cfg := &config.Config{}
	opts := &outdatedOptions{}

	outdated, err := getOutdatedCasks(cfg, []string{}, opts)
	if err != nil {
		t.Errorf("getOutdatedCasks failed: %v", err)
	}

	if len(outdated) != 0 {
		t.Errorf("Expected empty outdated casks list, got %d", len(outdated))
	}
}

func TestGetInstalledVersions(t *testing.T) {
	logger.Init(false, false, true)

	// Test with non-existent formula
	cfg := &config.Config{
		HomebrewCellar: "/non/existent/path",
	}

	versions, err := getInstalledVersions(cfg, "non-existent")
	if err != nil {
		t.Errorf("getInstalledVersions should not error for non-existent formula: %v", err)
	}
	if len(versions) != 0 {
		t.Errorf("Expected empty versions list, got %d", len(versions))
	}
}

func TestGetInstalledVersionsWithTemp(t *testing.T) {
	logger.Init(false, false, true)

	// Create temp directory structure
	tempDir, err := os.MkdirTemp("", "outdated-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cfg := &config.Config{
		HomebrewCellar: tempDir,
	}

	// Create formula with multiple versions
	formulaDir := filepath.Join(tempDir, "test-formula")
	os.MkdirAll(formulaDir, 0755)

	versions := []string{"1.0.0", "1.1.0", "2.0.0"}
	for _, version := range versions {
		versionDir := filepath.Join(formulaDir, version)
		os.MkdirAll(versionDir, 0755)
	}

	installedVersions, err := getInstalledVersions(cfg, "test-formula")
	if err != nil {
		t.Errorf("getInstalledVersions failed: %v", err)
	}

	if len(installedVersions) != len(versions) {
		t.Errorf("Expected %d versions, got %d", len(versions), len(installedVersions))
	}
}

func TestGetLatestVersion(t *testing.T) {
	logger.Init(false, false, true)

	// Test with empty list
	latest := getLatestVersion([]string{})
	if latest != "" {
		t.Errorf("Expected empty string for empty list, got %s", latest)
	}

	// Test with versions
	versions := []string{"1.0.0", "2.0.0", "1.5.0"}
	latest = getLatestVersion(versions)
	if latest != "2.0.0" {
		t.Errorf("Expected 2.0.0 as latest, got %s", latest)
	}
}

func TestIsVersionOutdated(t *testing.T) {
	logger.Init(false, false, true)

	// Test same versions
	if isVersionOutdated("1.0.0", "1.0.0") {
		t.Error("Expected same versions to not be outdated")
	}

	// Test different versions
	if !isVersionOutdated("1.0.0", "2.0.0") {
		t.Error("Expected different versions to be outdated")
	}

	// Test empty current version
	if isVersionOutdated("1.0.0", "") {
		t.Error("Expected empty current version to not be outdated")
	}
}

func TestIsPinned(t *testing.T) {
	logger.Init(false, false, true)

	// Create temp directory structure
	tempDir, err := os.MkdirTemp("", "outdated-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cfg := &config.Config{
		HomebrewLibrary: tempDir,
	}

	// Test unpinned formula
	if isPinned(cfg, "unpinned-formula") {
		t.Error("Expected unpinned formula to return false")
	}

	// Create pinned formula
	pinnedDir := filepath.Join(tempDir, "PinnedKegs")
	os.MkdirAll(pinnedDir, 0755)

	pinnedFile := filepath.Join(pinnedDir, "pinned-formula")
	os.WriteFile(pinnedFile, []byte(""), 0644)

	if !isPinned(cfg, "pinned-formula") {
		t.Error("Expected pinned formula to return true")
	}
}

func TestOutputJSON(t *testing.T) {
	logger.Init(false, false, true)

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	outdatedItems := []OutdatedInfo{
		{
			Name:              "test-formula",
			InstalledVersions: []string{"1.0.0"},
			CurrentVersion:    "2.0.0",
			Outdated:          true,
		},
		{
			Name:              "current-formula",
			InstalledVersions: []string{"1.0.0"},
			CurrentVersion:    "1.0.0",
			Outdated:          false,
		},
	}

	err := outputJSON(outdatedItems)

	// Restore stdout
	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Errorf("outputJSON failed: %v", err)
	}

	// Read captured output
	var buf bytes.Buffer
	buf.ReadFrom(r)

	// Parse JSON to verify structure
	var jsonOutput []OutdatedInfo
	err = json.Unmarshal(buf.Bytes(), &jsonOutput)
	if err != nil {
		t.Errorf("Failed to parse JSON output: %v", err)
	}

	// Should only include outdated items
	if len(jsonOutput) != 1 {
		t.Errorf("Expected 1 outdated item in JSON, got %d", len(jsonOutput))
	}

	if jsonOutput[0].Name != "test-formula" {
		t.Errorf("Expected test-formula in JSON output, got %s", jsonOutput[0].Name)
	}
}

func TestOutputText(t *testing.T) {
	logger.Init(false, false, true)

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	outdatedItems := []OutdatedInfo{
		{
			Name:              "test-formula",
			InstalledVersions: []string{"1.0.0"},
			CurrentVersion:    "2.0.0",
			Outdated:          true,
		},
		{
			Name:              "current-formula",
			InstalledVersions: []string{"1.0.0"},
			CurrentVersion:    "1.0.0",
			Outdated:          false,
		},
	}

	opts := &outdatedOptions{
		quiet:   false,
		verbose: false,
	}

	err := outputText(outdatedItems, opts)

	// Restore stdout
	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Errorf("outputText failed: %v", err)
	}

	// Read captured output
	var buf bytes.Buffer
	buf.ReadFrom(r)

	// Should contain the outdated formula
	if !strings.Contains(buf.String(), "test-formula") {
		t.Error("Expected output to contain test-formula")
	}

	// Should not contain the current formula
	if strings.Contains(buf.String(), "current-formula") {
		t.Error("Expected output to not contain current-formula")
	}
}

func TestOutputTextQuiet(t *testing.T) {
	logger.Init(false, false, true)

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	outdatedItems := []OutdatedInfo{
		{
			Name:              "test-formula",
			InstalledVersions: []string{"1.0.0"},
			CurrentVersion:    "2.0.0",
			Outdated:          true,
		},
	}

	opts := &outdatedOptions{
		quiet: true,
	}

	err := outputText(outdatedItems, opts)

	// Restore stdout
	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Errorf("outputText with quiet failed: %v", err)
	}

	// Read captured output
	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	// Should contain only the formula name
	if !strings.Contains(output, "test-formula") {
		t.Error("Expected quiet output to contain test-formula")
	}
}

func TestOutputTextVerbose(t *testing.T) {
	logger.Init(false, false, true)

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	outdatedItems := []OutdatedInfo{
		{
			Name:              "test-formula",
			InstalledVersions: []string{"1.0.0", "1.1.0"},
			CurrentVersion:    "2.0.0",
			PinnedVersion:     "1.1.0",
			Outdated:          true,
		},
	}

	opts := &outdatedOptions{
		verbose: true,
	}

	err := outputText(outdatedItems, opts)

	// Restore stdout
	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Errorf("outputText with verbose failed: %v", err)
	}

	// Read captured output
	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	// Should contain version information
	if !strings.Contains(output, "1.0.0, 1.1.0") {
		t.Error("Expected verbose output to contain installed versions")
	}
	if !strings.Contains(output, "pinned at") {
		t.Error("Expected verbose output to contain pinned information")
	}
}

func TestOutdatedOptions(t *testing.T) {
	opts := &outdatedOptions{
		jsonOutput: true,
		greedy:     true,
		verbose:    true,
		fetchHead:  true,
		quiet:      true,
		cask:       true,
	}

	if !opts.jsonOutput {
		t.Error("Expected jsonOutput to be true")
	}
	if !opts.verbose {
		t.Error("Expected verbose to be true")
	}
	if !opts.cask {
		t.Error("Expected cask to be true")
	}
}

func TestOutdatedInfo(t *testing.T) {
	info := OutdatedInfo{
		Name:              "test-formula",
		InstalledVersions: []string{"1.0.0"},
		CurrentVersion:    "2.0.0",
		PinnedVersion:     "1.0.0",
		Outdated:          true,
	}

	if info.Name != "test-formula" {
		t.Errorf("Expected Name to be test-formula, got %s", info.Name)
	}
	if !info.Outdated {
		t.Error("Expected Outdated to be true")
	}
}
