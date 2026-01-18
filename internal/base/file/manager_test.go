// Copyright (c) 2026 honeybbq
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package file_test

import (
	"context"
	"testing"

	"github.com/honeybbq/goubus/v2/internal/base/file"
	"github.com/honeybbq/goubus/v2/internal/testutil"
)

func TestFileManager(t *testing.T) {
	ctx := context.Background()
	mock := testutil.NewMockTransport()

	t.Run("Read_Generic", func(t *testing.T) {
		mock.AddResponse("file", "read", map[string]any{
			"data": "hello world",
		})

		mgr := file.New(mock)

		res, err := mgr.Read(ctx, "/tmp/test", false)
		if err != nil {
			t.Fatalf("Read failed: %v", err)
		}

		if res.Data != "hello world" {
			t.Errorf("expected hello world, got %s", res.Data)
		}
	})

	t.Run("List_Generic", func(t *testing.T) {
		mock.AddResponse("file", "list", map[string]any{
			"entries": []map[string]any{
				{
					"name":  "test.txt",
					"size":  123,
					"mode":  33188,
					"atime": 1737109342,
					"mtime": 1737109342,
					"uid":   0,
					"gid":   0,
				},
			},
		})

		mgr := file.New(mock)

		list, err := mgr.List(ctx, "/tmp")
		if err != nil {
			t.Fatalf("List failed: %v", err)
		}

		if len(list.Entries) != 1 || list.Entries[0].Name != "test.txt" {
			t.Errorf("unexpected list data: %+v", list)
		}
	})
}
