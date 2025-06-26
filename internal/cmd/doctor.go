package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/pilshchikov/homebrew-go/internal/config"
	"github.com/spf13/cobra"
)

// NewDoctorCmd creates the doctor command
func NewDoctorCmd(cfg *config.Config) *cobra.Command {
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:     "doctor",
		Aliases: []string{"dr"},
		Short:   "Check your system for potential problems",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDoctor(cfg, jsonOutput)
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")

	return cmd
}

// DoctorResult represents the result of a doctor check
type DoctorResult struct {
	Status      string            `json:"status"`
	Checks      []DoctorCheck     `json:"checks"`
	Warnings    []string          `json:"warnings,omitempty"`
	Environment map[string]string `json:"environment"`
	HasIssues   bool              `json:"has_issues"`
	Message     string            `json:"message"`
}

// DoctorCheck represents a single doctor check
type DoctorCheck struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Status      string `json:"status"` // "ok", "warning", "error"
	Message     string `json:"message,omitempty"`
	Path        string `json:"path,omitempty"`
}

func runDoctor(cfg *config.Config, jsonOutput bool) error {
	result := &DoctorResult{
		Checks:      []DoctorCheck{},
		Warnings:    []string{},
		Environment: make(map[string]string),
		HasIssues:   false,
	}

	// Check if directories exist
	dirs := map[string]string{
		"HOMEBREW_PREFIX":   cfg.HomebrewPrefix,
		"HOMEBREW_CELLAR":   cfg.HomebrewCellar,
		"HOMEBREW_CASKROOM": cfg.HomebrewCaskroom,
		"HOMEBREW_CACHE":    cfg.HomebrewCache,
	}

	for name, path := range dirs {
		check := DoctorCheck{
			Name:        name,
			Description: "Directory existence check",
			Path:        path,
		}

		if _, err := os.Stat(path); os.IsNotExist(err) {
			check.Status = "warning"
			check.Message = "Directory does not exist"
			result.Warnings = append(result.Warnings, fmt.Sprintf("%s (%s) does not exist", name, path))
			result.HasIssues = true
		} else {
			check.Status = "ok"
			check.Message = "Directory exists"
		}

		result.Checks = append(result.Checks, check)
	}

	// Check permissions
	for name, path := range dirs {
		check := DoctorCheck{
			Name:        name + "_permissions",
			Description: "Directory permissions check",
			Path:        path,
		}

		if info, err := os.Stat(path); err == nil {
			if info.Mode().Perm()&0200 == 0 {
				check.Status = "warning"
				check.Message = "Directory is not writable"
				result.Warnings = append(result.Warnings, fmt.Sprintf("%s (%s) is not writable", name, path))
				result.HasIssues = true
			} else {
				check.Status = "ok"
				check.Message = "Directory is writable"
			}
		} else {
			check.Status = "error"
			check.Message = "Cannot check permissions (directory does not exist)"
		}

		result.Checks = append(result.Checks, check)
	}

	// Check for conflicting software
	macportsCheck := DoctorCheck{
		Name:        "macports_conflict",
		Description: "MacPorts conflict check",
		Path:        "/opt/local/bin/port",
	}

	if _, err := os.Stat("/opt/local/bin/port"); err == nil {
		macportsCheck.Status = "warning"
		macportsCheck.Message = "MacPorts is installed and may conflict with Homebrew"
		result.Warnings = append(result.Warnings, "You have MacPorts installed which may conflict with Homebrew")
		result.HasIssues = true
	} else {
		macportsCheck.Status = "ok"
		macportsCheck.Message = "No MacPorts installation detected"
	}

	result.Checks = append(result.Checks, macportsCheck)

	// Add environment information
	result.Environment["go_version"] = runtime.Version()
	result.Environment["platform"] = runtime.GOOS + "/" + runtime.GOARCH

	// Set overall status and message
	if result.HasIssues {
		result.Status = "warning"
		result.Message = "System has warnings that should be addressed"
	} else {
		result.Status = "ok"
		result.Message = "Your system is ready to brew!"
	}

	// Output results
	if jsonOutput {
		data, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal doctor results to JSON: %w", err)
		}
		fmt.Println(string(data))
	} else {
		// Traditional text output
		fmt.Printf("Please note that these warnings are just used to help the Homebrew maintainers\n")
		fmt.Printf("with debugging if you file an issue. If everything you use Homebrew for is\n")
		fmt.Printf("working fine: please don't worry or file an issue; just ignore this. Thanks!\n\n")

		fmt.Printf("==> Checking Homebrew directories\n")
		for _, check := range result.Checks {
			if strings.Contains(check.Name, "_permissions") {
				continue // Skip permissions in directory section
			}
			if strings.Contains(check.Name, "conflict") {
				continue // Skip conflicts in directory section
			}
			if check.Status == "warning" || check.Status == "error" {
				fmt.Printf("Warning: %s (%s) %s.\n", check.Name, check.Path, check.Message)
			}
		}

		fmt.Printf("==> Checking permissions\n")
		for _, check := range result.Checks {
			if !strings.Contains(check.Name, "_permissions") {
				continue // Only show permission checks
			}
			if check.Status == "warning" || check.Status == "error" {
				fmt.Printf("Warning: %s.\n", check.Message)
			}
		}

		fmt.Printf("==> Checking for conflicting software\n")
		for _, check := range result.Checks {
			if !strings.Contains(check.Name, "conflict") {
				continue // Only show conflict checks
			}
			if check.Status == "warning" || check.Status == "error" {
				fmt.Printf("Warning: %s.\n", check.Message)
			}
		}

		fmt.Printf("==> Checking Go environment\n")
		fmt.Printf("Go version: %s\n", result.Environment["go_version"])

		if !result.HasIssues {
			fmt.Printf("\n==> Your system is ready to brew! 🍺\n")
		} else {
			fmt.Printf("\n==> Please address the warnings above before continuing.\n")
		}
	}

	return nil
}
