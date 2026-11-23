# Changelog

All notable changes to this module are documented here. This project follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## v2.0.0-beta.1 (2026-11-23)

### Added

- `Caller` (`Call(ctx, service, method, data) (Result, error)`), `CallerFunc`, and `NewJSONResult`: narrow call contract accepted by every service client and `Call[T]`, decoupling callers that borrow a connection from `Transport` (`Caller` plus `Close() error`).
- `CallMiddleware` and `ChainMiddleware(caller, middlewares...)` to wrap ubus calls in arrival order across both transports.
- `goubus.List(ctx, caller, patterns...)` and `Lister`, implemented by both `*RPCClient` and `*SocketClient`, returning `map[string]ObjectSignatures` with wire argument types normalized to `ArgType` (`ArgTypeString`, `ArgTypeNumber`, `ArgTypeBoolean`, `ArgTypeArray`, `ArgTypeObject`, `ArgTypeUnknown`). `SocketClient.List` also populates its `objpath -> objid` lookup cache.
- 15 hardware-independent typed service packages under `services/` (`container`, `dhcp`, `file`, `hostapd`, `log`, `luci`, `network`, `rc`, `rpcsys`, `service`, `session`, `system`, `uci`, `wireless`, `wpasupplicant`).
- Runtime adaptation for firmware differences, verified against fixtures from OpenWrt 24.10.5 (`x86_generic`) and OpenWrt 25.12.0-rc1 (CMCC RAX3000M):
  - `luci.Client.Time` tries `getUnixtime` (25.12), falls back to `getLocaltime` (24.10) on `ErrMethodNotFound`, `ErrInvalidCommand`, or `ErrPermissionDenied`, and atomically caches the working method. `luci.WithTimeMethod` pins one explicitly.
  - `dhcp.Client.AddLease` inspects `goubus.List(ctx, caller, "dhcp")` to send `duid` as `[]string` or `string`, falling back to array-then-string probing on `ErrInvalidParameter` when signature lookup is unavailable. `dhcp.WithDUIDFormat` pins `DUIDString` or `DUIDArray`.
  - `StringOrSlice`, `StringInt`, and `UnixTime` alongside `Bool` (`BoolValue`) in the root package to decode scalar-or-array strings, quoted integers, Unix-second or RFC3339 timestamps, and numeric or string booleans.
- `*goubus.CallError` (`Service`, `Method`, `Code`, `Message`, `Err`), `ErrCall`, `StatusOK` through `StatusConnectionFailed` (`0`..`10`), and `StatusError(code)` in the root package. `*CallError` unwraps to both `ErrCall` and the mapped status sentinel.
- `WithHTTPClient` and `WithRPCEndpoint` on `NewRPCClient`; `(*RPCClient).Authenticate(ctx)` to force a fresh `session.login`; `DialTimeout()`, `ReadTimeout()`, `WriteTimeout()`, and `PeerID()` accessors on `*SocketClient`; `Values` and `StateValues` on `*uci.OptionContext`.
- Public API snapshot gate (`api/*.txt` and `internal/tools/apidump` with internal-type leak checks), exported GoDoc coverage test (`doc_coverage_test.go`), and architecture/design documentation (`docs/architecture.md`, `docs/design.md`).

### Changed

- **Breaking:** Standardize RPC initialisms in the public API (`RPCClient`, `NewRPCClient`, `RPCOption`, and `WithRPCLogger`) and exposed service fields (`MAC`, `MACAddr`, `IfName`, and `PID`).
- **Breaking:** Minimum Go version is now 1.26.
- **Breaking:** `goubus.Call[T]` now returns `(T, error)` instead of `(*T, error)`. Pass a pointer type (such as `goubus.Call[*BoardInfo]`) when a struct pointer is desired, or decode directly into value, map, and slice types.
- **Breaking:** Hardware-specific `profiles/<device>/<service>` packages are replaced by `github.com/honeybbq/goubus/v2/services/<service>` (`wpa_supplicant` is now `wpasupplicant`). Every service type is named `Client` instead of `Manager`, constructed with `New(caller)`.
- **Breaking:** `Transport.SetLogger` is removed. Transports discard logs by default; pass `WithRPCLogger(logger)` to `NewRPCClient` or `WithSocketLogger(logger)` to `NewSocketClient` / `NewSocketClientFromConn`. `RPCOption`, `SocketOption`, `luci.Option`, and `dhcp.Option` are now sealed interfaces rather than function types.
- **Breaking:** Root sentinels (`ErrNotFound`, `ErrPermissionDenied`, `ErrInvalidParameter`, and the rest) move from `errdefs` to `github.com/honeybbq/goubus/v2`, and `UbusStatusOK`..`UbusStatusConnectionFailed` / `MapUbusCodeToError` are renamed to `StatusOK`..`StatusConnectionFailed` / `StatusError`. Match sentinels with `errors.Is` instead of `errdefs.Is*` helpers.
- **Breaking:** Redundant `Get` prefixes are dropped on service methods (`luci.Client.Version`, `Time`, `Processes`, `Features`, `BlockDevices`, `USBDevices`, `BoardJSON`, `DHCPLeases`, ...; `container.Client.Features`; `network.Client.ProtoHandlers` and `Wireless().Validate`; `service.Client.Data`; `hostapd.AP(name).Clients` and `Status`; `uci.Package(name).All` and `Section.First`).
- **Breaking:** `luci.Client.Features` returns `map[string]luci.Feature` (supporting both boolean flags on 24.10 and subfeature maps on 25.12), `BlockDevices` returns `map[string]luci.BlockDevice` keyed by device name, and `USBDevices` returns `*luci.USBInfo` (`Devices` and `Ports`).
- `uci.SectionValues` read and `MarshalJSON` methods (`First`, `Get`, `All`, `Len`, `Clone`, `MarshalJSON`) use value receivers so non-addressable `SectionValues` marshal cleanly, while mutators (`Set`, `SetList`, `SetScalar`, `Append`, `Delete`) use pointer receivers.
- `NewRPCClient` accepts `http://` and `https://` URLs (with an optional endpoint path) in addition to bare hostnames, defaults to a 30s HTTP client timeout, formats `String()`, `GoString()`, and `LogValue()` with `password` and `ubus_rpc_session` redacted, redacts outgoing RPC request bodies and `"session"` responses in debug logs, and clears credentials and calls `session.destroy` on `Close()`.
- `RPCClient.Call` does not retry `ErrPermissionDenied`, and successful calls extend the local session expiry by `rpcd`'s sliding idle timeout; call `(*RPCClient).Authenticate(ctx)` to replace the cached session via `session.login`.
- Incoming wire frames are bounded before allocation: `blobmsg.ReadMessage` enforces minimum header length and `blobmsg.MaxMessageBytes` (1 MiB) on Unix socket frames, and `RPCClient` caps HTTP response bodies at 4 MiB.
- CI and security workflows pin GitHub Actions by commit SHA, test against Go 1.26 and 1.27, cross-compile `linux/mipsle`, `linux/arm`, `linux/386`, and `linux/arm64`, run live OpenWrt 24.10 and 25.12 container integration tests, check `apidump`, `govulncheck`, CodeQL, and OpenSSF Scorecard, and build GitHub release notes from `CHANGELOG.md` via `scripts/release-notes.sh`.

### Fixed

- Both `RPCClient.Call` and `SocketClient.Call` now inspect the ubus status code (and JSON-RPC `error` object) before returning a `Result`, so status-only calls such as `system.Reboot` and `uci.Commit` report `*CallError` instead of silently succeeding on non-zero status codes.
- `SocketClient` applies read/write deadlines per round trip, honors `Call` context cancellation during in-flight socket reads, decodes integer-encoded boolean attributes and `BLOBMSG_TYPE_UNSPEC` (`null`) container entries in `blobmsg` payloads, and avoids 32-bit `int` constant overflow in `blobmsg` attribute encoding on `mipsle`, `arm`, and `386` targets.
- `RPCClient` hoists explicit `ubus_rpc_session` fields out of RPC payload objects into JSON-RPC `params[0]` for `uhttpd-mod-ubus`.
- `uci.PackageContext.Changes` accepts array-shaped `changes` responses from `rpcd`, and `uci.OptionContext.AddToList` / `DeleteFromList` treat `ErrNoData` alongside `ErrNotFound` as a missing option.
- `dhcp` lease decoders unwrap outer `"device"` and `"interfaces"` maps, stamp `Interface` onto returned leases, accept `"ip6addr"`, `"address"`, `"ipv6"`, and `"ipv6-addr"` keys on `IPv6Lease`, and accept both string addresses and numeric lengths in `IPv6RA.Prefix`.
- `network.InterfaceContext.Status` falls back to matching the interface in `network.interface` `dump` when `network.interface.<name>` `status` fails.
- `luci` decodes string-quoted `PID`/`PPID` values in `Processes`, unsigned 64-bit byte counters in `MountPoint` and `BlockDevice`, and `Ports` lists on `BoardInterface`.

### Removed

- `profiles/x86_generic/*`, `profiles/cmcc_rax3000m/*`, and `internal/base/*`.
- Public `errdefs` package and its `Is*` / `Wrapf` helpers.
- `.goreleaser.yaml` (goubus is a Go library module without release binaries).

### Migrating from v2.0.0-alpha1 and v2.0.0-alpha2

| Before (`v2.0.0-alpha1` / `alpha2`) | Now |
| --- | --- |
| `github.com/honeybbq/goubus/v2/profiles/<device>/<svc>` | `github.com/honeybbq/goubus/v2/services/<svc>` |
| `profiles/<device>/wpa_supplicant` | `github.com/honeybbq/goubus/v2/services/wpasupplicant` |
| `*<svc>.Manager` (`system.New(transport)`) | `*<svc>.Client` (`system.New(caller)`) |
| `RpcClient`, `NewRpcClient`, `RpcOption`, `WithRpcLogger` | `RPCClient`, `NewRPCClient`, `RPCOption`, `WithRPCLogger` |
| `goubus.Call[T](ctx, transport, ...)` -> `(*T, error)` | `goubus.Call[T](ctx, caller, ...)` -> `(T, error)` (`Call[*T]` for struct pointers) |
| `transport.SetLogger(logger)` | `goubus.WithRPCLogger(logger)` / `goubus.WithSocketLogger(logger)` in constructor |
| `errdefs.ErrNotFound`, `errdefs.ErrPermissionDenied`, ... | `goubus.ErrNotFound`, `goubus.ErrPermissionDenied`, ... |
| `errdefs.IsNotFound(err)`, `errdefs.IsPermissionDenied(err)`, ... | `errors.Is(err, goubus.ErrNotFound)`, `errors.Is(err, goubus.ErrPermissionDenied)`, ... |
| `UbusStatus*`, `MapUbusCodeToError(code)` | `Status*`, `StatusError(code)` |
| Unstructured call error string | `errors.Is(err, goubus.ErrCall)` and `callErr, ok := errors.AsType[*goubus.CallError](err)` |
| `luci` `GetLocaltime(ctx)` / `GetUnixtime(ctx)` | `luci.Client.Time(ctx)` (probes automatically, or pin with `luci.WithTimeMethod`) |
| `luci` `GetVersion`, `GetProcessList`, `GetBoardJSON`, `GetDHCPLeases`, ... | `Version`, `Processes`, `BoardJSON`, `DHCPLeases` (`Get` prefix removed across `luci`) |
| `luci` `GetFeatures` -> `map[string]goubus.Bool` | `Features` -> `map[string]luci.Feature` (`.Enabled`, `.Subfeatures`) |
| `luci` `GetBlockDevices` -> `[]BlockDevice`, `GetUSBDevices` -> `[]USBDevice` | `BlockDevices` -> `map[string]luci.BlockDevice`, `USBDevices` -> `*luci.USBInfo` |
| Per-device `dhcp` DUID dialect | `dhcp.Client.AddLease` (detects via `goubus.List` or fallback, or pin with `dhcp.WithDUIDFormat`) |
| `container.GetFeatures`, `service.GetData` | `container.Client.Features`, `service.Client.Data` |
| `network.GetProtoHandlers`, `network.Wireless().GetValidate` | `network.Client.ProtoHandlers`, `network.Client.Wireless().Validate` |
| `hostapd.AP(name).GetClients` / `GetStatus` | `hostapd.Client.AP(name).Clients` / `Status` |
| `uci.Package(name).GetAll`, `section.GetFirst(opt)` | `uci.Client.Package(name).All`, `section.First(opt)` |
| `.Mac`, `.MacAddr`, `.Ifname`, `.Pid` fields | `.MAC`, `.MACAddr`, `.IfName`, `.PID` |

## v2.0.0-alpha2 (2026-08-01)

### Added
- `NewSocketClientFromConn` for creating a `SocketClient` from an existing `net.Conn`. Contributed by [@mook](https://github.com/mook) in [#6](https://github.com/HoneyBBQ/goubus/pull/6).

### Fixed
- `SocketClient` honors context cancellation during the `HELLO` handshake and closes the owned connection when initialization fails.
- Pinned `golangci-lint` to `v2.12.2` in CI and resolved existing lint findings.

### Changed
- Updated GitHub Actions for checkout, Go setup, coverage upload, and release workflows.

## v2.0.0-alpha1 (2026-01-18)

### Added
- Hardware profile packages under `profiles/` (`profiles/x86_generic`, `profiles/cmcc_rax3000m`) backed by shared dialect logic in `internal/base`.
- Request and response types for service calls (such as `dhcp.AddLeaseRequest`), `log/slog` transport logging, GoDoc comments across public packages, and the `errdefs` sentinel package.

### Changed
- **Breaking:** Module path changed to `github.com/honeybbq/goubus/v2`.
- **Breaking:** Unified `Transport` interface and moved `RPCClient` and `SocketClient` from `transport/` to the root `goubus` package.
- Switched license from Apache-2.0 to MIT.
- Updated minimum Go version to 1.25 and switched `examples/` to `log/slog`.

### Removed
- `transport/` subpackage.
