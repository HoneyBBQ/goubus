// Copyright (c) 2026 honeybbq
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package file_test

import (
	"context"
	"testing"

	"github.com/honeybbq/goubus/v2/internal/testutil"
	"github.com/honeybbq/goubus/v2/profiles/cmcc_rax3000m/file"
)

const raxTestDataDir = "../../../internal/testdata/rax3000m/"

func TestRaxFileManager(t *testing.T) {
	ctx := context.Background()
	mock := testutil.NewMockTransport()
	mgr := file.New(mock)

	t.Run("FileOps", func(t *testing.T) {
		testRaxFileOps(t, ctx, mock, mgr)
	})

	t.Run("Metadata", func(t *testing.T) {
		testRaxFileMetadata(t, ctx, mock, mgr)
	})

	t.Run("System", func(t *testing.T) {
		testRaxFileSystem(t, ctx, mock, mgr)
	})
}

func testRaxFileOps(t *testing.T, ctx context.Context, mock *testutil.MockTransport, mgr *file.Manager) {
	t.Helper()
	t.Run("List", func(t *testing.T) {
		err := mock.AddResponseFromFile("file", "list", raxTestDataDir+"file_list_etc_config.json")
		if err != nil {
			t.Fatalf("failed to load testdata: %v", err)
		}

		entries, err := mgr.List(ctx, "/etc/config")
		if err != nil {
			t.Fatalf("List failed: %v", err)
		}

		if len(entries.Entries) == 0 {
			t.Error("expected non-empty list for /etc/config")
		}
	})

	t.Run("Read", func(t *testing.T) {
		mock.AddResponse("file", "read", map[string]any{"data": "test content"})

		res, err := mgr.Read(ctx, "/etc/hosts", false)
		if err != nil {
			t.Fatalf("Read failed: %v", err)
		}

		if res.Data != "test content" {
			t.Errorf("expected test content, got %s", res.Data)
		}
	})

	t.Run("Write", func(t *testing.T) {
		mock.AddResponse("file", "write", map[string]any{})

		err := mgr.Write(ctx, "/tmp/test", "content", false, 0644, false)
		if err != nil {
			t.Fatalf("Write failed: %v", err)
		}
	})

	t.Run("Remove", func(t *testing.T) {
		mock.AddResponse("file", "remove", map[string]any{})

		err := mgr.Remove(ctx, "/tmp/test")
		if err != nil {
			t.Fatalf("Remove failed: %v", err)
		}
	})
}

func testRaxFileMetadata(t *testing.T, ctx context.Context, mock *testutil.MockTransport, mgr *file.Manager) {
	t.Helper()
	t.Run("Stat", func(t *testing.T) {
		err := mock.AddResponseFromFile("file", "stat", raxTestDataDir+"file_stat_network.json")
		if err != nil {
			t.Fatalf("failed to load testdata: %v", err)
		}

		res, err := mgr.Stat(ctx, "/etc/config/network")
		if err != nil {
			t.Fatalf("Stat failed: %v", err)
		}

		if res.Type != "file" {
			t.Errorf("expected file, got %s", res.Type)
		}
	})

	t.Run("MD5", func(t *testing.T) {
		mock.AddResponse("file", "md5", map[string]any{"md5": "d41d8cd98f00b204e9800998ecf8427e"})

		res, err := mgr.MD5(ctx, "/tmp/test")
		if err != nil {
			t.Fatalf("MD5 failed: %v", err)
		}

		if res != "d41d8cd98f00b204e9800998ecf8427e" {
			t.Errorf("expected md5, got %s", res)
		}
	})

	t.Run("LStat", func(t *testing.T) {
		mock.AddResponse("file", "lstat", map[string]any{"type": "link"})

		res, err := mgr.LStat(ctx, "/tmp/link")
		if err != nil {
			t.Fatalf("LStat failed: %v", err)
		}

		if res.Type != "link" {
			t.Errorf("expected link, got %s", res.Type)
		}
	})
}

func testRaxFileSystem(t *testing.T, ctx context.Context, mock *testutil.MockTransport, mgr *file.Manager) {
	t.Helper()
	t.Run("Exec", func(t *testing.T) {
		mock.AddResponse("file", "exec", map[string]any{"code": 0, "stdout": "ok"})

		res, err := mgr.Exec(ctx, "ls", []string{"-l"}, nil)
		if err != nil {
			t.Fatalf("Exec failed: %v", err)
		}

		if res.Stdout != "ok" {
			t.Errorf("expected ok, got %v", res.Stdout)
		}
	})
}
