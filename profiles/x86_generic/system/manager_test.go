// Copyright (c) 2026 honeybbq
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package system_test

import (
	"context"
	"testing"

	"github.com/honeybbq/goubus/v2/internal/testutil"
	"github.com/honeybbq/goubus/v2/profiles/x86_generic/system"
)

func TestX86SystemManager(t *testing.T) {
	ctx := context.Background()
	mock := testutil.NewMockTransport()

	t.Run("Board_x86_Generic_Data", func(t *testing.T) {
		err := mock.AddResponseFromFile("system", "board", "../../../internal/testdata/x86_generic/system_board.json")
		if err != nil {
			t.Fatalf("failed to load x86 testdata: %v", err)
		}

		mgr := system.New(mock)

		board, err := mgr.Board(ctx)
		if err != nil {
			t.Fatalf("Board failed: %v", err)
		}

		// Assume data content for x86 firmware.
		if board.BoardName == "" {
			t.Error("expected non-empty board name from x86 data")
		}
	})

	t.Run("Info_x86_RealData", func(t *testing.T) {
		const path = "../../../internal/testdata/x86_generic/system_info.json"

		err := mock.AddResponseFromFile("system", "info", path)
		if err != nil {
			t.Fatalf("failed to load x86 testdata: %v", err)
		}

		mgr := system.New(mock)

		info, err := mgr.Info(ctx)
		if err != nil {
			t.Fatalf("Info failed: %v", err)
		}

		if info.Uptime == 0 {
			t.Error("expected non-zero uptime from x86 data")
		}
	})
}
