# Contributing to WAL Job Queue

Thank you for your interest in contributing! This document provides guidelines for contributing to the project.

## Code of Conduct

Please be respectful and constructive in all interactions.

## How to Contribute

### Reporting Bugs

- Use the GitHub issue tracker
- Include detailed steps to reproduce
- Specify your environment (OS, Go version, etc.)

### Suggesting Features

- Open an issue with a clear description
- Explain the use case and benefits
- Be open to discussion

### Pull Requests

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes
4. Add tests for new functionality
5. Ensure all tests pass (`go test ./...`)
6. Commit with clear messages (`git commit -s -m 'Add amazing feature'`)
7. Push to your fork
8. Open a Pull Request

### Coding Standards

- Follow standard Go conventions
- Use `gofmt` for formatting
- Add comments for exported functions
- Write unit tests for new code
- Keep functions focused and small

### Commit Messages

- Use present tense ("Add feature" not "Added feature")
- Keep first line under 50 characters
- Reference issues when applicable
- Sign your commits (`-s` flag)

## Development Setup

```bash
# Clone your fork
git clone https://github.com/YOUR_USERNAME/distributed-job-scheduler.git

# Install dependencies
go mod download

# Run tests
go test ./...

# Run locally
go run cmd/server/main.go
```

## Testing

- Write table-driven tests where appropriate
- Aim for >80% code coverage
- Test edge cases and error conditions

## Questions?

Open an issue for questions or clarifications.

Thank you for contributing!
