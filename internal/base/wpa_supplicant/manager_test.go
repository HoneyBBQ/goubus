// Copyright (c) 2026 honeybbq
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package wpa_supplicant_test

import (
	"context"
	"testing"

	"github.com/honeybbq/goubus/v2/internal/base/wpa_supplicant"
	"github.com/honeybbq/goubus/v2/internal/testutil"
)

func TestWpaSupplicantManager(t *testing.T) {
	ctx := context.Background()
	mock := testutil.NewMockTransport()
	mgr := wpa_supplicant.New(mock)

	t.Run("IfaceStatus", func(t *testing.T) {
		expectedStatus := map[string]any{
			"wpa_state":  "COMPLETED",
			"ip_address": "192.168.1.100",
		}
		mock.AddResponse("wpa_supplicant", "iface_status", expectedStatus)

		status, err := mgr.IfaceStatus(ctx, "wlan0")
		if err != nil {
			t.Fatalf("IfaceStatus failed: %v", err)
		}

		if status["wpa_state"] != "COMPLETED" {
			t.Errorf("unexpected status: %v", status)
		}
	})

	t.Run("STA", func(t *testing.T) {
		testWpaSTA(t, ctx, mock, mgr)
	})
}

func testWpaSTA(t *testing.T, ctx context.Context, mock *testutil.MockTransport, mgr *wpa_supplicant.Manager) {
	t.Helper()
	testWpaSTAReload(t, ctx, mock, mgr)
	testWpaSTAWPSStart(t, ctx, mock, mgr)
	testWpaSTAWPSCancel(t, ctx, mock, mgr)
	testWpaSTAControl(t, ctx, mock, mgr)
}

func testWpaSTAReload(t *testing.T, ctx context.Context, mock *testutil.MockTransport, mgr *wpa_supplicant.Manager) {
	t.Helper()
	t.Run("Reload", func(t *testing.T) {
		mock.AddResponse("wpa_supplicant.wlan0", "reload", map[string]any{"result": 0})

		err := mgr.STA("wpa_supplicant.wlan0").Reload(ctx)
		if err != nil {
			t.Fatalf("Reload failed: %v", err)
		}

		call := mock.GetLastCall()
		if call.Service != "wpa_supplicant.wlan0" || call.Method != "reload" {
			t.Errorf("unexpected call: %s.%s", call.Service, call.Method)
		}
	})
}

func testWpaSTAWPSStart(t *testing.T, ctx context.Context, mock *testutil.MockTransport, mgr *wpa_supplicant.Manager) {
	t.Helper()
	t.Run("WPSStart", func(t *testing.T) {
		mock.AddResponse("wpa_supplicant.wlan0", "wps_start", map[string]any{"result": 0})

		err := mgr.STA("wpa_supplicant.wlan0").WPSStart(ctx, true)
		if err != nil {
			t.Fatalf("WPSStart failed: %v", err)
		}

		call := mock.GetLastCall()

		params, ok := call.Data.(map[string]any)
		if !ok {
			t.Fatalf("call.Data is not map[string]any")
		}

		if params["multi_ap"] != true {
			t.Errorf("unexpected params: %v", params)
		}
	})
}

func testWpaSTAWPSCancel(t *testing.T, ctx context.Context, mock *testutil.MockTransport, mgr *wpa_supplicant.Manager) {
	t.Helper()
	t.Run("WPSCancel", func(t *testing.T) {
		mock.AddResponse("wpa_supplicant.wlan0", "wps_cancel", map[string]any{"result": 0})

		err := mgr.STA("wpa_supplicant.wlan0").WPSCancel(ctx)
		if err != nil {
			t.Fatalf("WPSCancel failed: %v", err)
		}

		call := mock.GetLastCall()
		if call.Service != "wpa_supplicant.wlan0" || call.Method != "wps_cancel" {
			t.Errorf("unexpected call: %s.%s", call.Service, call.Method)
		}
	})
}

func testWpaSTAControl(t *testing.T, ctx context.Context, mock *testutil.MockTransport, mgr *wpa_supplicant.Manager) {
	t.Helper()
	t.Run("Control", func(t *testing.T) {
		mock.AddResponse("wpa_supplicant.wlan0", "control", map[string]any{"result": "OK"})

		res, err := mgr.STA("wpa_supplicant.wlan0").Control(ctx, "PING")
		if err != nil {
			t.Fatalf("Control failed: %v", err)
		}

		if res != "OK" {
			t.Errorf("unexpected response: %s", res)
		}

		call := mock.GetLastCall()

		params, ok := call.Data.(map[string]any)
		if !ok {
			t.Fatalf("call.Data is not map[string]any")
		}

		if params["command"] != "PING" {
			t.Errorf("unexpected params: %v", params)
		}
	})
}
