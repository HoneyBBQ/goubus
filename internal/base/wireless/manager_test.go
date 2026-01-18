// Copyright (c) 2026 honeybbq
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package wireless_test

import (
	"context"
	"testing"

	"github.com/honeybbq/goubus/v2/internal/base/wireless"
	"github.com/honeybbq/goubus/v2/internal/testutil"
)

func TestWirelessManager(t *testing.T) {
	ctx := context.Background()
	mock := testutil.NewMockTransport()
	mgr := wireless.New(mock)

	t.Run("Devices", func(t *testing.T) {
		testWirelessDevices(t, ctx, mock, mgr)
	})

	t.Run("Info", func(t *testing.T) {
		testWirelessInfo(t, ctx, mock, mgr)
	})

	t.Run("Scan", func(t *testing.T) {
		testWirelessScan(t, ctx, mock, mgr)
	})

	t.Run("AssocList_Generic", func(t *testing.T) {
		testWirelessAssocList(t, ctx, mock, mgr)
	})

	t.Run("Lists", func(t *testing.T) {
		testWirelessLists(t, ctx, mock, mgr)
	})

	t.Run("PhyName", func(t *testing.T) {
		testWirelessPhyName(t, ctx, mock, mgr)
	})
}

func testWirelessDevices(t *testing.T, ctx context.Context, mock *testutil.MockTransport, mgr *wireless.Manager) {
	t.Helper()
	mock.AddResponse("iwinfo", "devices", map[string]any{
		"devices": []string{"radio0", "radio1"},
	})

	devices, err := mgr.Devices(ctx)
	if err != nil {
		t.Fatalf("Devices failed: %v", err)
	}

	if len(devices) != 2 || devices[0] != "radio0" {
		t.Errorf("unexpected devices: %v", devices)
	}
}

func testWirelessInfo(t *testing.T, ctx context.Context, mock *testutil.MockTransport, mgr *wireless.Manager) {
	t.Helper()
	mock.AddResponse("iwinfo", "info", map[string]any{
		"phy":     "phy0",
		"ssid":    "OpenWrt",
		"channel": 36,
	})

	info, err := mgr.Info(ctx, "wlan0")
	if err != nil {
		t.Fatalf("Info failed: %v", err)
	}

	if info.SSID != "OpenWrt" {
		t.Errorf("unexpected info: %+v", info)
	}
}

func testWirelessScan(t *testing.T, ctx context.Context, mock *testutil.MockTransport, mgr *wireless.Manager) {
	t.Helper()
	mock.AddResponse("iwinfo", "scan", map[string]any{
		"results": []map[string]any{
			{"ssid": "OtherSSID", "bssid": "00:11:22:33:44:55"},
		},
	})

	results, err := mgr.Scan(ctx, "wlan0")
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	if len(results) != 1 || results[0].SSID != "OtherSSID" {
		t.Errorf("unexpected scan results: %v", results)
	}
}

func testWirelessAssocList(t *testing.T, ctx context.Context, mock *testutil.MockTransport, mgr *wireless.Manager) {
	t.Helper()
	mock.AddResponse("iwinfo", "assoclist", map[string]any{
		"results": []map[string]any{
			{
				"mac":    "00:11:22:33:44:55",
				"signal": -50,
				"rx":     map[string]any{"rate": 100000},
			},
		},
	})

	clients, err := mgr.AssocList(ctx, "wlan0")
	if err != nil {
		t.Fatalf("AssocList failed: %v", err)
	}

	if len(clients) != 1 || clients[0].Mac != "00:11:22:33:44:55" {
		t.Errorf("unexpected client data: %+v", clients[0])
	}
}

func testWirelessLists(t *testing.T, ctx context.Context, mock *testutil.MockTransport, mgr *wireless.Manager) {
	t.Helper()
	mock.AddResponse("iwinfo", "freqlist", map[string]any{"results": []any{2412, 2417}})
	mock.AddResponse("iwinfo", "txpowerlist", map[string]any{"results": []any{20, 23}})
	mock.AddResponse("iwinfo", "countrylist", map[string]any{"results": []any{"US", "CN"}})
	mock.AddResponse("iwinfo", "survey", map[string]any{"results": []any{map[string]any{"mhz": 2412}}})

	res, err := mgr.FreqList(ctx, "wlan0")
	if err != nil || len(res) != 2 {
		t.Errorf("FreqList failed: %v", err)
	}

	res, err = mgr.TxPowerList(ctx, "wlan0")
	if err != nil || len(res) != 2 {
		t.Errorf("TxPowerList failed: %v", err)
	}

	res, err = mgr.CountryList(ctx, "wlan0")
	if err != nil || len(res) != 2 {
		t.Errorf("CountryList failed: %v", err)
	}

	res, err = mgr.Survey(ctx, "wlan0")
	if err != nil || len(res) != 1 {
		t.Errorf("Survey failed: %v", err)
	}
}

func testWirelessPhyName(t *testing.T, ctx context.Context, mock *testutil.MockTransport, mgr *wireless.Manager) {
	t.Helper()
	mock.AddResponse("iwinfo", "phyname", map[string]any{"phyname": "phy0"})

	phy, err := mgr.PhyName(ctx, "radio0")
	if err != nil {
		t.Fatalf("PhyName failed: %v", err)
	}

	if phy != "phy0" {
		t.Errorf("expected phy0, got %s", phy)
	}
}
