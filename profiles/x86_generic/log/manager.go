// Copyright (c) 2026 honeybbq
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package log

import (
	"context"

	"github.com/honeybbq/goubus/v2"
	"github.com/honeybbq/goubus/v2/internal/base/log"
)

// Manager handles log operations for standard x86/generic OpenWrt.
type Manager struct {
	base *log.Manager
}

func New(t goubus.Transport) *Manager {
	return &Manager{
		base: log.New(t),
	}
}

func (m *Manager) Read(ctx context.Context, lines int, stream bool, oneshot bool) (*Log, error) {
	return m.base.Read(ctx, lines, stream, oneshot)
}

func (m *Manager) Write(ctx context.Context, event string) error {
	return m.base.Write(ctx, event)
}

// Type aliases for public use.
type (
	Log = log.Log
)
