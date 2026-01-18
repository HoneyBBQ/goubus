// Copyright (c) 2026 honeybbq
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package wireless

import (
	"context"

	"github.com/honeybbq/goubus/v2"
	"github.com/honeybbq/goubus/v2/internal/base/wireless"
)

// Manager handles wireless operations for standard x86/generic OpenWrt.
type Manager struct {
	base *wireless.Manager
}

func New(t goubus.Transport) *Manager {
	return &Manager{
		base: wireless.New(t),
	}
}

func (m *Manager) Devices(ctx context.Context) ([]string, error) {
	return m.base.Devices(ctx)
}

func (m *Manager) AssocList(ctx context.Context, device string) ([]Assoc, error) {
	return m.base.AssocList(ctx, device)
}

func (m *Manager) Info(ctx context.Context, device string) (*Info, error) {
	return m.base.Info(ctx, device)
}

func (m *Manager) Scan(ctx context.Context, device string) ([]ScanResult, error) {
	return m.base.Scan(ctx, device)
}

func (m *Manager) FreqList(ctx context.Context, device string) ([]any, error) {
	return m.base.FreqList(ctx, device)
}

func (m *Manager) TxPowerList(ctx context.Context, device string) ([]any, error) {
	return m.base.TxPowerList(ctx, device)
}

func (m *Manager) CountryList(ctx context.Context, device string) ([]any, error) {
	return m.base.CountryList(ctx, device)
}

func (m *Manager) Survey(ctx context.Context, device string) ([]any, error) {
	return m.base.Survey(ctx, device)
}

func (m *Manager) PhyName(ctx context.Context, section string) (string, error) {
	return m.base.PhyName(ctx, section)
}

// Type aliases for public use.
type (
	Info       = wireless.Info
	ScanResult = wireless.ScanResult
	Assoc      = wireless.Assoc
	AssocRate  = wireless.AssocRate
)
