package rc

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
	// Profile names are fixed by this test table.
	// #nosec G304
	data, err := os.ReadFile(filepath.Join("..", "..", "internal", "testdata", profile, "rc_list.json"))
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
		if got := recover(); got != "goubus/rc: nil caller" {
			t.Fatalf("panic = %v", got)
		}
	}()
	New(nil)
}

func TestSamples(t *testing.T) {
	for _, profile := range []string{"rax3000m", "x86_generic"} {
		t.Run(profile, func(t *testing.T) {
			raw := fixture(t, profile)
			assertCall(t, "rc", "list", map[string]any{}, raw, func(c *Client) error {
				got, err := c.List(context.Background(), "", false)
				if err == nil {
					info, ok := got["boot"]
					if !ok || info.Start != 10 || info.Stop != 90 || !bool(info.Enabled) {
						t.Errorf("boot status = %+v, want fixture values", info)
					}
				}

				return err
			})
		})
	}
}

func TestMethods(t *testing.T) {
	ctx := context.Background()
	params := map[string]any{"name": "network", "skip_running_check": true}
	assertCall(t, "rc", "list", params, json.RawMessage(`{}`), func(c *Client) error {
		_, err := c.List(ctx, "network", true)

		return err
	})
	request := InitRequest{Name: "network", Action: "restart"}
	assertCall(t, "rc", "init", request, json.RawMessage(`{}`), func(c *Client) error {
		return c.Init(ctx, "network", "restart")
	})
}
