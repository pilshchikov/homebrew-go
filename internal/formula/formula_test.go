package formula

import (
	"testing"
)

func TestFormulaValidation(t *testing.T) {
	tests := []struct {
		name    string
		formula Formula
		wantErr bool
	}{
		{
			name: "valid formula",
			formula: Formula{
				Name:    "test-formula",
				Version: "1.0.0",
				URL:     "https://example.com/test-1.0.0.tar.gz",
				SHA256:  "abcd1234",
			},
			wantErr: false,
		},
		{
			name: "missing name",
			formula: Formula{
				Version: "1.0.0",
				URL:     "https://example.com/test-1.0.0.tar.gz",
				SHA256:  "abcd1234",
			},
			wantErr: true,
		},
		{
			name: "missing version",
			formula: Formula{
				Name:   "test-formula",
				URL:    "https://example.com/test-1.0.0.tar.gz",
				SHA256: "abcd1234",
			},
			wantErr: true,
		},
		{
			name: "missing URL and HEAD",
			formula: Formula{
				Name:    "test-formula",
				Version: "1.0.0",
			},
			wantErr: true,
		},
		{
			name: "valid HEAD-only formula",
			formula: Formula{
				Name:    "test-formula",
				Version: "HEAD",
				Head: &Head{
					URL: "https://github.com/user/repo.git",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.formula.IsValid()
			if (err != nil) != tt.wantErr {
				t.Errorf("Formula.IsValid() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestFormulaComparison(t *testing.T) {
	f1 := &Formula{Name: "test", Version: "1.0.0"}
	f2 := &Formula{Name: "test", Version: "1.1.0"}
	f3 := &Formula{Name: "test", Version: "1.0.0"}

	if !f2.IsNewer(f1) {
		t.Error("f2 should be newer than f1")
	}

	if f1.IsNewer(f2) {
		t.Error("f1 should not be newer than f2")
	}

	if !f1.IsSameVersion(f3) {
		t.Error("f1 and f3 should have the same version")
	}
}

func TestValidateName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid name", "test-formula", false},
		{"valid with numbers", "test123", false},
		{"valid with underscores", "test_formula", false},
		{"invalid uppercase", "Test-Formula", true},
		{"invalid spaces", "test formula", true},
		{"invalid reserved", "brew", true},
		{"empty name", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateName(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateName() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetFullName(t *testing.T) {
	tests := []struct {
		formula  Formula
		expected string
	}{
		{
			Formula{Name: "wget", Tap: "homebrew/core"},
			"wget",
		},
		{
			Formula{Name: "custom", Tap: "user/repo"},
			"user/repo/custom",
		},
		{
			Formula{Name: "local", Tap: ""},
			"local",
		},
	}

	for _, tt := range tests {
		result := tt.formula.GetFullName()
		if result != tt.expected {
			t.Errorf("GetFullName() = %v, want %v", result, tt.expected)
		}
	}
}

func TestHasBottle(t *testing.T) {
	formula := Formula{
		Name:    "test",
		Version: "1.0.0",
		Bottle: &Bottle{
			Stable: &BottleSpec{
				Files: map[string]BottleFile{
					"monterey": {
						URL:    "https://example.com/test-1.0.0.monterey.bottle.tar.gz",
						SHA256: "abc123",
					},
				},
			},
		},
	}

	if !formula.HasBottle("monterey") {
		t.Error("Formula should have monterey bottle")
	}

	if formula.HasBottle("big_sur") {
		t.Error("Formula should not have big_sur bottle")
	}
}

func TestParseFormula(t *testing.T) {
	yamlData := `
name: test-formula
version: 1.0.0
homepage: https://example.com
desc: A test formula
url: https://example.com/test-1.0.0.tar.gz
sha256: abcd1234efgh5678
dependencies:
  - dependency1
  - dependency2
`

	formula, err := ParseFormula([]byte(yamlData))
	if err != nil {
		t.Fatalf("ParseFormula() error = %v", err)
	}

	if formula.Name != "test-formula" {
		t.Errorf("Name = %v, want test-formula", formula.Name)
	}

	if formula.Version != "1.0.0" {
		t.Errorf("Version = %v, want 1.0.0", formula.Version)
	}

	if len(formula.Dependencies) != 2 {
		t.Errorf("Dependencies count = %v, want 2", len(formula.Dependencies))
	}
}

func TestGetCellarPath(t *testing.T) {
	formula := Formula{
		Name:    "test-formula",
		Version: "1.0.0",
	}

	cellarRoot := "/opt/homebrew/Cellar"
	expected := "/opt/homebrew/Cellar/test-formula/1.0.0"
	result := formula.GetCellarPath(cellarRoot)

	if result != expected {
		t.Errorf("GetCellarPath() = %v, want %v", result, expected)
	}
}

func TestGetInstallReceipt(t *testing.T) {
	formula := Formula{
		Name:    "test-formula",
		Version: "1.0.0",
	}

	cellarRoot := "/opt/homebrew/Cellar"
	expected := "/opt/homebrew/Cellar/test-formula/1.0.0/INSTALL_RECEIPT.json"
	result := formula.GetInstallReceipt(cellarRoot)

	if result != expected {
		t.Errorf("GetInstallReceipt() = %v, want %v", result, expected)
	}
}

func TestHasOption(t *testing.T) {
	formula := Formula{
		Name:    "test-formula",
		Version: "1.0.0",
		Options: []Option{
			{Name: "with-ssl", Description: "Build with SSL support"},
			{Name: "with-docs", Description: "Build with documentation"},
		},
	}

	if !formula.HasOption("with-ssl") {
		t.Error("Formula should have 'with-ssl' option")
	}

	if !formula.HasOption("with-docs") {
		t.Error("Formula should have 'with-docs' option")
	}

	if formula.HasOption("with-debug") {
		t.Error("Formula should not have 'with-debug' option")
	}
}

func TestGetOption(t *testing.T) {
	formula := Formula{
		Name:    "test-formula",
		Version: "1.0.0",
		Options: []Option{
			{Name: "with-ssl", Description: "Build with SSL support"},
			{Name: "with-docs", Description: "Build with documentation"},
		},
	}

	option := formula.GetOption("with-ssl")
	if option == nil {
		t.Error("GetOption() should return option for 'with-ssl'")
	} else if option.Description != "Build with SSL support" {
		t.Errorf("GetOption().Description = %v, want 'Build with SSL support'", option.Description)
	}

	option = formula.GetOption("non-existent")
	if option != nil {
		t.Error("GetOption() should return nil for non-existent option")
	}
}

func TestGetDependencies(t *testing.T) {
	tests := []struct {
		name         string
		formula      Formula
		includeBuild bool
		expected     []string
	}{
		{
			name: "runtime dependencies only",
			formula: Formula{
				Dependencies:      []string{"dep1", "dep2"},
				BuildDependencies: []string{"build-dep1"},
			},
			includeBuild: false,
			expected:     []string{"dep1", "dep2"},
		},
		{
			name: "all dependencies",
			formula: Formula{
				Dependencies:      []string{"dep1", "dep2"},
				BuildDependencies: []string{"build-dep1"},
			},
			includeBuild: true,
			expected:     []string{"dep1", "dep2", "build-dep1"},
		},
		{
			name: "no dependencies",
			formula: Formula{
				Dependencies:      []string{},
				BuildDependencies: []string{},
			},
			includeBuild: false,
			expected:     []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.formula.GetDependencies(tt.includeBuild)
			if len(result) != len(tt.expected) {
				t.Errorf("GetDependencies() returned %d deps, want %d", len(result), len(tt.expected))
				return
			}
			for i, dep := range result {
				if dep != tt.expected[i] {
					t.Errorf("GetDependencies()[%d] = %v, want %v", i, dep, tt.expected[i])
				}
			}
		})
	}
}

func TestGetBottleURL(t *testing.T) {
	formula := Formula{
		Name:    "test",
		Version: "1.0.0",
		Bottle: &Bottle{
			Stable: &BottleSpec{
				Files: map[string]BottleFile{
					"monterey": {
						URL:    "https://example.com/test-1.0.0.monterey.bottle.tar.gz",
						SHA256: "abc123",
					},
					"big_sur": {
						URL:    "https://example.com/test-1.0.0.big_sur.bottle.tar.gz",
						SHA256: "def456",
					},
				},
			},
		},
	}

	url := formula.GetBottleURL("monterey")
	expected := "https://example.com/test-1.0.0.monterey.bottle.tar.gz"
	if url != expected {
		t.Errorf("GetBottleURL() = %v, want %v", url, expected)
	}

	url = formula.GetBottleURL("linux")
	if url != "" {
		t.Errorf("GetBottleURL() for non-existent platform should return empty string, got %v", url)
	}

	// Test formula without bottle
	formulaNoBottle := Formula{Name: "test", Version: "1.0.0"}
	url = formulaNoBottle.GetBottleURL("monterey")
	if url != "" {
		t.Errorf("GetBottleURL() for formula without bottle should return empty string, got %v", url)
	}
}

func TestGetBottleSHA256(t *testing.T) {
	formula := Formula{
		Name:    "test",
		Version: "1.0.0",
		Bottle: &Bottle{
			Stable: &BottleSpec{
				Files: map[string]BottleFile{
					"monterey": {
						URL:    "https://example.com/test-1.0.0.monterey.bottle.tar.gz",
						SHA256: "abc123",
					},
				},
			},
		},
	}

	sha256 := formula.GetBottleSHA256("monterey")
	expected := "abc123"
	if sha256 != expected {
		t.Errorf("GetBottleSHA256() = %v, want %v", sha256, expected)
	}

	sha256 = formula.GetBottleSHA256("linux")
	if sha256 != "" {
		t.Errorf("GetBottleSHA256() for non-existent platform should return empty string, got %v", sha256)
	}
}

func TestIsHeadOnly(t *testing.T) {
	tests := []struct {
		name     string
		formula  Formula
		expected bool
	}{
		{
			name: "HEAD-only formula",
			formula: Formula{
				Name:    "test",
				Version: "HEAD",
				Head: &Head{
					URL: "https://github.com/user/repo.git",
				},
			},
			expected: true,
		},
		{
			name: "stable formula",
			formula: Formula{
				Name:    "test",
				Version: "1.0.0",
				URL:     "https://example.com/test-1.0.0.tar.gz",
			},
			expected: false,
		},
		{
			name: "formula without HEAD",
			formula: Formula{
				Name:    "test",
				Version: "1.0.0",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.formula.IsHeadOnly()
			if result != tt.expected {
				t.Errorf("IsHeadOnly() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestIsStable(t *testing.T) {
	tests := []struct {
		name     string
		formula  Formula
		expected bool
	}{
		{
			name: "stable formula with URL",
			formula: Formula{
				Name:    "test",
				Version: "1.0.0",
				URL:     "https://example.com/test-1.0.0.tar.gz",
			},
			expected: true,
		},
		{
			name: "HEAD-only formula",
			formula: Formula{
				Name:    "test",
				Version: "HEAD",
				Head: &Head{
					URL: "https://github.com/user/repo.git",
				},
			},
			expected: false,
		},
		{
			name: "formula without URL or HEAD",
			formula: Formula{
				Name:    "test",
				Version: "1.0.0",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.formula.IsStable()
			if result != tt.expected {
				t.Errorf("IsStable() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestIsOlder(t *testing.T) {
	f1 := &Formula{Name: "test", Version: "1.0.0"}
	f2 := &Formula{Name: "test", Version: "1.1.0"}
	f3 := &Formula{Name: "test", Version: "1.0.0"}

	if !f1.IsOlder(f2) {
		t.Error("f1 should be older than f2")
	}

	if f2.IsOlder(f1) {
		t.Error("f2 should not be older than f1")
	}

	if f1.IsOlder(f3) {
		t.Error("f1 should not be older than f3 (same version)")
	}
}

func TestToYAML(t *testing.T) {
	formula := Formula{
		Name:         "test-formula",
		Version:      "1.0.0",
		Homepage:     "https://example.com",
		Description:  "A test formula",
		URL:          "https://example.com/test-1.0.0.tar.gz",
		SHA256:       "abcd1234",
		Dependencies: []string{"dep1", "dep2"},
	}

	yamlData, err := formula.ToYAML()
	if err != nil {
		t.Fatalf("ToYAML() error = %v", err)
	}

	// Parse the YAML back to verify it's correct
	parsedFormula, err := ParseFormula(yamlData)
	if err != nil {
		t.Fatalf("ParseFormula() error = %v", err)
	}

	if parsedFormula.Name != formula.Name {
		t.Errorf("ToYAML() -> ParseFormula() Name = %v, want %v", parsedFormula.Name, formula.Name)
	}

	if parsedFormula.Version != formula.Version {
		t.Errorf("ToYAML() -> ParseFormula() Version = %v, want %v", parsedFormula.Version, formula.Version)
	}

	if len(parsedFormula.Dependencies) != len(formula.Dependencies) {
		t.Errorf("ToYAML() -> ParseFormula() Dependencies count = %v, want %v", len(parsedFormula.Dependencies), len(formula.Dependencies))
	}
}
