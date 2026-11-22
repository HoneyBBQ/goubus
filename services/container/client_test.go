package container

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/honeybbq/goubus/v2"
)

func fixture(t *testing.T, profile, name string) json.RawMessage {
	t.Helper()
	// Profile and sample names are fixed by the test table.
	// #nosec G304
	data, err := os.ReadFile(filepath.Join("..", "..", "internal", "testdata", profile, name))
	if err != nil {
		t.Fatal(err)
	}

	return data
}

func assertCall(t *testing.T, service, method string, params any, raw json.RawMessage, invoke func(*Client) error) {
	t.Helper()
	called := false
	caller := goubus.CallerFunc(func(
		_ context.Context,
		gotService, gotMethod string,
		gotParams any,
	) (goubus.Result, error) {
		called = true
		if gotService != service || gotMethod != method || !reflect.DeepEqual(gotParams, params) {
			t.Errorf("Call(%q, %q, %#v), want (%q, %q, %#v)", gotService, gotMethod, gotParams, service, method, params)
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
		if got := recover(); got != "goubus/container: nil caller" {
			t.Fatalf("panic = %v", got)
		}
	}()
	New(nil)
}

func TestSamples(t *testing.T) {
	for _, tc := range []struct {
		profile string
		name    string
		method  string
	}{
		{profile: "rax3000m", name: "container_get_features.json", method: "features"},
		{profile: "rax3000m", name: "container_list.json", method: "list"},
		{profile: "x86_generic", name: "container_list.json", method: "list"},
	} {
		t.Run(tc.profile+"/"+tc.name, func(t *testing.T) {
			raw := fixture(t, tc.profile, tc.name)
			params := any(map[string]any{})
			if tc.method == "features" {
				params = nil
			}
			method := map[string]string{"features": "get_features", "list": "list"}[tc.method]
			assertCall(t, "container", method, params, raw, func(c *Client) error {
				if tc.method == "features" {
					_, err := c.Features(context.Background())

					return err
				}
				_, err := c.List(context.Background(), "", false)

				return err
			})
		})
	}
}

func TestMethods(t *testing.T) {
	ctx := context.Background()
	req := SetRequest{Name: "demo"}
	methods := []struct {
		params any
		invoke func(*Client) error
		method string
	}{
		{method: "set", params: req, invoke: func(c *Client) error { return c.Set(ctx, req) }},
		{method: "add", params: req, invoke: func(c *Client) error { return c.Add(ctx, req) }},
		{
			method: "list", params: map[string]any{"name": "demo", "verbose": true},
			invoke: func(c *Client) error {
				_, err := c.List(ctx, "demo", true)

				return err
			},
		},
		{
			method: "delete", params: map[string]any{"name": "demo", "instance": "one"},
			invoke: func(c *Client) error { return c.Delete(ctx, "demo", "one") },
		},
		{
			method: "state", params: map[string]any{"name": "demo", "spawn": true},
			invoke: func(c *Client) error {
				_, err := c.State(ctx, "demo", true)

				return err
			},
		},
		{method: "get_features", invoke: func(c *Client) error {
			_, err := c.Features(ctx)

			return err
		}},
		{
			method: "console_set", params: map[string]any{"name": "demo", "instance": "one"},
			invoke: func(c *Client) error { return c.ConsoleSet(ctx, "demo", "one") },
		},
		{
			method: "console_attach", params: map[string]any{"name": "demo", "instance": "one"},
			invoke: func(c *Client) error { return c.ConsoleAttach(ctx, "demo", "one") },
		},
	}
	for _, tc := range methods {
		t.Run(tc.method, func(t *testing.T) {
			assertCall(t, "container", tc.method, tc.params, json.RawMessage(`{}`), tc.invoke)
		})
	}
}
