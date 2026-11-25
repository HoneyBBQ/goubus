# Architecture

```text
goubus/
├── *.go                      # package goubus: native UDS ubus protocol (Client,
│                             #   Ping, Listen/SendEvent, Subscribe, AddObject) and
│                             #   shared contracts (Caller, Transport, Call[T],
│                             #   List, *CallError, status sentinels)
├── rpc/                      # package rpc: HTTP/HTTPS JSON-RPC 2.0 client for
│                             #   OpenWrt's uhttpd-mod-ubus + rpcd gateway
├── api/                      # exported API snapshots (goubus.txt, rpc.txt)
├── ci/images/                # pinned OpenWrt container rootfs digests
├── scripts/
│   ├── ci-image.sh           # builds OpenWrt lab container images
│   ├── openwrt-lab.sh        # starts and stops live OpenWrt test containers
│   └── release-notes.sh      # extracts tag release notes from CHANGELOG.md
└── internal/
    ├── blobmsg/              # ubus socket message header and binary blobmsg codec
    ├── errs/                 # sentinel definitions and %w wrapping helpers
    └── tools/
        ├── apidump/          # generates and verifies api/*.txt
        └── testlab/          # TLS CA, Unix socket bridge, and readiness probe
```

## Protocol boundary: native `ubus` vs HTTP JSON-RPC gateway

OpenWrt's `ubusd` daemon speaks **only** binary `blobmsg` frames over a local Unix domain socket (`package goubus`). It has no built-in HTTP server, JSON-RPC parser, or password/token session layer.

Remote HTTP/HTTPS access on OpenWrt is provided by an optional gateway stack—`uhttpd-mod-ubus` (or `nginx-mod-ubus`) translating stateless JSON-RPC `"call"` and `"list"` requests into local UDS operations, backed by `rpcd`'s `"session"` object for authentication and ACLs. Isolating that gateway client in `package rpc` (`github.com/honeybbq/goubus/v2/rpc`) keeps two properties intact:

```text
goubus/v2/rpc -> goubus/v2 -> internal/{blobmsg,errs}
```

1. **Zero HTTP/TLS footprint on routers.** Go daemons and CLI tools running directly on OpenWrt import only `github.com/honeybbq/goubus/v2`, keeping `net/http` and `crypto/tls` out of flash-constrained `mipsle`/`arm` binaries.
2. **100% shared contracts.** `*goubus.Client` and `*rpc.Client` both implement `goubus.Transport` (`Caller` + `Close`) and `goubus.Lister`, returning the same `goubus.Result`, `*goubus.CallError`, `goubus.ObjectSignatures`, and sentinel errors.

## Wire paths

### 1. Native Unix domain socket (`goubus.Client`)

`NewClient` (or `NewClientFromConn` over any `net.Conn`, including SSH-forwarded Unix sockets) connects to `/tmp/run/ubus/ubus.sock` (falling back to `/var/run/ubus/ubus.sock` when present), completes the `UBUS_MSG_HELLO` handshake to learn its `peerID`, and starts a background demultiplexing read loop:

- **Concurrent requests (`Call`, `List`, `SendEvent`, `AddObject`, `Subscribe`).** Each outbound request allocates an in-flight `uint16` sequence number (`hdr.Seq`) and writes its binary `blobmsg` frame under a write lock. The read loop routes incoming `UBUS_MSG_DATA` and `UBUS_MSG_STATUS` frames back to the matching pending caller, so slow operations (such as wireless scans) never block concurrent calls on the same connection.
- **System events (`Listen`, `SendEvent`).** `Listen` registers an anonymous local object (`UBUS_MSG_ADD_OBJECT`) and invokes `UBUS_SYSTEM_OBJECT_EVENT` (`objid = 1`, method `"register"`). Broadcasts arrive as `UBUS_MSG_INVOKE` frames and are delivered in order through a bounded per-subscription queue.
- **Object pub/sub (`Subscribe`, `RegisteredObject.Notify`).** `Subscribe` binds an anonymous listener object via `UBUS_MSG_SUBSCRIBE` (`UBUS_ATTR_OBJID` + `UBUS_ATTR_TARGET`). If the target object unregisters, `ubusd` sends `UBUS_MSG_UNSUBSCRIBE` and `Subscription.Done()` closes automatically.
- **Object registration (`AddObject`).** Publishes a named object and its `blobmsg_policy` method signatures. Incoming `UBUS_MSG_INVOKE` calls run in goroutines—allowing handlers to make re-entrant `Client.Call` or `RegisteredObject.Notify` requests without deadlocking—and reply with `UBUS_MSG_DATA` + `UBUS_MSG_STATUS`.

### 2. Remote HTTP/HTTPS JSON-RPC gateway (`rpc.Client`)

`rpc.NewClient` logs in via `session.login` using the bootstrap session ID `00000000000000000000000000000000` and caches `ubus_rpc_session` and its idle `timeout`. Each `Call` checks local expiry, re-authenticates under a lock when idle-expired, hoists any explicit `ubus_rpc_session` field into JSON-RPC `params[0]`, and posts to `/ubus` (or `rpc.WithEndpoint`). Successful calls slide local expiry forward; `Close` calls `session.destroy` and zeroes stored credentials.

Both clients enforce wire size limits (`1 MiB` per UDS frame, `4 MiB` per HTTP response body) and convert non-zero status codes into `*goubus.CallError` before returning a `Result`.
