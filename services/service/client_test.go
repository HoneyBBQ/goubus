package service

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/honeybbq/goubus/v2"
)

func fixture(t *testing.T, profile string) json.RawMessage {
	t.Helper()
	// Profile values are fixed by this test table.
	// #nosec G304
	data, err := os.ReadFile(filepath.Join("..", "..", "internal", "testdata", profile, "service_list.json"))
	if err != nil {
		t.Fatal(err)
	}

	return data
}

func assertCall(t *testing.T, method string, params any, raw json.RawMessage, invoke func(*Client) error) {
	t.Helper()
	called := false
	caller := goubus.CallerFunc(func(_ context.Context, service, gotMethod string, gotParams any) (goubus.Result, error) {
		called = true
		if service != "service" || gotMethod != method || !reflect.DeepEqual(gotParams, params) {
			t.Errorf("Call(%q, %q, %#v), want (%q, %q, %#v)", service, gotMethod, gotParams, "service", method, params)
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
		if got := recover(); got != "goubus/service: nil caller" {
			t.Fatalf("panic = %v", got)
		}
	}()
	New(nil)
}

func TestSamples(t *testing.T) {
	ctx := context.Background()
	for _, profile := range []string{"rax3000m", "x86_generic"} {
		t.Run(profile, func(t *testing.T) {
			assertCall(t, "list", map[string]any{}, fixture(t, profile), func(c *Client) error {
				got, err := c.List(ctx, "", false)
				if err == nil && len(got) == 0 {
					t.Error("service fixture decoded without entries")
				}

				return err
			})
		})
	}
}

type methodCase struct {
	params any
	invoke func(*Client) error
	method string
}

func TestMethods(t *testing.T) {
	ctx := context.Background()
	methods := append(serviceActionMethodCases(ctx), serviceDataMethodCases(ctx)...)
	runMethodCases(t, methods)
}

func serviceActionMethodCases(ctx context.Context) []methodCase {
	req := SetRequest{Name: "demo"}
	event := EventRequest{Data: map[string]any{"x": 1}, Type: "reload"}

	return []methodCase{
		{
			method: "list", params: map[string]any{"name": "demo", "verbose": true},
			invoke: func(c *Client) error {
				_, err := c.List(ctx, "demo", true)

				return err
			},
		},
		{
			method: "delete", params: map[string]any{"name": "demo", "instance": "main"},
			invoke: func(c *Client) error { return c.Delete(ctx, "demo", "main") },
		},
		{
			method: "signal", params: map[string]any{"name": "demo", "instance": "main", "signal": 15},
			invoke: func(c *Client) error { return c.Signal(ctx, "demo", "main", 15) },
		},
		{method: "set", params: req, invoke: func(c *Client) error { return c.Set(ctx, req) }},
		{method: "add", params: req, invoke: func(c *Client) error { return c.Add(ctx, req) }},
		{
			method: "update_start", params: map[string]any{"name": "demo"},
			invoke: func(c *Client) error { return c.UpdateStart(ctx, "demo") },
		},
		{
			method: "update_complete", params: map[string]any{"name": "demo"},
			invoke: func(c *Client) error { return c.UpdateComplete(ctx, "demo") },
		},
		{method: "event", params: event, invoke: func(c *Client) error { return c.Event(ctx, event) }},
	}
}

func serviceDataMethodCases(ctx context.Context) []methodCase {
	validation := ValidateRequest{Package: "demo", Type: "config", Service: "main"}

	return []methodCase{
		{
			method: "validate", params: validation,
			invoke: func(c *Client) error {
				_, err := c.Validate(ctx, validation)

				return err
			},
		},
		{
			method: "get_data", params: map[string]any{"name": "demo", "instance": "main", "type": "status"},
			invoke: func(c *Client) error {
				_, err := c.Data(ctx, "demo", "main", "status")

				return err
			},
		},
		{
			method: "set_data",
			params: map[string]any{"name": "demo", "instance": "main", "data": map[string]any{"x": true}},
			invoke: func(c *Client) error { return c.SetData(ctx, "demo", "main", map[string]any{"x": true}) },
		},
		{
			method: "state", params: map[string]any{"name": "demo", "spawn": true},
			invoke: func(c *Client) error {
				_, err := c.State(ctx, "demo", true)

				return err
			},
		},
		{
			method: "watchdog",
			params: map[string]any{"name": "demo", "instance": "main", "mode": 1, "timeout": 30},
			invoke: func(c *Client) error { return c.Watchdog(ctx, "demo", "main", 1, 30) },
		},
	}
}

func runMethodCases(t *testing.T, methods []methodCase) {
	t.Helper()
	for _, tc := range methods {
		t.Run(tc.method, func(t *testing.T) {
			assertCall(t, tc.method, tc.params, json.RawMessage(`{}`), tc.invoke)
		})
	}
}
