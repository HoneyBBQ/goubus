// Copyright (c) 2026 honeybbq
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package rc_test

import (
	"context"
	"testing"

	"github.com/honeybbq/goubus/v2/internal/testutil"
	"github.com/honeybbq/goubus/v2/profiles/cmcc_rax3000m/rc"
)

func TestRaxRCManager(t *testing.T) {
	ctx := context.Background()
	mock := testutil.NewMockTransport()

	t.Run("List_RAX3000M_Data", func(t *testing.T) {
		err := mock.AddResponseFromFile("rc", "list", "../../../internal/testdata/rax3000m/rc_list.json")
		if err != nil {
			t.Fatalf("failed to load testdata: %v", err)
		}

		mgr := rc.New(mock)

		list, err := mgr.List(ctx, "", false)
		if err != nil {
			t.Fatalf("List failed: %v", err)
		}

		if len(list) == 0 {
			t.Error("expected non-empty list")
		}
	})

	t.Run("Init", func(t *testing.T) {
		mock.AddResponse("rc", "init", map[string]any{})
		mgr := rc.New(mock)

		err := mgr.Init(ctx, "network", "restart")
		if err != nil {
			t.Fatalf("Init failed: %v", err)
		}
	})
}
