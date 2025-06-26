#!/bin/bash
set -e

# Go development environment setup
echo "Setting up Go development environment..."

# Update package lists
sudo apt-get update

apt_get_install() {
  sudo apt-get install -y \
    -o Dpkg::Options::=--force-confdef \
    -o Dpkg::Options::=--force-confnew \
    "$@"
}

# Install essential development tools
apt_get_install \
  build-essential \
  git \
  curl \
  wget \
  unzip \
  shellcheck \
  openssh-server \
  zsh

# Install Go tools
echo "Installing Go development tools..."
go install golang.org/x/tools/cmd/goimports@latest
go install golang.org/x/tools/cmd/godoc@latest
go install golang.org/x/tools/cmd/goyacc@latest
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
go install github.com/securecodewarrior/github-action-gosec@latest
go install github.com/sonatypecommunity/nancy@latest

# Install GitHub CLI for development workflow
if ! command -v gh &> /dev/null; then
  curl -fsSL https://cli.github.com/packages/githubcli-archive-keyring.gpg | sudo dd of=/usr/share/keyrings/githubcli-archive-keyring.gpg
  sudo chmod go+r /usr/share/keyrings/githubcli-archive-keyring.gpg
  echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/githubcli-archive-keyring.gpg] https://cli.github.com/packages stable main" | sudo tee /etc/apt/sources.list.d/github-cli.list > /dev/null
  sudo apt-get update
  apt_get_install gh
fi

# Download Go module dependencies
echo "Downloading Go module dependencies..."
go mod download
go mod verify

# Build the project to ensure everything works
echo "Building project..."
make build || go build -o build/brew ./cmd/brew

# Run tests to verify setup
echo "Running tests..."
go test -v ./... || echo "Some tests failed, but environment is set up"

# Start the SSH server so that `gh cs ssh` works
sudo service ssh start

echo "Go development environment setup complete!"
echo "Available commands:"
echo "  make build    - Build the project"
echo "  make test     - Run tests"
echo "  make coverage - Generate test coverage"
echo "  make lint     - Run linters"
