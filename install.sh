#!/bin/bash
# Homebrew Go Installation Script
# 
# This script installs Homebrew Go, a complete rewrite of Homebrew from Ruby to Go
# Created by Stepan Pilshchikov (https://github.com/pilshchikov)
#
# Usage: /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/pilshchikov/homebrew-go/main/install.sh)"

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# GitHub repository information
GITHUB_REPO="pilshchikov/homebrew-go"
GITHUB_API_URL="https://api.github.com/repos/${GITHUB_REPO}"
GITHUB_RELEASES_URL="${GITHUB_API_URL}/releases/latest"

# Installation prefix
DEFAULT_PREFIX="/usr/local"
if [[ "$(uname -m)" == "arm64" && "$(uname -s)" == "Darwin" ]]; then
    DEFAULT_PREFIX="/opt/homebrew"
fi

# Override prefix if HOMEBREW_PREFIX is set
PREFIX="${HOMEBREW_PREFIX:-$DEFAULT_PREFIX}"
BINDIR="${PREFIX}/bin"

# Logging functions
log_info() {
    echo -e "${BLUE}==> ${NC}$1"
}

log_success() {
    echo -e "${GREEN}==> ${NC}$1"
}

log_warning() {
    echo -e "${YELLOW}==> Warning: ${NC}$1"
}

log_error() {
    echo -e "${RED}==> Error: ${NC}$1" >&2
}

# Check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Detect platform and architecture
detect_platform() {
    local os arch
    
    # Detect OS
    case "$(uname -s)" in
        Darwin) os="Darwin" ;;
        Linux)  os="Linux" ;;
        *)
            log_error "Unsupported operating system: $(uname -s)"
            log_error "Homebrew Go supports macOS (Darwin) and Linux only."
            exit 1
            ;;
    esac
    
    # Detect architecture
    case "$(uname -m)" in
        x86_64|amd64) arch="x86_64" ;;
        arm64|aarch64) arch="arm64" ;;
        *)
            log_error "Unsupported architecture: $(uname -m)"
            log_error "Homebrew Go supports x86_64 and arm64 architectures only."
            exit 1
            ;;
    esac
    
    echo "${os}_${arch}"
}

# Check prerequisites
check_prerequisites() {
    log_info "Checking prerequisites..."
    
    # Check for required commands
    local missing_commands=()
    
    if ! command_exists curl; then
        missing_commands+=("curl")
    fi
    
    if ! command_exists tar; then
        missing_commands+=("tar")
    fi
    
    if ! command_exists mkdir; then
        missing_commands+=("mkdir")
    fi
    
    if [[ ${#missing_commands[@]} -gt 0 ]]; then
        log_error "Missing required commands: ${missing_commands[*]}"
        log_error "Please install these commands and try again."
        exit 1
    fi
    
    log_success "Prerequisites check passed"
}

# Get latest release information
get_latest_release() {
    log_info "Fetching latest release information..."
    
    local response
    if ! response=$(curl -s "${GITHUB_RELEASES_URL}"); then
        log_error "Failed to fetch release information from GitHub"
        log_error "Please check your internet connection and try again."
        exit 1
    fi
    
    # Extract tag name and download URL
    local tag_name download_url
    tag_name=$(echo "$response" | grep -o '"tag_name": *"[^"]*"' | cut -d'"' -f4)
    
    if [[ -z "$tag_name" ]]; then
        log_error "Could not parse release information"
        log_error "Please visit https://github.com/${GITHUB_REPO}/releases to download manually."
        exit 1
    fi
    
    local platform
    platform=$(detect_platform)
    local archive_name="homebrew-go_${platform}.tar.gz"
    
    download_url=$(echo "$response" | grep -o '"browser_download_url": *"[^"]*'"$archive_name"'"' | cut -d'"' -f4)
    
    if [[ -z "$download_url" ]]; then
        log_error "Could not find download URL for platform: $platform"
        log_error "Available releases:"
        echo "$response" | grep -o '"name": *"homebrew-go_[^"]*"' | cut -d'"' -f4 | sed 's/^/  /'
        exit 1
    fi
    
    echo "$tag_name|$download_url"
}

# Check if directory is writable
check_write_permission() {
    local dir="$1"
    
    if [[ ! -d "$dir" ]]; then
        # Try to create directory
        if ! mkdir -p "$dir" 2>/dev/null; then
            return 1
        fi
    fi
    
    # Test write permission
    if ! touch "${dir}/.homebrew-go-test" 2>/dev/null; then
        return 1
    fi
    
    rm -f "${dir}/.homebrew-go-test"
    return 0
}

# Check if we need sudo
check_sudo_needed() {
    if check_write_permission "$BINDIR"; then
        echo "false"
    else
        echo "true"
    fi
}

# Download and install
install_homebrew_go() {
    local tag_and_url="$1"
    local tag_name="${tag_and_url%|*}"
    local download_url="${tag_and_url#*|}"
    
    log_info "Installing Homebrew Go ${tag_name}..."
    log_info "Platform: $(detect_platform)"
    log_info "Install location: ${BINDIR}/brew"
    
    # Create temporary directory
    local temp_dir
    temp_dir=$(mktemp -d)
    trap "rm -rf '$temp_dir'" EXIT
    
    # Download archive
    log_info "Downloading from: $download_url"
    local archive_path="${temp_dir}/homebrew-go.tar.gz"
    
    if ! curl -L -o "$archive_path" "$download_url"; then
        log_error "Failed to download Homebrew Go"
        exit 1
    fi
    
    # Extract archive
    log_info "Extracting archive..."
    if ! tar -xzf "$archive_path" -C "$temp_dir"; then
        log_error "Failed to extract archive"
        exit 1
    fi
    
    # Find the binary
    local binary_path
    binary_path=$(find "$temp_dir" -name "brew" -type f | head -1)
    
    if [[ ! -f "$binary_path" ]]; then
        log_error "Could not find 'brew' binary in the downloaded archive"
        exit 1
    fi
    
    # Make binary executable
    chmod +x "$binary_path"
    
    # Check if we need sudo
    local use_sudo
    use_sudo=$(check_sudo_needed)
    
    # Install binary
    log_info "Installing binary to ${BINDIR}/brew..."
    
    if [[ "$use_sudo" == "true" ]]; then
        log_warning "Installation requires administrator privileges"
        if ! sudo mkdir -p "$BINDIR"; then
            log_error "Failed to create directory: $BINDIR"
            exit 1
        fi
        
        if ! sudo cp "$binary_path" "${BINDIR}/brew"; then
            log_error "Failed to install binary"
            exit 1
        fi
        
        sudo chmod +x "${BINDIR}/brew"
    else
        mkdir -p "$BINDIR"
        if ! cp "$binary_path" "${BINDIR}/brew"; then
            log_error "Failed to install binary"
            exit 1
        fi
    fi
    
    log_success "Homebrew Go ${tag_name} installed successfully!"
}

# Verify installation
verify_installation() {
    log_info "Verifying installation..."
    
    if ! command_exists brew; then
        log_warning "The 'brew' command is not in your PATH"
        log_info "Add ${BINDIR} to your PATH by adding this line to your shell profile:"
        log_info "  export PATH=\"${BINDIR}:\$PATH\""
        log_info ""
        log_info "For immediate use, run:"
        log_info "  export PATH=\"${BINDIR}:\$PATH\""
        return 1
    fi
    
    # Test brew command
    local version
    if version=$(brew --version 2>/dev/null); then
        log_success "Installation verified successfully!"
        log_info "Homebrew Go version: $(echo "$version" | head -1)"
        return 0
    else
        log_error "Installation verification failed"
        return 1
    fi
}

# Show next steps
show_next_steps() {
    log_info "Next steps:"
    echo "  1. Make sure ${BINDIR} is in your PATH"
    echo "  2. Run 'brew --version' to verify the installation"
    echo "  3. Run 'brew help' to see available commands"
    echo "  4. Try 'brew search git' to search for packages"
    echo ""
    log_info "Example commands:"
    echo "  brew install git       # Install git"
    echo "  brew search python     # Search for Python packages"
    echo "  brew info node         # Get information about Node.js"
    echo "  brew list              # List installed packages"
    echo ""
    log_info "Documentation: https://github.com/${GITHUB_REPO}"
    echo ""
    log_warning "Note: This is an independent educational project, not affiliated with the official Homebrew."
}

# Main installation flow
main() {
    cat << 'EOF'

██╗  ██╗ ██████╗ ███╗   ███╗███████╗██████╗ ██████╗ ███████╗██╗    ██╗     ██████╗  ██████╗ 
██║  ██║██╔═══██╗████╗ ████║██╔════╝██╔══██╗██╔══██╗██╔════╝██║    ██║    ██╔════╝ ██╔═══██╗
███████║██║   ██║██╔████╔██║█████╗  ██████╔╝██████╔╝█████╗  ██║ █╗ ██║    ██║  ███╗██║   ██║
██╔══██║██║   ██║██║╚██╔╝██║██╔══╝  ██╔══██╗██╔══██╗██╔══╝  ██║███╗██║    ██║   ██║██║   ██║
██║  ██║╚██████╔╝██║ ╚═╝ ██║███████╗██████╔╝██║  ██║███████╗╚███╔███╔╝    ╚██████╔╝╚██████╔╝
╚═╝  ╚═╝ ╚═════╝ ╚═╝     ╚═╝╚══════╝╚═════╝ ╚═╝  ╚═╝╚══════╝ ╚══╝╚══╝      ╚═════╝  ╚═════╝ 

EOF
    
    echo "Homebrew Go - A Go implementation of the Homebrew package manager"
    echo "Created by Stepan Pilshchikov (https://github.com/pilshchikov)"
    echo "Educational project - Not affiliated with the official Homebrew"
    echo ""
    
    # Check if already installed
    if command_exists brew; then
        local current_version
        current_version=$(brew --version 2>/dev/null | head -1 || echo "unknown")
        log_warning "Homebrew (or Homebrew Go) is already installed: $current_version"
        read -p "Do you want to continue and potentially overwrite it? (y/N): " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            log_info "Installation cancelled."
            exit 0
        fi
    fi
    
    check_prerequisites
    
    local release_info
    release_info=$(get_latest_release)
    
    install_homebrew_go "$release_info"
    
    if verify_installation; then
        show_next_steps
    else
        log_info "Installation completed, but verification failed."
        log_info "You may need to add ${BINDIR} to your PATH."
        show_next_steps
    fi
    
    log_success "Installation complete! 🍺"
}

# Handle Ctrl+C gracefully
trap 'log_error "Installation interrupted by user"; exit 130' INT

# Run main function
main "$@"