package log

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/honeybbq/goubus/v2"
)

func assertCall(t *testing.T, method string, params any, raw json.RawMessage, invoke func(*Client) error) {
	t.Helper()
	called := false
	caller := goubus.CallerFunc(func(_ context.Context, service, gotMethod string, gotParams any) (goubus.Result, error) {
		called = true
		if service != "log" || gotMethod != method || !reflect.DeepEqual(gotParams, params) {
			t.Errorf("Call(%q, %q, %#v), want (%q, %q, %#v)", service, gotMethod, gotParams, "log", method, params)
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
		if got := recover(); got != "goubus/log: nil caller" {
			t.Fatalf("panic = %v", got)
		}
	}()
	New(nil)
}

func TestSamples(t *testing.T) {
	for _, profile := range []string{"rax3000m"} {
		t.Run(profile, func(t *testing.T) {
			// Profile values are fixed by this test table.
			// #nosec G304
			raw, err := os.ReadFile(filepath.Join("..", "..", "internal", "testdata", profile, "log_read.json"))
			if err != nil {
				t.Fatal(err)
			}
			if len(raw) == 0 {
				t.Skip("empty log sample")
			}
			params := map[string]any{"lines": 20, "stream": false, "oneshot": true}
			assertCall(t, "read", params, raw, func(c *Client) error {
				got, err := c.Read(context.Background(), 20, false, true)
				if err == nil && len(got.Log) == 0 {
					t.Error("non-empty sample decoded without log entries")
				}

				return err
			})
		})
	}
}

func TestMethods(t *testing.T) {
	ctx := context.Background()
	params := map[string]any{"lines": 7, "stream": true, "oneshot": false}
	assertCall(t, "read", params, json.RawMessage(`{"log":[]}`), func(c *Client) error {
		_, err := c.Read(ctx, 7, true, false)

		return err
	})
	assertCall(t, "write", map[string]any{"event": "event"}, json.RawMessage(`{}`), func(c *Client) error {
		return c.Write(ctx, "event")
	})
}
