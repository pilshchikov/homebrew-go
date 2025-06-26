package tap

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"

	"github.com/pilshchikov/homebrew-go/internal/config"
	"github.com/pilshchikov/homebrew-go/internal/formula"
	"github.com/pilshchikov/homebrew-go/internal/logger"
)

// Tap represents a Homebrew tap (third-party repository)
type Tap struct {
	Name        string `json:"name"`
	FullName    string `json:"full_name"`
	User        string `json:"user"`
	Repository  string `json:"repository"`
	Remote      string `json:"remote"`
	Path        string `json:"path"`
	Installed   bool   `json:"installed"`
	Official    bool   `json:"official"`
	Formulae    int    `json:"formulae_count"`
	Casks       int    `json:"casks_count"`
}

// Manager handles tap operations
type Manager struct {
	cfg *config.Config
}

// ProgressWriter implements io.Writer for git progress reporting
type ProgressWriter struct {
	prefix string
}

// Write implements io.Writer interface for progress reporting
func (pw *ProgressWriter) Write(p []byte) (n int, err error) {
	message := string(p)
	// Clean up git progress messages and log them
	if strings.TrimSpace(message) != "" {
		logger.Debug("%s: %s", pw.prefix, strings.TrimSpace(message))
	}
	return len(p), nil
}

// NewManager creates a new tap manager
func NewManager(cfg *config.Config) *Manager {
	return &Manager{cfg: cfg}
}

// ListTaps returns all installed taps
func (m *Manager) ListTaps() ([]*Tap, error) {
	tapsDir := filepath.Join(m.cfg.HomebrewRepository, "Library", "Taps")
	
	var taps []*Tap
	
	err := filepath.WalkDir(tapsDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		
		if !d.IsDir() {
			return nil
		}
		
		// Skip the root taps directory
		if path == tapsDir {
			return nil
		}
		
		// Check if this is a tap directory (contains formulae or casks)
		if m.isTapDirectory(path) {
			tap, err := m.loadTap(path)
			if err != nil {
				logger.Warn("Failed to load tap at %s: %v", path, err)
				return nil
			}
			taps = append(taps, tap)
		}
		
		return nil
	})
	
	if err != nil {
		return nil, fmt.Errorf("failed to walk taps directory: %w", err)
	}
	
	// Sort taps by name
	sort.Slice(taps, func(i, j int) bool {
		return taps[i].Name < taps[j].Name
	})
	
	return taps, nil
}

// GetTap returns a specific tap by name
func (m *Manager) GetTap(name string) (*Tap, error) {
	tapPath := m.getTapPath(name)
	
	if !m.isTapDirectory(tapPath) {
		return nil, fmt.Errorf("tap %s not found", name)
	}
	
	return m.loadTap(tapPath)
}

// AddTap adds (installs) a new tap
func (m *Manager) AddTap(name, remote string, options *TapOptions) error {
	if options == nil {
		options = &TapOptions{}
	}
	
	logger.Progress("Tapping %s", name)
	
	// Validate tap name
	if err := m.validateTapName(name); err != nil {
		return fmt.Errorf("invalid tap name: %w", err)
	}
	
	// Check if tap already exists
	if tap, _ := m.GetTap(name); tap != nil && tap.Installed {
		if !options.Force {
			return fmt.Errorf("tap %s already tapped", name)
		}
		logger.Info("Tap %s already exists, forcing re-tap", name)
	}
	
	// Determine remote URL
	if remote == "" {
		remote = m.getDefaultRemote(name)
	}
	
	tapPath := m.getTapPath(name)
	
	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(tapPath), 0755); err != nil {
		return fmt.Errorf("failed to create tap directory: %w", err)
	}
	
	// Clone the repository
	logger.Step("Cloning %s", remote)
	progressWriter := &ProgressWriter{prefix: fmt.Sprintf("Clone %s", name)}
	cloneOptions := &git.CloneOptions{
		URL:      remote,
		Progress: progressWriter,
	}
	
	if options.Shallow {
		cloneOptions.Depth = 1
	}
	
	if options.Branch != "" {
		cloneOptions.ReferenceName = plumbing.ReferenceName("refs/heads/" + options.Branch)
		cloneOptions.SingleBranch = true
	}
	
	repo, err := git.PlainClone(tapPath, false, cloneOptions)
	if err != nil {
		return fmt.Errorf("failed to clone tap: %w", err)
	}
	
	// Verify the tap
	if err := m.verifyTap(tapPath); err != nil {
		// Clean up on failure
		os.RemoveAll(tapPath)
		return fmt.Errorf("tap verification failed: %w", err)
	}
	
	logger.Success("Tapped %s (%d formulae)", name, m.countFormulae(tapPath))
	
	// Update tap info
	if !options.Quiet {
		tap, _ := m.loadTap(tapPath)
		if tap != nil {
			logger.Info("Tap info: %d formulae, %d casks", tap.Formulae, tap.Casks)
		}
	}
	
	_ = repo // TODO: Store repo reference if needed
	
	return nil
}

// RemoveTap removes (uninstalls) a tap
func (m *Manager) RemoveTap(name string, options *TapOptions) error {
	if options == nil {
		options = &TapOptions{}
	}
	
	logger.Progress("Untapping %s", name)
	
	// Check if tap exists
	tap, err := m.GetTap(name)
	if err != nil {
		return fmt.Errorf("tap %s not found", name)
	}
	
	if !tap.Installed {
		return fmt.Errorf("tap %s is not installed", name)
	}
	
	// Check for installed formulae from this tap
	if !options.Force {
		installedFormulae, err := m.getInstalledFormulaeFromTap(tap)
		if err != nil {
			return fmt.Errorf("failed to check installed formulae: %w", err)
		}
		
		if len(installedFormulae) > 0 {
			return fmt.Errorf("tap %s has installed formulae: %s\nUse --force to remove anyway",
				name, strings.Join(installedFormulae, ", "))
		}
	}
	
	// Remove the tap directory
	if err := os.RemoveAll(tap.Path); err != nil {
		return fmt.Errorf("failed to remove tap directory: %w", err)
	}
	
	logger.Success("Untapped %s", name)
	return nil
}

// UpdateTap updates a specific tap
func (m *Manager) UpdateTap(name string) error {
	logger.Progress("Updating tap %s", name)
	
	tap, err := m.GetTap(name)
	if err != nil {
		return fmt.Errorf("tap %s not found", name)
	}
	
	if !tap.Installed {
		return fmt.Errorf("tap %s is not installed", name)
	}
	
	// Open the git repository
	repo, err := git.PlainOpen(tap.Path)
	if err != nil {
		return fmt.Errorf("failed to open tap repository: %w", err)
	}
	
	// Get the working tree
	workTree, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get working tree: %w", err)
	}
	
	// Pull updates
	progressWriter := &ProgressWriter{prefix: fmt.Sprintf("Update %s", name)}
	err = workTree.Pull(&git.PullOptions{
		RemoteName: "origin",
		Progress:   progressWriter,
	})
	
	if err == git.NoErrAlreadyUpToDate {
		logger.Info("Tap %s is already up to date", name)
		return nil
	} else if err != nil {
		return fmt.Errorf("failed to update tap: %w", err)
	}
	
	logger.Success("Updated tap %s", name)
	return nil
}

// TapOptions contains options for tap operations
type TapOptions struct {
	Force   bool
	Quiet   bool
	Shallow bool
	Branch  string
}

func (m *Manager) getTapPath(name string) string {
	parts := strings.Split(name, "/")
	if len(parts) != 2 {
		// Default to homebrew org if not specified
		parts = []string{"homebrew", name}
	}
	
	return filepath.Join(m.cfg.HomebrewRepository, "Library", "Taps", 
		parts[0], "homebrew-"+parts[1])
}

func (m *Manager) validateTapName(name string) error {
	// Basic validation for tap names
	if name == "" {
		return fmt.Errorf("tap name cannot be empty")
	}
	
	if strings.Contains(name, " ") {
		return fmt.Errorf("tap name cannot contain spaces")
	}
	
	return nil
}

func (m *Manager) getDefaultRemote(name string) string {
	parts := strings.Split(name, "/")
	if len(parts) == 2 {
		return fmt.Sprintf("https://github.com/%s/homebrew-%s.git", parts[0], parts[1])
	}
	return fmt.Sprintf("https://github.com/homebrew/homebrew-%s.git", name)
}

func (m *Manager) isTapDirectory(path string) bool {
	// Check if directory contains Formula or Casks subdirectories
	formulaDir := filepath.Join(path, "Formula")
	casksDir := filepath.Join(path, "Casks")
	
	if _, err := os.Stat(formulaDir); err == nil {
		return true
	}
	
	if _, err := os.Stat(casksDir); err == nil {
		return true
	}
	
	return false
}

func (m *Manager) loadTap(path string) (*Tap, error) {
	// Extract tap name from path
	relPath, err := filepath.Rel(filepath.Join(m.cfg.HomebrewRepository, "Library", "Taps"), path)
	if err != nil {
		return nil, fmt.Errorf("failed to get relative path: %w", err)
	}
	
	parts := strings.Split(relPath, string(filepath.Separator))
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid tap path structure")
	}
	
	user := parts[0]
	repo := strings.TrimPrefix(parts[1], "homebrew-")
	name := user + "/" + repo
	
	tap := &Tap{
		Name:       name,
		FullName:   "homebrew/" + repo,
		User:       user,
		Repository: repo,
		Path:       path,
		Installed:  true,
		Official:   user == "homebrew",
		Formulae:   m.countFormulae(path),
		Casks:      m.countCasks(path),
	}
	
	// Try to get remote URL
	if remote := m.getRemoteURL(path); remote != "" {
		tap.Remote = remote
	}
	
	return tap, nil
}

func (m *Manager) countFormulae(tapPath string) int {
	formulaDir := filepath.Join(tapPath, "Formula")
	files, err := os.ReadDir(formulaDir)
	if err != nil {
		return 0
	}
	
	count := 0
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".rb") {
			count++
		}
	}
	
	return count
}

func (m *Manager) countCasks(tapPath string) int {
	casksDir := filepath.Join(tapPath, "Casks")
	files, err := os.ReadDir(casksDir)
	if err != nil {
		return 0
	}
	
	count := 0
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".rb") {
			count++
		}
	}
	
	return count
}

func (m *Manager) getRemoteURL(tapPath string) string {
	repo, err := git.PlainOpen(tapPath)
	if err != nil {
		return ""
	}
	
	config, err := repo.Config()
	if err != nil {
		return ""
	}
	
	if remote, ok := config.Remotes["origin"]; ok && len(remote.URLs) > 0 {
		return remote.URLs[0]
	}
	
	return ""
}

func (m *Manager) verifyTap(tapPath string) error {
	// Check if tap has either Formula or Casks directory
	formulaDir := filepath.Join(tapPath, "Formula")
	casksDir := filepath.Join(tapPath, "Casks")
	
	hasFormulae := false
	hasCasks := false
	
	if _, err := os.Stat(formulaDir); err == nil {
		hasFormulae = true
	}
	
	if _, err := os.Stat(casksDir); err == nil {
		hasCasks = true
	}
	
	if !hasFormulae && !hasCasks {
		return fmt.Errorf("tap does not contain Formula or Casks directories")
	}
	
	return nil
}

func (m *Manager) getInstalledFormulaeFromTap(tap *Tap) ([]string, error) {
	var installedFormulae []string
	
	// Get all formulae from this tap
	tapFormulae, err := tap.ListFormulae()
	if err != nil {
		return nil, fmt.Errorf("failed to list formulae from tap: %w", err)
	}
	
	// Check which ones are installed by looking in the cellar
	for _, formulaName := range tapFormulae {
		formulaDir := filepath.Join(m.cfg.HomebrewCellar, formulaName)
		if _, err := os.Stat(formulaDir); err == nil {
			// Formula is installed, verify it's from this tap by checking install receipt
			if m.isFormulaFromTap(formulaName, tap.Name) {
				installedFormulae = append(installedFormulae, formulaName)
			}
		}
	}
	
	return installedFormulae, nil
}

// isFormulaFromTap checks if an installed formula is from a specific tap
func (m *Manager) isFormulaFromTap(formulaName, tapName string) bool {
	// Use simple heuristic: check if the formula exists in the tap's Formula directory
	tapPath := m.getTapPath(tapName)
	formulaInTap := filepath.Join(tapPath, "Formula", formulaName+".rb")
	yamlInTap := filepath.Join(tapPath, "Formula", formulaName+".yaml")
	
	if _, err := os.Stat(formulaInTap); err == nil {
		return true
	}
	if _, err := os.Stat(yamlInTap); err == nil {
		return true
	}
	
	// TODO: For more accuracy, parse install receipt to check tap information
	// receiptPath := filepath.Join(m.cfg.HomebrewCellar, formulaName, "INSTALL_RECEIPT.json")
	
	return false
}

// GetFormula returns a formula from this tap
func (t *Tap) GetFormula(name string) (*formula.Formula, error) {
	formulaPath := filepath.Join(t.Path, "Formula", name+".rb")
	
	// For now, we'll look for YAML files instead of Ruby files
	// In a complete implementation, we'd parse Ruby DSL
	yamlPath := filepath.Join(t.Path, "Formula", name+".yaml")
	
	if _, err := os.Stat(yamlPath); err == nil {
		data, err := os.ReadFile(yamlPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read formula file: %w", err)
		}
		
		f, err := formula.ParseFormula(data)
		if err != nil {
			return nil, fmt.Errorf("failed to parse formula: %w", err)
		}
		
		f.Tap = t.Name
		f.Path = yamlPath
		
		return f, nil
	}
	
	// Check if Ruby file exists (for display purposes)
	if _, err := os.Stat(formulaPath); err == nil {
		return nil, fmt.Errorf("formula %s exists but Ruby DSL parsing not implemented yet", name)
	}
	
	return nil, fmt.Errorf("formula %s not found in tap %s", name, t.Name)
}

// ListFormulae returns all formulae in this tap
func (t *Tap) ListFormulae() ([]string, error) {
	formulaDir := filepath.Join(t.Path, "Formula")
	files, err := os.ReadDir(formulaDir)
	if err != nil {
		return nil, err
	}
	
	var formulae []string
	for _, file := range files {
		if !file.IsDir() && (strings.HasSuffix(file.Name(), ".rb") || strings.HasSuffix(file.Name(), ".yaml")) {
			name := strings.TrimSuffix(file.Name(), filepath.Ext(file.Name()))
			formulae = append(formulae, name)
		}
	}
	
	sort.Strings(formulae)
	return formulae, nil
}