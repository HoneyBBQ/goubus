// Copyright (c) 2026 honeybbq
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package system_test

import (
	"context"
	"testing"

	"github.com/honeybbq/goubus/v2/internal/testutil"
	"github.com/honeybbq/goubus/v2/profiles/cmcc_rax3000m/system"
)

func TestRaxSystemManager(t *testing.T) {
	ctx := context.Background()
	mock := testutil.NewMockTransport()

	testRaxSystemBoard(t, ctx, mock)
	testRaxSystemInfo(t, ctx, mock)
	testRaxSystemReboot(t, ctx, mock)
	testRaxSystemWatchdog(t, ctx, mock)
	testRaxSystemSignal(t, ctx, mock)
	testRaxSystemValidateFirmwareImage(t, ctx, mock)
	testRaxSystemSysupgrade(t, ctx, mock)
}

func testRaxSystemBoard(t *testing.T, ctx context.Context, mock *testutil.MockTransport) {
	t.Helper()
	t.Run("Board_RAX3000M_Data", func(t *testing.T) {
		err := mock.AddResponseFromFile("system", "board", "../../../internal/testdata/rax3000m/system_board.json")
		if err != nil {
			t.Fatalf("failed to load rax testdata: %v", err)
		}

		mgr := system.New(mock)

		board, err := mgr.Board(ctx)
		if err != nil {
			t.Fatalf("Board failed: %v", err)
		}

		if board.Model != "CMCC RAX3000M" {
			t.Errorf("expected CMCC RAX3000M, got %s", board.Model)
		}
	})
}

func testRaxSystemInfo(t *testing.T, ctx context.Context, mock *testutil.MockTransport) {
	t.Helper()
	t.Run("Info", func(t *testing.T) {
		err := mock.AddResponseFromFile("system", "info", "../../../internal/testdata/rax3000m/system_info.json")
		if err != nil {
			t.Fatalf("failed to load rax testdata: %v", err)
		}

		mgr := system.New(mock)

		info, err := mgr.Info(ctx)
		if err != nil {
			t.Fatalf("Info failed: %v", err)
		}

		if info.Uptime == 0 {
			t.Error("expected non-zero uptime")
		}
	})
}

func testRaxSystemReboot(t *testing.T, ctx context.Context, mock *testutil.MockTransport) {
	t.Helper()
	t.Run("Reboot", func(t *testing.T) {
		mock.AddResponse("system", "reboot", map[string]any{})
		mgr := system.New(mock)

		err := mgr.Reboot(ctx)
		if err != nil {
			t.Fatalf("Reboot failed: %v", err)
		}
	})
}

func testRaxSystemWatchdog(t *testing.T, ctx context.Context, mock *testutil.MockTransport) {
	t.Helper()
	t.Run("Watchdog", func(t *testing.T) {
		mock.AddResponse("system", "watchdog", map[string]any{})
		mgr := system.New(mock)

		err := mgr.Watchdog(ctx, system.WatchdogRequest{Timeout: 30})
		if err != nil {
			t.Fatalf("Watchdog failed: %v", err)
		}
	})
}

func testRaxSystemSignal(t *testing.T, ctx context.Context, mock *testutil.MockTransport) {
	t.Helper()
	t.Run("Signal", func(t *testing.T) {
		mock.AddResponse("system", "signal", map[string]any{})
		mgr := system.New(mock)

		err := mgr.Signal(ctx, 1234, 15)
		if err != nil {
			t.Fatalf("Signal failed: %v", err)
		}
	})
}

func testRaxSystemValidateFirmwareImage(t *testing.T, ctx context.Context, mock *testutil.MockTransport) {
	t.Helper()
	t.Run("ValidateFirmwareImage", func(t *testing.T) {
		mock.AddResponse("system", "validate_firmware_image", map[string]any{"valid": true})
		mgr := system.New(mock)

		res, err := mgr.ValidateFirmwareImage(ctx, "/tmp/firmware.bin")
		if err != nil {
			t.Fatalf("ValidateFirmwareImage failed: %v", err)
		}

		if res["valid"] != true {
			t.Errorf("expected valid true, got %v", res["valid"])
		}
	})
}

func testRaxSystemSysupgrade(t *testing.T, ctx context.Context, mock *testutil.MockTransport) {
	t.Helper()
	t.Run("Sysupgrade", func(t *testing.T) {
		mock.AddResponse("system", "sysupgrade", map[string]any{})
		mgr := system.New(mock)

		err := mgr.Sysupgrade(ctx, system.SysupgradeRequest{Path: "/tmp/firmware.bin"})
		if err != nil {
			t.Fatalf("Sysupgrade failed: %v", err)
		}
	})
}
