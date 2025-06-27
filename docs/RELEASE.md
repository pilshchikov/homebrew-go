# Release Process

This document describes the simplified automated release process for Homebrew Go.

## 🎯 Simple Release Process

The release system is fully automated with date-based versioning. You just need to trigger it!

### Version Format

All releases use the format: **`YYYY.M.DD.BUILD_ID`**

- **YYYY**: Full year (e.g., 2025)
- **M**: Month without leading zero (e.g., 6 for June, 12 for December)  
- **DD**: Day with leading zero (e.g., 01, 26)
- **BUILD_ID**: GitHub Actions run number (auto-incremented)

**Examples:**
- `2025.6.26.123` - June 26, 2025, build #123
- `2025.12.01.456` - December 1, 2025, build #456

## 🚀 How to Create a Release

### Method 1: Release Script (Recommended)

```bash
# Create a regular release
./scripts/release.sh

# Create a pre-release  
./scripts/release.sh --prerelease

# Check repository status
./scripts/release.sh --status
```

### Method 2: GitHub Actions UI

1. Go to the [Actions tab](https://github.com/pilshchikov/homebrew-go/actions)
2. Select "Release" workflow
3. Click "Run workflow"
4. Optionally check "Mark as pre-release"
5. Click "Run workflow"

### Method 3: GitHub CLI

```bash
# Regular release
gh workflow run release.yml

# Pre-release
gh workflow run release.yml --field prerelease=true
```

## ⚡ What Happens Automatically

When you trigger a release, the system automatically:

1. **Generates Version**: Creates version number based on current date and build number
2. **Runs Tests**: Executes full test suite with cache cleaning
3. **Creates Git Tag**: Tags the commit with the generated version
4. **Builds Binaries**: Cross-platform builds for:
   - Linux (amd64, arm64)
   - macOS (amd64, arm64) 
   - Windows (amd64, arm64)
5. **Creates GitHub Release**: With release notes and binary attachments
6. **Builds Docker Image**: Multi-architecture image pushed to GitHub Container Registry
7. **Generates Checksums**: SHA256 checksums for all artifacts

## 📦 Release Artifacts

Each release automatically includes:

- **Binaries**: `homebrew-go_Darwin_x86_64.tar.gz`, `homebrew-go_Linux_x86_64.tar.gz`, etc.
- **Docker Images**: `ghcr.io/pilshchikov/homebrew-go:YYYY.M.DD.BUILD_ID`
- **Checksums**: `checksums.txt` with SHA256 hashes
- **Source Code**: Automatic GitHub source archives

## 🔧 Prerequisites

To use the release script, you need:

1. **GitHub CLI** installed:
   ```bash
   brew install gh
   # or
   curl -fsSL https://cli.github.com/packages/rpm/rpm.list | sudo tee /etc/yum.repos.d/github-cli.repo
   ```

2. **GitHub CLI authenticated**:
   ```bash
   gh auth login
   ```

3. **Git repository** with push access

## 📊 Monitoring Releases

### Check Release Status
```bash
# View recent workflow runs
gh run list --limit 5

# Monitor specific workflow
gh run watch

# View releases
gh release list
```

### View Release Details
- **GitHub Releases**: https://github.com/pilshchikov/homebrew-go/releases
- **GitHub Actions**: https://github.com/pilshchikov/homebrew-go/actions
- **Docker Images**: https://github.com/pilshchikov/homebrew-go/pkgs/container/homebrew-go

## 🐛 Troubleshooting

### Release Not Appearing

1. **Check Actions Tab**: Go to GitHub Actions and look for failed workflows
2. **Check Permissions**: Ensure repository has "Read and write" permissions for Actions
3. **Verify Authentication**: Run `gh auth status` to check GitHub CLI auth

### Build Failures

1. **Test Failures**: The workflow runs tests first - fix any failing tests
2. **Permission Issues**: Check if `GITHUB_TOKEN` has proper permissions
3. **GoReleaser Issues**: Check `.goreleaser.yml` configuration

### Script Issues

```bash
# Check if you're in the right directory
pwd
git remote -v

# Verify GitHub CLI
gh auth status
gh repo view

# Check repository status
./scripts/release.sh --status
```

## 🎉 Success!

After a successful release:

- ✅ GitHub release created with version `YYYY.M.DD.BUILD_ID`
- ✅ Binaries available for download
- ✅ Docker image published
- ✅ Release notes generated
- ✅ All artifacts include checksums

## 📋 Example Workflow

```bash
# 1. Make your changes and commit
git add .
git commit -m "feat: add new feature"
git push

# 2. Create release
./scripts/release.sh

# 3. Monitor progress
# Script will automatically open GitHub Actions in browser

# 4. Check release
gh release list
```

That's it! The release system handles everything else automatically. 🚀