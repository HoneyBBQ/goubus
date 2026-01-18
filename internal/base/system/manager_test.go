// Copyright (c) 2026 honeybbq
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package system_test

import (
	"context"
	"testing"

	"github.com/honeybbq/goubus/v2/internal/base/system"
	"github.com/honeybbq/goubus/v2/internal/testutil"
)

func TestSystemManager(t *testing.T) {
	ctx := context.Background()
	mock := testutil.NewMockTransport()
	mgr := system.New(mock)

	t.Run("Info", func(t *testing.T) {
		mock.AddResponse("system", "info", map[string]any{
			"uptime": 1234,
			"load":   []any{1, 2, 3},
			"memory": map[string]any{"total": 1024, "free": 512},
		})

		info, err := mgr.Info(ctx)
		if err != nil {
			t.Fatalf("Info failed: %v", err)
		}

		if info.Uptime != 1234 {
			t.Errorf("unexpected uptime: %d", info.Uptime)
		}
	})

	t.Run("Board", func(t *testing.T) {
		testSystemBoard(t, ctx, mock, mgr)
	})

	t.Run("BasicOperations", func(t *testing.T) {
		testSystemBasicOps(t, ctx, mock, mgr)
	})

	t.Run("Firmware", func(t *testing.T) {
		testSystemFirmware(t, ctx, mock, mgr)
	})
}

func testSystemBoard(t *testing.T, ctx context.Context, mock *testutil.MockTransport, mgr *system.Manager) {
	t.Helper()
	mock.AddResponse("system", "board", map[string]any{
		"model": "Generic OpenWrt Device",
		"release": map[string]any{
			"version": "Snapshot",
		},
	})

	board, err := mgr.Board(ctx)
	if err != nil {
		t.Fatalf("Board failed: %v", err)
	}

	if board.Model != "Generic OpenWrt Device" {
		t.Errorf("unexpected model: %s", board.Model)
	}
}

func testSystemBasicOps(t *testing.T, ctx context.Context, mock *testutil.MockTransport, mgr *system.Manager) {
	t.Helper()
	t.Run("Reboot", func(t *testing.T) {
		mock.AddResponse("system", "reboot", map[string]any{"result": 0})

		err := mgr.Reboot(ctx)
		if err != nil {
			t.Errorf("Reboot failed: %v", err)
		}
	})

	t.Run("Watchdog", func(t *testing.T) {
		mock.AddResponse("system", "watchdog", map[string]any{})

		err := mgr.Watchdog(ctx, system.WatchdogRequest{Timeout: 30})
		if err != nil {
			t.Fatalf("Watchdog failed: %v", err)
		}
	})

	t.Run("Signal", func(t *testing.T) {
		mock.AddResponse("system", "signal", map[string]any{})

		err := mgr.Signal(ctx, 1234, 15)
		if err != nil {
			t.Fatalf("Signal failed: %v", err)
		}
	})
}

func testSystemFirmware(t *testing.T, ctx context.Context, mock *testutil.MockTransport, mgr *system.Manager) {
	t.Helper()
	t.Run("Validate", func(t *testing.T) {
		mock.AddResponse("system", "validate_firmware_image", map[string]any{"valid": true, "checksum": "123456"})

		res, err := mgr.ValidateFirmwareImage(ctx, "/tmp/firmware.bin")
		if err != nil {
			t.Fatalf("ValidateFirmwareImage failed: %v", err)
		}

		if res["valid"] != true {
			t.Errorf("expected valid image, got %+v", res)
		}
	})

	t.Run("Sysupgrade", func(t *testing.T) {
		mock.AddResponse("system", "sysupgrade", map[string]any{"result": 0})

		err := mgr.Sysupgrade(ctx, system.SysupgradeRequest{Path: "/tmp/firmware.bin"})
		if err != nil {
			t.Fatalf("Sysupgrade failed: %v", err)
		}
	})
}
