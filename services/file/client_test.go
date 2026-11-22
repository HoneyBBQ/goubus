package file

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/honeybbq/goubus/v2"
)

type methodCase struct {
	params any
	invoke func(*Client) error
	method string
}

func fixture(t *testing.T, name string) json.RawMessage {
	t.Helper()
	// Fixture names are fixed by this test package.
	// #nosec G304
	data, err := os.ReadFile(filepath.Join("..", "..", "internal", "testdata", "rax3000m", name))
	if err != nil {
		t.Fatal(err)
	}

	return data
}

func assertCall(t *testing.T, method string, params any, raw json.RawMessage, invoke func(*Client) error) {
	t.Helper()
	called := false
	caller := goubus.CallerFunc(func(
		_ context.Context,
		service, gotMethod string,
		gotParams any,
	) (goubus.Result, error) {
		called = true
		if service != "file" || gotMethod != method || !reflect.DeepEqual(gotParams, params) {
			t.Errorf("Call(%q, %q, %#v), want (%q, %q, %#v)", service, gotMethod, gotParams, "file", method, params)
		}

		return goubus.NewJSONResult(raw), nil
	})
	err := invoke(New(caller))
	if err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("caller was not invoked")
	}
}

func TestNewPanicsOnNilCaller(t *testing.T) {
	defer func() {
		if got := recover(); got != "goubus/file: nil caller" {
			t.Fatalf("panic = %v", got)
		}
	}()
	New(nil)
}

func TestListSample(t *testing.T) {
	ctx := context.Background()
	raw := fixture(t, "file_list_etc_config.json")
	params := map[string]any{"path": "/etc/config"}
	assertCall(t, "list", params, raw, func(c *Client) error {
		got, err := c.List(ctx, "/etc/config")
		if err != nil {
			return err
		}
		if len(got.Entries) == 0 {
			t.Fatal("file list sample decoded without entries")
		}
		entry := got.Entries[0]
		if entry.Mode == 0 || entry.Atime == 0 || entry.Mtime == 0 || entry.Ctime == 0 || entry.Inode == 0 {
			t.Errorf("file list metadata was not decoded: %+v", entry)
		}

		return nil
	})
}

func TestStatSample(t *testing.T) {
	ctx := context.Background()
	raw := fixture(t, "file_stat_network.json")
	params := map[string]any{"path": "/etc/config/network"}
	assertCall(t, "stat", params, raw, func(c *Client) error {
		got, err := c.Stat(ctx, "/etc/config/network")
		if err != nil {
			return err
		}
		metadataMissing := got.Size != 1860 || got.Mode == 0 || got.Atime == 0 || got.Mtime == 0
		metadataMissing = metadataMissing || got.Ctime == 0 || got.Inode == 0 || got.UID != 0 || got.GID != 0
		if metadataMissing {
			t.Errorf("file stat metadata was not decoded: %+v", got)
		}

		return nil
	})
}

func TestReadNotFound(t *testing.T) {
	caller := goubus.CallerFunc(func(context.Context, string, string, any) (goubus.Result, error) {
		return nil, goubus.ErrNotFound
	})
	_, err := New(caller).Read(context.Background(), "/missing", false)
	if !errors.Is(err, goubus.ErrNotFound) {
		t.Fatalf("Read error = %v, want ErrNotFound", err)
	}
}

func TestReadWriteMethods(t *testing.T) {
	ctx := context.Background()
	methods := []methodCase{
		{
			method: "read", params: map[string]any{"path": "/tmp/a", "base64": true},
			invoke: func(c *Client) error {
				_, err := c.Read(ctx, "/tmp/a", true)

				return err
			},
		},
		{
			method: "write",
			params: map[string]any{
				"path": "/tmp/a", "data": "x", "append": true,
				"mode": int(os.FileMode(0600)), "base64": true,
			},
			invoke: func(c *Client) error { return c.Write(ctx, "/tmp/a", "x", true, 0600, true) },
		},
	}
	runMethodCases(t, methods)
}

func TestFileMetadataMethods(t *testing.T) {
	ctx := context.Background()
	methods := []methodCase{
		{
			method: "list", params: map[string]any{"path": "/tmp"},
			invoke: func(c *Client) error {
				_, err := c.List(ctx, "/tmp")

				return err
			},
		},
		{
			method: "stat", params: map[string]any{"path": "/tmp/a"},
			invoke: func(c *Client) error {
				_, err := c.Stat(ctx, "/tmp/a")

				return err
			},
		},
		{
			method: "remove", params: map[string]any{"path": "/tmp/a"},
			invoke: func(c *Client) error { return c.Remove(ctx, "/tmp/a") },
		},
	}
	runMethodCases(t, methods)
}

func TestHashExecMethods(t *testing.T) {
	ctx := context.Background()
	methods := []methodCase{
		{
			method: "md5", params: map[string]any{"path": "/tmp/a"},
			invoke: func(c *Client) error {
				_, err := c.MD5(ctx, "/tmp/a")

				return err
			},
		},
		{
			method: "exec",
			params: map[string]any{
				"command": "echo", "params": []string{"ok"}, "env": map[string]string{"A": "b"},
			},
			invoke: func(c *Client) error {
				_, err := c.Exec(ctx, "echo", []string{"ok"}, map[string]string{"A": "b"})

				return err
			},
		},
		{
			method: "lstat", params: map[string]any{"path": "/tmp/link"},
			invoke: func(c *Client) error {
				_, err := c.LStat(ctx, "/tmp/link")

				return err
			},
		},
	}
	runMethodCases(t, methods)
}

func runMethodCases(t *testing.T, methods []methodCase) {
	t.Helper()
	for _, tc := range methods {
		t.Run(tc.method, func(t *testing.T) {
			raw := json.RawMessage(`{}`)
			if tc.method == "md5" {
				raw = json.RawMessage(`{"md5":"abc"}`)
			}
			assertCall(t, tc.method, tc.params, raw, tc.invoke)
		})
	}
}
