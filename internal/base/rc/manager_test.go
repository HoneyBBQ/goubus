// Copyright (c) 2026 honeybbq
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package rc_test

import (
	"context"
	"testing"

	"github.com/honeybbq/goubus/v2/internal/base/rc"
	"github.com/honeybbq/goubus/v2/internal/testutil"
)

func TestRCManager(t *testing.T) {
	ctx := context.Background()
	mock := testutil.NewMockTransport()

	t.Run("List_Generic", func(t *testing.T) {
		mock.AddResponse("rc", "list", map[string]any{
			"firewall": map[string]any{
				"enabled": true,
				"running": true,
			},
		})

		mgr := rc.New(mock)

		scripts, err := mgr.List(ctx, "", false)
		if err != nil {
			t.Fatalf("List failed: %v", err)
		}

		if s, ok := scripts["firewall"]; !ok || !bool(s.Enabled) || !bool(s.Running) {
			t.Errorf("unexpected rc data: %+v", scripts)
		}
	})
}
