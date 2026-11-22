// Package services groups all 15 typed OpenWrt ubus clients behind one shared
// [goubus.Caller].
//
// When a program running on a router only touches one or two ubus objects,
// import those packages directly (for example
// [github.com/honeybbq/goubus/v2/services/system] or
// [github.com/honeybbq/goubus/v2/services/uci]) so unused service packages are
// not compiled into the binary.
//
// When a controller or dashboard works across many ubus objects, call [New]
// once per connection. The returned [Client] holds one instance of every
// service client, preserving per-client runtime caches such as the probed LuCI
// time method and DHCP DUID wire format. Pass [WithLuCI] or [WithDHCP] to pin
// firmware behavior when known ahead of time.
//
// [Client] borrows its [goubus.Caller] and never closes it; closing the
// underlying [goubus.Transport] remains the caller's responsibility.
package services
