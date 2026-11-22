# goubus

OpenWrt ubus client in pure Go, over Unix sockets and HTTP JSON-RPC.

[![Go Reference](https://pkg.go.dev/badge/github.com/honeybbq/goubus/v2.svg)](https://pkg.go.dev/github.com/honeybbq/goubus/v2) [![CI](https://github.com/honeybbq/goubus/actions/workflows/ci.yml/badge.svg)](https://github.com/honeybbq/goubus/actions/workflows/ci.yml) [![Codecov](https://codecov.io/gh/honeybbq/goubus/branch/main/graph/badge.svg)](https://codecov.io/gh/honeybbq/goubus) [![Release](https://img.shields.io/github/v/release/honeybbq/goubus?include_prereleases&sort=semver)](https://github.com/honeybbq/goubus/releases) [![Go version](https://img.shields.io/github/go-mod/go-version/honeybbq/goubus)](go.mod) [![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

[Docs](https://pkg.go.dev/github.com/honeybbq/goubus/v2) · [Architecture](docs/architecture.md) · [Design](docs/design.md) · [Examples](examples/) · [Changelog](CHANGELOG.md) · [中文文档](README_CN.md)

## Why this library

- **Local socket and remote JSON-RPC.** Talk directly to `ubusd` on the router over `/tmp/run/ubus/ubus.sock` with a pure-Go binary `blobmsg` codec, or manage a router from outside over HTTP/HTTPS through `uhttpd-mod-ubus` and `rpcd`. No CGO and no third-party dependencies.
- **15 typed services, imported singly or together.** An on-router binary can import only `services/system` so unused packages stay out of the build. A controller or dashboard can call `services.New(conn)` to hold one shared instance of every service client.
- **Runtime handling of OpenWrt release differences.** OpenWrt 24.10's `luci` object exposes `getLocaltime`, while 25.12 exposes `getUnixtime`; `luci.Client.Time` probes and caches the working method, or `luci.WithTimeMethod` pins it. `dhcp.Client.AddLease` reads the router's `ubus list` signature to send `duid` as an array or string. `Bool`, `StringOrSlice`, `StringInt`, and `UnixTime` decode fields whose JSON types vary across daemons.
- **One-line generic calls for custom objects.** Third-party OpenWrt packages that have no typed package in `services/` (such as `mwan3` or `banip`) run on the same `Caller` through `goubus.Call[T]`.
- **Signature inspection and standard Go errors.** `goubus.List` reads object and method argument types in one normalized format over both transports. Non-zero ubus statuses and JSON-RPC refusals return `*goubus.CallError` before payload decoding, matching sentinels like `goubus.ErrNotFound` and `goubus.ErrPermissionDenied` via `errors.Is`.

## Install

```sh
go get github.com/honeybbq/goubus/v2
```

Go 1.25 or newer is required. Until `v2.0.0` is tagged, `go get` without a version skips pre-releases: pass an explicit pre-release tag from [Releases](https://github.com/honeybbq/goubus/releases) or a commit on this branch (note that `v2.0.0-alpha2` predates the `services/*` redesign). Upgrading from `v2.0.0-alpha1` or `v2.0.0-alpha2`? See the [migration table in CHANGELOG.md](CHANGELOG.md#migrating-from-v200-alpha1-and-v200-alpha2).

## Quick start

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/honeybbq/goubus/v2"
	"github.com/honeybbq/goubus/v2/services/system"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var (
		conn goubus.Transport
		err  error
	)
	if host := os.Getenv("UBUS_URL"); host != "" {
		conn, err = goubus.NewRpcClient(ctx, host, os.Getenv("UBUS_USER"), os.Getenv("UBUS_PASSWORD"))
	} else {
		conn, err = goubus.NewSocketClient(ctx, "")
	}
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = conn.Close() }()

	sys := system.New(conn)
	board, err := sys.Board(ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s (%s %s)\n", board.Model, board.Release.Version, board.Release.Revision)
}
```

## Usage

### Connecting: on-router socket vs remote RPC

When your Go binary runs on the router itself, connect to `ubusd` over the local Unix socket. Passing `""` uses `/tmp/run/ubus/ubus.sock`:

```go
conn, err := goubus.NewSocketClient(ctx, "")
if err != nil {
	return err
}
defer conn.Close()
```

When your program runs on another machine, connect to `uhttpd-mod-ubus` over HTTP or HTTPS. If the URL has no path, `NewRpcClient` uses `/ubus`, logs in through `session.login`, refreshes the `rpcd` session when its timeout expires, and calls `session.destroy` on `Close`:

```go
conn, err := goubus.NewRpcClient(ctx, "https://192.168.1.1", "root", os.Getenv("UBUS_PASSWORD"))
if err != nil {
	return err
}
defer conn.Close()
```

Both `*SocketClient` and `*RpcClient` implement `goubus.Transport` (`Caller` plus `Close`). Service clients accept `goubus.Caller` and never close the underlying transport.

### Single service package vs `services.New`

For a small agent or CLI deployed onto router flash, import only the service packages you call:

```go
import "github.com/honeybbq/goubus/v2/services/system"

info, err := system.New(conn).Info(ctx)
```

For a control plane, exporter, or web backend that talks to many ubus objects over one connection, construct the shared collection once so per-client caches (such as `luci` and `dhcp` runtime probes) stay warm:

```go
import "github.com/honeybbq/goubus/v2/services"

api := services.New(conn)
info, err := api.System().Info(ctx)
if err != nil {
	return err
}
wan, err := api.Network().Interface("wan").Status(ctx)
```

### Handling errors

Every non-zero ubus status (`1` to `10`) and every JSON-RPC error response returns a `*goubus.CallError` before any response JSON is decoded. Match causes with `errors.Is` and read wire details with `errors.As`:

```go
wan, err := api.Network().Interface("wan6").Status(ctx)
switch {
case errors.Is(err, goubus.ErrNotFound):
	// ubus object or interface does not exist
case errors.Is(err, goubus.ErrPermissionDenied):
	// rpcd ACL denied the session
case err != nil:
	var callErr *goubus.CallError
	if errors.As(err, &callErr) {
		log.Printf("%s.%s failed with code %d (%s)",
			callErr.Service, callErr.Method, callErr.Code, callErr.Message)
	}
	return err
}
_ = wan
```

### Calling third-party ubus objects

OpenWrt packages outside the 15 built-in services (for example `mwan3`, `banip`, or custom `rpcd` scripts) work directly with `goubus.Call[T]`:

```go
type Mwan3Interface struct {
	Status string      `json:"status"`
	Online int         `json:"online"`
	Up     goubus.Bool `json:"up"`
}

type Mwan3Status struct {
	Interfaces map[string]Mwan3Interface `json:"interfaces"`
}

mwan, err := goubus.Call[Mwan3Status](ctx, conn, "mwan3", "status", map[string]any{
	"section": "interfaces",
})
```

For methods that return only a ubus status code and no payload, call `_, err := conn.Call(ctx, "system", "reboot", nil)`.

### Inspecting objects with `List`

`goubus.List` queries visible objects and their argument types (`string`, `number`, `boolean`, `array`, `object`, `unknown`) on either transport:

```go
objects, err := goubus.List(ctx, conn, "dhcp")
if err != nil {
	return err
}
fmt.Println(objects["dhcp"]["add_lease"]) // map[duid:array ip:string ...]
```

`List` reports what `ubusd` registers on the bus; over HTTP JSON-RPC, calling a listed method still requires `rpcd` ACL access for the authenticated session.

## Packages

| Import | ubus objects | What it covers |
| --- | --- | --- |
| `github.com/honeybbq/goubus/v2` | any | `SocketClient`, `RpcClient`, `Caller`, `Call[T]`, `List`, `*CallError`, status sentinels, `Bool`, `StringOrSlice`, `StringInt`, `UnixTime` |
| `github.com/honeybbq/goubus/v2/services` | all below | Shared client collection (`services.New`) and constructor options |
| `github.com/honeybbq/goubus/v2/services/container` | `container` | Procd container instances, state, features, and console |
| `github.com/honeybbq/goubus/v2/services/dhcp` | `dhcp` | `odhcpd` / `dnsmasq` IPv4 and IPv6 leases, IPv6 RA prefixes, and `add_lease` |
| `github.com/honeybbq/goubus/v2/services/file` | `file` | File read/write, directory listings, `stat`/`lstat`, MD5, and command execution |
| `github.com/honeybbq/goubus/v2/services/hostapd` | `hostapd`, `hostapd.*` | AP status, associated stations, client deauth/ban, and channel switch |
| `github.com/honeybbq/goubus/v2/services/log` | `log` | Reading `logd` ring buffer entries and writing syslog events |
| `github.com/honeybbq/goubus/v2/services/luci` | `luci`, `luci-rpc` | LuCI version, time, processes, mounts, block/USB devices, LEDs, board JSON, network/wireless devices, host hints, and leases |
| `github.com/honeybbq/goubus/v2/services/network` | `network`, `network.device`, `network.interface*`, `network.wireless` | `netifd` interfaces, devices, host routes, protocol handlers, and radio control |
| `github.com/honeybbq/goubus/v2/services/rc` | `rc` | `/etc/init.d` script state and actions (`start`, `stop`, `restart`, `enable`, `disable`) |
| `github.com/honeybbq/goubus/v2/services/rpcsys` | `rpc-sys` | Installed package list, password changes, factory reset, and sysupgrade stages |
| `github.com/honeybbq/goubus/v2/services/service` | `service` | `procd` service definitions, instances, state, signals, and watchdogs |
| `github.com/honeybbq/goubus/v2/services/session` | `session` | `rpcd` session login, creation, key-value storage, and ACL grants |
| `github.com/honeybbq/goubus/v2/services/system` | `system` | Board metadata, uptime/memory/storage info, watchdog, signals, image validation, and sysupgrade |
| `github.com/honeybbq/goubus/v2/services/uci` | `uci` | Reading, staging, applying, confirming, and reverting `/etc/config/*` sections and options |
| `github.com/honeybbq/goubus/v2/services/wireless` | `iwinfo` | Wireless devices, radio info, SSID scans, station assoclists, frequencies, TX power, and channel survey |
| `github.com/honeybbq/goubus/v2/services/wpasupplicant` | `wpa_supplicant`, `wpa_supplicant.*` | Station interface status, reconfiguration, WPS, and control commands |

## Documentation

The full API reference is on [pkg.go.dev](https://pkg.go.dev/github.com/honeybbq/goubus/v2).

- [Architecture](docs/architecture.md): package tree, dependency rules, and how `RpcClient` and `SocketClient` execute calls on the wire.
- [Design](docs/design.md): ownership (`Caller` vs `Transport`), middleware, `*CallError` mapping, `List` normalization, OpenWrt 24.10/25.12 quirks, credential redaction, and `api/*.txt` snapshots.
- [Examples](examples/): standalone programs for `system`, `network`, `uci`, `file`, `service`, `session`, and transport benchmarks.
- [Changelog](CHANGELOG.md): release history and the migration guide from `v2.0.0-alpha1` / `v2.0.0-alpha2`.

## Compatibility

| Target | Tested against |
| --- | --- |
| OpenWrt 24.10 | Captured JSON and binary `blobmsg` fixtures from `x86/64` generic (`24.10.5`). |
| OpenWrt 25.12 | Captured JSON and binary `blobmsg` fixtures from CMCC RAX3000M (`25.12.0-rc1`). |
| Go | 1.25 and newer; pure Go, no CGO. |

Which ubus objects exist on a router depends on its installed OpenWrt packages (`rpcd-mod-file`, `rpcd-mod-iwinfo`, `luci-base`, `odhcpd`, and others). Over HTTP JSON-RPC, each call is also subject to the `rpcd` ACLs of the logged-in user.

## Contributing

See [CONTRIBUTING.md](.github/CONTRIBUTING.md) for local setup and CI checks, and [Contributing test data](docs/CONTRIBUTING_DATA.md) to submit `ubus call` / `ubus -v list` captures from another OpenWrt target or release.

## License

[MIT](LICENSE)
