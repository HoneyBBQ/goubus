<div align="center">
  <h1>goubus</h1>
  <p><strong>OpenWrt ubus Go Client</strong></p>

  <p>
    <a href="https://github.com/honeybbq/goubus/actions/workflows/ci.yml"><img src="https://github.com/honeybbq/goubus/actions/workflows/ci.yml/badge.svg" alt="CI Status"></a>
    <a href="https://codecov.io/gh/honeybbq/goubus"><img src="https://codecov.io/gh/honeybbq/goubus/branch/main/graph/badge.svg" alt="Coverage"></a>
    <a href="https://goreportcard.com/report/github.com/honeybbq/goubus"><img src="https://goreportcard.com/badge/github.com/honeybbq/goubus" alt="Go Report Card"></a>
    <a href="https://pkg.go.dev/github.com/honeybbq/goubus"><img src="https://pkg.go.dev/badge/github.com/honeybbq/goubus.svg" alt="Go Reference"></a>
  </p>

  <p>
    <a href="https://github.com/honeybbq/goubus/releases"><img src="https://img.shields.io/github/v/release/honeybbq/goubus?display_name=tag" alt="Release"></a>
    <a href="https://golang.org/"><img src="https://img.shields.io/badge/go-1.25-blue" alt="Go Version"></a>
    <a href="LICENSE"><img src="https://img.shields.io/github/license/honeybbq/goubus" alt="License"></a>
    <a href="https://github.com/honeybbq/goubus/graphs/contributors"><img src="https://img.shields.io/github/contributors/honeybbq/goubus" alt="Contributors"></a>
  </p>

  <p>
    <a href="https://github.com/honeybbq/goubus/issues"><img src="https://img.shields.io/github/issues/honeybbq/goubus" alt="Issues"></a>
    <a href="https://github.com/honeybbq/goubus/pulls"><img src="https://img.shields.io/github/issues-pr/honeybbq/goubus" alt="Pull Requests"></a>
    <a href="https://github.com/honeybbq/goubus/stargazers"><img src="https://img.shields.io/github/stars/honeybbq/goubus" alt="Stars"></a>
    <img src="https://img.shields.io/github/repo-size/honeybbq/goubus" alt="Repo Size">
  </p>

  <p>
    <code>goubus</code> provides an interface to interact with OpenWrt's <strong>ubus</strong> (micro bus) system, using a <strong>Profile</strong> system to support hardware-specific differences (like CMCC RAX3000M) while maintaining a consistent base implementation.
  </p>

  <p>
    <a href="README_CN.md">中文文档</a> | 
    <a href="https://github.com/honeybbq/goubus/tree/main/examples">Examples</a> | 
    <a href="https://pkg.go.dev/github.com/honeybbq/goubus/v2">Documentation</a>
  </p>
</div>

---

## Key Features

- **Device Profiles**: Native support for hardware-specific dialects (e.g., **CMCC RAX3000M**, **X86 Generic**).
- **Selective Imports**: Import only what you need, avoiding overhead.
- **Dual Transport**: Supports both **HTTP JSON-RPC** (remote) and **Unix Socket** (local).
- **Type-Safe API**: Fully typed requests and responses.
- **Context Aware**: Support for `context.Context` cancellation and timeouts.

## Installation

```bash
go get github.com/honeybbq/goubus/v2
```

## Quick Start

### 1. Initialize Transport

```go
import (
    "context"
    "github.com/honeybbq/goubus/v2"
)

ctx := context.Background()

// Remote via RPC (Requires rpcd and proper ACLs)
caller, _ := goubus.NewRpcClient(ctx, "192.168.1.1", "root", "password")

// Local via Unix Socket (Running on the device)
caller, _ := goubus.NewSocketClient(ctx, "/var/run/ubus/ubus.sock")
```

### 2. Use Device Specific Managers

Import the profile matching your hardware:

```go
import (
    "github.com/honeybbq/goubus/v2/profiles/cmcc_rax3000m/system"
    "github.com/honeybbq/goubus/v2/profiles/cmcc_rax3000m/network"
)

// Initialize Managers
sysSvc := system.New(caller)
netSvc := network.New(caller)

// Fetch System Board Info
board, _ := sysSvc.Board(ctx)
fmt.Printf("Model: %s, Kernel: %s\n", board.Model, board.Kernel)

// Dump all network interfaces
ifaces, _ := netSvc.Dump(ctx)
```

### 3. Debugging & Logging

`goubus` natively supports `log/slog`. You can inject your own logger to see raw ubus interactions (requests and responses):

```go
import "log/slog"

// Create a debug logger
logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))

// Inject it into the transport
caller.SetLogger(logger)
```

## Implemented Objects

| Object        | Description                                             |
| :------------ | :------------------------------------------------------ |
| **System**    | Board, Info, Reboot, Watchdog, Signal, Sysupgrade       |
| **Network**   | Interface control, Device status, Routing, Namespaces   |
| **Wireless**  | IWInfo (Scan, Assoclist), Wireless radio control        |
| **UCI**       | Full CRUD, Commit/Rollback, State tracking              |
| **DHCP**      | IPv4/v6 Leases, IPv6 RA, Static Lease management        |
| **Service**   | Service lifecycle, Validation, Custom data              |
| **Session**   | Login, Access control, Grant/Revoke                     |
| **Container** | LxC container management, Console access                |
| **Hostapd**   | Low-level AP management (Kick clients, Switch channels) |
| **RPC-SYS**   | Package management, Factory reset, Firmware validation  |

## Project Architecture

`goubus` is designed with a multi-layer architecture for code reuse:

- **Core Layer (`goubus/`)**: Contains the transport implementations (HTTP RPC and Unix Socket), raw ubus message handling (`blobmsg`), results parsing, and error definitions.
- **Base Layer (`goubus/internal/base/`)**: Provides generic, reusable implementations for standard ubus objects (e.g., system, network, uci). This layer encapsulates the common logic that applies to most OpenWrt devices.
- **Profile Layer (`goubus/profiles/`)**: The public API entry point. Profiles (e.g., `cmcc_rax3000m`, `generic`) use **Dialects** to handle hardware-specific quirks (like parameter types or special method names) while exposing a consistent, high-level interface.
- **Examples & TestData (`examples/`, `internal/testdata/`)**: Full integration tests using real hardware data and usage examples.

## Comparison

| Feature         | HTTP (JSON-RPC)   | Unix Socket    |
| --------------- | ----------------- | -------------- |
| **Use Case**    | Remote management | On-device apps |
| **Auth**        | Required          | Not required   |
| **Performance** | Network overhead  | Low latency    |

## Contributing & Device Support

This repository includes Profiles for RAX3000M and x86. The x86 Profile is derived from virtual machine data; given the diversity of x86 hardware and drivers, the implementation is undergoing further refinement.

Contributions of **Profiles** or **testdata** for other hardware are welcome.

### How to help out
1. **Copy**: Copy `profiles/x86_generic` to a new folder.
2. **Test**: Run tests using your real hardware data to find parsing errors.
3. **Fix**: Use a **Dialect** to handle the differences.

Guide on how to dump test data: [Contributing Test Data Guide](docs/CONTRIBUTING_DATA.md).

## License

MIT License - See [LICENSE](LICENSE) file for details.

## Acknowledgments

Inspired by [Kubernetes client-go](https://github.com/kubernetes/client-go) and [moby/moby](https://github.com/moby/moby).
