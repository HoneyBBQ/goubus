// Copyright (c) 2026 honeybbq
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package session

import (
	"context"
	"time"

	"github.com/honeybbq/goubus/v2"
)

// Manager provides an interface for managing ubus sessions.
type Manager struct {
	caller goubus.Transport
}

// New creates a new base session Manager.
func New(t goubus.Transport) *Manager {
	return &Manager{caller: t}
}

// Create creates a new session.
func (m *Manager) Create(ctx context.Context, timeout int) (*Data, error) {
	params := map[string]any{"timeout": timeout}

	sessionData, err := goubus.Call[Data](ctx, m.caller, "session", "create", params)
	if err != nil {
		return nil, err
	}

	sessionData.ExpireTime = time.Now().Add(time.Duration(sessionData.Timeout) * time.Second)

	return sessionData, nil
}

// List lists all active sessions.
func (m *Manager) List(ctx context.Context) (map[string]any, error) {
	res, err := goubus.Call[map[string]any](ctx, m.caller, "session", "list", nil)
	if err != nil {
		return nil, err
	}

	return *res, nil
}

// Grant grants access to a session.
func (m *Manager) Grant(ctx context.Context, req GrantRequest) error {
	_, err := m.caller.Call(ctx, "session", "grant", req)

	return err
}

// Revoke revokes access from a session.
func (m *Manager) Revoke(ctx context.Context, req GrantRequest) error {
	_, err := m.caller.Call(ctx, "session", "revoke", req)

	return err
}

// Access checks access for a session.
func (m *Manager) Access(ctx context.Context, req AccessRequest) (bool, error) {
	res, err := goubus.Call[map[string]bool](ctx, m.caller, "session", "access", req)
	if err != nil {
		return false, err
	}

	return (*res)["access"], nil
}

// Set sets session values.
func (m *Manager) Set(ctx context.Context, session string, values map[string]any) error {
	req := map[string]any{
		"ubus_rpc_session": session,
		"values":           values,
	}
	_, err := m.caller.Call(ctx, "session", "set", req)

	return err
}

// Get retrieves session values.
func (m *Manager) Get(ctx context.Context, session string, keys []string) (map[string]any, error) {
	req := map[string]any{
		"ubus_rpc_session": session,
		"keys":             keys,
	}

	res, err := goubus.Call[map[string]any](ctx, m.caller, "session", "get", req)
	if err != nil {
		return nil, err
	}

	return *res, nil
}

// Unset unsets session values.
func (m *Manager) Unset(ctx context.Context, session string, keys []string) error {
	req := map[string]any{
		"ubus_rpc_session": session,
		"keys":             keys,
	}
	_, err := m.caller.Call(ctx, "session", "unset", req)

	return err
}

// Destroy destroys a session.
func (m *Manager) Destroy(ctx context.Context, session string) error {
	req := map[string]any{"ubus_rpc_session": session}
	_, err := m.caller.Call(ctx, "session", "destroy", req)

	return err
}

// Login performs a session login.
func (m *Manager) Login(ctx context.Context, req LoginRequest) (*Data, error) {
	sessionData, err := goubus.Call[Data](ctx, m.caller, "session", "login", req)
	if err != nil {
		return nil, err
	}

	sessionData.ExpireTime = time.Now().Add(time.Duration(sessionData.Timeout) * time.Second)

	return sessionData, nil
}
