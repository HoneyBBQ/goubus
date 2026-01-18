// Copyright (c) 2026 honeybbq
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package container_test

import (
	"context"
	"testing"

	"github.com/honeybbq/goubus/v2/internal/testutil"
	"github.com/honeybbq/goubus/v2/profiles/cmcc_rax3000m/container"
)

func TestRaxContainerManager(t *testing.T) {
	ctx := context.Background()
	mock := testutil.NewMockTransport()

	t.Run("List_RealData", func(t *testing.T) {
		err := mock.AddResponseFromFile("container", "list", "../../../internal/testdata/rax3000m/container_list.json")
		if err != nil {
			t.Fatalf("failed to load testdata: %v", err)
		}

		mgr := container.New(mock)

		_, err = mgr.List(ctx, "", false)
		if err != nil {
			t.Fatalf("List failed: %v", err)
		}
	})

	t.Run("Lifecycle_Methods", func(t *testing.T) {
		mock.AddResponse("container", "set", map[string]any{})
		mock.AddResponse("container", "add", map[string]any{})
		mock.AddResponse("container", "delete", map[string]any{})
		mock.AddResponse("container", "state", map[string]any{})
		mock.AddResponse("container", "get_features", map[string]any{})
		mock.AddResponse("container", "console_set", map[string]any{})
		mock.AddResponse("container", "console_attach", map[string]any{})

		mgr := container.New(mock)
		_ = mgr.Set(ctx, container.SetRequest{Name: "test"})
		_ = mgr.Add(ctx, container.SetRequest{Name: "test"})
		_ = mgr.Delete(ctx, "test", "instance1")
		_, _ = mgr.State(ctx, "test", true)
		_, _ = mgr.GetFeatures(ctx)
		_ = mgr.ConsoleSet(ctx, "test", "instance1")
		_ = mgr.ConsoleAttach(ctx, "test", "instance1")
	})
}
