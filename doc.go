// Package goubus provides Go clients for OpenWrt ubus over HTTP JSON-RPC and
// Unix sockets. [Caller] is the narrow interface for making calls; [Transport]
// adds cleanup for clients that own resources.
//
// Create an [RpcClient] for a remote JSON-RPC endpoint or a [SocketClient] for
// the local ubus socket, then use [Call] to decode a response:
//
//	client, err := NewSocketClient(ctx, "")
//	if err != nil {
//		return err
//	}
//	defer client.Close()
//
//	info, err := Call[map[string]any](ctx, client, "system", "info", nil)
//	if err != nil {
//		return err
//	}
//	_ = info
//
// [Lister] implementations can also inspect object and method signatures
// across both transports.
package goubus
