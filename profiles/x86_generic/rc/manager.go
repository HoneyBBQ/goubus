// Copyright (c) 2026 honeybbq
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package rc

import (
	"context"

	"github.com/honeybbq/goubus/v2"
	"github.com/honeybbq/goubus/v2/internal/base/rc"
)

// Manager handles init script operations for standard x86/generic OpenWrt.
type Manager struct {
	base *rc.Manager
}

func New(t goubus.Transport) *Manager {
	return &Manager{
		base: rc.New(t),
	}
}

func (m *Manager) List(ctx context.Context, name string, skipRunningCheck bool) (map[string]ListInfo, error) {
	return m.base.List(ctx, name, skipRunningCheck)
}

func (m *Manager) Init(ctx context.Context, name, action string) error {
	return m.base.Init(ctx, name, action)
}

// Type aliases for public use.
type (
	ListInfo = rc.ListInfo
)
