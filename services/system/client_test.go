package system

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
	// #nosec G304
	data, err := os.ReadFile(filepath.Join("..", "..", "internal", "testdata", profile, name))
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
		if service != "system" || gotMethod != method || !reflect.DeepEqual(gotParams, params) {
			t.Errorf("Call(%q, %q, %#v), want (%q, %q, %#v)", service, gotMethod, gotParams, "system", method, params)
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
		if got := recover(); got != "goubus/system: nil caller" {
			t.Fatalf("panic = %v", got)
		}
	}()
	New(nil)
}

func TestSamples(t *testing.T) {
	ctx := context.Background()
	for _, profile := range []string{"rax3000m", "x86_generic"} {
		for _, tc := range []struct {
			file   string
			method string
		}{
			{"system_info.json", "info"},
			{"system_board.json", "board"},
		} {
			t.Run(profile+"/"+tc.file, func(t *testing.T) {
				assertCall(t, tc.method, nil, fixture(t, profile, tc.file), func(c *Client) error {
					return invokeSample(ctx, t, c, profile, tc.method)
				})
			})
		}
	}
}

func invokeSample(ctx context.Context, t *testing.T, c *Client, profile, method string) error {
	t.Helper()
	if method == "info" {
		got, err := c.Info(ctx)
		if err == nil {
			totalBytes := got.Memory.Total
			uptimeSeconds := got.Uptime
			if totalBytes == 0 || uptimeSeconds == 0 || got.Memory.Shared == 0 ||
				got.Memory.Buffered == 0 || got.Root.Avail == 0 {
				t.Errorf("expanded system counters missing: %+v", got)
			}
		}

		return err
	}

	got, err := c.Board(ctx)
	missingRelease := got.RootfsType == "" || got.Release.BuildDate == ""
	if profile == "rax3000m" {
		missingRelease = missingRelease || got.Release.FirmwareURL == ""
	}
	if err == nil && missingRelease {
		t.Errorf("board release fields missing: %+v", got)
	}

	return err
}

func TestMethods(t *testing.T) {
	ctx := context.Background()
	watchdog := WatchdogRequest{Timeout: 30}
	upgrade := SysupgradeRequest{Path: "/tmp/fw.bin"}
	validation := validateFirmwareImageRequest{Path: "/tmp/fw.bin"}
	for _, tc := range []struct {
		params any
		invoke func(*Client) error
		method string
	}{
		{method: "reboot", invoke: func(c *Client) error { return c.Reboot(ctx) }},
		{method: "watchdog", params: watchdog, invoke: func(c *Client) error { return c.Watchdog(ctx, watchdog) }},
		{
			method: "signal", params: signalRequest{Pid: 123, Signum: 15},
			invoke: func(c *Client) error { return c.Signal(ctx, 123, 15) },
		},
		{
			method: "validate_firmware_image", params: validation,
			invoke: func(c *Client) error {
				_, err := c.ValidateFirmwareImage(ctx, validation.Path)

				return err
			},
		},
		{method: "sysupgrade", params: upgrade, invoke: func(c *Client) error { return c.Sysupgrade(ctx, upgrade) }},
	} {
		t.Run(tc.method, func(t *testing.T) {
			raw := json.RawMessage(`{}`)
			if tc.method == "validate_firmware_image" {
				raw = json.RawMessage(`{"valid":true}`)
			}
			assertCall(t, tc.method, tc.params, raw, tc.invoke)
		})
	}
}
