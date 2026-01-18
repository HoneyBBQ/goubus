// Copyright (c) 2026 honeybbq
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package dhcp_test

import (
	"context"
	"testing"

	"github.com/honeybbq/goubus/v2/internal/base/dhcp"
	"github.com/honeybbq/goubus/v2/internal/testutil"
)

type mockDhcpDialect struct{}

func (d mockDhcpDialect) PrepareAddLease(req dhcp.AddLeaseRequest) any {
	return map[string]any{
		"ip":  req.IP,
		"mac": req.Mac,
	}
}

func TestDhcpManager(t *testing.T) {
	ctx := context.Background()
	mock := testutil.NewMockTransport()

	t.Run("AddLease_Generic", func(t *testing.T) {
		mock.AddResponse("dhcp", "add_lease", map[string]any{"result": 0})

		mgr := dhcp.New(mock, mockDhcpDialect{})

		err := mgr.AddLease(ctx, dhcp.AddLeaseRequest{
			IP:  "192.168.1.100",
			Mac: "00:11:22:33:44:55",
		})
		if err != nil {
			t.Fatalf("AddLease failed: %v", err)
		}

		call := mock.GetLastCall()
		if call.Method != "add_lease" {
			t.Errorf("expected method add_lease, got %s", call.Method)
		}
	})
}
