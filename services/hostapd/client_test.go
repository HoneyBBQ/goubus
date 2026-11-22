package hostapd

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/honeybbq/goubus/v2"
)

func fixture(t *testing.T, name string) json.RawMessage {
	t.Helper()
	// Sample names are fixed by this test package.
	// #nosec G304
	data, err := os.ReadFile(filepath.Join("..", "..", "internal", "testdata", "rax3000m", name))
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
		if got := recover(); got != "goubus/hostapd: nil caller" {
			t.Fatalf("panic = %v", got)
		}
	}()
	New(nil)
}

func TestSamples(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		file   string
		method string
	}{
		{file: "hostapd_get_clients.json", method: "get_clients"},
		{file: "hostapd_get_status.json", method: "get_status"},
	} {
		t.Run(tc.method, func(t *testing.T) {
			assertCall(t, "hostapd.wlan0", tc.method, nil, fixture(t, tc.file), func(c *Client) error {
				if tc.method == "get_clients" {
					_, err := c.AP("hostapd.wlan0").Clients(ctx)

					return err
				}
				_, err := c.AP("hostapd.wlan0").Status(ctx)

				return err
			})
		})
	}
}

func TestMethods(t *testing.T) {
	ctx := context.Background()
	methods := []struct {
		params  any
		invoke  func(*Client) error
		service string
		method  string
	}{
		{
			service: "hostapd", method: "reload", params: map[string]any{"phy": "phy0", "radio": 1},
			invoke: func(c *Client) error { return c.Reload(ctx, "phy0", 1) },
		},
		{
			service: "hostapd", method: "bss_info", params: map[string]any{"iface": "wlan0"},
			invoke: func(c *Client) error {
				_, err := c.BSSInfo(ctx, "wlan0")

				return err
			},
		},
		{
			service: "hostapd.wlan0", method: "get_clients",
			invoke: func(c *Client) error {
				_, err := c.AP("hostapd.wlan0").Clients(ctx)

				return err
			},
		},
		{
			service: "hostapd.wlan0", method: "get_status",
			invoke: func(c *Client) error {
				_, err := c.AP("hostapd.wlan0").Status(ctx)

				return err
			},
		},
		{
			service: "hostapd.wlan0", method: "del_client",
			params: map[string]any{
				"addr": "00:11:22:33:44:55", "reason": 3, "deauth": true, "ban_time": 30,
			},
			invoke: func(c *Client) error {
				return c.AP("hostapd.wlan0").DelClient(ctx, "00:11:22:33:44:55", 3, true, 30)
			},
		},
		{
			service: "hostapd.wlan0", method: "switch_chan",
			params: map[string]any{"freq": 5180, "bandwidth": 80},
			invoke: func(c *Client) error { return c.AP("hostapd.wlan0").SwitchChan(ctx, 5180, 80) },
		},
	}
	for _, tc := range methods {
		t.Run(tc.method, func(t *testing.T) {
			assertCall(t, tc.service, tc.method, tc.params, json.RawMessage(`{}`), tc.invoke)
		})
	}
}
