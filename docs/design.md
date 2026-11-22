# Design

These conventions apply to both transports and all service packages.

## Calls and ownership

`Caller` has one method, `Call`. Service clients take a `Caller` because they
invoke methods but do not own the connection. `Transport` adds `Close` for
code that owns the connection or RPC session. A `CallerFunc` and
`NewJSONResult` are enough to test a typed client without a router.

`CallMiddleware` is a function from `Caller` to `Caller`. Use
`ChainMiddleware` to wrap calls for tracing or metrics without changing either
transport. The first middleware in the list is the outermost wrapper. The
chain returns a `Caller`, not a `Transport`; keep the original transport for
cleanup. A wrapper that only implements `Call` does not forward `Lister`; use
the underlying transport for discovery or explicitly forward `List` when
wrapping a caller used for discovery.

`Result` exposes payload decoding, not wire status. `goubus.Call[T]` invokes
`Caller.Call` and decodes the result into `T`. Use `Caller.Call` directly for
status-only methods: a successful call is not required to carry a payload.

## Errors

Both transports check the ubus status before returning a `Result`. A nonzero
status produces a `*CallError` immediately, including for methods whose caller
never decodes a payload. The HTTP transport also returns `*CallError` for a
JSON-RPC error response.

`*CallError` carries the object (`Service`), method, numeric `Code`, and the
server's `Message` when present. Its `Unwrap() []error` exposes `ErrCall` and,
when present, the underlying cause. Use `errors.Is(err, goubus.ErrCall)` to
match the category, `errors.Is(err, goubus.ErrPermissionDenied)` (or another
root-package sentinel) to match a cause, and `errors.As` with a
`*goubus.CallError` target to inspect the fields. Connection and decoding
errors need not be `*CallError`. `ErrNoData` can also come from decoding a
successful call with no payload; that does not mean the operation failed.

## Discovery

`goubus.List` accepts a `Caller` and uses its optional `Lister` interface. A
caller without it returns `ErrNotSupported`. With no patterns or with `"*"`,
the transport requests all visible objects. The HTTP transport uses the
sessionless JSON-RPC `list` method, while the socket transport reads lookup
responses from `ubusd` and also uses them to populate its object ID cache.

Both return the same shape: object name, then method name, then argument name
and `ArgType`. RPC type names (including aliases such as `integer` or `table`)
and socket blobmsg type IDs map to `string`, `number`, `boolean`, `array`,
`object`, or `unknown`. A signature describes the visible method and its
arguments; it does not grant permission to invoke it.

## Firmware differences

Hardware model is not a reliable proxy for the installed firmware, packages,
or ubus schema. Maintaining a package per model duplicated service code and
made a model name part of the API. v2 keeps each service in one package and
uses the actual response shape or the router's runtime behavior to select a
representation:

| Difference | Handling |
| --- | --- |
| Booleans arrive as booleans, numbers, or strings. | `Bool` decodes each form and marshals as a JSON boolean. |
| A field arrives as a string or a list of strings. | `StringOrSlice` exposes a single slice. |
| An integer arrives as a number or a quoted integer. | `StringInt` exposes an integer. |
| A time arrives as Unix seconds or an RFC3339 string. | `UnixTime` wraps `time.Time`. |
| LuCI uses `getUnixtime` or `getLocaltime`. | `luci.Client.Time` tries `getUnixtime`, falls back on method-not-found, invalid-command, or permission-denied errors, and atomically caches a method after success. |
| DHCP `add_lease` expects `duid` as a string or an array. | With a DUID present, `dhcp.Client.AddLease` uses the signature when available; otherwise it tries an array and falls back to a string only on `ErrInvalidParameter`. |

The detected signature is cached immediately; a fallback format is cached only
when the add succeeds.

To pin known behavior, use `luci.WithTimeMethod("getLocaltime")` or
`dhcp.WithDUIDFormat(dhcp.DUIDString)`. The facade forwards these through
`services.WithLuCI` and `services.WithDHCP`. The `Option` interfaces are
sealed: callers select package-provided constructor options rather than
implementing arbitrary configuration callbacks.

## Logging and credentials

Both transports default to a discard `slog` handler. Opt in with
`WithRpcLogger` or `WithSocketLogger`. RPC client formatting (`String`,
`GoString`, and `LogValue`) redacts the password and session token. RPC call
request bodies and responses to `session` calls are also redacted. This is
not a general payload filter: other RPC responses and socket debug logs can
contain application data. Do not publish logs from sensitive calls without
reviewing them. Use HTTPS for remote credentials and keep them out of URLs and
custom middleware logs.

## Public API snapshots

`api/*.txt` records exported declarations. `go run ./internal/tools/apidump
-check` fails on snapshot drift or when exported signatures leak internal or
unexported types. CI and the release workflow run this check. To accept an
intentional API change, run `go run ./internal/tools/apidump` and review the
snapshot diff in the same change as the code. The doc coverage test separately
checks that exported declarations are documented.
