// Copyright (c) 2026 honeybbq
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package hostapd

import (
	"context"

	"github.com/honeybbq/goubus/v2"
)

// Manager provides an interface for managing hostapd (WiFi AP).
type Manager struct {
	caller goubus.Transport
}

// New creates a new base hostapd Manager.
func New(t goubus.Transport) *Manager {
	return &Manager{caller: t}
}

// Reload reloads hostapd configuration.
func (m *Manager) Reload(ctx context.Context, phy string, radio int) error {
	params := map[string]any{
		"phy":   phy,
		"radio": radio,
	}
	_, err := m.caller.Call(ctx, "hostapd", "reload", params)

	return err
}

// BSSInfo retrieves BSS information for an interface.
func (m *Manager) BSSInfo(ctx context.Context, iface string) (map[string]any, error) {
	params := map[string]any{"iface": iface}

	res, err := goubus.Call[map[string]any](ctx, m.caller, "hostapd", "bss_info", params)
	if err != nil {
		return nil, err
	}

	return *res, nil
}

// AP returns an APContext for a specific hostapd AP instance.
func (m *Manager) AP(name string) *APContext {
	return &APContext{
		manager: m,
		name:    name,
	}
}

type APContext struct {
	manager *Manager
	name    string
}

// GetClients retrieves the list of connected clients.
func (c *APContext) GetClients(ctx context.Context) (map[string]any, error) {
	res, err := goubus.Call[map[string]any](ctx, c.manager.caller, c.name, "get_clients", nil)
	if err != nil {
		return nil, err
	}

	return *res, nil
}

// GetStatus retrieves the status of the AP.
func (c *APContext) GetStatus(ctx context.Context) (map[string]any, error) {
	res, err := goubus.Call[map[string]any](ctx, c.manager.caller, c.name, "get_status", nil)
	if err != nil {
		return nil, err
	}

	return *res, nil
}

// DelClient removes a connected client.
func (c *APContext) DelClient(ctx context.Context, addr string, reason int, deauth bool, banTime int) error {
	params := map[string]any{
		"addr":     addr,
		"reason":   reason,
		"deauth":   deauth,
		"ban_time": banTime,
	}
	_, err := c.manager.caller.Call(ctx, c.name, "del_client", params)

	return err
}

// SwitchChan switches the channel of the AP.
func (c *APContext) SwitchChan(ctx context.Context, freq, bandwidth int) error {
	params := map[string]any{
		"freq":      freq,
		"bandwidth": bandwidth,
	}
	_, err := c.manager.caller.Call(ctx, c.name, "switch_chan", params)

	return err
}
