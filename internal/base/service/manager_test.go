// Copyright (c) 2026 honeybbq
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package service_test

import (
	"context"
	"testing"

	"github.com/honeybbq/goubus/v2/internal/base/service"
	"github.com/honeybbq/goubus/v2/internal/testutil"
)

func TestServiceManager(t *testing.T) {
	ctx := context.Background()
	mock := testutil.NewMockTransport()

	t.Run("List_Generic", func(t *testing.T) {
		mock.AddResponse("service", "list", map[string]any{
			"uhttpd": map[string]any{
				"instances": map[string]any{
					"main": map[string]any{
						"running": true,
						"pid":     1001,
					},
				},
			},
		})

		mgr := service.New(mock)

		services, err := mgr.List(ctx, "", false)
		if err != nil {
			t.Fatalf("List failed: %v", err)
		}

		if s, ok := services["uhttpd"]; !ok || !bool(s.Instances["main"].Running) {
			t.Errorf("unexpected service data: %+v", services)
		}
	})
}
