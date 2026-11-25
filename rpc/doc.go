// Package rpc implements an HTTP/HTTPS JSON-RPC 2.0 client for OpenWrt's
// uhttpd-mod-ubus gateway and rpcd session service.
//
// Native ubus speaks binary blobmsg frames over a local Unix domain socket
// (implemented by the root [github.com/honeybbq/goubus/v2] package). Remote
// management over HTTP or HTTPS goes through OpenWrt's web gateway plugin
// (uhttpd-mod-ubus or nginx-mod-ubus) paired with rpcd session authentication.
//
// [Client] implements [goubus.Transport] and [goubus.Lister], so callers built
// on [goubus.Caller], [goubus.Call], and [goubus.List] work unchanged across
// local Unix sockets and remote HTTP/HTTPS endpoints.
package rpc
