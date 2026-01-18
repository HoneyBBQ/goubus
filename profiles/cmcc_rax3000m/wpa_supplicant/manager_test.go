// Copyright (c) 2026 honeybbq
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package wpa_supplicant_test

import (
	"context"
	"testing"

	"github.com/honeybbq/goubus/v2/internal/testutil"
	"github.com/honeybbq/goubus/v2/profiles/cmcc_rax3000m/wpa_supplicant"
)

func TestRaxWpaSupplicantManager(t *testing.T) {
	ctx := context.Background()
	mock := testutil.NewMockTransport()

	t.Run("STA_Methods", func(t *testing.T) {
		mock.AddResponse("phy1-sta0", "reload", map[string]any{})
		mock.AddResponse("phy1-sta0", "wps_start", map[string]any{})
		mock.AddResponse("phy1-sta0", "wps_cancel", map[string]any{})
		mock.AddResponse("phy1-sta0", "control", map[string]any{"result": "OK"})

		mgr := wpa_supplicant.New(mock)
		sta := mgr.STA("phy1-sta0")
		_ = sta.Reload(ctx)
		_ = sta.WPSStart(ctx, false)
		_ = sta.WPSCancel(ctx)
		_, _ = sta.Control(ctx, "PING")
	})

	t.Run("Global_Methods", func(t *testing.T) {
		mock.AddResponse("wpa_supplicant", "iface_status", map[string]any{})
		mgr := wpa_supplicant.New(mock)
		_, _ = mgr.IfaceStatus(ctx, "wlan1")
	})
}
