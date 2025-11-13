# Contributing to Achievo

Thank you for your interest in contributing to Achievo! This document provides guidelines for contributing.

## Development Setup

1. Fork the repository
2. Clone your fork: `git clone https://github.com/yourusername/Achievo.git`
3. Install dependencies: `go mod download`
4. Build: `go build -o achievo.exe ./cmd/achievo`
5. Run tests: `go test ./...`

## Code Style

- Follow Go conventions and idioms
- Use `go fmt` to format code
- Run `go vet ./...` before committing
- Write unit tests for new features
- Add comments for exported functions and types

## Pull Request Process

1. Create a feature branch from `main`
2. Make your changes
3. Add tests if applicable
4. Ensure all tests pass: `go test ./...`
5. Update documentation if needed
6. Submit a pull request with a clear description

## Areas for Contribution

- Game rule files for popular games
- LLM prompt improvements
- Additional detection methods
- Performance optimizations
- Documentation improvements
- Bug fixes

## Questions?

Open an issue on GitHub for questions or discussions.

