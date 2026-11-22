// Package goubus is an OpenWrt ubus client over local Unix sockets and HTTP
// JSON-RPC.
//
// Open a connection with [NewSocketClient] on the router (talking to ubusd
// over /tmp/run/ubus/ubus.sock in pure Go) or [NewRPCClient] from another host
// (talking to uhttpd-mod-ubus and rpcd over HTTP or HTTPS). Both implement
// [Transport], which combines [Caller] with Close.
//
// Typed clients for 15 standard OpenWrt ubus domains live under [services/*]
// (such as [github.com/honeybbq/goubus/v2/services/system],
// [github.com/honeybbq/goubus/v2/services/network], and
// [github.com/honeybbq/goubus/v2/services/uci]); import only the packages a
// binary uses. For ubus objects without a typed package, decode any call
// directly with [Call]:
//
//	conn, err := goubus.NewSocketClient(ctx, "")
//	if err != nil {
//		return err
//	}
//	defer conn.Close()
//
//	type status struct {
//		Enabled goubus.Bool `json:"enabled"`
//	}
//	res, err := goubus.Call[status](ctx, conn, "mwan3", "status", nil)
//
// # Errors and discovery
//
// Non-zero ubus statuses and JSON-RPC refusals return a [*CallError] before
// payload decoding. Match status causes with errors.Is against [ErrNotFound],
// [ErrPermissionDenied], and the other root sentinels (or [ErrCall] for any
// call refusal), and inspect Service, Method, Code, and Message with
// errors.AsType.
//
// [List] queries object and method signatures on any [Caller] that implements
// [Lister], normalizing wire argument types to [ArgType] across both
// transports. [Bool], [StringOrSlice], [StringInt], and [UnixTime] decode JSON
// fields whose representation varies across OpenWrt releases.
//
// # Documentation
//
//   - [services/*]: 15 typed client packages for system, network, uci, dhcp,
//     luci, wireless, and nine other OpenWrt ubus domains.
//   - [architecture]: module layout, dependency direction, and wire paths.
//   - [design]: calls, errors, signature discovery, firmware quirks, and API
//     snapshots.
//
// [services/*]: https://pkg.go.dev/github.com/honeybbq/goubus/v2#section-directories
// [architecture]: https://github.com/honeybbq/goubus/blob/main/docs/architecture.md
// [design]: https://github.com/honeybbq/goubus/blob/main/docs/design.md
package goubus
