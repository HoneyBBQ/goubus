// Copyright (c) 2026 honeybbq
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package container

import (
	"context"

	"github.com/honeybbq/goubus/v2"
	"github.com/honeybbq/goubus/v2/internal/base/container"
)

// Manager handles container operations for CMCC RAX3000M.
type Manager struct {
	base *container.Manager
}

func New(t goubus.Transport) *Manager {
	return &Manager{
		base: container.New(t),
	}
}

func (m *Manager) Set(ctx context.Context, req SetRequest) error {
	return m.base.Set(ctx, req)
}

func (m *Manager) Add(ctx context.Context, req SetRequest) error {
	return m.base.Add(ctx, req)
}

func (m *Manager) List(ctx context.Context, name string, verbose bool) (map[string]any, error) {
	return m.base.List(ctx, name, verbose)
}

func (m *Manager) Delete(ctx context.Context, name, instance string) error {
	return m.base.Delete(ctx, name, instance)
}

func (m *Manager) State(ctx context.Context, name string, spawn bool) (map[string]any, error) {
	return m.base.State(ctx, name, spawn)
}

func (m *Manager) GetFeatures(ctx context.Context) (map[string]any, error) {
	return m.base.GetFeatures(ctx)
}

func (m *Manager) ConsoleSet(ctx context.Context, name, instance string) error {
	return m.base.ConsoleSet(ctx, name, instance)
}

func (m *Manager) ConsoleAttach(ctx context.Context, name, instance string) error {
	return m.base.ConsoleAttach(ctx, name, instance)
}

// Type aliases for public use.
type (
	SetRequest = container.SetRequest
)
