// Copyright (c) 2026 honeybbq
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package system

import (
	"context"

	"github.com/honeybbq/goubus/v2"
)

// Manager provides methods to interact with system-wide information.
type Manager struct {
	caller goubus.Transport
}

// New creates a new base system Manager.
func New(t goubus.Transport) *Manager {
	return &Manager{caller: t}
}

// Info retrieves runtime system information.
func (m *Manager) Info(ctx context.Context) (*Info, error) {
	return goubus.Call[Info](ctx, m.caller, "system", "info", nil)
}

// Board retrieves board hardware information.
func (m *Manager) Board(ctx context.Context) (*BoardInfo, error) {
	return goubus.Call[BoardInfo](ctx, m.caller, "system", "board", nil)
}

// Reboot reboots the system.
func (m *Manager) Reboot(ctx context.Context) error {
	_, err := m.caller.Call(ctx, "system", "reboot", nil)

	return err
}

// Watchdog configures the system watchdog.
func (m *Manager) Watchdog(ctx context.Context, req WatchdogRequest) error {
	_, err := m.caller.Call(ctx, "system", "watchdog", req)

	return err
}

// Signal sends a signal to a process.
func (m *Manager) Signal(ctx context.Context, pid, signum int) error {
	req := SignalRequest{Pid: pid, Signum: signum}
	_, err := m.caller.Call(ctx, "system", "signal", req)

	return err
}

// ValidateFirmwareImage validates a firmware image file.
func (m *Manager) ValidateFirmwareImage(ctx context.Context, path string) (map[string]any, error) {
	req := ValidateFirmwareImageRequest{Path: path}

	res, err := goubus.Call[map[string]any](ctx, m.caller, "system", "validate_firmware_image", req)
	if err != nil {
		return nil, err
	}

	return *res, nil
}

// Sysupgrade performs a system upgrade.
func (m *Manager) Sysupgrade(ctx context.Context, req SysupgradeRequest) error {
	_, err := m.caller.Call(ctx, "system", "sysupgrade", req)

	return err
}
