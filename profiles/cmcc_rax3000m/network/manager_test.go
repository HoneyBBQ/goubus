// Copyright (c) 2026 honeybbq
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package network_test

import (
	"context"
	"testing"

	"github.com/honeybbq/goubus/v2/internal/testutil"
	"github.com/honeybbq/goubus/v2/profiles/cmcc_rax3000m/network"
)

func TestRaxNetworkManager(t *testing.T) {
	ctx := context.Background()
	mock := testutil.NewMockTransport()
	mgr := network.New(mock)

	t.Run("Dump", func(t *testing.T) {
		testRaxNetworkDump(t, ctx, mock, mgr)
	})

	t.Run("Device", func(t *testing.T) {
		testRaxNetworkDevice(t, ctx, mock, mgr)
	})

	t.Run("Interface", func(t *testing.T) {
		testRaxNetworkInterface(t, ctx, mock, mgr)
	})

	t.Run("Wireless", func(t *testing.T) {
		testRaxNetworkWireless(t, ctx, mock, mgr)
	})

	t.Run("General", func(t *testing.T) {
		testRaxNetworkGeneral(t, ctx, mock, mgr)
	})
}

func testRaxNetworkDump(t *testing.T, ctx context.Context, mock *testutil.MockTransport, mgr *network.Manager) {
	t.Helper()

	const path = "../../../internal/testdata/rax3000m/network_interface_dump.json"

	err := mock.AddResponseFromFile("network.interface", "dump", path)
	if err != nil {
		t.Fatalf("failed to load testdata: %v", err)
	}

	ifaces, err := mgr.Dump(ctx)
	if err != nil {
		t.Fatalf("Dump failed: %v", err)
	}

	foundLan := false

	for _, iface := range ifaces {
		if iface.Interface == "lan" {
			foundLan = true

			if iface.Device != "br-lan" {
				t.Errorf("expected br-lan, got %s", iface.Device)
			}
		}
	}

	if !foundLan {
		t.Error("lan interface not found in dump")
	}
}

func testRaxNetworkDevice(t *testing.T, ctx context.Context, mock *testutil.MockTransport, mgr *network.Manager) {
	t.Helper()
	t.Run("Status", func(t *testing.T) {
		mock.AddResponse("network.device", "status", map[string]any{
			"type":    "bridge",
			"up":      true,
			"macaddr": "c8:75:f4:74:c6:94",
		})

		status, err := mgr.Devices().Status(ctx, "br-lan")
		if err != nil {
			t.Fatalf("Status failed: %v", err)
		}

		if status["br-lan"].MacAddr == "" {
			t.Errorf("expected non-empty MacAddr")
		}
	})

	t.Run("Methods", func(t *testing.T) {
		mock.AddResponse("network.device", "set_alias", map[string]any{})
		mock.AddResponse("network.device", "set_state", map[string]any{})
		mock.AddResponse("network.device", "stp_init", map[string]any{})

		devs := mgr.Devices()
		_ = devs.SetAlias(ctx, network.DeviceSetAliasRequest{Device: "eth0", Alias: []string{"wan"}})
		_ = devs.SetState(ctx, network.DeviceSetStateRequest{Name: "eth0"})
		_ = devs.STPInit(ctx)
	})
}

func testRaxNetworkInterface(t *testing.T, ctx context.Context, mock *testutil.MockTransport, mgr *network.Manager) {
	t.Helper()
	t.Run("Status", func(t *testing.T) {
		mock.AddResponse("network.interface.lan", "status", map[string]any{"up": true, "device": "br-lan"})

		status, err := mgr.Interface("lan").Status(ctx)
		if err != nil {
			t.Fatalf("Status failed: %v", err)
		}

		if bool(status.Up) != true {
			t.Errorf("expected up true, got %v", status.Up)
		}
	})

	t.Run("Methods", func(t *testing.T) {
		iface := mgr.Interface("lan")

		mock.AddResponse("network.interface.lan", "up", map[string]any{})

		_ = iface.Up(ctx)

		mock.AddResponse("network.interface.lan", "down", map[string]any{})

		_ = iface.Down(ctx)
	})
}

func testRaxNetworkWireless(t *testing.T, ctx context.Context, mock *testutil.MockTransport, mgr *network.Manager) {
	t.Helper()
	t.Run("Status", func(t *testing.T) {
		const path = "../../../internal/testdata/rax3000m/network_wireless_status.json"

		err := mock.AddResponseFromFile("network.wireless", "status", path)
		if err != nil {
			t.Fatalf("failed to load testdata: %v", err)
		}

		status, err := mgr.Wireless().Status(ctx, "radio0")
		if err != nil {
			t.Fatalf("Status failed: %v", err)
		}

		if len(status) == 0 {
			t.Fatal("expected non-empty result")
		}
	})
}

func testRaxNetworkGeneral(t *testing.T, ctx context.Context, mock *testutil.MockTransport, mgr *network.Manager) {
	t.Helper()
	t.Run("RestartReload", func(t *testing.T) {
		mock.AddResponse("network", "restart", map[string]any{})

		_ = mgr.Restart(ctx)

		mock.AddResponse("network", "reload", map[string]any{})

		_ = mgr.Reload(ctx)
	})
}
