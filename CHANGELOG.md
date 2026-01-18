# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [2.0.0-alpha1] - 2026-01-18

### Added
- Explicit hardware profile architecture: different hardware/firmware versions are now separated into their own packages under `profiles/` (e.g., `goubus/profiles/x86_generic`, `goubus/profiles/cmcc_rax3000m`).
- `internal/base` core layer: shared service logic is now centralized and configurable via dialects.
- Strong type definitions for hardware-specific requests (e.g., `dhcp.AddLeaseRequest`).
- Structured logging support using `log/slog` throughout the library and examples.
- Comprehensive GoDoc documentation for all public APIs.
- Functional Options pattern for client initialization (planned).
- Improved error handling with specialized `errdefs` package and consistent error wrapping.

### Changed
- **BREAKING**: Module path updated to `github.com/honeybbq/goubus/v2`.
- **BREAKING**: Unified `Transport` interface for better abstraction.
- **BREAKING**: Relocated transport implementations from `transport/` subpackage to the root `goubus` package for cleaner API access.
- **LICENSE**: Switched from Apache License 2.0 to MIT License.
- Refactored all examples to use `slog` instead of `fmt` or `log`.
- Updated Go version requirement to 1.25.

### Removed
- Removed old `transport/` package.
