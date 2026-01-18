// Copyright (c) 2026 honeybbq
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package system

import (
	"context"

	"github.com/honeybbq/goubus/v2"
	"github.com/honeybbq/goubus/v2/internal/base/system"
)

// Manager handles system operations for CMCC RAX3000M.
type Manager struct {
	base *system.Manager
}

func New(t goubus.Transport) *Manager {
	return &Manager{
		base: system.New(t),
	}
}

func (m *Manager) Info(ctx context.Context) (*Info, error) {
	return m.base.Info(ctx)
}

func (m *Manager) Board(ctx context.Context) (*BoardInfo, error) {
	return m.base.Board(ctx)
}

func (m *Manager) Reboot(ctx context.Context) error {
	return m.base.Reboot(ctx)
}

func (m *Manager) Watchdog(ctx context.Context, req WatchdogRequest) error {
	return m.base.Watchdog(ctx, req)
}

func (m *Manager) Signal(ctx context.Context, pid, signum int) error {
	return m.base.Signal(ctx, pid, signum)
}

func (m *Manager) ValidateFirmwareImage(ctx context.Context, path string) (map[string]any, error) {
	return m.base.ValidateFirmwareImage(ctx, path)
}

func (m *Manager) Sysupgrade(ctx context.Context, req SysupgradeRequest) error {
	return m.base.Sysupgrade(ctx, req)
}

// Type aliases for public use.
type (
	Info                         = system.Info
	BoardInfo                    = system.BoardInfo
	WatchdogRequest              = system.WatchdogRequest
	SignalRequest                = system.SignalRequest
	ValidateFirmwareImageRequest = system.ValidateFirmwareImageRequest
	SysupgradeRequest            = system.SysupgradeRequest
)
