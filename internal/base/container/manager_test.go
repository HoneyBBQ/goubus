// Copyright (c) 2026 honeybbq
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package container_test

import (
	"context"
	"testing"

	"github.com/honeybbq/goubus/v2/internal/base/container"
	"github.com/honeybbq/goubus/v2/internal/testutil"
)

func TestContainerManager(t *testing.T) {
	ctx := context.Background()
	mock := testutil.NewMockTransport()

	t.Run("List", func(t *testing.T) {
		mock.AddResponse("container", "list", map[string]any{
			"test-container": map[string]any{
				"status": "running",
			},
		})

		mgr := container.New(mock)

		list, err := mgr.List(ctx, "", false)
		if err != nil {
			t.Fatalf("List failed: %v", err)
		}

		if _, ok := list["test-container"]; !ok {
			t.Error("expected test-container in list")
		}
	})
}
