# Architecture

```text
goubus/
├── *.go                      # package goubus: Caller, Transport, Call[T], List,
│                             #   RpcClient, SocketClient, *CallError, sentinels,
│                             #   and flexible JSON types (Bool, StringOrSlice,
│                             #   StringInt, UnixTime)
├── services/
│   ├── client.go             # package services: shared Client holding all 15
│   │                         #   typed clients over one Caller
│   ├── container/            # ubus "container"
│   ├── dhcp/                 # ubus "dhcp" (leases, RA, adaptive add_lease)
│   ├── file/                 # ubus "file" (read, write, list, stat, md5, exec)
│   ├── hostapd/              # ubus "hostapd" and "hostapd.<iface>"
│   ├── log/                  # ubus "log"
│   ├── luci/                 # ubus "luci" and "luci-rpc" (adaptive time method)
│   ├── network/              # ubus "network", "network.device",
│   │                         #   "network.interface*", "network.wireless"
│   ├── rc/                   # ubus "rc"
│   ├── rpcsys/               # ubus "rpc-sys"
│   ├── service/              # ubus "service"
│   ├── session/              # ubus "session"
│   ├── system/               # ubus "system"
│   ├── uci/                  # ubus "uci" (PackageContext, SectionValues, apply)
│   ├── wireless/             # ubus "iwinfo"
│   └── wpasupplicant/        # ubus "wpa_supplicant" and "wpa_supplicant.<iface>"
├── api/                      # exported API of every public package, checked in CI
├── examples/                 # runnable programs for individual services and benchmarks
├── scripts/
│   └── release-notes.sh      # extracts tag release notes from CHANGELOG.md
└── internal/
    ├── blobmsg/              # ubus socket message header and binary blobmsg codec
    ├── rpc/                  # JSON-RPC 2.0 envelope and rpcd session wire structs
    ├── errs/                 # sentinel definitions and %w wrapping helpers
    ├── logging/              # default discard slog logger
    ├── testdata/             # sanitized OpenWrt 24.10.5 and 25.12.0-rc1 fixtures
    └── tools/apidump/        # generates and verifies api/*.txt
```

## Dependencies

Imports flow in one direction across the module:

```text
services -> services/<pkg> -> goubus -> internal/{blobmsg,rpc,errs,logging}
```

- `services/<pkg>` packages never import one another. Importing `services/system` compiles only `goubus`, `services/system`, and the four `internal/` runtime packages.
- `services` imports `goubus` and all 15 `services/<pkg>` packages to construct one shared collection when a program needs many objects at once.
- `goubus` has zero third-party runtime dependencies and uses no CGO, so binaries cross-compile cleanly to `mipsle`, `arm`, `arm64`, and `amd64` OpenWrt targets.
- `internal/testdata/` and `internal/tools/apidump/` are used only during testing and CI checks.

## Wire paths

Every typed method builds its request arguments and calls either `Caller.Call` (when only the ubus status matters) or `goubus.Call[T]` (when a JSON object is expected back). What happens underneath depends on the transport:

1. **HTTP/HTTPS JSON-RPC (`RpcClient`).** `NewRpcClient` logs in via `session.login` using the bootstrap session ID `00000000000000000000000000000000` and records `ubus_rpc_session` along with its `timeout`. Each `Call` checks whether the cached session is still valid, re-authenticates if it expired, and posts a JSON-RPC 2.0 `"call"` request (`[sessionID, service, method, args]`) to `/ubus` (or the path set with `WithRpcEndpoint`). `parseUbusResponse` inspects any JSON-RPC `error` object first, then reads the ubus `[statusCode, payload]` tuple in `result`. `Close` calls `session.destroy` and zeroes the stored credentials.
2. **Local Unix socket (`SocketClient`).** `NewSocketClient` (or `NewSocketClientFromConn`) connects to `/tmp/run/ubus/ubus.sock` and reads the server's `UBUS_MSG_HELLO` frame to learn its `peerID`. On `Call`, the client resolves the object name (`"system"`, `"network.interface.wan"`, ...) to a 32-bit `objid` via `UBUS_MSG_LOOKUP` and caches it in memory. It encodes the call arguments into binary `blobmsg` attributes inside `UBUS_MSG_INVOKE`, collects `UBUS_MSG_DATA` frames, and stops at `UBUS_MSG_STATUS`. Socket round trips are serialized on one connection under a mutex.

Both transports convert non-zero status codes into `*goubus.CallError` before returning a `Result`, and both implement `Lister` so `goubus.List` returns identical `ObjectSignatures` maps regardless of how the client connected. See [Design](design.md) for error matching, cancellation semantics, and OpenWrt version quirks.
