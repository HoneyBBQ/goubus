// Copyright (c) 2026 honeybbq
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package session_test

import (
	"context"
	"testing"

	"github.com/honeybbq/goubus/v2/internal/base/session"
	"github.com/honeybbq/goubus/v2/internal/testutil"
)

func TestSessionManager(t *testing.T) {
	ctx := context.Background()
	mock := testutil.NewMockTransport()

	t.Run("Create_Generic", func(t *testing.T) {
		mock.AddResponse("session", "create", map[string]any{
			"ubus_rpc_session": "1234567890abcdef",
			"timeout":          3600,
		})

		mgr := session.New(mock)

		sess, err := mgr.Create(ctx, 3600)
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}

		if sess.UbusRPCSession != "1234567890abcdef" || sess.Timeout != 3600 {
			t.Errorf("unexpected session data: %+v", sess)
		}
	})
}
