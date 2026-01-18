// Copyright (c) 2026 honeybbq
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package wpa_supplicant

import (
	"context"

	"github.com/honeybbq/goubus/v2"
	"github.com/honeybbq/goubus/v2/internal/base/wpa_supplicant"
)

// Manager handles wpa_supplicant operations for CMCC RAX3000M.
type Manager struct {
	base *wpa_supplicant.Manager
}

func New(t goubus.Transport) *Manager {
	return &Manager{
		base: wpa_supplicant.New(t),
	}
}

func (m *Manager) IfaceStatus(ctx context.Context, name string) (map[string]any, error) {
	return m.base.IfaceStatus(ctx, name)
}

func (m *Manager) STA(name string) *STAContext {
	return m.base.STA(name)
}

// Type aliases for public use.
type (
	STAContext = wpa_supplicant.STAContext
)
