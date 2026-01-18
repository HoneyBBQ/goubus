// Copyright (c) 2026 honeybbq
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package luci_test

import (
	"context"
	"testing"

	"github.com/honeybbq/goubus/v2/internal/testutil"
	"github.com/honeybbq/goubus/v2/profiles/x86_generic/luci"
)

func TestX86LuciManager(t *testing.T) {
	ctx := context.Background()
	mock := testutil.NewMockTransport()

	t.Run("GetDHCPLeases_RealData", func(t *testing.T) {
		const path = "../../../internal/testdata/x86_generic/luci_rpc_getDHCPLeases.json"

		err := mock.AddResponseFromFile("luci-rpc", "getDHCPLeases", path)
		if err != nil {
			t.Fatalf("failed to load testdata: %v", err)
		}

		mgr := luci.New(mock)

		_, err = mgr.GetDHCPLeases(ctx, 0)
		if err != nil {
			t.Fatalf("GetDHCPLeases failed: %v", err)
		}
		// x86 data might be empty, as long as parsing succeeds, it passes.
	})
}
