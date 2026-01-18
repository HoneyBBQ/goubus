// Copyright (c) 2026 honeybbq
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package service

import (
	"context"

	"github.com/honeybbq/goubus/v2"
	"github.com/honeybbq/goubus/v2/internal/base/service"
)

// Manager handles service operations for standard x86/generic OpenWrt.
type Manager struct {
	base *service.Manager
}

func New(t goubus.Transport) *Manager {
	return &Manager{
		base: service.New(t),
	}
}

func (m *Manager) List(ctx context.Context, name string, verbose bool) (map[string]Info, error) {
	return m.base.List(ctx, name, verbose)
}

func (m *Manager) Delete(ctx context.Context, name, instance string) error {
	return m.base.Delete(ctx, name, instance)
}

func (m *Manager) Signal(ctx context.Context, name, instance string, signal int) error {
	return m.base.Signal(ctx, name, instance, signal)
}

func (m *Manager) Set(ctx context.Context, req SetRequest) error {
	return m.base.Set(ctx, req)
}

func (m *Manager) Add(ctx context.Context, req SetRequest) error {
	return m.base.Add(ctx, req)
}

func (m *Manager) UpdateStart(ctx context.Context, name string) error {
	return m.base.UpdateStart(ctx, name)
}

func (m *Manager) UpdateComplete(ctx context.Context, name string) error {
	return m.base.UpdateComplete(ctx, name)
}

func (m *Manager) Event(ctx context.Context, req EventRequest) error {
	return m.base.Event(ctx, req)
}

func (m *Manager) Validate(ctx context.Context, req ValidateRequest) (map[string]any, error) {
	return m.base.Validate(ctx, req)
}

func (m *Manager) GetData(ctx context.Context, name, instance, dataType string) (map[string]any, error) {
	return m.base.GetData(ctx, name, instance, dataType)
}

func (m *Manager) State(ctx context.Context, name string, spawn bool) (map[string]any, error) {
	return m.base.State(ctx, name, spawn)
}

func (m *Manager) Watchdog(ctx context.Context, name, instance string, mode, timeout int) error {
	return m.base.Watchdog(ctx, name, instance, mode, timeout)
}

// Type aliases for public use.
type (
	Info            = service.Info
	Instance        = service.Instance
	SetRequest      = service.SetRequest
	EventRequest    = service.EventRequest
	ValidateRequest = service.ValidateRequest
)
