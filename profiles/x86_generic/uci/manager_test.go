// Copyright (c) 2026 honeybbq
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package uci_test

import (
	"context"
	"testing"

	"github.com/honeybbq/goubus/v2/internal/testutil"
	"github.com/honeybbq/goubus/v2/profiles/x86_generic/uci"
)

func TestX86UCIManager(t *testing.T) {
	ctx := context.Background()
	mock := testutil.NewMockTransport()

	t.Run("GetAll_Network_RealData", func(t *testing.T) {
		const path = "../../../internal/testdata/x86_generic/uci_get_network.json"

		err := mock.AddResponseFromFile("uci", "get", path)
		if err != nil {
			t.Fatalf("failed to load testdata: %v", err)
		}

		mgr := uci.New(mock)

		sections, err := mgr.Package("network").GetAll(ctx)
		if err != nil {
			t.Fatalf("GetAll failed: %v", err)
		}

		lan, ok := sections["lan"]
		if !ok {
			t.Fatal("lan section not found in x86 network config")
		}

		if lan.Type != "interface" {
			t.Errorf("expected type interface, got %s", lan.Type)
		}
	})
}
