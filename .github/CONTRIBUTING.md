# Contributing to goubus

First off, thank you for considering contributing to `goubus`! It's people like you that make the open-source community such a great place.

## How Can I Contribute?

### Reporting Bugs
- Use the **Bug Report** template when opening an issue.
- Describe the bug in detail and provide steps to reproduce it.
- Include information about your OpenWrt version and architecture.

### Suggesting Enhancements
- Use the **Feature Request** template.
- Explain why this enhancement would be useful to most users.
- If it's a new ubus service support, please provide sample `ubus call` outputs.

### Pull Requests
1. Fork the repository and create your branch from `main`.
2. If you've added code that should be tested, add tests in `_test.go` files.
3. Ensure the test suite passes (`go test ./...`).
4. Make sure your code lints (`golangci-lint run`).
5. Use descriptive commit messages following the [Conventional Commits](https://www.conventionalcommits.org/) specification.

## Development Setup

### Prerequisites
- Go 1.25 or later
- `golangci-lint` (for linting)
- Access to an OpenWrt device or a mock environment for testing

### Quality Checks
Before committing your changes, please run:
```bash
go fmt ./...
golangci-lint run
go test ./...
```

## Style Guide
- Follow the standard Go coding conventions (Effective Go).
- All exported functions and types MUST have GoDoc comments.
- Keep functions focused and small.
- Use `errdefs.Wrapf` or `%w` for error wrapping to preserve error chains.
- Prefer `log/slog` for any logging needs.

## License
By contributing, you agree that your contributions will be licensed under the MIT License.
