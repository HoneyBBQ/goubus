// Copyright (c) 2026 honeybbq
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package luci_test

import (
	"context"
	"testing"

	"github.com/honeybbq/goubus/v2/internal/base/luci"
	"github.com/honeybbq/goubus/v2/internal/testutil"
)

type mockLuciDialect struct {
	method string
}

func (d mockLuciDialect) GetTimeMethod() string { return d.method }

func TestLuciManager(t *testing.T) {
	ctx := context.Background()
	mock := testutil.NewMockTransport()
	testLuciGetVersion(t, ctx, mock)
	testLuciGetTimeUnix(t, ctx, mock)
	testLuciGetTimeLocal(t, ctx, mock)
	testLuciGetInitList(t, ctx, mock)
	testLuciGetTimezones(t, ctx, mock)
	testLuciGetHostHints(t, ctx, mock)
}

func testLuciGetVersion(t *testing.T, ctx context.Context, mock *testutil.MockTransport) {
	t.Helper()
	t.Run("GetVersion", func(t *testing.T) {
		mock.AddResponse("luci", "getVersion", map[string]any{
			"revision": "25.349.49508~946f77a",
			"branch":   "LuCI (HEAD detached at 946f77a) branch",
		})

		mgr := luci.New(mock, mockLuciDialect{method: "getUnixtime"})

		ver, err := mgr.GetVersion(ctx)
		if err != nil {
			t.Fatalf("GetVersion failed: %v", err)
		}

		if ver.Revision != "25.349.49508~946f77a" {
			t.Errorf("expected revision 25.349.49508~946f77a, got %s", ver.Revision)
		}
	})
}

func testLuciGetTimeUnix(t *testing.T, ctx context.Context, mock *testutil.MockTransport) {
	t.Helper()
	t.Run("GetTime_Unixtime", func(t *testing.T) {
		mock.AddResponse("luci", "getUnixtime", map[string]any{
			"result": 1737109342,
		})

		mgr := luci.New(mock, mockLuciDialect{method: "getUnixtime"})

		time, err := mgr.GetTime(ctx)
		if err != nil {
			t.Fatalf("GetTime failed: %v", err)
		}

		if time.Unix() != 1737109342 {
			t.Errorf("expected unix time 1737109342, got %d", time.Unix())
		}

		call := mock.GetLastCall()
		if call.Method != "getUnixtime" {
			t.Errorf("expected method getUnixtime, got %s", call.Method)
		}
	})
}

func testLuciGetTimeLocal(t *testing.T, ctx context.Context, mock *testutil.MockTransport) {
	t.Helper()
	t.Run("GetTime_Localtime", func(t *testing.T) {
		mock.AddResponse("luci", "getLocaltime", map[string]any{
			"result": 1737109342,
		})

		mgr := luci.New(mock, mockLuciDialect{method: "getLocaltime"})

		time, err := mgr.GetTime(ctx)
		if err != nil {
			t.Fatalf("GetTime failed: %v", err)
		}

		if time.Unix() != 1737109342 {
			t.Errorf("expected unix time 1737109342, got %d", time.Unix())
		}

		call := mock.GetLastCall()
		if call.Method != "getLocaltime" {
			t.Errorf("expected method getLocaltime, got %s", call.Method)
		}
	})
}

func testLuciGetInitList(t *testing.T, ctx context.Context, mock *testutil.MockTransport) {
	t.Helper()
	t.Run("GetInitList", func(t *testing.T) {
		mock.AddResponse("luci", "getInitList", map[string]any{
			"firewall": map[string]any{"enabled": true},
		})

		mgr := luci.New(mock, mockLuciDialect{method: "getUnixtime"})

		list, err := mgr.GetInitList(ctx, "")
		if err != nil {
			t.Fatalf("GetInitList failed: %v", err)
		}

		if _, ok := list["firewall"]; !ok {
			t.Error("expected firewall in init list")
		}
	})
}

func testLuciGetTimezones(t *testing.T, ctx context.Context, mock *testutil.MockTransport) {
	t.Helper()
	t.Run("GetTimezones", func(t *testing.T) {
		mock.AddResponse("luci", "getTimezones", map[string]any{
			"UTC": "UTC",
		})

		mgr := luci.New(mock, mockLuciDialect{method: "getUnixtime"})

		tz, err := mgr.GetTimezones(ctx)
		if err != nil {
			t.Fatalf("GetTimezones failed: %v", err)
		}

		if _, ok := tz["UTC"]; !ok {
			t.Error("expected UTC in timezones")
		}
	})
}

func testLuciGetHostHints(t *testing.T, ctx context.Context, mock *testutil.MockTransport) {
	t.Helper()
	t.Run("LuciRPC_Hints", func(t *testing.T) {
		mock.AddResponse("luci-rpc", "getHostHints", map[string]any{
			"00:11:22:33:44:55": map[string]any{"name": "test-host"},
		})

		mgr := luci.New(mock, mockLuciDialect{method: "getUnixtime"})

		hints, err := mgr.GetHostHints(ctx)
		if err != nil {
			t.Fatalf("GetHostHints failed: %v", err)
		}

		if h, ok := hints["00:11:22:33:44:55"]; !ok || h.Name != "test-host" {
			t.Error("unexpected host hints")
		}
	})
}
