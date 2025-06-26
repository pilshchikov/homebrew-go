package formula

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/hashicorp/go-version"
	"gopkg.in/yaml.v3"
)

// Formula represents a Homebrew formula
type Formula struct {
	Name        string            `yaml:"name" json:"name"`
	Version     string            `yaml:"version" json:"version"`
	Homepage    string            `yaml:"homepage" json:"homepage"`
	Description string            `yaml:"desc" json:"desc"`
	License     string            `yaml:"license" json:"license"`
	URL         string            `yaml:"url" json:"url"`
	SHA256      string            `yaml:"sha256" json:"sha256"`
	Dependencies []string         `yaml:"dependencies,omitempty" json:"dependencies,omitempty"`
	BuildDependencies []string    `yaml:"build_dependencies,omitempty" json:"build_dependencies,omitempty"`
	TestDependencies []string     `yaml:"test_dependencies,omitempty" json:"test_dependencies,omitempty"`
	Options     []Option          `yaml:"options,omitempty" json:"options,omitempty"`
	Conflicts   []string          `yaml:"conflicts,omitempty" json:"conflicts,omitempty"`
	Caveats     string            `yaml:"caveats,omitempty" json:"caveats,omitempty"`
	KegOnly     bool              `yaml:"keg_only,omitempty" json:"keg_only,omitempty"`
	KegOnlyReason string          `yaml:"keg_only_reason,omitempty" json:"keg_only_reason,omitempty"`
	Pour        *PourBottle       `yaml:"pour_bottle,omitempty" json:"pour_bottle,omitempty"`
	Bottle      *Bottle           `yaml:"bottle,omitempty" json:"bottle,omitempty"`
	Head        *Head             `yaml:"head,omitempty" json:"head,omitempty"`
	Service     *Service          `yaml:"service,omitempty" json:"service,omitempty"`
	Livecheck   *Livecheck        `yaml:"livecheck,omitempty" json:"livecheck,omitempty"`
	Deprecated  bool              `yaml:"deprecated,omitempty" json:"deprecated,omitempty"`
	Disabled    bool              `yaml:"disabled,omitempty" json:"disabled,omitempty"`
	Requirements []Requirement    `yaml:"requirements,omitempty" json:"requirements,omitempty"`
	Patches     []Patch           `yaml:"patches,omitempty" json:"patches,omitempty"`
	Resources   []Resource        `yaml:"resources,omitempty" json:"resources,omitempty"`
	
	// Runtime information
	Tap         string    `yaml:"tap,omitempty" json:"tap,omitempty"`
	FullName    string    `yaml:"full_name,omitempty" json:"full_name,omitempty"`
	Path        string    `yaml:"path,omitempty" json:"path,omitempty"`
	Installed   bool      `yaml:"installed,omitempty" json:"installed,omitempty"`
	Linked      bool      `yaml:"linked,omitempty" json:"linked,omitempty"`
	Pinned      bool      `yaml:"pinned,omitempty" json:"pinned,omitempty"`
	Outdated    bool      `yaml:"outdated,omitempty" json:"outdated,omitempty"`
	CreatedAt   time.Time `yaml:"created_at,omitempty" json:"created_at,omitempty"`
	UpdatedAt   time.Time `yaml:"updated_at,omitempty" json:"updated_at,omitempty"`
}

// Option represents a formula option
type Option struct {
	Name        string `yaml:"name" json:"name"`
	Description string `yaml:"description" json:"description"`
	Default     bool   `yaml:"default,omitempty" json:"default,omitempty"`
}

// PourBottle represents bottle pouring configuration
type PourBottle struct {
	OnlyIf string `yaml:"only_if,omitempty" json:"only_if,omitempty"`
}

// Bottle represents a binary bottle
type Bottle struct {
	Stable *BottleSpec `yaml:"stable,omitempty" json:"stable,omitempty"`
	Head   *BottleSpec `yaml:"head,omitempty" json:"head,omitempty"`
}

// BottleSpec represents bottle specification
type BottleSpec struct {
	Rebuild int                    `yaml:"rebuild,omitempty" json:"rebuild,omitempty"`
	RootURL string                 `yaml:"root_url,omitempty" json:"root_url,omitempty"`
	Files   map[string]BottleFile  `yaml:"files" json:"files"`
}

// BottleFile represents a bottle file for a specific platform
type BottleFile struct {
	CellarPath string `yaml:"cellar,omitempty" json:"cellar,omitempty"`
	URL        string `yaml:"url" json:"url"`
	SHA256     string `yaml:"sha256" json:"sha256"`
}

// Head represents HEAD version information
type Head struct {
	URL    string `yaml:"url" json:"url"`
	Branch string `yaml:"branch,omitempty" json:"branch,omitempty"`
}

// Service represents a service configuration
type Service struct {
	Run          []string          `yaml:"run,omitempty" json:"run,omitempty"`
	RunType      string            `yaml:"run_type,omitempty" json:"run_type,omitempty"`
	Environment  map[string]string `yaml:"environment,omitempty" json:"environment,omitempty"`
	KeepAlive    bool              `yaml:"keep_alive,omitempty" json:"keep_alive,omitempty"`
	LogPath      string            `yaml:"log_path,omitempty" json:"log_path,omitempty"`
	ErrorLogPath string            `yaml:"error_log_path,omitempty" json:"error_log_path,omitempty"`
	WorkingDir   string            `yaml:"working_dir,omitempty" json:"working_dir,omitempty"`
}

// Livecheck represents livecheck configuration
type Livecheck struct {
	URL      string `yaml:"url,omitempty" json:"url,omitempty"`
	Regex    string `yaml:"regex,omitempty" json:"regex,omitempty"`
	Strategy string `yaml:"strategy,omitempty" json:"strategy,omitempty"`
	Skip     bool   `yaml:"skip,omitempty" json:"skip,omitempty"`
}

// Requirement represents a system requirement
type Requirement struct {
	Name    string `yaml:"name" json:"name"`
	Version string `yaml:"version,omitempty" json:"version,omitempty"`
	Cask    string `yaml:"cask,omitempty" json:"cask,omitempty"`
}

// Patch represents a patch to apply
type Patch struct {
	URL    string `yaml:"url,omitempty" json:"url,omitempty"`
	Data   string `yaml:"data,omitempty" json:"data,omitempty"`
	Strip  int    `yaml:"strip,omitempty" json:"strip,omitempty"`
}

// Resource represents an additional resource
type Resource struct {
	Name    string `yaml:"name" json:"name"`
	URL     string `yaml:"url" json:"url"`
	SHA256  string `yaml:"sha256" json:"sha256"`
	Version string `yaml:"version,omitempty" json:"version,omitempty"`
}

// IsValid checks if the formula is valid
func (f *Formula) IsValid() error {
	if f.Name == "" {
		return fmt.Errorf("formula name is required")
	}
	
	if f.Version == "" {
		return fmt.Errorf("formula version is required")
	}
	
	if f.URL == "" && f.Head == nil {
		return fmt.Errorf("formula must have either URL or HEAD")
	}
	
	if f.URL != "" && f.SHA256 == "" {
		return fmt.Errorf("formula with URL must have SHA256")
	}
	
	// Validate version format (skip for HEAD versions)
	if f.Version != "HEAD" {
		if _, err := version.NewVersion(f.Version); err != nil {
			return fmt.Errorf("invalid version format: %w", err)
		}
	}
	
	return nil
}

// GetFullName returns the full name including tap
func (f *Formula) GetFullName() string {
	if f.Tap != "" && f.Tap != "homebrew/core" {
		return f.Tap + "/" + f.Name
	}
	return f.Name
}

// GetCellarPath returns the cellar path for this formula
func (f *Formula) GetCellarPath(cellar string) string {
	return filepath.Join(cellar, f.Name, f.Version)
}

// GetInstallReceipt returns the install receipt path
func (f *Formula) GetInstallReceipt(cellar string) string {
	return filepath.Join(f.GetCellarPath(cellar), "INSTALL_RECEIPT.json")
}

// HasOption checks if the formula has a specific option
func (f *Formula) HasOption(name string) bool {
	for _, opt := range f.Options {
		if opt.Name == name {
			return true
		}
	}
	return false
}

// GetOption returns the option with the given name
func (f *Formula) GetOption(name string) *Option {
	for _, opt := range f.Options {
		if opt.Name == name {
			return &opt
		}
	}
	return nil
}

// GetDependencies returns all dependencies (including build dependencies)
func (f *Formula) GetDependencies(includeBuild bool) []string {
	deps := make([]string, len(f.Dependencies))
	copy(deps, f.Dependencies)
	
	if includeBuild {
		deps = append(deps, f.BuildDependencies...)
	}
	
	return deps
}

// GetBottleURL returns the bottle URL for the current platform
func (f *Formula) GetBottleURL(platform string) string {
	if f.Bottle == nil || f.Bottle.Stable == nil {
		return ""
	}
	
	if file, ok := f.Bottle.Stable.Files[platform]; ok {
		return file.URL
	}
	
	return ""
}

// GetBottleSHA256 returns the bottle SHA256 for the current platform
func (f *Formula) GetBottleSHA256(platform string) string {
	if f.Bottle == nil || f.Bottle.Stable == nil {
		return ""
	}
	
	if file, ok := f.Bottle.Stable.Files[platform]; ok {
		return file.SHA256
	}
	
	return ""
}

// HasBottle checks if the formula has a bottle for the current platform
func (f *Formula) HasBottle(platform string) bool {
	if f.Bottle == nil || f.Bottle.Stable == nil {
		return false
	}
	
	_, ok := f.Bottle.Stable.Files[platform]
	return ok
}

// IsHeadOnly checks if the formula is HEAD-only
func (f *Formula) IsHeadOnly() bool {
	return f.URL == "" && f.Head != nil
}

// IsStable checks if the formula has a stable version
func (f *Formula) IsStable() bool {
	return f.URL != "" && f.Version != ""
}

// ValidateName validates the formula name
func ValidateName(name string) error {
	// Formula names must be lowercase and can contain letters, numbers, and hyphens
	re := regexp.MustCompile(`^[a-z0-9][a-z0-9_-]*$`)
	if !re.MatchString(name) {
		return fmt.Errorf("invalid formula name: %s", name)
	}
	
	// Check for reserved names
	reserved := []string{"brew", "homebrew", "core", "cask", "test"}
	for _, r := range reserved {
		if name == r {
			return fmt.Errorf("formula name '%s' is reserved", name)
		}
	}
	
	return nil
}

// ParseFormula parses a formula from YAML data
func ParseFormula(data []byte) (*Formula, error) {
	var formula Formula
	if err := yaml.Unmarshal(data, &formula); err != nil {
		return nil, fmt.Errorf("failed to parse formula: %w", err)
	}
	
	if err := formula.IsValid(); err != nil {
		return nil, fmt.Errorf("invalid formula: %w", err)
	}
	
	return &formula, nil
}

// ToYAML converts the formula to YAML
func (f *Formula) ToYAML() ([]byte, error) {
	return yaml.Marshal(f)
}

// Compare compares two formulas by version
func (f *Formula) Compare(other *Formula) int {
	v1, err1 := version.NewVersion(f.Version)
	v2, err2 := version.NewVersion(other.Version)
	
	if err1 != nil || err2 != nil {
		return strings.Compare(f.Version, other.Version)
	}
	
	return v1.Compare(v2)
}

// IsNewer checks if this formula is newer than the other
func (f *Formula) IsNewer(other *Formula) bool {
	return f.Compare(other) > 0
}

// IsOlder checks if this formula is older than the other
func (f *Formula) IsOlder(other *Formula) bool {
	return f.Compare(other) < 0
}

// IsSameVersion checks if this formula has the same version as the other
func (f *Formula) IsSameVersion(other *Formula) bool {
	return f.Compare(other) == 0
}