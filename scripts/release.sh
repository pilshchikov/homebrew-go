#!/bin/bash

# Simplified Release Script for Homebrew Go
# This script triggers the GitHub Actions release workflow

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to show current status
show_status() {
    print_status "Repository Status:"
    echo "  Branch: $(git branch --show-current)"
    echo "  Latest commit: $(git log -1 --oneline)"
    echo "  Existing releases: $(git tag -l | wc -l) releases"
    if [[ $(git tag -l | wc -l) -gt 0 ]]; then
        echo "  Latest release: $(git tag -l | sort -V | tail -1)"
    fi
    echo
}

# Function to trigger release
trigger_release() {
    local prerelease=${1:-false}
    
    print_status "Triggering release workflow..."
    
    # Check if we're in a git repo
    if ! git rev-parse --git-dir > /dev/null 2>&1; then
        print_error "Not in a git repository!"
        exit 1
    fi
    
    # Check for uncommitted changes
    if [[ -n $(git status --porcelain) ]]; then
        print_warning "You have uncommitted changes:"
        git status --short
        read -p "Do you want to continue? (y/N): " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            print_error "Aborted"
            exit 1
        fi
    fi
    
    # Check if gh CLI is available
    if ! command -v gh &> /dev/null; then
        print_error "GitHub CLI (gh) is not installed!"
        echo "Install it with: brew install gh"
        echo "Or download from: https://cli.github.com/"
        exit 1
    fi
    
    # Check if user is authenticated
    if ! gh auth status &> /dev/null; then
        print_error "You're not authenticated with GitHub CLI!"
        echo "Run: gh auth login"
        exit 1
    fi
    
    # Push any local commits
    current_branch=$(git branch --show-current)
    if [[ "$current_branch" != "main" ]]; then
        print_warning "You're not on main branch (currently on: $current_branch)"
        read -p "Do you want to continue? (y/N): " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            print_error "Aborted"
            exit 1
        fi
    fi
    
    print_status "Pushing latest changes to $current_branch..."
    git push origin "$current_branch"
    
    # Trigger the workflow
    print_status "Triggering GitHub Actions release workflow..."
    
    if [[ "$prerelease" == "true" ]]; then
        gh workflow run release.yml --field prerelease=true
        print_status "Release workflow triggered with pre-release flag"
    else
        gh workflow run release.yml
        print_status "Release workflow triggered"
    fi
    
    print_success "Release workflow started!"
    echo
    print_status "The release will be automatically versioned as: $(date '+%Y.%-m.%d').<build_number>"
    print_status "Monitor progress at: https://github.com/pilshchikov/homebrew-go/actions"
    
    # Wait a moment and show actions
    sleep 2
    print_status "Recent workflow runs:"
    gh run list --limit 3
    
    # Try to open the actions page
    if command -v open >/dev/null 2>&1; then
        print_status "Opening GitHub Actions in 3 seconds..."
        sleep 3
        open "https://github.com/pilshchikov/homebrew-go/actions"
    fi
}

# Function to show help
show_help() {
    echo "Homebrew Go Release Script"
    echo
    echo "This script triggers automatic releases with date-based versioning (YYYY.M.DD.BUILD_ID)"
    echo
    echo "Usage: $0 [OPTIONS]"
    echo
    echo "Options:"
    echo "  -h, --help       Show this help message"
    echo "  -s, --status     Show repository status"
    echo "  -p, --prerelease Create as pre-release"
    echo
    echo "Examples:"
    echo "  $0                # Create regular release"
    echo "  $0 --prerelease   # Create pre-release"
    echo "  $0 --status       # Show status"
    echo
    echo "Version Format:"
    echo "  Releases are automatically versioned as: YYYY.M.DD.BUILD_ID"
    echo "  Example: 2025.6.26.123 (June 26, 2025, build #123)"
    echo
    echo "The script will:"
    echo "  1. Check repository status"
    echo "  2. Push any local changes"
    echo "  3. Trigger GitHub Actions release workflow"
    echo "  4. The workflow will automatically:"
    echo "     - Generate version number"
    echo "     - Run tests"
    echo "     - Create git tag"
    echo "     - Build cross-platform binaries"
    echo "     - Create GitHub release"
    echo "     - Build and push Docker images"
}

# Parse command line arguments
PRERELEASE="false"
while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--help)
            show_help
            exit 0
            ;;
        -s|--status)
            show_status
            exit 0
            ;;
        -p|--prerelease)
            PRERELEASE="true"
            shift
            ;;
        -*)
            print_error "Unknown option: $1"
            show_help
            exit 1
            ;;
        *)
            print_error "Unexpected argument: $1"
            show_help
            exit 1
            ;;
    esac
done

# Show current status
show_status

# Trigger release
trigger_release "$PRERELEASE"