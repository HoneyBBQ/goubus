// Copyright (c) 2026 honeybbq
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package session_test

import (
	"context"
	"testing"

	"github.com/honeybbq/goubus/v2/internal/testutil"
	"github.com/honeybbq/goubus/v2/profiles/cmcc_rax3000m/session"
)

func TestRaxSessionManager(t *testing.T) {
	ctx := context.Background()
	mock := testutil.NewMockTransport()

	t.Run("List_RealData", func(t *testing.T) {
		err := mock.AddResponseFromFile("session", "list", "../../../internal/testdata/rax3000m/session_list.json")
		if err != nil {
			t.Fatalf("failed to load testdata: %v", err)
		}

		mgr := session.New(mock)

		_, err = mgr.List(ctx)
		if err != nil {
			t.Fatalf("List failed: %v", err)
		}
	})

	t.Run("Auth_Methods", func(t *testing.T) {
		mock.AddResponse("session", "login", map[string]any{"ubus_rpc_session": "test-session", "timeout": 3600})
		mock.AddResponse("session", "grant", map[string]any{})
		mock.AddResponse("session", "revoke", map[string]any{})
		mock.AddResponse("session", "access", map[string]any{"access": true})
		mock.AddResponse("session", "destroy", map[string]any{})

		mgr := session.New(mock)
		_, _ = mgr.Login(ctx, session.LoginRequest{Username: "root", Password: "password"})
		_ = mgr.Grant(ctx, session.GrantRequest{Session: "test", Scope: "ubus", Objects: []string{"*"}})
		_ = mgr.Revoke(ctx, session.GrantRequest{Session: "test", Scope: "ubus", Objects: []string{"*"}})
		_, _ = mgr.Access(ctx, session.AccessRequest{Session: "test", Scope: "ubus", Object: "system", Function: "info"})
		_ = mgr.Destroy(ctx, "test")
	})
}
