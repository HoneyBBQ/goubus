// Copyright (c) 2026 honeybbq
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package hostapd_test

import (
	"context"
	"testing"

	"github.com/honeybbq/goubus/v2/internal/testutil"
	"github.com/honeybbq/goubus/v2/profiles/cmcc_rax3000m/hostapd"
)

func TestRaxHostapdManager(t *testing.T) {
	ctx := context.Background()
	mock := testutil.NewMockTransport()

	t.Run("AP_Methods", func(t *testing.T) {
		mock.AddResponse("phy0-ap0", "get_clients", map[string]any{"clients": map[string]any{}})
		mock.AddResponse("phy0-ap0", "get_status", map[string]any{"status": "up"})
		mock.AddResponse("phy0-ap0", "del_client", map[string]any{})
		mock.AddResponse("phy0-ap0", "switch_chan", map[string]any{})

		mgr := hostapd.New(mock)
		ap := mgr.AP("phy0-ap0")
		_, _ = ap.GetClients(ctx)
		_, _ = ap.GetStatus(ctx)
		_ = ap.DelClient(ctx, "AA:BB:CC:DD:EE:FF", 1, true, 30)
		_ = ap.SwitchChan(ctx, 2412, 20)
	})

	t.Run("Global_Methods", func(t *testing.T) {
		mock.AddResponse("hostapd", "reload", map[string]any{})
		mock.AddResponse("hostapd", "bss_info", map[string]any{})

		mgr := hostapd.New(mock)
		_ = mgr.Reload(ctx, "phy0", 0)
		_, _ = mgr.BSSInfo(ctx, "wlan0")
	})
}
