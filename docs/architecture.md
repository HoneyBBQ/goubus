# Architecture

```text
goubus/
|-- *.go                 # goubus: call and list contracts, transports,
|                         #   status errors, and shared JSON types
|-- services/
|   |-- client.go         # facade: one instance of each service client
|   `-- <pkg>/            # 15 independent clients for ubus service domains
|-- api/                  # snapshots of every public package
|-- examples/             # runnable programs using the public API
`-- internal/
    |-- blobmsg/          # ubus framing and binary attribute codec
    |-- rpc/              # JSON-RPC and ubus wire response types
    |-- errs/             # shared error identities and wrapping helpers
    |-- logging/          # shared slog helpers
    |-- testdata/         # captured device and firmware responses for tests
    `-- tools/apidump/    # writes and checks api/
```

## Dependencies

Application code can import the root package, a single service package, or the
`services` facade. Dependencies within the module point in one direction:

```text
services -> services/<pkg> -> goubus -> internal/{blobmsg,rpc,errs,logging}
```

The facade also imports `goubus` directly for `Caller`. A service package owns
its methods, request and response types, and any firmware adaptation for its
objects. Service packages do not import one another. The root package does not
import services, and the internal protocol packages do not import the public
packages. Tests read `internal/testdata/`; neither those fixtures nor the API
snapshot tool are runtime dependencies.

## Call path

1. A typed service method selects an ubus object and method and builds its
   parameters. It calls `Caller.Call` for an operation without a response
   payload, or `goubus.Call[T]` to decode one.
2. `RpcClient` authenticates with `rpcd` and sends calls to the
   `uhttpd-mod-ubus` JSON-RPC endpoint over HTTP or HTTPS. `SocketClient`
   performs the ubus HELLO handshake, resolves object IDs, and invokes methods
   over a Unix socket using the Go `blobmsg` codec.
3. The transport checks the call status before returning a `Result`. The
   service decodes the payload into its own types. Applications can also call
   the root API directly for objects that have no typed client.

Both transports implement `Lister` for discovery. The service facade does not
own or close the transport; the application that constructs it does. See
[Design](design.md) for error handling, signature normalization, and firmware
adaptation.
