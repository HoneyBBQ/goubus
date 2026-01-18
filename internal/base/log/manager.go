// Copyright (c) 2026 honeybbq
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package log

import (
	"context"

	"github.com/honeybbq/goubus/v2"
)

// Manager provides methods to interact with the system log.
type Manager struct {
	caller goubus.Transport
}

// New creates a new base log Manager.
func New(t goubus.Transport) *Manager {
	return &Manager{caller: t}
}

// Read retrieves log entries.
func (m *Manager) Read(ctx context.Context, lines int, stream bool, oneshot bool) (*Log, error) {
	params := map[string]any{
		"lines":   lines,
		"stream":  stream,
		"oneshot": oneshot,
	}

	return goubus.Call[Log](ctx, m.caller, "log", "read", params)
}

// Write sends a log entry.
func (m *Manager) Write(ctx context.Context, event string) error {
	params := map[string]any{
		"event": event,
	}
	_, err := m.caller.Call(ctx, "log", "write", params)

	return err
}
