// Copyright (c) 2026 honeybbq
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package rc

import (
	"context"

	"github.com/honeybbq/goubus/v2"
)

// Manager provides methods to interact with init scripts.
type Manager struct {
	caller goubus.Transport
}

// New creates a new base RC Manager.
func New(t goubus.Transport) *Manager {
	return &Manager{caller: t}
}

// List retrieves a list of available init scripts.
func (m *Manager) List(ctx context.Context, name string, skipRunningCheck bool) (map[string]ListInfo, error) {
	params := make(map[string]any)
	if name != "" {
		params["name"] = name
	}

	if skipRunningCheck {
		params["skip_running_check"] = true
	}

	resp, err := goubus.Call[map[string]ListInfo](ctx, m.caller, "rc", "list", params)
	if err != nil {
		return nil, err
	}

	return *resp, nil
}

// Init performs an init script action.
func (m *Manager) Init(ctx context.Context, name, action string) error {
	req := InitRequest{
		Name:   name,
		Action: action,
	}
	_, err := m.caller.Call(ctx, "rc", "init", req)

	return err
}
