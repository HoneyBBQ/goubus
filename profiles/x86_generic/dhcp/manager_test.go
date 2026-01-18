// Copyright (c) 2026 honeybbq
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package dhcp_test

import (
	"context"
	"testing"

	"github.com/honeybbq/goubus/v2/internal/testutil"
	"github.com/honeybbq/goubus/v2/profiles/x86_generic/dhcp"
)

func TestX86DHCPManager(t *testing.T) {
	ctx := context.Background()
	mock := testutil.NewMockTransport()

	t.Run("AddLease_NoConversion", func(t *testing.T) {
		mock.AddResponse("dhcp", "add_lease", map[string]any{"result": 0})

		mgr := dhcp.New(mock)
		req := dhcp.AddLeaseRequest{
			IP:   "192.168.1.100",
			Mac:  "11:22:33:44:55:66",
			DUID: "standard-duid",
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

		// Verify conversion for x86 (standard expects array for mac, string for duid)
		macs, isMac := data["mac"].([]string)
		if !isMac || len(macs) == 0 || macs[0] != "11:22:33:44:55:66" {
			t.Errorf("expected mac to be []string, got %v", data["mac"])
		}

		duid, isDuid := data["duid"].(string)
		if !isDuid || duid != "standard-duid" {
			t.Errorf("expected duid to be string, got %v", data["duid"])
		}
	})
}
