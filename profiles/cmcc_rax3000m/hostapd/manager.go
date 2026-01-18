// Copyright (c) 2026 honeybbq
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package hostapd

import (
	"context"

	"github.com/honeybbq/goubus/v2"
	"github.com/honeybbq/goubus/v2/internal/base/hostapd"
)

// Manager handles hostapd operations for CMCC RAX3000M.
type Manager struct {
	base *hostapd.Manager
}

func New(t goubus.Transport) *Manager {
	return &Manager{
		base: hostapd.New(t),
	}
}

func (m *Manager) Reload(ctx context.Context, phy string, radio int) error {
	return m.base.Reload(ctx, phy, radio)
}

func (m *Manager) BSSInfo(ctx context.Context, iface string) (map[string]any, error) {
	return m.base.BSSInfo(ctx, iface)
}

func (m *Manager) AP(name string) *APContext {
	return m.base.AP(name)
}

// Type aliases for public use.
type (
	APContext = hostapd.APContext
)
