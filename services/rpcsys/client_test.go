package rpcsys

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/honeybbq/goubus/v2"
)

func fixture(t *testing.T) json.RawMessage {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "internal", "testdata", "rax3000m", "rpc_sys_packagelist.json"))
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
		if service != "rpc-sys" || gotMethod != method || !reflect.DeepEqual(gotParams, params) {
			t.Errorf("Call(%q, %q, %#v), want (%q, %q, %#v)", service, gotMethod, gotParams, "rpc-sys", method, params)
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
		if got := recover(); got != "goubus/rpcsys: nil caller" {
			t.Fatalf("panic = %v", got)
		}
	}()
	New(nil)
}

func TestPackageListSample(t *testing.T) {
	raw := fixture(t)
	assertCall(t, "packagelist", map[string]any{"all": true}, raw, func(c *Client) error {
		got, err := c.PackageList(context.Background(), true)
		packages, ok := got["packages"].(map[string]any)
		if err == nil && (!ok || len(packages) == 0) {
			t.Error("package fixture decoded without package entries")
		}

		return err
	})
}

func TestMethods(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		params any
		invoke func(*Client) error
		method string
	}{
		{
			method: "packagelist", params: map[string]any{"all": false},
			invoke: func(c *Client) error {
				_, err := c.PackageList(ctx, false)

				return err
			},
		},
		{
			method: "password_set", params: map[string]any{"user": "root", "password": "secret"},
			invoke: func(c *Client) error { return c.PasswordSet(ctx, "root", "secret") },
		},
		{method: "factory", invoke: func(c *Client) error { return c.Factory(ctx) }},
		{
			method: "upgrade_start", params: map[string]any{"keep": true},
			invoke: func(c *Client) error { return c.UpgradeStart(ctx, true) },
		},
		{method: "upgrade_test", invoke: func(c *Client) error { return c.UpgradeTest(ctx) }},
		{method: "upgrade_clean", invoke: func(c *Client) error { return c.UpgradeClean(ctx) }},
		{method: "reboot", invoke: func(c *Client) error { return c.Reboot(ctx) }},
	} {
		t.Run(tc.method, func(t *testing.T) {
			raw := json.RawMessage(`{}`)
			if tc.method == "packagelist" {
				raw = json.RawMessage(`{"packages":{}}`)
			}
			assertCall(t, tc.method, tc.params, raw, tc.invoke)
		})
	}
}
