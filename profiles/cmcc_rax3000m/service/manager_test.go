// Copyright (c) 2026 honeybbq
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package service_test

import (
	"context"
	"testing"

	"github.com/honeybbq/goubus/v2/internal/testutil"
	"github.com/honeybbq/goubus/v2/profiles/cmcc_rax3000m/service"
)

func TestRaxServiceManager(t *testing.T) {
	ctx := context.Background()
	mock := testutil.NewMockTransport()

	testRaxServiceList(t, ctx, mock)
	testRaxServiceDelete(t, ctx, mock)
	testRaxServiceSignal(t, ctx, mock)
	testRaxServiceSetAdd(t, ctx, mock)
	testRaxServiceUpdate(t, ctx, mock)
	testRaxServiceEvent(t, ctx, mock)
	testRaxServiceValidate(t, ctx, mock)
	testRaxServiceGetSetData(t, ctx, mock)
	testRaxServiceState(t, ctx, mock)
	testRaxServiceWatchdog(t, ctx, mock)
}

func testRaxServiceList(t *testing.T, ctx context.Context, mock *testutil.MockTransport) {
	t.Helper()
	t.Run("List_RAX3000M_Data", func(t *testing.T) {
		err := mock.AddResponseFromFile("service", "list", "../../../internal/testdata/rax3000m/service_list.json")
		if err != nil {
			t.Fatalf("failed to load testdata: %v", err)
		}

		mgr := service.New(mock)

		list, err := mgr.List(ctx, "", false)
		if err != nil {
			t.Fatalf("List failed: %v", err)
		}

		if len(list) == 0 {
			t.Error("expected non-empty list")
		}
	})
}

func testRaxServiceDelete(t *testing.T, ctx context.Context, mock *testutil.MockTransport) {
	t.Helper()
	t.Run("Delete", func(t *testing.T) {
		mock.AddResponse("service", "delete", map[string]any{})
		mgr := service.New(mock)

		err := mgr.Delete(ctx, "test", "inst1")
		if err != nil {
			t.Fatalf("Delete failed: %v", err)
		}
	})
}

func testRaxServiceSignal(t *testing.T, ctx context.Context, mock *testutil.MockTransport) {
	t.Helper()
	t.Run("Signal", func(t *testing.T) {
		mock.AddResponse("service", "signal", map[string]any{})
		mgr := service.New(mock)

		err := mgr.Signal(ctx, "test", "inst1", 15)
		if err != nil {
			t.Fatalf("Signal failed: %v", err)
		}
	})
}

func testRaxServiceSetAdd(t *testing.T, ctx context.Context, mock *testutil.MockTransport) {
	t.Helper()
	t.Run("Set/Add", func(t *testing.T) {
		mock.AddResponse("service", "set", map[string]any{})
		mock.AddResponse("service", "add", map[string]any{})
		mgr := service.New(mock)

		err := mgr.Set(ctx, service.SetRequest{Name: "test"})
		if err != nil {
			t.Fatalf("Set failed: %v", err)
		}

		err = mgr.Add(ctx, service.SetRequest{Name: "test"})
		if err != nil {
			t.Fatalf("Add failed: %v", err)
		}
	})
}

func testRaxServiceUpdate(t *testing.T, ctx context.Context, mock *testutil.MockTransport) {
	t.Helper()
	t.Run("Update", func(t *testing.T) {
		mock.AddResponse("service", "update_start", map[string]any{})
		mock.AddResponse("service", "update_complete", map[string]any{})
		mgr := service.New(mock)

		err := mgr.UpdateStart(ctx, "test")
		if err != nil {
			t.Fatalf("UpdateStart failed: %v", err)
		}

		err = mgr.UpdateComplete(ctx, "test")
		if err != nil {
			t.Fatalf("UpdateComplete failed: %v", err)
		}
	})
}

func testRaxServiceEvent(t *testing.T, ctx context.Context, mock *testutil.MockTransport) {
	t.Helper()
	t.Run("Event", func(t *testing.T) {
		mock.AddResponse("service", "event", map[string]any{})
		mgr := service.New(mock)

		err := mgr.Event(ctx, service.EventRequest{Type: "test"})
		if err != nil {
			t.Fatalf("Event failed: %v", err)
		}
	})
}

func testRaxServiceValidate(t *testing.T, ctx context.Context, mock *testutil.MockTransport) {
	t.Helper()
	t.Run("Validate", func(t *testing.T) {
		mock.AddResponse("service", "validate", map[string]any{"valid": true})
		mgr := service.New(mock)

		res, err := mgr.Validate(ctx, service.ValidateRequest{Package: "test"})
		if err != nil {
			t.Fatalf("Validate failed: %v", err)
		}

		if res["valid"] != true {
			t.Errorf("expected valid true")
		}
	})
}

func testRaxServiceGetSetData(t *testing.T, ctx context.Context, mock *testutil.MockTransport) {
	t.Helper()
	t.Run("GetData/SetData", func(t *testing.T) {
		mock.AddResponse("service", "get_data", map[string]any{"foo": "bar"})
		mock.AddResponse("service", "set_data", map[string]any{})
		mgr := service.New(mock)

		res, err := mgr.GetData(ctx, "test", "inst1", "type1")
		if err != nil {
			t.Fatalf("GetData failed: %v", err)
		}

		if res["foo"] != "bar" {
			t.Errorf("expected bar")
		}

		err = mgr.SetData(ctx, "test", "inst1", map[string]any{"foo": "baz"})
		if err != nil {
			t.Fatalf("SetData failed: %v", err)
		}
	})
}

func testRaxServiceState(t *testing.T, ctx context.Context, mock *testutil.MockTransport) {
	t.Helper()
	t.Run("State", func(t *testing.T) {
		mock.AddResponse("service", "state", map[string]any{"running": true})
		mgr := service.New(mock)

		res, err := mgr.State(ctx, "test", false)
		if err != nil {
			t.Fatalf("State failed: %v", err)
		}

		if res["running"] != true {
			t.Errorf("expected running true")
		}
	})
}

func testRaxServiceWatchdog(t *testing.T, ctx context.Context, mock *testutil.MockTransport) {
	t.Helper()
	t.Run("Watchdog", func(t *testing.T) {
		mock.AddResponse("service", "watchdog", map[string]any{})
		mgr := service.New(mock)

		err := mgr.Watchdog(ctx, "test", "inst1", 1, 30)
		if err != nil {
			t.Fatalf("Watchdog failed: %v", err)
		}
	})
}
