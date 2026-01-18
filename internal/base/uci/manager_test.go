// Copyright (c) 2026 honeybbq
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package uci_test

import (
	"context"
	"testing"

	"github.com/honeybbq/goubus/v2/internal/base/uci"
	"github.com/honeybbq/goubus/v2/internal/testutil"
)

type mockUciDialect struct{}

const methodSet = "set"

func TestUciManager(t *testing.T) {
	ctx := context.Background()
	mock := testutil.NewMockTransport()
	mgr := uci.New(mock, mockUciDialect{})

	testUciConfigs(t, ctx, mock, mgr)
	testUciApplyConfirmRollback(t, ctx, mock, mgr)
	testUciPackageOperations(t, ctx, mock, mgr)
	testUciSectionOperations(t, ctx, mock, mgr)
	testUciOptionOperations(t, ctx, mock, mgr)
}

func testUciConfigs(t *testing.T, ctx context.Context, mock *testutil.MockTransport, mgr *uci.Manager) {
	t.Helper()
	t.Run("Configs_Generic", func(t *testing.T) {
		mock.AddResponse("uci", "configs", map[string]any{
			"configs": []string{"network", "wireless", "system"},
		})

		configs, err := mgr.Configs(ctx)
		if err != nil {
			t.Fatalf("Configs failed: %v", err)
		}

		if len(configs) != 3 || configs[0] != "network" {
			t.Errorf("unexpected configs: %v", configs)
		}
	})
}

func testUciApplyConfirmRollback(t *testing.T, ctx context.Context, mock *testutil.MockTransport, mgr *uci.Manager) {
	t.Helper()
	t.Run("Apply_Confirm_Rollback", func(t *testing.T) {
		mock.AddResponse("uci", "apply", map[string]any{"result": 0})
		mock.AddResponse("uci", "confirm", map[string]any{"result": 0})
		mock.AddResponse("uci", "rollback", map[string]any{"result": 0})

		err := mgr.Apply(ctx, true, 30)
		if err != nil {
			t.Errorf("Apply failed: %v", err)
		}

		err = mgr.Confirm(ctx)
		if err != nil {
			t.Errorf("Confirm failed: %v", err)
		}

		err = mgr.Rollback(ctx)
		if err != nil {
			t.Errorf("Rollback failed: %v", err)
		}
	})
}

func testUciPackageOperations(t *testing.T, ctx context.Context, mock *testutil.MockTransport, mgr *uci.Manager) {
	t.Helper()
	t.Run("Package_Operations", func(t *testing.T) {
		pkg := mgr.Package("testpkg")
		testUciPackageGetAll(t, ctx, mock, pkg)
		testUciPackageAdd(t, ctx, mock, pkg)
		testUciPackageCommitRevert(t, ctx, mock, pkg)
	})
}

func testUciPackageGetAll(t *testing.T, ctx context.Context, mock *testutil.MockTransport, pkg *uci.PackageContext) {
	t.Helper()
	t.Run("GetAll", func(t *testing.T) {
		mock.AddResponse("uci", "get", map[string]any{
			"values": map[string]any{
				"s1": map[string]any{".type": "t1", "opt1": "v1"},
			},
		})

		sections, err := pkg.GetAll(ctx)
		if err != nil {
			t.Fatalf("GetAll failed: %v", err)
		}

		if len(sections) != 1 || sections["s1"].Type != "t1" {
			t.Errorf("unexpected sections: %v", sections)
		}
	})
}

func testUciPackageAdd(t *testing.T, ctx context.Context, mock *testutil.MockTransport, pkg *uci.PackageContext) {
	t.Helper()
	t.Run("Add", func(t *testing.T) {
		mock.AddResponse("uci", "add", map[string]any{"result": 0})

		values := uci.NewSectionValues()
		values.Set("opt1", "v1")

		err := pkg.Add(ctx, "t1", "s2", values)
		if err != nil {
			t.Errorf("Add failed: %v", err)
		}

		call := mock.GetLastCall()

		req, ok := call.Data.(uci.Request)
		if !ok {
			t.Fatalf("call.Data is not Request")
		}

		if req.Config != "testpkg" || req.Type != "t1" || req.Name != "s2" || req.Values["opt1"] != "v1" {
			t.Errorf("unexpected request: %+v", req)
		}
	})
}

func testUciPackageCommitRevert(
	t *testing.T,
	ctx context.Context,
	mock *testutil.MockTransport,
	pkg *uci.PackageContext,
) {
	t.Helper()
	t.Run("Commit_Revert", func(t *testing.T) {
		mock.AddResponse("uci", "commit", map[string]any{"result": 0})
		mock.AddResponse("uci", "revert", map[string]any{"result": 0})

		err := pkg.Commit(ctx)
		if err != nil {
			t.Errorf("Commit failed: %v", err)
		}

		err = pkg.Revert(ctx)
		if err != nil {
			t.Errorf("Revert failed: %v", err)
		}
	})
}

func testUciSectionOperations(t *testing.T, ctx context.Context, mock *testutil.MockTransport, mgr *uci.Manager) {
	t.Helper()
	t.Run("Section_Operations", func(t *testing.T) {
		sec := mgr.Package("testpkg").Section("s1")

		t.Run("Get", func(t *testing.T) {
			mock.AddResponse("uci", "get", map[string]any{
				"values": map[string]any{"opt1": "v1"},
			})

			s, err := sec.Get(ctx)
			if err != nil {
				t.Fatalf("Get failed: %v", err)
			}

			val, _ := s.GetFirst("opt1")
			if val != "v1" {
				t.Errorf("unexpected value: %v", val)
			}
		})

		t.Run("SetValues", func(t *testing.T) {
			mock.AddResponse("uci", "set", map[string]any{"result": 0})

			values := uci.NewSectionValues()
			values.Set("opt2", "v2")

			err := sec.SetValues(ctx, values)
			if err != nil {
				t.Errorf("SetValues failed: %v", err)
			}
		})

		t.Run("Delete_Rename", func(t *testing.T) {
			mock.AddResponse("uci", "delete", map[string]any{"result": 0})
			mock.AddResponse("uci", "rename", map[string]any{"result": 0})

			err := sec.Delete(ctx)
			if err != nil {
				t.Errorf("Delete failed: %v", err)
			}

			err = sec.Rename(ctx, "s1_new")
			if err != nil {
				t.Errorf("Rename failed: %v", err)
			}
		})
	})
}

func testUciOptionOperations(t *testing.T, ctx context.Context, mock *testutil.MockTransport, mgr *uci.Manager) {
	t.Helper()
	t.Run("Option_Operations", func(t *testing.T) {
		opt := mgr.Package("testpkg").Section("s1").Option("o1")
		testUciOptionGet(t, ctx, mock, opt)
		testUciOptionAddToList(t, ctx, mock, opt)
		testUciOptionDeleteFromList(t, ctx, mock, opt)
	})
}

func testUciOptionGet(t *testing.T, ctx context.Context, mock *testutil.MockTransport, opt *uci.OptionContext) {
	t.Helper()
	t.Run("Get", func(t *testing.T) {
		mock.AddResponse("uci", "get", map[string]any{"value": "val1"})

		val, err := opt.Get(ctx)
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}

		if val != "val1" {
			t.Errorf("expected val1, got %s", val)
		}
	})
}

func testUciOptionAddToList(t *testing.T, ctx context.Context, mock *testutil.MockTransport, opt *uci.OptionContext) {
	t.Helper()
	t.Run("AddToList", func(t *testing.T) {
		mock.AddResponse("uci", "get", map[string]any{"value": ""})
		mock.AddResponse("uci", "set", map[string]any{"result": 0})

		err := opt.AddToList(ctx, "item1")
		if err != nil {
			t.Errorf("AddToList failed: %v", err)
		}

		setReq := findLastSetRequest(t, mock)

		list := readStringList(t, setReq.Values["o1"])
		if len(list) != 1 || list[0] != "item1" {
			t.Errorf("unexpected list: %v", list)
		}

		mock.AddResponse("uci", "get", map[string]any{"value": "item1 item2"})

		err = opt.AddToList(ctx, "item3")
		if err != nil {
			t.Errorf("AddToList failed: %v", err)
		}

		setReq = findLastSetRequest(t, mock)

		list = readStringList(t, setReq.Values["o1"])
		if len(list) != 3 || list[2] != "item3" {
			t.Errorf("unexpected list: %v", list)
		}
	})
}

func testUciOptionDeleteFromList(
	t *testing.T,
	ctx context.Context,
	mock *testutil.MockTransport,
	opt *uci.OptionContext,
) {
	t.Helper()
	t.Run("DeleteFromList", func(t *testing.T) {
		mock.AddResponse("uci", "get", map[string]any{"value": "item1 item2 item3"})
		mock.AddResponse("uci", "set", map[string]any{"result": 0})

		err := opt.DeleteFromList(ctx, "item2")
		if err != nil {
			t.Errorf("DeleteFromList failed: %v", err)
		}

		setReq := findLastSetRequest(t, mock)

		list := readStringList(t, setReq.Values["o1"])
		if len(list) != 2 || list[1] != "item3" {
			t.Errorf("unexpected list: %v", list)
		}
	})
}

func findLastSetRequest(t *testing.T, mock *testutil.MockTransport) *uci.Request {
	t.Helper()

	for i := len(mock.Calls) - 1; i >= 0; i-- {
		if mock.Calls[i].Method == methodSet {
			req, ok := mock.Calls[i].Data.(uci.Request)
			if ok {
				return &req
			}
		}
	}

	t.Fatal("set call not found")

	return nil
}

func readStringList(t *testing.T, value any) []string {
	t.Helper()

	list, ok := value.([]string)
	if !ok {
		t.Fatalf("setReq.Values['o1'] is not []string")
	}

	return list
}
