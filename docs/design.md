# Design

The conventions below hold across the root `goubus` package and all 15 `services/*` packages.

## Calls and ownership

`Caller` has a single method:

```go
type Caller interface {
	Call(ctx context.Context, service, method string, data any) (Result, error)
}
```

`Transport` embeds `Caller` and adds `Close() error`. Service clients (`system.New`, `uci.New`, `services.New`, ...) take a `Caller` rather than a `Transport`: they borrow the connection and never close it. The code that called `NewSocketClient` or `NewRpcClient` owns `Close`.

Because `Caller` has one method, unit tests for code built on typed service clients do not need a router or a mock socket. Combine `CallerFunc` with `NewJSONResult`:

```go
caller := goubus.CallerFunc(func(ctx context.Context, service, method string, data any) (goubus.Result, error) {
	return goubus.NewJSONResult(json.RawMessage(`{"uptime": 3600}`)), nil
})
info, err := system.New(caller).Info(ctx)
```

`Result` decodes payload bytes; it does not carry wire status. Many ubus methods (`system.reboot`, `uci.commit`, `network.reload`) return status `0` with no data frame. Call `Caller.Call` directly for status-only methods. `goubus.Call[T]` runs `Caller.Call` and then `Result.Unmarshal(&target)`, which returns `ErrNoData` when a successful call carried an empty or `null` payload.

### Middleware

`CallMiddleware` wraps a `Caller` (`func(next Caller) Caller`). `ChainMiddleware(caller, m1, m2)` applies `m1` outermost, so `m1` sees each call first and its result last:

```go
traced := goubus.ChainMiddleware(conn, auditMiddleware, metricsMiddleware)
api := services.New(traced)
```

Two details to keep in mind when wrapping a transport:

- `ChainMiddleware` returns a `Caller`, not a `Transport`. Keep the original `conn` reference for `defer conn.Close()`.
- `goubus.List` checks `caller.(Lister)`. A middleware built with `CallerFunc` implements only `Call`; pass the underlying transport to `goubus.List` (or implement `Lister` on a custom wrapper struct if `dhcp.Client.AddLease` runs through the middleware and relies on signature discovery).

## Errors

Errors fall into two groups:

1. **A refusal reported by `ubusd` or `uhttpd-mod-ubus`.** Both transports check the ubus status before returning a `Result`. Any non-zero ubus status code (`1` through `10`), any JSON-RPC `error` object over HTTP, and any socket object lookup failure that maps to a ubus status returns a `*goubus.CallError` immediately, even when the caller never decodes a response body.
2. **A transport, decoding, or caller condition.** Dial failures and non-2xx HTTP responses wrap `ErrConnectionFailed` (an invalid socket path in `NewSocketClient` also wraps `ErrNotUnixSocket`). Using a closed client returns `ErrClosed`. Malformed wire frames or JSON type mismatches wrap `ErrInvalidResponse`. Decoding an empty payload returns `ErrNoData`. Calling `List` on a `Caller` without `Lister` returns `ErrNotSupported`. Context expiration or cancellation preserves `ctx.Err()`.

`*CallError` carries `Service`, `Method`, numeric `Code`, `Message` (when supplied by JSON-RPC), and the mapped sentinel `Err`. Its `Unwrap() []error` returns `[ErrCall, Err]`, so `errors.Is` matches both the general `ErrCall` class and the specific cause:

```go
err := uci.New(conn).Package("firewall").Commit(ctx)
switch {
case errors.Is(err, goubus.ErrPermissionDenied):
	// rpcd ACL refused uci.commit
case errors.Is(err, goubus.ErrNotFound):
	// uci object or config package is missing
case errors.Is(err, goubus.ErrCall):
	// any other ubus or JSON-RPC refusal
}

var callErr *goubus.CallError
if errors.As(err, &callErr) {
	log.Printf("service=%s method=%s code=%d msg=%q",
		callErr.Service, callErr.Method, callErr.Code, callErr.Message)
}
```

### Wire status mapping

| ubus code | JSON-RPC code | Sentinel | Meaning |
| --- | --- | --- | --- |
| `1` (`StatusInvalidCommand`) | `-32600` | `ErrInvalidCommand` | Malformed request or unsupported command |
| `2` (`StatusInvalidParameter`) | `-32602` | `ErrInvalidParameter` | Missing or wrong-typed method argument |
| `3` (`StatusMethodNotFound`) | `-32601` | `ErrMethodNotFound` | Object exists, method does not |
| `4` (`StatusNotFound`) | `-32000` | `ErrNotFound` | ubus object or target resource not found |
| `5` (`StatusNoData`) | n/a | `ErrNoData` | Operation returned `UBUS_STATUS_NO_DATA` (or `Unmarshal` saw no payload) |
| `6` (`StatusPermissionDenied`) | `-32002` | `ErrPermissionDenied` | Denied by `ubusd` or `rpcd` session ACL |
| `7` (`StatusTimeout`) | n/a | `ErrTimeout` | Daemon timed out handling the call |
| `8` (`StatusNotSupported`) | n/a | `ErrNotSupported` | Operation not supported |
| `9` (`StatusUnknown`) | other codes | `ErrUnknown` | Unclassified ubus or JSON-RPC error |
| `10` (`StatusConnectionFailed`) | n/a | `ErrConnectionFailed` | Daemon failed to reach upstream component |

### Context cancellation

Every network or socket method honors `ctx`. The two transports differ in what happens to the connection after cancellation:

- **`RpcClient`:** cancelling `ctx` aborts only the in-flight `http.Request`. The `RpcClient` and its cached `rpcd` session remain usable for later calls.
- **`SocketClient`:** ubus socket frames travel sequentially over one stream connection. If `ctx` is cancelled while the client is blocked in the `HELLO` handshake or waiting for a call/lookup reply, `SocketClient` closes the underlying `net.Conn` via `context.AfterFunc` so the blocked read returns immediately with `ctx.Err()`. Once cancelled mid-flight, that `SocketClient` is closed and a new one must be opened.

## Object discovery (`List`)

`goubus.List(ctx, caller, patterns...)` inspects registered objects and their method argument signatures:

- Passing no patterns or `"*"` lists every visible object. Passing an empty string `""` returns `ErrInvalidParameter`.
- `RpcClient.List` invokes the sessionless JSON-RPC `"list"` method.
- `SocketClient.List` sends `UBUS_MSG_LOOKUP` frames, ignores `ErrNotFound` on literal patterns that are not registered (matching JSON-RPC `"list"`), and records each returned `objpath -> objid` entry in its lookup cache.

Wire argument types differ between `uhttpd-mod-ubus` (strings such as `"integer"`, `"int32"`, `"table"`, `"int8"`) and `ubusd` (numeric `blobmsg` type IDs). Both normalize to `ArgType`:

| `ArgType` | Wire inputs mapped |
| --- | --- |
| `ArgTypeString` (`"string"`) | `"string"`, `blobmsg` `STRING` |
| `ArgTypeNumber` (`"number"`) | `"number"`, `"integer"`, `"int16"`, `"int32"`, `"int64"`, `"double"`, and the matching numeric `blobmsg` types |
| `ArgTypeBoolean` (`"boolean"`) | `"boolean"`, `"bool"`, `"int8"`, `blobmsg` `INT8` |
| `ArgTypeArray` (`"array"`) | `"array"`, `blobmsg` `ARRAY` |
| `ArgTypeObject` (`"object"`) | `"object"`, `"table"`, `blobmsg` `TABLE` |
| `ArgTypeUnknown` (`"unknown"`) | Unrecognized wire type names or IDs |

`List` reports bus visibility, not authorization. Over HTTP JSON-RPC, `"list"` runs without a session token and shows every registered object, while `Call` checks the logged-in user's `rpcd` ACLs.

## Firmware and wire differences

Earlier alphas split services by router model (`profiles/x86_generic`, `profiles/cmcc_rax3000m`). In practice, ubus schemas follow the OpenWrt release and installed package versions (`luci-base`, `odhcpd`, `netifd`, `rpcd`), not the board PCB. Maintaining per-device packages duplicated all 15 clients and forced callers to hardcode hardware names in imports.

`goubus` v2 keeps one package per ubus domain under `services/` and resolves differences from the wire response or runtime probing, verified against fixtures from OpenWrt 24.10.5 (`x86_generic`) and 25.12.0-rc1 (CMCC RAX3000M):

| OpenWrt behavior | How `goubus` handles it |
| --- | --- |
| Booleans encoded as `true`/`false`, `1`/`0`, `"1"`/`"0"`, `"yes"`/`"no"`, or `"on"`/`"off"` | `goubus.Bool` decodes all forms (`""` and `null` as `false`) and marshals as a JSON boolean. |
| Fields encoded as `"eth0"` in one call and `["eth0", "eth1"]` in another | `goubus.StringOrSlice` decodes both into `[]string` (`""` and `null` as `nil`); `.First()` returns the first element or `""`. |
| Integer fields such as `PID`/`PPID` in `luci.getProcessList` returned as strings (`"1"`) or numbers (`1`) | `goubus.StringInt` decodes both into `int64` (`""` and `null` as `0`). |
| Timestamps returned as Unix seconds (integer, float, or quoted number) or RFC3339 strings | `goubus.UnixTime` embeds `time.Time` (UTC) and marshals back as Unix seconds (`0` when zero). |
| `luci` exposes `getLocaltime` on OpenWrt 24.10 and `getUnixtime` on 25.12 | `luci.Client.Time` tries `getUnixtime` first; on `ErrMethodNotFound`, `ErrInvalidCommand`, or `ErrPermissionDenied` it falls back to `getLocaltime` and caches whichever succeeded via atomic CAS. |
| `luci.getFeatures` returns booleans on 24.10 (`"firewall": true`) and nested subfeature maps on 25.12 (`"dnsmasq": {"dhcpv6": true}`) | `luci.Feature` sets `Enabled = true` and populates `Subfeatures` when given an object, or decodes a scalar directly into `Enabled`. |
| `dhcp.add_lease` expects `duid` as `[]string` on some `odhcpd` builds and `string` on others | When `AddLeaseRequest.DUID` is set, `dhcp.Client.AddLease` checks `goubus.List(ctx, caller, "dhcp")` first; if unavailable, it sends `[]string{duid}` and retries `duid` as a string only on `ErrInvalidParameter`, caching the format once a call succeeds. |
| `dhcp` lease responses nest interfaces under `"device"` or `"interfaces"`, and `IPv6Lease` uses `"ip6addr"`, `"address"`, `"ipv6"`, or `"ipv6-addr"` | `dhcp` unwraps the outer container, stamps `Interface` onto each lease/RA struct, and normalizes all four IPv6 address keys into `IPAddr` and `Address`. `IPv6RA` accepts `"prefix"` as either an address string or a numeric prefix length. |
| `network.interface.<name>` `status` may be absent while `network.interface` `dump` is available | `network.InterfaceContext.Status` calls `network.interface.<name>.status` first and falls back to matching `<name>` in `DumpInterfaces`. |
| UCI `option` (scalar) vs `list` (array) values must round-trip without losing single-item list types | `uci.SectionValues` tracks scalar vs list kinds separately; use `SetList` to keep a one-element UCI `list` serialized as a JSON array. |

When you already know the target firmware's behavior and want to skip probing, pin it at construction time:

```go
luciClient := luci.New(conn, luci.WithTimeMethod(luci.TimeMethodLocaltime))
dhcpClient := dhcp.New(conn, dhcp.WithDUIDFormat(dhcp.DUIDString))

// Or through the shared services collection:
api := services.New(conn,
	services.WithLuCI(luci.WithTimeMethod(luci.TimeMethodLocaltime)),
	services.WithDHCP(dhcp.WithDUIDFormat(dhcp.DUIDString)),
)
```

All `Option` interfaces (`RpcOption`, `SocketOption`, `services.Option`, `luci.Option`, `dhcp.Option`) are sealed with unexported methods so new internal fields can be added without breaking callers.

## Logging and credentials

Both transports discard logs by default. Pass `WithRpcLogger(logger)` or `WithSocketLogger(logger)` to emit `log/slog` records.

Credential redaction covers the RPC session lifecycle:

- `RpcClient.String()`, `GoString()`, and `LogValue()` format `password` and `ubus_rpc_session` as `"<redacted>"`.
- `RpcClient` debug logs always redact outgoing JSON-RPC request bodies (which carry the session ID and, during login, the password) and redact responses to any call on the `"session"` service.
- `RpcClient.Close()` clears `password` and `sessionData` in memory and sends `session.destroy`.

Debug logging is not a general-purpose secret scrubber: non-`"session"` RPC response previews (capped at 512 bytes) and `SocketClient` debug logs (truncated JSON and hex frame previews) include ordinary ubus payloads, which can contain Wi-Fi keys, PPPoE passwords, or UCI secrets when querying those objects. Keep debug logs out of shared outputs unless reviewed, use HTTPS for remote `RpcClient` connections, and do not embed credentials in URLs.

## Stability and API snapshots

Every public package's exported surface is recorded in `api/*.txt`. Running `go run ./internal/tools/apidump -check` fails if any exported constant, function, type, struct field, or method drifts from the snapshot, or if an exported signature leaks an `internal/` or unexported type. Both CI and the release workflow enforce this check alongside `doc_coverage_test.go`, which requires a GoDoc comment on every exported declaration, field, and method.

When changing the public API intentionally, refresh the snapshot and commit the diff in `api/*.txt` together with the code:

```sh
go run ./internal/tools/apidump
```
