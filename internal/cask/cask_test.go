package cask

import (
	"runtime"
	"testing"
	"time"
)

func TestCask_GetDownloadURL(t *testing.T) {
	tests := []struct {
		name     string
		cask     *Cask
		expected string
	}{
		{
			name: "single URL",
			cask: &Cask{
				URL: []CaskURL{
					{URL: "https://example.com/app.dmg"},
				},
			},
			expected: "https://example.com/app.dmg",
		},
		{
			name: "multiple URLs",
			cask: &Cask{
				URL: []CaskURL{
					{URL: "https://example.com/app.dmg"},
					{URL: "https://backup.com/app.dmg"},
				},
			},
			expected: "https://example.com/app.dmg",
		},
		{
			name:     "no URLs",
			cask:     &Cask{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.cask.GetDownloadURL()
			if result != tt.expected {
				t.Errorf("GetDownloadURL() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestCask_HasApplication(t *testing.T) {
	tests := []struct {
		name     string
		cask     *Cask
		expected bool
	}{
		{
			name: "has application",
			cask: &Cask{
				Artifacts: []CaskArtifact{
					{
						App: []CaskApp{
							{Source: "MyApp.app"},
						},
					},
				},
			},
			expected: true,
		},
		{
			name: "no application",
			cask: &Cask{
				Artifacts: []CaskArtifact{
					{
						Binary: []CaskBinary{
							{Source: "mybinary"},
						},
					},
				},
			},
			expected: false,
		},
		{
			name:     "no artifacts",
			cask:     &Cask{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.cask.HasApplication()
			if result != tt.expected {
				t.Errorf("HasApplication() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestCask_GetPrimaryAppName(t *testing.T) {
	tests := []struct {
		name     string
		cask     *Cask
		expected string
	}{
		{
			name: "app with target",
			cask: &Cask{
				Name: "test-app",
				Artifacts: []CaskArtifact{
					{
						App: []CaskApp{
							{Source: "MyApp.app", Target: "Custom Name.app"},
						},
					},
				},
			},
			expected: "Custom Name.app",
		},
		{
			name: "app without target",
			cask: &Cask{
				Name: "test-app",
				Artifacts: []CaskArtifact{
					{
						App: []CaskApp{
							{Source: "MyApp.app"},
						},
					},
				},
			},
			expected: "MyApp.app",
		},
		{
			name: "no app",
			cask: &Cask{
				Name: "test-app",
			},
			expected: "test-app.app",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.cask.GetPrimaryAppName()
			if result != tt.expected {
				t.Errorf("GetPrimaryAppName() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestCask_IsCompatibleWithPlatform(t *testing.T) {
	tests := []struct {
		name     string
		cask     *Cask
		expected bool
	}{
		{
			name: "no arch requirements on macOS",
			cask: &Cask{
				Token: "test-app",
			},
			expected: runtime.GOOS == "darwin",
		},
		{
			name: "compatible arch requirement",
			cask: &Cask{
				Token: "test-app",
				Depends: []CaskDependency{
					{
						Arch: []string{"x86_64", "arm64"},
					},
				},
			},
			expected: runtime.GOOS == "darwin",
		},
		{
			name: "incompatible arch requirement",
			cask: &Cask{
				Token: "test-app",
				Depends: []CaskDependency{
					{
						Arch: []string{"powerpc"},
					},
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.cask.IsCompatibleWithPlatform()
			if result != tt.expected {
				t.Errorf("IsCompatibleWithPlatform() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestCask_NeedsSudo(t *testing.T) {
	tests := []struct {
		name     string
		cask     *Cask
		expected bool
	}{
		{
			name: "has pkg installer",
			cask: &Cask{
				Artifacts: []CaskArtifact{
					{
						Pkg: []string{"installer.pkg"},
					},
				},
			},
			expected: true,
		},
		{
			name: "has custom installer",
			cask: &Cask{
				Artifacts: []CaskArtifact{
					{
						Installer: []CaskInstaller{
							{Manual: "Run installer manually"},
						},
					},
				},
			},
			expected: true,
		},
		{
			name: "regular app to Applications",
			cask: &Cask{
				Artifacts: []CaskArtifact{
					{
						App: []CaskApp{
							{Source: "MyApp.app"},
						},
					},
				},
			},
			expected: false,
		},
		{
			name: "app to system directory",
			cask: &Cask{
				Artifacts: []CaskArtifact{
					{
						App: []CaskApp{
							{Source: "MyApp.app", Target: "/System/Applications/MyApp.app"},
						},
					},
				},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.cask.NeedsSudo()
			if result != tt.expected {
				t.Errorf("NeedsSudo() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestCask_GetFileExtension(t *testing.T) {
	tests := []struct {
		name     string
		cask     *Cask
		expected string
	}{
		{
			name: "DMG download",
			cask: &Cask{
				URL: []CaskURL{
					{URL: "https://example.com/app.dmg"},
				},
			},
			expected: ".dmg",
		},
		{
			name: "PKG download",
			cask: &Cask{
				URL: []CaskURL{
					{URL: "https://example.com/app.pkg"},
				},
			},
			expected: ".pkg",
		},
		{
			name: "ZIP download",
			cask: &Cask{
				URL: []CaskURL{
					{URL: "https://example.com/app.zip"},
				},
			},
			expected: ".zip",
		},
		{
			name: "tar.gz download",
			cask: &Cask{
				URL: []CaskURL{
					{URL: "https://example.com/app.tar.gz"},
				},
			},
			expected: ".tar.gz",
		},
		{
			name: "unknown extension",
			cask: &Cask{
				URL: []CaskURL{
					{URL: "https://example.com/app.unknown"},
				},
			},
			expected: ".dmg",
		},
		{
			name:     "no URL",
			cask:     &Cask{},
			expected: ".dmg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.cask.GetFileExtension()
			if result != tt.expected {
				t.Errorf("GetFileExtension() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestCask_GetCacheFileName(t *testing.T) {
	tests := []struct {
		name     string
		cask     *Cask
		expected string
	}{
		{
			name: "with version",
			cask: &Cask{
				Token:   "my-app",
				Version: "1.0.0",
				URL: []CaskURL{
					{URL: "https://example.com/app.dmg"},
				},
			},
			expected: "my-app-1.0.0.dmg",
		},
		{
			name: "without version",
			cask: &Cask{
				Token: "my-app",
				URL: []CaskURL{
					{URL: "https://example.com/app.dmg"},
				},
			},
			expected: "my-app.dmg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.cask.GetCacheFileName()
			if result != tt.expected {
				t.Errorf("GetCacheFileName() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestCask_IsInstalled(t *testing.T) {
	now := time.Now()
	
	tests := []struct {
		name     string
		cask     *Cask
		expected bool
	}{
		{
			name: "installed",
			cask: &Cask{
				InstallTime: &now,
			},
			expected: true,
		},
		{
			name: "not installed",
			cask: &Cask{
				InstallTime: nil,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.cask.IsInstalled()
			if result != tt.expected {
				t.Errorf("IsInstalled() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestCask_RequiresManualInstallation(t *testing.T) {
	tests := []struct {
		name     string
		cask     *Cask
		expected bool
	}{
		{
			name: "manual installer",
			cask: &Cask{
				Artifacts: []CaskArtifact{
					{
						Installer: []CaskInstaller{
							{Manual: "Please run the installer manually"},
						},
					},
				},
			},
			expected: true,
		},
		{
			name: "automatic installer",
			cask: &Cask{
				Artifacts: []CaskArtifact{
					{
						App: []CaskApp{
							{Source: "MyApp.app"},
						},
					},
				},
			},
			expected: false,
		},
		{
			name:     "no artifacts",
			cask:     &Cask{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.cask.RequiresManualInstallation()
			if result != tt.expected {
				t.Errorf("RequiresManualInstallation() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestCask_Validate(t *testing.T) {
	tests := []struct {
		name      string
		cask      *Cask
		expectErr bool
	}{
		{
			name: "valid cask",
			cask: &Cask{
				Token:   "my-app",
				Version: "1.0.0",
				URL:     []CaskURL{{URL: "https://example.com/app.dmg"}},
				Sha256:  "abc123",
				Artifacts: []CaskArtifact{
					{
						App: []CaskApp{{Source: "MyApp.app"}},
					},
				},
			},
			expectErr: false,
		},
		{
			name: "missing token",
			cask: &Cask{
				Version: "1.0.0",
				URL:     []CaskURL{{URL: "https://example.com/app.dmg"}},
				Sha256:  "abc123",
				Artifacts: []CaskArtifact{
					{
						App: []CaskApp{{Source: "MyApp.app"}},
					},
				},
			},
			expectErr: true,
		},
		{
			name: "missing version",
			cask: &Cask{
				Token:  "my-app",
				URL:    []CaskURL{{URL: "https://example.com/app.dmg"}},
				Sha256: "abc123",
				Artifacts: []CaskArtifact{
					{
						App: []CaskApp{{Source: "MyApp.app"}},
					},
				},
			},
			expectErr: true,
		},
		{
			name: "missing URL",
			cask: &Cask{
				Token:   "my-app",
				Version: "1.0.0",
				Sha256:  "abc123",
				Artifacts: []CaskArtifact{
					{
						App: []CaskApp{{Source: "MyApp.app"}},
					},
				},
			},
			expectErr: true,
		},
		{
			name: "missing SHA256",
			cask: &Cask{
				Token:   "my-app",
				Version: "1.0.0",
				URL:     []CaskURL{{URL: "https://example.com/app.dmg"}},
				Artifacts: []CaskArtifact{
					{
						App: []CaskApp{{Source: "MyApp.app"}},
					},
				},
			},
			expectErr: true,
		},
		{
			name: "missing artifacts",
			cask: &Cask{
				Token:   "my-app",
				Version: "1.0.0",
				URL:     []CaskURL{{URL: "https://example.com/app.dmg"}},
				Sha256:  "abc123",
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cask.Validate()
			if tt.expectErr && err == nil {
				t.Error("Expected validation error but got none")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("Expected no validation error but got: %v", err)
			}
		})
	}
}

func TestCask_GetBinaries(t *testing.T) {
	tests := []struct {
		name     string
		cask     *Cask
		expected []CaskBinary
	}{
		{
			name: "has binaries",
			cask: &Cask{
				Artifacts: []CaskArtifact{
					{
						Binary: []CaskBinary{
							{Source: "mybinary", Target: "mybin"},
							{Source: "anotherbinary"},
						},
					},
				},
			},
			expected: []CaskBinary{
				{Source: "mybinary", Target: "mybin"},
				{Source: "anotherbinary"},
			},
		},
		{
			name: "no binaries",
			cask: &Cask{
				Artifacts: []CaskArtifact{
					{
						App: []CaskApp{
							{Source: "MyApp.app"},
						},
					},
				},
			},
			expected: []CaskBinary{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.cask.GetBinaries()
			if len(result) != len(tt.expected) {
				t.Errorf("GetBinaries() returned %d binaries, want %d", len(result), len(tt.expected))
				return
			}
			for i, binary := range result {
				if binary.Source != tt.expected[i].Source || binary.Target != tt.expected[i].Target {
					t.Errorf("GetBinaries()[%d] = %+v, want %+v", i, binary, tt.expected[i])
				}
			}
		})
	}
}

func TestCask_GetInstallPath(t *testing.T) {
	tests := []struct {
		name     string
		cask     *Cask
		caskRoot string
		expected string
	}{
		{
			name: "basic install path",
			cask: &Cask{
				Token: "my-app",
			},
			caskRoot: "/opt/homebrew/Caskroom",
			expected: "/opt/homebrew/Caskroom/my-app",
		},
		{
			name: "different cask root",
			cask: &Cask{
				Token: "another-app",
			},
			caskRoot: "/usr/local/Caskroom",
			expected: "/usr/local/Caskroom/another-app",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.cask.GetInstallPath(tt.caskRoot)
			if result != tt.expected {
				t.Errorf("GetInstallPath() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestCask_GetCaveats(t *testing.T) {
	tests := []struct {
		name     string
		cask     *Cask
		expected string
	}{
		{
			name: "has caveats",
			cask: &Cask{
				Caveats: "Please restart your computer after installation.",
			},
			expected: "Please restart your computer after installation.",
		},
		{
			name: "no caveats",
			cask: &Cask{
				Caveats: "",
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.cask.GetCaveats()
			if result != tt.expected {
				t.Errorf("GetCaveats() = %v, want %v", result, tt.expected)
			}
		})
	}
}