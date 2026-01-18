// Copyright (c) 2026 honeybbq
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package luci_test

import (
	"context"
	"testing"

	"github.com/honeybbq/goubus/v2/internal/testutil"
	"github.com/honeybbq/goubus/v2/profiles/cmcc_rax3000m/luci"
)

func TestRaxLuciManagerLeases(t *testing.T) {
	ctx := context.Background()
	mock := testutil.NewMockTransport()

	t.Run("GetDHCPLeases_RealData", func(t *testing.T) {
		err := mock.AddResponseFromFile("luci-rpc", "getDHCPLeases",
			"../../../internal/testdata/rax3000m/luci_rpc_getDHCPLeases.json")
		if err != nil {
			t.Fatalf("failed to load testdata: %v", err)
		}

		mgr := luci.New(mock)

		res, err := mgr.GetDHCPLeases(ctx, 4)
		if err != nil {
			t.Fatalf("GetDHCPLeases failed: %v", err)
		}

		if len(res.IPv4Leases) == 0 {
			t.Log("Note: No leases in real data")
		}
	})
}

func TestRaxLuciManagerBoard(t *testing.T) {
	ctx := context.Background()
	mock := testutil.NewMockTransport()

	testRaxLuciGetBoardJSON(t, ctx, mock)
	testRaxLuciGetHostHints(t, ctx, mock)
	testRaxLuciGetNetworkDevices(t, ctx, mock)
	testRaxLuciGetDHCPLeasesFamily(t, ctx, mock)
	testRaxLuciGetVersion(t, ctx, mock)
	testRaxLuciGetUnixtime(t, ctx, mock)
	testRaxLuciGetFeatures(t, ctx, mock)
	testRaxLuciGetProcessList(t, ctx, mock)
}

func testRaxLuciGetBoardJSON(t *testing.T, ctx context.Context, mock *testutil.MockTransport) {
	t.Helper()
	t.Run("GetBoardJSON_RealData", func(t *testing.T) {
		err := mock.AddResponseFromFile("luci-rpc", "getBoardJSON",
			"../../../internal/testdata/rax3000m/luci_rpc_getBoardJSON.json")
		if err != nil {
			t.Fatalf("failed to load testdata: %v", err)
		}

		mgr := luci.New(mock)

		board, err := mgr.GetBoardJSON(ctx)
		if err != nil {
			t.Fatalf("GetBoardJSON failed: %v", err)
		}

		if board.Model.Name != "CMCC RAX3000M" {
			t.Errorf("expected RAX3000M, got %s", board.Model.Name)
		}
	})
}

func testRaxLuciGetHostHints(t *testing.T, ctx context.Context, mock *testutil.MockTransport) {
	t.Helper()
	t.Run("GetHostHints_RealData", func(t *testing.T) {
		err := mock.AddResponseFromFile("luci-rpc", "getHostHints",
			"../../../internal/testdata/rax3000m/luci_rpc_getHostHints.json")
		if err != nil {
			t.Fatalf("failed to load testdata: %v", err)
		}

		mgr := luci.New(mock)

		hints, err := mgr.GetHostHints(ctx)
		if err != nil {
			t.Fatalf("GetHostHints failed: %v", err)
		}

		if len(hints) == 0 {
			t.Log("Note: No hints in real data")
		}
	})
}

func testRaxLuciGetNetworkDevices(t *testing.T, ctx context.Context, mock *testutil.MockTransport) {
	t.Helper()
	t.Run("GetNetworkDevices_RealData", func(t *testing.T) {
		err := mock.AddResponseFromFile("luci-rpc", "getNetworkDevices",
			"../../../internal/testdata/rax3000m/luci_rpc_getNetworkDevices.json")
		if err != nil {
			t.Fatalf("failed to load testdata: %v", err)
		}

		mgr := luci.New(mock)

		devs, err := mgr.GetNetworkDevices(ctx)
		if err != nil {
			t.Fatalf("GetNetworkDevices failed: %v", err)
		}

		if len(devs) == 0 {
			t.Log("Note: No network devices in real data")
		}
	})
}

func testRaxLuciGetDHCPLeasesFamily(t *testing.T, ctx context.Context, mock *testutil.MockTransport) {
	t.Helper()
	t.Run("GetDHCPLeases_Family_Data", func(t *testing.T) {
		err := mock.AddResponseFromFile("luci-rpc", "getDHCPLeases",
			"../../../internal/testdata/rax3000m/luci_rpc_getDHCPLeases.json")
		if err != nil {
			t.Fatalf("failed to load testdata: %v", err)
		}

		mgr := luci.New(mock)

		res, err := mgr.GetDHCPLeases(ctx, 6)
		if err != nil {
			t.Fatalf("GetDHCPLeases failed: %v", err)
		}

		if len(res.IPv6Leases) == 0 {
			t.Log("Note: No leases in real data")
		}
	})
}

func testRaxLuciGetVersion(t *testing.T, ctx context.Context, mock *testutil.MockTransport) {
	t.Helper()
	t.Run("GetVersion", func(t *testing.T) {
		err := mock.AddResponseFromFile("luci", "getVersion", "../../../internal/testdata/rax3000m/luci_getVersion.json")
		if err != nil {
			t.Fatalf("failed to load testdata: %v", err)
		}

		mgr := luci.New(mock)

		res, err := mgr.GetVersion(ctx)
		if err != nil {
			t.Fatalf("GetVersion failed: %v", err)
		}

		if res == nil {
			t.Fatal("expected non-nil result")
		}
	})
}

func testRaxLuciGetUnixtime(t *testing.T, ctx context.Context, mock *testutil.MockTransport) {
	t.Helper()
	t.Run("GetUnixtime", func(t *testing.T) {
		err := mock.AddResponseFromFile("luci", "getUnixtime", "../../../internal/testdata/rax3000m/luci_getUnixtime.json")
		if err != nil {
			t.Fatalf("failed to load testdata: %v", err)
		}

		mgr := luci.New(mock)

		res, err := mgr.GetUnixtime(ctx)
		if err != nil {
			t.Fatalf("GetUnixtime failed: %v", err)
		}

		if res.IsZero() {
			t.Error("expected non-zero time")
		}
	})
}

func testRaxLuciGetFeatures(t *testing.T, ctx context.Context, mock *testutil.MockTransport) {
	t.Helper()
	t.Run("GetFeatures", func(t *testing.T) {
		mock.AddResponse("luci", "getFeatures", map[string]any{"dhcp": true, "ipv6": true})
		mgr := luci.New(mock)

		res, err := mgr.GetFeatures(ctx)
		if err != nil {
			t.Fatalf("GetFeatures failed: %v", err)
		}

		if res == nil {
			t.Fatal("expected non-nil result")
		}
	})
}

func testRaxLuciGetProcessList(t *testing.T, ctx context.Context, mock *testutil.MockTransport) {
	t.Helper()
	t.Run("GetProcessList", func(t *testing.T) {
		mock.AddResponse("luci", "getProcessList", []any{
			map[string]any{"pid": 1, "user": "root", "command": "init"},
		})

		mgr := luci.New(mock)

		res, err := mgr.GetProcessList(ctx)
		if err != nil {
			t.Fatalf("GetProcessList failed: %v", err)
		}

		if len(res) == 0 {
			t.Error("expected non-empty process list")
		}
	})
}
