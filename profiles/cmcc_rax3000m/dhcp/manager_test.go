// Copyright (c) 2026 honeybbq
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package dhcp_test

import (
	"context"
	"testing"

	"github.com/honeybbq/goubus/v2/internal/testutil"
	"github.com/honeybbq/goubus/v2/profiles/cmcc_rax3000m/dhcp"
)

func TestRaxDHCPManager(t *testing.T) {
	ctx := context.Background()
	mock := testutil.NewMockTransport()
	mgr := dhcp.New(mock)

	t.Run("AddLease", func(t *testing.T) {
		testRaxDHCPAddLease(t, ctx, mock, mgr)
	})

	t.Run("IPv6", func(t *testing.T) {
		testRaxDHCPIPv6(t, ctx, mock, mgr)
	})
}

func testRaxDHCPAddLease(t *testing.T, ctx context.Context, mock *testutil.MockTransport, mgr *dhcp.Manager) {
	t.Helper()
	mock.AddResponse("dhcp", "add_lease", map[string]any{"result": 0})

	req := dhcp.AddLeaseRequest{
		IP:   "192.168.233.100",
		Mac:  "AA:BB:CC:DD:EE:FF",
		DUID: "00010001...",
	}

	err := mgr.AddLease(ctx, req)
	if err != nil {
		t.Fatalf("AddLease failed: %v", err)
	}

	call := mock.GetLastCall()

	data, isMap := call.Data.(map[string]any)

	if !isMap {
		t.Fatalf("expected map[string]any for call data, got %T", call.Data)
	}

	// Verify conversion to Array for RAX
	macs, isMac := data["mac"].([]string)
	if !isMac || len(macs) != 1 || macs[0] != "AA:BB:CC:DD:EE:FF" {
		t.Errorf("expected mac to be []string with one element, got %v", data["mac"])
	}

	duids, isDuid := data["duid"].([]string)
	if !isDuid || len(duids) != 1 || duids[0] != "00010001..." {
		t.Errorf("expected duid to be []string with one element, got %v", data["duid"])
	}
}

func testRaxDHCPIPv6(t *testing.T, ctx context.Context, mock *testutil.MockTransport, mgr *dhcp.Manager) {
	t.Helper()
	t.Run("IPv6Leases", func(t *testing.T) {
		const path = "../../../internal/testdata/rax3000m/dhcp_ipv6leases.json"

		err := mock.AddResponseFromFile("dhcp", "ipv6leases", path)
		if err != nil {
			t.Fatalf("failed to load testdata: %v", err)
		}

		res, err := mgr.IPv6Leases(ctx)
		if err != nil {
			t.Fatalf("IPv6Leases failed: %v", err)
		}

		if len(res) == 0 {
			t.Fatal("expected non-empty result")
		}
	})

	t.Run("IPv6RA", func(t *testing.T) {
		const path = "../../../internal/testdata/rax3000m/dhcp_ipv6ra.json"

		err := mock.AddResponseFromFile("dhcp", "ipv6ra", path)
		if err != nil {
			t.Fatalf("failed to load testdata: %v", err)
		}

		res, err := mgr.IPv6RA(ctx)
		if err != nil {
			t.Fatalf("IPv6RA failed: %v", err)
		}

		if len(res) == 0 {
			t.Fatal("expected non-empty result")
		}
	})
}
