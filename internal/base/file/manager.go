// Copyright (c) 2026 honeybbq
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package file

import (
	"context"
	"os"

	"github.com/honeybbq/goubus/v2"
	"github.com/honeybbq/goubus/v2/errdefs"
)

// Manager provides methods to interact with the device's filesystem.
type Manager struct {
	caller goubus.Transport
}

// New creates a new base file Manager.
func New(t goubus.Transport) *Manager {
	return &Manager{caller: t}
}

// Read retrieves file contents.
func (m *Manager) Read(ctx context.Context, path string, base64 bool) (*Read, error) {
	params := map[string]any{"path": path}
	if base64 {
		params["base64"] = true
	}

	res, err := goubus.Call[Read](ctx, m.caller, "file", "read", params)
	if err != nil && errdefs.IsNotFound(err) {
		return nil, errdefs.Wrapf(err, "file module not found or file does not exist at %s", path)
	}

	return res, err
}

// Write writes data to a file.
func (m *Manager) Write(ctx context.Context, path, data string, isAppend bool, mode os.FileMode, base64 bool) error {
	params := map[string]any{
		"path": path,
		"data": data,
	}
	if isAppend {
		params["append"] = true
	}

	if mode != 0 {
		params["mode"] = int(mode)
	}

	if base64 {
		params["base64"] = true
	}

	_, err := m.caller.Call(ctx, "file", "write", params)

	return err
}

// List lists directory contents.
func (m *Manager) List(ctx context.Context, path string) (*List, error) {
	params := map[string]any{"path": path}

	return goubus.Call[List](ctx, m.caller, "file", "list", params)
}

// Stat retrieves file metadata.
func (m *Manager) Stat(ctx context.Context, path string) (*Stat, error) {
	params := map[string]any{"path": path}

	return goubus.Call[Stat](ctx, m.caller, "file", "stat", params)
}

// Remove deletes a file.
func (m *Manager) Remove(ctx context.Context, path string) error {
	params := map[string]any{"path": path}
	_, err := m.caller.Call(ctx, "file", "remove", params)

	return err
}

// MD5 calculates the MD5 hash of a file.
func (m *Manager) MD5(ctx context.Context, path string) (string, error) {
	params := map[string]any{"path": path}

	res, err := goubus.Call[map[string]string](ctx, m.caller, "file", "md5", params)
	if err != nil {
		return "", err
	}

	return (*res)["md5"], nil
}

// Exec executes a command on the device.
func (m *Manager) Exec(ctx context.Context, command string, params []string, env map[string]string) (*Exec, error) {
	req := map[string]any{
		"command": command,
	}
	if len(params) > 0 {
		req["params"] = params
	}

	if len(env) > 0 {
		req["env"] = env
	}

	return goubus.Call[Exec](ctx, m.caller, "file", "exec", req)
}

// LStat retrieves symbolic link metadata (RAX300M specific).
func (m *Manager) LStat(ctx context.Context, path string) (*Stat, error) {
	params := map[string]any{"path": path}

	return goubus.Call[Stat](ctx, m.caller, "file", "lstat", params)
}
