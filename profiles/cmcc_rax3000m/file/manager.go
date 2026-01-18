// Copyright (c) 2026 honeybbq
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package file

import (
	"context"
	"os"

	"github.com/honeybbq/goubus/v2"
	"github.com/honeybbq/goubus/v2/internal/base/file"
)

// Manager handles file operations for CMCC RAX3000M.
type Manager struct {
	base *file.Manager
}

func New(t goubus.Transport) *Manager {
	return &Manager{
		base: file.New(t),
	}
}

func (m *Manager) Read(ctx context.Context, path string, base64 bool) (*Read, error) {
	return m.base.Read(ctx, path, base64)
}

func (m *Manager) List(ctx context.Context, path string) (*List, error) {
	return m.base.List(ctx, path)
}

func (m *Manager) Write(ctx context.Context, path, data string, isAppend bool, mode os.FileMode, base64 bool) error {
	return m.base.Write(ctx, path, data, isAppend, mode, base64)
}

func (m *Manager) Stat(ctx context.Context, path string) (*Stat, error) {
	return m.base.Stat(ctx, path)
}

func (m *Manager) Remove(ctx context.Context, path string) error {
	return m.base.Remove(ctx, path)
}

func (m *Manager) MD5(ctx context.Context, path string) (string, error) {
	return m.base.MD5(ctx, path)
}

func (m *Manager) Exec(ctx context.Context, command string, params []string, env map[string]string) (*Exec, error) {
	return m.base.Exec(ctx, command, params, env)
}

func (m *Manager) LStat(ctx context.Context, path string) (*Stat, error) {
	return m.base.LStat(ctx, path)
}

// Type aliases for public use.
type (
	Read = file.Read
	List = file.List
	Stat = file.Stat
	Exec = file.Exec
)
