// Copyright (c) 2026 honeybbq
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package wpa_supplicant

import (
	"context"

	"github.com/honeybbq/goubus/v2"
)

// Manager provides an interface for managing wpa_supplicant (WiFi STA).
type Manager struct {
	caller goubus.Transport
}

// New creates a new base wpa_supplicant Manager.
func New(t goubus.Transport) *Manager {
	return &Manager{caller: t}
}

// IfaceStatus retrieves the status of a wireless interface.
func (m *Manager) IfaceStatus(ctx context.Context, name string) (map[string]any, error) {
	params := map[string]any{"name": name}

	res, err := goubus.Call[map[string]any](ctx, m.caller, "wpa_supplicant", "iface_status", params)
	if err != nil {
		return nil, err
	}

	return *res, nil
}

// STA returns a STAContext for a specific wpa_supplicant STA instance.
func (m *Manager) STA(name string) *STAContext {
	return &STAContext{
		manager: m,
		name:    name,
	}
}

type STAContext struct {
	manager *Manager
	name    string
}

// Reload reloads the STA interface.
func (c *STAContext) Reload(ctx context.Context) error {
	_, err := c.manager.caller.Call(ctx, c.name, "reload", nil)

	return err
}

// WPSStart starts WPS on the interface.
func (c *STAContext) WPSStart(ctx context.Context, multiAP bool) error {
	params := map[string]any{"multi_ap": multiAP}
	_, err := c.manager.caller.Call(ctx, c.name, "wps_start", params)

	return err
}

// WPSCancel cancels a pending WPS operation.
func (c *STAContext) WPSCancel(ctx context.Context) error {
	_, err := c.manager.caller.Call(ctx, c.name, "wps_cancel", nil)

	return err
}

// Control sends a control command to wpa_supplicant.
func (c *STAContext) Control(ctx context.Context, command string) (string, error) {
	params := map[string]any{"command": command}

	res, err := goubus.Call[map[string]string](ctx, c.manager.caller, c.name, "control", params)
	if err != nil {
		return "", err
	}

	return (*res)["result"], nil
}
