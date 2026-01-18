// Copyright (c) 2026 honeybbq
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package wireless_test

import (
	"context"
	"testing"

	"github.com/honeybbq/goubus/v2/internal/testutil"
	"github.com/honeybbq/goubus/v2/profiles/cmcc_rax3000m/wireless"
)

const raxTestDataDir = "../../../internal/testdata/rax3000m/"

func TestRaxWirelessManager(t *testing.T) {
	ctx := context.Background()
	mock := testutil.NewMockTransport()
	mgr := wireless.New(mock)

	t.Run("AssocList_RAX3000M_Data", func(t *testing.T) {
		testRaxAssocList(t, ctx, mock, mgr)
	})

	t.Run("Info_RealData", func(t *testing.T) {
		testRaxInfo(t, ctx, mock, mgr)
	})

	t.Run("Scan", func(t *testing.T) {
		testRaxScan(t, ctx, mock, mgr)
	})

	t.Run("FreqList", func(t *testing.T) {
		testRaxFreqList(t, ctx, mock, mgr)
	})

	t.Run("TxPowerList", func(t *testing.T) {
		testRaxTxPowerList(t, ctx, mock, mgr)
	})

	t.Run("CountryList", func(t *testing.T) {
		testRaxCountryList(t, ctx, mock, mgr)
	})

	t.Run("Survey", func(t *testing.T) {
		testRaxSurvey(t, ctx, mock, mgr)
	})

	t.Run("PhyName", func(t *testing.T) {
		testRaxPhyName(t, ctx, mock, mgr)
	})

	t.Run("Devices", func(t *testing.T) {
		testRaxDevices(t, ctx, mock, mgr)
	})
}

func testRaxAssocList(t *testing.T, ctx context.Context, mock *testutil.MockTransport, mgr *wireless.Manager) {
	t.Helper()

	err := mock.AddResponseFromFile("iwinfo", "assoclist", raxTestDataDir+"iwinfo_assoclist_phy0.json")
	if err != nil {
		t.Fatalf("failed to load testdata: %v", err)
	}

	assoc, err := mgr.AssocList(ctx, "phy0-ap0")
	if err != nil {
		t.Fatalf("AssocList failed: %v", err)
	}

	if len(assoc) != 0 {
		t.Logf("Found %d associated stations", len(assoc))
	}
}

func testRaxInfo(t *testing.T, ctx context.Context, mock *testutil.MockTransport, mgr *wireless.Manager) {
	t.Helper()

	err := mock.AddResponseFromFile("iwinfo", "info", raxTestDataDir+"iwinfo_info_phy0.json")
	if err != nil {
		t.Fatalf("failed to load testdata: %v", err)
	}

	info, err := mgr.Info(ctx, "phy0-ap0")
	if err != nil {
		t.Fatalf("Info failed: %v", err)
	}

	t.Logf("Note: SSID in real data is '%s'", info.SSID)

	if info.BSSID == "" {
		t.Errorf("expected non-empty BSSID")
	}
}

func testRaxScan(t *testing.T, ctx context.Context, mock *testutil.MockTransport, mgr *wireless.Manager) {
	t.Helper()

	err := mock.AddResponseFromFile("iwinfo", "scan", raxTestDataDir+"iwinfo_scan_phy0.json")
	if err != nil {
		t.Fatalf("failed to load testdata: %v", err)
	}

	res, err := mgr.Scan(ctx, "phy0-ap0")
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	if res == nil {
		t.Fatal("expected non-nil result")
	}
}

func testRaxFreqList(t *testing.T, ctx context.Context, mock *testutil.MockTransport, mgr *wireless.Manager) {
	t.Helper()

	err := mock.AddResponseFromFile("iwinfo", "freqlist", raxTestDataDir+"iwinfo_freqlist_phy0.json")
	if err != nil {
		t.Fatalf("failed to load testdata: %v", err)
	}

	res, err := mgr.FreqList(ctx, "phy0-ap0")
	if err != nil {
		t.Fatalf("FreqList failed: %v", err)
	}

	if res == nil {
		t.Fatal("expected non-nil result")
	}
}

func testRaxTxPowerList(t *testing.T, ctx context.Context, mock *testutil.MockTransport, mgr *wireless.Manager) {
	t.Helper()
	mock.AddResponse("iwinfo", "txpowerlist", map[string]any{"results": []any{}})

	res, err := mgr.TxPowerList(ctx, "phy0-ap0")
	if err != nil {
		t.Fatalf("TxPowerList failed: %v", err)
	}

	if res == nil {
		t.Fatal("expected non-nil result")
	}
}

func testRaxCountryList(t *testing.T, ctx context.Context, mock *testutil.MockTransport, mgr *wireless.Manager) {
	t.Helper()
	mock.AddResponse("iwinfo", "countrylist", map[string]any{"results": []any{}})

	res, err := mgr.CountryList(ctx, "phy0-ap0")
	if err != nil {
		t.Fatalf("CountryList failed: %v", err)
	}

	if res == nil {
		t.Fatal("expected non-nil result")
	}
}

func testRaxSurvey(t *testing.T, ctx context.Context, mock *testutil.MockTransport, mgr *wireless.Manager) {
	t.Helper()

	err := mock.AddResponseFromFile("iwinfo", "survey", raxTestDataDir+"iwinfo_survey_phy0.json")
	if err != nil {
		t.Fatalf("failed to load testdata: %v", err)
	}

	res, err := mgr.Survey(ctx, "phy0-ap0")
	if err != nil {
		t.Fatalf("Survey failed: %v", err)
	}

	if res == nil {
		t.Fatal("expected non-nil result")
	}
}

func testRaxPhyName(t *testing.T, ctx context.Context, mock *testutil.MockTransport, mgr *wireless.Manager) {
	t.Helper()
	mock.AddResponse("iwinfo", "phyname", map[string]any{"phyname": "phy0"})

	res, err := mgr.PhyName(ctx, "wifinet1")
	if err != nil {
		t.Fatalf("PhyName failed: %v", err)
	}

	if res != "phy0" {
		t.Errorf("expected phy0, got %s", res)
	}
}

func testRaxDevices(t *testing.T, ctx context.Context, mock *testutil.MockTransport, mgr *wireless.Manager) {
	t.Helper()

	err := mock.AddResponseFromFile("iwinfo", "devices", raxTestDataDir+"iwinfo_devices.json")
	if err != nil {
		t.Fatalf("failed to load testdata: %v", err)
	}

	res, err := mgr.Devices(ctx)
	if err != nil {
		t.Fatalf("Devices failed: %v", err)
	}

	if len(res) == 0 {
		t.Error("expected non-empty devices")
	}
}
