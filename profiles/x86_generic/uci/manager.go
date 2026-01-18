// Copyright (c) 2026 honeybbq
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package uci

import (
	"context"

	"github.com/honeybbq/goubus/v2"
	"github.com/honeybbq/goubus/v2/internal/base/uci"
)

type StandardDialect struct{}

// Manager handles UCI operations for standard x86/generic OpenWrt.
type Manager struct {
	base *uci.Manager
}

func New(t goubus.Transport) *Manager {
	return &Manager{
		base: uci.New(t, StandardDialect{}),
	}
}

func (m *Manager) Package(name string) *PackageContext {
	return m.base.Package(name)
}

func (m *Manager) Configs(ctx context.Context) ([]string, error) {
	return m.base.Configs(ctx)
}

func (m *Manager) State(ctx context.Context, req StateRequest) (*GetResponse, error) {
	return m.base.State(ctx, req)
}

func (m *Manager) Apply(ctx context.Context, rollback bool, timeout int) error {
	return m.base.Apply(ctx, rollback, timeout)
}

func (m *Manager) Confirm(ctx context.Context) error {
	return m.base.Confirm(ctx)
}

func (m *Manager) Rollback(ctx context.Context) error {
	return m.base.Rollback(ctx)
}

func (m *Manager) ReloadConfig(ctx context.Context) error {
	return m.base.ReloadConfig(ctx)
}

// Type aliases for public use.
type (
	SectionValues   = uci.SectionValues
	Section         = uci.Section
	PackageContext  = uci.PackageContext
	SectionContext  = uci.SectionContext
	OptionContext   = uci.OptionContext
	StateRequest    = uci.StateRequest
	GetResponse     = uci.GetResponse
	ChangesResponse = uci.ChangesResponse
)

func NewSectionValues() SectionValues {
	return uci.NewSectionValues()
}
