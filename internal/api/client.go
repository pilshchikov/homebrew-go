package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/pilshchikov/homebrew-go/internal/config"
	"github.com/pilshchikov/homebrew-go/internal/cask"
	"github.com/pilshchikov/homebrew-go/internal/formula"
	"github.com/pilshchikov/homebrew-go/internal/logger"
	"github.com/pilshchikov/homebrew-go/internal/utils"
)

// Client handles API requests to Homebrew's official endpoints
type Client struct {
	config     *config.Config
	httpClient *http.Client
	apiDomain  string
	userAgent  string
}

// FormulaAPIResponse represents the API response for a formula
type FormulaAPIResponse struct {
	Name                string                 `json:"name"`
	FullName            string                 `json:"full_name"`
	Tap                 string                 `json:"tap"`
	Oldname             string                 `json:"oldname,omitempty"`
	Aliases             []string               `json:"aliases"`
	VersionedFormulae   []string               `json:"versioned_formulae"`
	Desc                string                 `json:"desc"`
	License             string                 `json:"license"`
	Homepage            string                 `json:"homepage"`
	Versions            map[string]interface{} `json:"versions"`
	Urls                map[string]interface{} `json:"urls"`
	Revision            int                    `json:"revision"`
	VersionScheme       int                    `json:"version_scheme"`
	Bottle              map[string]interface{} `json:"bottle"`
	KegOnly             bool                   `json:"keg_only"`
	KegOnlyReason       map[string]string      `json:"keg_only_reason,omitempty"`
	Options             []interface{}          `json:"options"`
	BuildDependencies   []string               `json:"build_dependencies"`
	Dependencies        []string               `json:"dependencies"`
	TestDependencies    []string               `json:"test_dependencies"`
	RecommendedDependencies []string           `json:"recommended_dependencies"`
	OptionalDependencies    []string           `json:"optional_dependencies"`
	UsesFromMacos       []interface{}          `json:"uses_from_macos"`
	Requirements        []interface{}          `json:"requirements"`
	ConflictsWith       []string               `json:"conflicts_with"`
	ConflictsWithReasons []string              `json:"conflicts_with_reasons"`
	LinkOverwrite       []string               `json:"link_overwrite"`
	Caveats             string                 `json:"caveats,omitempty"`
	Installed           []interface{}          `json:"installed"`
	LinkedKeg           string                 `json:"linked_keg,omitempty"`
	Pinned              bool                   `json:"pinned"`
	Outdated            bool                   `json:"outdated"`
	Deprecated          bool                   `json:"deprecated"`
	DeprecationDate     string                 `json:"deprecation_date,omitempty"`
	DeprecationReason   string                 `json:"deprecation_reason,omitempty"`
	Disabled            bool                   `json:"disabled"`
	DisableDate         string                 `json:"disable_date,omitempty"`
	DisableReason       string                 `json:"disable_reason,omitempty"`
	PostInstallDefined  bool                   `json:"post_install_defined"`
	Service             map[string]interface{} `json:"service,omitempty"`
	TapGitHead          string                 `json:"tap_git_head"`
	RubySourcePath      string                 `json:"ruby_source_path"`
	RubySourceChecksum  map[string]string      `json:"ruby_source_checksum"`
}

// SearchResult represents a search result
type SearchResult struct {
	Name        string `json:"name"`
	FullName    string `json:"full_name"`
	Tap         string `json:"tap"`
	Desc        string `json:"desc"`
	Homepage    string `json:"homepage"`
	Deprecated  bool   `json:"deprecated"`
	Disabled    bool   `json:"disabled"`
}

// NewClient creates a new API client
func NewClient(cfg *config.Config) *Client {
	apiDomain := os.Getenv("HOMEBREW_API_DOMAIN")
	if apiDomain == "" {
		apiDomain = "https://formulae.brew.sh/api"
	}

	userAgent := fmt.Sprintf("Homebrew-Go/3.0.0 (%s; %s) Go/%s", 
		runtime.GOOS, runtime.GOARCH, "1.20")

	return &Client{
		config: cfg,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		apiDomain: apiDomain,
		userAgent: userAgent,
	}
}

// GetFormula fetches formula data from the API
func (c *Client) GetFormula(name string) (*formula.Formula, error) {
	logger.Debug("Fetching formula %s from API", name)
	
	url := fmt.Sprintf("%s/formula/%s.json", c.apiDomain, name)
	
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "application/json")
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch formula: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode == 404 {
		return nil, fmt.Errorf("formula %s not found", name)
	}
	
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("API request failed with status %d", resp.StatusCode)
	}
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}
	
	var apiResponse FormulaAPIResponse
	if err := json.Unmarshal(body, &apiResponse); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}
	
	// Convert API response to our Formula struct
	f := &formula.Formula{
		Name:        apiResponse.Name,
		FullName:    apiResponse.FullName,
		Description: apiResponse.Desc,
		Homepage:    apiResponse.Homepage,
		License:     apiResponse.License,
		Dependencies: apiResponse.Dependencies,
		BuildDependencies: apiResponse.BuildDependencies,
		TestDependencies:  apiResponse.TestDependencies,
		Caveats:     apiResponse.Caveats,
		KegOnly:     apiResponse.KegOnly,
		Deprecated:  apiResponse.Deprecated,
		Disabled:    apiResponse.Disabled,
	}
	
	// Extract version information
	if versions, ok := apiResponse.Versions["stable"].(string); ok {
		f.Version = versions
	}
	
	// Extract URL information
	if urls, ok := apiResponse.Urls["stable"].(map[string]interface{}); ok {
		if url, ok := urls["url"].(string); ok {
			f.URL = url
		}
		if sha256, ok := urls["checksum"].(string); ok {
			f.SHA256 = sha256
		}
	}
	
	// Extract bottle information
	if bottle, ok := apiResponse.Bottle["stable"].(map[string]interface{}); ok {
		if files, ok := bottle["files"].(map[string]interface{}); ok {
			f.Bottle = &formula.Bottle{
				Stable: &formula.BottleSpec{
					Rebuild: 0,
					Files:   make(map[string]formula.BottleFile),
				},
			}
			
			for platform, fileInfo := range files {
				if fileData, ok := fileInfo.(map[string]interface{}); ok {
					bottleFile := formula.BottleFile{}
					if url, ok := fileData["url"].(string); ok {
						bottleFile.URL = url
					}
					if sha256, ok := fileData["sha256"].(string); ok {
						bottleFile.SHA256 = sha256
					}
					f.Bottle.Stable.Files[platform] = bottleFile
				}
			}
		}
	}
	
	logger.Debug("Successfully fetched formula %s", name)
	return f, nil
}

// SearchFormulae searches for formulae by name or description
func (c *Client) SearchFormulae(query string) ([]SearchResult, error) {
	logger.Debug("Searching formulae for: %s", query)
	
	// For now, we'll fetch all formulae and filter locally
	// In a production implementation, we'd use a dedicated search endpoint
	formulaeList, err := c.listAllFormulae()
	if err != nil {
		return nil, fmt.Errorf("failed to get formulae list: %w", err)
	}
	
	var results []SearchResult
	query = strings.ToLower(query)
	
	for _, formulaName := range formulaeList {
		if strings.Contains(strings.ToLower(formulaName), query) {
			// Fetch detailed info for matching formulae
			if len(results) < 20 { // Limit results
				if formula, err := c.GetFormula(formulaName); err == nil {
					result := SearchResult{
						Name:       formula.Name,
						FullName:   formula.FullName,
						Desc:       formula.Description,
						Homepage:   formula.Homepage,
						Deprecated: formula.Deprecated,
						Disabled:   formula.Disabled,
					}
					results = append(results, result)
				}
			}
		}
	}
	
	logger.Debug("Found %d formulae matching '%s'", len(results), query)
	return results, nil
}

// listAllFormulae gets the list of all available formulae
func (c *Client) listAllFormulae() ([]string, error) {
	// Check cache first
	cacheFile := filepath.Join(c.config.HomebrewCache, "api", "formula_names.txt")
	if c.isCacheValid(cacheFile) {
		if names, err := c.readCachedNames(cacheFile); err == nil {
			return names, nil
		}
	}
	
	// Fetch from API
	url := fmt.Sprintf("%s/formula.json", c.apiDomain)
	
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "application/json")
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch formulae list: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("API request failed with status %d", resp.StatusCode)
	}
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}
	
	var formulae []map[string]interface{}
	if err := json.Unmarshal(body, &formulae); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}
	
	var names []string
	for _, f := range formulae {
		if name, ok := f["name"].(string); ok {
			names = append(names, name)
		}
	}
	
	// Cache the results
	c.cacheNames(cacheFile, names)
	
	return names, nil
}

// isCacheValid checks if the cache file is recent enough
func (c *Client) isCacheValid(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	
	// Cache is valid for 1 hour
	return time.Since(info.ModTime()) < time.Hour
}

// readCachedNames reads formulae names from cache
func (c *Client) readCachedNames(filename string) ([]string, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	
	names := strings.Split(strings.TrimSpace(string(data)), "\n")
	return names, nil
}

// cacheNames saves formulae names to cache
func (c *Client) cacheNames(filename string, names []string) {
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		logger.Warn("Failed to create cache directory: %v", err)
		return
	}
	
	data := strings.Join(names, "\n")
	if err := os.WriteFile(filename, []byte(data), 0644); err != nil {
		logger.Warn("Failed to cache formulae names: %v", err)
	}
}

// DownloadBottle downloads a bottle file
func (c *Client) DownloadBottle(formula *formula.Formula, platform string) (string, error) {
	if formula.Bottle == nil || formula.Bottle.Stable == nil {
		return "", fmt.Errorf("no bottle available for %s", formula.Name)
	}
	
	bottleFile, exists := formula.Bottle.Stable.Files[platform]
	if !exists {
		return "", fmt.Errorf("no bottle available for platform %s", platform)
	}
	
	logger.Progress("Downloading bottle for %s", formula.Name)
	
	// Create download directory
	downloadDir := filepath.Join(c.config.HomebrewCache, "downloads")
	if err := os.MkdirAll(downloadDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create download directory: %w", err)
	}
	
	// Generate filename
	filename := fmt.Sprintf("%s-%s.%s.bottle.tar.gz", 
		formula.Name, formula.Version, platform)
	filepath := filepath.Join(downloadDir, filename)
	
	// Check if already downloaded and verified
	if c.isFileValid(filepath, bottleFile.SHA256) {
		logger.Debug("Using cached bottle: %s", filename)
		return filepath, nil
	}
	
	// Download the bottle
	req, err := http.NewRequest("GET", bottleFile.URL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("User-Agent", c.userAgent)
	
	// Add authentication for GitHub Container Registry if needed
	if strings.Contains(bottleFile.URL, "ghcr.io") {
		if err := c.addGHCRAuth(req); err != nil {
			logger.Debug("GHCR authentication failed: %v", err)
			// Continue without auth - bottles should be public
		}
	}
	
	// Attempt download with retry logic for authentication issues
	resp, err := c.downloadWithRetry(req, bottleFile.URL)
	if err != nil {
		return "", fmt.Errorf("failed to download bottle: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("download failed with status %d for %s", resp.StatusCode, bottleFile.URL)
	}
	
	// Save to file
	file, err := os.Create(filepath)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()
	
	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to save bottle: %w", err)
	}
	
	// Verify checksum
	if !c.isFileValid(filepath, bottleFile.SHA256) {
		os.Remove(filepath)
		return "", fmt.Errorf("bottle checksum verification failed")
	}
	
	logger.Success("Downloaded bottle: %s", filename)
	return filepath, nil
}

// isFileValid checks if a file exists and has the correct checksum
func (c *Client) isFileValid(filepath, expectedSHA256 string) bool {
	if _, err := os.Stat(filepath); os.IsNotExist(err) {
		return false
	}
	
	if expectedSHA256 == "" {
		return true // No checksum to verify
	}
	
	// Verify SHA256 checksum
	return utils.VerifySHA256(filepath, expectedSHA256) == nil
}

// addGHCRAuth adds authentication for GitHub Container Registry
func (c *Client) addGHCRAuth(req *http.Request) error {
	// First, try to use personal access token from environment if available
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		logger.Debug("Using GitHub token from environment")
		req.Header.Set("Authorization", "Bearer "+token)
		return nil
	}
	
	// For public repositories, try to get an anonymous token
	// This follows the Docker Registry v2 authentication flow
	authURL := "https://ghcr.io/token"
	
	// Extract repository from the request URL to build proper scope
	repository := "homebrew/core" // Default for Homebrew bottles
	if strings.Contains(req.URL.Path, "homebrew") {
		// Try to extract more specific repository info if needed
		parts := strings.Split(req.URL.Path, "/")
		if len(parts) >= 3 {
			repository = strings.Join(parts[1:3], "/")
		}
	}
	
	scope := fmt.Sprintf("repository:%s:pull", repository)
	tokenURL := fmt.Sprintf("%s?service=ghcr.io&scope=%s", authURL, scope)
	
	logger.Debug("Requesting GHCR token for scope: %s", scope)
	
	tokenReq, err := http.NewRequest("GET", tokenURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create token request: %w", err)
	}
	
	// Set appropriate headers for token request
	tokenReq.Header.Set("User-Agent", c.userAgent)
	tokenReq.Header.Set("Accept", "application/json")
	
	tokenResp, err := c.httpClient.Do(tokenReq)
	if err != nil {
		return fmt.Errorf("failed to get GHCR token: %w", err)
	}
	defer tokenResp.Body.Close()
	
	if tokenResp.StatusCode != 200 {
		// Log the error but don't fail - some public repos might work without auth
		logger.Debug("GHCR token request failed with status %d, continuing without auth", tokenResp.StatusCode)
		return nil
	}
	
	body, err := io.ReadAll(tokenResp.Body)
	if err != nil {
		logger.Debug("Failed to read GHCR token response: %v", err)
		return nil // Continue without auth
	}
	
	var tokenResponse struct {
		Token        string `json:"token"`
		AccessToken  string `json:"access_token"`
		ExpiresIn    int    `json:"expires_in"`
		IssuedAt     string `json:"issued_at"`
	}
	
	if err := json.Unmarshal(body, &tokenResponse); err != nil {
		logger.Debug("Failed to parse GHCR token response: %v", err)
		return nil // Continue without auth
	}
	
	// Use either token or access_token field
	token := tokenResponse.Token
	if token == "" {
		token = tokenResponse.AccessToken
	}
	
	if token != "" {
		logger.Debug("Successfully obtained GHCR token")
		req.Header.Set("Authorization", "Bearer "+token)
	} else {
		logger.Debug("No token received from GHCR, continuing without auth")
	}
	
	return nil
}

// downloadWithRetry attempts to download with retry logic for authentication issues
func (c *Client) downloadWithRetry(req *http.Request, url string) (*http.Response, error) {
	maxRetries := 2
	
	for attempt := 0; attempt <= maxRetries; attempt++ {
		// Clone the request for retry attempts
		reqClone := req.Clone(req.Context())
		
		resp, err := c.httpClient.Do(reqClone)
		if err != nil {
			if attempt == maxRetries {
				return nil, err
			}
			logger.Debug("Download attempt %d failed: %v, retrying...", attempt+1, err)
			continue
		}
		
		// If we get a 401/403 and this is a GHCR URL, try to refresh auth
		if (resp.StatusCode == 401 || resp.StatusCode == 403) && strings.Contains(url, "ghcr.io") && attempt < maxRetries {
			resp.Body.Close()
			logger.Debug("Authentication failed (status %d), refreshing token and retrying...", resp.StatusCode)
			
			// Clear any existing auth header and re-authenticate
			reqClone.Header.Del("Authorization")
			if err := c.addGHCRAuth(reqClone); err != nil {
				logger.Debug("Failed to refresh GHCR auth: %v", err)
			}
			continue
		}
		
		// Success or non-auth related error
		return resp, nil
	}
	
	return nil, fmt.Errorf("download failed after %d attempts", maxRetries+1)
}

// GetPlatformTag returns the platform tag for bottle selection
func (c *Client) GetPlatformTag() string {
	// This should match Homebrew's platform detection logic
	switch runtime.GOOS {
	case "darwin":
		if runtime.GOARCH == "arm64" {
			return "arm64_sequoia" // Latest macOS version
		}
		return "x86_64_sequoia"
	case "linux":
		if runtime.GOARCH == "arm64" {
			return "arm64_linux"
		}
		return "x86_64_linux"
	default:
		return runtime.GOOS + "_" + runtime.GOARCH
	}
}

// GetCask fetches a specific cask by name from the API
func (c *Client) GetCask(name string) (*cask.Cask, error) {
	url := fmt.Sprintf("%s/cask/%s.json", c.apiDomain, name)
	
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch cask: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("cask '%s' not found", name)
	}
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed: %s", resp.Status)
	}
	
	var apiResponse map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		return nil, fmt.Errorf("failed to decode API response: %w", err)
	}
	
	return c.parseCaskFromAPI(apiResponse)
}

// SearchCasks searches for casks matching the given query
func (c *Client) SearchCasks(query string) ([]*cask.Cask, error) {
	// For now, use a simple approach - in practice this would use dedicated search endpoints
	url := fmt.Sprintf("%s/cask.json", c.apiDomain)
	
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to search casks: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed: %s", resp.Status)
	}
	
	var caskList []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&caskList); err != nil {
		return nil, fmt.Errorf("failed to decode search response: %w", err)
	}
	
	var results []*cask.Cask
	queryLower := strings.ToLower(query)
	
	for _, caskData := range caskList {
		// Basic search - check if query matches token or name
		if token, ok := caskData["token"].(string); ok {
			if strings.Contains(strings.ToLower(token), queryLower) {
				if c, err := c.parseCaskFromAPI(caskData); err == nil {
					results = append(results, c)
				}
			}
		}
		
		if name, ok := caskData["name"].(string); ok {
			if strings.Contains(strings.ToLower(name), queryLower) {
				if c, err := c.parseCaskFromAPI(caskData); err == nil {
					results = append(results, c)
				}
			}
		}
		
		// Limit results to avoid too many matches
		if len(results) >= 50 {
			break
		}
	}
	
	return results, nil
}

// parseCaskFromAPI converts API response to Cask struct
func (c *Client) parseCaskFromAPI(apiData map[string]interface{}) (*cask.Cask, error) {
	caskData := &cask.Cask{}
	
	// Extract basic information
	if token, ok := apiData["token"].(string); ok {
		caskData.Token = token
	}
	
	if name, ok := apiData["name"].(string); ok {
		caskData.Name = name
	}
	
	if fullName, ok := apiData["full_name"].(string); ok {
		caskData.FullName = fullName
	}
	
	if homepage, ok := apiData["homepage"].(string); ok {
		caskData.Homepage = homepage
	}
	
	if desc, ok := apiData["desc"].(string); ok {
		caskData.Description = desc
	}
	
	if version, ok := apiData["version"].(string); ok {
		caskData.Version = version
	}
	
	if sha256, ok := apiData["sha256"].(string); ok {
		caskData.Sha256 = sha256
	}
	
	if caveats, ok := apiData["caveats"].(string); ok {
		caskData.Caveats = caveats
	}
	
	// Extract URL information
	if urlData, ok := apiData["url"].([]interface{}); ok && len(urlData) > 0 {
		for _, urlItem := range urlData {
			if urlMap, ok := urlItem.(map[string]interface{}); ok {
				caskURL := cask.CaskURL{}
				if url, ok := urlMap["url"].(string); ok {
					caskURL.URL = url
				}
				caskData.URL = append(caskData.URL, caskURL)
			}
		}
	} else if urlStr, ok := apiData["url"].(string); ok {
		// Handle simple string URL
		caskData.URL = []cask.CaskURL{{URL: urlStr}}
	}
	
	// Extract artifacts
	if artifactsData, ok := apiData["artifacts"].([]interface{}); ok && len(artifactsData) > 0 {
		artifact := cask.CaskArtifact{}
		
		for _, artifactItem := range artifactsData {
			if artifactMap, ok := artifactItem.(map[string]interface{}); ok {
				// Extract apps
				if apps, ok := artifactMap["app"].([]interface{}); ok {
					for _, appItem := range apps {
						if appStr, ok := appItem.(string); ok {
							artifact.App = append(artifact.App, cask.CaskApp{Source: appStr})
						} else if appMap, ok := appItem.(map[string]interface{}); ok {
							app := cask.CaskApp{}
							if source, ok := appMap["source"].(string); ok {
								app.Source = source
							}
							if target, ok := appMap["target"].(string); ok {
								app.Target = target
							}
							artifact.App = append(artifact.App, app)
						}
					}
				}
				
				// Extract binaries
				if binaries, ok := artifactMap["binary"].([]interface{}); ok {
					for _, binaryItem := range binaries {
						if binaryStr, ok := binaryItem.(string); ok {
							artifact.Binary = append(artifact.Binary, cask.CaskBinary{Source: binaryStr})
						} else if binaryMap, ok := binaryItem.(map[string]interface{}); ok {
							binary := cask.CaskBinary{}
							if source, ok := binaryMap["source"].(string); ok {
								binary.Source = source
							}
							if target, ok := binaryMap["target"].(string); ok {
								binary.Target = target
							}
							artifact.Binary = append(artifact.Binary, binary)
						}
					}
				}
				
				// Extract packages
				if pkgs, ok := artifactMap["pkg"].([]interface{}); ok {
					for _, pkgItem := range pkgs {
						if pkgStr, ok := pkgItem.(string); ok {
							artifact.Pkg = append(artifact.Pkg, pkgStr)
						}
					}
				}
			}
		}
		
		caskData.Artifacts = []cask.CaskArtifact{artifact}
	}
	
	// Extract dependencies
	if depsData, ok := apiData["depends_on"].(map[string]interface{}); ok {
		dep := cask.CaskDependency{}
		
		if macosData, ok := depsData["macos"].(map[string]interface{}); ok {
			macos := &cask.CaskMacOSRequirement{}
			if min, ok := macosData[">="].(string); ok {
				macos.Minimum = min
			}
			if max, ok := macosData["<="].(string); ok {
				macos.Maximum = max
			}
			if exact, ok := macosData["=="].(string); ok {
				macos.Exact = exact
			}
			dep.Macos = macos
		}
		
		if archData, ok := depsData["arch"].([]interface{}); ok {
			for _, archItem := range archData {
				if archStr, ok := archItem.(string); ok {
					dep.Arch = append(dep.Arch, archStr)
				}
			}
		}
		
		caskData.Depends = []cask.CaskDependency{dep}
	}
	
	// Basic validation
	if caskData.Token == "" {
		return nil, fmt.Errorf("invalid cask data: missing token")
	}
	
	return caskData, nil
}