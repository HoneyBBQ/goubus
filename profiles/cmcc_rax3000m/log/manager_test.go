// Copyright (c) 2026 honeybbq
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package log_test

import (
	"context"
	"testing"

	"github.com/honeybbq/goubus/v2/internal/testutil"
	logpkg "github.com/honeybbq/goubus/v2/profiles/cmcc_rax3000m/log"
)

func TestRaxLogManager(t *testing.T) {
	ctx := context.Background()
	mock := testutil.NewMockTransport()

	t.Run("Write", func(t *testing.T) {
		mock.AddResponse("log", "write", map[string]any{})
		mgr := logpkg.New(mock)

		err := mgr.Write(ctx, "test event")
		if err != nil {
			t.Fatalf("Write failed: %v", err)
		}
	})

	t.Run("Read_RealData", func(t *testing.T) {
		// Although real data is currently empty, ensure calling logic is correct.
		err := mock.AddResponseFromFile("log", "read", "../../../internal/testdata/rax3000m/log_read.json")
		if err != nil {
			t.Fatalf("failed to load testdata: %v", err)
		}

		mgr := logpkg.New(mock)

		_, err = mgr.Read(ctx, 5, false, true)
		if err != nil {
			t.Fatalf("Read failed: %v", err)
		}
	})
}
