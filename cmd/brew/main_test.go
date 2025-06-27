package main

import (
	"testing"
)

func TestRun(t *testing.T) {
	// Since run() calls os.Exit on error, we can't easily test it
	// without modifying the function. We'll test this through integration tests instead.
	// This test exists to document that we're aware of the limitation.
	t.Skip("run() function tested via integration tests")
}

func TestMainFunction(t *testing.T) {
	// We can't easily test main() since it calls run() which may exit.
	// This test exists to document that we're aware of the limitation.
	t.Skip("main() function tested via integration tests")
}

func TestGlobalVariables(t *testing.T) {
	// Test that version variables exist and have default values
	if Version == "" {
		t.Error("Version should have a default value")
	}

	if GitCommit == "" {
		t.Error("GitCommit should have a default value")
	}

	if BuildDate == "" {
		t.Error("BuildDate should have a default value")
	}

	// Test that they have expected default values
	expectedDefaults := map[string]string{
		"Version":   "dev",
		"GitCommit": "unknown",
		"BuildDate": "unknown",
	}

	actualValues := map[string]string{
		"Version":   Version,
		"GitCommit": GitCommit,
		"BuildDate": BuildDate,
	}

	for key, expected := range expectedDefaults {
		if actual := actualValues[key]; actual != expected {
			t.Errorf("%s = %v, want %v", key, actual, expected)
		}
	}
}
