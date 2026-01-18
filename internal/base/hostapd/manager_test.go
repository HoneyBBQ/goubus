// Copyright (c) 2026 honeybbq
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package hostapd_test

import (
	"context"
	"testing"

	"github.com/honeybbq/goubus/v2/internal/base/hostapd"
	"github.com/honeybbq/goubus/v2/internal/testutil"
)

func TestHostapdManager(t *testing.T) {
	ctx := context.Background()
	mock := testutil.NewMockTransport()
	mgr := hostapd.New(mock)

	t.Run("General", func(t *testing.T) {
		testHostapdGeneral(t, ctx, mock, mgr)
	})

	t.Run("AP", func(t *testing.T) {
		testHostapdAP(t, ctx, mock, mgr)
	})
}

func testHostapdGeneral(t *testing.T, ctx context.Context, mock *testutil.MockTransport, mgr *hostapd.Manager) {
	t.Helper()
	t.Run("Reload", func(t *testing.T) {
		mock.AddResponse("hostapd", "reload", map[string]any{"result": 0})

		err := mgr.Reload(ctx, "phy0", 0)
		if err != nil {
			t.Fatalf("Reload failed: %v", err)
		}

		call := mock.GetLastCall()
		if call.Service != "hostapd" || call.Method != "reload" {
			t.Errorf("unexpected call: %s.%s", call.Service, call.Method)
		}

		params, ok := call.Data.(map[string]any)
		if !ok {
			t.Fatalf("call.Data is not map[string]any")
		}

		if params["phy"] != "phy0" || params["radio"] != 0 {
			t.Errorf("unexpected params: %v", params)
		}
	})

	t.Run("BSSInfo", func(t *testing.T) {
		expectedInfo := map[string]any{
			"iface": "wlan0",
			"ssid":  "OpenWrt",
		}
		mock.AddResponse("hostapd", "bss_info", expectedInfo)

		info, err := mgr.BSSInfo(ctx, "wlan0")
		if err != nil {
			t.Fatalf("BSSInfo failed: %v", err)
		}

		if info["ssid"] != "OpenWrt" {
			t.Errorf("unexpected bss info: %v", info)
		}
	})
}

func testHostapdAP(t *testing.T, ctx context.Context, mock *testutil.MockTransport, mgr *hostapd.Manager) {
	t.Helper()
	testHostapdGetClients(t, ctx, mock, mgr)
	testHostapdGetStatus(t, ctx, mock, mgr)
	testHostapdDelClient(t, ctx, mock, mgr)
	testHostapdSwitchChan(t, ctx, mock, mgr)
}

func testHostapdGetClients(t *testing.T, ctx context.Context, mock *testutil.MockTransport, mgr *hostapd.Manager) {
	t.Helper()
	t.Run("GetClients", func(t *testing.T) {
		expectedClients := map[string]any{
			"clients": []any{
				map[string]any{"mac": "00:11:22:33:44:55"},
			},
		}
		mock.AddResponse("hostapd.wlan0", "get_clients", expectedClients)

		clients, err := mgr.AP("hostapd.wlan0").GetClients(ctx)
		if err != nil {
			t.Fatalf("GetClients failed: %v", err)
		}

		clientsList, ok := clients["clients"].([]any)
		if !ok {
			t.Fatalf("clients['clients'] is not []any")
		}

		if len(clientsList) != 1 {
			t.Errorf("unexpected clients: %v", clients)
		}
	})
}

func testHostapdGetStatus(t *testing.T, ctx context.Context, mock *testutil.MockTransport, mgr *hostapd.Manager) {
	t.Helper()
	t.Run("GetStatus", func(t *testing.T) {
		expectedStatus := map[string]any{
			"status": "up",
		}
		mock.AddResponse("hostapd.wlan0", "get_status", expectedStatus)

		status, err := mgr.AP("hostapd.wlan0").GetStatus(ctx)
		if err != nil {
			t.Fatalf("GetStatus failed: %v", err)
		}

		if status["status"] != "up" {
			t.Errorf("unexpected status: %v", status)
		}
	})
}

func testHostapdDelClient(t *testing.T, ctx context.Context, mock *testutil.MockTransport, mgr *hostapd.Manager) {
	t.Helper()
	t.Run("DelClient", func(t *testing.T) {
		mock.AddResponse("hostapd.wlan0", "del_client", map[string]any{"result": 0})

		err := mgr.AP("hostapd.wlan0").DelClient(ctx, "00:11:22:33:44:55", 1, true, 0)
		if err != nil {
			t.Fatalf("DelClient failed: %v", err)
		}

		call := mock.GetLastCall()

		params, ok := call.Data.(map[string]any)
		if !ok {
			t.Fatalf("call.Data is not map[string]any")
		}

		if params["addr"] != "00:11:22:33:44:55" || params["reason"] != 1 || params["deauth"] != true {
			t.Errorf("unexpected params: %v", params)
		}
	})
}

func testHostapdSwitchChan(t *testing.T, ctx context.Context, mock *testutil.MockTransport, mgr *hostapd.Manager) {
	t.Helper()
	t.Run("SwitchChan", func(t *testing.T) {
		mock.AddResponse("hostapd.wlan0", "switch_chan", map[string]any{"result": 0})

		err := mgr.AP("hostapd.wlan0").SwitchChan(ctx, 5180, 80)
		if err != nil {
			t.Fatalf("SwitchChan failed: %v", err)
		}

		call := mock.GetLastCall()

		params, ok := call.Data.(map[string]any)
		if !ok {
			t.Fatalf("call.Data is not map[string]any")
		}

		if params["freq"] != 5180 || params["bandwidth"] != 80 {
			t.Errorf("unexpected params: %v", params)
		}
	})
}
