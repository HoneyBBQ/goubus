// Copyright (c) 2026 honeybbq
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package rpcsys

import (
	"context"

	"github.com/honeybbq/goubus/v2"
	"github.com/honeybbq/goubus/v2/internal/base/rpcsys"
)

// Manager handles rpc-sys operations for CMCC RAX3000M.
type Manager struct {
	base *rpcsys.Manager
}

func New(t goubus.Transport) *Manager {
	return &Manager{
		base: rpcsys.New(t),
	}
}

func (m *Manager) PackageList(ctx context.Context, all bool) (map[string]any, error) {
	return m.base.PackageList(ctx, all)
}

func (m *Manager) PasswordSet(ctx context.Context, user, password string) error {
	return m.base.PasswordSet(ctx, user, password)
}

func (m *Manager) Factory(ctx context.Context) error {
	return m.base.Factory(ctx)
}

func (m *Manager) UpgradeStart(ctx context.Context, keep bool) error {
	return m.base.UpgradeStart(ctx, keep)
}

func (m *Manager) UpgradeTest(ctx context.Context) error {
	return m.base.UpgradeTest(ctx)
}

func (m *Manager) UpgradeClean(ctx context.Context) error {
	return m.base.UpgradeClean(ctx)
}

func (m *Manager) Reboot(ctx context.Context) error {
	return m.base.Reboot(ctx)
}
