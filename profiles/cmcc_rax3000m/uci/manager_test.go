// Copyright (c) 2026 honeybbq
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package uci_test

import (
	"context"
	"testing"

	baseuci "github.com/honeybbq/goubus/v2/internal/base/uci"
	"github.com/honeybbq/goubus/v2/internal/testutil"
	raxuci "github.com/honeybbq/goubus/v2/profiles/cmcc_rax3000m/uci"
)

func TestRaxUCIManager(t *testing.T) {
	ctx := context.Background()
	mock := testutil.NewMockTransport()

	testRaxUCIGetAllNetwork(t, ctx, mock)
	testRaxUCIGetSingleSection(t, ctx, mock)
	testRaxUCIGetSingleOption(t, ctx, mock)
	testRaxUCISet(t, ctx, mock)
	testRaxUCIAdd(t, ctx, mock)
	testRaxUCIDelete(t, ctx, mock)
	testRaxUCICommit(t, ctx, mock)
	testRaxUCIChanges(t, ctx, mock)
	testRaxUCIRevert(t, ctx, mock)
	testRaxUCIApply(t, ctx, mock)
	testRaxUCIStateNetwork(t, ctx, mock)
}

func testRaxUCIGetAllNetwork(t *testing.T, ctx context.Context, mock *testutil.MockTransport) {
	t.Helper()
	t.Run("GetAll_Network_RealData", func(t *testing.T) {
		err := mock.AddResponseFromFile("uci", "get", "../../../internal/testdata/rax3000m/uci_get_network.json")
		if err != nil {
			t.Fatalf("failed to load testdata: %v", err)
		}

		mgr := raxuci.New(mock)

		sections, err := mgr.Package("network").GetAll(ctx)
		if err != nil {
			t.Fatalf("GetAll failed: %v", err)
		}

		lan, hasLan := sections["lan"]
		if !hasLan {
			t.Fatal("lan section not found in network config")
		}

		if lan.Type != "interface" {
			t.Errorf("expected type interface, got %s", lan.Type)
		}

		ip, hasIp := lan.Values.First("ipaddr")
		if !hasIp || ip != "192.168.233.1" {
			t.Errorf("expected ipaddr 192.168.233.1, got %s", ip)
		}
	})
}

func testRaxUCIGetSingleSection(t *testing.T, ctx context.Context, mock *testutil.MockTransport) {
	t.Helper()
	t.Run("Get_SingleSection", func(t *testing.T) {
		mock.AddResponse("uci", "get", map[string]any{
			"values": map[string]any{
				".type":  "interface",
				"proto":  "static",
				"ipaddr": "192.168.1.1",
			},
		})

		mgr := raxuci.New(mock)

		res, err := mgr.Package("network").Section("lan").Get(ctx)
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}

		proto, _ := res.Values.First("proto")
		if proto != "static" {
			t.Errorf("expected static, got %v", proto)
		}
	})
}

func testRaxUCIGetSingleOption(t *testing.T, ctx context.Context, mock *testutil.MockTransport) {
	t.Helper()
	t.Run("Get_SingleOption", func(t *testing.T) {
		mock.AddResponse("uci", "get", map[string]any{
			"value": "192.168.1.1",
		})

		mgr := raxuci.New(mock)

		res, err := mgr.Package("network").Section("lan").Option("ipaddr").Get(ctx)
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}

		if res != "192.168.1.1" {
			t.Errorf("expected 192.168.1.1, got %v", res)
		}
	})
}

func testRaxUCISet(t *testing.T, ctx context.Context, mock *testutil.MockTransport) {
	t.Helper()
	t.Run("Set", func(t *testing.T) {
		mock.AddResponse("uci", "set", map[string]any{})
		mgr := raxuci.New(mock)

		err := mgr.Package("network").Section("lan").Option("ipaddr").Set(ctx, "192.168.1.2")
		if err != nil {
			t.Fatalf("Set failed: %v", err)
		}
	})
}

func testRaxUCIAdd(t *testing.T, ctx context.Context, mock *testutil.MockTransport) {
	t.Helper()
	t.Run("Add", func(t *testing.T) {
		mock.AddResponse("uci", "add", map[string]any{"name": "new_section"})
		mgr := raxuci.New(mock)

		err := mgr.Package("network").Add(ctx, "interface", "new_section", baseuci.NewSectionValues())
		if err != nil {
			t.Fatalf("Add failed: %v", err)
		}
	})
}

func testRaxUCIDelete(t *testing.T, ctx context.Context, mock *testutil.MockTransport) {
	t.Helper()
	t.Run("Delete", func(t *testing.T) {
		mock.AddResponse("uci", "delete", map[string]any{})
		mgr := raxuci.New(mock)

		err := mgr.Package("network").Section("lan").Delete(ctx)
		if err != nil {
			t.Fatalf("Delete failed: %v", err)
		}
	})
}

func testRaxUCICommit(t *testing.T, ctx context.Context, mock *testutil.MockTransport) {
	t.Helper()
	t.Run("Commit", func(t *testing.T) {
		mock.AddResponse("uci", "commit", map[string]any{})
		mgr := raxuci.New(mock)

		err := mgr.Package("network").Commit(ctx)
		if err != nil {
			t.Fatalf("Commit failed: %v", err)
		}
	})
}

func testRaxUCIChanges(t *testing.T, ctx context.Context, mock *testutil.MockTransport) {
	t.Helper()
	t.Run("Changes", func(t *testing.T) {
		mock.AddResponse("uci", "changes", map[string]any{"changes": map[string]any{}})
		mgr := raxuci.New(mock)

		res, err := mgr.Package("network").Changes(ctx)
		if err != nil {
			t.Fatalf("Changes failed: %v", err)
		}

		if len(res.Changes) != 0 {
			t.Errorf("expected 0 changes, got %d", len(res.Changes))
		}
	})
}

func testRaxUCIRevert(t *testing.T, ctx context.Context, mock *testutil.MockTransport) {
	t.Helper()
	t.Run("Revert", func(t *testing.T) {
		mock.AddResponse("uci", "revert", map[string]any{})
		mgr := raxuci.New(mock)

		err := mgr.Package("network").Revert(ctx)
		if err != nil {
			t.Fatalf("Revert failed: %v", err)
		}
	})
}

func testRaxUCIApply(t *testing.T, ctx context.Context, mock *testutil.MockTransport) {
	t.Helper()
	t.Run("Apply", func(t *testing.T) {
		mock.AddResponse("uci", "apply", map[string]any{})
		mgr := raxuci.New(mock)

		err := mgr.Apply(ctx, true, 30)
		if err != nil {
			t.Fatalf("Apply failed: %v", err)
		}
	})
}

func testRaxUCIStateNetwork(t *testing.T, ctx context.Context, mock *testutil.MockTransport) {
	t.Helper()
	t.Run("State_Network_RealData", func(t *testing.T) {
		err := mock.AddResponseFromFile("uci", "state", "../../../internal/testdata/rax3000m/uci_get_network.json")
		if err != nil {
			t.Fatalf("failed to load testdata: %v", err)
		}

		mgr := raxuci.New(mock)

		sections, err := mgr.Package("network").State(ctx)
		if err != nil {
			t.Fatalf("State failed: %v", err)
		}

		if len(sections) == 0 {
			t.Error("expected non-empty state")
		}
	})
}
