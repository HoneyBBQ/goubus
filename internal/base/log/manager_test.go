// Copyright (c) 2026 honeybbq
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package log_test

import (
	"context"
	"testing"

	logpkg "github.com/honeybbq/goubus/v2/internal/base/log"
	"github.com/honeybbq/goubus/v2/internal/testutil"
)

func TestLogManager(t *testing.T) {
	ctx := context.Background()
	mock := testutil.NewMockTransport()

	t.Run("Read_Generic", func(t *testing.T) {
		mock.AddResponse("log", "read", map[string]any{
			"log": []map[string]any{
				{
					"text":     "user.info test log entry",
					"time":     1737109342,
					"id":       1,
					"priority": 6,
					"source":   1,
				},
			},
		})

		mgr := logpkg.New(mock)

		log, err := mgr.Read(ctx, 10, false, true)
		if err != nil {
			t.Fatalf("Read failed: %v", err)
		}

		if len(log.Log) != 1 || log.Log[0].Text != "user.info test log entry" {
			t.Errorf("unexpected log data: %+v", log)
		}
	})
}
