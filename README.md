# goubus

A Go client for OpenWrt ubus over HTTP JSON-RPC and Unix sockets.

[![Go Reference](https://pkg.go.dev/badge/github.com/honeybbq/goubus/v2.svg)](https://pkg.go.dev/github.com/honeybbq/goubus/v2) [![CI](https://github.com/honeybbq/goubus/actions/workflows/ci.yml/badge.svg)](https://github.com/honeybbq/goubus/actions/workflows/ci.yml) [![Codecov](https://codecov.io/gh/honeybbq/goubus/branch/main/graph/badge.svg)](https://codecov.io/gh/honeybbq/goubus) [![Release](https://img.shields.io/github/v/release/honeybbq/goubus?include_prereleases&sort=semver)](https://github.com/honeybbq/goubus/releases) [![Go version](https://img.shields.io/github/go-mod/go-version/honeybbq/goubus)](go.mod) [![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

[Docs](https://pkg.go.dev/github.com/honeybbq/goubus/v2) · [Architecture](docs/architecture.md) · [Design](docs/design.md) · [Examples](examples/) · [Changelog](CHANGELOG.md) · [中文文档](README_CN.md)

## Why this library

- **One `Caller`, two transports.** Use HTTP or HTTPS JSON-RPC through `uhttpd-mod-ubus` remotely, or talk directly to `ubusd` over a Unix socket. The binary `blobmsg` codec is pure Go, with no CGO or third-party dependencies.
- **Typed services.** Import any of the 15 `services/<pkg>` packages on its own, or use `services.Client` to share one caller across all of them.
- **Runtime introspection.** `goubus.List` exposes `ubus list` object and method signatures in the same format on both transports.
- **Firmware differences.** Clients adapt to method and argument differences at runtime. `Bool`, `StringOrSlice`, `StringInt`, and `UnixTime` decode fields whose JSON representation varies.
- **Middleware and logging.** `CallMiddleware` wraps calls independently of the transport. `log/slog` logging is off by default; RPC client formatting and RPC credential-bearing traffic are redacted as described in [Design](docs/design.md#logging-and-credentials).

## Install

```sh
go get github.com/honeybbq/goubus/v2
```

Until `v2.0.0` is released, select a prerelease explicitly, for example `go get github.com/honeybbq/goubus/v2@v2.0.0-alpha2`, or use a newer prerelease tag from [Releases](https://github.com/honeybbq/goubus/releases). The examples below use the redesigned API: `alpha2` predates it. Use a checkout of this source or a tag that includes the redesign when building the examples.

## Quick start

Run this program on the router to use the default socket (`/tmp/run/ubus/ubus.sock`). For remote use, set `UBUS_URL` to the router's HTTPS URL and set `UBUS_USERNAME` and `UBUS_PASSWORD` for an `rpcd` account with access to the methods you call. The RPC path defaults to `/ubus` when the URL has no path.

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/honeybbq/goubus/v2"
	"github.com/honeybbq/goubus/v2/services"
	"github.com/honeybbq/goubus/v2/services/system"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var conn goubus.Transport
	var err error
	if url := os.Getenv("UBUS_URL"); url != "" {
		conn, err = goubus.NewRpcClient(ctx, url, os.Getenv("UBUS_USERNAME"), os.Getenv("UBUS_PASSWORD"))
	} else {
		conn, err = goubus.NewSocketClient(ctx, "")
	}
	if err != nil {
		return err
	}
	defer func() { _ = conn.Close() }()

	board, err := system.New(conn).Board(ctx)
	if err != nil {
		return err
	}
	fmt.Println(board.Model)

	info, err := services.New(conn).System().Info(ctx)
	if err != nil {
		return err
	}
	fmt.Printf("Uptime: %d s\n", info.Uptime)

	objects, err := goubus.List(ctx, conn)
	if err != nil {
		return err
	}
	fmt.Printf("Objects: %d\n", len(objects))

	if _, ok := objects["example.stats"]["read"]; ok {
		type stats struct {
			Requests int `json:"requests"`
		}
		result, err := goubus.Call[stats](ctx, conn, "example.stats", "read", nil)
		if err != nil {
			return err
		}
		fmt.Printf("Requests: %d\n", result.Requests)
	}
	return nil
}
```

Use `system.New(conn)` if you only need the system client, or `services.New(conn)` to keep one instance of each client. `List` reports the objects visible to the transport, not the permissions for invoking each method. The last call shows how to decode a custom object, `example.stats`, if the router provides it.

## Packages

| Import | Purpose |
| --- | --- |
| `github.com/honeybbq/goubus/v2` | Transports, call and list interfaces, status errors, and flexible JSON types. |
| `github.com/honeybbq/goubus/v2/services` | One client collection for all typed services. |
| `github.com/honeybbq/goubus/v2/services/container` | Container configuration, state, and console operations. |
| `github.com/honeybbq/goubus/v2/services/dhcp` | DHCP leases, static lease creation, and IPv6 router advertisements. |
| `github.com/honeybbq/goubus/v2/services/file` | File access, metadata, and command execution through the `file` object. |
| `github.com/honeybbq/goubus/v2/services/hostapd` | Access point status, connected clients, and channel control. |
| `github.com/honeybbq/goubus/v2/services/log` | Read and write system log entries. |
| `github.com/honeybbq/goubus/v2/services/luci` | LuCI system information and `luci-rpc` board, network, and lease data. |
| `github.com/honeybbq/goubus/v2/services/network` | Network interfaces, devices, routes, and radio configuration and status. |
| `github.com/honeybbq/goubus/v2/services/rc` | List init scripts and run init actions. |
| `github.com/honeybbq/goubus/v2/services/rpcsys` | Installed package information and system maintenance through `rpc-sys`. |
| `github.com/honeybbq/goubus/v2/services/service` | Service instances, configuration, and lifecycle. |
| `github.com/honeybbq/goubus/v2/services/session` | rpcd sessions, session data, and access controls. |
| `github.com/honeybbq/goubus/v2/services/system` | Board information, runtime status, and system operations. |
| `github.com/honeybbq/goubus/v2/services/uci` | Read and update UCI packages, sections, and options. |
| `github.com/honeybbq/goubus/v2/services/wireless` | `iwinfo` device status, scans, and station information. |
| `github.com/honeybbq/goubus/v2/services/wpasupplicant` | `wpa_supplicant` station status and control. |

## Documentation

- [API reference](https://pkg.go.dev/github.com/honeybbq/goubus/v2): public types, options, and methods.
- [Architecture](docs/architecture.md): package layout and dependencies.
- [Design](docs/design.md): calls, errors, discovery, firmware differences, and API stability.
- [Examples](examples/): complete programs for individual services.
- [Changelog](CHANGELOG.md): release history and migration from the hardware profiles.

## Compatibility

| Target | Coverage |
| --- | --- |
| OpenWrt 24.10 | Real-world response fixtures from x86 generic (24.10.5). |
| OpenWrt 25.12 | Real-world response fixtures from CMCC RAX3000M (25.12.0-rc1). |
| Go | 1.25 and newer; pure Go, no CGO. |

Available objects depend on installed OpenWrt packages and configuration. Remote invocation also depends on rpcd ACLs.

## Contributing

See [Contributing test data](docs/CONTRIBUTING_DATA.md) to capture responses and signatures and add a regression test.

## License

[MIT](LICENSE)
