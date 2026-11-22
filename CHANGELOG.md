# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## Unreleased

### Added
- Added `Caller`, `Transport`, call middleware, generic result decoding, and `Lister` interfaces for a transport-independent core API.
- Added `goubus.List` to inspect objects and method argument signatures over both HTTP JSON-RPC and Unix sockets, with normalized argument types.
- Added flexible JSON types for boolean, scalar-or-list, quoted integer, and Unix timestamp representations.
- Added 15 independent typed service packages and the `services.Client` facade for sharing one caller across the service clients.
- Added runtime compatibility handling for LuCI time methods and DHCP DUID argument formats, with options to pin known firmware behavior.
- Added an API snapshot gate, leak checks, fixture-driven service tests, and redesigned architecture and design documentation.

### Changed
- **BREAKING:** Replaced hardware-specific `profiles/<device>/<service>` imports with `github.com/honeybbq/goubus/v2/services/<service>`. Construct a service with its package's `New(caller)`, or use `services.New(caller)` for the complete facade. No per-device profile is required.
- **BREAKING:** Replaced the old transport-oriented call contract with `Caller.Call(ctx, service, method, data) (Result, error)`. Transports implement `Caller`; `Transport` adds `Close` for connection owners. Configure transports with constructor options instead of `SetLogger`.
- Moved public call errors and status sentinels to the root `goubus` package. Call failures now include `*CallError` before payload decoding, including status-only calls; inspect causes with `errors.Is` and fields with `errors.As`.
- Added real response coverage for OpenWrt 24.10 and 25.12 samples. Regression tests now decode network interface/device and wireless status, LuCI features and process data, mount/block/USB/board data, DHCP lease variants, and other captured service responses across the fixture sets.
- Set default transport logging to discard and redact RPC credentials and session data in the documented logging paths.
- Updated CI and releases to pin GitHub Actions by SHA, check the public API snapshot, run race tests before release, and generate release notes from the matching changelog section.

### Fixed
- Check ubus response status before returning a result, so `*CallError` is reported even when a caller does not decode a response body.
- Decode firmware response variants covered by real fixtures, including scalar or object LuCI feature flags, string-valued process fields, unsigned mount/device counters, board port lists, and differing DHCPv6 address keys and nesting.
- Adapt interface status to fall back to the interface dump when `network.interface.<name>.status` is unavailable.

### Removed
- Removed `profiles/*` and the duplicated `internal/base` service-manager layer. Migrate imports to the matching `services/<name>` package; use `services.New(caller)` when one aggregate is more convenient.
- Removed the legacy public `errdefs` package and its helper functions. Import error sentinels from `github.com/honeybbq/goubus/v2` and use the standard `errors.Is` / `errors.As` APIs.
- Removed GoReleaser configuration. The release workflow now builds and tests the module, checks the API snapshot, and publishes notes extracted from `CHANGELOG.md`.

## v2.0.0-alpha2 (2026-08-01)

### Added
- Added `NewSocketClientFromConn` for creating a socket transport from an existing connection. Thanks to [@mook](https://github.com/mook) for the contribution in [#6](https://github.com/HoneyBBQ/goubus/pull/6).

### Fixed
- Made the socket HELLO handshake honor context cancellation and close the owned connection when initialization fails.
- Restored reproducible lint checks by pinning `golangci-lint` to v2.12.2 and resolving the existing findings.

### Changed
- Updated the GitHub Actions used for checkout, Go setup, coverage uploads, and releases.

## v2.0.0-alpha1 (2026-01-18)

### Added
- Explicit hardware profile architecture: different hardware/firmware versions are now separated into their own packages under `profiles/` (e.g., `goubus/profiles/x86_generic`, `goubus/profiles/cmcc_rax3000m`).
- `internal/base` core layer: shared service logic is centralized and configurable via dialects.
- Strong type definitions for hardware-specific requests (e.g., `dhcp.AddLeaseRequest`).
- Structured logging support using `log/slog` throughout the library and examples.
- Comprehensive GoDoc documentation for all public APIs.
- Improved error handling with a specialized `errdefs` package and consistent error wrapping.

### Changed
- **BREAKING:** Module path updated to `github.com/honeybbq/goubus/v2`.
- **BREAKING:** Unified `Transport` interface for better abstraction.
- **BREAKING:** Relocated transport implementations from `transport/` subpackage to the root `goubus` package for cleaner API access.
- Switched from Apache License 2.0 to MIT.
- Refactored examples to use `slog` instead of `fmt` or `log`.
- Updated the Go version requirement to 1.25.

### Removed
- Removed the old `transport/` package.
