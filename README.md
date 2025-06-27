# Homebrew Go

[![GitHub release](https://img.shields.io/github/release/pilshchikov/homebrew-go.svg)](https://github.com/pilshchikov/homebrew-go/releases)
[![Go Version](https://img.shields.io/github/go-mod/go-version/pilshchikov/homebrew-go)](https://github.com/pilshchikov/homebrew-go)
[![License](https://img.shields.io/github/license/pilshchikov/homebrew-go)](https://github.com/pilshchikov/homebrew-go/blob/HEAD/LICENSE.txt)
[![CI](https://github.com/pilshchikov/homebrew-go/workflows/CI/badge.svg)](https://github.com/pilshchikov/homebrew-go/actions)

A complete rewrite of the [Homebrew](https://brew.sh) package manager from Ruby to Go, maintaining full compatibility with the original Homebrew API and functionality.

## About This Project

This project was created by **Stepan Pilshchikov** ([GitHub](https://github.com/pilshchikov)) as a fun side project to explore the capabilities of Claude Code and demonstrate a complete language migration of a complex system.

**Important Disclaimer:** This is an independent project and is not affiliated with the official Homebrew project. It does not pretend to replace or compete with the original Homebrew. This is purely an educational and experimental endeavor.

## Features

✅ **Complete Go Implementation** - Fully rewritten from Ruby to Go  
✅ **API Compatibility** - Works with all original Homebrew APIs and endpoints  
✅ **All Commands** - Support for install, uninstall, update, upgrade, search, info, and more  
✅ **Formulae & Casks** - Full support for both formulae and cask installations  
✅ **Tap System** - Compatible with existing Homebrew taps  
✅ **Cross-Platform** - Supports macOS and Linux  
✅ **Performance** - Enhanced performance through Go's concurrency and efficiency  

## Installation

### Quick Install

Install Homebrew Go using our installation script:

```bash
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/pilshchikov/homebrew-go/main/install.sh)"
```

### Manual Installation

1. Download the appropriate release for your platform from [GitHub Releases](https://github.com/pilshchikov/homebrew-go/releases):
   - **M1/M2 Macs**: `homebrew-go_Darwin_arm64.tar.gz`
   - **Intel Macs**: `homebrew-go_Darwin_x86_64.tar.gz`
   - **Linux x64**: `homebrew-go_Linux_x86_64.tar.gz`
   - **Linux ARM**: `homebrew-go_Linux_arm64.tar.gz`

2. Extract and install the binary:

```bash
# Extract the archive
tar -xzf homebrew-go_*.tar.gz

# Move binary to your PATH (requires sudo on most systems)
sudo mv brew /usr/local/bin/brew

# Make executable
sudo chmod +x /usr/local/bin/brew

# Verify installation
brew --version
```

### Build from Source

If you have Go installed:

```bash
git clone https://github.com/pilshchikov/homebrew-go.git
cd homebrew-go
go build -o build/brew ./cmd/brew
./build/brew --version
```

## Usage

Homebrew Go maintains the same command-line interface as the original Homebrew:

```bash
# Install a package
brew install git

# Search for packages
brew search python

# Update package database
brew update

# Upgrade installed packages
brew upgrade

# Get package information
brew info node

# List installed packages
brew list

# Uninstall a package
brew uninstall wget
```

### Additional Commands

```bash
# Show dependencies
brew deps <formula>

# Show packages that depend on a formula
brew uses <formula>

# Pin a package version
brew pin <formula>

# Unpin a package
brew unpin <formula>

# Link/unlink packages
brew link <formula>
brew unlink <formula>

# Package management
brew cleanup
brew doctor
```

## Architecture

This Go implementation maintains the same architecture concepts as the original Homebrew:

- **Formulae** - Package definitions for command-line tools
- **Casks** - Package definitions for GUI applications
- **Taps** - Third-party repositories of formulae and casks
- **Cellar** - Installation directory for packages
- **API** - RESTful API for package metadata

### Key Differences from Ruby Version

- **Performance** - Faster execution due to Go's compiled nature
- **Concurrency** - Better parallel processing for installations
- **Memory Usage** - Lower memory footprint
- **Dependencies** - No Ruby runtime dependency required

## Configuration

Homebrew Go respects the same environment variables as the original:

```bash
# Installation prefix (default: /opt/homebrew on Apple Silicon, /usr/local on Intel)
export HOMEBREW_PREFIX="/custom/path"

# Disable analytics
export HOMEBREW_NO_ANALYTICS=1

# Disable auto-update
export HOMEBREW_NO_AUTO_UPDATE=1

# GitHub API token for higher rate limits
export HOMEBREW_GITHUB_API_TOKEN="your_token_here"
```

## Compatibility

- **macOS** - Supports Apple Silicon (M1/M2) and Intel processors
- **Linux** - Full Linux support with package management
- **Formulae** - Compatible with existing Homebrew formulae
- **Casks** - Works with existing Homebrew casks
- **Taps** - Can use existing third-party taps

## Contributing

Interested in contributing? Check out our [CONTRIBUTING.md](CONTRIBUTING.md) for development setup, testing, and contribution guidelines.

## Project Status

This project is actively maintained and fully functional. All core Homebrew features have been implemented and tested. The project serves as a proof-of-concept for large-scale language migration while maintaining full backwards compatibility.

## License

This project is licensed under the same BSD 2-Clause License as the original Homebrew project. See [LICENSE.txt](LICENSE.txt) for details.

## Acknowledgments

- Original [Homebrew](https://brew.sh) project and maintainers
- [Claude Code](https://claude.ai/code) for development assistance
- Go community for excellent tooling and libraries

## Support

For questions, issues, or contributions, please visit our [GitHub repository](https://github.com/pilshchikov/homebrew-go).

**Note:** This is an independent project. For official Homebrew support, please visit [brew.sh](https://brew.sh).
