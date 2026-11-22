package session

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
	// #nosec G304
	data, err := os.ReadFile(filepath.Join("..", "..", "internal", "testdata", profile, "session_list.json"))
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
		if service != "session" || gotMethod != method || !reflect.DeepEqual(gotParams, params) {
			t.Errorf("Call(%q, %q, %#v), want (%q, %q, %#v)", service, gotMethod, gotParams, "session", method, params)
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
		if got := recover(); got != "goubus/session: nil caller" {
			t.Fatalf("panic = %v", got)
		}
	}()
	New(nil)
}

func TestSamples(t *testing.T) {
	ctx := context.Background()
	for _, profile := range []string{"rax3000m", "x86_generic"} {
		t.Run(profile, func(t *testing.T) {
			assertCall(t, "list", nil, fixture(t, profile), func(c *Client) error {
				got, err := c.List(ctx)
				if err == nil && len(got) == 0 {
					t.Error("session fixture decoded without entries")
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
	methods := append(sessionCoreMethodCases(ctx), sessionStateMethodCases(ctx)...)
	runMethodCases(t, methods)
}

func sessionCoreMethodCases(ctx context.Context) []methodCase {
	grant := GrantRequest{Session: "sid", Scope: "ubus", Objects: []string{"system"}}
	access := AccessRequest{Session: "sid", Scope: "ubus", Object: "system", Function: "info"}

	return []methodCase{
		{
			method: "create", params: map[string]any{"timeout": 300},
			invoke: func(c *Client) error {
				_, err := c.Create(ctx, 300)

				return err
			},
		},
		{method: "list", invoke: func(c *Client) error {
			_, err := c.List(ctx)

			return err
		}},
		{method: "grant", params: grant, invoke: func(c *Client) error { return c.Grant(ctx, grant) }},
		{method: "revoke", params: grant, invoke: func(c *Client) error { return c.Revoke(ctx, grant) }},
		{method: "access", params: access, invoke: func(c *Client) error {
			_, err := c.Access(ctx, access)

			return err
		}},
	}
}

func sessionStateMethodCases(ctx context.Context) []methodCase {
	sessionKey := "ubus_rpc_session"
	keys := []string{"x"}
	values := map[string]any{"x": 1}
	login := LoginRequest{Username: "root", Password: "secret", Timeout: 300}

	return []methodCase{
		{
			method: "set", params: map[string]any{sessionKey: "sid", "values": values},
			invoke: func(c *Client) error { return c.Set(ctx, "sid", values) },
		},
		{
			method: "get", params: map[string]any{sessionKey: "sid", "keys": keys},
			invoke: func(c *Client) error {
				_, err := c.Get(ctx, "sid", keys)

				return err
			},
		},
		{
			method: "unset", params: map[string]any{sessionKey: "sid", "keys": keys},
			invoke: func(c *Client) error { return c.Unset(ctx, "sid", keys) },
		},
		{
			method: "destroy", params: map[string]any{sessionKey: "sid"},
			invoke: func(c *Client) error { return c.Destroy(ctx, "sid") },
		},
		{method: "login", params: login, invoke: func(c *Client) error {
			_, err := c.Login(ctx, login)

			return err
		}},
	}
}

func runMethodCases(t *testing.T, methods []methodCase) {
	t.Helper()
	for _, tc := range methods {
		t.Run(tc.method, func(t *testing.T) {
			raw := json.RawMessage(`{}`)
			if tc.method == "access" {
				raw = json.RawMessage(`{"access":true}`)
			}
			if tc.method == "create" || tc.method == "login" {
				raw = json.RawMessage(`{"ubus_rpc_session":"sid","timeout":300}`)
			}
			assertCall(t, tc.method, tc.params, raw, tc.invoke)
		})
	}
}
