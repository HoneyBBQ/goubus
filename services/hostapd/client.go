// Package hostapd provides a client for the ubus hostapd service.
package hostapd

import (
	"context"

	"github.com/honeybbq/goubus/v2"
)

// Client provides access to the ubus hostapd service.
type Client struct {
	caller goubus.Caller
}

// New creates a hostapd client for caller.
func New(caller goubus.Caller) *Client {
	if caller == nil {
		panic("goubus/hostapd: nil caller")
	}

	return &Client{caller: caller}
}

// Reload reloads hostapd configuration.
func (c *Client) Reload(ctx context.Context, phy string, radio int) error {
	_, err := c.caller.Call(ctx, "hostapd", "reload", map[string]any{"phy": phy, "radio": radio})

	return err
}

// BSSInfo retrieves BSS information for an interface.
func (c *Client) BSSInfo(ctx context.Context, iface string) (map[string]any, error) {
	return goubus.Call[map[string]any](ctx, c.caller, "hostapd", "bss_info", map[string]any{"iface": iface})
}

// AP returns a context for a hostapd AP instance.
func (c *Client) AP(name string) *APContext {
	return &APContext{caller: c.caller, name: name}
}

// APContext provides access to one hostapd AP instance.
type APContext struct {
	caller goubus.Caller
	name   string
}

// Clients retrieves the connected clients for the AP.
func (a *APContext) Clients(ctx context.Context) (map[string]any, error) {
	return goubus.Call[map[string]any](ctx, a.caller, a.name, "get_clients", nil)
}

// Status retrieves the status of the AP.
func (a *APContext) Status(ctx context.Context) (map[string]any, error) {
	return goubus.Call[map[string]any](ctx, a.caller, a.name, "get_status", nil)
}

// DelClient removes a connected client.
func (a *APContext) DelClient(ctx context.Context, addr string, reason int, deauth bool, banTime int) error {
	params := map[string]any{
		"addr": addr, "reason": reason, "deauth": deauth, "ban_time": banTime,
	}
	_, err := a.caller.Call(ctx, a.name, "del_client", params)

	return err
}

// SwitchChan switches the channel of the AP.
func (a *APContext) SwitchChan(ctx context.Context, freq, bandwidth int) error {
	_, err := a.caller.Call(ctx, a.name, "switch_chan", map[string]any{"freq": freq, "bandwidth": bandwidth})

	return err
}
