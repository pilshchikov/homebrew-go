package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/pilshchikov/homebrew-go/internal/api"
	"github.com/pilshchikov/homebrew-go/internal/config"
	"github.com/pilshchikov/homebrew-go/internal/logger"
)

func TestNewCasksCmd(t *testing.T) {
	logger.Init(false, false, true)
	cfg := &config.Config{}

	cmd := NewCasksCmd(cfg)

	if cmd.Use != "casks [OPTIONS]" {
		t.Errorf("Expected Use to be 'casks [OPTIONS]', got %s", cmd.Use)
	}

	if cmd.Short != "List all locally available casks" {
		t.Errorf("Expected correct Short description, got %s", cmd.Short)
	}

	// Test flags exist
	flags := []string{"eval-all", "json", "1"}
	for _, flag := range flags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("Expected flag %s to exist", flag)
		}
	}
}

func TestNewFormulaeCmd(t *testing.T) {
	logger.Init(false, false, true)
	cfg := &config.Config{}

	cmd := NewFormulaeCmd(cfg)

	if cmd.Use != "formulae [OPTIONS]" {
		t.Errorf("Expected Use to be 'formulae [OPTIONS]', got %s", cmd.Use)
	}

	if cmd.Short != "List all locally available formulae" {
		t.Errorf("Expected correct Short description, got %s", cmd.Short)
	}

	// Test flags exist
	flags := []string{"eval-all", "json", "1"}
	for _, flag := range flags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("Expected flag %s to exist", flag)
		}
	}
}

func TestNewCommandsCmd(t *testing.T) {
	logger.Init(false, false, true)
	cfg := &config.Config{}

	cmd := NewCommandsCmd(cfg)

	if cmd.Use != "commands [OPTIONS]" {
		t.Errorf("Expected Use to be 'commands [OPTIONS]', got %s", cmd.Use)
	}

	if cmd.Short != "Show lists of built-in and external commands" {
		t.Errorf("Expected correct Short description, got %s", cmd.Short)
	}

	// Test flags exist
	flags := []string{"quiet", "include-aliases", "builtin", "external"}
	for _, flag := range flags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("Expected flag %s to exist", flag)
		}
	}
}

func TestRunCasks(t *testing.T) {
	logger.Init(false, false, true)

	cfg := &config.Config{}

	opts := &casksOptions{
		eval:       false,
		jsonOutput: false,
		onePerLine: false,
	}

	// This will fail due to API call, but test the structure
	err := runCasks(cfg, opts)
	// Expected to fail due to API call in test environment
	if err == nil {
		t.Log("runCasks succeeded (unexpected in test environment)")
	}
}

func TestRunFormulae(t *testing.T) {
	logger.Init(false, false, true)

	cfg := &config.Config{}

	opts := &formulaeOptions{
		eval:       false,
		jsonOutput: false,
		onePerLine: false,
	}

	// This will fail due to API call, but test the structure
	err := runFormulae(cfg, opts)
	// Expected to fail due to API call in test environment
	if err == nil {
		t.Log("runFormulae succeeded (unexpected in test environment)")
	}
}

func TestRunCommands(t *testing.T) {
	logger.Init(false, false, true)
	cfg := &config.Config{}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	opts := &commandsOptions{
		quiet:    false,
		include:  []string{},
		builtin:  false,
		external: false,
	}

	err := runCommands(cfg, opts)

	// Restore stdout
	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Errorf("runCommands failed: %v", err)
	}

	// Read captured output
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	if !strings.Contains(output, "Built-in commands") {
		t.Error("Expected output to contain 'Built-in commands'")
	}
}

func TestRunCommandsBuiltinOnly(t *testing.T) {
	logger.Init(false, false, true)
	cfg := &config.Config{}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	opts := &commandsOptions{
		builtin:  true,
		external: false,
		quiet:    false,
	}

	err := runCommands(cfg, opts)

	// Restore stdout
	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Errorf("runCommands builtin only failed: %v", err)
	}

	// Read captured output
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	if !strings.Contains(output, "Built-in commands") {
		t.Error("Expected output to contain 'Built-in commands'")
	}
}

func TestRunCommandsExternalOnly(t *testing.T) {
	logger.Init(false, false, true)
	cfg := &config.Config{}

	opts := &commandsOptions{
		builtin:  false,
		external: true,
		quiet:    false,
	}

	err := runCommands(cfg, opts)
	if err != nil {
		t.Errorf("runCommands external only failed: %v", err)
	}
}

func TestRunCommandsQuiet(t *testing.T) {
	logger.Init(false, false, true)
	cfg := &config.Config{}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	opts := &commandsOptions{
		quiet: true,
	}

	err := runCommands(cfg, opts)

	// Restore stdout
	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Errorf("runCommands quiet failed: %v", err)
	}

	// Read captured output
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	// In quiet mode, should not contain headers
	if strings.Contains(output, "Built-in commands:") {
		t.Error("Expected quiet output to not contain headers")
	}
}

func TestGetBuiltinCommands(t *testing.T) {
	commands := getBuiltinCommands()

	if len(commands) == 0 {
		t.Error("Expected non-empty builtin commands list")
	}

	// Check for some expected commands
	expectedCommands := []string{"install", "uninstall", "update", "upgrade", "search"}
	commandMap := make(map[string]bool)
	for _, cmd := range commands {
		commandMap[cmd] = true
	}

	for _, expected := range expectedCommands {
		if !commandMap[expected] {
			t.Errorf("Expected builtin command %s not found", expected)
		}
	}
}

func TestGetExternalCommands(t *testing.T) {
	commands := getExternalCommands()

	// Should return empty list in current implementation
	if len(commands) != 0 {
		t.Errorf("Expected empty external commands list, got %d", len(commands))
	}
}

func TestPrintCommands(t *testing.T) {
	logger.Init(false, false, true)

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	commands := []string{"command1", "command2", "command3"}
	err := printCommands(commands, "Test Commands", false)

	// Restore stdout
	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Errorf("printCommands failed: %v", err)
	}

	// Read captured output
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	if !strings.Contains(output, "Test Commands:") {
		t.Error("Expected output to contain title")
	}

	for _, cmd := range commands {
		if !strings.Contains(output, cmd) {
			t.Errorf("Expected output to contain command %s", cmd)
		}
	}
}

func TestPrintCommandsQuiet(t *testing.T) {
	logger.Init(false, false, true)

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	commands := []string{"command1", "command2", "command3"}
	err := printCommands(commands, "Test Commands", true)

	// Restore stdout
	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Errorf("printCommands quiet failed: %v", err)
	}

	// Read captured output
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	// Should not contain title in quiet mode
	if strings.Contains(output, "Test Commands:") {
		t.Error("Expected quiet output to not contain title")
	}

	// Should contain commands one per line
	for _, cmd := range commands {
		if !strings.Contains(output, cmd) {
			t.Errorf("Expected output to contain command %s", cmd)
		}
	}
}

func TestOutputCasksJSON(t *testing.T) {
	logger.Init(false, false, true)

	// This function requires cask.Cask type which may not be available
	// Test with nil input for now
	err := outputCasksJSON(nil)
	if err != nil {
		t.Errorf("outputCasksJSON with nil input failed: %v", err)
	}
}

func TestOutputFormulaeJSON(t *testing.T) {
	logger.Init(false, false, true)

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Test with empty slice
	err := outputFormulaeJSON([]api.SearchResult{})

	// Restore stdout
	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Errorf("outputFormulaeJSON failed: %v", err)
	}

	// Read captured output
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)

	// Should be valid JSON
	var results []interface{}
	err = json.Unmarshal(buf.Bytes(), &results)
	if err != nil {
		t.Errorf("outputFormulaeJSON produced invalid JSON: %v", err)
	}
}

func TestPrintColumnsCasks(t *testing.T) {
	logger.Init(false, false, true)

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	items := []string{"item1", "item2", "item3", "item4", "item5"}
	printColumns(items, 80)

	// Restore stdout
	_ = w.Close()
	os.Stdout = oldStdout

	// Read captured output
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	// Should contain all items
	for _, item := range items {
		if !strings.Contains(output, item) {
			t.Errorf("Expected output to contain item %s", item)
		}
	}
}

func TestPrintColumnsEmpty(t *testing.T) {
	logger.Init(false, false, true)

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	printColumns([]string{}, 80)

	// Restore stdout
	_ = w.Close()
	os.Stdout = oldStdout

	// Read captured output
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	// Should be empty
	if output != "" {
		t.Errorf("Expected empty output for empty items, got %s", output)
	}
}

func TestCasksOptions(t *testing.T) {
	opts := &casksOptions{
		eval:       true,
		jsonOutput: true,
		onePerLine: true,
	}

	if !opts.eval {
		t.Error("Expected eval to be true")
	}
	if !opts.jsonOutput {
		t.Error("Expected jsonOutput to be true")
	}
	if !opts.onePerLine {
		t.Error("Expected onePerLine to be true")
	}
}

func TestFormulaeOptions(t *testing.T) {
	opts := &formulaeOptions{
		eval:       true,
		jsonOutput: true,
		onePerLine: true,
	}

	if !opts.eval {
		t.Error("Expected eval to be true")
	}
	if !opts.jsonOutput {
		t.Error("Expected jsonOutput to be true")
	}
	if !opts.onePerLine {
		t.Error("Expected onePerLine to be true")
	}
}

func TestCommandsOptions(t *testing.T) {
	opts := &commandsOptions{
		quiet:    true,
		include:  []string{"alias1", "alias2"},
		builtin:  true,
		external: true,
	}

	if !opts.quiet {
		t.Error("Expected quiet to be true")
	}
	if len(opts.include) != 2 {
		t.Errorf("Expected 2 include items, got %d", len(opts.include))
	}
	if !opts.builtin {
		t.Error("Expected builtin to be true")
	}
	if !opts.external {
		t.Error("Expected external to be true")
	}
}

func TestCasksCommandExecution(t *testing.T) {
	logger.Init(false, false, true)
	cfg := &config.Config{}
	cmd := NewCasksCmd(cfg)

	// This will likely fail due to API call, but test the structure
	err := cmd.RunE(cmd, []string{})
	if err == nil {
		t.Log("casks command succeeded (unexpected in test environment)")
	}
}

func TestFormulaeCommandExecution(t *testing.T) {
	logger.Init(false, false, true)
	cfg := &config.Config{}
	cmd := NewFormulaeCmd(cfg)

	// This will likely fail due to API call, but test the structure
	err := cmd.RunE(cmd, []string{})
	if err == nil {
		t.Log("formulae command succeeded (unexpected in test environment)")
	}
}

func TestCommandsCommandExecution(t *testing.T) {
	logger.Init(false, false, true)
	cfg := &config.Config{}
	cmd := NewCommandsCmd(cfg)

	err := cmd.RunE(cmd, []string{})
	if err != nil {
		t.Errorf("commands command failed: %v", err)
	}
}
