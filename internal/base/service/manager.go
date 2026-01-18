// Copyright (c) 2026 honeybbq
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package service

import (
	"context"

	"github.com/honeybbq/goubus/v2"
)

// Manager provides an interface for managing system services.
type Manager struct {
	caller goubus.Transport
}

// New creates a new base service Manager.
func New(t goubus.Transport) *Manager {
	return &Manager{caller: t}
}

// List retrieves the status of all services.
func (m *Manager) List(ctx context.Context, name string, verbose bool) (map[string]Info, error) {
	params := make(map[string]any)
	if name != "" {
		params["name"] = name
	}

	if verbose {
		params["verbose"] = true
	}

	res, err := goubus.Call[map[string]Info](ctx, m.caller, "service", "list", params)
	if err != nil {
		return nil, err
	}

	return *res, nil
}

// Delete removes a service instance.
func (m *Manager) Delete(ctx context.Context, name, instance string) error {
	params := map[string]any{
		"name":     name,
		"instance": instance,
	}
	_, err := m.caller.Call(ctx, "service", "delete", params)

	return err
}

// Signal sends a Unix signal to a service instance.
func (m *Manager) Signal(ctx context.Context, name, instance string, signal int) error {
	params := map[string]any{
		"name":     name,
		"instance": instance,
		"signal":   signal,
	}
	_, err := m.caller.Call(ctx, "service", "signal", params)

	return err
}

// Set configures a service.
func (m *Manager) Set(ctx context.Context, req SetRequest) error {
	_, err := m.caller.Call(ctx, "service", "set", req)

	return err
}

// Add adds a service.
func (m *Manager) Add(ctx context.Context, req SetRequest) error {
	_, err := m.caller.Call(ctx, "service", "add", req)

	return err
}

// UpdateStart marks the start of a service update.
func (m *Manager) UpdateStart(ctx context.Context, name string) error {
	params := map[string]any{"name": name}
	_, err := m.caller.Call(ctx, "service", "update_start", params)

	return err
}

// UpdateComplete marks the completion of a service update.
func (m *Manager) UpdateComplete(ctx context.Context, name string) error {
	params := map[string]any{"name": name}
	_, err := m.caller.Call(ctx, "service", "update_complete", params)

	return err
}

// Event sends an event to the service.
func (m *Manager) Event(ctx context.Context, req EventRequest) error {
	_, err := m.caller.Call(ctx, "service", "event", req)

	return err
}

// Validate validates the service configuration.
func (m *Manager) Validate(ctx context.Context, req ValidateRequest) (map[string]any, error) {
	res, err := goubus.Call[map[string]any](ctx, m.caller, "service", "validate", req)
	if err != nil {
		return nil, err
	}

	return *res, nil
}

// GetData retrieves data from the service.
func (m *Manager) GetData(ctx context.Context, name, instance, dataType string) (map[string]any, error) {
	params := map[string]any{
		"name":     name,
		"instance": instance,
		"type":     dataType,
	}

	res, err := goubus.Call[map[string]any](ctx, m.caller, "service", "get_data", params)
	if err != nil {
		return nil, err
	}

	return *res, nil
}

// SetData sets data for the service (RAX300M specific).
func (m *Manager) SetData(ctx context.Context, name, instance string, data map[string]any) error {
	params := map[string]any{
		"name":     name,
		"instance": instance,
		"data":     data,
	}
	_, err := m.caller.Call(ctx, "service", "set_data", params)

	return err
}

// State retrieves the spawn state of a service.
func (m *Manager) State(ctx context.Context, name string, spawn bool) (map[string]any, error) {
	params := map[string]any{
		"name":  name,
		"spawn": spawn,
	}

	res, err := goubus.Call[map[string]any](ctx, m.caller, "service", "state", params)
	if err != nil {
		return nil, err
	}

	return *res, nil
}

// Watchdog configures the watchdog for a service instance.
func (m *Manager) Watchdog(ctx context.Context, name, instance string, mode, timeout int) error {
	params := map[string]any{
		"name":     name,
		"instance": instance,
		"mode":     mode,
		"timeout":  timeout,
	}
	_, err := m.caller.Call(ctx, "service", "watchdog", params)

	return err
}
