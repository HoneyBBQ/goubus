// Copyright (c) 2026 honeybbq
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package network_test

import (
	"context"
	"testing"

	"github.com/honeybbq/goubus/v2/internal/base/network"
	"github.com/honeybbq/goubus/v2/internal/testutil"
)

type mockNetworkDialect struct{}

func TestNetworkManagerDumpInterfaces(t *testing.T) {
	ctx := context.Background()
	mock := testutil.NewMockTransport()
	mock.AddResponse("network.interface", "dump", map[string]any{
		"interface": []map[string]any{
			{"interface": "lan", "up": true},
		},
	})

	mgr := network.New(mock, mockNetworkDialect{})

	ifaces, err := mgr.DumpInterfaces(ctx)
	if err != nil {
		t.Fatalf("Dump failed: %v", err)
	}

	if len(ifaces) == 0 || ifaces[0].Interface != "lan" {
		t.Errorf("unexpected interface data: %+v", ifaces)
	}
}

func TestNetworkManagerRestart(t *testing.T) {
	ctx := context.Background()
	mock := testutil.NewMockTransport()
	mock.AddResponse("network", "restart", map[string]any{"result": 0})

	mgr := network.New(mock, mockNetworkDialect{})

	err := mgr.Restart(ctx)
	if err != nil {
		t.Errorf("Restart failed: %v", err)
	}
}

func TestNetworkManagerInterfaceUpDown(t *testing.T) {
	ctx := context.Background()
	mock := testutil.NewMockTransport()
	mock.AddResponse("network.interface.lan", "up", map[string]any{"result": 0})
	mock.AddResponse("network.interface.lan", "down", map[string]any{"result": 0})

	mgr := network.New(mock, mockNetworkDialect{})

	err := mgr.Interface("lan").Up(ctx)
	if err != nil {
		t.Errorf("Up failed: %v", err)
	}

	err = mgr.Interface("lan").Down(ctx)
	if err != nil {
		t.Errorf("Down failed: %v", err)
	}
}

func TestNetworkManagerDeviceStatus(t *testing.T) {
	ctx := context.Background()
	mock := testutil.NewMockTransport()
	mock.AddResponse("network.device", "status", map[string]any{
		"eth0": map[string]any{
			"type":    "device",
			"up":      true,
			"macaddr": "00:aa:bb:cc:dd:ee",
		},
	})

	mgr := network.New(mock, mockNetworkDialect{})

	devices, err := mgr.Devices().Status(ctx, "")
	if err != nil {
		t.Fatalf("Device Status failed: %v", err)
	}

	if dev, ok := devices["eth0"]; !ok || dev.MacAddr != "00:aa:bb:cc:dd:ee" {
		t.Errorf("unexpected device data: %+v", devices)
	}
}
