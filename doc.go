// Package goubus implements OpenWrt's native ubus IPC protocol in pure Go.
//
// Native ubus speaks binary blobmsg frames over a local Unix domain socket
// (/tmp/run/ubus/ubus.sock) directly with ubusd. Open a multiplexed connection
// on the router with [NewClient] (or over a pre-established stream such as an
// SSH-forwarded Unix socket with [NewClientFromConn]):
//
//	conn, err := goubus.NewClient(ctx, "")
//	if err != nil {
//		return err
//	}
//	defer conn.Close()
//
//	type boardInfo struct {
//		Model    string `json:"model"`
//		Hostname string `json:"hostname"`
//	}
//	board, err := goubus.Call[boardInfo](ctx, conn, "system", "board", nil)
//
// Over the same multiplexed [*Client], programs can concurrently invoke
// methods ([Client.Call]), inspect registered signatures ([Client.List]),
// probe liveness ([Client.Ping]), broadcast and receive system events
// ([Client.SendEvent], [Client.Listen]), subscribe to object notifications
// ([Client.Subscribe]), and publish Go objects onto the bus
// ([Client.AddObject]).
//
// # Remote HTTP/HTTPS JSON-RPC gateway
//
// Native ubus uses Unix domain sockets only; OpenWrt exposes HTTP/HTTPS
// JSON-RPC through a separate web-server plugin (uhttpd-mod-ubus) paired with
// rpcd session authentication. Import [github.com/honeybbq/goubus/v2/rpc] when
// managing a router remotely over HTTP/HTTPS—its client implements [Transport]
// and [Lister], while on-router binaries importing only [goubus] avoid linking
// net/http or crypto/tls.
//
// # Errors and discovery
//
// Non-zero ubus statuses return [*CallError] before payload decoding. Match
// status causes with errors.Is against [ErrNotFound], [ErrPermissionDenied],
// and the other root sentinels (or [ErrCall] for any call refusal), and inspect
// Service, Method, Code, and Message with errors.AsType. [List] normalizes
// argument policies across both transports to [ArgType].
package goubus
