#!/bin/bash
# Homebrew Go - Simple Installation Script
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Default values
BINARY_NAME="brew-go"
INSTALL_DIR="$HOME/.local/bin"

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --name)
            BINARY_NAME="$2"
            shift 2
            ;;
        --dir)
            INSTALL_DIR="$2"
            shift 2
            ;;
        --help|-h)
            echo "Homebrew Go Installer"
            echo ""
            echo "Usage: $0 [OPTIONS]"
            echo ""
            echo "Options:"
            echo "  --name NAME    Install binary with custom name (default: brew-go)"
            echo "  --dir DIR      Install to custom directory (default: ~/.local/bin)"
            echo "  --help, -h     Show this help message"
            echo ""
            echo "Examples:"
            echo "  $0                                    # Install as 'brew-go'"
            echo "  $0 --name hbrew                      # Install as 'hbrew'"
            echo "  $0 --name brew                       # Install as 'brew'"
            echo "  $0 --name brew --dir /usr/local/bin  # Install as 'brew' in /usr/local/bin"
            echo ""
            echo "Note: You may need sudo for system directories like /usr/local/bin"
            exit 0
            ;;
        *)
            echo -e "${RED}Error: Unknown option $1${NC}"
            echo "Use --help for usage information"
            exit 1
            ;;
    esac
done

# Detect OS and Architecture
OS="$(uname -s)"
ARCH="$(uname -m)"

echo -e "${GREEN}🍺 Homebrew Go Installer${NC}"
echo "Detecting system: $OS $ARCH"
echo -e "${BLUE}Binary name: $BINARY_NAME${NC}"
echo -e "${BLUE}Install directory: $INSTALL_DIR${NC}"

# Map to GoReleaser naming convention
case "$OS" in
    "Darwin")
        OS_NAME="Darwin"
        ;;
    "Linux")
        OS_NAME="Linux"
        ;;
    *)
        echo -e "${RED}Error: Unsupported OS: $OS${NC}"
        exit 1
        ;;
esac

case "$ARCH" in
    "x86_64"|"amd64")
        ARCH_NAME="x86_64"
        ;;
    "arm64"|"aarch64")
        ARCH_NAME="arm64"
        ;;
    *)
        echo -e "${RED}Error: Unsupported architecture: $ARCH${NC}"
        exit 1
        ;;
esac

# Get latest release
echo "Fetching latest release..."
LATEST_TAG=$(curl -s https://api.github.com/repos/pilshchikov/homebrew-go/releases/latest | grep '"tag_name"' | cut -d'"' -f4)

if [ -z "$LATEST_TAG" ]; then
    echo -e "${RED}Error: Could not fetch latest release${NC}"
    exit 1
fi

echo "Latest version: $LATEST_TAG"

# Download URL
FILENAME="homebrew-go_${OS_NAME}_${ARCH_NAME}.tar.gz"
DOWNLOAD_URL="https://github.com/pilshchikov/homebrew-go/releases/download/${LATEST_TAG}/${FILENAME}"

echo "Downloading: $DOWNLOAD_URL"

# Create temporary directory
TMP_DIR=$(mktemp -d)
cd "$TMP_DIR"

# Download and extract
curl -L "$DOWNLOAD_URL" | tar xz

# Create install directory if it doesn't exist
mkdir -p "$INSTALL_DIR"

# Check if we need sudo for the install directory
if [ ! -w "$INSTALL_DIR" ]; then
    echo -e "${YELLOW}Note: $INSTALL_DIR requires sudo access${NC}"
    sudo mv brew "$INSTALL_DIR/$BINARY_NAME"
else
    # Move binary
    mv brew "$INSTALL_DIR/$BINARY_NAME"
fi

echo -e "${GREEN}✅ Homebrew Go installed successfully!${NC}"
echo
echo "Binary installed to: $INSTALL_DIR/$BINARY_NAME"
echo

# Check if install directory is in PATH and add it automatically
if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
    echo -e "${YELLOW}📝 Adding $INSTALL_DIR to PATH...${NC}"
    
    # Detect shell and add to appropriate rc file
    if [[ "$SHELL" == *"zsh"* ]] || [[ -n "$ZSH_VERSION" ]]; then
        SHELL_RC="$HOME/.zshrc"
    elif [[ "$SHELL" == *"bash"* ]] || [[ -n "$BASH_VERSION" ]]; then
        SHELL_RC="$HOME/.bashrc"
    else
        SHELL_RC="$HOME/.profile"
    fi
    
    # Add PATH export if not already there
    if [[ "$INSTALL_DIR" == "$HOME/.local/bin" ]]; then
        PATH_EXPORT='export PATH="$HOME/.local/bin:$PATH"'
    else
        PATH_EXPORT="export PATH=\"$INSTALL_DIR:\$PATH\""
    fi
    
    if ! grep -q "$PATH_EXPORT" "$SHELL_RC" 2>/dev/null; then
        echo "$PATH_EXPORT" >> "$SHELL_RC"
        echo -e "${GREEN}✅ Added to $SHELL_RC${NC}"
        echo -e "${YELLOW}⚠️  Run 'source $SHELL_RC' or restart your terminal to use $BINARY_NAME${NC}"
    else
        echo -e "${BLUE}ℹ️  PATH already configured in $SHELL_RC${NC}"
    fi
    echo
fi

echo -e "${YELLOW}🚀 Try it now:${NC}"
echo "$INSTALL_DIR/$BINARY_NAME --version"

# Cleanup
cd /
rm -rf "$TMP_DIR"