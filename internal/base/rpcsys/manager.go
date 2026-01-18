// Copyright (c) 2026 honeybbq
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package rpcsys

import (
	"context"

	"github.com/honeybbq/goubus/v2"
)

// Manager provides an interface for 'rpc-sys' (System/Package management).
type Manager struct {
	caller goubus.Transport
}

// New creates a new base rpcsys Manager.
func New(t goubus.Transport) *Manager {
	return &Manager{caller: t}
}

// PackageList retrieves the list of installed packages.
func (m *Manager) PackageList(ctx context.Context, all bool) (map[string]any, error) {
	params := map[string]any{"all": all}

	res, err := goubus.Call[map[string]any](ctx, m.caller, "rpc-sys", "packagelist", params)
	if err != nil {
		return nil, err
	}

	return *res, nil
}

// PasswordSet sets the password for a user.
func (m *Manager) PasswordSet(ctx context.Context, user, password string) error {
	params := map[string]any{
		"user":     user,
		"password": password,
	}
	_, err := m.caller.Call(ctx, "rpc-sys", "password_set", params)

	return err
}

// Factory performs a factory reset.
func (m *Manager) Factory(ctx context.Context) error {
	_, err := m.caller.Call(ctx, "rpc-sys", "factory", nil)

	return err
}

// UpgradeStart starts a system upgrade.
func (m *Manager) UpgradeStart(ctx context.Context, keep bool) error {
	params := map[string]any{"keep": keep}
	_, err := m.caller.Call(ctx, "rpc-sys", "upgrade_start", params)

	return err
}

// UpgradeTest tests a system upgrade image.
func (m *Manager) UpgradeTest(ctx context.Context) error {
	_, err := m.caller.Call(ctx, "rpc-sys", "upgrade_test", nil)

	return err
}

// UpgradeClean cleans up after a system upgrade.
func (m *Manager) UpgradeClean(ctx context.Context) error {
	_, err := m.caller.Call(ctx, "rpc-sys", "upgrade_clean", nil)

	return err
}

// Reboot reboots the system.
func (m *Manager) Reboot(ctx context.Context) error {
	_, err := m.caller.Call(ctx, "rpc-sys", "reboot", nil)

	return err
}
