package wpasupplicant

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

var errUnexpectedWPAMethod = errors.New("unknown wpa_supplicant test method")

func fixture(t *testing.T) json.RawMessage {
	t.Helper()
	path := filepath.Join(
		"..", "..", "internal", "testdata", "rax3000m", "wpa_supplicant_iface_status.json",
	)
	// #nosec G304 -- The fixture path is fixed by this test package.
	data, err := os.ReadFile(path)
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
		if got := recover(); got != "goubus/wpasupplicant: nil caller" {
			t.Fatalf("panic = %v", got)
		}
	}()
	New(nil)
}

func TestIfaceStatusSample(t *testing.T) {
	assertCall(t, "wpa_supplicant", "iface_status", map[string]any{"name": "wlan0"}, fixture(t), func(c *Client) error {
		got, err := c.IfaceStatus(context.Background(), "wlan0")
		if err == nil && len(got) == 0 {
			t.Error("interface status fixture decoded without fields")
		}

		return err
	})
}

func TestMethods(t *testing.T) {
	ctx := context.Background()
	object := "wpa_supplicant.wlan0"
	methods := []struct {
		params  any
		service string
		method  string
	}{
		{service: "wpa_supplicant", method: "iface_status", params: map[string]any{"name": "wlan0"}},
		{service: object, method: "reload"},
		{service: object, method: "wps_start", params: map[string]any{"multi_ap": true}},
		{service: object, method: "wps_cancel"},
		{service: object, method: "control", params: map[string]any{"command": "PING"}},
	}
	for _, tc := range methods {
		t.Run(tc.method, func(t *testing.T) {
			raw := json.RawMessage(`{}`)
			if tc.method == "control" {
				raw = json.RawMessage(`{"result":"PONG"}`)
			}
			assertCall(t, tc.service, tc.method, tc.params, raw, func(c *Client) error {
				return invokeMethod(ctx, c, object, tc.method)
			})
		})
	}
}

func invokeMethod(ctx context.Context, c *Client, object, method string) error {
	switch method {
	case "iface_status":
		_, err := c.IfaceStatus(ctx, "wlan0")

		return err
	case "reload":
		return c.STA(object).Reload(ctx)
	case "wps_start":
		return c.STA(object).WPSStart(ctx, true)
	case "wps_cancel":
		return c.STA(object).WPSCancel(ctx)
	case "control":
		_, err := c.STA(object).Control(ctx, "PING")

		return err
	default:
		return errUnexpectedWPAMethod
	}
}
