package wireless

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
	// Sample names are fixed by the test table.
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
		if service != "iwinfo" || gotMethod != method || !reflect.DeepEqual(gotParams, params) {
			t.Errorf("Call(%q, %q, %#v), want (%q, %q, %#v)", service, gotMethod, gotParams, "iwinfo", method, params)
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
		if got := recover(); got != "goubus/wireless: nil caller" {
			t.Fatalf("panic = %v", got)
		}
	}()
	New(nil)
}

func TestSamples(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		profile string
		file    string
		method  string
	}{
		{"rax3000m", "iwinfo_devices.json", "devices"},
		{"x86_generic", "iwinfo_devices.json", "devices"},
		{"rax3000m", "iwinfo_assoclist.json", "assoclist"},
		{"rax3000m", "iwinfo_assoclist_phy0.json", "assoclist"},
		{"rax3000m", "iwinfo_freqlist_phy0.json", "freqlist"},
		{"rax3000m", "iwinfo_info_phy0.json", "info"},
		{"rax3000m", "iwinfo_scan_phy0.json", "scan"},
		{"rax3000m", "iwinfo_survey_phy0.json", "survey"},
	}
	for _, tc := range cases {
		t.Run(tc.profile+"/"+tc.file, func(t *testing.T) {
			params := any(map[string]any{"device": "phy0"})
			if tc.method == "devices" {
				params = nil
			}
			assertCall(t, tc.method, params, fixture(t, tc.profile, tc.file), func(c *Client) error {
				switch tc.method {
				case "devices":
					_, err := c.Devices(ctx)

					return err
				case "assoclist":
					_, err := c.AssocList(ctx, "phy0")

					return err
				case "freqlist":
					_, err := c.FreqList(ctx, "phy0")

					return err
				case "info":
					_, err := c.Info(ctx, "phy0")

					return err
				case "scan":
					_, err := c.Scan(ctx, "phy0")

					return err
				default:
					_, err := c.Survey(ctx, "phy0")

					return err
				}
			})
		})
	}
}

func TestMethods(t *testing.T) {
	ctx := context.Background()
	methods := []struct {
		params any
		method string
	}{
		{method: "assoclist", params: map[string]any{"device": "wlan0"}},
		{method: "countrylist", params: map[string]any{"device": "wlan0"}},
		{method: "devices"},
		{method: "freqlist", params: map[string]any{"device": "wlan0"}},
		{method: "info", params: map[string]any{"device": "wlan0"}},
		{method: "phyname", params: map[string]any{"section": "radio0"}},
		{method: "scan", params: map[string]any{"device": "wlan0"}},
		{method: "survey", params: map[string]any{"device": "wlan0"}},
		{method: "txpowerlist", params: map[string]any{"device": "wlan0"}},
	}
	for _, tc := range methods {
		t.Run(tc.method, func(t *testing.T) {
			raw := json.RawMessage(`{"results":[]}`)
			if tc.method == "devices" {
				raw = json.RawMessage(`{"devices":[]}`)
			}
			if tc.method == "info" {
				raw = json.RawMessage(`{}`)
			}
			if tc.method == "phyname" {
				raw = json.RawMessage(`{"phyname":"phy0"}`)
			}
			assertCall(t, tc.method, tc.params, raw, func(c *Client) error {
				return invokeMethod(ctx, c, tc.method)
			})
		})
	}
}

func invokeMethod(ctx context.Context, c *Client, method string) error {
	switch method {
	case "assoclist":
		_, err := c.AssocList(ctx, "wlan0")

		return err
	case "countrylist":
		_, err := c.CountryList(ctx, "wlan0")

		return err
	case "devices":
		_, err := c.Devices(ctx)

		return err
	case "freqlist":
		_, err := c.FreqList(ctx, "wlan0")

		return err
	case "info":
		_, err := c.Info(ctx, "wlan0")

		return err
	case "phyname":
		_, err := c.PhyName(ctx, "radio0")

		return err
	case "scan":
		_, err := c.Scan(ctx, "wlan0")

		return err
	case "survey":
		_, err := c.Survey(ctx, "wlan0")

		return err
	case "txpowerlist":
		_, err := c.TxPowerList(ctx, "wlan0")

		return err
	default:
		return nil
	}
}
