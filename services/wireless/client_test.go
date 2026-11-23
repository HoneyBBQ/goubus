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

func TestExpandedWirelessFixtureFields(t *testing.T) {
	client := expandedWirelessFixtureClient(t)
	assertExpandedInfo(t, client)
	assertExpandedScan(t, client)
	assertExpandedAssociation(t, client)
}

func expandedWirelessFixtureClient(t *testing.T) *Client {
	t.Helper()
	caller := goubus.CallerFunc(func(_ context.Context, _, method string, _ any) (goubus.Result, error) {
		name := ""
		switch method {
		case "info":
			name = "iwinfo_info_phy0.json"
		case "scan":
			name = "iwinfo_scan_phy0.json"
		case "assoclist":
			name = "iwinfo_assoclist_phy0.json"
		default:
			t.Fatalf("unexpected iwinfo method %q", method)
		}

		return goubus.NewJSONResult(fixture(t, "rax3000m", name)), nil
	})

	return New(caller)
}

func assertExpandedInfo(t *testing.T, client *Client) {
	t.Helper()
	info, err := client.Info(t.Context(), "phy0")
	if err != nil || info.Frequency != 2412 || info.Encryption == nil ||
		!bool(info.Encryption.Enabled) || len(info.HWModes) == 0 {
		t.Fatalf("Info expanded fields = %+v, %v", info, err)
	}
}

func assertExpandedScan(t *testing.T, client *Client) {
	t.Helper()
	scan, err := client.Scan(t.Context(), "phy0")
	if err != nil || len(scan) == 0 || scan[0].Band == 0 || scan[0].MHz != 2412 || scan[0].Encryption == nil {
		t.Fatalf("Scan expanded fields = %+v, %v", scan, err)
	}
}

func assertExpandedAssociation(t *testing.T, client *Client) {
	t.Helper()
	stations, err := client.AssocList(t.Context(), "phy0")
	if err != nil || len(stations) == 0 || stations[0].Rx.Packets != 510 || stations[0].Rx.Bytes != 33375 ||
		stations[0].Tx.Packets != 102 || stations[0].Tx.Bytes != 10723 || stations[0].Throughput == 0 {
		t.Fatalf("Assoc expanded fields = %+v, %v", stations, err)
	}
}

func TestAssocDecodesBlobmsgBooleanNumbers(t *testing.T) {
	client := New(goubus.CallerFunc(func(context.Context, string, string, any) (goubus.Result, error) {
		return goubus.NewJSONResult([]byte(`{"results":[{"authorized":1,"authenticated":0}]}`)), nil
	}))
	stations, err := client.AssocList(context.Background(), "phy0")
	if err != nil {
		t.Fatal(err)
	}
	if len(stations) != 1 || !bool(stations[0].Authorized) || bool(stations[0].Authenticated) {
		t.Fatalf("association booleans = %+v", stations)
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
