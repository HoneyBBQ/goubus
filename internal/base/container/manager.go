// Copyright (c) 2026 honeybbq
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package container

import (
	"context"

	"github.com/honeybbq/goubus/v2"
)

// Manager provides an interface for managing LxC containers.
type Manager struct {
	caller goubus.Transport
}

// New creates a new base container Manager.
func New(t goubus.Transport) *Manager {
	return &Manager{caller: t}
}

// Set configures a container.
func (m *Manager) Set(ctx context.Context, req SetRequest) error {
	_, err := m.caller.Call(ctx, "container", "set", req)

	return err
}

// Add adds a container.
func (m *Manager) Add(ctx context.Context, req SetRequest) error {
	_, err := m.caller.Call(ctx, "container", "add", req)

	return err
}

// List retrieves the status of containers.
func (m *Manager) List(ctx context.Context, name string, verbose bool) (map[string]any, error) {
	params := make(map[string]any)
	if name != "" {
		params["name"] = name
	}

	if verbose {
		params["verbose"] = true
	}

	res, err := goubus.Call[map[string]any](ctx, m.caller, "container", "list", params)
	if err != nil {
		return nil, err
	}

	return *res, nil
}

// Delete removes a container instance.
func (m *Manager) Delete(ctx context.Context, name, instance string) error {
	params := map[string]any{
		"name":     name,
		"instance": instance,
	}
	_, err := m.caller.Call(ctx, "container", "delete", params)

	return err
}

// State retrieves the state of a container.
func (m *Manager) State(ctx context.Context, name string, spawn bool) (map[string]any, error) {
	params := map[string]any{
		"name":  name,
		"spawn": spawn,
	}

	res, err := goubus.Call[map[string]any](ctx, m.caller, "container", "state", params)
	if err != nil {
		return nil, err
	}

	return *res, nil
}

// GetFeatures retrieves supported container features.
func (m *Manager) GetFeatures(ctx context.Context) (map[string]any, error) {
	res, err := goubus.Call[map[string]any](ctx, m.caller, "container", "get_features", nil)
	if err != nil {
		return nil, err
	}

	return *res, nil
}

// ConsoleSet sets the console for a container instance.
func (m *Manager) ConsoleSet(ctx context.Context, name, instance string) error {
	params := map[string]any{
		"name":     name,
		"instance": instance,
	}
	_, err := m.caller.Call(ctx, "container", "console_set", params)

	return err
}

// ConsoleAttach attaches to the console of a container instance.
func (m *Manager) ConsoleAttach(ctx context.Context, name, instance string) error {
	params := map[string]any{
		"name":     name,
		"instance": instance,
	}
	_, err := m.caller.Call(ctx, "container", "console_attach", params)

	return err
}
