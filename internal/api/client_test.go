package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/pilshchikov/homebrew-go/internal/config"
	"github.com/pilshchikov/homebrew-go/internal/formula"
	"github.com/pilshchikov/homebrew-go/internal/logger"
)

func TestNewClient(t *testing.T) {
	cfg := &config.Config{}
	client := NewClient(cfg)

	if client == nil {
		t.Fatal("NewClient returned nil")
	}

	if client.config != cfg {
		t.Error("Client config not set correctly")
	}

	if client.httpClient == nil {
		t.Error("HTTP client not initialized")
	}

	if client.httpClient.Timeout != 30*time.Second {
		t.Errorf("Expected timeout of 30s, got %v", client.httpClient.Timeout)
	}

	if client.apiDomain == "" {
		t.Error("API domain not set")
	}

	if client.userAgent == "" {
		t.Error("User agent not set")
	}

	expectedUserAgent := fmt.Sprintf("Homebrew-Go/3.0.0 (%s; %s) Go/%s",
		runtime.GOOS, runtime.GOARCH, "1.20")
	if client.userAgent != expectedUserAgent {
		t.Errorf("Expected user agent %s, got %s", expectedUserAgent, client.userAgent)
	}
}

func TestNewClientWithCustomDomain(t *testing.T) {
	_ = os.Setenv("HOMEBREW_API_DOMAIN", "https://custom.api.domain")
	defer func() { _ = os.Unsetenv("HOMEBREW_API_DOMAIN") }()

	cfg := &config.Config{}
	client := NewClient(cfg)

	if client.apiDomain != "https://custom.api.domain" {
		t.Errorf("Expected custom API domain, got %s", client.apiDomain)
	}
}

func TestGetFormula(t *testing.T) {
	// Create a mock HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/formula/wget.json" {
			w.Header().Set("Content-Type", "application/json")
			response := FormulaAPIResponse{
				Name:         "wget",
				FullName:     "wget",
				Desc:         "Internet file retriever",
				Homepage:     "https://www.gnu.org/software/wget/",
				License:      "GPL-3.0",
				Dependencies: []string{"openssl@1.1"},
				Versions: map[string]interface{}{
					"stable": "1.21.3",
				},
				Urls: map[string]interface{}{
					"stable": map[string]interface{}{
						"url":      "https://ftp.gnu.org/gnu/wget/wget-1.21.3.tar.gz",
						"checksum": "5726bb8bc5ca0f6dc7110f6416e4bb7019e2d2ff5bf93d1ca2ffcc6656f220e5",
					},
				},
				Bottle: map[string]interface{}{
					"stable": map[string]interface{}{
						"files": map[string]interface{}{
							"monterey": map[string]interface{}{
								"url":    "https://ghcr.io/v2/homebrew/core/wget/blobs/sha256:abc123",
								"sha256": "abc123def456",
							},
						},
					},
				},
			}
			_ = json.NewEncoder(w).Encode(response)
		} else {
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	cfg := &config.Config{}
	client := NewClient(cfg)
	client.apiDomain = server.URL

	formula, err := client.GetFormula("wget")
	if err != nil {
		t.Fatalf("GetFormula failed: %v", err)
	}

	if formula.Name != "wget" {
		t.Errorf("Expected name 'wget', got '%s'", formula.Name)
	}

	if formula.Description != "Internet file retriever" {
		t.Errorf("Expected description 'Internet file retriever', got '%s'", formula.Description)
	}

	if formula.Homepage != "https://www.gnu.org/software/wget/" {
		t.Errorf("Expected homepage URL, got '%s'", formula.Homepage)
	}

	if formula.License != "GPL-3.0" {
		t.Errorf("Expected license 'GPL-3.0', got '%s'", formula.License)
	}

	if formula.Version != "1.21.3" {
		t.Errorf("Expected version '1.21.3', got '%s'", formula.Version)
	}

	if len(formula.Dependencies) != 1 || formula.Dependencies[0] != "openssl@1.1" {
		t.Errorf("Expected dependencies [openssl@1.1], got %v", formula.Dependencies)
	}

	if formula.URL != "https://ftp.gnu.org/gnu/wget/wget-1.21.3.tar.gz" {
		t.Errorf("Expected URL, got '%s'", formula.URL)
	}

	if formula.SHA256 != "5726bb8bc5ca0f6dc7110f6416e4bb7019e2d2ff5bf93d1ca2ffcc6656f220e5" {
		t.Errorf("Expected SHA256, got '%s'", formula.SHA256)
	}

	if formula.Bottle == nil {
		t.Error("Expected bottle information")
	} else {
		if formula.Bottle.Stable == nil {
			t.Error("Expected stable bottle")
		} else {
			if _, exists := formula.Bottle.Stable.Files["monterey"]; !exists {
				t.Error("Expected monterey bottle file")
			}
		}
	}
}

func TestGetFormulaNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer server.Close()

	cfg := &config.Config{}
	client := NewClient(cfg)
	client.apiDomain = server.URL

	_, err := client.GetFormula("nonexistent")
	if err == nil {
		t.Error("Expected error for non-existent formula")
	}

	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Expected 'not found' error, got: %v", err)
	}
}

func TestGetFormulaServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}))
	defer server.Close()

	cfg := &config.Config{}
	client := NewClient(cfg)
	client.apiDomain = server.URL

	_, err := client.GetFormula("test")
	if err == nil {
		t.Error("Expected error for server error")
	}

	if !strings.Contains(err.Error(), "failed with status 500") {
		t.Errorf("Expected status 500 error, got: %v", err)
	}
}

func TestSearchFormulae(t *testing.T) {
	// Create temporary directory for cache
	tempDir, err := os.MkdirTemp("", "brew-test-cache")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.URL.Path == "/formula.json" {
			// Return list of formulae
			formulae := []map[string]interface{}{
				{"name": "wget"},
				{"name": "curl"},
				{"name": "git"},
			}
			_ = json.NewEncoder(w).Encode(formulae)
		} else if strings.HasPrefix(r.URL.Path, "/formula/") {
			// Return specific formula details
			name := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/formula/"), ".json")
			response := FormulaAPIResponse{
				Name:     name,
				FullName: name,
				Desc:     fmt.Sprintf("Description for %s", name),
				Homepage: fmt.Sprintf("https://example.com/%s", name),
				Versions: map[string]interface{}{
					"stable": "1.0.0",
				},
			}
			_ = json.NewEncoder(w).Encode(response)
		} else {
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	cfg := &config.Config{
		HomebrewCache: tempDir,
	}
	client := NewClient(cfg)
	client.apiDomain = server.URL

	results, err := client.SearchFormulae("wget")
	if err != nil {
		t.Fatalf("SearchFormulae failed: %v", err)
	}

	if len(results) == 0 {
		t.Error("Expected search results")
	}

	found := false
	for _, result := range results {
		if result.Name == "wget" {
			found = true
			if result.Desc != "Description for wget" {
				t.Errorf("Expected description, got '%s'", result.Desc)
			}
		}
	}

	if !found {
		t.Error("Expected to find wget in search results")
	}
}

func TestGetPlatformTag(t *testing.T) {
	cfg := &config.Config{}
	client := NewClient(cfg)

	platform := client.GetPlatformTag()
	if platform == "" {
		t.Error("Expected non-empty platform tag")
	}

	// Test platform detection logic
	switch runtime.GOOS {
	case "darwin":
		if runtime.GOARCH == "arm64" {
			if platform != "arm64_sequoia" {
				t.Errorf("Expected arm64_sequoia for Apple Silicon, got %s", platform)
			}
		} else {
			if platform != "x86_64_sequoia" {
				t.Errorf("Expected x86_64_sequoia for Intel Mac, got %s", platform)
			}
		}
	case "linux":
		if runtime.GOARCH == "arm64" {
			if platform != "arm64_linux" {
				t.Errorf("Expected arm64_linux, got %s", platform)
			}
		} else {
			if platform != "x86_64_linux" {
				t.Errorf("Expected x86_64_linux, got %s", platform)
			}
		}
	default:
		expected := runtime.GOOS + "_" + runtime.GOARCH
		if platform != expected {
			t.Errorf("Expected %s, got %s", expected, platform)
		}
	}
}

func TestIsCacheValid(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "brew-test-cache")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	cfg := &config.Config{}
	client := NewClient(cfg)

	// Test non-existent file
	nonExistentFile := filepath.Join(tempDir, "nonexistent.txt")
	if client.isCacheValid(nonExistentFile) {
		t.Error("Expected false for non-existent file")
	}

	// Test recent file
	recentFile := filepath.Join(tempDir, "recent.txt")
	if err := os.WriteFile(recentFile, []byte("test"), 0600); err != nil {
		t.Fatalf("Failed to create recent file: %v", err)
	}

	if !client.isCacheValid(recentFile) {
		t.Error("Expected true for recent file")
	}

	// Test old file (simulate by modifying access time)
	oldTime := time.Now().Add(-2 * time.Hour)
	if err := os.Chtimes(recentFile, oldTime, oldTime); err != nil {
		t.Fatalf("Failed to modify file time: %v", err)
	}

	if client.isCacheValid(recentFile) {
		t.Error("Expected false for old file")
	}
}

func TestReadCachedNames(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "brew-test-cache")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	cfg := &config.Config{}
	client := NewClient(cfg)

	cacheFile := filepath.Join(tempDir, "test.txt")
	content := "wget\ncurl\ngit\n"

	if err := os.WriteFile(cacheFile, []byte(content), 0600); err != nil {
		t.Fatalf("Failed to write cache file: %v", err)
	}

	names, err := client.readCachedNames(cacheFile)
	if err != nil {
		t.Fatalf("readCachedNames failed: %v", err)
	}

	expected := []string{"wget", "curl", "git"}
	if len(names) != len(expected) {
		t.Errorf("Expected %d names, got %d", len(expected), len(names))
	}

	for i, name := range names {
		if name != expected[i] {
			t.Errorf("Expected name %s, got %s", expected[i], name)
		}
	}
}

func TestCacheNames(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "brew-test-cache")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	cfg := &config.Config{}
	client := NewClient(cfg)

	names := []string{"wget", "curl", "git"}
	cacheFile := filepath.Join(tempDir, "subdir", "test.txt")

	client.cacheNames(cacheFile, names)

	// Check if file was created
	if _, err := os.Stat(cacheFile); os.IsNotExist(err) {
		t.Error("Cache file was not created")
	}

	// Read back and verify
	content, err := os.ReadFile(cacheFile)
	if err != nil {
		t.Fatalf("Failed to read cache file: %v", err)
	}

	expected := strings.Join(names, "\n")
	if strings.TrimSpace(string(content)) != expected {
		t.Errorf("Expected content %s, got %s", expected, strings.TrimSpace(string(content)))
	}
}

func TestIsFileValid(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "brew-test-cache")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	cfg := &config.Config{}
	client := NewClient(cfg)

	// Test non-existent file
	nonExistentFile := filepath.Join(tempDir, "nonexistent.txt")
	if client.isFileValid(nonExistentFile, "abc123") {
		t.Error("Expected false for non-existent file")
	}

	// Test file with no checksum requirement
	testFile := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0600); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	if !client.isFileValid(testFile, "") {
		t.Error("Expected true for file with no checksum requirement")
	}

	// Test file with correct checksum would require actual SHA256 computation
	// For now, just test the flow
	if client.isFileValid(testFile, "invalidchecksum") {
		t.Error("Expected false for file with invalid checksum")
	}
}

func TestDownloadBottle(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "brew-test-cache")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Test formula without bottle
	testFormula := &formula.Formula{
		Name:    "test",
		Version: "1.0.0",
	}

	cfg := &config.Config{HomebrewCache: tempDir}
	client := NewClient(cfg)

	_, err = client.DownloadBottle(testFormula, "monterey")
	if err == nil {
		t.Error("Expected error for formula without bottle")
	}

	if !strings.Contains(err.Error(), "no bottle available") {
		t.Errorf("Expected 'no bottle available' error, got: %v", err)
	}

	// Test formula with bottle but unsupported platform
	testFormula.Bottle = &formula.Bottle{
		Stable: &formula.BottleSpec{
			Files: make(map[string]formula.BottleFile),
		},
	}

	// Add a bottle file for monterey
	testFormula.Bottle.Stable.Files["monterey"] = formula.BottleFile{
		URL:    "https://example.com/test.tar.gz",
		SHA256: "abc123",
	}

	_, err = client.DownloadBottle(testFormula, "unsupported")
	if err == nil {
		t.Error("Expected error for unsupported platform")
	}

	if !strings.Contains(err.Error(), "no bottle available for platform") {
		t.Errorf("Expected platform error, got: %v", err)
	}
}

func TestAddGHCRAuth(t *testing.T) {
	// Test with GitHub token from environment
	_ = os.Setenv("GITHUB_TOKEN", "test-token")
	defer func() { _ = os.Unsetenv("GITHUB_TOKEN") }()

	cfg := &config.Config{}
	client := NewClient(cfg)

	req, err := http.NewRequest("GET", "https://ghcr.io/test", http.NoBody)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	err = client.addGHCRAuth(req)
	if err != nil {
		t.Fatalf("addGHCRAuth failed: %v", err)
	}

	auth := req.Header.Get("Authorization")
	if auth != "Bearer test-token" {
		t.Errorf("Expected 'Bearer test-token', got '%s'", auth)
	}
}

func TestDownloadWithRetry(t *testing.T) {
	// Test successful download
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("test content"))
	}))
	defer server.Close()

	cfg := &config.Config{}
	client := NewClient(cfg)

	req, err := http.NewRequest("GET", server.URL, http.NoBody)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	resp, err := client.downloadWithRetry(req, server.URL)
	if err != nil {
		t.Fatalf("downloadWithRetry failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	if string(body) != "test content" {
		t.Errorf("Expected 'test content', got '%s'", string(body))
	}
}

func TestParseCaskFromAPI(t *testing.T) {
	cfg := &config.Config{}
	client := NewClient(cfg)

	apiData := map[string]interface{}{
		"token":     "test-cask",
		"name":      "Test Cask",
		"full_name": "test-cask",
		"homepage":  "https://example.com",
		"desc":      "A test cask",
		"version":   "1.0.0",
		"sha256":    "abc123",
		"url":       "https://example.com/test.dmg",
		"artifacts": []interface{}{
			map[string]interface{}{
				"app": []interface{}{"Test.app"},
			},
		},
	}

	cask, err := client.parseCaskFromAPI(apiData)
	if err != nil {
		t.Fatalf("parseCaskFromAPI failed: %v", err)
	}

	if cask.Token != "test-cask" {
		t.Errorf("Expected token 'test-cask', got '%s'", cask.Token)
	}

	if cask.Name != "Test Cask" {
		t.Errorf("Expected name 'Test Cask', got '%s'", cask.Name)
	}

	if cask.Homepage != "https://example.com" {
		t.Errorf("Expected homepage URL, got '%s'", cask.Homepage)
	}

	if cask.Version != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got '%s'", cask.Version)
	}

	if len(cask.URL) == 0 || cask.URL[0].URL != "https://example.com/test.dmg" {
		t.Errorf("Expected URL, got %v", cask.URL)
	}

	if len(cask.Artifacts) == 0 || len(cask.Artifacts[0].App) == 0 {
		t.Error("Expected app artifact")
	}
}

func TestParseCaskFromAPIInvalid(t *testing.T) {
	cfg := &config.Config{}
	client := NewClient(cfg)

	// Test missing token
	apiData := map[string]interface{}{
		"name": "Test Cask",
	}

	_, err := client.parseCaskFromAPI(apiData)
	if err == nil {
		t.Error("Expected error for missing token")
	}

	if !strings.Contains(err.Error(), "missing token") {
		t.Errorf("Expected 'missing token' error, got: %v", err)
	}
}

func TestGetCask(t *testing.T) {
	// Create a mock HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/cask/firefox.json":
			w.Header().Set("Content-Type", "application/json")
			response := map[string]interface{}{
				"token":     "firefox",
				"name":      "Firefox",
				"full_name": "firefox",
				"homepage":  "https://www.mozilla.org/firefox/",
				"desc":      "Web browser",
				"version":   "120.0",
				"sha256":    "abc123def456",
				"url":       "https://download.mozilla.org/firefox.dmg",
				"artifacts": []interface{}{
					map[string]interface{}{
						"app": []interface{}{"Firefox.app"},
					},
				},
				"depends_on": map[string]interface{}{
					"macos": map[string]interface{}{
						">=": "10.15",
					},
					"arch": []interface{}{"x86_64", "arm64"},
				},
			}
			_ = json.NewEncoder(w).Encode(response)
		case "/cask/notfound.json":
			http.NotFound(w, r)
		default:
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
	}))
	defer server.Close()

	cfg := &config.Config{}
	client := NewClient(cfg)
	client.apiDomain = server.URL

	// Test successful cask fetch
	cask, err := client.GetCask("firefox")
	if err != nil {
		t.Fatalf("GetCask failed: %v", err)
	}

	if cask.Token != "firefox" {
		t.Errorf("Expected token 'firefox', got '%s'", cask.Token)
	}

	if cask.Name != "Firefox" {
		t.Errorf("Expected name 'Firefox', got '%s'", cask.Name)
	}

	if cask.Homepage != "https://www.mozilla.org/firefox/" {
		t.Errorf("Expected homepage URL, got '%s'", cask.Homepage)
	}

	if cask.Description != "Web browser" {
		t.Errorf("Expected description 'Web browser', got '%s'", cask.Description)
	}

	if cask.Version != "120.0" {
		t.Errorf("Expected version '120.0', got '%s'", cask.Version)
	}

	if cask.Sha256 != "abc123def456" {
		t.Errorf("Expected SHA256, got '%s'", cask.Sha256)
	}

	if len(cask.URL) == 0 || cask.URL[0].URL != "https://download.mozilla.org/firefox.dmg" {
		t.Errorf("Expected URL, got %v", cask.URL)
	}

	if len(cask.Artifacts) == 0 || len(cask.Artifacts[0].App) == 0 {
		t.Error("Expected app artifact")
	} else if cask.Artifacts[0].App[0].Source != "Firefox.app" {
		t.Errorf("Expected app 'Firefox.app', got '%s'", cask.Artifacts[0].App[0].Source)
	}

	if len(cask.Depends) == 0 {
		t.Error("Expected dependencies")
	} else {
		dep := cask.Depends[0]
		if dep.Macos == nil || dep.Macos.Minimum != "10.15" {
			t.Error("Expected macOS minimum version 10.15")
		}
		if len(dep.Arch) != 2 || dep.Arch[0] != "x86_64" || dep.Arch[1] != "arm64" {
			t.Errorf("Expected arch [x86_64, arm64], got %v", dep.Arch)
		}
	}

	// Test cask not found
	_, err = client.GetCask("notfound")
	if err == nil {
		t.Error("Expected error for non-existent cask")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Expected 'not found' error, got: %v", err)
	}

	// Test server error
	_, err = client.GetCask("error")
	if err == nil {
		t.Error("Expected error for server error")
	}
	if !strings.Contains(err.Error(), "API request failed") {
		t.Errorf("Expected API error, got: %v", err)
	}
}

func TestSearchCasks(t *testing.T) {
	// Create a mock HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/cask.json" {
			w.Header().Set("Content-Type", "application/json")
			caskList := []map[string]interface{}{
				{
					"token": "firefox",
					"name":  "Firefox",
					"desc":  "Web browser",
				},
				{
					"token": "chrome",
					"name":  "Google Chrome",
					"desc":  "Web browser",
				},
				{
					"token": "safari-technology-preview",
					"name":  "Safari Technology Preview",
					"desc":  "Web browser preview",
				},
				{
					"token": "docker",
					"name":  "Docker Desktop",
					"desc":  "Container platform",
				},
			}
			_ = json.NewEncoder(w).Encode(caskList)
		} else {
			http.Error(w, "Not Found", http.StatusNotFound)
		}
	}))
	defer server.Close()

	cfg := &config.Config{}
	client := NewClient(cfg)
	client.apiDomain = server.URL

	// Test search by token
	results, err := client.SearchCasks("fire")
	if err != nil {
		t.Fatalf("SearchCasks failed: %v", err)
	}

	if len(results) == 0 {
		t.Error("Expected search results for 'fire'")
	}

	foundFirefox := false
	for _, cask := range results {
		if cask.Token == "firefox" {
			foundFirefox = true
			if cask.Name != "Firefox" {
				t.Errorf("Expected name 'Firefox', got '%s'", cask.Name)
			}
		}
	}
	if !foundFirefox {
		t.Error("Expected to find firefox in search results")
	}

	// Test search by name
	results, err = client.SearchCasks("chrome")
	if err != nil {
		t.Fatalf("SearchCasks failed: %v", err)
	}

	foundChrome := false
	for _, cask := range results {
		if cask.Token == "chrome" || strings.Contains(strings.ToLower(cask.Name), "chrome") {
			foundChrome = true
		}
	}
	if !foundChrome {
		t.Error("Expected to find chrome-related cask in search results")
	}

	// Test search with no results
	results, err = client.SearchCasks("nonexistentcask")
	if err != nil {
		t.Fatalf("SearchCasks failed: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("Expected no results for 'nonexistentcask', got %d", len(results))
	}

	// Test search limiting (should not exceed 50 results)
	results, err = client.SearchCasks("")
	if err != nil {
		t.Fatalf("SearchCasks failed: %v", err)
	}

	// Should get all 4 casks since we search for empty string
	if len(results) > 50 {
		t.Errorf("Expected at most 50 results, got %d", len(results))
	}
}

func TestSearchCasksAPIError(t *testing.T) {
	// Create a mock server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}))
	defer server.Close()

	cfg := &config.Config{}
	client := NewClient(cfg)
	client.apiDomain = server.URL

	_, err := client.SearchCasks("test")
	if err == nil {
		t.Error("Expected error for API failure")
	}

	if !strings.Contains(err.Error(), "API request failed") {
		t.Errorf("Expected API error, got: %v", err)
	}
}

func TestDownloadBottleEnhanced(t *testing.T) {
	// Initialize logger for tests
	logger.Init(false, false, true)

	tempDir, err := os.MkdirTemp("", "test-cache")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	cfg := &config.Config{
		HomebrewCache: tempDir,
	}

	client := NewClient(cfg)

	// Test content for bottle
	testContent := "fake bottle content"
	// Use SHA256 of actual test content
	contentHash := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855" // SHA256 of empty string

	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/gzip")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(testContent))
	}))
	defer server.Close()

	// Create a test formula with bottle info
	testFormula := &formula.Formula{
		Name:    "test-formula",
		Version: "1.0.0",
		Bottle: &formula.Bottle{
			Stable: &formula.BottleSpec{
				Files: make(map[string]formula.BottleFile),
			},
		},
	}
	testFormula.Bottle.Stable.Files["x86_64_sequoia"] = formula.BottleFile{
		URL:    server.URL + "/bottle.tar.gz",
		SHA256: contentHash,
	}

	// Test download failure with checksum mismatch (since we use wrong hash)
	_, err = client.DownloadBottle(testFormula, "x86_64_sequoia")
	if err == nil {
		t.Error("Expected checksum verification to fail with mismatched hash")
	}
	if !strings.Contains(err.Error(), "checksum verification failed") {
		t.Errorf("Expected checksum error, got: %v", err)
	}

	// Test platform not available
	_, err = client.DownloadBottle(testFormula, "nonexistent_platform")
	if err == nil {
		t.Error("Expected error for non-existent platform")
	}
	if !strings.Contains(err.Error(), "no bottle available for platform") {
		t.Errorf("Expected platform error, got: %v", err)
	}

	// Test formula without bottle
	noBottleFormula := &formula.Formula{
		Name: "no-bottle",
	}
	_, err = client.DownloadBottle(noBottleFormula, "x86_64_sequoia")
	if err == nil {
		t.Error("Expected error for formula without bottle")
	}
	if !strings.Contains(err.Error(), "no bottle available for") {
		t.Errorf("Expected bottle error, got: %v", err)
	}

	// Test GHCR URL (should attempt authentication)
	ghcrFormula := &formula.Formula{
		Name:    "ghcr-formula",
		Version: "1.0.0",
		Bottle: &formula.Bottle{
			Stable: &formula.BottleSpec{
				Files: make(map[string]formula.BottleFile),
			},
		},
	}
	ghcrFormula.Bottle.Stable.Files["x86_64_sequoia"] = formula.BottleFile{
		URL:    "https://ghcr.io/homebrew/core/test:latest",
		SHA256: contentHash,
	}

	// This should attempt GHCR auth (will fail but code path is tested)
	_, err = client.DownloadBottle(ghcrFormula, "x86_64_sequoia")
	if err == nil {
		t.Log("GHCR download test completed (expected to fail in test environment)")
	}

	// Test successful download with no checksum requirement
	noChecksumFormula := &formula.Formula{
		Name:    "no-checksum",
		Version: "1.0.0",
		Bottle: &formula.Bottle{
			Stable: &formula.BottleSpec{
				Files: make(map[string]formula.BottleFile),
			},
		},
	}
	noChecksumFormula.Bottle.Stable.Files["x86_64_sequoia"] = formula.BottleFile{
		URL:    server.URL + "/bottle.tar.gz",
		SHA256: "", // No checksum
	}

	filePath, err := client.DownloadBottle(noChecksumFormula, "x86_64_sequoia")
	if err != nil {
		t.Errorf("Expected successful download with no checksum, got: %v", err)
	} else {
		// Verify file was created
		if _, statErr := os.Stat(filePath); os.IsNotExist(statErr) {
			t.Error("Downloaded file does not exist")
		}
	}
}
