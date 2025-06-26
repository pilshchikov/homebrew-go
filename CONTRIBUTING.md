# Contributing to Homebrew Go

Thank you for your interest in contributing to Homebrew Go! This project welcomes contributions from developers of all experience levels.

## Table of Contents

- [Getting Started](#getting-started)
- [Development Setup](#development-setup)
- [Building and Testing](#building-and-testing)
- [Code Style and Guidelines](#code-style-and-guidelines)
- [Debugging](#debugging)
- [Submitting Changes](#submitting-changes)
- [Project Structure](#project-structure)

## Getting Started

### Prerequisites

- **Go 1.21 or higher** - Install from [golang.org](https://golang.org/dl/)
- **Git** - For version control
- **Make** - For build automation
- **GitHub CLI** (optional) - For easier PR management

### Fork and Clone

1. Fork the repository on GitHub: [https://github.com/pilshchikov/homebrew-go](https://github.com/pilshchikov/homebrew-go)
2. Clone your fork:

```bash
git clone https://github.com/your-username/homebrew-go.git
cd homebrew-go
```

3. Add upstream remote:

```bash
git remote add upstream https://github.com/pilshchikov/homebrew-go.git
```

4. Create the `build` directory:

```bash
mkdir -p build
```

## Development Setup

### Install Dependencies

```bash
# Download Go modules
go mod download

# Verify dependencies
go mod verify

# Install Go development tools
go install golang.org/x/tools/cmd/goimports@latest
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
go install github.com/goreleaser/goreleaser@latest
```

### Environment Setup

Set up your development environment variables:

```bash
# Optional: Set custom Homebrew prefix for testing
export HOMEBREW_PREFIX="$HOME/.local/homebrew-go"

# Optional: Enable debug logging
export HOMEBREW_DEBUG=1

# Optional: Disable analytics during development
export HOMEBREW_NO_ANALYTICS=1
```

## Building and Testing

### Building the Project

```bash
# Build the main binary
go build -o build/brew ./cmd/brew

# Build with race detection (for development)
go build -race -o build/brew ./cmd/brew

# Build for specific platforms (cross-compilation)
GOOS=linux GOARCH=amd64 go build -o build/brew-linux-amd64 ./cmd/brew
GOOS=darwin GOARCH=arm64 go build -o build/brew-darwin-arm64 ./cmd/brew
GOOS=darwin GOARCH=amd64 go build -o build/brew-darwin-amd64 ./cmd/brew

# Test GoReleaser build (requires GoReleaser installed)
goreleaser build --snapshot --clean
```

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run tests with coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Run specific test packages
go test ./internal/cmd/...
go test ./internal/api/...

# Run integration tests
go test ./tests/integration/...

# Run tests with race detection
go test -race ./...
```

### Running Locally

```bash
# Run the built binary
./build/brew --version
./build/brew help

# Test basic commands
./build/brew search git
./build/brew info node
```

## Code Style and Guidelines

### Go Standards

- Follow [Effective Go](https://golang.org/doc/effective_go.html) guidelines
- Use `gofmt` for formatting (handled automatically by editors)
- Use `goimports` for import management
- Follow Go naming conventions (exported vs unexported)

### Linting

```bash
# Run linter
golangci-lint run

# Fix automatically fixable issues
golangci-lint run --fix

# Check specific directories
golangci-lint run ./internal/cmd/...

# Format code
gofmt -w .
goimports -w .
```

### Code Organization

- Keep functions small and focused
- Use descriptive variable and function names
- Add comments for exported functions and complex logic
- Group related functionality in packages
- Minimize dependencies between packages

### Testing Guidelines

- Write tests for all new functionality
- Use table-driven tests where appropriate
- Mock external dependencies
- Test error conditions
- Aim for >80% test coverage

Example test structure:

```go
func TestInstallCommand(t *testing.T) {
    tests := []struct {
        name     string
        args     []string
        wantErr  bool
        expected string
    }{
        {
            name:     "valid formula",
            args:     []string{"git"},
            wantErr:  false,
            expected: "Successfully installed git",
        },
        // Add more test cases...
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

## Debugging

### Debug Logging

Enable debug output:

```bash
export HOMEBREW_DEBUG=1
./build/brew install git
```

### Using Delve Debugger

```bash
# Install delve
go install github.com/go-delve/delve/cmd/dlv@latest

# Debug the application
dlv debug ./cmd/brew -- install git

# Debug tests
dlv test ./internal/cmd/
```

### Profiling

```bash
# CPU profiling
go build -o build/brew ./cmd/brew
./build/brew --cpuprofile=cpu.prof install git
go tool pprof cpu.prof

# Memory profiling
./build/brew --memprofile=mem.prof install git
go tool pprof mem.prof
```

### Common Debug Scenarios

1. **API Request Issues**: Check network connectivity and API endpoints
2. **File Permission Errors**: Verify installation directories and permissions  
3. **Formula Parsing**: Validate JSON/YAML parsing with sample data
4. **Concurrency Issues**: Use race detector: `go build -race`

## Submitting Changes

### Before Submitting

1. **Test thoroughly**:
   ```bash
   go test ./...
   golangci-lint run
   go build -o build/brew ./cmd/brew
   ```

2. **Update documentation** if needed
3. **Add/update tests** for your changes
4. **Run integration tests** if applicable

### Creating a Pull Request

1. **Create a feature branch**:
   ```bash
   git checkout -b feature/your-feature-name
   ```

2. **Make your changes** with clear, focused commits
3. **Push to your fork**:
   ```bash
   git push origin feature/your-feature-name
   ```

4. **Open a Pull Request** with:
   - Clear title describing the change
   - Detailed description of what and why
   - Reference any related issues
   - Screenshots/examples if applicable

### PR Guidelines

- **Keep PRs focused** - One feature/fix per PR
- **Write clear commit messages** - Use conventional commits format
- **Update tests** - Ensure all tests pass
- **Document breaking changes** - If any
- **Be responsive** - Address review feedback promptly

### Commit Message Format

```
type(scope): brief description

Longer description if needed, explaining what and why.

Fixes #123
```

Types: `feat`, `fix`, `docs`, `style`, `refactor`, `test`, `chore`

## Project Structure

```
homebrew-go/
├── cmd/brew/           # Main application entry point
├── internal/
│   ├── api/           # API client and data structures
│   ├── cmd/           # Command implementations
│   ├── config/        # Configuration management
│   ├── errors/        # Error types and handling
│   ├── formula/       # Formula parsing and management
│   ├── installer/     # Package installation logic
│   ├── logger/        # Logging utilities
│   ├── tap/           # Tap management
│   ├── utils/         # Utility functions
│   └── verification/  # Package verification
├── tests/
│   └── integration/   # Integration tests
├── completions/       # Shell completions
├── manpages/          # Manual pages
├── .github/           # CI/CD workflows
├── docs/              # Documentation (minimal)
└── Makefile           # Build automation
```

### Key Packages

- **`cmd/brew`** - CLI entry point and command parsing
- **`internal/cmd`** - Individual command implementations
- **`internal/api`** - Homebrew API client
- **`internal/installer`** - Package installation logic
- **`internal/formula`** - Formula parsing and validation

## Getting Help

- **Issues**: Report bugs or request features via [GitHub Issues](https://github.com/pilshchikov/homebrew-go/issues)
- **Discussions**: Ask questions in [GitHub Discussions](https://github.com/pilshchikov/homebrew-go/discussions)
- **Discord/Slack**: Join our community chat (if available)

## Development Resources

- [Go Documentation](https://golang.org/doc/)
- [Homebrew API Documentation](https://formulae.brew.sh/docs/api/)
- [Original Homebrew Source](https://github.com/Homebrew/brew)
- [Go Testing Guide](https://golang.org/doc/tutorial/add-a-test)

Thank you for contributing to Homebrew Go! 🍺
