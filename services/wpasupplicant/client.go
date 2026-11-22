// Package wpasupplicant provides a client for the ubus wpa_supplicant service.
package wpasupplicant

import (
	"context"

	"github.com/honeybbq/goubus/v2"
)

// Client provides access to the ubus wpa_supplicant service.
type Client struct {
	caller goubus.Caller
}

// New creates a wpa_supplicant client for caller.
func New(caller goubus.Caller) *Client {
	if caller == nil {
		panic("goubus/wpasupplicant: nil caller")
	}

	return &Client{caller: caller}
}

// IfaceStatus retrieves the status of a wireless interface.
func (c *Client) IfaceStatus(ctx context.Context, name string) (map[string]any, error) {
	return goubus.Call[map[string]any](ctx, c.caller, "wpa_supplicant", "iface_status", map[string]any{"name": name})
}

// STA returns a context for a wpa_supplicant STA object.
func (c *Client) STA(name string) *STAContext {
	return &STAContext{caller: c.caller, name: name}
}

// STAContext provides access to one wpa_supplicant STA object.
type STAContext struct {
	caller goubus.Caller
	name   string
}

// Reload reloads the STA interface.
func (s *STAContext) Reload(ctx context.Context) error {
	_, err := s.caller.Call(ctx, s.name, "reload", nil)

	return err
}

// WPSStart starts WPS on the interface.
func (s *STAContext) WPSStart(ctx context.Context, multiAP bool) error {
	_, err := s.caller.Call(ctx, s.name, "wps_start", map[string]any{"multi_ap": multiAP})

	return err
}

// WPSCancel cancels a pending WPS operation.
func (s *STAContext) WPSCancel(ctx context.Context) error {
	_, err := s.caller.Call(ctx, s.name, "wps_cancel", nil)

	return err
}

// Control sends a control command to wpa_supplicant.
func (s *STAContext) Control(ctx context.Context, command string) (string, error) {
	res, err := goubus.Call[map[string]string](ctx, s.caller, s.name, "control", map[string]any{"command": command})
	if err != nil {
		return "", err
	}

	return res["result"], nil
}
