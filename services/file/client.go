// Package file provides a client for the ubus file service.
package file

import (
	"context"
	"os"

	"github.com/honeybbq/goubus/v2"
)

// Client provides access to the ubus file service.
type Client struct {
	caller goubus.Caller
}

// New creates a file client for caller.
func New(caller goubus.Caller) *Client {
	if caller == nil {
		panic("goubus/file: nil caller")
	}

	return &Client{caller: caller}
}

// Read retrieves file contents.
func (c *Client) Read(ctx context.Context, path string, base64 bool) (*Read, error) {
	params := map[string]any{"path": path}
	if base64 {
		params["base64"] = true
	}

	return goubus.Call[*Read](ctx, c.caller, "file", "read", params)
}

// Write writes data to a file.
func (c *Client) Write(ctx context.Context, path, data string, isAppend bool, mode os.FileMode, base64 bool) error {
	params := map[string]any{"path": path, "data": data}
	if isAppend {
		params["append"] = true
	}
	if mode != 0 {
		params["mode"] = int(mode)
	}
	if base64 {
		params["base64"] = true
	}
	_, err := c.caller.Call(ctx, "file", "write", params)

	return err
}

// List lists directory contents.
func (c *Client) List(ctx context.Context, path string) (*List, error) {
	return goubus.Call[*List](ctx, c.caller, "file", "list", map[string]any{"path": path})
}

// Stat retrieves file metadata.
func (c *Client) Stat(ctx context.Context, path string) (*Stat, error) {
	return goubus.Call[*Stat](ctx, c.caller, "file", "stat", map[string]any{"path": path})
}

// Remove deletes a file.
func (c *Client) Remove(ctx context.Context, path string) error {
	_, err := c.caller.Call(ctx, "file", "remove", map[string]any{"path": path})

	return err
}

// MD5 calculates the MD5 hash of a file.
func (c *Client) MD5(ctx context.Context, path string) (string, error) {
	res, err := goubus.Call[map[string]string](ctx, c.caller, "file", "md5", map[string]any{"path": path})
	if err != nil {
		return "", err
	}

	return res["md5"], nil
}

// Exec executes a command on the device.
func (c *Client) Exec(ctx context.Context, command string, params []string, env map[string]string) (*Exec, error) {
	req := map[string]any{"command": command}
	if len(params) > 0 {
		req["params"] = params
	}
	if len(env) > 0 {
		req["env"] = env
	}

	return goubus.Call[*Exec](ctx, c.caller, "file", "exec", req)
}

// LStat retrieves symbolic link metadata.
func (c *Client) LStat(ctx context.Context, path string) (*Stat, error) {
	return goubus.Call[*Stat](ctx, c.caller, "file", "lstat", map[string]any{"path": path})
}

// List represents a directory listing.
type List struct {
	// Entries contains the directory entries.
	Entries []ListData `json:"entries"`
}

// ListData represents metadata for one directory entry.
type ListData struct {
	// Name is the entry name.
	Name string `json:"name"`
	// Type is the entry type.
	Type string `json:"type"`
	// User is the entry owner's user name.
	User string `json:"user"`
	// Group is the entry owner's group name.
	Group string `json:"group"`
	// Size is the entry size in bytes.
	Size uint64 `json:"size"`
	// Atime is the last access time as a Unix timestamp.
	Atime int64 `json:"atime"`
	// Mtime is the last modification time as a Unix timestamp.
	Mtime int64 `json:"mtime"`
	// Ctime is the last status change time as a Unix timestamp.
	Ctime int64 `json:"ctime"`
	// Inode is the entry inode number.
	Inode uint64 `json:"inode"`
	// Mode is the entry mode.
	Mode uint32 `json:"mode"`
	// UID is the entry owner's user identifier.
	UID uint32 `json:"uid"`
	// GID is the entry owner's group identifier.
	GID uint32 `json:"gid"`
}

// Stat represents file statistics.
type Stat struct {
	// Path is the file path.
	Path string `json:"path"`
	// Type is the file type.
	Type string `json:"type"`
	// User is the file owner's user name.
	User string `json:"user"`
	// Group is the file owner's group name.
	Group string `json:"group"`
	// Size is the file size in bytes.
	Size uint64 `json:"size"`
	// Atime is the last access time as a Unix timestamp.
	Atime int64 `json:"atime"`
	// Mtime is the last modification time as a Unix timestamp.
	Mtime int64 `json:"mtime"`
	// Ctime is the last status change time as a Unix timestamp.
	Ctime int64 `json:"ctime"`
	// Inode is the file inode number.
	Inode uint64 `json:"inode"`
	// Mode is the file mode.
	Mode uint32 `json:"mode"`
	// UID is the file owner's user identifier.
	UID uint32 `json:"uid"`
	// GID is the file owner's group identifier.
	GID uint32 `json:"gid"`
}

// Read represents file content.
type Read struct {
	// Data contains the file content.
	Data string `json:"data"`
}

// Exec represents command output.
type Exec struct {
	// Stdout contains standard output.
	Stdout string `json:"stdout"`
	// Stderr contains standard error output.
	Stderr string `json:"stderr"`
	// Code is the command exit code.
	Code int `json:"code"`
}
