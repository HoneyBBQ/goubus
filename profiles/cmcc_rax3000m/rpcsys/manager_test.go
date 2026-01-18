// Copyright (c) 2026 honeybbq
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package rpcsys_test

import (
	"context"
	"testing"

	"github.com/honeybbq/goubus/v2/internal/testutil"
	"github.com/honeybbq/goubus/v2/profiles/cmcc_rax3000m/rpcsys"
)

func TestRaxRpcSysManager(t *testing.T) {
	ctx := context.Background()
	mock := testutil.NewMockTransport()

	t.Run("PackageList_RealData", func(t *testing.T) {
		err := mock.AddResponseFromFile("rpc-sys", "packagelist",
			"../../../internal/testdata/rax3000m/rpc_sys_packagelist.json")
		if err != nil {
			t.Fatalf("failed to load testdata: %v", err)
		}

		mgr := rpcsys.New(mock)

		res, err := mgr.PackageList(ctx, true)
		if err != nil {
			t.Fatalf("PackageList failed: %v", err)
		}

		if len(res) == 0 {
			t.Error("expected non-empty package list")
		}
	})

	t.Run("Upgrade_Methods", func(t *testing.T) {
		mock.AddResponse("rpc-sys", "password_set", map[string]any{})
		mock.AddResponse("rpc-sys", "factory", map[string]any{})
		mock.AddResponse("rpc-sys", "upgrade_start", map[string]any{})
		mock.AddResponse("rpc-sys", "upgrade_test", map[string]any{})
		mock.AddResponse("rpc-sys", "upgrade_clean", map[string]any{})
		mock.AddResponse("rpc-sys", "reboot", map[string]any{})

		mgr := rpcsys.New(mock)
		_ = mgr.PasswordSet(ctx, "root", "password")
		_ = mgr.Factory(ctx)
		_ = mgr.UpgradeStart(ctx, true)
		_ = mgr.UpgradeTest(ctx)
		_ = mgr.UpgradeClean(ctx)
		_ = mgr.Reboot(ctx)
	})
}
