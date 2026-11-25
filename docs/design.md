# Design

## Package split: native bus vs web gateway

- **`github.com/honeybbq/goubus/v2`** implements the native `ubusd` Unix domain socket protocol (`Client`, concurrent `Call`, `List`, `Ping`, `Listen`/`SendEvent`, `Subscribe`, `AddObject`) together with the transport-agnostic contracts (`Caller`, `Transport`, `Result`, `Call[T]`, `Lister`, `*CallError`, and status sentinels).
- **`github.com/honeybbq/goubus/v2/rpc`** implements the stateless HTTP/HTTPS JSON-RPC 2.0 gateway protocol exposed by `uhttpd-mod-ubus` and `rpcd` (`rpc.Client`, implementing `goubus.Transport` and `goubus.Lister`).

Higher-level wrappers for specific OpenWrt daemons belong in consumer packages built on `goubus.Caller` and `goubus.Call[T]`.

## Calls and ownership

`Caller` has a single method:

```go
type Caller interface {
	Call(ctx context.Context, service, method string, data any) (Result, error)
}
```

`Transport` embeds `Caller` and adds `Close() error`. Functions that execute ubus calls should accept `Caller` rather than `Transport`: they borrow the connection and never close it. The code that called `goubus.NewClient` or `rpc.NewClient` owns `Close`.

Because `Caller` has one method, unit tests for downstream code do not need a router or a mock socket. Combine `CallerFunc` with `NewJSONResult`:

```go
caller := goubus.CallerFunc(func(ctx context.Context, service, method string, data any) (goubus.Result, error) {
	return goubus.NewJSONResult(json.RawMessage(`{"uptime": 3600}`)), nil
})

info, err := goubus.Call[struct {
	Uptime int64 `json:"uptime"`
}](ctx, caller, "system", "info", nil)
```

`Result` decodes payload bytes; it does not carry wire status. Status-only methods (`system.reboot`, `uci.commit`, `file.remove`) return status `0` with no data frame—call `Caller.Call` directly for those. `goubus.Call[T]` runs `Caller.Call` and then `Result.Unmarshal(&target)`, returning `ErrNoData` when a successful call carried an empty or `null` payload.

### Middleware

`CallMiddleware` wraps a `Caller` (`func(next Caller) Caller`). `ChainMiddleware(caller, m1, m2)` applies `m1` outermost:

```go
traced := goubus.ChainMiddleware(conn, auditMiddleware, metricsMiddleware)
board, err := goubus.Call[BoardInfo](ctx, traced, "system", "board", nil)
```

- `ChainMiddleware` returns a `Caller`, not a `Transport`. Keep the original `conn` reference for `defer conn.Close()`.
- `goubus.List` checks `caller.(Lister)`. Pass the underlying transport to `goubus.List` unless your custom wrapper also implements `Lister`.

## Multiplexing, events, pub/sub, and local objects (`goubus.Client`)

`*goubus.Client` runs a single reader goroutine over the Unix socket after the `HELLO` handshake:

- **Concurrent `Call` and `List`:** multiple goroutines share one `*goubus.Client` safely. Cancelling one call's `ctx` abandons only that pending sequence number without closing the shared socket connection. When a cached object restarts on `ubusd` under a new numeric ID, `Call` and `Subscribe` automatically evict the stale ID, re-lookup, and retry once. `Client.Done()` and `Client.Err()` expose connection shutdown to reconnect loops.
- **`Listen` and `SendEvent`:** `Listen(ctx, pattern, handler)` subscribes to the system event bus (`objid = 1`); passing `""` matches `"*"`. Each `*Subscription` buffers and delivers events sequentially on its own goroutine so slow handlers never stall socket reads. Call `sub.Close()` to unregister.
- **`Subscribe`:** `Subscribe(ctx, object, handler)` attaches to notifications emitted by a named object. `sub.Done()` closes when `sub.Close()` is called, the client closes, or the target object is removed from `ubusd` (`UBUS_MSG_UNSUBSCRIBE`).
- **`AddObject`:** `AddObject(ctx, ObjectDef{...})` publishes a named object and its `MethodSignature` policies on `ubusd`, with optional `OnSubscribe` state transitions and `RegisteredObject.Done()`. Each incoming invocation runs `MethodHandler` in its own goroutine with caller `User`/`Group` metadata, so handlers can invoke `client.Call` or `pub.Notify` re-entrantly without deadlocking. Returning a `*CallError` or root sentinel (`ErrInvalidParameter`, `ErrPermissionDenied`, ...) reports the corresponding wire status code to the caller.

Remote Go programs that have SSH access to a router can dial `/tmp/run/ubus/ubus.sock` over `golang.org/x/crypto/ssh` and pass the resulting `net.Conn` to `goubus.NewClientFromConn` for full bidirectional ubus support without `uhttpd-mod-ubus`.

## Errors

Errors fall into two groups:

1. **A refusal reported by `ubusd` or `uhttpd-mod-ubus`.** Both clients check the status before returning a `Result`. Any non-zero ubus status code (`1`..`10`), any JSON-RPC `error` object over HTTP, and any socket lookup status failure returns `*goubus.CallError` immediately.
2. **A transport, decoding, or caller condition.** Dial failures and non-2xx HTTP responses wrap `ErrConnectionFailed` (an invalid filesystem path in `goubus.NewClient` also wraps `ErrNotUnixSocket`). Per-request read timeouts wrap `ErrTimeout`. Using a closed client returns `ErrClosed`. Malformed wire frames or JSON type mismatches wrap `ErrInvalidResponse`. Decoding an empty payload returns `ErrNoData`. Context cancellation preserves `ctx.Err()` and `context.Cause(ctx)`.

`*CallError` unwraps to `[ErrCall, Err]`:

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

## Gateway session lifetime (`rpc.Client`)

`uhttpd-mod-ubus` reports `-32002 Access denied` both when the `rpcd` session expired/restarted and when ACLs deny the method. Because automatically re-logging in could conceal lost session state or uncommitted UCI staging, `rpc.Client.Call` returns `ErrPermissionDenied` without retrying. Active calls slide local session expiry forward by `rpcd`'s idle timeout, and `(*rpc.Client).Authenticate(ctx)` forces a fresh login when needed.

Stateless monitoring programs can opt into automatic single re-authentication on `-32002` directly via `rpc.WithAutoReauth(true)` (deduplicated across concurrent callers), or apply custom policy via `CallMiddleware`:

```go
func retryOnDenied(client *rpc.Client) goubus.CallMiddleware {
	return func(next goubus.Caller) goubus.Caller {
		return goubus.CallerFunc(func(ctx context.Context, service, method string, data any) (goubus.Result, error) {
			result, err := next.Call(ctx, service, method, data)
			if !errors.Is(err, goubus.ErrPermissionDenied) {
				return result, err
			}
			if err := client.Authenticate(ctx); err != nil {
				return nil, err
			}

			return next.Call(ctx, service, method, data)
		})
	}
}
```

## Object discovery (`List`)

`goubus.List(ctx, caller, patterns...)` normalizes `ubusd` numeric `blobmsg` type IDs and `uhttpd-mod-ubus` type strings into `ArgType` (`ArgTypeString`, `ArgTypeNumber`, `ArgTypeBoolean`, `ArgTypeArray`, `ArgTypeObject`, `ArgTypeUnknown`). Note that `ubus` only advertises input `blobmsg_policy` signatures on the bus, not return schemas.

## Logging, credentials, and API snapshots

Both `goubus.Client` and `rpc.Client` discard logs by default (`goubus.WithLogger`, `rpc.WithLogger`). `rpc.Client` redacts `password` and `ubus_rpc_session` in `String()`, `GoString()`, `LogValue()`, outgoing request debug logs, and `"session"` responses, and zeroes credentials on `Close()`.

Public APIs are locked in `api/goubus.txt` and `api/rpc.txt` via `go run ./internal/tools/apidump -check`, enforced alongside `internal/tools/apidump` GoDoc coverage checks and `golangci-lint` (`gofumpt`, `gci`, `decorder`, `funcorder`) in CI.
