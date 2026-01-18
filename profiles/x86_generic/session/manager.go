// Copyright (c) 2026 honeybbq
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package session

import (
	"context"

	"github.com/honeybbq/goubus/v2"
	"github.com/honeybbq/goubus/v2/internal/base/session"
)

// Manager handles session operations for standard x86/generic OpenWrt.
type Manager struct {
	base *session.Manager
}

func New(t goubus.Transport) *Manager {
	return &Manager{
		base: session.New(t),
	}
}

func (m *Manager) Create(ctx context.Context, timeout int) (*Data, error) {
	return m.base.Create(ctx, timeout)
}

func (m *Manager) List(ctx context.Context) (map[string]any, error) {
	return m.base.List(ctx)
}

func (m *Manager) Grant(ctx context.Context, req GrantRequest) error {
	return m.base.Grant(ctx, req)
}

func (m *Manager) Revoke(ctx context.Context, req GrantRequest) error {
	return m.base.Revoke(ctx, req)
}

func (m *Manager) Access(ctx context.Context, req AccessRequest) (bool, error) {
	return m.base.Access(ctx, req)
}

func (m *Manager) Set(ctx context.Context, sessionID string, values map[string]any) error {
	return m.base.Set(ctx, sessionID, values)
}

func (m *Manager) Get(ctx context.Context, sessionID string, keys []string) (map[string]any, error) {
	return m.base.Get(ctx, sessionID, keys)
}

func (m *Manager) Unset(ctx context.Context, sessionID string, keys []string) error {
	return m.base.Unset(ctx, sessionID, keys)
}

func (m *Manager) Destroy(ctx context.Context, sessionID string) error {
	return m.base.Destroy(ctx, sessionID)
}

func (m *Manager) Login(ctx context.Context, req LoginRequest) (*Data, error) {
	return m.base.Login(ctx, req)
}

// Type aliases for public use.
type (
	Data          = session.Data
	GrantRequest  = session.GrantRequest
	AccessRequest = session.AccessRequest
	LoginRequest  = session.LoginRequest
)
