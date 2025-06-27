package cask

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	platformDarwin = "darwin"
	extensionDMG   = ".dmg"
)

// Cask represents a Homebrew cask for GUI applications
type Cask struct {
	Name        string           `json:"name"`
	Token       string           `json:"token"`
	FullName    string           `json:"full_name"`
	Homepage    string           `json:"homepage"`
	Description string           `json:"desc"`
	Version     string           `json:"version"`
	URL         []CaskURL        `json:"url"`
	Artifacts   []CaskArtifact   `json:"artifacts"`
	Depends     []CaskDependency `json:"depends_on"`
	Conflicts   []CaskConflict   `json:"conflicts_with"`
	AutoUpdates bool             `json:"auto_updates"`
	Container   *CaskContainer   `json:"container,omitempty"`
	Caveats     string           `json:"caveats,omitempty"`
	Languages   []string         `json:"languages,omitempty"`
	Sha256      string           `json:"sha256"`
	Appcast     *CaskAppcast     `json:"appcast,omitempty"`
	Tags        []string         `json:"tags,omitempty"`
	Deprecated  bool             `json:"deprecated"`
	Disabled    bool             `json:"disabled"`
	Ruby        string           `json:"ruby_source_path"`
	TapGitHead  string           `json:"tap_git_head"`
	InstallTime *time.Time       `json:"installed,omitempty"`
	InstalledBy string           `json:"installed_by,omitempty"`
}

// CaskURL represents download URLs for different versions/platforms
type CaskURL struct {
	URL      string                 `json:"url"`
	Branch   string                 `json:"branch,omitempty"`
	Tag      string                 `json:"tag,omitempty"`
	Revision string                 `json:"revision,omitempty"`
	Using    string                 `json:"using,omitempty"`
	Data     map[string]interface{} `json:"data,omitempty"`
}

// CaskArtifact represents what gets installed from a cask
type CaskArtifact struct {
	App         []CaskApp         `json:"app,omitempty"`
	Binary      []CaskBinary      `json:"binary,omitempty"`
	Manpage     []string          `json:"manpage,omitempty"`
	Pkg         []string          `json:"pkg,omitempty"`
	Installer   []CaskInstaller   `json:"installer,omitempty"`
	Suite       []CaskSuite       `json:"suite,omitempty"`
	Prefpane    []string          `json:"prefpane,omitempty"`
	Qlplugin    []string          `json:"qlplugin,omitempty"`
	Mdimporter  []string          `json:"mdimporter,omitempty"`
	Dictionary  []string          `json:"dictionary,omitempty"`
	Font        []string          `json:"font,omitempty"`
	Service     []string          `json:"service,omitempty"`
	Colorpicker []string          `json:"colorpicker,omitempty"`
	Vst         []string          `json:"vst,omitempty"`
	Vst3        []string          `json:"vst3,omitempty"`
	Au          []string          `json:"au,omitempty"`
	StageTarget []CaskStageTarget `json:"stage_only,omitempty"`
	Uninstall   []CaskUninstall   `json:"uninstall,omitempty"`
	Zap         []CaskZap         `json:"zap,omitempty"`
}

// CaskApp represents an application bundle
type CaskApp struct {
	Source string `json:"source"`
	Target string `json:"target,omitempty"`
}

// CaskBinary represents a command-line binary
type CaskBinary struct {
	Source string `json:"source"`
	Target string `json:"target,omitempty"`
}

// CaskInstaller represents a pkg installer
type CaskInstaller struct {
	Manual  string                 `json:"manual,omitempty"`
	Script  map[string]interface{} `json:"script,omitempty"`
	Allow   []string               `json:"allow_untrusted,omitempty"`
	Choices []CaskChoice           `json:"choices,omitempty"`
}

// CaskChoice represents installer choices
type CaskChoice struct {
	ChoiceIdentifier string `json:"choiceIdentifier"`
	ChoiceAttribute  string `json:"choiceAttribute"`
	AttributeSetting int    `json:"attributeSetting"`
}

// CaskSuite represents an application suite
type CaskSuite struct {
	Source string `json:"source"`
	Target string `json:"target,omitempty"`
}

// CaskStageTarget represents files to stage but not install
type CaskStageTarget struct {
	Source string `json:"source"`
	Target string `json:"target,omitempty"`
}

// CaskUninstall represents uninstall instructions
type CaskUninstall struct {
	Delete    []string               `json:"delete,omitempty"`
	Trash     []string               `json:"trash,omitempty"`
	Rmdir     []string               `json:"rmdir,omitempty"`
	Script    map[string]interface{} `json:"script,omitempty"`
	Pkgutil   []string               `json:"pkgutil,omitempty"`
	Signal    []CaskSignal           `json:"signal,omitempty"`
	LoginItem []string               `json:"login_item,omitempty"`
	Quit      []string               `json:"quit,omitempty"`
	Launchctl []string               `json:"launchctl,omitempty"`
	KillAll   []string               `json:"kext,omitempty"`
}

// CaskZap represents complete removal instructions
type CaskZap struct {
	Delete    []string     `json:"delete,omitempty"`
	Trash     []string     `json:"trash,omitempty"`
	Rmdir     []string     `json:"rmdir,omitempty"`
	Script    interface{}  `json:"script,omitempty"`
	Pkgutil   []string     `json:"pkgutil,omitempty"`
	Signal    []CaskSignal `json:"signal,omitempty"`
	LoginItem []string     `json:"login_item,omitempty"`
}

// CaskSignal represents process signals for uninstallation
type CaskSignal struct {
	Signal []string `json:"signal"`
	Pid    string   `json:"pid"`
}

// CaskDependency represents dependencies for casks
type CaskDependency struct {
	Macos   *CaskMacOSRequirement `json:"macos,omitempty"`
	Arch    []string              `json:"arch,omitempty"`
	Cask    []string              `json:"cask,omitempty"`
	Formula []string              `json:"formula,omitempty"`
}

// CaskMacOSRequirement represents macOS version requirements
type CaskMacOSRequirement struct {
	Minimum string `json:">=,omitempty"`
	Maximum string `json:"<=,omitempty"`
	Exact   string `json:"==,omitempty"`
}

// CaskConflict represents conflicts with other software
type CaskConflict struct {
	Cask    []string `json:"cask,omitempty"`
	Formula []string `json:"formula,omitempty"`
}

// CaskContainer represents container extraction settings
type CaskContainer struct {
	Type   string `json:"type,omitempty"`
	Nested string `json:"nested,omitempty"`
}

// CaskAppcast represents update checking information
type CaskAppcast struct {
	URL           string `json:"url"`
	Checkpoint    string `json:"checkpoint,omitempty"`
	MustContain   string `json:"must_contain,omitempty"`
	Configuration string `json:"configuration,omitempty"`
}

// GetDownloadURL returns the primary download URL for the cask
func (c *Cask) GetDownloadURL() string {
	if len(c.URL) > 0 {
		return c.URL[0].URL
	}
	return ""
}

// GetApplications returns all app artifacts
func (c *Cask) GetApplications() []CaskApp {
	return c.Artifacts[0].App
}

// GetBinaries returns all binary artifacts
func (c *Cask) GetBinaries() []CaskBinary {
	return c.Artifacts[0].Binary
}

// HasApplication checks if the cask contains any app artifacts
func (c *Cask) HasApplication() bool {
	if len(c.Artifacts) == 0 {
		return false
	}
	return len(c.Artifacts[0].App) > 0
}

// GetPrimaryAppName returns the name of the primary application
func (c *Cask) GetPrimaryAppName() string {
	if c.HasApplication() {
		apps := c.GetApplications()
		if len(apps) > 0 {
			appName := apps[0].Target
			if appName == "" {
				// Extract from source if target not specified
				appName = filepath.Base(apps[0].Source)
			}
			return appName
		}
	}
	return c.Name + ".app"
}

// IsCompatibleWithPlatform checks if the cask is compatible with current platform
func (c *Cask) IsCompatibleWithPlatform() bool {
	// Check architecture requirements
	if len(c.Depends) > 0 && len(c.Depends[0].Arch) > 0 {
		currentArch := runtime.GOARCH
		if currentArch == "amd64" {
			currentArch = "x86_64"
		}

		compatible := false
		for _, arch := range c.Depends[0].Arch {
			if arch == currentArch {
				compatible = true
				break
			}
		}
		if !compatible {
			return false
		}
	}

	// For now, assume macOS compatibility (casks are primarily for macOS)
	return runtime.GOOS == platformDarwin
}

// GetInstallPath returns the installation path for the cask
func (c *Cask) GetInstallPath(caskRoot string) string {
	return filepath.Join(caskRoot, c.Token)
}

// GetApplicationPath returns the path where applications are installed
func (c *Cask) GetApplicationPath() string {
	return "/Applications"
}

// NeedsSudo checks if the cask installation requires sudo privileges
func (c *Cask) NeedsSudo() bool {
	if len(c.Artifacts) == 0 {
		return false
	}

	artifacts := c.Artifacts[0]

	// Check for system-level installations
	if len(artifacts.Pkg) > 0 {
		return true
	}

	if len(artifacts.Installer) > 0 {
		return true
	}

	// Check for system directories
	systemPaths := []string{
		"/System/",
		"/usr/",
		"/Library/",
	}

	for _, app := range artifacts.App {
		target := app.Target
		if target == "" {
			target = c.GetApplicationPath()
		}

		for _, sysPath := range systemPaths {
			if strings.HasPrefix(target, sysPath) {
				return true
			}
		}
	}

	return false
}

// GetFileExtension returns the expected file extension for downloads
func (c *Cask) GetFileExtension() string {
	url := c.GetDownloadURL()
	if url == "" {
		return extensionDMG // Default for macOS
	}

	// Extract extension from URL
	if strings.Contains(url, extensionDMG) {
		return extensionDMG
	} else if strings.Contains(url, ".pkg") {
		return ".pkg"
	} else if strings.Contains(url, ".zip") {
		return ".zip"
	} else if strings.Contains(url, ".tar.gz") {
		return ".tar.gz"
	} else if strings.Contains(url, ".tar.bz2") {
		return ".tar.bz2"
	} else if strings.Contains(url, ".tar.xz") {
		return ".tar.xz"
	}

	return extensionDMG // Default
}

// GetCacheFileName returns the filename for caching downloads
func (c *Cask) GetCacheFileName() string {
	ext := c.GetFileExtension()
	if c.Version != "" {
		return fmt.Sprintf("%s-%s%s", c.Token, c.Version, ext)
	}
	return fmt.Sprintf("%s%s", c.Token, ext)
}

// IsInstalled checks if the cask is currently installed
func (c *Cask) IsInstalled() bool {
	return c.InstallTime != nil
}

// RequiresManualInstallation checks if manual installation steps are needed
func (c *Cask) RequiresManualInstallation() bool {
	if len(c.Artifacts) == 0 {
		return false
	}

	artifacts := c.Artifacts[0]

	// Check for manual installer
	for _, installer := range artifacts.Installer {
		if installer.Manual != "" {
			return true
		}
	}

	return false
}

// GetCaveats returns user-facing installation notes
func (c *Cask) GetCaveats() string {
	return c.Caveats
}

// Validate checks if the cask definition is valid
func (c *Cask) Validate() error {
	if c.Token == "" {
		return fmt.Errorf("cask token is required")
	}

	if c.Version == "" {
		return fmt.Errorf("cask version is required")
	}

	if len(c.URL) == 0 || c.URL[0].URL == "" {
		return fmt.Errorf("cask download URL is required")
	}

	if c.Sha256 == "" {
		return fmt.Errorf("cask SHA256 checksum is required")
	}

	if len(c.Artifacts) == 0 {
		return fmt.Errorf("cask must have at least one artifact")
	}

	return nil
}
