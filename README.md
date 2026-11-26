<h1 align="center">goubus</h1>

<p align="center">
  <strong>Pure-Go OpenWrt <code>ubus</code> protocol library — full-duplex native Unix domain sockets, plus an optional HTTP/HTTPS JSON-RPC gateway subpackage.</strong><br>
  Zero CGO · Zero third-party dependencies · MIPS / ARM / x86
</p>

<p align="center">
  <a href="https://pkg.go.dev/github.com/honeybbq/goubus/v2"><img alt="Go Reference" src="https://pkg.go.dev/badge/github.com/honeybbq/goubus/v2.svg"></a>
  <a href="https://github.com/honeybbq/goubus/actions/workflows/ci.yml"><img alt="CI" src="https://github.com/honeybbq/goubus/actions/workflows/ci.yml/badge.svg"></a>
  <a href="https://codecov.io/gh/honeybbq/goubus"><img alt="Codecov" src="https://codecov.io/gh/honeybbq/goubus/branch/main/graph/badge.svg"></a>
  <a href="https://scorecard.dev/viewer/?uri=github.com/HoneyBBQ/goubus"><img alt="OpenSSF Scorecard" src="https://api.scorecard.dev/projects/github.com/HoneyBBQ/goubus/badge"></a>
  <a href="https://github.com/honeybbq/goubus/releases"><img alt="Release" src="https://img.shields.io/github/v/release/honeybbq/goubus?include_prereleases&amp;sort=semver"></a>
  <a href="go.mod"><img alt="Go version" src="https://img.shields.io/github/go-mod/go-version/honeybbq/goubus"></a>
  <a href="LICENSE"><img alt="License: MIT" src="https://img.shields.io/badge/License-MIT-blue.svg"></a>
</p>

<p align="center">
  <a href="https://pkg.go.dev/github.com/honeybbq/goubus/v2">Docs</a> ·
  <a href="docs/architecture.md">Architecture</a> ·
  <a href="docs/design.md">Design</a> ·
  <a href="CHANGELOG.md">Changelog</a> ·
  <a href="README_CN.md">中文文档</a>
</p>

## Overview

OpenWrt's `ubusd` speaks binary `blobmsg` frames over a local Unix domain socket (`package goubus`), while remote HTTP/HTTPS JSON-RPC access is served by OpenWrt's `uhttpd-mod-ubus` + `rpcd` web gateway (`package rpc`):

- **Full-duplex multiplexed native `ubus` (`goubus`).** Connect directly to `/tmp/run/ubus/ubus.sock` with `goubus.NewClient` (or any `net.Conn`, such as an SSH-forwarded socket, via `NewClientFromConn`). Concurrent `Call` and `List` requests multiplex by sequence ID over one connection without head-of-line blocking, and on-router binaries never link `net/http` or `crypto/tls`.
- **Events, subscriptions, and Go `ubus` objects (`goubus`).** Broadcast and listen to system events (`SendEvent`, `Listen`), subscribe to object notifications (`Subscribe`), and publish Go methods onto `ubusd` (`AddObject`, `RegisteredObject.Notify`) in pure Go without `libubus` or CGO.
- **Remote HTTP/HTTPS JSON-RPC gateway (`goubus/v2/rpc`).** Manage routers from external hosts via `rpc.NewClient`, with automatic `rpcd` session login, sliding idle expiry, explicit `Authenticate`, and credential redaction—implementing the exact same `goubus.Transport` and `goubus.Lister` interfaces.
- **Wire-first errors and signature introspection.** Non-zero ubus statuses (`1`..`10`) and JSON-RPC errors return `*goubus.CallError` before payload decoding, and `goubus.List` normalizes argument signatures into `ArgType`.

## Packages

| Import path | Role |
| --- | --- |
| `github.com/honeybbq/goubus/v2` | Native UDS `Client` (`Call`, `List`, `Ping`, `Listen`, `SendEvent`, `Subscribe`, `AddObject`), `Caller` / `Transport` / `Call[T]`, and `*CallError` & status sentinels |
| `github.com/honeybbq/goubus/v2/rpc` | Remote HTTP/HTTPS JSON-RPC 2.0 `Client` for `uhttpd-mod-ubus` and `rpcd` (`Transport` + `Lister`) |

## Install

```sh
go get github.com/honeybbq/goubus/v2@latest
```

Requires Go 1.27 or newer.

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
	"github.com/honeybbq/goubus/v2/rpc"
)

type boardInfo struct {
	Model   string `json:"model"`
	Release struct {
		Version  string `json:"version"`
		Revision string `json:"revision"`
	} `json:"release"`
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var (
		conn goubus.Transport
		err  error
	)
	if host := os.Getenv("UBUS_URL"); host != "" {
		conn, err = rpc.NewClient(ctx, host, os.Getenv("UBUS_USER"), os.Getenv("UBUS_PASSWORD"))
	} else {
		conn, err = goubus.NewClient(ctx, "")
	}
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = conn.Close() }()

	board, err := goubus.Call[boardInfo](ctx, conn, "system", "board", nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s (%s %s)\n", board.Model, board.Release.Version, board.Release.Revision)
}
```

## Usage

### Calling methods and inspecting signatures

```go
type wanStatus struct {
	Proto  string   `json:"proto"`
	DNS    []string `json:"dns-server"`
	Up     bool     `json:"up"`
	Uptime int64    `json:"uptime"`
}

wan, err := goubus.Call[wanStatus](ctx, conn, "network.interface.wan", "status", nil)

// Status-only call with no response body:
_, err = conn.Call(ctx, "uci", "commit", map[string]any{"config": "network"})

// Introspect registered objects and argument types (ubus -v list):
sigs, err := goubus.List(ctx, conn, "uci")
```

### Events, subscriptions, and publishing Go objects (native UDS)

```go
sock, err := goubus.NewClient(ctx, "")
if err != nil {
	return err
}
defer sock.Close()

// Listen to broadcast events (ubus listen) and emit one (ubus send):
evSub, err := sock.Listen(ctx, "network.interface", func(ev goubus.Event) {
	log.Printf("event %s: %s", ev.Type, ev.Data)
})
if err != nil {
	return err
}
defer evSub.Close()

_ = sock.SendEvent(ctx, "custom.ready", map[string]any{"ok": true})

// Publish a pure-Go ubus object onto ubusd:
obj, err := sock.AddObject(ctx, goubus.ObjectDef{
	Name: "myapp.greeter",
	Methods: map[string]goubus.Method{
		"hello": {
			Policy: goubus.MethodSignature{"name": goubus.ArgTypeString},
			Handler: func(ctx context.Context, req goubus.CallRequest) (any, error) {
				var in struct {
					Name string `json:"name"`
				}
				if err := req.Unmarshal(&in); err != nil {
					return nil, goubus.ErrInvalidParameter
				}
				return map[string]any{"greeting": "Hello, " + in.Name}, nil
			},
		},
	},
})
if err != nil {
	return err
}
defer obj.Close()
```

### Error handling

```go
_, err := conn.Call(ctx, "network.interface.wan6", "status", nil)
switch {
case errors.Is(err, goubus.ErrNotFound):
	// ubus object or target resource does not exist
case errors.Is(err, goubus.ErrPermissionDenied):
	// denied by ubusd or rpcd ACL
case err != nil:
	if callErr, ok := errors.AsType[*goubus.CallError](err); ok {
		log.Printf("%s.%s failed: code=%d msg=%q",
			callErr.Service, callErr.Method, callErr.Code, callErr.Message)
	}
	return err
}
```

## Documentation

Full API reference: [pkg.go.dev/github.com/honeybbq/goubus/v2](https://pkg.go.dev/github.com/honeybbq/goubus/v2).

- [Architecture](docs/architecture.md): native UDS vs `uhttpd-mod-ubus` gateway split, package layout, and wire paths.
- [Design](docs/design.md): ownership, multiplexing, events/objects, `*CallError` mapping, gateway sessions, and API snapshots.
- [Changelog](CHANGELOG.md): release notes and version history.

## Compatibility

| Target | Verified against |
| --- | --- |
| OpenWrt 21.02 – 25.12 | Live `21.02`, `22.03`, `23.05`, `24.10`, and `25.12` containers in parallel CI (multiplexed Unix socket, `Listen`/`SendEvent`, `AddObject`/`Subscribe`, HTTP RPC, and HTTPS RPC) plus binary `blobmsg` corpus. |
| Go | 1.27 and newer (CI tests Go 1.27 and cross-builds `linux/mipsle`, `linux/arm`, `linux/386`, and `linux/arm64`); pure Go, no CGO. |

## Contributing

See [CONTRIBUTING.md](.github/CONTRIBUTING.md) for local setup, formatting (`golangci-lint fmt`), API snapshot checks, and running the live OpenWrt container suite.

## License

[MIT](LICENSE)
